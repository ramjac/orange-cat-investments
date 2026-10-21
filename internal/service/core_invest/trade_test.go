package core_invest_test

import (
	"context"
	"testing"

	repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	service "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingBrokerClient struct {
	createdOrders []alpaca.CreateOrderRequest
	accountIDs    []string
	fillPrice     float64
	status        string
}

func (r *recordingBrokerClient) CreateOrder(ctx context.Context, accountID string, req alpaca.CreateOrderRequest) (*alpaca.Order, error) {
	r.accountIDs = append(r.accountIDs, accountID)
	r.createdOrders = append(r.createdOrders, req)

	price := r.fillPrice
	if price == 0 {
		price = 150.0
	}
	status := r.status
	if status == "" {
		status = "executed"
	}

	return &alpaca.Order{
		ID:             "broker-ord-123",
		ClientOrderID:  req.ClientOrderID,
		Symbol:         req.Symbol,
		Qty:            req.Qty,
		Side:           req.Side,
		Type:           req.Type,
		Status:         status,
		FilledAvgPrice: &price,
	}, nil
}

func (r *recordingBrokerClient) GetAccount(ctx context.Context, accountID string) (*alpaca.Account, error) {
	return &alpaca.Account{ID: accountID, Status: "ACTIVE"}, nil
}

func (r *recordingBrokerClient) ListPositions(ctx context.Context, accountID string) ([]alpaca.Position, error) {
	return []alpaca.Position{}, nil
}

func (r *recordingBrokerClient) ListOrders(ctx context.Context, accountID string, status string, limit int) ([]alpaca.Order, error) {
	return []alpaca.Order{}, nil
}

func TestTradeExecutionWithBrokerageAPI(t *testing.T) {
	repository := repo.NewMockRepository()
	recordingBroker := &recordingBrokerClient{
		fillPrice: 155.50,
		status:    "executed",
	}

	svc := service.NewServiceWithBroker(repository, recordingBroker)
	ctx := context.Background()

	opts := service.ExecuteBrokerageOrderOpts{
		PortfolioID: "port-trade-001",
		BrokerName:  "AlpacaBroker",
		Symbol:      "ORNG",
		Side:        "buy",
		Quantity:    25.0,
		Price:       150.0,
	}

	// Execute brokerage order without reliance on dummy sleep delays
	order, err := svc.ExecuteBrokerageOrder(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, order)

	// Verify order status and filled price in DB persistence
	assert.Equal(t, "port-trade-001", order.PortfolioID)
	assert.Equal(t, "ORNG", order.Symbol)
	assert.Equal(t, "buy", order.Side)
	assert.Equal(t, 25.0, order.Quantity)
	assert.Equal(t, 155.50, order.Price)
	assert.Equal(t, "executed", order.Status)

	// Verify call to brokerage API was properly formatted and routed
	require.Len(t, recordingBroker.createdOrders, 1)
	assert.Equal(t, "port-trade-001", recordingBroker.accountIDs[0])
	assert.Equal(t, "ORNG", recordingBroker.createdOrders[0].Symbol)
	assert.Equal(t, 25.0, recordingBroker.createdOrders[0].Qty)
	assert.Equal(t, "buy", recordingBroker.createdOrders[0].Side)
	assert.Equal(t, "market", recordingBroker.createdOrders[0].Type)
	assert.Equal(t, order.OrderID, recordingBroker.createdOrders[0].ClientOrderID)

	// Verify persistence in repository by fetching the order back
	fetchedOrder, err := svc.GetBrokerageOrder(ctx, order.OrderID)
	require.NoError(t, err)
	assert.Equal(t, order.OrderID, fetchedOrder.OrderID)
	assert.Equal(t, "executed", fetchedOrder.Status)
}
