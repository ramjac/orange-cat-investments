package main

import (
	"context"

	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (s *WorkforceGRPCServer) SubmitLeaveRequest(ctx context.Context, in *workforcev1.SubmitLeaveRequestMessage) (*workforcev1.SubmitLeaveResponse, error) {
	lr, err := s.svc.CreateLeaveRequest(ctx, workforce_svc.CreateLeaveRequestParams{
		EmployeeID: in.EmployeeID,
		LeaveType:  in.LeaveType,
		StartDate:  in.StartDate,
		EndDate:    in.EndDate,
		Reason:     strPtr(in.Reason),
	})
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitLeaveResponse{LeaveRequest: toProtoLeaveRequest(lr)}, nil
}

func (s *WorkforceGRPCServer) GetLeaveRequest(ctx context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	lr, err := s.svc.GetLeaveRequest(ctx, in.LeaveID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetLeaveRequestResponse{LeaveRequest: toProtoLeaveRequest(lr)}, nil
}

func (s *WorkforceGRPCServer) ListLeaveRequests(ctx context.Context, in *workforcev1.ListLeaveRequestsRequest) (*workforcev1.ListLeaveRequestsResponse, error) {
	reqs, err := s.svc.ListLeaveRequests(ctx, in.EmployeeID, in.Status)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.LeaveRequest
	for _, r := range reqs {
		out = append(out, toProtoLeaveRequest(r))
	}
	return &workforcev1.ListLeaveRequestsResponse{LeaveRequests: out}, nil
}

func (s *WorkforceGRPCServer) UpdateLeaveStatus(ctx context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	updated, err := s.svc.UpdateLeaveRequestStatus(ctx, in.LeaveID, in.Status)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateLeaveStatusResponse{LeaveRequest: toProtoLeaveRequest(updated)}, nil
}

func (s *WorkforceGRPCServer) CreateReviewCycle(ctx context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	params := workforce_svc.CreateReviewCycleParams{
		EmployeeID:             in.EmployeeID,
		ReviewType:             in.ReviewType,
		ReviewPeriod:           in.ReviewPeriod,
		Phase:                  in.Phase,
		ScheduledFor:           in.ScheduledFor,
		ReviewerID:             strPtr(in.ReviewerID),
		Notes:                  strPtr(in.Notes),
		SelfAppraisal:          strPtr(in.SelfAppraisal),
		PeerFeedback:           strPtr(in.PeerFeedback),
		PeerReviewerID:         strPtr(in.PeerReviewerID),
		ManagerAssessment:      strPtr(in.ManagerAssessment),
		KPIRatings:             strPtr(in.KPIRatings),
		HabitatRecommendations: strPtr(in.HabitatRecommendations),
		ClinicalAuditorID:      strPtr(in.ClinicalAuditorID),
		VHRRecordID:            strPtr(in.VHRRecordID),
	}
	if in.HasSelfScore {
		v := in.SelfScore
		params.SelfScore = &v
	}
	if in.HasPeerScore {
		v := in.PeerScore
		params.PeerScore = &v
	}
	if in.HasManagerScore {
		v := in.ManagerScore
		params.ManagerScore = &v
	}
	if in.HasWellnessScore {
		v := in.WellnessAuditScore
		params.WellnessAuditScore = &v
	}
	rc, err := s.svc.CreateReviewCycle(ctx, params)
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateReviewCycleResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) GetReviewCycle(ctx context.Context, in *workforcev1.GetReviewCycleRequest) (*workforcev1.GetReviewCycleResponse, error) {
	rc, err := s.svc.GetReviewCycle(ctx, in.ReviewID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetReviewCycleResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) ListReviewCycles(ctx context.Context, in *workforcev1.ListReviewCyclesRequest) (*workforcev1.ListReviewCyclesResponse, error) {
	rcs, err := s.svc.ListReviewCycles(ctx, in.EmployeeID, in.Status, in.ReviewType)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.ReviewCycle
	for _, r := range rcs {
		out = append(out, toProtoReviewCycle(r))
	}
	return &workforcev1.ListReviewCyclesResponse{ReviewCycles: out}, nil
}

func (s *WorkforceGRPCServer) UpdateReviewCycle(ctx context.Context, in *workforcev1.UpdateReviewCycleRequest) (*workforcev1.UpdateReviewCycleResponse, error) {
	params := workforce_svc.UpdateReviewCycleParams{
		ID:                     in.ReviewID,
		ReviewPeriod:           strPtr(in.ReviewPeriod),
		Phase:                  strPtr(in.Phase),
		Status:                 strPtr(in.Status),
		ReviewerID:             strPtr(in.ReviewerID),
		Notes:                  strPtr(in.Notes),
		SelfAppraisal:          strPtr(in.SelfAppraisal),
		PeerFeedback:           strPtr(in.PeerFeedback),
		PeerReviewerID:         strPtr(in.PeerReviewerID),
		ManagerAssessment:      strPtr(in.ManagerAssessment),
		KPIRatings:             strPtr(in.KPIRatings),
		HabitatRecommendations: strPtr(in.HabitatRecommendations),
		ClinicalAuditorID:      strPtr(in.ClinicalAuditorID),
		VHRRecordID:            strPtr(in.VHRRecordID),
	}
	if in.HasScore {
		v := in.Score
		params.Score = &v
	}
	if in.HasSelfScore {
		v := in.SelfScore
		params.SelfScore = &v
	}
	if in.HasPeerScore {
		v := in.PeerScore
		params.PeerScore = &v
	}
	if in.HasManagerScore {
		v := in.ManagerScore
		params.ManagerScore = &v
	}
	if in.HasWellnessScore {
		v := in.WellnessAuditScore
		params.WellnessAuditScore = &v
	}
	rc, err := s.svc.UpdateReviewCycle(ctx, params)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateReviewCycleResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitSelfAppraisal(ctx context.Context, in *workforcev1.SubmitSelfAppraisalRequest) (*workforcev1.SubmitSelfAppraisalResponse, error) {
	rc, err := s.svc.SubmitSelfAppraisal(ctx, in.ReviewID, in.Appraisal, in.Score)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitSelfAppraisalResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitPeerFeedback(ctx context.Context, in *workforcev1.SubmitPeerFeedbackRequest) (*workforcev1.SubmitPeerFeedbackResponse, error) {
	rc, err := s.svc.SubmitPeerFeedback(ctx, in.ReviewID, in.PeerReviewerID, in.Feedback, in.Score)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitPeerFeedbackResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitManagerAssessment(ctx context.Context, in *workforcev1.SubmitManagerAssessmentRequest) (*workforcev1.SubmitManagerAssessmentResponse, error) {
	rc, err := s.svc.SubmitManagerAssessment(ctx, in.ReviewID, in.ManagerID, in.Assessment, in.Score, in.KPIRatings)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitManagerAssessmentResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitVeterinaryAudit(ctx context.Context, in *workforcev1.SubmitVeterinaryAuditRequest) (*workforcev1.SubmitVeterinaryAuditResponse, error) {
	rc, err := s.svc.SubmitVeterinaryAudit(ctx, in.ReviewID, in.AuditorID, in.WellnessScore, in.Recommendations, strPtr(in.VHRRecordID))
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitVeterinaryAuditResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}
