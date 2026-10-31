package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Mock Hardware generator...")

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@rabbitmq-service.oci-internal.svc.cluster.local:5672/"
	}

	pub, _ := envelope.NewPubSubWithFallback(amqpURL, logger)

	coreInvestAddr := os.Getenv("CORE_INVEST_ADDR")
	if coreInvestAddr == "" {
		coreInvestAddr = "core-invest-api-service.oci-core.svc.cluster.local:50053"
	}
	facilitiesAddr := os.Getenv("FACILITIES_ADDR")
	if facilitiesAddr == "" {
		facilitiesAddr = "facilities-api-service.oci-core.svc.cluster.local:50052"
	}

	var tlsConfig *tls.Config
	if os.Getenv("INSECURE") != "true" {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}

	ciConn, err := grpcutil.Dial(context.Background(), coreInvestAddr, tlsConfig)
	if err != nil {
		logger.Error("Failed to dial CoreInvestService", "error", err)
	} else {
		logger.Info("Connected to CoreInvestService", "addr", coreInvestAddr)
	}
	defer ciConn.Close()

	facConn, err := grpcutil.Dial(context.Background(), facilitiesAddr, tlsConfig)
	if err != nil {
		logger.Error("Failed to dial FacilitiesService", "error", err)
	} else {
		logger.Info("Connected to FacilitiesService", "addr", facilitiesAddr)
	}
	defer facConn.Close()

	_ = coreinvestv1.NewCoreInvestServiceClient(ciConn)
	facClient := facilitiesv1.NewFacilitiesServiceClient(facConn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	go runCatSpottedGenerator(ctx, pub, logger)
	go runEnvironmentalGenerator(ctx, facClient, logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("Shutting down mock hardware generator...")
}

func runCatSpottedGenerator(ctx context.Context, pub message.Publisher, logger *slog.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	activities := []string{"zooming", "sleeping", "eating", "playing", "grooming"}

	felineID := os.Getenv("OBSERVATION_FELINE_ID")
	if felineID == "" {
		felineID = "feline-observation-target"
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			activity := activities[rand.Intn(len(activities))]
			confidence := 0.8 + rand.Float64()*0.19 // between 0.80 and 0.99

			event := envelope.EventEnvelope[envelope.CatSpottedPayload]{
				EventID:       watermill.NewUUID(),
				EventType:     "observation.cat_spotted.v1",
				OccurredAt:    time.Now().UTC(),
				CorrelationID: "correlation-trace-" + watermill.NewUUID()[:8],
				Payload: envelope.CatSpottedPayload{
					CameraID:        "CAM-ORANGE-01",
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

func runEnvironmentalGenerator(ctx context.Context, facClient facilitiesv1.FacilitiesServiceClient, logger *slog.Logger) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if facClient == nil {
				continue
			}
			temp := 22.0 + rand.Float64()*3.0 // 22-25 C
			humidity := 40.0 + rand.Float64()*20.0 // 40-60%
			lux := 200.0 + rand.Float64()*100.0
			noise := 30.0 + rand.Float64()*20.0

			_, err := facClient.IngestEnvironmentalTelemetry(ctx, &facilitiesv1.IngestTelemetryRequest{
				ZoneID: "zone-alpha",
				TemperatureC: temp,
				RelativeHumidityPct: humidity,
				LightIntensityLux: lux,
				NoiseLevelDB: noise,
			})
			if err != nil {
				logger.Error("Failed to ingest environmental telemetry", "error", err)
			} else {
				logger.Info("Ingested environmental telemetry", "temp", temp, "humidity", humidity)
			}
		}
	}
}
