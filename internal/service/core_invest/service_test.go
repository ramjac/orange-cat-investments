package core_invest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	service "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyPublisher struct {
	publishedTopic   string
	publishedPayload any
	err              error
}

func (p *dummyPublisher) Publish(topic string, payload any) error {
	p.publishedTopic = topic
	p.publishedPayload = payload
	return p.err
}

type errRepo struct {
	repo.Repository
}

func (e *errRepo) CreateBacktestRun(ctx context.Context, run *repo.BacktestRun) (*repo.BacktestRun, error) {
	return nil, errors.New("db create error")
}

func (e *errRepo) GetBacktestRunByID(ctx context.Context, id string) (*repo.BacktestRun, error) {
	return nil, errors.New("db get error")
}

func TestCoreInvestService(t *testing.T) {
	repository := repo.NewMockRepository()
	pub := &dummyPublisher{}
	mockBroker := &service.MockBrokerClient{FillPrice: 50.0, Status: "executed"}
	svc := service.NewServiceWithAll(repository, pub, mockBroker, nil)
	ctx := context.Background()

	t.Run("RegisterCameraStream", func(t *testing.T) {
		stream, err := svc.RegisterCameraStream(ctx, "cam-01", "rtsp://camera/stream", "rtsp")
		require.NoError(t, err)
		assert.Equal(t, "stream-uuid-created", stream.StreamID)

		_, err = svc.RegisterCameraStream(ctx, "", "rtsp://camera/stream", "rtsp")
		assert.Error(t, err)

		_, err = svc.RegisterCameraStream(ctx, "cam-01", "rtsp://camera/stream", "invalid-proto")
		assert.Error(t, err)
	})

	t.Run("ExecuteBrokerageOrder", func(t *testing.T) {
		order, err := svc.ExecuteBrokerageOrder(ctx, service.ExecuteBrokerageOrderOpts{
			PortfolioID: "port-001",
			BrokerName:  "InteractiveBrokers",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    100,
			Price:       50.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "executed", order.Status)

		_, err = svc.ExecuteBrokerageOrder(ctx, service.ExecuteBrokerageOrderOpts{
			PortfolioID: "port-001",
			BrokerName:  "Broker",
			Symbol:      "ORNG",
			Side:        "invalid",
			Quantity:    100,
			Price:       50.0,
		})
		assert.Error(t, err)

		_, err = svc.ExecuteBrokerageOrder(ctx, service.ExecuteBrokerageOrderOpts{
			PortfolioID: "port-001",
			BrokerName:  "Broker",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    -5,
			Price:       50.0,
		})
		assert.Error(t, err)

		noBrokerSvc := service.NewServiceWithPublisher(repository, pub)
		_, err = noBrokerSvc.ExecuteBrokerageOrder(ctx, service.ExecuteBrokerageOrderOpts{
			PortfolioID: "port-001",
			BrokerName:  "Broker",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    10,
			Price:       50.0,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "broker client is not configured: order execution unavailable")
	})

	t.Run("ExecuteBrokerageOrderWithBrokerClient", func(t *testing.T) {
		mockBroker := &service.MockBrokerClient{
			FillPrice: 228.45,
			Status:    "filled",
		}
		brokerSvc := service.NewServiceWithBroker(repository, mockBroker)
		order, err := brokerSvc.ExecuteBrokerageOrder(ctx, service.ExecuteBrokerageOrderOpts{
			PortfolioID: "port-001",
			BrokerName:  "Alpaca",
			Symbol:      "AAPL",
			Side:        "buy",
			Quantity:    15,
			Price:       225.0,
		})
		require.NoError(t, err)
		assert.Equal(t, "filled", order.Status)
		assert.Equal(t, 228.45, order.Price)
	})

	t.Run("StartBacktestRun", func(t *testing.T) {
		start := time.Now().Add(-24 * time.Hour)
		end := time.Now()

		t.Run("Success with custom parameters and publisher", func(t *testing.T) {
			p := &dummyPublisher{}
			s := service.NewServiceWithPublisher(repository, p)
			run, err := s.StartBacktestRun(ctx, "strat-001", start, end, `{"lookback":30}`)
			require.NoError(t, err)
			assert.Equal(t, "pending", run.Status)
			assert.Equal(t, "strat-001", run.StrategyID)
			assert.Equal(t, `{"lookback":30}`, run.Parameters)
			assert.Equal(t, "events.investment.backtest_requested.v1", p.publishedTopic)
			payloadMap, ok := p.publishedPayload.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, run.BacktestID, payloadMap["backtest_id"])
			assert.Equal(t, "strat-001", payloadMap["strategy_id"])
		})

		t.Run("Success with empty parameters defaulting to json object", func(t *testing.T) {
			s := service.NewService(repository)
			run, err := s.StartBacktestRun(ctx, "strat-002", start, end, "")
			require.NoError(t, err)
			assert.Equal(t, "{}", run.Parameters)
		})

		t.Run("Missing strategy_id error", func(t *testing.T) {
			_, err := svc.StartBacktestRun(ctx, "", start, end, "{}")
			require.Error(t, err)
			assert.Equal(t, "strategy_id is required", err.Error())
		})

		t.Run("end_date before start_date error", func(t *testing.T) {
			_, err := svc.StartBacktestRun(ctx, "strat-001", end, start, "{}")
			require.Error(t, err)
			assert.Equal(t, "end_date must be after start_date", err.Error())
		})

		t.Run("Repository create error", func(t *testing.T) {
			failingSvc := service.NewService(&errRepo{Repository: repository})
			_, err := failingSvc.StartBacktestRun(ctx, "strat-001", start, end, "{}")
			require.Error(t, err)
			assert.Equal(t, "db create error", err.Error())
		})
	})

	t.Run("GetBacktestRun", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			run, err := svc.GetBacktestRun(ctx, "backtest-001")
			require.NoError(t, err)
			assert.Equal(t, "backtest-001", run.BacktestID)
		})

		t.Run("Empty ID error", func(t *testing.T) {
			_, err := svc.GetBacktestRun(ctx, "")
			require.Error(t, err)
			assert.Equal(t, "backtest id cannot be empty", err.Error())
		})

		t.Run("Repository get error", func(t *testing.T) {
			failingSvc := service.NewService(&errRepo{Repository: repository})
			_, err := failingSvc.GetBacktestRun(ctx, "backtest-001")
			require.Error(t, err)
			assert.Equal(t, "db get error", err.Error())
		})
	})

	t.Run("ListBacktestRuns", func(t *testing.T) {
		runs, err := svc.ListBacktestRuns(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, runs)
	})

	t.Run("GenerateStatement", func(t *testing.T) {
		stmt, err := svc.GenerateStatement(ctx, "port-001", 2024)
		require.NoError(t, err)
		assert.Equal(t, 2024, stmt["year"])
		assert.Equal(t, "port-001", stmt["portfolio_id"])
		assert.Equal(t, "https://statements.oci.local/documents/port-001/2024/form1099b-port-001-2024.pdf", stmt["form_1099b_url"])

		// Invalid portfolio
		_, err = svc.GenerateStatement(ctx, "", 2024)
		assert.Error(t, err)

		// Invalid year
		_, err = svc.GenerateStatement(ctx, "port-001", 1990)
		assert.Error(t, err)
	})

	t.Run("ListMLModels", func(t *testing.T) {
		models, err := svc.ListMLModels(ctx)
		require.NoError(t, err)
		assert.Len(t, models, 1)
		assert.Equal(t, "mdl-yolov8-cat-pose-v3", models[0]["model_id"])

		expectedKeys := []string{"model_id", "name", "version", "accuracy", "drift", "status", "created_at"}
		for _, key := range expectedKeys {
			assert.Contains(t, models[0], key)
		}
	})

	t.Run("LogModelDrift", func(t *testing.T) {
		payload := map[string]interface{}{
			"model_id":    "mdl-yolov8-cat-pose-v3",
			"metric":      "accuracy",
			"drift_value": 0.012,
		}

		res, err := svc.LogModelDrift(ctx, payload)
		require.NoError(t, err)
		assert.Equal(t, "drift-018f-99", res["drift_id"])
		assert.Equal(t, "recorded", res["status"])
		assert.Equal(t, 0.012, res["confidence_drift"])
		assert.NotEmpty(t, res["evaluated_at"])
		assert.Equal(t, payload, res["payload"])

		// Nil payload test case
		resNil, err := svc.LogModelDrift(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "drift-018f-99", resNil["drift_id"])
		assert.Nil(t, resNil["payload"])
	})

	t.Run("CameraStreamsQueryMethods", func(t *testing.T) {
		stream, err := svc.RegisterCameraStream(ctx, "cam-02", "rtsp://camera2/stream", "rtsp")
		require.NoError(t, err)

		fetched, err := svc.GetCameraStream(ctx, stream.StreamID)
		require.NoError(t, err)
		assert.Equal(t, stream.StreamID, fetched.StreamID)

		_, err = svc.GetCameraStream(ctx, "")
		assert.Error(t, err)

		streams, err := svc.ListCameraStreams(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, streams)
	})

	t.Run("BrokerageOrdersQueryMethods", func(t *testing.T) {
		order, err := svc.ExecuteBrokerageOrder(ctx, service.ExecuteBrokerageOrderOpts{
			PortfolioID: "port-002",
			BrokerName:  "InteractiveBrokers",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    50,
			Price:       10.0,
		})
		require.NoError(t, err)

		fetched, err := svc.GetBrokerageOrder(ctx, order.OrderID)
		require.NoError(t, err)
		assert.Equal(t, order.OrderID, fetched.OrderID)

		_, err = svc.GetBrokerageOrder(ctx, "")
		assert.Error(t, err)

		orders, err := svc.ListBrokerageOrders(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, orders)
	})

	t.Run("CreateRecurringDeposit_And_List", func(t *testing.T) {
		dep, err := svc.CreateRecurringDeposit(ctx, service.CreateRecurringDepositOpts{
			CustomerID:  "cust-longterm-arthur",
			PortfolioID: "018f3a9a-2222-7000-8000-000000000002",
			AmountUSD:   500.00,
			Frequency:   "monthly",
			BankAccount: "Chase Checking (****4821)",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, dep.DepositID)
		assert.Equal(t, 500.00, dep.AmountUSD)

		list, err := svc.ListRecurringDeposits(ctx, "cust-longterm-arthur")
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})
}

func TestCoreInvestService_WithBankIntegration(t *testing.T) {
	repository := repo.NewMockRepository()
	ctx := context.Background()

	// 1. Approved bank client
	bankOk := &service.MockBankClient{AllowTransfer: true}
	svcOk := service.NewServiceWithIntegrations(repository, nil, bankOk)

	dep, err := svcOk.CreateRecurringDeposit(ctx, service.CreateRecurringDepositOpts{
		CustomerID:  "cust-longterm-arthur",
		PortfolioID: "port-001",
		AmountUSD:   250.00,
		AccountID:   "acc_arthur_chase_chk",
	})
	require.NoError(t, err)
	assert.Equal(t, 250.00, dep.AmountUSD)

	// 2. Declined bank client (NSF)
	bankDeclined := &service.MockBankClient{AllowTransfer: false}
	svcDeclined := service.NewServiceWithIntegrations(repository, nil, bankDeclined)

	_, err = svcDeclined.CreateRecurringDeposit(ctx, service.CreateRecurringDepositOpts{
		CustomerID:  "cust-active-chloe",
		PortfolioID: "port-002",
		AmountUSD:   100000.00,
		AccountID:   "acc_chloe_svcu_chk",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bank declined deposit authorization")
}


