package facilitiesv1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type facilitiesServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFacilitiesServiceClient(cc grpc.ClientConnInterface) FacilitiesServiceClient {
	return &facilitiesServiceClient{cc: cc}
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
	fn func(FacilitiesServiceServer, context.Context, *Req) (*Resp, error),
) (any, error) {
	in := new(Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return fn(srv.(FacilitiesServiceServer), ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: fullMethod}
	handler := func(ctx context.Context, req any) (any, error) {
		return fn(srv.(FacilitiesServiceServer), ctx, req.(*Req))
	}
	return interceptor(ctx, in, info, handler)
}

func (c *facilitiesServiceClient) GetAsset(ctx context.Context, in *GetAssetRequest) (*GetAssetResponse, error) {
	return invokeUnary[GetAssetRequest, GetAssetResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/GetAsset", in)
}
func (c *facilitiesServiceClient) ListAssets(ctx context.Context, in *ListAssetsRequest) (*ListAssetsResponse, error) {
	return invokeUnary[ListAssetsRequest, ListAssetsResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/ListAssets", in)
}
func (c *facilitiesServiceClient) CreateAsset(ctx context.Context, in *CreateAssetRequest) (*CreateAssetResponse, error) {
	return invokeUnary[CreateAssetRequest, CreateAssetResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/CreateAsset", in)
}
func (c *facilitiesServiceClient) CreateMaintenanceLog(ctx context.Context, in *CreateMaintenanceLogRequest) (*CreateMaintenanceLogResponse, error) {
	return invokeUnary[CreateMaintenanceLogRequest, CreateMaintenanceLogResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/CreateMaintenanceLog", in)
}
func (c *facilitiesServiceClient) ListMaintenanceLogs(ctx context.Context, in *ListMaintenanceLogsRequest) (*ListMaintenanceLogsResponse, error) {
	return invokeUnary[ListMaintenanceLogsRequest, ListMaintenanceLogsResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/ListMaintenanceLogs", in)
}
func (c *facilitiesServiceClient) SyncOfflineLogs(ctx context.Context, in *SyncOfflineLogsRequest) (*SyncOfflineLogsResponse, error) {
	return invokeUnary[SyncOfflineLogsRequest, SyncOfflineLogsResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/SyncOfflineLogs", in)
}
func (c *facilitiesServiceClient) CreateFirmwareRelease(ctx context.Context, in *CreateFirmwareReleaseRequest) (*CreateFirmwareReleaseResponse, error) {
	return invokeUnary[CreateFirmwareReleaseRequest, CreateFirmwareReleaseResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/CreateFirmwareRelease", in)
}
func (c *facilitiesServiceClient) ListFirmwareReleases(ctx context.Context, in *ListFirmwareReleasesRequest) (*ListFirmwareReleasesResponse, error) {
	return invokeUnary[ListFirmwareReleasesRequest, ListFirmwareReleasesResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/ListFirmwareReleases", in)
}
func (c *facilitiesServiceClient) CreateOTAJob(ctx context.Context, in *CreateOTAJobRequest) (*CreateOTAJobResponse, error) {
	return invokeUnary[CreateOTAJobRequest, CreateOTAJobResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/CreateOTAJob", in)
}
func (c *facilitiesServiceClient) ListOTAJobs(ctx context.Context, in *ListOTAJobsRequest) (*ListOTAJobsResponse, error) {
	return invokeUnary[ListOTAJobsRequest, ListOTAJobsResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/ListOTAJobs", in)
}
func (c *facilitiesServiceClient) UpdateOTAJobStatus(ctx context.Context, in *UpdateOTAJobStatusRequest) (*UpdateOTAJobStatusResponse, error) {
	return invokeUnary[UpdateOTAJobStatusRequest, UpdateOTAJobStatusResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/UpdateOTAJobStatus", in)
}
func (c *facilitiesServiceClient) IngestEnvironmentalTelemetry(ctx context.Context, in *IngestTelemetryRequest) (*IngestTelemetryResponse, error) {
	return invokeUnary[IngestTelemetryRequest, IngestTelemetryResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/IngestEnvironmentalTelemetry", in)
}
func (c *facilitiesServiceClient) ControlPTZ(ctx context.Context, in *ControlPTZRequest) (*ControlPTZResponse, error) {
	return invokeUnary[ControlPTZRequest, ControlPTZResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/ControlPTZ", in)
}
func (c *facilitiesServiceClient) CalibrateCameraLens(ctx context.Context, in *CalibrateCameraLensRequest) (*CalibrateCameraLensResponse, error) {
	return invokeUnary[CalibrateCameraLensRequest, CalibrateCameraLensResponse](ctx, c.cc, "/oci.facilities.v1.FacilitiesService/CalibrateCameraLens", in)
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

type UnimplementedFacilitiesServiceServer struct{}

func (UnimplementedFacilitiesServiceServer) GetAsset(context.Context, *GetAssetRequest) (*GetAssetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAsset not implemented")
}
func (UnimplementedFacilitiesServiceServer) ListAssets(context.Context, *ListAssetsRequest) (*ListAssetsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListAssets not implemented")
}
func (UnimplementedFacilitiesServiceServer) CreateAsset(context.Context, *CreateAssetRequest) (*CreateAssetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateAsset not implemented")
}
func (UnimplementedFacilitiesServiceServer) CreateMaintenanceLog(context.Context, *CreateMaintenanceLogRequest) (*CreateMaintenanceLogResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateMaintenanceLog not implemented")
}
func (UnimplementedFacilitiesServiceServer) ListMaintenanceLogs(context.Context, *ListMaintenanceLogsRequest) (*ListMaintenanceLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListMaintenanceLogs not implemented")
}
func (UnimplementedFacilitiesServiceServer) SyncOfflineLogs(context.Context, *SyncOfflineLogsRequest) (*SyncOfflineLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncOfflineLogs not implemented")
}
func (UnimplementedFacilitiesServiceServer) CreateFirmwareRelease(context.Context, *CreateFirmwareReleaseRequest) (*CreateFirmwareReleaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateFirmwareRelease not implemented")
}
func (UnimplementedFacilitiesServiceServer) ListFirmwareReleases(context.Context, *ListFirmwareReleasesRequest) (*ListFirmwareReleasesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListFirmwareReleases not implemented")
}
func (UnimplementedFacilitiesServiceServer) CreateOTAJob(context.Context, *CreateOTAJobRequest) (*CreateOTAJobResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateOTAJob not implemented")
}
func (UnimplementedFacilitiesServiceServer) ListOTAJobs(context.Context, *ListOTAJobsRequest) (*ListOTAJobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOTAJobs not implemented")
}
func (UnimplementedFacilitiesServiceServer) UpdateOTAJobStatus(context.Context, *UpdateOTAJobStatusRequest) (*UpdateOTAJobStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOTAJobStatus not implemented")
}
func (UnimplementedFacilitiesServiceServer) IngestEnvironmentalTelemetry(context.Context, *IngestTelemetryRequest) (*IngestTelemetryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IngestEnvironmentalTelemetry not implemented")
}
func (UnimplementedFacilitiesServiceServer) StreamEnvironmentalTelemetry(*StreamTelemetryRequest, FacilitiesService_StreamEnvironmentalTelemetryServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamEnvironmentalTelemetry not implemented")
}
func (UnimplementedFacilitiesServiceServer) ControlPTZ(context.Context, *ControlPTZRequest) (*ControlPTZResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControlPTZ not implemented")
}
func (UnimplementedFacilitiesServiceServer) CalibrateCameraLens(context.Context, *CalibrateCameraLensRequest) (*CalibrateCameraLensResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CalibrateCameraLens not implemented")
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
		{MethodName: "GetAsset", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/GetAsset", FacilitiesServiceServer.GetAsset)
		}},
		{MethodName: "ListAssets", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/ListAssets", FacilitiesServiceServer.ListAssets)
		}},
		{MethodName: "CreateAsset", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/CreateAsset", FacilitiesServiceServer.CreateAsset)
		}},
		{MethodName: "CreateMaintenanceLog", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/CreateMaintenanceLog", FacilitiesServiceServer.CreateMaintenanceLog)
		}},
		{MethodName: "ListMaintenanceLogs", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/ListMaintenanceLogs", FacilitiesServiceServer.ListMaintenanceLogs)
		}},
		{MethodName: "SyncOfflineLogs", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/SyncOfflineLogs", FacilitiesServiceServer.SyncOfflineLogs)
		}},
		{MethodName: "CreateFirmwareRelease", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/CreateFirmwareRelease", FacilitiesServiceServer.CreateFirmwareRelease)
		}},
		{MethodName: "ListFirmwareReleases", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/ListFirmwareReleases", FacilitiesServiceServer.ListFirmwareReleases)
		}},
		{MethodName: "CreateOTAJob", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/CreateOTAJob", FacilitiesServiceServer.CreateOTAJob)
		}},
		{MethodName: "ListOTAJobs", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/ListOTAJobs", FacilitiesServiceServer.ListOTAJobs)
		}},
		{MethodName: "UpdateOTAJobStatus", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/UpdateOTAJobStatus", FacilitiesServiceServer.UpdateOTAJobStatus)
		}},
		{MethodName: "IngestEnvironmentalTelemetry", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/IngestEnvironmentalTelemetry", FacilitiesServiceServer.IngestEnvironmentalTelemetry)
		}},
		{MethodName: "ControlPTZ", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/ControlPTZ", FacilitiesServiceServer.ControlPTZ)
		}},
		{MethodName: "CalibrateCameraLens", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.facilities.v1.FacilitiesService/CalibrateCameraLens", FacilitiesServiceServer.CalibrateCameraLens)
		}},
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
