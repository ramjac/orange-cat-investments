package handler

import (
	"encoding/json"
	"net/http"

	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/session"
)

func (h *WorkforceHandler) listLeaveRequests(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	status := r.URL.Query().Get("status")

	sess := session.EmployeeFromContext(r.Context())
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized: valid employee session required")
		return
	}

	if employeeID != "" && employeeID != sess.EmployeeID && sess.Role != "hr" && sess.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden: access to leave requests of another employee is restricted")
		return
	}
	if employeeID == "" && sess.Role != "hr" && sess.Role != "admin" {
		employeeID = sess.EmployeeID
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.ListLeaveRequests(r.Context(), &workforcev1.ListLeaveRequestsRequest{
			EmployeeID: employeeID,
			Status:     status,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		list := resp.LeaveRequests
		if list == nil {
			list = []*workforcev1.LeaveRequest{}
		}
		writeJSON(w, http.StatusOK, list)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
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

	sess := session.EmployeeFromContext(r.Context())
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized: valid employee session required")
		return
	}

	if req.EmployeeID == "" {
		req.EmployeeID = sess.EmployeeID
	} else if req.EmployeeID != sess.EmployeeID && sess.Role != "hr" && sess.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden: cannot submit leave request for another employee")
		return
	}

	if h.bllClient != nil {
		reason := ""
		if req.Reason != nil {
			reason = *req.Reason
		}
		resp, err := h.bllClient.SubmitLeaveRequest(r.Context(), &workforcev1.SubmitLeaveRequestMessage{
			EmployeeID: req.EmployeeID,
			LeaveType:  req.LeaveType,
			StartDate:  req.StartDate,
			EndDate:    req.EndDate,
			Reason:     reason,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, resp.LeaveRequest)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
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

	sess := session.EmployeeFromContext(r.Context())
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized: valid employee session required")
		return
	}

	if sess.Role != "hr" && sess.Role != "admin" && sess.Role != "manager" {
		writeError(w, http.StatusForbidden, "forbidden: insufficient permissions to update leave request status")
		return
	}

	if h.bllClient != nil {
		resp, err := h.bllClient.UpdateLeaveStatus(r.Context(), &workforcev1.UpdateLeaveStatusRequest{
			LeaveID: id,
			Status:  req.Status,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp.LeaveRequest)
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}
