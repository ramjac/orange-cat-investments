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
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
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

	t.Run("Multi-Phase Review Workflow and Veterinary Audit Endpoints", func(t *testing.T) {
		// Create review cycle
		createPayload := map[string]any{
			"employee_id":   "emp-human-bob",
			"review_type":   "performance",
			"review_period": "2026-Q3",
			"scheduled_for": "2026-11-28",
		}
		createData, _ := json.Marshal(createPayload)
		createReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles", bytes.NewReader(createData))
		createRec := httptest.NewRecorder()
		mux.ServeHTTP(createRec, createReq)
		assert.Equal(t, http.StatusCreated, createRec.Code)
		var rc workforce.ReviewCycle
		json.Unmarshal(createRec.Body.Bytes(), &rc)

		// 1. POST /self-appraisal
		selfPayload := map[string]any{
			"self_appraisal": "Completed 100% telemetry calibration.",
			"self_score":     4.88,
		}
		selfData, _ := json.Marshal(selfPayload)
		selfReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles/"+rc.ReviewID+"/self-appraisal", bytes.NewReader(selfData))
		selfRec := httptest.NewRecorder()
		mux.ServeHTTP(selfRec, selfReq)
		assert.Equal(t, http.StatusOK, selfRec.Code)
		var selfUpdated workforce.ReviewCycle
		json.Unmarshal(selfRec.Body.Bytes(), &selfUpdated)
		assert.Equal(t, "peer_feedback", selfUpdated.Phase)
		assert.Equal(t, 4.88, *selfUpdated.SelfScore)

		// 2. POST /peer-feedback
		peerPayload := map[string]any{
			"peer_reviewer_id": "emp-human-frank",
			"peer_feedback":   "Great teamwork on IoT security.",
			"peer_score":      4.90,
		}
		peerData, _ := json.Marshal(peerPayload)
		peerReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles/"+rc.ReviewID+"/peer-feedback", bytes.NewReader(peerData))
		peerRec := httptest.NewRecorder()
		mux.ServeHTTP(peerRec, peerReq)
		assert.Equal(t, http.StatusOK, peerRec.Code)
		var peerUpdated workforce.ReviewCycle
		json.Unmarshal(peerRec.Body.Bytes(), &peerUpdated)
		assert.Equal(t, "manager_review", peerUpdated.Phase)

		// 3. POST /manager-assessment
		mgrPayload := map[string]any{
			"manager_id":         "emp-human-alice",
			"manager_assessment": "Superb leadership on facilities telemetry.",
			"manager_score":      4.95,
			"kpi_ratings":        map[string]float64{"hardware_uptime": 5.0},
		}
		mgrData, _ := json.Marshal(mgrPayload)
		mgrReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles/"+rc.ReviewID+"/manager-assessment", bytes.NewReader(mgrData))
		mgrRec := httptest.NewRecorder()
		mux.ServeHTTP(mgrRec, mgrReq)
		assert.Equal(t, http.StatusOK, mgrRec.Code)
		var mgrUpdated workforce.ReviewCycle
		json.Unmarshal(mgrRec.Body.Bytes(), &mgrUpdated)
		assert.Equal(t, "completed", mgrUpdated.Phase)
		assert.Equal(t, "completed", mgrUpdated.Status)
		assert.Equal(t, 4.95, *mgrUpdated.Score)

		// 4. POST /vet-audit
		vetPayload := map[string]any{
			"clinical_auditor_id":     "emp-human-elena",
			"wellness_audit_score":    4.96,
			"habitat_recommendations": "Add extra perch cushion in zone 2.",
			"vhr_record_id":           "vhr-12345",
		}
		vetData, _ := json.Marshal(vetPayload)
		vetReq := httptest.NewRequest("POST", "/api/v1/workforce/review-cycles/"+rc.ReviewID+"/vet-audit", bytes.NewReader(vetData))
		vetRec := httptest.NewRecorder()
		mux.ServeHTTP(vetRec, vetReq)
		assert.Equal(t, http.StatusOK, vetRec.Code)
		var vetUpdated workforce.ReviewCycle
		json.Unmarshal(vetRec.Body.Bytes(), &vetUpdated)
		assert.Equal(t, 4.96, *vetUpdated.WellnessAuditScore)
		assert.Equal(t, "Add extra perch cushion in zone 2.", *vetUpdated.HabitatRecommendations)
		assert.Equal(t, "emp-human-elena", *vetUpdated.ClinicalAuditorID)
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
		assert.Equal(t, 5000.00, slip.GrossPay)
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

func TestHubAndDocuments_Endpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("List Hub Folders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/hub/folders", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		folders := res["folders"].([]any)
		assert.True(t, len(folders) >= 4)
	})

	t.Run("Create Hub Folder", func(t *testing.T) {
		payload := map[string]any{
			"mount_point": "Trading Risk Protocols",
			"quota":       "50 GB",
			"groups":      []string{"trading", "risk"},
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/workforce/hub/folders", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var folder nextcloud.GroupFolder
		err := json.Unmarshal(rec.Body.Bytes(), &folder)
		assert.NoError(t, err)
		assert.Equal(t, "Trading Risk Protocols", folder.MountPoint)
	})

	t.Run("List and Search Documents", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/documents?q=Onboarding", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		docs := res["documents"].([]any)
		assert.NotEmpty(t, docs)
	})

	t.Run("Get Document by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/documents/doc-001", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var doc nextcloud.Document
		err := json.Unmarshal(rec.Body.Bytes(), &doc)
		assert.NoError(t, err)
		assert.Equal(t, "doc-001", doc.DocumentID)
		assert.Contains(t, doc.Title, "Onboarding")
	})

	t.Run("Publish Intranet Document", func(t *testing.T) {
		payload := map[string]any{
			"title":      "Cleanroom Sanitization SOP",
			"department": "Facilities & Maintenance SOPs",
			"category":   "SOP",
			"author":     "emp-human-bob",
			"content":    "Strict sanitization procedures for Habitat Alpha cleanrooms.",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/workforce/documents", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var doc nextcloud.Document
		err := json.Unmarshal(rec.Body.Bytes(), &doc)
		assert.NoError(t, err)
		assert.NotEmpty(t, doc.DocumentID)
		assert.Equal(t, "Cleanroom Sanitization SOP", doc.Title)
		assert.NotEmpty(t, doc.WebDAVPath)
	})

	t.Run("Nextcloud Compatibility Endpoints", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nextcloud/api/v1/documents", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		chatPayload := map[string]any{
			"sender":    "emp-human-alice",
			"recipient": "emp-human-rick",
			"room":      "general",
			"message":   "Hello world!",
		}
		data, _ := json.Marshal(chatPayload)
		chatReq := httptest.NewRequest("POST", "/nextcloud/api/v1/chat/messages", bytes.NewReader(data))
		chatRec := httptest.NewRecorder()
		mux.ServeHTTP(chatRec, chatReq)
		assert.Equal(t, http.StatusCreated, chatRec.Code)
	})
}

func TestCalendar_Endpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("List Calendar Events", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/calendar/events", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		events := res["events"].([]any)
		assert.NotEmpty(t, events)
	})

	t.Run("Create Calendar Event", func(t *testing.T) {
		payload := map[string]any{
			"calendar":    "Shift Rotations",
			"title":       "Night Habitat Patrol",
			"department":  "Facilities & Maintenance SOPs",
			"location":    "Habitat Perimeter",
			"organizer":   "emp-human-bob",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/workforce/calendar/events", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var ev nextcloud.CalendarEvent
		err := json.Unmarshal(rec.Body.Bytes(), &ev)
		assert.NoError(t, err)
		assert.NotEmpty(t, ev.EventID)
		assert.Equal(t, "Night Habitat Patrol", ev.Title)
	})
}

func TestExecutiveReports_Endpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("Get BI Metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/bi/metrics?period=2026-10", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "2026-10", res["period"])
		assert.NotNil(t, res["financial"])
		assert.NotNil(t, res["workforce"])
	})

	t.Run("List Executive Reports", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/reports/monthly", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		items := res["items"].([]any)
		assert.GreaterOrEqual(t, len(items), 2)
	})

	var createdReportID string
	t.Run("Generate Monthly Executive Report", func(t *testing.T) {
		payload := map[string]any{
			"period":       "2026-10",
			"generated_by": "alice_test",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/workforce/reports/monthly/generate", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var report workforce.ExecutiveReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		assert.NoError(t, err)
		assert.Equal(t, "rep-2026-10", report.ReportID)
		assert.Equal(t, "archived", report.Status)
		createdReportID = report.ReportID
	})

	t.Run("Get Executive Report By ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/reports/monthly/"+createdReportID, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var report workforce.ExecutiveReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		assert.NoError(t, err)
		assert.Equal(t, createdReportID, report.ReportID)
	})

	t.Run("Download Executive Report PDF", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/reports/monthly/"+createdReportID+"/download?format=pdf", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=\"executive_summary_report_2026-10.pdf\"")
		assert.True(t, bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-1.4")))
	})

	t.Run("Download Executive Report CSV", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/reports/monthly/"+createdReportID+"/download?format=csv", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=\"executive_summary_report_2026-10.csv\"")
		assert.Contains(t, rec.Body.String(), "Total Revenue")
	})
}



