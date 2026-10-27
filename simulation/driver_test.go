package simulation

import (
	"context"
	"net/http"
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
