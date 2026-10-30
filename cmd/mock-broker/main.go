package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-broker/handler"
	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
	"github.com/orange-cat-investments/oci/pkg/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "mock-broker")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		_ = telemetry.InitMetrics("")
		_ = telemetry.InitMetrics("mock-broker")
		defer shutdownTelemetry(ctx)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8089"
	}

	apiKeyID := os.Getenv("APCA_API_KEY_ID")
	if apiKeyID == "" {
		apiKeyID = "CK_OCI_TEST_KEY_01"
	}

	apiSecretKey := os.Getenv("APCA_API_SECRET_KEY")
	if apiSecretKey == "" {
		apiSecretKey = "SK_OCI_TEST_SECRET_01"
	}

	tickInterval := time.Second
	if val := os.Getenv("TICK_INTERVAL_MS"); val != "" {
		if ms, err := strconv.Atoi(val); err == nil && ms > 0 {
			tickInterval = time.Duration(ms) * time.Millisecond
		}
	}

	logger.Info("initializing Faux Alpaca Broker Microservice",
		"port", port,
		"tick_interval", tickInterval.String(),
		"auth_key_prefix", apiKeyID[:min(len(apiKeyID), 6)]+"...",
	)

	// 1. In-memory ledger
	store := ledger.NewStore()

	// 2. Market simulation engine
	mktEngine := engine.NewMarketEngine(store, tickInterval, engine.DefaultAssets())

	// 3. Populate startup seeds (Arthur & Chloe accounts)
	ledger.SeedDefaultAccounts(store, mktEngine.InitialPrices())
	logger.Info("startup seed accounts and initial positions loaded into memory")

	// 4. Start background market simulation loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mktEngine.Start(ctx)
	logger.Info("stochastic market price simulation loop started (independent GBM)")

	// 5. Setup HTTP router and handlers
	mux := http.NewServeMux()
	h := handler.NewHandler(store, mktEngine)
	h.RegisterRoutes(mux)

	// Wrap with authentication middleware
	authenticatedHandler := AuthMiddleware(apiKeyID, apiSecretKey, mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      authenticatedHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Graceful shutdown handler
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("faux Alpaca Broker API server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to listen", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownCh
	logger.Info("shutting down faux Alpaca Broker API server...")

	mktEngine.Stop()
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown returned error", "error", err)
	}

	logger.Info("mock broker microservice stopped cleanly")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
