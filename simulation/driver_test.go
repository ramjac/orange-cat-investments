package simulation

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

func TestRegistryPersonas(t *testing.T) {
	registry := NewRegistry()
	personas := registry.List()
	assert.Len(t, personas, 13)

	expectedPersonas := []string{
		"emp-feline-garfield",
		"emp-feline-barneby",
		"emp-human-alice",
		"emp-human-bob",
		"emp-human-carol",
		"emp-human-david",
		"emp-human-elena",
		"emp-human-frank",
		"cust-longterm-arthur",
		"cust-active-chloe",
		"emp-human-rick",
		"emp-human-elise",
		"emp-human-marcus",
	}

	for _, id := range expectedPersonas {
		p, ok := registry.Get(id)
		require.True(t, ok, "persona %s should exist in registry", id)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.RoleTitle)
		assert.NotEmpty(t, p.Department)
		assert.NotEmpty(t, p.Actions)
	}
}

func TestDriverLiveExecutionAgainstRunningStack(t *testing.T) {
	cfg := DefaultConfig()

	// Probe whether the live OCI stack is reachable
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(cfg.EmployeeBFFURL + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skipf("skipping live simulation test: running OCI stack not reachable at %s", cfg.EmployeeBFFURL)
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)
	assert.Greater(t, summary.TotalActions, 0)
	assert.Equal(t, 0, summary.Failures, "all actions must succeed on a healthy running stack")
}

func TestDriver_MidIterationReauth(t *testing.T) {
	callCount := 0
	authCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/customer/auth/login", "/api/v1/employee/auth/login":
			authCount++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":     "authenticated",
				"csrf_token": "csrf-tok-123",
			})
		case "/api/v1/workforce/care-schedules/emp-feline-garfield":
			callCount++
			if callCount == 1 {
				// Simulate session expiration / mid-iteration 401
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: invalid or expired customer session"})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := &Config{
		EmployeeBFFURL: server.URL,
		CustomerBFFURL: server.URL,
		TargetPersona:  "emp-feline-garfield",
		Iterations:     1,
		Mode:           "step",
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	summary, err := driver.Run(context.Background())
	require.NoError(t, err)

	// Mid-iteration 401 should trigger re-authentication and retry
	assert.Greater(t, authCount, 1, "AuthenticatePersona should have been triggered upon 401")
	assert.Greater(t, summary.Successes, 0)
}
