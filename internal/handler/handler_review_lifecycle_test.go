package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
)

func TestWorkforceReviewCyclesEndpoints(t *testing.T) {
	mux := setupTestServer()

	t.Run("List All Review Cycles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/review-cycles", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []*workforcev1.ReviewCycle
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)
	})

	t.Run("List Review Cycles With Filter", func(t *testing.T) {
		reqFilter := httptest.NewRequest("GET", "/workforce/review-cycles?employee_id=emp-feline-garfield&status=completed&review_type=feline_health_assessment", nil)
		recFilter := httptest.NewRecorder()
		mux.ServeHTTP(recFilter, reqFilter)

		assert.Equal(t, http.StatusOK, recFilter.Code)
		var filteredList []*workforcev1.ReviewCycle
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
		var created workforcev1.ReviewCycle
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
		var fetched workforcev1.ReviewCycle
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
		var updated workforcev1.ReviewCycle
		err = json.Unmarshal(updateRec.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, "completed", updated.Status)
		assert.Equal(t, score, updated.Score)
		assert.Equal(t, evalNotes, updated.Notes)
		assert.NotEmpty(t, updated.CompletedAt)

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
		var rc workforcev1.ReviewCycle
		_ = json.Unmarshal(createRec.Body.Bytes(), &rc)

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
		var selfUpdated workforcev1.ReviewCycle
		_ = json.Unmarshal(selfRec.Body.Bytes(), &selfUpdated)
		assert.Equal(t, "peer_feedback", selfUpdated.Phase)
		assert.Equal(t, 4.88, selfUpdated.SelfScore)

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
		var peerUpdated workforcev1.ReviewCycle
		_ = json.Unmarshal(peerRec.Body.Bytes(), &peerUpdated)
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
		var mgrUpdated workforcev1.ReviewCycle
		_ = json.Unmarshal(mgrRec.Body.Bytes(), &mgrUpdated)
		assert.Equal(t, "completed", mgrUpdated.Phase)
		assert.Equal(t, "completed", mgrUpdated.Status)
		assert.Equal(t, 4.95, mgrUpdated.Score)

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
		var vetUpdated workforcev1.ReviewCycle
		_ = json.Unmarshal(vetRec.Body.Bytes(), &vetUpdated)
		assert.Equal(t, 4.96, vetUpdated.WellnessAuditScore)
		assert.Equal(t, "Add extra perch cushion in zone 2.", vetUpdated.HabitatRecommendations)
		assert.Equal(t, "emp-human-elena", vetUpdated.ClinicalAuditorID)
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
		var res internalbllv1.OnboardingResult
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
		var res internalbllv1.OffboardingResult
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
