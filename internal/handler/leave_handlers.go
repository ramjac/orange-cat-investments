package handler

import (
	"encoding/json"
	"net/http"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
)

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
