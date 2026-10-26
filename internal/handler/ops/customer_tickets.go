package ops

import (
	"encoding/json"
	"net/http"
	"strings"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
)

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

	if h.bllClient != nil {
		resp, err := h.bllClient.ListCustomerTickets(r.Context(), &opsv1.ListCustomerTicketsRequest{
			CustomerID: customerID,
			Limit:      50,
			Offset:     0,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": resp.Tickets,
			"total": len(resp.Tickets),
		})
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
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

	if h.bllClient != nil {
		resp, err := h.bllClient.CreateCustomerTicket(r.Context(), &opsv1.CreateCustomerTicketRequest{
			CustomerID: req.CustomerID,
			Category:   req.Category,
			Title:      req.Title,
			Body:       req.Body,
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

func (h *Handler) ListCustomerTicketComments(w http.ResponseWriter, r *http.Request) {
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
		var publicComments []*opsv1.TicketComment
		for _, c := range resp.Comments {
			if !c.IsInternal {
				publicComments = append(publicComments, c)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": publicComments,
			"total": len(publicComments),
		})
		return
	}

	if h.service == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

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

	if h.bllClient != nil {
		author := req.AuthorUsername
		if author == "" {
			author = extractCustomerID(r)
		}
		resp, err := h.bllClient.AddTicketComment(r.Context(), &opsv1.AddTicketCommentRequest{
			TicketID:       id,
			AuthorUsername: author,
			Comment:        req.Comment,
			IsInternal:     false,
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
