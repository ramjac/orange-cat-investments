package plaid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client handles communication with the Bank / Plaid API.
type Client struct {
	baseURL    string
	clientID   string
	secret     string
	httpClient *http.Client
}

// Config holds configuration parameters for the Bank client.
type Config struct {
	BaseURL    string
	ClientID   string
	Secret     string
	HTTPClient *http.Client
}

// NewClient creates a new Bank client.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8088"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &Client{
		baseURL:    baseURL,
		clientID:   cfg.ClientID,
		secret:     cfg.Secret,
		httpClient: httpClient,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, target any) error {
	fullURL := c.baseURL + path
	var bodyReader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
	}
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.clientID != "" {
		req.Header.Set("PLAID-CLIENT-ID", c.clientID)
	}
	if c.secret != "" {
		req.Header.Set("PLAID-SECRET", c.secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			ErrorMessage string `json:"error_message"`
			Error        string `json:"error"`
			ErrorCode    string `json:"error_code"`
			ReturnCode   string `json:"return_code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp.ErrorMessage
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return fmt.Errorf("bank API error (%d): %s (code=%s)", resp.StatusCode, msg, errResp.ReturnCode)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// CreateLinkToken initiates an account linking session token.
func (c *Client) CreateLinkToken(ctx context.Context, clientUserID string) (string, error) {
	req := map[string]string{"client_user_id": clientUserID}
	var resp struct {
		LinkToken string `json:"link_token"`
	}
	if err := c.do(ctx, http.MethodPost, "/link/token/create", req, &resp); err != nil {
		return "", err
	}
	return resp.LinkToken, nil
}

// ExchangePublicToken exchanges a public link token for an access token.
func (c *Client) ExchangePublicToken(ctx context.Context, publicToken, clientUserID string) (string, error) {
	req := map[string]string{
		"public_token":   publicToken,
		"client_user_id": clientUserID,
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(ctx, http.MethodPost, "/item/public_token/exchange", req, &resp); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

// GetAccounts fetches accounts owned by a customer or persona.
func (c *Client) GetAccounts(ctx context.Context, ownerID string) ([]Account, error) {
	path := "/accounts"
	if ownerID != "" {
		path += "?owner_id=" + url.QueryEscape(ownerID)
	}
	var resp struct {
		Accounts []Account `json:"accounts"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Accounts, nil
}

// GetBalances retrieves real-time available and current balances.
func (c *Client) GetBalances(ctx context.Context, accessToken string, accountIDs []string) ([]AccountWithBalance, error) {
	req := map[string]any{
		"access_token": accessToken,
		"account_ids":  accountIDs,
	}
	var resp struct {
		Accounts []AccountWithBalance `json:"accounts"`
	}
	if err := c.do(ctx, http.MethodPost, "/accounts/balance/get", req, &resp); err != nil {
		return nil, err
	}
	return resp.Accounts, nil
}

// AuthorizeTransfer checks funds and returns pre-authorization decision.
func (c *Client) AuthorizeTransfer(ctx context.Context, accountID string, amount float64, transferType string) (*TransferAuthorization, error) {
	req := map[string]any{
		"account_id": accountID,
		"amount":     amount,
		"type":       transferType,
	}
	var resp struct {
		Authorization TransferAuthorization `json:"authorization"`
	}
	if err := c.do(ctx, http.MethodPost, "/transfer/authorization/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Authorization, nil
}

// CreateTransfer initiates an ACH transfer.
func (c *Client) CreateTransfer(ctx context.Context, req CreateTransferRequest) (*Transfer, error) {
	var resp struct {
		Transfer Transfer `json:"transfer"`
	}
	if err := c.do(ctx, http.MethodPost, "/transfer/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Transfer, nil
}

// GetTransfer fetches details of a specific transfer.
func (c *Client) GetTransfer(ctx context.Context, transferID string) (*Transfer, error) {
	path := "/transfer/get?transfer_id=" + url.QueryEscape(transferID)
	var resp struct {
		Transfer Transfer `json:"transfer"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Transfer, nil
}

// ListTransfers fetches transfers filtered by accountID.
func (c *Client) ListTransfers(ctx context.Context, accountID string) ([]Transfer, error) {
	path := "/transfer/list"
	if accountID != "" {
		path += "?account_id=" + url.QueryEscape(accountID)
	}
	var resp struct {
		Transfers []Transfer `json:"transfers"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Transfers, nil
}
