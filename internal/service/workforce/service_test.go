package workforce

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/stretchr/testify/assert"
)

func setupWorkforceService(t *testing.T) (Service, context.Context) {
	t.Helper()
	repo := workforce.NewRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()
	return svc, ctx
}

func TestGetEmployee(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	emp, err := svc.GetEmployee(ctx, "emp-feline-garfield")
	assert.NoError(t, err)
	assert.Equal(t, "Garfield", emp.FirstName)
}

func TestListEmployees(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	// List all employees without filtering
	all, err := svc.ListEmployees(ctx, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, all)

	// Filter by employeeType "feline"
	felines, err := svc.ListEmployees(ctx, "feline", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, felines)
	for _, emp := range felines {
		assert.Equal(t, "feline", emp.EmployeeType)
	}

	// Filter by employeeType "human"
	humans, err := svc.ListEmployees(ctx, "human", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, humans)
	for _, emp := range humans {
		assert.Equal(t, "human", emp.EmployeeType)
	}

	// Filter by status "active"
	actives, err := svc.ListEmployees(ctx, "", "active")
	assert.NoError(t, err)
	assert.NotEmpty(t, actives)
	for _, emp := range actives {
		assert.Equal(t, "active", emp.Status)
	}

	// Filter by status "onboarding"
	onboardings, err := svc.ListEmployees(ctx, "", "onboarding")
	assert.NoError(t, err)
	assert.NotEmpty(t, onboardings)
	for _, emp := range onboardings {
		assert.Equal(t, "onboarding", emp.Status)
	}

	// Filter by both employeeType "feline" and status "active"
	activeFelines, err := svc.ListEmployees(ctx, "feline", "active")
	assert.NoError(t, err)
	assert.NotEmpty(t, activeFelines)
	for _, emp := range activeFelines {
		assert.Equal(t, "feline", emp.EmployeeType)
		assert.Equal(t, "active", emp.Status)
	}

	// Filter with non-matching filter
	none, err := svc.ListEmployees(ctx, "nonexistent_type", "terminated")
	assert.NoError(t, err)
	assert.Empty(t, none)
}

func TestUpdateEmployeeStatus(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	// Validation: Missing employee_id
	_, err := svc.UpdateEmployeeStatus(ctx, "", "active")
	assert.Error(t, err)
	assert.Equal(t, "employee id and status are required", err.Error())

	// Validation: Missing status
	_, err = svc.UpdateEmployeeStatus(ctx, "emp-feline-garfield", "")
	assert.Error(t, err)
	assert.Equal(t, "employee id and status are required", err.Error())

	// Validation: Both empty
	_, err = svc.UpdateEmployeeStatus(ctx, "", "")
	assert.Error(t, err)
	assert.Equal(t, "employee id and status are required", err.Error())

	// Repository Error: Non-existent employee ID
	_, err = svc.UpdateEmployeeStatus(ctx, "non-existent-employee-id", "active")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "employee with id non-existent-employee-id not found")

	// Success: Update employee status
	updated, err := svc.UpdateEmployeeStatus(ctx, "emp-feline-garfield", "on_leave")
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, "on_leave", updated.Status)

	// Verify that GetEmployee returns the updated status
	fetched, err := svc.GetEmployee(ctx, "emp-feline-garfield")
	assert.NoError(t, err)
	assert.Equal(t, "on_leave", fetched.Status)
}

func TestOnboardEmployee(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	feline := &workforce.Employee{
		FirstName:    "Sylvester",
		EmployeeType: "feline",
		RoleTitle:    "Junior Perch Inspector",
		Department:   "Habitat Ops",
	}
	cs := &workforce.CareSchedule{
		DietaryPlan:  "Tuna delight",
		FeedingTimes: []string{"09:00 AM", "05:00 PM"},
	}

	created, err := svc.OnboardEmployee(ctx, feline, cs)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.EmployeeID)
	assert.Equal(t, "onboarding", created.Status)

	// Verify onboarding tasks were generated
	tasks, err := svc.ListOnboardingChecklist(ctx, created.EmployeeID)
	assert.NoError(t, err)
	assert.Len(t, tasks, 4)

	// Verify care schedule was stored
	storedCS, err := svc.GetCareSchedule(ctx, created.EmployeeID)
	assert.NoError(t, err)
	assert.Equal(t, "Tuna delight", storedCS.DietaryPlan)
}

func TestCareSchedule(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	// Happy path: Update care schedule with valid FelineID
	notes := "Daily vitamins"
	cs := &workforce.CareSchedule{
		FelineID:             "emp-feline-garfield",
		DietaryPlan:          "Salmon & Catnip Supreme",
		FeedingTimes:         []string{"08:00 AM", "12:00 PM", "06:00 PM"},
		SpecialMedicalNeeds:  &notes,
		EmergencyMedicalHold: false,
	}

	updated, err := svc.UpdateCareSchedule(ctx, cs)
	assert.NoError(t, err)
	assert.Equal(t, "emp-feline-garfield", updated.FelineID)
	assert.Equal(t, "Salmon & Catnip Supreme", updated.DietaryPlan)

	// Get updated Care Schedule
	fetched, err := svc.GetCareSchedule(ctx, "emp-feline-garfield")
	assert.NoError(t, err)
	assert.Equal(t, "Salmon & Catnip Supreme", fetched.DietaryPlan)

	// Boundary/Validation: nil CareSchedule
	_, err = svc.UpdateCareSchedule(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid care schedule and feline id required")

	// Boundary/Validation: CareSchedule with empty FelineID
	_, err = svc.UpdateCareSchedule(ctx, &workforce.CareSchedule{DietaryPlan: "Tuna"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid care schedule and feline id required")

	// Boundary/Validation: GetCareSchedule with empty felineID
	_, err = svc.GetCareSchedule(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feline id cannot be empty")
}

func TestSetEmergencyMedicalHold(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	cs, err := svc.SetEmergencyMedicalHold(ctx, "emp-feline-garfield", true)
	assert.NoError(t, err)
	assert.True(t, cs.EmergencyMedicalHold)

	cs2, err := svc.SetEmergencyMedicalHold(ctx, "emp-feline-garfield", false)
	assert.NoError(t, err)
	assert.False(t, cs2.EmergencyMedicalHold)
}

func TestLeaveRequests(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	reason := "Premium catnip break & laser chasing recharge"
	req, err := svc.CreateLeaveRequest(ctx, CreateLeaveRequestParams{
		EmployeeID: "emp-feline-garfield",
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-10",
		EndDate:    "2026-10-12",
		Reason:     &reason,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, req.LeaveID)
	assert.Equal(t, "catnip_break", req.LeaveType)
	assert.Equal(t, "pending", req.Status)

	// Get Leave Request
	fetched, err := svc.GetLeaveRequest(ctx, req.LeaveID)
	assert.NoError(t, err)
	assert.Equal(t, req.LeaveID, fetched.LeaveID)

	// List Leave Requests
	list, err := svc.ListLeaveRequests(ctx, "emp-feline-garfield", "pending")
	assert.NoError(t, err)
	assert.NotEmpty(t, list)

	// Approve Leave Request
	approved, err := svc.UpdateLeaveRequestStatus(ctx, req.LeaveID, "approved")
	assert.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)

	// Validation: Missing employee_id
	_, err = svc.CreateLeaveRequest(ctx, CreateLeaveRequestParams{
		EmployeeID: "",
		LeaveType:  "vacation",
		StartDate:  "2026-10-10",
		EndDate:    "2026-10-12",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "employee_id is required")

	// Validation: Invalid leave_type
	_, err = svc.CreateLeaveRequest(ctx, CreateLeaveRequestParams{
		EmployeeID: "emp-feline-garfield",
		LeaveType:  "unsupported_type",
		StartDate:  "2026-10-10",
		EndDate:    "2026-10-12",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid leave_type")

	// Validation: Invalid date format
	_, err = svc.CreateLeaveRequest(ctx, CreateLeaveRequestParams{
		EmployeeID: "emp-feline-garfield",
		LeaveType:  "catnip_break",
		StartDate:  "10-10-2026",
		EndDate:    "2026-10-12",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")

	// Validation: End date before start date
	_, err = svc.CreateLeaveRequest(ctx, CreateLeaveRequestParams{
		EmployeeID: "emp-feline-garfield",
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-15",
		EndDate:    "2026-10-12",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end_date must be on or after start_date")

	// Validation: Invalid status update
	_, err = svc.UpdateLeaveRequestStatus(ctx, req.LeaveID, "invalid_status")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestReviewCycles(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	notes := "Annual feline agility, sunbeam tracking, and alpha response readiness"
	reviewer := "emp-human-elena"
	rc, err := svc.CreateReviewCycle(ctx, CreateReviewCycleParams{
		EmployeeID:   "emp-feline-garfield",
		ReviewType:   "feline_health_assessment",
		ScheduledFor: "2026-11-20",
		ReviewerID:   &reviewer,
		Notes:        &notes,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, rc.ReviewID)
	assert.Equal(t, "feline_health_assessment", rc.ReviewType)
	assert.Equal(t, "scheduled", rc.Status)
	assert.Equal(t, "2026-11-20", rc.ScheduledFor)

	// Get Review Cycle
	fetched, err := svc.GetReviewCycle(ctx, rc.ReviewID)
	assert.NoError(t, err)
	assert.Equal(t, rc.ReviewID, fetched.ReviewID)

	// List Review Cycles
	list, err := svc.ListReviewCycles(ctx, "emp-feline-garfield", "scheduled", "feline_health_assessment")
	assert.NoError(t, err)
	assert.NotEmpty(t, list)

	// Update Review Cycle (Evaluation)
	score := 4.92
	newStatus := "completed"
	evalNotes := "Phenomenal alpha perch surveillance metrics and optimal vitals"
	updated, err := svc.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:     rc.ReviewID,
		Status: &newStatus,
		Score:  &score,
		Notes:  &evalNotes,
	})
	assert.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, score, *updated.Score)
	assert.Equal(t, evalNotes, *updated.Notes)
	assert.NotNil(t, updated.CompletedAt)

	// Validation: Missing employee_id
	_, err = svc.CreateReviewCycle(ctx, CreateReviewCycleParams{
		EmployeeID:   "",
		ReviewType:   "performance",
		ScheduledFor: "2026-11-20",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "employee_id is required")

	// Validation: Invalid review_type
	_, err = svc.CreateReviewCycle(ctx, CreateReviewCycleParams{
		EmployeeID:   "emp-feline-garfield",
		ReviewType:   "invalid_type",
		ScheduledFor: "2026-11-20",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid review_type")

	// Validation: Invalid scheduled_for date format
	_, err = svc.CreateReviewCycle(ctx, CreateReviewCycleParams{
		EmployeeID:   "emp-feline-garfield",
		ReviewType:   "performance",
		ScheduledFor: "11-20-2026",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")

	// Validation: Score out of range (> 5.0)
	badScore := 6.5
	_, err = svc.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:    rc.ReviewID,
		Score: &badScore,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "score must be between 0.00 and 5.00")

	// Validation: Negative score
	negScore := -1.0
	_, err = svc.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:    rc.ReviewID,
		Score: &negScore,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "score must be between 0.00 and 5.00")

	// Validation: Invalid status update
	badStatus := "flying_high"
	_, err = svc.UpdateReviewCycle(ctx, UpdateReviewCycleParams{
		ID:     rc.ReviewID,
		Status: &badStatus,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestWorkplaceIncidents(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	// Validations for CreateWorkplaceIncident
	_, err := svc.CreateWorkplaceIncident(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title and description are required")

	_, err = svc.CreateWorkplaceIncident(ctx, &workforce.WorkplaceIncident{Description: "Missing title"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title and description are required")

	_, err = svc.CreateWorkplaceIncident(ctx, &workforce.WorkplaceIncident{Title: "Missing description"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title and description are required")

	// Create incident with empty category to test default fallback
	incDefaultCategory := &workforce.WorkplaceIncident{
		Title:       "Spilled Water Bowl",
		Description: "Hydro disruption near server rack",
	}
	createdDefault, err := svc.CreateWorkplaceIncident(ctx, incDefaultCategory)
	assert.NoError(t, err)
	assert.Equal(t, "habitat_disruption", createdDefault.Category)

	inc1 := &workforce.WorkplaceIncident{
		Title:       "Perch Dispute",
		Category:    "perch_dispute",
		Severity:    "low",
		Status:      "open",
		Description: "Two felines contested same sunbeam perch",
	}
	created1, err := svc.CreateWorkplaceIncident(ctx, inc1)
	assert.NoError(t, err)

	// Valid update to under_review
	updated1, err := svc.UpdateWorkplaceIncidentStatus(ctx, created1.IncidentID, "under_review", "Manager reviewing")
	assert.NoError(t, err)
	assert.Equal(t, "under_review", updated1.Status)

	inc2 := &workforce.WorkplaceIncident{
		Title:       "Catnip Storage Breach",
		Category:    "security_breach",
		Severity:    "high",
		Status:      "resolved",
		Description: "Unauthorized feline accessed secondary vault",
	}
	_, err = svc.CreateWorkplaceIncident(ctx, inc2)
	assert.NoError(t, err)

	// Test ListWorkplaceIncidents - all
	allIncidents, err := svc.ListWorkplaceIncidents(ctx, "")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(allIncidents), 3)

	// Test ListWorkplaceIncidents - status filter "under_review"
	underReviewIncidents, err := svc.ListWorkplaceIncidents(ctx, "under_review")
	assert.NoError(t, err)
	assert.NotEmpty(t, underReviewIncidents)
	for _, item := range underReviewIncidents {
		assert.Equal(t, "under_review", item.Status)
	}

	// Test ListWorkplaceIncidents - status filter "resolved"
	resolvedIncidents, err := svc.ListWorkplaceIncidents(ctx, "resolved")
	assert.NoError(t, err)
	assert.NotEmpty(t, resolvedIncidents)
	for _, item := range resolvedIncidents {
		assert.Equal(t, "resolved", item.Status)
	}

	// Invalid update should be rejected
	_, err = svc.UpdateWorkplaceIncidentStatus(ctx, created1.IncidentID, "invalid_custom_status", "note")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid incident status")
}

func TestVHRRecords(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	feline1 := "emp-feline-garfield"
	feline2 := "emp-feline-barneby"

	// Create VHR Record for feline1 with explicit VisitDate
	rec1 := &workforce.VHRRecord{
		FelineID:          feline1,
		VisitDate:         "2026-05-10",
		WeightKG:          5.8,
		DentalScore:       4,
		VaccinationStatus: "up_to_date",
		ClinicalNotes:     "Healthy feline, whisker symmetry optimal",
	}
	created1, err := svc.CreateVHRRecord(ctx, rec1)
	assert.NoError(t, err)
	assert.NotEmpty(t, created1.RecordID)
	assert.Equal(t, feline1, created1.FelineID)
	assert.Equal(t, "2026-05-10", created1.VisitDate)
	assert.Equal(t, 4, created1.DentalScore)

	// Create VHR Record for feline1 with empty VisitDate (should default to today)
	rec2 := &workforce.VHRRecord{
		FelineID:          feline1,
		WeightKG:          5.9,
		DentalScore:       5,
		VaccinationStatus: "up_to_date",
		ClinicalNotes:     "Follow-up checkup",
	}
	created2, err := svc.CreateVHRRecord(ctx, rec2)
	assert.NoError(t, err)
	assert.NotEmpty(t, created2.VisitDate)

	// Create VHR Record for feline2
	rec3 := &workforce.VHRRecord{
		FelineID:          feline2,
		VisitDate:         "2026-06-01",
		WeightKG:          4.5,
		DentalScore:       3,
		VaccinationStatus: "booster_due",
		ClinicalNotes:     "Barneby routine dental check",
	}
	created3, err := svc.CreateVHRRecord(ctx, rec3)
	assert.NoError(t, err)
	assert.Equal(t, feline2, created3.FelineID)

	// List VHR Records for feline1
	records1, err := svc.ListVHRRecords(ctx, feline1)
	assert.NoError(t, err)
	assert.Len(t, records1, 2)
	for _, r := range records1 {
		assert.Equal(t, feline1, r.FelineID)
	}

	// List VHR Records with empty felineID (returns all records)
	allRecords, err := svc.ListVHRRecords(ctx, "")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(allRecords), 3)

	// List VHR Records for non-existent felineID
	emptyRecords, err := svc.ListVHRRecords(ctx, "non-existent-feline")
	assert.NoError(t, err)
	assert.Empty(t, emptyRecords)

	// Validation: nil record
	_, err = svc.CreateVHRRecord(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feline_id is required for VHR record")

	// Validation: empty feline_id
	_, err = svc.CreateVHRRecord(ctx, &workforce.VHRRecord{DentalScore: 3})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feline_id is required for VHR record")

	// Validation: dental score < 1
	_, err = svc.CreateVHRRecord(ctx, &workforce.VHRRecord{FelineID: feline1, DentalScore: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dental score must be between 1 and 5")

	// Validation: dental score > 5
	_, err = svc.CreateVHRRecord(ctx, &workforce.VHRRecord{FelineID: feline1, DentalScore: 6})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dental score must be between 1 and 5")
}

func TestUpdateOnboardingTaskAndChecklist(t *testing.T) {
	svc, ctx := setupWorkforceService(t)

	// List checklist for seeded employee Rick
	rickTasks, err := svc.ListOnboardingChecklist(ctx, "emp-human-rick")
	assert.NoError(t, err)
	assert.NotEmpty(t, rickTasks)

	taskToUpdate := rickTasks[0]
	assert.False(t, taskToUpdate.IsCompleted)
	assert.Nil(t, taskToUpdate.CompletedAt)

	// Happy path: Update task to completed
	updatedTask, err := svc.UpdateOnboardingTask(ctx, taskToUpdate.ChecklistID, true)
	assert.NoError(t, err)
	assert.True(t, updatedTask.IsCompleted)
	assert.NotNil(t, updatedTask.CompletedAt)

	// Toggle path: Update task back to uncompleted
	toggledTask, err := svc.UpdateOnboardingTask(ctx, taskToUpdate.ChecklistID, false)
	assert.NoError(t, err)
	assert.False(t, toggledTask.IsCompleted)
	assert.Nil(t, toggledTask.CompletedAt)

	// Validation error: empty taskID for UpdateOnboardingTask
	_, err = svc.UpdateOnboardingTask(ctx, "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task id cannot be empty")

	// Repository error: non-existent taskID for UpdateOnboardingTask
	_, err = svc.UpdateOnboardingTask(ctx, "non-existent-task-id", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Validation error: empty employeeID for ListOnboardingChecklist
	_, err = svc.ListOnboardingChecklist(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "employee id cannot be empty")
}

type testEventPublisher struct {
	published []any
}

func (p *testEventPublisher) Publish(topic string, payload any) error {
	p.published = append(p.published, payload)
	return nil
}

type testHRClient struct {
	created  []*frappe.Timesheet
	approved []string
	canceled []string
	attended []*frappe.Attendance
}

func (c *testHRClient) CreateTimesheet(ctx context.Context, ts *frappe.Timesheet) (*frappe.Timesheet, error) {
	ts.Name = fmt.Sprintf("TS-FRAPPE-%03d", len(c.created)+1)
	ts.Normalize()
	c.created = append(c.created, ts)
	return ts, nil
}

func (c *testHRClient) GetTimesheet(ctx context.Context, name string) (*frappe.Timesheet, error) {
	for _, ts := range c.created {
		if ts.Name == name || ts.TimesheetID == name {
			return ts, nil
		}
	}
	return nil, fmt.Errorf("timesheet not found")
}

func (c *testHRClient) ListTimesheets(ctx context.Context, employeeID, status string) ([]frappe.Timesheet, error) {
	var res []frappe.Timesheet
	for _, ts := range c.created {
		if employeeID != "" && ts.Employee != employeeID && ts.EmployeeID != employeeID {
			continue
		}
		if status != "" && !strings.EqualFold(ts.Status, status) {
			continue
		}
		res = append(res, *ts)
	}
	return res, nil
}

func (c *testHRClient) ApproveTimesheet(ctx context.Context, name string) (*frappe.Timesheet, error) {
	c.approved = append(c.approved, name)
	ts, err := c.GetTimesheet(ctx, name)
	if err != nil {
		return &frappe.Timesheet{Name: name, Status: "Approved"}, nil
	}
	ts.Status = "Approved"
	return ts, nil
}

func (c *testHRClient) CancelTimesheet(ctx context.Context, name string) (*frappe.Timesheet, error) {
	c.canceled = append(c.canceled, name)
	ts, err := c.GetTimesheet(ctx, name)
	if err != nil {
		return &frappe.Timesheet{Name: name, Status: "Cancelled"}, nil
	}
	ts.Status = "Cancelled"
	return ts, nil
}

func (c *testHRClient) MarkAttendance(ctx context.Context, att *frappe.Attendance) (*frappe.Attendance, error) {
	c.attended = append(c.attended, att)
	return att, nil
}

func (c *testHRClient) GetAttendance(ctx context.Context, name string) (*frappe.Attendance, error) {
	return nil, nil
}

func (c *testHRClient) ListAttendance(ctx context.Context, employeeID string) ([]frappe.Attendance, error) {
	return nil, nil
}

func (c *testHRClient) GetSalarySlips(ctx context.Context, employeeID string) ([]frappe.SalarySlip, error) {
	return nil, nil
}

func (c *testHRClient) GetSalarySlip(ctx context.Context, name string) (*frappe.SalarySlip, error) {
	return nil, nil
}

func (c *testHRClient) CreateSalarySlip(ctx context.Context, slip *frappe.SalarySlip) (*frappe.SalarySlip, error) {
	return slip, nil
}

func (c *testHRClient) CreatePayrollEntry(ctx context.Context, pe *frappe.PayrollEntry) (*frappe.PayrollEntry, error) {
	return pe, nil
}

func TestTimesheets_SubmitAndGet(t *testing.T) {
	repo := workforce.NewRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	startTime := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 10, 15, 16, 30, 0, 0, time.UTC)

	// Successful submit
	ts := &frappe.Timesheet{
		EmployeeID:   "emp-test-01",
		ShiftDate:    "2026-10-15",
		StartTime:    &startTime,
		EndTime:      &endTime,
		BreakMinutes: 30,
		TaskCode:     "habitat_maintenance",
		Billable:     true,
	}

	created, err := svc.SubmitTimesheet(ctx, ts)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.TimesheetID)
	assert.Equal(t, 8.0, created.HoursWorked)
	assert.Equal(t, "submitted", created.Status)
	assert.Equal(t, "Facilities Operations", created.CostCenter)

	// Fetch by ID
	fetched, err := svc.GetTimesheet(ctx, created.TimesheetID)
	assert.NoError(t, err)
	assert.Equal(t, created.TimesheetID, fetched.TimesheetID)

	// Overlapping shift rejection
	overlapStart := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	overlapEnd := time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)
	overlapTS := &frappe.Timesheet{
		EmployeeID:   "emp-test-01",
		ShiftDate:    "2026-10-15",
		StartTime:    &overlapStart,
		EndTime:      &overlapEnd,
		BreakMinutes: 0,
		TaskCode:     "admin",
	}
	_, err = svc.SubmitTimesheet(ctx, overlapTS)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overlaps with existing timesheet")
}

func TestTimesheets_ValidationErrors(t *testing.T) {
	repo := workforce.NewRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	// Nil timesheet
	_, err := svc.SubmitTimesheet(ctx, nil)
	assert.Error(t, err)

	// Empty employee
	_, err = svc.SubmitTimesheet(ctx, &frappe.Timesheet{})
	assert.Error(t, err)

	// Start >= End
	startTime := time.Date(2026, 10, 15, 16, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	_, err = svc.SubmitTimesheet(ctx, &frappe.Timesheet{
		EmployeeID: "emp-test-01",
		StartTime:  &startTime,
		EndTime:    &endTime,
	})
	assert.Error(t, err)

	// Shift duration > 24h
	longEnd := startTime.Add(25 * time.Hour)
	_, err = svc.SubmitTimesheet(ctx, &frappe.Timesheet{
		EmployeeID: "emp-test-01",
		StartTime:  &startTime,
		EndTime:    &longEnd,
	})
	assert.Error(t, err)

	// Invalid task code
	validEnd := startTime.Add(4 * time.Hour)
	_, err = svc.SubmitTimesheet(ctx, &frappe.Timesheet{
		EmployeeID: "emp-test-01",
		StartTime:  &startTime,
		EndTime:    &validEnd,
		TaskCode:   "laser_pointer_chase",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task_code")
}

func TestTimesheets_ApproveAndReject(t *testing.T) {
	repo := workforce.NewRepository(nil)
	publisher := &testEventPublisher{}
	hrClient := &testHRClient{}
	svc := NewServiceWithIntegrations(repo, publisher, hrClient)
	ctx := context.Background()

	startTime := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 10, 15, 17, 0, 0, 0, time.UTC)

	ts, err := svc.SubmitTimesheet(ctx, &frappe.Timesheet{
		EmployeeID:   "emp-test-02",
		ShiftDate:    "2026-10-15",
		StartTime:    &startTime,
		EndTime:      &endTime,
		BreakMinutes: 60,
		TaskCode:     "feline_feeding",
		Billable:     false,
	})
	assert.NoError(t, err)
	assert.Equal(t, 7.0, ts.HoursWorked)

	// Approve
	approved, err := svc.ApproveTimesheet(ctx, ts.TimesheetID, "emp-lead-01")
	assert.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)
	assert.NotNil(t, approved.ApprovedBy)
	assert.Equal(t, "emp-lead-01", *approved.ApprovedBy)
	assert.NotNil(t, approved.FrappeTimesheetID)
	assert.Equal(t, "TS-FRAPPE-001", *approved.FrappeTimesheetID)

	// Check HR client called
	assert.Len(t, hrClient.created, 1)
	assert.Len(t, hrClient.approved, 1)

	// Check Event Publisher called
	assert.Len(t, publisher.published, 1)

	// Cannot reject approved timesheet
	_, err = svc.RejectTimesheet(ctx, ts.TimesheetID, "emp-lead-01", "late")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reject an already approved timesheet")

	// Submit another timesheet to reject
	ts2, err := svc.SubmitTimesheet(ctx, &frappe.Timesheet{
		EmployeeID: "emp-test-03",
		ShiftDate:  "2026-10-16",
		StartTime:  &startTime,
		EndTime:    &endTime,
		TaskCode:   "admin",
	})
	assert.NoError(t, err)

	rejected, err := svc.RejectTimesheet(ctx, ts2.TimesheetID, "emp-lead-01", "Missing shift report")
	assert.NoError(t, err)
	assert.Equal(t, "rejected", rejected.Status)

	// List filtering
	all, err := svc.ListTimesheets(ctx, TimesheetFilter{
		EmployeeID: "emp-test-02",
	})
	assert.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestPayroll_SalarySlipsAndRun(t *testing.T) {
	repo := workforce.NewRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	// List default seeded salary slips for Bob
	slips, err := svc.ListSalarySlips(ctx, "emp-human-bob")
	assert.NoError(t, err)
	assert.NotEmpty(t, slips)
	assert.Equal(t, "emp-human-bob", slips[0].EmployeeID)

	// Get single salary slip
	slip, err := svc.GetSalarySlip(ctx, slips[0].SalarySlipID)
	assert.NoError(t, err)
	assert.Equal(t, slips[0].SalarySlipID, slip.SalarySlipID)
	assert.Equal(t, 5833.33, slip.GrossPay)
	assert.NotEmpty(t, slip.Earnings)
	assert.NotEmpty(t, slip.Deductions)

	// Execute a new payroll run
	pe, err := svc.ProcessPayrollRun(ctx, &frappe.PayrollEntry{
		StartDate: "2026-10-01",
		EndDate:   "2026-10-31",
		Employees: []string{"emp-human-bob", "emp-human-alice"},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, pe.PayrollEntryID)
	assert.Equal(t, 2, pe.SalarySlipsCreated)
	assert.Equal(t, 10000.0, pe.TotalGrossPay)

	// Verify newly generated salary slip exists
	updatedSlips, err := svc.ListSalarySlips(ctx, "emp-human-bob")
	assert.NoError(t, err)
	assert.True(t, len(updatedSlips) >= 3)
}

