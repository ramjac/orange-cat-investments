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

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orange-cat-investments/oci/internal/handler"
	ops_handler "github.com/orange-cat-investments/oci/internal/handler/ops"
	ops_repo "github.com/orange-cat-investments/oci/internal/repository/ops"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
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
	hrURL := os.Getenv("HR_SYSTEM_URL")
	if hrURL == "" {
		hrURL = os.Getenv("FRAPPE_HR_URL")
	}
	if hrURL != "" {
		hrClient = frappe.NewClient(frappe.Config{
			BaseURL:   hrURL,
			APIKey:    os.Getenv("FRAPPE_API_KEY"),
			APISecret: os.Getenv("FRAPPE_API_SECRET"),
		})
		logger.Info("connected Employee BFF to HR management system", "url", hrURL)
	}

	var workspaceClient wfService.WorkspaceClient
	workspaceURL := os.Getenv("WORKSPACE_URL")
	if workspaceURL == "" {
		workspaceURL = os.Getenv("NEXTCLOUD_URL")
	}
	if workspaceURL != "" {
		workspaceClient = nextcloud.NewClient(nextcloud.Config{
			BaseURL:       workspaceURL,
			AdminUser:     os.Getenv("NEXTCLOUD_ADMIN_USER"),
			AdminPassword: os.Getenv("NEXTCLOUD_ADMIN_PASSWORD"),
		})
		logger.Info("Workspace intranet client initialized", "url", workspaceURL)
	}

	var wfSvc wfService.Service
	if hrClient != nil || workspaceClient != nil {
		wfSvc = wfService.NewServiceWithIntegrations(repo, nil, hrClient, workspaceClient)
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

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	valkeyPass := os.Getenv("VALKEY_PASSWORD")

	sessionStore := sessionvalkey.NewEmployeeSessionStore(valkeyAddr, valkeyPass, 0, logger)
	cache := cachevalkey.NewCache(valkeyAddr, valkeyPass, 0, logger)
	rateLimiter := cachevalkey.NewRateLimiterWithAddr(valkeyAddr, valkeyPass, 0, logger)
	wfHandler.SetCache(cache)

	isProd := os.Getenv("ENV") == "production" || os.Getenv("OCI_ENV") == "production"

	issueTrackerURL := os.Getenv("FORGEJO_URL")
	if issueTrackerURL == "" {
		issueTrackerURL = os.Getenv("ISSUE_TRACKER_URL")
	}

	authHandler := NewAuthHandler(sessionStore, issueTrackerURL, isProd)
	authHandler.RegisterAuthRoutes(mux)
	wfHandler.RegisterRoutes(mux)

	var opsRepo ops_repo.Repository
	if pool != nil {
		opsRepo = ops_repo.NewRepository(pool)
	} else {
		opsRepo = ops_repo.NewMockRepository()
	}

	var issueTracker ops_svc.IssueTrackerClient
	if issueTrackerURL != "" {
		issueTracker = forgejo.NewClient(forgejo.Config{
			BaseURL: issueTrackerURL,
			Token:   os.Getenv("FORGEJO_TOKEN"),
		})
		logger.Info("connected Employee BFF to Forgejo issue tracker", "url", issueTrackerURL)
	}

	var opsSvc ops_svc.Service
	if issueTracker != nil {
		opsSvc = ops_svc.NewServiceWithIssueTracker(opsRepo, issueTracker)
	} else {
		opsSvc = ops_svc.NewService(opsRepo)
	}
	opsH := ops_handler.NewHandler(opsSvc)
	registerProtectedOpsRoutes(mux, opsH, sessionStore, rateLimiter)

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL != "" {
		watermillLogger := watermill.NewStdLogger(false, false)
		amqpConfig := amqp.NewDurablePubSubConfig(amqpURL, amqp.GenerateQueueNameTopicName)
		amqpSub, err := amqp.NewSubscriber(amqpConfig, watermillLogger)
		if err == nil {
			listener := cachevalkey.NewInvalidationListener(cache, amqpSub, logger)
			listener.Start(context.Background())
			defer listener.Stop()
			logger.Info("connected Employee BFF to RabbitMQ for event-driven cache invalidation")
		} else {
			logger.Warn("failed to initialize RabbitMQ subscriber for cache invalidation", "error", err)
		}
	}

	// Incident Alert Real-Time SSE Stream & Broadcast Hub
	incidentBroker := NewIncidentBroker(logger)
	sessionAuth := SessionAuthMiddleware(sessionStore)
	serviceAuth := InternalServiceAuthMiddleware()

	mux.Handle("GET /api/v1/employee/incidents/stream", sessionAuth(http.HandlerFunc(incidentBroker.HandleStream)))
	mux.Handle("GET /employee/incidents/stream", sessionAuth(http.HandlerFunc(incidentBroker.HandleStream)))
	mux.Handle("POST /api/v1/employee/incidents/broadcast", serviceAuth(http.HandlerFunc(incidentBroker.HandleBroadcast)))
	mux.Handle("POST /employee/incidents/broadcast", serviceAuth(http.HandlerFunc(incidentBroker.HandleBroadcast)))

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
