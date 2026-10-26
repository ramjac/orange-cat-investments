package workforcev1

import (
	"context"

	_ "github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type workforceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkforceServiceClient(cc grpc.ClientConnInterface) WorkforceServiceClient {
	return &workforceServiceClient{cc: cc}
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
	fn func(WorkforceServiceServer, context.Context, *Req) (*Resp, error),
) (any, error) {
	in := new(Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return fn(srv.(WorkforceServiceServer), ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: fullMethod}
	handler := func(ctx context.Context, req any) (any, error) {
		return fn(srv.(WorkforceServiceServer), ctx, req.(*Req))
	}
	return interceptor(ctx, in, info, handler)
}

func (c *workforceServiceClient) GetEmployee(ctx context.Context, in *GetEmployeeRequest) (*GetEmployeeResponse, error) {
	return invokeUnary[GetEmployeeRequest, GetEmployeeResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/GetEmployee", in)
}
func (c *workforceServiceClient) ListEmployees(ctx context.Context, in *ListEmployeesRequest) (*ListEmployeesResponse, error) {
	return invokeUnary[ListEmployeesRequest, ListEmployeesResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListEmployees", in)
}
func (c *workforceServiceClient) CreateEmployee(ctx context.Context, in *CreateEmployeeRequest) (*CreateEmployeeResponse, error) {
	return invokeUnary[CreateEmployeeRequest, CreateEmployeeResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/CreateEmployee", in)
}
func (c *workforceServiceClient) OnboardEmployee(ctx context.Context, in *OnboardEmployeeRequest) (*OnboardEmployeeResponse, error) {
	return invokeUnary[OnboardEmployeeRequest, OnboardEmployeeResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/OnboardEmployee", in)
}
func (c *workforceServiceClient) UpdateEmployeeStatus(ctx context.Context, in *UpdateEmployeeStatusRequest) (*UpdateEmployeeStatusResponse, error) {
	return invokeUnary[UpdateEmployeeStatusRequest, UpdateEmployeeStatusResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateEmployeeStatus", in)
}
func (c *workforceServiceClient) ListOnboardingChecklist(ctx context.Context, in *ListOnboardingChecklistRequest) (*ListOnboardingChecklistResponse, error) {
	return invokeUnary[ListOnboardingChecklistRequest, ListOnboardingChecklistResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListOnboardingChecklist", in)
}
func (c *workforceServiceClient) UpdateOnboardingTask(ctx context.Context, in *UpdateOnboardingTaskRequest) (*UpdateOnboardingTaskResponse, error) {
	return invokeUnary[UpdateOnboardingTaskRequest, UpdateOnboardingTaskResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateOnboardingTask", in)
}
func (c *workforceServiceClient) GetCareSchedule(ctx context.Context, in *GetCareScheduleRequest) (*GetCareScheduleResponse, error) {
	return invokeUnary[GetCareScheduleRequest, GetCareScheduleResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/GetCareSchedule", in)
}
func (c *workforceServiceClient) UpdateCareSchedule(ctx context.Context, in *UpdateCareScheduleRequest) (*UpdateCareScheduleResponse, error) {
	return invokeUnary[UpdateCareScheduleRequest, UpdateCareScheduleResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateCareSchedule", in)
}
func (c *workforceServiceClient) SetMedicalHold(ctx context.Context, in *SetMedicalHoldRequest) (*SetMedicalHoldResponse, error) {
	return invokeUnary[SetMedicalHoldRequest, SetMedicalHoldResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/SetMedicalHold", in)
}
func (c *workforceServiceClient) SubmitLeaveRequest(ctx context.Context, in *SubmitLeaveRequestMessage) (*SubmitLeaveResponse, error) {
	return invokeUnary[SubmitLeaveRequestMessage, SubmitLeaveResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/SubmitLeaveRequest", in)
}
func (c *workforceServiceClient) GetLeaveRequest(ctx context.Context, in *GetLeaveRequestRequest) (*GetLeaveRequestResponse, error) {
	return invokeUnary[GetLeaveRequestRequest, GetLeaveRequestResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/GetLeaveRequest", in)
}
func (c *workforceServiceClient) ListLeaveRequests(ctx context.Context, in *ListLeaveRequestsRequest) (*ListLeaveRequestsResponse, error) {
	return invokeUnary[ListLeaveRequestsRequest, ListLeaveRequestsResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListLeaveRequests", in)
}
func (c *workforceServiceClient) UpdateLeaveStatus(ctx context.Context, in *UpdateLeaveStatusRequest) (*UpdateLeaveStatusResponse, error) {
	return invokeUnary[UpdateLeaveStatusRequest, UpdateLeaveStatusResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateLeaveStatus", in)
}
func (c *workforceServiceClient) CreateReviewCycle(ctx context.Context, in *CreateReviewCycleRequest) (*CreateReviewCycleResponse, error) {
	return invokeUnary[CreateReviewCycleRequest, CreateReviewCycleResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/CreateReviewCycle", in)
}
func (c *workforceServiceClient) GetReviewCycle(ctx context.Context, in *GetReviewCycleRequest) (*GetReviewCycleResponse, error) {
	return invokeUnary[GetReviewCycleRequest, GetReviewCycleResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/GetReviewCycle", in)
}
func (c *workforceServiceClient) ListReviewCycles(ctx context.Context, in *ListReviewCyclesRequest) (*ListReviewCyclesResponse, error) {
	return invokeUnary[ListReviewCyclesRequest, ListReviewCyclesResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListReviewCycles", in)
}
func (c *workforceServiceClient) UpdateReviewCycle(ctx context.Context, in *UpdateReviewCycleRequest) (*UpdateReviewCycleResponse, error) {
	return invokeUnary[UpdateReviewCycleRequest, UpdateReviewCycleResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateReviewCycle", in)
}
func (c *workforceServiceClient) SubmitSelfAppraisal(ctx context.Context, in *SubmitSelfAppraisalRequest) (*SubmitSelfAppraisalResponse, error) {
	return invokeUnary[SubmitSelfAppraisalRequest, SubmitSelfAppraisalResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/SubmitSelfAppraisal", in)
}
func (c *workforceServiceClient) SubmitPeerFeedback(ctx context.Context, in *SubmitPeerFeedbackRequest) (*SubmitPeerFeedbackResponse, error) {
	return invokeUnary[SubmitPeerFeedbackRequest, SubmitPeerFeedbackResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/SubmitPeerFeedback", in)
}
func (c *workforceServiceClient) SubmitManagerAssessment(ctx context.Context, in *SubmitManagerAssessmentRequest) (*SubmitManagerAssessmentResponse, error) {
	return invokeUnary[SubmitManagerAssessmentRequest, SubmitManagerAssessmentResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/SubmitManagerAssessment", in)
}
func (c *workforceServiceClient) SubmitVeterinaryAudit(ctx context.Context, in *SubmitVeterinaryAuditRequest) (*SubmitVeterinaryAuditResponse, error) {
	return invokeUnary[SubmitVeterinaryAuditRequest, SubmitVeterinaryAuditResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/SubmitVeterinaryAudit", in)
}
func (c *workforceServiceClient) CreateVHRRecord(ctx context.Context, in *CreateVHRRecordRequest) (*CreateVHRRecordResponse, error) {
	return invokeUnary[CreateVHRRecordRequest, CreateVHRRecordResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/CreateVHRRecord", in)
}
func (c *workforceServiceClient) ListVHRRecords(ctx context.Context, in *ListVHRRecordsRequest) (*ListVHRRecordsResponse, error) {
	return invokeUnary[ListVHRRecordsRequest, ListVHRRecordsResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListVHRRecords", in)
}
func (c *workforceServiceClient) CreateIncident(ctx context.Context, in *CreateIncidentRequest) (*CreateIncidentResponse, error) {
	return invokeUnary[CreateIncidentRequest, CreateIncidentResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/CreateIncident", in)
}
func (c *workforceServiceClient) ListIncidents(ctx context.Context, in *ListIncidentsRequest) (*ListIncidentsResponse, error) {
	return invokeUnary[ListIncidentsRequest, ListIncidentsResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListIncidents", in)
}
func (c *workforceServiceClient) UpdateIncidentStatus(ctx context.Context, in *UpdateIncidentStatusRequest) (*UpdateIncidentStatusResponse, error) {
	return invokeUnary[UpdateIncidentStatusRequest, UpdateIncidentStatusResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateIncidentStatus", in)
}
func (c *workforceServiceClient) CreateExecutiveReport(ctx context.Context, in *CreateExecutiveReportRequest) (*CreateExecutiveReportResponse, error) {
	return invokeUnary[CreateExecutiveReportRequest, CreateExecutiveReportResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/CreateExecutiveReport", in)
}
func (c *workforceServiceClient) GetExecutiveReport(ctx context.Context, in *GetExecutiveReportRequest) (*GetExecutiveReportResponse, error) {
	return invokeUnary[GetExecutiveReportRequest, GetExecutiveReportResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/GetExecutiveReport", in)
}
func (c *workforceServiceClient) GetExecutiveReportByPeriod(ctx context.Context, in *GetExecutiveReportByPeriodRequest) (*GetExecutiveReportByPeriodResponse, error) {
	return invokeUnary[GetExecutiveReportByPeriodRequest, GetExecutiveReportByPeriodResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/GetExecutiveReportByPeriod", in)
}
func (c *workforceServiceClient) ListExecutiveReports(ctx context.Context, in *ListExecutiveReportsRequest) (*ListExecutiveReportsResponse, error) {
	return invokeUnary[ListExecutiveReportsRequest, ListExecutiveReportsResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/ListExecutiveReports", in)
}
func (c *workforceServiceClient) UpdateExecutiveReportStatus(ctx context.Context, in *UpdateExecutiveReportStatusRequest) (*UpdateExecutiveReportStatusResponse, error) {
	return invokeUnary[UpdateExecutiveReportStatusRequest, UpdateExecutiveReportStatusResponse](ctx, c.cc, "/oci.workforce.v1.WorkforceService/UpdateExecutiveReportStatus", in)
}

type UnimplementedWorkforceServiceServer struct{}

func (UnimplementedWorkforceServiceServer) GetEmployee(context.Context, *GetEmployeeRequest) (*GetEmployeeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEmployee not implemented")
}
func (UnimplementedWorkforceServiceServer) ListEmployees(context.Context, *ListEmployeesRequest) (*ListEmployeesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListEmployees not implemented")
}
func (UnimplementedWorkforceServiceServer) CreateEmployee(context.Context, *CreateEmployeeRequest) (*CreateEmployeeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateEmployee not implemented")
}
func (UnimplementedWorkforceServiceServer) OnboardEmployee(context.Context, *OnboardEmployeeRequest) (*OnboardEmployeeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnboardEmployee not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateEmployeeStatus(context.Context, *UpdateEmployeeStatusRequest) (*UpdateEmployeeStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateEmployeeStatus not implemented")
}
func (UnimplementedWorkforceServiceServer) ListOnboardingChecklist(context.Context, *ListOnboardingChecklistRequest) (*ListOnboardingChecklistResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOnboardingChecklist not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateOnboardingTask(context.Context, *UpdateOnboardingTaskRequest) (*UpdateOnboardingTaskResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOnboardingTask not implemented")
}
func (UnimplementedWorkforceServiceServer) GetCareSchedule(context.Context, *GetCareScheduleRequest) (*GetCareScheduleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCareSchedule not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateCareSchedule(context.Context, *UpdateCareScheduleRequest) (*UpdateCareScheduleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateCareSchedule not implemented")
}
func (UnimplementedWorkforceServiceServer) SetMedicalHold(context.Context, *SetMedicalHoldRequest) (*SetMedicalHoldResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetMedicalHold not implemented")
}
func (UnimplementedWorkforceServiceServer) SubmitLeaveRequest(context.Context, *SubmitLeaveRequestMessage) (*SubmitLeaveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitLeaveRequest not implemented")
}
func (UnimplementedWorkforceServiceServer) GetLeaveRequest(context.Context, *GetLeaveRequestRequest) (*GetLeaveRequestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLeaveRequest not implemented")
}
func (UnimplementedWorkforceServiceServer) ListLeaveRequests(context.Context, *ListLeaveRequestsRequest) (*ListLeaveRequestsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListLeaveRequests not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateLeaveStatus(context.Context, *UpdateLeaveStatusRequest) (*UpdateLeaveStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateLeaveStatus not implemented")
}
func (UnimplementedWorkforceServiceServer) CreateReviewCycle(context.Context, *CreateReviewCycleRequest) (*CreateReviewCycleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateReviewCycle not implemented")
}
func (UnimplementedWorkforceServiceServer) GetReviewCycle(context.Context, *GetReviewCycleRequest) (*GetReviewCycleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReviewCycle not implemented")
}
func (UnimplementedWorkforceServiceServer) ListReviewCycles(context.Context, *ListReviewCyclesRequest) (*ListReviewCyclesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListReviewCycles not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateReviewCycle(context.Context, *UpdateReviewCycleRequest) (*UpdateReviewCycleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateReviewCycle not implemented")
}
func (UnimplementedWorkforceServiceServer) SubmitSelfAppraisal(context.Context, *SubmitSelfAppraisalRequest) (*SubmitSelfAppraisalResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitSelfAppraisal not implemented")
}
func (UnimplementedWorkforceServiceServer) SubmitPeerFeedback(context.Context, *SubmitPeerFeedbackRequest) (*SubmitPeerFeedbackResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitPeerFeedback not implemented")
}
func (UnimplementedWorkforceServiceServer) SubmitManagerAssessment(context.Context, *SubmitManagerAssessmentRequest) (*SubmitManagerAssessmentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitManagerAssessment not implemented")
}
func (UnimplementedWorkforceServiceServer) SubmitVeterinaryAudit(context.Context, *SubmitVeterinaryAuditRequest) (*SubmitVeterinaryAuditResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitVeterinaryAudit not implemented")
}
func (UnimplementedWorkforceServiceServer) CreateVHRRecord(context.Context, *CreateVHRRecordRequest) (*CreateVHRRecordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateVHRRecord not implemented")
}
func (UnimplementedWorkforceServiceServer) ListVHRRecords(context.Context, *ListVHRRecordsRequest) (*ListVHRRecordsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListVHRRecords not implemented")
}
func (UnimplementedWorkforceServiceServer) CreateIncident(context.Context, *CreateIncidentRequest) (*CreateIncidentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateIncident not implemented")
}
func (UnimplementedWorkforceServiceServer) ListIncidents(context.Context, *ListIncidentsRequest) (*ListIncidentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListIncidents not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateIncidentStatus(context.Context, *UpdateIncidentStatusRequest) (*UpdateIncidentStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateIncidentStatus not implemented")
}
func (UnimplementedWorkforceServiceServer) CreateExecutiveReport(context.Context, *CreateExecutiveReportRequest) (*CreateExecutiveReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateExecutiveReport not implemented")
}
func (UnimplementedWorkforceServiceServer) GetExecutiveReport(context.Context, *GetExecutiveReportRequest) (*GetExecutiveReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetExecutiveReport not implemented")
}
func (UnimplementedWorkforceServiceServer) GetExecutiveReportByPeriod(context.Context, *GetExecutiveReportByPeriodRequest) (*GetExecutiveReportByPeriodResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetExecutiveReportByPeriod not implemented")
}
func (UnimplementedWorkforceServiceServer) ListExecutiveReports(context.Context, *ListExecutiveReportsRequest) (*ListExecutiveReportsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListExecutiveReports not implemented")
}
func (UnimplementedWorkforceServiceServer) UpdateExecutiveReportStatus(context.Context, *UpdateExecutiveReportStatusRequest) (*UpdateExecutiveReportStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateExecutiveReportStatus not implemented")
}

var WorkforceService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "oci.workforce.v1.WorkforceService",
	HandlerType: (*WorkforceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetEmployee", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/GetEmployee", WorkforceServiceServer.GetEmployee)
		}},
		{MethodName: "ListEmployees", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListEmployees", WorkforceServiceServer.ListEmployees)
		}},
		{MethodName: "CreateEmployee", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/CreateEmployee", WorkforceServiceServer.CreateEmployee)
		}},
		{MethodName: "OnboardEmployee", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/OnboardEmployee", WorkforceServiceServer.OnboardEmployee)
		}},
		{MethodName: "UpdateEmployeeStatus", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateEmployeeStatus", WorkforceServiceServer.UpdateEmployeeStatus)
		}},
		{MethodName: "ListOnboardingChecklist", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListOnboardingChecklist", WorkforceServiceServer.ListOnboardingChecklist)
		}},
		{MethodName: "UpdateOnboardingTask", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateOnboardingTask", WorkforceServiceServer.UpdateOnboardingTask)
		}},
		{MethodName: "GetCareSchedule", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/GetCareSchedule", WorkforceServiceServer.GetCareSchedule)
		}},
		{MethodName: "UpdateCareSchedule", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateCareSchedule", WorkforceServiceServer.UpdateCareSchedule)
		}},
		{MethodName: "SetMedicalHold", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/SetMedicalHold", WorkforceServiceServer.SetMedicalHold)
		}},
		{MethodName: "SubmitLeaveRequest", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/SubmitLeaveRequest", WorkforceServiceServer.SubmitLeaveRequest)
		}},
		{MethodName: "GetLeaveRequest", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/GetLeaveRequest", WorkforceServiceServer.GetLeaveRequest)
		}},
		{MethodName: "ListLeaveRequests", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListLeaveRequests", WorkforceServiceServer.ListLeaveRequests)
		}},
		{MethodName: "UpdateLeaveStatus", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateLeaveStatus", WorkforceServiceServer.UpdateLeaveStatus)
		}},
		{MethodName: "CreateReviewCycle", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/CreateReviewCycle", WorkforceServiceServer.CreateReviewCycle)
		}},
		{MethodName: "GetReviewCycle", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/GetReviewCycle", WorkforceServiceServer.GetReviewCycle)
		}},
		{MethodName: "ListReviewCycles", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListReviewCycles", WorkforceServiceServer.ListReviewCycles)
		}},
		{MethodName: "UpdateReviewCycle", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateReviewCycle", WorkforceServiceServer.UpdateReviewCycle)
		}},
		{MethodName: "SubmitSelfAppraisal", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/SubmitSelfAppraisal", WorkforceServiceServer.SubmitSelfAppraisal)
		}},
		{MethodName: "SubmitPeerFeedback", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/SubmitPeerFeedback", WorkforceServiceServer.SubmitPeerFeedback)
		}},
		{MethodName: "SubmitManagerAssessment", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/SubmitManagerAssessment", WorkforceServiceServer.SubmitManagerAssessment)
		}},
		{MethodName: "SubmitVeterinaryAudit", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/SubmitVeterinaryAudit", WorkforceServiceServer.SubmitVeterinaryAudit)
		}},
		{MethodName: "CreateVHRRecord", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/CreateVHRRecord", WorkforceServiceServer.CreateVHRRecord)
		}},
		{MethodName: "ListVHRRecords", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListVHRRecords", WorkforceServiceServer.ListVHRRecords)
		}},
		{MethodName: "CreateIncident", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/CreateIncident", WorkforceServiceServer.CreateIncident)
		}},
		{MethodName: "ListIncidents", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListIncidents", WorkforceServiceServer.ListIncidents)
		}},
		{MethodName: "UpdateIncidentStatus", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateIncidentStatus", WorkforceServiceServer.UpdateIncidentStatus)
		}},
		{MethodName: "CreateExecutiveReport", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/CreateExecutiveReport", WorkforceServiceServer.CreateExecutiveReport)
		}},
		{MethodName: "GetExecutiveReport", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/GetExecutiveReport", WorkforceServiceServer.GetExecutiveReport)
		}},
		{MethodName: "GetExecutiveReportByPeriod", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/GetExecutiveReportByPeriod", WorkforceServiceServer.GetExecutiveReportByPeriod)
		}},
		{MethodName: "ListExecutiveReports", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/ListExecutiveReports", WorkforceServiceServer.ListExecutiveReports)
		}},
		{MethodName: "UpdateExecutiveReportStatus", Handler: func(srv any, ctx context.Context, dec func(any) error, i grpc.UnaryServerInterceptor) (any, error) {
			return dispatchUnary(srv, ctx, dec, i, "/oci.workforce.v1.WorkforceService/UpdateExecutiveReportStatus", WorkforceServiceServer.UpdateExecutiveReportStatus)
		}},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/oci/workforce/v1/workforce.proto",
}

func RegisterWorkforceServiceServer(s grpc.ServiceRegistrar, srv WorkforceServiceServer) {
	s.RegisterService(&WorkforceService_ServiceDesc, srv)
}
