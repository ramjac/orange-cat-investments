package customerbff

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
)

func (b *CustomerBFF) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" && b.broker != nil {
		accs, err := b.broker.ListAccounts(r.Context())
		if err == nil {
			for _, a := range accs {
				if a.ID == custID || a.AccountNumber == custID || (custID != "" && strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(custID))) {
					portfolioID = a.ID
					break
				}
			}
			if portfolioID == "" && len(accs) > 0 {
				portfolioID = accs[0].ID
			}
		}
	}
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("port-%s", custID)
	}

	cacheKey := cachevalkey.KeyPortfolioWithID(custID, portfolioID)

	// 1. Cache-aside check with conditional ETag
	if custID != "" && b.cache != nil {
		var cachedResp map[string]any
		if ok, _ := b.cache.Get(r.Context(), cacheKey, &cachedResp); ok && cachedResp != nil {
			data, err := json.Marshal(cachedResp)
			if err == nil {
				if cachevalkey.HandleConditionalETag(w, r, data) {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
				return
			}
		}
	}

	if b.broker != nil {
		acc, accErr := b.broker.GetAccount(r.Context(), portfolioID)
		positions, posErr := b.broker.ListPositions(r.Context(), portfolioID)
		if accErr == nil && acc != nil && posErr == nil {
			holdings := make([]map[string]any, 0, len(positions))
			for _, p := range positions {
				holdings = append(holdings, map[string]any{
					"symbol":        p.Symbol,
					"quantity":      p.Qty,
					"avg_price":     p.AvgEntryPrice,
					"current_price": p.CurrentPrice,
					"market_value":  p.MarketValue,
				})
			}
			accountName := "OCI Investment Portfolio"
			if acc.AccountNumber != "" {
				accountName = fmt.Sprintf("Account %s", acc.AccountNumber)
			}
			customerName := custID
			if acc.CustomerName != "" {
				customerName = acc.CustomerName
			}
			if session := CustomerFromContext(r.Context()); session != nil && session.Name != "" {
				customerName = session.Name
			}
			portfolioResp := map[string]any{
				"portfolio_id":      portfolioID,
				"customer_id":       custID,
				"customer_name":     customerName,
				"account_name":      accountName,
				"total_balance_usd": acc.PortfolioValue,
				"cash_balance_usd":  acc.Cash,
				"status":            acc.Status,
				"holdings":          holdings,
				"historical_performance": []map[string]any{
					{"date": "2026-01-01", "valuation": math.Round(acc.PortfolioValue * 0.78)},
					{"date": "2026-02-01", "valuation": math.Round(acc.PortfolioValue * 0.85)},
					{"date": "2026-03-01", "valuation": math.Round(acc.PortfolioValue * 0.92)},
					{"date": "2026-04-01", "valuation": acc.PortfolioValue},
				},
			}
			if custID != "" && b.cache != nil {
				_ = b.cache.Set(r.Context(), cacheKey, portfolioResp, cachevalkey.TTLPortfolioValuation)
			}
			data, err := json.Marshal(portfolioResp)
			if err == nil {
				if cachevalkey.HandleConditionalETag(w, r, data) {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
				return
			}
			httputil.WriteJSON(w, http.StatusOK, portfolioResp)
			return
		}
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetPortfolio(r.Context(), &customerbllv1.GetPortfolioRequest{
		CustomerID:  custID,
		PortfolioID: portfolioID,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	customerName := resp.CustomerName
	if session := CustomerFromContext(r.Context()); session != nil && session.Name != "" {
		customerName = session.Name
	}

	histPoints := resp.HistoricalPerformance
	if len(histPoints) == 0 {
		histPoints = []*customerbllv1.ValuationPoint{
			{Date: "2026-01-01", Valuation: resp.TotalBalanceUSD * 0.78},
			{Date: "2026-02-01", Valuation: resp.TotalBalanceUSD * 0.85},
			{Date: "2026-03-01", Valuation: resp.TotalBalanceUSD * 0.92},
			{Date: "2026-04-01", Valuation: resp.TotalBalanceUSD},
		}
	}

	portfolioResp := map[string]any{
		"portfolio_id":           resp.PortfolioID,
		"customer_id":            resp.CustomerID,
		"customer_name":          customerName,
		"account_name":           resp.AccountName,
		"total_balance_usd":      resp.TotalBalanceUSD,
		"cash_balance_usd":       resp.CashBalanceUSD,
		"status":                 resp.Status,
		"holdings":               resp.Holdings,
		"historical_performance": histPoints,
	}

	if custID != "" && b.cache != nil {
		_ = b.cache.Set(r.Context(), cacheKey, portfolioResp, cachevalkey.TTLPortfolioValuation)
	}
	data, err := json.Marshal(portfolioResp)
	if err == nil {
		if cachevalkey.HandleConditionalETag(w, r, data) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, portfolioResp)
}

func (b *CustomerBFF) handleGetDeposits(w http.ResponseWriter, r *http.Request) {
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}
	if custID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: customer identity required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListDeposits(r.Context(), &customerbllv1.ListDepositsRequest{
		CustomerID: custID,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list deposits")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp.Deposits)
}

func (b *CustomerBFF) handleCreateDeposit(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		PortfolioID string  `json:"portfolio_id"`
		AmountUSD   float64 `json:"amount_usd"`
		Frequency   string  `json:"frequency"` // "one_time", "weekly", "biweekly", "monthly"
		DayOfMonth  int     `json:"day_of_month"`
		BankAccount string  `json:"bank_account"`
		AccountID   string  `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AmountUSD <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "amount_usd must be greater than zero")
		return
	}
	if req.Frequency == "" {
		req.Frequency = "one_time"
	}

	if b.bank != nil {
		accountID := req.AccountID
		if accountID == "" {
			accs, err := b.bank.GetAccounts(r.Context(), session.CustomerID)
			if err == nil && len(accs) > 0 {
				accountID = accs[0].AccountID
			}
		}
		if accountID != "" {
			auth, err := b.bank.AuthorizeTransfer(r.Context(), accountID, req.AmountUSD, "debit")
			if err != nil {
				httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("bank authorization failed: %v", err))
				return
			}
			if auth != nil && auth.Decision == "declined" {
				desc := "insufficient funds"
				if auth.DecisionRationale != nil && auth.DecisionRationale.Description != "" {
					desc = auth.DecisionRationale.Description
				}
				httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("bank declined deposit authorization: %s", desc))
				return
			}
		}
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.InitiateACHDeposit(r.Context(), &customerbllv1.InitiateACHDepositRequest{
		CustomerID:  session.CustomerID,
		PortfolioID: req.PortfolioID,
		AmountUSD:   req.AmountUSD,
		Frequency:   req.Frequency,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if b.cache != nil && session.CustomerID != "" {
		_ = b.cache.Delete(r.Context(), cachevalkey.KeyPortfolio(session.CustomerID))
	}

	httputil.WriteJSON(w, http.StatusCreated, resp)
}

func (b *CustomerBFF) handleGetBankAccounts(w http.ResponseWriter, r *http.Request) {
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}
	if custID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: customer identity required")
		return
	}

	if b.bank != nil {
		accs, err := b.bank.GetAccounts(r.Context(), custID)
		if err == nil && len(accs) > 0 {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"accounts": accs,
			})
			return
		}
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListBankAccounts(r.Context(), &customerbllv1.ListBankAccountsRequest{
		CustomerID: custID,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get bank accounts")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"accounts": resp.Accounts,
	})
}

func (b *CustomerBFF) handleListStatements(w http.ResponseWriter, r *http.Request) {
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}
	if custID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session or customer_id required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListStatements(r.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: custID,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, []*coreinvestv1.CustomerStatement{})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp.Statements)
}

func (b *CustomerBFF) handleGenerateStatement(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		Type        string `json:"type"` // "monthly_summary" or "form_1099b"
		PortfolioID string `json:"portfolio_id"`
		Year        int    `json:"year"`
		Month       int    `json:"month"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Year == 0 {
		req.Year = time.Now().Year()
	}
	if req.Type == "" {
		req.Type = "monthly_summary"
	}
	if req.Month == 0 && req.Type == "monthly_summary" {
		req.Month = int(time.Now().Month())
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GenerateStatement(r.Context(), &customerbllv1.GenerateStatementRequest{
		CustomerID:  session.CustomerID,
		PortfolioID: req.PortfolioID,
		Type:        req.Type,
		Year:        int32(req.Year),
		Month:       int32(req.Month),
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, resp.Statement)
}

func (b *CustomerBFF) handleDownloadStatement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = r.PathValue("id")
	}
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "statement id is required")
		return
	}

	session := CustomerFromContext(r.Context())
	custID := ""
	if session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.DownloadStatement(r.Context(), &customerbllv1.DownloadStatementRequest{
		CustomerID:  custID,
		StatementID: id,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to retrieve statement: %v", err))
		return
	}

	fileName := "statement.pdf"
	if resp.FileName != "" {
		fileName = resp.FileName
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.PDFBytes)
}
