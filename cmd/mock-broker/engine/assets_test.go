package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAssets(t *testing.T) {
	assets := DefaultAssets()

	require.NotEmpty(t, assets)
	assert.Len(t, assets, 5)

	symbols := make(map[string]bool)
	assetIDs := make(map[string]bool)

	for _, asset := range assets {
		assert.NotEmpty(t, asset.Symbol, "asset symbol should not be empty")
		assert.NotEmpty(t, asset.Name, "asset name should not be empty")
		assert.NotEmpty(t, asset.AssetID, "asset ID should not be empty")
		assert.NotEmpty(t, asset.Exchange, "asset exchange should not be empty")
		assert.Greater(t, asset.InitialPrice, 0.0, "asset initial price should be positive")
		assert.Greater(t, asset.Volatility, 0.0, "asset volatility should be positive")
		assert.Greater(t, asset.SpreadBps, 0.0, "asset spread bps should be positive")
		assert.Greater(t, asset.AvgVolumePerSec, int64(0), "asset avg volume per sec should be positive")

		assert.False(t, symbols[asset.Symbol], "duplicate symbol found: %s", asset.Symbol)
		assert.False(t, assetIDs[asset.AssetID], "duplicate asset ID found: %s", asset.AssetID)

		symbols[asset.Symbol] = true
		assetIDs[asset.AssetID] = true
	}

	expectedSymbols := []string{"AAPL", "NVDA", "MSFT", "TSLA", "SPY"}
	for _, sym := range expectedSymbols {
		assert.True(t, symbols[sym], "expected symbol %s in default assets", sym)
	}
}
