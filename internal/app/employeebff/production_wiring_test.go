package employeebff_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/app/employeebff"
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
}
