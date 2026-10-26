package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

func (h *WorkforceHandler) listTimesheets(w http.ResponseWriter, r *http.Request) {
	empID := r.URL.Query().Get("employee_id")
	status := r.URL.Query().Get("status")
	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")
	taskCode := r.URL.Query().Get("task_code")

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.ListTimesheets(r.Context(), &internalbllv1.ListTimesheetsRequest{
		EmployeeID: empID,
		Status:     status,
		FromDate:   fromDate,
		ToDate:     toDate,
		TaskCode:   taskCode,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := resp.Timesheets
	if items == nil {
		items = []*frappe.Timesheet{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": len(items),
	})
}

func (h *WorkforceHandler) submitTimesheet(w http.ResponseWriter, r *http.Request) {
	var ts frappe.Timesheet
	if err := json.NewDecoder(r.Body).Decode(&ts); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.SubmitTimesheet(r.Context(), &internalbllv1.SubmitTimesheetRequest{Timesheet: &ts})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp.Timesheet)
}

func (h *WorkforceHandler) getTimesheet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "timesheet id parameter is required")
		return
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetTimesheet(r.Context(), &internalbllv1.GetTimesheetRequest{TimesheetID: id})
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp.Timesheet)
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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
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

		resp, err := h.bllClient.ApproveTimesheet(r.Context(), &internalbllv1.ApproveTimesheetRequest{
			TimesheetID: id,
			ApproverID:  approver,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.Timesheet)
		return

	case "rejected":
		reviewer := req.ReviewerID
		if reviewer == "" {
			reviewer = req.ApproverID
		}

		resp, err := h.bllClient.RejectTimesheet(r.Context(), &internalbllv1.RejectTimesheetRequest{
			TimesheetID: id,
			ReviewerID:  reviewer,
			Reason:      req.Reason,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.Timesheet)
		return

	default:
		writeError(w, http.StatusBadRequest, "status must be 'approved' or 'rejected'")
	}
}

func (h *WorkforceHandler) listSalarySlips(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.ListSalarySlips(r.Context(), &internalbllv1.ListSalarySlipsRequest{EmployeeID: employeeID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	slips := resp.SalarySlips
	if slips == nil {
		slips = []frappe.SalarySlip{}
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

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.GetSalarySlip(r.Context(), &internalbllv1.GetSalarySlipRequest{SlipID: id})
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp.SalarySlip)
}

func (h *WorkforceHandler) processPayrollRun(w http.ResponseWriter, r *http.Request) {
	var pe frappe.PayrollEntry
	if err := json.NewDecoder(r.Body).Decode(&pe); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if h.bllClient == nil {
		writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
		return
	}

	resp, err := h.bllClient.ProcessPayrollRun(r.Context(), &internalbllv1.ProcessPayrollRunRequest{PayrollEntry: &pe})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp.PayrollEntry)
}
