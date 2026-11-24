package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func outgoingActorCtx(r *http.Request) context.Context {
	ctx := r.Context()
	if sess := session.EmployeeFromContext(ctx); sess != nil {
		return grpcutil.WithOutgoingActor(ctx, grpcutil.ActorContext{
			ActorID:    sess.EmployeeID,
			Role:       sess.Role,
			Department: sess.Department,
		})
	}
	return ctx
}

type WorkforceHandler struct {
	bllClient             internalbllv1.InternalBLLServiceClient
	EnableFelineWorkforce bool
	cache                 cachevalkey.Cache
}

func (h *WorkforceHandler) SetCache(c cachevalkey.Cache) {
	h.cache = c
}

func (h *WorkforceHandler) SetBLLClient(client internalbllv1.InternalBLLServiceClient) {
	h.bllClient = client
}

func NewWorkforceHandler(bllClient internalbllv1.InternalBLLServiceClient) *WorkforceHandler {
	return &WorkforceHandler{
		bllClient:             bllClient,
		EnableFelineWorkforce: true,
	}
}

func NewWorkforceHandlerWithBLL(bllClient internalbllv1.InternalBLLServiceClient) *WorkforceHandler {
	return NewWorkforceHandler(bllClient)
}

// RegisterRoutes registers the workforce endpoints. protect, when non-nil, is
// applied to the employee-sensitive timesheet & payroll routes so that a
// session-authentication middleware runs before them; the ownership checks in
// those handlers depend on the authenticated session being present in context.
func (h *WorkforceHandler) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	if protect == nil {
		protect = func(next http.Handler) http.Handler { return next }
	}
	mux.HandleFunc("GET /api/v1/workforce/employees", h.listEmployees)
	mux.HandleFunc("GET /api/v1/workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /api/v1/workforce/leave-requests", h.createLeaveRequest)
	mux.Handle("PUT /api/v1/workforce/leave-requests/{id}/status", protect(http.HandlerFunc(h.updateLeaveRequestStatus)))
	mux.HandleFunc("GET /api/v1/workforce/review-cycles", h.listReviewCycles)
	mux.Handle("POST /api/v1/workforce/review-cycles", protect(http.HandlerFunc(h.createReviewCycle)))
	mux.HandleFunc("GET /api/v1/workforce/review-cycles/{id}", h.getReviewCycle)
	mux.Handle("PUT /api/v1/workforce/review-cycles/{id}", protect(http.HandlerFunc(h.updateReviewCycle)))
	mux.HandleFunc("POST /api/v1/workforce/review-cycles/{id}/self-appraisal", h.handleReviewSelfAppraisal)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles/{id}/peer-feedback", h.handleReviewPeerFeedback)
	mux.Handle("POST /api/v1/workforce/review-cycles/{id}/manager-assessment", protect(http.HandlerFunc(h.handleReviewManagerAssessment)))
	mux.Handle("POST /api/v1/workforce/review-cycles/{id}/vet-audit", protect(http.HandlerFunc(h.handleReviewVeterinaryAudit)))

	mux.HandleFunc("GET /api/v1/employee/review-cycles", h.listReviewCycles)
	mux.Handle("POST /api/v1/employee/review-cycles", protect(http.HandlerFunc(h.createReviewCycle)))
	mux.HandleFunc("GET /api/v1/employee/review-cycles/{id}", h.getReviewCycle)
	mux.Handle("PUT /api/v1/employee/review-cycles/{id}", protect(http.HandlerFunc(h.updateReviewCycle)))
	mux.HandleFunc("POST /api/v1/employee/review-cycles/{id}/self-appraisal", h.handleReviewSelfAppraisal)
	mux.HandleFunc("POST /api/v1/employee/review-cycles/{id}/peer-feedback", h.handleReviewPeerFeedback)
	mux.Handle("POST /api/v1/employee/review-cycles/{id}/manager-assessment", protect(http.HandlerFunc(h.handleReviewManagerAssessment)))
	mux.Handle("POST /api/v1/employee/review-cycles/{id}/vet-audit", protect(http.HandlerFunc(h.handleReviewVeterinaryAudit)))
	mux.HandleFunc("POST /api/v1/workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /api/v1/workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /api/v1/workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.Handle("GET /api/v1/workforce/timesheets", protect(http.HandlerFunc(h.listTimesheets)))
	mux.Handle("POST /api/v1/workforce/timesheets", protect(http.HandlerFunc(h.submitTimesheet)))
	mux.Handle("GET /api/v1/workforce/timesheets/{id}", protect(http.HandlerFunc(h.getTimesheet)))
	mux.Handle("PUT /api/v1/workforce/timesheets/{id}/status", protect(http.HandlerFunc(h.updateTimesheetStatus)))
	mux.Handle("GET /api/v1/workforce/payroll/slips", protect(http.HandlerFunc(h.listSalarySlips)))
	mux.Handle("GET /api/v1/workforce/payroll/slips/{id}", protect(http.HandlerFunc(h.getSalarySlip)))
	mux.Handle("POST /api/v1/workforce/payroll/run", protect(http.HandlerFunc(h.processPayrollRun)))
	mux.Handle("GET /api/v1/employee/payroll/slips", protect(http.HandlerFunc(h.listSalarySlips)))
	mux.Handle("GET /api/v1/employee/payroll/slips/{id}", protect(http.HandlerFunc(h.getSalarySlip)))
	mux.HandleFunc("POST /api/v1/workforce/employees/onboard", h.handleOnboardEmployee)
	mux.Handle("POST /api/v1/workforce/employees/{id}/offboard", protect(http.HandlerFunc(h.handleOffboardEmployee)))
	mux.HandleFunc("GET /api/v1/workforce/employees/{id}/lifecycle", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("POST /api/v1/employee/lifecycle/onboard", h.handleOnboardEmployee)
	mux.Handle("POST /api/v1/employee/lifecycle/{id}/offboard", protect(http.HandlerFunc(h.handleOffboardEmployee)))
	mux.HandleFunc("GET /api/v1/employee/lifecycle/{id}", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("POST /api/v1/webhooks/frappe-hr", h.handleFrappeHRWebhook)

	// Intranet Hub Folders
	mux.HandleFunc("GET /api/v1/workforce/hub/folders", h.listGroupFolders)
	mux.HandleFunc("POST /api/v1/workforce/hub/folders", h.createGroupFolder)
	mux.HandleFunc("GET /api/v1/employee/hub/folders", h.listGroupFolders)
	mux.HandleFunc("POST /api/v1/employee/hub/folders", h.createGroupFolder)

	// Intranet Documents
	mux.HandleFunc("GET /api/v1/workforce/documents", h.listDocuments)
	mux.HandleFunc("GET /api/v1/workforce/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /api/v1/workforce/documents", h.publishDocument)
	mux.HandleFunc("GET /api/v1/employee/documents", h.listDocuments)
	mux.HandleFunc("GET /api/v1/employee/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /api/v1/employee/documents", h.publishDocument)

	// Shared Calendar Events
	mux.HandleFunc("GET /api/v1/workforce/calendar/events", h.listCalendarEvents)
	mux.HandleFunc("POST /api/v1/workforce/calendar/events", h.createCalendarEvent)
	mux.HandleFunc("GET /api/v1/employee/calendar/events", h.listCalendarEvents)
	mux.HandleFunc("POST /api/v1/employee/calendar/events", h.createCalendarEvent)

	// Direct route aliases matching OpenAPI spec & Nextcloud compatibility
	mux.HandleFunc("GET /workforce/hub/folders", h.listGroupFolders)
	mux.HandleFunc("POST /workforce/hub/folders", h.createGroupFolder)
	mux.HandleFunc("GET /workforce/documents", h.listDocuments)
	mux.HandleFunc("GET /workforce/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /workforce/documents", h.publishDocument)
	mux.HandleFunc("GET /workforce/calendar/events", h.listCalendarEvents)
	mux.HandleFunc("POST /workforce/calendar/events", h.createCalendarEvent)

	mux.HandleFunc("GET /nextcloud/api/v1/documents", h.listDocuments)
	mux.HandleFunc("GET /api/v1/nextcloud/documents", h.listDocuments)
	mux.HandleFunc("GET /nextcloud/api/v1/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /nextcloud/api/v1/documents", h.publishDocument)
	mux.HandleFunc("POST /api/v1/nextcloud/documents", h.publishDocument)
	mux.HandleFunc("GET /nextcloud/api/v1/calendar/events", h.listCalendarEvents)
	mux.HandleFunc("POST /nextcloud/api/v1/calendar/events", h.createCalendarEvent)
	mux.HandleFunc("GET /api/v1/nextcloud/calendar/events", h.listCalendarEvents)
	mux.HandleFunc("POST /api/v1/nextcloud/calendar/events", h.createCalendarEvent)
	mux.HandleFunc("GET /nextcloud/api/v1/hub/folders", h.listGroupFolders)
	mux.HandleFunc("GET /api/v1/nextcloud/hub/folders", h.listGroupFolders)
	mux.HandleFunc("POST /nextcloud/api/v1/chat/messages", h.handleNextcloudChat)
	mux.HandleFunc("POST /api/v1/nextcloud/chat/messages", h.handleNextcloudChat)

	mux.HandleFunc("GET /workforce/employees", h.listEmployees)
	mux.HandleFunc("POST /workforce/employees/onboard", h.handleOnboardEmployee)
	mux.Handle("POST /workforce/employees/{id}/offboard", protect(http.HandlerFunc(h.handleOffboardEmployee)))
	mux.HandleFunc("GET /workforce/employees/{id}/lifecycle", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("GET /workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /workforce/leave-requests", h.createLeaveRequest)
	mux.Handle("PUT /workforce/leave-requests/{id}/status", protect(http.HandlerFunc(h.updateLeaveRequestStatus)))
	mux.HandleFunc("GET /workforce/review-cycles", h.listReviewCycles)
	mux.Handle("POST /workforce/review-cycles", protect(http.HandlerFunc(h.createReviewCycle)))
	mux.HandleFunc("GET /workforce/review-cycles/{id}", h.getReviewCycle)
	mux.Handle("PUT /workforce/review-cycles/{id}", protect(http.HandlerFunc(h.updateReviewCycle)))
	mux.HandleFunc("POST /workforce/review-cycles/{id}/self-appraisal", h.handleReviewSelfAppraisal)
	mux.HandleFunc("POST /workforce/review-cycles/{id}/peer-feedback", h.handleReviewPeerFeedback)
	mux.Handle("POST /workforce/review-cycles/{id}/manager-assessment", protect(http.HandlerFunc(h.handleReviewManagerAssessment)))
	mux.Handle("POST /workforce/review-cycles/{id}/vet-audit", protect(http.HandlerFunc(h.handleReviewVeterinaryAudit)))
	mux.HandleFunc("POST /workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.HandleFunc("GET /workforce/timesheets", h.listTimesheets)
	mux.HandleFunc("POST /workforce/timesheets", h.submitTimesheet)
	mux.HandleFunc("GET /workforce/timesheets/{id}", h.getTimesheet)
	mux.HandleFunc("PUT /workforce/timesheets/{id}/status", h.updateTimesheetStatus)
	mux.HandleFunc("GET /workforce/payroll/slips", h.listSalarySlips)
	mux.HandleFunc("GET /workforce/payroll/slips/{id}", h.getSalarySlip)
	mux.HandleFunc("POST /workforce/payroll/run", h.processPayrollRun)
	mux.HandleFunc("POST /webhooks/frappe-hr", h.handleFrappeHRWebhook)

	// Business Intelligence & Automated Executive Reporting
	mux.HandleFunc("GET /api/v1/workforce/bi/metrics", h.getBIMetrics)
	mux.HandleFunc("GET /api/v1/employee/bi/metrics", h.getBIMetrics)
	mux.HandleFunc("GET /api/v1/workforce/reports/monthly", h.listExecutiveReports)
	mux.HandleFunc("POST /api/v1/workforce/reports/monthly/generate", h.generateMonthlyReport)
	mux.HandleFunc("GET /api/v1/workforce/reports/monthly/{id}", h.getExecutiveReport)
	mux.HandleFunc("GET /api/v1/workforce/reports/monthly/{id}/download", h.downloadExecutiveReport)
	mux.HandleFunc("GET /api/v1/employee/reports/monthly", h.listExecutiveReports)
	mux.HandleFunc("POST /api/v1/employee/reports/monthly/generate", h.generateMonthlyReport)
	mux.HandleFunc("GET /api/v1/employee/reports/monthly/{id}", h.getExecutiveReport)
	mux.HandleFunc("GET /api/v1/employee/reports/monthly/{id}/download", h.downloadExecutiveReport)

	mux.HandleFunc("GET /workforce/bi/metrics", h.getBIMetrics)
	mux.HandleFunc("GET /workforce/reports/monthly", h.listExecutiveReports)
	mux.HandleFunc("POST /workforce/reports/monthly/generate", h.generateMonthlyReport)
	mux.HandleFunc("GET /workforce/reports/monthly/{id}", h.getExecutiveReport)
	mux.HandleFunc("GET /workforce/reports/monthly/{id}/download", h.downloadExecutiveReport)

	// Feline-specific workforce routes gated by EnableFelineWorkforce
	if h.EnableFelineWorkforce {
		mux.HandleFunc("GET /api/v1/workforce/care-schedules/{felineId}", h.getCareSchedule)
		mux.HandleFunc("PUT /api/v1/workforce/care-schedules/{felineId}", h.updateCareSchedule)
		mux.Handle("POST /api/v1/workforce/care-schedules/{felineId}/medical-hold", protect(http.HandlerFunc(h.toggleEmergencyMedicalHold)))
		mux.HandleFunc("POST /api/v1/workforce/vhr", h.createVHRRecord)
		mux.HandleFunc("GET /api/v1/workforce/vhr", h.listVHRRecords)

		mux.HandleFunc("GET /workforce/care-schedules/{felineId}", h.getCareSchedule)
		mux.HandleFunc("PUT /workforce/care-schedules/{felineId}", h.updateCareSchedule)
		mux.Handle("POST /workforce/care-schedules/{felineId}/medical-hold", protect(http.HandlerFunc(h.toggleEmergencyMedicalHold)))
		mux.HandleFunc("POST /workforce/vhr", h.createVHRRecord)
		mux.HandleFunc("GET /workforce/vhr", h.listVHRRecords)
	}
}

func (h *WorkforceHandler) listEmployees(w http.ResponseWriter, r *http.Request) {
	empType := r.URL.Query().Get("employee_type")
	status := r.URL.Query().Get("status")

	if h.bllClient != nil {
		resp, err := h.bllClient.ListEmployees(r.Context(), &workforcev1.ListEmployeesRequest{
			EmployeeType: empType,
			Status:       status,
		})
		if err != nil {
			writeGRPCError(w, "ListEmployees", err, http.StatusInternalServerError)
			return
		}
		items := resp.Employees
		if items == nil {
			items = []*workforcev1.Employee{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
		return
	}

	writeError(w, http.StatusServiceUnavailable, "workforce service unavailable")
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func containsSensitiveDetails(msg string) bool {
	lower := strings.ToLower(msg)
	for _, needle := range []string{
		"dial tcp",
		"connection refused",
		"transport:",
		"user=",
		"database=",
		"password=",
		"sqlstate",
		"failed to connect to",
		"rpc error:",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func sanitizeWorkforceError(httpStatus int, message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed != "" && !containsSensitiveDetails(trimmed) {
		return trimmed
	}
	switch httpStatus {
	case http.StatusServiceUnavailable:
		return "workforce service temporarily unavailable"
	case http.StatusGatewayTimeout:
		return "workforce service request timed out"
	case http.StatusNotFound:
		return "resource not found"
	case http.StatusBadRequest:
		return "invalid request parameters"
	default:
		return "internal workforce service error"
	}
}

func writeGRPCError(w http.ResponseWriter, op string, err error, fallbackStatus int) {
	st, ok := status.FromError(err)
	code := status.Code(err)
	rawMsg := err.Error()
	if ok && st.Message() != "" {
		rawMsg = st.Message()
	}

	httpStatus := fallbackStatus
	switch code {
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	}

	if httpStatus >= 500 || code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.Internal || code == codes.Unknown {
		slog.Error("internal-bll workforce gRPC call failed",
			"downstream", "internal-bll",
			"operation", op,
			"grpc_code", code.String(),
			"http_status", httpStatus,
			"error", err,
		)
	}

	writeJSON(w, httpStatus, map[string]string{
		"code":    http.StatusText(httpStatus),
		"message": sanitizeWorkforceError(httpStatus, rawMsg),
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	if status >= 500 || containsSensitiveDetails(message) {
		slog.Error("workforce handler error",
			"downstream", "internal-bll",
			"http_status", status,
			"error", message,
		)
	}
	if status == http.StatusBadRequest && (strings.Contains(strings.ToLower(message), "code = unavailable") || strings.Contains(strings.ToLower(message), "connection refused")) {
		status = http.StatusServiceUnavailable
	}
	if status == http.StatusBadRequest && strings.Contains(strings.ToLower(message), "code = deadlineexceeded") {
		status = http.StatusGatewayTimeout
	}
	writeJSON(w, status, map[string]string{
		"code":    http.StatusText(status),
		"message": sanitizeWorkforceError(status, message),
	})
}
