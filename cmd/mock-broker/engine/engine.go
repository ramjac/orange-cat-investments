package engine

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
)

// MarketEngine coordinates stochastic asset price generation, quote synthesis, and bar aggregators.
type MarketEngine struct {
	mu            sync.RWMutex
	assets        map[string]AssetConfig
	currentPrice  map[string]float64
	previousClose map[string]float64
	quotes        map[string]ledger.Quote
	aggregators   map[string]*BarAggregator
	gbm           *GBMCalculator
	store         *ledger.Store

	tickInterval time.Duration
	stopCh       chan struct{}
	doneCh       chan struct{}
}

// NewMarketEngine creates and initializes a MarketEngine.
func NewMarketEngine(store *ledger.Store, tickInterval time.Duration, assets []AssetConfig) *MarketEngine {
	if tickInterval <= 0 {
		tickInterval = time.Second
	}
	if len(assets) == 0 {
		assets = DefaultAssets()
	}

	now := time.Now().UTC()
	gbm := NewGBMCalculator(time.Now().UnixNano())

	assetMap := make(map[string]AssetConfig, len(assets))
	prices := make(map[string]float64, len(assets))
	prevCloses := make(map[string]float64, len(assets))
	quotes := make(map[string]ledger.Quote, len(assets))
	aggregators := make(map[string]*BarAggregator, len(assets))

	for _, a := range assets {
		assetMap[a.Symbol] = a
		prices[a.Symbol] = a.InitialPrice
		prevCloses[a.Symbol] = math.Round(a.InitialPrice*0.99*100.0) / 100.0

		bid, ask := CalculateSpread(a.InitialPrice, a.SpreadBps)
		quotes[a.Symbol] = ledger.Quote{
			BidPrice:  bid,
			BidSize:   100,
			AskPrice:  ask,
			AskSize:   100,
			Timestamp: now,
		}
		aggregators[a.Symbol] = NewBarAggregator(a.Symbol, 1440, a.InitialPrice, now)
	}

	return &MarketEngine{
		assets:        assetMap,
		currentPrice:  prices,
		previousClose: prevCloses,
		quotes:        quotes,
		aggregators:   aggregators,
		gbm:           gbm,
		store:         store,
		tickInterval:  tickInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// InitialPrices returns a map of symbol -> current price.
func (e *MarketEngine) InitialPrices() map[string]float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	res := make(map[string]float64, len(e.currentPrice))
	for k, v := range e.currentPrice {
		res[k] = v
	}
	return res
}

// Start runs the periodic market simulation loop in a background goroutine.
func (e *MarketEngine) Start(ctx context.Context) {
	ticker := time.NewTicker(e.tickInterval)
	defer ticker.Stop()
	defer close(e.doneCh)

	dtSeconds := e.tickInterval.Seconds()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case t := <-ticker.C:
			e.tick(t.UTC(), dtSeconds)
		}
	}
}

// Stop signals the engine to terminate.
func (e *MarketEngine) Stop() {
	close(e.stopCh)
	<-e.doneCh
}

// tick executes one step of the stochastic price simulation for all assets.
func (e *MarketEngine) tick(now time.Time, dtSeconds float64) {
	e.mu.Lock()

	prices := make(map[string]float64, len(e.assets))
	quotes := make(map[string]ledger.Quote, len(e.assets))

	for symbol, cfg := range e.assets {
		current := e.currentPrice[symbol]
		newPrice := e.gbm.StepPrice(current, cfg.AnnualDrift, cfg.Volatility, dtSeconds)
		bid, ask := CalculateSpread(newPrice, cfg.SpreadBps)
		vol := e.gbm.GenerateTickVolume(cfg.AvgVolumePerSec, dtSeconds)

		e.currentPrice[symbol] = newPrice
		q := ledger.Quote{
			BidPrice:  bid,
			BidSize:   vol / 2,
			AskPrice:  ask,
			AskSize:   vol / 2,
			Timestamp: now,
		}
		e.quotes[symbol] = q

		prices[symbol] = newPrice
		quotes[symbol] = q

		// Update candlestick aggregator
		if agg, ok := e.aggregators[symbol]; ok {
			agg.Update(newPrice, vol, now)
		}
	}

	e.mu.Unlock()

	// Update ledger valuations and pending limit orders
	if e.store != nil {
		e.store.UpdateAccountValuations(prices)
		e.store.CheckLimitOrders(quotes)
	}
}

// GetQuote returns the latest NBBO quote for a symbol.
func (e *MarketEngine) GetQuote(symbol string) (ledger.Quote, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	q, ok := e.quotes[symbol]
	return q, ok
}

// GetAllQuotes returns the latest quotes for all active assets.
func (e *MarketEngine) GetAllQuotes() map[string]ledger.Quote {
	e.mu.RLock()
	defer e.mu.RUnlock()

	res := make(map[string]ledger.Quote, len(e.quotes))
	for k, v := range e.quotes {
		res[k] = v
	}
	return res
}

// GetLatestBar returns the latest 1-min bar for a symbol.
func (e *MarketEngine) GetLatestBar(symbol string) (ledger.Bar, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	agg, ok := e.aggregators[symbol]
	if !ok {
		return ledger.Bar{}, false
	}
	return agg.GetLatestBar(), true
}

// GetBars returns historical bars for a symbol with timeframe and limit.
func (e *MarketEngine) GetBars(symbol, timeframe string, limit int, start, end *time.Time) ([]ledger.Bar, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	agg, ok := e.aggregators[symbol]
	if !ok {
		return nil, false
	}
	return agg.GetBars(timeframe, limit, start, end), true
}

// GetPriceSummary returns current price, previous close, and percent change.
func (e *MarketEngine) GetPriceSummary(symbol string) (price, changePct float64, ok bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	p, exists := e.currentPrice[symbol]
	if !exists {
		return 0, 0, false
	}
	prev := e.previousClose[symbol]
	pct := 0.0
	if prev > 0 {
		pct = math.Round(((p-prev)/prev)*10000.0) / 100.0
	}
	return p, pct, true
}
