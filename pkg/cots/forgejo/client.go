package forgejo

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

// Client interacts with the Forgejo REST API v1.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Config provides configuration for the Forgejo client.
type Config struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// User represents a Forgejo user account.
type User struct {
	ID        int64     `json:"id"`
	UserName  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	IsAdmin   bool      `json:"is_admin"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created"`
}

// CreateUserOption specifies parameters for creating a new Forgejo user.
type CreateUserOption struct {
	Username           string `json:"username"`
	Email              string `json:"email"`
	FullName           string `json:"full_name,omitempty"`
	Password           string `json:"password"`
	MustChangePassword *bool  `json:"must_change_password,omitempty"`
	SendNotify         bool   `json:"send_notify,omitempty"`
}

// PublicKey represents an SSH public key registered to a user.
type PublicKey struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Key         string    `json:"key"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	ReadOnly    bool      `json:"read_only"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateKeyOption specifies parameters for registering an SSH public key.
type CreateKeyOption struct {
	Title    string `json:"title"`
	Key      string `json:"key"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

// NewClient initializes a new Forgejo REST API client.
func NewClient(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimSuffix(cfg.BaseURL, "/"),
		token:      cfg.Token,
		httpClient: hc,
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Forgejo request payload: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create Forgejo HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	return req, nil
}

// CreateUser creates a new user account in Forgejo via admin API.
func (c *Client) CreateUser(ctx context.Context, opt CreateUserOption) (*User, error) {
	if opt.Username == "" || opt.Email == "" {
		return nil, errors.New("username and email are required to create a Forgejo user")
	}
	if opt.Password == "" {
		opt.Password = fmt.Sprintf("TempPass!%d", time.Now().UnixNano())
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/api/v1/admin/users", opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateUser request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateUser returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var user User
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo user response: %w", err)
	}
	return &user, nil
}

// GetUser retrieves a Forgejo user by username.
func (c *Client) GetUser(ctx context.Context, username string) (*User, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/api/v1/users/"+username, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo GetUser request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user %s not found in Forgejo", username)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo GetUser returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var user User
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo user response: %w", err)
	}
	return &user, nil
}

// AddUserPublicKey adds an SSH public key to the specified user's account.
func (c *Client) AddUserPublicKey(ctx context.Context, username string, opt CreateKeyOption) (*PublicKey, error) {
	if username == "" || opt.Key == "" {
		return nil, errors.New("username and SSH key are required")
	}
	if opt.Title == "" {
		opt.Title = fmt.Sprintf("Key-%d", time.Now().Unix())
	}

	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%s/keys", username), opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo AddUserPublicKey request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo AddUserPublicKey returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var key PublicKey
	if err := json.Unmarshal(bodyBytes, &key); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo key response: %w", err)
	}
	return &key, nil
}

// DeleteUser deletes or disables a user account in Forgejo.
func (c *Client) DeleteUser(ctx context.Context, username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}

	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%s", username), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute Forgejo DeleteUser request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("forgejo DeleteUser returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
