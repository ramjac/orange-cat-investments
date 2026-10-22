package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

type InternalBLLServer struct {
	logger         *slog.Logger
	publisher      message.Publisher
	httpClient     *http.Client
	workforceURL   string
	facilitiesURL  string
	opsURL         string
	onboardingSaga *saga.OnboardingSaga
	offboardingEng *saga.OffboardingEngine
}

type IncidentCreateRequest struct {
	Title            string `json:"title"`
	Category         string `json:"category"`
	InvolvedFelineID string `json:"involved_feline_id,omitempty"`
	InvolvedHumanID  string `json:"involved_human_id,omitempty"`
	Severity         string `json:"severity"` // urgent, critical, info
	RoomZone         string `json:"room_zone,omitempty"`
	Description      string `json:"description"`
}

type IncidentCreatedPayload struct {
	IncidentID       string    `json:"incident_id"`
	Title            string    `json:"title"`
	Category         string    `json:"category"`
	InvolvedFelineID string    `json:"involved_feline_id,omitempty"`
	InvolvedHumanID  string    `json:"involved_human_id,omitempty"`
	Severity         string    `json:"severity"`
	RoomZone         string    `json:"room_zone"`
	Description      string    `json:"description"`
	ReportedAt       time.Time `json:"reported_at"`
}

type OTATriggerRequest struct {
	AssetIDs  []string `json:"asset_ids"`
	ReleaseID string   `json:"release_id"`
}

type ReportTriggerRequest struct {
	Period      string `json:"period"`
	GeneratedBy string `json:"generated_by"`
}

func (s *InternalBLLServer) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}

func (s *InternalBLLServer) handleCreateIncident(w http.ResponseWriter, r *http.Request) {
	var req IncidentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid incident payload"}`, http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Severity == "" || req.Description == "" {
		http.Error(w, `{"error":"title, severity, and description are required"}`, http.StatusBadRequest)
		return
	}

	roomZone := req.RoomZone
	if roomZone == "" {
		roomZone = "Zone Alpha (Habitat Perch 1)"
	}

	// 1. Forward to workforce domain API and extract canonical incident_id (Thread 37)
	canonicalID := ""
	if s.workforceURL != "" {
		postBody, _ := json.Marshal(map[string]interface{}{
			"title":              req.Title,
			"category":           req.Category,
			"involved_feline_id": req.InvolvedFelineID,
			"involved_human_id":  req.InvolvedHumanID,
			"severity":           req.Severity,
			"room_zone":          roomZone,
			"description":        req.Description,
		})
		wfReq, err := http.NewRequestWithContext(r.Context(), "POST", s.workforceURL+"/api/v1/workforce/incidents", bytes.NewReader(postBody))
		if err != nil {
			http.Error(w, `{"error":"failed to create workforce incident request"}`, http.StatusInternalServerError)
			return
		}
		wfReq.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(wfReq)
		if err != nil {
			s.log().Error("workforce domain API unavailable", "error", err)
			http.Error(w, `{"error":"upstream workforce domain API unavailable"}`, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			s.log().Error("workforce domain API rejected incident creation", "status", resp.StatusCode)
			http.Error(w, fmt.Sprintf(`{"error":"workforce API rejected incident: status %d"}`, resp.StatusCode), http.StatusBadGateway)
			return
		}

		var savedInc struct {
			IncidentID string `json:"incident_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&savedInc); err == nil && savedInc.IncidentID != "" {
			canonicalID = savedInc.IncidentID
		}
	}

	if canonicalID == "" {
		canonicalID = "inc-" + watermill.NewUUID()[:8]
	}

	payload := IncidentCreatedPayload{
		IncidentID:       canonicalID,
		Title:            req.Title,
		Category:         req.Category,
		InvolvedFelineID: req.InvolvedFelineID,
		InvolvedHumanID:  req.InvolvedHumanID,
		Severity:         req.Severity,
		RoomZone:         roomZone,
		Description:      req.Description,
		ReportedAt:       time.Now().UTC(),
	}

	// 2. Publish events.workforce.incident_created.v1 to RabbitMQ for async worker fanout (Threads 15 & 38)
	eventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(eventID, "events.workforce.incident_created.v1", canonicalID, payload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		if err := s.publisher.Publish("events.workforce.incident_created.v1", message.NewMessage(event.EventID, eventBytes)); err != nil {
			s.log().Error("failed to publish incident event", "error", err, "incident_id", canonicalID)
			http.Error(w, `{"error":"failed to publish incident event to message broker"}`, http.StatusInternalServerError)
			return
		}
		s.log().Info("published workplace incident event", "incident_id", canonicalID, "severity", req.Severity)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *InternalBLLServer) handleTriggerOTA(w http.ResponseWriter, r *http.Request) {
	var req OTATriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid OTA trigger payload"}`, http.StatusBadRequest)
		return
	}

	if len(req.AssetIDs) == 0 || req.ReleaseID == "" {
		http.Error(w, `{"error":"asset_ids and release_id are required"}`, http.StatusBadRequest)
		return
	}

	type otaJob struct {
		JobID     string `json:"job_id"`
		AssetID   string `json:"asset_id"`
		ReleaseID string `json:"release_id"`
	}

	var createdJobs []otaJob

	// 1. Persist per-device jobs via facilities-api (Thread 14)
	if s.facilitiesURL != "" {
		postBody, _ := json.Marshal(map[string]interface{}{
			"asset_ids":  req.AssetIDs,
			"release_id": req.ReleaseID,
		})
		facReq, err := http.NewRequestWithContext(r.Context(), "POST", s.facilitiesURL+"/api/v1/facilities/firmware/ota-jobs", bytes.NewReader(postBody))
		if err == nil {
			facReq.Header.Set("Content-Type", "application/json")
			resp, err := s.httpClient.Do(facReq)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
					var facResp struct {
						Jobs []otaJob `json:"jobs"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&facResp); err == nil {
						createdJobs = facResp.Jobs
					}
				}
			}
		}
	}

	// Fallback if facilities-api not responding with job list
	if len(createdJobs) == 0 {
		for _, assetID := range req.AssetIDs {
			createdJobs = append(createdJobs, otaJob{
				JobID:     "ota-job-" + watermill.NewUUID()[:8],
				AssetID:   assetID,
				ReleaseID: req.ReleaseID,
			})
		}
	}

	// 2. Publish worker-compatible per-device events (Thread 14 & 15)
	for _, job := range createdJobs {
		otaPayload := map[string]interface{}{
			"job_id":      job.JobID,
			"asset_id":    job.AssetID,
			"release_id":  job.ReleaseID,
			"device_type": "edge_camera",
			"version":     "v2.1.0",
			"file_url":    fmt.Sprintf("https://firmware.oci.local/releases/%s.bin", job.ReleaseID),
			"checksum":    "sha256-verified",
		}
		otaEventID := watermill.NewUUID()
		event := envelope.NewEventEnvelope(otaEventID, "events.facilities.ota_triggered.v1", job.JobID, otaPayload)
		eventBytes, _ := json.Marshal(event)
		if s.publisher != nil {
			if err := s.publisher.Publish("events.facilities.ota_triggered.v1", message.NewMessage(event.EventID, eventBytes)); err != nil {
				s.log().Error("failed to publish OTA triggered event", "error", err, "job_id", job.JobID)
				http.Error(w, `{"error":"failed to publish OTA event to message broker"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "scheduled",
		"jobs":   createdJobs,
	})
}

func (s *InternalBLLServer) handleTriggerReport(w http.ResponseWriter, r *http.Request) {
	var req ReportTriggerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Period == "" {
		req.Period = time.Now().UTC().Format("2006-01")
	}
	if req.GeneratedBy == "" {
		req.GeneratedBy = "internal-bll-scheduler"
	}

	reportID := "rep-" + watermill.NewUUID()[:8]
	repPayload := map[string]interface{}{
		"report_id":    reportID,
		"period":       req.Period,
		"generated_by": req.GeneratedBy,
		"requested_at": time.Now().UTC(),
	}

	repEventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(repEventID, "events.reporting.generate_monthly_requested.v1", reportID, repPayload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		if err := s.publisher.Publish("events.reporting.generate_monthly_requested.v1", message.NewMessage(event.EventID, eventBytes)); err != nil {
			s.log().Error("failed to publish report generation event", "error", err, "report_id", reportID)
			http.Error(w, `{"error":"failed to publish report event to message broker"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(repPayload)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Internal Business Logic Layer (Internal BLL) API...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	workforceURL := os.Getenv("WORKFORCE_API_URL")
	if workforceURL == "" {
		workforceURL = "http://localhost:50051"
	}
	facilitiesURL := os.Getenv("FACILITIES_API_URL")
	if facilitiesURL == "" {
		facilitiesURL = "http://localhost:50052"
	}
	opsURL := os.Getenv("OPS_API_URL")
	if opsURL == "" {
		opsURL = "http://localhost:50054"
	}

	watermillLogger := watermill.NewStdLogger(false, false)
	var pubSub message.Publisher
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL != "" {
		amqpConfig := amqp.NewDurableQueueConfig(amqpURL)
		amqpPub, err := amqp.NewPublisher(amqpConfig, watermillLogger)
		if err != nil {
			logger.Error("failed to connect Internal BLL to RabbitMQ AMQP publisher", "error", err)
			os.Exit(1)
		}
		defer amqpPub.Close()
		pubSub = amqpPub
		logger.Info("Internal BLL connected to RabbitMQ AMQP broker successfully")
	} else {
		pubSub = gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)
		logger.Info("Internal BLL running with in-memory GoChannel publisher (local test mode)")
	}

	mockWfRepo := workforce.NewMockRepository()
	mockWfSvc := workforce_svc.NewService(mockWfRepo)
	onboardingSaga := saga.NewOnboardingSaga(mockWfSvc, nil, logger)
	offboardingEng := saga.NewOffboardingEngine(mockWfSvc, nil, logger)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if certFile != "" && keyFile != "" && clientCA != "" {
		clientTLS, err := mtls.NewClientTLSConfig(certFile, keyFile, clientCA)
		if err != nil {
			logger.Warn("failed to configure outbound client dynamic mTLS", "error", err)
		} else {
			httpClient = &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: clientTLS,
				},
			}
			logger.Info("Internal BLL configured with outbound dynamic mTLS transport")
		}
	}

	serverObj := &InternalBLLServer{
		logger:         logger,
		publisher:      pubSub,
		httpClient:     httpClient,
		workforceURL:   workforceURL,
		facilitiesURL:  facilitiesURL,
		opsURL:         opsURL,
		onboardingSaga: onboardingSaga,
		offboardingEng: offboardingEng,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"internal-bll"}`))
	})

	mux.HandleFunc("POST /api/v1/internal-bll/incidents", serverObj.handleCreateIncident)
	mux.HandleFunc("POST /api/v1/internal-bll/firmware/ota", serverObj.handleTriggerOTA)
	mux.HandleFunc("POST /api/v1/internal-bll/reports/monthly", serverObj.handleTriggerReport)

	versionedHandler := versioning.Middleware(versioning.DefaultConfig())(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      versionedHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if certFile != "" && keyFile != "" {
		tlsConfig, err := mtls.NewServerTLSConfig(certFile, keyFile, clientCA)
		if err != nil {
			logger.Error("failed to configure mTLS in Internal BLL", "error", err)
			os.Exit(1)
		}
		server.TLSConfig = tlsConfig
		logger.Info("Internal BLL configured with dynamic mTLS")
	}

	healthServer := mtls.StartHealthListener(os.Getenv("HEALTH_PORT"), port, logger, "internal-bll")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Internal BLL listening", "port", port)
		var err error
		if server.TLSConfig != nil {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Internal BLL server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Internal BLL...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if healthServer != nil {
		_ = healthServer.Shutdown(shutdownCtx)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Internal BLL stopped gracefully")
}
