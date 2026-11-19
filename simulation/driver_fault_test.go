package simulation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriver_TestFaultScenarios(t *testing.T) {
	var mu sync.Mutex
	bankState := map[string]any{}
	brokerState := map[string]any{}
	bankReset := false
	brokerReset := false

	mockBank := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sim/faults" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&bankState)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(bankState)
		case http.MethodDelete:
			bankState = map[string]any{}
			bankReset = true
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer mockBank.Close()

	mockBroker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sim/faults" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			brokerState = map[string]any{}
			_ = json.NewDecoder(r.Body).Decode(&brokerState)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(brokerState)
		case http.MethodDelete:
			brokerState = map[string]any{}
			brokerReset = true
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer mockBroker.Close()

	mockCustomerBFF := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/customer/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "customer_session", Value: "sess-test", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "csrf_token", Value: "csrf-test", Path: "/"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"session":{"session_id":"sess-test"}}`))
		case "/api/v1/customer/trading/orders":
			mu.Lock()
			outage, _ := brokerState["outage"].(bool)
			halted, _ := brokerState["halted_symbols"].([]any)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if outage {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"broker service unavailable"}`))
				return
			}
			if len(halted) > 0 {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"error":"broker rejected request: trading halted"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"order_id":"ord-1"}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer mockCustomerBFF.Close()

	cfg := DefaultConfig()
	cfg.Mode = "live"
	cfg.CustomerBFFURL = mockCustomerBFF.URL
	cfg.BankURL = mockBank.URL
	cfg.BrokerURL = mockBroker.URL
	cfg.FaultScenarios = true

	reg := &Registry{personas: make(map[string]*Persona)}
	reg.Register(&Persona{
		ID:      "cust-longterm-arthur",
		Name:    "Arthur Pendelton",
		Email:   "arthur@oci.local",
		Type:    "customer",
		Actions: []Action{},
	})

	drv, err := NewDriver(cfg)
	require.NoError(t, err)
	drv.registry = reg

	err = drv.testFaultScenarios(context.Background())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, bankReset, "mock-bank /sim/faults must be reset via defer")
	assert.True(t, brokerReset, "mock-broker /sim/faults must be reset via defer")
}
