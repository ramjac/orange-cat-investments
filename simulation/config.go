package simulation

import (
	"time"
)

// Config holds options for driving the OCI simulation.
type Config struct {
	// ServerURL is the base URL for the OCI Domain API server (default: http://localhost:8080).
	ServerURL string

	// EmployeeBFFURL is the base URL for the Employee BFF server (default: http://localhost:8081).
	EmployeeBFFURL string

	// Mode determines how simulation actions are executed:
	// "mock"       - runs against embedded mock server handlers without external dependencies.
	// "live"       - sends HTTP requests to real running OCI stack services.
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
		Mode:           "mock",
		TargetPersona:  "all",
		Iterations:     1,
		ActionDelay:    100 * time.Millisecond,
		Verbose:        false,
	}
}
