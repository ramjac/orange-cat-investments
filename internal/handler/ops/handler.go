package ops

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/orange-cat-investments/oci/internal/service/ops"
)

type Handler struct {
	service ops.Service
}

func NewHandler(service ops.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/ops/pebble/alerts", h.GetPebbleAlerts)
	mux.HandleFunc("POST /api/v1/ops/pebble/ack", h.SubmitPebbleAck)
	mux.HandleFunc("GET /api/v1/ops/tickets", h.ListTickets)
	mux.HandleFunc("POST /api/v1/ops/tickets", h.CreateTicket)
	mux.HandleFunc("POST /api/v1/ops/notifications/push", h.SendPushNotification)
	mux.HandleFunc("POST /api/v1/ops/notifications", h.SendPushNotification)

	// Fallback/direct path aliases
	mux.HandleFunc("GET /ops/pebble/alerts", h.GetPebbleAlerts)
	mux.HandleFunc("POST /ops/pebble/ack", h.SubmitPebbleAck)
	mux.HandleFunc("GET /ops/tickets", h.ListTickets)
	mux.HandleFunc("POST /ops/tickets", h.CreateTicket)
	mux.HandleFunc("POST /ops/notifications/push", h.SendPushNotification)
	mux.HandleFunc("POST /ops/notifications", h.SendPushNotification)
}

func (h *Handler) GetPebbleAlerts(w http.ResponseWriter, r *http.Request) {
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
	var req ops.PebbleAckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
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

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int32(20)
	offset := int32(0)

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = int32(l)
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = int32(o)
	}

	tickets, err := h.service.ListITTickets(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"items": tickets,
		"total": len(tickets),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

type createTicketRequest struct {
	ForgejoIssueID *int64 `json:"forgejo_issue_id,omitempty"`
	ForgejoRepo    string `json:"forgejo_repo"`
	Title          string `json:"title"`
	Body           string `json:"body,omitempty"`
	AuthorUsername string `json:"author_username"`
}

func (h *Handler) SendPushNotification(w http.ResponseWriter, r *http.Request) {
	var req ops.PushNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.service.SendPushNotification(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ticket, err := h.service.CreateITTicket(r.Context(), req.ForgejoRepo, req.Title, req.Body, req.AuthorUsername, req.ForgejoIssueID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ticket)
}
