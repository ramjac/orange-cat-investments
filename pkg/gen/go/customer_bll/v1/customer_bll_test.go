package customerbllv1

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCustomerBLLServer struct {
	UnimplementedCustomerBLLServiceServer
}

func (m *mockCustomerBLLServer) GetPortfolio(ctx context.Context, in *GetPortfolioRequest) (*GetPortfolioResponse, error) {
	return &GetPortfolioResponse{
		PortfolioID:     "port-" + in.CustomerID,
		CustomerID:      in.CustomerID,
		TotalBalanceUSD: 25000.0,
	}, nil
}

func TestCustomerBLLGRPC_RoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpcutil.NewServer(nil)
	RegisterCustomerBLLServiceServer(srv, &mockCustomerBLLServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	client := NewCustomerBLLServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := client.GetPortfolio(ctx, &GetPortfolioRequest{CustomerID: "cust-01"})
	require.NoError(t, err)
	assert.Equal(t, "port-cust-01", res.PortfolioID)
	assert.Equal(t, 25000.0, res.TotalBalanceUSD)
}
