package workforce

import (
	"context"
	"errors"
	"fmt"
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
	CreateLeaveRequest(ctx context.Context, params CreateLeaveRequestParams) (*workforce.LeaveRequest, error)
	GetLeaveRequest(ctx context.Context, id string) (*workforce.LeaveRequest, error)
	ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*workforce.LeaveRequest, error)
	UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*workforce.LeaveRequest, error)

	// Review Cycles
	CreateReviewCycle(ctx context.Context, params CreateReviewCycleParams) (*workforce.ReviewCycle, error)
	GetReviewCycle(ctx context.Context, id string) (*workforce.ReviewCycle, error)
	ListReviewCycles(ctx context.Context, employeeID, status, reviewType string) ([]*workforce.ReviewCycle, error)
	UpdateReviewCycle(ctx context.Context, params UpdateReviewCycleParams) (*workforce.ReviewCycle, error)

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
	EmployeeID   string
	ReviewType   string
	ScheduledFor string
	ReviewerID   *string
	Notes        *string
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

	rc := &workforce.ReviewCycle{
		EmployeeID:   params.EmployeeID,
		ReviewType:   params.ReviewType,
		ScheduledFor: params.ScheduledFor,
		Status:       "scheduled",
		ReviewerID:   params.ReviewerID,
		Notes:        params.Notes,
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
	ID          string
	Status      *string
	ReviewerID  *string
	Score       *float64
	Notes       *string
	CompletedAt *time.Time
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

	if rc.Status == "completed" {
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
