package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
)

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

	t.Run("List Documents Validation Error - Search Query Exceeds Max Length", func(t *testing.T) {
		longQuery := "a"
		for i := 0; i < 105; i++ {
			longQuery += "a"
		}
		req := httptest.NewRequest("GET", "/api/v1/workforce/documents?q="+longQuery, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "exceeds maximum length of 100 characters")
	})

	t.Run("List Documents Validation Error - Search Query Contains Control Characters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/documents?q=invalid%00query", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "contains invalid control characters")
	})

	t.Run("List Documents Validation Error - Department Exceeds Max Length", func(t *testing.T) {
		longDept := "a"
		for i := 0; i < 105; i++ {
			longDept += "a"
		}
		req := httptest.NewRequest("GET", "/api/v1/workforce/documents?department="+longDept, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "department parameter exceeds maximum length")
	})

	t.Run("List Documents Validation Error - Category Contains Control Characters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/documents?category=bad%0Acat", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "category parameter contains invalid control characters")
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
			"calendar":   "Shift Rotations",
			"title":      "Night Habitat Patrol",
			"department": "Facilities & Maintenance SOPs",
			"location":   "Habitat Perimeter",
			"organizer":  "emp-human-bob",
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
		var report internalbllv1.ExecutiveReportSummary
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
		var report workforcev1.ExecutiveReport
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
