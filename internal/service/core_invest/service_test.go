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
	svc := service.NewServiceWithPublisher(repository, pub)
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
		order, err := svc.ExecuteBrokerageOrder(ctx, "port-001", "InteractiveBrokers", "ORNG", "buy", 100, 50.0, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "executed", order.Status)

		_, err = svc.ExecuteBrokerageOrder(ctx, "port-001", "Broker", "ORNG", "invalid", 100, 50.0, nil, nil)
		assert.Error(t, err)

		_, err = svc.ExecuteBrokerageOrder(ctx, "port-001", "Broker", "ORNG", "buy", -5, 50.0, nil, nil)
		assert.Error(t, err)
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
}
