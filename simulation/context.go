package simulation

import (
	"context"
)

type personaCtxKey struct{}

// WithPersona attaches the active Persona to the context.
func WithPersona(ctx context.Context, p *Persona) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, personaCtxKey{}, p)
}

// PersonaFromContext retrieves the active Persona from context if present.
func PersonaFromContext(ctx context.Context) *Persona {
	if ctx == nil {
		return nil
	}
	if p, ok := ctx.Value(personaCtxKey{}).(*Persona); ok {
		return p
	}
	return nil
}
