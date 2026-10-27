package coreinvestv1

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCoreInvestServer struct {
	UnimplementedCoreInvestServiceServer
}

func (m *mockCoreInvestServer) GetPortfolio(ctx context.Context, in *GetPortfolioRequest) (*GetPortfolioResponse, error) {
	return &GetPortfolioResponse{
		Portfolio: &Portfolio{
			PortfolioID:     in.PortfolioID,
			TotalBalanceUSD: 10000.0,
			CashBalanceUSD:  2500.0,
		},
		Holdings: []*PortfolioHolding{
			{Symbol: "AAPL", Quantity: 10, CurrentPriceUSD: 220.0},
		},
	}, nil
}
func (m *mockCoreInvestServer) ListHoldings(ctx context.Context, in *ListHoldingsRequest) (*ListHoldingsResponse, error) {
	return &ListHoldingsResponse{}, nil
}
func (m *mockCoreInvestServer) UpdatePortfolioBalance(ctx context.Context, in *UpdatePortfolioBalanceRequest) (*UpdatePortfolioBalanceResponse, error) {
	return &UpdatePortfolioBalanceResponse{PortfolioID: in.PortfolioID, UpdatedCashBalanceUSD: 3000.0}, nil
}
func (m *mockCoreInvestServer) CreateOrder(ctx context.Context, in *CreateOrderRequest) (*CreateOrderResponse, error) {
	return &CreateOrderResponse{Order: &BrokerageOrder{OrderID: "ord-01", Symbol: in.Symbol}}, nil
}
func (m *mockCoreInvestServer) ListOrders(ctx context.Context, in *ListOrdersRequest) (*ListOrdersResponse, error) {
	return &ListOrdersResponse{}, nil
}
func (m *mockCoreInvestServer) IngestObservation(ctx context.Context, in *IngestObservationRequest) (*IngestObservationResponse, error) {
	return &IngestObservationResponse{EventID: "obs-01", Status: "ingested"}, nil
}
func (m *mockCoreInvestServer) ListObservations(ctx context.Context, in *ListObservationsRequest) (*ListObservationsResponse, error) {
	return &ListObservationsResponse{}, nil
}
func (m *mockCoreInvestServer) CreateRecurringDeposit(ctx context.Context, in *CreateRecurringDepositRequest) (*CreateRecurringDepositResponse, error) {
	return &CreateRecurringDepositResponse{Deposit: &RecurringDeposit{DepositID: "dep-01", AmountUSD: in.AmountUSD}}, nil
}
func (m *mockCoreInvestServer) ListRecurringDeposits(ctx context.Context, in *ListRecurringDepositsRequest) (*ListRecurringDepositsResponse, error) {
	return &ListRecurringDepositsResponse{}, nil
}
func (m *mockCoreInvestServer) AdvanceRecurringDeposit(ctx context.Context, in *AdvanceRecurringDepositRequest) (*AdvanceRecurringDepositResponse, error) {
	return &AdvanceRecurringDepositResponse{Status: "advanced"}, nil
}
func (m *mockCoreInvestServer) ListPortfolios(ctx context.Context, in *ListPortfoliosRequest) (*ListPortfoliosResponse, error) {
	return &ListPortfoliosResponse{}, nil
}
func (m *mockCoreInvestServer) GenerateStatement(ctx context.Context, in *GenerateStatementRequest) (*GenerateStatementResponse, error) {
	return &GenerateStatementResponse{StatementID: "stmt-01", Status: "generated"}, nil
}
func (m *mockCoreInvestServer) StreamPortfolioValuation(req *StreamPortfolioValuationRequest, stream CoreInvestService_StreamPortfolioValuationServer) error {
	count := req.UpdateCount
	if count <= 0 {
		count = 3
	}
	base := 10000.0
	for i := 0; i < int(count); i++ {
		val := base + float64(i)*25.5
		err := stream.Send(&PortfolioValuationEvent{
			PortfolioID:        req.PortfolioID,
			TotalBalanceUSD:    val,
			CashBalanceUSD:     2500.0,
			InvestedBalanceUSD: val - 2500.0,
			ChangePct:          float64(i) * 0.25,
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func TestCoreInvestGRPC_RoundTripAndStreaming(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpcutil.NewServer(nil)
	RegisterCoreInvestServiceServer(srv, &mockCoreInvestServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	client := NewCoreInvestServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Unary
	p, err := client.GetPortfolio(ctx, &GetPortfolioRequest{PortfolioID: "port-01"})
	require.NoError(t, err)
	assert.Equal(t, 10000.0, p.Portfolio.TotalBalanceUSD)
	assert.Len(t, p.Holdings, 1)

	// Streaming
	stream, err := client.StreamPortfolioValuation(ctx, &StreamPortfolioValuationRequest{
		PortfolioID: "port-01",
		UpdateCount: 3,
	})
	require.NoError(t, err)

	var events []*PortfolioValuationEvent
	for {
		ev, err := stream.Recv()
		if err != nil {
			break
		}
		events = append(events, ev)
	}

	assert.Len(t, events, 3)
	assert.Equal(t, 10000.0, events[0].TotalBalanceUSD)
	assert.Equal(t, fmt.Sprintf("%.1f", 10051.0), fmt.Sprintf("%.1f", events[2].TotalBalanceUSD))
}
