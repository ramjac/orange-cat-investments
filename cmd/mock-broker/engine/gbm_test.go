package engine

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGBMCalculator(t *testing.T) {
	t.Run("default seed (0)", func(t *testing.T) {
		calc := NewGBMCalculator(0)
		require.NotNil(t, calc)
		val := calc.NextStandardNormal()
		assert.False(t, math.IsNaN(val))
		assert.False(t, math.IsInf(val, 0))
	})

	t.Run("deterministic with identical explicit seed", func(t *testing.T) {
		seed := int64(123456789)
		calc1 := NewGBMCalculator(seed)
		calc2 := NewGBMCalculator(seed)

		for i := 0; i < 10; i++ {
			val1 := calc1.NextStandardNormal()
			val2 := calc2.NextStandardNormal()
			assert.InDelta(t, val1, val2, 1e-9)
		}
	})
}

func TestNextStandardNormal_Distribution(t *testing.T) {
	calc := NewGBMCalculator(42)
	n := 10000
	sum := 0.0
	sumSq := 0.0

	for i := 0; i < n; i++ {
		val := calc.NextStandardNormal()
		assert.False(t, math.IsNaN(val))
		assert.False(t, math.IsInf(val, 0))
		sum += val
		sumSq += val * val
	}

	mean := sum / float64(n)
	variance := (sumSq / float64(n)) - (mean * mean)

	// Standard normal distribution has mean 0 and variance 1
	assert.InDelta(t, 0.0, mean, 0.05, "Mean should be close to 0")
	assert.InDelta(t, 1.0, variance, 0.05, "Variance should be close to 1")
}

func TestStepPrice(t *testing.T) {
	calc := NewGBMCalculator(100)

	t.Run("normal price stepping", func(t *testing.T) {
		initialPrice := 150.00
		drift := 0.08
		vol := 0.20
		dtSeconds := 1.0

		newPrice := calc.StepPrice(initialPrice, drift, vol, dtSeconds)
		assert.Greater(t, newPrice, 0.0)

		// Check 2 decimal places precision
		rounded := math.Round(newPrice*100.0) / 100.0
		assert.Equal(t, rounded, newPrice, "Price should be rounded to 2 decimal places")
	})

	t.Run("floor price when currentPrice <= 0.01", func(t *testing.T) {
		lowPrices := []float64{0.01, 0.005, 0.0, -10.0}

		for _, price := range lowPrices {
			newPrice := calc.StepPrice(price, 0.05, 0.10, 1.0)
			assert.Greater(t, newPrice, 0.0, "Stepped price should remain positive when starting <= 0.01")
		}
	})
}

func TestCalculateSpread(t *testing.T) {
	t.Run("standard spread calculation", func(t *testing.T) {
		price := 100.00
		spreadBps := 0.0010 // 10 bps = 0.1% = total spread 0.10, halfSpread = 0.05

		bid, ask := CalculateSpread(price, spreadBps)
		assert.Equal(t, 99.95, bid)
		assert.Equal(t, 100.05, ask)
		assert.Greater(t, ask, bid)
	})

	t.Run("minimum half-spread floor ($0.005)", func(t *testing.T) {
		price := 1.00
		spreadBps := 0.0001 // halfSpread would be 1.00 * 0.00005 = 0.00005 < 0.005

		bid, ask := CalculateSpread(price, spreadBps)
		// halfSpread floor to 0.005 -> bid = 1.00 - 0.005 = 0.995 -> rounded 1.00, ask = 1.00 + 0.005 = 1.005 -> rounded 1.01
		assert.Less(t, bid, ask)
		assert.InDelta(t, 0.01, ask-bid, 1e-9)
	})

	t.Run("ensure ask > bid condition when ask <= bid", func(t *testing.T) {
		// Small price where bid and ask round to same cent without ask adjustment
		bid, ask := CalculateSpread(0.01, 0.0001)
		assert.Greater(t, ask, bid, "Ask must always be greater than bid")
		assert.InDelta(t, 0.01, ask-bid, 1e-9)
	})
}

func TestGenerateTickVolume(t *testing.T) {
	calc := NewGBMCalculator(777)

	t.Run("volume generation within noise bounds", func(t *testing.T) {
		avgVol := int64(100)
		dtSeconds := 2.0 // expected mean volume = 200

		for i := 0; i < 50; i++ {
			vol := calc.GenerateTickVolume(avgVol, dtSeconds)
			// Noise range is [0.6, 1.4], so 200 * 0.6 = 120, 200 * 1.4 = 280
			assert.GreaterOrEqual(t, vol, int64(120))
			assert.LessOrEqual(t, vol, int64(280))
		}
	})

	t.Run("minimum volume floor (at least 1)", func(t *testing.T) {
		vol := calc.GenerateTickVolume(0, 0.0)
		assert.Equal(t, int64(1), vol, "Volume should be at least 1")
	})
}
