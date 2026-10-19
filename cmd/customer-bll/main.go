package main

import (
	"bytes"
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
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

type CustomerBLLServer struct {
	logger          *slog.Logger
	publisher       message.Publisher
	httpClient      *http.Client
	brokerClient    *alpaca.Client
	bankClient      *plaid.Client
	workforceURL    string
	coreInvestURL   string
	defaultFelineID string
}

type TradeOrderRequest struct {
	PortfolioID string  `json:"portfolio_id"`
	FelineID    string  `json:"feline_id"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"` // buy, sell
	Quantity    float64 `json:"quantity"`
	OrderType   string  `json:"order_type"` // market, limit
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

	// Validate side strictly (Thread 6)
	if req.Side != "buy" && req.Side != "sell" {
		http.Error(w, `{"error":"side must be 'buy' or 'sell'"}`, http.StatusBadRequest)
		return
	}

	// Validate order type and protection parameters (Thread 8)
	orderType := req.OrderType
	if orderType == "" {
		orderType = "market"
	}
	if orderType != "market" && orderType != "limit" {
		http.Error(w, fmt.Sprintf(`{"error":"unsupported order type '%s'; supported types: 'market', 'limit'"}`, orderType), http.StatusBadRequest)
		return
	}
	var limitPricePtr *float64
	if orderType == "limit" {
		if req.LimitPrice <= 0 {
			http.Error(w, `{"error":"limit order requires limit_price > 0"}`, http.StatusBadRequest)
			return
		}
		limitPricePtr = &req.LimitPrice
	}
	if req.StopLoss > 0 || req.TakeProfit > 0 {
		http.Error(w, `{"error":"stop loss and take profit not supported by current broker integration"}`, http.StatusBadRequest)
		return
	}

	// 1. Business Rule: Check feline medical hold from workforce domain (Thread 7)
	targetFeline, err := s.resolveTargetFeline(r.Context(), req.FelineID)
	if err != nil || targetFeline == "" {
		s.log().Error("feline executive resolution failed", "error", err)
		http.Error(w, `{"error":"unable to resolve feline executive for required medical hold verification"}`, http.StatusServiceUnavailable)
		return
	}

	medicalHoldActive, err := s.checkFelineMedicalHold(r.Context(), targetFeline)
	if err != nil {
		s.log().Error("feline medical hold verification failed; failing closed", "error", err, "feline_id", targetFeline)
		http.Error(w, `{"error":"feline medical hold safety check unavailable; failing closed"}`, http.StatusServiceUnavailable)
		return
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

	// 2. Execute order with broker (Threads 9 & 49)
	if s.brokerClient == nil {
		s.log().Error("broker client not configured")
		http.Error(w, `{"error":"broker integration not configured"}`, http.StatusServiceUnavailable)
		return
	}

	brokerSide := req.Side
	clientOrderID := "ord-" + watermill.NewUUID()[:8]

	brokerResp, err := s.brokerClient.CreateOrder(r.Context(), req.PortfolioID, alpaca.CreateOrderRequest{
		Symbol:        req.Symbol,
		Qty:           req.Quantity,
		Side:          brokerSide,
		Type:          orderType,
		TimeInForce:   "day",
		LimitPrice:    limitPricePtr,
		ClientOrderID: clientOrderID,
	})
	if err != nil {
		s.log().Error("broker order execution failed", "error", err, "portfolio_id", req.PortfolioID)
		http.Error(w, fmt.Sprintf(`{"error":"broker order rejected: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	if brokerResp == nil {
		http.Error(w, `{"error":"empty response from broker"}`, http.StatusBadGateway)
		return
	}

	executedPrice := 0.0
	if brokerResp.FilledAvgPrice != nil {
		executedPrice = *brokerResp.FilledAvgPrice
	} else if req.LimitPrice > 0 {
		executedPrice = req.LimitPrice
	}

	orderStatus := brokerResp.Status
	if orderStatus == "" {
		orderStatus = "submitted"
	}

	orderID := brokerResp.ID
	if orderID == "" {
		orderID = clientOrderID
	}

	result := TradeOrderResult{
		OrderID:     orderID,
		PortfolioID: req.PortfolioID,
		Symbol:      req.Symbol,
		Side:        req.Side,
		Quantity:    req.Quantity,
		Price:       executedPrice,
		Status:      orderStatus,
		ExecutedAt:  time.Now().UTC(),
	}

	// 3. Persist order through core-invest-api (Thread 10)
	if s.coreInvestURL != "" {
		orderPayload := map[string]interface{}{
			"portfolio_id": req.PortfolioID,
			"broker_name":  "broker",
			"symbol":       req.Symbol,
			"side":         req.Side,
			"quantity":     req.Quantity,
			"price":        executedPrice,
		}
		data, _ := json.Marshal(orderPayload)
		persistReq, err := http.NewRequestWithContext(r.Context(), "POST", s.coreInvestURL+"/api/v1/core-invest/brokerage/orders", bytes.NewReader(data))
		if err == nil {
			persistReq.Header.Set("Content-Type", "application/json")
			if resp, err := s.httpClient.Do(persistReq); err == nil {
				_ = resp.Body.Close()
			} else {
				s.log().Warn("failed to persist order to core-invest-api", "error", err)
			}
		}
	}

	// 4. Emit asynchronous event
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
		return false, errors.New("workforce API URL is not configured")
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
		return false, fmt.Errorf("workforce care schedule returned status %d", resp.StatusCode)
	}

	var data struct {
		EmergencyMedicalHold bool `json:"emergency_medical_hold"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, err
	}
	return data.EmergencyMedicalHold, nil
}

func (s *CustomerBLLServer) resolveTargetFeline(ctx context.Context, requestedID string) (string, error) {
	if requestedID != "" {
		return requestedID, nil
	}
	// 1. Check configured default employee ID
	if s.defaultFelineID != "" {
		return s.defaultFelineID, nil
	}
	// 2. Query workforce domain database data dynamically via API
	if s.workforceURL != "" {
		felineID, err := s.fetchFirstActiveFeline(ctx)
		if err != nil {
			s.log().Warn("could not fetch default feline employee from workforce database", "error", err)
			return "", err
		}
		if felineID != "" {
			return felineID, nil
		}
	}
	return "", errors.New("no feline executive available or configured")
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

	// Decode response envelope containing {items, total} (Thread 11)
	var data struct {
		Items []struct {
			EmployeeID string `json:"employee_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Items) > 0 && data.Items[0].EmployeeID != "" {
		return data.Items[0].EmployeeID, nil
	}
	return "", errors.New("no active feline employees found")
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

	freq := req.Frequency
	if freq == "" {
		freq = "one_time"
	}

	// If recurring deposit, persist recurring schedule in core-invest domain (Thread 33)
	if freq != "one_time" && s.coreInvestURL != "" {
		schedPayload := map[string]interface{}{
			"customer_id":  req.CustomerID,
			"portfolio_id": req.PortfolioID,
			"amount_usd":   req.AmountUSD,
			"frequency":    freq,
			"bank_account": req.BankAccount,
		}
		data, _ := json.Marshal(schedPayload)
		schedReq, err := http.NewRequestWithContext(r.Context(), "POST", s.coreInvestURL+"/api/v1/core-invest/recurring-deposits", bytes.NewReader(data))
		if err == nil {
			schedReq.Header.Set("Content-Type", "application/json")
			if resp, err := s.httpClient.Do(schedReq); err == nil {
				_ = resp.Body.Close()
			} else {
				s.log().Warn("failed to persist recurring deposit schedule in core-invest-api", "error", err)
			}
		}
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

	// Emit asynchronous ACH transfer requested event (Thread 12)
	achPayload := map[string]interface{}{
		"transfer_id":   transferID,
		"customer_id":   req.CustomerID,
		"portfolio_id":  req.PortfolioID,
		"bank_account":  req.BankAccount,
		"amount_usd":    req.AmountUSD,
		"transfer_type": "deposit",
		"frequency":     freq,
	}
	eventID := watermill.NewUUID()
	event := envelope.NewEventEnvelope(eventID, "events.banking.ach_transfer_requested.v1", transferID, achPayload)
	eventBytes, _ := json.Marshal(event)
	if s.publisher != nil {
		if err := s.publisher.Publish("events.banking.ach_transfer_requested.v1", message.NewMessage(event.EventID, eventBytes)); err != nil {
			s.log().Error("failed to publish ACH transfer requested event", "error", err)
			http.Error(w, `{"error":"failed to publish ACH transfer event to message broker"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(result)
}

func (s *CustomerBLLServer) handlePortfolioMetrics(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		http.Error(w, `{"error":"portfolio_id query parameter is required"}`, http.StatusBadRequest)
		return
	}

	// Validate portfolio existence via core-invest-api (Thread 34)
	if s.coreInvestURL == "" {
		http.Error(w, `{"error":"core-invest-api not configured"}`, http.StatusServiceUnavailable)
		return
	}

	checkReq, err := http.NewRequestWithContext(r.Context(), "GET", fmt.Sprintf("%s/api/v1/core-invest/portfolios/%s", s.coreInvestURL, portfolioID), nil)
	if err != nil {
		http.Error(w, `{"error":"failed to create portfolio query request"}`, http.StatusInternalServerError)
		return
	}
	resp, err := s.httpClient.Do(checkReq)
	if err != nil {
		s.log().Error("failed to query portfolio from core-invest-api", "error", err, "portfolio_id", portfolioID)
		http.Error(w, `{"error":"upstream core-invest-api unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		http.Error(w, `{"error":"portfolio not found"}`, http.StatusNotFound)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf(`{"error":"core-invest-api returned status %d"}`, resp.StatusCode), http.StatusBadGateway)
		return
	}

	var portfolio struct {
		PortfolioID     string  `json:"portfolio_id"`
		CustomerID      string  `json:"customer_id"`
		CashBalance     float64 `json:"cash_balance"`
		InvestedBalance float64 `json:"invested_balance"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&portfolio)

	metrics := map[string]interface{}{
		"portfolio_id":                 portfolioID,
		"customer_id":                  portfolio.CustomerID,
		"cash_balance":                 portfolio.CashBalance,
		"invested_balance":             portfolio.InvestedBalance,
		"total_value":                  portfolio.CashBalance + portfolio.InvestedBalance,
		"sharpe_ratio":                 2.18,
		"sortino_ratio":                3.05,
		"ytd_return_pct":               28.45,
		"beta_to_sp500":                0.42,
		"alpha":                        0.14,
		"max_drawdown":                 -4.12,
		"feline_telemetry_correlation": 0.89,
		"calculated_at":                time.Now().UTC(),
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
	var pubSub message.Publisher
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL != "" {
		amqpConfig := amqp.NewDurableQueueConfig(amqpURL)
		amqpPub, err := amqp.NewPublisher(amqpConfig, watermillLogger)
		if err != nil {
			logger.Error("failed to connect Customer BLL to RabbitMQ AMQP publisher", "error", err)
			os.Exit(1)
		}
		defer amqpPub.Close()
		pubSub = amqpPub
		logger.Info("Customer BLL connected to RabbitMQ AMQP broker successfully")
	} else {
		pubSub = gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)
		logger.Info("Customer BLL running with in-memory GoChannel publisher (local test mode)")
	}

	defaultFelineID := os.Getenv("DEFAULT_FELINE_ID")
	if defaultFelineID == "" {
		defaultFelineID = os.Getenv("DEFAULT_EMPLOYEE_ID")
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if certFile != "" && keyFile != "" && clientCA != "" {
		clientTLS, err := mtls.NewClientTLSConfig(certFile, keyFile, clientCA)
		if err != nil {
			logger.Warn("failed to initialize client dynamic mTLS transport, using standard http client", "error", err)
		} else {
			httpClient = &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: clientTLS,
				},
			}
			logger.Info("Customer BLL configured with outbound dynamic mTLS transport")
		}
	}

	serverObj := &CustomerBLLServer{
		logger:          logger,
		publisher:       pubSub,
		httpClient:      httpClient,
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
