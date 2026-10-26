package handler

import (
	"encoding/json"
	"net/http"
	"time"

	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
)

func (h *WorkforceHandler) listReviewCycles(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	status := r.URL.Query().Get("status")
	reviewType := r.URL.Query().Get("review_type")

	list, err := h.wfSvc.ListReviewCycles(r.Context(), employeeID, status, reviewType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *WorkforceHandler) createReviewCycle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmployeeID              string   `json:"employee_id"`
		ReviewType              string   `json:"review_type"`
		ReviewPeriod            string   `json:"review_period,omitempty"`
		Phase                   string   `json:"phase,omitempty"`
		ScheduledFor            string   `json:"scheduled_for"`
		ReviewerID              *string  `json:"reviewer_id,omitempty"`
		Notes                   *string  `json:"notes,omitempty"`
		SelfAppraisal           *string  `json:"self_appraisal,omitempty"`
		SelfScore               *float64 `json:"self_score,omitempty"`
		PeerFeedback            *string  `json:"peer_feedback,omitempty"`
		PeerScore               *float64 `json:"peer_score,omitempty"`
		PeerReviewerID          *string  `json:"peer_reviewer_id,omitempty"`
		ManagerAssessment       *string  `json:"manager_assessment,omitempty"`
		ManagerScore            *float64 `json:"manager_score,omitempty"`
		KPIRatings              any      `json:"kpi_ratings,omitempty"`
		WellnessAuditScore      *float64 `json:"wellness_audit_score,omitempty"`
		HabitatRecommendations *string  `json:"habitat_recommendations,omitempty"`
		ClinicalAuditorID       *string  `json:"clinical_auditor_id,omitempty"`
		VHRRecordID             *string  `json:"vhr_record_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body: "+err.Error())
		return
	}

	var kpiStr *string
	if req.KPIRatings != nil {
		switch v := req.KPIRatings.(type) {
		case string:
			kpiStr = &v
		default:
			if b, err := json.Marshal(v); err == nil {
				s := string(b)
				kpiStr = &s
			}
		}
	}

	created, err := h.wfSvc.CreateReviewCycle(r.Context(), wfService.CreateReviewCycleParams{
		EmployeeID:              req.EmployeeID,
		ReviewType:              req.ReviewType,
		ReviewPeriod:            req.ReviewPeriod,
		Phase:                   req.Phase,
		ScheduledFor:            req.ScheduledFor,
		ReviewerID:              req.ReviewerID,
		Notes:                   req.Notes,
		SelfAppraisal:           req.SelfAppraisal,
		SelfScore:               req.SelfScore,
		PeerFeedback:            req.PeerFeedback,
		PeerScore:               req.PeerScore,
		PeerReviewerID:          req.PeerReviewerID,
		ManagerAssessment:       req.ManagerAssessment,
		ManagerScore:            req.ManagerScore,
		KPIRatings:              kpiStr,
		WellnessAuditScore:      req.WellnessAuditScore,
		HabitatRecommendations: req.HabitatRecommendations,
		ClinicalAuditorID:       req.ClinicalAuditorID,
		VHRRecordID:             req.VHRRecordID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkforceHandler) getReviewCycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	rc, err := h.wfSvc.GetReviewCycle(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, rc)
}

func (h *WorkforceHandler) updateReviewCycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		ReviewPeriod            *string    `json:"review_period,omitempty"`
		Phase                   *string    `json:"phase,omitempty"`
		Status                  *string    `json:"status,omitempty"`
		ReviewerID              *string    `json:"reviewer_id,omitempty"`
		Score                   *float64   `json:"score,omitempty"`
		Notes                   *string    `json:"notes,omitempty"`
		SelfAppraisal           *string    `json:"self_appraisal,omitempty"`
		SelfScore               *float64   `json:"self_score,omitempty"`
		PeerFeedback            *string    `json:"peer_feedback,omitempty"`
		PeerScore               *float64   `json:"peer_score,omitempty"`
		PeerReviewerID          *string    `json:"peer_reviewer_id,omitempty"`
		ManagerAssessment       *string    `json:"manager_assessment,omitempty"`
		ManagerScore            *float64   `json:"manager_score,omitempty"`
		KPIRatings              any        `json:"kpi_ratings,omitempty"`
		WellnessAuditScore      *float64   `json:"wellness_audit_score,omitempty"`
		HabitatRecommendations *string    `json:"habitat_recommendations,omitempty"`
		ClinicalAuditorID       *string    `json:"clinical_auditor_id,omitempty"`
		VHRRecordID             *string    `json:"vhr_record_id,omitempty"`
		CompletedAt             *time.Time `json:"completed_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body: "+err.Error())
		return
	}

	var kpiStr *string
	if req.KPIRatings != nil {
		switch v := req.KPIRatings.(type) {
		case string:
			kpiStr = &v
		default:
			if b, err := json.Marshal(v); err == nil {
				s := string(b)
				kpiStr = &s
			}
		}
	}

	updated, err := h.wfSvc.UpdateReviewCycle(r.Context(), wfService.UpdateReviewCycleParams{
		ID:                      id,
		ReviewPeriod:            req.ReviewPeriod,
		Phase:                   req.Phase,
		Status:                  req.Status,
		ReviewerID:              req.ReviewerID,
		Score:                   req.Score,
		Notes:                   req.Notes,
		SelfAppraisal:           req.SelfAppraisal,
		SelfScore:               req.SelfScore,
		PeerFeedback:            req.PeerFeedback,
		PeerScore:               req.PeerScore,
		PeerReviewerID:          req.PeerReviewerID,
		ManagerAssessment:       req.ManagerAssessment,
		ManagerScore:            req.ManagerScore,
		KPIRatings:              kpiStr,
		WellnessAuditScore:      req.WellnessAuditScore,
		HabitatRecommendations: req.HabitatRecommendations,
		ClinicalAuditorID:       req.ClinicalAuditorID,
		VHRRecordID:             req.VHRRecordID,
		CompletedAt:             req.CompletedAt,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkforceHandler) handleReviewSelfAppraisal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		SelfAppraisal string  `json:"self_appraisal"`
		SelfScore     float64 `json:"self_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	updated, err := h.wfSvc.SubmitSelfAppraisal(r.Context(), id, req.SelfAppraisal, req.SelfScore)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkforceHandler) handleReviewPeerFeedback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		PeerReviewerID string  `json:"peer_reviewer_id"`
		PeerFeedback   string  `json:"peer_feedback"`
		PeerScore      float64 `json:"peer_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	updated, err := h.wfSvc.SubmitPeerFeedback(r.Context(), id, req.PeerReviewerID, req.PeerFeedback, req.PeerScore)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkforceHandler) handleReviewManagerAssessment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		ManagerID         string  `json:"manager_id"`
		ManagerAssessment string  `json:"manager_assessment"`
		ManagerScore      float64 `json:"manager_score"`
		KPIRatings        any     `json:"kpi_ratings,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	var kpiStr string
	if req.KPIRatings != nil {
		switch v := req.KPIRatings.(type) {
		case string:
			kpiStr = v
		default:
			if b, err := json.Marshal(v); err == nil {
				kpiStr = string(b)
			}
		}
	}

	updated, err := h.wfSvc.SubmitManagerAssessment(r.Context(), id, req.ManagerID, req.ManagerAssessment, req.ManagerScore, kpiStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkforceHandler) handleReviewVeterinaryAudit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		ClinicalAuditorID       string   `json:"clinical_auditor_id"`
		WellnessAuditScore      float64  `json:"wellness_audit_score"`
		HabitatRecommendations string   `json:"habitat_recommendations"`
		VHRRecordID             *string  `json:"vhr_record_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	updated, err := h.wfSvc.SubmitVeterinaryAudit(r.Context(), id, req.ClinicalAuditorID, req.WellnessAuditScore, req.HabitatRecommendations, req.VHRRecordID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
