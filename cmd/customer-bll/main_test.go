package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
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

func TestCustomerBLL_OrderExecution_NormalAndMedicalHold(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	// Mock workforce service to test medical hold check
	medicalHoldStatus := false
	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"feline_id":              "emp-feline-garfield",
			"emergency_medical_hold": medicalHoldStatus,
		})
	}))
	defer mockWorkforceServer.Close()

	bll := &CustomerBLLServer{
		publisher:    pubSub,
		httpClient:   mockWorkforceServer.Client(),
		workforceURL: mockWorkforceServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/orders", bll.handleValidateAndExecuteOrder)

	// Case 1: Medical hold false -> Order executes successfully
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
	assert.Equal(t, "executed", res.Status)
	assert.Equal(t, "AAPL", res.Symbol)

	// Case 2: Medical hold active -> Order rejected
	medicalHoldStatus = true
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/customer-bll/orders", bytes.NewReader(body))
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusConflict, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "emergency medical hold active")
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

	// Setting default employee via configuration
	bll := &CustomerBLLServer{
		publisher:       pubSub,
		httpClient:      mockWorkforceServer.Client(),
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
			// Emulate database returning active feline employee roster
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"employee_id": "emp-db-feline-alpha", "first_name": "Barnaby", "employee_type": "feline", "status": "active"},
			})
			return
		}
		queriedCareSchedulePath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"emergency_medical_hold": false,
		})
	}))
	defer mockWorkforceServer.Close()

	// defaultFelineID is NOT configured, must query database data from workforce API
	bll := &CustomerBLLServer{
		publisher:    pubSub,
		httpClient:   mockWorkforceServer.Client(),
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


func TestCustomerBLL_ACHDeposit(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	bll := &CustomerBLLServer{
		publisher:  pubSub,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/customer-bll/deposits", bll.handleACHDeposit)

	depositReq := ACHDepositRequest{
		CustomerID:  "cust-123",
		PortfolioID: "port-123",
		BankAccount: "act-456",
		AmountUSD:   500.0,
		Frequency:   "monthly",
	}
	body, _ := json.Marshal(depositReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customer-bll/deposits", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var res ACHDepositResult
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, "pending_settlement", res.Status)
	assert.Equal(t, 500.0, res.AmountUSD)
}
