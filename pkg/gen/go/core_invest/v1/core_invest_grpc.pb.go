package coreinvestv1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type coreInvestServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCoreInvestServiceClient(cc grpc.ClientConnInterface) CoreInvestServiceClient {
	return &coreInvestServiceClient{cc: cc}
}

func invokeUnary[Req any, Resp any](ctx context.Context, cc grpc.ClientConnInterface, method string, in *Req) (*Resp, error) {
	out := new(Resp)
	if err := cc.Invoke(ctx, method, in, out); err != nil {
		return nil, err
	}
	return out, nil
}

func dispatchUnary[Req any, Resp any](
	srv any,
	ctx context.Context,
	dec func(any) error,
	interceptor grpc.UnaryServerInterceptor,
	fullMethod string,
	fn func(CoreInvestServiceServer, context.Context, *Req) (*Resp, error),
) (any, error) {
	in := new(Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return fn(srv.(CoreInvestServiceServer), ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: fullMethod}
	handler := func(ctx context.Context, req any) (any, error) {
		return fn(srv.(CoreInvestServiceServer), ctx, req.(*Req))
	}
	return interceptor(ctx, in, info, handler)
}

func (c *coreInvestServiceClient) GetPortfolio(ctx context.Context, in *GetPortfolioRequest) (*GetPortfolioResponse, error) {
	return invokeUnary[GetPortfolioRequest, GetPortfolioResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GetPortfolio", in)
}
func (c *coreInvestServiceClient) ListPortfolios(ctx context.Context, in *ListPortfoliosRequest) (*ListPortfoliosResponse, error) {
	return invokeUnary[ListPortfoliosRequest, ListPortfoliosResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListPortfolios", in)
}
func (c *coreInvestServiceClient) CreatePortfolio(ctx context.Context, in *CreatePortfolioRequest) (*CreatePortfolioResponse, error) {
	return invokeUnary[CreatePortfolioRequest, CreatePortfolioResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/CreatePortfolio", in)
}
func (c *coreInvestServiceClient) UpdatePortfolio(ctx context.Context, in *UpdatePortfolioRequest) (*UpdatePortfolioResponse, error) {
	return invokeUnary[UpdatePortfolioRequest, UpdatePortfolioResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/UpdatePortfolio", in)
}
func (c *coreInvestServiceClient) ListHoldings(ctx context.Context, in *ListHoldingsRequest) (*ListHoldingsResponse, error) {
	return invokeUnary[ListHoldingsRequest, ListHoldingsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListHoldings", in)
}
func (c *coreInvestServiceClient) UpsertHolding(ctx context.Context, in *UpsertHoldingRequest) (*UpsertHoldingResponse, error) {
	return invokeUnary[UpsertHoldingRequest, UpsertHoldingResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/UpsertHolding", in)
}
func (c *coreInvestServiceClient) UpdatePortfolioBalance(ctx context.Context, in *UpdatePortfolioBalanceRequest) (*UpdatePortfolioBalanceResponse, error) {
	return invokeUnary[UpdatePortfolioBalanceRequest, UpdatePortfolioBalanceResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/UpdatePortfolioBalance", in)
}
func (c *coreInvestServiceClient) RegisterCustomer(ctx context.Context, in *RegisterCustomerRequest) (*RegisterCustomerResponse, error) {
	return invokeUnary[RegisterCustomerRequest, RegisterCustomerResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/RegisterCustomer", in)
}
func (c *coreInvestServiceClient) CloseAndLiquidateAccount(ctx context.Context, in *CloseAndLiquidateAccountRequest) (*CloseAndLiquidateAccountResponse, error) {
	return invokeUnary[CloseAndLiquidateAccountRequest, CloseAndLiquidateAccountResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/CloseAndLiquidateAccount", in)
}
func (c *coreInvestServiceClient) ExecuteGDPRErasure(ctx context.Context, in *ExecuteGDPRErasureRequest) (*ExecuteGDPRErasureResponse, error) {
	return invokeUnary[ExecuteGDPRErasureRequest, ExecuteGDPRErasureResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ExecuteGDPRErasure", in)
}
func (c *coreInvestServiceClient) GetGDPRErasureCertificate(ctx context.Context, in *GetGDPRErasureCertificateRequest) (*GetGDPRErasureCertificateResponse, error) {
	return invokeUnary[GetGDPRErasureCertificateRequest, GetGDPRErasureCertificateResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GetGDPRErasureCertificate", in)
}
func (c *coreInvestServiceClient) CreateAPIKey(ctx context.Context, in *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	return invokeUnary[CreateAPIKeyRequest, CreateAPIKeyResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/CreateAPIKey", in)
}
func (c *coreInvestServiceClient) ListAPIKeys(ctx context.Context, in *ListAPIKeysRequest) (*ListAPIKeysResponse, error) {
	return invokeUnary[ListAPIKeysRequest, ListAPIKeysResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListAPIKeys", in)
}
func (c *coreInvestServiceClient) RevokeAPIKey(ctx context.Context, in *RevokeAPIKeyRequest) (*RevokeAPIKeyResponse, error) {
	return invokeUnary[RevokeAPIKeyRequest, RevokeAPIKeyResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/RevokeAPIKey", in)
}
func (c *coreInvestServiceClient) CreateWebhook(ctx context.Context, in *CreateWebhookRequest) (*CreateWebhookResponse, error) {
	return invokeUnary[CreateWebhookRequest, CreateWebhookResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/CreateWebhook", in)
}
func (c *coreInvestServiceClient) ListWebhooks(ctx context.Context, in *ListWebhooksRequest) (*ListWebhooksResponse, error) {
	return invokeUnary[ListWebhooksRequest, ListWebhooksResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListWebhooks", in)
}
func (c *coreInvestServiceClient) DeleteWebhook(ctx context.Context, in *DeleteWebhookRequest) (*DeleteWebhookResponse, error) {
	return invokeUnary[DeleteWebhookRequest, DeleteWebhookResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/DeleteWebhook", in)
}
func (c *coreInvestServiceClient) CreateOrder(ctx context.Context, in *CreateOrderRequest) (*CreateOrderResponse, error) {
	return invokeUnary[CreateOrderRequest, CreateOrderResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/CreateOrder", in)
}
func (c *coreInvestServiceClient) ListOrders(ctx context.Context, in *ListOrdersRequest) (*ListOrdersResponse, error) {
	return invokeUnary[ListOrdersRequest, ListOrdersResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListOrders", in)
}
func (c *coreInvestServiceClient) RegisterCameraStream(ctx context.Context, in *RegisterCameraStreamRequest) (*RegisterCameraStreamResponse, error) {
	return invokeUnary[RegisterCameraStreamRequest, RegisterCameraStreamResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/RegisterCameraStream", in)
}
func (c *coreInvestServiceClient) GetCameraStream(ctx context.Context, in *GetCameraStreamRequest) (*GetCameraStreamResponse, error) {
	return invokeUnary[GetCameraStreamRequest, GetCameraStreamResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GetCameraStream", in)
}
func (c *coreInvestServiceClient) ListCameraStreams(ctx context.Context, in *ListCameraStreamsRequest) (*ListCameraStreamsResponse, error) {
	return invokeUnary[ListCameraStreamsRequest, ListCameraStreamsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListCameraStreams", in)
}
func (c *coreInvestServiceClient) IngestObservation(ctx context.Context, in *IngestObservationRequest) (*IngestObservationResponse, error) {
	return invokeUnary[IngestObservationRequest, IngestObservationResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/IngestObservation", in)
}
func (c *coreInvestServiceClient) ListObservations(ctx context.Context, in *ListObservationsRequest) (*ListObservationsResponse, error) {
	return invokeUnary[ListObservationsRequest, ListObservationsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListObservations", in)
}
func (c *coreInvestServiceClient) StartBacktestRun(ctx context.Context, in *StartBacktestRunRequest) (*StartBacktestRunResponse, error) {
	return invokeUnary[StartBacktestRunRequest, StartBacktestRunResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/StartBacktestRun", in)
}
func (c *coreInvestServiceClient) GetBacktestRun(ctx context.Context, in *GetBacktestRunRequest) (*GetBacktestRunResponse, error) {
	return invokeUnary[GetBacktestRunRequest, GetBacktestRunResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GetBacktestRun", in)
}
func (c *coreInvestServiceClient) ListBacktestRuns(ctx context.Context, in *ListBacktestRunsRequest) (*ListBacktestRunsResponse, error) {
	return invokeUnary[ListBacktestRunsRequest, ListBacktestRunsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListBacktestRuns", in)
}
func (c *coreInvestServiceClient) ListMLModels(ctx context.Context, in *ListMLModelsRequest) (*ListMLModelsResponse, error) {
	return invokeUnary[ListMLModelsRequest, ListMLModelsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListMLModels", in)
}
func (c *coreInvestServiceClient) LogModelDrift(ctx context.Context, in *LogModelDriftRequest) (*LogModelDriftResponse, error) {
	return invokeUnary[LogModelDriftRequest, LogModelDriftResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/LogModelDrift", in)
}
func (c *coreInvestServiceClient) CreateRecurringDeposit(ctx context.Context, in *CreateRecurringDepositRequest) (*CreateRecurringDepositResponse, error) {
	return invokeUnary[CreateRecurringDepositRequest, CreateRecurringDepositResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/CreateRecurringDeposit", in)
}
func (c *coreInvestServiceClient) ListRecurringDeposits(ctx context.Context, in *ListRecurringDepositsRequest) (*ListRecurringDepositsResponse, error) {
	return invokeUnary[ListRecurringDepositsRequest, ListRecurringDepositsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListRecurringDeposits", in)
}
func (c *coreInvestServiceClient) AdvanceRecurringDeposit(ctx context.Context, in *AdvanceRecurringDepositRequest) (*AdvanceRecurringDepositResponse, error) {
	return invokeUnary[AdvanceRecurringDepositRequest, AdvanceRecurringDepositResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/AdvanceRecurringDeposit", in)
}
func (c *coreInvestServiceClient) ClaimDueRecurringDeposits(ctx context.Context, in *ClaimDueRecurringDepositsRequest) (*ClaimDueRecurringDepositsResponse, error) {
	return invokeUnary[ClaimDueRecurringDepositsRequest, ClaimDueRecurringDepositsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ClaimDueRecurringDeposits", in)
}
func (c *coreInvestServiceClient) GenerateStatement(ctx context.Context, in *GenerateStatementRequest) (*GenerateStatementResponse, error) {
	return invokeUnary[GenerateStatementRequest, GenerateStatementResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GenerateStatement", in)
}
func (c *coreInvestServiceClient) ListStatements(ctx context.Context, in *ListStatementsRequest) (*ListStatementsResponse, error) {
	return invokeUnary[ListStatementsRequest, ListStatementsResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/ListStatements", in)
}
func (c *coreInvestServiceClient) GetStatement(ctx context.Context, in *GetStatementRequest) (*GetStatementResponse, error) {
	return invokeUnary[GetStatementRequest, GetStatementResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GetStatement", in)
}
func (c *coreInvestServiceClient) GenerateMonthlyStatement(ctx context.Context, in *GenerateMonthlyStatementRequest) (*GenerateMonthlyStatementResponse, error) {
	return invokeUnary[GenerateMonthlyStatementRequest, GenerateMonthlyStatementResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GenerateMonthlyStatement", in)
}
func (c *coreInvestServiceClient) GenerateForm1099B(ctx context.Context, in *GenerateForm1099BRequest) (*GenerateForm1099BResponse, error) {
	return invokeUnary[GenerateForm1099BRequest, GenerateForm1099BResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GenerateForm1099B", in)
}
func (c *coreInvestServiceClient) GetStatementDocument(ctx context.Context, in *GetStatementDocumentRequest) (*GetStatementDocumentResponse, error) {
	return invokeUnary[GetStatementDocumentRequest, GetStatementDocumentResponse](ctx, c.cc, "/oci.core_invest.v1.CoreInvestService/GetStatementDocument", in)
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

type UnimplementedCoreInvestServiceServer struct{}

func (UnimplementedCoreInvestServiceServer) GetPortfolio(context.Context, *GetPortfolioRequest) (*GetPortfolioResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPortfolio not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListPortfolios(context.Context, *ListPortfoliosRequest) (*ListPortfoliosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPortfolios not implemented")
}
func (UnimplementedCoreInvestServiceServer) CreatePortfolio(context.Context, *CreatePortfolioRequest) (*CreatePortfolioResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreatePortfolio not implemented")
}
func (UnimplementedCoreInvestServiceServer) UpdatePortfolio(context.Context, *UpdatePortfolioRequest) (*UpdatePortfolioResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePortfolio not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListHoldings(context.Context, *ListHoldingsRequest) (*ListHoldingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListHoldings not implemented")
}
func (UnimplementedCoreInvestServiceServer) UpsertHolding(context.Context, *UpsertHoldingRequest) (*UpsertHoldingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpsertHolding not implemented")
}
func (UnimplementedCoreInvestServiceServer) UpdatePortfolioBalance(context.Context, *UpdatePortfolioBalanceRequest) (*UpdatePortfolioBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePortfolioBalance not implemented")
}
func (UnimplementedCoreInvestServiceServer) RegisterCustomer(context.Context, *RegisterCustomerRequest) (*RegisterCustomerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterCustomer not implemented")
}
func (UnimplementedCoreInvestServiceServer) CloseAndLiquidateAccount(context.Context, *CloseAndLiquidateAccountRequest) (*CloseAndLiquidateAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CloseAndLiquidateAccount not implemented")
}
func (UnimplementedCoreInvestServiceServer) ExecuteGDPRErasure(context.Context, *ExecuteGDPRErasureRequest) (*ExecuteGDPRErasureResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecuteGDPRErasure not implemented")
}
func (UnimplementedCoreInvestServiceServer) GetGDPRErasureCertificate(context.Context, *GetGDPRErasureCertificateRequest) (*GetGDPRErasureCertificateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGDPRErasureCertificate not implemented")
}
func (UnimplementedCoreInvestServiceServer) CreateAPIKey(context.Context, *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateAPIKey not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListAPIKeys(context.Context, *ListAPIKeysRequest) (*ListAPIKeysResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListAPIKeys not implemented")
}
func (UnimplementedCoreInvestServiceServer) RevokeAPIKey(context.Context, *RevokeAPIKeyRequest) (*RevokeAPIKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RevokeAPIKey not implemented")
}
func (UnimplementedCoreInvestServiceServer) CreateWebhook(context.Context, *CreateWebhookRequest) (*CreateWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateWebhook not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListWebhooks(context.Context, *ListWebhooksRequest) (*ListWebhooksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListWebhooks not implemented")
}
func (UnimplementedCoreInvestServiceServer) DeleteWebhook(context.Context, *DeleteWebhookRequest) (*DeleteWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteWebhook not implemented")
}
func (UnimplementedCoreInvestServiceServer) CreateOrder(context.Context, *CreateOrderRequest) (*CreateOrderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateOrder not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListOrders(context.Context, *ListOrdersRequest) (*ListOrdersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrders not implemented")
}
func (UnimplementedCoreInvestServiceServer) RegisterCameraStream(context.Context, *RegisterCameraStreamRequest) (*RegisterCameraStreamResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterCameraStream not implemented")
}
func (UnimplementedCoreInvestServiceServer) GetCameraStream(context.Context, *GetCameraStreamRequest) (*GetCameraStreamResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCameraStream not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListCameraStreams(context.Context, *ListCameraStreamsRequest) (*ListCameraStreamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListCameraStreams not implemented")
}
func (UnimplementedCoreInvestServiceServer) IngestObservation(context.Context, *IngestObservationRequest) (*IngestObservationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IngestObservation not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListObservations(context.Context, *ListObservationsRequest) (*ListObservationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListObservations not implemented")
}
func (UnimplementedCoreInvestServiceServer) StartBacktestRun(context.Context, *StartBacktestRunRequest) (*StartBacktestRunResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartBacktestRun not implemented")
}
func (UnimplementedCoreInvestServiceServer) GetBacktestRun(context.Context, *GetBacktestRunRequest) (*GetBacktestRunResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBacktestRun not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListBacktestRuns(context.Context, *ListBacktestRunsRequest) (*ListBacktestRunsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBacktestRuns not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListMLModels(context.Context, *ListMLModelsRequest) (*ListMLModelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListMLModels not implemented")
}
func (UnimplementedCoreInvestServiceServer) LogModelDrift(context.Context, *LogModelDriftRequest) (*LogModelDriftResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LogModelDrift not implemented")
}
func (UnimplementedCoreInvestServiceServer) CreateRecurringDeposit(context.Context, *CreateRecurringDepositRequest) (*CreateRecurringDepositResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRecurringDeposit not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListRecurringDeposits(context.Context, *ListRecurringDepositsRequest) (*ListRecurringDepositsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRecurringDeposits not implemented")
}
func (UnimplementedCoreInvestServiceServer) AdvanceRecurringDeposit(context.Context, *AdvanceRecurringDepositRequest) (*AdvanceRecurringDepositResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AdvanceRecurringDeposit not implemented")
}
func (UnimplementedCoreInvestServiceServer) ClaimDueRecurringDeposits(context.Context, *ClaimDueRecurringDepositsRequest) (*ClaimDueRecurringDepositsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClaimDueRecurringDeposits not implemented")
}
func (UnimplementedCoreInvestServiceServer) GenerateStatement(context.Context, *GenerateStatementRequest) (*GenerateStatementResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateStatement not implemented")
}
func (UnimplementedCoreInvestServiceServer) ListStatements(context.Context, *ListStatementsRequest) (*ListStatementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStatements not implemented")
}
func (UnimplementedCoreInvestServiceServer) GetStatement(context.Context, *GetStatementRequest) (*GetStatementResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatement not implemented")
}
func (UnimplementedCoreInvestServiceServer) GenerateMonthlyStatement(context.Context, *GenerateMonthlyStatementRequest) (*GenerateMonthlyStatementResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateMonthlyStatement not implemented")
}
func (UnimplementedCoreInvestServiceServer) GenerateForm1099B(context.Context, *GenerateForm1099BRequest) (*GenerateForm1099BResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateForm1099B not implemented")
}
func (UnimplementedCoreInvestServiceServer) GetStatementDocument(context.Context, *GetStatementDocumentRequest) (*GetStatementDocumentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatementDocument not implemented")
}
func (UnimplementedCoreInvestServiceServer) StreamPortfolioValuation(*StreamPortfolioValuationRequest, CoreInvestService_StreamPortfolioValuationServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamPortfolioValuation not implemented")
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
		{MethodName: "GetPortfolio", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GetPortfolio", CoreInvestServiceServer.GetPortfolio)
		}},
		{MethodName: "ListPortfolios", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListPortfolios", CoreInvestServiceServer.ListPortfolios)
		}},
		{MethodName: "CreatePortfolio", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/CreatePortfolio", CoreInvestServiceServer.CreatePortfolio)
		}},
		{MethodName: "UpdatePortfolio", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/UpdatePortfolio", CoreInvestServiceServer.UpdatePortfolio)
		}},
		{MethodName: "ListHoldings", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListHoldings", CoreInvestServiceServer.ListHoldings)
		}},
		{MethodName: "UpsertHolding", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/UpsertHolding", CoreInvestServiceServer.UpsertHolding)
		}},
		{MethodName: "UpdatePortfolioBalance", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/UpdatePortfolioBalance", CoreInvestServiceServer.UpdatePortfolioBalance)
		}},
		{MethodName: "RegisterCustomer", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/RegisterCustomer", CoreInvestServiceServer.RegisterCustomer)
		}},
		{MethodName: "CloseAndLiquidateAccount", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/CloseAndLiquidateAccount", CoreInvestServiceServer.CloseAndLiquidateAccount)
		}},
		{MethodName: "ExecuteGDPRErasure", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ExecuteGDPRErasure", CoreInvestServiceServer.ExecuteGDPRErasure)
		}},
		{MethodName: "GetGDPRErasureCertificate", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GetGDPRErasureCertificate", CoreInvestServiceServer.GetGDPRErasureCertificate)
		}},
		{MethodName: "CreateAPIKey", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/CreateAPIKey", CoreInvestServiceServer.CreateAPIKey)
		}},
		{MethodName: "ListAPIKeys", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListAPIKeys", CoreInvestServiceServer.ListAPIKeys)
		}},
		{MethodName: "RevokeAPIKey", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/RevokeAPIKey", CoreInvestServiceServer.RevokeAPIKey)
		}},
		{MethodName: "CreateWebhook", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/CreateWebhook", CoreInvestServiceServer.CreateWebhook)
		}},
		{MethodName: "ListWebhooks", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListWebhooks", CoreInvestServiceServer.ListWebhooks)
		}},
		{MethodName: "DeleteWebhook", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/DeleteWebhook", CoreInvestServiceServer.DeleteWebhook)
		}},
		{MethodName: "CreateOrder", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/CreateOrder", CoreInvestServiceServer.CreateOrder)
		}},
		{MethodName: "ListOrders", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListOrders", CoreInvestServiceServer.ListOrders)
		}},
		{MethodName: "RegisterCameraStream", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/RegisterCameraStream", CoreInvestServiceServer.RegisterCameraStream)
		}},
		{MethodName: "GetCameraStream", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GetCameraStream", CoreInvestServiceServer.GetCameraStream)
		}},
		{MethodName: "ListCameraStreams", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListCameraStreams", CoreInvestServiceServer.ListCameraStreams)
		}},
		{MethodName: "IngestObservation", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/IngestObservation", CoreInvestServiceServer.IngestObservation)
		}},
		{MethodName: "ListObservations", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListObservations", CoreInvestServiceServer.ListObservations)
		}},
		{MethodName: "StartBacktestRun", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/StartBacktestRun", CoreInvestServiceServer.StartBacktestRun)
		}},
		{MethodName: "GetBacktestRun", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GetBacktestRun", CoreInvestServiceServer.GetBacktestRun)
		}},
		{MethodName: "ListBacktestRuns", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListBacktestRuns", CoreInvestServiceServer.ListBacktestRuns)
		}},
		{MethodName: "ListMLModels", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListMLModels", CoreInvestServiceServer.ListMLModels)
		}},
		{MethodName: "LogModelDrift", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/LogModelDrift", CoreInvestServiceServer.LogModelDrift)
		}},
		{MethodName: "CreateRecurringDeposit", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/CreateRecurringDeposit", CoreInvestServiceServer.CreateRecurringDeposit)
		}},
		{MethodName: "ListRecurringDeposits", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListRecurringDeposits", CoreInvestServiceServer.ListRecurringDeposits)
		}},
		{MethodName: "AdvanceRecurringDeposit", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/AdvanceRecurringDeposit", CoreInvestServiceServer.AdvanceRecurringDeposit)
		}},
		{MethodName: "ClaimDueRecurringDeposits", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ClaimDueRecurringDeposits", CoreInvestServiceServer.ClaimDueRecurringDeposits)
		}},
		{MethodName: "GenerateStatement", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GenerateStatement", CoreInvestServiceServer.GenerateStatement)
		}},
		{MethodName: "ListStatements", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/ListStatements", CoreInvestServiceServer.ListStatements)
		}},
		{MethodName: "GetStatement", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GetStatement", CoreInvestServiceServer.GetStatement)
		}},
		{MethodName: "GenerateMonthlyStatement", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GenerateMonthlyStatement", CoreInvestServiceServer.GenerateMonthlyStatement)
		}},
		{MethodName: "GenerateForm1099B", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GenerateForm1099B", CoreInvestServiceServer.GenerateForm1099B)
		}},
		{MethodName: "GetStatementDocument", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.core_invest.v1.CoreInvestService/GetStatementDocument", CoreInvestServiceServer.GetStatementDocument)
		}},
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
