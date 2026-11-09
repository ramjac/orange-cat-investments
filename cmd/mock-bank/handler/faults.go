package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type BankFaults struct {
	mu              sync.RWMutex
	Outage          bool `json:"outage"`
	LatencyMs       int  `json:"latency_ms"`
	RejectTransfers bool `json:"reject_transfers"`
}

func (f *BankFaults) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		f.mu.RLock()
		defer f.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(f)
	case http.MethodPut:
		var req BankFaults
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.Outage = req.Outage
		f.LatencyMs = req.LatencyMs
		f.RejectTransfers = req.RejectTransfers
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	case http.MethodDelete:
		f.mu.Lock()
		f.Outage = false
		f.LatencyMs = 0
		f.RejectTransfers = false
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (f *BankFaults) check(w http.ResponseWriter, action string) bool {
	f.mu.RLock()
	outage := f.Outage
	latency := f.LatencyMs
	rejectTransfers := f.RejectTransfers
	f.mu.RUnlock()

	if latency > 0 {
		time.Sleep(time.Duration(latency) * time.Millisecond)
	}

	if outage {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "mock bank service outage injected"})
		return true
	}

	if action == "transfer" && rejectTransfers {
		w.WriteHeader(http.StatusPreconditionFailed)
		json.NewEncoder(w).Encode(map[string]string{"error": "transfer rejected by injected fault"})
		return true
	}

	return false
}
