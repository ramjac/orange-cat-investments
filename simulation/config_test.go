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

func TestResolveRoutingURLs(t *testing.T) {
	// Case 1: Empty inputs return standard localhost defaults
	srv, bff, cust, forgejo := ResolveRoutingURLs("", "", "", "", "")
	assert.Equal(t, "http://localhost:8080", srv)
	assert.Equal(t, "http://localhost:8081", bff)
	assert.Equal(t, "http://localhost:8083", cust)
	assert.Equal(t, "http://localhost:3000", forgejo)

	// Case 2: Unified URL pointing to staging oci-dev ingress
	srv, bff, cust, forgejo = ResolveRoutingURLs("https://dev.api.oci.local", "", "", "", "")
	assert.Equal(t, "https://dev.api.oci.local", srv)
	assert.Equal(t, "https://dev.employee-bff.oci.local", bff)
	assert.Equal(t, "https://dev.customer-bff.oci.local", cust)
	assert.Equal(t, "https://forgejo.oci.local", forgejo)

	// Case 3: Unified URL pointing to custom dev ingress
	srv, bff, cust, forgejo = ResolveRoutingURLs("https://dev.api.internal.local", "", "", "", "")
	assert.Equal(t, "https://dev.api.internal.local", srv)
	assert.Equal(t, "https://dev.employee-bff.internal.local", bff)
	assert.Equal(t, "https://dev.customer-bff.internal.local", cust)
	assert.Equal(t, "https://forgejo.internal.local", forgejo)

	// Case 4: Unified URL pointing to localhost:8080 port scheme
	srv, bff, cust, forgejo = ResolveRoutingURLs("http://localhost:8080", "", "", "", "")
	assert.Equal(t, "http://localhost:8080", srv)
	assert.Equal(t, "http://localhost:8081", bff)
	assert.Equal(t, "http://localhost:8083", cust)
	assert.Equal(t, "http://localhost:3000", forgejo)

	// Case 5: Explicit flags override unified URL routing
	srv, bff, cust, forgejo = ResolveRoutingURLs("https://dev.api.oci.local", "https://override.api.oci.local", "https://override.bff.oci.local", "https://override.cust.oci.local", "https://override.forgejo.oci.local")
	assert.Equal(t, "https://override.api.oci.local", srv)
	assert.Equal(t, "https://override.bff.oci.local", bff)
	assert.Equal(t, "https://override.cust.oci.local", cust)
	assert.Equal(t, "https://override.forgejo.oci.local", forgejo)
}

func TestConfigValidateTarget(t *testing.T) {
	// Mock mode always succeeds regardless of URL strings
	mockCfg := &Config{
		Mode:      "mock",
		ServerURL: "https://api.oci.local",
	}
	assert.NoError(t, mockCfg.ValidateTarget(), "mock mode should not fail target validation")

	// Live mode targeting staging dev.api.oci.local succeeds
	devCfg := &Config{
		Mode:           "live",
		ServerURL:      "https://dev.api.oci.local",
		EmployeeBFFURL: "https://dev.employee-bff.oci.local",
		CustomerBFFURL: "https://dev.customer-bff.oci.local",
	}
	assert.NoError(t, devCfg.ValidateTarget(), "live mode targeting dev.api.oci.local should pass")

	// Live mode targeting localhost succeeds
	localCfg := &Config{
		Mode:           "live",
		ServerURL:      "http://localhost:8080",
		EmployeeBFFURL: "http://localhost:8081",
		CustomerBFFURL: "http://localhost:8083",
	}
	assert.NoError(t, localCfg.ValidateTarget(), "live mode targeting localhost should pass")

	// Live mode targeting production api.oci.local must be blocked
	prodCfg := &Config{
		Mode:           "live",
		ServerURL:      "https://api.oci.local",
		EmployeeBFFURL: "https://employee-bff.oci.local",
		CustomerBFFURL: "https://customer-bff.oci.local",
	}
	err := prodCfg.ValidateTarget()
	assert.Error(t, err, "live mode targeting production api.oci.local must be blocked")
	assert.Contains(t, err.Error(), "strictly restricted to 'oci-dev'")

	// Live mode targeting oci-core internal cluster service must be blocked
	prodK8sCfg := &Config{
		Mode:           "live",
		ServerURL:      "http://oci-server-service.oci-core.svc.cluster.local:8080",
		EmployeeBFFURL: "http://oci-employee-bff-service.oci-core.svc.cluster.local:8081",
	}
	err = prodK8sCfg.ValidateTarget()
	assert.Error(t, err, "live mode targeting oci-core namespace service must be blocked")
	assert.Contains(t, err.Error(), "strictly restricted to 'oci-dev'")

	// Verify NewDriver blocks invalid targets
	_, err = NewDriver(prodCfg)
	assert.Error(t, err, "NewDriver must fail if target is production")
}

