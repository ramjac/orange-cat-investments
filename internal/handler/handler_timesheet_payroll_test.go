package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/session"
	"github.com/stretchr/testify/assert"
)

func TestWorkforceTimesheetsEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("Submit and Fetch Timesheet", func(t *testing.T) {
		payload := map[string]any{
			"employee_id":   "emp-bob-01",
			"shift_date":    "2026-10-15",
			"start_time":    "2026-10-15T08:00:00Z",
			"end_time":      "2026-10-15T16:30:00Z",
			"break_minutes": 30,
			"task_code":     "habitat_maintenance",
			"billable":      true,
			"notes":         "Routine perch and ventilation check",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/workforce/timesheets", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var created frappe.Timesheet
		err := json.Unmarshal(rec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.TimesheetID)
		assert.Equal(t, "emp-bob-01", created.EmployeeID)
		assert.Equal(t, 8.0, created.HoursWorked)
		assert.Equal(t, "submitted", created.Status)
		assert.Equal(t, "Facilities Operations", created.CostCenter)

		// Get by ID
		getReq := httptest.NewRequest("GET", "/api/v1/workforce/timesheets/"+created.TimesheetID, nil)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusOK, getRec.Code)
		var fetched frappe.Timesheet
		err = json.Unmarshal(getRec.Body.Bytes(), &fetched)
		assert.NoError(t, err)
		assert.Equal(t, created.TimesheetID, fetched.TimesheetID)

		// Direct alias get
		aliasReq := httptest.NewRequest("GET", "/workforce/timesheets/"+created.TimesheetID, nil)
		aliasRec := httptest.NewRecorder()
		mux.ServeHTTP(aliasRec, aliasReq)
		assert.Equal(t, http.StatusOK, aliasRec.Code)

		// List
		listReq := httptest.NewRequest("GET", "/api/v1/workforce/timesheets?employee_id=emp-bob-01", nil)
		listRec := httptest.NewRecorder()
		mux.ServeHTTP(listRec, listReq)

		assert.Equal(t, http.StatusOK, listRec.Code)
		var listResp struct {
			Items []*frappe.Timesheet `json:"items"`
			Total int                 `json:"total"`
		}
		err = json.Unmarshal(listRec.Body.Bytes(), &listResp)
		assert.NoError(t, err)
		assert.Equal(t, 1, listResp.Total)

		// Approve
		approvePayload := map[string]any{
			"status":      "approved",
			"approver_id": "emp-manager-01",
		}
		approveData, _ := json.Marshal(approvePayload)
		approveReq := httptest.NewRequest("PUT", "/api/v1/workforce/timesheets/"+created.TimesheetID+"/status", bytes.NewReader(approveData))
		approveRec := httptest.NewRecorder()
		mux.ServeHTTP(approveRec, approveReq)

		assert.Equal(t, http.StatusOK, approveRec.Code)
		var approved frappe.Timesheet
		err = json.Unmarshal(approveRec.Body.Bytes(), &approved)
		assert.NoError(t, err)
		assert.Equal(t, "approved", approved.Status)
		assert.NotNil(t, approved.ApprovedBy)
		assert.Equal(t, "emp-manager-01", *approved.ApprovedBy)
	})

	t.Run("Submit and Reject Timesheet", func(t *testing.T) {
		payload := map[string]any{
			"employee_id": "emp-alice-01",
			"shift_date":  "2026-10-16",
			"start_time":  "2026-10-16T09:00:00Z",
			"end_time":    "2026-10-16T17:00:00Z",
			"task_code":   "admin",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/workforce/timesheets", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var created frappe.Timesheet
		_ = json.Unmarshal(rec.Body.Bytes(), &created)

		// Reject
		rejectPayload := map[string]any{
			"status":      "rejected",
			"reviewer_id": "emp-lead-01",
			"reason":      "Missing project code allocation",
		}
		rejectData, _ := json.Marshal(rejectPayload)
		rejectReq := httptest.NewRequest("PUT", "/workforce/timesheets/"+created.TimesheetID+"/status", bytes.NewReader(rejectData))
		rejectRec := httptest.NewRecorder()
		mux.ServeHTTP(rejectRec, rejectReq)

		assert.Equal(t, http.StatusOK, rejectRec.Code)
		var rejected frappe.Timesheet
		_ = json.Unmarshal(rejectRec.Body.Bytes(), &rejected)
		assert.Equal(t, "rejected", rejected.Status)
	})

	t.Run("Validation Error - Invalid Task Code", func(t *testing.T) {
		payload := map[string]any{
			"employee_id": "emp-bob-01",
			"shift_date":  "2026-10-17",
			"start_time":  "2026-10-17T08:00:00Z",
			"end_time":    "2026-10-17T16:00:00Z",
			"task_code":   "napping_on_desk",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/workforce/timesheets", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestWorkforcePayrollEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("Execute Payroll Run", func(t *testing.T) {
		payload := map[string]any{
			"start_date": "2026-10-01",
			"end_date":   "2026-10-31",
			"employees":  []string{"emp-human-bob", "emp-human-alice"},
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/workforce/payroll/run", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var pe frappe.PayrollEntry
		err := json.Unmarshal(rec.Body.Bytes(), &pe)
		assert.NoError(t, err)
		assert.NotEmpty(t, pe.PayrollEntryID)
		assert.Equal(t, 2, pe.SalarySlipsCreated)
	})

	t.Run("List Salary Slips for Employee", func(t *testing.T) {
		bobSession := &session.EmployeeSession{
			EmployeeID: "emp-human-bob",
			Role:       "staff",
		}

		ctx := session.WithEmployeeSession(httptest.NewRequest("GET", "/", nil).Context(), bobSession)
		req := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips?employee_id=emp-human-bob", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Items []frappe.SalarySlip `json:"items"`
			Total int                 `json:"total"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Items)
		assert.Equal(t, "emp-human-bob", resp.Items[0].EmployeeID)

		// Test employee BFF path alias
		aliasReq := httptest.NewRequest("GET", "/api/v1/employee/payroll/slips?employee_id=emp-human-bob", nil).WithContext(ctx)
		aliasRec := httptest.NewRecorder()
		mux.ServeHTTP(aliasRec, aliasReq)
		assert.Equal(t, http.StatusOK, aliasRec.Code)

		// Get specific salary slip by ID
		slipID := resp.Items[0].SalarySlipID
		getReq := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips/"+slipID, nil).WithContext(ctx)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusOK, getRec.Code)
		var slip frappe.SalarySlip
		err = json.Unmarshal(getRec.Body.Bytes(), &slip)
		assert.NoError(t, err)
		assert.Equal(t, slipID, slip.SalarySlipID)
		assert.Equal(t, 5000.00, slip.GrossPay)
	})

	t.Run("Salary Slips Unauthenticated - 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips?employee_id=emp-human-bob", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)

		getReq := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips/slip-bob-1", nil)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)
		assert.Equal(t, http.StatusUnauthorized, getRec.Code)
	})

	t.Run("Salary Slips Unauthorized Access - 403 Forbidden", func(t *testing.T) {
		aliceSession := &session.EmployeeSession{
			EmployeeID: "emp-human-alice",
			Role:       "staff",
		}
		ctx := session.WithEmployeeSession(httptest.NewRequest("GET", "/", nil).Context(), aliceSession)

		// Alice trying to list Bob's slips
		req := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips?employee_id=emp-human-bob", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		// Alice trying to get Bob's specific slip
		getReq := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips/slip-bob-1", nil).WithContext(ctx)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)
		assert.Equal(t, http.StatusForbidden, getRec.Code)
	})

	t.Run("Salary Slips Privileged Access - 200 OK for HR", func(t *testing.T) {
		hrSession := &session.EmployeeSession{
			EmployeeID: "emp-human-hr-lead",
			Role:       "hr",
		}
		ctx := session.WithEmployeeSession(httptest.NewRequest("GET", "/", nil).Context(), hrSession)

		req := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips?employee_id=emp-human-bob", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		getReq := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips/slip-bob-1", nil).WithContext(ctx)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)
		assert.Equal(t, http.StatusOK, getRec.Code)
	})
}
