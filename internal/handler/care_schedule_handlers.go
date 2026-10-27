package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (h *WorkforceHandler) resolveFeline(r *http.Request, felineID string) (string, string) {
	canonicalID := felineID
	var felineAlias string
	if h.bllClient != nil {
		if resp, err := h.bllClient.GetEmployee(r.Context(), &workforcev1.GetEmployeeRequest{EmployeeID: felineID}); err == nil && resp.Employee != nil {
			canonicalID = resp.Employee.EmployeeID
			if resp.Employee.FirstName != "" {
				felineAlias = fmt.Sprintf("emp-feline-%s", strings.ToLower(resp.Employee.FirstName))
			}
		}
	}
	return canonicalID, felineAlias
}

func (h *WorkforceHandler) getCareSchedule(w http.ResponseWriter, r *http.Request) {
	felineID := r.PathValue("felineId")
	if felineID == "" {
		writeError(w, http.StatusBadRequest, "felineId parameter is required")
		return
	}

	canonicalID, felineAlias := h.resolveFeline(r, felineID)

	if h.cache != nil {
		var cached workforcev1.CareSchedule
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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetCareSchedule(r.Context(), &workforcev1.GetCareScheduleRequest{FelineID: canonicalID})
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	cs := resp.CareSchedule

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

	canonicalID, felineAlias := h.resolveFeline(r, felineID)

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	specialNeeds := ""
	if req.SpecialMedicalNeeds != nil {
		specialNeeds = *req.SpecialMedicalNeeds
	}
	perchZone := ""
	if req.PreferredPerchZone != nil {
		perchZone = *req.PreferredPerchZone
	}
	caretakerID := ""
	if req.CaretakerID != nil {
		caretakerID = *req.CaretakerID
	}
	resp, err := h.bllClient.UpdateCareSchedule(r.Context(), &workforcev1.UpdateCareScheduleRequest{
		FelineID:             canonicalID,
		DietaryPlan:          req.DietaryPlan,
		FeedingTimes:         req.FeedingTimes,
		SpecialMedicalNeeds:  specialNeeds,
		PreferredPerchZone:   perchZone,
		EmergencyMedicalHold: req.EmergencyMedicalHold,
		CaretakerID:          caretakerID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated := resp.CareSchedule
	var updatedFelineID string
	if resp.CareSchedule != nil {
		updatedFelineID = resp.CareSchedule.FelineID
	}

	if h.cache != nil {
		var currVer int64
		_, _ = h.cache.Get(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), &currVer)
		_ = h.cache.Set(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), currVer+1, 24*time.Hour)

		keys := []string{cachevalkey.KeyCareSchedule(canonicalID)}
		if felineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineID))
		}
		if updatedFelineID != "" && updatedFelineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(updatedFelineID))
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

	canonicalID, felineAlias := h.resolveFeline(r, felineID)

	var req struct {
		EmergencyMedicalHold bool `json:"emergency_medical_hold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.SetMedicalHold(r.Context(), &workforcev1.SetMedicalHoldRequest{
		FelineID:             canonicalID,
		EmergencyMedicalHold: req.EmergencyMedicalHold,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated := resp.CareSchedule
	var updatedFelineID string
	if resp.CareSchedule != nil {
		updatedFelineID = resp.CareSchedule.FelineID
	}

	if h.cache != nil {
		var currVer int64
		_, _ = h.cache.Get(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), &currVer)
		_ = h.cache.Set(r.Context(), cachevalkey.KeyCareScheduleVersion(canonicalID), currVer+1, 24*time.Hour)

		keys := []string{cachevalkey.KeyCareSchedule(canonicalID)}
		if felineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineID))
		}
		if updatedFelineID != "" && updatedFelineID != canonicalID {
			keys = append(keys, cachevalkey.KeyCareSchedule(updatedFelineID))
		}
		if felineAlias != "" {
			keys = append(keys, cachevalkey.KeyCareSchedule(felineAlias))
		}
		_ = h.cache.Delete(r.Context(), keys...)
	}

	writeJSON(w, http.StatusOK, updated)
}
