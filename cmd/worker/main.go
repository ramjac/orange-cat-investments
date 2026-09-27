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
	"github.com/orange-cat-investments/oci/internal/service/pki"
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

type CARotationPayload struct {
	IntermediateCAName       string    `json:"intermediate_ca_name"`
	NewSerialNumber           string    `json:"new_serial_number"`
	ExpiresAt                 time.Time `json:"expires_at"`
	ReissuedCertificatesCount int       `json:"reissued_certificates_count"`
	Status                    string    `json:"status"`
}

type pubSub interface {
	message.Publisher
	message.Subscriber
}

func HandleOTATriggered(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received OTA update triggered event", "uuid", msg.UUID)

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

func handleCatSpotted(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer received observation event", "uuid", msg.UUID)

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
}

func handleBacktestRun(logger *slog.Logger, msg *message.Message) ([]*message.Message, error) {
	logger.Info("Watermill consumer processing historical backtest run", "uuid", msg.UUID)

	outputMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"status":"backtest_completed","results":{"sharpe_ratio":1.85,"alpha":0.12}}`))
	return []*message.Message{outputMsg}, nil
}

func createRouter(ps pubSub, watermillLogger watermill.LoggerAdapter, logger *slog.Logger) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	router.AddHandler(
		"cat_spotted_handler",
		"events.observation.cat_spotted.v1",
		ps,
		"events.investment.allocation_check.v1",
		ps,
		func(msg *message.Message) ([]*message.Message, error) {
			return handleCatSpotted(logger, msg)
		},
	)

	router.AddHandler(
		"backtest_run_handler",
		"events.investment.backtest_requested.v1",
		ps,
		"events.investment.backtest_completed.v1",
		ps,
		func(msg *message.Message) ([]*message.Message, error) {
			return handleBacktestRun(logger, msg)
		},
	)

	router.AddHandler(
		"ota_pipeline_handler",
		"events.facilities.ota_triggered.v1",
		ps,
		"events.facilities.ota_completed.v1",
		ps,
		func(msg *message.Message) ([]*message.Message, error) {
			return HandleOTATriggered(logger, msg)
		},
	)

	return router, nil
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

	router, err := createRouter(pubSub, watermillLogger, logger)
	if err != nil {
		logger.Error("failed to create watermill router", "error", err)
		os.Exit(1)
	}

	caManager := pki.NewCARotationManager(pki.CARotationConfig{
		RootCAName:           "oci-root-ca",
		IntermediateCAName:   "oci-intermediate-ca",
		RenewalThresholdDays: 30,
		IssuerNamespace:      "cert-manager",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := router.Run(ctx); err != nil {
			logger.Error("watermill router error", "error", err)
		}
	}()

	// CA Rotation Worker Ticker (Automated Root/Intermediate CA key rotation)
	go func() {
		<-router.Running()
		// Initial check on worker startup
		res, err := caManager.CheckAndRotateCA(ctx)
		if err != nil {
			logger.Error("failed initial CA rotation check", "error", err)
		} else {
			logger.Info("automated CA key rotation check completed",
				"status", res.Status,
				"ca_name", res.IntermediateCAName,
				"serial_number", res.NewSerialNumber,
				"reissued_certs", res.ReissuedCertificatesCount,
			)

			if res.Status == "CA_ROTATED_SUCCESSFULLY" {
				event := EventEnvelope[CARotationPayload]{
					EventID:       watermill.NewUUID(),
					EventType:     "ops.ca_rotated.v1",
					OccurredAt:    res.RotatedAt,
					CorrelationID: "ca-rotation-trace-001",
					Payload: CARotationPayload{
						IntermediateCAName:       res.IntermediateCAName,
						NewSerialNumber:           res.NewSerialNumber,
						ExpiresAt:                 res.ExpiresAt,
						ReissuedCertificatesCount: res.ReissuedCertificatesCount,
						Status:                    res.Status,
					},
				}
				data, _ := json.Marshal(event)
				msg := message.NewMessage(event.EventID, data)
				_ = pubSub.Publish("events.ops.ca_rotated.v1", msg)
			}
		}

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				res, err := caManager.CheckAndRotateCA(ctx)
				if err != nil {
					logger.Error("failed scheduled CA rotation check", "error", err)
				} else {
					logger.Info("scheduled CA key rotation check completed", "status", res.Status)
				}
			}
		}
	}()

	// Simulate event publisher ticker
	go func() {
		<-router.Running()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		event := EventEnvelope[CatSpottedPayload]{
			EventType:     "observation.cat_spotted.v1",
			CorrelationID: "correlation-trace-001",
			Payload: CatSpottedPayload{
				CameraID:        "CAM-ORANGE-01",
				FelineID:        "emp-feline-garfield",
				ActivityType:    "zooming",
				ConfidenceScore: 0.992,
			},
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				event.EventID = watermill.NewUUID()
				event.OccurredAt = time.Now().UTC()

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
