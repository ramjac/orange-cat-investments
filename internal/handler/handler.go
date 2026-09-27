package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
)

type WorkforceHandler struct {
	wfSvc            wfService.Service
	onboardingSaga   *saga.OnboardingSaga
	offboardingEng   *saga.OffboardingEngine
}

func NewWorkforceHandler(
	wfSvc wfService.Service,
	onboardSaga *saga.OnboardingSaga,
	offboardEng *saga.OffboardingEngine,
) *WorkforceHandler {
	return &WorkforceHandler{
		wfSvc:          wfSvc,
		onboardingSaga: onboardSaga,
		offboardingEng: offboardEng,
	}
}

func (h *WorkforceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workforce/employees", h.listEmployees)
	mux.HandleFunc("GET /api/v1/workforce/care-schedules/{felineId}", h.getCareSchedule)
	mux.HandleFunc("PUT /api/v1/workforce/care-schedules/{felineId}", h.updateCareSchedule)
	mux.HandleFunc("POST /api/v1/workforce/care-schedules/{felineId}/medical-hold", h.toggleEmergencyMedicalHold)
	mux.HandleFunc("POST /api/v1/webhooks/frappe-hr", h.handleFrappeHRWebhook)
}

func (h *WorkforceHandler) listEmployees(w http.ResponseWriter, r *http.Request) {
	empType := r.URL.Query().Get("employee_type")
	status := r.URL.Query().Get("status")

	employees, err := h.wfSvc.ListEmployees(r.Context(), empType, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": employees,
		"total": len(employees),
	})
}

func (h *WorkforceHandler) getCareSchedule(w http.ResponseWriter, r *http.Request) {
	felineID := r.PathValue("felineId")
	if felineID == "" {
		writeError(w, http.StatusBadRequest, "felineId parameter is required")
		return
	}

	cs, err := h.wfSvc.GetCareSchedule(r.Context(), felineID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
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
		FelineID:             felineID,
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

	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkforceHandler) toggleEmergencyMedicalHold(w http.ResponseWriter, r *http.Request) {
	felineID := r.PathValue("felineId")
	if felineID == "" {
		writeError(w, http.StatusBadRequest, "felineId parameter is required")
		return
	}

	var req struct {
		EmergencyMedicalHold bool `json:"emergency_medical_hold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	updated, err := h.wfSvc.SetEmergencyMedicalHold(r.Context(), felineID, req.EmergencyMedicalHold)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

type FrappeHRWebhookPayload struct {
	Event        string   `json:"event"`
	EmployeeID   string   `json:"employee_id"`
	EmployeeType string   `json:"employee_type,omitempty"`
	FirstName    string   `json:"first_name,omitempty"`
	LastName     string   `json:"last_name,omitempty"`
	Email        string   `json:"email,omitempty"`
	RoleTitle    string   `json:"role_title,omitempty"`
	Department   string   `json:"department,omitempty"`
	Status       string   `json:"status,omitempty"`
	DietaryPlan  string   `json:"dietary_plan,omitempty"`
	FeedingTimes []string `json:"feeding_times,omitempty"`
}

func (h *WorkforceHandler) handleFrappeHRWebhook(w http.ResponseWriter, r *http.Request) {
	var payload FrappeHRWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	event := strings.ToLower(payload.Event)
	status := strings.ToLower(payload.Status)

	if event == "employee_created" || status == "onboarding" {
		req := saga.OnboardingRequest{
			EmployeeID:   payload.EmployeeID,
			EmployeeType: payload.EmployeeType,
			FirstName:    payload.FirstName,
			LastName:     payload.LastName,
			Email:        payload.Email,
			RoleTitle:    payload.RoleTitle,
			Department:   payload.Department,
			DietaryPlan:  payload.DietaryPlan,
			FeedingTimes: payload.FeedingTimes,
		}
		if req.EmployeeType == "" {
			req.EmployeeType = "human"
		}
		if req.FirstName == "" {
			req.FirstName = "Employee"
		}

		res, err := h.onboardingSaga.Execute(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "processed",
			"action": "onboarding_saga_executed",
			"result": res,
		})
		return
	}

	if event == "employee_separated" || status == "separated" || status == "retired" {
		req := saga.OffboardingRequest{
			EmployeeID:   payload.EmployeeID,
			Reason:       "Frappe HR webhook trigger",
			TargetStatus: status,
		}

		res, err := h.offboardingEng.Execute(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "processed",
			"action": "offboarding_engine_executed",
			"result": res,
		})
		return
	}

	// Status update
	if payload.EmployeeID != "" && payload.Status != "" {
		updated, err := h.wfSvc.UpdateEmployeeStatus(r.Context(), payload.EmployeeID, payload.Status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "processed",
			"action": "employee_status_updated",
			"result": updated,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ignored",
		"action": "no_matching_saga_trigger",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"code":    http.StatusText(status),
		"message": message,
	})
}
