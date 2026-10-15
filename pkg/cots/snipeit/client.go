package snipeit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client interacts with the Snipe-IT IT Asset Management REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Config provides configuration for the Snipe-IT client.
type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// Asset represents a physical hardware asset tracked in Snipe-IT.
type Asset struct {
	ID           int64     `json:"id"`
	AssetTag     string    `json:"asset_tag"`
	Name         string    `json:"name"`
	Model        string    `json:"model,omitempty"`
	Serial       string    `json:"serial,omitempty"`
	StatusLabel  string    `json:"status_label,omitempty"`
	AssignedTo   string    `json:"assigned_to,omitempty"`
	AssignedType string    `json:"assigned_type,omitempty"` // "user"
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// CheckoutRequest specifies parameters for checking out hardware to an employee.
type CheckoutRequest struct {
	AssetID      int64  `json:"asset_id"`
	AssignedUser string `json:"assigned_user"`
	Note         string `json:"note,omitempty"`
}

// CheckinRequest specifies parameters for checking in / returning hardware.
type CheckinRequest struct {
	AssetID int64  `json:"asset_id"`
	Note    string `json:"note,omitempty"`
}

// AssetRecoveryTicket represents an equipment recovery workflow for departing employees.
type AssetRecoveryTicket struct {
	TicketID     string    `json:"ticket_id"`
	EmployeeID   string    `json:"employee_id"`
	Assets       []Asset   `json:"assets"`
	Status       string    `json:"status"` // "Pending Recovery", "Completed"
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewClient initializes a new Snipe-IT client.
func NewClient(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimSuffix(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: hc,
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Snipe-IT request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create Snipe-IT HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return req, nil
}

// CheckoutAsset checks out an IT asset to an employee.
func (c *Client) CheckoutAsset(ctx context.Context, req CheckoutRequest) (*Asset, error) {
	if req.AssetID == 0 || req.AssignedUser == "" {
		return nil, errors.New("asset_id and assigned_user are required for checkout")
	}

	path := fmt.Sprintf("/api/v1/hardware/%d/checkout", req.AssetID)
	payload := map[string]any{
		"assigned_user": req.AssignedUser,
		"checkout_to_type": "user",
		"note": req.Note,
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Snipe-IT CheckoutAsset request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("snipe-it CheckoutAsset returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return &Asset{
		ID:           req.AssetID,
		AssignedTo:   req.AssignedUser,
		AssignedType: "user",
		StatusLabel:  "Deployed",
		Notes:        req.Note,
	}, nil
}

// CheckinAsset checks in an IT asset from an employee during offboarding.
func (c *Client) CheckinAsset(ctx context.Context, assetID int64, req CheckinRequest) (*Asset, error) {
	if assetID == 0 {
		return nil, errors.New("asset_id is required for checkin")
	}

	path := fmt.Sprintf("/api/v1/hardware/%d/checkin", assetID)
	payload := map[string]any{
		"note": req.Note,
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Snipe-IT CheckinAsset request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("snipe-it CheckinAsset returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return &Asset{
		ID:          assetID,
		StatusLabel: "Ready to Deploy",
		Notes:       req.Note,
	}, nil
}

// ListAssignedAssets returns all hardware assets currently assigned to an employee.
func (c *Client) ListAssignedAssets(ctx context.Context, employeeID string) ([]Asset, error) {
	if employeeID == "" {
		return nil, errors.New("employee_id cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/hardware?assigned_to=%s", employeeID)
	httpReq, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Snipe-IT ListAssignedAssets request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snipe-it ListAssignedAssets returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		Rows []Asset `json:"rows"`
	}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode assets list response: %w", err)
	}
	return res.Rows, nil
}

// CreateAssetRecoveryTicket creates an equipment recovery ticket for asset return during employee offboarding.
func (c *Client) CreateAssetRecoveryTicket(ctx context.Context, employeeID, reason string) (*AssetRecoveryTicket, error) {
	if employeeID == "" {
		return nil, errors.New("employee_id cannot be empty")
	}

	assets, _ := c.ListAssignedAssets(ctx, employeeID)
	ticketID := fmt.Sprintf("RECOVER-%s-%d", employeeID, time.Now().Unix())

	ticket := &AssetRecoveryTicket{
		TicketID:   ticketID,
		EmployeeID: employeeID,
		Assets:     assets,
		Status:     "Pending Recovery",
		Reason:     reason,
		CreatedAt:  time.Now().UTC(),
	}

	return ticket, nil
}
