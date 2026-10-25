package opsv1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
)

type opsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewOpsServiceClient(cc grpc.ClientConnInterface) OpsServiceClient {
	return &opsServiceClient{cc: cc}
}

func (c *opsServiceClient) CreateTicket(ctx context.Context, in *CreateTicketRequest) (*CreateTicketResponse, error) {
	out := new(CreateTicketResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/CreateTicket", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) GetTicket(ctx context.Context, in *GetTicketRequest) (*GetTicketResponse, error) {
	out := new(GetTicketResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/GetTicket", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) ListTickets(ctx context.Context, in *ListTicketsRequest) (*ListTicketsResponse, error) {
	out := new(ListTicketsResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/ListTickets", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) AcknowledgeTicket(ctx context.Context, in *AcknowledgeTicketRequest) (*AcknowledgeTicketResponse, error) {
	out := new(AcknowledgeTicketResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/AcknowledgeTicket", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) CloseTicket(ctx context.Context, in *CloseTicketRequest) (*CloseTicketResponse, error) {
	out := new(CloseTicketResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/CloseTicket", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) ListAlerts(ctx context.Context, in *ListAlertsRequest) (*ListAlertsResponse, error) {
	out := new(ListAlertsResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/ListAlerts", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) CreateAlert(ctx context.Context, in *CreateAlertRequest) (*CreateAlertResponse, error) {
	out := new(CreateAlertResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/CreateAlert", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *opsServiceClient) AcknowledgeAlert(ctx context.Context, in *AcknowledgeAlertRequest) (*AcknowledgeAlertResponse, error) {
	out := new(AcknowledgeAlertResponse)
	err := c.cc.Invoke(ctx, "/oci.ops.v1.OpsService/AcknowledgeAlert", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server registration and method dispatchers

func _OpsService_CreateTicket_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateTicketRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).CreateTicket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/CreateTicket",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).CreateTicket(ctx, req.(*CreateTicketRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_GetTicket_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(GetTicketRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).GetTicket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/GetTicket",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).GetTicket(ctx, req.(*GetTicketRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_ListTickets_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListTicketsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).ListTickets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/ListTickets",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).ListTickets(ctx, req.(*ListTicketsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_AcknowledgeTicket_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(AcknowledgeTicketRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).AcknowledgeTicket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/AcknowledgeTicket",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).AcknowledgeTicket(ctx, req.(*AcknowledgeTicketRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_CloseTicket_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CloseTicketRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).CloseTicket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/CloseTicket",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).CloseTicket(ctx, req.(*CloseTicketRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_ListAlerts_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListAlertsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).ListAlerts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/ListAlerts",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).ListAlerts(ctx, req.(*ListAlertsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_CreateAlert_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateAlertRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).CreateAlert(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/CreateAlert",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).CreateAlert(ctx, req.(*CreateAlertRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OpsService_AcknowledgeAlert_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(AcknowledgeAlertRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OpsServiceServer).AcknowledgeAlert(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.ops.v1.OpsService/AcknowledgeAlert",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(OpsServiceServer).AcknowledgeAlert(ctx, req.(*AcknowledgeAlertRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var OpsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "oci.ops.v1.OpsService",
	HandlerType: (*OpsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "CreateTicket", Handler: _OpsService_CreateTicket_Handler},
		{MethodName: "GetTicket", Handler: _OpsService_GetTicket_Handler},
		{MethodName: "ListTickets", Handler: _OpsService_ListTickets_Handler},
		{MethodName: "AcknowledgeTicket", Handler: _OpsService_AcknowledgeTicket_Handler},
		{MethodName: "CloseTicket", Handler: _OpsService_CloseTicket_Handler},
		{MethodName: "ListAlerts", Handler: _OpsService_ListAlerts_Handler},
		{MethodName: "CreateAlert", Handler: _OpsService_CreateAlert_Handler},
		{MethodName: "AcknowledgeAlert", Handler: _OpsService_AcknowledgeAlert_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/oci/ops/v1/ops.proto",
}

func RegisterOpsServiceServer(s grpc.ServiceRegistrar, srv OpsServiceServer) {
	s.RegisterService(&OpsService_ServiceDesc, srv)
}
