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

// Action represents a single daily workflow step executed by an OCI employee persona.
type Action struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, client *Client) (*ActionResult, error)
}

// Persona represents an OCI employee (human or feline) with daily simulation actions.
type Persona struct {
	ID         string
	Name       string
	Type       string // "human" or "feline"
	RoleTitle  string
	Department string
	Actions    []Action
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
