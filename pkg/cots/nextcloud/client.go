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

// GroupFolder represents a Nextcloud departmental group folder.
type GroupFolder struct {
	ID         int64    `json:"id"`
	MountPoint string   `json:"mount_point"`
	Quota      string   `json:"quota"`
	Groups     []string `json:"groups"`
}

// Document represents an intranet or handbook document hosted in Nextcloud.
type Document struct {
	DocumentID string    `json:"document_id"`
	Title      string    `json:"title"`
	Department string    `json:"department,omitempty"`
	Category   string    `json:"category,omitempty"`
	Content    string    `json:"content,omitempty"`
	Author     string    `json:"author"`
	SharedWith []string  `json:"shared_with,omitempty"`
	WebDAVPath string    `json:"webdav_path,omitempty"`
	Version    string    `json:"version,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

// CalendarEvent represents an event in a Nextcloud shared company calendar.
type CalendarEvent struct {
	EventID     string    `json:"event_id"`
	Calendar    string    `json:"calendar"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Department  string    `json:"department,omitempty"`
	Location    string    `json:"location,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Organizer   string    `json:"organizer,omitempty"`
	Status      string    `json:"status,omitempty"`
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
// For production OCI applications, a non-empty Password is required.
func (c *Client) CreateUser(ctx context.Context, opt CreateUserOption) (*User, error) {
	if opt.UserID == "" {
		return nil, errors.New("userid cannot be empty")
	}
	if opt.Password == "" {
		return nil, errors.New("password cannot be empty")
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
	if resp.StatusCode == http.StatusNotFound {
		folder := "Documents"
		if doc.Department != "" {
			folder = doc.Department
		}
		slug := strings.ToLower(strings.ReplaceAll(doc.Title, " ", "_"))
		webdavPath := fmt.Sprintf("/remote.php/dav/files/admin/%s/%s.md", folder, slug)
		_ = c.UploadWebDAVFile(ctx, webdavPath, "text/markdown", []byte(doc.Content))
		created := doc
		if created.DocumentID == "" {
			created.DocumentID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
		}
		if created.CreatedAt.IsZero() {
			created.CreatedAt = time.Now().UTC()
		}
		created.UpdatedAt = time.Now().UTC()
		created.WebDAVPath = webdavPath
		return &created, nil
	}
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
	if created.CreatedAt.IsZero() {
		created.CreatedAt = time.Now().UTC()
	}
	if created.UpdatedAt.IsZero() {
		created.UpdatedAt = time.Now().UTC()
	}
	if created.WebDAVPath == "" {
		folder := "General"
		if created.Department != "" {
			folder = created.Department
		}
		slug := strings.ToLower(strings.ReplaceAll(created.Title, " ", "_"))
		created.WebDAVPath = fmt.Sprintf("/remote.php/dav/files/admin/%s/%s.md", folder, slug)
	}
	return &created, nil
}

// ListGroupFolders returns departmental group folders configured in Nextcloud.
func (c *Client) ListGroupFolders(ctx context.Context) ([]GroupFolder, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/apps/groupfolders/folders?format=json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list group folders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var wrapper struct {
			OCS struct {
				Data map[string]GroupFolder `json:"data"`
			} `json:"ocs"`
			Folders []GroupFolder `json:"folders"`
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(bodyBytes, &wrapper); err == nil {
			if len(wrapper.Folders) > 0 {
				return wrapper.Folders, nil
			}
			if len(wrapper.OCS.Data) > 0 {
				folders := make([]GroupFolder, 0, len(wrapper.OCS.Data))
				for _, f := range wrapper.OCS.Data {
					folders = append(folders, f)
				}
				return folders, nil
			}
		}
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("nextcloud ListGroupFolders returned status %d: %s", resp.StatusCode, string(bodyBytes))
}

// CreateGroupFolder configures a new departmental group folder in Nextcloud.
func (c *Client) CreateGroupFolder(ctx context.Context, mountPoint string, groups []string, quota string) (*GroupFolder, error) {
	if mountPoint == "" {
		return nil, errors.New("mount_point cannot be empty")
	}
	form := url.Values{}
	form.Set("mountpoint", mountPoint)
	req, err := c.newRequest(ctx, http.MethodPost, "/apps/groupfolders/folders?format=json", form)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create group folder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create group folder returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if quota == "" {
		quota = "50 GB"
	}
	return &GroupFolder{
		ID:         time.Now().Unix(),
		MountPoint: mountPoint,
		Quota:      quota,
		Groups:     groups,
	}, nil
}

// ListDocuments retrieves intranet documents with optional department, category, or search filters.
func (c *Client) ListDocuments(ctx context.Context, department, category, query string) ([]Document, error) {
	u, err := url.Parse(fmt.Sprintf("%s/nextcloud/api/v1/documents", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if department != "" {
		q.Set("department", department)
	}
	if category != "" {
		q.Set("category", category)
	}
	if query != "" {
		q.Set("q", query)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list Nextcloud documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var wrapper struct {
			Documents []Document `json:"documents"`
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(bodyBytes, &wrapper); err == nil && len(wrapper.Documents) > 0 {
			return filterDocuments(wrapper.Documents, department, category, query), nil
		}
		var list []Document
		if err := json.Unmarshal(bodyBytes, &list); err == nil && len(list) > 0 {
			return filterDocuments(list, department, category, query), nil
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		return []Document{}, nil
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("nextcloud ListDocuments returned status %d: %s", resp.StatusCode, string(bodyBytes))
}

// GetDocument fetches a specific intranet document by ID.
func (c *Client) GetDocument(ctx context.Context, id string) (*Document, error) {
	if id == "" {
		return nil, errors.New("document id cannot be empty")
	}
	reqURL := fmt.Sprintf("%s/nextcloud/api/v1/documents/%s", c.baseURL, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get Nextcloud document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var doc Document
		if err := json.NewDecoder(resp.Body).Decode(&doc); err == nil {
			return &doc, nil
		}
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("document %s not found in Nextcloud (status %d): %s", id, resp.StatusCode, string(bodyBytes))
}

// ListCalendarEvents returns scheduled shifts, maintenance windows, and company milestones.
func (c *Client) ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]CalendarEvent, error) {
	u, err := url.Parse(fmt.Sprintf("%s/nextcloud/api/v1/calendar/events", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if calendar != "" {
		q.Set("calendar", calendar)
	}
	if department != "" {
		q.Set("department", department)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var wrapper struct {
			Events []CalendarEvent `json:"events"`
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(bodyBytes, &wrapper); err == nil && len(wrapper.Events) > 0 {
			return filterCalendarEvents(wrapper.Events, calendar, department), nil
		}
		var list []CalendarEvent
		if err := json.Unmarshal(bodyBytes, &list); err == nil && len(list) > 0 {
			return filterCalendarEvents(list, calendar, department), nil
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		return []CalendarEvent{}, nil
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("nextcloud ListCalendarEvents returned status %d: %s", resp.StatusCode, string(bodyBytes))
}

// CreateCalendarEvent schedules a new shift, maintenance, or milestone calendar event.
func (c *Client) CreateCalendarEvent(ctx context.Context, event CalendarEvent) (*CalendarEvent, error) {
	if event.Title == "" {
		return nil, errors.New("event title cannot be empty")
	}
	if event.Calendar == "" {
		event.Calendar = "Shift Rotations"
	}
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Status == "" {
		event.Status = "scheduled"
	}
	if event.StartTime.IsZero() {
		event.StartTime = time.Now().UTC().Add(1 * time.Hour)
	}
	if event.EndTime.IsZero() {
		event.EndTime = event.StartTime.Add(2 * time.Hour)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to encode calendar event: %w", err)
	}

	reqURL := fmt.Sprintf("%s/nextcloud/api/v1/calendar/events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute CreateCalendarEvent request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create calendar event returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var created CalendarEvent
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if created.EventID == "" {
		created = event
	}
	return &created, nil
}

func filterDocuments(docs []Document, dept, cat, q string) []Document {
	var filtered []Document
	for _, d := range docs {
		if dept != "" && !strings.EqualFold(d.Department, dept) {
			continue
		}
		if cat != "" && !strings.EqualFold(d.Category, cat) {
			continue
		}
		if q != "" {
			term := strings.ToLower(q)
			if !strings.Contains(strings.ToLower(d.Title), term) && !strings.Contains(strings.ToLower(d.Content), term) {
				continue
			}
		}
		filtered = append(filtered, d)
	}
	return filtered
}

func filterCalendarEvents(events []CalendarEvent, cal, dept string) []CalendarEvent {
	var filtered []CalendarEvent
	for _, ev := range events {
		if cal != "" && !strings.EqualFold(ev.Calendar, cal) {
			continue
		}
		if dept != "" && !strings.EqualFold(ev.Department, dept) {
			continue
		}
		filtered = append(filtered, ev)
	}
	return filtered
}

// DefaultGroupFolders returns the 4 standard departmental group folders for OCI.
func DefaultGroupFolders() []GroupFolder {
	return []GroupFolder{
		{
			ID:         1,
			MountPoint: "HR & Policies",
			Quota:      "50 GB",
			Groups:     []string{"hr", "workforce", "all-staff"},
		},
		{
			ID:         2,
			MountPoint: "Facilities & Maintenance SOPs",
			Quota:      "100 GB",
			Groups:     []string{"facilities", "operations", "technicians"},
		},
		{
			ID:         3,
			MountPoint: "Veterinary Protocols",
			Quota:      "50 GB",
			Groups:     []string{"veterinary", "medical", "caretakers"},
		},
		{
			ID:         4,
			MountPoint: "Executive Reporting & Finance",
			Quota:      "25 GB",
			Groups:     []string{"executive", "finance", "audit"},
		},
	}
}

// DefaultDocuments provides pre-populated intranet knowledge base documents.
func DefaultDocuments() []Document {
	now := time.Now().UTC()
	return []Document{
		{
			DocumentID: "doc-001",
			Title:      "OCI Employee & Feline Care Onboarding Handbook",
			Department: "HR & Policies",
			Category:   "Policy",
			Author:     "user-operations-lead",
			Content:    "# OCI Employee & Feline Care Onboarding Handbook\n\nWelcome to Orange Cat Investments! This handbook outlines company standards, feline executive etiquette, catnip break guidelines, and IT equipment policies.\n\n### 1. Feline Executive Respect\n- Always respect feline napping schedules.\n- Never alter dietary formulas without designated veterinary medical staff approval.\n\n### 2. IT & Security\n- All laptop assets are secured via Snipe-IT.\n- Use Git SSH keys for all Forgejo repositories.",
			SharedWith: []string{"group-staff", "group-operations"},
			WebDAVPath: "/remote.php/dav/files/admin/HR & Policies/onboarding_handbook.md",
			Version:    "1.4",
			CreatedAt:  now.Add(-48 * time.Hour),
			UpdatedAt:  now.Add(-12 * time.Hour),
		},
		{
			DocumentID: "doc-002",
			Title:      "Zero-Trust Microservices & PKI Security Standard",
			Department: "Facilities & Maintenance SOPs",
			Category:   "Architecture",
			Author:     "user-infrastructure-lead",
			Content:    "# Zero-Trust Microservices & PKI Security Standard\n\n### Principles\n- Zero long-lived or static certificates.\n- Dynamic cert-manager X.509 rotation in Go standard library `crypto/tls`.\n- Opaque session cookies in Valkey; no raw JWTs exposed to web browsers.\n- Double-submit CSRF enforcement on all state-changing endpoints.",
			SharedWith: []string{"group-engineering", "group-facilities"},
			WebDAVPath: "/remote.php/dav/files/admin/Facilities & Maintenance SOPs/zero_trust_pki_spec.md",
			Version:    "2.1",
			CreatedAt:  now.Add(-72 * time.Hour),
			UpdatedAt:  now.Add(-24 * time.Hour),
		},
		{
			DocumentID: "doc-003",
			Title:      "Veterinary Protocols & Emergency Medical Trading Holds",
			Department: "Veterinary Protocols",
			Category:   "Clinical Protocol",
			Author:     "user-veterinary-lead",
			Content:    "# Veterinary Clinical Protocols & Emergency Trading Holds\n\n### Protocol 402: Emergency Trading Halt\nWhen a feline executive exhibits elevated resting heart rates (>180 bpm) or signs of distress, an immediate Emergency Medical Hold must be placed via `/api/v1/workforce/care-schedules/{felineId}/medical-hold`.\n\n- All automated trading triggers are suspended instantaneously.\n- Senior veterinary staff conducts an in-person physical assessment.",
			SharedWith: []string{"group-veterinary", "group-operations"},
			WebDAVPath: "/remote.php/dav/files/admin/Veterinary Protocols/emergency_medical_holds.md",
			Version:    "1.0",
			CreatedAt:  now.Add(-36 * time.Hour),
			UpdatedAt:  now.Add(-6 * time.Hour),
		},
		{
			DocumentID: "doc-004",
			Title:      "Habitat Environmental Monitoring & Climate Control SOP",
			Department: "Facilities & Maintenance SOPs",
			Category:   "SOP",
			Author:     "user-facilities-lead",
			Content:    "# Habitat Climate Control & IoT Sensor Maintenance\n\n### Maintenance Guidelines\n- Cleanroom temperature must be maintained between 21.0C and 23.5C.\n- Humidity levels must stay between 40% and 55%.\n- Calibrate automated feeder load cells bi-weekly.",
			SharedWith: []string{"group-facilities"},
			WebDAVPath: "/remote.php/dav/files/admin/Facilities & Maintenance SOPs/habitat_climate_control_sop.md",
			Version:    "1.2",
			CreatedAt:  now.Add(-96 * time.Hour),
			UpdatedAt:  now.Add(-48 * time.Hour),
		},
		{
			DocumentID: "doc-005",
			Title:      "Executive Strategy & Q3 Capital Allocation Review",
			Department: "Executive Reporting & Finance",
			Category:   "Report",
			Author:     "user-operations-lead",
			Content:    "# Executive Strategy & Q3 Capital Allocation Review\n\n### Financial Highlights\n- Core Invest trading volume increased 34% QoQ.\n- White-label SMB deployment operational margins reached 68%.\n- Feline executive wellness score across all habitats averaged 98.4%.",
			SharedWith: []string{"group-executive"},
			WebDAVPath: "/remote.php/dav/files/admin/Executive Reporting & Finance/q3_executive_review.md",
			Version:    "3.0",
			CreatedAt:  now.Add(-120 * time.Hour),
			UpdatedAt:  now.Add(-72 * time.Hour),
		},
	}
}

// DefaultCalendarEvents provides pre-populated shared calendar events.
func DefaultCalendarEvents() []CalendarEvent {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return []CalendarEvent{
		{
			EventID:     "evt-001",
			Calendar:    "Shift Rotations",
			Title:       "Feline Morning Care & Nutrition Shift",
			Description: "Morning feeding and hydration check for all feline executives in Habitat Alpha.",
			Department:  "Veterinary Protocols",
			Location:    "Habitat Cleanroom Alpha",
			StartTime:   today.Add(8 * time.Hour),
			EndTime:     today.Add(12 * time.Hour),
			Organizer:   "user-veterinary-lead",
			Status:      "confirmed",
		},
		{
			EventID:     "evt-002",
			Calendar:    "Habitat Maintenance Windows",
			Title:       "HVAC Filter Replacement & Sensor Recalibration",
			Description: "Scheduled filter swap and telemetry check for Cleanroom B air handlers.",
			Department:  "Facilities & Maintenance SOPs",
			Location:    "Main Habitat B",
			StartTime:   today.Add(13 * time.Hour),
			EndTime:     today.Add(15*time.Hour + 30*time.Minute),
			Organizer:   "user-facilities-lead",
			Status:      "scheduled",
		},
		{
			EventID:     "evt-003",
			Calendar:    "Corporate Milestones",
			Title:       "OCI All-Hands & Platform Strategy Review",
			Description: "Quarterly company meeting covering multi-persona simulation and White-label release.",
			Department:  "Executive Reporting & Finance",
			Location:    "Virtual Hub Room 1",
			StartTime:   today.Add(16 * time.Hour),
			EndTime:     today.Add(17 * time.Hour),
			Organizer:   "user-operations-lead",
			Status:      "confirmed",
		},
		{
			EventID:     "evt-004",
			Calendar:    "Veterinary Audits",
			Title:       "Senior Executive Wellness & VHR Physical",
			Description: "Comprehensive health audit and blood chemistry review for all senior feline executives.",
			Department:  "Veterinary Protocols",
			Location:    "Veterinary Clinic Bay 2",
			StartTime:   today.Add(24*time.Hour + 10*time.Hour),
			EndTime:     today.Add(24*time.Hour + 11*time.Hour + 30*time.Minute),
			Organizer:   "user-veterinary-lead",
			Status:      "scheduled",
		},
	}
}

// UploadWebDAVFile uploads raw file bytes to Nextcloud WebDAV path via HTTP PUT.
func (c *Client) UploadWebDAVFile(ctx context.Context, webdavPath, contentType string, content []byte) error {
	if webdavPath == "" {
		return errors.New("webdavPath cannot be empty")
	}
	if !strings.HasPrefix(webdavPath, "/") {
		webdavPath = "/" + webdavPath
	}

	url := c.baseURL + webdavPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create WebDAV PUT request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file via WebDAV to %s: %w", webdavPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("nextcloud WebDAV upload to %s returned status %d: %s", webdavPath, resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DownloadWebDAVFile downloads file bytes from Nextcloud WebDAV path via HTTP GET.
func (c *Client) DownloadWebDAVFile(ctx context.Context, webdavPath string) ([]byte, error) {
	if webdavPath == "" {
		return nil, errors.New("webdavPath cannot be empty")
	}
	if !strings.HasPrefix(webdavPath, "/") {
		webdavPath = "/" + webdavPath
	}

	url := c.baseURL + webdavPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebDAV GET request: %w", err)
	}

	if c.adminUser != "" && c.adminPassword != "" {
		req.SetBasicAuth(c.adminUser, c.adminPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file via WebDAV from %s: %w", webdavPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nextcloud WebDAV download from %s returned status %d: %s", webdavPath, resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// ArchiveComplianceReport uploads and archives an executive report into Executive/Reports/YYYY-MM/ via WebDAV.
func (c *Client) ArchiveComplianceReport(ctx context.Context, period, filename, contentType string, content []byte) (string, error) {
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	webdavPath := fmt.Sprintf("/remote.php/dav/files/admin/Executive/Reports/%s/%s", period, filename)
	err := c.UploadWebDAVFile(ctx, webdavPath, contentType, content)
	return webdavPath, err
}
