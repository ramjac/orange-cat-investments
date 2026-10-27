package simulation

import (
	"context"
	"time"
)

// ActionResult records the execution status and outcome of a persona action.
type ActionResult struct {
	PersonaID   string    `json:"persona_id"`
	PersonaName string    `json:"persona_name"`
	ActionName  string    `json:"action_name"`
	Endpoint    string    `json:"endpoint,omitempty"`
	Success     bool      `json:"success"`
	StatusCode  int       `json:"status_code,omitempty"`
	Details     string    `json:"details"`
	Timestamp   time.Time `json:"timestamp"`
}

// Action represents a single daily workflow step executed by an OCI employee or customer persona.
type Action struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, client *Client) (*ActionResult, error)
}

// Persona represents an OCI persona (human employee, feline executive, or customer investor).
type Persona struct {
	ID            string
	Name          string
	Type          string // "human", "feline", or "customer"
	RoleTitle     string
	Department    string
	Role          string
	Email         string
	ActivityLevel float64
	Actions       []Action
}

// AuthRole returns the persona's role used for authentication and RBAC checks.
func (p *Persona) AuthRole() string {
	if p == nil {
		return "staff"
	}
	if p.Role != "" {
		return p.Role
	}
	if p.Type == "customer" {
		return "customer"
	}
	switch p.ID {
	case "emp-feline-garfield":
		return "admin"
	case "emp-feline-barneby":
		return "staff"
	case "emp-human-alice":
		return "hr"
	case "emp-human-bob":
		return "facilities"
	case "emp-human-carol":
		return "investment"
	case "emp-human-david":
		return "investment"
	case "emp-human-elena":
		return "veterinary"
	case "emp-human-frank":
		return "ops"
	case "emp-human-rick":
		return "staff"
	case "emp-human-elise":
		return "engineering"
	case "emp-human-marcus":
		return "engineering"
	default:
		return "staff"
	}
}

// AuthEmail returns the persona's email used for authentication handshakes.
func (p *Persona) AuthEmail() string {
	if p == nil {
		return ""
	}
	if p.Email != "" {
		return p.Email
	}
	switch p.ID {
	case "emp-feline-garfield":
		return "garfield@oci.local"
	case "emp-feline-barneby":
		return "barneby@oci.local"
	case "emp-human-alice":
		return "alice.vance@oci.local"
	case "emp-human-bob":
		return "bob.builder@oci.local"
	case "emp-human-carol":
		return "carol.danvers@oci.local"
	case "emp-human-david":
		return "david.quant@oci.local"
	case "emp-human-elena":
		return "elena.rostova@oci.local"
	case "emp-human-frank":
		return "frank.ops@oci.local"
	case "emp-human-rick":
		return "rick.newhire@oci.local"
	case "emp-human-elise":
		return "elise.dev@oci.local"
	case "emp-human-marcus":
		return "marcus.lead@oci.local"
	case "cust-longterm-arthur":
		return "arthur@oci.local"
	case "cust-active-chloe":
		return "chloe@oci.local"
	default:
		return "user@oci.local"
	}
}

// Registry stores all registered OCI personas available for simulation.
type Registry struct {
	personas map[string]*Persona
	order    []string
}

// NewRegistry initializes a registry populated with all 8 OCI employee personas.
func NewRegistry() *Registry {
	r := &Registry{
		personas: make(map[string]*Persona),
		order:    make([]string, 0),
	}
	r.registerAllDefaultPersonas()
	return r
}

// Register adds a persona to the registry.
func (r *Registry) Register(p *Persona) {
	if _, exists := r.personas[p.ID]; !exists {
		r.order = append(r.order, p.ID)
	}
	r.personas[p.ID] = p
}

// Get retrieves a persona by ID or Name.
func (r *Registry) Get(idOrName string) (*Persona, bool) {
	if p, ok := r.personas[idOrName]; ok {
		return p, true
	}
	for _, p := range r.personas {
		if p.Name == idOrName {
			return p, true
		}
	}
	return nil, false
}

// List returns all registered personas in order.
func (r *Registry) List() []*Persona {
	list := make([]*Persona, 0, len(r.order))
	for _, id := range r.order {
		list = append(list, r.personas[id])
	}
	return list
}
