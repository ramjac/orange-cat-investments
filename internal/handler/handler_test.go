package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() *http.ServeMux {
	repo := workforce.NewRepository(nil)
	svc := wfService.NewService(repo)
	onboardSaga := saga.NewOnboardingSaga(svc, nil, nil)
	offboardEng := saga.NewOffboardingEngine(svc, nil, nil)

	h := NewWorkforceHandler(svc, onboardSaga, offboardEng)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestWorkforceCareScheduleEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("Get Care Schedule", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/care-schedules/emp-feline-garfield", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var cs workforce.CareSchedule
		err := json.Unmarshal(rec.Body.Bytes(), &cs)
		assert.NoError(t, err)
		assert.Equal(t, "emp-feline-garfield", cs.FelineID)
		assert.False(t, cs.EmergencyMedicalHold)
	})

	t.Run("Update Care Schedule", func(t *testing.T) {
		body := map[string]any{
			"dietary_plan":           "Laser-guided salmon dinner",
			"feeding_times":          []string{"07:00 AM", "01:00 PM", "07:00 PM"},
			"emergency_medical_hold": false,
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/api/v1/workforce/care-schedules/emp-feline-garfield", bytes.NewReader(data))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var updated workforce.CareSchedule
		err := json.Unmarshal(rec.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, "Laser-guided salmon dinner", updated.DietaryPlan)
	})

	t.Run("Toggle Emergency Medical Hold", func(t *testing.T) {
		body := map[string]bool{
			"emergency_medical_hold": true,
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold", bytes.NewReader(data))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var updated workforce.CareSchedule
		err := json.Unmarshal(rec.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.True(t, updated.EmergencyMedicalHold)
	})
}

func TestWorkforceWebhookEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("Frappe HR Webhook - Onboarding Event", func(t *testing.T) {
		payload := FrappeHRWebhookPayload{
			Event:        "employee_created",
			EmployeeID:   "emp-feline-tom",
			EmployeeType: "feline",
			FirstName:    "Tom",
			RoleTitle:    "Observation Trainee",
			Department:   "Feline Ops",
			DietaryPlan:  "Tuna Feast",
			FeedingTimes: []string{"08:00 AM", "05:00 PM"},
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/webhooks/frappe-hr", bytes.NewReader(data))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "processed", res["status"])
		assert.Equal(t, "onboarding_saga_executed", res["action"])
	})

	t.Run("Frappe HR Webhook - Offboarding Event", func(t *testing.T) {
		payload := FrappeHRWebhookPayload{
			Event:      "employee_separated",
			EmployeeID: "emp-feline-tom",
			Status:     "separated",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/webhooks/frappe-hr", bytes.NewReader(data))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "processed", res["status"])
		assert.Equal(t, "offboarding_engine_executed", res["action"])
	})
}

func TestWorkforceLeaveRequestsEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("List All Leave Requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/leave-requests", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []*workforce.LeaveRequest
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)
	})

	t.Run("List Leave Requests With Filter", func(t *testing.T) {
		reqFilter := httptest.NewRequest("GET", "/workforce/leave-requests?employee_id=emp-feline-garfield&status=approved", nil)
		recFilter := httptest.NewRecorder()
		mux.ServeHTTP(recFilter, reqFilter)

		assert.Equal(t, http.StatusOK, recFilter.Code)
		var filteredList []*workforce.LeaveRequest
		err := json.Unmarshal(recFilter.Body.Bytes(), &filteredList)
		assert.NoError(t, err)
		assert.NotEmpty(t, filteredList)
		assert.Equal(t, "emp-feline-garfield", filteredList[0].EmployeeID)
		assert.Equal(t, "approved", filteredList[0].Status)
	})

	t.Run("Create and Update Leave Request Status", func(t *testing.T) {
		// 1. POST /api/v1/workforce/leave-requests (Submit request)
		createPayload := map[string]any{
			"employee_id": "emp-feline-barneby",
			"leave_type":  "catnip_break",
			"start_date":  "2026-10-25",
			"end_date":    "2026-10-27",
			"reason":      "Post-backtest alpha relaxation and catnip break",
		}
		data, _ := json.Marshal(createPayload)
		createReq := httptest.NewRequest("POST", "/api/v1/workforce/leave-requests", bytes.NewReader(data))
		createRec := httptest.NewRecorder()
		mux.ServeHTTP(createRec, createReq)

		assert.Equal(t, http.StatusCreated, createRec.Code)
		var created workforce.LeaveRequest
		err := json.Unmarshal(createRec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.LeaveID)
		assert.Equal(t, "catnip_break", created.LeaveType)
		assert.Equal(t, "pending", created.Status)
		assert.Equal(t, "2026-10-25", created.StartDate)

		// 2. PUT /api/v1/workforce/leave-requests/{id}/status (Approve request)
		statusPayload := map[string]string{
			"status": "approved",
		}
		statusData, _ := json.Marshal(statusPayload)
		updateReq := httptest.NewRequest("PUT", "/api/v1/workforce/leave-requests/"+created.LeaveID+"/status", bytes.NewReader(statusData))
		updateRec := httptest.NewRecorder()
		mux.ServeHTTP(updateRec, updateReq)

		assert.Equal(t, http.StatusOK, updateRec.Code)
		var updated workforce.LeaveRequest
		err = json.Unmarshal(updateRec.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, "approved", updated.Status)
	})

	t.Run("Validation Error - Invalid Leave Type", func(t *testing.T) {
		badPayload := map[string]any{
			"employee_id": "emp-feline-barneby",
			"leave_type":  "sleeping_all_day_invalid",
			"start_date":  "2026-10-25",
			"end_date":    "2026-10-27",
		}
		badData, _ := json.Marshal(badPayload)
		badReq := httptest.NewRequest("POST", "/api/v1/workforce/leave-requests", bytes.NewReader(badData))
		badRec := httptest.NewRecorder()
		mux.ServeHTTP(badRec, badReq)

		assert.Equal(t, http.StatusBadRequest, badRec.Code)
	})
}

func TestWorkforceReviewCyclesEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("List All Review Cycles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/review-cycles", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []*workforce.ReviewCycle
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)
	})

	t.Run("List Review Cycles With Filter", func(t *testing.T) {
		reqFilter := httptest.NewRequest("GET", "/workforce/review-cycles?employee_id=emp-feline-garfield&status=completed&review_type=feline_health_assessment", nil)
		recFilter := httptest.NewRecorder()
		mux.ServeHTTP(recFilter, reqFilter)

		assert.Equal(t, http.StatusOK, recFilter.Code)
		var filteredList []*workforce.ReviewCycle
		err := json.Unmarshal(recFilter.Body.Bytes(), &filteredList)
		assert.NoError(t, err)
		assert.NotEmpty(t, filteredList)
		assert.Equal(t, "emp-feline-garfield", filteredList[0].EmployeeID)
		assert.Equal(t, "completed", filteredList[0].Status)
		assert.Equal(t, "feline_health_assessment", filteredList[0].ReviewType)
	})

	t.Run("Create, Fetch, and Grade Review Cycle", func(t *testing.T) {
		reviewerID := "emp-human-alice"
		notes := "Quarterly executive leadership & perch observation performance check"
		createPayload := map[string]any{
			"employee_id":   "emp-feline-garfield",
			"review_type":   "performance",
			"scheduled_for": "2026-11-28",
			"reviewer_id":   reviewerID,
			"notes":         notes,
		}
		data, _ := json.Marshal(createPayload)
		createReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles", bytes.NewReader(data))
		createRec := httptest.NewRecorder()
		mux.ServeHTTP(createRec, createReq)

		assert.Equal(t, http.StatusCreated, createRec.Code)
		var created workforce.ReviewCycle
		err := json.Unmarshal(createRec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.ReviewID)
		assert.Equal(t, "performance", created.ReviewType)
		assert.Equal(t, "scheduled", created.Status)
		assert.Equal(t, "2026-11-28", created.ScheduledFor)

		// Fetch review cycle by ID
		getReq := httptest.NewRequest("GET", "/workforce/review-cycles/"+created.ReviewID, nil)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusOK, getRec.Code)
		var fetched workforce.ReviewCycle
		err = json.Unmarshal(getRec.Body.Bytes(), &fetched)
		assert.NoError(t, err)
		assert.Equal(t, created.ReviewID, fetched.ReviewID)

		// Grade/complete review cycle
		score := 4.95
		status := "completed"
		evalNotes := "Unmatched alpha instinct, flawless execution on executive naps"
		updatePayload := map[string]any{
			"status": status,
			"score":  score,
			"notes":  evalNotes,
		}
		updateData, _ := json.Marshal(updatePayload)
		updateReq := httptest.NewRequest("PUT", "/api/v1/workforce/review-cycles/"+created.ReviewID, bytes.NewReader(updateData))
		updateRec := httptest.NewRecorder()
		mux.ServeHTTP(updateRec, updateReq)

		assert.Equal(t, http.StatusOK, updateRec.Code)
		var updated workforce.ReviewCycle
		err = json.Unmarshal(updateRec.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, "completed", updated.Status)
		assert.Equal(t, score, *updated.Score)
		assert.Equal(t, evalNotes, *updated.Notes)
		assert.NotNil(t, updated.CompletedAt)

		// PUT validation error (score out of range)
		badScorePayload := map[string]any{
			"score": 9.9,
		}
		badScoreData, _ := json.Marshal(badScorePayload)
		badScoreReq := httptest.NewRequest("PUT", "/api/v1/workforce/review-cycles/"+created.ReviewID, bytes.NewReader(badScoreData))
		badScoreRec := httptest.NewRecorder()
		mux.ServeHTTP(badScoreRec, badScoreReq)

		assert.Equal(t, http.StatusBadRequest, badScoreRec.Code)
	})

	t.Run("Validation Error - Invalid Review Type", func(t *testing.T) {
		badPayload := map[string]any{
			"employee_id":   "emp-feline-garfield",
			"review_type":   "invalid_type",
			"scheduled_for": "2026-11-28",
		}
		badData, _ := json.Marshal(badPayload)
		badReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles", bytes.NewReader(badData))
		badRec := httptest.NewRecorder()
		mux.ServeHTTP(badRec, badReq)

		assert.Equal(t, http.StatusBadRequest, badRec.Code)
	})
}

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

	t.Run("List Salary Slips for Employee", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips?employee_id=emp-human-bob", nil)
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
		aliasReq := httptest.NewRequest("GET", "/api/v1/employee/payroll/slips?employee_id=emp-human-bob", nil)
		aliasRec := httptest.NewRecorder()
		mux.ServeHTTP(aliasRec, aliasReq)
		assert.Equal(t, http.StatusOK, aliasRec.Code)

		// Get specific salary slip by ID
		slipID := resp.Items[0].SalarySlipID
		getReq := httptest.NewRequest("GET", "/api/v1/workforce/payroll/slips/"+slipID, nil)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusOK, getRec.Code)
		var slip frappe.SalarySlip
		err = json.Unmarshal(getRec.Body.Bytes(), &slip)
		assert.NoError(t, err)
		assert.Equal(t, slipID, slip.SalarySlipID)
		assert.Equal(t, 5833.33, slip.GrossPay)
	})

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
}

func TestWorkforceLifecycleEndpoints(t *testing.T) {
	mux := setupTestServer()

	var employeeID string

	t.Run("Onboard Human Employee via Lifecycle API", func(t *testing.T) {
		payload := map[string]any{
			"first_name":    "Nala",
			"last_name":     "Simba",
			"email":         "nala.simba@oci.local",
			"role_title":    "Junior Facilities Engineer",
			"department":    "Facilities Operations",
			"employee_type": "human",
			"ssh_key":       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
			"storage_quota": "50 GB",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/workforce/employees/onboard", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var res saga.OnboardingResult
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.NotEmpty(t, res.EmployeeID)
		assert.Equal(t, "onboarding", res.Status)
		assert.NotNil(t, res.VCSAccount)
		assert.NotNil(t, res.Workspace)
		assert.NotEmpty(t, res.HardwareAssets)
		employeeID = res.EmployeeID
	})

	t.Run("Query Employee Lifecycle Record", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees/"+employeeID+"/lifecycle", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var lc map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &lc)
		assert.NoError(t, err)
		assert.Equal(t, "onboarding", lc["status"])
		assert.Equal(t, "nala.simba", lc["vcs_username"])
	})

	t.Run("Offboard Employee via Lifecycle API", func(t *testing.T) {
		payload := map[string]any{
			"reason":        "Contract concluded",
			"target_status": "separated",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/workforce/employees/"+employeeID+"/offboard", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res saga.OffboardingResult
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, employeeID, res.EmployeeID)
		assert.Equal(t, "separated", res.Status)
		assert.True(t, res.IdentityRevoked)
		assert.True(t, res.SessionsRevoked)
		assert.True(t, res.StorageArchived)
		assert.NotNil(t, res.HardwareTickets)
	})
}


