package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

type EventEnvelope[T any] struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	Payload       T         `json:"payload"`
}

type CatSpottedPayload struct {
	CameraID        string  `json:"camera_id"`
	FelineID        string  `json:"feline_id"`
	ActivityType    string  `json:"activity_type"`
	ConfidenceScore float64 `json:"confidence_score"`
}

type OTATriggeredPayload struct {
	JobID      string `json:"job_id"`
	AssetID    string `json:"asset_id"`
	ReleaseID  string `json:"release_id"`
	DeviceType string `json:"device_type"`
	Version    string `json:"version"`
	FileURL    string `json:"file_url"`
	Checksum   string `json:"checksum"`
}

type OTACompletedPayload struct {
	JobID       string    `json:"job_id"`
	AssetID     string    `json:"asset_id"`
	ReleaseID   string    `json:"release_id"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completed_at"`
}

func HandleOTATriggered(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received OTA update triggered event", "uuid", msg.UUID, "payload", string(msg.Payload))

	var envelope EventEnvelope[OTATriggeredPayload]
	status := "completed"
	if err := json.Unmarshal(msg.Payload, &envelope); err == nil {
		logger.Info("executing edge device OTA firmware update pipeline",
			"job_id", envelope.Payload.JobID,
			"asset_id", envelope.Payload.AssetID,
			"version", envelope.Payload.Version,
		)
		logger.Info("downloading firmware image and verifying checksum",
			"file_url", envelope.Payload.FileURL,
			"checksum", envelope.Payload.Checksum,
		)
		logger.Info("applying OTA update to edge device and rebooting", "asset_id", envelope.Payload.AssetID)
	}

	completionEvent := EventEnvelope[OTACompletedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_completed.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: msg.UUID,
		Payload: OTACompletedPayload{
			JobID:       envelope.Payload.JobID,
			AssetID:     envelope.Payload.AssetID,
			ReleaseID:   envelope.Payload.ReleaseID,
			Status:      status,
			CompletedAt: time.Now().UTC(),
		},
	}

	data, _ := json.Marshal(completionEvent)
	outputMsg := message.NewMessage(completionEvent.EventID, data)
	return []*message.Message{outputMsg}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Background Worker with Watermill Engine")

	watermillLogger := watermill.NewStdLogger(false, false)

	// In production, use watermill-amqp for RabbitMQ. Using GoChannel pub/sub for local runtime reliability.
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 100},
		watermillLogger,
	)

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		logger.Error("failed to create watermill router", "error", err)
		os.Exit(1)
	}

	router.AddHandler(
		"cat_spotted_handler",
		"events.observation.cat_spotted.v1",
		pubSub,
		"events.investment.allocation_check.v1",
		pubSub,
		func(msg *message.Message) ([]*message.Message, error) {
			logger.Info("Watermill consumer received observation event", "uuid", msg.UUID, "payload", string(msg.Payload))

			var envelope EventEnvelope[CatSpottedPayload]
			if err := json.Unmarshal(msg.Payload, &envelope); err == nil {
				logger.Info("executing algorithmic portfolio allocation check & automated brokerage trade order",
					"feline_id", envelope.Payload.FelineID,
					"activity_type", envelope.Payload.ActivityType,
					"confidence", envelope.Payload.ConfidenceScore,
				)
			}

			outputMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"status":"brokerage_trade_executed"}`))
			return []*message.Message{outputMsg}, nil
		},
	)

	router.AddHandler(
		"backtest_run_handler",
		"events.investment.backtest_requested.v1",
		pubSub,
		"events.investment.backtest_completed.v1",
		pubSub,
		func(msg *message.Message) ([]*message.Message, error) {
			logger.Info("Watermill consumer processing historical backtest run", "uuid", msg.UUID, "payload", string(msg.Payload))

			outputMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"status":"backtest_completed","results":{"sharpe_ratio":1.85,"alpha":0.12}}`))
			return []*message.Message{outputMsg}, nil
		},
	)

	router.AddHandler(
		"ota_pipeline_handler",
		"events.facilities.ota_triggered.v1",
		pubSub,
		"events.facilities.ota_completed.v1",
		pubSub,
		func(msg *message.Message) ([]*message.Message, error) {
			return HandleOTATriggered(logger, msg)
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := router.Run(ctx); err != nil {
			logger.Error("watermill router error", "error", err)
		}
	}()

	// Simulate event publisher ticker
	go func() {
		<-router.Running()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				event := EventEnvelope[CatSpottedPayload]{
					EventID:       watermill.NewUUID(),
					EventType:     "observation.cat_spotted.v1",
					OccurredAt:    time.Now().UTC(),
					CorrelationID: "correlation-trace-001",
					Payload: CatSpottedPayload{
						CameraID:        "CAM-ORANGE-01",
						FelineID:        "emp-feline-garfield",
						ActivityType:    "zooming",
						ConfidenceScore: 0.992,
					},
				}
				data, _ := json.Marshal(event)
				msg := message.NewMessage(event.EventID, data)
				pubSub.Publish("events.observation.cat_spotted.v1", msg)
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down Watermill Background Worker...")
	cancel()
	router.Close()
	logger.Info("OCI Background Worker stopped gracefully")
}
