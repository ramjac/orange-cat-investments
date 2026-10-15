package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
)

type WorkforceHandler struct {
	wfSvc                 wfService.Service
	onboardingSaga        *saga.OnboardingSaga
	offboardingEng        *saga.OffboardingEngine
	EnableFelineWorkforce bool
}

func NewWorkforceHandler(
	wfSvc wfService.Service,
	onboardSaga *saga.OnboardingSaga,
	offboardEng *saga.OffboardingEngine,
) *WorkforceHandler {
	return &WorkforceHandler{
		wfSvc:                 wfSvc,
		onboardingSaga:        onboardSaga,
		offboardingEng:        offboardEng,
		EnableFelineWorkforce: true,
	}
}

func (h *WorkforceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workforce/employees", h.listEmployees)
	mux.HandleFunc("GET /api/v1/workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /api/v1/workforce/leave-requests", h.createLeaveRequest)
	mux.HandleFunc("PUT /api/v1/workforce/leave-requests/{id}/status", h.updateLeaveRequestStatus)
	mux.HandleFunc("GET /api/v1/workforce/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /api/v1/workforce/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /api/v1/workforce/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /api/v1/workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /api/v1/workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /api/v1/workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.HandleFunc("GET /api/v1/workforce/timesheets", h.listTimesheets)
	mux.HandleFunc("POST /api/v1/workforce/timesheets", h.submitTimesheet)
	mux.HandleFunc("GET /api/v1/workforce/timesheets/{id}", h.getTimesheet)
	mux.HandleFunc("PUT /api/v1/workforce/timesheets/{id}/status", h.updateTimesheetStatus)
	mux.HandleFunc("GET /api/v1/workforce/payroll/slips", h.listSalarySlips)
	mux.HandleFunc("GET /api/v1/workforce/payroll/slips/{id}", h.getSalarySlip)
	mux.HandleFunc("POST /api/v1/workforce/payroll/run", h.processPayrollRun)
	mux.HandleFunc("GET /api/v1/employee/payroll/slips", h.listSalarySlips)
	mux.HandleFunc("GET /api/v1/employee/payroll/slips/{id}", h.getSalarySlip)
	mux.HandleFunc("POST /api/v1/workforce/employees/onboard", h.handleOnboardEmployee)
	mux.HandleFunc("POST /api/v1/workforce/employees/{id}/offboard", h.handleOffboardEmployee)
	mux.HandleFunc("GET /api/v1/workforce/employees/{id}/lifecycle", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("POST /api/v1/employee/lifecycle/onboard", h.handleOnboardEmployee)
	mux.HandleFunc("POST /api/v1/employee/lifecycle/{id}/offboard", h.handleOffboardEmployee)
	mux.HandleFunc("GET /api/v1/employee/lifecycle/{id}", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("POST /api/v1/webhooks/frappe-hr", h.handleFrappeHRWebhook)

	// Direct route aliases matching OpenAPI spec
	mux.HandleFunc("GET /workforce/employees", h.listEmployees)
	mux.HandleFunc("POST /workforce/employees/onboard", h.handleOnboardEmployee)
	mux.HandleFunc("POST /workforce/employees/{id}/offboard", h.handleOffboardEmployee)
	mux.HandleFunc("GET /workforce/employees/{id}/lifecycle", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("GET /workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /workforce/leave-requests", h.createLeaveRequest)
	mux.HandleFunc("PUT /workforce/leave-requests/{id}/status", h.updateLeaveRequestStatus)
	mux.HandleFunc("GET /workforce/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /workforce/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /workforce/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /workforce/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.HandleFunc("GET /workforce/timesheets", h.listTimesheets)
	mux.HandleFunc("POST /workforce/timesheets", h.submitTimesheet)
	mux.HandleFunc("GET /workforce/timesheets/{id}", h.getTimesheet)
	mux.HandleFunc("PUT /workforce/timesheets/{id}/status", h.updateTimesheetStatus)
	mux.HandleFunc("GET /workforce/payroll/slips", h.listSalarySlips)
	mux.HandleFunc("GET /workforce/payroll/slips/{id}", h.getSalarySlip)
	mux.HandleFunc("POST /workforce/payroll/run", h.processPayrollRun)
	mux.HandleFunc("POST /webhooks/frappe-hr", h.handleFrappeHRWebhook)

	// Feline-specific workforce routes gated by EnableFelineWorkforce
	if h.EnableFelineWorkforce {
		mux.HandleFunc("GET /api/v1/workforce/care-schedules/{felineId}", h.getCareSchedule)
		mux.HandleFunc("PUT /api/v1/workforce/care-schedules/{felineId}", h.updateCareSchedule)
		mux.HandleFunc("POST /api/v1/workforce/care-schedules/{felineId}/medical-hold", h.toggleEmergencyMedicalHold)
		mux.HandleFunc("POST /api/v1/workforce/vhr", h.createVHRRecord)
		mux.HandleFunc("GET /api/v1/workforce/vhr", h.listVHRRecords)

		mux.HandleFunc("GET /workforce/care-schedules/{felineId}", h.getCareSchedule)
		mux.HandleFunc("PUT /workforce/care-schedules/{felineId}", h.updateCareSchedule)
		mux.HandleFunc("POST /workforce/care-schedules/{felineId}/medical-hold", h.toggleEmergencyMedicalHold)
		mux.HandleFunc("POST /workforce/vhr", h.createVHRRecord)
		mux.HandleFunc("GET /workforce/vhr", h.listVHRRecords)
	}
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

	created, err := h.wfSvc.CreateLeaveRequest(r.Context(), wfService.CreateLeaveRequestParams{
		EmployeeID: req.EmployeeID,
		LeaveType:  req.LeaveType,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		Reason:     req.Reason,
	})
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

	created, err := h.wfSvc.CreateReviewCycle(r.Context(), wfService.CreateReviewCycleParams{
		EmployeeID:   req.EmployeeID,
		ReviewType:   req.ReviewType,
		ScheduledFor: req.ScheduledFor,
		ReviewerID:   req.ReviewerID,
		Notes:        req.Notes,
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

	updated, err := h.wfSvc.UpdateReviewCycle(r.Context(), wfService.UpdateReviewCycleParams{
		ID:          id,
		Status:      req.Status,
		ReviewerID:  req.ReviewerID,
		Score:       req.Score,
		Notes:       req.Notes,
		CompletedAt: req.CompletedAt,
	})
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

func (h *WorkforceHandler) listTimesheets(w http.ResponseWriter, r *http.Request) {
	filter := wfService.TimesheetFilter{
		EmployeeID: r.URL.Query().Get("employee_id"),
		Status:     r.URL.Query().Get("status"),
		FromDate:   r.URL.Query().Get("from_date"),
		ToDate:     r.URL.Query().Get("to_date"),
		TaskCode:   r.URL.Query().Get("task_code"),
	}

	timesheets, err := h.wfSvc.ListTimesheets(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": timesheets,
		"total": len(timesheets),
	})
}

func (h *WorkforceHandler) submitTimesheet(w http.ResponseWriter, r *http.Request) {
	var ts frappe.Timesheet
	if err := json.NewDecoder(r.Body).Decode(&ts); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	created, err := h.wfSvc.SubmitTimesheet(r.Context(), &ts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkforceHandler) getTimesheet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "timesheet id parameter is required")
		return
	}

	ts, err := h.wfSvc.GetTimesheet(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ts)
}

func (h *WorkforceHandler) updateTimesheetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "timesheet id parameter is required")
		return
	}

	var req struct {
		Status     string `json:"status"`
		ApproverID string `json:"approver_id"`
		ReviewerID string `json:"reviewer_id"`
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	switch strings.ToLower(req.Status) {
	case "approved":
		approver := req.ApproverID
		if approver == "" {
			approver = req.ReviewerID
		}
		if approver == "" {
			writeError(w, http.StatusBadRequest, "approver_id is required for approval")
			return
		}
		updated, err := h.wfSvc.ApproveTimesheet(r.Context(), id, approver)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case "rejected":
		reviewer := req.ReviewerID
		if reviewer == "" {
			reviewer = req.ApproverID
		}
		if reviewer == "" {
			writeError(w, http.StatusBadRequest, "reviewer_id is required for rejection")
			return
		}
		if req.Reason == "" {
			writeError(w, http.StatusBadRequest, "reason is required for rejection")
			return
		}
		updated, err := h.wfSvc.RejectTimesheet(r.Context(), id, reviewer, req.Reason)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated)
	default:
		writeError(w, http.StatusBadRequest, "invalid status: must be 'approved' or 'rejected'")
	}
}

func (h *WorkforceHandler) listSalarySlips(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	slips, err := h.wfSvc.ListSalarySlips(r.Context(), employeeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": slips,
		"total": len(slips),
	})
}

func (h *WorkforceHandler) getSalarySlip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "salary slip id parameter is required")
		return
	}
	slip, err := h.wfSvc.GetSalarySlip(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, slip)
}

func (h *WorkforceHandler) processPayrollRun(w http.ResponseWriter, r *http.Request) {
	var pe frappe.PayrollEntry
	if err := json.NewDecoder(r.Body).Decode(&pe); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	result, err := h.wfSvc.ProcessPayrollRun(r.Context(), &pe)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *WorkforceHandler) handleOnboardEmployee(w http.ResponseWriter, r *http.Request) {
	var req saga.OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid onboarding request payload")
		return
	}
	if req.FirstName == "" {
		writeError(w, http.StatusBadRequest, "first_name is required")
		return
	}
	if req.EmployeeType == "" {
		req.EmployeeType = "human"
	}

	result, err := h.onboardingSaga.Execute(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *WorkforceHandler) handleOffboardEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req saga.OffboardingRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.EmployeeID == "" {
		req.EmployeeID = id
	}
	if req.EmployeeID == "" {
		writeError(w, http.StatusBadRequest, "employee_id is required")
		return
	}

	result, err := h.offboardingEng.Execute(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *WorkforceHandler) handleGetEmployeeLifecycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "employee_id is required")
		return
	}

	emp, err := h.wfSvc.GetEmployee(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("employee %s not found", id))
		return
	}

	tasks, _ := h.wfSvc.ListOnboardingChecklist(r.Context(), id)
	cs, _ := h.wfSvc.GetCareSchedule(r.Context(), id)

	username := strings.ToLower(emp.FirstName)
	if emp.LastName != nil && *emp.LastName != "" {
		username = strings.ToLower(fmt.Sprintf("%s.%s", emp.FirstName, *emp.LastName))
	}

	response := map[string]any{
		"employee":         emp,
		"status":           emp.Status,
		"care_schedule":    cs,
		"checklist_tasks":  tasks,
		"vcs_username":     username,
		"workspace_quota":  "25 GB",
		"hardware_summary": fmt.Sprintf("Standard IT asset allocation for %s", emp.RoleTitle),
	}
	writeJSON(w, http.StatusOK, response)
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
