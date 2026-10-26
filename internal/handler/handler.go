package handler

import (
	"encoding/json"
	"net/http"

	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

type WorkforceHandler struct {
	bllClient             internalbllv1.InternalBLLServiceClient
	wfSvc                 wfService.Service
	onboardingSaga        *saga.OnboardingSaga
	offboardingEng        *saga.OffboardingEngine
	EnableFelineWorkforce bool
	cache                 cachevalkey.Cache
}

func (h *WorkforceHandler) SetCache(c cachevalkey.Cache) {
	h.cache = c
}

func (h *WorkforceHandler) SetBLLClient(client internalbllv1.InternalBLLServiceClient) {
	h.bllClient = client
}

func NewWorkforceHandler(
	wfSvc wfService.Service,
	onboardSaga *saga.OnboardingSaga,
	offboardEng *saga.OffboardingEngine,
) *WorkforceHandler {
	return &WorkforceHandler{
		wfSvc:                 wfSvc,
		onboardingSaga:        onboardSaga,
		offboardingEng:        offboardEng,
		EnableFelineWorkforce: true,
	}
}

func NewWorkforceHandlerWithBLL(bllClient internalbllv1.InternalBLLServiceClient) *WorkforceHandler {
	return &WorkforceHandler{
		bllClient:             bllClient,
		EnableFelineWorkforce: true,
	}
}

func (h *WorkforceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workforce/employees", h.listEmployees)
	mux.HandleFunc("GET /api/v1/workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /api/v1/workforce/leave-requests", h.createLeaveRequest)
	mux.HandleFunc("PUT /api/v1/workforce/leave-requests/{id}/status", h.updateLeaveRequestStatus)
	mux.HandleFunc("GET /api/v1/workforce/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /api/v1/workforce/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /api/v1/workforce/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles/{id}/self-appraisal", h.handleReviewSelfAppraisal)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles/{id}/peer-feedback", h.handleReviewPeerFeedback)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles/{id}/manager-assessment", h.handleReviewManagerAssessment)
	mux.HandleFunc("POST /api/v1/workforce/review-cycles/{id}/vet-audit", h.handleReviewVeterinaryAudit)

	mux.HandleFunc("GET /api/v1/employee/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /api/v1/employee/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /api/v1/employee/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /api/v1/employee/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /api/v1/employee/review-cycles/{id}/self-appraisal", h.handleReviewSelfAppraisal)
	mux.HandleFunc("POST /api/v1/employee/review-cycles/{id}/peer-feedback", h.handleReviewPeerFeedback)
	mux.HandleFunc("POST /api/v1/employee/review-cycles/{id}/manager-assessment", h.handleReviewManagerAssessment)
	mux.HandleFunc("POST /api/v1/employee/review-cycles/{id}/vet-audit", h.handleReviewVeterinaryAudit)
	mux.HandleFunc("POST /api/v1/workforce/incidents", h.createIncident)
	mux.HandleFunc("GET /api/v1/workforce/incidents", h.listIncidents)
	mux.HandleFunc("PUT /api/v1/workforce/incidents/{id}/status", h.updateIncidentStatus)
	mux.HandleFunc("GET /api/v1/workforce/timesheets", h.listTimesheets)
	mux.HandleFunc("POST /api/v1/workforce/timesheets", h.submitTimesheet)
	mux.HandleFunc("GET /api/v1/workforce/timesheets/{id}", h.getTimesheet)
	mux.HandleFunc("PUT /api/v1/workforce/timesheets/{id}/status", h.updateTimesheetStatus)
	mux.HandleFunc("GET /api/v1/workforce/payroll/slips", h.listSalarySlips)
	mux.HandleFunc("GET /api/v1/workforce/payroll/slips/{id}", h.getSalarySlip)
	mux.HandleFunc("POST /api/v1/workforce/payroll/run", h.processPayrollRun)
	mux.HandleFunc("GET /api/v1/employee/payroll/slips", h.listSalarySlips)
	mux.HandleFunc("GET /api/v1/employee/payroll/slips/{id}", h.getSalarySlip)
	mux.HandleFunc("POST /api/v1/workforce/employees/onboard", h.handleOnboardEmployee)
	mux.HandleFunc("POST /api/v1/workforce/employees/{id}/offboard", h.handleOffboardEmployee)
	mux.HandleFunc("GET /api/v1/workforce/employees/{id}/lifecycle", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("POST /api/v1/employee/lifecycle/onboard", h.handleOnboardEmployee)
	mux.HandleFunc("POST /api/v1/employee/lifecycle/{id}/offboard", h.handleOffboardEmployee)
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
	mux.HandleFunc("POST /workforce/employees/{id}/offboard", h.handleOffboardEmployee)
	mux.HandleFunc("GET /workforce/employees/{id}/lifecycle", h.handleGetEmployeeLifecycle)
	mux.HandleFunc("GET /workforce/leave-requests", h.listLeaveRequests)
	mux.HandleFunc("POST /workforce/leave-requests", h.createLeaveRequest)
	mux.HandleFunc("PUT /workforce/leave-requests/{id}/status", h.updateLeaveRequestStatus)
	mux.HandleFunc("GET /workforce/review-cycles", h.listReviewCycles)
	mux.HandleFunc("POST /workforce/review-cycles", h.createReviewCycle)
	mux.HandleFunc("GET /workforce/review-cycles/{id}", h.getReviewCycle)
	mux.HandleFunc("PUT /workforce/review-cycles/{id}", h.updateReviewCycle)
	mux.HandleFunc("POST /workforce/review-cycles/{id}/self-appraisal", h.handleReviewSelfAppraisal)
	mux.HandleFunc("POST /workforce/review-cycles/{id}/peer-feedback", h.handleReviewPeerFeedback)
	mux.HandleFunc("POST /workforce/review-cycles/{id}/manager-assessment", h.handleReviewManagerAssessment)
	mux.HandleFunc("POST /workforce/review-cycles/{id}/vet-audit", h.handleReviewVeterinaryAudit)
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
		mux.HandleFunc("POST /api/v1/workforce/care-schedules/{felineId}/medical-hold", h.toggleEmergencyMedicalHold)
		mux.HandleFunc("POST /api/v1/workforce/vhr", h.createVHRRecord)
		mux.HandleFunc("GET /api/v1/workforce/vhr", h.listVHRRecords)

		mux.HandleFunc("GET /workforce/care-schedules/{felineId}", h.getCareSchedule)
		mux.HandleFunc("PUT /workforce/care-schedules/{felineId}", h.updateCareSchedule)
		mux.HandleFunc("POST /workforce/care-schedules/{felineId}/medical-hold", h.toggleEmergencyMedicalHold)
		mux.HandleFunc("POST /workforce/vhr", h.createVHRRecord)
		mux.HandleFunc("GET /workforce/vhr", h.listVHRRecords)
	}
}

func (h *WorkforceHandler) listEmployees(w http.ResponseWriter, r *http.Request) {
	empType := r.URL.Query().Get("employee_type")
	status := r.URL.Query().Get("status")

	employees, err := h.wfSvc.ListEmployees(r.Context(), empType, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": employees,
		"total": len(employees),
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"code":    http.StatusText(status),
		"message": message,
	})
}
