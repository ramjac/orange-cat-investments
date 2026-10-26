package ledger

import (
	"errors"
	"fmt"
	"time"
)

// CreateOrder places an equity order and executes it if it is a market order.
func (s *Store) CreateOrder(accountID string, req CreateOrderRequest, quote Quote) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc, exists := s.accounts[accountID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	now := time.Now().UTC()
	order, err := parseOrderRequest(req, now)
	if err != nil {
		return nil, err
	}

	if s.positions[accountID] == nil {
		s.positions[accountID] = make(map[string]*Position)
	}

	if order.Type == "market" {
		if err := s.executeMarketOrder(acc, accountID, order, quote, now); err != nil {
			return nil, err
		}
	} else if order.Type == "limit" {
		order.Status = "accepted" // Pending limit fill
	}

	s.recalculateAccountValuation(acc, accountID)

	s.orders[order.ID] = order
	s.accountOrders[accountID] = append([]*Order{order}, s.accountOrders[accountID]...)

	s.broadcastTradeUpdateAsync(order, now)

	copyOrder := *order
	return &copyOrder, nil
}

func parseOrderRequest(req CreateOrderRequest, now time.Time) (*Order, error) {
	qty := req.Qty.Float64()
	if req.Symbol == "" || qty <= 0 {
		return nil, ErrInvalidOrderPayload
	}
	if req.Side != "buy" && req.Side != "sell" {
		return nil, errors.New("side must be buy or sell")
	}
	if req.Type == "" {
		req.Type = "market"
	}
	if req.TimeInForce == "" {
		req.TimeInForce = "day"
	}

	var limitPrice *float64
	if req.LimitPrice != nil {
		lp := req.LimitPrice.Float64()
		limitPrice = &lp
	}

	orderID := fmt.Sprintf("ord-%d", now.UnixNano())
	clientOrderID := req.ClientOrderID
	if clientOrderID == "" {
		clientOrderID = fmt.Sprintf("client-%d", now.UnixNano())
	}

	return &Order{
		ID:            orderID,
		ClientOrderID: clientOrderID,
		CreatedAt:     now,
		UpdatedAt:     now,
		SubmittedAt:   now,
		AssetID:       fmt.Sprintf("ast-%s", req.Symbol),
		Symbol:        req.Symbol,
		AssetClass:    "us_equity",
		Qty:           qty,
		FilledQty:     0,
		Type:          req.Type,
		Side:          req.Side,
		TimeInForce:   req.TimeInForce,
		LimitPrice:    limitPrice,
		Status:        "new",
	}, nil
}

func (s *Store) executeMarketOrder(acc *Account, accountID string, order *Order, quote Quote, now time.Time) error {
	if order.Side == "buy" {
		return s.executeMarketBuy(acc, accountID, order, quote, now)
	} else if order.Side == "sell" {
		return s.executeMarketSell(acc, accountID, order, quote, now)
	}
	return nil
}

func (s *Store) executeMarketBuy(acc *Account, accountID string, order *Order, quote Quote, now time.Time) error {
	fillPrice := quote.AskPrice
	if fillPrice <= 0 {
		fillPrice = quote.BidPrice
	}
	if fillPrice <= 0 {
		return errors.New("market quote unavailable for execution")
	}

	totalCost := order.Qty * fillPrice
	if acc.Cash < totalCost {
		return fmt.Errorf("%w: required $%.2f, available $%.2f", ErrInsufficientCash, totalCost, acc.Cash)
	}

	// Debit cash
	acc.Cash -= totalCost

	// Update position
	pos, hasPos := s.positions[accountID][order.Symbol]
	if !hasPos {
		pos = &Position{
			AssetID:       order.AssetID,
			Symbol:        order.Symbol,
			Exchange:      "NASDAQ",
			AssetClass:    "us_equity",
			Qty:           order.Qty,
			AvgEntryPrice: fillPrice,
			Side:          "long",
			CostBasis:     totalCost,
			CurrentPrice:  fillPrice,
			LastdayPrice:  fillPrice * 0.99,
		}
		s.positions[accountID][order.Symbol] = pos
	} else {
		newTotalQty := pos.Qty + order.Qty
		newCostBasis := pos.CostBasis + totalCost
		pos.Qty = newTotalQty
		pos.CostBasis = newCostBasis
		pos.AvgEntryPrice = newCostBasis / newTotalQty
		pos.CurrentPrice = fillPrice
	}

	order.Status = "filled"
	order.FilledQty = order.Qty
	order.FilledAvgPrice = &fillPrice
	order.FilledAt = &now
	return nil
}

func (s *Store) executeMarketSell(acc *Account, accountID string, order *Order, quote Quote, now time.Time) error {
	fillPrice := quote.BidPrice
	if fillPrice <= 0 {
		fillPrice = quote.AskPrice
	}
	if fillPrice <= 0 {
		return errors.New("market quote unavailable for execution")
	}

	pos, hasPos := s.positions[accountID][order.Symbol]
	if !hasPos || pos.Qty < order.Qty {
		avail := 0.0
		if hasPos {
			avail = pos.Qty
		}
		return fmt.Errorf("%w: requested %.4f shares, have %.4f shares", ErrInsufficientShares, order.Qty, avail)
	}

	proceeds := order.Qty * fillPrice
	acc.Cash += proceeds

	pos.Qty -= order.Qty
	pos.CostBasis -= order.Qty * pos.AvgEntryPrice
	if pos.CostBasis < 0 {
		pos.CostBasis = 0
	}
	pos.CurrentPrice = fillPrice

	if pos.Qty <= 0.00001 {
		delete(s.positions[accountID], order.Symbol)
	}

	order.Status = "filled"
	order.FilledQty = order.Qty
	order.FilledAvgPrice = &fillPrice
	order.FilledAt = &now
	return nil
}

func (s *Store) broadcastTradeUpdateAsync(order *Order, now time.Time) {
	tradeUpdate := TradeUpdate{
		Event:       order.Status,
		ExecutionID: fmt.Sprintf("exec-%d", now.UnixNano()),
		Order:       *order,
		Timestamp:   now,
		Price:       order.FilledAvgPrice,
		Qty:         &order.FilledQty,
	}
	go s.BroadcastTradeUpdate(tradeUpdate)
}

func (s *Store) recalculateAccountValuation(acc *Account, accountID string) {
	longMarketValue := 0.0
	for _, pos := range s.positions[accountID] {
		pos.MarketValue = pos.Qty * pos.CurrentPrice
		pos.UnrealizedPL = pos.MarketValue - pos.CostBasis
		if pos.CostBasis > 0 {
			pos.UnrealizedPLPC = pos.UnrealizedPL / pos.CostBasis
		}
		longMarketValue += pos.MarketValue
	}

	acc.LongMarketValue = longMarketValue
	acc.Equity = acc.Cash + acc.LongMarketValue
	acc.PortfolioValue = acc.Equity
	acc.BuyingPower = acc.Cash * 2.0
}
