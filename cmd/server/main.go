package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	core_invest_handler "github.com/orange-cat-investments/oci/internal/handler/core_invest"
	facilities_handler "github.com/orange-cat-investments/oci/internal/handler/facilities"
	ops_handler "github.com/orange-cat-investments/oci/internal/handler/ops"
	workforce_handler "github.com/orange-cat-investments/oci/internal/handler"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	facilities_repo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	ops_repo "github.com/orange-cat-investments/oci/internal/repository/ops"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI API Server...")

	dbURL := os.Getenv("DATABASE_URL")
	var pool *pgxpool.Pool
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), dbURL)
		if err != nil {
			logger.Error("failed to connect to database pool", "error", err)
		} else {
			defer pool.Close()
			logger.Info("connected to PostgreSQL database pool successfully")
		}
	} else {
		logger.Info("no DATABASE_URL provided, running with mock repository for local development")
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Initialize Core Invest Domain Service & Handlers
	var ciRepo core_invest_repo.Repository
	if pool != nil {
		ciRepo = core_invest_repo.NewRepository(pool)
	} else {
		ciRepo = core_invest_repo.NewMockRepository()
	}

	ciService := core_invest_svc.NewService(ciRepo)
	ciHandler := core_invest_handler.NewHandler(ciService)
	ciHandler.RegisterRoutes(mux)

	// Initialize Operations & Wearable Alerts Domain Service & Handlers
	opsRepo := ops_repo.NewRepository(pool)
	opsService := ops_svc.NewService(opsRepo)
	opsHandler := ops_handler.NewHandler(opsService)
	opsHandler.RegisterRoutes(mux)

	// Initialize Facilities Domain Service & Handlers
	var facRepo facilities_repo.Repository
	if pool != nil {
		facRepo = facilities_repo.NewRepository(pool)
	} else {
		facRepo = facilities_repo.NewMockRepository()
	}
	facSvc := facilities_svc.NewService(facRepo)
	facHandler := facilities_handler.NewHandler(facSvc)
	facHandler.RegisterRoutes(mux)

	// Initialize Workforce Domain Service, Sagas & Handlers
	wfRepo := workforce_repo.NewRepository(pool)
	wfSvc := workforce_svc.NewService(wfRepo)
	onboardSaga := saga.NewOnboardingSaga(wfSvc, nil, logger)
	offboardEng := saga.NewOffboardingEngine(wfSvc, nil, logger)
	wfHandler := workforce_handler.NewWorkforceHandler(wfSvc, onboardSaga, offboardEng)
	wfHandler.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("API server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down API Server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", "error", err)
	}

	logger.Info("API Server stopped gracefully")
}
