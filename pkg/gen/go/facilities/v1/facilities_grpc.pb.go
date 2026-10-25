package facilitiesv1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
)

type facilitiesServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFacilitiesServiceClient(cc grpc.ClientConnInterface) FacilitiesServiceClient {
	return &facilitiesServiceClient{cc: cc}
}

func (c *facilitiesServiceClient) GetAsset(ctx context.Context, in *GetAssetRequest) (*GetAssetResponse, error) {
	out := new(GetAssetResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/GetAsset", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) ListAssets(ctx context.Context, in *ListAssetsRequest) (*ListAssetsResponse, error) {
	out := new(ListAssetsResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/ListAssets", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) CreateAsset(ctx context.Context, in *CreateAssetRequest) (*CreateAssetResponse, error) {
	out := new(CreateAssetResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/CreateAsset", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) CreateMaintenanceLog(ctx context.Context, in *CreateMaintenanceLogRequest) (*CreateMaintenanceLogResponse, error) {
	out := new(CreateMaintenanceLogResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/CreateMaintenanceLog", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) ListMaintenanceLogs(ctx context.Context, in *ListMaintenanceLogsRequest) (*ListMaintenanceLogsResponse, error) {
	out := new(ListMaintenanceLogsResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/ListMaintenanceLogs", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) ListFirmwareReleases(ctx context.Context, in *ListFirmwareReleasesRequest) (*ListFirmwareReleasesResponse, error) {
	out := new(ListFirmwareReleasesResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/ListFirmwareReleases", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) CreateOTAJob(ctx context.Context, in *CreateOTAJobRequest) (*CreateOTAJobResponse, error) {
	out := new(CreateOTAJobResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/CreateOTAJob", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) ListOTAJobs(ctx context.Context, in *ListOTAJobsRequest) (*ListOTAJobsResponse, error) {
	out := new(ListOTAJobsResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/ListOTAJobs", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) IngestEnvironmentalTelemetry(ctx context.Context, in *IngestTelemetryRequest) (*IngestTelemetryResponse, error) {
	out := new(IngestTelemetryResponse)
	err := c.cc.Invoke(ctx, "/oci.facilities.v1.FacilitiesService/IngestEnvironmentalTelemetry", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *facilitiesServiceClient) StreamEnvironmentalTelemetry(ctx context.Context, in *StreamTelemetryRequest) (FacilitiesService_StreamEnvironmentalTelemetryClient, error) {
	stream, err := c.cc.NewStream(ctx, &FacilitiesService_ServiceDesc.Streams[0], "/oci.facilities.v1.FacilitiesService/StreamEnvironmentalTelemetry")
	if err != nil {
		return nil, err
	}
	x := &facilitiesServiceStreamEnvironmentalTelemetryClient{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type facilitiesServiceStreamEnvironmentalTelemetryClient struct {
	grpc.ClientStream
}

func (x *facilitiesServiceStreamEnvironmentalTelemetryClient) Recv() (*TelemetrySample, error) {
	m := new(TelemetrySample)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server registration and method dispatchers

func _FacilitiesService_GetAsset_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(GetAssetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).GetAsset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/GetAsset",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).GetAsset(ctx, req.(*GetAssetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_ListAssets_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListAssetsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).ListAssets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/ListAssets",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).ListAssets(ctx, req.(*ListAssetsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_CreateAsset_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateAssetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).CreateAsset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/CreateAsset",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).CreateAsset(ctx, req.(*CreateAssetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_CreateMaintenanceLog_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateMaintenanceLogRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).CreateMaintenanceLog(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/CreateMaintenanceLog",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).CreateMaintenanceLog(ctx, req.(*CreateMaintenanceLogRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_ListMaintenanceLogs_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListMaintenanceLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).ListMaintenanceLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/ListMaintenanceLogs",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).ListMaintenanceLogs(ctx, req.(*ListMaintenanceLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_ListFirmwareReleases_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListFirmwareReleasesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).ListFirmwareReleases(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/ListFirmwareReleases",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).ListFirmwareReleases(ctx, req.(*ListFirmwareReleasesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_CreateOTAJob_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateOTAJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).CreateOTAJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/CreateOTAJob",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).CreateOTAJob(ctx, req.(*CreateOTAJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_ListOTAJobs_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListOTAJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).ListOTAJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/ListOTAJobs",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).ListOTAJobs(ctx, req.(*ListOTAJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_IngestEnvironmentalTelemetry_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(IngestTelemetryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FacilitiesServiceServer).IngestEnvironmentalTelemetry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.facilities.v1.FacilitiesService/IngestEnvironmentalTelemetry",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(FacilitiesServiceServer).IngestEnvironmentalTelemetry(ctx, req.(*IngestTelemetryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FacilitiesService_StreamEnvironmentalTelemetry_Handler(srv any, stream grpc.ServerStream) error {
	m := new(StreamTelemetryRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(FacilitiesServiceServer).StreamEnvironmentalTelemetry(m, &facilitiesServiceStreamEnvironmentalTelemetryServer{ServerStream: stream})
}

type facilitiesServiceStreamEnvironmentalTelemetryServer struct {
	grpc.ServerStream
}

func (x *facilitiesServiceStreamEnvironmentalTelemetryServer) Send(m *TelemetrySample) error {
	return x.ServerStream.SendMsg(m)
}

var FacilitiesService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "oci.facilities.v1.FacilitiesService",
	HandlerType: (*FacilitiesServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetAsset", Handler: _FacilitiesService_GetAsset_Handler},
		{MethodName: "ListAssets", Handler: _FacilitiesService_ListAssets_Handler},
		{MethodName: "CreateAsset", Handler: _FacilitiesService_CreateAsset_Handler},
		{MethodName: "CreateMaintenanceLog", Handler: _FacilitiesService_CreateMaintenanceLog_Handler},
		{MethodName: "ListMaintenanceLogs", Handler: _FacilitiesService_ListMaintenanceLogs_Handler},
		{MethodName: "ListFirmwareReleases", Handler: _FacilitiesService_ListFirmwareReleases_Handler},
		{MethodName: "CreateOTAJob", Handler: _FacilitiesService_CreateOTAJob_Handler},
		{MethodName: "ListOTAJobs", Handler: _FacilitiesService_ListOTAJobs_Handler},
		{MethodName: "IngestEnvironmentalTelemetry", Handler: _FacilitiesService_IngestEnvironmentalTelemetry_Handler},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamEnvironmentalTelemetry",
			Handler:       _FacilitiesService_StreamEnvironmentalTelemetry_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/oci/facilities/v1/facilities.proto",
}

func RegisterFacilitiesServiceServer(s grpc.ServiceRegistrar, srv FacilitiesServiceServer) {
	s.RegisterService(&FacilitiesService_ServiceDesc, srv)
}
