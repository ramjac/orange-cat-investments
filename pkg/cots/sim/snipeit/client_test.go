package snipeit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimSnipeITClient_OfflineFallback(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	ticket, err := client.CreateAssetRecoveryTicket(context.Background(), "emp-sim", "Separation")
	require.NoError(t, err)
	assert.Equal(t, "emp-sim", ticket.EmployeeID)
	assert.Equal(t, "Pending Recovery", ticket.Status)
	assert.NotEmpty(t, ticket.Assets)
}
