package sim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnipeITSimClient(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	ticket, err := client.CreateAssetRecoveryTicket(context.Background(), "emp-sim", "Separation")
	require.NoError(t, err)
	assert.Equal(t, "emp-sim", ticket.EmployeeID)
}
