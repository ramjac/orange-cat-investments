package sim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrappeSimClient(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	m, err := client.GetFinancialMetrics(context.Background(), "2026-10")
	require.NoError(t, err)
	assert.Equal(t, "USD", m.Currency)
}
