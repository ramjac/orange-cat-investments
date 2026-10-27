package opsv1

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOpsServer struct {
	UnimplementedOpsServiceServer
}

func (m *mockOpsServer) CreateTicket(ctx context.Context, in *CreateTicketRequest) (*CreateTicketResponse, error) {
	return &CreateTicketResponse{Ticket: &ITTicket{TicketID: "tkt-01", Title: in.Title, Severity: in.Severity}}, nil
}
func (m *mockOpsServer) GetTicket(ctx context.Context, in *GetTicketRequest) (*GetTicketResponse, error) {
	return &GetTicketResponse{Ticket: &ITTicket{TicketID: in.TicketID, State: "open"}}, nil
}
func (m *mockOpsServer) ListTickets(ctx context.Context, in *ListTicketsRequest) (*ListTicketsResponse, error) {
	return &ListTicketsResponse{Tickets: []*ITTicket{{TicketID: "tkt-01"}}}, nil
}
func (m *mockOpsServer) AcknowledgeTicket(ctx context.Context, in *AcknowledgeTicketRequest) (*AcknowledgeTicketResponse, error) {
	return &AcknowledgeTicketResponse{Ticket: &ITTicket{TicketID: in.TicketID, AcknowledgedBy: in.AcknowledgedBy}}, nil
}
func (m *mockOpsServer) CloseTicket(ctx context.Context, in *CloseTicketRequest) (*CloseTicketResponse, error) {
	return &CloseTicketResponse{Ticket: &ITTicket{TicketID: in.TicketID, State: "closed"}}, nil
}
func (m *mockOpsServer) ListAlerts(ctx context.Context, in *ListAlertsRequest) (*ListAlertsResponse, error) {
	return &ListAlertsResponse{Alerts: []*PebbleAlert{{AlertID: "alt-01"}}}, nil
}
func (m *mockOpsServer) CreateAlert(ctx context.Context, in *CreateAlertRequest) (*CreateAlertResponse, error) {
	return &CreateAlertResponse{Alert: &PebbleAlert{AlertID: "alt-01", Title: in.Title}}, nil
}
func (m *mockOpsServer) AcknowledgeAlert(ctx context.Context, in *AcknowledgeAlertRequest) (*AcknowledgeAlertResponse, error) {
	return &AcknowledgeAlertResponse{AlertID: in.AlertID, Status: "acknowledged"}, nil
}

func TestOpsGRPC_RoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpcutil.NewServer(nil)
	RegisterOpsServiceServer(srv, &mockOpsServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	client := NewOpsServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ticket, err := client.CreateTicket(ctx, &CreateTicketRequest{
		Title:    "Network issue",
		Severity: "urgent",
	})
	require.NoError(t, err)
	assert.Equal(t, "tkt-01", ticket.Ticket.TicketID)
	assert.Equal(t, "Network issue", ticket.Ticket.Title)

	ack, err := client.AcknowledgeAlert(ctx, &AcknowledgeAlertRequest{
		AlertID:        "alt-01",
		AcknowledgedBy: "emp-tech-01",
	})
	require.NoError(t, err)
	assert.Equal(t, "acknowledged", ack.Status)
}
