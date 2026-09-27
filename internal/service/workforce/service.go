package workforce

import (
	"context"
	"errors"

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
