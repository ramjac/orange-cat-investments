package customerbll

import (
	"context"
	"strings"

	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ListCustomerTickets(ctx context.Context, in *customerbllv1.ListCustomerTicketsRequest) (*customerbllv1.ListCustomerTicketsResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops service unavailable")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if in.CustomerID != "" {
		res, err := s.opsClient.ListCustomerTickets(ctx, &opsv1.ListCustomerTicketsRequest{
			CustomerID: in.CustomerID,
			Limit:      limit,
			Offset:     in.Offset,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list customer tickets: %v", err)
		}
		return &customerbllv1.ListCustomerTicketsResponse{Tickets: res.Tickets}, nil
	}
	res, err := s.opsClient.ListTickets(ctx, &opsv1.ListTicketsRequest{Limit: limit, Offset: in.Offset})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tickets: %v", err)
	}
	return &customerbllv1.ListCustomerTicketsResponse{Tickets: res.Tickets}, nil
}

func (s *Server) CreateCustomerTicket(ctx context.Context, in *customerbllv1.CreateCustomerTicketRequest) (*customerbllv1.CreateCustomerTicketResponse, error) {
	if in == nil || strings.TrimSpace(in.Title) == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops service unavailable")
	}
	category := in.Category
	if category == "" {
		category = "General Inquiry"
	}
	if in.CustomerID != "" {
		res, err := s.opsClient.CreateCustomerTicket(ctx, &opsv1.CreateCustomerTicketRequest{
			CustomerID: in.CustomerID,
			Category:   category,
			Title:      in.Title,
			Body:       in.Body,
		})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return &customerbllv1.CreateCustomerTicketResponse{Ticket: res.Ticket}, nil
	}
	res, err := s.opsClient.CreateTicket(ctx, &opsv1.CreateTicketRequest{
		Title:    in.Title,
		Body:     in.Body,
		Severity: in.Severity,
		Category: category,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create ticket: %v", err)
	}
	return &customerbllv1.CreateCustomerTicketResponse{Ticket: res.Ticket}, nil
}

func (s *Server) GetCustomerTicket(ctx context.Context, in *customerbllv1.GetCustomerTicketRequest) (*customerbllv1.GetCustomerTicketResponse, error) {
	if in == nil || in.TicketID == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops service unavailable")
	}
	res, err := s.opsClient.GetTicket(ctx, &opsv1.GetTicketRequest{TicketID: in.TicketID})
	if err != nil || res == nil || res.Ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	if in.CustomerID != "" && res.Ticket.CustomerID != "" && res.Ticket.CustomerID != in.CustomerID {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	return &customerbllv1.GetCustomerTicketResponse{Ticket: res.Ticket}, nil
}

func (s *Server) ListCustomerTicketComments(ctx context.Context, in *customerbllv1.ListCustomerTicketCommentsRequest) (*customerbllv1.ListCustomerTicketCommentsResponse, error) {
	if in == nil || in.TicketID == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops service unavailable")
	}
	tRes, err := s.opsClient.GetTicket(ctx, &opsv1.GetTicketRequest{TicketID: in.TicketID})
	if err != nil || tRes == nil || tRes.Ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	if in.CustomerID != "" && tRes.Ticket.CustomerID != "" && tRes.Ticket.CustomerID != in.CustomerID {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	res, err := s.opsClient.ListTicketComments(ctx, &opsv1.ListTicketCommentsRequest{
		TicketID:        in.TicketID,
		IncludeInternal: false,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list comments: %v", err)
	}
	return &customerbllv1.ListCustomerTicketCommentsResponse{Comments: res.Comments}, nil
}

func (s *Server) AddCustomerTicketComment(ctx context.Context, in *customerbllv1.AddCustomerTicketCommentRequest) (*customerbllv1.AddCustomerTicketCommentResponse, error) {
	if in == nil || in.TicketID == "" || strings.TrimSpace(in.Comment) == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket_id and comment are required")
	}
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops service unavailable")
	}
	tRes, err := s.opsClient.GetTicket(ctx, &opsv1.GetTicketRequest{TicketID: in.TicketID})
	if err != nil || tRes == nil || tRes.Ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	if in.CustomerID != "" && tRes.Ticket.CustomerID != "" && tRes.Ticket.CustomerID != in.CustomerID {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	author := in.AuthorUsername
	if author == "" {
		author = in.CustomerID
	}
	if author == "" {
		author = "Customer"
	}
	res, err := s.opsClient.AddTicketComment(ctx, &opsv1.AddTicketCommentRequest{
		TicketID:       in.TicketID,
		AuthorUsername: author,
		Comment:        in.Comment,
		IsInternal:     false,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &customerbllv1.AddCustomerTicketCommentResponse{Comment: res.Comment}, nil
}
