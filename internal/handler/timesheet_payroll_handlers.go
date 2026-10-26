package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
)

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
