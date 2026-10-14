package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

// Handler serves HTTP endpoints for the mock bank API.
type Handler struct {
	store  *ledger.LedgerStore
	engine *engine.SettlementEngine
}

// NewHandler creates a new Handler instance.
func NewHandler(store *ledger.LedgerStore, eng *engine.SettlementEngine) *Handler {
	return &Handler{
		store:  store,
		engine: eng,
	}
}

// RegisterRoutes registers banking endpoints on an http.ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.handleHealthz)

	// Plaid-style Auth & Link
	mux.HandleFunc("POST /link/token/create", h.handleCreateLinkToken)
	mux.HandleFunc("POST /item/public_token/exchange", h.handleExchangePublicToken)

	// Accounts & Balances
	mux.HandleFunc("GET /accounts", h.handleGetAccounts)
	mux.HandleFunc("POST /accounts/get", h.handleGetAccounts)
	mux.HandleFunc("POST /accounts/balance/get", h.handleGetBalances)

	// ACH Transfers
	mux.HandleFunc("POST /transfer/authorization/create", h.handleTransferAuthorization)
	mux.HandleFunc("POST /transfer/create", h.handleCreateTransfer)
	mux.HandleFunc("GET /transfer/get", h.handleGetTransfer)
	mux.HandleFunc("POST /transfer/get", h.handleGetTransfer)
	mux.HandleFunc("GET /transfer/list", h.handleListTransfers)

	// Sandbox / Simulation helpers
	mux.HandleFunc("POST /sandbox/transfer/simulate", h.handleSimulateTransfer)
	mux.HandleFunc("POST /sandbox/accounts/reset", h.handleResetAccounts)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "mock-bank",
	})
}

func (h *Handler) handleCreateLinkToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ClientUserID string `json:"client_user_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.ClientUserID == "" {
		req.ClientUserID = "cust-longterm-arthur"
	}

	tok := h.store.CreateLinkToken(req.ClientUserID)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"link_token": tok.LinkToken,
		"expiration": tok.ExpiresAt.Format(time.RFC3339),
		"request_id": tok.LinkToken,
	})
}

func (h *Handler) handleExchangePublicToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		PublicToken  string `json:"public_token"`
		ClientUserID string `json:"client_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.ClientUserID == "" {
		req.ClientUserID = "cust-longterm-arthur"
	}

	accessTok, err := h.store.ExchangePublicToken(req.PublicToken, req.ClientUserID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": accessTok.AccessToken,
		"item_id":      accessTok.ItemID,
		"request_id":   accessTok.AccessToken,
	})
}

func (h *Handler) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ownerID := r.URL.Query().Get("owner_id")
	accessToken := r.URL.Query().Get("access_token")

	// Also check body if POST
	if r.Method == http.MethodPost {
		var req struct {
			AccessToken string `json:"access_token"`
			OwnerID     string `json:"owner_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.AccessToken != "" {
				accessToken = req.AccessToken
			}
			if req.OwnerID != "" {
				ownerID = req.OwnerID
			}
		}
	}

	var accounts []*ledger.BankAccount
	if accessToken != "" {
		accs, err := h.store.GetAccountsByAccessToken(accessToken)
		if err == nil {
			accounts = accs
		}
	} else if ownerID != "" {
		accounts = h.store.GetAccountsByOwner(ownerID)
	} else {
		// Default to Arthur's accounts
		accounts = h.store.GetAccountsByOwner("cust-longterm-arthur")
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts":   accounts,
		"total_count": len(accounts),
	})
}

func (h *Handler) handleGetBalances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		AccessToken string   `json:"access_token"`
		AccountIDs  []string `json:"account_ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var accounts []*ledger.BankAccount
	if req.AccessToken != "" {
		accs, err := h.store.GetAccountsByAccessToken(req.AccessToken)
		if err == nil {
			accounts = accs
		}
	}

	if len(accounts) == 0 {
		if len(req.AccountIDs) > 0 {
			for _, id := range req.AccountIDs {
				if acc, err := h.store.GetAccount(id); err == nil {
					accounts = append(accounts, acc)
				}
			}
		} else {
			// fallback default
			accounts = h.store.GetAccountsByOwner("cust-longterm-arthur")
		}
	}

	type AccountBalanceResponse struct {
		AccountID string `json:"account_id"`
		Name      string `json:"name"`
		Balances  struct {
			Available        float64 `json:"available"`
			Current          float64 `json:"current"`
			IsoCurrencyCode string  `json:"iso_currency_code"`
		} `json:"balances"`
		Mask string `json:"mask"`
		Type string `json:"type"`
	}

	var respList []AccountBalanceResponse
	for _, acc := range accounts {
		var item AccountBalanceResponse
		item.AccountID = acc.AccountID
		item.Name = acc.Name
		item.Balances.Available = acc.AvailableBalance
		item.Balances.Current = acc.CurrentBalance
		item.Balances.IsoCurrencyCode = acc.Currency
		item.Mask = acc.AccountNumber
		item.Type = acc.AccountType
		respList = append(respList, item)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": respList,
	})
}

func (h *Handler) handleTransferAuthorization(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		AccountID string                 `json:"account_id"`
		Amount    ledger.FlexibleFloat64 `json:"amount"`
		Type      string                 `json:"type"` // debit or credit
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	acc, err := h.store.GetAccount(req.AccountID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "account not found"})
		return
	}

	amount := req.Amount.Float64()
	if amount <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "amount must be greater than zero"})
		return
	}

	if req.Type == "" || req.Type == "debit" {
		if acc.AvailableBalance < amount {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"authorization": map[string]interface{}{
					"id":       "auth_declined",
					"decision": "declined",
					"decision_rationale": map[string]string{
						"code":        "NSF",
						"description": "Insufficient funds in bank account",
					},
				},
			})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authorization": map[string]interface{}{
			"id":                 "auth_approved",
			"decision":           "approved",
			"decision_rationale": nil,
		},
	})
}

func (h *Handler) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		AccountID   string                 `json:"account_id"`
		Amount      ledger.FlexibleFloat64 `json:"amount"`
		Direction   string                 `json:"direction"` // debit or credit
		Description string                 `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Direction == "" {
		req.Direction = "debit"
	}
	if req.Description == "" {
		req.Description = "OCI Investment Capital ACH"
	}

	transfer, err := h.store.CreateTransfer(req.AccountID, req.Amount.Float64(), req.Direction, req.Description)
	if err != nil {
		if err == ledger.ErrInsufficientFunds {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error_code":    "INSUFFICIENT_FUNDS",
				"error_message": "insufficient funds to complete ACH transfer (R01)",
				"return_code":   "R01",
				"transfer":      transfer,
			})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer": transfer,
	})
}

func (h *Handler) handleGetTransfer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	transferID := r.URL.Query().Get("transfer_id")
	if transferID == "" && r.Method == http.MethodPost {
		var req struct {
			TransferID string `json:"transfer_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		transferID = req.TransferID
	}

	if transferID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "transfer_id is required"})
		return
	}

	transfer, err := h.store.GetTransfer(transferID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer": transfer,
	})
}

func (h *Handler) handleListTransfers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	accountID := r.URL.Query().Get("account_id")
	transfers := h.store.ListTransfers(accountID)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"transfers":   transfers,
		"total_count": len(transfers),
	})
}

func (h *Handler) handleSimulateTransfer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		TransferID string `json:"transfer_id"`
		EventType  string `json:"event_type"` // settled or returned
		ReturnCode string `json:"return_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var updated *ledger.ACHTransfer
	var err error

	if req.EventType == "returned" || req.EventType == "failed" {
		updated, err = h.engine.SimulateReturn(context.Background(), req.TransferID, req.ReturnCode)
	} else {
		updated, err = h.engine.SimulateSettlement(context.Background(), req.TransferID)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "simulated",
		"transfer": updated,
	})
}

func (h *Handler) handleResetAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	h.store.Seed()
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "reset_successful",
		"message": "in-memory bank accounts reset to seed baseline",
	})
}
