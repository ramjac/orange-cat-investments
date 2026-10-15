package workforce

import (
	"context"
	"fmt"
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
	ReviewID                string     `json:"review_id"`
	EmployeeID              string     `json:"employee_id"`
	ReviewType              string     `json:"review_type"`
	ReviewPeriod            string     `json:"review_period"`
	Phase                   string     `json:"phase"`
	ScheduledFor            string     `json:"scheduled_for"`
	Status                  string     `json:"status"`
	ReviewerID              *string    `json:"reviewer_id,omitempty"`
	Score                   *float64   `json:"score,omitempty"`
	Notes                   *string    `json:"notes,omitempty"`
	SelfAppraisal           *string    `json:"self_appraisal,omitempty"`
	SelfScore               *float64   `json:"self_score,omitempty"`
	PeerFeedback            *string    `json:"peer_feedback,omitempty"`
	PeerScore               *float64   `json:"peer_score,omitempty"`
	PeerReviewerID          *string    `json:"peer_reviewer_id,omitempty"`
	ManagerAssessment       *string    `json:"manager_assessment,omitempty"`
	ManagerScore            *float64   `json:"manager_score,omitempty"`
	KPIRatings              *string    `json:"kpi_ratings,omitempty"`
	WellnessAuditScore      *float64   `json:"wellness_audit_score,omitempty"`
	HabitatRecommendations *string    `json:"habitat_recommendations,omitempty"`
	ClinicalAuditorID       *string    `json:"clinical_auditor_id,omitempty"`
	VHRRecordID             *string    `json:"vhr_record_id,omitempty"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
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

type Repository interface {
	GetEmployeeByID(ctx context.Context, id string) (*Employee, error)
	ListEmployees(ctx context.Context, employeeType, status string) ([]*Employee, error)
	CreateEmployee(ctx context.Context, emp *Employee) (*Employee, error)
	UpdateEmployeeStatus(ctx context.Context, id, status string) (*Employee, error)
	CreateOnboardingTask(ctx context.Context, task *OnboardingTask) (*OnboardingTask, error)
	ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*OnboardingTask, error)
	UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*OnboardingTask, error)
	GetCareScheduleByFelineID(ctx context.Context, felineID string) (*CareSchedule, error)
	UpsertCareSchedule(ctx context.Context, cs *CareSchedule) (*CareSchedule, error)
	SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*CareSchedule, error)

	// Leave Requests
	CreateLeaveRequest(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error)
	GetLeaveRequestByID(ctx context.Context, id string) (*LeaveRequest, error)
	ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*LeaveRequest, error)
	UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*LeaveRequest, error)

	// Review Cycles
	CreateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error)
	GetReviewCycleByID(ctx context.Context, id string) (*ReviewCycle, error)
	ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*ReviewCycle, error)
	UpdateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error)

	// Electronic Veterinary Health Records (VHR)
	CreateVHRRecord(ctx context.Context, rec *VHRRecord) (*VHRRecord, error)
	ListVHRRecords(ctx context.Context, felineID string) ([]*VHRRecord, error)

	// Workplace Incidents
	CreateWorkplaceIncident(ctx context.Context, inc *WorkplaceIncident) (*WorkplaceIncident, error)
	ListWorkplaceIncidents(ctx context.Context, status string) ([]*WorkplaceIncident, error)
	UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*WorkplaceIncident, error)

	// Executive Management Reports
	CreateExecutiveReport(ctx context.Context, report *ExecutiveReport) (*ExecutiveReport, error)
	GetExecutiveReportByID(ctx context.Context, id string) (*ExecutiveReport, error)
	GetExecutiveReportByPeriod(ctx context.Context, period string) (*ExecutiveReport, error)
	ListExecutiveReports(ctx context.Context, limit, offset int) ([]*ExecutiveReport, error)
	UpdateExecutiveReportStatus(ctx context.Context, id, status string, webdavPDF, webdavCSV *string) (*ExecutiveReport, error)
}

type pgxRepository struct {
	db *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed workforce.Repository.
// If db is nil, it delegates to NewMockRepository() for test and mock execution.
func NewRepository(db *pgxpool.Pool) Repository {
	if db == nil {
		return NewMockRepository()
	}
	return &pgxRepository{db: db}
}

func strPtr(s string) *string {
	return &s
}

func (r *pgxRepository) GetEmployeeByID(ctx context.Context, id string) (*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
	          FROM workforce.employees WHERE employee_id::text = $1`
	var emp Employee
	err := r.db.QueryRow(ctx, query, id).Scan(
		&emp.EmployeeID, &emp.EmployeeType, &emp.FirstName, &emp.LastName, &emp.Email,
		&emp.RoleTitle, &emp.Department, &emp.Status, &emp.HiredAt, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("employee with id %s not found: %w", id, err)
	}
	return &emp, nil
}

func (r *pgxRepository) ListEmployees(ctx context.Context, employeeType, status string) ([]*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at
	          FROM workforce.employees
	          WHERE ($1 = '' OR employee_type = $1)
	            AND ($2 = '' OR status = $2)
	          ORDER BY hired_at ASC, created_at ASC`
	rows, err := r.db.Query(ctx, query, employeeType, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Employee
	for rows.Next() {
		var emp Employee
		if err := rows.Scan(
			&emp.EmployeeID, &emp.EmployeeType, &emp.FirstName, &emp.LastName, &emp.Email,
			&emp.RoleTitle, &emp.Department, &emp.Status, &emp.HiredAt, &emp.CreatedAt, &emp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &emp)
	}
	if list == nil {
		list = []*Employee{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) CreateEmployee(ctx context.Context, emp *Employee) (*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.employees (
		employee_id, employee_type, first_name, last_name, email, role_title, department, status, hired_at
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4, $5, $6, $7,
		COALESCE(NULLIF($8, ''), 'onboarding'),
		COALESCE(NULLIF($9::timestamptz, '0001-01-01 00:00:00+00'::timestamptz), CURRENT_TIMESTAMP)
	) RETURNING employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at`
	var res Employee
	err := r.db.QueryRow(ctx, query,
		emp.EmployeeID, emp.EmployeeType, emp.FirstName, emp.LastName, emp.Email,
		emp.RoleTitle, emp.Department, emp.Status, emp.HiredAt,
	).Scan(
		&res.EmployeeID, &res.EmployeeType, &res.FirstName, &res.LastName, &res.Email,
		&res.RoleTitle, &res.Department, &res.Status, &res.HiredAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) UpdateEmployeeStatus(ctx context.Context, id, status string) (*Employee, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.employees
	          SET status = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE employee_id::text = $1
	          RETURNING employee_id::text, employee_type, first_name, last_name, email, role_title, department, status, hired_at, created_at, updated_at`
	var res Employee
	err := r.db.QueryRow(ctx, query, id, status).Scan(
		&res.EmployeeID, &res.EmployeeType, &res.FirstName, &res.LastName, &res.Email,
		&res.RoleTitle, &res.Department, &res.Status, &res.HiredAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("employee with id %s not found: %w", id, err)
	}
	return &res, nil
}

func (r *pgxRepository) CreateOnboardingTask(ctx context.Context, task *OnboardingTask) (*OnboardingTask, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.onboarding_checklists (
		checklist_id, employee_id, task_name, category, is_completed, completed_at, assigned_to
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2::uuid, $3, $4, $5, $6, NULLIF($7, '')::uuid
	) RETURNING checklist_id::text, employee_id::text, task_name, category, is_completed, completed_at, assigned_to::text, created_at, updated_at`
	var res OnboardingTask
	var assignedTo *string
	err := r.db.QueryRow(ctx, query,
		task.ChecklistID, task.EmployeeID, task.TaskName, task.Category, task.IsCompleted, task.CompletedAt, task.AssignedTo,
	).Scan(
		&res.ChecklistID, &res.EmployeeID, &res.TaskName, &res.Category, &res.IsCompleted, &res.CompletedAt, &assignedTo, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.AssignedTo = assignedTo
	return &res, nil
}

func (r *pgxRepository) ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*OnboardingTask, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT checklist_id::text, employee_id::text, task_name, category, is_completed, completed_at, assigned_to::text, created_at, updated_at
	          FROM workforce.onboarding_checklists
	          WHERE employee_id::text = $1
	          ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*OnboardingTask
	for rows.Next() {
		var res OnboardingTask
		var assignedTo *string
		if err := rows.Scan(&res.ChecklistID, &res.EmployeeID, &res.TaskName, &res.Category, &res.IsCompleted, &res.CompletedAt, &assignedTo, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		res.AssignedTo = assignedTo
		list = append(list, &res)
	}
	if list == nil {
		list = []*OnboardingTask{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*OnboardingTask, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.onboarding_checklists
	          SET is_completed = $2,
	              completed_at = CASE WHEN $2 = TRUE THEN CURRENT_TIMESTAMP ELSE NULL END,
	              updated_at = CURRENT_TIMESTAMP
	          WHERE checklist_id::text = $1
	          RETURNING checklist_id::text, employee_id::text, task_name, category, is_completed, completed_at, assigned_to::text, created_at, updated_at`
	var res OnboardingTask
	var assignedTo *string
	err := r.db.QueryRow(ctx, query, taskID, isCompleted).Scan(
		&res.ChecklistID, &res.EmployeeID, &res.TaskName, &res.Category, &res.IsCompleted, &res.CompletedAt, &assignedTo, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("task with id %s not found: %w", taskID, err)
	}
	res.AssignedTo = assignedTo
	return &res, nil
}

func (r *pgxRepository) GetCareScheduleByFelineID(ctx context.Context, felineID string) (*CareSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT schedule_id::text, feline_id::text, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id::text, created_at, updated_at
	          FROM workforce.care_schedules
	          WHERE feline_id::text = $1`
	var cs CareSchedule
	var caretakerID *string
	err := r.db.QueryRow(ctx, query, felineID).Scan(
		&cs.ScheduleID, &cs.FelineID, &cs.DietaryPlan, &cs.FeedingTimes, &cs.SpecialMedicalNeeds, &cs.PreferredPerchZone, &cs.EmergencyMedicalHold, &caretakerID, &cs.CreatedAt, &cs.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("care schedule for feline %s not found: %w", felineID, err)
	}
	cs.CaretakerID = caretakerID
	return &cs, nil
}

func (r *pgxRepository) UpsertCareSchedule(ctx context.Context, cs *CareSchedule) (*CareSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.care_schedules (
		schedule_id, feline_id, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2::uuid, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid
	) ON CONFLICT (feline_id) DO UPDATE SET
		dietary_plan = EXCLUDED.dietary_plan,
		feeding_times = EXCLUDED.feeding_times,
		special_medical_needs = EXCLUDED.special_medical_needs,
		preferred_perch_zone = EXCLUDED.preferred_perch_zone,
		emergency_medical_hold = EXCLUDED.emergency_medical_hold,
		caretaker_id = EXCLUDED.caretaker_id,
		updated_at = CURRENT_TIMESTAMP
	RETURNING schedule_id::text, feline_id::text, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id::text, created_at, updated_at`
	var res CareSchedule
	var caretakerID *string
	err := r.db.QueryRow(ctx, query,
		cs.ScheduleID, cs.FelineID, cs.DietaryPlan, cs.FeedingTimes, cs.SpecialMedicalNeeds, cs.PreferredPerchZone, cs.EmergencyMedicalHold, cs.CaretakerID,
	).Scan(
		&res.ScheduleID, &res.FelineID, &res.DietaryPlan, &res.FeedingTimes, &res.SpecialMedicalNeeds, &res.PreferredPerchZone, &res.EmergencyMedicalHold, &caretakerID, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.CaretakerID = caretakerID
	return &res, nil
}

func (r *pgxRepository) SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*CareSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.care_schedules (
		schedule_id, feline_id, dietary_plan, feeding_times, emergency_medical_hold
	) VALUES (
		gen_random_uuid_v7(), $1::uuid, 'Standard feline nutritional plan', ARRAY['09:00 AM', '05:00 PM'], $2
	) ON CONFLICT (feline_id) DO UPDATE SET
		emergency_medical_hold = EXCLUDED.emergency_medical_hold,
		updated_at = CURRENT_TIMESTAMP
	RETURNING schedule_id::text, feline_id::text, dietary_plan, feeding_times, special_medical_needs, preferred_perch_zone, emergency_medical_hold, caretaker_id::text, created_at, updated_at`
	var res CareSchedule
	var caretakerID *string
	err := r.db.QueryRow(ctx, query, felineID, hold).Scan(
		&res.ScheduleID, &res.FelineID, &res.DietaryPlan, &res.FeedingTimes, &res.SpecialMedicalNeeds, &res.PreferredPerchZone, &res.EmergencyMedicalHold, &caretakerID, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.CaretakerID = caretakerID
	return &res, nil
}

func (r *pgxRepository) CreateLeaveRequest(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.leave_requests (
		leave_id, employee_id, leave_type, start_date, end_date, status, reason
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4::date, $5::date, COALESCE(NULLIF($6, ''), 'pending'), $7
	) RETURNING leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at`
	var res LeaveRequest
	err := r.db.QueryRow(ctx, query, req.LeaveID, req.EmployeeID, req.LeaveType, req.StartDate, req.EndDate, req.Status, req.Reason).Scan(
		&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) GetLeaveRequestByID(ctx context.Context, id string) (*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at
	          FROM workforce.leave_requests WHERE leave_id::text = $1`
	var res LeaveRequest
	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leave request with id %s not found: %w", id, err)
	}
	return &res, nil
}

func (r *pgxRepository) ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at
	          FROM workforce.leave_requests
	          WHERE ($1 = '' OR employee_id::text = $1)
	            AND ($2 = '' OR status = $2)
	          ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, employeeID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*LeaveRequest
	for rows.Next() {
		var res LeaveRequest
		if err := rows.Scan(&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &res)
	}
	if list == nil {
		list = []*LeaveRequest{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*LeaveRequest, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.leave_requests
	          SET status = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE leave_id::text = $1
	          RETURNING leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at`
	var res LeaveRequest
	err := r.db.QueryRow(ctx, query, id, status).Scan(
		&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leave request with id %s not found: %w", id, err)
	}
	return &res, nil
}

func scanReviewCycle(row interface{ Scan(dest ...any) error }) (*ReviewCycle, error) {
	var res ReviewCycle
	var schedFor string
	var revID, peerRevID, clinAudID, vhrID *string
	err := row.Scan(
		&res.ReviewID, &res.EmployeeID, &res.ReviewType, &res.ReviewPeriod, &res.Phase, &schedFor, &res.Status,
		&revID, &res.Score, &res.Notes, &res.SelfAppraisal, &res.SelfScore, &res.PeerFeedback, &res.PeerScore,
		&peerRevID, &res.ManagerAssessment, &res.ManagerScore, &res.KPIRatings,
		&res.WellnessAuditScore, &res.HabitatRecommendations, &clinAudID, &vhrID,
		&res.CompletedAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.ScheduledFor = schedFor
	res.ReviewerID = revID
	res.PeerReviewerID = peerRevID
	res.ClinicalAuditorID = clinAudID
	res.VHRRecordID = vhrID
	return &res, nil
}

func (r *pgxRepository) CreateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.review_cycles (
	              review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
	              reviewer_id, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	              peer_reviewer_id, manager_assessment, manager_score, kpi_ratings,
	              wellness_audit_score, habitat_recommendations, clinical_auditor_id, vhr_record_id, completed_at
	          ) VALUES (
	              COALESCE($1, gen_random_uuid_v7()), $2, $3, COALESCE(NULLIF($4, ''), '2026-Q3'), COALESCE(NULLIF($5, ''), 'self_appraisal'),
	              $6, COALESCE(NULLIF($7, ''), 'scheduled'), $8, $9, $10, $11, $12, $13, $14,
	              $15, $16, $17, $18::jsonb, $19, $20, $21, $22, $23
	          ) RETURNING review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                      reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                      peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                      wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                      completed_at, created_at, updated_at`
	var revUUID *string
	if rc.ReviewID != "" {
		revUUID = &rc.ReviewID
	}
	row := r.db.QueryRow(ctx, query,
		revUUID, rc.EmployeeID, rc.ReviewType, rc.ReviewPeriod, rc.Phase,
		rc.ScheduledFor, rc.Status, rc.ReviewerID, rc.Score, rc.Notes,
		rc.SelfAppraisal, rc.SelfScore, rc.PeerFeedback, rc.PeerScore,
		rc.PeerReviewerID, rc.ManagerAssessment, rc.ManagerScore, rc.KPIRatings,
		rc.WellnessAuditScore, rc.HabitatRecommendations, rc.ClinicalAuditorID, rc.VHRRecordID,
		rc.CompletedAt,
	)
	return scanReviewCycle(row)
}

func (r *pgxRepository) GetReviewCycleByID(ctx context.Context, id string) (*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                 reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                 peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                 wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                 completed_at, created_at, updated_at
	          FROM workforce.review_cycles
	          WHERE review_id::text = $1`
	row := r.db.QueryRow(ctx, query, id)
	res, err := scanReviewCycle(row)
	if err != nil {
		return nil, fmt.Errorf("review cycle with id %s not found: %w", id, err)
	}
	return res, nil
}

func (r *pgxRepository) ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                 reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                 peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                 wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                 completed_at, created_at, updated_at
	          FROM workforce.review_cycles
	          WHERE ($1 = '' OR employee_id::text = $1)
	            AND ($2 = '' OR status = $2)
	            AND ($3 = '' OR review_type = $3)
	          ORDER BY scheduled_for DESC, created_at DESC`
	rows, err := r.db.Query(ctx, query, employeeID, status, reviewType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ReviewCycle
	for rows.Next() {
		res, err := scanReviewCycle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	if list == nil {
		list = []*ReviewCycle{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.review_cycles
	          SET status = COALESCE($2, status),
	              phase = CASE WHEN $3::text IS NOT NULL THEN $3::text ELSE phase END,
	              reviewer_id = CASE WHEN $4::text IS NOT NULL THEN $4::uuid ELSE reviewer_id END,
	              score = CASE WHEN $5::numeric IS NOT NULL THEN $5::numeric ELSE score END,
	              notes = CASE WHEN $6::text IS NOT NULL THEN $6::text ELSE notes END,
	              self_appraisal = CASE WHEN $7::text IS NOT NULL THEN $7::text ELSE self_appraisal END,
	              self_score = CASE WHEN $8::numeric IS NOT NULL THEN $8::numeric ELSE self_score END,
	              peer_feedback = CASE WHEN $9::text IS NOT NULL THEN $9::text ELSE peer_feedback END,
	              peer_score = CASE WHEN $10::numeric IS NOT NULL THEN $10::numeric ELSE peer_score END,
	              peer_reviewer_id = CASE WHEN $11::text IS NOT NULL THEN $11::uuid ELSE peer_reviewer_id END,
	              manager_assessment = CASE WHEN $12::text IS NOT NULL THEN $12::text ELSE manager_assessment END,
	              manager_score = CASE WHEN $13::numeric IS NOT NULL THEN $13::numeric ELSE manager_score END,
	              kpi_ratings = CASE WHEN $14::text IS NOT NULL THEN $14::jsonb ELSE kpi_ratings END,
	              wellness_audit_score = CASE WHEN $15::numeric IS NOT NULL THEN $15::numeric ELSE wellness_audit_score END,
	              habitat_recommendations = CASE WHEN $16::text IS NOT NULL THEN $16::text ELSE habitat_recommendations END,
	              clinical_auditor_id = CASE WHEN $17::text IS NOT NULL THEN $17::uuid ELSE clinical_auditor_id END,
	              vhr_record_id = CASE WHEN $18::text IS NOT NULL THEN $18::uuid ELSE vhr_record_id END,
	              completed_at = CASE WHEN $19::timestamptz IS NOT NULL THEN $19::timestamptz ELSE completed_at END,
	              updated_at = CURRENT_TIMESTAMP
	          WHERE review_id::text = $1
	          RETURNING review_id::text, employee_id::text, review_type, review_period, phase, scheduled_for::text, status,
	                    reviewer_id::text, score, notes, self_appraisal, self_score, peer_feedback, peer_score,
	                    peer_reviewer_id::text, manager_assessment, manager_score, kpi_ratings::text,
	                    wellness_audit_score, habitat_recommendations, clinical_auditor_id::text, vhr_record_id::text,
	                    completed_at, created_at, updated_at`
	row := r.db.QueryRow(ctx, query,
		rc.ReviewID, rc.Status, rc.Phase, rc.ReviewerID, rc.Score, rc.Notes,
		rc.SelfAppraisal, rc.SelfScore, rc.PeerFeedback, rc.PeerScore,
		rc.PeerReviewerID, rc.ManagerAssessment, rc.ManagerScore, rc.KPIRatings,
		rc.WellnessAuditScore, rc.HabitatRecommendations, rc.ClinicalAuditorID, rc.VHRRecordID,
		rc.CompletedAt,
	)
	res, err := scanReviewCycle(row)
	if err != nil {
		return nil, fmt.Errorf("review cycle with id %s not found: %w", rc.ReviewID, err)
	}
	return res, nil
}

func (r *pgxRepository) CreateVHRRecord(ctx context.Context, rec *VHRRecord) (*VHRRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.vhr_records (
		record_id, feline_id, veterinarian_id, visit_date, weight_kg, dental_score, vaccination_status, prescriptions, clinical_notes
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4::date, $5, $6, $7, $8, $9
	) RETURNING record_id::text, feline_id::text, veterinarian_id::text, visit_date::text, weight_kg, dental_score, vaccination_status, prescriptions, clinical_notes, created_at, updated_at`
	var res VHRRecord
	var vetID *string
	var vDate string
	err := r.db.QueryRow(ctx, query, rec.RecordID, rec.FelineID, rec.VeterinarianID, rec.VisitDate, rec.WeightKG, rec.DentalScore, rec.VaccinationStatus, rec.Prescriptions, rec.ClinicalNotes).Scan(
		&res.RecordID, &res.FelineID, &vetID, &vDate, &res.WeightKG, &res.DentalScore, &res.VaccinationStatus, &res.Prescriptions, &res.ClinicalNotes, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	res.VeterinarianID = vetID
	res.VisitDate = vDate
	return &res, nil
}

func (r *pgxRepository) ListVHRRecords(ctx context.Context, felineID string) ([]*VHRRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT record_id::text, feline_id::text, veterinarian_id::text, visit_date::text, weight_kg, dental_score, vaccination_status, prescriptions, clinical_notes, created_at, updated_at
	          FROM workforce.vhr_records
	          WHERE ($1 = '' OR feline_id::text = $1)
	          ORDER BY visit_date DESC`
	rows, err := r.db.Query(ctx, query, felineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*VHRRecord
	for rows.Next() {
		var res VHRRecord
		var vetID *string
		var vDate string
		if err := rows.Scan(&res.RecordID, &res.FelineID, &vetID, &vDate, &res.WeightKG, &res.DentalScore, &res.VaccinationStatus, &res.Prescriptions, &res.ClinicalNotes, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, err
		}
		res.VeterinarianID = vetID
		res.VisitDate = vDate
		list = append(list, &res)
	}
	if list == nil {
		list = []*VHRRecord{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) CreateWorkplaceIncident(ctx context.Context, inc *WorkplaceIncident) (*WorkplaceIncident, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.incidents (
		incident_id, title, category, involved_feline_id, involved_human_id, severity, status, description, resolution_notes
	) VALUES (
		COALESCE(NULLIF($1, '')::uuid, gen_random_uuid_v7()), $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'low'), COALESCE(NULLIF($7, ''), 'open'), $8, $9
	) RETURNING incident_id::text, title, category, involved_feline_id::text, involved_human_id::text, severity, status, description, resolution_notes, reported_at, resolved_at`
	var res WorkplaceIncident
	var fID, hID *string
	err := r.db.QueryRow(ctx, query, inc.IncidentID, inc.Title, inc.Category, inc.InvolvedFelineID, inc.InvolvedHumanID, inc.Severity, inc.Status, inc.Description, inc.ResolutionNotes).Scan(
		&res.IncidentID, &res.Title, &res.Category, &fID, &hID, &res.Severity, &res.Status, &res.Description, &res.ResolutionNotes, &res.ReportedAt, &res.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	res.InvolvedFelineID = fID
	res.InvolvedHumanID = hID
	return &res, nil
}

func (r *pgxRepository) ListWorkplaceIncidents(ctx context.Context, status string) ([]*WorkplaceIncident, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT incident_id::text, title, category, involved_feline_id::text, involved_human_id::text, severity, status, description, resolution_notes, reported_at, resolved_at
	          FROM workforce.incidents
	          WHERE ($1 = '' OR status = $1)
	          ORDER BY reported_at DESC`
	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*WorkplaceIncident
	for rows.Next() {
		var res WorkplaceIncident
		var fID, hID *string
		if err := rows.Scan(&res.IncidentID, &res.Title, &res.Category, &fID, &hID, &res.Severity, &res.Status, &res.Description, &res.ResolutionNotes, &res.ReportedAt, &res.ResolvedAt); err != nil {
			return nil, err
		}
		res.InvolvedFelineID = fID
		res.InvolvedHumanID = hID
		list = append(list, &res)
	}
	if list == nil {
		list = []*WorkplaceIncident{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*WorkplaceIncident, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	switch status {
	case "open", "under_review", "resolved":
	default:
		return nil, fmt.Errorf("invalid incident status %q: status must be one of 'open', 'under_review', 'resolved'", status)
	}

	query := `UPDATE workforce.incidents
	          SET status = $2,
	              resolution_notes = CASE WHEN $3 <> '' THEN $3 ELSE resolution_notes END,
	              resolved_at = CASE WHEN $2 = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END
	          WHERE incident_id::text = $1
	          RETURNING incident_id::text, title, category, involved_feline_id::text, involved_human_id::text, severity, status, description, resolution_notes, reported_at, resolved_at`
	var res WorkplaceIncident
	var fID, hID *string
	err := r.db.QueryRow(ctx, query, id, status, resolutionNotes).Scan(
		&res.IncidentID, &res.Title, &res.Category, &fID, &hID, &res.Severity, &res.Status, &res.Description, &res.ResolutionNotes, &res.ReportedAt, &res.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	res.InvolvedFelineID = fID
	res.InvolvedHumanID = hID
	return &res, nil
}

func (r *pgxRepository) CreateExecutiveReport(ctx context.Context, report *ExecutiveReport) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.executive_reports (
	              report_id, period, title, status,
	              total_revenue, operating_expenses, payroll_expenses, net_profit,
	              operational_burn, cash_reserves, runway_months,
	              total_headcount, human_headcount, feline_headcount,
	              attendance_rate, overtime_hours, timesheets_approved,
	              webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by
	          ) VALUES (
	              $1, $2, $3, COALESCE($4, 'archived'),
	              $5, $6, $7, $8,
	              $9, $10, $11,
	              $12, $13, $14,
	              $15, $16, $17,
	              $18, $19, $20, $21, COALESCE($22, 'system-reporting-worker')
	          ) RETURNING report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at`

	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query,
		report.ReportID, report.Period, report.Title, report.Status,
		report.TotalRevenue, report.OperatingExpenses, report.PayrollExpenses, report.NetProfit,
		report.OperationalBurn, report.CashReserves, report.RunwayMonths,
		report.TotalHeadcount, report.HumanHeadcount, report.FelineHeadcount,
		report.AttendanceRate, report.OvertimeHours, report.TimesheetsApproved,
		report.WebDAVPathPDF, report.WebDAVPathCSV, report.CSVContent, report.PDFContent, report.GeneratedBy,
	).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) GetExecutiveReportByID(ctx context.Context, id string) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at
	          FROM workforce.executive_reports WHERE report_id = $1`
	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) GetExecutiveReportByPeriod(ctx context.Context, period string) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at
	          FROM workforce.executive_reports WHERE period = $1 ORDER BY created_at DESC LIMIT 1`
	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query, period).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) ListExecutiveReports(ctx context.Context, limit, offset int) ([]*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at
	          FROM workforce.executive_reports
	          ORDER BY period DESC, created_at DESC
	          LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ExecutiveReport
	for rows.Next() {
		var res ExecutiveReport
		err := rows.Scan(
			&res.ReportID, &res.Period, &res.Title, &res.Status,
			&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
			&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
			&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
			&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
			&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
			&res.CreatedAt, &res.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &res)
	}
	if list == nil {
		list = []*ExecutiveReport{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateExecutiveReportStatus(ctx context.Context, id, status string, webdavPDF, webdavCSV *string) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.executive_reports
	          SET status = $2,
	              webdav_path_pdf = COALESCE($3, webdav_path_pdf),
	              webdav_path_csv = COALESCE($4, webdav_path_csv),
	              updated_at = CURRENT_TIMESTAMP
	          WHERE report_id = $1
	          RETURNING report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at`

	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query, id, status, webdavPDF, webdavCSV).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}


