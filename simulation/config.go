package simulation

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config holds options for driving the OCI simulation.
type Config struct {
	// ServerURL is the base URL for the OCI Domain API server (default: http://localhost:8080).
	ServerURL string

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
		ServerURL:      "http://localhost:8080",
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
		{"ServerURL", c.ServerURL},
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

// ResolveRoutingURLs determines the effective ServerURL, EmployeeBFFURL, CustomerBFFURL, and ForgejoURL
// based on a unified URL flag (e.g., https://dev.api.oci.local or http://localhost:8080)
// and any explicit service URL overrides.
func ResolveRoutingURLs(unifiedURL, serverURL, bffURL, custBFFURL, forgejoURL string) (string, string, string, string) {
	resolvedServer := strings.TrimRight(serverURL, "/")
	resolvedBFF := strings.TrimRight(bffURL, "/")
	resolvedCustBFF := strings.TrimRight(custBFFURL, "/")
	resolvedForgejo := strings.TrimRight(forgejoURL, "/")
	uURL := strings.TrimRight(strings.TrimSpace(unifiedURL), "/")

	if uURL != "" {
		if resolvedServer == "" {
			resolvedServer = uURL
		}
		if resolvedBFF == "" {
			if strings.Contains(uURL, "dev.api.oci.local") {
				resolvedBFF = strings.Replace(uURL, "dev.api.oci.local", "dev.employee-bff.oci.local", 1)
			} else if strings.Contains(uURL, ":8080") {
				resolvedBFF = strings.Replace(uURL, ":8080", ":8081", 1)
			} else if strings.Contains(uURL, "dev.api.") {
				resolvedBFF = strings.Replace(uURL, "dev.api.", "dev.employee-bff.", 1)
			} else {
				resolvedBFF = uURL
			}
		}
		if resolvedCustBFF == "" {
			if strings.Contains(uURL, "dev.api.oci.local") {
				resolvedCustBFF = strings.Replace(uURL, "dev.api.oci.local", "dev.customer-bff.oci.local", 1)
			} else if strings.Contains(uURL, ":8080") {
				resolvedCustBFF = strings.Replace(uURL, ":8080", ":8083", 1)
			} else if strings.Contains(uURL, "dev.api.") {
				resolvedCustBFF = strings.Replace(uURL, "dev.api.", "dev.customer-bff.", 1)
			} else {
				resolvedCustBFF = uURL
			}
		}
		if resolvedForgejo == "" {
			if strings.Contains(uURL, "dev.api.oci.local") {
				if strings.HasPrefix(uURL, "http://") {
					resolvedForgejo = "http://forgejo.oci.local"
				} else {
					resolvedForgejo = "https://forgejo.oci.local"
				}
			} else if strings.Contains(uURL, ":8080") {
				resolvedForgejo = strings.Replace(uURL, ":8080", ":3000", 1)
			} else if strings.Contains(uURL, "dev.api.") {
				resolvedForgejo = strings.Replace(uURL, "dev.api.", "forgejo.", 1)
			} else {
				resolvedForgejo = "http://localhost:3000"
			}
		}
	}

	if resolvedServer == "" {
		resolvedServer = "http://localhost:8080"
	}
	if resolvedBFF == "" {
		resolvedBFF = "http://localhost:8081"
	}
	if resolvedCustBFF == "" {
		resolvedCustBFF = "http://localhost:8083"
	}
	if resolvedForgejo == "" {
		resolvedForgejo = "http://localhost:3000"
	}

	return resolvedServer, resolvedBFF, resolvedCustBFF, resolvedForgejo
}

// ResolveExternalURLs resolves the mock broker and mock bank URLs based on unifiedURL and explicit flags.
func ResolveExternalURLs(unifiedURL, brokerURL, bankURL string) (string, string) {
	resolvedBroker := strings.TrimRight(brokerURL, "/")
	resolvedBank := strings.TrimRight(bankURL, "/")
	uURL := strings.TrimRight(strings.TrimSpace(unifiedURL), "/")

	if uURL != "" {
		if resolvedBroker == "" {
			if strings.Contains(uURL, "dev.api.oci.local") {
				if strings.HasPrefix(uURL, "http://") {
					resolvedBroker = "http://broker-api.sim.oci.local"
				} else {
					resolvedBroker = "https://broker-api.sim.oci.local"
				}
			} else if strings.Contains(uURL, ":8080") {
				resolvedBroker = strings.Replace(uURL, ":8080", ":8089", 1)
			}
		}
		if resolvedBank == "" {
			if strings.Contains(uURL, "dev.api.oci.local") {
				if strings.HasPrefix(uURL, "http://") {
					resolvedBank = "http://plaid-api.sim.oci.local"
				} else {
					resolvedBank = "https://plaid-api.sim.oci.local"
				}
			} else if strings.Contains(uURL, ":8080") {
				resolvedBank = strings.Replace(uURL, ":8080", ":8088", 1)
			}
		}
	}

	if resolvedBroker == "" {
		resolvedBroker = "http://localhost:8089"
	}
	if resolvedBank == "" {
		resolvedBank = "http://localhost:8088"
	}

	return resolvedBroker, resolvedBank
}
