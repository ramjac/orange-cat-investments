package handler

import (
	"encoding/json"
	"net/http"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
)

func (h *WorkforceHandler) createVHRRecord(w http.ResponseWriter, r *http.Request) {
	var rec workforce.VHRRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.wfSvc.CreateVHRRecord(r.Context(), &rec)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkforceHandler) listVHRRecords(w http.ResponseWriter, r *http.Request) {
	felineID := r.URL.Query().Get("feline_id")
	records, err := h.wfSvc.ListVHRRecords(r.Context(), felineID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *WorkforceHandler) createIncident(w http.ResponseWriter, r *http.Request) {
	var inc workforce.WorkplaceIncident
	if err := json.NewDecoder(r.Body).Decode(&inc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.wfSvc.CreateWorkplaceIncident(r.Context(), &inc)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkforceHandler) listIncidents(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	list, err := h.wfSvc.ListWorkplaceIncidents(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
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
	updated, err := h.wfSvc.UpdateWorkplaceIncidentStatus(r.Context(), id, req.Status, req.ResolutionNotes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
