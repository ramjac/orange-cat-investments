package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// IncidentBroker manages active SSE client connections and fans out incident events.
type IncidentBroker struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
	logger  *slog.Logger
}

// NewIncidentBroker creates an IncidentBroker instance.
func NewIncidentBroker(logger *slog.Logger) *IncidentBroker {
	if logger == nil {
		logger = slog.Default()
	}
	return &IncidentBroker{
		clients: make(map[chan []byte]bool),
		logger:  logger,
	}
}

// AddClient registers a new SSE client channel.
func (b *IncidentBroker) AddClient(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[ch] = true
}

// RemoveClient unregisters and closes an SSE client channel.
func (b *IncidentBroker) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[ch]; ok {
		delete(b.clients, ch)
		close(ch)
	}
}

// ClientCount returns the number of active SSE clients.
func (b *IncidentBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Broadcast sends raw incident JSON data to all active SSE subscribers.
func (b *IncidentBroker) Broadcast(msg []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			b.logger.Warn("dropping incident event for slow SSE client")
		}
	}
}

// HandleStream handles Server-Sent Events (SSE) connections from the Employee Web Portal.
func (b *IncidentBroker) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial connected handshake event
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	clientChan := make(chan []byte, 16)
	b.AddClient(clientChan)
	defer b.RemoveClient(clientChan)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: incident\ndata: %s\n\n", string(msg))
			flusher.Flush()
		}
	}
}

// HandleBroadcast receives an incident payload (from internal BLL, worker, or webhooks)
// and fans it out across all connected Employee Portal SSE sessions.
func (b *IncidentBroker) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Verify valid JSON
	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, `{"error":"invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	b.Broadcast(body)
	b.logger.Info("broadcasted workplace incident to SSE clients", "client_count", b.ClientCount())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"broadcasted"}`))
}
