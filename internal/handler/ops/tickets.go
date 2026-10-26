package ops

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

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
	category := r.URL.Query().Get("category")
	state := r.URL.Query().Get("state")
	severity := r.URL.Query().Get("severity")

	if h.bllClient != nil {
		resp, err := h.bllClient.ListTickets(r.Context(), &opsv1.ListTicketsRequest{
			Source: source,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tickets := resp.Tickets
		filtered := tickets
		if category != "" || state != "" || severity != "" {
			filtered = make([]*opsv1.ITTicket, 0, len(tickets))
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": filtered,
			"total": len(filtered),
		})
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

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
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.GetTicket(r.Context(), &opsv1.GetTicketRequest{TicketID: id})
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp.Ticket)
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
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

	if h.bllClient != nil {
		resp, err := h.bllClient.CreateTicket(r.Context(), &opsv1.CreateTicketRequest{
			ForgejoRepo:    req.ForgejoRepo,
			Title:          req.Title,
			Body:           req.Body,
			AuthorUsername: req.AuthorUsername,
			Severity:       req.Severity,
			Category:       req.Category,
			Source:         req.Source,
			CustomerID:     req.CustomerID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp.Ticket)
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
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

	if h.bllClient != nil {
		resp, err := h.bllClient.UpdateTicket(r.Context(), &opsv1.UpdateTicketRequest{
			TicketID:         id,
			State:            req.State,
			AssigneeUsername: req.AssigneeUsername,
			ResolutionNotes:  req.ResolutionNotes,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp.Ticket)
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
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
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.ListTicketComments(r.Context(), &opsv1.ListTicketCommentsRequest{
			TicketID: id,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": resp.Comments,
			"total": len(resp.Comments),
		})
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
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

	if h.bllClient != nil {
		resp, err := h.bllClient.AddTicketComment(r.Context(), &opsv1.AddTicketCommentRequest{
			TicketID:       id,
			AuthorUsername: req.AuthorUsername,
			Comment:        req.Comment,
			IsInternal:     req.IsInternal,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp.Comment)
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
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

	if h.bllClient != nil {
		resp, err := h.bllClient.SendPushNotification(r.Context(), &opsv1.SendPushNotificationRequest{
			Topic:    req.Topic,
			Title:    req.Title,
			Message:  req.Message,
			Priority: req.Priority,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
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

	res, err := h.service.SendPushNotification(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
