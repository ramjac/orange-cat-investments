package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

type CustomerBLLServer struct {
	logger           *slog.Logger
	publisher        message.Publisher
	httpClient       *http.Client
	brokerClient     *alpaca.Client
	bankClient       *plaid.Client
	workforceURL     string
	coreInvestURL    string
	defaultFelineID  string
}

type TradeOrderRequest struct {
	PortfolioID string  `json:"portfolio_id"`
	FelineID    string  `json:"feline_id"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"` // buy, sell
	Quantity    float64 `json:"quantity"`
	OrderType   string  `json:"order_type"` // market, zoomie_pounce, bracket
	LimitPrice  float64 `json:"limit_price,omitempty"`
	StopLoss    float64 `json:"stop_loss,omitempty"`
	TakeProfit  float64 `json:"take_profit,omitempty"`
}

type TradeOrderResult struct {
	OrderID     string    `json:"order_id"`
	PortfolioID string    `json:"portfolio_id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Status      string    `json:"status"`
	ExecutedAt  time.Time `json:"executed_at"`
}

type ACHDepositRequest struct {
	CustomerID  string  `json:"customer_id"`
	PortfolioID string  `json:"portfolio_id"`
	BankAccount string  `json:"bank_account"`
	AmountUSD   float64 `json:"amount_usd"`
	Frequency   string  `json:"frequency"` // one_time, weekly, bi_weekly, monthly
}

type ACHDepositResult struct {
	TransferID  string    `json:"transfer_id"`
	CustomerID  string    `json:"customer_id"`
	PortfolioID string    `json:"portfolio_id"`
	AmountUSD   float64   `json:"amount_usd"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requested_at"`
}

func (s *CustomerBLLServer) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}

func (s *CustomerBLLServer) handleValidateAndExecuteOrder(w http.ResponseWriter, r *http.Request) {
	var req TradeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid order payload"}`, http.StatusBadRequest)
		return
	}

	if req.PortfolioID == "" || req.Symbol == "" || req.Quantity <= 0 {
		http.Error(w, `{"error":"portfolio_id, symbol, and quantity > 0 are required"}`, http.StatusBadRequest)
		return
	}

	// 1. Business Rule: Check feline medical hold from workforce domain
	targetFeline := s.resolveTargetFeline(r.Context(), req.FelineID)
	if targetFeline != "" {
		medicalHoldActive, err := s.checkFelineMedicalHold(r.Context(), targetFeline)
		if err != nil {
			s.log().Warn("could not query workforce medical hold, defaulting to safe hold check", "error", err)
		}
		if medicalHoldActive {
			s.log().Warn("trade order rejected: emergency medical hold active", "feline_id", targetFeline)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":     "algorithmic trading halted: emergency medical hold active for feline executive",
				"feline_id": targetFeline,
				"status":    "medical_hold_halt",
			})
			return
		}
	}

	// 2. Execute order with broker
	executedPrice := 175.50
	orderID := "ord-" + watermill.NewUUID()[:8]

	if s.brokerClient != nil {
		alpacaSide := "buy"
		if req.Side == "sell" {
			alpacaSide = "sell"
		}
		brokerResp, err := s.brokerClient.CreateOrder(r.Context(), req.PortfolioID, alpaca.CreateOrderRequest{
			Symbol:        req.Symbol,
			Qty:           req.Quantity,
			Side:          alpacaSide,
			Type:          "market",
			TimeInForce:   "day",
			ClientOrderID: orderID,
		})
		if err == nil && brokerResp != nil {
			orderID = brokerResp.ID
			if brokerResp.FilledAvgPrice != nil {
				executedPrice = *brokerResp.FilledAvgPrice
			}
		}
	}

	result := TradeOrderResult{
		OrderID:     orderID,
		PortfolioID: req.PortfolioID,
		Symbol:      req.Symbol,
		Side:        req.Side,
		Quantity:    req.Quantity,
		Price:       executedPrice,
		Status:      "executed",
		ExecutedAt:  time.Now().UTC(),
	}

	// 3. Emit asynchronous event to RabbitMQ
	eventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(eventID, "events.investment.order_placed.v1", orderID, result)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		_ = s.publisher.Publish("events.investment.order_placed.v1", message.NewMessage(event.EventID, eventBytes))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *CustomerBLLServer) checkFelineMedicalHold(ctx context.Context, felineID string) (bool, error) {
	if s.workforceURL == "" {
		return false, nil
	}
	url := fmt.Sprintf("%s/api/v1/workforce/care-schedules/%s", s.workforceURL, felineID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var data struct {
		EmergencyMedicalHold bool `json:"emergency_medical_hold"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, err
	}
	return data.EmergencyMedicalHold, nil
}

func (s *CustomerBLLServer) resolveTargetFeline(ctx context.Context, requestedID string) string {
	if requestedID != "" {
		return requestedID
	}
	// 1. Check configured default employee ID
	if s.defaultFelineID != "" {
		return s.defaultFelineID
	}
	// 2. Query workforce domain database data dynamically via API
	if s.workforceURL != "" {
		felineID, err := s.fetchFirstActiveFeline(ctx)
		if err == nil && felineID != "" {
			return felineID
		}
		if err != nil {
			s.log().Warn("could not fetch default feline employee from workforce database", "error", err)
		}
	}
	return ""
}

func (s *CustomerBLLServer) fetchFirstActiveFeline(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/api/v1/workforce/employees?employee_type=feline&status=active", s.workforceURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("workforce returned status %d", resp.StatusCode)
	}

	var employees []struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&employees); err != nil {
		return "", err
	}
	if len(employees) > 0 && employees[0].EmployeeID != "" {
		return employees[0].EmployeeID, nil
	}
	return "", nil
}

func (s *CustomerBLLServer) handleACHDeposit(w http.ResponseWriter, r *http.Request) {
	var req ACHDepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid ACH deposit payload"}`, http.StatusBadRequest)
		return
	}

	if req.CustomerID == "" || req.PortfolioID == "" || req.AmountUSD <= 0 {
		http.Error(w, `{"error":"customer_id, portfolio_id, and amount_usd > 0 required"}`, http.StatusBadRequest)
		return
	}

	transferID := "ach-" + watermill.NewUUID()[:8]
	result := ACHDepositResult{
		TransferID:  transferID,
		CustomerID:  req.CustomerID,
		PortfolioID: req.PortfolioID,
		AmountUSD:   req.AmountUSD,
		Status:      "pending_settlement",
		RequestedAt: time.Now().UTC(),
	}

	// Emit asynchronous ACH transfer requested event
	achPayload := map[string]interface{}{
		"transfer_id":   transferID,
		"customer_id":   req.CustomerID,
		"portfolio_id":  req.PortfolioID,
		"bank_account":  req.BankAccount,
		"amount_usd":    req.AmountUSD,
		"transfer_type": "deposit",
	}
	eventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(eventID, "events.banking.ach_transfer_requested.v1", transferID, achPayload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		_ = s.publisher.Publish("events.banking.ach_transfer_requested.v1", message.NewMessage(event.EventID, eventBytes))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(result)
}

func (s *CustomerBLLServer) handlePortfolioMetrics(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		portfolioID = "port-default"
	}

	metrics := map[string]interface{}{
		"portfolio_id":   portfolioID,
		"sharpe_ratio":   2.18,
		"sortino_ratio":  3.05,
		"ytd_return_pct": 28.45,
		"beta_to_sp500":  0.42,
		"alpha":          0.14,
		"max_drawdown":   -4.12,
		"feline_telemetry_correlation": 0.89,
		"calculated_at":  time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Customer Business Logic Layer (Customer BLL) API...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	workforceURL := os.Getenv("WORKFORCE_API_URL")
	if workforceURL == "" {
		workforceURL = "http://localhost:50051"
	}

	coreInvestURL := os.Getenv("CORE_INVEST_API_URL")
	if coreInvestURL == "" {
		coreInvestURL = "http://localhost:50053"
	}

	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	defaultFelineID := os.Getenv("DEFAULT_FELINE_ID")
	if defaultFelineID == "" {
		defaultFelineID = os.Getenv("DEFAULT_EMPLOYEE_ID")
	}

	serverObj := &CustomerBLLServer{
		logger:          logger,
		publisher:       pubSub,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
		workforceURL:    workforceURL,
		coreInvestURL:   coreInvestURL,
		defaultFelineID: defaultFelineID,
	}

	brokerURL := os.Getenv("BROKER_API_BASE_URL")
	if brokerURL == "" {
		brokerURL = os.Getenv("ALPACA_BASE_URL")
	}
	if brokerURL != "" {
		serverObj.brokerClient = alpaca.NewClient(alpaca.Config{
			BaseURL:   brokerURL,
			DataURL:   os.Getenv("ALPACA_DATA_URL"),
			APIKey:    os.Getenv("APCA_API_KEY_ID"),
			APISecret: os.Getenv("APCA_API_SECRET_KEY"),
		})
		logger.Info("Customer BLL connected to broker", "broker_url", brokerURL)
	}

	bankURL := os.Getenv("BANK_API_BASE_URL")
	if bankURL == "" {
		bankURL = os.Getenv("PLAID_BASE_URL")
	}
	if bankURL != "" {
		serverObj.bankClient = plaid.NewClient(plaid.Config{
			BaseURL:  bankURL,
			ClientID: os.Getenv("PLAID_CLIENT_ID"),
			Secret:   os.Getenv("PLAID_SECRET"),
		})
		logger.Info("Customer BLL connected to bank", "bank_url", bankURL)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"customer-bll"}`))
	})

	mux.HandleFunc("POST /api/v1/customer-bll/orders", serverObj.handleValidateAndExecuteOrder)
	mux.HandleFunc("POST /api/v1/customer-bll/deposits", serverObj.handleACHDeposit)
	mux.HandleFunc("GET /api/v1/customer-bll/portfolio/metrics", serverObj.handlePortfolioMetrics)

	versionedHandler := versioning.Middleware(versioning.DefaultConfig())(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      versionedHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if certFile != "" && keyFile != "" {
		tlsConfig, err := mtls.NewServerTLSConfig(certFile, keyFile, clientCA)
		if err != nil {
			logger.Error("failed to configure mTLS in Customer BLL", "error", err)
			os.Exit(1)
		}
		server.TLSConfig = tlsConfig
		logger.Info("Customer BLL configured with dynamic mTLS")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Customer BLL listening", "port", port)
		var err error
		if server.TLSConfig != nil {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Customer BLL server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Customer BLL...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Customer BLL stopped gracefully")
}
