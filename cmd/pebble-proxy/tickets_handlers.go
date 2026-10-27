package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

func (h *GatewayHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit := int32(50)
	offset := int32(0)
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = int32(l)
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = int32(o)
	}

	source := r.URL.Query().Get("source")
	category := r.URL.Query().Get("category")
	state := r.URL.Query().Get("state")

	res, err := h.bllClient.ListTickets(r.Context(), &opsv1.ListTicketsRequest{
		Limit:    limit,
		Offset:   offset,
		Source:   source,
		Category: category,
		State:    state,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tickets := res.Tickets
	if tickets == nil {
		tickets = []*opsv1.ITTicket{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tickets)
}

func (h *GatewayHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		http.Error(w, "internal-bll client unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Title          string `json:"title"`
		Body           string `json:"body"`
		Severity       string `json:"severity"`
		Category       string `json:"category"`
		Source         string `json:"source"`
		AuthorUsername string `json:"author_username"`
		IsUrgent       bool   `json:"is_urgent"`
		ForgejoRepo    string `json:"forgejo_repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	res, err := h.bllClient.CreateTicket(r.Context(), &opsv1.CreateTicketRequest{
		Title:          req.Title,
		Body:           req.Body,
		Severity:       req.Severity,
		Category:       req.Category,
		Source:         req.Source,
		AuthorUsername: req.AuthorUsername,
		IsUrgent:       req.IsUrgent,
		ForgejoRepo:    req.ForgejoRepo,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res.Ticket)
}

func (h *GatewayHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.URL.Query().Get("id")
	}

	if h.bllClient == nil {
		http.Error(w, "ticket not found", http.StatusNotFound)
		return
	}

	res, err := h.bllClient.GetTicket(r.Context(), &opsv1.GetTicketRequest{
		TicketID: ticketID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res.Ticket)
}

func (h *GatewayHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.URL.Query().Get("id")
	}

	var req struct {
		State            string `json:"state"`
		AssigneeUsername string `json:"assignee_username"`
		ResolutionNotes  string `json:"resolution_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.bllClient == nil {
		http.Error(w, "internal-bll client unavailable", http.StatusServiceUnavailable)
		return
	}

	res, err := h.bllClient.UpdateTicket(r.Context(), &opsv1.UpdateTicketRequest{
		TicketID:         ticketID,
		State:            req.State,
		AssigneeUsername: req.AssigneeUsername,
		ResolutionNotes:  req.ResolutionNotes,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res.Ticket)
}

func (h *GatewayHandler) ListTicketComments(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.URL.Query().Get("id")
	}

	if h.bllClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"comments":[],"total":0}`))
		return
	}

	res, err := h.bllClient.ListTicketComments(r.Context(), &opsv1.ListTicketCommentsRequest{
		TicketID:        ticketID,
		IncludeInternal: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	comments := res.Comments
	if comments == nil {
		comments = []*opsv1.TicketComment{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"comments": comments,
		"total":    len(comments),
	})
}

func (h *GatewayHandler) AddTicketComment(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.URL.Query().Get("id")
	}

	var req struct {
		AuthorUsername string `json:"author_username"`
		Comment        string `json:"comment"`
		IsInternal     bool   `json:"is_internal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.bllClient == nil {
		http.Error(w, "internal-bll client unavailable", http.StatusServiceUnavailable)
		return
	}

	res, err := h.bllClient.AddTicketComment(r.Context(), &opsv1.AddTicketCommentRequest{
		TicketID:       ticketID,
		AuthorUsername: req.AuthorUsername,
		Comment:        req.Comment,
		IsInternal:     req.IsInternal,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res.Comment)
}

func (h *GatewayHandler) SendPushNotification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic    string `json:"topic"`
		Title    string `json:"title"`
		Message  string `json:"message"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.bllClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "dispatched",
			"provider": "ntfy",
			"topic":    req.Topic,
			"title":    req.Title,
			"message":  req.Message,
		})
		return
	}

	res, err := h.bllClient.SendPushNotification(r.Context(), &opsv1.SendPushNotificationRequest{
		Topic:    req.Topic,
		Title:    req.Title,
		Message:  req.Message,
		Priority: req.Priority,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *GatewayHandler) ListCustomerTickets(w http.ResponseWriter, r *http.Request) {
	custID := r.Header.Get("X-Customer-ID")
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}

	if h.bllClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
		return
	}

	res, err := h.bllClient.ListCustomerTickets(r.Context(), &opsv1.ListCustomerTicketsRequest{
		CustomerID: custID,
		Limit:      50,
		Offset:     0,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tickets := res.Tickets
	if tickets == nil {
		tickets = []*opsv1.ITTicket{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tickets)
}

func (h *GatewayHandler) CreateCustomerTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string `json:"customer_id"`
		Category   string `json:"category"`
		Title      string `json:"title"`
		Body       string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.CustomerID == "" {
		req.CustomerID = r.Header.Get("X-Customer-ID")
	}

	if h.bllClient == nil {
		http.Error(w, "internal-bll client unavailable", http.StatusServiceUnavailable)
		return
	}

	res, err := h.bllClient.CreateCustomerTicket(r.Context(), &opsv1.CreateCustomerTicketRequest{
		CustomerID: req.CustomerID,
		Category:   req.Category,
		Title:      req.Title,
		Body:       req.Body,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res.Ticket)
}
