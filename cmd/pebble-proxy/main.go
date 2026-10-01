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
	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
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
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
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
