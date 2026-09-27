package core_invest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/orange-cat-investments/oci/internal/service/core_invest"
)

type Handler struct {
	service core_invest.Service
}

func NewHandler(service core_invest.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /core-invest/streams", h.ListCameraStreams)
	mux.HandleFunc("POST /core-invest/streams", h.RegisterCameraStream)
	mux.HandleFunc("GET /core-invest/streams/{id}", h.GetCameraStreamByID)

	mux.HandleFunc("GET /core-invest/brokerage/orders", h.ListBrokerageOrders)
	mux.HandleFunc("POST /core-invest/brokerage/orders", h.ExecuteBrokerageOrder)
	mux.HandleFunc("GET /core-invest/brokerage/orders/{id}", h.GetBrokerageOrderByID)

	mux.HandleFunc("GET /core-invest/backtesting/runs", h.ListBacktestRuns)
	mux.HandleFunc("POST /core-invest/backtesting/runs", h.StartBacktestRun)
	mux.HandleFunc("GET /core-invest/backtesting/runs/{id}", h.GetBacktestRunByID)
}

func (h *Handler) GetCameraStreamByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		http.Error(w, "stream id is required", http.StatusBadRequest)
		return
	}

	stream, err := h.service.GetCameraStream(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stream)
}

func (h *Handler) ListCameraStreams(w http.ResponseWriter, r *http.Request) {
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

	streams, err := h.service.ListCameraStreams(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(streams)
}

type CreateStreamRequest struct {
	CameraAssetID string `json:"camera_asset_id"`
	StreamURL     string `json:"stream_url"`
	Protocol      string `json:"protocol"`
}

func (h *Handler) RegisterCameraStream(w http.ResponseWriter, r *http.Request) {
	var req CreateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	stream, err := h.service.RegisterCameraStream(r.Context(), req.CameraAssetID, req.StreamURL, req.Protocol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(stream)
}

func (h *Handler) ListBrokerageOrders(w http.ResponseWriter, r *http.Request) {
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

	orders, err := h.service.ListBrokerageOrders(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

type CreateOrderRequest struct {
	PortfolioID string  `json:"portfolio_id"`
	StrategyID  *string `json:"strategy_id,omitempty"`
	EventID     *string `json:"event_id,omitempty"`
	BrokerName  string  `json:"broker_name"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
}

func (h *Handler) ExecuteBrokerageOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.service.ExecuteBrokerageOrder(r.Context(), req.PortfolioID, req.BrokerName, req.Symbol, req.Side, req.Quantity, req.Price, req.StrategyID, req.EventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) GetBrokerageOrderByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		http.Error(w, "order id is required", http.StatusBadRequest)
		return
	}

	order, err := h.service.GetBrokerageOrder(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) ListBacktestRuns(w http.ResponseWriter, r *http.Request) {
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

	runs, err := h.service.ListBacktestRuns(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

type CreateBacktestRequest struct {
	StrategyID string    `json:"strategy_id"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Parameters string    `json:"parameters"`
}

func (h *Handler) StartBacktestRun(w http.ResponseWriter, r *http.Request) {
	var req CreateBacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	run, err := h.service.StartBacktestRun(r.Context(), req.StrategyID, req.StartDate, req.EndDate, req.Parameters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(run)
}

func (h *Handler) GetBacktestRunByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		http.Error(w, "backtest id is required", http.StatusBadRequest)
		return
	}

	run, err := h.service.GetBacktestRun(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}
