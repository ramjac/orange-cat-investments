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

	assert.Equal(t, "http://localhost:8081", cfg.EmployeeBFFURL)
	assert.Equal(t, "http://localhost:8083", cfg.CustomerBFFURL)
	assert.Equal(t, "http://localhost:8089", cfg.BrokerURL)
	assert.Equal(t, "http://localhost:8088", cfg.BankURL)
	assert.Equal(t, "live", cfg.Mode)
	assert.Equal(t, "all", cfg.TargetPersona)
	assert.Equal(t, 1, cfg.Iterations)
	assert.Equal(t, 100*time.Millisecond, cfg.ActionDelay)
	assert.False(t, cfg.Verbose)
}

func TestResolveRoutingURLs(t *testing.T) {
	// Case 1: Empty inputs return standard localhost defaults
	bff, cust, forgejo := ResolveRoutingURLs("", "", "")
	assert.Equal(t, "http://localhost:8081", bff)
	assert.Equal(t, "http://localhost:8083", cust)
	assert.Equal(t, "http://localhost:3000", forgejo)

	// Case 2: Explicit flags are used
	bff, cust, forgejo = ResolveRoutingURLs("https://override.bff.oci.local", "https://override.cust.oci.local", "https://override.forgejo.oci.local")
	assert.Equal(t, "https://override.bff.oci.local", bff)
	assert.Equal(t, "https://override.cust.oci.local", cust)
	assert.Equal(t, "https://override.forgejo.oci.local", forgejo)
}

func TestConfigValidateTarget(t *testing.T) {
	// Live mode targeting staging dev.employee-bff.oci.local succeeds
	devCfg := &Config{
		Mode:           "live",
		EmployeeBFFURL: "https://dev.employee-bff.oci.local",
		CustomerBFFURL: "https://dev.customer-bff.oci.local",
	}
	assert.NoError(t, devCfg.ValidateTarget(), "live mode targeting dev.employee-bff.oci.local should pass")

	// Live mode targeting localhost succeeds
	localCfg := &Config{
		Mode:           "live",
		EmployeeBFFURL: "http://localhost:8081",
		CustomerBFFURL: "http://localhost:8083",
	}
	assert.NoError(t, localCfg.ValidateTarget(), "live mode targeting localhost should pass")

	// Live mode targeting production api.oci.local must be blocked
	prodCfg := &Config{
		Mode:           "live",
		EmployeeBFFURL: "https://employee-bff.oci.local",
		CustomerBFFURL: "https://customer-bff.oci.local",
	}
	err := prodCfg.ValidateTarget()
	assert.Error(t, err, "live mode targeting production api.oci.local must be blocked")
	assert.Contains(t, err.Error(), "strictly restricted to 'oci-dev'")

	// Live mode targeting oci-core internal cluster service must be blocked
	prodK8sCfg := &Config{
		Mode:           "live",
		EmployeeBFFURL: "http://oci-employee-bff-service.oci-core.svc.cluster.local:8081",
	}
	err = prodK8sCfg.ValidateTarget()
	assert.Error(t, err, "live mode targeting oci-core namespace service must be blocked")
	assert.Contains(t, err.Error(), "strictly restricted to 'oci-dev'")

	// Verify NewDriver blocks invalid targets
	_, err = NewDriver(prodCfg)
	assert.Error(t, err, "NewDriver must fail if target is production")
}
