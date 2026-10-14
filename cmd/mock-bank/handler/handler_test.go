package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func setupTestServer() (*httptest.Server, *ledger.LedgerStore, *engine.SettlementEngine) {
	store := ledger.NewLedgerStore()
	eng := engine.NewSettlementEngine(store, nil, 100*time.Millisecond)
	h := NewHandler(store, eng)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	return ts, store, eng
}

func TestHandler_Healthz(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("failed to GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_LinkAndExchange(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	// 1. Create Link Token
	body := bytes.NewReader([]byte(`{"client_user_id":"cust-longterm-arthur"}`))
	resp, err := http.Post(ts.URL+"/link/token/create", "application/json", body)
	if err != nil {
		t.Fatalf("failed to create link token: %v", err)
	}
	defer resp.Body.Close()

	var linkResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&linkResp)
	if linkResp["link_token"] == nil {
		t.Fatal("expected link_token in response")
	}

	// 2. Exchange public token
	exchangeBody := bytes.NewReader([]byte(`{"public_token":"public-test-token","client_user_id":"cust-longterm-arthur"}`))
	resp2, err := http.Post(ts.URL+"/item/public_token/exchange", "application/json", exchangeBody)
	if err != nil {
		t.Fatalf("failed to exchange public token: %v", err)
	}
	defer resp2.Body.Close()

	var exchResp map[string]interface{}
	_ = json.NewDecoder(resp2.Body).Decode(&exchResp)
	if exchResp["access_token"] == nil {
		t.Fatal("expected access_token in response")
	}
}

func TestHandler_GetBalances(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	body := bytes.NewReader([]byte(`{"access_token":"access-sandbox-arthur"}`))
	resp, err := http.Post(ts.URL+"/accounts/balance/get", "application/json", body)
	if err != nil {
		t.Fatalf("failed to get balances: %v", err)
	}
	defer resp.Body.Close()

	var balResp struct {
		Accounts []struct {
			AccountID string `json:"account_id"`
			Balances  struct {
				Available float64 `json:"available"`
				Current   float64 `json:"current"`
			} `json:"balances"`
		} `json:"accounts"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&balResp)

	if len(balResp.Accounts) != 2 {
		t.Fatalf("expected 2 accounts for Arthur, got %d", len(balResp.Accounts))
	}
	if balResp.Accounts[0].Balances.Available <= 0 {
		t.Errorf("expected positive available balance, got %f", balResp.Accounts[0].Balances.Available)
	}
}

func TestHandler_CreateTransfer_SuccessAndNSF(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	// 1. Successful transfer
	trBody := bytes.NewReader([]byte(`{
		"account_id": "acc_arthur_chase_chk",
		"amount": 500.00,
		"direction": "debit",
		"description": "Arthur Monthly Investment"
	}`))
	resp, err := http.Post(ts.URL+"/transfer/create", "application/json", trBody)
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var trResp struct {
		Transfer struct {
			TransferID string  `json:"transfer_id"`
			Status     string  `json:"status"`
			Amount     float64 `json:"amount"`
		} `json:"transfer"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&trResp)
	if trResp.Transfer.TransferID == "" || trResp.Transfer.Status != "pending" {
		t.Errorf("unexpected transfer response: %+v", trResp)
	}

	// 2. Insufficient Funds Transfer (Chloe has 22,400 available, request 50,000)
	nsfBody := bytes.NewReader([]byte(`{
		"account_id": "acc_chloe_svcu_chk",
		"amount": 50000.00,
		"direction": "debit"
	}`))
	resp2, err := http.Post(ts.URL+"/transfer/create", "application/json", nsfBody)
	if err != nil {
		t.Fatalf("failed to post NSF transfer: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on NSF, got %d", resp2.StatusCode)
	}

	var nsfResp map[string]interface{}
	_ = json.NewDecoder(resp2.Body).Decode(&nsfResp)
	if nsfResp["return_code"] != "R01" {
		t.Errorf("expected return_code R01, got %v", nsfResp["return_code"])
	}
}
