package simulation

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config holds options for driving the OCI simulation.
type Config struct {
	// EmployeeBFFURL is the base URL for the Employee BFF server (default: http://localhost:8081).
	EmployeeBFFURL string

	// CustomerBFFURL is the base URL for the Customer BFF server (default: http://localhost:8083).
	CustomerBFFURL string

	// BrokerURL is the base URL for the external / mock stock broker API (default: http://localhost:8089).
	BrokerURL string

	// BankURL is the base URL for the external / mock bank API (default: http://localhost:8088).
	BankURL string

	// ForgejoURL is the base URL for the Forgejo Git repository host (default: http://localhost:3000).
	ForgejoURL string

	// Mode determines how simulation actions are executed:
	// "live"       - sends HTTP requests to real running OCI stack services (default).
	// "step"       - runs a single iteration of each persona's actions and prints results.
	// "continuous" - continuously loops persona daily actions until context cancellation.
	Mode string

	// TargetPersona restricts execution to a specific persona ID/name, or "all" for all personas.
	TargetPersona string

	// Iterations specifies how many full daily action loops to perform (0 = infinite in continuous mode).
	Iterations int

	// ActionDelay specifies the pause between persona action executions.
	ActionDelay time.Duration

	// Verbose controls detailed logging of request/response payloads.
	Verbose bool
}

// DefaultConfig returns reasonable defaults for local simulation runs.
func DefaultConfig() *Config {
	return &Config{
		EmployeeBFFURL: "http://localhost:8081",
		CustomerBFFURL: "http://localhost:8083",
		BrokerURL:      "http://localhost:8089",
		BankURL:        "http://localhost:8088",
		ForgejoURL:     "http://localhost:3000",
		Mode:           "live",
		TargetPersona:  "all",
		Iterations:     1,
		ActionDelay:    100 * time.Millisecond,
		Verbose:        false,
	}
}

// ValidateTarget checks that the simulation does not target the production environment (oci-core).
// The simulation suite is strictly restricted to oci-dev (e.g. dev.api.oci.local, oci-dev cluster services,
// or local development/testing endpoints). The production namespace (oci-core) must remain clean.
func (c *Config) ValidateTarget() error {
	targets := []struct {
		name string
		url  string
	}{
		{"EmployeeBFFURL", c.EmployeeBFFURL},
		{"CustomerBFFURL", c.CustomerBFFURL},
		{"BrokerURL", c.BrokerURL},
		{"BankURL", c.BankURL},
		{"ForgejoURL", c.ForgejoURL},
	}

	for _, t := range targets {
		raw := strings.TrimSpace(t.url)
		if raw == "" {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("simulation target %s ('%s') is invalid URL: %w", t.name, t.url, err)
		}
		if parsed.User != nil {
			return fmt.Errorf("simulation target %s ('%s') is rejected: unexpected userinfo in URL", t.name, t.url)
		}
		hostname := strings.ToLower(parsed.Hostname())
		if hostname == "" {
			continue
		}

		// Disallow explicit production domains or namespace references
		isProdNamespace := strings.Contains(hostname, "oci-core") || strings.Contains(hostname, ".production")
		isProdDomain := (strings.Contains(hostname, "api.oci.local") && !strings.Contains(hostname, "dev.api.oci.local")) ||
			(strings.Contains(hostname, "employee-bff.oci.local") && !strings.Contains(hostname, "dev.employee-bff.oci.local")) ||
			(strings.Contains(hostname, "customer-bff.oci.local") && !strings.Contains(hostname, "dev.customer-bff.oci.local")) ||
			(strings.Contains(hostname, "employee.oci.local") && !strings.Contains(hostname, "dev.employee.oci.local")) ||
			(strings.Contains(hostname, "customer.oci.local") && !strings.Contains(hostname, "dev.customer.oci.local"))

		if isProdNamespace || isProdDomain {
			return fmt.Errorf("simulation target %s ('%s') is rejected: simulation is strictly restricted to 'oci-dev' (e.g. https://dev.api.oci.local or local dev/test servers); production 'oci-core' is protected and must remain clean", t.name, t.url)
		}
	}

	return nil
}

// ResolveRoutingURLs determines the effective EmployeeBFFURL, CustomerBFFURL, and ForgejoURL
// based on any explicit service URL overrides.
func ResolveRoutingURLs(bffURL, custBFFURL, forgejoURL string) (string, string, string) {
	resolvedBFF := strings.TrimRight(bffURL, "/")
	resolvedCustBFF := strings.TrimRight(custBFFURL, "/")
	resolvedForgejo := strings.TrimRight(forgejoURL, "/")

	if resolvedBFF == "" {
		resolvedBFF = "http://localhost:8081"
	}
	if resolvedCustBFF == "" {
		resolvedCustBFF = "http://localhost:8083"
	}
	if resolvedForgejo == "" {
		resolvedForgejo = "http://localhost:3000"
	}

	return resolvedBFF, resolvedCustBFF, resolvedForgejo
}

// ResolveExternalURLs resolves the mock broker and mock bank URLs based on explicit flags.
func ResolveExternalURLs(brokerURL, bankURL string) (string, string) {
	resolvedBroker := strings.TrimRight(brokerURL, "/")
	resolvedBank := strings.TrimRight(bankURL, "/")

	if resolvedBroker == "" {
		resolvedBroker = "http://localhost:8089"
	}
	if resolvedBank == "" {
		resolvedBank = "http://localhost:8088"
	}

	return resolvedBroker, resolvedBank
}
