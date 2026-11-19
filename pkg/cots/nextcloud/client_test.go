package nextcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextcloudClient_CreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ocs/v1.php/cloud/users", r.URL.Path)
		assert.Equal(t, "true", r.Header.Get("OCS-APIRequest"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "rick.newhire", r.FormValue("userid"))
		assert.Equal(t, "50 GB", r.FormValue("quota"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
	})

	user, err := client.CreateUser(context.Background(), CreateUserOption{
		UserID:      "rick.newhire",
		Password:    "SecretDevPass123!",
		DisplayName: "Rick Newhire",
		Email:       "rick.newhire@oci.local",
		Quota:       "50 GB",
		Groups:      []string{"operations", "workforce"},
	})
	require.NoError(t, err)
	assert.Equal(t, "rick.newhire", user.ID)
	assert.Equal(t, "50 GB", user.Quota)
}

func TestNextcloudClient_CreateUser_PasswordRequired(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://dummy.local"})
	_, err := client.CreateUser(context.Background(), CreateUserOption{
		UserID: "test.user",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password cannot be empty")
}

func TestNextcloudClient_DisableUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ocs/v1.php/cloud/users/separated.user/disable", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
	})

	err := client.DisableUser(context.Background(), "separated.user")
	assert.NoError(t, err)
}

func TestNextcloudClient_LockAndArchiveUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
	})

	err := client.LockAndArchiveUser(context.Background(), "departing.user", "/vault/archives/departing.user.tar.gz")
	assert.NoError(t, err)
}

func TestNextcloudClient_ListAndCreateGroupFolders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/apps/groupfolders/folders" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"folders":[{"id":1,"mount_point":"HR & Policies","quota":"50 GB","groups":["hr"]}]}`))
		case r.URL.Path == "/apps/groupfolders/folders" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":2}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	folders, err := client.ListGroupFolders(context.Background())
	require.NoError(t, err)
	require.Len(t, folders, 1)
	assert.Equal(t, "HR & Policies", folders[0].MountPoint)

	created, err := client.CreateGroupFolder(context.Background(), "Veterinary Protocols", []string{"veterinary"}, "50 GB")
	require.NoError(t, err)
	assert.Equal(t, "Veterinary Protocols", created.MountPoint)
	assert.Equal(t, "50 GB", created.Quota)
}

func TestNextcloudClient_ListAndGetDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/nextcloud/api/v1/documents" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"documents":[{"document_id":"doc-001","title":"OCI Onboarding Handbook","department":"HR & Policies","author":"emp-human-alice"}]}`))
		case r.URL.Path == "/nextcloud/api/v1/documents/doc-001" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"document_id":"doc-001","title":"OCI Onboarding Handbook","department":"HR & Policies","author":"emp-human-alice","content":"Welcome to OCI!"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	docs, err := client.ListDocuments(context.Background(), "HR & Policies", "", "")
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "doc-001", docs[0].DocumentID)

	doc, err := client.GetDocument(context.Background(), "doc-001")
	require.NoError(t, err)
	assert.Equal(t, "OCI Onboarding Handbook", doc.Title)
	assert.Equal(t, "Welcome to OCI!", doc.Content)
}

func TestNextcloudClient_PublishDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/nextcloud/api/v1/documents", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"document_id":"doc-999","title":"Zero-Trust Standard","department":"Facilities & Maintenance SOPs","author":"emp-human-frank"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	doc, err := client.PublishDocument(context.Background(), Document{
		Title:      "Zero-Trust Standard",
		Department: "Facilities & Maintenance SOPs",
		Author:     "emp-human-frank",
	})
	require.NoError(t, err)
	assert.Equal(t, "doc-999", doc.DocumentID)
	assert.Contains(t, doc.WebDAVPath, "Facilities & Maintenance SOPs")
}

func TestNextcloudClient_CalendarEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/nextcloud/api/v1/calendar/events" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"events":[{"event_id":"evt-1","calendar":"Shift Rotations","title":"Morning Care","department":"Veterinary Protocols"}]}`))
		case r.URL.Path == "/nextcloud/api/v1/calendar/events" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"event_id":"evt-2","calendar":"Habitat Maintenance Windows","title":"HVAC Swap"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	events, err := client.ListCalendarEvents(context.Background(), "Shift Rotations", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Morning Care", events[0].Title)

	created, err := client.CreateCalendarEvent(context.Background(), CalendarEvent{
		Calendar: "Habitat Maintenance Windows",
		Title:    "HVAC Swap",
	})
	require.NoError(t, err)
	assert.Equal(t, "evt-2", created.EventID)
}

func TestNextcloudClient_Defaults(t *testing.T) {
	folders := DefaultGroupFolders()
	assert.Len(t, folders, 4)

	docs := DefaultDocuments()
	assert.True(t, len(docs) >= 5)

	events := DefaultCalendarEvents()
	assert.True(t, len(events) >= 4)
}

func TestNextcloudClient_UploadWebDAVAndArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Contains(t, r.URL.Path, "/remote.php/dav/files/admin/Executive/Reports/2026-10/")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
	})

	path, err := client.ArchiveComplianceReport(context.Background(), "2026-10", "executive_summary_report_2026-10.pdf", "application/pdf", []byte("%PDF-1.4 mock"))
	require.NoError(t, err)
	assert.Equal(t, "/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_report_2026-10.pdf", path)
}

func TestNextcloudClient_DownloadWebDAV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/remote.php/dav/files/admin/Executive/Reports/2026-10/report.pdf", r.URL.Path)
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4 test stream"))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
	})

	content, err := client.DownloadWebDAVFile(context.Background(), "/remote.php/dav/files/admin/Executive/Reports/2026-10/report.pdf")
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-1.4 test stream"), content)
}

func TestNextcloudClient_ListDocuments_CapsAtMaxLimit(t *testing.T) {
	docs := make([]Document, 150)
	for i := range docs {
		docs[i] = Document{
			DocumentID: fmt.Sprintf("doc-%03d", i),
			Title:      fmt.Sprintf("Policy %03d", i),
			Department: "HR & Policies",
		}
	}
	payload, err := json.Marshal(map[string]any{"documents": docs})
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, AdminUser: "admin", AdminPassword: "secret"})
	res, err := client.ListDocuments(context.Background(), "HR & Policies", "", "")
	require.NoError(t, err)
	assert.Len(t, res, maxDocumentsLimit)
}

func TestNextcloudClient_LiveNextcloudHTMLFallbackAndWebDAVMKCOL(t *testing.T) {
	var mu sync.Mutex
	createdDirs := map[string]bool{}
	uploadedFiles := map[string]string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		require.True(t, ok, "basic auth required")
		assert.Equal(t, "admin", user)
		assert.Equal(t, "secret", pass)

		switch {
		case strings.HasPrefix(r.URL.Path, "/nextcloud/api/v1/") || strings.HasPrefix(r.URL.Path, "/apps/groupfolders/"):
			// Simulate vanilla Nextcloud 28 Apache redirecting unknown routes to HTML login/dashboard
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Nextcloud</body></html>"))
		case r.Method == "MKCOL":
			mu.Lock()
			createdDirs[r.URL.Path] = true
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/remote.php/dav/files/admin/"):
			parent := r.URL.Path[:strings.LastIndex(r.URL.Path, "/")]
			mu.Lock()
			exists := createdDirs[parent]
			mu.Unlock()
			if !exists {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte("<d:error>Parent node does not exist</d:error>"))
				return
			}
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			uploadedFiles[r.URL.Path] = string(body)
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AdminUser:     "admin",
		AdminPassword: "secret",
	})

	// 1. PublishDocument should fall back to WebDAV MKCOL + PUT and record metadata
	doc, err := client.PublishDocument(context.Background(), Document{
		Title:   "ADR 0003: Self-Hosted GitOps, Multi-Stage Packaging & Gated Promotion Pipeline",
		Content: "# ADR 0003\nGitOps specification.",
		Author:  "emp-human-marcus",
	})
	require.NoError(t, err)
	assert.Contains(t, doc.WebDAVPath, "/remote.php/dav/files/admin/Documents/adr_0003_")

	mu.Lock()
	assert.True(t, createdDirs["/remote.php/dav/files/admin/Documents"])
	assert.Equal(t, "# ADR 0003\nGitOps specification.", uploadedFiles[doc.WebDAVPath])
	mu.Unlock()

	// 2. ListDocuments should return seeded + published docs without failing on HTML response
	docs, err := client.ListDocuments(context.Background(), "", "", "ADR 0003")
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, doc.DocumentID, docs[0].DocumentID)

	// 3. ListGroupFolders and ListCalendarEvents should gracefully return defaults on HTML response
	folders, err := client.ListGroupFolders(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, folders)

	events, err := client.ListCalendarEvents(context.Background(), "Habitat Maintenance Windows", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, events)

	// 4. ArchiveComplianceReport should create nested Executive/Reports/<period> dirs via MKCOL and PUT
	archivePath, err := client.ArchiveComplianceReport(context.Background(), "2026-11", "report.pdf", "application/pdf", []byte("%PDF-1.4"))
	require.NoError(t, err)
	assert.Equal(t, "/remote.php/dav/files/admin/Executive/Reports/2026-11/report.pdf", archivePath)

	mu.Lock()
	assert.True(t, createdDirs["/remote.php/dav/files/admin/Executive"])
	assert.True(t, createdDirs["/remote.php/dav/files/admin/Executive/Reports"])
	assert.True(t, createdDirs["/remote.php/dav/files/admin/Executive/Reports/2026-11"])
	assert.Equal(t, "%PDF-1.4", uploadedFiles[archivePath])
	mu.Unlock()
}
