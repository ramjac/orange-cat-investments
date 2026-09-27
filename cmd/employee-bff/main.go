package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/orange-cat-investments/oci/internal/handler"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting OCI Employee BFF API Server...")

	repo := workforce.NewRepository(nil)
	wfSvc := wfService.NewService(repo)
	onboardingSaga := saga.NewOnboardingSaga(wfSvc, nil, logger)
	offboardingEng := saga.NewOffboardingEngine(wfSvc, nil, logger)

	wfHandler := handler.NewWorkforceHandler(wfSvc, onboardingSaga, offboardingEng)

	mux := http.NewServeMux()
	wfHandler.RegisterRoutes(mux)

	// Health check endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	logger.Info("Employee BFF listening", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Error("Employee BFF HTTP server error", "error", err)
		os.Exit(1)
	}
}
