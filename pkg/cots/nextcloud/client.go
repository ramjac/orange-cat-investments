package nextcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client interacts with the Nextcloud OCS Provisioning and WebDAV APIs.
type Client struct {
	baseURL       string
	adminUser     string
	adminPassword string
	httpClient    *http.Client
}

// Config provides configuration for the Nextcloud client.
type Config struct {
	BaseURL       string
	AdminUser     string
	AdminPassword string
	HTTPClient    *http.Client
}

// User represents a Nextcloud user account.
type User struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayname"`
	Email       string   `json:"email"`
	Quota       string   `json:"quota"`
	Groups      []string `json:"groups,omitempty"`
	Enabled     bool     `json:"enabled"`
}

// CreateUserOption specifies parameters for provisioning a new Nextcloud user.
type CreateUserOption struct {
	UserID      string   `json:"userid"`
	Password    string   `json:"password"`
	DisplayName string   `json:"display_name,omitempty"`
	Email       string   `json:"email,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	Quota       string   `json:"quota,omitempty"` // e.g. "25 GB" or "50 GB"
}

// Document represents an intranet or handbook document hosted in Nextcloud.
type Document struct {
	DocumentID string    `json:"document_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content,omitempty"`
	Author     string    `json:"author"`
	SharedWith []string  `json:"shared_with,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewClient initializes a new Nextcloud API client.
func NewClient(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:       strings.TrimSuffix(cfg.BaseURL, "/"),
		adminUser:     cfg.AdminUser,
		adminPassword: cfg.AdminPassword,
		httpClient:    hc,
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, form url.Values) (*http.Request, error) {
	var bodyReader io.Reader
	if form != nil {
		bodyReader = strings.NewReader(form.Encode())
	}

	fullURL := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create Nextcloud HTTP request: %w", err)
	}

	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}

	return req, nil
}

// CreateUser provisions a new user with quota and group memberships.
func (c *Client) CreateUser(ctx context.Context, opt CreateUserOption) (*User, error) {
	if opt.UserID == "" {
		return nil, errors.New("userid cannot be empty")
	}
	if opt.Password == "" {
		opt.Password = fmt.Sprintf("NextCloud!%d", time.Now().UnixNano())
	}
	if opt.Quota == "" {
		opt.Quota = "25 GB"
	}

	form := url.Values{}
	form.Set("userid", opt.UserID)
	form.Set("password", opt.Password)
	if opt.DisplayName != "" {
		form.Set("displayName", opt.DisplayName)
	}
	if opt.Email != "" {
		form.Set("email", opt.Email)
	}
	form.Set("quota", opt.Quota)
	for _, g := range opt.Groups {
		form.Add("groups[]", g)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/ocs/v1.php/cloud/users?format=json", form)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Nextcloud CreateUser request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("nextcloud CreateUser returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return &User{
		ID:          opt.UserID,
		DisplayName: opt.DisplayName,
		Email:       opt.Email,
		Quota:       opt.Quota,
		Groups:      opt.Groups,
		Enabled:     true,
	}, nil
}

// DisableUser disables access for a specified user in Nextcloud.
func (c *Client) DisableUser(ctx context.Context, username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}

	req, err := c.newRequest(ctx, http.MethodPut, fmt.Sprintf("/ocs/v1.php/cloud/users/%s/disable?format=json", username), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute Nextcloud DisableUser request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("nextcloud DisableUser returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// LockAndArchiveUser locks the user account and records an archival snapshot in compliance vault.
func (c *Client) LockAndArchiveUser(ctx context.Context, username, complianceVaultPath string) error {
	if err := c.DisableUser(ctx, username); err != nil {
		return fmt.Errorf("failed to lock user account during offboarding: %w", err)
	}

	// Archive personal files via WebDAV/compliance move
	form := url.Values{}
	form.Set("archive_target", complianceVaultPath)
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/ocs/v1.php/cloud/users/%s/archive?format=json", username), form)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to archive user files to vault: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// PublishDocument publishes or updates an intranet standard operating procedure document.
func (c *Client) PublishDocument(ctx context.Context, doc Document) (*Document, error) {
	if doc.Title == "" {
		return nil, errors.New("document title cannot be empty")
	}

	payload, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode document payload: %w", err)
	}

	url := fmt.Sprintf("%s/nextcloud/api/v1/documents", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PublishDocument request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("publish document returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var created Document
	_ = json.Unmarshal(bodyBytes, &created)
	if created.DocumentID == "" {
		created = doc
		if created.DocumentID == "" {
			created.DocumentID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
		}
	}
	return &created, nil
}
