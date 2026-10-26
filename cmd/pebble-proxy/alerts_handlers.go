package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
)

func (h *GatewayHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"pebble-proxy"}`))
}

func (h *GatewayHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
		return
	}

	res, err := h.bllClient.ListAlerts(r.Context(), &opsv1.ListAlertsRequest{
		UnacknowledgedOnly: true,
	})
	if err != nil {
		h.logger.Error("failed to list alerts from internal-bll", "error", err)
		http.Error(w, `{"error":"failed to list alerts"}`, http.StatusInternalServerError)
		return
	}

	alerts := res.Alerts
	payloads := make([]*opssvc.PebbleAlertPayload, 0, len(alerts))
	for _, a := range alerts {
		priority := "HIGH"
		sev := strings.ToLower(a.Severity)
		if sev == "urgent" || sev == "critical" {
			priority = "CRITICAL"
		} else if sev == "low" {
			priority = "LOW"
		}

		createdAt := time.Now().UTC()
		if a.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, a.CreatedAt); err == nil {
				createdAt = t
			}
		}

		dict := map[uint32]string{
			opssvc.PebbleKeyCmd:       "1",
			opssvc.PebbleKeyTicketID:  a.TicketID,
			opssvc.PebbleKeyTitle:     a.Title,
			opssvc.PebbleKeyBody:      a.Body,
			opssvc.PebbleKeyPriority:  priority,
			opssvc.PebbleKeyAuthor:    a.Author,
			opssvc.PebbleKeyTimestamp: createdAt.Format(time.RFC3339),
		}

		payloads = append(payloads, &opssvc.PebbleAlertPayload{
			TicketID:       a.TicketID,
			Title:          a.Title,
			Priority:       priority,
			Body:           a.Body,
			Author:         a.Author,
			CreatedAt:      createdAt,
			AppMessageDict: dict,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payloads)
}

func (h *GatewayHandler) SubmitAck(w http.ResponseWriter, r *http.Request) {
	var req opssvc.PebbleAckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.TicketID == "" {
		http.Error(w, "ticket_id is required for Pebble ACK", http.StatusBadRequest)
		return
	}

	ackBy := req.AcknowledgedBy
	if ackBy == "" {
		if req.WatchID != "" {
			ackBy = req.WatchID
		} else if req.DeviceID != "" {
			ackBy = req.DeviceID
		} else if onCall := os.Getenv("DEFAULT_ON_CALL_ENGINEER"); onCall != "" {
			ackBy = onCall
		} else {
			ackBy = "emp-oncall-engineer"
		}
	}

	if h.bllClient == nil {
		http.Error(w, "internal-bll client unavailable", http.StatusServiceUnavailable)
		return
	}

	res, err := h.bllClient.AcknowledgeAlert(r.Context(), &opsv1.AcknowledgeAlertRequest{
		AlertID:        req.TicketID,
		AcknowledgedBy: ackBy,
		WatchID:        req.WatchID,
		DeviceID:       req.DeviceID,
		ButtonID:       int32(req.ButtonID),
	})
	if err != nil {
		h.logger.Error("failed to acknowledge alert via internal-bll", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ackTime := time.Now().UTC()
	if res.AcknowledgedAt != "" {
		if t, err := time.Parse(time.RFC3339, res.AcknowledgedAt); err == nil {
			ackTime = t
		}
	}

	resp := &opssvc.PebbleAckResponse{
		TicketID:       res.AlertID,
		State:          res.State,
		AcknowledgedAt: ackTime,
		AcknowledgedBy: res.AcknowledgedBy,
		StatusMessage:  res.StatusMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *GatewayHandler) HandleIncidentBroadcast(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IncidentID  string `json:"incident_id"`
		Title       string `json:"title"`
		Category    string `json:"category"`
		Severity    string `json:"severity"`
		RoomZone    string `json:"room_zone"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid incident payload"}`, http.StatusBadRequest)
		return
	}

	ticketID := req.IncidentID
	if ticketID == "" {
		ticketID = fmt.Sprintf("inc-%d", time.Now().UnixNano())
	}

	title := req.Title
	if req.RoomZone != "" {
		title = fmt.Sprintf("🚨 %s (%s)", req.Title, req.RoomZone)
	} else {
		title = fmt.Sprintf("🚨 %s", req.Title)
	}

	severity := req.Severity
	if severity == "" {
		severity = "critical"
	}

	ticketCategory := "habitat"
	if req.Category == "unauthorized_treat" {
		ticketCategory = "general"
	} else if req.Category == "software" || req.Category == "hardware" || req.Category == "trading" {
		ticketCategory = req.Category
	}

	desc := req.Description
	if req.IncidentID != "" {
		desc = fmt.Sprintf("[Workforce Incident ID: %s] %s", req.IncidentID, req.Description)
	}

	if h.bllClient == nil {
		http.Error(w, "internal-bll client unavailable", http.StatusServiceUnavailable)
		return
	}

	_, _ = h.bllClient.CreateAlert(r.Context(), &opsv1.CreateAlertRequest{
		TicketID: ticketID,
		Title:    title,
		Severity: severity,
		RoomZone: req.RoomZone,
	})

	ticketResp, err := h.bllClient.CreateTicket(r.Context(), &opsv1.CreateTicketRequest{
		Title:          title,
		Body:           desc,
		Severity:       severity,
		Category:       ticketCategory,
		Source:         "system",
		AuthorUsername: "incident-pipeline",
		IsUrgent:       true,
	})
	if err != nil {
		h.logger.Error("failed to create incident alert ticket for pebble", "error", err)
		http.Error(w, `{"error":"failed to create incident alert"}`, http.StatusInternalServerError)
		return
	}

	created := ticketResp.Ticket
	if created == nil {
		created = &opsv1.ITTicket{
			TicketID:       ticketID,
			Title:          title,
			Body:           desc,
			State:          "open",
			Severity:       severity,
			Category:       ticketCategory,
			Source:         "system",
			AuthorUsername: "incident-pipeline",
			IsUrgent:       true,
			CreatedAt:      time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
		}
	} else if created.TicketID == "" {
		created.TicketID = ticketID
	}

	h.logger.Info("registered workplace incident alert for Pebble wearable push", "ticket_id", created.TicketID, "severity", severity)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func requireServiceAuth(next http.HandlerFunc) http.HandlerFunc {
	expectedKey := os.Getenv("INTERNAL_SERVICE_KEY")
	if expectedKey == "" {
		expectedKey = "oci-internal-service-secret"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Internal-Service-Key")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if token != expectedKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized: valid internal service key required",
			})
			return
		}
		next(w, r)
	}
}
