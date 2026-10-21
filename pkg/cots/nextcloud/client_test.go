package nextcloud

import (
	"context"
	"net/http"
	"net/http/httptest"
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
		Password:    "SecurePass123!",
		DisplayName: "Rick Newhire",
		Email:       "rick.newhire@oci.local",
		Quota:       "50 GB",
		Groups:      []string{"operations", "workforce"},
	})
	require.NoError(t, err)
	assert.Equal(t, "rick.newhire", user.ID)
	assert.Equal(t, "50 GB", user.Quota)
}

func TestNextcloudClient_CreateUser_EmptyPassword(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://localhost"})

	_, err := client.CreateUser(context.Background(), CreateUserOption{
		UserID: "user1",
	})
	require.Error(t, err)
	assert.Equal(t, "password cannot be empty", err.Error())
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
