package internalbll

import (
	"context"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateTicket(ctx context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.CreateTicket(ctx, in)
	return res, s.logDomainError("ops-api", "CreateTicket", err)
}

func (s *Server) GetTicket(ctx context.Context, in *opsv1.GetTicketRequest) (*opsv1.GetTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.GetTicket(ctx, in)
	return res, s.logDomainError("ops-api", "GetTicket", err)
}

func (s *Server) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListTicketsResponse{}, nil
	}
	res, err := s.opsClient.ListTickets(ctx, in)
	return res, s.logDomainError("ops-api", "ListTickets", err)
}

func (s *Server) UpdateTicket(ctx context.Context, in *opsv1.UpdateTicketRequest) (*opsv1.UpdateTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.UpdateTicket(ctx, in)
	return res, s.logDomainError("ops-api", "UpdateTicket", err)
}

func (s *Server) AcknowledgeTicket(ctx context.Context, in *opsv1.AcknowledgeTicketRequest) (*opsv1.AcknowledgeTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.AcknowledgeTicket(ctx, in)
	return res, s.logDomainError("ops-api", "AcknowledgeTicket", err)
}

func (s *Server) CloseTicket(ctx context.Context, in *opsv1.CloseTicketRequest) (*opsv1.CloseTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.CloseTicket(ctx, in)
	return res, s.logDomainError("ops-api", "CloseTicket", err)
}

func (s *Server) CreateCustomerTicket(ctx context.Context, in *opsv1.CreateCustomerTicketRequest) (*opsv1.CreateCustomerTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.CreateCustomerTicket(ctx, in)
	return res, s.logDomainError("ops-api", "CreateCustomerTicket", err)
}

func (s *Server) ListCustomerTickets(ctx context.Context, in *opsv1.ListCustomerTicketsRequest) (*opsv1.ListCustomerTicketsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListCustomerTicketsResponse{}, nil
	}
	res, err := s.opsClient.ListCustomerTickets(ctx, in)
	return res, s.logDomainError("ops-api", "ListCustomerTickets", err)
}

func (s *Server) AddTicketComment(ctx context.Context, in *opsv1.AddTicketCommentRequest) (*opsv1.AddTicketCommentResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.AddTicketComment(ctx, in)
	return res, s.logDomainError("ops-api", "AddTicketComment", err)
}

func (s *Server) ListTicketComments(ctx context.Context, in *opsv1.ListTicketCommentsRequest) (*opsv1.ListTicketCommentsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListTicketCommentsResponse{}, nil
	}
	res, err := s.opsClient.ListTicketComments(ctx, in)
	return res, s.logDomainError("ops-api", "ListTicketComments", err)
}

func (s *Server) ListAlerts(ctx context.Context, in *opsv1.ListAlertsRequest) (*opsv1.ListAlertsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListAlertsResponse{}, nil
	}
	res, err := s.opsClient.ListAlerts(ctx, in)
	return res, s.logDomainError("ops-api", "ListAlerts", err)
}

func (s *Server) CreateAlert(ctx context.Context, in *opsv1.CreateAlertRequest) (*opsv1.CreateAlertResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.CreateAlert(ctx, in)
	return res, s.logDomainError("ops-api", "CreateAlert", err)
}

func (s *Server) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.AcknowledgeAlert(ctx, in)
	return res, s.logDomainError("ops-api", "AcknowledgeAlert", err)
}

func (s *Server) SendPushNotification(ctx context.Context, in *opsv1.SendPushNotificationRequest) (*opsv1.SendPushNotificationResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.SendPushNotification(ctx, in)
	return res, s.logDomainError("ops-api", "SendPushNotification", err)
}
