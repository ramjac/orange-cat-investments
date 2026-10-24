package workforcev1

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWorkforceServer struct {
	employees map[string]*Employee
}

func (m *mockWorkforceServer) GetEmployee(ctx context.Context, in *GetEmployeeRequest) (*GetEmployeeResponse, error) {
	return &GetEmployeeResponse{Employee: m.employees[in.EmployeeID]}, nil
}

func (m *mockWorkforceServer) ListEmployees(ctx context.Context, in *ListEmployeesRequest) (*ListEmployeesResponse, error) {
	var list []*Employee
	for _, e := range m.employees {
		if in.EmployeeType == "" || e.EmployeeType == in.EmployeeType {
			list = append(list, e)
		}
	}
	return &ListEmployeesResponse{Employees: list}, nil
}

func (m *mockWorkforceServer) CreateEmployee(ctx context.Context, in *CreateEmployeeRequest) (*CreateEmployeeResponse, error) {
	emp := &Employee{
		EmployeeID:   "emp-test-01",
		EmployeeType: in.EmployeeType,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Email:        in.Email,
		RoleTitle:    in.RoleTitle,
		Department:   in.Department,
	}
	m.employees[emp.EmployeeID] = emp
	return &CreateEmployeeResponse{Employee: emp}, nil
}

func (m *mockWorkforceServer) GetCareSchedule(ctx context.Context, in *GetCareScheduleRequest) (*GetCareScheduleResponse, error) {
	return &GetCareScheduleResponse{CareSchedule: &CareSchedule{FelineID: in.FelineID, DietaryPlan: "Salmon"}}, nil
}
func (m *mockWorkforceServer) UpdateCareSchedule(ctx context.Context, in *UpdateCareScheduleRequest) (*UpdateCareScheduleResponse, error) {
	return &UpdateCareScheduleResponse{CareSchedule: &CareSchedule{FelineID: in.FelineID, DietaryPlan: in.DietaryPlan}}, nil
}
func (m *mockWorkforceServer) SetMedicalHold(ctx context.Context, in *SetMedicalHoldRequest) (*SetMedicalHoldResponse, error) {
	return &SetMedicalHoldResponse{CareSchedule: &CareSchedule{FelineID: in.FelineID, EmergencyMedicalHold: in.EmergencyMedicalHold}}, nil
}
func (m *mockWorkforceServer) SubmitLeaveRequest(ctx context.Context, in *SubmitLeaveRequestMessage) (*SubmitLeaveResponse, error) {
	return &SubmitLeaveResponse{LeaveRequest: &LeaveRequest{EmployeeID: in.EmployeeID, LeaveType: in.LeaveType}}, nil
}
func (m *mockWorkforceServer) ListLeaveRequests(ctx context.Context, in *ListLeaveRequestsRequest) (*ListLeaveRequestsResponse, error) {
	return &ListLeaveRequestsResponse{}, nil
}
func (m *mockWorkforceServer) UpdateLeaveStatus(ctx context.Context, in *UpdateLeaveStatusRequest) (*UpdateLeaveStatusResponse, error) {
	return &UpdateLeaveStatusResponse{LeaveRequest: &LeaveRequest{Status: in.Status}}, nil
}
func (m *mockWorkforceServer) CreateReviewCycle(ctx context.Context, in *CreateReviewCycleRequest) (*CreateReviewCycleResponse, error) {
	return &CreateReviewCycleResponse{ReviewCycle: &ReviewCycle{EmployeeID: in.EmployeeID, ReviewPeriod: in.ReviewPeriod}}, nil
}
func (m *mockWorkforceServer) ListReviewCycles(ctx context.Context, in *ListReviewCyclesRequest) (*ListReviewCyclesResponse, error) {
	return &ListReviewCyclesResponse{}, nil
}
func (m *mockWorkforceServer) CreateVHRRecord(ctx context.Context, in *CreateVHRRecordRequest) (*CreateVHRRecordResponse, error) {
	return &CreateVHRRecordResponse{Record: &VHRRecord{FelineID: in.FelineID, ClinicalNotes: in.ClinicalNotes}}, nil
}
func (m *mockWorkforceServer) ListVHRRecords(ctx context.Context, in *ListVHRRecordsRequest) (*ListVHRRecordsResponse, error) {
	return &ListVHRRecordsResponse{}, nil
}
func (m *mockWorkforceServer) CreateIncident(ctx context.Context, in *CreateIncidentRequest) (*CreateIncidentResponse, error) {
	return &CreateIncidentResponse{Incident: &WorkplaceIncident{Title: in.Title, Severity: in.Severity}}, nil
}
func (m *mockWorkforceServer) ListIncidents(ctx context.Context, in *ListIncidentsRequest) (*ListIncidentsResponse, error) {
	return &ListIncidentsResponse{}, nil
}

func TestWorkforceGRPC_RoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpcutil.NewServer(nil)
	RegisterWorkforceServiceServer(srv, &mockWorkforceServer{
		employees: map[string]*Employee{
			"emp-01": {EmployeeID: "emp-01", FirstName: "Marcus", RoleTitle: "Lead Engineer"},
		},
	})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	client := NewWorkforceServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.GetEmployee(ctx, &GetEmployeeRequest{EmployeeID: "emp-01"})
	require.NoError(t, err)
	require.NotNil(t, resp.Employee)
	assert.Equal(t, "Marcus", resp.Employee.FirstName)

	sched, err := client.GetCareSchedule(ctx, &GetCareScheduleRequest{FelineID: "fel-01"})
	require.NoError(t, err)
	assert.Equal(t, "Salmon", sched.CareSchedule.DietaryPlan)
}
