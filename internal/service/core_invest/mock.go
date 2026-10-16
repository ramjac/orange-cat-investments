package core_invest

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
)

// MockBrokerClient implements BrokerClient for unit and integration tests.
type MockBrokerClient struct {
	FillPrice     float64
	Status        string
	CreateOrderFn func(ctx context.Context, accountID string, req alpaca.CreateOrderRequest) (*alpaca.Order, error)
}

func (m *MockBrokerClient) CreateOrder(ctx context.Context, accountID string, req alpaca.CreateOrderRequest) (*alpaca.Order, error) {
	if m.CreateOrderFn != nil {
		return m.CreateOrderFn(ctx, accountID, req)
	}
	price := m.FillPrice
	if price <= 0 {
		price = 50.0
	}
	status := m.Status
	if status == "" {
		status = "executed"
	}
	now := time.Now().UTC()
	return &alpaca.Order{
		ID:             "mock-ord-1",
		Symbol:         req.Symbol,
		Qty:            req.Qty,
		FilledQty:      req.Qty,
		FilledAvgPrice: &price,
		Status:         status,
		FilledAt:       &now,
	}, nil
}

func (m *MockBrokerClient) GetAccount(ctx context.Context, accountID string) (*alpaca.Account, error) {
	return &alpaca.Account{ID: accountID, Status: "ACTIVE"}, nil
}

func (m *MockBrokerClient) ListPositions(ctx context.Context, accountID string) ([]alpaca.Position, error) {
	return []alpaca.Position{}, nil
}

func (m *MockBrokerClient) ListOrders(ctx context.Context, accountID string, status string, limit int) ([]alpaca.Order, error) {
	return []alpaca.Order{}, nil
}

// MockBankClient implements BankClient for unit and integration tests.
type MockBankClient struct {
	AllowTransfer bool
}

func (m *MockBankClient) CreateTransfer(ctx context.Context, req plaid.CreateTransferRequest) (*plaid.Transfer, error) {
	return &plaid.Transfer{TransferID: "tr-test-1", Status: "pending", Amount: req.Amount}, nil
}

func (m *MockBankClient) GetTransfer(ctx context.Context, transferID string) (*plaid.Transfer, error) {
	return &plaid.Transfer{TransferID: transferID, Status: "settled"}, nil
}

func (m *MockBankClient) AuthorizeTransfer(ctx context.Context, accountID string, amount float64, transferType string) (*plaid.TransferAuthorization, error) {
	if m.AllowTransfer {
		return &plaid.TransferAuthorization{ID: "auth-1", Decision: "approved"}, nil
	}
	return &plaid.TransferAuthorization{
		ID:       "auth-2",
		Decision: "declined",
		DecisionRationale: &plaid.DecisionRationale{
			Code:        "NSF",
			Description: "insufficient funds",
		},
	}, nil
}

func (m *MockBankClient) GetBalances(ctx context.Context, accessToken string, accountIDs []string) ([]plaid.AccountWithBalance, error) {
	return nil, nil
}

func (m *MockBankClient) GetAccounts(ctx context.Context, ownerID string) ([]plaid.Account, error) {
	return nil, nil
}

func (m *MockBankClient) ExchangePublicToken(ctx context.Context, publicToken, clientUserID string) (string, error) {
	return "mock-access-token-12345", nil
}

