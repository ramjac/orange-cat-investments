package engine

import (
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
)

// BarAggregator maintains real-time rolling OHLCV candlestick bars in a ring buffer.
type BarAggregator struct {
	mu         sync.RWMutex
	symbol     string
	capacity   int
	minuteBars []ledger.Bar
	head       int
	count      int

	currentBar *ledger.Bar
	currentMin time.Time
}

// NewBarAggregator creates a new BarAggregator for a symbol with ring capacity (e.g., 1440 for 24h of 1m bars).
func NewBarAggregator(symbol string, capacity int, initialPrice float64, startTime time.Time) *BarAggregator {
	if capacity <= 0 {
		capacity = 1440
	}

	roundedTime := startTime.Truncate(time.Minute)
	ba := &BarAggregator{
		symbol:     symbol,
		capacity:   capacity,
		minuteBars: make([]ledger.Bar, capacity),
		currentMin: roundedTime,
		currentBar: &ledger.Bar{
			Timestamp: roundedTime,
			Open:      initialPrice,
			High:      initialPrice,
			Low:       initialPrice,
			Close:     initialPrice,
			Volume:    0,
		},
	}

	// Pre-seed some past bars so charts aren't completely empty on startup
	ba.seedHistoricalBars(initialPrice, startTime)
	return ba
}

func (ba *BarAggregator) seedHistoricalBars(basePrice float64, now time.Time) {
	rounded := now.Truncate(time.Minute)
	// Seed last 60 minutes
	price := basePrice
	for i := 60; i >= 1; i-- {
		t := rounded.Add(time.Duration(-i) * time.Minute)
		open := price
		high := open * 1.002
		low := open * 0.998
		closeP := (high + low) / 2.0
		bar := ledger.Bar{
			Timestamp: t,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeP,
			Volume:    50000,
		}
		ba.minuteBars[ba.head] = bar
		ba.head = (ba.head + 1) % ba.capacity
		if ba.count < ba.capacity {
			ba.count++
		}
		price = closeP
	}
}

// Update records a new price tick into the current minute bar.
func (ba *BarAggregator) Update(price float64, volume int64, t time.Time) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	tickMin := t.Truncate(time.Minute)
	if tickMin.After(ba.currentMin) {
		// Flush current bar to ring buffer
		if ba.currentBar != nil {
			ba.minuteBars[ba.head] = *ba.currentBar
			ba.head = (ba.head + 1) % ba.capacity
			if ba.count < ba.capacity {
				ba.count++
			}
		}

		// Start new bar
		ba.currentMin = tickMin
		ba.currentBar = &ledger.Bar{
			Timestamp: tickMin,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    volume,
		}
		return
	}

	// Update existing bar
	if ba.currentBar == nil {
		ba.currentBar = &ledger.Bar{
			Timestamp: tickMin,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    volume,
		}
		return
	}

	if price > ba.currentBar.High {
		ba.currentBar.High = price
	}
	if price < ba.currentBar.Low {
		ba.currentBar.Low = price
	}
	ba.currentBar.Close = price
	ba.currentBar.Volume += volume
}

// GetLatestBar returns the current active candlestick bar.
func (ba *BarAggregator) GetLatestBar() ledger.Bar {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	if ba.currentBar != nil {
		return *ba.currentBar
	}
	return ledger.Bar{}
}

// GetBars returns chronological bars matching timeframe (1Min, 5Min, 1Hour, 1Day) and limit.
func (ba *BarAggregator) GetBars(timeframe string, limit int, start, end *time.Time) []ledger.Bar {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	// Extract minute bars in chronological order
	var raw []ledger.Bar
	startIdx := (ba.head - ba.count + ba.capacity) % ba.capacity
	for i := 0; i < ba.count; i++ {
		idx := (startIdx + i) % ba.capacity
		b := ba.minuteBars[idx]
		if start != nil && b.Timestamp.Before(*start) {
			continue
		}
		if end != nil && b.Timestamp.After(*end) {
			continue
		}
		raw = append(raw, b)
	}

	// Include current forming bar if matching range
	if ba.currentBar != nil {
		cb := *ba.currentBar
		if (start == nil || !cb.Timestamp.Before(*start)) && (end == nil || !cb.Timestamp.After(*end)) {
			raw = append(raw, cb)
		}
	}

	var interval time.Duration
	switch timeframe {
	case "5Min":
		interval = 5 * time.Minute
	case "1Hour":
		interval = time.Hour
	case "1Day":
		interval = 24 * time.Hour
	default: // "1Min" or fallback
		interval = time.Minute
	}

	if interval == time.Minute {
		if limit > 0 && len(raw) > limit {
			return raw[len(raw)-limit:]
		}
		return raw
	}

	// Aggregate into larger timeframe bars
	var aggregated []ledger.Bar
	var currentAgg *ledger.Bar
	var aggBucket time.Time

	for _, b := range raw {
		bucket := b.Timestamp.Truncate(interval)
		if currentAgg == nil || bucket != aggBucket {
			if currentAgg != nil {
				aggregated = append(aggregated, *currentAgg)
			}
			aggBucket = bucket
			currentAgg = &ledger.Bar{
				Timestamp: bucket,
				Open:      b.Open,
				High:      b.High,
				Low:       b.Low,
				Close:     b.Close,
				Volume:    b.Volume,
			}
		} else {
			if b.High > currentAgg.High {
				currentAgg.High = b.High
			}
			if b.Low < currentAgg.Low {
				currentAgg.Low = b.Low
			}
			currentAgg.Close = b.Close
			currentAgg.Volume += b.Volume
		}
	}

	if currentAgg != nil {
		aggregated = append(aggregated, *currentAgg)
	}

	if limit > 0 && len(aggregated) > limit {
		return aggregated[len(aggregated)-limit:]
	}
	return aggregated
}
