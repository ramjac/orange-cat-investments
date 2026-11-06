import re

with open("internal/handler/handler_timesheet_payroll_test.go", "r") as f:
    code = f.read()

tests_to_add = """
	t.Run("Security: 401 without session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/timesheets", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Security: 403 cross-employee read", func(t *testing.T) {
		// Create a timesheet for bob
		payload := map[string]any{
			"employee_id":   "emp-bob-01",
			"shift_date":    "2026-10-15",
			"start_time":    "2026-10-15T08:00:00Z",
			"end_time":      "2026-10-15T16:30:00Z",
			"break_minutes": 30,
			"task_code":     "habitat_maintenance",
			"billable":      true,
			"notes":         "notes",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/workforce/timesheets", bytes.NewReader(data))
		req = setContext(req, "emp-bob-01", "staff")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var created frappe.Timesheet
		json.Unmarshal(rec.Body.Bytes(), &created)

		// Alice tries to read Bob's timesheet
		getReq := httptest.NewRequest("GET", "/api/v1/workforce/timesheets/"+created.TimesheetID, nil)
		getReq = setContext(getReq, "emp-alice-01", "staff")
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)
		assert.Equal(t, http.StatusForbidden, getRec.Code)
	})

	t.Run("Security: IDOR query override", func(t *testing.T) {
		// Alice tries to list Bob's timesheets
		listReq := httptest.NewRequest("GET", "/api/v1/workforce/timesheets?employee_id=emp-bob-01", nil)
		listReq = setContext(listReq, "emp-alice-01", "staff")
		listRec := httptest.NewRecorder()
		mux.ServeHTTP(listRec, listReq)

		// Wait, because we override empID = sess.EmployeeID, it actually queries alice's timesheets (which might be empty).
		// We can just assert OK, but let's check it doesn't return Bob's.
		assert.Equal(t, http.StatusOK, listRec.Code)
		var listResp struct {
			Items []*frappe.Timesheet `json:"items"`
			Total int                 `json:"total"`
		}
		json.Unmarshal(listRec.Body.Bytes(), &listResp)
		// Should have 0 of Bob's
		for _, item := range listResp.Items {
			assert.NotEqual(t, "emp-bob-01", item.EmployeeID)
		}
	})

	t.Run("Security: 403 non-elevated update status", func(t *testing.T) {
		updatePayload := map[string]any{
			"status": "approved",
			"approver_id": "emp-alice-01",
		}
		updateData, _ := json.Marshal(updatePayload)
		updateReq := httptest.NewRequest("PUT", "/api/v1/workforce/timesheets/dummy-id/status", bytes.NewReader(updateData))
		updateReq = setContext(updateReq, "emp-alice-01", "staff")
		updateRec := httptest.NewRecorder()
		mux.ServeHTTP(updateRec, updateReq)

		assert.Equal(t, http.StatusForbidden, updateRec.Code)
	})
"""

code = code.replace("""	t.Run("Validation Error - Invalid Task Code", func(t *testing.T) {""", tests_to_add + """\n\tt.Run("Validation Error - Invalid Task Code", func(t *testing.T) {""")

with open("internal/handler/handler_timesheet_payroll_test.go", "w") as f:
    f.write(code)
