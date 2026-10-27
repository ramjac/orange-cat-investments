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

// Issue represents a Forgejo repository issue.
type Issue struct {
	ID        int64     `json:"id"`
	Index     int64     `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	User      User      `json:"user"`
	Comments  int       `json:"comments"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateIssueOption specifies parameters for creating a new issue in Forgejo.
type CreateIssueOption struct {
	Title    string   `json:"title"`
	Body     string   `json:"body,omitempty"`
	Assignee string   `json:"assignee,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// UpdateIssueOption specifies parameters for modifying an existing issue.
type UpdateIssueOption struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
	State *string `json:"state,omitempty"`
}

// Comment represents a comment on a Forgejo issue.
type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCommentOption specifies parameters for adding a comment to an issue.
type CreateCommentOption struct {
	Body string `json:"body"`
}

// Branch represents a git branch in a Forgejo repository.
type Branch struct {
	Name      string     `json:"name"`
	Commit    CommitMeta `json:"commit"`
	Protected bool       `json:"protected"`
}

// CommitMeta contains metadata about a git commit.
type CommitMeta struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateBranchOption defines parameters for creating a new branch.
type CreateBranchOption struct {
	BranchName string `json:"new_branch_name"`
	OldRefName string `json:"old_ref_name"`
}

// CreateFileOption defines parameters for creating a file on a branch.
type CreateFileOption struct {
	Branch  string `json:"branch"`
	Content string `json:"content"`
	Message string `json:"message"`
}

// PullRequest represents a Forgejo pull request.
type PullRequest struct {
	ID        int64             `json:"id"`
	Index     int64             `json:"number"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	State     string            `json:"state"`
	Head      PullRequestBranch `json:"head"`
	Base      PullRequestBranch `json:"base"`
	Merged    bool              `json:"merged"`
	CreatedAt time.Time         `json:"created_at"`
}

// PullRequestBranch represents the head or base branch of a PR.
type PullRequestBranch struct {
	Ref string `json:"ref"`
}

// CreatePullRequestOption defines parameters for opening a pull request.
type CreatePullRequestOption struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// Organization represents a Forgejo organization.
type Organization struct {
	ID          int64  `json:"id"`
	UserName    string `json:"username"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

// CreateOrgOption defines parameters for creating an organization.
type CreateOrgOption struct {
	UserName    string `json:"username"`
	FullName    string `json:"full_name,omitempty"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

// Repository represents a Forgejo git repository.
type Repository struct {
	ID            int64  `json:"id"`
	Owner         *User  `json:"owner,omitempty"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HasActions    bool   `json:"has_actions"`
	HasIssues     bool   `json:"has_issues"`
	HasPull       bool   `json:"has_pull_requests"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
}

// CreateRepoOption defines parameters for creating a new repository.
type CreateRepoOption struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private"`
	AutoInit      bool   `json:"auto_init"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

// UpdateRepoOption defines parameters for updating an existing repository.
type UpdateRepoOption struct {
	Description *string `json:"description,omitempty"`
	Private     *bool   `json:"private,omitempty"`
	HasActions  *bool   `json:"has_actions,omitempty"`
	HasIssues   *bool   `json:"has_issues,omitempty"`
	HasPull     *bool   `json:"has_pull_requests,omitempty"`
}

// CreatePullReviewOption defines parameters for reviewing a PR.
type CreatePullReviewOption struct {
	Event string `json:"event"`
	Body  string `json:"body"`
}

// MergePullRequestOption defines parameters for merging a PR.
type MergePullRequestOption struct {
	Do string `json:"Do"`
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
// For production OCI applications, a non-empty Password is required.
func (c *Client) CreateUser(ctx context.Context, opt CreateUserOption) (*User, error) {
	if opt.Username == "" || opt.Email == "" {
		return nil, errors.New("username and email are required to create a Forgejo user")
	}
	if opt.Password == "" {
		return nil, errors.New("password is required to create a Forgejo user")
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

// CreateIssue creates a new issue in the specified repository.
func (c *Client) CreateIssue(ctx context.Context, owner, repo string, opt CreateIssueOption) (*Issue, error) {
	if owner == "" || repo == "" {
		return nil, errors.New("owner and repo are required to create an issue")
	}
	if opt.Title == "" {
		return nil, errors.New("title is required to create an issue")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner, repo)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateIssue request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateIssue returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var issue Issue
	if err := json.Unmarshal(bodyBytes, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo issue response: %w", err)
	}
	return &issue, nil
}

// GetIssue retrieves an issue by its index number.
func (c *Client) GetIssue(ctx context.Context, owner, repo string, index int64) (*Issue, error) {
	if owner == "" || repo == "" || index <= 0 {
		return nil, errors.New("owner, repo, and valid index are required to get an issue")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, index)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo GetIssue request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo GetIssue returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var issue Issue
	if err := json.Unmarshal(bodyBytes, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo issue response: %w", err)
	}
	return &issue, nil
}

// UpdateIssue updates an issue's title, body, or state (open/closed).
func (c *Client) UpdateIssue(ctx context.Context, owner, repo string, index int64, opt UpdateIssueOption) (*Issue, error) {
	if owner == "" || repo == "" || index <= 0 {
		return nil, errors.New("owner, repo, and valid index are required to update an issue")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, index)
	req, err := c.newRequest(ctx, http.MethodPatch, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo UpdateIssue request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("forgejo UpdateIssue returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var issue Issue
	if err := json.Unmarshal(bodyBytes, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo issue response: %w", err)
	}
	return &issue, nil
}

// CreateIssueComment posts a comment on the specified issue.
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, index int64, body string) (*Comment, error) {
	if owner == "" || repo == "" || index <= 0 {
		return nil, errors.New("owner, repo, and valid index are required")
	}
	if body == "" {
		return nil, errors.New("comment body cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", owner, repo, index)
	opt := CreateCommentOption{Body: body}
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateIssueComment request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateIssueComment returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var comment Comment
	if err := json.Unmarshal(bodyBytes, &comment); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo comment response: %w", err)
	}
	return &comment, nil
}

// ListIssueComments retrieves all comments for the specified issue.
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, index int64) ([]*Comment, error) {
	if owner == "" || repo == "" || index <= 0 {
		return nil, errors.New("owner, repo, and valid index are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", owner, repo, index)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo ListIssueComments request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo ListIssueComments returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var comments []*Comment
	if err := json.Unmarshal(bodyBytes, &comments); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo comments response: %w", err)
	}
	return comments, nil
}

// CreateBranch creates a new git branch in the repository.
func (c *Client) CreateBranch(ctx context.Context, owner, repo string, opt CreateBranchOption) (*Branch, error) {
	if owner == "" || repo == "" || opt.BranchName == "" {
		return nil, errors.New("owner, repo, and BranchName are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/branches", owner, repo)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateBranch request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateBranch returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var branch Branch
	if err := json.Unmarshal(bodyBytes, &branch); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo branch response: %w", err)
	}
	return &branch, nil
}

// ListBranches retrieves the list of git branches for the repository.
func (c *Client) ListBranches(ctx context.Context, owner, repo string) ([]*Branch, error) {
	if owner == "" || repo == "" {
		return nil, errors.New("owner and repo are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/branches", owner, repo)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo ListBranches request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo ListBranches returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var branches []*Branch
	if err := json.Unmarshal(bodyBytes, &branches); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo branches response: %w", err)
	}
	return branches, nil
}

// CreateOrUpdateFile creates or updates a file in the repository branch.
func (c *Client) CreateOrUpdateFile(ctx context.Context, owner, repo, filepath string, opt CreateFileOption) (map[string]any, error) {
	if owner == "" || repo == "" || filepath == "" {
		return nil, errors.New("owner, repo, and filepath are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", owner, repo, filepath)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateOrUpdateFile request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateOrUpdateFile returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo file response: %w", err)
	}
	return result, nil
}

// CreatePullRequest opens a pull request in the repository.
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, opt CreatePullRequestOption) (*PullRequest, error) {
	if owner == "" || repo == "" || opt.Title == "" || opt.Head == "" || opt.Base == "" {
		return nil, errors.New("owner, repo, title, head, and base are required to create a PR")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner, repo)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreatePullRequest request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreatePullRequest returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var pr PullRequest
	if err := json.Unmarshal(bodyBytes, &pr); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo PR response: %w", err)
	}
	return &pr, nil
}

// ListPullRequests retrieves open pull requests for the repository.
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string) ([]*PullRequest, error) {
	if owner == "" || repo == "" {
		return nil, errors.New("owner and repo are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner, repo)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo ListPullRequests request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo ListPullRequests returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var prs []*PullRequest
	if err := json.Unmarshal(bodyBytes, &prs); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo PRs response: %w", err)
	}
	return prs, nil
}

// CreatePullReview submits a review (APPROVED, REJECTED, COMMENT) on a pull request.
func (c *Client) CreatePullReview(ctx context.Context, owner, repo string, index int64, opt CreatePullReviewOption) (map[string]any, error) {
	if owner == "" || repo == "" || index <= 0 {
		return nil, errors.New("owner, repo, and valid index are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/reviews", owner, repo, index)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreatePullReview request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("forgejo CreatePullReview returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo review response: %w", err)
	}
	return result, nil
}

// MergePullRequest merges an approved pull request into the base branch.
func (c *Client) MergePullRequest(ctx context.Context, owner, repo string, index int64, opt MergePullRequestOption) (map[string]any, error) {
	if owner == "" || repo == "" || index <= 0 {
		return nil, errors.New("owner, repo, and valid index are required")
	}
	if opt.Do == "" {
		opt.Do = "merge"
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", owner, repo, index)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo MergePullRequest request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo MergePullRequest returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo merge response: %w", err)
	}
	return result, nil
}

// CreateOrganization creates a new organization in Forgejo.
func (c *Client) CreateOrganization(ctx context.Context, opt CreateOrgOption) (*Organization, error) {
	if opt.UserName == "" {
		return nil, errors.New("organization username is required")
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/api/v1/orgs", opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateOrganization request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateOrganization returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var org Organization
	if err := json.Unmarshal(bodyBytes, &org); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo organization response: %w", err)
	}
	return &org, nil
}

// GetOrganization retrieves an organization by name.
func (c *Client) GetOrganization(ctx context.Context, orgName string) (*Organization, error) {
	if orgName == "" {
		return nil, errors.New("organization name cannot be empty")
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/api/v1/orgs/"+orgName, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo GetOrganization request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo GetOrganization returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var org Organization
	if err := json.Unmarshal(bodyBytes, &org); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo organization response: %w", err)
	}
	return &org, nil
}

// CreateOrgRepository creates a new repository under an organization.
func (c *Client) CreateOrgRepository(ctx context.Context, orgName string, opt CreateRepoOption) (*Repository, error) {
	if orgName == "" || opt.Name == "" {
		return nil, errors.New("organization name and repository name are required")
	}

	path := fmt.Sprintf("/api/v1/orgs/%s/repos", orgName)
	req, err := c.newRequest(ctx, http.MethodPost, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo CreateOrgRepository request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo CreateOrgRepository returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var repo Repository
	if err := json.Unmarshal(bodyBytes, &repo); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo repository response: %w", err)
	}
	return &repo, nil
}

// GetRepository retrieves repository metadata by owner and name.
func (c *Client) GetRepository(ctx context.Context, owner, repoName string) (*Repository, error) {
	if owner == "" || repoName == "" {
		return nil, errors.New("owner and repo name are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s", owner, repoName)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo GetRepository request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo GetRepository returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var repo Repository
	if err := json.Unmarshal(bodyBytes, &repo); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo repository response: %w", err)
	}
	return &repo, nil
}

// UpdateRepository updates repository settings (such as enabling actions, issues, PRs).
func (c *Client) UpdateRepository(ctx context.Context, owner, repoName string, opt UpdateRepoOption) (*Repository, error) {
	if owner == "" || repoName == "" {
		return nil, errors.New("owner and repo name are required")
	}

	path := fmt.Sprintf("/api/v1/repos/%s/%s", owner, repoName)
	req, err := c.newRequest(ctx, http.MethodPatch, path, opt)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Forgejo UpdateRepository request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forgejo UpdateRepository returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var repo Repository
	if err := json.Unmarshal(bodyBytes, &repo); err != nil {
		return nil, fmt.Errorf("failed to decode Forgejo repository response: %w", err)
	}
	return &repo, nil
}

