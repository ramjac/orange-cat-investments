package nextcloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimNextcloudClient_CreateUser_DefaultPassword(t *testing.T) {
	var capturedPasswords []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)
		capturedPasswords = append(capturedPasswords, r.FormValue("password"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})

	u1, err1 := client.CreateUser(context.Background(), CreateUserOption{UserID: "sim_user1"})
	require.NoError(t, err1)
	assert.Equal(t, "sim_user1", u1.ID)

	u2, err2 := client.CreateUser(context.Background(), CreateUserOption{UserID: "sim_user2"})
	require.NoError(t, err2)
	assert.Equal(t, "sim_user2", u2.ID)

	require.Len(t, capturedPasswords, 2)
	assert.True(t, strings.HasPrefix(capturedPasswords[0], "NextCloud!"), "simulation client generates NextCloud! password prefix")
	assert.True(t, strings.HasPrefix(capturedPasswords[1], "NextCloud!"), "simulation client generates NextCloud! password prefix")
}

func TestSimNextcloudClient_OfflineFallbacks(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	ctx := context.Background()

	// 1. CreateUser offline fallback
	u, err := client.CreateUser(ctx, CreateUserOption{UserID: "offline_user", DisplayName: "Offline User"})
	require.NoError(t, err)
	assert.Equal(t, "offline_user", u.ID)
	assert.Equal(t, "25 GB", u.Quota)

	// 2. ListGroupFolders fallback
	folders, err := client.ListGroupFolders(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, folders)

	// 3. ListDocuments fallback
	docs, err := client.ListDocuments(ctx, "", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// 4. GetDocument fallback
	doc, err := client.GetDocument(ctx, "doc-001")
	require.NoError(t, err)
	assert.Equal(t, "doc-001", doc.DocumentID)

	// 5. ListCalendarEvents fallback
	events, err := client.ListCalendarEvents(ctx, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, events)
}
