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
}
