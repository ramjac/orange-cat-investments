package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerBLL_Healthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"customer-bll"}`))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"service":"customer-bll"`)
}

func TestCustomerBLL_OrderExecution_Validations(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	bll := &CustomerBLLServer{
		publisher:  pubSub,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/orders", bll.handleValidateAndExecuteOrder)

	// 1. Missing fields
	reqBody, _ := json.Marshal(map[string]interface{}{"portfolio_id": ""})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(reqBody)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// 2. Invalid side (Thread 6)
	reqBody, _ = json.Marshal(TradeOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "hold",
		Quantity:    10,
	})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(reqBody)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "side must be 'buy' or 'sell'")

	// 3. Unsupported order type (Thread 8)
	reqBody, _ = json.Marshal(TradeOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    10,
		OrderType:   "bracket",
	})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(reqBody)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported order type")

	// 4. Limit order with price <= 0 (Thread 8)
	reqBody, _ = json.Marshal(TradeOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    10,
		OrderType:   "limit",
		LimitPrice:  0,
	})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(reqBody)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "limit order requires limit_price > 0")
}

func TestCustomerBLL_OrderExecution_NormalAndMedicalHold(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	// Mock workforce service
	medicalHoldStatus := false
	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"feline_id":              "emp-feline-garfield",
			"emergency_medical_hold": medicalHoldStatus,
		})
	}))
	defer mockWorkforceServer.Close()

	// Mock broker service
	mockBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		price := 182.50
		_ = json.NewEncoder(w).Encode(alpaca.Order{
			ID:             "ord-alpaca-999",
			Status:         "filled",
			FilledAvgPrice: &price,
			FilledQty:      10,
		})
	}))
	defer mockBrokerServer.Close()

	var orderPersisted atomic.Bool
	mockCoreInvestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/brokerage/orders") {
			orderPersisted.Store(true)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "persisted"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockCoreInvestServer.Close()

	bll := &CustomerBLLServer{
		publisher:     pubSub,
		httpClient:    mockWorkforceServer.Client(),
		brokerClient:  alpaca.NewClient(alpaca.Config{BaseURL: mockBrokerServer.URL}),
		workforceURL:  mockWorkforceServer.URL,
		coreInvestURL: mockCoreInvestServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/orders", bll.handleValidateAndExecuteOrder)

	// Case 1: Medical hold false -> Order executes successfully and persists to core-invest
	orderReq := TradeOrderRequest{
		PortfolioID: "port-001",
		FelineID:    "emp-feline-garfield",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    10.0,
		OrderType:   "market",
	}
	body, _ := json.Marshal(orderReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var res TradeOrderResult
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, "filled", res.Status)
	assert.Equal(t, "AAPL", res.Symbol)
	assert.Equal(t, 182.50, res.Price)
	assert.True(t, orderPersisted.Load(), "order must be persisted via core-invest-api")

	// Case 2: Medical hold active -> Order rejected
	medicalHoldStatus = true
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(body))
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusConflict, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "emergency medical hold active")
}

func TestCustomerBLL_OrderExecution_MedicalHoldFailClosed(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	// Workforce returns 500 error
	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockWorkforceServer.Close()

	bll := &CustomerBLLServer{
		publisher:       pubSub,
		httpClient:      mockWorkforceServer.Client(),
		workforceURL:    mockWorkforceServer.URL,
		defaultFelineID: "emp-feline-felix",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/orders", bll.handleValidateAndExecuteOrder)

	orderReq := TradeOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    5.0,
	}
	body, _ := json.Marshal(orderReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "feline medical hold safety check unavailable; failing closed")
}

func TestCustomerBLL_DefaultFeline_FromConfiguration(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	var queriedFelineID string
	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queriedFelineID = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"emergency_medical_hold": false,
		})
	}))
	defer mockWorkforceServer.Close()

	mockBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(alpaca.Order{ID: "ord-1", Status: "submitted"})
	}))
	defer mockBrokerServer.Close()

	// Setting default employee via configuration
	bll := &CustomerBLLServer{
		publisher:       pubSub,
		httpClient:      mockWorkforceServer.Client(),
		brokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: mockBrokerServer.URL}),
		workforceURL:    mockWorkforceServer.URL,
		defaultFelineID: "emp-configured-cat-42",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/orders", bll.handleValidateAndExecuteOrder)

	// Order request has NO FelineID
	orderReq := TradeOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "MSFT",
		Side:        "buy",
		Quantity:    5.0,
		OrderType:   "market",
	}
	body, _ := json.Marshal(orderReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, queriedFelineID, "emp-configured-cat-42")
}

func TestCustomerBLL_DefaultFeline_FromWorkforceDatabase(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	var queriedCareSchedulePath string
	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/workforce/employees" {
			// Emulate database returning active feline employee roster in {items, total} envelope (Thread 11)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"employee_id": "emp-db-feline-alpha", "first_name": "Barnaby", "employee_type": "feline", "status": "active"},
				},
				"total": 1,
			})
			return
		}
		queriedCareSchedulePath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"emergency_medical_hold": false,
		})
	}))
	defer mockWorkforceServer.Close()

	mockBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(alpaca.Order{ID: "ord-2", Status: "submitted"})
	}))
	defer mockBrokerServer.Close()

	// defaultFelineID is NOT configured, must query database data from workforce API
	bll := &CustomerBLLServer{
		publisher:    pubSub,
		httpClient:   mockWorkforceServer.Client(),
		brokerClient: alpaca.NewClient(alpaca.Config{BaseURL: mockBrokerServer.URL}),
		workforceURL: mockWorkforceServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/orders", bll.handleValidateAndExecuteOrder)

	// Order request has NO FelineID
	orderReq := TradeOrderRequest{
		PortfolioID: "port-002",
		Symbol:      "GOOGL",
		Side:        "buy",
		Quantity:    2.0,
		OrderType:   "market",
	}
	body, _ := json.Marshal(orderReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, queriedCareSchedulePath, "emp-db-feline-alpha")
}

func TestCustomerBLL_ACHDeposit_OneTimeAndRecurring(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	var recurringCreated atomic.Bool
	mockCoreInvestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/recurring-deposits") {
			recurringCreated.Store(true)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "created"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockCoreInvestServer.Close()

	bll := &CustomerBLLServer{
		publisher:     pubSub,
		httpClient:    mockCoreInvestServer.Client(),
		coreInvestURL: mockCoreInvestServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/deposits", bll.handleACHDeposit)

	// Case 1: One-time deposit
	depositReq := ACHDepositRequest{
		CustomerID:  "cust-123",
		PortfolioID: "port-123",
		BankAccount: "act-456",
		AmountUSD:   500.0,
		Frequency:   "one_time",
	}
	body, _ := json.Marshal(depositReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customer-bll/deposits", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.False(t, recurringCreated.Load())

	// Case 2: Recurring monthly deposit (Thread 33)
	depositReq.Frequency = "monthly"
	body2, _ := json.Marshal(depositReq)
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/customer-bll/deposits", bytes.NewReader(body2))
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusAccepted, rec2.Code)
	assert.True(t, recurringCreated.Load(), "recurring schedule should be persisted to core-invest-api")
}

func TestCustomerBLL_PortfolioMetrics_Validation(t *testing.T) {
	mockCoreInvestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/portfolios/port-existing") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"portfolio_id":     "port-existing",
				"customer_id":      "cust-test-01",
				"cash_balance":     2500.50,
				"invested_balance": 15000.0,
			})
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer mockCoreInvestServer.Close()

	bll := &CustomerBLLServer{
		httpClient:    mockCoreInvestServer.Client(),
		coreInvestURL: mockCoreInvestServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/customer-bll/portfolio/metrics", bll.handlePortfolioMetrics)

	// Case 1: Missing portfolio_id
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/customer-bll/portfolio/metrics", nil)
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Case 2: Nonexistent portfolio (Thread 34)
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/customer-bll/portfolio/metrics?portfolio_id=port-nonexistent", nil)
	mux.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusNotFound, rec2.Code)

	// Case 3: Existing portfolio -> metrics returned with actual balances
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/api/v1/customer-bll/portfolio/metrics?portfolio_id=port-existing", nil)
	mux.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
	var metrics map[string]interface{}
	err := json.Unmarshal(rec3.Body.Bytes(), &metrics)
	require.NoError(t, err)
	assert.Equal(t, "port-existing", metrics["portfolio_id"])
	assert.Equal(t, 2500.50, metrics["cash_balance"])
	assert.Equal(t, 17500.50, metrics["total_value"])
}

func TestCustomerBLL_GetStreams(t *testing.T) {
	// Case 1: core-invest returns active streams
	mockCoreInvestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/core-invest/streams" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"stream_id":  "stream-01",
					"stream_url": "webrtc://streams.oci.local/live/test-cam",
					"protocol":   "webrtc",
					"status":     "active",
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockCoreInvestServer.Close()

	bll := &CustomerBLLServer{
		httpClient:    mockCoreInvestServer.Client(),
		coreInvestURL: mockCoreInvestServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/customer-bll/streams", bll.handleGetStreams)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/customer-bll/streams", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(1), resp["total"])

	// Case 2: core-invest is unavailable
	bllEmpty := &CustomerBLLServer{
		httpClient:    http.DefaultClient,
		coreInvestURL: "http://127.0.0.1:54321", // non-existent
	}
	muxEmpty := http.NewServeMux()
	muxEmpty.HandleFunc("GET /api/v1/customer-bll/streams", bllEmpty.handleGetStreams)

	recEmpty := httptest.NewRecorder()
	reqEmpty := httptest.NewRequest("GET", "/api/v1/customer-bll/streams", nil)
	muxEmpty.ServeHTTP(recEmpty, reqEmpty)

	assert.Equal(t, http.StatusOK, recEmpty.Code)
	var respEmpty map[string]interface{}
	err = json.Unmarshal(recEmpty.Body.Bytes(), &respEmpty)
	require.NoError(t, err)
	assert.Equal(t, float64(0), respEmpty["total"])
}
