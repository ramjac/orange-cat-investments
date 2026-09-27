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

func TestWorkforceHandlerEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("Get Garfield Care Schedule", func(t *testing.T) {
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

	t.Run("Update Garfield Care Schedule", func(t *testing.T) {
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

	t.Run("Leave Requests HTTP Endpoints", func(t *testing.T) {
		// 1. GET /api/v1/workforce/leave-requests (List all)
		req := httptest.NewRequest("GET", "/api/v1/workforce/leave-requests", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []*workforce.LeaveRequest
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)

		// 2. GET /workforce/leave-requests with filter (alias path)
		reqFilter := httptest.NewRequest("GET", "/workforce/leave-requests?employee_id=emp-feline-garfield&status=approved", nil)
		recFilter := httptest.NewRecorder()
		mux.ServeHTTP(recFilter, reqFilter)

		assert.Equal(t, http.StatusOK, recFilter.Code)
		var filteredList []*workforce.LeaveRequest
		err = json.Unmarshal(recFilter.Body.Bytes(), &filteredList)
		assert.NoError(t, err)
		assert.NotEmpty(t, filteredList)
		assert.Equal(t, "emp-feline-garfield", filteredList[0].EmployeeID)
		assert.Equal(t, "approved", filteredList[0].Status)

		// 3. POST /api/v1/workforce/leave-requests (Submit request)
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
		err = json.Unmarshal(createRec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.LeaveID)
		assert.Equal(t, "catnip_break", created.LeaveType)
		assert.Equal(t, "pending", created.Status)
		assert.Equal(t, "2026-10-25", created.StartDate)

		// 4. PUT /api/v1/workforce/leave-requests/{id}/status (Approve request)
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

		// 5. POST validation error (invalid leave_type)
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

	t.Run("Review Cycles HTTP Endpoints", func(t *testing.T) {
		// 1. GET /api/v1/workforce/review-cycles (List all)
		req := httptest.NewRequest("GET", "/api/v1/workforce/review-cycles", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []*workforce.ReviewCycle
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)

		// 2. GET /workforce/review-cycles with filters (alias path)
		reqFilter := httptest.NewRequest("GET", "/workforce/review-cycles?employee_id=emp-feline-garfield&status=completed&review_type=feline_health_assessment", nil)
		recFilter := httptest.NewRecorder()
		mux.ServeHTTP(recFilter, reqFilter)

		assert.Equal(t, http.StatusOK, recFilter.Code)
		var filteredList []*workforce.ReviewCycle
		err = json.Unmarshal(recFilter.Body.Bytes(), &filteredList)
		assert.NoError(t, err)
		assert.NotEmpty(t, filteredList)
		assert.Equal(t, "emp-feline-garfield", filteredList[0].EmployeeID)
		assert.Equal(t, "completed", filteredList[0].Status)
		assert.Equal(t, "feline_health_assessment", filteredList[0].ReviewType)

		// 3. POST /api/v1/workforce/review-cycles (Schedule review)
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
		err = json.Unmarshal(createRec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.ReviewID)
		assert.Equal(t, "performance", created.ReviewType)
		assert.Equal(t, "scheduled", created.Status)
		assert.Equal(t, "2026-11-28", created.ScheduledFor)

		// 4. GET /workforce/review-cycles/{id} (Get by ID)
		getReq := httptest.NewRequest("GET", "/workforce/review-cycles/"+created.ReviewID, nil)
		getRec := httptest.NewRecorder()
		mux.ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusOK, getRec.Code)
		var fetched workforce.ReviewCycle
		err = json.Unmarshal(getRec.Body.Bytes(), &fetched)
		assert.NoError(t, err)
		assert.Equal(t, created.ReviewID, fetched.ReviewID)

		// 5. PUT /api/v1/workforce/review-cycles/{id} (Grade/complete review)
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

		// 6. POST validation error (invalid review_type)
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

		// 7. PUT validation error (score out of range)
		badScorePayload := map[string]any{
			"score": 9.9,
		}
		badScoreData, _ := json.Marshal(badScorePayload)
		badScoreReq := httptest.NewRequest("PUT", "/api/v1/workforce/review-cycles/"+created.ReviewID, bytes.NewReader(badScoreData))
		badScoreRec := httptest.NewRecorder()
		mux.ServeHTTP(badScoreRec, badScoreReq)

		assert.Equal(t, http.StatusBadRequest, badScoreRec.Code)
	})
}


