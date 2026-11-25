package workforce

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/outbox"
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
	outboxStore      *outbox.MemoryStore
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
		outboxStore:      outbox.NewMemoryStore(outbox.SchemaWorkforce),
	}
	repo.seedDefaults()
	return repo
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
	lastName := ""
	if emp.LastName != nil {
		lastName = *emp.LastName
	}
	email := ""
	if emp.Email != nil {
		email = *emp.Email
	}
	r.enqueueOutbox(ctx, "events.workforce.employee_onboarded.v1", emp.EmployeeID, envelope.EmployeeOnboardedPayload{
		EmployeeID:   emp.EmployeeID,
		EmployeeType: emp.EmployeeType,
		FirstName:    emp.FirstName,
		LastName:     lastName,
		Email:        email,
		RoleTitle:    emp.RoleTitle,
		Department:   emp.Department,
	})
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
		r.enqueueOutbox(ctx, envelope.TopicCareScheduleUpdated, cs.FelineID, map[string]any{
			"feline_id":              cs.FelineID,
			"emergency_medical_hold": cs.EmergencyMedicalHold,
		})
		return cs, nil
	}

	cs.EmergencyMedicalHold = hold
	cs.UpdatedAt = time.Now().UTC()
	r.enqueueOutbox(ctx, envelope.TopicCareScheduleUpdated, cs.FelineID, map[string]any{
		"feline_id":              cs.FelineID,
		"emergency_medical_hold": cs.EmergencyMedicalHold,
	})
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

func (r *mockRepository) ClaimOutboxEvents(ctx context.Context, limit int32, lease time.Duration) ([]*outbox.Record, error) {
	if r.outboxStore == nil {
		return []*outbox.Record{}, nil
	}
	return r.outboxStore.ClaimPending(limit, lease), nil
}

func (r *mockRepository) AckOutboxEvents(ctx context.Context, publishedIDs []int64, failures []outbox.Failure) (int, error) {
	if r.outboxStore == nil {
		return 0, nil
	}
	r.outboxStore.MarkPublished(publishedIDs)
	for _, f := range failures {
		r.outboxStore.MarkFailed(f.ID, f.Error, 0)
	}
	return len(publishedIDs) + len(failures), nil
}

func (r *mockRepository) enqueueOutbox(ctx context.Context, topic, aggregateID string, payload any) {
	if r.outboxStore == nil {
		return
	}
	if evID, raw, err := outbox.BuildEnvelopeBytes(ctx, "", topic, aggregateID, payload); err == nil {
		r.outboxStore.Enqueue(evID, topic, aggregateID, raw)
	}
}
