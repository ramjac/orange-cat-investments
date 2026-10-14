package alpaca

import (
	"time"
)

type Account struct {
	ID               string    `json:"id"`
	AccountNumber    string    `json:"account_number"`
	Status           string    `json:"status"`
	Currency         string    `json:"currency"`
	Cash             float64   `json:"cash,string"`
	PortfolioValue   float64   `json:"portfolio_value,string"`
	BuyingPower      float64   `json:"buying_power,string"`
	Equity           float64   `json:"equity,string"`
	LongMarketValue  float64   `json:"long_market_value,string"`
	ShortMarketValue float64   `json:"short_market_value,string"`
	LastEquity       float64   `json:"last_equity,string"`
	CreatedAt        time.Time `json:"created_at"`
	CustomerName     string    `json:"customer_name,omitempty"`
}

type Order struct {
	ID             string     `json:"id"`
	ClientOrderID  string     `json:"client_order_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	FilledAt       *time.Time `json:"filled_at,omitempty"`
	CanceledAt     *time.Time `json:"canceled_at,omitempty"`
	AssetID        string     `json:"asset_id"`
	Symbol         string     `json:"symbol"`
	AssetClass     string     `json:"asset_class"`
	Qty            float64    `json:"qty,string"`
	FilledQty      float64    `json:"filled_qty,string"`
	Type           string     `json:"type"`
	Side           string     `json:"side"`
	TimeInForce    string     `json:"time_in_force"`
	LimitPrice     *float64   `json:"limit_price,string,omitempty"`
	FilledAvgPrice *float64   `json:"filled_avg_price,string,omitempty"`
	Status         string     `json:"status"`
}

type CreateOrderRequest struct {
	Symbol        string   `json:"symbol"`
	Qty           float64  `json:"qty"`
	Side          string   `json:"side"`
	Type          string   `json:"type"`
	TimeInForce   string   `json:"time_in_force"`
	LimitPrice    *float64 `json:"limit_price,omitempty"`
	ClientOrderID string   `json:"client_order_id,omitempty"`
}

type Position struct {
	AssetID        string  `json:"asset_id"`
	Symbol         string  `json:"symbol"`
	Exchange       string  `json:"exchange"`
	AssetClass     string  `json:"asset_class"`
	Qty            float64 `json:"qty,string"`
	AvgEntryPrice  float64 `json:"avg_entry_price,string"`
	Side           string  `json:"side"`
	MarketValue    float64 `json:"market_value,string"`
	CostBasis      float64 `json:"cost_basis,string"`
	UnrealizedPL   float64 `json:"unrealized_pl,string"`
	UnrealizedPLPC float64 `json:"unrealized_plpc,string"`
	CurrentPrice   float64 `json:"current_price,string"`
	LastdayPrice   float64 `json:"lastday_price,string"`
	ChangeToday    float64 `json:"change_today,string"`
}

type Quote struct {
	BidPrice  float64   `json:"bp"`
	BidSize   int64     `json:"bs"`
	AskPrice  float64   `json:"ap"`
	AskSize   int64     `json:"as"`
	Timestamp time.Time `json:"t"`
}

type Bar struct {
	Timestamp time.Time `json:"t"`
	Open      float64   `json:"o"`
	High      float64   `json:"h"`
	Low       float64   `json:"l"`
	Close     float64   `json:"c"`
	Volume    int64     `json:"v"`
}

type PortfolioHistory struct {
	Timestamp     []int64   `json:"timestamp"`
	Equity        []float64 `json:"equity"`
	ProfitLoss    []float64 `json:"profit_loss"`
	ProfitLossPct []float64 `json:"profit_loss_pct"`
	BaseValue     float64   `json:"base_value"`
	Timeframe     string    `json:"timeframe"`
}
