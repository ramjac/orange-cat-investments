package workforce

import (
	"context"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

type (
	CareSchedule = workforce.CareSchedule
	Employee     = workforce.Employee
)

var NewMockRepository = workforce.NewMockRepository

type EventPublisher interface {
	Publish(topic string, payload any) error
}

type HRClient interface {
	CreateTimesheet(ctx context.Context, ts *frappe.Timesheet) (*frappe.Timesheet, error)
	GetTimesheet(ctx context.Context, name string) (*frappe.Timesheet, error)
	ListTimesheets(ctx context.Context, employeeID, status string) ([]frappe.Timesheet, error)
	ApproveTimesheet(ctx context.Context, name string) (*frappe.Timesheet, error)
	CancelTimesheet(ctx context.Context, name string) (*frappe.Timesheet, error)
	MarkAttendance(ctx context.Context, att *frappe.Attendance) (*frappe.Attendance, error)
	GetAttendance(ctx context.Context, name string) (*frappe.Attendance, error)
	ListAttendance(ctx context.Context, employeeID string) ([]frappe.Attendance, error)
	GetSalarySlips(ctx context.Context, employeeID string) ([]frappe.SalarySlip, error)
	GetSalarySlip(ctx context.Context, name string) (*frappe.SalarySlip, error)
	CreateSalarySlip(ctx context.Context, slip *frappe.SalarySlip) (*frappe.SalarySlip, error)
	CreatePayrollEntry(ctx context.Context, pe *frappe.PayrollEntry) (*frappe.PayrollEntry, error)
	GetFinancialMetrics(ctx context.Context, period string) (*frappe.FinancialMetrics, error)
}

type WorkspaceClient interface {
	ListGroupFolders(ctx context.Context) ([]nextcloud.GroupFolder, error)
	CreateGroupFolder(ctx context.Context, mountPoint string, groups []string, quota string) (*nextcloud.GroupFolder, error)
	ListDocuments(ctx context.Context, department, category, query string) ([]nextcloud.Document, error)
	GetDocument(ctx context.Context, id string) (*nextcloud.Document, error)
	PublishDocument(ctx context.Context, doc nextcloud.Document) (*nextcloud.Document, error)
	ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]nextcloud.CalendarEvent, error)
	CreateCalendarEvent(ctx context.Context, event nextcloud.CalendarEvent) (*nextcloud.CalendarEvent, error)
	UploadWebDAVFile(ctx context.Context, webdavPath, contentType string, content []byte) error
	DownloadWebDAVFile(ctx context.Context, webdavPath string) ([]byte, error)
	ArchiveComplianceReport(ctx context.Context, period, filename, contentType string, content []byte) (string, error)
}

type EmployeeService interface {
	GetEmployee(ctx context.Context, id string) (*workforce.Employee, error)
	ListEmployees(ctx context.Context, employeeType, status string) ([]*workforce.Employee, error)
	OnboardEmployee(ctx context.Context, emp *workforce.Employee, careSchedule *workforce.CareSchedule) (*workforce.Employee, error)
	UpdateEmployeeStatus(ctx context.Context, id, status string) (*workforce.Employee, error)
	ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*workforce.OnboardingTask, error)
	UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*workforce.OnboardingTask, error)
}

type CareVHRService interface {
	GetCareSchedule(ctx context.Context, felineID string) (*workforce.CareSchedule, error)
	UpdateCareSchedule(ctx context.Context, cs *workforce.CareSchedule) (*workforce.CareSchedule, error)
	SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*workforce.CareSchedule, error)
	CreateVHRRecord(ctx context.Context, rec *workforce.VHRRecord) (*workforce.VHRRecord, error)
	ListVHRRecords(ctx context.Context, felineID string) ([]*workforce.VHRRecord, error)
}

type LeaveIncidentService interface {
	CreateLeaveRequest(ctx context.Context, params CreateLeaveRequestParams) (*workforce.LeaveRequest, error)
	GetLeaveRequest(ctx context.Context, id string) (*workforce.LeaveRequest, error)
	ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*workforce.LeaveRequest, error)
	UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*workforce.LeaveRequest, error)
	CreateWorkplaceIncident(ctx context.Context, inc *workforce.WorkplaceIncident) (*workforce.WorkplaceIncident, error)
	ListWorkplaceIncidents(ctx context.Context, status string) ([]*workforce.WorkplaceIncident, error)
	UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*workforce.WorkplaceIncident, error)
}

type ReviewService interface {
	CreateReviewCycle(ctx context.Context, params CreateReviewCycleParams) (*workforce.ReviewCycle, error)
	GetReviewCycle(ctx context.Context, id string) (*workforce.ReviewCycle, error)
	ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*workforce.ReviewCycle, error)
	UpdateReviewCycle(ctx context.Context, params UpdateReviewCycleParams) (*workforce.ReviewCycle, error)
	SubmitSelfAppraisal(ctx context.Context, id, appraisal string, score float64) (*workforce.ReviewCycle, error)
	SubmitPeerFeedback(ctx context.Context, id, peerReviewerID, feedback string, score float64) (*workforce.ReviewCycle, error)
	SubmitManagerAssessment(ctx context.Context, id, managerID, assessment string, score float64, kpiRatings string) (*workforce.ReviewCycle, error)
	SubmitVeterinaryAudit(ctx context.Context, id, auditorID string, wellnessScore float64, recommendations string, vhrRecordID *string) (*workforce.ReviewCycle, error)
}

type PayrollTimesheetService interface {
	SubmitTimesheet(ctx context.Context, ts *frappe.Timesheet) (*frappe.Timesheet, error)
	GetTimesheet(ctx context.Context, id string) (*frappe.Timesheet, error)
	ListTimesheets(ctx context.Context, filter TimesheetFilter) ([]*frappe.Timesheet, error)
	ApproveTimesheet(ctx context.Context, timesheetID, approverID string) (*frappe.Timesheet, error)
	RejectTimesheet(ctx context.Context, timesheetID, reviewerID, reason string) (*frappe.Timesheet, error)
	ListSalarySlips(ctx context.Context, employeeID string) ([]frappe.SalarySlip, error)
	GetSalarySlip(ctx context.Context, id string) (*frappe.SalarySlip, error)
	ProcessPayrollRun(ctx context.Context, pe *frappe.PayrollEntry) (*frappe.PayrollEntry, error)
}

type IntranetService interface {
	ListGroupFolders(ctx context.Context) ([]nextcloud.GroupFolder, error)
	CreateGroupFolder(ctx context.Context, mountPoint string, groups []string, quota string) (*nextcloud.GroupFolder, error)
	ListDocuments(ctx context.Context, department, category, query string) ([]nextcloud.Document, error)
	GetDocument(ctx context.Context, id string) (*nextcloud.Document, error)
	PublishDocument(ctx context.Context, doc nextcloud.Document) (*nextcloud.Document, error)
	ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]nextcloud.CalendarEvent, error)
	CreateCalendarEvent(ctx context.Context, event nextcloud.CalendarEvent) (*nextcloud.CalendarEvent, error)
}

type ExecutiveReportingService interface {
	GetFinancialMetrics(ctx context.Context, period string) (*frappe.FinancialMetrics, error)
	GetWorkforceMetrics(ctx context.Context, period string) (*reporting.WorkforceMetrics, error)
	GenerateMonthlyReport(ctx context.Context, period, generatedBy string) (*workforce.ExecutiveReport, error)
	ListExecutiveReports(ctx context.Context, limit, offset int) ([]*workforce.ExecutiveReport, error)
	GetExecutiveReport(ctx context.Context, id string) (*workforce.ExecutiveReport, error)
	GetExecutiveReportByPeriod(ctx context.Context, period string) (*workforce.ExecutiveReport, error)
	DownloadWebDAVFile(ctx context.Context, webdavPath string) ([]byte, error)
}

type Service interface {
	EmployeeService
	CareVHRService
	LeaveIncidentService
	ReviewService
	PayrollTimesheetService
	IntranetService
	ExecutiveReportingService
}

type workforceService struct {
	repo              workforce.Repository
	publisher         EventPublisher
	hrClient          HRClient
	workspaceClient   WorkspaceClient
	memMu             sync.RWMutex
	memTimesheets     map[string]*frappe.Timesheet
	memSalarySlips    map[string]*frappe.SalarySlip
	memPayrollEntries map[string]*frappe.PayrollEntry
	memGroupFolders   []nextcloud.GroupFolder
	memDocuments      map[string]nextcloud.Document
	memCalendarEvents map[string]nextcloud.CalendarEvent
}

func seedDefaultSalarySlips() map[string]*frappe.SalarySlip {
	return make(map[string]*frappe.SalarySlip)
}

func seedDefaultGroupFolders() []nextcloud.GroupFolder {
	return nextcloud.DefaultGroupFolders()
}

func seedDefaultDocuments() map[string]nextcloud.Document {
	m := make(map[string]nextcloud.Document)
	for _, d := range nextcloud.DefaultDocuments() {
		m[d.DocumentID] = d
	}
	return m
}

func seedDefaultCalendarEvents() map[string]nextcloud.CalendarEvent {
	m := make(map[string]nextcloud.CalendarEvent)
	for _, e := range nextcloud.DefaultCalendarEvents() {
		m[e.EventID] = e
	}
	return m
}

func NewService(repo workforce.Repository) Service {
	return &workforceService{
		repo:              repo,
		memTimesheets:     make(map[string]*frappe.Timesheet),
		memSalarySlips:    seedDefaultSalarySlips(),
		memPayrollEntries: make(map[string]*frappe.PayrollEntry),
		memGroupFolders:   seedDefaultGroupFolders(),
		memDocuments:      seedDefaultDocuments(),
		memCalendarEvents: seedDefaultCalendarEvents(),
	}
}

func NewServiceWithIntegrations(repo workforce.Repository, publisher EventPublisher, hrClient HRClient, workspaceClients ...WorkspaceClient) Service {
	var ws WorkspaceClient
	if len(workspaceClients) > 0 {
		ws = workspaceClients[0]
	}
	return &workforceService{
		repo:              repo,
		publisher:         publisher,
		hrClient:          hrClient,
		workspaceClient:   ws,
		memTimesheets:     make(map[string]*frappe.Timesheet),
		memSalarySlips:    seedDefaultSalarySlips(),
		memPayrollEntries: make(map[string]*frappe.PayrollEntry),
		memGroupFolders:   seedDefaultGroupFolders(),
		memDocuments:      seedDefaultDocuments(),
		memCalendarEvents: seedDefaultCalendarEvents(),
	}
}
