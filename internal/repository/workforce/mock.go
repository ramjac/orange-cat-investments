package workforce

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type mockRepository struct {
	mu            sync.RWMutex
	employees     map[string]*Employee
	checklists    map[string]*OnboardingTask
	careSchedules map[string]*CareSchedule
	leaveRequests map[string]*LeaveRequest
	reviewCycles  map[string]*ReviewCycle
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

func (r *mockRepository) seedReviewCycles(now time.Time) {
	garfieldID := "emp-feline-garfield"
	barnebyID := "emp-feline-barneby"
	aliceID := "emp-human-alice"
	elenaID := "emp-human-elena"

	garfieldReviewID := "018e0000-0000-7000-8000-000000000020"
	completedAt := now.Add(-72 * time.Hour)
	garfieldScore := 4.95
	garfieldReviewNotes := "Superb whisker symmetry, resting purr acoustics 92dB, optimal alpha sunbeam positioning. Lasagna tolerance remains peak."
	r.reviewCycles[garfieldReviewID] = &ReviewCycle{
		ReviewID:                garfieldReviewID,
		EmployeeID:              garfieldID,
		ReviewType:              "feline_health_assessment",
		ReviewPeriod:            "2026-Q3",
		Phase:                   "completed",
		ScheduledFor:            now.Add(-72 * time.Hour).Format("2006-01-02"),
		Status:                  "completed",
		ReviewerID:              &elenaID,
		Score:                   &garfieldScore,
		Notes:                   &garfieldReviewNotes,
		SelfAppraisal:           strPtr("Maintained executive posture on alpha perch; nap duration exceeded target by 12%."),
		SelfScore:               floatPtr(5.00),
		ManagerAssessment:       strPtr("Executive feline leadership has stabilized market confidence across retail portfolios."),
		ManagerScore:            floatPtr(4.95),
		KPIRatings:              strPtr(`{"purr_acoustics": 4.95, "sunbeam_coverage": 5.0, "observation_fidelity": 4.9}`),
		WellnessAuditScore:      floatPtr(4.95),
		HabitatRecommendations: strPtr("Increase sunlight perch access in Habitat Alpha; maintain lasagna moderation protocol."),
		ClinicalAuditorID:       &elenaID,
		CompletedAt:             &completedAt,
		CreatedAt:               now.Add(-7 * 24 * time.Hour),
		UpdatedAt:               completedAt,
	}

	barnebyReviewID := "018e0000-0000-7000-8000-000000000021"
	barnebyReviewNotes := "Quarterly weight check and alpha perch mobility audit"
	r.reviewCycles[barnebyReviewID] = &ReviewCycle{
		ReviewID:                barnebyReviewID,
		EmployeeID:              barnebyID,
		ReviewType:              "feline_health_assessment",
		ReviewPeriod:            "2026-Q3",
		Phase:                   "manager_review",
		ScheduledFor:            now.Add(48 * time.Hour).Format("2006-01-02"),
		Status:                  "in_progress",
		ReviewerID:              &elenaID,
		Notes:                   &barnebyReviewNotes,
		SelfAppraisal:           strPtr("Climbed perch 42 times this week; verified perimeter integrity."),
		SelfScore:               floatPtr(4.80),
		PeerFeedback:            strPtr("Great team perch rotation coordination."),
		PeerScore:               floatPtr(4.75),
		PeerReviewerID:          &garfieldID,
		KPIRatings:              strPtr(`{"perch_agility": 4.85, "catnip_temperance": 4.75}`),
		ClinicalAuditorID:       &elenaID,
		CreatedAt:               now.Add(-24 * time.Hour),
		UpdatedAt:               now.Add(-24 * time.Hour),
	}

	aliceReviewID := "018e0000-0000-7000-8000-000000000022"
	aliceReviewNotes := "Joint human-feline HR operations & catnip compliance review"
	r.reviewCycles[aliceReviewID] = &ReviewCycle{
		ReviewID:          aliceReviewID,
		EmployeeID:        aliceID,
		ReviewType:        "performance",
		ReviewPeriod:      "2026-Q3",
		Phase:             "self_appraisal",
		ScheduledFor:      now.Add(7 * 24 * time.Hour).Format("2006-01-02"),
		Status:            "scheduled",
		Notes:             &aliceReviewNotes,
		SelfAppraisal:     strPtr("Automated developer provisioning across Forgejo, Nextcloud, and Snipe-IT."),
		SelfScore:         floatPtr(4.85),
		KPIRatings:        strPtr(`{"onboarding_efficiency": 4.9, "compliance_adherence": 4.8}`),
		CreatedAt:         now.Add(-48 * time.Hour),
		UpdatedAt:         now.Add(-48 * time.Hour),
	}
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

func (r *mockRepository) CreateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if rc.ReviewID == "" {
		rc.ReviewID = fmt.Sprintf("018e0000-0000-7000-8000-%012d", len(r.reviewCycles)+1)
	}
	if rc.ReviewPeriod == "" {
		rc.ReviewPeriod = "2026-Q3"
	}
	if rc.Phase == "" {
		rc.Phase = "self_appraisal"
	}
	if rc.Status == "" {
		rc.Status = "scheduled"
	}
	rc.CreatedAt = now
	rc.UpdatedAt = now

	r.reviewCycles[rc.ReviewID] = rc
	return rc, nil
}

func (r *mockRepository) GetReviewCycleByID(ctx context.Context, id string) (*ReviewCycle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rc, ok := r.reviewCycles[id]
	if !ok {
		return nil, fmt.Errorf("review cycle with id %s not found", id)
	}
	return rc, nil
}

func (r *mockRepository) ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*ReviewCycle, error) {
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

func (r *mockRepository) UpdateReviewCycle(ctx context.Context, rc *ReviewCycle) (*ReviewCycle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.reviewCycles[rc.ReviewID]
	if !ok {
		return nil, fmt.Errorf("review cycle with id %s not found", rc.ReviewID)
	}

	if rc.ReviewPeriod != "" {
		existing.ReviewPeriod = rc.ReviewPeriod
	}
	if rc.Phase != "" {
		existing.Phase = rc.Phase
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
	if rc.SelfAppraisal != nil {
		existing.SelfAppraisal = rc.SelfAppraisal
	}
	if rc.SelfScore != nil {
		existing.SelfScore = rc.SelfScore
	}
	if rc.PeerFeedback != nil {
		existing.PeerFeedback = rc.PeerFeedback
	}
	if rc.PeerScore != nil {
		existing.PeerScore = rc.PeerScore
	}
	if rc.PeerReviewerID != nil {
		existing.PeerReviewerID = rc.PeerReviewerID
	}
	if rc.ManagerAssessment != nil {
		existing.ManagerAssessment = rc.ManagerAssessment
	}
	if rc.ManagerScore != nil {
		existing.ManagerScore = rc.ManagerScore
	}
	if rc.KPIRatings != nil {
		existing.KPIRatings = rc.KPIRatings
	}
	if rc.WellnessAuditScore != nil {
		existing.WellnessAuditScore = rc.WellnessAuditScore
	}
	if rc.HabitatRecommendations != nil {
		existing.HabitatRecommendations = rc.HabitatRecommendations
	}
	if rc.ClinicalAuditorID != nil {
		existing.ClinicalAuditorID = rc.ClinicalAuditorID
	}
	if rc.VHRRecordID != nil {
		existing.VHRRecordID = rc.VHRRecordID
	}
	if rc.CompletedAt != nil {
		existing.CompletedAt = rc.CompletedAt
	}
	existing.UpdatedAt = time.Now().UTC()
	return existing, nil
}

func (r *mockRepository) CreateVHRRecord(ctx context.Context, rec *VHRRecord) (*VHRRecord, error) {
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

func (r *mockRepository) ListVHRRecords(ctx context.Context, felineID string) ([]*VHRRecord, error) {
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

func (r *mockRepository) CreateWorkplaceIncident(ctx context.Context, inc *WorkplaceIncident) (*WorkplaceIncident, error) {
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

func (r *mockRepository) ListWorkplaceIncidents(ctx context.Context, status string) ([]*WorkplaceIncident, error) {
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

func (r *mockRepository) UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*WorkplaceIncident, error) {
	now := time.Now().UTC()
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

func (r *mockRepository) seedExecutiveReports(now time.Time) {
	pdfPath8 := "/remote.php/dav/files/admin/Executive/Reports/2026-08/executive_summary_report_2026-08.pdf"
	csvPath8 := "/remote.php/dav/files/admin/Executive/Reports/2026-08/executive_summary_report_2026-08.csv"
	r.executiveReports["rep-2026-08"] = &ExecutiveReport{
		ReportID:           "rep-2026-08",
		Period:             "2026-08",
		Title:              "OCI Executive Management Report - August 2026",
		Status:             "archived",
		TotalRevenue:       1180000.00,
		OperatingExpenses:  310000.00,
		PayrollExpenses:    395000.00,
		NetProfit:          475000.00,
		OperationalBurn:    705000.00,
		CashReserves:       9200000.00,
		RunwayMonths:       13.05,
		TotalHeadcount:     11,
		HumanHeadcount:     9,
		FelineHeadcount:    2,
		AttendanceRate:     98.80,
		OvertimeHours:      38.00,
		TimesheetsApproved: 16,
		WebDAVPathPDF:      &pdfPath8,
		WebDAVPathCSV:      &csvPath8,
		GeneratedBy:        "system-reporting-worker",
		CreatedAt:          now.Add(-60 * 24 * time.Hour),
		UpdatedAt:          now.Add(-60 * 24 * time.Hour),
	}

	pdfPath9 := "/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_report_2026-09.pdf"
	csvPath9 := "/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_report_2026-09.csv"
	r.executiveReports["rep-2026-09"] = &ExecutiveReport{
		ReportID:           "rep-2026-09",
		Period:             "2026-09",
		Title:              "OCI Executive Management Report - September 2026",
		Status:             "archived",
		TotalRevenue:       1240000.00,
		OperatingExpenses:  335000.00,
		PayrollExpenses:    410000.00,
		NetProfit:          495000.00,
		OperationalBurn:    745000.00,
		CashReserves:       9650000.00,
		RunwayMonths:       12.95,
		TotalHeadcount:     12,
		HumanHeadcount:     10,
		FelineHeadcount:    2,
		AttendanceRate:     99.10,
		OvertimeHours:      41.50,
		TimesheetsApproved: 18,
		WebDAVPathPDF:      &pdfPath9,
		WebDAVPathCSV:      &csvPath9,
		GeneratedBy:        "system-reporting-worker",
		CreatedAt:          now.Add(-30 * 24 * time.Hour),
		UpdatedAt:          now.Add(-30 * 24 * time.Hour),
	}
}

func (r *mockRepository) CreateExecutiveReport(ctx context.Context, report *ExecutiveReport) (*ExecutiveReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if report.ReportID == "" {
		report.ReportID = fmt.Sprintf("rep-%s", report.Period)
	}
	now := time.Now().UTC()
	if report.CreatedAt.IsZero() {
		report.CreatedAt = now
	}
	if report.UpdatedAt.IsZero() {
		report.UpdatedAt = now
	}
	if report.Status == "" {
		report.Status = "archived"
	}
	if report.GeneratedBy == "" {
		report.GeneratedBy = "system-reporting-worker"
	}

	r.executiveReports[report.ReportID] = report
	return report, nil
}

func (r *mockRepository) GetExecutiveReportByID(ctx context.Context, id string) (*ExecutiveReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rep, ok := r.executiveReports[id]
	if !ok {
		return nil, fmt.Errorf("executive report with id %s not found", id)
	}
	return rep, nil
}

func (r *mockRepository) GetExecutiveReportByPeriod(ctx context.Context, period string) (*ExecutiveReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched *ExecutiveReport
	for _, rep := range r.executiveReports {
		if rep.Period == period {
			if matched == nil || rep.CreatedAt.After(matched.CreatedAt) {
				matched = rep
			}
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("executive report for period %s not found", period)
	}
	return matched, nil
}

func (r *mockRepository) ListExecutiveReports(ctx context.Context, limit, offset int) ([]*ExecutiveReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*ExecutiveReport
	for _, rep := range r.executiveReports {
		list = append(list, rep)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Period != list[j].Period {
			return list[i].Period > list[j].Period
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if offset >= len(list) {
		return []*ExecutiveReport{}, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (r *mockRepository) UpdateExecutiveReportStatus(ctx context.Context, id, status string, webdavPDF, webdavCSV *string) (*ExecutiveReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rep, ok := r.executiveReports[id]
	if !ok {
		return nil, fmt.Errorf("executive report with id %s not found", id)
	}
	rep.Status = status
	if webdavPDF != nil {
		rep.WebDAVPathPDF = webdavPDF
	}
	if webdavCSV != nil {
		rep.WebDAVPathCSV = webdavCSV
	}
	rep.UpdatedAt = time.Now().UTC()
	return rep, nil
}



