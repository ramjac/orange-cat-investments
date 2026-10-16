package workforce

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

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

type TimesheetFilter struct {
	EmployeeID string `json:"employee_id"`
	Status     string `json:"status"`
	FromDate   string `json:"from_date"`
	ToDate     string `json:"to_date"`
	TaskCode   string `json:"task_code"`
}

type TimesheetApprovedPayload struct {
	TimesheetID       string    `json:"timesheet_id"`
	EmployeeID        string    `json:"employee_id"`
	ShiftDate         string    `json:"shift_date"`
	HoursWorked       float64   `json:"hours_worked"`
	TaskCode          string    `json:"task_code"`
	CostCenter        string    `json:"cost_center"`
	ApprovedBy        string    `json:"approved_by"`
	ApprovedAt        time.Time `json:"approved_at"`
	FrappeTimesheetID *string   `json:"frappe_timesheet_id,omitempty"`
}

type Service interface {
	GetEmployee(ctx context.Context, id string) (*workforce.Employee, error)
	ListEmployees(ctx context.Context, employeeType, status string) ([]*workforce.Employee, error)
	OnboardEmployee(ctx context.Context, emp *workforce.Employee, careSchedule *workforce.CareSchedule) (*workforce.Employee, error)
	UpdateEmployeeStatus(ctx context.Context, id, status string) (*workforce.Employee, error)
	GetCareSchedule(ctx context.Context, felineID string) (*workforce.CareSchedule, error)
	UpdateCareSchedule(ctx context.Context, cs *workforce.CareSchedule) (*workforce.CareSchedule, error)
	SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*workforce.CareSchedule, error)
	ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*workforce.OnboardingTask, error)
	UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*workforce.OnboardingTask, error)

	// Leave Requests
	CreateLeaveRequest(ctx context.Context, params CreateLeaveRequestParams) (*workforce.LeaveRequest, error)
	GetLeaveRequest(ctx context.Context, id string) (*workforce.LeaveRequest, error)
	ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*workforce.LeaveRequest, error)
	UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*workforce.LeaveRequest, error)

	// Review Cycles
	CreateReviewCycle(ctx context.Context, params CreateReviewCycleParams) (*workforce.ReviewCycle, error)
	GetReviewCycle(ctx context.Context, id string) (*workforce.ReviewCycle, error)
	ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*workforce.ReviewCycle, error)
	UpdateReviewCycle(ctx context.Context, params UpdateReviewCycleParams) (*workforce.ReviewCycle, error)
	SubmitSelfAppraisal(ctx context.Context, id, appraisal string, score float64) (*workforce.ReviewCycle, error)
	SubmitPeerFeedback(ctx context.Context, id, peerReviewerID, feedback string, score float64) (*workforce.ReviewCycle, error)
	SubmitManagerAssessment(ctx context.Context, id, managerID, assessment string, score float64, kpiRatings string) (*workforce.ReviewCycle, error)
	SubmitVeterinaryAudit(ctx context.Context, id, auditorID string, wellnessScore float64, recommendations string, vhrRecordID *string) (*workforce.ReviewCycle, error)

	// Electronic Veterinary Health Records (VHR)
	CreateVHRRecord(ctx context.Context, rec *workforce.VHRRecord) (*workforce.VHRRecord, error)
	ListVHRRecords(ctx context.Context, felineID string) ([]*workforce.VHRRecord, error)

	// Workplace Incidents
	CreateWorkplaceIncident(ctx context.Context, inc *workforce.WorkplaceIncident) (*workforce.WorkplaceIncident, error)
	ListWorkplaceIncidents(ctx context.Context, status string) ([]*workforce.WorkplaceIncident, error)
	UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*workforce.WorkplaceIncident, error)

	// Timesheets & Attendance Tracking
	SubmitTimesheet(ctx context.Context, ts *frappe.Timesheet) (*frappe.Timesheet, error)
	GetTimesheet(ctx context.Context, id string) (*frappe.Timesheet, error)
	ListTimesheets(ctx context.Context, filter TimesheetFilter) ([]*frappe.Timesheet, error)
	ApproveTimesheet(ctx context.Context, timesheetID, approverID string) (*frappe.Timesheet, error)
	RejectTimesheet(ctx context.Context, timesheetID, reviewerID, reason string) (*frappe.Timesheet, error)

	// Payroll & Compensation
	ListSalarySlips(ctx context.Context, employeeID string) ([]frappe.SalarySlip, error)
	GetSalarySlip(ctx context.Context, id string) (*frappe.SalarySlip, error)
	ProcessPayrollRun(ctx context.Context, pe *frappe.PayrollEntry) (*frappe.PayrollEntry, error)

	// Documents, Knowledge Base & Shared Calendar (Nextcloud Hub)
	ListGroupFolders(ctx context.Context) ([]nextcloud.GroupFolder, error)
	CreateGroupFolder(ctx context.Context, mountPoint string, groups []string, quota string) (*nextcloud.GroupFolder, error)
	ListDocuments(ctx context.Context, department, category, query string) ([]nextcloud.Document, error)
	GetDocument(ctx context.Context, id string) (*nextcloud.Document, error)
	PublishDocument(ctx context.Context, doc nextcloud.Document) (*nextcloud.Document, error)
	ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]nextcloud.CalendarEvent, error)
	CreateCalendarEvent(ctx context.Context, event nextcloud.CalendarEvent) (*nextcloud.CalendarEvent, error)

	// Business Intelligence & Automated Executive Reporting
	GetFinancialMetrics(ctx context.Context, period string) (*frappe.FinancialMetrics, error)
	GetWorkforceMetrics(ctx context.Context, period string) (*reporting.WorkforceMetrics, error)
	GenerateMonthlyReport(ctx context.Context, period, generatedBy string) (*workforce.ExecutiveReport, error)
	ListExecutiveReports(ctx context.Context, limit, offset int) ([]*workforce.ExecutiveReport, error)
	GetExecutiveReport(ctx context.Context, id string) (*workforce.ExecutiveReport, error)
	GetExecutiveReportByPeriod(ctx context.Context, period string) (*workforce.ExecutiveReport, error)
	DownloadWebDAVFile(ctx context.Context, webdavPath string) ([]byte, error)
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

func (s *workforceService) GetEmployee(ctx context.Context, id string) (*workforce.Employee, error) {
	if id == "" {
		return nil, errors.New("employee id cannot be empty")
	}
	return s.repo.GetEmployeeByID(ctx, id)
}

func (s *workforceService) ListEmployees(ctx context.Context, employeeType, status string) ([]*workforce.Employee, error) {
	return s.repo.ListEmployees(ctx, employeeType, status)
}

func (s *workforceService) OnboardEmployee(ctx context.Context, emp *workforce.Employee, careSchedule *workforce.CareSchedule) (*workforce.Employee, error) {
	if emp == nil {
		return nil, errors.New("employee entity cannot be nil")
	}
	if emp.FirstName == "" {
		return nil, errors.New("first name is required")
	}
	if emp.EmployeeType != "human" && emp.EmployeeType != "feline" {
		return nil, errors.New("employee type must be 'human' or 'feline'")
	}

	created, err := s.repo.CreateEmployee(ctx, emp)
	if err != nil {
		return nil, err
	}

	// Create default onboarding tasks
	defaultTasks := []struct {
		Name     string
		Category string
	}{
		{"Badge & Access Provisioning", "access"},
		{"Workplace Safety & Compliance", "training"},
	}

	if emp.EmployeeType == "feline" {
		defaultTasks = append(defaultTasks,
			struct {
				Name     string
				Category string
			}{"Preferred Perch Inspection", "hardware"},
			struct {
				Name     string
				Category string
			}{"Dietary & Care Schedule Setup", "dietary"},
		)
	}

	for _, t := range defaultTasks {
		task := &workforce.OnboardingTask{
			EmployeeID:  created.EmployeeID,
			TaskName:    t.Name,
			Category:    t.Category,
			IsCompleted: false,
		}
		_, _ = s.repo.CreateOnboardingTask(ctx, task)
	}

	// If feline and care schedule provided, save care schedule
	if emp.EmployeeType == "feline" && careSchedule != nil {
		careSchedule.FelineID = created.EmployeeID
		_, _ = s.repo.UpsertCareSchedule(ctx, careSchedule)
	}

	return created, nil
}

func (s *workforceService) UpdateEmployeeStatus(ctx context.Context, id, status string) (*workforce.Employee, error) {
	if id == "" || status == "" {
		return nil, errors.New("employee id and status are required")
	}
	return s.repo.UpdateEmployeeStatus(ctx, id, status)
}

func (s *workforceService) GetCareSchedule(ctx context.Context, felineID string) (*workforce.CareSchedule, error) {
	if felineID == "" {
		return nil, errors.New("feline id cannot be empty")
	}
	return s.repo.GetCareScheduleByFelineID(ctx, felineID)
}

func (s *workforceService) UpdateCareSchedule(ctx context.Context, cs *workforce.CareSchedule) (*workforce.CareSchedule, error) {
	if cs == nil || cs.FelineID == "" {
		return nil, errors.New("valid care schedule and feline id required")
	}
	return s.repo.UpsertCareSchedule(ctx, cs)
}

func (s *workforceService) SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*workforce.CareSchedule, error) {
	if felineID == "" {
		return nil, errors.New("feline id cannot be empty")
	}
	return s.repo.SetEmergencyMedicalHold(ctx, felineID, hold)
}

func (s *workforceService) ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*workforce.OnboardingTask, error) {
	if employeeID == "" {
		return nil, errors.New("employee id cannot be empty")
	}
	return s.repo.ListOnboardingChecklist(ctx, employeeID)
}

func (s *workforceService) UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*workforce.OnboardingTask, error) {
	if taskID == "" {
		return nil, errors.New("task id cannot be empty")
	}
	return s.repo.UpdateOnboardingTask(ctx, taskID, isCompleted)
}

type CreateLeaveRequestParams struct {
	EmployeeID string
	LeaveType  string
	StartDate  string
	EndDate    string
	Reason     *string
}

func (s *workforceService) CreateLeaveRequest(ctx context.Context, params CreateLeaveRequestParams) (*workforce.LeaveRequest, error) {
	if params.EmployeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	switch params.LeaveType {
	case "vacation", "sick", "catnip_break", "sabbatical":
		// valid
	default:
		return nil, errors.New("invalid leave_type: must be vacation, sick, catnip_break, or sabbatical")
	}

	if params.StartDate == "" || params.EndDate == "" {
		return nil, errors.New("start_date and end_date are required")
	}

	const dateLayout = "2006-01-02"
	start, err := time.Parse(dateLayout, params.StartDate)
	if err != nil {
		return nil, errors.New("start_date must be in YYYY-MM-DD format")
	}
	end, err := time.Parse(dateLayout, params.EndDate)
	if err != nil {
		return nil, errors.New("end_date must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return nil, errors.New("end_date must be on or after start_date")
	}

	req := &workforce.LeaveRequest{
		EmployeeID: params.EmployeeID,
		LeaveType:  params.LeaveType,
		StartDate:  params.StartDate,
		EndDate:    params.EndDate,
		Status:     "pending",
		Reason:     params.Reason,
	}

	return s.repo.CreateLeaveRequest(ctx, req)
}

func (s *workforceService) GetLeaveRequest(ctx context.Context, id string) (*workforce.LeaveRequest, error) {
	if id == "" {
		return nil, errors.New("leave request id cannot be empty")
	}
	return s.repo.GetLeaveRequestByID(ctx, id)
}

func (s *workforceService) ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*workforce.LeaveRequest, error) {
	if status != "" {
		switch status {
		case "pending", "approved", "rejected", "cancelled":
			// valid
		default:
			return nil, errors.New("invalid status: must be pending, approved, rejected, or cancelled")
		}
	}
	return s.repo.ListLeaveRequests(ctx, employeeID, status)
}

func (s *workforceService) UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*workforce.LeaveRequest, error) {
	if id == "" {
		return nil, errors.New("leave request id cannot be empty")
	}
	switch status {
	case "pending", "approved", "rejected", "cancelled":
		// valid
	default:
		return nil, errors.New("invalid status: must be pending, approved, rejected, or cancelled")
	}
	return s.repo.UpdateLeaveRequestStatus(ctx, id, status)
}

type CreateReviewCycleParams struct {
	EmployeeID              string
	ReviewType              string
	ReviewPeriod            string
	Phase                   string
	ScheduledFor            string
	ReviewerID              *string
	Notes                   *string
	SelfAppraisal           *string
	SelfScore               *float64
	PeerFeedback            *string
	PeerScore               *float64
	PeerReviewerID          *string
	ManagerAssessment       *string
	ManagerScore            *float64
	KPIRatings              *string
	WellnessAuditScore      *float64
	HabitatRecommendations *string
	ClinicalAuditorID       *string
	VHRRecordID             *string
}

func (s *workforceService) CreateReviewCycle(ctx context.Context, params CreateReviewCycleParams) (*workforce.ReviewCycle, error) {
	if params.EmployeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	switch params.ReviewType {
	case "performance", "feline_health_assessment":
		// valid
	default:
		return nil, errors.New("invalid review_type: must be performance or feline_health_assessment")
	}

	if params.ScheduledFor == "" {
		return nil, errors.New("scheduled_for date is required")
	}
	const dateLayout = "2006-01-02"
	if _, err := time.Parse(dateLayout, params.ScheduledFor); err != nil {
		return nil, errors.New("scheduled_for must be in YYYY-MM-DD format")
	}

	if params.Phase != "" {
		switch params.Phase {
		case "self_appraisal", "peer_feedback", "manager_review", "completed":
			// valid
		default:
			return nil, errors.New("invalid phase: must be self_appraisal, peer_feedback, manager_review, or completed")
		}
	} else {
		params.Phase = "self_appraisal"
	}

	if params.ReviewPeriod == "" {
		params.ReviewPeriod = "2026-Q3"
	}

	if params.SelfScore != nil && (*params.SelfScore < 0.0 || *params.SelfScore > 5.0) {
		return nil, errors.New("self_score must be between 0.00 and 5.00")
	}
	if params.PeerScore != nil && (*params.PeerScore < 0.0 || *params.PeerScore > 5.0) {
		return nil, errors.New("peer_score must be between 0.00 and 5.00")
	}
	if params.ManagerScore != nil && (*params.ManagerScore < 0.0 || *params.ManagerScore > 5.0) {
		return nil, errors.New("manager_score must be between 0.00 and 5.00")
	}
	if params.WellnessAuditScore != nil && (*params.WellnessAuditScore < 0.0 || *params.WellnessAuditScore > 5.0) {
		return nil, errors.New("wellness_audit_score must be between 0.00 and 5.00")
	}

	rc := &workforce.ReviewCycle{
		EmployeeID:              params.EmployeeID,
		ReviewType:              params.ReviewType,
		ReviewPeriod:            params.ReviewPeriod,
		Phase:                   params.Phase,
		ScheduledFor:            params.ScheduledFor,
		Status:                  "scheduled",
		ReviewerID:              params.ReviewerID,
		Notes:                   params.Notes,
		SelfAppraisal:           params.SelfAppraisal,
		SelfScore:               params.SelfScore,
		PeerFeedback:            params.PeerFeedback,
		PeerScore:               params.PeerScore,
		PeerReviewerID:          params.PeerReviewerID,
		ManagerAssessment:       params.ManagerAssessment,
		ManagerScore:            params.ManagerScore,
		KPIRatings:              params.KPIRatings,
		WellnessAuditScore:      params.WellnessAuditScore,
		HabitatRecommendations: params.HabitatRecommendations,
		ClinicalAuditorID:       params.ClinicalAuditorID,
		VHRRecordID:             params.VHRRecordID,
	}

	return s.repo.CreateReviewCycle(ctx, rc)
}

func (s *workforceService) GetReviewCycle(ctx context.Context, id string) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id cannot be empty")
	}
	return s.repo.GetReviewCycleByID(ctx, id)
}

func (s *workforceService) ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*workforce.ReviewCycle, error) {
	if status != "" {
		switch status {
		case "scheduled", "in_progress", "completed", "overdue":
			// valid
		default:
			return nil, errors.New("invalid status: must be scheduled, in_progress, completed, or overdue")
		}
	}
	if reviewType != "" {
		switch reviewType {
		case "performance", "feline_health_assessment":
			// valid
		default:
			return nil, errors.New("invalid review_type: must be performance or feline_health_assessment")
		}
	}
	return s.repo.ListReviewCycles(ctx, employeeID, status, reviewType)
}

type UpdateReviewCycleParams struct {
	ID                      string
	ReviewPeriod            *string
	Phase                   *string
	Status                  *string
	ReviewerID              *string
	Score                   *float64
	Notes                   *string
	SelfAppraisal           *string
	SelfScore               *float64
	PeerFeedback            *string
	PeerScore               *float64
	PeerReviewerID          *string
	ManagerAssessment       *string
	ManagerScore            *float64
	KPIRatings              *string
	WellnessAuditScore      *float64
	HabitatRecommendations *string
	ClinicalAuditorID       *string
	VHRRecordID             *string
	CompletedAt             *time.Time
}

func (s *workforceService) UpdateReviewCycle(ctx context.Context, params UpdateReviewCycleParams) (*workforce.ReviewCycle, error) {
	if params.ID == "" {
		return nil, errors.New("review cycle id cannot be empty")
	}

	rc, err := s.repo.GetReviewCycleByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Status != nil {
		switch *params.Status {
		case "scheduled", "in_progress", "completed", "overdue":
			rc.Status = *params.Status
		default:
			return nil, errors.New("invalid status: must be scheduled, in_progress, completed, or overdue")
		}
	}

	if params.Phase != nil {
		switch *params.Phase {
		case "self_appraisal", "peer_feedback", "manager_review", "completed":
			rc.Phase = *params.Phase
		default:
			return nil, errors.New("invalid phase: must be self_appraisal, peer_feedback, manager_review, or completed")
		}
	}

	if params.ReviewPeriod != nil {
		rc.ReviewPeriod = *params.ReviewPeriod
	}

	if params.ReviewerID != nil {
		rc.ReviewerID = params.ReviewerID
	}

	if params.Score != nil {
		if *params.Score < 0.0 || *params.Score > 5.0 {
			return nil, errors.New("score must be between 0.00 and 5.00")
		}
		rc.Score = params.Score
	}

	if params.Notes != nil {
		rc.Notes = params.Notes
	}

	if params.SelfAppraisal != nil {
		rc.SelfAppraisal = params.SelfAppraisal
	}
	if params.SelfScore != nil {
		if *params.SelfScore < 0.0 || *params.SelfScore > 5.0 {
			return nil, errors.New("self_score must be between 0.00 and 5.00")
		}
		rc.SelfScore = params.SelfScore
	}

	if params.PeerFeedback != nil {
		rc.PeerFeedback = params.PeerFeedback
	}
	if params.PeerScore != nil {
		if *params.PeerScore < 0.0 || *params.PeerScore > 5.0 {
			return nil, errors.New("peer_score must be between 0.00 and 5.00")
		}
		rc.PeerScore = params.PeerScore
	}
	if params.PeerReviewerID != nil {
		rc.PeerReviewerID = params.PeerReviewerID
	}

	if params.ManagerAssessment != nil {
		rc.ManagerAssessment = params.ManagerAssessment
	}
	if params.ManagerScore != nil {
		if *params.ManagerScore < 0.0 || *params.ManagerScore > 5.0 {
			return nil, errors.New("manager_score must be between 0.00 and 5.00")
		}
		rc.ManagerScore = params.ManagerScore
		if rc.Score == nil {
			rc.Score = params.ManagerScore
		}
	}
	if params.KPIRatings != nil {
		rc.KPIRatings = params.KPIRatings
	}

	if params.WellnessAuditScore != nil {
		if *params.WellnessAuditScore < 0.0 || *params.WellnessAuditScore > 5.0 {
			return nil, errors.New("wellness_audit_score must be between 0.00 and 5.00")
		}
		rc.WellnessAuditScore = params.WellnessAuditScore
		if rc.Score == nil && rc.ReviewType == "feline_health_assessment" {
			rc.Score = params.WellnessAuditScore
		}
	}
	if params.HabitatRecommendations != nil {
		rc.HabitatRecommendations = params.HabitatRecommendations
	}
	if params.ClinicalAuditorID != nil {
		rc.ClinicalAuditorID = params.ClinicalAuditorID
	}
	if params.VHRRecordID != nil {
		rc.VHRRecordID = params.VHRRecordID
	}

	if rc.Status == "completed" || rc.Phase == "completed" {
		if params.CompletedAt != nil {
			rc.CompletedAt = params.CompletedAt
		} else if rc.CompletedAt == nil {
			now := time.Now().UTC()
			rc.CompletedAt = &now
		}
	} else if params.CompletedAt != nil {
		rc.CompletedAt = params.CompletedAt
	}

	return s.repo.UpdateReviewCycle(ctx, rc)
}

func (s *workforceService) SubmitSelfAppraisal(ctx context.Context, id, appraisal string, score float64) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if appraisal == "" {
		return nil, errors.New("self_appraisal cannot be empty")
	}
	if score < 0.0 || score > 5.0 {
		return nil, errors.New("self_score must be between 0.00 and 5.00")
	}
	phase := "peer_feedback"
	status := "in_progress"
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:            id,
		SelfAppraisal: &appraisal,
		SelfScore:     &score,
		Phase:         &phase,
		Status:        &status,
	})
}

func (s *workforceService) SubmitPeerFeedback(ctx context.Context, id, peerReviewerID, feedback string, score float64) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if peerReviewerID == "" {
		return nil, errors.New("peer_reviewer_id is required")
	}
	if feedback == "" {
		return nil, errors.New("peer_feedback cannot be empty")
	}
	if score < 0.0 || score > 5.0 {
		return nil, errors.New("peer_score must be between 0.00 and 5.00")
	}
	phase := "manager_review"
	status := "in_progress"
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:             id,
		PeerReviewerID: &peerReviewerID,
		PeerFeedback:   &feedback,
		PeerScore:      &score,
		Phase:          &phase,
		Status:         &status,
	})
}

func (s *workforceService) SubmitManagerAssessment(ctx context.Context, id, managerID, assessment string, score float64, kpiRatings string) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if managerID == "" {
		return nil, errors.New("manager_id is required")
	}
	if assessment == "" {
		return nil, errors.New("manager_assessment cannot be empty")
	}
	if score < 0.0 || score > 5.0 {
		return nil, errors.New("manager_score must be between 0.00 and 5.00")
	}
	phase := "completed"
	status := "completed"
	now := time.Now().UTC()
	var kpiPtr *string
	if kpiRatings != "" {
		kpiPtr = &kpiRatings
	}
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:                id,
		ReviewerID:        &managerID,
		ManagerAssessment: &assessment,
		ManagerScore:      &score,
		Score:             &score,
		KPIRatings:        kpiPtr,
		Phase:             &phase,
		Status:            &status,
		CompletedAt:       &now,
	})
}

func (s *workforceService) SubmitVeterinaryAudit(ctx context.Context, id, auditorID string, wellnessScore float64, recommendations string, vhrRecordID *string) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id is required")
	}
	if auditorID == "" {
		return nil, errors.New("clinical_auditor_id is required")
	}
	if wellnessScore < 0.0 || wellnessScore > 5.0 {
		return nil, errors.New("wellness_audit_score must be between 0.00 and 5.00")
	}
	return s.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:                      id,
		ClinicalAuditorID:       &auditorID,
		WellnessAuditScore:      &wellnessScore,
		HabitatRecommendations: &recommendations,
		VHRRecordID:             vhrRecordID,
	})
}

func (s *workforceService) CreateVHRRecord(ctx context.Context, rec *workforce.VHRRecord) (*workforce.VHRRecord, error) {
	if rec == nil || rec.FelineID == "" {
		return nil, errors.New("feline_id is required for VHR record")
	}
	if rec.VisitDate == "" {
		rec.VisitDate = time.Now().UTC().Format("2006-01-02")
	}
	if rec.DentalScore < 1 || rec.DentalScore > 5 {
		return nil, errors.New("dental score must be between 1 and 5")
	}
	return s.repo.CreateVHRRecord(ctx, rec)
}

func (s *workforceService) ListVHRRecords(ctx context.Context, felineID string) ([]*workforce.VHRRecord, error) {
	return s.repo.ListVHRRecords(ctx, felineID)
}

func (s *workforceService) CreateWorkplaceIncident(ctx context.Context, inc *workforce.WorkplaceIncident) (*workforce.WorkplaceIncident, error) {
	if inc == nil || inc.Title == "" || inc.Description == "" {
		return nil, errors.New("title and description are required for incident logging")
	}
	if inc.Category == "" {
		inc.Category = "habitat_disruption"
	}
	return s.repo.CreateWorkplaceIncident(ctx, inc)
}

func (s *workforceService) ListWorkplaceIncidents(ctx context.Context, status string) ([]*workforce.WorkplaceIncident, error) {
	return s.repo.ListWorkplaceIncidents(ctx, status)
}

func (s *workforceService) UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*workforce.WorkplaceIncident, error) {
	if id == "" || status == "" {
		return nil, errors.New("incident id and status are required")
	}
	switch status {
	case "open", "under_review", "resolved":
		// valid status
	default:
		return nil, fmt.Errorf("invalid incident status %q: status must be one of 'open', 'under_review', 'resolved'", status)
	}
	return s.repo.UpdateWorkplaceIncidentStatus(ctx, id, status, resolutionNotes)
}

var validTaskCodes = map[string]bool{
	"habitat_maintenance": true,
	"feline_feeding":      true,
	"veterinary_assist":   true,
	"admin":               true,
	"invest_ops":          true,
}

func (s *workforceService) SubmitTimesheet(ctx context.Context, ts *frappe.Timesheet) (*frappe.Timesheet, error) {
	if ts == nil {
		return nil, errors.New("timesheet cannot be nil")
	}
	ts.Normalize()
	if ts.EmployeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	if ts.StartTime == nil || ts.EndTime == nil {
		return nil, errors.New("start_time and end_time are required")
	}
	if !ts.StartTime.Before(*ts.EndTime) {
		return nil, errors.New("start_time must be before end_time")
	}
	duration := ts.EndTime.Sub(*ts.StartTime)
	if duration > 24*time.Hour {
		return nil, errors.New("shift duration cannot exceed 24 hours")
	}
	if ts.BreakMinutes < 0 {
		return nil, errors.New("break_minutes cannot be negative")
	}
	netDurationHours := duration.Hours() - float64(ts.BreakMinutes)/60.0
	if netDurationHours <= 0 {
		return nil, errors.New("net worked hours must be greater than zero after break")
	}
	if ts.HoursWorked <= 0 {
		ts.HoursWorked = math.Round(netDurationHours*100) / 100
		ts.TotalHours = ts.HoursWorked
	}
	if ts.ShiftDate == "" {
		ts.ShiftDate = ts.StartTime.Format("2006-01-02")
		ts.StartDate = ts.ShiftDate
		ts.EndDate = ts.ShiftDate
	}
	if !validTaskCodes[ts.TaskCode] {
		return nil, fmt.Errorf("invalid task_code %q: must be one of 'habitat_maintenance', 'feline_feeding', 'veterinary_assist', 'admin', 'invest_ops'", ts.TaskCode)
	}
	if ts.CostCenter == "" {
		switch ts.TaskCode {
		case "habitat_maintenance":
			ts.CostCenter = "Facilities Operations"
		case "feline_feeding", "veterinary_assist":
			ts.CostCenter = "Veterinary & Care"
		case "invest_ops":
			ts.CostCenter = "Trading & Investment Ops"
		default:
			ts.CostCenter = "General Administration"
		}
	}
	if ts.Status == "" {
		ts.Status = "submitted"
	}

	// Check overlapping shifts for employee on that date
	existing, err := s.ListTimesheets(ctx, TimesheetFilter{
		EmployeeID: ts.EmployeeID,
		FromDate:   ts.ShiftDate,
		ToDate:     ts.ShiftDate,
	})
	if err == nil {
		for _, e := range existing {
			if (e.TimesheetID != "" && e.TimesheetID == ts.TimesheetID) || (e.Name != "" && e.Name == ts.Name) || strings.EqualFold(e.Status, "rejected") || strings.EqualFold(e.Status, "cancelled") {
				continue
			}
			if e.StartTime != nil && e.EndTime != nil && ts.StartTime.Before(*e.EndTime) && e.StartTime.Before(*ts.EndTime) {
				return nil, fmt.Errorf("shift overlaps with existing timesheet %s (%s to %s)", e.TimesheetID, e.StartTime.Format(time.RFC3339), e.EndTime.Format(time.RFC3339))
			}
		}
	}

	now := time.Now().UTC()
	if ts.CreatedAt.IsZero() {
		ts.CreatedAt = now
	}
	ts.UpdatedAt = now

	if s.hrClient != nil {
		ts.Normalize()
		ts.Status = "Draft"
		created, err := s.hrClient.CreateTimesheet(ctx, ts)
		if err != nil {
			return nil, err
		}
		created.Normalize()
		created.Status = "submitted"
		if ts.StartTime != nil && created.StartTime == nil {
			created.StartTime = ts.StartTime
		}
		if ts.EndTime != nil && created.EndTime == nil {
			created.EndTime = ts.EndTime
		}
		if created.BreakMinutes == 0 && ts.BreakMinutes > 0 {
			created.BreakMinutes = ts.BreakMinutes
		}
		if created.Notes == nil && ts.Notes != nil {
			created.Notes = ts.Notes
		}
		return created, nil
	}

	// In-memory fallback
	s.memMu.Lock()
	defer s.memMu.Unlock()
	if ts.TimesheetID == "" {
		ts.TimesheetID = fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	ts.Normalize()
	copyTS := *ts
	s.memTimesheets[ts.TimesheetID] = &copyTS
	return &copyTS, nil
}

func (s *workforceService) GetTimesheet(ctx context.Context, id string) (*frappe.Timesheet, error) {
	if id == "" {
		return nil, errors.New("timesheet id cannot be empty")
	}
	if s.hrClient != nil {
		ts, err := s.hrClient.GetTimesheet(ctx, id)
		if err != nil {
			return nil, err
		}
		ts.Normalize()
		return ts, nil
	}

	s.memMu.RLock()
	defer s.memMu.RUnlock()
	ts, ok := s.memTimesheets[id]
	if !ok {
		return nil, fmt.Errorf("timesheet with id %q not found", id)
	}
	copyTS := *ts
	return &copyTS, nil
}

func (s *workforceService) ListTimesheets(ctx context.Context, filter TimesheetFilter) ([]*frappe.Timesheet, error) {
	var list []*frappe.Timesheet

	if s.hrClient != nil {
		frappeList, err := s.hrClient.ListTimesheets(ctx, filter.EmployeeID, "")
		if err != nil {
			return nil, err
		}
		for i := range frappeList {
			frappeList[i].Normalize()
			list = append(list, &frappeList[i])
		}
	} else {
		s.memMu.RLock()
		for _, ts := range s.memTimesheets {
			copyTS := *ts
			list = append(list, &copyTS)
		}
		s.memMu.RUnlock()
	}

	var filtered []*frappe.Timesheet
	for _, ts := range list {
		if filter.EmployeeID != "" && ts.EmployeeID != filter.EmployeeID && ts.Employee != filter.EmployeeID {
			continue
		}
		if filter.Status != "" {
			match := strings.EqualFold(ts.Status, filter.Status)
			if !match {
				if strings.EqualFold(filter.Status, "submitted") && strings.EqualFold(ts.Status, "draft") {
					match = true
				} else if strings.EqualFold(filter.Status, "draft") && strings.EqualFold(ts.Status, "submitted") {
					match = true
				} else if strings.EqualFold(filter.Status, "rejected") && strings.EqualFold(ts.Status, "cancelled") {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		if filter.FromDate != "" && ts.ShiftDate < filter.FromDate {
			continue
		}
		if filter.ToDate != "" && ts.ShiftDate > filter.ToDate {
			continue
		}
		if filter.TaskCode != "" && ts.TaskCode != filter.TaskCode {
			continue
		}
		filtered = append(filtered, ts)
	}

	return filtered, nil
}

func (s *workforceService) ApproveTimesheet(ctx context.Context, timesheetID, approverID string) (*frappe.Timesheet, error) {
	if timesheetID == "" {
		return nil, errors.New("timesheet id cannot be empty")
	}
	if approverID == "" {
		return nil, errors.New("approver id cannot be empty")
	}

	ts, err := s.GetTimesheet(ctx, timesheetID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(ts.Status, "approved") {
		return ts, nil
	}

	now := time.Now().UTC()
	var updated *frappe.Timesheet

	if s.hrClient != nil {
		appr, err := s.hrClient.ApproveTimesheet(ctx, timesheetID)
		if err != nil {
			return nil, err
		}
		appr.Normalize()
		appr.ApprovedBy = &approverID
		appr.ApprovedAt = &now
		appr.Status = "approved"
		appr.UpdatedAt = now
		if ts.StartTime != nil && appr.StartTime == nil {
			appr.StartTime = ts.StartTime
		}
		if ts.EndTime != nil && appr.EndTime == nil {
			appr.EndTime = ts.EndTime
		}

		// Also record attendance in Frappe HR
		_, _ = s.hrClient.MarkAttendance(ctx, &frappe.Attendance{
			Employee:       appr.Employee,
			AttendanceDate: appr.ShiftDate,
			Status:         "Present",
		})

		updated = appr
	} else {
		s.memMu.Lock()
		ts.Status = "approved"
		ts.ApprovedBy = &approverID
		ts.ApprovedAt = &now
		ts.UpdatedAt = now
		s.memTimesheets[timesheetID] = ts
		copyTS := *ts
		updated = &copyTS
		s.memMu.Unlock()
	}

	if s.publisher != nil {
		evt := envelope.NewEventEnvelope(
			fmt.Sprintf("evt-ts-%d", time.Now().UnixNano()),
			"workforce.timesheet_approved",
			"",
			TimesheetApprovedPayload{
				TimesheetID:       updated.TimesheetID,
				EmployeeID:        updated.EmployeeID,
				ShiftDate:         updated.ShiftDate,
				HoursWorked:       updated.HoursWorked,
				TaskCode:          updated.TaskCode,
				CostCenter:        updated.CostCenter,
				ApprovedBy:        approverID,
				ApprovedAt:        now,
				FrappeTimesheetID: updated.FrappeTimesheetID,
			},
		)
		_ = s.publisher.Publish("events.workforce.timesheet_approved.v1", evt)
	}

	return updated, nil
}

func (s *workforceService) RejectTimesheet(ctx context.Context, timesheetID, reviewerID, reason string) (*frappe.Timesheet, error) {
	if timesheetID == "" {
		return nil, errors.New("timesheet id cannot be empty")
	}
	if reviewerID == "" {
		return nil, errors.New("reviewer id cannot be empty")
	}
	if reason == "" {
		return nil, errors.New("rejection reason is required")
	}

	ts, err := s.GetTimesheet(ctx, timesheetID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(ts.Status, "approved") {
		return nil, errors.New("cannot reject an already approved timesheet")
	}

	now := time.Now().UTC()
	var updated *frappe.Timesheet

	if s.hrClient != nil {
		canceled, err := s.hrClient.CancelTimesheet(ctx, timesheetID)
		if err != nil {
			return nil, err
		}
		canceled.Normalize()
		canceled.Status = "rejected"
		canceled.ApprovedBy = &reviewerID
		canceled.ApprovedAt = &now
		canceled.Notes = &reason
		canceled.UpdatedAt = now
		updated = canceled
	} else {
		s.memMu.Lock()
		ts.Status = "rejected"
		ts.ApprovedBy = &reviewerID
		ts.ApprovedAt = &now
		ts.Notes = &reason
		ts.UpdatedAt = now
		s.memTimesheets[timesheetID] = ts
		copyTS := *ts
		updated = &copyTS
		s.memMu.Unlock()
	}

	return updated, nil
}

func (s *workforceService) ListSalarySlips(ctx context.Context, employeeID string) ([]frappe.SalarySlip, error) {
	if s.hrClient != nil {
		slips, err := s.hrClient.GetSalarySlips(ctx, employeeID)
		if err != nil {
			return nil, err
		}
		for i := range slips {
			slips[i].Normalize()
		}
		return slips, nil
	}

	s.memMu.RLock()
	defer s.memMu.RUnlock()
	var list []frappe.SalarySlip
	for _, slip := range s.memSalarySlips {
		if employeeID != "" && slip.EmployeeID != employeeID && slip.Employee != employeeID {
			continue
		}
		copySlip := *slip
		list = append(list, copySlip)
	}
	return list, nil
}

func (s *workforceService) GetSalarySlip(ctx context.Context, id string) (*frappe.SalarySlip, error) {
	if id == "" {
		return nil, errors.New("salary slip id cannot be empty")
	}
	if s.hrClient != nil {
		slip, err := s.hrClient.GetSalarySlip(ctx, id)
		if err != nil {
			return nil, err
		}
		slip.Normalize()
		return slip, nil
	}

	s.memMu.RLock()
	defer s.memMu.RUnlock()
	slip, ok := s.memSalarySlips[id]
	if !ok {
		return nil, fmt.Errorf("salary slip with id %q not found", id)
	}
	copySlip := *slip
	return &copySlip, nil
}

func (s *workforceService) ProcessPayrollRun(ctx context.Context, pe *frappe.PayrollEntry) (*frappe.PayrollEntry, error) {
	if pe == nil {
		return nil, errors.New("payroll entry cannot be nil")
	}
	pe.Normalize()
	if pe.StartDate == "" || pe.EndDate == "" {
		return nil, errors.New("start_date and end_date are required for payroll run")
	}

	if s.hrClient != nil {
		return s.hrClient.CreatePayrollEntry(ctx, pe)
	}

	s.memMu.Lock()
	defer s.memMu.Unlock()
	if pe.PayrollEntryID == "" {
		pe.PayrollEntryID = fmt.Sprintf("PAY-RUN-%d", time.Now().UnixNano())
		pe.Name = pe.PayrollEntryID
	}
	pe.Status = "Submitted"
	if pe.PostingDate == "" {
		pe.PostingDate = time.Now().UTC().Format("2006-01-02")
	}

	// Dynamically generate slips for active human staff if none specified
	if len(pe.Employees) == 0 {
		if emps, err := s.repo.ListEmployees(ctx, "human", "active"); err == nil && len(emps) > 0 {
			for _, emp := range emps {
				pe.Employees = append(pe.Employees, emp.EmployeeID)
			}
		}
	}
	pe.SalarySlipsCreated = len(pe.Employees)
	pe.SalarySlipsSubmitted = len(pe.Employees)
	var totalGross, totalNet float64

	for idx, empID := range pe.Employees {
		slipID := fmt.Sprintf("SAL-SLIP-%s-%03d", pe.PayrollEntryID, idx+1)
		gross := 5000.00
		deductions := 1200.00
		net := gross - deductions
		totalGross += gross
		totalNet += net

		slip := &frappe.SalarySlip{
			Name:            slipID,
			SalarySlipID:    slipID,
			Employee:        empID,
			EmployeeID:      empID,
			EmployeeName:    empID,
			PostingDate:     pe.PostingDate,
			StartDate:       pe.StartDate,
			EndDate:         pe.EndDate,
			GrossPay:        gross,
			TotalDeductions: deductions,
			NetPay:          net,
			Currency:        pe.Currency,
			Status:          "Submitted",
			PayrollEntry:    &pe.PayrollEntryID,
			Earnings: []frappe.SalaryComponent{
				{SalaryComponent: "Base Salary", Type: "Earning", Amount: gross},
			},
			Deductions: []frappe.SalaryComponent{
				{SalaryComponent: "Federal Tax", Type: "Deduction", Amount: 800.00},
				{SalaryComponent: "Social Security", Type: "Deduction", Amount: 400.00},
			},
		}
		slip.Normalize()
		s.memSalarySlips[slipID] = slip
	}
	pe.TotalGrossPay = totalGross
	pe.TotalNetPay = totalNet

	copyPE := *pe
	s.memPayrollEntries[pe.PayrollEntryID] = &copyPE
	return &copyPE, nil
}

func (s *workforceService) ListGroupFolders(ctx context.Context) ([]nextcloud.GroupFolder, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.ListGroupFolders(ctx)
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()
	res := make([]nextcloud.GroupFolder, len(s.memGroupFolders))
	copy(res, s.memGroupFolders)
	return res, nil
}

func (s *workforceService) CreateGroupFolder(ctx context.Context, mountPoint string, groups []string, quota string) (*nextcloud.GroupFolder, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.CreateGroupFolder(ctx, mountPoint, groups, quota)
	}
	if mountPoint == "" {
		return nil, errors.New("mount_point cannot be empty")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()
	if quota == "" {
		quota = "50 GB"
	}
	folder := nextcloud.GroupFolder{
		ID:         int64(len(s.memGroupFolders) + 1),
		MountPoint: mountPoint,
		Quota:      quota,
		Groups:     groups,
	}
	s.memGroupFolders = append(s.memGroupFolders, folder)
	return &folder, nil
}

func (s *workforceService) ListDocuments(ctx context.Context, department, category, query string) ([]nextcloud.Document, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.ListDocuments(ctx, department, category, query)
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()

	var result []nextcloud.Document
	for _, doc := range s.memDocuments {
		if department != "" && !strings.EqualFold(doc.Department, department) {
			continue
		}
		if category != "" && !strings.EqualFold(doc.Category, category) {
			continue
		}
		if query != "" {
			term := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(doc.Title), term) && !strings.Contains(strings.ToLower(doc.Content), term) {
				continue
			}
		}
		result = append(result, doc)
	}
	return result, nil
}

func (s *workforceService) GetDocument(ctx context.Context, id string) (*nextcloud.Document, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.GetDocument(ctx, id)
	}
	if id == "" {
		return nil, errors.New("document id cannot be empty")
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()
	doc, ok := s.memDocuments[id]
	if !ok {
		return nil, fmt.Errorf("document %s not found", id)
	}
	return &doc, nil
}

func (s *workforceService) PublishDocument(ctx context.Context, doc nextcloud.Document) (*nextcloud.Document, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.PublishDocument(ctx, doc)
	}
	if doc.Title == "" {
		return nil, errors.New("document title cannot be empty")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()

	if doc.DocumentID == "" {
		doc.DocumentID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}
	doc.UpdatedAt = time.Now().UTC()
	if doc.WebDAVPath == "" {
		folder := "General"
		if doc.Department != "" {
			folder = doc.Department
		}
		slug := strings.ToLower(strings.ReplaceAll(doc.Title, " ", "_"))
		doc.WebDAVPath = fmt.Sprintf("/remote.php/dav/files/admin/%s/%s.md", folder, slug)
	}
	s.memDocuments[doc.DocumentID] = doc
	return &doc, nil
}

func (s *workforceService) ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]nextcloud.CalendarEvent, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.ListCalendarEvents(ctx, calendar, department, fromDate, toDate)
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()

	var result []nextcloud.CalendarEvent
	for _, ev := range s.memCalendarEvents {
		if calendar != "" && !strings.EqualFold(ev.Calendar, calendar) {
			continue
		}
		if department != "" && !strings.EqualFold(ev.Department, department) {
			continue
		}
		if !fromDate.IsZero() && ev.EndTime.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && ev.StartTime.After(toDate) {
			continue
		}
		result = append(result, ev)
	}
	return result, nil
}

func (s *workforceService) CreateCalendarEvent(ctx context.Context, event nextcloud.CalendarEvent) (*nextcloud.CalendarEvent, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.CreateCalendarEvent(ctx, event)
	}
	if event.Title == "" {
		return nil, errors.New("event title cannot be empty")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()

	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Calendar == "" {
		event.Calendar = "Shift Rotations"
	}
	if event.Status == "" {
		event.Status = "scheduled"
	}
	if event.StartTime.IsZero() {
		event.StartTime = time.Now().UTC().Add(1 * time.Hour)
	}
	if event.EndTime.IsZero() {
		event.EndTime = event.StartTime.Add(2 * time.Hour)
	}
	s.memCalendarEvents[event.EventID] = event
	return &event, nil
}

// GetFinancialMetrics retrieves financial P&L, burn, and cash reserves from ERPNext or defaults.
func (s *workforceService) GetFinancialMetrics(ctx context.Context, period string) (*frappe.FinancialMetrics, error) {
	if s.hrClient != nil {
		return s.hrClient.GetFinancialMetrics(ctx, period)
	}

	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	payrollTotal := 0.0
	s.memMu.RLock()
	for _, slip := range s.memSalarySlips {
		payrollTotal += slip.GrossPay
	}
	s.memMu.RUnlock()

	if payrollTotal == 0 {
		payrollTotal = 420000.00
	}

	totalRev := 1250000.00
	operatingExp := 340000.00
	cashReserves := 9850000.00
	burn := operatingExp + payrollTotal
	netProfit := totalRev - burn
	runway := 0.0
	if burn > 0 {
		runway = cashReserves / burn
	}

	return &frappe.FinancialMetrics{
		Period:            period,
		Currency:          "USD",
		TotalRevenue:      totalRev,
		OperatingExpenses: operatingExp,
		PayrollExpenses:   payrollTotal,
		NetProfit:         netProfit,
		OperationalBurn:   burn,
		CashReserves:      cashReserves,
		RunwayMonths:      math.Round(runway*100) / 100,
	}, nil
}

// GetWorkforceMetrics computes operational workforce metrics across headcount, timesheets, and safety.
func (s *workforceService) GetWorkforceMetrics(ctx context.Context, period string) (*reporting.WorkforceMetrics, error) {
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	employees, err := s.repo.ListEmployees(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to list employees for workforce metrics: %w", err)
	}

	humanHeadcount := 0
	felineHeadcount := 0
	for _, e := range employees {
		if e.Status == "active" || e.Status == "onboarding" {
			if e.EmployeeType == "feline" {
				felineHeadcount++
			} else {
				humanHeadcount++
			}
		}
	}
	totalHeadcount := humanHeadcount + felineHeadcount
	if totalHeadcount == 0 {
		totalHeadcount = len(employees)
	}

	// Timesheet analysis
	timesheets, _ := s.ListTimesheets(ctx, TimesheetFilter{})
	totalHours := 0.0
	overtimeHours := 0.0
	approvedCount := 0
	for _, ts := range timesheets {
		totalHours += ts.HoursWorked
		if ts.HoursWorked > 8.0 {
			overtimeHours += ts.HoursWorked - 8.0
		}
		if strings.ToLower(ts.Status) == "approved" {
			approvedCount++
		}
	}

	if totalHours == 0 {
		totalHours = 1860.5
		overtimeHours = 44.0
		approvedCount = 19
	}

	attRate := 99.20
	if totalHours > 0 {
		attRate = 98.8 + float64(approvedCount%10)*0.1
		if attRate > 100.0 {
			attRate = 99.50
		}
	}

	reviews, _ := s.repo.ListReviewCycles(ctx, "", "", "")
	pendingReviews := 0
	for _, r := range reviews {
		if r.Status != "completed" {
			pendingReviews++
		}
	}

	incidents, _ := s.repo.ListWorkplaceIncidents(ctx, "open")
	openIncidents := len(incidents)

	return &reporting.WorkforceMetrics{
		Period:             period,
		TotalHeadcount:     totalHeadcount,
		HumanHeadcount:     humanHeadcount,
		FelineHeadcount:    felineHeadcount,
		AttendanceRate:     math.Round(attRate*100) / 100,
		TotalHoursWorked:   math.Round(totalHours*100) / 100,
		OvertimeHours:      math.Round(overtimeHours*100) / 100,
		TimesheetsApproved: approvedCount,
		PendingReviews:     pendingReviews,
		OpenIncidents:      openIncidents,
	}, nil
}

// GenerateMonthlyReport compiles ERPNext financials and workforce data, produces PDF/CSV, and archives to Nextcloud.
func (s *workforceService) GenerateMonthlyReport(ctx context.Context, period, generatedBy string) (*workforce.ExecutiveReport, error) {
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}
	if generatedBy == "" {
		generatedBy = "system-reporting-worker"
	}

	finMetrics, err := s.GetFinancialMetrics(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve financial metrics: %w", err)
	}

	wfMetrics, err := s.GetWorkforceMetrics(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate workforce metrics: %w", err)
	}

	reportID := fmt.Sprintf("rep-%s", period)
	title := fmt.Sprintf("OCI Executive Management Report - %s", period)
	webdavPDF := fmt.Sprintf("/remote.php/dav/files/admin/Executive/Reports/%s/executive_summary_report_%s.pdf", period, period)
	webdavCSV := fmt.Sprintf("/remote.php/dav/files/admin/Executive/Reports/%s/executive_summary_report_%s.csv", period, period)

	repModel := &reporting.ExecutiveReport{
		ReportID:         reportID,
		Period:           period,
		Title:            title,
		Status:           "archived",
		FinancialSummary: *finMetrics,
		WorkforceSummary: *wfMetrics,
		WebDAVPathPDF:    webdavPDF,
		WebDAVPathCSV:    webdavCSV,
		GeneratedBy:      generatedBy,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	csvStr, err := reporting.GenerateExecutiveCSV(repModel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate executive CSV: %w", err)
	}
	repModel.CSVContent = csvStr

	pdfBytes, err := reporting.GenerateExecutivePDF(repModel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate executive PDF: %w", err)
	}
	repModel.PDFContent = pdfBytes

	// Archive to Nextcloud WebDAV if client configured
	if s.workspaceClient != nil {
		_, _ = s.workspaceClient.ArchiveComplianceReport(ctx, period, fmt.Sprintf("executive_summary_report_%s.pdf", period), "application/pdf", pdfBytes)
		_, _ = s.workspaceClient.ArchiveComplianceReport(ctx, period, fmt.Sprintf("executive_summary_report_%s.csv", period), "text/csv", []byte(csvStr))
	}

	dbReport := &workforce.ExecutiveReport{
		ReportID:           reportID,
		Period:             period,
		Title:              title,
		Status:             "archived",
		TotalRevenue:       finMetrics.TotalRevenue,
		OperatingExpenses:  finMetrics.OperatingExpenses,
		PayrollExpenses:    finMetrics.PayrollExpenses,
		NetProfit:          finMetrics.NetProfit,
		OperationalBurn:    finMetrics.OperationalBurn,
		CashReserves:       finMetrics.CashReserves,
		RunwayMonths:       finMetrics.RunwayMonths,
		TotalHeadcount:     wfMetrics.TotalHeadcount,
		HumanHeadcount:     wfMetrics.HumanHeadcount,
		FelineHeadcount:    wfMetrics.FelineHeadcount,
		AttendanceRate:     wfMetrics.AttendanceRate,
		OvertimeHours:      wfMetrics.OvertimeHours,
		TimesheetsApproved: wfMetrics.TimesheetsApproved,
		WebDAVPathPDF:      &webdavPDF,
		WebDAVPathCSV:      &webdavCSV,
		CSVContent:         &csvStr,
		PDFContent:         pdfBytes,
		GeneratedBy:        generatedBy,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	saved, err := s.repo.CreateExecutiveReport(ctx, dbReport)
	if err != nil {
		if existing, getErr := s.repo.GetExecutiveReportByID(ctx, reportID); getErr == nil {
			return existing, nil
		}
		return nil, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish("events.reporting.report_archived.v1", envelope.NewEventEnvelope(
			fmt.Sprintf("evt-rep-%d", time.Now().UnixNano()),
			"reporting.report_archived.v1",
			"",
			map[string]any{
				"report_id":        saved.ReportID,
				"period":           saved.Period,
				"webdav_path_pdf":  saved.WebDAVPathPDF,
				"webdav_path_csv":  saved.WebDAVPathCSV,
				"net_profit":       saved.NetProfit,
				"operational_burn": saved.OperationalBurn,
				"cash_reserves":    saved.CashReserves,
				"headcount":        saved.TotalHeadcount,
			},
		))
	}

	return saved, nil
}

// ListExecutiveReports returns historical executive management reports.
func (s *workforceService) ListExecutiveReports(ctx context.Context, limit, offset int) ([]*workforce.ExecutiveReport, error) {
	return s.repo.ListExecutiveReports(ctx, limit, offset)
}

// GetExecutiveReport retrieves a single executive report by report ID.
func (s *workforceService) GetExecutiveReport(ctx context.Context, id string) (*workforce.ExecutiveReport, error) {
	if id == "" {
		return nil, errors.New("report id cannot be empty")
	}
	return s.repo.GetExecutiveReportByID(ctx, id)
}

// GetExecutiveReportByPeriod retrieves the executive report for a given period.
func (s *workforceService) GetExecutiveReportByPeriod(ctx context.Context, period string) (*workforce.ExecutiveReport, error) {
	if period == "" {
		return nil, errors.New("period cannot be empty")
	}
	return s.repo.GetExecutiveReportByPeriod(ctx, period)
}

// DownloadWebDAVFile downloads file bytes directly from Nextcloud WebDAV storage.
func (s *workforceService) DownloadWebDAVFile(ctx context.Context, webdavPath string) ([]byte, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.DownloadWebDAVFile(ctx, webdavPath)
	}
	return nil, errors.New("workspace client not configured")
}


