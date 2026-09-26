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
				logger.Info("executing algorithmic portfolio allocation check", "feline_id", envelope.Payload.FelineID)
			}

			outputMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"status":"allocation_checked"}`))
			return []*message.Message{outputMsg}, nil
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
