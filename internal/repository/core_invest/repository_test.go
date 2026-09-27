package core_invest_test

import (
	"context"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoreInvestRepository(t *testing.T) {
	repo := core_invest.NewMockRepository()
	ctx := context.Background()

	t.Run("CameraStream Operations", func(t *testing.T) {
		stream, err := repo.CreateCameraStream(ctx, &core_invest.CameraStream{
			CameraAssetID: "cam-001",
			StreamURL:     "rtsp://test/stream",
			Protocol:      "rtsp",
			Status:        "active",
		})
		require.NoError(t, err)
		assert.Equal(t, "stream-uuid-created", stream.StreamID)

		fetched, err := repo.GetCameraStreamByID(ctx, stream.StreamID)
		require.NoError(t, err)
		assert.Equal(t, stream.StreamID, fetched.StreamID)

		list, err := repo.ListCameraStreams(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})

	t.Run("BrokerageOrder Operations", func(t *testing.T) {
		order, err := repo.CreateBrokerageOrder(ctx, &core_invest.BrokerageOrder{
			PortfolioID: "port-001",
			BrokerName:  "InteractiveBrokers",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    50,
			Price:       100,
			Status:      "pending",
		})
		require.NoError(t, err)
		assert.Equal(t, "order-uuid-created", order.OrderID)

		now := time.Now().UTC()
		updated, err := repo.UpdateBrokerageOrderStatus(ctx, order.OrderID, "executed", &now)
		require.NoError(t, err)
		assert.Equal(t, "executed", updated.Status)
	})

	t.Run("BacktestRun Operations", func(t *testing.T) {
		run, err := repo.CreateBacktestRun(ctx, &core_invest.BacktestRun{
			StrategyID: "strat-001",
			StartDate:  time.Now().Add(-24 * time.Hour),
			EndDate:    time.Now(),
			Parameters: "{}",
		})
		require.NoError(t, err)
		assert.Equal(t, "backtest-uuid-created", run.BacktestID)

		updated, err := repo.UpdateBacktestRunResults(ctx, run.BacktestID, "completed", `{"return":0.15}`)
		require.NoError(t, err)
		assert.Equal(t, "completed", updated.Status)
	})

	t.Run("PgxRepository nil DB error", func(t *testing.T) {
		pgxRepo := core_invest.NewRepository(nil)
		_, err := pgxRepo.GetCameraStreamByID(ctx, "123")
		assert.Error(t, err)
	})
}
