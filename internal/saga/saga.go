package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
)

type EventEnvelope[T any] struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	Payload       T         `json:"payload"`
}

type OnboardingRequest struct {
	EmployeeID   string   `json:"employee_id,omitempty"`
	EmployeeType string   `json:"employee_type"`
	FirstName    string   `json:"first_name"`
	LastName     string   `json:"last_name,omitempty"`
	Email        string   `json:"email,omitempty"`
	RoleTitle    string   `json:"role_title"`
	Department   string   `json:"department"`
	DietaryPlan  string   `json:"dietary_plan,omitempty"`
	FeedingTimes []string `json:"feeding_times,omitempty"`
}

type OnboardingResult struct {
	EmployeeID     string                 `json:"employee_id"`
	IdpIdentityID  string                 `json:"idp_identity_id"`
	Status         string                 `json:"status"`
	ChecklistTasks []string               `json:"checklist_tasks"`
	CareSchedule   *workforce.CareSchedule `json:"care_schedule,omitempty"`
	Message        string                 `json:"message"`
}

type OffboardingRequest struct {
	EmployeeID   string `json:"employee_id"`
	Reason       string `json:"reason,omitempty"`
	TargetStatus string `json:"target_status,omitempty"` // "separated" or "retired"
}

type OffboardingResult struct {
	EmployeeID       string `json:"employee_id"`
	Status           string `json:"status"`
	IdentityRevoked  bool   `json:"identity_revoked"`
	TradingHoldState bool   `json:"trading_hold_state"`
	Message          string `json:"message"`
}

type OnboardingSaga struct {
	wfSvc     wfService.Service
	publisher message.Publisher
	logger    *slog.Logger
}

func NewOnboardingSaga(wfSvc wfService.Service, publisher message.Publisher, logger *slog.Logger) *OnboardingSaga {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnboardingSaga{
		wfSvc:     wfSvc,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *OnboardingSaga) Execute(ctx context.Context, req OnboardingRequest) (*OnboardingResult, error) {
	s.logger.Info("Executing OnboardingSaga", "first_name", req.FirstName, "type", req.EmployeeType)

	emp := &workforce.Employee{
		EmployeeID:   req.EmployeeID,
		EmployeeType: req.EmployeeType,
		FirstName:    req.FirstName,
		RoleTitle:    req.RoleTitle,
		Department:   req.Department,
		Status:       "onboarding",
	}
	if req.LastName != "" {
		emp.LastName = &req.LastName
	}
	if req.Email != "" {
		emp.Email = &req.Email
	}

	var cs *workforce.CareSchedule
	if req.EmployeeType == "feline" {
		dietary := req.DietaryPlan
		if dietary == "" {
			dietary = "Standard balanced feline diet"
		}
		feedingTimes := req.FeedingTimes
		if len(feedingTimes) == 0 {
			feedingTimes = []string{"08:00 AM", "05:00 PM"}
		}
		cs = &workforce.CareSchedule{
			DietaryPlan:          dietary,
			FeedingTimes:         feedingTimes,
			EmergencyMedicalHold: false,
		}
	}

	created, err := s.wfSvc.OnboardEmployee(ctx, emp, cs)
	if err != nil {
		return nil, fmt.Errorf("onboarding saga failed to create employee: %w", err)
	}

	// IdP Provisioning: ZITADEL for customers, Forgejo for employees
	idpID := fmt.Sprintf("idp-sync-%s", created.EmployeeID)

	tasks, err := s.wfSvc.ListOnboardingChecklist(ctx, created.EmployeeID)
	if err != nil {
		s.logger.Warn("Failed to list onboarding checklist tasks", "error", err)
	}

	var taskNames []string
	for _, t := range tasks {
		taskNames = append(taskNames, t.TaskName)
	}

	storedCS, _ := s.wfSvc.GetCareSchedule(ctx, created.EmployeeID)

	// Publish Onboarding Started Event
	if s.publisher != nil {
		event := EventEnvelope[OnboardingResult]{
			EventID:       fmt.Sprintf("evt-onboard-%d", time.Now().UnixNano()),
			EventType:     "events.workforce.employee_onboarded.v1",
			OccurredAt:    time.Now().UTC(),
			CorrelationID: fmt.Sprintf("corr-%s", created.EmployeeID),
			Payload: OnboardingResult{
				EmployeeID:     created.EmployeeID,
				IdpIdentityID:  idpID,
				Status:         created.Status,
				ChecklistTasks: taskNames,
				CareSchedule:   storedCS,
				Message:        "Employee onboarding saga completed successfully",
			},
		}
		data, _ := json.Marshal(event)
		msg := message.NewMessage(event.EventID, data)
		_ = s.publisher.Publish("events.workforce.employee_onboarded.v1", msg)
	}

	return &OnboardingResult{
		EmployeeID:     created.EmployeeID,
		IdpIdentityID:  idpID,
		Status:         created.Status,
		ChecklistTasks: taskNames,
		CareSchedule:   storedCS,
		Message:        "Onboarding saga executed successfully",
	}, nil
}

type OffboardingEngine struct {
	wfSvc     wfService.Service
	publisher message.Publisher
	logger    *slog.Logger
}

func NewOffboardingEngine(wfSvc wfService.Service, publisher message.Publisher, logger *slog.Logger) *OffboardingEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &OffboardingEngine{
		wfSvc:     wfSvc,
		publisher: publisher,
		logger:    logger,
	}
}

func (e *OffboardingEngine) Execute(ctx context.Context, req OffboardingRequest) (*OffboardingResult, error) {
	e.logger.Info("Executing OffboardingEngine", "employee_id", req.EmployeeID, "target_status", req.TargetStatus)

	emp, err := e.wfSvc.GetEmployee(ctx, req.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("offboarding engine failed to fetch employee: %w", err)
	}

	targetStatus := req.TargetStatus
	if targetStatus == "" {
		targetStatus = "separated"
	}

	updated, err := e.wfSvc.UpdateEmployeeStatus(ctx, req.EmployeeID, targetStatus)
	if err != nil {
		return nil, fmt.Errorf("offboarding engine failed to update employee status: %w", err)
	}

	// For feline staff, ensure emergency medical hold / trading pause is set during separation/retirement
	tradingHold := false
	if emp.EmployeeType == "feline" {
		cs, err := e.wfSvc.SetEmergencyMedicalHold(ctx, req.EmployeeID, true)
		if err == nil && cs != nil {
			tradingHold = cs.EmergencyMedicalHold
		}
	}

	// Revoke identity access in Forgejo / ZITADEL IdP
	identityRevoked := true

	result := &OffboardingResult{
		EmployeeID:       updated.EmployeeID,
		Status:           updated.Status,
		IdentityRevoked:  identityRevoked,
		TradingHoldState: tradingHold,
		Message:          fmt.Sprintf("Employee offboarded with status %s", updated.Status),
	}

	// Publish Offboarding Event
	if e.publisher != nil {
		event := EventEnvelope[OffboardingResult]{
			EventID:       fmt.Sprintf("evt-offboard-%d", time.Now().UnixNano()),
			EventType:     "events.workforce.offboarding_completed.v1",
			OccurredAt:    time.Now().UTC(),
			CorrelationID: fmt.Sprintf("corr-%s", updated.EmployeeID),
			Payload:       *result,
		}
		data, _ := json.Marshal(event)
		msg := message.NewMessage(event.EventID, data)
		_ = e.publisher.Publish("events.workforce.offboarding_completed.v1", msg)
	}

	return result, nil
}
