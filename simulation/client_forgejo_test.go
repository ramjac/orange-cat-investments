package simulation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoForgejoAs_TokensFromEnv(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		ForgejoURL: server.URL,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// 1. When no env vars are set, no Authorization header should be set
	os.Unsetenv("FORGEJO_TOKEN")
	os.Unsetenv("FORGEJO_ELISE_TOKEN")
	os.Unsetenv("FORGEJO_MARCUS_TOKEN")

	capturedAuth = ""
	_, _, err = client.DoForgejoAs(ctx, "emp-human-marcus", "GET", "/api/v1/user", nil)
	require.NoError(t, err)
	assert.Equal(t, "", capturedAuth, "expected no Authorization header when env vars are unset")

	capturedAuth = ""
	_, _, err = client.DoForgejoAs(ctx, "emp-human-elise", "GET", "/api/v1/user", nil)
	require.NoError(t, err)
	assert.Equal(t, "", capturedAuth, "expected no Authorization header when env vars are unset")

	capturedAuth = ""
	_, _, err = client.DoForgejoAs(ctx, "generic-user", "GET", "/api/v1/user", nil)
	require.NoError(t, err)
	assert.Equal(t, "", capturedAuth, "expected no Authorization header when env vars are unset")

	// 2. When user-specific token is set
	t.Setenv("FORGEJO_MARCUS_TOKEN", "marcus-secret-token-123")
	capturedAuth = ""
	_, _, err = client.DoForgejoAs(ctx, "emp-human-marcus", "GET", "/api/v1/user", nil)
	require.NoError(t, err)
	assert.Equal(t, "token marcus-secret-token-123", capturedAuth)

	t.Setenv("FORGEJO_ELISE_TOKEN", "elise-secret-token-456")
	capturedAuth = ""
	_, _, err = client.DoForgejoAs(ctx, "emp-human-elise", "GET", "/api/v1/user", nil)
	require.NoError(t, err)
	assert.Equal(t, "token elise-secret-token-456", capturedAuth)

	// 3. Fallback to default FORGEJO_TOKEN when generic user or user token not specified
	t.Setenv("FORGEJO_TOKEN", "default-forgejo-token-789")
	capturedAuth = ""
	_, _, err = client.DoForgejoAs(ctx, "generic-user", "GET", "/api/v1/user", nil)
	require.NoError(t, err)
	assert.Equal(t, "token default-forgejo-token-789", capturedAuth)
}
