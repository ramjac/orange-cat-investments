package engine

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// GBMCalculator simulates equity price evolution using Geometric Brownian Motion.
type GBMCalculator struct {
	mu  sync.Mutex
	rnd *rand.Rand
}

// NewGBMCalculator creates a new initialized GBM calculator.
func NewGBMCalculator(seed int64) *GBMCalculator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &GBMCalculator{
		rnd: rand.New(rand.NewSource(seed)),
	}
}

// NextStandardNormal generates a sample from standard normal distribution N(0, 1) using Box-Muller.
func (c *GBMCalculator) NextStandardNormal() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	u1 := c.rnd.Float64()
	for u1 <= 0.0 {
		u1 = c.rnd.Float64()
	}
	u2 := c.rnd.Float64()

	return math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
}

// StepPrice calculates the new price after time interval dtSeconds (in seconds).
// dt is converted to years assuming 252 trading days * 6.5 trading hours * 3600 seconds per day.
func (c *GBMCalculator) StepPrice(currentPrice, annualDrift, annualVol, dtSeconds float64) float64 {
	if currentPrice <= 0.01 {
		currentPrice = 0.01
	}

	const tradingSecondsPerYear = 252.0 * 6.5 * 3600.0
	dt := dtSeconds / tradingSecondsPerYear

	z := c.NextStandardNormal()
	exponent := (annualDrift - 0.5*annualVol*annualVol)*dt + annualVol*math.Sqrt(dt)*z
	newPrice := currentPrice * math.Exp(exponent)

	// Round to 2 decimal places (standard US dollar cents)
	return math.Round(newPrice*100.0) / 100.0
}

// CalculateSpread returns bid and ask prices based on current price and spread in basis points.
func CalculateSpread(price, spreadBps float64) (bidPrice, askPrice float64) {
	halfSpread := price * (spreadBps / 2.0)
	if halfSpread < 0.005 {
		halfSpread = 0.005 // minimum 1 cent spread ($0.01)
	}

	bid := math.Round((price-halfSpread)*100.0) / 100.0
	ask := math.Round((price+halfSpread)*100.0) / 100.0

	if ask <= bid {
		ask = bid + 0.01
	}

	return bid, ask
}

// GenerateTickVolume generates simulated trading volume for a time interval.
func (c *GBMCalculator) GenerateTickVolume(avgVolumePerSec int64, dtSeconds float64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	mean := float64(avgVolumePerSec) * dtSeconds
	// Add random noise (+- 40%)
	noise := 0.6 + c.rnd.Float64()*0.8
	vol := int64(math.Round(mean * noise))
	if vol < 1 {
		vol = 1
	}
	return vol
}
