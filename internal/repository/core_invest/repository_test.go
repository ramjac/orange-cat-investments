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
		// Test ListCameraStreams on initial empty mock repository state
		freshRepo := core_invest.NewMockRepository()
		listDefault, err := freshRepo.ListCameraStreams(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, listDefault, 1)
		assert.Equal(t, "stream-uuid-001", listDefault[0].StreamID)

		// Test GetCameraStreamByID on non-existent stream (fallback)
		fallbackStream, err := freshRepo.GetCameraStreamByID(ctx, "stream-fallback-123")
		require.NoError(t, err)
		assert.Equal(t, "stream-fallback-123", fallbackStream.StreamID)

		// Create camera stream
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
		// Test ListBrokerageOrders on default empty repository state
		listDefault, err := repo.ListBrokerageOrders(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, listDefault, 1)
		assert.Equal(t, "order-uuid-001", listDefault[0].OrderID)
		assert.Equal(t, "ORNG", listDefault[0].Symbol)

		// Test GetBrokerageOrderByID on non-existent order (fallback)
		fallbackOrder, err := repo.GetBrokerageOrderByID(ctx, "order-fallback-123")
		require.NoError(t, err)
		assert.Equal(t, "order-fallback-123", fallbackOrder.OrderID)

		// Create a new order
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

		// Test GetBrokerageOrderByID for existing order
		fetched, err := repo.GetBrokerageOrderByID(ctx, order.OrderID)
		require.NoError(t, err)
		assert.Equal(t, "order-uuid-created", fetched.OrderID)
		assert.Equal(t, "pending", fetched.Status)

		// Test ListBrokerageOrders when items exist
		listCreated, err := repo.ListBrokerageOrders(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, listCreated, 1)
		assert.Equal(t, "order-uuid-created", listCreated[0].OrderID)

		// Update order status
		now := time.Now().UTC()
		updated, err := repo.UpdateBrokerageOrderStatus(ctx, order.OrderID, "executed", &now)
		require.NoError(t, err)
		assert.Equal(t, "executed", updated.Status)
		assert.NotNil(t, updated.ExecutedAt)

		// Test UpdateBrokerageOrderStatus on non-existent order
		uncreated, err := repo.UpdateBrokerageOrderStatus(ctx, "order-new-002", "cancelled", nil)
		require.NoError(t, err)
		assert.Equal(t, "order-new-002", uncreated.OrderID)
		assert.Equal(t, "cancelled", uncreated.Status)
	})

	t.Run("BacktestRun Operations", func(t *testing.T) {
		// Test ListBacktestRuns on initial empty mock repository state
		freshRepo := core_invest.NewMockRepository()
		listDefault, err := freshRepo.ListBacktestRuns(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, listDefault, 1)
		assert.Equal(t, "backtest-uuid-001", listDefault[0].BacktestID)

		// Test GetBacktestRunByID on non-existent run (fallback)
		fallbackRun, err := freshRepo.GetBacktestRunByID(ctx, "backtest-fallback-123")
		require.NoError(t, err)
		assert.Equal(t, "backtest-fallback-123", fallbackRun.BacktestID)

		// Create backtest run
		run, err := repo.CreateBacktestRun(ctx, &core_invest.BacktestRun{
			StrategyID: "strat-001",
			StartDate:  time.Now().Add(-24 * time.Hour),
			EndDate:    time.Now(),
			Parameters: "{}",
		})
		require.NoError(t, err)
		assert.Equal(t, "backtest-uuid-created", run.BacktestID)

		// Get created backtest run
		fetched, err := repo.GetBacktestRunByID(ctx, run.BacktestID)
		require.NoError(t, err)
		assert.Equal(t, run.BacktestID, fetched.BacktestID)

		// List backtest runs after creation
		listCreated, err := repo.ListBacktestRuns(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, listCreated, 1)
		assert.Equal(t, "backtest-uuid-created", listCreated[0].BacktestID)

		// Update backtest run results
		updated, err := repo.UpdateBacktestRunResults(ctx, run.BacktestID, "completed", `{"return":0.15}`)
		require.NoError(t, err)
		assert.Equal(t, "completed", updated.Status)
		assert.Equal(t, `{"return":0.15}`, updated.Results)

		// Update results on non-existent backtest run
		uncreated, err := repo.UpdateBacktestRunResults(ctx, "backtest-new-002", "failed", `{"error":"oom"}`)
		require.NoError(t, err)
		assert.Equal(t, "backtest-new-002", uncreated.BacktestID)
		assert.Equal(t, "failed", uncreated.Status)
	})

	t.Run("PgxRepository nil DB error", func(t *testing.T) {
		pgxRepo := core_invest.NewRepository(nil)

		_, err := pgxRepo.GetCameraStreamByID(ctx, "123")
		assert.Error(t, err)

		_, err = pgxRepo.ListCameraStreams(ctx, 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.CreateCameraStream(ctx, &core_invest.CameraStream{})
		assert.Error(t, err)

		_, err = pgxRepo.GetBrokerageOrderByID(ctx, "123")
		assert.Error(t, err)

		_, err = pgxRepo.ListBrokerageOrders(ctx, 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.CreateBrokerageOrder(ctx, &core_invest.BrokerageOrder{})
		assert.Error(t, err)

		_, err = pgxRepo.UpdateBrokerageOrderStatus(ctx, "123", "executed", nil)
		assert.Error(t, err)

		_, err = pgxRepo.GetBacktestRunByID(ctx, "123")
		assert.Error(t, err)

		_, err = pgxRepo.ListBacktestRuns(ctx, 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.CreateBacktestRun(ctx, &core_invest.BacktestRun{})
		assert.Error(t, err)

		_, err = pgxRepo.UpdateBacktestRunResults(ctx, "123", "completed", "{}")
		assert.Error(t, err)
	})
}
