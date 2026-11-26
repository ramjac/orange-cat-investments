package customerbff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *CustomerBFF) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" && custID != "" {
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

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetPortfolio(r.Context(), &customerbllv1.GetPortfolioRequest{
		CustomerID:  custID,
		PortfolioID: portfolioID,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			httputil.WriteError(w, http.StatusNotFound, "portfolio not found")
			return
		}
		b.handleGRPCError(w, "GetPortfolio", err, http.StatusBadGateway)
		return
	}

	customerName := resp.CustomerName
	if session := CustomerFromContext(r.Context()); session != nil && session.Name != "" {
		customerName = session.Name
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
		"historical_performance": resp.HistoricalPerformance,
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
	session := CustomerFromContext(r.Context())
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListDeposits(r.Context(), &customerbllv1.ListDepositsRequest{
		CustomerID: custID,
	})
	if err != nil {
		b.handleGRPCError(w, "ListDeposits", err, http.StatusInternalServerError)
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

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.InitiateACHDeposit(r.Context(), &customerbllv1.InitiateACHDepositRequest{
		CustomerID:  session.CustomerID,
		PortfolioID: req.PortfolioID,
		BankAccount: req.BankAccount,
		AccountID:   req.AccountID,
		AmountUSD:   req.AmountUSD,
		Frequency:   req.Frequency,
		DayOfMonth:  int32(req.DayOfMonth),
	})
	if err != nil {
		b.handleGRPCError(w, "InitiateACHDeposit", err, http.StatusBadGateway)
		return
	}

	if b.cache != nil && session.CustomerID != "" {
		_ = b.cache.Delete(r.Context(), cachevalkey.KeyPortfolio(session.CustomerID))
	}

	httputil.WriteJSON(w, http.StatusCreated, resp)
}

func (b *CustomerBFF) handleGetBankAccounts(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListBankAccounts(r.Context(), &customerbllv1.ListBankAccountsRequest{
		CustomerID: custID,
	})
	if err != nil {
		b.handleGRPCError(w, "ListBankAccounts", err, http.StatusBadGateway)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"accounts": resp.Accounts,
	})
}

func (b *CustomerBFF) handleListStatements(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListStatements(r.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: custID,
	})
	if err != nil {
		b.log().Error("customer-bll gRPC call failed", "downstream", "customer-bll", "operation", "ListStatements", "grpc_code", status.Code(err).String(), "error", err)
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
		b.handleGRPCError(w, "GenerateStatement", err, http.StatusInternalServerError)
		return
	}

	if resp.DownloadURL != "" {
		w.Header().Set("X-PreSigned-Download-URL", resp.DownloadURL)
		if resp.Statement != nil {
			raw, _ := json.Marshal(resp.Statement)
			var payload map[string]any
			if json.Unmarshal(raw, &payload) == nil {
				payload["download_url"] = resp.DownloadURL
				httputil.WriteJSON(w, http.StatusCreated, payload)
				return
			}
		}
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
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.DownloadStatement(r.Context(), &customerbllv1.DownloadStatementRequest{
		CustomerID:  custID,
		StatementID: id,
	})
	if err != nil {
		b.handleGRPCError(w, "DownloadStatement", err, http.StatusInternalServerError)
		return
	}

	fileName := "statement.pdf"
	if resp.FileName != "" {
		fileName = resp.FileName
	}

	if resp.DownloadURL != "" {
		w.Header().Set("X-PreSigned-Download-URL", resp.DownloadURL)
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.PDFBytes)
}

func (b *CustomerBFF) handleGetPortfolioHistory(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" && custID != "" {
		portfolioID = fmt.Sprintf("port-%s", custID)
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "3M"
	}

	validPeriods := map[string]bool{"1M": true, "3M": true, "6M": true, "1Y": true, "YTD": true}
	if !validPeriods[period] {
		httputil.WriteError(w, http.StatusBadRequest, "invalid period")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetPortfolioHistory(r.Context(), &customerbllv1.GetPortfolioHistoryRequest{
		CustomerID:  custID,
		PortfolioID: portfolioID,
		Period:      period,
	})
	if err != nil {
		b.handleGRPCError(w, "GetPortfolioHistory", err, http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"points": resp.Points,
	})
}
