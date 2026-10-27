package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() *http.ServeMux {
	mockBLL := newMockInternalBLLClient()
	h := NewWorkforceHandler(mockBLL)
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
		var cs workforcev1.CareSchedule
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
		var updated workforcev1.CareSchedule
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
		var updated workforcev1.CareSchedule
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
		var list []*workforcev1.LeaveRequest
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)
	})

	t.Run("List Leave Requests With Filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/leave-requests?employee_id=emp-feline-garfield&status=approved", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []*workforcev1.LeaveRequest
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.NotEmpty(t, list)
		assert.Equal(t, "emp-feline-garfield", list[0].EmployeeID)
		assert.Equal(t, "approved", list[0].Status)
	})

	t.Run("Create and Approve Leave Request", func(t *testing.T) {
		payload := map[string]any{
			"employee_id": "emp-feline-garfield",
			"leave_type":  "catnip_break",
			"start_date":  "2026-11-01",
			"end_date":    "2026-11-03",
			"reason":      "Alpha perch recovery session",
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/workforce/leave-requests", bytes.NewReader(data))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var created workforcev1.LeaveRequest
		err := json.Unmarshal(rec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.LeaveID)
		assert.Equal(t, "pending", created.Status)

		// Approve
		updatePayload := map[string]string{"status": "approved"}
		updateData, _ := json.Marshal(updatePayload)
		updateReq := httptest.NewRequest("PUT", "/api/v1/workforce/leave-requests/"+created.LeaveID+"/status", bytes.NewReader(updateData))
		updateRec := httptest.NewRecorder()

		mux.ServeHTTP(updateRec, updateReq)

		assert.Equal(t, http.StatusOK, updateRec.Code)
		var updated workforcev1.LeaveRequest
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
