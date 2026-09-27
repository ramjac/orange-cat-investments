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

	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Pebble Companion App Gateway Proxy")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	repo := opsrepo.NewRepository(nil)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Pebble Companion Gateway Proxy stopped gracefully")
}
