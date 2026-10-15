package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoClient_CreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/admin/users", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "token secret-token", r.Header.Get("Authorization"))

		var opt CreateUserOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.Equal(t, "elise.dev", opt.Username)
		assert.Equal(t, "elise@oci.local", opt.Email)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(User{
			ID:        42,
			UserName:  opt.Username,
			Email:     opt.Email,
			FullName:  "Elise Dev",
			Active:    true,
			CreatedAt: time.Now().UTC(),
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	user, err := client.CreateUser(context.Background(), CreateUserOption{
		Username: "elise.dev",
		Email:    "elise@oci.local",
		FullName: "Elise Dev",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "elise.dev", user.UserName)
}

func TestForgejoClient_AddUserPublicKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/admin/users/elise.dev/keys", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var opt CreateKeyOption
		err := json.NewDecoder(r.Body).Decode(&opt)
		require.NoError(t, err)
		assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...", opt.Key)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PublicKey{
			ID:        101,
			Title:     opt.Title,
			Key:       opt.Key,
			ReadOnly:  false,
			CreatedAt: time.Now().UTC(),
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	key, err := client.AddUserPublicKey(context.Background(), "elise.dev", CreateKeyOption{
		Title: "MacBook Pro Key",
		Key:   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(101), key.ID)
	assert.Equal(t, "MacBook Pro Key", key.Title)
}

func TestForgejoClient_DeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/admin/users/separated.user", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "secret-token",
	})

	err := client.DeleteUser(context.Background(), "separated.user")
	assert.NoError(t, err)
}
