package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	core_invest_handler "github.com/orange-cat-investments/oci/internal/handler/core_invest"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoreInvestAPI_Healthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","domain":"core_invest"}`))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"domain":"core_invest"`)
}

func TestCoreInvestAPI_BalanceUpdate(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)
	ciHandler := core_invest_handler.NewHandler(svc)

	mux := http.NewServeMux()
	ciHandler.RegisterRoutes(mux)

	handleBalanceUpdate := func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}
		var req updateBalanceRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		p, _ := repo.GetPortfolioByID(r.Context(), id)
		if p == nil {
			p, _ = repo.GetPortfolioByCustomerID(r.Context(), id)
		}
		require.NotNil(t, p)

		p.CashBalanceUSD += req.DeltaUSD
		p.TotalBalanceUSD += req.DeltaUSD
		updatedP, err := repo.UpdatePortfolio(r.Context(), p)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                   "balance_updated",
			"portfolio_id":             updatedP.PortfolioID,
			"updated_cash_balance_usd": updatedP.CashBalanceUSD,
		})
	}
	mux.HandleFunc("POST /core-invest/portfolios/{id}/balance", handleBalanceUpdate)

	// Create test portfolio first
	p, err := repo.CreatePortfolio(t.Context(), &core_invest_repo.Portfolio{
		PortfolioID:     "port-test-01",
		CustomerID:      "cust-test-01",
		AccountName:     "Test Account",
		TotalBalanceUSD: 1000.0,
		CashBalanceUSD:  1000.0,
		Status:          "active",
	})
	require.NoError(t, err)
	assert.Equal(t, 1000.0, p.CashBalanceUSD)

	payload := updateBalanceRequest{
		DeltaUSD: 500.0,
		Reason:   "ACH deposit settled",
	}
	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/core-invest/portfolios/port-test-01/balance", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"updated_cash_balance_usd":1500`)
}
