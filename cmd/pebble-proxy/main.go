package main

import (
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

	"github.com/jackc/pgx/v5/pgxpool"
	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Pebble Companion App Gateway Proxy")

	enablePebbleGateway := os.Getenv("ENABLE_PEBBLE_GATEWAY") != "false"
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && os.Getenv("ENABLE_PEBBLE_GATEWAY") == "" {
		enablePebbleGateway = false
	}
	if !enablePebbleGateway {
		logger.Info("Pebble companion gateway proxy is disabled via configuration (ENABLE_PEBBLE_GATEWAY=false)")
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dbPool *pgxpool.Pool
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			logger.Warn("failed to connect to postgresql pool, falling back to mock persistence", "error", err)
		} else {
			dbPool = pool
			defer dbPool.Close()
			logger.Info("connected to postgresql database pool successfully")
		}
	} else {
		logger.Info("DATABASE_URL not set, running with in-memory persistence fallback")
	}

	repo := opsrepo.NewRepository(dbPool)
	svc := opssvc.NewService(repo)
	handler := opshandler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"pebble-proxy"}`))
	})
	handler.RegisterRoutes(mux)

	// Ingest workplace safety incidents into high-priority wearable alerts
	incidentHandler := handleIncidentAlertBroadcast(repo, logger)
	mux.HandleFunc("POST /api/v1/ops/pebble/incidents/broadcast", incidentHandler)
	mux.HandleFunc("POST /ops/pebble/incidents/broadcast", incidentHandler)
	mux.HandleFunc("POST /ops/pebble/alerts/incident", incidentHandler)

	versionedHandler := versioning.Middleware(versioning.DefaultConfig())(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      versionedHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Pebble companion gateway proxy listening", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Pebble Companion Gateway Proxy...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Pebble Companion Gateway Proxy stopped gracefully")
}

func handleIncidentAlertBroadcast(repo opsrepo.Repository, logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IncidentID  string `json:"incident_id"`
			Title       string `json:"title"`
			Category    string `json:"category"`
			Severity    string `json:"severity"`
			RoomZone    string `json:"room_zone"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid incident payload"}`, http.StatusBadRequest)
			return
		}

		ticketID := req.IncidentID
		if ticketID == "" {
			ticketID = fmt.Sprintf("inc-%d", time.Now().UnixNano())
		}

		title := req.Title
		if req.RoomZone != "" {
			title = fmt.Sprintf("🚨 %s (%s)", req.Title, req.RoomZone)
		} else {
			title = fmt.Sprintf("🚨 %s", req.Title)
		}

		severity := req.Severity
		if severity == "" {
			severity = "critical"
		}

		desc := req.Description
		ticket := &opsrepo.ITTicket{
			TicketID:       ticketID,
			Title:          title,
			Body:           &desc,
			State:          "open",
			Severity:       severity,
			Category:       "incident",
			Source:         "workforce-incidents",
			AuthorUsername: "incident-pipeline",
			IsUrgent:       true,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		created, err := repo.CreateITTicket(r.Context(), ticket)
		if err != nil {
			logger.Error("failed to create incident alert ticket for pebble", "error", err)
			http.Error(w, `{"error":"failed to create incident alert"}`, http.StatusInternalServerError)
			return
		}

		logger.Info("registered workplace incident alert for Pebble wearable push", "ticket_id", created.TicketID, "severity", severity)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	}
}

