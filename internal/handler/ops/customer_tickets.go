package ops

import (
	"encoding/json"
	"net/http"
	"strings"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

type createCustomerTicketRequest struct {
	CustomerID string `json:"customer_id"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	Body       string `json:"body"`
}

type addCustomerCommentRequest struct {
	AuthorUsername string `json:"author_username"`
	Comment        string `json:"comment"`
}

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

	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

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
}

func (h *Handler) CreateCustomerTicket(w http.ResponseWriter, r *http.Request) {
	var req createCustomerTicketRequest
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

	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

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
}

func (h *Handler) GetCustomerTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	resp, err := h.bllClient.GetTicket(r.Context(), &opsv1.GetTicketRequest{TicketID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp.Ticket)
}

func (h *Handler) ListCustomerTicketComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

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
}

func (h *Handler) AddCustomerTicketComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	var req addCustomerCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

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
}
