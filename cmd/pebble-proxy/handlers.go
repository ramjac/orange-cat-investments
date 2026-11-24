package main

import (
	"log/slog"
	"sync"
	"time"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

const (
	alertCreatedAtTTL        = time.Hour
	maxAlertCreatedAtEntries = 512
)

type alertCreatedAtEntry struct {
	createdAt time.Time
	expiresAt time.Time
}

type GatewayHandler struct {
	bllClient        internalbllv1.InternalBLLServiceClient
	logger           *slog.Logger
	alertCreatedAtMu sync.RWMutex
	alertCreatedAt   map[string]alertCreatedAtEntry
}

func NewGatewayHandler(bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) *GatewayHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &GatewayHandler{
		bllClient:      bllClient,
		logger:         logger,
		alertCreatedAt: make(map[string]alertCreatedAtEntry),
	}
}

func (h *GatewayHandler) evictExpiredAlertsLocked(now time.Time) {
	for id, entry := range h.alertCreatedAt {
		if !now.Before(entry.expiresAt) {
			delete(h.alertCreatedAt, id)
		}
	}
}

func (h *GatewayHandler) recordAlertCreatedAt(ticketID string, createdAt time.Time) {
	if ticketID == "" || createdAt.IsZero() {
		return
	}
	now := time.Now().UTC()
	h.alertCreatedAtMu.Lock()
	defer h.alertCreatedAtMu.Unlock()

	if h.alertCreatedAt == nil {
		h.alertCreatedAt = make(map[string]alertCreatedAtEntry)
	}
	h.evictExpiredAlertsLocked(now)

	if _, exists := h.alertCreatedAt[ticketID]; !exists {
		for len(h.alertCreatedAt) >= maxAlertCreatedAtEntries {
			var oldestID string
			var oldestExp time.Time
			first := true
			for id, entry := range h.alertCreatedAt {
				if first || entry.expiresAt.Before(oldestExp) {
					oldestID = id
					oldestExp = entry.expiresAt
					first = false
				}
			}
			if oldestID == "" {
				break
			}
			delete(h.alertCreatedAt, oldestID)
		}
	}

	h.alertCreatedAt[ticketID] = alertCreatedAtEntry{
		createdAt: createdAt,
		expiresAt: now.Add(alertCreatedAtTTL),
	}
}

func (h *GatewayHandler) popAlertCreatedAt(ticketID string) (time.Time, bool) {
	if ticketID == "" {
		return time.Time{}, false
	}
	now := time.Now().UTC()
	h.alertCreatedAtMu.Lock()
	defer h.alertCreatedAtMu.Unlock()

	entry, ok := h.alertCreatedAt[ticketID]
	if ok {
		delete(h.alertCreatedAt, ticketID)
	}
	h.evictExpiredAlertsLocked(now)
	if !ok || !now.Before(entry.expiresAt) {
		return time.Time{}, false
	}
	return entry.createdAt, true
}

func (h *GatewayHandler) pruneInactiveAlerts(activeTicketIDs map[string]struct{}) {
	now := time.Now().UTC()
	h.alertCreatedAtMu.Lock()
	defer h.alertCreatedAtMu.Unlock()

	for id, entry := range h.alertCreatedAt {
		if !now.Before(entry.expiresAt) {
			delete(h.alertCreatedAt, id)
			continue
		}
		if _, active := activeTicketIDs[id]; !active {
			delete(h.alertCreatedAt, id)
		}
	}
}
