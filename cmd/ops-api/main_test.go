package main

import (
	"net"
	"testing"

	ops_repo "github.com/orange-cat-investments/oci/internal/repository/ops"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestOpsAPI_GRPC(t *testing.T) {
	repo := ops_repo.NewMockRepository()
	svc := ops_svc.NewService(repo)

	grpcServer := grpcutil.NewServer(nil)
	grpcSrv := NewOpsGRPCServer(svc)
	opsv1.RegisterOpsServiceServer(grpcServer, grpcSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	conn, err := grpcutil.Dial(t.Context(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// Test Health
	healthClient := grpc_health_v1.NewHealthClient(conn)
	hRes, err := healthClient.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, hRes.Status)

	// Test Service RPC
	client := opsv1.NewOpsServiceClient(conn)
	res, err := client.ListTickets(t.Context(), &opsv1.ListTicketsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, res)

	// Create a customer ticket and verify CloseTicket treats resolution notes as ordinary data
	created, err := client.CreateCustomerTicket(t.Context(), &opsv1.CreateCustomerTicketRequest{
		CustomerID: "cust-gdpr-ops-test",
		Category:   "billing",
		Title:      "Billing question",
		Body:       "Customer PII body",
	})
	require.NoError(t, err)
	require.NotNil(t, created.Ticket)

	closed, err := client.CloseTicket(t.Context(), &opsv1.CloseTicketRequest{
		TicketID:        created.Ticket.TicketID,
		ResolutionNotes: "GDPR_ARTICLE_17_ERASURE",
	})
	require.NoError(t, err)
	assert.Equal(t, "closed", closed.Ticket.State)
	assert.Equal(t, "GDPR_ARTICLE_17_ERASURE", closed.Ticket.ResolutionNotes)
	assert.Equal(t, "Customer PII body", closed.Ticket.Body)

	// Verify explicit ScrubCustomerGDPRData RPC scrubs customer data
	scrubRes, err := client.ScrubCustomerGDPRData(t.Context(), &opsv1.ScrubCustomerGDPRDataRequest{
		CustomerID: "cust-gdpr-ops-test",
	})
	require.NoError(t, err)
	assert.Equal(t, "cust-gdpr-ops-test", scrubRes.CustomerID)
	assert.GreaterOrEqual(t, scrubRes.ScrubbedRecordsCount, int32(1))
	assert.Equal(t, []string{"ops.it_tickets", "ops.ticket_comments"}, scrubRes.AnonymizedTables)
}
