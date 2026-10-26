package employeebff

import (
	"encoding/json"
	"net/http"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
)

// RegisterQuantRoutes registers quant, research, and core-invest employee API routes.
// Invariant: EmployeeBFF delegates all business logic and domain persistence to Internal BLL over gRPC.
func (b *EmployeeBFF) RegisterQuantRoutes(mux *http.ServeMux) {
	// Streams
	listStreams := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		res, err := b.bllClient.ListStreams(r.Context(), &coreinvestv1.ListCameraStreamsRequest{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if res.Streams == nil {
			_ = json.NewEncoder(w).Encode([]*coreinvestv1.CameraStream{})
		} else {
			_ = json.NewEncoder(w).Encode(res.Streams)
		}
	}

	createStream := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			CameraAssetID string `json:"camera_asset_id"`
			AssetID       string `json:"asset_id"`
			FelineID      string `json:"feline_id"`
			StreamURL     string `json:"stream_url"`
			Resolution    string `json:"resolution"`
			Framerate     int32  `json:"framerate"`
			Protocol      string `json:"protocol"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		assetID := req.CameraAssetID
		if assetID == "" {
			assetID = req.AssetID
		}
		res, err := b.bllClient.RegisterStream(r.Context(), &coreinvestv1.RegisterCameraStreamRequest{
			AssetID:    assetID,
			FelineID:   req.FelineID,
			StreamURL:  req.StreamURL,
			Resolution: req.Resolution,
			Framerate:  req.Framerate,
			Protocol:   req.Protocol,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res.Stream)
	}

	mux.HandleFunc("GET /core-invest/streams", listStreams)
	mux.HandleFunc("GET /api/v1/core-invest/streams", listStreams)
	mux.HandleFunc("POST /core-invest/streams", createStream)
	mux.HandleFunc("POST /api/v1/core-invest/streams", createStream)

	// Brokerage Orders
	listOrders := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		portfolioID := r.URL.Query().Get("portfolio_id")
		res, err := b.bllClient.ListBrokerageOrders(r.Context(), &coreinvestv1.ListOrdersRequest{
			PortfolioID: portfolioID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if res.Orders == nil {
			_ = json.NewEncoder(w).Encode([]*coreinvestv1.BrokerageOrder{})
		} else {
			_ = json.NewEncoder(w).Encode(res.Orders)
		}
	}

	createOrder := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			PortfolioID string  `json:"portfolio_id"`
			Symbol      string  `json:"symbol"`
			Side        string  `json:"side"`
			Quantity    float64 `json:"quantity"`
			Price       float64 `json:"price"`
			OrderType   string  `json:"order_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.OrderType == "" {
			req.OrderType = "limit"
		}
		res, err := b.bllClient.ExecuteBrokerageOrder(r.Context(), &coreinvestv1.CreateOrderRequest{
			PortfolioID: req.PortfolioID,
			Symbol:      req.Symbol,
			Side:        req.Side,
			Quantity:    req.Quantity,
			Price:       req.Price,
			OrderType:   req.OrderType,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res.Order)
	}

	mux.HandleFunc("GET /core-invest/brokerage/orders", listOrders)
	mux.HandleFunc("GET /api/v1/core-invest/brokerage/orders", listOrders)
	mux.HandleFunc("POST /core-invest/brokerage/orders", createOrder)
	mux.HandleFunc("POST /api/v1/core-invest/brokerage/orders", createOrder)

	// Backtest Runs
	listBacktests := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		res, err := b.bllClient.ListBacktestRuns(r.Context(), &coreinvestv1.ListBacktestRunsRequest{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if res.Runs == nil {
			_ = json.NewEncoder(w).Encode([]*coreinvestv1.BacktestRun{})
		} else {
			_ = json.NewEncoder(w).Encode(res.Runs)
		}
	}

	createBacktest := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			ResearcherID   string  `json:"researcher_id"`
			StrategyName   string  `json:"strategy_name"`
			StrategyID     string  `json:"strategy_id"`
			StartDate      string  `json:"start_date"`
			EndDate        string  `json:"end_date"`
			InitialCapital float64 `json:"initial_capital"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		strategy := req.StrategyName
		if strategy == "" {
			strategy = req.StrategyID
		}
		if req.InitialCapital == 0 {
			req.InitialCapital = 100000.0
		}
		res, err := b.bllClient.StartBacktestRun(r.Context(), &coreinvestv1.StartBacktestRunRequest{
			ResearcherID:   req.ResearcherID,
			StrategyName:   strategy,
			StartDate:      req.StartDate,
			EndDate:        req.EndDate,
			InitialCapital: req.InitialCapital,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res.Run)
	}

	mux.HandleFunc("GET /core-invest/backtesting/runs", listBacktests)
	mux.HandleFunc("GET /api/v1/core-invest/backtesting/runs", listBacktests)
	mux.HandleFunc("POST /core-invest/backtesting/runs", createBacktest)
	mux.HandleFunc("POST /api/v1/core-invest/backtesting/runs", createBacktest)

	// Recurring Deposits
	listDeposits := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		customerID := r.URL.Query().Get("customer_id")
		res, err := b.bllClient.ListRecurringDeposits(r.Context(), &coreinvestv1.ListRecurringDepositsRequest{
			CustomerID: customerID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if res.Deposits == nil {
			_ = json.NewEncoder(w).Encode([]*coreinvestv1.RecurringDeposit{})
		} else {
			_ = json.NewEncoder(w).Encode(res.Deposits)
		}
	}

	createDeposit := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			CustomerID  string  `json:"customer_id"`
			PortfolioID string  `json:"portfolio_id"`
			AmountUSD   float64 `json:"amount_usd"`
			Frequency   string  `json:"frequency"`
			BankAccount string  `json:"bank_account"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := b.bllClient.CreateRecurringDeposit(r.Context(), &coreinvestv1.CreateRecurringDepositRequest{
			CustomerID:  req.CustomerID,
			PortfolioID: req.PortfolioID,
			AmountUSD:   req.AmountUSD,
			Frequency:   req.Frequency,
			BankAccount: req.BankAccount,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res.Deposit)
	}

	mux.HandleFunc("GET /core-invest/recurring-deposits", listDeposits)
	mux.HandleFunc("GET /api/v1/core-invest/recurring-deposits", listDeposits)
	mux.HandleFunc("POST /core-invest/recurring-deposits", createDeposit)
	mux.HandleFunc("POST /api/v1/core-invest/recurring-deposits", createDeposit)
}
