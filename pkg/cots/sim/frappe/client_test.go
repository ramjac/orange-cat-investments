package frappe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimFrappeClient_OfflineFallback(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	metrics, err := client.GetFinancialMetrics(context.Background(), "2026-10")
	require.NoError(t, err)
	assert.Equal(t, "USD", metrics.Currency)
	assert.True(t, metrics.TotalRevenue > 0)
	assert.True(t, metrics.CashReserves > 0)
}
