package engine

import (
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHistoricalBars(t *testing.T) {
	start := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC)

	// Test 1Min timeframe (61 bars: 10:00 through 11:00 inclusive)
	bars1Min := GenerateHistoricalBars("AAPL", start, end, "1Min")
	assert.Len(t, bars1Min, 61)
	assert.Equal(t, start, bars1Min[0].Timestamp)
	assert.Equal(t, end, bars1Min[len(bars1Min)-1].Timestamp)

	// Test 5Min timeframe (13 bars: 10:00, 10:05, ..., 11:00)
	bars5Min := GenerateHistoricalBars("AAPL", start, end, "5Min")
	assert.Len(t, bars5Min, 13)

	// Test 1Hour timeframe (2 bars: 10:00, 11:00)
	bars1Hour := GenerateHistoricalBars("AAPL", start, end, "1Hour")
	assert.Len(t, bars1Hour, 2)

	// Test 1Day timeframe (1 bar for same day)
	bars1Day := GenerateHistoricalBars("AAPL", start, end, "1Day")
	assert.Len(t, bars1Day, 1)

	// Test invalid bounds (start after end)
	emptyBars := GenerateHistoricalBars("AAPL", end, start, "1Min")
	assert.Empty(t, emptyBars)

	// Verify bar fields correctness
	firstBar := bars1Min[0]
	assert.Equal(t, 100.0, firstBar.Open)
	assert.Greater(t, firstBar.High, firstBar.Open)
	assert.Less(t, firstBar.Low, firstBar.Open)
	assert.Equal(t, int64(50000), firstBar.Volume)
}

func TestNewBarAggregator_CapacityAndSeeding(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	// Test default capacity fallback when capacity <= 0
	agg := NewBarAggregator("TSLA", 0, 100.0, now)
	assert.Equal(t, 1440, agg.capacity)

	// Pre-seeded bars check (60 seeded bars + 1 current forming bar)
	bars := agg.GetBars("1Min", 0, nil, nil)
	require.Len(t, bars, 61)

	// Verify chronological order of pre-seeded bars
	for i := 1; i < len(bars); i++ {
		assert.True(t, bars[i].Timestamp.After(bars[i-1].Timestamp) || bars[i].Timestamp.Equal(bars[i-1].Timestamp))
	}

	// Verify the bounds of the first seeded bar and last forming bar
	firstSeeded := bars[0]
	expectedFirstTime := now.Truncate(time.Minute).Add(-60 * time.Minute)
	assert.Equal(t, expectedFirstTime, firstSeeded.Timestamp)

	currentForming := bars[len(bars)-1]
	assert.Equal(t, now.Truncate(time.Minute), currentForming.Timestamp)
	assert.Equal(t, 100.0, currentForming.Open)
}

func TestBarAggregator_RingBufferWrapAround(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	// Small capacity of 5 bars
	agg := NewBarAggregator("TEST", 5, 50.0, startTime)

	// Update ticks across 10 distinct minutes
	for i := 1; i <= 10; i++ {
		tickTime := startTime.Add(time.Duration(i) * time.Minute)
		agg.Update(50.0+float64(i), 100*int64(i), tickTime)
	}

	// Internal ring count should not exceed capacity of 5
	agg.mu.RLock()
	assert.LessOrEqual(t, agg.count, 5)
	agg.mu.RUnlock()

	// GetBars should return at most capacity (5) + current forming bar or limit
	bars := agg.GetBars("1Min", 0, nil, nil)
	assert.LessOrEqual(t, len(bars), 6) // ring buffer + current bar

	// Verify returned bars are strictly in chronological order
	for i := 1; i < len(bars); i++ {
		assert.True(t, bars[i].Timestamp.After(bars[i-1].Timestamp))
	}
}

func TestBarAggregator_GetBars_TimeframesAndAggregation(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	agg := NewBarAggregator("BTC", 120, 100.0, startTime)

	// Add ticks for 15 minutes, 1 minute per bucket
	for m := 1; m <= 15; m++ {
		t := startTime.Add(time.Duration(m) * time.Minute)
		// First tick of minute sets open/high/low/close
		agg.Update(100.0+float64(m), 100, t)
		// Second tick of minute updates high/low/close and volume
		agg.Update(100.0+float64(m)+0.5, 50, t.Add(30*time.Second))
	}

	// 1. "1Min" or fallback/unknown timeframe
	bars1Min := agg.GetBars("1Min", 0, nil, nil)
	assert.NotEmpty(t, bars1Min)

	barsDefault := agg.GetBars("UnknownTimeframe", 0, nil, nil)
	assert.Equal(t, len(bars1Min), len(barsDefault))

	// 2. "5Min" timeframe aggregation
	bars5Min := agg.GetBars("5Min", 0, nil, nil)
	assert.NotEmpty(t, bars5Min)
	// Check aggregation properties for 5Min bars
	for _, b := range bars5Min {
		assert.GreaterOrEqual(t, b.High, b.Open)
		assert.GreaterOrEqual(t, b.High, b.Close)
		assert.LessOrEqual(t, b.Low, b.Open)
		assert.LessOrEqual(t, b.Low, b.Close)
		assert.Greater(t, b.Volume, int64(0))
	}

	// 3. "1Hour" timeframe aggregation
	bars1Hour := agg.GetBars("1Hour", 0, nil, nil)
	assert.NotEmpty(t, bars1Hour)
	// All 15 minutes fit within 1 or 2 hour buckets
	assert.LessOrEqual(t, len(bars1Hour), 2)

	// 4. "1Day" timeframe aggregation
	bars1Day := agg.GetBars("1Day", 0, nil, nil)
	assert.NotEmpty(t, bars1Day)
	assert.LessOrEqual(t, len(bars1Day), 2)
}

func TestBarAggregator_GetBars_StartEndFiltering(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	agg := NewBarAggregator("NVDA", 120, 150.0, now)

	// Filter with start and end times that cover a subset of the 60 pre-seeded bars
	start := now.Add(-30 * time.Minute)
	end := now.Add(-10 * time.Minute)

	bars := agg.GetBars("1Min", 0, &start, &end)
	require.NotEmpty(t, bars)

	for _, b := range bars {
		assert.False(t, b.Timestamp.Before(start), "bar timestamp %v is before start %v", b.Timestamp, start)
		assert.False(t, b.Timestamp.After(end), "bar timestamp %v is after end %v", b.Timestamp, end)
	}
}

func TestBarAggregator_GetBars_LimitHandling(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	agg := NewBarAggregator("AMZN", 120, 200.0, now)

	totalBars := agg.GetBars("1Min", 0, nil, nil)
	require.Greater(t, len(totalBars), 10)

	// Test limit = 5
	limitedBars := agg.GetBars("1Min", 5, nil, nil)
	assert.Len(t, limitedBars, 5)

	// Verify limited bars match the tail (most recent) of total bars
	expectedTail := totalBars[len(totalBars)-5:]
	for i := range limitedBars {
		assert.Equal(t, expectedTail[i].Timestamp, limitedBars[i].Timestamp)
		assert.Equal(t, expectedTail[i].Close, limitedBars[i].Close)
	}

	// Test limit greater than total bars available
	excessiveLimitBars := agg.GetBars("1Min", 1000, nil, nil)
	assert.Equal(t, len(totalBars), len(excessiveLimitBars))

	// Test 5Min aggregation with limit
	limited5Min := agg.GetBars("5Min", 2, nil, nil)
	assert.LessOrEqual(t, len(limited5Min), 2)
}

func TestMarketEngine_GetBarsIntegration(t *testing.T) {
	store := ledger.NewStore()
	assets := []AssetConfig{
		{
			Symbol:          "AAPL",
			Name:            "Apple Inc",
			AssetID:         "ast-aapl",
			Exchange:        "NASDAQ",
			InitialPrice:    220.0,
			AnnualDrift:     0.05,
			Volatility:      0.15,
			SpreadBps:       0.0005,
			AvgVolumePerSec: 50,
		},
	}

	eng := NewMarketEngine(store, 10*time.Millisecond, assets)

	// Test existing symbol
	bars, ok := eng.GetBars("AAPL", "1Min", 10, nil, nil)
	assert.True(t, ok)
	assert.NotEmpty(t, bars)
	assert.LessOrEqual(t, len(bars), 10)

	// Test unknown symbol
	nilBars, ok := eng.GetBars("UNKNOWN", "1Min", 10, nil, nil)
	assert.False(t, ok)
	assert.Nil(t, nilBars)
}
