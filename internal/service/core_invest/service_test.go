package core_invest_test

import (
	"context"
	"testing"
	"time"

	repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	service "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyPublisher struct {
	publishedTopic string
}

func (p *dummyPublisher) Publish(topic string, payload any) error {
	p.publishedTopic = topic
	return nil
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
		run, err := svc.StartBacktestRun(ctx, "strat-001", start, end, "{}")
		require.NoError(t, err)
		assert.Equal(t, "pending", run.Status)
		assert.Equal(t, "events.investment.backtest_requested.v1", pub.publishedTopic)

		_, err = svc.StartBacktestRun(ctx, "", start, end, "{}")
		assert.Error(t, err)

		_, err = svc.StartBacktestRun(ctx, "strat-001", end, start, "{}")
		assert.Error(t, err)
	})

	t.Run("ListMLModels", func(t *testing.T) {
		models, err := svc.ListMLModels(ctx)
		require.NoError(t, err)
		assert.Len(t, models, 2)

		expectedIDs := []string{"mdl-yolov8-cat-pose-v3", "mdl-whisker-audio-v1"}
		expectedKeys := []string{"model_id", "name", "version", "accuracy", "drift", "status", "created_at"}

		for i, model := range models {
			for _, key := range expectedKeys {
				assert.Contains(t, model, key, "model at index %d should contain key %s", i, key)
			}
			assert.Equal(t, expectedIDs[i], model["model_id"])
		}
	})

	t.Run("LogModelDrift", func(t *testing.T) {
		payload := map[string]interface{}{"model_id": "mdl-yolov8-cat-pose-v3", "metric": "accuracy_drop"}
		res, err := svc.LogModelDrift(ctx, payload)
		require.NoError(t, err)
		assert.Equal(t, "recorded", res["status"])
		assert.Equal(t, "drift-018f-99", res["drift_id"])
		assert.Equal(t, 0.012, res["confidence_drift"])
		assert.Equal(t, payload, res["payload"])
		assert.NotEmpty(t, res["evaluated_at"])
	})

	t.Run("ClassifyAcoustics", func(t *testing.T) {
		res, err := svc.ClassifyAcoustics(ctx, "feline-007")
		require.NoError(t, err)
		assert.Equal(t, "ac-018f-777", res["acoustic_id"])
		assert.Equal(t, "feline-007", res["feline_id"])
		assert.Equal(t, "purring", res["vocalization_type"])
		assert.Equal(t, 28.5, res["frequency_hz"])
		assert.Equal(t, 88.2, res["decibel_level"])
		assert.Equal(t, 0.982, res["confidence"])
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
		order, err := svc.ExecuteBrokerageOrder(ctx, "port-002", "InteractiveBrokers", "ORNG", "buy", 50, 10.0, nil, nil)
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

	t.Run("BacktestRunsQueryMethods", func(t *testing.T) {
		start := time.Now().Add(-24 * time.Hour)
		end := time.Now()
		run, err := svc.StartBacktestRun(ctx, "strat-002", start, end, "{}")
		require.NoError(t, err)

		fetched, err := svc.GetBacktestRun(ctx, run.BacktestID)
		require.NoError(t, err)
		assert.Equal(t, run.BacktestID, fetched.BacktestID)

		_, err = svc.GetBacktestRun(ctx, "")
		assert.Error(t, err)

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
