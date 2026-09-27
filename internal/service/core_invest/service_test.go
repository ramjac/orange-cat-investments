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
}
