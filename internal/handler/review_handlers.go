package handler

import (
	"encoding/json"
	"net/http"

	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (h *WorkforceHandler) listReviewCycles(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	status := r.URL.Query().Get("status")
	reviewType := r.URL.Query().Get("review_type")

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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
}

func (h *WorkforceHandler) getReviewCycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "review cycle id parameter is required")
		return
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetReviewCycle(r.Context(), &workforcev1.GetReviewCycleRequest{ReviewID: id})
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp.ReviewCycle)
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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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
}
