package ledger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreAccountAndOrders(t *testing.T) {
	store := NewStore()

	acc := &Account{
		ID:           "test-acc-1",
		Cash:         50000.0,
		CustomerName: "Test Trader",
	}
	err := store.CreateAccount(acc)
	require.NoError(t, err)

	fetched, err := store.GetAccount("test-acc-1")
	require.NoError(t, err)
	assert.Equal(t, 50000.0, fetched.Cash)
	assert.Equal(t, "ACTIVE", fetched.Status)

	quote := Quote{
		BidPrice:  149.95,
		AskPrice:  150.05,
		Timestamp: time.Now().UTC(),
	}

	// 1. Buy Order
	buyReq := CreateOrderRequest{
		Symbol: "AAPL",
		Qty:    10.0,
		Side:   "buy",
		Type:   "market",
	}
	order, err := store.CreateOrder("test-acc-1", buyReq, quote)
	require.NoError(t, err)
	assert.Equal(t, "filled", order.Status)
	assert.Equal(t, 10.0, order.FilledQty)
	assert.Equal(t, 150.05, *order.FilledAvgPrice)

	// Check updated cash
	updatedAcc, _ := store.GetAccount("test-acc-1")
	assert.Equal(t, 50000.0-(10.0*150.05), updatedAcc.Cash)

	// Check position
	pos, err := store.GetPosition("test-acc-1", "AAPL")
	require.NoError(t, err)
	assert.Equal(t, 10.0, pos.Qty)
	assert.Equal(t, 150.05, pos.AvgEntryPrice)

	// 2. Insufficient Cash Buy Order
	failBuy := CreateOrderRequest{
		Symbol: "AAPL",
		Qty:    1000.0, // $150,050 exceeds available cash
		Side:   "buy",
		Type:   "market",
	}
	_, err = store.CreateOrder("test-acc-1", failBuy, quote)
	assert.ErrorIs(t, err, ErrInsufficientCash)

	// 3. Partial Sell Order
	sellQuote := Quote{
		BidPrice: 155.00,
		AskPrice: 155.10,
	}
	sellReq := CreateOrderRequest{
		Symbol: "AAPL",
		Qty:    4.0,
		Side:   "sell",
		Type:   "market",
	}
	sellOrder, err := store.CreateOrder("test-acc-1", sellReq, sellQuote)
	require.NoError(t, err)
	assert.Equal(t, "filled", sellOrder.Status)
	assert.Equal(t, 4.0, sellOrder.FilledQty)

	// Check remaining position
	posAfterSell, err := store.GetPosition("test-acc-1", "AAPL")
	require.NoError(t, err)
	assert.Equal(t, 6.0, posAfterSell.Qty)

	// 4. Oversell
	oversellReq := CreateOrderRequest{
		Symbol: "AAPL",
		Qty:    10.0, // only 6 remaining
		Side:   "sell",
		Type:   "market",
	}
	_, err = store.CreateOrder("test-acc-1", oversellReq, sellQuote)
	assert.ErrorIs(t, err, ErrInsufficientShares)

	// 5. Order listing
	orders := store.ListOrders("test-acc-1", "all", 10)
	assert.Len(t, orders, 2)
}

func TestCreateOrderValidationAndLimit(t *testing.T) {
	store := NewStore()

	acc := &Account{
		ID:   "test-acc-2",
		Cash: 10000.0,
	}
	err := store.CreateAccount(acc)
	require.NoError(t, err)

	quote := Quote{
		BidPrice:  100.0,
		AskPrice:  101.0,
		Timestamp: time.Now().UTC(),
	}

	// Invalid payload: empty symbol
	_, err = store.CreateOrder("test-acc-2", CreateOrderRequest{Symbol: "", Qty: 1, Side: "buy"}, quote)
	assert.ErrorIs(t, err, ErrInvalidOrderPayload)

	// Invalid payload: qty <= 0
	_, err = store.CreateOrder("test-acc-2", CreateOrderRequest{Symbol: "AAPL", Qty: 0, Side: "buy"}, quote)
	assert.ErrorIs(t, err, ErrInvalidOrderPayload)

	// Invalid payload: invalid side
	_, err = store.CreateOrder("test-acc-2", CreateOrderRequest{Symbol: "AAPL", Qty: 1, Side: "hold"}, quote)
	assert.ErrorContains(t, err, "side must be buy or sell")

	// Limit order
	limitPriceVal := FlexibleFloat64(95.0)
	limitReq := CreateOrderRequest{
		Symbol:     "AAPL",
		Qty:        2.0,
		Side:       "buy",
		Type:       "limit",
		LimitPrice: &limitPriceVal,
	}
	limitOrder, err := store.CreateOrder("test-acc-2", limitReq, quote)
	require.NoError(t, err)
	assert.Equal(t, "accepted", limitOrder.Status)
	assert.NotNil(t, limitOrder.LimitPrice)
	assert.Equal(t, 95.0, *limitOrder.LimitPrice)
}

func TestStoreSeedAccounts(t *testing.T) {
	store := NewStore()
	initialPrices := map[string]float64{
		"AAPL": 224.30,
		"NVDA": 138.80,
		"MSFT": 448.20,
		"TSLA": 247.12,
	}

	SeedDefaultAccounts(store, initialPrices)

	// Verify Arthur's account
	arthurID := "018f3a9a-2222-7000-8000-000000000002"
	arthur, err := store.GetAccount(arthurID)
	require.NoError(t, err)
	assert.Equal(t, "Arthur Pendelton", arthur.CustomerName)
	assert.Equal(t, 32100.25, arthur.Cash)

	arthurPositions := store.ListPositions(arthurID)
	assert.Len(t, arthurPositions, 4)

	// Verify Chloe's account
	chloeID := "018f3a9a-2222-7000-8000-000000000003"
	chloe, err := store.GetAccount(chloeID)
	require.NoError(t, err)
	assert.Equal(t, "Chloe Spark", chloe.CustomerName)
	assert.Equal(t, 15400.00, chloe.Cash)

	chloePositions := store.ListPositions(chloeID)
	assert.Len(t, chloePositions, 2)
}
