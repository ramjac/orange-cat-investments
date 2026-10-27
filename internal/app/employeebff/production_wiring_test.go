package employeebff_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/app/employeebff"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProductionBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	listEmployeesCalled     bool
	listLeaveRequestsCalled bool
	listAssetsCalled        bool
	listTicketsCalled       bool
	listStreamsCalled       bool
	getStreamCalled         bool
	listOrdersCalled        bool
	getOrderCalled          bool
	listBacktestsCalled     bool
	getBacktestCalled       bool
}

func (m *mockProductionBLLClient) ListEmployees(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	m.listEmployeesCalled = true
	return &workforcev1.ListEmployeesResponse{
		Employees: []*workforcev1.Employee{
			{
				EmployeeID: "emp-test-1",
				FirstName:  "Test",
				LastName:   "Employee",
				RoleTitle:  "Engineer",
				Department: "engineering",
			},
		},
	}, nil
}

func (m *mockProductionBLLClient) ListLeaveRequests(ctx context.Context, in *workforcev1.ListLeaveRequestsRequest) (*workforcev1.ListLeaveRequestsResponse, error) {
	m.listLeaveRequestsCalled = true
	return &workforcev1.ListLeaveRequestsResponse{
		LeaveRequests: []*workforcev1.LeaveRequest{
			{LeaveID: "leave-1", EmployeeID: "emp-test-1", Status: "approved"},
		},
	}, nil
}

func (m *mockProductionBLLClient) ListAssets(ctx context.Context, in *facilitiesv1.ListAssetsRequest) (*facilitiesv1.ListAssetsResponse, error) {
	m.listAssetsCalled = true
	return &facilitiesv1.ListAssetsResponse{
		Assets: []*facilitiesv1.HardwareAsset{
			{AssetID: "asset-1", SerialNumber: "SN1234", AssetType: "dispenser", Model: "Feeder Pro"},
		},
	}, nil
}

func (m *mockProductionBLLClient) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	m.listTicketsCalled = true
	return &opsv1.ListTicketsResponse{
		Tickets: []*opsv1.ITTicket{
			{TicketID: "ticket-1", Title: "Network Issue", State: "open"},
		},
	}, nil
}

func (m *mockProductionBLLClient) ListStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	m.listStreamsCalled = true
	return &coreinvestv1.ListCameraStreamsResponse{
		Streams: []*coreinvestv1.CameraStream{
			{StreamID: "stream-1", AssetID: "asset-1", Protocol: "rtsp"},
		},
	}, nil
}

func (m *mockProductionBLLClient) GetCameraStream(ctx context.Context, in *coreinvestv1.GetCameraStreamRequest) (*coreinvestv1.GetCameraStreamResponse, error) {
	m.getStreamCalled = true
	return &coreinvestv1.GetCameraStreamResponse{
		Stream: &coreinvestv1.CameraStream{StreamID: in.StreamID, AssetID: "asset-1"},
	}, nil
}

func (m *mockProductionBLLClient) ListBrokerageOrders(ctx context.Context, in *coreinvestv1.ListOrdersRequest) (*coreinvestv1.ListOrdersResponse, error) {
	m.listOrdersCalled = true
	return &coreinvestv1.ListOrdersResponse{
		Orders: []*coreinvestv1.BrokerageOrder{
			{OrderID: "ord-1", Symbol: "AAPL", Side: "buy"},
		},
	}, nil
}

func (m *mockProductionBLLClient) GetBrokerageOrder(ctx context.Context, in *coreinvestv1.GetOrderRequest) (*coreinvestv1.GetOrderResponse, error) {
	m.getOrderCalled = true
	return &coreinvestv1.GetOrderResponse{
		Order: &coreinvestv1.BrokerageOrder{OrderID: in.OrderID, Symbol: "AAPL"},
	}, nil
}

func (m *mockProductionBLLClient) ListBacktestRuns(ctx context.Context, in *coreinvestv1.ListBacktestRunsRequest) (*coreinvestv1.ListBacktestRunsResponse, error) {
	m.listBacktestsCalled = true
	return &coreinvestv1.ListBacktestRunsResponse{
		Runs: []*coreinvestv1.BacktestRun{
			{RunID: "run-1", StrategyName: "alpha-1"},
		},
	}, nil
}

func (m *mockProductionBLLClient) GetBacktestRun(ctx context.Context, in *coreinvestv1.GetBacktestRunRequest) (*coreinvestv1.GetBacktestRunResponse, error) {
	m.getBacktestCalled = true
	return &coreinvestv1.GetBacktestRunResponse{
		Run: &coreinvestv1.BacktestRun{RunID: in.RunID, StrategyName: "alpha-1"},
	}, nil
}

func TestEmployeeBFF_ProductionWiring(t *testing.T) {
	mockBLL := &mockProductionBLLClient{}
	bff := employeebff.NewEmployeeBFF(mockBLL, nil)
	router := bff.SetupRouter()

	sessStore := bff.SessionStore()
	sess, err := sessStore.Create("emp-test-1", "testuser@example.com", "Test User", "engineering", "Engineering")
	require.NoError(t, err)

	// 1. Workforce Employees route
	reqEmp := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/employees", nil)
	reqEmp.AddCookie(&http.Cookie{
		Name:  "__Host-employee-session",
		Value: sess.SessionID,
	})
	recEmp := httptest.NewRecorder()
	router.ServeHTTP(recEmp, reqEmp)
	assert.Equal(t, http.StatusOK, recEmp.Code)
	assert.True(t, mockBLL.listEmployeesCalled, "expected ListEmployees RPC to be invoked")

	// 2. Workforce Leave Requests route
	reqLeave := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/leave-requests", nil)
	reqLeave.AddCookie(&http.Cookie{
		Name:  "__Host-employee-session",
		Value: sess.SessionID,
	})
	recLeave := httptest.NewRecorder()
	router.ServeHTTP(recLeave, reqLeave)
	assert.Equal(t, http.StatusOK, recLeave.Code)
	assert.True(t, mockBLL.listLeaveRequestsCalled, "expected ListLeaveRequests RPC to be invoked")

	// 3. Facilities Assets route
	reqAssets := httptest.NewRequest(http.MethodGet, "/api/v1/facilities/assets", nil)
	recAssets := httptest.NewRecorder()
	router.ServeHTTP(recAssets, reqAssets)
	assert.Equal(t, http.StatusOK, recAssets.Code)
	assert.True(t, mockBLL.listAssetsCalled, "expected ListAssets RPC to be invoked")

	// 4. Operations Tickets route (requires authenticated employee session)
	reqTickets := httptest.NewRequest(http.MethodGet, "/api/v1/ops/tickets", nil)
	reqTickets.AddCookie(&http.Cookie{
		Name:  "__Host-employee-session",
		Value: sess.SessionID,
	})
	recTickets := httptest.NewRecorder()
	router.ServeHTTP(recTickets, reqTickets)
	assert.Equal(t, http.StatusOK, recTickets.Code)
	assert.True(t, mockBLL.listTicketsCalled, "expected ListTickets RPC to be invoked")

	// 5. Quant Streams route
	reqStreams := httptest.NewRequest(http.MethodGet, "/core-invest/streams", nil)
	recStreams := httptest.NewRecorder()
	router.ServeHTTP(recStreams, reqStreams)
	assert.Equal(t, http.StatusOK, recStreams.Code)
	assert.True(t, mockBLL.listStreamsCalled, "expected ListStreams RPC to be invoked")

	// 6. Quant Orders route
	reqOrders := httptest.NewRequest(http.MethodGet, "/core-invest/brokerage/orders", nil)
	recOrders := httptest.NewRecorder()
	router.ServeHTTP(recOrders, reqOrders)
	assert.Equal(t, http.StatusOK, recOrders.Code)
	assert.True(t, mockBLL.listOrdersCalled, "expected ListBrokerageOrders RPC to be invoked")

	// 7. Quant Backtest Runs route
	reqBacktests := httptest.NewRequest(http.MethodGet, "/core-invest/backtesting/runs", nil)
	recBacktests := httptest.NewRecorder()
	router.ServeHTTP(recBacktests, reqBacktests)
	assert.Equal(t, http.StatusOK, recBacktests.Code)
	assert.True(t, mockBLL.listBacktestsCalled, "expected ListBacktestRuns RPC to be invoked")

	// 8. Quant Stream detail route
	reqStreamDetail := httptest.NewRequest(http.MethodGet, "/core-invest/streams/stream-1", nil)
	recStreamDetail := httptest.NewRecorder()
	router.ServeHTTP(recStreamDetail, reqStreamDetail)
	assert.Equal(t, http.StatusOK, recStreamDetail.Code)
	assert.True(t, mockBLL.getStreamCalled, "expected GetCameraStream RPC to be invoked")

	// 9. Quant Order detail route
	reqOrderDetail := httptest.NewRequest(http.MethodGet, "/core-invest/brokerage/orders/ord-1", nil)
	recOrderDetail := httptest.NewRecorder()
	router.ServeHTTP(recOrderDetail, reqOrderDetail)
	assert.Equal(t, http.StatusOK, recOrderDetail.Code)
	assert.True(t, mockBLL.getOrderCalled, "expected GetBrokerageOrder RPC to be invoked")

	// 10. Quant Backtest Run detail route
	reqBacktestDetail := httptest.NewRequest(http.MethodGet, "/core-invest/backtesting/runs/run-1", nil)
	recBacktestDetail := httptest.NewRecorder()
	router.ServeHTTP(recBacktestDetail, reqBacktestDetail)
	assert.Equal(t, http.StatusOK, recBacktestDetail.Code)
	assert.True(t, mockBLL.getBacktestCalled, "expected GetBacktestRun RPC to be invoked")
}
