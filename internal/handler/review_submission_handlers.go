package handler

import (
	"encoding/json"
	"net/http"

	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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
}
