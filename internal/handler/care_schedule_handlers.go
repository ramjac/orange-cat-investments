package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
)

func (h *WorkforceHandler) getCareSchedule(w http.ResponseWriter, r *http.Request) {
	felineID := r.PathValue("felineId")
	if felineID == "" {
		writeError(w, http.StatusBadRequest, "felineId parameter is required")
		return
	}

	canonicalID := felineID
	var felineAlias string
	if emp, err := h.wfSvc.GetEmployee(r.Context(), felineID); err == nil && emp != nil {
		canonicalID = emp.EmployeeID
		if emp.FirstName != "" {
			felineAlias = fmt.Sprintf("emp-feline-%s", strings.ToLower(emp.FirstName))
		}
	}

	if h.cache != nil {
		var cached workforce.CareSchedule
		if ok, _ := h.cache.Get(r.Context(), cachevalkey.KeyCareSchedule(canonicalID), &cached); ok {
			data, err := json.Marshal(cached)
			if err == nil {
				if cachevalkey.HandleConditionalETag(w, r, data) {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
				return
			}
		}
	}

	var startVer int64
	if h.cache != nil {
		_, _ = h.cache.Get(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), &startVer)
	}

	cs, err := h.wfSvc.GetCareSchedule(r.Context(), canonicalID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if h.cache != nil && cs != nil {
		var currVer int64
		_, _ = h.cache.Get(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), &currVer)
		if currVer == startVer {
			_ = h.cache.Set(r.Context(), cachevalkey.KeyCareSchedule(canonicalID), cs, cachevalkey.TTLCareSchedule)
			if felineID != canonicalID {
				_ = h.cache.Set(r.Context(), cachevalkey.KeyCareSchedule(felineID), cs, cachevalkey.TTLCareSchedule)
			}
			if felineAlias != "" {
				_ = h.cache.Set(r.Context(), cachevalkey.KeyCareSchedule(felineAlias), cs, cachevalkey.TTLCareSchedule)
			}
		}
	}

	data, err := json.Marshal(cs)
	if err == nil {
		if cachevalkey.HandleConditionalETag(w, r, data) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}

	writeJSON(w, http.StatusOK, cs)
}

func (h *WorkforceHandler) updateCareSchedule(w http.ResponseWriter, r *http.Request) {
	felineID := r.PathValue("felineId")
	if felineID == "" {
		writeError(w, http.StatusBadRequest, "felineId parameter is required")
		return
	}

	canonicalID := felineID
	var felineAlias string
	if emp, err := h.wfSvc.GetEmployee(r.Context(), felineID); err == nil && emp != nil {
		canonicalID = emp.EmployeeID
		if emp.FirstName != "" {
			felineAlias = fmt.Sprintf("emp-feline-%s", strings.ToLower(emp.FirstName))
		}
	}

	var req struct {
		DietaryPlan          string   `json:"dietary_plan"`
		FeedingTimes         []string `json:"feeding_times"`
		SpecialMedicalNeeds  *string  `json:"special_medical_needs,omitempty"`
		PreferredPerchZone   *string  `json:"preferred_perch_zone,omitempty"`
		EmergencyMedicalHold bool     `json:"emergency_medical_hold"`
		CaretakerID          *string  `json:"caretaker_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body")
		return
	}

	cs := &workforce.CareSchedule{
		FelineID:             canonicalID,
		DietaryPlan:          req.DietaryPlan,
		FeedingTimes:         req.FeedingTimes,
		SpecialMedicalNeeds:  req.SpecialMedicalNeeds,
		PreferredPerchZone:   req.PreferredPerchZone,
		EmergencyMedicalHold: req.EmergencyMedicalHold,
		CaretakerID:          req.CaretakerID,
	}

	updated, err := h.wfSvc.UpdateCareSchedule(r.Context(), cs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.cache != nil {
		var currVer int64
		_, _ = h.cache.Get(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), &currVer)
		_ = h.cache.Set(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), currVer+1, 24*time.Hour)

		keys := []string{cachevalkey.KeyCareSchedule(canonicalID)}
		if felineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineID))
		}
		if updated != nil && updated.FelineID != "" && updated.FelineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(updated.FelineID))
		}
		if felineAlias != "" {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineAlias))
		}
		_ = h.cache.Delete(r.Context(), keys...)
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkforceHandler) toggleEmergencyMedicalHold(w http.ResponseWriter, r *http.Request) {
	felineID := r.PathValue("felineId")
	if felineID == "" {
		writeError(w, http.StatusBadRequest, "felineId parameter is required")
		return
	}

	canonicalID := felineID
	var felineAlias string
	if emp, err := h.wfSvc.GetEmployee(r.Context(), felineID); err == nil && emp != nil {
		canonicalID = emp.EmployeeID
		if emp.FirstName != "" {
			felineAlias = fmt.Sprintf("emp-feline-%s", strings.ToLower(emp.FirstName))
		}
	}

	var req struct {
		EmergencyMedicalHold bool `json:"emergency_medical_hold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	updated, err := h.wfSvc.SetEmergencyMedicalHold(r.Context(), canonicalID, req.EmergencyMedicalHold)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.cache != nil {
		var currVer int64
		_, _ = h.cache.Get(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), &currVer)
		_ = h.cache.Set(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), currVer+1, 24*time.Hour)

		keys := []string{cachevalkey.KeyCareSchedule(canonicalID)}
		if felineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineID))
		}
		if updated != nil && updated.FelineID != "" && updated.FelineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(updated.FelineID))
		}
		if felineAlias != "" {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineAlias))
		}
		_ = h.cache.Delete(r.Context(), keys...)
	}

	writeJSON(w, http.StatusOK, updated)
}
