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
	listOrdersCalled        bool
	listBacktestsCalled     bool
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

func (m *mockProductionBLLClient) ListBrokerageOrders(ctx context.Context, in *coreinvestv1.ListOrdersRequest) (*coreinvestv1.ListOrdersResponse, error) {
	m.listOrdersCalled = true
	return &coreinvestv1.ListOrdersResponse{
		Orders: []*coreinvestv1.BrokerageOrder{
			{OrderID: "ord-1", Symbol: "AAPL", Side: "buy"},
		},
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

func TestEmployeeBFF_ProductionWiring(t *testing.T) {
	mockBLL := &mockProductionBLLClient{}
	bff := employeebff.NewEmployeeBFF(mockBLL, nil)
	router := bff.SetupRouter()

	// 1. Workforce Employees route
	reqEmp := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/employees", nil)
	recEmp := httptest.NewRecorder()
	router.ServeHTTP(recEmp, reqEmp)
	assert.Equal(t, http.StatusOK, recEmp.Code)
	assert.True(t, mockBLL.listEmployeesCalled, "expected ListEmployees RPC to be invoked")

	// 2. Workforce Leave Requests route
	reqLeave := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/leave-requests", nil)
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
	sessStore := bff.SessionStore()
	sess, err := sessStore.Create("emp-test-1", "testuser@example.com", "Test User", "engineering", "Engineering")
	require.NoError(t, err)

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
}
