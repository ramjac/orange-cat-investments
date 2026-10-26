package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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
}

func (h *WorkforceHandler) handleOnboardEmployee(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmployeeID   string   `json:"employee_id"`
		EmployeeType string   `json:"employee_type"`
		FirstName    string   `json:"first_name"`
		LastName     string   `json:"last_name"`
		Email        string   `json:"email"`
		RoleTitle    string   `json:"role_title"`
		Department   string   `json:"department"`
		DietaryPlan  string   `json:"dietary_plan"`
		FeedingTimes []string `json:"feeding_times"`
		StorageQuota string   `json:"storage_quota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EmployeeType == "" {
		req.EmployeeType = "human"
	}
	if req.FirstName == "" {
		req.FirstName = "Employee"
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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
		StorageQuota: req.StorageQuota,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp.SagaResult)
}

func (h *WorkforceHandler) handleOffboardEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		EmployeeID   string `json:"employee_id"`
		Reason       string `json:"reason"`
		TargetStatus string `json:"target_status"`
	}
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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

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
}

func (h *WorkforceHandler) handleGetEmployeeLifecycle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "employee_id is required")
		return
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetEmployeeLifecycle(r.Context(), &internalbllv1.GetEmployeeLifecycleRequest{
		EmployeeID: id,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("employee %s not found", id))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
