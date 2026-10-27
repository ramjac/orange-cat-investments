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
	return s.opsClient.CreateTicket(ctx, in)
}

func (s *Server) GetTicket(ctx context.Context, in *opsv1.GetTicketRequest) (*opsv1.GetTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.GetTicket(ctx, in)
}

func (s *Server) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListTicketsResponse{}, nil
	}
	return s.opsClient.ListTickets(ctx, in)
}

func (s *Server) UpdateTicket(ctx context.Context, in *opsv1.UpdateTicketRequest) (*opsv1.UpdateTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.UpdateTicket(ctx, in)
}

func (s *Server) AcknowledgeTicket(ctx context.Context, in *opsv1.AcknowledgeTicketRequest) (*opsv1.AcknowledgeTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.AcknowledgeTicket(ctx, in)
}

func (s *Server) CloseTicket(ctx context.Context, in *opsv1.CloseTicketRequest) (*opsv1.CloseTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.CloseTicket(ctx, in)
}

func (s *Server) CreateCustomerTicket(ctx context.Context, in *opsv1.CreateCustomerTicketRequest) (*opsv1.CreateCustomerTicketResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.CreateCustomerTicket(ctx, in)
}

func (s *Server) ListCustomerTickets(ctx context.Context, in *opsv1.ListCustomerTicketsRequest) (*opsv1.ListCustomerTicketsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListCustomerTicketsResponse{}, nil
	}
	return s.opsClient.ListCustomerTickets(ctx, in)
}

func (s *Server) AddTicketComment(ctx context.Context, in *opsv1.AddTicketCommentRequest) (*opsv1.AddTicketCommentResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.AddTicketComment(ctx, in)
}

func (s *Server) ListTicketComments(ctx context.Context, in *opsv1.ListTicketCommentsRequest) (*opsv1.ListTicketCommentsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListTicketCommentsResponse{}, nil
	}
	return s.opsClient.ListTicketComments(ctx, in)
}

func (s *Server) ListAlerts(ctx context.Context, in *opsv1.ListAlertsRequest) (*opsv1.ListAlertsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListAlertsResponse{}, nil
	}
	return s.opsClient.ListAlerts(ctx, in)
}

func (s *Server) CreateAlert(ctx context.Context, in *opsv1.CreateAlertRequest) (*opsv1.CreateAlertResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.CreateAlert(ctx, in)
}

func (s *Server) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.AcknowledgeAlert(ctx, in)
}

func (s *Server) SendPushNotification(ctx context.Context, in *opsv1.SendPushNotificationRequest) (*opsv1.SendPushNotificationResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	return s.opsClient.SendPushNotification(ctx, in)
}
