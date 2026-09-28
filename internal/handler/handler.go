package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	mux.HandleFunc("GET /api/v1/workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /api/v1/workforce/leave-requests", h.createLeaveRequest)
	mux.HandleFunc("PUT /api/v1/workforce/leave-requests/{id}/status", h.updateLeaveRequestStatus)
	mux.HandleFunc("GET /api/v1/workforce/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /api/v1/workforce/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /api/v1/workforce/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /api/v1/workforce/vhr", h.createVHRRecord)
	mux.HandleFunc("GET /api/v1/workforce/vhr", h.listVHRRecords)
	mux.HandleFunc("POST /api/v1/workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /api/v1/workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /api/v1/workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.HandleFunc("POST /api/v1/webhooks/frappe-hr", h.handleFrappeHRWebhook)

	// Direct route aliases matching OpenAPI spec
	mux.HandleFunc("GET /workforce/employees", h.listEmployees)
	mux.HandleFunc("GET /workforce/care-schedules/{felineId}", h.getCareSchedule)
	mux.HandleFunc("PUT /workforce/care-schedules/{felineId}", h.updateCareSchedule)
	mux.HandleFunc("POST /workforce/care-schedules/{felineId}/medical-hold", h.toggleEmergencyMedicalHold)
	mux.HandleFunc("GET /workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /workforce/leave-requests", h.createLeaveRequest)
	mux.HandleFunc("PUT /workforce/leave-requests/{id}/status", h.updateLeaveRequestStatus)
	mux.HandleFunc("GET /workforce/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /workforce/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /workforce/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /workforce/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /workforce/vhr", h.createVHRRecord)
	mux.HandleFunc("GET /workforce/vhr", h.listVHRRecords)
	mux.HandleFunc("POST /workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.HandleFunc("POST /webhooks/frappe-hr", h.handleFrappeHRWebhook)
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

func (h *WorkforceHandler) listLeaveRequests(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	status := r.URL.Query().Get("status")

	list, err := h.wfSvc.ListLeaveRequests(r.Context(), employeeID, status)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if list == nil {
		list = []*workforce.LeaveRequest{}
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *WorkforceHandler) createLeaveRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmployeeID string  `json:"employee_id"`
		LeaveType  string  `json:"leave_type"`
		StartDate  string  `json:"start_date"`
		EndDate    string  `json:"end_date"`
		Reason     *string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body: "+err.Error())
		return
	}

	created, err := h.wfSvc.CreateLeaveRequest(r.Context(), req.EmployeeID, req.LeaveType, req.StartDate, req.EndDate, req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkforceHandler) updateLeaveRequestStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "leave request id parameter is required")
		return
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body: "+err.Error())
		return
	}

	updated, err := h.wfSvc.UpdateLeaveRequestStatus(r.Context(), id, req.Status)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

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
		EmployeeID   string  `json:"employee_id"`
		ReviewType   string  `json:"review_type"`
		ScheduledFor string  `json:"scheduled_for"`
		ReviewerID   *string `json:"reviewer_id,omitempty"`
		Notes        *string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body: "+err.Error())
		return
	}

	created, err := h.wfSvc.CreateReviewCycle(r.Context(), req.EmployeeID, req.ReviewType, req.ScheduledFor, req.ReviewerID, req.Notes)
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
		Status      *string    `json:"status,omitempty"`
		ReviewerID  *string    `json:"reviewer_id,omitempty"`
		Score       *float64   `json:"score,omitempty"`
		Notes       *string    `json:"notes,omitempty"`
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request JSON body: "+err.Error())
		return
	}

	updated, err := h.wfSvc.UpdateReviewCycle(r.Context(), id, req.Status, req.ReviewerID, req.Score, req.Notes, req.CompletedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

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
