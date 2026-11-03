package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
)

// Simulation fixture identifiers for the emulated edge devices. Hardcoded fixture
// data is permitted inside isolated external-sim mock services (see AGENTS.md).
const (
	fixtureCameraID = "CAM-ORANGE-01"
	// Seeded demo location zone ("Alpha Sunbeam Lounge", see scripts/seed-dev.sql).
	fixtureZoneID   = "018f3a9a-1111-7000-8000-000000000001"
	fixtureFelineID = "feline-observation-target"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Mock Hardware generator in external-sim...")

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		logger.Warn("RABBITMQ_URL not configured; events will be dropped (in-memory publisher fallback)")
	}
	pub := envelope.NewPublisherWithFallback(amqpURL, logger)

	internalBLLAddr := os.Getenv("INTERNAL_BLL_ADDR")
	if internalBLLAddr == "" {
		internalBLLAddr = "internal-bll-service.oci-core.svc.cluster.local:8086"
	}

	tlsConfig, err := mtls.NewClientTLSConfig(os.Getenv("TLS_CERT_FILE"), os.Getenv("TLS_KEY_FILE"), os.Getenv("CLIENT_CA_FILE"))
	if err != nil {
		logger.Error("Failed to configure mTLS", "error", err)
		os.Exit(1)
	}

	bllConn, err := grpcutil.Dial(context.Background(), internalBLLAddr, tlsConfig)
	if err != nil {
		logger.Error("Failed to dial InternalBLL", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to InternalBLL", "addr", internalBLLAddr)
	defer bllConn.Close()

	bllClient := internalbllv1.NewInternalBLLServiceClient(bllConn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runCatSpottedGenerator(ctx, pub, bllClient, logger)
	go runEnvironmentalGenerator(ctx, bllClient, logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("Shutting down mock hardware generator...")
}

// fetchFelineTarget resolves the observation target from Internal BLL, falling back
// to the simulation fixture when the platform is unreachable.
func fetchFelineTarget(ctx context.Context, bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) string {
	res, err := bllClient.GetFelineTarget(ctx, &internalbllv1.GetFelineTargetRequest{})
	if err == nil && res != nil && res.FelineID != "" {
		return res.FelineID
	}
	if err != nil {
		logger.Warn("failed to resolve feline target from Internal BLL, using simulation fixture", "error", err)
	}
	return fixtureFelineID
}

func runCatSpottedGenerator(ctx context.Context, pub message.Publisher, bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	activities := []string{"zooming", "sleeping", "eating", "playing", "grooming"}

	felineID := os.Getenv("OBSERVATION_FELINE_ID")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if felineID == "" {
				felineID = fetchFelineTarget(ctx, bllClient, logger)
			}

			activity := activities[rand.Intn(len(activities))]
			confidence := 0.8 + rand.Float64()*0.19 // between 0.80 and 0.99

			event := envelope.EventEnvelope[envelope.CatSpottedPayload]{
				EventID:       uuid.Must(uuid.NewV7()).String(),
				EventType:     "observation.cat_spotted.v1",
				OccurredAt:    time.Now().UTC(),
				CorrelationID: "correlation-trace-" + uuid.Must(uuid.NewV7()).String()[:8],
				Payload: envelope.CatSpottedPayload{
					CameraID:        fixtureCameraID,
					FelineID:        felineID,
					ActivityType:    activity,
					ConfidenceScore: confidence,
				},
			}
			data, _ := json.Marshal(event)
			msg := message.NewMessage(event.EventID, data)
			if err := pub.Publish("events.observation.cat_spotted.v1", msg); err != nil {
				logger.Error("Failed to publish cat spotted event", "error", err)
			} else {
				logger.Info("Published cat spotted event", "activity", activity, "confidence", confidence)
			}
		}
	}
}

func runEnvironmentalGenerator(ctx context.Context, bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			zoneID := os.Getenv("OBSERVATION_ZONE_ID")
			if zoneID == "" {
				zoneID = fixtureZoneID
			}
			temp := 22.0 + rand.Float64()*3.0      // 22-25 C
			humidity := 40.0 + rand.Float64()*20.0 // 40-60%
			lux := 200.0 + rand.Float64()*100.0
			noise := 30.0 + rand.Float64()*20.0

			_, err := bllClient.IngestEnvironmentalTelemetry(ctx, &facilitiesv1.IngestTelemetryRequest{
				ZoneID:              zoneID,
				TemperatureC:        temp,
				RelativeHumidityPct: humidity,
				LightIntensityLux:   lux,
				NoiseLevelDB:        noise,
			})
			if err != nil {
				logger.Error("Failed to ingest environmental telemetry via BLL", "error", err)
			} else {
				logger.Info("Ingested environmental telemetry", "temp", temp, "humidity", humidity)
			}
		}
	}
}
