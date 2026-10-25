package coreinvestv1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
)

type coreInvestServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCoreInvestServiceClient(cc grpc.ClientConnInterface) CoreInvestServiceClient {
	return &coreInvestServiceClient{cc: cc}
}

func (c *coreInvestServiceClient) GetPortfolio(ctx context.Context, in *GetPortfolioRequest) (*GetPortfolioResponse, error) {
	out := new(GetPortfolioResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/GetPortfolio", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) ListHoldings(ctx context.Context, in *ListHoldingsRequest) (*ListHoldingsResponse, error) {
	out := new(ListHoldingsResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/ListHoldings", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) UpdatePortfolioBalance(ctx context.Context, in *UpdatePortfolioBalanceRequest) (*UpdatePortfolioBalanceResponse, error) {
	out := new(UpdatePortfolioBalanceResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/UpdatePortfolioBalance", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) CreateOrder(ctx context.Context, in *CreateOrderRequest) (*CreateOrderResponse, error) {
	out := new(CreateOrderResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/CreateOrder", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) ListOrders(ctx context.Context, in *ListOrdersRequest) (*ListOrdersResponse, error) {
	out := new(ListOrdersResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/ListOrders", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) IngestObservation(ctx context.Context, in *IngestObservationRequest) (*IngestObservationResponse, error) {
	out := new(IngestObservationResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/IngestObservation", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) ListObservations(ctx context.Context, in *ListObservationsRequest) (*ListObservationsResponse, error) {
	out := new(ListObservationsResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/ListObservations", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) CreateRecurringDeposit(ctx context.Context, in *CreateRecurringDepositRequest) (*CreateRecurringDepositResponse, error) {
	out := new(CreateRecurringDepositResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/CreateRecurringDeposit", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) ListRecurringDeposits(ctx context.Context, in *ListRecurringDepositsRequest) (*ListRecurringDepositsResponse, error) {
	out := new(ListRecurringDepositsResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/ListRecurringDeposits", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) AdvanceRecurringDeposit(ctx context.Context, in *AdvanceRecurringDepositRequest) (*AdvanceRecurringDepositResponse, error) {
	out := new(AdvanceRecurringDepositResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/AdvanceRecurringDeposit", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) ListPortfolios(ctx context.Context, in *ListPortfoliosRequest) (*ListPortfoliosResponse, error) {
	out := new(ListPortfoliosResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/ListPortfolios", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) GenerateStatement(ctx context.Context, in *GenerateStatementRequest) (*GenerateStatementResponse, error) {
	out := new(GenerateStatementResponse)
	err := c.cc.Invoke(ctx, "/oci.core_invest.v1.CoreInvestService/GenerateStatement", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreInvestServiceClient) StreamPortfolioValuation(ctx context.Context, in *StreamPortfolioValuationRequest) (CoreInvestService_StreamPortfolioValuationClient, error) {
	stream, err := c.cc.NewStream(ctx, &CoreInvestService_ServiceDesc.Streams[0], "/oci.core_invest.v1.CoreInvestService/StreamPortfolioValuation")
	if err != nil {
		return nil, err
	}
	x := &coreInvestServiceStreamPortfolioValuationClient{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type coreInvestServiceStreamPortfolioValuationClient struct {
	grpc.ClientStream
}

func (x *coreInvestServiceStreamPortfolioValuationClient) Recv() (*PortfolioValuationEvent, error) {
	m := new(PortfolioValuationEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server registration and method dispatchers

func _CoreInvestService_GetPortfolio_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(GetPortfolioRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).GetPortfolio(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/GetPortfolio",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).GetPortfolio(ctx, req.(*GetPortfolioRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_ListHoldings_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListHoldingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).ListHoldings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/ListHoldings",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).ListHoldings(ctx, req.(*ListHoldingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_UpdatePortfolioBalance_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(UpdatePortfolioBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).UpdatePortfolioBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/UpdatePortfolioBalance",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).UpdatePortfolioBalance(ctx, req.(*UpdatePortfolioBalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_CreateOrder_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateOrderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).CreateOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/CreateOrder",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).CreateOrder(ctx, req.(*CreateOrderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_ListOrders_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListOrdersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).ListOrders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/ListOrders",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).ListOrders(ctx, req.(*ListOrdersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_IngestObservation_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(IngestObservationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).IngestObservation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/IngestObservation",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).IngestObservation(ctx, req.(*IngestObservationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_ListObservations_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListObservationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).ListObservations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/ListObservations",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).ListObservations(ctx, req.(*ListObservationsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_CreateRecurringDeposit_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateRecurringDepositRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).CreateRecurringDeposit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/CreateRecurringDeposit",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).CreateRecurringDeposit(ctx, req.(*CreateRecurringDepositRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_ListRecurringDeposits_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListRecurringDepositsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).ListRecurringDeposits(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/ListRecurringDeposits",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).ListRecurringDeposits(ctx, req.(*ListRecurringDepositsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_AdvanceRecurringDeposit_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(AdvanceRecurringDepositRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).AdvanceRecurringDeposit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/AdvanceRecurringDeposit",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).AdvanceRecurringDeposit(ctx, req.(*AdvanceRecurringDepositRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_ListPortfolios_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListPortfoliosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).ListPortfolios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/ListPortfolios",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).ListPortfolios(ctx, req.(*ListPortfoliosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_GenerateStatement_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(GenerateStatementRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreInvestServiceServer).GenerateStatement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.core_invest.v1.CoreInvestService/GenerateStatement",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(CoreInvestServiceServer).GenerateStatement(ctx, req.(*GenerateStatementRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CoreInvestService_StreamPortfolioValuation_Handler(srv any, stream grpc.ServerStream) error {
	m := new(StreamPortfolioValuationRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CoreInvestServiceServer).StreamPortfolioValuation(m, &coreInvestServiceStreamPortfolioValuationServer{ServerStream: stream})
}

type coreInvestServiceStreamPortfolioValuationServer struct {
	grpc.ServerStream
}

func (x *coreInvestServiceStreamPortfolioValuationServer) Send(m *PortfolioValuationEvent) error {
	return x.ServerStream.SendMsg(m)
}

var CoreInvestService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "oci.core_invest.v1.CoreInvestService",
	HandlerType: (*CoreInvestServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetPortfolio", Handler: _CoreInvestService_GetPortfolio_Handler},
		{MethodName: "ListHoldings", Handler: _CoreInvestService_ListHoldings_Handler},
		{MethodName: "UpdatePortfolioBalance", Handler: _CoreInvestService_UpdatePortfolioBalance_Handler},
		{MethodName: "CreateOrder", Handler: _CoreInvestService_CreateOrder_Handler},
		{MethodName: "ListOrders", Handler: _CoreInvestService_ListOrders_Handler},
		{MethodName: "IngestObservation", Handler: _CoreInvestService_IngestObservation_Handler},
		{MethodName: "ListObservations", Handler: _CoreInvestService_ListObservations_Handler},
		{MethodName: "CreateRecurringDeposit", Handler: _CoreInvestService_CreateRecurringDeposit_Handler},
		{MethodName: "ListRecurringDeposits", Handler: _CoreInvestService_ListRecurringDeposits_Handler},
		{MethodName: "AdvanceRecurringDeposit", Handler: _CoreInvestService_AdvanceRecurringDeposit_Handler},
		{MethodName: "ListPortfolios", Handler: _CoreInvestService_ListPortfolios_Handler},
		{MethodName: "GenerateStatement", Handler: _CoreInvestService_GenerateStatement_Handler},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamPortfolioValuation",
			Handler:       _CoreInvestService_StreamPortfolioValuation_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/oci/core_invest/v1/core_invest.proto",
}

func RegisterCoreInvestServiceServer(s grpc.ServiceRegistrar, srv CoreInvestServiceServer) {
	s.RegisterService(&CoreInvestService_ServiceDesc, srv)
}
