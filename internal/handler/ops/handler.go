package ops

import (
	"encoding/json"
	"net/http"

	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

type Handler struct {
	bllClient internalbllv1.InternalBLLServiceClient
	service   opssvc.Service
}

func NewHandler(service opssvc.Service) *Handler {
	return &Handler{service: service}
}

func NewHandlerWithBLL(bllClient internalbllv1.InternalBLLServiceClient) *Handler {
	return &Handler{bllClient: bllClient}
}

func (h *Handler) SetBLLClient(client internalbllv1.InternalBLLServiceClient) {
	h.bllClient = client
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Pebble Watch Endpoints
	mux.HandleFunc("GET /api/v1/ops/pebble/alerts", h.GetPebbleAlerts)
	mux.HandleFunc("POST /api/v1/ops/pebble/ack", h.SubmitPebbleAck)

	// General IT Helpdesk Tickets
	mux.HandleFunc("GET /api/v1/ops/tickets", h.ListTickets)
	mux.HandleFunc("POST /api/v1/ops/tickets", h.CreateTicket)
	mux.HandleFunc("GET /api/v1/ops/tickets/{id}", h.GetTicket)
	mux.HandleFunc("PUT /api/v1/ops/tickets/{id}", h.UpdateTicket)
	mux.HandleFunc("GET /api/v1/ops/tickets/{id}/comments", h.ListTicketComments)
	mux.HandleFunc("POST /api/v1/ops/tickets/{id}/comments", h.AddTicketComment)

	// Push Notifications
	mux.HandleFunc("POST /api/v1/ops/notifications/push", h.SendPushNotification)
	mux.HandleFunc("POST /api/v1/ops/notifications", h.SendPushNotification)

	// Customer Bug & Feedback Intake
	mux.HandleFunc("GET /api/v1/customer/support/tickets", h.ListCustomerTickets)
	mux.HandleFunc("POST /api/v1/customer/support/tickets", h.CreateCustomerTicket)
	mux.HandleFunc("GET /api/v1/customer/support/tickets/{id}", h.GetCustomerTicket)
	mux.HandleFunc("GET /api/v1/customer/support/tickets/{id}/comments", h.ListCustomerTicketComments)
	mux.HandleFunc("POST /api/v1/customer/support/tickets/{id}/comments", h.AddCustomerTicketComment)

	// Direct path aliases (without /api/v1)
	mux.HandleFunc("GET /ops/pebble/alerts", h.GetPebbleAlerts)
	mux.HandleFunc("POST /ops/pebble/ack", h.SubmitPebbleAck)
	mux.HandleFunc("GET /ops/tickets", h.ListTickets)
	mux.HandleFunc("POST /ops/tickets", h.CreateTicket)
	mux.HandleFunc("GET /ops/tickets/{id}", h.GetTicket)
	mux.HandleFunc("PUT /ops/tickets/{id}", h.UpdateTicket)
	mux.HandleFunc("GET /ops/tickets/{id}/comments", h.ListTicketComments)
	mux.HandleFunc("POST /ops/tickets/{id}/comments", h.AddTicketComment)
	mux.HandleFunc("POST /ops/notifications/push", h.SendPushNotification)
	mux.HandleFunc("POST /ops/notifications", h.SendPushNotification)
	mux.HandleFunc("GET /customer/support/tickets", h.ListCustomerTickets)
	mux.HandleFunc("POST /customer/support/tickets", h.CreateCustomerTicket)
	mux.HandleFunc("GET /customer/support/tickets/{id}", h.GetCustomerTicket)
	mux.HandleFunc("GET /customer/support/tickets/{id}/comments", h.ListCustomerTicketComments)
	mux.HandleFunc("POST /customer/support/tickets/{id}/comments", h.AddCustomerTicketComment)
}

func (h *Handler) GetPebbleAlerts(w http.ResponseWriter, r *http.Request) {
	if h.bllClient != nil {
		resp, err := h.bllClient.ListAlerts(r.Context(), &opsv1.ListAlertsRequest{UnacknowledgedOnly: true})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		alerts := resp.Alerts
		if alerts == nil {
			alerts = []*opsv1.PebbleAlert{}
		}
		_ = json.NewEncoder(w).Encode(alerts)
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	alerts, err := h.service.GetPendingAlerts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(alerts)
}

func (h *Handler) SubmitPebbleAck(w http.ResponseWriter, r *http.Request) {
	var req opssvc.PebbleAckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.AcknowledgeAlert(r.Context(), &opsv1.AcknowledgeAlertRequest{
			AlertID:        req.TicketID,
			AcknowledgedBy: req.AcknowledgedBy,
			ButtonID:       int32(req.ButtonID),
			WatchID:        req.WatchID,
			DeviceID:       req.DeviceID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	resp, err := h.service.ProcessPebbleAck(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
