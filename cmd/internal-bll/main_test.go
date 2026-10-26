package main

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/orange-cat-investments/oci/internal/app/internalbll"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
)

type mockPub struct {
	mu     sync.Mutex
	topics []string
}

func (m *mockPub) Publish(topic string, _ any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.topics = append(m.topics, topic)
	return nil
}

type mockWFClient struct {
	workforcev1.WorkforceServiceClient
}

func (m *mockWFClient) CreateIncident(_ context.Context, in *workforcev1.CreateIncidentRequest) (*workforcev1.CreateIncidentResponse, error) {
	return &workforcev1.CreateIncidentResponse{
		Incident: &workforcev1.WorkplaceIncident{
			IncidentID:  "inc-test-001",
			Title:       in.Title,
			Category:    in.Category,
			Severity:    in.Severity,
			Status:      "open",
			Description: in.Description,
		},
	}, nil
}

func (m *mockWFClient) OnboardEmployee(_ context.Context, in *workforcev1.OnboardEmployeeRequest) (*workforcev1.OnboardEmployeeResponse, error) {
	return &workforcev1.OnboardEmployeeResponse{
		Employee: &workforcev1.Employee{
			EmployeeID:   "emp-test-001",
			EmployeeType: in.Employee.EmployeeType,
			FirstName:    in.Employee.FirstName,
			RoleTitle:    in.Employee.RoleTitle,
			Department:   in.Employee.Department,
			Status:       "onboarding",
		},
	}, nil
}

func (m *mockWFClient) UpdateEmployeeStatus(_ context.Context, in *workforcev1.UpdateEmployeeStatusRequest) (*workforcev1.UpdateEmployeeStatusResponse, error) {
	return &workforcev1.UpdateEmployeeStatusResponse{
		Employee: &workforcev1.Employee{
			EmployeeID: in.EmployeeID,
			Status:     in.Status,
		},
	}, nil
}

func (m *mockWFClient) ListOnboardingChecklist(_ context.Context, _ *workforcev1.ListOnboardingChecklistRequest) (*workforcev1.ListOnboardingChecklistResponse, error) {
	return &workforcev1.ListOnboardingChecklistResponse{}, nil
}

func (m *mockWFClient) GetCareSchedule(_ context.Context, _ *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	return &workforcev1.GetCareScheduleResponse{}, nil
}

func (m *mockWFClient) GetEmployee(_ context.Context, in *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	return &workforcev1.GetEmployeeResponse{
		Employee: &workforcev1.Employee{
			EmployeeID: in.EmployeeID,
			FirstName:  "Alice",
			Status:     "onboarding",
		},
	}, nil
}

type mockFacClient struct {
	facilitiesv1.FacilitiesServiceClient
}

func (m *mockFacClient) CreateOTAJob(_ context.Context, in *facilitiesv1.CreateOTAJobRequest) (*facilitiesv1.CreateOTAJobResponse, error) {
	var jobs []*facilitiesv1.DeviceOTAJob
	for _, aid := range in.AssetIDs {
		jobs = append(jobs, &facilitiesv1.DeviceOTAJob{
			JobID:     "ota-job-001",
			AssetID:   aid,
			ReleaseID: in.ReleaseID,
			Status:    "queued",
		})
	}
	return &facilitiesv1.CreateOTAJobResponse{Jobs: jobs}, nil
}

type mockOpsClient struct {
	opsv1.OpsServiceClient
}

func (m *mockOpsClient) CreateTicket(_ context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	return &opsv1.CreateTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID: "tkt-pebble-001",
			Title:    in.Title,
			Severity: in.Severity,
			IsUrgent: in.IsUrgent,
		},
	}, nil
}

func TestInternalBLLCreateIncidentAndOTA(t *testing.T) {
	pub := &mockPub{}
	srv := internalbll.NewServer(&mockWFClient{}, &mockFacClient{}, &mockOpsClient{}, pub, nil)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpcutil.NewServer(nil)
	internalbllv1.RegisterInternalBLLServiceServer(grpcSrv, srv)
	go func() { _ = grpcSrv.Serve(lis) }()
	defer grpcSrv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := internalbllv1.NewInternalBLLServiceClient(conn)

	// 1. Create incident & verify event publication
	incResp, err := client.CreateIncident(context.Background(), &workforcev1.CreateIncidentRequest{
		Title:       "Unauthorized Perch",
		Category:    "unauthorized_perch",
		Severity:    "high",
		Description: "Biscuit climbed server rack",
	})
	if err != nil {
		t.Fatalf("CreateIncident: %v", err)
	}
	if incResp.Incident == nil || incResp.Incident.IncidentID != "inc-test-001" {
		t.Fatalf("unexpected incident response: %+v", incResp)
	}

	// 2. Trigger OTA rollout & verify event publication
	otaResp, err := client.TriggerOTARollout(context.Background(), &facilitiesv1.CreateOTAJobRequest{
		AssetIDs:  []string{"asset-001", "asset-002"},
		ReleaseID: "rel-v2.1",
	})
	if err != nil {
		t.Fatalf("TriggerOTARollout: %v", err)
	}
	if len(otaResp.Jobs) != 2 {
		t.Fatalf("expected 2 OTA jobs, got %d", len(otaResp.Jobs))
	}

	// 3. Onboard employee via saga & verify event publication
	onboardResp, err := client.OnboardEmployee(context.Background(), &internalbllv1.OnboardEmployeeSagaRequest{
		EmployeeType: "human",
		FirstName:    "Alice",
		RoleTitle:    "Engineer",
		Department:   "IT",
	})
	if err != nil {
		t.Fatalf("OnboardEmployee: %v", err)
	}
	if onboardResp.Employee == nil || onboardResp.Employee.EmployeeID != "emp-test-001" {
		t.Fatalf("unexpected onboard response: %+v", onboardResp)
	}

	// 4. Broadcast Pebble alert via Ops
	pebbleResp, err := client.BroadcastPebbleIncidentAlert(context.Background(), &internalbllv1.BroadcastPebbleIncidentAlertRequest{
		IncidentID:  "inc-test-001",
		Title:       "Unauthorized Perch",
		Category:    "unauthorized_perch",
		Severity:    "high",
		Description: "Biscuit climbed server rack",
	})
	if err != nil {
		t.Fatalf("BroadcastPebbleIncidentAlert: %v", err)
	}
	if pebbleResp.TicketID != "tkt-pebble-001" {
		t.Fatalf("unexpected pebble ticket_id: %s", pebbleResp.TicketID)
	}

	pub.mu.Lock()
	topicCount := len(pub.topics)
	pub.mu.Unlock()
	if topicCount != 4 {
		t.Fatalf("expected 4 published events (1 incident + 2 OTA + 1 onboard), got %d", topicCount)
	}
}
