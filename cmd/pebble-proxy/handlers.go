package main

import (
	"log/slog"
	"sync"
	"time"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

type GatewayHandler struct {
	bllClient        internalbllv1.InternalBLLServiceClient
	logger           *slog.Logger
	alertCreatedAtMu sync.RWMutex
	alertCreatedAt   map[string]time.Time
}

func NewGatewayHandler(bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) *GatewayHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &GatewayHandler{
		bllClient:      bllClient,
		logger:         logger,
		alertCreatedAt: make(map[string]time.Time),
	}
}

func (h *GatewayHandler) recordAlertCreatedAt(ticketID string, createdAt time.Time) {
	if ticketID == "" || createdAt.IsZero() {
		return
	}
	h.alertCreatedAtMu.Lock()
	if h.alertCreatedAt == nil {
		h.alertCreatedAt = make(map[string]time.Time)
	}
	h.alertCreatedAt[ticketID] = createdAt
	h.alertCreatedAtMu.Unlock()
}

func (h *GatewayHandler) popAlertCreatedAt(ticketID string) (time.Time, bool) {
	if ticketID == "" {
		return time.Time{}, false
	}
	h.alertCreatedAtMu.Lock()
	createdAt, ok := h.alertCreatedAt[ticketID]
	if ok {
		delete(h.alertCreatedAt, ticketID)
	}
	h.alertCreatedAtMu.Unlock()
	return createdAt, ok
}
