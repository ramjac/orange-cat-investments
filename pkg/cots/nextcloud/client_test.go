package nextcloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
		DisplayName: "Rick Newhire",
		Email:       "rick.newhire@oci.local",
		Quota:       "50 GB",
		Groups:      []string{"operations", "workforce"},
	})
	require.NoError(t, err)
	assert.Equal(t, "rick.newhire", user.ID)
	assert.Equal(t, "50 GB", user.Quota)
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
