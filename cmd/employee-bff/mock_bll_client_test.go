package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

type mockInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	mu            sync.Mutex
	careSchedules map[string]*workforcev1.CareSchedule
	folders       []nextcloud.GroupFolder
	tickets       map[string]*opsv1.ITTicket
	alerts        []*opsv1.PebbleAlert
}

func newMockInternalBLLClient() *mockInternalBLLClient {
	return &mockInternalBLLClient{
		careSchedules: make(map[string]*workforcev1.CareSchedule),
		tickets:       make(map[string]*opsv1.ITTicket),
		folders: []nextcloud.GroupFolder{
			{ID: 1, MountPoint: "Executive Board Documents", Quota: "100 GB", Groups: []string{"exec", "admin"}},
			{ID: 2, MountPoint: "Facilities & Habitats", Quota: "50 GB", Groups: []string{"facilities"}},
		},
		alerts: []*opsv1.PebbleAlert{
			{
				AlertID:  "ticket-test-123",
				TicketID: "ticket-test-123",
				Title:    "Pod OOMKilled Warning",
				Severity: "HIGH",
				Body:     "Memory usage exceeded threshold",
				Author:   "emp-human-frank",
			},
		},
	}
}

func (m *mockInternalBLLClient) GetEmployee(ctx context.Context, in *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	return &workforcev1.GetEmployeeResponse{
		Employee: &workforcev1.Employee{
			EmployeeID: in.EmployeeID,
			FirstName:  in.EmployeeID,
			Status:     "active",
		},
	}, nil
}

func (m *mockInternalBLLClient) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs, ok := m.careSchedules[in.FelineID]
	if !ok {
		return nil, fmt.Errorf("care schedule for %s not found", in.FelineID)
	}
	return &workforcev1.GetCareScheduleResponse{CareSchedule: cs}, nil
}

func (m *mockInternalBLLClient) UpdateCareSchedule(ctx context.Context, in *workforcev1.UpdateCareScheduleRequest) (*workforcev1.UpdateCareScheduleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs := &workforcev1.CareSchedule{
		FelineID:             in.FelineID,
		DietaryPlan:          in.DietaryPlan,
		FeedingTimes:         in.FeedingTimes,
		SpecialMedicalNeeds:  in.SpecialMedicalNeeds,
		PreferredPerchZone:   in.PreferredPerchZone,
		EmergencyMedicalHold: in.EmergencyMedicalHold,
		CaretakerID:          in.CaretakerID,
	}
	m.careSchedules[in.FelineID] = cs
	return &workforcev1.UpdateCareScheduleResponse{CareSchedule: cs}, nil
}

func (m *mockInternalBLLClient) ListGroupFolders(ctx context.Context, in *internalbllv1.ListGroupFoldersRequest) (*internalbllv1.ListGroupFoldersResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &internalbllv1.ListGroupFoldersResponse{GroupFolders: m.folders}, nil
}

func (m *mockInternalBLLClient) ListAlerts(ctx context.Context, in *opsv1.ListAlertsRequest) (*opsv1.ListAlertsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &opsv1.ListAlertsResponse{Alerts: m.alerts}, nil
}

func (m *mockInternalBLLClient) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	return &opsv1.AcknowledgeAlertResponse{
		AlertID:        in.AlertID,
		State:          "acknowledged",
		AcknowledgedBy: in.AcknowledgedBy,
		AcknowledgedAt: time.Now().UTC().Format(time.RFC3339),
		StatusMessage:  "Ticket acknowledged via Pebble Watch",
	}, nil
}

func (m *mockInternalBLLClient) CreateTicket(ctx context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ticketID := fmt.Sprintf("ticket-%d", len(m.tickets)+1)
	t := &opsv1.ITTicket{
		TicketID:       ticketID,
		ForgejoRepo:    in.ForgejoRepo,
		Title:          in.Title,
		Body:           in.Body,
		State:          "open",
		AuthorUsername: in.AuthorUsername,
		Severity:       in.Severity,
		Category:       in.Category,
		Source:         in.Source,
	}
	m.tickets[ticketID] = t
	return &opsv1.CreateTicketResponse{Ticket: t}, nil
}

func (m *mockInternalBLLClient) GetTicket(ctx context.Context, in *opsv1.GetTicketRequest) (*opsv1.GetTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[in.TicketID]
	if !ok {
		t = &opsv1.ITTicket{
			TicketID:       in.TicketID,
			ForgejoRepo:    "infra/microk8s",
			Title:          "Sample Ticket",
			State:          "open",
			AuthorUsername: "emp-human-frank",
		}
	}
	return &opsv1.GetTicketResponse{Ticket: t}, nil
}

func (m *mockInternalBLLClient) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*opsv1.ITTicket
	for _, t := range m.tickets {
		list = append(list, t)
	}
	if len(list) == 0 {
		list = []*opsv1.ITTicket{
			{
				TicketID:       "ticket-test-123",
				ForgejoRepo:    "infra/microk8s",
				Title:          "Pod OOMKilled Warning",
				State:          "open",
				AuthorUsername: "emp-human-frank",
			},
		}
	}
	return &opsv1.ListTicketsResponse{Tickets: list}, nil
}

func (m *mockInternalBLLClient) UpdateTicket(ctx context.Context, in *opsv1.UpdateTicketRequest) (*opsv1.UpdateTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[in.TicketID]
	if !ok {
		t = &opsv1.ITTicket{TicketID: in.TicketID}
		m.tickets[in.TicketID] = t
	}
	t.State = in.State
	t.AssigneeUsername = in.AssigneeUsername
	t.ResolutionNotes = in.ResolutionNotes
	return &opsv1.UpdateTicketResponse{Ticket: t}, nil
}

func (m *mockInternalBLLClient) ListTicketComments(ctx context.Context, in *opsv1.ListTicketCommentsRequest) (*opsv1.ListTicketCommentsResponse, error) {
	return &opsv1.ListTicketCommentsResponse{
		Comments: []*opsv1.TicketComment{
			{
				CommentID:      "comm-test-1",
				TicketID:       in.TicketID,
				AuthorUsername: "emp-human-elise",
				Comment:        "Sample comment",
				IsInternal:     false,
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) AddTicketComment(ctx context.Context, in *opsv1.AddTicketCommentRequest) (*opsv1.AddTicketCommentResponse, error) {
	return &opsv1.AddTicketCommentResponse{
		Comment: &opsv1.TicketComment{
			CommentID:      "comm-test-1",
			TicketID:       in.TicketID,
			AuthorUsername: in.AuthorUsername,
			Comment:        in.Comment,
			IsInternal:     in.IsInternal,
		},
	}, nil
}

func (m *mockInternalBLLClient) ListCustomerTickets(ctx context.Context, in *opsv1.ListCustomerTicketsRequest) (*opsv1.ListCustomerTicketsResponse, error) {
	return &opsv1.ListCustomerTicketsResponse{
		Tickets: []*opsv1.ITTicket{
			{
				TicketID:       "cust-ticket-999",
				Title:          "Sample Customer Ticket",
				Source:         "customer",
				CustomerID:     in.CustomerID,
				AuthorUsername: in.CustomerID,
				State:          "open",
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) CreateCustomerTicket(ctx context.Context, in *opsv1.CreateCustomerTicketRequest) (*opsv1.CreateCustomerTicketResponse, error) {
	return &opsv1.CreateCustomerTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID:       "cust-ticket-999",
			Title:          in.Title,
			Source:         "customer",
			CustomerID:     in.CustomerID,
			AuthorUsername: in.CustomerID,
			State:          "open",
		},
	}, nil
}

func (m *mockInternalBLLClient) SendPushNotification(ctx context.Context, in *opsv1.SendPushNotificationRequest) (*opsv1.SendPushNotificationResponse, error) {
	return &opsv1.SendPushNotificationResponse{
		Status:       "dispatched",
		Provider:     "ntfy",
		Topic:        in.Topic,
		DispatchedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
