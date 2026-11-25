package grpcutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestActorContextPropagation(t *testing.T) {
	ctx := context.Background()
	_, ok := ActorFromIncomingContext(ctx)
	assert.False(t, ok)

	outCtx := WithOutgoingActor(ctx, ActorContext{
		ActorID:    "emp-human-elena",
		Role:       "VETERINARY",
		Department: "Veterinary Medicine",
	})
	md, ok := metadata.FromOutgoingContext(outCtx)
	require.True(t, ok)

	inCtx := metadata.NewIncomingContext(context.Background(), md)
	actor, ok := ActorFromIncomingContext(inCtx)
	require.True(t, ok)
	assert.Equal(t, "emp-human-elena", actor.ActorID)
	assert.Equal(t, "veterinary", actor.Role)
	assert.Equal(t, "Veterinary Medicine", actor.Department)

	inDirect := WithIncomingActor(context.Background(), ActorContext{
		ActorID: "emp-human-alice",
		Role:    "HR",
	})
	actor2, ok := ActorFromIncomingContext(inDirect)
	require.True(t, ok)
	assert.Equal(t, "emp-human-alice", actor2.ActorID)
	assert.Equal(t, "hr", actor2.Role)
}
