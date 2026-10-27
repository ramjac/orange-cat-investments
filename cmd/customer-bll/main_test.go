package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/internal/app/customerbll"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type stubWorkforceServer struct {
	workforcev1.UnimplementedWorkforceServiceServer
	medicalHold     bool
	failCareSched   bool
	queriedFelineID string
	employees       []*workforcev1.Employee
}

func (s *stubWorkforceServer) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	s.queriedFelineID = in.FelineID
	if s.failCareSched {
		return nil, errors.New("workforce care schedule database failure")
	}
	return &workforcev1.GetCareScheduleResponse{
		CareSchedule: &workforcev1.CareSchedule{
			FelineID:             in.FelineID,
			EmergencyMedicalHold: s.medicalHold,
		},
	}, nil
}

func (s *stubWorkforceServer) ListEmployees(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	return &workforcev1.ListEmployeesResponse{
		Employees: s.employees,
	}, nil
}

type stubCoreInvestServer struct {
	coreinvestv1.UnimplementedCoreInvestServiceServer
	orderPersisted   atomic.Bool
	recurringCreated atomic.Bool
	portfolios       map[string]*coreinvestv1.Portfolio
	streams          []*coreinvestv1.CameraStream
	failStreams      bool
}

func (s *stubCoreInvestServer) CreateOrder(ctx context.Context, in *coreinvestv1.CreateOrderRequest) (*coreinvestv1.CreateOrderResponse, error) {
	s.orderPersisted.Store(true)
	return &coreinvestv1.CreateOrderResponse{
		Order: &coreinvestv1.BrokerageOrder{
			OrderID:     "ord-persisted-1",
			PortfolioID: in.PortfolioID,
			Symbol:      in.Symbol,
			Side:        in.Side,
			Quantity:    in.Quantity,
			Price:       in.Price,
			Status:      "filled",
		},
	}, nil
}

func (s *stubCoreInvestServer) CreateRecurringDeposit(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error) {
	if in.Frequency != "one_time" {
		s.recurringCreated.Store(true)
	}
	return &coreinvestv1.CreateRecurringDepositResponse{
		Deposit: &coreinvestv1.RecurringDeposit{
			DepositID:   "dep-01",
			CustomerID:  in.CustomerID,
			PortfolioID: in.PortfolioID,
			AmountUSD:   in.AmountUSD,
			Frequency:   in.Frequency,
			Status:      "active",
		},
	}, nil
}

func (s *stubCoreInvestServer) GetPortfolio(ctx context.Context, in *coreinvestv1.GetPortfolioRequest) (*coreinvestv1.GetPortfolioResponse, error) {
	if s.portfolios != nil {
		if p, ok := s.portfolios[in.PortfolioID]; ok {
			return &coreinvestv1.GetPortfolioResponse{Portfolio: p}, nil
		}
	}
	return nil, status.Error(codes.NotFound, "portfolio not found")
}

func (s *stubCoreInvestServer) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	if s.failStreams {
		return nil, status.Error(codes.Unavailable, "core-invest unavailable")
	}
	return &coreinvestv1.ListCameraStreamsResponse{Streams: s.streams}, nil
}

func (s *stubCoreInvestServer) StreamPortfolioValuation(ctx context.Context, in *coreinvestv1.StreamPortfolioValuationRequest) (coreinvestv1.CoreInvestService_StreamPortfolioValuationClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func startCustomerBLLTestServer(t *testing.T, srv *customerbll.Server) customerbllv1.CustomerBLLServiceClient {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	grpcSrv := grpcutil.NewServer(nil)
	customerbllv1.RegisterCustomerBLLServiceServer(grpcSrv, srv)
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(func() { grpcSrv.Stop() })

	conn, err := grpcutil.Dial(t.Context(), lis.Addr().String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return customerbllv1.NewCustomerBLLServiceClient(conn)
}

func TestCustomerBLL_GRPCHealth(t *testing.T) {
	srv := customerbll.NewServer(customerbll.Config{})
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	grpcSrv := grpcutil.NewServer(nil)
	customerbllv1.RegisterCustomerBLLServiceServer(grpcSrv, srv)
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	go func() { _ = grpcSrv.Serve(lis) }()
	defer grpcSrv.Stop()

	conn, err := grpcutil.Dial(t.Context(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	hc := grpc_health_v1.NewHealthClient(conn)
	res, err := hc.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, res.Status)
}

func TestCustomerBLL_OrderExecution_Validations(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	bll := customerbll.NewServer(customerbll.Config{Publisher: pubSub})
	client := startCustomerBLLTestServer(t, bll)

	// 1. Missing fields
	_, err := client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{PortfolioID: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// 2. Invalid side
	_, err = client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "hold",
		Quantity:    10,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "side must be 'buy' or 'sell'")

	// 3. Unsupported order type
	_, err = client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    10,
		OrderType:   "bracket",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported order type")

	// 4. Limit order with price <= 0
	_, err = client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    10,
		OrderType:   "limit",
		LimitPrice:  0,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit order requires limit_price > 0")
}

func TestCustomerBLL_OrderExecution_NormalAndMedicalHold(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	wfStub := &stubWorkforceServer{medicalHold: false}
	ciStub := &stubCoreInvestServer{}

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

	bll := customerbll.NewServer(customerbll.Config{
		Publisher:    pubSub,
		BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: mockBrokerServer.URL}),
	})
	bll.SetWorkforceClient(wfStub)
	bll.SetCoreInvestClient(ciStub)

	client := startCustomerBLLTestServer(t, bll)

	// Case 1: Medical hold false -> Order executes successfully and persists to core-invest
	orderReq := &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-001",
		FelineID:    "emp-feline-garfield",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    10.0,
		OrderType:   "market",
	}
	res, err := client.ValidateAndExecuteOrder(t.Context(), orderReq)
	require.NoError(t, err)
	assert.Equal(t, "filled", res.Status)
	assert.Equal(t, "AAPL", res.Symbol)
	assert.Equal(t, 182.50, res.Price)
	assert.True(t, ciStub.orderPersisted.Load(), "order must be persisted via core-invest-api")

	// Case 2: Medical hold active -> Order rejected
	wfStub.medicalHold = true
	_, err = client.ValidateAndExecuteOrder(t.Context(), orderReq)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), "emergency medical hold active")
}

func TestCustomerBLL_OrderExecution_MedicalHoldFailClosed(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	wfStub := &stubWorkforceServer{failCareSched: true}
	bll := customerbll.NewServer(customerbll.Config{
		Publisher:       pubSub,
		DefaultFelineID: "emp-feline-felix",
	})
	bll.SetWorkforceClient(wfStub)

	client := startCustomerBLLTestServer(t, bll)

	_, err := client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "AAPL",
		Side:        "buy",
		Quantity:    5.0,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, err.Error(), "feline medical hold safety check unavailable; failing closed")
}

func TestCustomerBLL_DefaultFeline_FromConfiguration(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	wfStub := &stubWorkforceServer{medicalHold: false}
	mockBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(alpaca.Order{ID: "ord-1", Status: "submitted"})
	}))
	defer mockBrokerServer.Close()

	bll := customerbll.NewServer(customerbll.Config{
		Publisher:       pubSub,
		BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: mockBrokerServer.URL}),
		DefaultFelineID: "emp-configured-cat-42",
	})
	bll.SetWorkforceClient(wfStub)

	client := startCustomerBLLTestServer(t, bll)

	res, err := client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-001",
		Symbol:      "MSFT",
		Side:        "buy",
		Quantity:    5.0,
		OrderType:   "market",
	})
	require.NoError(t, err)
	assert.Equal(t, "submitted", res.Status)
	assert.Equal(t, "emp-configured-cat-42", wfStub.queriedFelineID)
}

func TestCustomerBLL_DefaultFeline_FromWorkforceDatabase(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	wfStub := &stubWorkforceServer{
		medicalHold: false,
		employees: []*workforcev1.Employee{
			{EmployeeID: "emp-db-feline-alpha", FirstName: "Barnaby", EmployeeType: "feline", Status: "active"},
		},
	}

	mockBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(alpaca.Order{ID: "ord-2", Status: "submitted"})
	}))
	defer mockBrokerServer.Close()

	bll := customerbll.NewServer(customerbll.Config{
		Publisher:    pubSub,
		BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: mockBrokerServer.URL}),
	})
	bll.SetWorkforceClient(wfStub)

	client := startCustomerBLLTestServer(t, bll)

	res, err := client.ValidateAndExecuteOrder(t.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		PortfolioID: "port-002",
		Symbol:      "GOOGL",
		Side:        "buy",
		Quantity:    2.0,
		OrderType:   "market",
	})
	require.NoError(t, err)
	assert.Equal(t, "submitted", res.Status)
	assert.Equal(t, "emp-db-feline-alpha", wfStub.queriedFelineID)
}

func TestCustomerBLL_ACHDeposit_OneTimeAndRecurring(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	ciStub := &stubCoreInvestServer{}
	bll := customerbll.NewServer(customerbll.Config{Publisher: pubSub})
	bll.SetCoreInvestClient(ciStub)

	client := startCustomerBLLTestServer(t, bll)

	// Case 1: One-time deposit
	res, err := client.InitiateACHDeposit(t.Context(), &customerbllv1.InitiateACHDepositRequest{
		CustomerID:  "cust-123",
		PortfolioID: "port-123",
		BankAccount: "act-456",
		AmountUSD:   500.0,
		Frequency:   "one_time",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending_settlement", res.Status)
	assert.False(t, ciStub.recurringCreated.Load())

	// Case 2: Recurring monthly deposit
	res2, err := client.InitiateACHDeposit(t.Context(), &customerbllv1.InitiateACHDepositRequest{
		CustomerID:  "cust-123",
		PortfolioID: "port-123",
		BankAccount: "act-456",
		AmountUSD:   500.0,
		Frequency:   "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending_settlement", res2.Status)
	assert.True(t, ciStub.recurringCreated.Load(), "recurring schedule should be persisted to core-invest-api")
}

func TestCustomerBLL_PortfolioMetrics_Validation(t *testing.T) {
	ciStub := &stubCoreInvestServer{
		portfolios: map[string]*coreinvestv1.Portfolio{
			"port-existing": {
				PortfolioID:     "port-existing",
				CustomerID:      "cust-test-01",
				CashBalanceUSD:  2500.50,
				TotalBalanceUSD: 17500.50,
			},
		},
	}

	bll := customerbll.NewServer(customerbll.Config{})
	bll.SetCoreInvestClient(ciStub)
	client := startCustomerBLLTestServer(t, bll)

	// Case 1: Missing portfolio_id
	_, err := client.GetPortfolioMetrics(t.Context(), &customerbllv1.GetPortfolioMetricsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// Case 2: Nonexistent portfolio
	_, err = client.GetPortfolioMetrics(t.Context(), &customerbllv1.GetPortfolioMetricsRequest{PortfolioID: "port-nonexistent"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))

	// Case 3: Existing portfolio -> metrics returned with actual balances
	metrics, err := client.GetPortfolioMetrics(t.Context(), &customerbllv1.GetPortfolioMetricsRequest{PortfolioID: "port-existing"})
	require.NoError(t, err)
	assert.Equal(t, "port-existing", metrics.PortfolioID)
	assert.Equal(t, 2500.50, metrics.CashBalance)
	assert.Equal(t, 17500.50, metrics.TotalValue)
}

func TestCustomerBLL_GetStreams(t *testing.T) {
	ciStub := &stubCoreInvestServer{
		streams: []*coreinvestv1.CameraStream{
			{
				StreamID:  "stream-01",
				StreamURL: "webrtc://streams.oci.local/live/test-cam",
				Protocol:  "webrtc",
				Status:    "active",
			},
		},
	}

	bll := customerbll.NewServer(customerbll.Config{})
	bll.SetCoreInvestClient(ciStub)
	client := startCustomerBLLTestServer(t, bll)

	resp, err := client.ListCameraStreams(t.Context(), &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Streams, 1)

	// Case 2: core-invest is unavailable -> returns empty slice gracefully
	ciStub.failStreams = true
	respEmpty, err := client.ListCameraStreams(t.Context(), &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	assert.Empty(t, respEmpty.Streams)
}
