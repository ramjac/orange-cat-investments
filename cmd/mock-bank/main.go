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

	"github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/handler"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	clientID := os.Getenv("PLAID_CLIENT_ID")
	if clientID == "" {
		clientID = "client_oci_sim_01"
	}

	secret := os.Getenv("PLAID_SECRET")
	if secret == "" {
		secret = "sec_oci_sim_test_8829"
	}

	webhookURL := os.Getenv("OCI_WEBHOOK_URL")

	tickInterval := time.Second
	if val := os.Getenv("ACH_TICK_INTERVAL_MS"); val != "" {
		if ms, err := strconv.Atoi(val); err == nil && ms > 0 {
			tickInterval = time.Duration(ms) * time.Millisecond
		}
	}

	prefix := clientID
	if len(prefix) > 6 {
		prefix = prefix[:6]
	}
	logger.Info("initializing Mock Bank API Microservice",
		"port", port,
		"tick_interval", tickInterval.String(),
		"webhook_url", webhookURL,
		"client_id_prefix", prefix+"...",
	)

	// 1. In-memory ledger
	store := ledger.NewLedgerStore()
	logger.Info("startup seed accounts and balances loaded into memory (Arthur, Chloe, OCI Custody)")

	// 2. Webhook dispatcher and ACH settlement engine
	dispatcher := engine.NewWebhookDispatcher(webhookURL)
	settleEngine := engine.NewSettlementEngine(store, dispatcher, tickInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settleEngine.Start(ctx)
	logger.Info("ACH settlement engine started")

	// 3. HTTP router and handlers
	mux := http.NewServeMux()
	h := handler.NewHandler(store, settleEngine)
	h.RegisterRoutes(mux)

	authenticatedHandler := RequireBankAuth(clientID, secret)(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      authenticatedHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 4. Graceful shutdown handler
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("mock Bank API server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to listen", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownCh
	logger.Info("shutting down mock Bank API server...")

	settleEngine.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}

	logger.Info("mock Bank API server terminated cleanly")
}
