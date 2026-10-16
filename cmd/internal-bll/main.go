package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
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
	employeeBFFURL string
	pebbleProxyURL string
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

	incidentID := "inc-" + watermill.NewUUID()[:8]
	roomZone := req.RoomZone
	if roomZone == "" {
		roomZone = "Zone Alpha (Habitat Perch 1)"
	}

	payload := IncidentCreatedPayload{
		IncidentID:       incidentID,
		Title:            req.Title,
		Category:         req.Category,
		InvolvedFelineID: req.InvolvedFelineID,
		InvolvedHumanID:  req.InvolvedHumanID,
		Severity:         req.Severity,
		RoomZone:         roomZone,
		Description:      req.Description,
		ReportedAt:       time.Now().UTC(),
	}

	// Forward to workforce domain API if reachable
	if s.workforceURL != "" {
		postBody, _ := json.Marshal(map[string]interface{}{
			"title":              req.Title,
			"category":           req.Category,
			"involved_feline_id": req.InvolvedFelineID,
			"involved_human_id":  req.InvolvedHumanID,
			"severity":           req.Severity,
			"description":        req.Description,
		})
		wfReq, err := http.NewRequestWithContext(r.Context(), "POST", s.workforceURL+"/api/v1/workforce/incidents", bytes.NewReader(postBody))
		if err == nil {
			wfReq.Header.Set("Content-Type", "application/json")
			_, _ = s.httpClient.Do(wfReq)
		}
	}

	// Publish events.workforce.incident_created.v1 to RabbitMQ for web portal and Pebble watch alerting
	eventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(eventID, "events.workforce.incident_created.v1", incidentID, payload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		_ = s.publisher.Publish("events.workforce.incident_created.v1", message.NewMessage(event.EventID, eventBytes))
		s.log().Info("published workplace incident event", "incident_id", incidentID, "severity", req.Severity)
	}

	// Fan out to Employee BFF SSE hub for real-time web banner
	if s.employeeBFFURL != "" {
		postBody, _ := json.Marshal(payload)
		bcastReq, err := http.NewRequestWithContext(r.Context(), "POST", s.employeeBFFURL+"/api/v1/employee/incidents/broadcast", bytes.NewReader(postBody))
		if err == nil {
			bcastReq.Header.Set("Content-Type", "application/json")
			_, _ = s.httpClient.Do(bcastReq)
		}
	}

	// Fan out to Pebble Proxy gateway for smartwatch vibrating alerts
	if s.pebbleProxyURL != "" {
		postBody, _ := json.Marshal(payload)
		pebbleReq, err := http.NewRequestWithContext(r.Context(), "POST", s.pebbleProxyURL+"/api/v1/ops/pebble/incidents/broadcast", bytes.NewReader(postBody))
		if err == nil {
			pebbleReq.Header.Set("Content-Type", "application/json")
			_, _ = s.httpClient.Do(pebbleReq)
		}
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

	jobID := "ota-job-" + watermill.NewUUID()[:8]
	otaPayload := map[string]interface{}{
		"job_id":     jobID,
		"asset_ids":  req.AssetIDs,
		"release_id": req.ReleaseID,
		"status":     "scheduled",
	}

	otaEventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(otaEventID, "events.facilities.ota_triggered.v1", jobID, otaPayload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		_ = s.publisher.Publish("events.facilities.ota_triggered.v1", message.NewMessage(event.EventID, eventBytes))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(otaPayload)
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
		_ = s.publisher.Publish("events.reporting.generate_monthly_requested.v1", message.NewMessage(event.EventID, eventBytes))
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
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	mockWfRepo := workforce.NewMockRepository()
	mockWfSvc := workforce_svc.NewService(mockWfRepo)
	onboardingSaga := saga.NewOnboardingSaga(mockWfSvc, nil, logger)
	offboardingEng := saga.NewOffboardingEngine(mockWfSvc, nil, logger)

	employeeBFFURL := os.Getenv("EMPLOYEE_BFF_URL")
	if employeeBFFURL == "" {
		employeeBFFURL = "http://localhost:8081"
	}
	pebbleProxyURL := os.Getenv("PEBBLE_PROXY_URL")
	if pebbleProxyURL == "" {
		pebbleProxyURL = "http://localhost:8082"
	}

	serverObj := &InternalBLLServer{
		logger:         logger,
		publisher:      pubSub,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		workforceURL:   workforceURL,
		facilitiesURL:  facilitiesURL,
		opsURL:         opsURL,
		employeeBFFURL: employeeBFFURL,
		pebbleProxyURL: pebbleProxyURL,
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

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if certFile != "" && keyFile != "" {
		tlsConfig, err := mtls.NewServerTLSConfig(certFile, keyFile, clientCA)
		if err != nil {
			logger.Error("failed to configure mTLS in Internal BLL", "error", err)
			os.Exit(1)
		}
		server.TLSConfig = tlsConfig
		logger.Info("Internal BLL configured with dynamic mTLS")
	}

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

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Internal BLL stopped gracefully")
}
