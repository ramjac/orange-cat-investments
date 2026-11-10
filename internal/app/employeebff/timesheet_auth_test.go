package employeebff_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/app/employeebff"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTimesheetBLLClient serves a single fixed timesheet so we can exercise the
// ownership authorization path of the timesheet handlers end-to-end.
type mockTimesheetBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	timesheet *frappe.Timesheet
}

func (m *mockTimesheetBLLClient) GetTimesheet(ctx context.Context, in *internalbllv1.GetTimesheetRequest) (*internalbllv1.GetTimesheetResponse, error) {
	return &internalbllv1.GetTimesheetResponse{Timesheet: m.timesheet}, nil
}

// newTimesheetAuthBFF builds an Employee BFF wired to a fixed timesheet owned by
// emp-owner, mirroring the production router (including session authentication on
// the sensitive workforce routes).
func newTimesheetAuthBFF(t *testing.T) (*employeebff.EmployeeBFF, http.Handler) {
	t.Helper()
	mockBLL := &mockTimesheetBLLClient{
		timesheet: &frappe.Timesheet{TimesheetID: "ts-1", EmployeeID: "emp-owner"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bff := employeebff.NewEmployeeBFF(mockBLL, logger)
	return bff, bff.SetupRouter()
}

func sessionCookie(t *testing.T, bff *employeebff.EmployeeBFF, employeeID, role string) *http.Cookie {
	t.Helper()
	sess, err := bff.SessionStore().Create(employeeID, employeeID+"@oci.local", employeeID, role, "engineering")
	require.NoError(t, err)
	return &http.Cookie{Name: "employee_session", Value: sess.SessionID}
}

// TestTimesheetRoutes_RequireSession proves the sensitive workforce routes are
// actually session-protected: a request without a valid employee session is
// rejected with 401, an authenticated owner gets 200, and a different standard
// employee is blocked with 403 (the IDOR guard) instead of reading the record.
func TestTimesheetRoutes_RequireSession(t *testing.T) {
	bff, handler := newTimesheetAuthBFF(t)

	t.Run("no session is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/timesheets/ts-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("owner can read own timesheet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/timesheets/ts-1", nil)
		req.AddCookie(sessionCookie(t, bff, "emp-owner", "staff"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("other standard employee is forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/timesheets/ts-1", nil)
		req.AddCookie(sessionCookie(t, bff, "emp-intruder", "staff"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("elevated role can read any timesheet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/timesheets/ts-1", nil)
		req.AddCookie(sessionCookie(t, bff, "emp-hr", "hr"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
