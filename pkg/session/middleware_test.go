package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerSessionAndCSRFMiddleware(t *testing.T) {
	store := sessionvalkey.NewMemoryCustomerSessionStore()
	sess, err := store.Create("cust-test", "Test Customer", "test@oci.local")
	require.NoError(t, err)

	handler := CustomerSessionAuthMiddleware(store)(CSRFProtectMiddleware("csrf_token")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cs := CustomerFromContext(r.Context())
		require.NotNil(t, cs)
		assert.Equal(t, "cust-test", cs.CustomerID)
		w.WriteHeader(http.StatusOK)
	})))

	// Missing CSRF on POST -> 403
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/orders", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Matching CSRF on POST -> 200
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/orders", nil)
	req2.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: "tok123"})
	req2.Header.Set("X-CSRF-Token", "tok123")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestEmployeeRoleMiddleware(t *testing.T) {
	store := sessionvalkey.NewMemoryEmployeeSessionStore()
	sess, err := store.Create("emp-1", "hr@oci.local", "HR User", "hr", "HR")
	require.NoError(t, err)

	h := EmployeeSessionAuthMiddleware(store)(RequireRoleMiddleware("facilities")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/facilities/assets", nil)
	req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
