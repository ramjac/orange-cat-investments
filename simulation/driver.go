package simulation

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Driver manages and executes simulation scenarios using employee personas.
type Driver struct {
	cfg      *Config
	client   *Client
	registry *Registry
	logger   *slog.Logger
}

// Summary stores the aggregate results of a simulation execution run.
type Summary struct {
	TotalActions int            `json:"total_actions"`
	Successes    int            `json:"successes"`
	Failures     int            `json:"failures"`
	Duration     time.Duration  `json:"duration"`
	Results      []ActionResult `json:"results"`
}

// NewDriver constructs a simulation Driver using the provided Config.
func NewDriver(cfg *Config) (*Driver, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize simulation client: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	return &Driver{
		cfg:      cfg,
		client:   client,
		registry: NewRegistry(),
		logger:   logger,
	}, nil
}

// Close closes any open client connections and test servers.
func (d *Driver) Close() {
	if d.client != nil {
		d.client.Close()
	}
}

// Registry returns the underlying persona registry.
func (d *Driver) Registry() *Registry {
	return d.registry
}

// Run executes the simulation based on the configured mode and target persona.
func (d *Driver) Run(ctx context.Context) (*Summary, error) {
	startTime := time.Now()
	summary := &Summary{
		Results: make([]ActionResult, 0),
	}

	personasToRun := d.getMatchingPersonas()
	if len(personasToRun) == 0 {
		return summary, fmt.Errorf("no matching personas found for target: %s", d.cfg.TargetPersona)
	}

	d.logger.Info("starting OCI stack simulation drive",
		"mode", d.cfg.Mode,
		"target_persona", d.cfg.TargetPersona,
		"matching_personas", len(personasToRun),
		"iterations", d.cfg.Iterations,
	)

	currentIteration := 0
	for {
		select {
		case <-ctx.Done():
			summary.Duration = time.Since(startTime)
			return summary, ctx.Err()
		default:
		}

		currentIteration++
		d.logger.Info("executing simulation iteration", "iteration", currentIteration)

		for _, p := range personasToRun {
			d.logger.Info("simulating persona day actions", "persona_id", p.ID, "persona_name", p.Name, "role", p.RoleTitle)

			for _, act := range p.Actions {
				select {
				case <-ctx.Done():
					summary.Duration = time.Since(startTime)
					return summary, ctx.Err()
				default:
				}

				res, err := act.Execute(ctx, d.client)
				if err != nil {
					res = &ActionResult{
						PersonaID:   p.ID,
						PersonaName: p.Name,
						ActionName:  act.Name,
						Success:     false,
						Details:     fmt.Sprintf("Execution error: %v", err),
						Timestamp:   time.Now().UTC(),
					}
				}

				summary.TotalActions++
				if res.Success {
					summary.Successes++
				} else {
					summary.Failures++
				}
				summary.Results = append(summary.Results, *res)

				if d.cfg.Verbose {
					d.logger.Info("persona action completed",
						"persona", res.PersonaName,
						"action", res.ActionName,
						"success", res.Success,
						"status", res.StatusCode,
						"details", res.Details,
					)
				}

				if d.cfg.ActionDelay > 0 {
					time.Sleep(d.cfg.ActionDelay)
				}
			}
		}

		if d.cfg.Mode != "continuous" && d.cfg.Iterations > 0 && currentIteration >= d.cfg.Iterations {
			break
		}
	}

	summary.Duration = time.Since(startTime)
	d.logger.Info("simulation drive completed",
		"total_actions", summary.TotalActions,
		"successes", summary.Successes,
		"failures", summary.Failures,
		"duration", summary.Duration,
	)

	return summary, nil
}

func (d *Driver) getMatchingPersonas() []*Persona {
	if strings.EqualFold(d.cfg.TargetPersona, "all") || d.cfg.TargetPersona == "" {
		return d.registry.List()
	}

	if p, ok := d.registry.Get(d.cfg.TargetPersona); ok {
		return []*Persona{p}
	}

	return nil
}
