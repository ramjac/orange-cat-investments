package workforce

import (
	"context"
	"fmt"
	"sort"
	"sync"
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
	ReviewID     string     `json:"review_id"`
	EmployeeID   string     `json:"employee_id"`
	ReviewType   string     `json:"review_type"`
	ScheduledFor string     `json:"scheduled_for"`
	Status       string     `json:"status"`
	ReviewerID   *string    `json:"reviewer_id,omitempty"`
	Score        *float64   `json:"score,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
}

type pgxRepository struct {
	db            *pgxpool.Pool
	mu            sync.RWMutex
	employees     map[string]*Employee
	checklists    map[string]*OnboardingTask
	careSchedules map[string]*CareSchedule
	leaveRequests map[string]*LeaveRequest
	reviewCycles  map[string]*ReviewCycle
	vhrRecords    map[string]*VHRRecord
	incidents     map[string]*WorkplaceIncident
}

func NewRepository(db *pgxpool.Pool) Repository {
	repo := &pgxRepository{
		db:            db,
		employees:     make(map[string]*Employee),
		checklists:    make(map[string]*OnboardingTask),
		careSchedules: make(map[string]*CareSchedule),
		leaveRequests: make(map[string]*LeaveRequest),
		reviewCycles:  make(map[string]*ReviewCycle),
		vhrRecords:    make(map[string]*VHRRecord),
		incidents:     make(map[string]*WorkplaceIncident),
	}

	// Seed sample personas into in-memory store for dev/testing when DB pool is uninitialized
	repo.seedDefaults()
	return repo
}

func (r *pgxRepository) seedDefaults() {
	now := time.Now().UTC()
	garfieldID := "emp-feline-garfield"
	r.employees[garfieldID] = &Employee{
		EmployeeID:   garfieldID,
		EmployeeType: "feline",
		FirstName:    "Garfield",
		RoleTitle:    "Chief Observation Officer",
		Department:   "Executive Feline Suite",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	r.careSchedules[garfieldID] = &CareSchedule{
		ScheduleID:           "cs-garfield-01",
		FelineID:             garfieldID,
		DietaryPlan:          "High-protein salmon pate & prescription kibble",
		FeedingTimes:         []string{"08:00 AM", "12:00 PM", "06:00 PM"},
		EmergencyMedicalHold: false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	barnebyID := "emp-feline-barneby"
	r.employees[barnebyID] = &Employee{
		EmployeeID:   barnebyID,
		EmployeeType: "feline",
		FirstName:    "Barneby",
		RoleTitle:    "Senior Alpha Perch Analyst",
		Department:   "Alpha Perch Research",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	r.careSchedules[barnebyID] = &CareSchedule{
		ScheduleID:           "cs-barneby-01",
		FelineID:             barnebyID,
		DietaryPlan:          "Grain-free organic turkey pate",
		FeedingTimes:         []string{"07:30 AM", "05:30 PM"},
		EmergencyMedicalHold: false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	aliceID := "emp-human-alice"
	r.employees[aliceID] = &Employee{
		EmployeeID:   aliceID,
		EmployeeType: "human",
		FirstName:    "Alice",
		LastName:     strPtr("Vance"),
		Email:        strPtr("alice.vance@oci.local"),
		RoleTitle:    "Head of Human & Feline Resources",
		Department:   "Workforce Operations",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	elenaID := "emp-human-elena"
	r.employees[elenaID] = &Employee{
		EmployeeID:   elenaID,
		EmployeeType: "human",
		FirstName:    "Elena",
		LastName:     strPtr("Rostova"),
		Email:        strPtr("elena.rostova@oci.local"),
		RoleTitle:    "Chief Veterinary Officer",
		Department:   "Feline Health & Welfare",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	rickID := "emp-human-rick"
	r.employees[rickID] = &Employee{
		EmployeeID:   rickID,
		EmployeeType: "human",
		FirstName:    "Rick",
		LastName:     strPtr("Newhire"),
		Email:        strPtr("rick.newhire@oci.local"),
		RoleTitle:    "Junior Operations Associate",
		Department:   "Workforce Operations",
		Status:       "onboarding",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	r.checklists["chk-rick-01"] = &OnboardingTask{
		ChecklistID: "chk-rick-01",
		EmployeeID:  rickID,
		TaskName:    "Forgejo Account Provisioning & SPIFFE ID Issuance",
		Category:    "access",
		IsCompleted: false,
		AssignedTo:  &aliceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r.checklists["chk-rick-02"] = &OnboardingTask{
		ChecklistID: "chk-rick-02",
		EmployeeID:  rickID,
		TaskName:    "Catnip Safety Orientation & Workstation Setup",
		Category:    "training",
		IsCompleted: false,
		AssignedTo:  &aliceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	eliseID := "emp-human-elise"
	r.employees[eliseID] = &Employee{
		EmployeeID:   eliseID,
		EmployeeType: "human",
		FirstName:    "Elise",
		LastName:     strPtr("Dev"),
		Email:        strPtr("elise.dev@oci.local"),
		RoleTitle:    "Software Developer",
		Department:   "Engineering & Platform Development",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	frankID := "emp-human-frank"
	r.employees[frankID] = &Employee{
		EmployeeID:   frankID,
		EmployeeType: "human",
		FirstName:    "Frank",
		LastName:     strPtr("Operations"),
		Email:        strPtr("frank.ops@oci.local"),
		RoleTitle:    "Platform Security & K8s Infrastructure Lead",
		Department:   "Platform Security & Infrastructure",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	bobID := "emp-human-bob"
	r.employees[bobID] = &Employee{
		EmployeeID:   bobID,
		EmployeeType: "human",
		FirstName:    "Bob",
		LastName:     strPtr("Builder"),
		Email:        strPtr("bob.builder@oci.local"),
		RoleTitle:    "Lead Facilities & Edge Telemetry Engineer",
		Department:   "Habitat Facilities & Infrastructure",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Seed foundational leave requests & feline catnip breaks
	garfieldLeaveID := "018e0000-0000-7000-8000-000000000010"
	r.leaveRequests[garfieldLeaveID] = &LeaveRequest{
		LeaveID:    garfieldLeaveID,
		EmployeeID: garfieldID,
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-01",
		EndDate:    "2026-10-03",
		Status:     "approved",
		Reason:     strPtr("Mandatory post-alpha observation rest & organic catnip relaxation"),
		CreatedAt:  now.Add(-24 * time.Hour),
		UpdatedAt:  now.Add(-24 * time.Hour),
	}

	barnebyLeaveID := "018e0000-0000-7000-8000-000000000011"
	r.leaveRequests[barnebyLeaveID] = &LeaveRequest{
		LeaveID:    barnebyLeaveID,
		EmployeeID: barnebyID,
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-05",
		EndDate:    "2026-10-06",
		Status:     "pending",
		Reason:     strPtr("Alpha perch rotation decompression and premium catnip session"),
		CreatedAt:  now.Add(-2 * time.Hour),
		UpdatedAt:  now.Add(-2 * time.Hour),
	}

	elenaLeaveID := "018e0000-0000-7000-8000-000000000012"
	r.leaveRequests[elenaLeaveID] = &LeaveRequest{
		LeaveID:    elenaLeaveID,
		EmployeeID: elenaID,
		LeaveType:  "vacation",
		StartDate:  "2026-10-15",
		EndDate:    "2026-10-20",
		Status:     "approved",
		Reason:     strPtr("Annual veterinary feline habitat conference"),
		CreatedAt:  now.Add(-48 * time.Hour),
		UpdatedAt:  now.Add(-48 * time.Hour),
	}

	// Seed review cycles (Performance reviews for humans, Health/Care assessments for felines)
	garfieldReviewID := "018e0000-0000-7000-8000-000000000020"
	completedAt := now.Add(-72 * time.Hour)
	garfieldScore := 4.95
	garfieldReviewNotes := "Superb whisker symmetry, resting purr acoustics 92dB, optimal alpha sunbeam positioning. Lasagna tolerance remains peak."
	r.reviewCycles[garfieldReviewID] = &ReviewCycle{
		ReviewID:     garfieldReviewID,
		EmployeeID:   garfieldID,
		ReviewType:   "feline_health_assessment",
		ScheduledFor: now.Add(-72 * time.Hour).Format("2006-01-02"),
		Status:       "completed",
		ReviewerID:   &elenaID,
		Score:        &garfieldScore,
		Notes:        &garfieldReviewNotes,
		CompletedAt:  &completedAt,
		CreatedAt:    now.Add(-7 * 24 * time.Hour),
		UpdatedAt:    completedAt,
	}

	barnebyReviewID := "018e0000-0000-7000-8000-000000000021"
	barnebyReviewNotes := "Quarterly weight check and alpha perch mobility audit"
	r.reviewCycles[barnebyReviewID] = &ReviewCycle{
		ReviewID:     barnebyReviewID,
		EmployeeID:   barnebyID,
		ReviewType:   "feline_health_assessment",
		ScheduledFor: now.Add(48 * time.Hour).Format("2006-01-02"),
		Status:       "scheduled",
		ReviewerID:   &elenaID,
		Notes:        &barnebyReviewNotes,
		CreatedAt:    now.Add(-24 * time.Hour),
		UpdatedAt:    now.Add(-24 * time.Hour),
	}

	aliceReviewID := "018e0000-0000-7000-8000-000000000022"
	aliceReviewNotes := "Joint human-feline HR operations & catnip compliance review"
	r.reviewCycles[aliceReviewID] = &ReviewCycle{
		ReviewID:     aliceReviewID,
		EmployeeID:   aliceID,
		ReviewType:   "performance",
		ScheduledFor: now.Add(7 * 24 * time.Hour).Format("2006-01-02"),
		Status:       "scheduled",
		Notes:        &aliceReviewNotes,
		CreatedAt:    now.Add(-48 * time.Hour),
		UpdatedAt:    now.Add(-48 * time.Hour),
	}
}

func strPtr(s string) *string {
	return &s
}

func (r *pgxRepository) GetEmployeeByID(ctx context.Context, id string) (*Employee, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	emp, ok := r.employees[id]
	if !ok {
		return nil, fmt.Errorf("employee with id %s not found", id)
	}
	return emp, nil
}

func (r *pgxRepository) ListEmployees(ctx context.Context, employeeType, status string) ([]*Employee, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*Employee
	for _, emp := range r.employees {
		if employeeType != "" && emp.EmployeeType != employeeType {
			continue
		}
		if status != "" && emp.Status != status {
			continue
		}
		list = append(list, emp)
	}
	return list, nil
}

func (r *pgxRepository) CreateEmployee(ctx context.Context, emp *Employee) (*Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if emp.EmployeeID == "" {
		emp.EmployeeID = fmt.Sprintf("emp-gen-%d", time.Now().UnixNano())
	}
	if emp.Status == "" {
		emp.Status = "onboarding"
	}
	emp.CreatedAt = now
	emp.UpdatedAt = now
	if emp.HiredAt.IsZero() {
		emp.HiredAt = now
	}

	r.employees[emp.EmployeeID] = emp
	return emp, nil
}

func (r *pgxRepository) UpdateEmployeeStatus(ctx context.Context, id, status string) (*Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	emp, ok := r.employees[id]
	if !ok {
		return nil, fmt.Errorf("employee with id %s not found", id)
	}

	emp.Status = status
	emp.UpdatedAt = time.Now().UTC()
	return emp, nil
}

func (r *pgxRepository) CreateOnboardingTask(ctx context.Context, task *OnboardingTask) (*OnboardingTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if task.ChecklistID == "" {
		task.ChecklistID = fmt.Sprintf("chk-%d", time.Now().UnixNano())
	}
	task.CreatedAt = now
	task.UpdatedAt = now

	r.checklists[task.ChecklistID] = task
	return task, nil
}

func (r *pgxRepository) ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*OnboardingTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*OnboardingTask
	for _, t := range r.checklists {
		if t.EmployeeID == employeeID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (r *pgxRepository) UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*OnboardingTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.checklists[taskID]
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	now := time.Now().UTC()
	task.IsCompleted = isCompleted
	if isCompleted {
		task.CompletedAt = &now
	} else {
		task.CompletedAt = nil
	}
	task.UpdatedAt = now
	return task, nil
}

func (r *pgxRepository) GetCareScheduleByFelineID(ctx context.Context, felineID string) (*CareSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cs, ok := r.careSchedules[felineID]
	if !ok {
		return nil, fmt.Errorf("care schedule for feline %s not found", felineID)
	}
	return cs, nil
}

func (r *pgxRepository) UpsertCareSchedule(ctx context.Context, cs *CareSchedule) (*CareSchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if existing, ok := r.careSchedules[cs.FelineID]; ok {
		cs.ScheduleID = existing.ScheduleID
		cs.CreatedAt = existing.CreatedAt
	} else {
		if cs.ScheduleID == "" {
			cs.ScheduleID = fmt.Sprintf("cs-%d", time.Now().UnixNano())
		}
		cs.CreatedAt = now
	}
	cs.UpdatedAt = now

	r.careSchedules[cs.FelineID] = cs
	return cs, nil
}

func (r *pgxRepository) SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*CareSchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cs, ok := r.careSchedules[felineID]
	if !ok {
		now := time.Now().UTC()
		cs = &CareSchedule{
			ScheduleID:           fmt.Sprintf("cs-%d", time.Now().UnixNano()),
			FelineID:             felineID,
			DietaryPlan:          "Standard feline nutritional plan",
			FeedingTimes:         []string{"09:00 AM", "05:00 PM"},
			EmergencyMedicalHold: hold,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		r.careSchedules[felineID] = cs
		return cs, nil
	}

	cs.EmergencyMedicalHold = hold
	cs.UpdatedAt = time.Now().UTC()
	return cs, nil
}

func (r *pgxRepository) CreateLeaveRequest(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error) {
	if r.db != nil {
		query := `INSERT INTO workforce.leave_requests (
			leave_id, employee_id, leave_type, start_date, end_date, status, reason
		) VALUES (
			COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2, $3, $4::date, $5::date, COALESCE(NULLIF($6, ''), 'pending'), $7
		) RETURNING leave_id::text, employee_id::text, leave_type, start_date::text, end_date::text, status, reason, created_at, updated_at`
		var res LeaveRequest
		err := r.db.QueryRow(ctx, query, req.LeaveID, req.EmployeeID, req.LeaveType, req.StartDate, req.EndDate, req.Status, req.Reason).Scan(
			&res.LeaveID, &res.EmployeeID, &res.LeaveType, &res.StartDate, &res.EndDate, &res.Status, &res.Reason, &res.CreatedAt, &res.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		r.mu.Lock()
		r.leaveRequests[res.LeaveID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if req.LeaveID == "" {
		req.LeaveID = fmt.Sprintf("leave-%d", time.Now().UnixNano())
	}
	if req.Status == "" {
		req.Status = "pending"
	}
	req.CreatedAt = now
	req.UpdatedAt = now

	r.leaveRequests[req.LeaveID] = req
	return req, nil
}

func (r *pgxRepository) GetLeaveRequestByID(ctx context.Context, id string) (*LeaveRequest, error) {
	if r.db != nil {
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

	r.mu.RLock()
	defer r.mu.RUnlock()

	req, ok := r.leaveRequests[id]
	if !ok {
		return nil, fmt.Errorf("leave request with id %s not found", id)
	}
	return req, nil
}

func (r *pgxRepository) ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*LeaveRequest, error) {
	if r.db != nil {
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
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if list == nil {
			list = []*LeaveRequest{}
		}
		return list, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*LeaveRequest
	for _, req := range r.leaveRequests {
		if employeeID != "" && req.EmployeeID != employeeID {
			continue
		}
		if status != "" && req.Status != status {
			continue
		}
		list = append(list, req)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	if list == nil {
		list = []*LeaveRequest{}
	}
	return list, nil
}

func (r *pgxRepository) UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*LeaveRequest, error) {
	if r.db != nil {
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
		r.mu.Lock()
		r.leaveRequests[res.LeaveID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	req, ok := r.leaveRequests[id]
	if !ok {
		return nil, fmt.Errorf("leave request with id %s not found", id)
	}

	req.Status = status
	req.UpdatedAt = time.Now().UTC()
	return req, nil
}

func (r *pgxRepository) CreateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	if r.db != nil {
		query := `INSERT INTO workforce.review_cycles (
		              review_id, employee_id, review_type, scheduled_for, status, reviewer_id, score, notes, completed_at
		          ) VALUES (
		              COALESCE($1, gen_random_uuid_v7()), $2, $3, $4, COALESCE($5, 'scheduled'), $6, $7, $8, $9
		          ) RETURNING review_id::text, employee_id::text, review_type, scheduled_for::text, status, reviewer_id::text, score, notes, completed_at, created_at, updated_at`
		var res ReviewCycle
		var revID *string
		var schedFor string
		var revUUID *string
		if rc.ReviewID != "" {
			revUUID = &rc.ReviewID
		}
		err := r.db.QueryRow(ctx, query, revUUID, rc.EmployeeID, rc.ReviewType, rc.ScheduledFor, rc.Status, rc.ReviewerID, rc.Score, rc.Notes, rc.CompletedAt).Scan(
			&res.ReviewID, &res.EmployeeID, &res.ReviewType, &schedFor, &res.Status, &revID, &res.Score, &res.Notes, &res.CompletedAt, &res.CreatedAt, &res.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		res.ScheduledFor = schedFor
		res.ReviewerID = revID
		r.mu.Lock()
		r.reviewCycles[res.ReviewID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if rc.ReviewID == "" {
		rc.ReviewID = fmt.Sprintf("018e0000-0000-7000-8000-%012d", len(r.reviewCycles)+1)
	}
	if rc.Status == "" {
		rc.Status = "scheduled"
	}
	rc.CreatedAt = now
	rc.UpdatedAt = now

	r.reviewCycles[rc.ReviewID] = rc
	return rc, nil
}

func (r *pgxRepository) GetReviewCycleByID(ctx context.Context, id string) (*ReviewCycle, error) {
	if r.db != nil {
		query := `SELECT review_id::text, employee_id::text, review_type, scheduled_for::text, status, reviewer_id::text, score, notes, completed_at, created_at, updated_at
		          FROM workforce.review_cycles
		          WHERE review_id::text = $1`
		var res ReviewCycle
		var revID *string
		var schedFor string
		err := r.db.QueryRow(ctx, query, id).Scan(
			&res.ReviewID, &res.EmployeeID, &res.ReviewType, &schedFor, &res.Status, &revID, &res.Score, &res.Notes, &res.CompletedAt, &res.CreatedAt, &res.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("review cycle with id %s not found: %w", id, err)
		}
		res.ScheduledFor = schedFor
		res.ReviewerID = revID
		return &res, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	rc, ok := r.reviewCycles[id]
	if !ok {
		return nil, fmt.Errorf("review cycle with id %s not found", id)
	}
	return rc, nil
}

func (r *pgxRepository) ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*ReviewCycle, error) {
	if r.db != nil {
		query := `SELECT review_id::text, employee_id::text, review_type, scheduled_for::text, status, reviewer_id::text, score, notes, completed_at, created_at, updated_at
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
			var res ReviewCycle
			var revID *string
			var schedFor string
			if err := rows.Scan(&res.ReviewID, &res.EmployeeID, &res.ReviewType, &schedFor, &res.Status, &revID, &res.Score, &res.Notes, &res.CompletedAt, &res.CreatedAt, &res.UpdatedAt); err != nil {
				return nil, err
			}
			res.ScheduledFor = schedFor
			res.ReviewerID = revID
			list = append(list, &res)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if list == nil {
			list = []*ReviewCycle{}
		}
		return list, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*ReviewCycle
	for _, rc := range r.reviewCycles {
		if employeeID != "" && rc.EmployeeID != employeeID {
			continue
		}
		if status != "" && rc.Status != status {
			continue
		}
		if reviewType != "" && rc.ReviewType != reviewType {
			continue
		}
		list = append(list, rc)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].ScheduledFor == list[j].ScheduledFor {
			return list[i].CreatedAt.After(list[j].CreatedAt)
		}
		return list[i].ScheduledFor > list[j].ScheduledFor
	})
	if list == nil {
		list = []*ReviewCycle{}
	}
	return list, nil
}

func (r *pgxRepository) UpdateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	if r.db != nil {
		query := `UPDATE workforce.review_cycles
		          SET status = COALESCE($2, status),
		              reviewer_id = CASE WHEN $3::text IS NOT NULL THEN $3::uuid ELSE reviewer_id END,
		              score = CASE WHEN $4::numeric IS NOT NULL THEN $4::numeric ELSE score END,
		              notes = CASE WHEN $5::text IS NOT NULL THEN $5::text ELSE notes END,
		              completed_at = CASE WHEN $6::timestamptz IS NOT NULL THEN $6::timestamptz ELSE completed_at END,
		              updated_at = CURRENT_TIMESTAMP
		          WHERE review_id::text = $1
		          RETURNING review_id::text, employee_id::text, review_type, scheduled_for::text, status, reviewer_id::text, score, notes, completed_at, created_at, updated_at`
		var res ReviewCycle
		var revID *string
		var schedFor string
		err := r.db.QueryRow(ctx, query, rc.ReviewID, rc.Status, rc.ReviewerID, rc.Score, rc.Notes, rc.CompletedAt).Scan(
			&res.ReviewID, &res.EmployeeID, &res.ReviewType, &schedFor, &res.Status, &revID, &res.Score, &res.Notes, &res.CompletedAt, &res.CreatedAt, &res.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("review cycle with id %s not found: %w", rc.ReviewID, err)
		}
		res.ScheduledFor = schedFor
		res.ReviewerID = revID
		r.mu.Lock()
		r.reviewCycles[res.ReviewID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.reviewCycles[rc.ReviewID]
	if !ok {
		return nil, fmt.Errorf("review cycle with id %s not found", rc.ReviewID)
	}

	if rc.Status != "" {
		existing.Status = rc.Status
	}
	if rc.ReviewerID != nil {
		existing.ReviewerID = rc.ReviewerID
	}
	if rc.Score != nil {
		existing.Score = rc.Score
	}
	if rc.Notes != nil {
		existing.Notes = rc.Notes
	}
	if rc.CompletedAt != nil {
		existing.CompletedAt = rc.CompletedAt
	}
	existing.UpdatedAt = time.Now().UTC()
	return existing, nil
}

func (r *pgxRepository) CreateVHRRecord(ctx context.Context, rec *VHRRecord) (*VHRRecord, error) {
	if r.db != nil {
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
		r.mu.Lock()
		r.vhrRecords[res.RecordID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if rec.RecordID == "" {
		rec.RecordID = fmt.Sprintf("vhr-%d", time.Now().UnixNano())
	}
	rec.CreatedAt = now
	rec.UpdatedAt = now
	r.vhrRecords[rec.RecordID] = rec
	return rec, nil
}

func (r *pgxRepository) ListVHRRecords(ctx context.Context, felineID string) ([]*VHRRecord, error) {
	if r.db != nil {
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
		return list, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*VHRRecord
	for _, rec := range r.vhrRecords {
		if felineID != "" && rec.FelineID != felineID {
			continue
		}
		list = append(list, rec)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].VisitDate > list[j].VisitDate
	})
	if list == nil {
		list = []*VHRRecord{}
	}
	return list, nil
}

func (r *pgxRepository) CreateWorkplaceIncident(ctx context.Context, inc *WorkplaceIncident) (*WorkplaceIncident, error) {
	if r.db != nil {
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
		r.mu.Lock()
		r.incidents[res.IncidentID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if inc.IncidentID == "" {
		inc.IncidentID = fmt.Sprintf("inc-%d", time.Now().UnixNano())
	}
	if inc.Status == "" {
		inc.Status = "open"
	}
	if inc.Severity == "" {
		inc.Severity = "low"
	}
	inc.ReportedAt = now
	r.incidents[inc.IncidentID] = inc
	return inc, nil
}

func (r *pgxRepository) ListWorkplaceIncidents(ctx context.Context, status string) ([]*WorkplaceIncident, error) {
	if r.db != nil {
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
		return list, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*WorkplaceIncident
	for _, inc := range r.incidents {
		if status != "" && inc.Status != status {
			continue
		}
		list = append(list, inc)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ReportedAt.After(list[j].ReportedAt)
	})
	if list == nil {
		list = []*WorkplaceIncident{}
	}
	return list, nil
}

func (r *pgxRepository) UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*WorkplaceIncident, error) {
	now := time.Now().UTC()
	if r.db != nil {
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
		r.mu.Lock()
		r.incidents[res.IncidentID] = &res
		r.mu.Unlock()
		return &res, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	inc, ok := r.incidents[id]
	if !ok {
		return nil, fmt.Errorf("incident with id %s not found", id)
	}

	switch status {
	case "open", "under_review", "resolved":
	default:
		return nil, fmt.Errorf("invalid incident status %q: status must be one of 'open', 'under_review', 'resolved'", status)
	}

	inc.Status = status
	if resolutionNotes != "" {
		inc.ResolutionNotes = &resolutionNotes
	}
	if status == "resolved" {
		inc.ResolvedAt = &now
	}
	return inc, nil
}
