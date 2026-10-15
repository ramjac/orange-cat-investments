package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orange-cat-investments/oci/internal/handler"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting OCI Employee BFF API Server...")

	dbURL := os.Getenv("DATABASE_URL")
	var pool *pgxpool.Pool
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), dbURL)
		if err != nil {
			logger.Warn("failed to connect to database pool", "error", err)
			if os.Getenv("ENV") == "production" || os.Getenv("OCI_ENV") == "production" || os.Getenv("REQUIRE_DB") == "true" {
				logger.Error("database connection required in production mode, exiting")
				os.Exit(1)
			}
		} else {
			defer pool.Close()
			logger.Info("connected to PostgreSQL database pool successfully")
		}
	} else if os.Getenv("ENV") == "production" || os.Getenv("OCI_ENV") == "production" || os.Getenv("REQUIRE_DB") == "true" {
		logger.Error("DATABASE_URL must be specified in production mode, exiting")
		os.Exit(1)
	} else {
		logger.Info("no DATABASE_URL provided, running with mock repository for local development")
	}

	var repo workforce.Repository
	if pool != nil {
		repo = workforce.NewRepository(pool)
	} else {
		repo = workforce.NewMockRepository()
	}

	var hrClient wfService.HRClient
	frappeURL := os.Getenv("FRAPPE_HR_URL")
	if frappeURL != "" {
		hrClient = frappe.NewClient(frappe.Config{
			BaseURL:   frappeURL,
			APIKey:    os.Getenv("FRAPPE_API_KEY"),
			APISecret: os.Getenv("FRAPPE_API_SECRET"),
		})
		logger.Info("Frappe HR client initialized", "url", frappeURL)
	}

	var wfSvc wfService.Service
	if hrClient != nil {
		wfSvc = wfService.NewServiceWithIntegrations(repo, nil, hrClient)
	} else {
		wfSvc = wfService.NewService(repo)
	}
	onboardingSaga := saga.NewOnboardingSaga(wfSvc, nil, logger)
	offboardingEng := saga.NewOffboardingEngine(wfSvc, nil, logger)

	wfHandler := handler.NewWorkforceHandler(wfSvc, onboardingSaga, offboardingEng)
	enableFeline := os.Getenv("ENABLE_FELINE_WORKFORCE") != "false"
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && os.Getenv("ENABLE_FELINE_WORKFORCE") == "" {
		enableFeline = false
	}
	wfHandler.EnableFelineWorkforce = enableFeline

	mux := http.NewServeMux()
	sessionStore := NewSessionStore()
	authHandler := NewAuthHandler(sessionStore)
	authHandler.RegisterAuthRoutes(mux)
	wfHandler.RegisterRoutes(mux)

	// Health check endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

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
		logger.Info("Employee BFF listening", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Employee BFF HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Employee BFF API Server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Employee BFF stopped gracefully")
}
