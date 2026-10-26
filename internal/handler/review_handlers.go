package handler

import (
	"encoding/json"
	"net/http"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (h *WorkforceHandler) listReviewCycles(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	status := r.URL.Query().Get("status")
	reviewType := r.URL.Query().Get("review_type")

	if h.bllClient != nil {
		resp, err := h.bllClient.ListReviewCycles(r.Context(), &workforcev1.ListReviewCyclesRequest{
			EmployeeID: employeeID,
			Status:     status,
			ReviewType: reviewType,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		list := resp.ReviewCycles
		if list == nil {
			list = []*workforcev1.ReviewCycle{}
		}
		writeJSON(w, http.StatusOK, list)
		return
	}

	if h.wfSvc != nil {
		list, err := h.wfSvc.ListReviewCycles(r.Context(), employeeID, status, reviewType)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if list == nil {
			list = []*workforce.ReviewCycle{}
		}
		writeJSON(w, http.StatusOK, list)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
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

	if h.bllClient != nil {
		reviewerID := ""
		if req.ReviewerID != nil {
			reviewerID = *req.ReviewerID
		}
		notes := ""
		if req.Notes != nil {
			notes = *req.Notes
		}
		selfAppraisal := ""
		if req.SelfAppraisal != nil {
			selfAppraisal = *req.SelfAppraisal
		}
		selfScore := 0.0
		hasSelfScore := false
		if req.SelfScore != nil {
			selfScore = *req.SelfScore
			hasSelfScore = true
		}
		peerFeedback := ""
		if req.PeerFeedback != nil {
			peerFeedback = *req.PeerFeedback
		}
		peerScore := 0.0
		if req.PeerScore != nil {
			peerScore = *req.PeerScore
		}
		peerReviewerID := ""
		if req.PeerReviewerID != nil {
			peerReviewerID = *req.PeerReviewerID
		}
		managerAssessment := ""
		if req.ManagerAssessment != nil {
			managerAssessment = *req.ManagerAssessment
		}
		managerScore := 0.0
		if req.ManagerScore != nil {
			managerScore = *req.ManagerScore
		}
		kpis := ""
		if kpiStr != nil {
			kpis = *kpiStr
		}
		wellnessScore := 0.0
		if req.WellnessAuditScore != nil {
			wellnessScore = *req.WellnessAuditScore
		}
		habitatRecs := ""
		if req.HabitatRecommendations != nil {
			habitatRecs = *req.HabitatRecommendations
		}
		clinicalAuditor := ""
		if req.ClinicalAuditorID != nil {
			clinicalAuditor = *req.ClinicalAuditorID
		}
		vhrID := ""
		if req.VHRRecordID != nil {
			vhrID = *req.VHRRecordID
		}

		resp, err := h.bllClient.CreateReviewCycle(r.Context(), &workforcev1.CreateReviewCycleRequest{
			EmployeeID:             req.EmployeeID,
			ReviewType:             req.ReviewType,
			ReviewPeriod:           req.ReviewPeriod,
			ScheduledFor:           req.ScheduledFor,
			Phase:                  req.Phase,
			ReviewerID:             reviewerID,
			Notes:                  notes,
			SelfAppraisal:          selfAppraisal,
			SelfScore:              selfScore,
			HasSelfScore:           hasSelfScore,
			PeerFeedback:           peerFeedback,
			PeerScore:              peerScore,
			PeerReviewerID:         peerReviewerID,
			ManagerAssessment:      managerAssessment,
			ManagerScore:           managerScore,
			KPIRatings:             kpis,
			WellnessAuditScore:     wellnessScore,
			HabitatRecommendations: habitatRecs,
			ClinicalAuditorID:      clinicalAuditor,
			VHRRecordID:            vhrID,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
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
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) getReviewCycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.GetReviewCycle(r.Context(), &workforcev1.GetReviewCycleRequest{ReviewID: id})
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
		rc, err := h.wfSvc.GetReviewCycle(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rc)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) updateReviewCycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		ReviewPeriod            *string  `json:"review_period,omitempty"`
		Phase                   *string  `json:"phase,omitempty"`
		Status                  *string  `json:"status,omitempty"`
		ReviewerID              *string  `json:"reviewer_id,omitempty"`
		Score                   *float64 `json:"score,omitempty"`
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

	if h.bllClient != nil {
		pbReq := &workforcev1.UpdateReviewCycleRequest{ReviewID: id}
		if req.ReviewPeriod != nil {
			pbReq.ReviewPeriod = *req.ReviewPeriod
		}
		if req.Phase != nil {
			pbReq.Phase = *req.Phase
		}
		if req.Status != nil {
			pbReq.Status = *req.Status
		}
		if req.ReviewerID != nil {
			pbReq.ReviewerID = *req.ReviewerID
		}
		if req.Score != nil {
			pbReq.Score = *req.Score
			pbReq.HasScore = true
		}
		if req.Notes != nil {
			pbReq.Notes = *req.Notes
		}
		if req.SelfAppraisal != nil {
			pbReq.SelfAppraisal = *req.SelfAppraisal
		}
		if req.SelfScore != nil {
			pbReq.SelfScore = *req.SelfScore
			pbReq.HasSelfScore = true
		}
		if req.PeerFeedback != nil {
			pbReq.PeerFeedback = *req.PeerFeedback
		}
		if req.PeerScore != nil {
			pbReq.PeerScore = *req.PeerScore
			pbReq.HasPeerScore = true
		}
		if req.PeerReviewerID != nil {
			pbReq.PeerReviewerID = *req.PeerReviewerID
		}
		if req.ManagerAssessment != nil {
			pbReq.ManagerAssessment = *req.ManagerAssessment
		}
		if req.ManagerScore != nil {
			pbReq.ManagerScore = *req.ManagerScore
			pbReq.HasManagerScore = true
		}
		if kpiStr != nil {
			pbReq.KPIRatings = *kpiStr
		}
		if req.WellnessAuditScore != nil {
			pbReq.WellnessAuditScore = *req.WellnessAuditScore
			pbReq.HasWellnessScore = true
		}
		if req.HabitatRecommendations != nil {
			pbReq.HabitatRecommendations = *req.HabitatRecommendations
		}
		if req.ClinicalAuditorID != nil {
			pbReq.ClinicalAuditorID = *req.ClinicalAuditorID
		}
		if req.VHRRecordID != nil {
			pbReq.VHRRecordID = *req.VHRRecordID
		}

		resp, err := h.bllClient.UpdateReviewCycle(r.Context(), pbReq)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
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
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) handleReviewSelfAppraisal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		SelfAppraisal string   `json:"self_appraisal"`
		SelfScore     *float64 `json:"self_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	score := 0.0
	if req.SelfScore != nil {
		score = *req.SelfScore
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.SubmitSelfAppraisal(r.Context(), &workforcev1.SubmitSelfAppraisalRequest{
			ReviewID:  id,
			Appraisal: req.SelfAppraisal,
			Score:     score,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
		updated, err := h.wfSvc.SubmitSelfAppraisal(r.Context(), id, req.SelfAppraisal, score)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) handleReviewPeerFeedback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		PeerReviewerID string   `json:"peer_reviewer_id"`
		PeerFeedback   string   `json:"peer_feedback"`
		PeerScore      *float64 `json:"peer_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	score := 0.0
	if req.PeerScore != nil {
		score = *req.PeerScore
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.SubmitPeerFeedback(r.Context(), &workforcev1.SubmitPeerFeedbackRequest{
			ReviewID:       id,
			PeerReviewerID: req.PeerReviewerID,
			Feedback:       req.PeerFeedback,
			Score:          score,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
		updated, err := h.wfSvc.SubmitPeerFeedback(r.Context(), id, req.PeerReviewerID, req.PeerFeedback, score)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) handleReviewManagerAssessment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		ManagerID         string   `json:"manager_id"`
		ManagerAssessment string   `json:"manager_assessment"`
		ManagerScore      *float64 `json:"manager_score"`
		KPIRatings        any      `json:"kpi_ratings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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

	score := 0.0
	if req.ManagerScore != nil {
		score = *req.ManagerScore
	}
	kpis := ""
	if kpiStr != nil {
		kpis = *kpiStr
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.SubmitManagerAssessment(r.Context(), &workforcev1.SubmitManagerAssessmentRequest{
			ReviewID:   id,
			ManagerID:  req.ManagerID,
			Assessment: req.ManagerAssessment,
			Score:      score,
			KPIRatings: kpis,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
		updated, err := h.wfSvc.SubmitManagerAssessment(r.Context(), id, req.ManagerID, req.ManagerAssessment, score, kpis)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) handleReviewVeterinaryAudit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	var req struct {
		ClinicalAuditorID       *string  `json:"clinical_auditor_id"`
		WellnessAuditScore      *float64 `json:"wellness_audit_score"`
		HabitatRecommendations *string  `json:"habitat_recommendations"`
		VHRRecordID             *string  `json:"vhr_record_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	auditor := ""
	if req.ClinicalAuditorID != nil {
		auditor = *req.ClinicalAuditorID
	}
	wellnessScore := 0.0
	if req.WellnessAuditScore != nil {
		wellnessScore = *req.WellnessAuditScore
	}
	recs := ""
	if req.HabitatRecommendations != nil {
		recs = *req.HabitatRecommendations
	}
	vhrID := ""
	if req.VHRRecordID != nil {
		vhrID = *req.VHRRecordID
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.SubmitVeterinaryAudit(r.Context(), &workforcev1.SubmitVeterinaryAuditRequest{
			ReviewID:        id,
			AuditorID:       auditor,
			WellnessScore:   wellnessScore,
			Recommendations: recs,
			VHRRecordID:     vhrID,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.ReviewCycle)
		return
	}

	if h.wfSvc != nil {
		updated, err := h.wfSvc.SubmitVeterinaryAudit(r.Context(), id, auditor, wellnessScore, recs, req.VHRRecordID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}
