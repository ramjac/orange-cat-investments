package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func TestFaultInjection_Disabled(t *testing.T) {
	os.Setenv("ENABLE_FAULT_INJECTION", "false")
	store := ledger.NewLedgerStore()
	eng := engine.NewSettlementEngine(store, nil, 0)
	h := NewHandler(store, eng)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/sim/faults", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when disabled, got %d", rr.Code)
	}
}

func TestFaultInjection_Enabled(t *testing.T) {
	os.Setenv("ENABLE_FAULT_INJECTION", "true")
	store := ledger.NewLedgerStore()
	eng := engine.NewSettlementEngine(store, nil, 0)
	h := NewHandler(store, eng)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 1. Enable Outage
	req := httptest.NewRequest(http.MethodPut, "/sim/faults", strings.NewReader(`{"outage":true}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 setting fault, got %d", rr.Code)
	}

	// 2. Test Bank Endpoint
	req2 := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 outage, got %d", rr2.Code)
	}

	// 3. Reset Fault
	req3 := httptest.NewRequest(http.MethodDelete, "/sim/faults", nil)
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusNoContent {
		t.Fatalf("expected 204 resetting fault, got %d", rr3.Code)
	}

	// 4. Test Bank Endpoint (Should be healthy)
	req4 := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rr4 := httptest.NewRecorder()
	mux.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK {
		t.Errorf("expected 200 healthy, got %d", rr4.Code)
	}
}
