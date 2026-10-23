package main

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/ops"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

type OpsGRPCServer struct {
	svc ops_svc.Service
}

func NewOpsGRPCServer(svc ops_svc.Service) *OpsGRPCServer {
	return &OpsGRPCServer{svc: svc}
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func toProtoTicket(t *ops.ITTicket) *opsv1.ITTicket {
	if t == nil {
		return nil
	}
	ackAt := ""
	if t.AcknowledgedAt != nil {
		ackAt = t.AcknowledgedAt.Format(time.RFC3339)
	}
	created := ""
	if !t.CreatedAt.IsZero() {
		created = t.CreatedAt.Format(time.RFC3339)
	}
	return &opsv1.ITTicket{
		TicketID:         t.TicketID,
		Title:            t.Title,
		Body:             strVal(t.Body),
		State:            t.State,
		Severity:         t.Severity,
		Category:         t.Category,
		Source:           t.Source,
		CustomerID:       strVal(t.CustomerID),
		AuthorUsername:   t.AuthorUsername,
		AssigneeUsername: strVal(t.AssigneeUsername),
		IsUrgent:         t.IsUrgent,
		AcknowledgedBy:   strVal(t.AcknowledgedBy),
		AcknowledgedAt:   ackAt,
		CreatedAt:        created,
	}
}

func (s *OpsGRPCServer) CreateTicket(ctx context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	ticket, err := s.svc.CreateITTicket(ctx, &ops_svc.CreateITTicketRequest{
		Title:          in.Title,
		Body:           in.Body,
		Severity:       in.Severity,
		Category:       in.Category,
		Source:         in.Source,
		CustomerID:     in.CustomerID,
		AuthorUsername: in.AuthorUsername,
		IsUrgent:       in.IsUrgent,
	})
	if err != nil {
		return nil, err
	}
	return &opsv1.CreateTicketResponse{Ticket: toProtoTicket(ticket)}, nil
}

func (s *OpsGRPCServer) GetTicket(ctx context.Context, in *opsv1.GetTicketRequest) (*opsv1.GetTicketResponse, error) {
	ticket, err := s.svc.GetITTicket(ctx, in.TicketID)
	if err != nil {
		return nil, err
	}
	return &opsv1.GetTicketResponse{Ticket: toProtoTicket(ticket)}, nil
}

func (s *OpsGRPCServer) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	tickets, err := s.svc.ListITTickets(ctx, limit, in.Offset)
	if err != nil {
		return nil, err
	}
	var out []*opsv1.ITTicket
	for _, t := range tickets {
		if in.State != "" && t.State != in.State {
			continue
		}
		if in.Category != "" && t.Category != in.Category {
			continue
		}
		out = append(out, toProtoTicket(t))
	}
	return &opsv1.ListTicketsResponse{Tickets: out}, nil
}

func (s *OpsGRPCServer) AcknowledgeTicket(ctx context.Context, in *opsv1.AcknowledgeTicketRequest) (*opsv1.AcknowledgeTicketResponse, error) {
	res, err := s.svc.ProcessPebbleAck(ctx, &ops_svc.PebbleAckRequest{
		TicketID:       in.TicketID,
		AcknowledgedBy: in.AcknowledgedBy,
	})
	if err != nil {
		return nil, err
	}
	t, _ := s.svc.GetITTicket(ctx, in.TicketID)
	if t == nil {
		t = &ops.ITTicket{
			TicketID:       res.TicketID,
			State:          res.State,
			AcknowledgedBy: &res.AcknowledgedBy,
			AcknowledgedAt: &res.AcknowledgedAt,
		}
	}
	return &opsv1.AcknowledgeTicketResponse{Ticket: toProtoTicket(t)}, nil
}

func (s *OpsGRPCServer) CloseTicket(ctx context.Context, in *opsv1.CloseTicketRequest) (*opsv1.CloseTicketResponse, error) {
	ticket, err := s.svc.UpdateITTicket(ctx, in.TicketID, &ops_svc.UpdateTicketRequest{
		State:           "closed",
		ResolutionNotes: in.ResolutionNotes,
	})
	if err != nil {
		return nil, err
	}
	return &opsv1.CloseTicketResponse{Ticket: toProtoTicket(ticket)}, nil
}

func (s *OpsGRPCServer) ListAlerts(ctx context.Context, in *opsv1.ListAlertsRequest) (*opsv1.ListAlertsResponse, error) {
	alerts, err := s.svc.GetPendingAlerts(ctx)
	if err != nil {
		return nil, err
	}
	var out []*opsv1.PebbleAlert
	for _, a := range alerts {
		out = append(out, &opsv1.PebbleAlert{
			AlertID:      a.TicketID,
			TicketID:     a.TicketID,
			Title:        a.Title,
			Severity:     a.Priority,
			Acknowledged: false,
			CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		})
	}
	return &opsv1.ListAlertsResponse{Alerts: out}, nil
}

func (s *OpsGRPCServer) CreateAlert(ctx context.Context, in *opsv1.CreateAlertRequest) (*opsv1.CreateAlertResponse, error) {
	ticketID := in.TicketID
	if ticketID == "" {
		ticket, err := s.svc.CreateITTicket(ctx, &ops_svc.CreateITTicketRequest{
			Title:          in.Title,
			Body:           in.Title,
			Severity:       in.Severity,
			AuthorUsername: "system-alert",
			IsUrgent:       true,
		})
		if err != nil {
			return nil, err
		}
		ticketID = ticket.TicketID
	}
	return &opsv1.CreateAlertResponse{
		Alert: &opsv1.PebbleAlert{
			AlertID:      ticketID,
			TicketID:     ticketID,
			Title:        in.Title,
			Severity:     in.Severity,
			RoomZone:     in.RoomZone,
			Acknowledged: false,
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *OpsGRPCServer) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	res, err := s.svc.ProcessPebbleAck(ctx, &ops_svc.PebbleAckRequest{
		TicketID:       in.AlertID,
		AcknowledgedBy: in.AcknowledgedBy,
		ButtonID:       uint8(in.ButtonID),
	})
	if err != nil {
		return nil, err
	}
	return &opsv1.AcknowledgeAlertResponse{
		AlertID: res.TicketID,
		Status:  fmt.Sprintf("acknowledged (%s)", res.StatusMessage),
	}, nil
}
