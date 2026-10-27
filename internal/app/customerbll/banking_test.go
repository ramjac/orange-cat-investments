package customerbll

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient

	createRecurringDepositFunc func(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error)
	listRecurringDepositsFunc  func(ctx context.Context, in *coreinvestv1.ListRecurringDepositsRequest) (*coreinvestv1.ListRecurringDepositsResponse, error)
	updatePortfolioBalanceFunc func(ctx context.Context, in *coreinvestv1.UpdatePortfolioBalanceRequest) (*coreinvestv1.UpdatePortfolioBalanceResponse, error)
	advanceRecurringDepositFunc func(ctx context.Context, in *coreinvestv1.AdvanceRecurringDepositRequest) (*coreinvestv1.AdvanceRecurringDepositResponse, error)
}

func (m *mockCoreInvestClient) CreateRecurringDeposit(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error) {
	if m.createRecurringDepositFunc != nil {
		return m.createRecurringDepositFunc(ctx, in)
	}
	return &coreinvestv1.CreateRecurringDepositResponse{
		Deposit: &coreinvestv1.RecurringDeposit{
			DepositID:   "dep-mock-01",
			CustomerID:  in.CustomerID,
			PortfolioID: in.PortfolioID,
			AmountUSD:   in.AmountUSD,
			Frequency:   in.Frequency,
			BankAccount: in.BankAccount,
			Status:      "active",
		},
	}, nil
}

func (m *mockCoreInvestClient) ListRecurringDeposits(ctx context.Context, in *coreinvestv1.ListRecurringDepositsRequest) (*coreinvestv1.ListRecurringDepositsResponse, error) {
	if m.listRecurringDepositsFunc != nil {
		return m.listRecurringDepositsFunc(ctx, in)
	}
	return &coreinvestv1.ListRecurringDepositsResponse{}, nil
}

func (m *mockCoreInvestClient) UpdatePortfolioBalance(ctx context.Context, in *coreinvestv1.UpdatePortfolioBalanceRequest) (*coreinvestv1.UpdatePortfolioBalanceResponse, error) {
	if m.updatePortfolioBalanceFunc != nil {
		return m.updatePortfolioBalanceFunc(ctx, in)
	}
	return &coreinvestv1.UpdatePortfolioBalanceResponse{}, nil
}

func (m *mockCoreInvestClient) AdvanceRecurringDeposit(ctx context.Context, in *coreinvestv1.AdvanceRecurringDepositRequest) (*coreinvestv1.AdvanceRecurringDepositResponse, error) {
	if m.advanceRecurringDepositFunc != nil {
		return m.advanceRecurringDepositFunc(ctx, in)
	}
	return &coreinvestv1.AdvanceRecurringDepositResponse{}, nil
}

type mockPublisher struct {
	publishedMessages map[string][]*message.Message
	publishErr        error
}

func newMockPublisher() *mockPublisher {
	return &mockPublisher{
		publishedMessages: make(map[string][]*message.Message),
	}
}

func (m *mockPublisher) Publish(topic string, messages ...*message.Message) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishedMessages[topic] = append(m.publishedMessages[topic], messages...)
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func TestInitiateACHDeposit_Validation(t *testing.T) {
	s := NewServer(Config{})

	tests := []struct {
		name string
		req  *customerbllv1.InitiateACHDepositRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing customer_id",
			req: &customerbllv1.InitiateACHDepositRequest{
				CustomerID: "",
				AmountUSD:  100.0,
			},
		},
		{
			name: "zero amount",
			req: &customerbllv1.InitiateACHDepositRequest{
				CustomerID: "cust-001",
				AmountUSD:  0,
			},
		},
		{
			name: "negative amount",
			req: &customerbllv1.InitiateACHDepositRequest{
				CustomerID: "cust-001",
				AmountUSD:  -50.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := s.InitiateACHDeposit(context.Background(), tt.req)
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestInitiateACHDeposit_PortfolioResolution(t *testing.T) {
	t.Run("explicit portfolio_id provided", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID:  "cust-123",
			PortfolioID: "port-explicit-99",
			AmountUSD:   200.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "port-explicit-99", resp.PortfolioID)
	})

	t.Run("broker match by account ID", func(t *testing.T) {
		brokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]alpaca.Account{
				{ID: "acc-matched-id", CustomerName: "Other Name", AccountNumber: "AC100"},
			})
		}))
		defer brokerServer.Close()

		s := NewServer(Config{
			BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: brokerServer.URL}),
		})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "acc-matched-id",
			AmountUSD:  200.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "acc-matched-id", resp.PortfolioID)
	})

	t.Run("broker match by account number", func(t *testing.T) {
		brokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]alpaca.Account{
				{ID: "port-from-accnum", CustomerName: "Other Name", AccountNumber: "AC-NUM-777"},
			})
		}))
		defer brokerServer.Close()

		s := NewServer(Config{
			BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: brokerServer.URL}),
		})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "AC-NUM-777",
			AmountUSD:  200.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "port-from-accnum", resp.PortfolioID)
	})

	t.Run("broker match by customer name substring", func(t *testing.T) {
		brokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]alpaca.Account{
				{ID: "port-from-name", CustomerName: "Garfield Cat", AccountNumber: "AC101"},
			})
		}))
		defer brokerServer.Close()

		s := NewServer(Config{
			BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: brokerServer.URL}),
		})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "Garfield",
			AmountUSD:  200.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "port-from-name", resp.PortfolioID)
	})

	t.Run("broker fallback to first account when no match", func(t *testing.T) {
		brokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]alpaca.Account{
				{ID: "first-account-id", CustomerName: "Unrelated Name", AccountNumber: "AC999"},
			})
		}))
		defer brokerServer.Close()

		s := NewServer(Config{
			BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: brokerServer.URL}),
		})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "no-match-cust",
			AmountUSD:  200.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "first-account-id", resp.PortfolioID)
	})

	t.Run("fallback default format port-<CustomerID> when broker is nil or errors", func(t *testing.T) {
		s := NewServer(Config{})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "cat-55",
			AmountUSD:  150.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "port-cat-55", resp.PortfolioID)
	})
}

func TestInitiateACHDeposit_BankAndFrequencyResolution(t *testing.T) {
	t.Run("fetch bank account from bank client when account_id is empty", func(t *testing.T) {
		bankServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"accounts": []plaid.Account{
					{AccountID: "bank-acc-found-123", Name: "Checking"},
				},
			})
		}))
		defer bankServer.Close()

		s := NewServer(Config{
			BankClient: plaid.NewClient(plaid.Config{BaseURL: bankServer.URL}),
		})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "cust-001",
			AmountUSD:  100.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "pending_settlement", resp.Status)
		assert.Equal(t, "one_time", resp.Frequency)
	})

	t.Run("default frequency to one_time when empty", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "cust-001",
			AmountUSD:  100.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "one_time", resp.Frequency)
	})
}

func TestInitiateACHDeposit_CoreInvestIntegration(t *testing.T) {
	t.Run("successful core-invest deposit creation", func(t *testing.T) {
		ciMock := &mockCoreInvestClient{}
		pub := newMockPublisher()

		s := NewServer(Config{
			Publisher: pub,
		})
		s.SetCoreInvestClient(ciMock)

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID:  "cust-100",
			PortfolioID: "port-100",
			BankAccount: "bank-act-100",
			AmountUSD:   350.0,
			Frequency:   "monthly",
		})
		require.NoError(t, err)
		assert.NotNil(t, resp.Deposit)
		assert.Equal(t, "dep-mock-01", resp.Deposit.DepositID)
		assert.Equal(t, "cust-100", resp.Deposit.CustomerID)
		assert.Equal(t, "port-100", resp.Deposit.PortfolioID)
		assert.Equal(t, 350.0, resp.Deposit.AmountUSD)
		assert.Equal(t, "monthly", resp.Deposit.Frequency)

		// Verify event publishing
		published := pub.publishedMessages[envelope.TopicACHTransferRequested]
		require.Len(t, published, 1)
		var env envelope.EventEnvelope[map[string]interface{}]
		err = json.Unmarshal(published[0].Payload, &env)
		require.NoError(t, err)
		assert.Equal(t, envelope.TopicACHTransferRequested, env.EventType)
		assert.Equal(t, "cust-100", env.Payload["customer_id"])
		assert.Equal(t, "port-100", env.Payload["portfolio_id"])
		assert.Equal(t, 350.0, env.Payload["amount_usd"])
		assert.Equal(t, "deposit", env.Payload["transfer_type"])
	})

	t.Run("core-invest error returns InvalidArgument status error", func(t *testing.T) {
		ciMock := &mockCoreInvestClient{
			createRecurringDepositFunc: func(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error) {
				return nil, errors.New("invalid deposit schedule parameters")
			},
		}

		s := NewServer(Config{})
		s.SetCoreInvestClient(ciMock)

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID: "cust-100",
			AmountUSD:  350.0,
		})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "invalid deposit schedule parameters")
	})

	t.Run("fallback createdDep when core-invest client is nil", func(t *testing.T) {
		s := NewServer(Config{})

		resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
			CustomerID:  "cust-fallback",
			PortfolioID: "port-fallback",
			BankAccount: "bank-01",
			AmountUSD:   500.0,
			Frequency:   "weekly",
		})
		require.NoError(t, err)
		assert.NotNil(t, resp.Deposit)
		assert.Contains(t, resp.Deposit.DepositID, "dep-")
		assert.Equal(t, "active", resp.Deposit.Status)
		assert.Equal(t, "weekly", resp.Deposit.Frequency)
	})
}

func TestInitiateACHDeposit_PublisherError(t *testing.T) {
	pub := newMockPublisher()
	pub.publishErr = errors.New("rabbitmq connection down")

	s := NewServer(Config{
		Publisher: pub,
	})

	resp, err := s.InitiateACHDeposit(context.Background(), &customerbllv1.InitiateACHDepositRequest{
		CustomerID: "cust-pub-err",
		AmountUSD:  100.0,
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "failed to publish ACH transfer event to message broker")
}
