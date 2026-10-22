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
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Workforce Domain API microservice...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pool *pgxpool.Pool
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(ctx, dbURL)
		if err != nil {
			logger.Warn("failed to connect to database pool, falling back to mock persistence", "error", err)
		} else {
			defer pool.Close()
			logger.Info("connected Workforce Domain API to PostgreSQL pool successfully")
		}
	} else {
		logger.Info("DATABASE_URL not set, running Workforce Domain API with in-memory persistence fallback")
	}

	var repo workforce.Repository
	if pool != nil {
		repo = workforce.NewRepository(pool)
	} else {
		repo = workforce.NewMockRepository()
	}

	svc := workforce_svc.NewService(repo)
	wfHandler := handler.NewWorkforceHandler(svc, nil, nil)
	enableFeline := os.Getenv("ENABLE_FELINE_WORKFORCE") != "false"
	wfHandler.EnableFelineWorkforce = enableFeline

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","domain":"workforce"}`))
	})
	wfHandler.RegisterRoutes(mux)

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
			logger.Error("failed to configure mTLS", "error", err)
			os.Exit(1)
		}
		server.TLSConfig = tlsConfig
		logger.Info("Workforce Domain API configured with dynamic mTLS")
	}

	healthServer := mtls.StartHealthListener(os.Getenv("HEALTH_PORT"), port, logger, "workforce-api")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Workforce Domain API listening", "port", port)
		var err error
		if server.TLSConfig != nil {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Workforce Domain API HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Workforce Domain API...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if healthServer != nil {
		_ = healthServer.Shutdown(shutdownCtx)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Workforce Domain API stopped gracefully")
}
