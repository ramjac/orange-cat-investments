package simulation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg, "DefaultConfig() should not return nil")

	assert.Equal(t, "http://localhost:8080", cfg.ServerURL)
	assert.Equal(t, "http://localhost:8081", cfg.EmployeeBFFURL)
	assert.Equal(t, "http://localhost:8083", cfg.CustomerBFFURL)
	assert.Equal(t, "http://localhost:8089", cfg.BrokerURL)
	assert.Equal(t, "http://localhost:8088", cfg.BankURL)
	assert.Equal(t, "mock", cfg.Mode)
	assert.Equal(t, "all", cfg.TargetPersona)
	assert.Equal(t, 1, cfg.Iterations)
	assert.Equal(t, 100*time.Millisecond, cfg.ActionDelay)
	assert.False(t, cfg.Verbose)
}
