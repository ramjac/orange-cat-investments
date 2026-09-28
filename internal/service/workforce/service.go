package workforce

import (
	"context"
	"errors"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
)

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
	CreateLeaveRequest(ctx context.Context, employeeID, leaveType, startDate, endDate string, reason *string) (*workforce.LeaveRequest, error)
	GetLeaveRequest(ctx context.Context, id string) (*workforce.LeaveRequest, error)
	ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*workforce.LeaveRequest, error)
	UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*workforce.LeaveRequest, error)

	// Review Cycles
	CreateReviewCycle(ctx context.Context, employeeID, reviewType, scheduledFor string, reviewerID, notes *string) (*workforce.ReviewCycle, error)
	GetReviewCycle(ctx context.Context, id string) (*workforce.ReviewCycle, error)
	ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*workforce.ReviewCycle, error)
	UpdateReviewCycle(ctx context.Context, id string, status, reviewerID *string, score *float64, notes *string, completedAt *time.Time) (*workforce.ReviewCycle, error)

	// Electronic Veterinary Health Records (VHR)
	CreateVHRRecord(ctx context.Context, rec *workforce.VHRRecord) (*workforce.VHRRecord, error)
	ListVHRRecords(ctx context.Context, felineID string) ([]*workforce.VHRRecord, error)

	// Workplace Incidents
	CreateWorkplaceIncident(ctx context.Context, inc *workforce.WorkplaceIncident) (*workforce.WorkplaceIncident, error)
	ListWorkplaceIncidents(ctx context.Context, status string) ([]*workforce.WorkplaceIncident, error)
	UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*workforce.WorkplaceIncident, error)
}

type workforceService struct {
	repo workforce.Repository
}

func NewService(repo workforce.Repository) Service {
	return &workforceService{repo: repo}
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

func (s *workforceService) CreateLeaveRequest(ctx context.Context, employeeID, leaveType, startDate, endDate string, reason *string) (*workforce.LeaveRequest, error) {
	if employeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	switch leaveType {
	case "vacation", "sick", "catnip_break", "sabbatical":
		// valid
	default:
		return nil, errors.New("invalid leave_type: must be vacation, sick, catnip_break, or sabbatical")
	}

	if startDate == "" || endDate == "" {
		return nil, errors.New("start_date and end_date are required")
	}

	const dateLayout = "2006-01-02"
	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return nil, errors.New("start_date must be in YYYY-MM-DD format")
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return nil, errors.New("end_date must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return nil, errors.New("end_date must be on or after start_date")
	}

	req := &workforce.LeaveRequest{
		EmployeeID: employeeID,
		LeaveType:  leaveType,
		StartDate:  startDate,
		EndDate:    endDate,
		Status:     "pending",
		Reason:     reason,
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

func (s *workforceService) CreateReviewCycle(ctx context.Context, employeeID, reviewType, scheduledFor string, reviewerID, notes *string) (*workforce.ReviewCycle, error) {
	if employeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	switch reviewType {
	case "performance", "feline_health_assessment":
		// valid
	default:
		return nil, errors.New("invalid review_type: must be performance or feline_health_assessment")
	}

	if scheduledFor == "" {
		return nil, errors.New("scheduled_for date is required")
	}
	const dateLayout = "2006-01-02"
	if _, err := time.Parse(dateLayout, scheduledFor); err != nil {
		return nil, errors.New("scheduled_for must be in YYYY-MM-DD format")
	}

	rc := &workforce.ReviewCycle{
		EmployeeID:   employeeID,
		ReviewType:   reviewType,
		ScheduledFor: scheduledFor,
		Status:       "scheduled",
		ReviewerID:   reviewerID,
		Notes:        notes,
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

func (s *workforceService) UpdateReviewCycle(ctx context.Context, id string, status, reviewerID *string, score *float64, notes *string, completedAt *time.Time) (*workforce.ReviewCycle, error) {
	if id == "" {
		return nil, errors.New("review cycle id cannot be empty")
	}

	rc, err := s.repo.GetReviewCycleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if status != nil {
		switch *status {
		case "scheduled", "in_progress", "completed", "overdue":
			rc.Status = *status
		default:
			return nil, errors.New("invalid status: must be scheduled, in_progress, completed, or overdue")
		}
	}

	if reviewerID != nil {
		rc.ReviewerID = reviewerID
	}

	if score != nil {
		if *score < 0.0 || *score > 5.0 {
			return nil, errors.New("score must be between 0.00 and 5.00")
		}
		rc.Score = score
	}

	if notes != nil {
		rc.Notes = notes
	}

	if rc.Status == "completed" {
		if completedAt != nil {
			rc.CompletedAt = completedAt
		} else if rc.CompletedAt == nil {
			now := time.Now().UTC()
			rc.CompletedAt = &now
		}
	} else if completedAt != nil {
		rc.CompletedAt = completedAt
	}

	return s.repo.UpdateReviewCycle(ctx, rc)
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
	return s.repo.UpdateWorkplaceIncidentStatus(ctx, id, status, resolutionNotes)
}
