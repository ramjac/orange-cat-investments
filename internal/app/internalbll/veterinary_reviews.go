package internalbll

import (
	"context"

	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.GetCareSchedule(ctx, in)
}

func (s *Server) UpdateCareSchedule(ctx context.Context, in *workforcev1.UpdateCareScheduleRequest) (*workforcev1.UpdateCareScheduleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.UpdateCareSchedule(ctx, in)
}

func (s *Server) SetMedicalHold(ctx context.Context, in *workforcev1.SetMedicalHoldRequest) (*workforcev1.SetMedicalHoldResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.SetMedicalHold(ctx, in)
}

func (s *Server) CreateVHRRecord(ctx context.Context, in *workforcev1.CreateVHRRecordRequest) (*workforcev1.CreateVHRRecordResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.CreateVHRRecord(ctx, in)
}

func (s *Server) ListVHRRecords(ctx context.Context, in *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error) {
	if s.wfClient == nil {
		return &workforcev1.ListVHRRecordsResponse{}, nil
	}
	return s.wfClient.ListVHRRecords(ctx, in)
}

func (s *Server) SubmitLeaveRequest(ctx context.Context, in *workforcev1.SubmitLeaveRequestMessage) (*workforcev1.SubmitLeaveResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.SubmitLeaveRequest(ctx, in)
}

func (s *Server) GetLeaveRequest(ctx context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.GetLeaveRequest(ctx, in)
}

func (s *Server) ListLeaveRequests(ctx context.Context, in *workforcev1.ListLeaveRequestsRequest) (*workforcev1.ListLeaveRequestsResponse, error) {
	if s.wfClient == nil {
		return &workforcev1.ListLeaveRequestsResponse{}, nil
	}
	return s.wfClient.ListLeaveRequests(ctx, in)
}

func (s *Server) UpdateLeaveStatus(ctx context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.UpdateLeaveStatus(ctx, in)
}

func (s *Server) CreateReviewCycle(ctx context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.CreateReviewCycle(ctx, in)
}

func (s *Server) GetReviewCycle(ctx context.Context, in *workforcev1.GetReviewCycleRequest) (*workforcev1.GetReviewCycleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.GetReviewCycle(ctx, in)
}

func (s *Server) ListReviewCycles(ctx context.Context, in *workforcev1.ListReviewCyclesRequest) (*workforcev1.ListReviewCyclesResponse, error) {
	if s.wfClient == nil {
		return &workforcev1.ListReviewCyclesResponse{}, nil
	}
	return s.wfClient.ListReviewCycles(ctx, in)
}

func (s *Server) UpdateReviewCycle(ctx context.Context, in *workforcev1.UpdateReviewCycleRequest) (*workforcev1.UpdateReviewCycleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.UpdateReviewCycle(ctx, in)
}

func (s *Server) SubmitSelfAppraisal(ctx context.Context, in *workforcev1.SubmitSelfAppraisalRequest) (*workforcev1.SubmitSelfAppraisalResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.SubmitSelfAppraisal(ctx, in)
}

func (s *Server) SubmitPeerFeedback(ctx context.Context, in *workforcev1.SubmitPeerFeedbackRequest) (*workforcev1.SubmitPeerFeedbackResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.SubmitPeerFeedback(ctx, in)
}

func (s *Server) SubmitManagerAssessment(ctx context.Context, in *workforcev1.SubmitManagerAssessmentRequest) (*workforcev1.SubmitManagerAssessmentResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.SubmitManagerAssessment(ctx, in)
}

func (s *Server) SubmitVeterinaryAudit(ctx context.Context, in *workforcev1.SubmitVeterinaryAuditRequest) (*workforcev1.SubmitVeterinaryAuditResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.SubmitVeterinaryAudit(ctx, in)
}
