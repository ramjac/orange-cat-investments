package handler

import (
	"encoding/json"
	"net/http"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (h *WorkforceHandler) createVHRRecord(w http.ResponseWriter, r *http.Request) {
	var rec struct {
		FelineID          string   `json:"feline_id"`
		VeterinarianID    string   `json:"veterinarian_id"`
		VisitDate         string   `json:"visit_date"`
		WeightKG          *float64 `json:"weight_kg"`
		DentalScore       *int32   `json:"dental_score"`
		VaccinationStatus *string  `json:"vaccination_status"`
		Prescriptions     *string  `json:"prescriptions"`
		ClinicalNotes     *string  `json:"clinical_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if h.bllClient != nil {
		weight := 0.0
		if rec.WeightKG != nil {
			weight = *rec.WeightKG
		}
		dental := int32(0)
		if rec.DentalScore != nil {
			dental = *rec.DentalScore
		}
		vax := ""
		if rec.VaccinationStatus != nil {
			vax = *rec.VaccinationStatus
		}
		rx := ""
		if rec.Prescriptions != nil {
			rx = *rec.Prescriptions
		}
		notes := ""
		if rec.ClinicalNotes != nil {
			notes = *rec.ClinicalNotes
		}

		resp, err := h.bllClient.CreateVHRRecord(r.Context(), &workforcev1.CreateVHRRecordRequest{
			FelineID:          rec.FelineID,
			VeterinarianID:    rec.VeterinarianID,
			VisitDate:         rec.VisitDate,
			WeightKG:          weight,
			DentalScore:       dental,
			VaccinationStatus: vax,
			Prescriptions:     rx,
			ClinicalNotes:     notes,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.Record)
		return
	}

	if h.wfSvc != nil {
		var vetID *string
		if rec.VeterinarianID != "" {
			vetID = &rec.VeterinarianID
		}
		weight := 0.0
		if rec.WeightKG != nil {
			weight = *rec.WeightKG
		}
		dental := 0
		if rec.DentalScore != nil {
			dental = int(*rec.DentalScore)
		}
		vax := ""
		if rec.VaccinationStatus != nil {
			vax = *rec.VaccinationStatus
		}
		notes := ""
		if rec.ClinicalNotes != nil {
			notes = *rec.ClinicalNotes
		}

		legacyRec := &workforce.VHRRecord{
			FelineID:          rec.FelineID,
			VeterinarianID:    vetID,
			VisitDate:         rec.VisitDate,
			WeightKG:          weight,
			DentalScore:       dental,
			VaccinationStatus: vax,
			Prescriptions:     rec.Prescriptions,
			ClinicalNotes:     notes,
		}
		created, err := h.wfSvc.CreateVHRRecord(r.Context(), legacyRec)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, created)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) listVHRRecords(w http.ResponseWriter, r *http.Request) {
	felineID := r.URL.Query().Get("feline_id")

	if h.bllClient != nil {
		resp, err := h.bllClient.ListVHRRecords(r.Context(), &workforcev1.ListVHRRecordsRequest{FelineID: felineID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		records := resp.Records
		if records == nil {
			records = []*workforcev1.VHRRecord{}
		}
		writeJSON(w, http.StatusOK, records)
		return
	}

	if h.wfSvc != nil {
		records, err := h.wfSvc.ListVHRRecords(r.Context(), felineID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, records)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) createIncident(w http.ResponseWriter, r *http.Request) {
	var inc struct {
		Title            string  `json:"title"`
		Category         string  `json:"category"`
		InvolvedFelineID *string `json:"involved_feline_id,omitempty"`
		InvolvedHumanID  *string `json:"involved_human_id,omitempty"`
		Severity         string  `json:"severity"`
		Description      string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&inc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if h.bllClient != nil {
		felineID := ""
		if inc.InvolvedFelineID != nil {
			felineID = *inc.InvolvedFelineID
		}
		humanID := ""
		if inc.InvolvedHumanID != nil {
			humanID = *inc.InvolvedHumanID
		}

		resp, err := h.bllClient.CreateIncident(r.Context(), &workforcev1.CreateIncidentRequest{
			Title:            inc.Title,
			Category:         inc.Category,
			InvolvedFelineID: felineID,
			InvolvedHumanID:  humanID,
			Severity:         inc.Severity,
			Description:      inc.Description,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.Incident)
		return
	}

	if h.wfSvc != nil {
		legacyInc := &workforce.WorkplaceIncident{
			Title:            inc.Title,
			Category:         inc.Category,
			InvolvedFelineID: inc.InvolvedFelineID,
			InvolvedHumanID:  inc.InvolvedHumanID,
			Severity:         inc.Severity,
			Description:      inc.Description,
		}
		created, err := h.wfSvc.CreateWorkplaceIncident(r.Context(), legacyInc)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, created)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) listIncidents(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	if h.bllClient != nil {
		resp, err := h.bllClient.ListIncidents(r.Context(), &workforcev1.ListIncidentsRequest{Status: status})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		incidents := resp.Incidents
		if incidents == nil {
			incidents = []*workforcev1.WorkplaceIncident{}
		}
		writeJSON(w, http.StatusOK, incidents)
		return
	}

	if h.wfSvc != nil {
		list, err := h.wfSvc.ListWorkplaceIncidents(r.Context(), status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, list)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func (h *WorkforceHandler) updateIncidentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Status          string `json:"status"`
		ResolutionNotes string `json:"resolution_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.UpdateIncidentStatus(r.Context(), &workforcev1.UpdateIncidentStatusRequest{
			IncidentID:      id,
			Status:          req.Status,
			ResolutionNotes: req.ResolutionNotes,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.Incident)
		return
	}

	if h.wfSvc != nil {
		updated, err := h.wfSvc.UpdateWorkplaceIncidentStatus(r.Context(), id, req.Status, req.ResolutionNotes)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}
