package workforce

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type mockRepository struct {
	mu               sync.RWMutex
	employees        map[string]*Employee
	checklists       map[string]*OnboardingTask
	careSchedules    map[string]*CareSchedule
	leaveRequests    map[string]*LeaveRequest
	reviewCycles     map[string]*ReviewCycle
	vhrRecords       map[string]*VHRRecord
	incidents        map[string]*WorkplaceIncident
	executiveReports map[string]*ExecutiveReport
}

// NewMockRepository returns an in-memory mock implementation of workforce.Repository pre-seeded with test personas.
func NewMockRepository() Repository {
	repo := &mockRepository{
		employees:        make(map[string]*Employee),
		checklists:       make(map[string]*OnboardingTask),
		careSchedules:    make(map[string]*CareSchedule),
		leaveRequests:    make(map[string]*LeaveRequest),
		reviewCycles:     make(map[string]*ReviewCycle),
		vhrRecords:       make(map[string]*VHRRecord),
		incidents:        make(map[string]*WorkplaceIncident),
		executiveReports: make(map[string]*ExecutiveReport),
	}
	repo.seedDefaults()
	return repo
}

func (r *mockRepository) seedDefaults() {
	now := time.Now().UTC()
	r.seedEmployees(now)
	r.seedCareSchedules(now)
	r.seedChecklists(now)
	r.seedLeaveRequests(now)
	r.seedReviewCycles(now)
	r.seedExecutiveReports(now)
}

func (r *mockRepository) seedEmployees(now time.Time) {
	r.seedFelineEmployees(now)
	r.seedHumanEmployees(now)
}

func (r *mockRepository) seedFelineEmployees(now time.Time) {
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
}

func (r *mockRepository) seedHumanEmployees(now time.Time) {
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
}

func (r *mockRepository) seedCareSchedules(now time.Time) {
	garfieldID := "emp-feline-garfield"
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
	r.careSchedules[barnebyID] = &CareSchedule{
		ScheduleID:           "cs-barneby-01",
		FelineID:             barnebyID,
		DietaryPlan:          "Grain-free organic turkey pate",
		FeedingTimes:         []string{"07:30 AM", "05:30 PM"},
		EmergencyMedicalHold: false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func (r *mockRepository) seedChecklists(now time.Time) {
	aliceID := "emp-human-alice"
	rickID := "emp-human-rick"

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
}

func (r *mockRepository) seedLeaveRequests(now time.Time) {
	garfieldID := "emp-feline-garfield"
	barnebyID := "emp-feline-barneby"
	elenaID := "emp-human-elena"

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
}

func floatPtr(f float64) *float64 {
	return &f
}

func (r *mockRepository) GetEmployeeByID(ctx context.Context, id string) (*Employee, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	emp, ok := r.employees[id]
	if !ok {
		return nil, fmt.Errorf("employee with id %s not found", id)
	}
	return emp, nil
}

func (r *mockRepository) ListEmployees(ctx context.Context, employeeType, status string) ([]*Employee, error) {
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

func (r *mockRepository) CreateEmployee(ctx context.Context, emp *Employee) (*Employee, error) {
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

func (r *mockRepository) UpdateEmployeeStatus(ctx context.Context, id, status string) (*Employee, error) {
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

func (r *mockRepository) CreateOnboardingTask(ctx context.Context, task *OnboardingTask) (*OnboardingTask, error) {
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

func (r *mockRepository) ListOnboardingChecklist(ctx context.Context, employeeID string) ([]*OnboardingTask, error) {
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

func (r *mockRepository) UpdateOnboardingTask(ctx context.Context, taskID string, isCompleted bool) (*OnboardingTask, error) {
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

func (r *mockRepository) GetCareScheduleByFelineID(ctx context.Context, felineID string) (*CareSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cs, ok := r.careSchedules[felineID]
	if !ok {
		return nil, fmt.Errorf("care schedule for feline %s not found", felineID)
	}
	return cs, nil
}

func (r *mockRepository) UpsertCareSchedule(ctx context.Context, cs *CareSchedule) (*CareSchedule, error) {
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

func (r *mockRepository) SetEmergencyMedicalHold(ctx context.Context, felineID string, hold bool) (*CareSchedule, error) {
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

func (r *mockRepository) CreateLeaveRequest(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error) {
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

func (r *mockRepository) GetLeaveRequestByID(ctx context.Context, id string) (*LeaveRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, ok := r.leaveRequests[id]
	if !ok {
		return nil, fmt.Errorf("leave request with id %s not found", id)
	}
	return req, nil
}

func (r *mockRepository) ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*LeaveRequest, error) {
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

func (r *mockRepository) UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*LeaveRequest, error) {
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
