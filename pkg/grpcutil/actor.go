package grpcutil

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	MetadataActorIDKey    = "x-oci-actor-id"
	MetadataActorEmailKey = "x-oci-actor-email"
	MetadataActorRoleKey  = "x-oci-actor-role"
	MetadataActorDeptKey  = "x-oci-actor-dept"
)

// ActorContext represents the authenticated caller propagated across BFF -> BLL gRPC metadata.
type ActorContext struct {
	ActorID    string
	Role       string
	Department string
	// Email is the authenticated identity's email address. Downstream
	// authorization layers correlate it to canonical domain identities
	// (e.g. workforce employee IDs) for owner-based policy rules.
	Email string
}

// WithOutgoingActor attaches authenticated actor identity metadata to an outgoing gRPC context.
func WithOutgoingActor(ctx context.Context, actor ActorContext) context.Context {
	pairs := make([]string, 0, 6)
	if id := strings.TrimSpace(actor.ActorID); id != "" {
		pairs = append(pairs, MetadataActorIDKey, id)
	}
	if role := strings.TrimSpace(actor.Role); role != "" {
		pairs = append(pairs, MetadataActorRoleKey, strings.ToLower(role))
	}
	if dept := strings.TrimSpace(actor.Department); dept != "" {
		pairs = append(pairs, MetadataActorDeptKey, dept)
	}
	if email := strings.TrimSpace(actor.Email); email != "" {
		pairs = append(pairs, MetadataActorEmailKey, email)
	}
	if len(pairs) == 0 {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...)
}

// WithIncomingActor attaches actor identity directly as incoming gRPC metadata (primarily for unit tests
// and trusted internal webhooks like Frappe HR callbacks).
func WithIncomingActor(ctx context.Context, actor ActorContext) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	if id := strings.TrimSpace(actor.ActorID); id != "" {
		md.Set(MetadataActorIDKey, id)
	}
	if role := strings.TrimSpace(actor.Role); role != "" {
		md.Set(MetadataActorRoleKey, strings.ToLower(role))
	}
	if dept := strings.TrimSpace(actor.Department); dept != "" {
		md.Set(MetadataActorDeptKey, dept)
	}
	if email := strings.TrimSpace(actor.Email); email != "" {
		md.Set(MetadataActorEmailKey, email)
	}
	return metadata.NewIncomingContext(ctx, md)
}

// ActorFromIncomingContext extracts the propagated ActorContext from incoming gRPC metadata.
// Returns (ActorContext{}, false) if no actor metadata is present.
func ActorFromIncomingContext(ctx context.Context) (ActorContext, bool) {
	if ctx == nil {
		return ActorContext{}, false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ActorContext{}, false
	}
	var actor ActorContext
	if vals := md.Get(MetadataActorIDKey); len(vals) > 0 {
		actor.ActorID = strings.TrimSpace(vals[0])
	}
	if vals := md.Get(MetadataActorRoleKey); len(vals) > 0 {
		actor.Role = strings.ToLower(strings.TrimSpace(vals[0]))
	}
	if vals := md.Get(MetadataActorDeptKey); len(vals) > 0 {
		actor.Department = strings.TrimSpace(vals[0])
	}
	if vals := md.Get(MetadataActorEmailKey); len(vals) > 0 {
		actor.Email = strings.TrimSpace(vals[0])
	}
	if actor.ActorID == "" && actor.Role == "" {
		return ActorContext{}, false
	}
	return actor, true
}
