package ledger

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrOrderNotFound       = errors.New("order not found")
	ErrPositionNotFound    = errors.New("position not found")
	ErrInsufficientCash    = errors.New("insufficient cash balance")
	ErrInsufficientShares  = errors.New("insufficient shares to sell")
	ErrInvalidOrderPayload = errors.New("invalid order payload")
	ErrOrderCannotBeCanceled = errors.New("order cannot be canceled")
)

type CreateOrderRequest struct {
	Symbol        string           `json:"symbol"`
	Qty           FlexibleFloat64  `json:"qty"`
	Side          string           `json:"side"` // "buy" or "sell"
	Type          string           `json:"type"` // "market", "limit"
	TimeInForce   string           `json:"time_in_force"`
	LimitPrice    *FlexibleFloat64 `json:"limit_price,omitempty"`
	ClientOrderID string           `json:"client_order_id,omitempty"`
}

type Store struct {
	mu             sync.RWMutex
	accounts       map[string]*Account
	positions      map[string]map[string]*Position // accountID -> symbol -> Position
	orders         map[string]*Order               // orderID -> Order
	accountOrders  map[string][]*Order             // accountID -> []*Order
	subscribers    map[chan TradeUpdate]struct{}
	subscribersMu  sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		accounts:      make(map[string]*Account),
		positions:     make(map[string]map[string]*Position),
		orders:        make(map[string]*Order),
		accountOrders: make(map[string][]*Order),
		subscribers:   make(map[chan TradeUpdate]struct{}),
	}
}

func (s *Store) CreateAccount(acc *Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if acc.ID == "" {
		acc.ID = fmt.Sprintf("acc-%d", time.Now().UnixNano())
	}
	if acc.AccountNumber == "" {
		acc.AccountNumber = fmt.Sprintf("OCI-%06d", len(s.accounts)+1)
	}
	if acc.Status == "" {
		acc.Status = "ACTIVE"
	}
	if acc.Currency == "" {
		acc.Currency = "USD"
	}
	acc.CreatedAt = time.Now().UTC()
	acc.BuyingPower = acc.Cash * 2.0
	acc.Equity = acc.Cash + acc.LongMarketValue - acc.ShortMarketValue
	acc.PortfolioValue = acc.Equity

	s.accounts[acc.ID] = acc
	if s.positions[acc.ID] == nil {
		s.positions[acc.ID] = make(map[string]*Position)
	}
	return nil
}

func (s *Store) GetAccount(id string) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, exists := s.accounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}
	copyAcc := *acc
	return &copyAcc, nil
}

func (s *Store) ListAccounts() []*Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*Account, 0, len(s.accounts))
	for _, acc := range s.accounts {
		copyAcc := *acc
		list = append(list, &copyAcc)
	}
	return list
}

func (s *Store) SetPosition(accountID string, pos *Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.positions[accountID] == nil {
		s.positions[accountID] = make(map[string]*Position)
	}
	s.positions[accountID][pos.Symbol] = pos
}

func (s *Store) GetPosition(accountID, symbol string) (*Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accPos, ok := s.positions[accountID]
	if !ok {
		return nil, ErrPositionNotFound
	}
	pos, ok := accPos[symbol]
	if !ok {
		return nil, ErrPositionNotFound
	}
	copyPos := *pos
	return &copyPos, nil
}

func (s *Store) ListPositions(accountID string) []*Position {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accPos, ok := s.positions[accountID]
	if !ok {
		return []*Position{}
	}
	list := make([]*Position, 0, len(accPos))
	for _, pos := range accPos {
		copyPos := *pos
		list = append(list, &copyPos)
	}
	return list
}

func (s *Store) GetOrder(accountID, orderID string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	copyOrder := *order
	return &copyOrder, nil
}

func (s *Store) ListOrders(accountID, status string, limit int) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := s.accountOrders[accountID]
	if limit <= 0 {
		limit = 50
	}

	result := make([]*Order, 0, limit)
	for _, ord := range all {
		if status == "" || status == "all" || ord.Status == status {
			copyOrd := *ord
			result = append(result, &copyOrd)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (s *Store) CancelOrder(accountID, orderID string) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if order.Status == "filled" || order.Status == "canceled" {
		return nil, fmt.Errorf("%w: status is %s", ErrOrderCannotBeCanceled, order.Status)
	}

	now := time.Now().UTC()
	order.Status = "canceled"
	order.CanceledAt = &now
	order.UpdatedAt = now

	tradeUpdate := TradeUpdate{
		Event:       "canceled",
		ExecutionID: fmt.Sprintf("exec-%d", now.UnixNano()),
		Order:       *order,
		Timestamp:   now,
	}
	go s.BroadcastTradeUpdate(tradeUpdate)

	copyOrd := *order
	return &copyOrd, nil
}

// UpdateAccountValuations refreshes all positions and portfolio values based on latest prices.
func (s *Store) UpdateAccountValuations(currentPrices map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for accID, acc := range s.accounts {
		accPositions := s.positions[accID]
		for sym, pos := range accPositions {
			if price, ok := currentPrices[sym]; ok && price > 0 {
				pos.CurrentPrice = price
				pos.MarketValue = pos.Qty * price
				pos.UnrealizedPL = pos.MarketValue - pos.CostBasis
				if pos.CostBasis > 0 {
					pos.UnrealizedPLPC = pos.UnrealizedPL / pos.CostBasis
				}
				if pos.LastdayPrice > 0 {
					pos.ChangeToday = (price - pos.LastdayPrice) / pos.LastdayPrice
				}
			}
		}
		s.recalculateAccountValuation(acc, accID)
	}
}

// CheckLimitOrders checks pending limit orders against latest quotes.
func (s *Store) CheckLimitOrders(quotes map[string]Quote) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for _, order := range s.orders {
		if order.Status != "accepted" || order.Type != "limit" || order.LimitPrice == nil {
			continue
		}

		quote, ok := quotes[order.Symbol]
		if !ok {
			continue
		}

		limitPrice := *order.LimitPrice
		shouldFill := false
		fillPrice := limitPrice

		if order.Side == "buy" && quote.AskPrice <= limitPrice && quote.AskPrice > 0 {
			shouldFill = true
			fillPrice = quote.AskPrice
		} else if order.Side == "sell" && quote.BidPrice >= limitPrice && quote.BidPrice > 0 {
			shouldFill = true
			fillPrice = quote.BidPrice
		}

		if shouldFill {
			// Find account
			for accID, orders := range s.accountOrders {
				for _, o := range orders {
					if o.ID == order.ID {
						acc := s.accounts[accID]
						if order.Side == "buy" {
							cost := order.Qty * fillPrice
							if acc.Cash >= cost {
								acc.Cash -= cost
								pos := s.positions[accID][order.Symbol]
								if pos == nil {
									pos = &Position{
										AssetID:       order.AssetID,
										Symbol:        order.Symbol,
										Exchange:      "NASDAQ",
										AssetClass:    "us_equity",
										Qty:           order.Qty,
										AvgEntryPrice: fillPrice,
										Side:          "long",
										CostBasis:     cost,
										CurrentPrice:  fillPrice,
									}
									s.positions[accID][order.Symbol] = pos
								} else {
									newQty := pos.Qty + order.Qty
									newBasis := pos.CostBasis + cost
									pos.Qty = newQty
									pos.CostBasis = newBasis
									pos.AvgEntryPrice = newBasis / newQty
									pos.CurrentPrice = fillPrice
								}
								order.Status = "filled"
								order.FilledQty = order.Qty
								order.FilledAvgPrice = &fillPrice
								order.FilledAt = &now
								s.recalculateAccountValuation(acc, accID)
							}
						} else if order.Side == "sell" {
							pos := s.positions[accID][order.Symbol]
							if pos != nil && pos.Qty >= order.Qty {
								proceeds := order.Qty * fillPrice
								acc.Cash += proceeds
								pos.Qty -= order.Qty
								pos.CostBasis -= order.Qty * pos.AvgEntryPrice
								pos.CurrentPrice = fillPrice
								if pos.Qty <= 0.00001 {
									delete(s.positions[accID], order.Symbol)
								}
								order.Status = "filled"
								order.FilledQty = order.Qty
								order.FilledAvgPrice = &fillPrice
								order.FilledAt = &now
								s.recalculateAccountValuation(acc, accID)
							}
						}
						break
					}
				}
			}
		}
	}
}

func (s *Store) SubscribeTradeUpdates() (chan TradeUpdate, func()) {
	ch := make(chan TradeUpdate, 50)
	s.subscribersMu.Lock()
	s.subscribers[ch] = struct{}{}
	s.subscribersMu.Unlock()

	unsubscribe := func() {
		s.subscribersMu.Lock()
		delete(s.subscribers, ch)
		close(ch)
		s.subscribersMu.Unlock()
	}
	return ch, unsubscribe
}

func (s *Store) BroadcastTradeUpdate(update TradeUpdate) {
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	for ch := range s.subscribers {
		select {
		case ch <- update:
		default:
		}
	}
}

// PortfolioHistoryPoint represents historical valuation at a timestamp.
type PortfolioHistoryPoint struct {
	Timestamp int64   `json:"timestamp"`
	Equity    float64 `json:"equity"`
	ProfitLoss float64 `json:"profit_loss"`
}

type PortfolioHistory struct {
	Timestamp     []int64   `json:"timestamp"`
	Equity        []float64 `json:"equity"`
	ProfitLoss    []float64 `json:"profit_loss"`
	ProfitLossPct []float64 `json:"profit_loss_pct"`
	BaseValue     float64   `json:"base_value"`
	Timeframe     string    `json:"timeframe"`
}

func (s *Store) GetPortfolioHistory(accountID string, period string) (*PortfolioHistory, error) {
	acc, err := s.GetAccount(accountID)
	if err != nil {
		return nil, err
	}

	history := &PortfolioHistory{
		Timestamp:     make([]int64, 0, 30),
		Equity:        make([]float64, 0, 30),
		ProfitLoss:    make([]float64, 0, 30),
		ProfitLossPct: make([]float64, 0, 30),
		BaseValue:     math.Round(acc.Equity * 0.85),
		Timeframe:     period,
	}

	now := time.Now().UTC()
	startVal := history.BaseValue
	currentVal := acc.Equity

	// Generate 30 daily equity points interpolating from startVal to currentVal
	for i := 29; i >= 0; i-- {
		t := now.Add(time.Duration(-i*24) * time.Hour).Unix()
		ratio := float64(30-i) / 30.0
		// Add some realistic variation
		noise := (float64(i%5) - 2.0) * 200.0
		val := math.Round(startVal + (currentVal-startVal)*ratio + noise)
		pl := val - startVal
		plpc := 0.0
		if startVal > 0 {
			plpc = math.Round((pl/startVal)*10000.0) / 10000.0
		}

		history.Timestamp = append(history.Timestamp, t)
		history.Equity = append(history.Equity, val)
		history.ProfitLoss = append(history.ProfitLoss, pl)
		history.ProfitLossPct = append(history.ProfitLossPct, plpc)
	}

	return history, nil
}
