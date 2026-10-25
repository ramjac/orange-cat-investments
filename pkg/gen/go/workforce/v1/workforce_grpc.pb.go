package workforcev1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
)

type workforceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkforceServiceClient(cc grpc.ClientConnInterface) WorkforceServiceClient {
	return &workforceServiceClient{cc: cc}
}

func (c *workforceServiceClient) GetEmployee(ctx context.Context, in *GetEmployeeRequest) (*GetEmployeeResponse, error) {
	out := new(GetEmployeeResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/GetEmployee", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) ListEmployees(ctx context.Context, in *ListEmployeesRequest) (*ListEmployeesResponse, error) {
	out := new(ListEmployeesResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/ListEmployees", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) CreateEmployee(ctx context.Context, in *CreateEmployeeRequest) (*CreateEmployeeResponse, error) {
	out := new(CreateEmployeeResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/CreateEmployee", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) GetCareSchedule(ctx context.Context, in *GetCareScheduleRequest) (*GetCareScheduleResponse, error) {
	out := new(GetCareScheduleResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/GetCareSchedule", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) UpdateCareSchedule(ctx context.Context, in *UpdateCareScheduleRequest) (*UpdateCareScheduleResponse, error) {
	out := new(UpdateCareScheduleResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/UpdateCareSchedule", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) SetMedicalHold(ctx context.Context, in *SetMedicalHoldRequest) (*SetMedicalHoldResponse, error) {
	out := new(SetMedicalHoldResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/SetMedicalHold", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) SubmitLeaveRequest(ctx context.Context, in *SubmitLeaveRequestMessage) (*SubmitLeaveResponse, error) {
	out := new(SubmitLeaveResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/SubmitLeaveRequest", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) ListLeaveRequests(ctx context.Context, in *ListLeaveRequestsRequest) (*ListLeaveRequestsResponse, error) {
	out := new(ListLeaveRequestsResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/ListLeaveRequests", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) UpdateLeaveStatus(ctx context.Context, in *UpdateLeaveStatusRequest) (*UpdateLeaveStatusResponse, error) {
	out := new(UpdateLeaveStatusResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/UpdateLeaveStatus", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) CreateReviewCycle(ctx context.Context, in *CreateReviewCycleRequest) (*CreateReviewCycleResponse, error) {
	out := new(CreateReviewCycleResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/CreateReviewCycle", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) ListReviewCycles(ctx context.Context, in *ListReviewCyclesRequest) (*ListReviewCyclesResponse, error) {
	out := new(ListReviewCyclesResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/ListReviewCycles", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) CreateVHRRecord(ctx context.Context, in *CreateVHRRecordRequest) (*CreateVHRRecordResponse, error) {
	out := new(CreateVHRRecordResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/CreateVHRRecord", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) ListVHRRecords(ctx context.Context, in *ListVHRRecordsRequest) (*ListVHRRecordsResponse, error) {
	out := new(ListVHRRecordsResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/ListVHRRecords", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) CreateIncident(ctx context.Context, in *CreateIncidentRequest) (*CreateIncidentResponse, error) {
	out := new(CreateIncidentResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/CreateIncident", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workforceServiceClient) ListIncidents(ctx context.Context, in *ListIncidentsRequest) (*ListIncidentsResponse, error) {
	out := new(ListIncidentsResponse)
	err := c.cc.Invoke(ctx, "/oci.workforce.v1.WorkforceService/ListIncidents", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server registration and method dispatchers

func _WorkforceService_GetEmployee_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(GetEmployeeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).GetEmployee(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/GetEmployee",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).GetEmployee(ctx, req.(*GetEmployeeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_ListEmployees_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListEmployeesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).ListEmployees(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/ListEmployees",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).ListEmployees(ctx, req.(*ListEmployeesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_CreateEmployee_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateEmployeeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).CreateEmployee(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/CreateEmployee",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).CreateEmployee(ctx, req.(*CreateEmployeeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_GetCareSchedule_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(GetCareScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).GetCareSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/GetCareSchedule",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).GetCareSchedule(ctx, req.(*GetCareScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_UpdateCareSchedule_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(UpdateCareScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).UpdateCareSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/UpdateCareSchedule",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).UpdateCareSchedule(ctx, req.(*UpdateCareScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_SetMedicalHold_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(SetMedicalHoldRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).SetMedicalHold(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/SetMedicalHold",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).SetMedicalHold(ctx, req.(*SetMedicalHoldRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_SubmitLeaveRequest_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(SubmitLeaveRequestMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).SubmitLeaveRequest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/SubmitLeaveRequest",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).SubmitLeaveRequest(ctx, req.(*SubmitLeaveRequestMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_ListLeaveRequests_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListLeaveRequestsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).ListLeaveRequests(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/ListLeaveRequests",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).ListLeaveRequests(ctx, req.(*ListLeaveRequestsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_UpdateLeaveStatus_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(UpdateLeaveStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).UpdateLeaveStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/UpdateLeaveStatus",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).UpdateLeaveStatus(ctx, req.(*UpdateLeaveStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_CreateReviewCycle_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateReviewCycleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).CreateReviewCycle(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/CreateReviewCycle",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).CreateReviewCycle(ctx, req.(*CreateReviewCycleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_ListReviewCycles_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListReviewCyclesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).ListReviewCycles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/ListReviewCycles",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).ListReviewCycles(ctx, req.(*ListReviewCyclesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_CreateVHRRecord_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateVHRRecordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).CreateVHRRecord(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/CreateVHRRecord",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).CreateVHRRecord(ctx, req.(*CreateVHRRecordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_ListVHRRecords_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListVHRRecordsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).ListVHRRecords(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/ListVHRRecords",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).ListVHRRecords(ctx, req.(*ListVHRRecordsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_CreateIncident_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(CreateIncidentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).CreateIncident(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/CreateIncident",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).CreateIncident(ctx, req.(*CreateIncidentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkforceService_ListIncidents_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(ListIncidentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkforceServiceServer).ListIncidents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/oci.workforce.v1.WorkforceService/ListIncidents",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(WorkforceServiceServer).ListIncidents(ctx, req.(*ListIncidentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var WorkforceService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "oci.workforce.v1.WorkforceService",
	HandlerType: (*WorkforceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetEmployee", Handler: _WorkforceService_GetEmployee_Handler},
		{MethodName: "ListEmployees", Handler: _WorkforceService_ListEmployees_Handler},
		{MethodName: "CreateEmployee", Handler: _WorkforceService_CreateEmployee_Handler},
		{MethodName: "GetCareSchedule", Handler: _WorkforceService_GetCareSchedule_Handler},
		{MethodName: "UpdateCareSchedule", Handler: _WorkforceService_UpdateCareSchedule_Handler},
		{MethodName: "SetMedicalHold", Handler: _WorkforceService_SetMedicalHold_Handler},
		{MethodName: "SubmitLeaveRequest", Handler: _WorkforceService_SubmitLeaveRequest_Handler},
		{MethodName: "ListLeaveRequests", Handler: _WorkforceService_ListLeaveRequests_Handler},
		{MethodName: "UpdateLeaveStatus", Handler: _WorkforceService_UpdateLeaveStatus_Handler},
		{MethodName: "CreateReviewCycle", Handler: _WorkforceService_CreateReviewCycle_Handler},
		{MethodName: "ListReviewCycles", Handler: _WorkforceService_ListReviewCycles_Handler},
		{MethodName: "CreateVHRRecord", Handler: _WorkforceService_CreateVHRRecord_Handler},
		{MethodName: "ListVHRRecords", Handler: _WorkforceService_ListVHRRecords_Handler},
		{MethodName: "CreateIncident", Handler: _WorkforceService_CreateIncident_Handler},
		{MethodName: "ListIncidents", Handler: _WorkforceService_ListIncidents_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/oci/workforce/v1/workforce.proto",
}

func RegisterWorkforceServiceServer(s grpc.ServiceRegistrar, srv WorkforceServiceServer) {
	s.RegisterService(&WorkforceService_ServiceDesc, srv)
}
