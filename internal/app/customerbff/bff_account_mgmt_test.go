package customerbff

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerBFF_SelfRegistration(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. Weak password rejected
	weakBody := `{"full_name":"New Investor","email":"new.investor@example.com","password":"weak"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/register", strings.NewReader(weakBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "password must be at least 8 characters long")

	// 2. Successful registration with NIST/OIDC password complexity
	validBody := `{"full_name":"Elena Rostova","email":"elena.rostova@example.com","password":"OrangeCat2026!#","target_fund":"OCI Alpha Growth Fund"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/register", strings.NewReader(validBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Status    string           `json:"status"`
		Session   *CustomerSession `json:"session"`
		CSRFToken string           `json:"csrf_token"`
		Result    struct {
			CustomerID      string  `json:"customer_id"`
			StartingBalance float64 `json:"starting_balance_usd"`
			PortfolioID     string  `json:"portfolio_id"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "registered", resp.Status)
	assert.NotEmpty(t, resp.Session.SessionID)
	assert.Equal(t, 0.00, resp.Result.StartingBalance)
	assert.NotEmpty(t, resp.Result.PortfolioID)

	// Verify session cookie set
	cookies := rec.Result().Cookies()
	var foundSessCookie bool
	for _, c := range cookies {
		if c.Name == "customer_session" && c.Value == resp.Session.SessionID {
			foundSessCookie = true
		}
	}
	assert.True(t, foundSessCookie)
}

func TestCustomerBFF_Statements_List_Generate_Download(t *testing.T) {
	_, mux := setupTestBFF()

	sessToken := "sess-arthur-token"
	csrfToken := "test-csrf-token"

	// 1. List statements
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/statements", nil)
	listReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	assert.Equal(t, http.StatusOK, listRec.Code)

	var stmts []map[string]interface{}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &stmts))
	assert.NotEmpty(t, stmts)

	// 2. Generate new Monthly Statement
	genBody := `{"type":"monthly_summary","year":2026,"month":3}`
	genReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/statements/generate", strings.NewReader(genBody))
	genReq.Header.Set("Content-Type", "application/json")
	genReq.Header.Set("X-CSRF-Token", csrfToken)
	genReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	genReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	genRec := httptest.NewRecorder()
	mux.ServeHTTP(genRec, genReq)
	assert.Equal(t, http.StatusCreated, genRec.Code)

	var genResp map[string]interface{}
	require.NoError(t, json.Unmarshal(genRec.Body.Bytes(), &genResp))
	stmtID := genResp["statement_id"].(string)
	assert.NotEmpty(t, stmtID)

	// 3. Download Statement PDF
	downReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/statements/"+stmtID+"/download", nil)
	downReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	downRec := httptest.NewRecorder()
	mux.ServeHTTP(downRec, downReq)
	assert.Equal(t, http.StatusOK, downRec.Code)
	assert.Equal(t, "application/pdf", downRec.Header().Get("Content-Type"))
	assert.True(t, bytes.HasPrefix(downRec.Body.Bytes(), []byte("%PDF-1.4")))
}

func TestCustomerBFF_AccountClosure_Liquidation(t *testing.T) {
	_, mux := setupTestBFF()

	sessToken := "sess-arthur-token"
	csrfToken := "test-csrf-token"

	closeBody := `{"portfolio_id":"port-cust-longterm-arthur"}`
	closeReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/account/close", strings.NewReader(closeBody))
	closeReq.Header.Set("Content-Type", "application/json")
	closeReq.Header.Set("X-CSRF-Token", csrfToken)
	closeReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	closeReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	closeRec := httptest.NewRecorder()
	mux.ServeHTTP(closeRec, closeReq)
	assert.Equal(t, http.StatusOK, closeRec.Code)

	var liqResp map[string]interface{}
	require.NoError(t, json.Unmarshal(closeRec.Body.Bytes(), &liqResp))
	assert.Equal(t, "closed", liqResp["account_status"])
	assert.NotEmpty(t, liqResp["transfer_id"])
}

func TestCustomerBFF_GDPRErasure_And_Certificate(t *testing.T) {
	_, mux := setupTestBFF()

	sessToken := "sess-chloe-token"
	csrfToken := "test-csrf-token"

	erasureReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/privacy/gdpr-erasure", nil)
	erasureReq.Header.Set("X-CSRF-Token", csrfToken)
	erasureReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	erasureReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	erasureRec := httptest.NewRecorder()
	mux.ServeHTTP(erasureRec, erasureReq)
	assert.Equal(t, http.StatusOK, erasureRec.Code)

	var erasureResp struct {
		Status      string                              `json:"status"`
		Certificate *reporting.GDPRDeletionCertificate `json:"certificate"`
	}
	require.NoError(t, json.Unmarshal(erasureRec.Body.Bytes(), &erasureResp))
	assert.Equal(t, "erased", erasureResp.Status)
	assert.NotEmpty(t, erasureResp.Certificate.CertificateID)
	assert.NotEmpty(t, erasureResp.Certificate.DigitalSignature)
	assert.Contains(t, erasureResp.Certificate.LegalBasis, "GDPR Article 17")

	certID := erasureResp.Certificate.CertificateID

	// Download GDPR certificate PDF
	downReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/privacy/gdpr-certificate/"+certID+"/download", nil)
	downRec := httptest.NewRecorder()
	mux.ServeHTTP(downRec, downReq)
	assert.Equal(t, http.StatusOK, downRec.Code)
	assert.Equal(t, "application/pdf", downRec.Header().Get("Content-Type"))
	assert.True(t, bytes.HasPrefix(downRec.Body.Bytes(), []byte("%PDF-1.4")))
}

func TestCustomerBFF_Search(t *testing.T) {
	bff, mux := setupTestBFF()
	_ = bff.SearchClient().IndexDocuments(context.Background(), "customer_catalog", []map[string]interface{}{
		{"id": "fund-alpha", "category": "fund", "symbol": "ALPHA", "title": "OCI Feline Behavioral Alpha Fund", "description": "Dynamic quantitative weighting based on active feline zoomie intervals."},
	})

	// Search for Alpha fund with valid Arthur session
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/search?q=Alpha", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var res map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	hits, ok := res["hits"].([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, hits)
}
