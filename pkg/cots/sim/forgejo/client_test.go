package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimForgejoClient_CreateUser_DefaultPassword(t *testing.T) {
	var capturedPasswords []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var opt CreateUserOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)

		capturedPasswords = append(capturedPasswords, opt.Password)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(User{
			ID:       100,
			UserName: opt.Username,
			Email:    opt.Email,
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	u1, err1 := client.CreateUser(context.Background(), CreateUserOption{
		Username: "user1",
		Email:    "user1@oci.local",
	})
	require.NoError(t, err1)
	assert.Equal(t, int64(100), u1.ID)

	u2, err2 := client.CreateUser(context.Background(), CreateUserOption{
		Username: "user2",
		Email:    "user2@oci.local",
	})
	require.NoError(t, err2)
	assert.Equal(t, int64(100), u2.ID)

	require.Len(t, capturedPasswords, 2)
	assert.True(t, strings.HasPrefix(capturedPasswords[0], "TempPass!"))
	assert.True(t, strings.HasPrefix(capturedPasswords[1], "TempPass!"))
}

func TestSimForgejoClient_OfflineFallback(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	u, err := client.CreateUser(context.Background(), CreateUserOption{
		Username: "offline.dev",
		Email:    "offline@oci.local",
	})
	require.NoError(t, err)
	assert.Equal(t, "offline.dev", u.UserName)
	assert.True(t, u.Active)
}
