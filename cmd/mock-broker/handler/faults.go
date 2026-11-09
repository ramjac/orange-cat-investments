package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type BrokerFaults struct {
	mu            sync.RWMutex
	Outage        bool     `json:"outage"`
	LatencyMs     int      `json:"latency_ms"`
	RejectOrders  bool     `json:"reject_orders"`
	HaltedSymbols []string `json:"halted_symbols"`
}

func (f *BrokerFaults) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		f.mu.RLock()
		defer f.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(f)
	case http.MethodPut:
		var req BrokerFaults
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.Outage = req.Outage
		f.LatencyMs = req.LatencyMs
		f.RejectOrders = req.RejectOrders
		f.HaltedSymbols = req.HaltedSymbols
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	case http.MethodDelete:
		f.mu.Lock()
		f.Outage = false
		f.LatencyMs = 0
		f.RejectOrders = false
		f.HaltedSymbols = nil
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (f *BrokerFaults) check(w http.ResponseWriter, action string, symbol string) bool {
	f.mu.RLock()
	outage := f.Outage
	latency := f.LatencyMs
	rejectOrders := f.RejectOrders
	haltedSymbols := f.HaltedSymbols
	f.mu.RUnlock()

	if latency > 0 {
		time.Sleep(time.Duration(latency) * time.Millisecond)
	}

	if outage {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"message": "mock broker service outage injected"})
		return true
	}

	if action == "order" && rejectOrders {
		w.WriteHeader(http.StatusPreconditionFailed)
		json.NewEncoder(w).Encode(map[string]string{"message": "order rejected by injected fault"})
		return true
	}

	if symbol != "" {
		for _, s := range haltedSymbols {
			if s == symbol {
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]string{"message": "trading halted for symbol by injected fault"})
				return true
			}
		}
	}

	return false
}
