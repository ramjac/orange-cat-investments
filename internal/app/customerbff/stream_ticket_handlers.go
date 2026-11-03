package customerbff

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
)

func (b *CustomerBFF) handleGetESGTransparency(w http.ResponseWriter, r *http.Request) {
	if b.cache != nil {
		var cachedStr string
		if ok, _ := b.cache.Get(r.Context(), cachevalkey.KeyESGMetrics(), &cachedStr); ok && cachedStr != "" {
			data := []byte(cachedStr)
			if cachevalkey.HandleConditionalETag(w, r, data) {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
			return
		}
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetESGTransparency(r.Context(), &customerbllv1.GetESGTransparencyRequest{})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to fetch ESG transparency metrics")
		return
	}

	esgResp := map[string]any{
		"overall_welfare_score":    resp.OverallWelfareScore,
		"veterinary_adherence_pct": resp.VeterinaryAdherencePct,
		"nutritional_compliance":   resp.NutritionalCompliance,
		"avg_daily_nap_hours":      resp.AvgDailyNapHours,
		"perch_comfort_rating":     resp.PerchComfortRating,
		"workplace_incidents_cnt":  resp.WorkplaceIncidentsCnt,
		"medical_holds_issued_ytd": resp.MedicalHoldsIssuedYTD,
		"feline_executives":        resp.FelineExecutives,
	}

	data, err := json.Marshal(esgResp)
	if err == nil {
		if b.cache != nil {
			_ = b.cache.Set(r.Context(), cachevalkey.KeyESGMetrics(), string(data), cachevalkey.TTLESGMetrics)
		}
		if cachevalkey.HandleConditionalETag(w, r, data) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, esgResp)
}

func (b *CustomerBFF) handleGetStreams(w http.ResponseWriter, r *http.Request) {
	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListCameraStreams(r.Context(), &customerbllv1.ListCameraStreamsRequest{})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, []*coreinvestv1.CameraStream{})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp.Streams)
}

func (b *CustomerBFF) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")

	if b.searchClient == nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{
			"hits":  []any{},
			"query": q,
			"total": 0,
		})
		return
	}

	filter := ""
	if category != "" {
		filter = fmt.Sprintf("category = '%s'", category)
	}

	results, err := b.searchClient.Search(r.Context(), "customer_catalog", q, meilisearch.SearchOptions{
		Filter: filter,
		Limit:  25,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("search failed: %v", err))
		return
	}

	httputil.WriteJSON(w, http.StatusOK, results)
}

func (b *CustomerBFF) handleListCustomerTickets(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil || session.CustomerID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	custID := session.CustomerID

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListCustomerTickets(r.Context(), &customerbllv1.ListCustomerTicketsRequest{
		CustomerID: custID,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}

	tickets := resp.Tickets
	if tickets == nil {
		tickets = []*opsv1.ITTicket{}
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"items": tickets,
		"total": len(tickets),
	})
}

func (b *CustomerBFF) handleGetCustomerTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.PathValue("id")
	}
	if ticketID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "ticket id is required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetCustomerTicket(r.Context(), &customerbllv1.GetCustomerTicketRequest{
		TicketID: ticketID,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "ticket not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp.Ticket)
}

func (b *CustomerBFF) handleCreateCustomerTicket(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
		Severity    string `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		req.Title = req.Subject
	}
	if req.Body == "" {
		req.Body = req.Description
	}
	if req.Severity == "" {
		req.Severity = req.Priority
	}
	if req.Severity == "" {
		req.Severity = "normal"
	}
	if req.Title == "" || req.Body == "" {
		httputil.WriteError(w, http.StatusBadRequest, "title and body are required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.CreateCustomerTicket(r.Context(), &customerbllv1.CreateCustomerTicketRequest{
		CustomerID: session.CustomerID,
		Category:   req.Category,
		Title:      req.Title,
		Body:       req.Body,
		Severity:   req.Severity,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	t := resp.Ticket
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"ticket_id":   t.TicketID,
		"customer_id": t.CustomerID,
		"title":       t.Title,
		"subject":     t.Title,
		"body":        t.Body,
		"description": t.Body,
		"category":    t.Category,
		"severity":    t.Severity,
		"priority":    t.Severity,
		"status":      t.State,
		"state":       t.State,
		"created_at":  t.CreatedAt,
	})
}

func (b *CustomerBFF) handleListCustomerTicketComments(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.PathValue("id")
	}
	if ticketID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "ticket id is required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListCustomerTicketComments(r.Context(), &customerbllv1.ListCustomerTicketCommentsRequest{
		TicketID: ticketID,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}

	comments := resp.Comments
	if comments == nil {
		comments = []*opsv1.TicketComment{}
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"items": comments,
		"total": len(comments),
	})
}

func (b *CustomerBFF) handleAddCustomerTicketComment(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		ticketID = r.PathValue("id")
	}
	if ticketID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "ticket id is required")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Comment == "" {
		httputil.WriteError(w, http.StatusBadRequest, "comment cannot be empty")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.AddCustomerTicketComment(r.Context(), &customerbllv1.AddCustomerTicketCommentRequest{
		CustomerID:     session.CustomerID,
		TicketID:       ticketID,
		AuthorUsername: session.CustomerID,
		Comment:        req.Comment,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, resp.Comment)
}
