package ops

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
)

type Handler struct {
	service opssvc.Service
}

func NewHandler(service opssvc.Service) *Handler {
	return &Handler{service: service}
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

	limit := int32(50)
	offset := int32(0)

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = int32(l)
		if limit > 100 {
			limit = 100
		}
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = int32(o)
	}

	source := r.URL.Query().Get("source")
	var tickets []*opssvc.ITTicket
	var err error

	if source != "" {
		tickets, err = h.service.ListITTicketsBySource(r.Context(), source, limit, offset)
	} else {
		tickets, err = h.service.ListITTickets(r.Context(), limit, offset)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by category or state if requested
	category := r.URL.Query().Get("category")
	state := r.URL.Query().Get("state")
	severity := r.URL.Query().Get("severity")

	filtered := tickets
	if category != "" || state != "" || severity != "" {
		filtered = make([]*opssvc.ITTicket, 0, len(tickets))
		for _, t := range tickets {
			if category != "" && t.Category != category {
				continue
			}
			if state != "" && t.State != state {
				continue
			}
			if severity != "" && t.Severity != severity {
				continue
			}
			filtered = append(filtered, t)
		}
	}

	response := map[string]any{
		"items": filtered,
		"total": len(filtered),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		// Fallback for non-pattern routers
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	ticket, err := h.service.GetITTicket(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ticket)
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req opssvc.CreateITTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ticket, err := h.service.CreateITTicket(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ticket)
}

func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	var req opssvc.UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := h.service.UpdateITTicket(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) ListTicketComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		// e.g. /api/v1/ops/tickets/{id}/comments
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	comments, err := h.service.ListTicketComments(r.Context(), id, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"items": comments,
		"total": len(comments),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) AddTicketComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	var req opssvc.AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	comment, err := h.service.AddTicketComment(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(comment)
}

func (h *Handler) SendPushNotification(w http.ResponseWriter, r *http.Request) {
	var req opssvc.PushNotificationRequest
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

// Customer Support Endpoints

func extractCustomerID(r *http.Request) string {
	if custID := r.Header.Get("X-Customer-ID"); custID != "" {
		return custID
	}
	if custID := r.URL.Query().Get("customer_id"); custID != "" {
		return custID
	}
	if custID, ok := r.Context().Value("customer_id").(string); ok && custID != "" {
		return custID
	}
	return ""
}

func (h *Handler) ListCustomerTickets(w http.ResponseWriter, r *http.Request) {
	customerID := extractCustomerID(r)
	if customerID == "" {
		http.Error(w, "customer_id is required via X-Customer-ID header or query parameter", http.StatusUnauthorized)
		return
	}

	tickets, err := h.service.ListITTicketsByCustomer(r.Context(), customerID, 50, 0)
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

func (h *Handler) CreateCustomerTicket(w http.ResponseWriter, r *http.Request) {
	var req opssvc.CreateCustomerTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.CustomerID == "" {
		req.CustomerID = extractCustomerID(r)
	}
	if req.CustomerID == "" {
		http.Error(w, "customer_id is required", http.StatusUnauthorized)
		return
	}

	ticket, err := h.service.CreateCustomerTicket(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ticket)
}

func (h *Handler) GetCustomerTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	ticket, err := h.service.GetITTicket(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ticket)
}

func (h *Handler) ListCustomerTicketComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	// Customer view: internal notes MUST be scrubbed!
	comments, err := h.service.ListTicketComments(r.Context(), id, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"items": comments,
		"total": len(comments),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) AddCustomerTicketComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	var req opssvc.AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Customer comments are strictly public
	req.IsInternal = false
	if req.AuthorUsername == "" {
		req.AuthorUsername = extractCustomerID(r)
	}

	comment, err := h.service.AddTicketComment(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(comment)
}
