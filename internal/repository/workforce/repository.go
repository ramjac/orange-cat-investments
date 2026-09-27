package workforce

import (
	"context"
	"fmt"
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
}

type pgxRepository struct {
	db             *pgxpool.Pool
	mu             sync.RWMutex
	employees      map[string]*Employee
	checklists     map[string]*OnboardingTask
	careSchedules  map[string]*CareSchedule
}

func NewRepository(db *pgxpool.Pool) Repository {
	repo := &pgxRepository{
		db:            db,
		employees:     make(map[string]*Employee),
		checklists:    make(map[string]*OnboardingTask),
		careSchedules: make(map[string]*CareSchedule),
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
