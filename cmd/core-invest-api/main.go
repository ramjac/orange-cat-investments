package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	core_invest_handler "github.com/orange-cat-investments/oci/internal/handler/core_invest"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

type updateBalanceRequest struct {
	TransferID string  `json:"transfer_id"`
	DeltaUSD   float64 `json:"delta_usd"`
	Reason     string  `json:"reason"`
}

var (
	balanceMu        sync.Mutex
	settledTransfers = make(map[string]map[string]interface{})
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Core Invest Domain API microservice...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
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
			logger.Info("connected Core Invest Domain API to PostgreSQL pool successfully")
		}
	} else {
		logger.Info("DATABASE_URL not set, running Core Invest Domain API with in-memory persistence fallback")
	}

	var repo core_invest_repo.Repository
	if pool != nil {
		repo = core_invest_repo.NewRepository(pool)
	} else {
		repo = core_invest_repo.NewMockRepository()
	}

	svc := core_invest_svc.NewService(repo)
	ciHandler := core_invest_handler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","domain":"core_invest"}`))
	})
	ciHandler.RegisterRoutes(mux)

	// Endpoint to update portfolio balance upon ACH settlement or dividend allocation
	handleBalanceUpdate := newBalanceUpdateHandler(repo, pool)
	mux.HandleFunc("POST /core-invest/portfolios/{id}/balance", handleBalanceUpdate)
	mux.HandleFunc("POST /api/v1/core-invest/portfolios/{id}/balance", handleBalanceUpdate)

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
		logger.Info("Core Invest Domain API configured with dynamic mTLS")
	}

	healthServer := mtls.StartHealthListener(os.Getenv("HEALTH_PORT"), port, logger, "core-invest-api")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Core Invest Domain API listening", "port", port)
		var err error
		if server.TLSConfig != nil {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Core Invest Domain API HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Core Invest Domain API...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if healthServer != nil {
		_ = healthServer.Shutdown(shutdownCtx)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Core Invest Domain API stopped gracefully")
}

func newBalanceUpdateHandler(repo core_invest_repo.Repository, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}
		if id == "" {
			http.Error(w, `{"error":"portfolio id required"}`, http.StatusBadRequest)
			return
		}
		var req updateBalanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
			return
		}

		balanceMu.Lock()
		defer balanceMu.Unlock()

		// Idempotency check: if transfer_id already settled, return existing result
		if req.TransferID != "" {
			if existing, found := settledTransfers[req.TransferID]; found {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(existing)
				return
			}
		}

		ctx := r.Context()
		if pool != nil {
			tx, err := pool.Begin(ctx)
			if err == nil {
				defer tx.Rollback(ctx)
				var portfolioID string
				var cashBalance, totalBalance float64
				err = tx.QueryRow(ctx, `
					UPDATE core_invest.investment_portfolios
					SET cash_balance_usd = cash_balance_usd + $1,
					    total_balance_usd = total_balance_usd + $1,
					    updated_at = CURRENT_TIMESTAMP
					WHERE portfolio_id::text = $2 OR customer_id = $2
					RETURNING portfolio_id, cash_balance_usd, total_balance_usd
				`, req.DeltaUSD, id).Scan(&portfolioID, &cashBalance, &totalBalance)

				if err == nil {
					_ = tx.Commit(ctx)
					respData := map[string]interface{}{
						"status":                    "balance_updated",
						"portfolio_id":              portfolioID,
						"updated_cash_balance_usd":  cashBalance,
						"updated_total_balance_usd": totalBalance,
						"transfer_id":               req.TransferID,
					}
					if req.TransferID != "" {
						settledTransfers[req.TransferID] = respData
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(respData)
					return
				}
			}
		}

		// Fallback for mock/in-memory repo under balanceMu lock
		p, err := repo.GetPortfolioByID(ctx, id)
		if err != nil || p == nil {
			p, err = repo.GetPortfolioByCustomerID(ctx, id)
		}
		if err != nil || p == nil {
			http.Error(w, `{"error":"portfolio not found"}`, http.StatusNotFound)
			return
		}

		p.CashBalanceUSD += req.DeltaUSD
		p.TotalBalanceUSD += req.DeltaUSD
		updatedP, err := repo.UpdatePortfolio(ctx, p)
		if err != nil {
			http.Error(w, `{"error":"failed to update balance"}`, http.StatusInternalServerError)
			return
		}

		respData := map[string]interface{}{
			"status":                    "balance_updated",
			"portfolio_id":              updatedP.PortfolioID,
			"updated_cash_balance_usd":  updatedP.CashBalanceUSD,
			"updated_total_balance_usd": updatedP.TotalBalanceUSD,
			"transfer_id":               req.TransferID,
		}
		if req.TransferID != "" {
			settledTransfers[req.TransferID] = respData
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(respData)
	}
}
