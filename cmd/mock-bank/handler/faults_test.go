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

func setupFaultTestServer(t *testing.T, enabled bool) *httptest.Server {
	t.Helper()
	t.Setenv("ENABLE_FAULT_INJECTION", boolToString(enabled))
	store := ledger.NewLedgerStore()
	eng := engine.NewSettlementEngine(store, nil, time.Millisecond)
	h := NewHandler(store, eng)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func setFaults(t *testing.T, ts *httptest.Server, payload string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/sim/faults", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("failed to build PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to PUT /sim/faults: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestBankFaults_NotRegisteredWhenDisabled(t *testing.T) {
	ts := setupFaultTestServer(t, false)

	resp, err := http.Get(ts.URL + "/sim/faults")
	if err != nil {
		t.Fatalf("failed to GET /sim/faults: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 when fault injection disabled, got %d", resp.StatusCode)
	}
}

func TestBankFaults_Outage(t *testing.T) {
	ts := setupFaultTestServer(t, true)

	if code := setFaults(t, ts, `{"outage":true}`); code != http.StatusOK {
		t.Fatalf("expected 200 setting outage, got %d", code)
	}

	resp, err := http.Post(ts.URL+"/accounts/balance/get", "application/json", bytes.NewReader([]byte(`{"access_token":"access-sandbox-arthur"}`)))
	if err != nil {
		t.Fatalf("failed to GET balances during outage: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 during outage, got %d", resp.StatusCode)
	}

	// GET state reports the active outage.
	stateResp, err := http.Get(ts.URL + "/sim/faults")
	if err != nil {
		t.Fatalf("failed to GET /sim/faults: %v", err)
	}
	defer stateResp.Body.Close()
	var state BankFaultsResponse
	_ = json.NewDecoder(stateResp.Body).Decode(&state)
	if !state.Outage {
		t.Errorf("expected outage=true in GET state, got %+v", state)
	}

	// DELETE resets to healthy.
	reset, err := http.NewRequest(http.MethodDelete, ts.URL+"/sim/faults", nil)
	if err != nil {
		t.Fatalf("failed to build DELETE request: %v", err)
	}
	resetResp, err := http.DefaultClient.Do(reset)
	if err != nil {
		t.Fatalf("failed to DELETE /sim/faults: %v", err)
	}
	resetResp.Body.Close()
	if resetResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 on reset, got %d", resetResp.StatusCode)
	}

	healthy, err := http.Post(ts.URL+"/accounts/balance/get", "application/json", bytes.NewReader([]byte(`{"access_token":"access-sandbox-arthur"}`)))
	if err != nil {
		t.Fatalf("failed to GET balances after reset: %v", err)
	}
	defer healthy.Body.Close()
	if healthy.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after reset, got %d", healthy.StatusCode)
	}
}

func TestBankFaults_RejectTransfers(t *testing.T) {
	ts := setupFaultTestServer(t, true)

	if code := setFaults(t, ts, `{"reject_transfers":true}`); code != http.StatusOK {
		t.Fatalf("expected 200 setting reject_transfers, got %d", code)
	}

	transferBody := bytes.NewReader([]byte(`{
		"account_id": "acc_arthur_chase_chk",
		"amount": 500.00,
		"direction": "debit"
	}`))
	resp, err := http.Post(ts.URL+"/transfer/create", "application/json", transferBody)
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("expected 422 for rejected transfer, got %d", resp.StatusCode)
	}

	// Non-transfer endpoints stay healthy.
	balances, err := http.Post(ts.URL+"/accounts/balance/get", "application/json", bytes.NewReader([]byte(`{"access_token":"access-sandbox-arthur"}`)))
	if err != nil {
		t.Fatalf("failed to GET balances: %v", err)
	}
	defer balances.Body.Close()
	if balances.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for balances during reject_transfers, got %d", balances.StatusCode)
	}
}

func TestBankFaults_Latency(t *testing.T) {
	ts := setupFaultTestServer(t, true)

	if code := setFaults(t, ts, `{"latency_ms":250}`); code != http.StatusOK {
		t.Fatalf("expected 200 setting latency, got %d", code)
	}

	balancePayload := bytes.NewReader([]byte(`{"access_token":"access-sandbox-arthur"}`))
	fast, err := http.Post(ts.URL+"/accounts/balance/get", "application/json", balancePayload)
	if err != nil {
		t.Fatalf("failed to GET balances: %v", err)
	}
	fast.Body.Close()

	start := time.Now()
	resp, err := http.Post(ts.URL+"/accounts/balance/get", "application/json", balancePayload)
	if err != nil {
		t.Fatalf("failed to GET balances with latency: %v", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with latency fault, got %d", resp.StatusCode)
	}
	if elapsed < 250*time.Millisecond {
		t.Errorf("expected at least 250ms of injected latency, got %v", elapsed)
	}
}
