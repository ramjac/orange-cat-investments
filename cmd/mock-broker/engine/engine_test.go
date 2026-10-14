package engine

import (
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGBMCalculator(t *testing.T) {
	calc := NewGBMCalculator(42)

	// Test normal distribution sampling
	val := calc.NextStandardNormal()
	assert.False(t, val == 0.0)

	// Test price step
	initialPrice := 100.0
	drift := 0.10
	vol := 0.25
	dt := 1.0 // 1 second

	stepPrice := calc.StepPrice(initialPrice, drift, vol, dt)
	assert.Greater(t, stepPrice, 50.0)
	assert.Less(t, stepPrice, 150.0)

	// Test Spread Calculation
	bid, ask := CalculateSpread(200.0, 0.0005)
	assert.Less(t, bid, ask)
	assert.Greater(t, bid, 199.0)
	assert.Less(t, ask, 201.0)
}

func TestBarAggregator(t *testing.T) {
	now := time.Now().UTC()
	agg := NewBarAggregator("AAPL", 60, 224.30, now)

	latest := agg.GetLatestBar()
	assert.Equal(t, 224.30, latest.Close)

	// Update tick in same minute
	agg.Update(225.00, 100, now.Add(5*time.Second))
	latest = agg.GetLatestBar()
	assert.Equal(t, 225.00, latest.High)
	assert.Equal(t, 225.00, latest.Close)
	assert.Equal(t, int64(100), latest.Volume)

	// Tick into next minute
	nextMin := now.Add(65 * time.Second)
	agg.Update(226.50, 250, nextMin)
	latest = agg.GetLatestBar()
	assert.Equal(t, 226.50, latest.Close)

	bars := agg.GetBars("1Min", 10, nil, nil)
	assert.NotEmpty(t, bars)
	assert.LessOrEqual(t, len(bars), 10)
}

func TestMarketEngine(t *testing.T) {
	store := ledger.NewStore()
	assets := []AssetConfig{
		{
			Symbol:          "TEST",
			Name:            "Test Stock",
			AssetID:         "ast-test",
			Exchange:        "NASDAQ",
			InitialPrice:    100.0,
			AnnualDrift:     0.10,
			Volatility:      0.20,
			SpreadBps:       0.0005,
			AvgVolumePerSec: 100,
		},
	}

	eng := NewMarketEngine(store, 10*time.Millisecond, assets)
	quote, ok := eng.GetQuote("TEST")
	require.True(t, ok)
	assert.Greater(t, quote.AskPrice, quote.BidPrice)

	prices := eng.InitialPrices()
	assert.Equal(t, 100.0, prices["TEST"])

	// Test 1-tick step
	eng.tick(time.Now().UTC(), 0.01)
	newQuote, ok := eng.GetQuote("TEST")
	require.True(t, ok)
	assert.Greater(t, newQuote.AskPrice, 0.0)

	price, _, ok := eng.GetPriceSummary("TEST")
	require.True(t, ok)
	assert.Greater(t, price, 0.0)
}
