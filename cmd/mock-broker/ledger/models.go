package ledger

import (
	"strconv"
	"strings"
	"time"
)

// FlexibleFloat64 parses JSON numbers whether sent as float numbers or strings.
type FlexibleFloat64 float64

func (f *FlexibleFloat64) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	if str == "" || str == "null" {
		*f = 0
		return nil
	}
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	*f = FlexibleFloat64(val)
	return nil
}

func (f FlexibleFloat64) Float64() float64 {
	return float64(f)
}

// Account represents an Alpaca Broker customer sub-account.
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
	AccountType      string    `json:"account_type"`
	CustomerName     string    `json:"customer_name,omitempty"`
}

// Order represents an Alpaca trading order.
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

// Position represents an active equity holding in an Alpaca account.
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

// Quote represents latest NBBO quote.
type Quote struct {
	BidPrice  float64   `json:"bp"`
	BidSize   int64     `json:"bs"`
	AskPrice  float64   `json:"ap"`
	AskSize   int64     `json:"as"`
	Timestamp time.Time `json:"t"`
}

// Bar represents an OHLCV candlestick bar.
type Bar struct {
	Timestamp time.Time `json:"t"`
	Open      float64   `json:"o"`
	High      float64   `json:"h"`
	Low       float64   `json:"l"`
	Close     float64   `json:"c"`
	Volume    int64     `json:"v"`
}

// TradeUpdate represents real-time trade event notification.
type TradeUpdate struct {
	Event       string     `json:"event"`
	ExecutionID string     `json:"execution_id"`
	Order       Order      `json:"order"`
	Timestamp   time.Time  `json:"timestamp"`
	Price       *float64   `json:"price,string,omitempty"`
	Qty         *float64   `json:"qty,string,omitempty"`
}
