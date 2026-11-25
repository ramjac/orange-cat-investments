package internalbll

import (
	"context"

	"github.com/orange-cat-investments/oci/pkg/authz/policy"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	res, err := s.wfClient.GetCareSchedule(ctx, in)
	return res, s.logDomainError("workforce-api", "GetCareSchedule", err)
}

func (s *Server) UpdateCareSchedule(ctx context.Context, in *workforcev1.UpdateCareScheduleRequest) (*workforcev1.UpdateCareScheduleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	res, err := s.wfClient.UpdateCareSchedule(ctx, in)
	if err == nil {
		telemetry.RecordCareScheduleUpdate(ctx)
	}
	return res, s.logDomainError("workforce-api", "UpdateCareSchedule", err)
}

func (s *Server) SetMedicalHold(ctx context.Context, in *workforcev1.SetMedicalHoldRequest) (*workforcev1.SetMedicalHoldResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	if err := s.authorizeWorkforce(ctx, "toggle_medical_hold", policy.WorkforceResource{
		FelineID: in.FelineID,
	}); err != nil {
		return nil, err
	}
	res, err := s.wfClient.SetMedicalHold(ctx, in)
	if err == nil {
		telemetry.RecordEmergencyHoldToggle(ctx)
		s.recordAudit(ctx, "workforce.medical_hold.updated", "care_schedule", in.FelineID, map[string]any{
			"feline_id":              in.FelineID,
			"emergency_medical_hold": in.EmergencyMedicalHold,
		})
	}
	return res, s.logDomainError("workforce-api", "SetMedicalHold", err)
}

func (s *Server) CreateVHRRecord(ctx context.Context, in *workforcev1.CreateVHRRecordRequest) (*workforcev1.CreateVHRRecordResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	res, err := s.wfClient.CreateVHRRecord(ctx, in)
	return res, s.logDomainError("workforce-api", "CreateVHRRecord", err)
}

func (s *Server) ListVHRRecords(ctx context.Context, in *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error) {
	if s.wfClient == nil {
		return &workforcev1.ListVHRRecordsResponse{}, nil
	}
	res, err := s.wfClient.ListVHRRecords(ctx, in)
	return res, s.logDomainError("workforce-api", "ListVHRRecords", err)
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
	// Owner-dependent authorization: the self-approval restriction depends on
	// the leave owner's canonical ID. If ownership cannot be established
	// (lookup failure or missing owner), fail closed instead of evaluating
	// with an empty owner, which would silently disable that restriction.
	lrResp, lrErr := s.wfClient.GetLeaveRequest(ctx, &workforcev1.GetLeaveRequestRequest{LeaveID: in.LeaveID})
	if lrErr != nil {
		return nil, status.Errorf(codes.Unavailable, "unable to verify leave request ownership: %v", lrErr)
	}
	if lrResp == nil || lrResp.LeaveRequest == nil || lrResp.LeaveRequest.EmployeeID == "" {
		return nil, status.Error(codes.Unavailable, "leave request owner unavailable; refusing to authorize status change")
	}
	if err := s.authorizeWorkforce(ctx, "approve_leave", policy.WorkforceResource{
		EmployeeID: lrResp.LeaveRequest.EmployeeID,
		ResourceID: in.LeaveID,
	}); err != nil {
		return nil, err
	}
	res, err := s.wfClient.UpdateLeaveStatus(ctx, in)
	if err == nil {
		s.recordAudit(ctx, "workforce.leave_request.status_updated", "leave_request", in.LeaveID, map[string]any{
			"leave_id": in.LeaveID,
			"status":   in.Status,
		})
	}
	return res, s.logDomainError("workforce-api", "UpdateLeaveStatus", err)
}

func (s *Server) CreateReviewCycle(ctx context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	if protected := protectedSubmissionFields(in.SelfAppraisal, in.HasSelfScore, in.PeerFeedback, in.HasPeerScore, in.PeerReviewerID, in.ManagerAssessment, in.HasManagerScore, in.KPIRatings, in.WellnessAuditScore, in.HasWellnessScore, in.HabitatRecommendations, in.ClinicalAuditorID, in.VHRRecordID); protected {
		return nil, status.Error(codes.InvalidArgument, "review submission fields are managed exclusively by their dedicated submission RPCs (SubmitSelfAppraisal, SubmitPeerFeedback, SubmitManagerAssessment, SubmitVeterinaryAudit)")
	}
	if err := s.authorizeWorkforce(ctx, "manage_review_cycle", policy.WorkforceResource{
		EmployeeID: in.EmployeeID,
	}); err != nil {
		return nil, err
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
	// The general review-cycle RPC must not carry submission payloads:
	// self-appraisals, peer feedback, manager assessments, and veterinary
	// audits are written only through their dedicated RPCs so the
	// field-specific policies (including self-approval restrictions) apply.
	if protected := protectedSubmissionFields(in.SelfAppraisal, in.HasSelfScore, in.PeerFeedback, in.HasPeerScore, in.PeerReviewerID, in.ManagerAssessment, in.HasManagerScore, in.KPIRatings, in.WellnessAuditScore, in.HasWellnessScore, in.HabitatRecommendations, in.ClinicalAuditorID, in.VHRRecordID); protected {
		return nil, status.Error(codes.InvalidArgument, "review submission fields are managed exclusively by their dedicated submission RPCs (SubmitSelfAppraisal, SubmitPeerFeedback, SubmitManagerAssessment, SubmitVeterinaryAudit)")
	}
	if err := s.authorizeWorkforce(ctx, "manage_review_cycle", policy.WorkforceResource{
		ResourceID: in.ReviewID,
	}); err != nil {
		return nil, err
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
	// Owner-dependent authorization: the self-assessment restriction depends
	// on the reviewed employee's canonical ID. If ownership cannot be
	// established (lookup failure or missing owner), fail closed instead of
	// evaluating with an empty owner, which would silently disable the
	// self-approval restriction.
	rcResp, rcErr := s.wfClient.GetReviewCycle(ctx, &workforcev1.GetReviewCycleRequest{ReviewID: in.ReviewID})
	if rcErr != nil {
		return nil, status.Errorf(codes.Unavailable, "unable to verify review cycle ownership: %v", rcErr)
	}
	if rcResp == nil || rcResp.ReviewCycle == nil || rcResp.ReviewCycle.EmployeeID == "" {
		return nil, status.Error(codes.Unavailable, "review cycle owner unavailable; refusing to authorize manager assessment")
	}
	if err := s.authorizeWorkforce(ctx, "submit_manager_assessment", policy.WorkforceResource{
		EmployeeID: rcResp.ReviewCycle.EmployeeID,
		ResourceID: in.ReviewID,
	}); err != nil {
		return nil, err
	}
	return s.wfClient.SubmitManagerAssessment(ctx, in)
}

func (s *Server) SubmitVeterinaryAudit(ctx context.Context, in *workforcev1.SubmitVeterinaryAuditRequest) (*workforcev1.SubmitVeterinaryAuditResponse, error) {
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	if err := s.authorizeWorkforce(ctx, "submit_vet_audit", policy.WorkforceResource{
		ResourceID: in.ReviewID,
	}); err != nil {
		return nil, err
	}
	return s.wfClient.SubmitVeterinaryAudit(ctx, in)
}

// protectedSubmissionFields reports whether a review-cycle create/update
// request carries any payload that belongs to the dedicated submission flows
// (self-appraisal, peer feedback, manager assessment, veterinary audit).
// Those fields are accepted only by their own RPCs so the field-specific
// authorization policies cannot be bypassed through the general
// manage_review_cycle route.
func protectedSubmissionFields(selfAppraisal string, hasSelfScore bool, peerFeedback string, hasPeerScore bool, peerReviewerID string, managerAssessment string, hasManagerScore bool, kpiRatings string, wellnessAuditScore float64, hasWellnessScore bool, habitatRecommendations string, clinicalAuditorID string, vhrRecordID string) bool {
	return selfAppraisal != "" || hasSelfScore ||
		peerFeedback != "" || hasPeerScore || peerReviewerID != "" ||
		managerAssessment != "" || hasManagerScore || kpiRatings != "" ||
		wellnessAuditScore != 0 || hasWellnessScore ||
		habitatRecommendations != "" || clinicalAuditorID != "" || vhrRecordID != ""
}
