package workforce

import (
	"context"
	"errors"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
)

type CreateReviewCycleParams struct {
	EmployeeID              string
	ReviewType              string
	ReviewPeriod            string
	Phase                   string
	ScheduledFor            string
	ReviewerID              *string
	Notes                   *string
	SelfAppraisal           *string
	SelfScore               *float64
	PeerFeedback            *string
	PeerScore               *float64
	PeerReviewerID          *string
	ManagerAssessment       *string
	ManagerScore            *float64
	KPIRatings              *string
	WellnessAuditScore      *float64
	HabitatRecommendations *string
	ClinicalAuditorID       *string
	VHRRecordID             *string
}

func (s *workforceService) CreateReviewCycle(ctx context.Context, params CreateReviewCycleParams) (*workforce.ReviewCycle, error) {
	if params.EmployeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	switch params.ReviewType {
	case "performance", "feline_health_assessment":
		// valid
	default:
		return nil, errors.New("invalid review_type: must be performance or feline_health_assessment")
	}

	if params.ScheduledFor == "" {
		return nil, errors.New("scheduled_for date is required")
	}
	const dateLayout = "2006-01-02"
	if _, err := time.Parse(dateLayout, params.ScheduledFor); err != nil {
		return nil, errors.New("scheduled_for must be in YYYY-MM-DD format")
	}

	if params.Phase != "" {
		switch params.Phase {
		case "self_appraisal", "peer_feedback", "manager_review", "completed":
			// valid
		default:
			return nil, errors.New("invalid phase: must be self_appraisal, peer_feedback, manager_review, or completed")
		}
	} else {
		params.Phase = "self_appraisal"
	}

	if params.ReviewPeriod == "" {
		params.ReviewPeriod = "2026-Q3"
	}

	if params.SelfScore != nil && (*params.SelfScore < 0.0 || *params.SelfScore > 5.0) {
		return nil, errors.New("self_score must be between 0.00 and 5.00")
	}
	if params.PeerScore != nil && (*params.PeerScore < 0.0 || *params.PeerScore > 5.0) {
		return nil, errors.New("peer_score must be between 0.00 and 5.00")
	}
	if params.ManagerScore != nil && (*params.ManagerScore < 0.0 || *params.ManagerScore > 5.0) {
		return nil, errors.New("manager_score must be between 0.00 and 5.00")
	}
	if params.WellnessAuditScore != nil && (*params.WellnessAuditScore < 0.0 || *params.WellnessAuditScore > 5.0) {
		return nil, errors.New("wellness_audit_score must be between 0.00 and 5.00")
	}

	rc := &workforce.ReviewCycle{
		EmployeeID:              params.EmployeeID,
		ReviewType:              params.ReviewType,
		ReviewPeriod:            params.ReviewPeriod,
		Phase:                   params.Phase,
		ScheduledFor:            params.ScheduledFor,
		Status:                  "scheduled",
		ReviewerID:              params.ReviewerID,
		Notes:                   params.Notes,
		SelfAppraisal:           params.SelfAppraisal,
		SelfScore:               params.SelfScore,
		PeerFeedback:            params.PeerFeedback,
		PeerScore:               params.PeerScore,
		PeerReviewerID:          params.PeerReviewerID,
		ManagerAssessment:       params.ManagerAssessment,
		ManagerScore:            params.ManagerScore,
		KPIRatings:              params.KPIRatings,
		WellnessAuditScore:      params.WellnessAuditScore,
		HabitatRecommendations: params.HabitatRecommendations,
		ClinicalAuditorID:       params.ClinicalAuditorID,
		VHRRecordID:             params.VHRRecordID,
	}

	return s.repo.CreateReviewCycle(ctx, rc)
}

func (s *workforceService) GetReviewCycle(ctx context.Context, id string) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id cannot be empty")
	}
	return s.repo.GetReviewCycleByID(ctx, id)
}

func (s *workforceService) ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*workforce.ReviewCycle, error) {
	if status != "" {
		switch status {
		case "scheduled", "in_progress", "completed", "overdue":
			// valid
		default:
			return nil, errors.New("invalid status: must be scheduled, in_progress, completed, or overdue")
		}
	}
	if reviewType != "" {
		switch reviewType {
		case "performance", "feline_health_assessment":
			// valid
		default:
			return nil, errors.New("invalid review_type: must be performance or feline_health_assessment")
		}
	}
	return s.repo.ListReviewCycles(ctx, employeeID, status, reviewType)
}

type UpdateReviewCycleParams struct {
	ID                      string
	ReviewPeriod            *string
	Phase                   *string
	Status                  *string
	ReviewerID              *string
	Score                   *float64
	Notes                   *string
	SelfAppraisal           *string
	SelfScore               *float64
	PeerFeedback            *string
	PeerScore               *float64
	PeerReviewerID          *string
	ManagerAssessment       *string
	ManagerScore            *float64
	KPIRatings              *string
	WellnessAuditScore      *float64
	HabitatRecommendations *string
	ClinicalAuditorID       *string
	VHRRecordID             *string
	CompletedAt             *time.Time
}

func (s *workforceService) UpdateReviewCycle(ctx context.Context, params UpdateReviewCycleParams) (*workforce.ReviewCycle, error) {
	if params.ID == "" {
		return nil, errors.New("review cycle id cannot be empty")
	}

	rc, err := s.repo.GetReviewCycleByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Status != nil {
		switch *params.Status {
		case "scheduled", "in_progress", "completed", "overdue":
			rc.Status = *params.Status
		default:
			return nil, errors.New("invalid status: must be scheduled, in_progress, completed, or overdue")
		}
	}

	if params.Phase != nil {
		switch *params.Phase {
		case "self_appraisal", "peer_feedback", "manager_review", "completed":
			rc.Phase = *params.Phase
		default:
			return nil, errors.New("invalid phase: must be self_appraisal, peer_feedback, manager_review, or completed")
		}
	}

	if params.ReviewPeriod != nil {
		rc.ReviewPeriod = *params.ReviewPeriod
	}

	if params.ReviewerID != nil {
		rc.ReviewerID = params.ReviewerID
	}

	if params.Score != nil {
		if *params.Score < 0.0 || *params.Score > 5.0 {
			return nil, errors.New("score must be between 0.00 and 5.00")
		}
		rc.Score = params.Score
	}

	if params.Notes != nil {
		rc.Notes = params.Notes
	}

	if params.SelfAppraisal != nil {
		rc.SelfAppraisal = params.SelfAppraisal
	}
	if params.SelfScore != nil {
		if *params.SelfScore < 0.0 || *params.SelfScore > 5.0 {
			return nil, errors.New("self_score must be between 0.00 and 5.00")
		}
		rc.SelfScore = params.SelfScore
	}

	if params.PeerFeedback != nil {
		rc.PeerFeedback = params.PeerFeedback
	}
	if params.PeerScore != nil {
		if *params.PeerScore < 0.0 || *params.PeerScore > 5.0 {
			return nil, errors.New("peer_score must be between 0.00 and 5.00")
		}
		rc.PeerScore = params.PeerScore
	}
	if params.PeerReviewerID != nil {
		rc.PeerReviewerID = params.PeerReviewerID
	}

	if params.ManagerAssessment != nil {
		rc.ManagerAssessment = params.ManagerAssessment
	}
	if params.ManagerScore != nil {
		if *params.ManagerScore < 0.0 || *params.ManagerScore > 5.0 {
			return nil, errors.New("manager_score must be between 0.00 and 5.00")
		}
		rc.ManagerScore = params.ManagerScore
		if rc.Score == nil {
			rc.Score = params.ManagerScore
		}
	}
	if params.KPIRatings != nil {
		rc.KPIRatings = params.KPIRatings
	}

	if params.WellnessAuditScore != nil {
		if *params.WellnessAuditScore < 0.0 || *params.WellnessAuditScore > 5.0 {
			return nil, errors.New("wellness_audit_score must be between 0.00 and 5.00")
		}
		rc.WellnessAuditScore = params.WellnessAuditScore
		if rc.Score == nil && rc.ReviewType == "feline_health_assessment" {
			rc.Score = params.WellnessAuditScore
		}
	}
	if params.HabitatRecommendations != nil {
		rc.HabitatRecommendations = params.HabitatRecommendations
	}
	if params.ClinicalAuditorID != nil {
		rc.ClinicalAuditorID = params.ClinicalAuditorID
	}
	if params.VHRRecordID != nil {
		rc.VHRRecordID = params.VHRRecordID
	}

	if rc.Status == "completed" || rc.Phase == "completed" {
		if params.CompletedAt != nil {
			rc.CompletedAt = params.CompletedAt
		} else if rc.CompletedAt == nil {
			now := time.Now().UTC()
			rc.CompletedAt = &now
		}
	} else if params.CompletedAt != nil {
		rc.CompletedAt = params.CompletedAt
	}

	return s.repo.UpdateReviewCycle(ctx, rc)
}

func (s *workforceService) SubmitSelfAppraisal(ctx context.Context, id, appraisal string, score float64) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if appraisal == "" {
		return nil, errors.New("self_appraisal cannot be empty")
	}
	if score < 0.0 || score > 5.0 {
		return nil, errors.New("self_score must be between 0.00 and 5.00")
	}
	phase := "peer_feedback"
	status := "in_progress"
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:            id,
		SelfAppraisal: &appraisal,
		SelfScore:     &score,
		Phase:         &phase,
		Status:        &status,
	})
}

func (s *workforceService) SubmitPeerFeedback(ctx context.Context, id, peerReviewerID, feedback string, score float64) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if peerReviewerID == "" {
		return nil, errors.New("peer_reviewer_id is required")
	}
	if feedback == "" {
		return nil, errors.New("peer_feedback cannot be empty")
	}
	if score < 0.0 || score > 5.0 {
		return nil, errors.New("peer_score must be between 0.00 and 5.00")
	}
	phase := "manager_review"
	status := "in_progress"
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:             id,
		PeerReviewerID: &peerReviewerID,
		PeerFeedback:   &feedback,
		PeerScore:      &score,
		Phase:          &phase,
		Status:         &status,
	})
}

func (s *workforceService) SubmitManagerAssessment(ctx context.Context, id, managerID, assessment string, score float64, kpiRatings string) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if managerID == "" {
		return nil, errors.New("manager_id is required")
	}
	if assessment == "" {
		return nil, errors.New("manager_assessment cannot be empty")
	}
	if score < 0.0 || score > 5.0 {
		return nil, errors.New("manager_score must be between 0.00 and 5.00")
	}
	phase := "completed"
	status := "completed"
	now := time.Now().UTC()
	var kpiPtr *string
	if kpiRatings != "" {
		kpiPtr = &kpiRatings
	}
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:                id,
		ReviewerID:        &managerID,
		ManagerAssessment: &assessment,
		ManagerScore:      &score,
		Score:             &score,
		KPIRatings:        kpiPtr,
		Phase:             &phase,
		Status:            &status,
		CompletedAt:       &now,
	})
}

func (s *workforceService) SubmitVeterinaryAudit(ctx context.Context, id, auditorID string, wellnessScore float64, recommendations string, vhrRecordID *string) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if auditorID == "" {
		return nil, errors.New("clinical_auditor_id is required")
	}
	if wellnessScore < 0.0 || wellnessScore > 5.0 {
		return nil, errors.New("wellness_audit_score must be between 0.00 and 5.00")
	}
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:                      id,
		ClinicalAuditorID:       &auditorID,
		WellnessAuditScore:      &wellnessScore,
		HabitatRecommendations: &recommendations,
		VHRRecordID:             vhrRecordID,
	})
}
