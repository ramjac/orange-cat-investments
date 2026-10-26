package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/orange-cat-investments/oci/internal/saga"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

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

	if h.bllClient != nil {
		resp, err := h.bllClient.HandleHRWebhook(r.Context(), &internalbllv1.HandleHRWebhookRequest{
			Event:        payload.Event,
			EmployeeID:   payload.EmployeeID,
			EmployeeType: payload.EmployeeType,
			FirstName:    payload.FirstName,
			LastName:     payload.LastName,
			Email:        payload.Email,
			RoleTitle:    payload.RoleTitle,
			Department:   payload.Department,
			Status:       payload.Status,
			DietaryPlan:  payload.DietaryPlan,
			FeedingTimes: payload.FeedingTimes,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
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

		if h.onboardingSaga == nil {
			writeError(w, http.StatusNotImplemented, "onboarding saga is not configured on domain API (orchestration belongs to BLL)")
			return
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
		if h.offboardingEng == nil {
			writeError(w, http.StatusNotImplemented, "offboarding engine is not configured on domain API (orchestration belongs to BLL)")
			return
		}

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

	if payload.EmployeeID != "" && payload.Status != "" && h.wfSvc != nil {
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

	if h.bllClient != nil {
		resp, err := h.bllClient.OnboardEmployee(r.Context(), &internalbllv1.OnboardEmployeeSagaRequest{
			EmployeeID:   req.EmployeeID,
			EmployeeType: req.EmployeeType,
			FirstName:    req.FirstName,
			LastName:     req.LastName,
			Email:        req.Email,
			RoleTitle:    req.RoleTitle,
			Department:   req.Department,
			DietaryPlan:  req.DietaryPlan,
			FeedingTimes: req.FeedingTimes,
			CaretakerID:  req.CaretakerID,
			SSHKey:       req.SSHKey,
			StorageQuota: req.StorageQuota,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.SagaResult)
		return
	}

	if h.onboardingSaga == nil {
		writeError(w, http.StatusNotImplemented, "onboarding saga is not configured on domain API (orchestration belongs to BLL)")
		return
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

	if h.bllClient != nil {
		resp, err := h.bllClient.OffboardEmployee(r.Context(), &internalbllv1.OffboardEmployeeSagaRequest{
			EmployeeID:   req.EmployeeID,
			Reason:       req.Reason,
			TargetStatus: req.TargetStatus,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.SagaResult)
		return
	}

	if h.offboardingEng == nil {
		writeError(w, http.StatusNotImplemented, "offboarding engine is not configured on domain API (orchestration belongs to BLL)")
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

	if h.bllClient != nil {
		resp, err := h.bllClient.GetEmployeeLifecycle(r.Context(), &internalbllv1.GetEmployeeLifecycleRequest{
			EmployeeID: id,
		})
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("employee %s not found", id))
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if h.wfSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
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
