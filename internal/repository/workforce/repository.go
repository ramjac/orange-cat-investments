package workforce

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Employee struct {
	EmployeeID   string    `json:"employee_id"`
	EmployeeType string    `json:"employee_type"`
	FirstName    string    `json:"first_name"`
	LastName     *string   `json:"last_name,omitempty"`
	Email        *string   `json:"email,omitempty"`
	RoleTitle    string    `json:"role_title"`
	Department   string    `json:"department"`
	Status       string    `json:"status"`
	HiredAt      time.Time `json:"hired_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type OnboardingTask struct {
	ChecklistID string     `json:"checklist_id"`
	EmployeeID  string     `json:"employee_id"`
	TaskName    string     `json:"task_name"`
	Category    string     `json:"category"`
	IsCompleted bool       `json:"is_completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	AssignedTo  *string    `json:"assigned_to,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CareSchedule struct {
	ScheduleID           string    `json:"schedule_id"`
	FelineID             string    `json:"feline_id"`
	DietaryPlan          string    `json:"dietary_plan"`
	FeedingTimes         []string  `json:"feeding_times"`
	SpecialMedicalNeeds  *string   `json:"special_medical_needs,omitempty"`
	PreferredPerchZone   *string   `json:"preferred_perch_zone,omitempty"`
	EmergencyMedicalHold bool      `json:"emergency_medical_hold"`
	CaretakerID          *string   `json:"caretaker_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type LeaveRequest struct {
	LeaveID    string    `json:"leave_id"`
	EmployeeID string    `json:"employee_id"`
	LeaveType  string    `json:"leave_type"`
	StartDate  string    `json:"start_date"`
	EndDate    string    `json:"end_date"`
	Status     string    `json:"status"`
	Reason     *string   `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ReviewCycle struct {
	ReviewID               string     `json:"review_id"`
	EmployeeID             string     `json:"employee_id"`
	ReviewType             string     `json:"review_type"`
	ReviewPeriod           string     `json:"review_period"`
	Phase                  string     `json:"phase"`
	ScheduledFor           string     `json:"scheduled_for"`
	Status                 string     `json:"status"`
	ReviewerID             *string    `json:"reviewer_id,omitempty"`
	Score                  *float64   `json:"score,omitempty"`
	Notes                  *string    `json:"notes,omitempty"`
	SelfAppraisal          *string    `json:"self_appraisal,omitempty"`
	SelfScore              *float64   `json:"self_score,omitempty"`
	PeerFeedback           *string    `json:"peer_feedback,omitempty"`
	PeerScore              *float64   `json:"peer_score,omitempty"`
	PeerReviewerID         *string    `json:"peer_reviewer_id,omitempty"`
	ManagerAssessment      *string    `json:"manager_assessment,omitempty"`
	ManagerScore           *float64   `json:"manager_score,omitempty"`
	KPIRatings             *string    `json:"kpi_ratings,omitempty"`
	WellnessAuditScore     *float64   `json:"wellness_audit_score,omitempty"`
	HabitatRecommendations *string    `json:"habitat_recommendations,omitempty"`
	ClinicalAuditorID      *string    `json:"clinical_auditor_id,omitempty"`
	VHRRecordID            *string    `json:"vhr_record_id,omitempty"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type VHRRecord struct {
	RecordID          string    `json:"record_id"`
	FelineID          string    `json:"feline_id"`
	VeterinarianID    *string   `json:"veterinarian_id,omitempty"`
	VisitDate         string    `json:"visit_date"`
	WeightKG          float64   `json:"weight_kg"`
	DentalScore       int       `json:"dental_score"`
	VaccinationStatus string    `json:"vaccination_status"`
	Prescriptions     *string   `json:"prescriptions,omitempty"`
	ClinicalNotes     string    `json:"clinical_notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type WorkplaceIncident struct {
	IncidentID       string     `json:"incident_id"`
	Title            string     `json:"title"`
	Category         string     `json:"category"`
	InvolvedFelineID *string    `json:"involved_feline_id,omitempty"`
	InvolvedHumanID  *string    `json:"involved_human_id,omitempty"`
	Severity         string     `json:"severity"`
	Status           string     `json:"status"`
	Description      string     `json:"description"`
	ResolutionNotes  *string    `json:"resolution_notes,omitempty"`
	ReportedAt       time.Time  `json:"reported_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

type ExecutiveReport struct {
	ReportID           string    `json:"report_id"`
	Period             string    `json:"period"`
	Title              string    `json:"title"`
	Status             string    `json:"status"`
	TotalRevenue       float64   `json:"total_revenue"`
	OperatingExpenses  float64   `json:"operating_expenses"`
	PayrollExpenses    float64   `json:"payroll_expenses"`
	NetProfit          float64   `json:"net_profit"`
	OperationalBurn    float64   `json:"operational_burn"`
	CashReserves       float64   `json:"cash_reserves"`
	RunwayMonths       float64   `json:"runway_months"`
	TotalHeadcount     int       `json:"total_headcount"`
	HumanHeadcount     int       `json:"human_headcount"`
	FelineHeadcount    int       `json:"feline_headcount"`
	AttendanceRate     float64   `json:"attendance_rate"`
	OvertimeHours      float64   `json:"overtime_hours"`
	TimesheetsApproved int       `json:"timesheets_approved"`
	WebDAVPathPDF      *string   `json:"webdav_path_pdf,omitempty"`
	WebDAVPathCSV      *string   `json:"webdav_path_csv,omitempty"`
	CSVContent         *string   `json:"csv_content,omitempty"`
	PDFContent         []byte    `json:"pdf_content,omitempty"`
	GeneratedBy        string    `json:"generated_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type EmployeeRepository interface {
	GetEmployeeByID(ctx context.Context, id string) (*Employee, error)
	ListEmployees(ctx context.Context, employeeType, status string) ([]*Employee, error)
	CreateEmployee(ctx context.Context, emp *Employee) (*Employee, error)
	UpdateEmployeeStatus(ctx context.Context, id, status string) (*Employee, error)
	CreateOnboardingTask(ctx context.Context, task *OnboardingTask) (*OnboardingTask, error)
	ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*OnboardingTask, error)
	UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*OnboardingTask, error)
}

type CareVHRRepository interface {
	GetCareScheduleByFelineID(ctx context.Context, felineID string) (*CareSchedule, error)
	UpsertCareSchedule(ctx context.Context, cs *CareSchedule) (*CareSchedule, error)
	SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*CareSchedule, error)
	CreateVHRRecord(ctx context.Context, rec *VHRRecord) (*VHRRecord, error)
	ListVHRRecords(ctx context.Context, felineID string) ([]*VHRRecord, error)
}

type LeaveIncidentRepository interface {
	CreateLeaveRequest(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error)
	GetLeaveRequestByID(ctx context.Context, id string) (*LeaveRequest, error)
	ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*LeaveRequest, error)
	UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*LeaveRequest, error)
	CreateWorkplaceIncident(ctx context.Context, inc *WorkplaceIncident) (*WorkplaceIncident, error)
	ListWorkplaceIncidents(ctx context.Context, status string) ([]*WorkplaceIncident, error)
	UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*WorkplaceIncident, error)
}

type ReviewRepository interface {
	CreateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error)
	GetReviewCycleByID(ctx context.Context, id string) (*ReviewCycle, error)
	ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*ReviewCycle, error)
	UpdateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error)
}

type ExecutiveReportRepository interface {
	CreateExecutiveReport(ctx context.Context, report *ExecutiveReport) (*ExecutiveReport, error)
	GetExecutiveReportByID(ctx context.Context, id string) (*ExecutiveReport, error)
	GetExecutiveReportByPeriod(ctx context.Context, period string) (*ExecutiveReport, error)
	ListExecutiveReports(ctx context.Context, limit, offset int) ([]*ExecutiveReport, error)
	UpdateExecutiveReportStatus(ctx context.Context, id, status string, webdavPDF, webdavCSV *string) (*ExecutiveReport, error)
}

type Repository interface {
	EmployeeRepository
	CareVHRRepository
	LeaveIncidentRepository
	ReviewRepository
	ExecutiveReportRepository
}

type pgxRepository struct {
	db *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed workforce.Repository.
func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepository{db: db}
}

func strPtr(s string) *string {
	return &s
}
