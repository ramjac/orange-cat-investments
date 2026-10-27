package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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

// VCSAccount represents a developer account provisioned in VCS (Forgejo).
type VCSAccount struct {
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	FullName  string  `json:"full_name"`
	SSHKeyIDs []int64 `json:"ssh_key_ids,omitempty"`
	Status    string  `json:"status"` // "Active", "Suspended"
}

// Workspace represents an intranet/storage workspace provisioned in Nextcloud.
type Workspace struct {
	Username     string   `json:"username"`
	Quota        string   `json:"quota"`
	GroupFolders []string `json:"group_folders"`
	Status       string   `json:"status"` // "Active", "Archived"
}

// AssetAllocation represents an IT hardware asset assigned in Snipe-IT.
type AssetAllocation struct {
	AssetTag    string `json:"asset_tag"`
	Name        string `json:"name"`
	Category    string `json:"category"` // e.g. "Laptop", "Workstation", "Badge", "Pebble Watch"
	StatusLabel string `json:"status_label"`
}

// AssetRecoverySummary represents equipment recovery tickets generated upon departure.
type AssetRecoverySummary struct {
	TicketID   string            `json:"ticket_id"`
	TotalItems int               `json:"total_items"`
	Status     string            `json:"status"`
	Assets     []AssetAllocation `json:"assets"`
}

// VCSClient defines source control and developer account operations.
type VCSClient interface {
	ProvisionAccount(ctx context.Context, username, email, fullName, sshKey string) (*VCSAccount, error)
	RevokeAccess(ctx context.Context, username string) error
}

// WorkspaceClient defines intranet and document storage operations.
type WorkspaceClient interface {
	ProvisionWorkspace(ctx context.Context, username, email, department, quota string) (*Workspace, error)
	LockAndArchiveStorage(ctx context.Context, username, vaultPath string) error
}

// HardwareClient defines IT hardware inventory operations.
type HardwareClient interface {
	AssignInitialHardware(ctx context.Context, employeeID, roleTitle, department string) ([]AssetAllocation, error)
	CreateAssetRecoveryTickets(ctx context.Context, employeeID, reason string) (*AssetRecoverySummary, error)
}

// AuthRevoker defines session and token revocation operations.
type AuthRevoker interface {
	RevokeEmployeeSessions(ctx context.Context, employeeID string) error
	RevokeOAuthTokens(ctx context.Context, employeeID string) error
}

// HRClient defines employee lifecycle synchronization with Frappe HR.
type HRClient interface {
	UpdateEmployeeStatus(ctx context.Context, employeeID, status string) error
	LogSeparation(ctx context.Context, employeeID, departureDate, reason string) error
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
	CaretakerID  string   `json:"caretaker_id,omitempty"`
	SSHKey       string   `json:"ssh_key,omitempty"`
	StorageQuota string   `json:"storage_quota,omitempty"`
}

type OnboardingResult struct {
	EmployeeID     string                  `json:"employee_id"`
	IdpIdentityID  string                  `json:"idp_identity_id"`
	Status         string                  `json:"status"`
	ChecklistTasks []string                `json:"checklist_tasks"`
	CareSchedule   *workforce.CareSchedule `json:"care_schedule,omitempty"`
	VCSAccount     *VCSAccount             `json:"vcs_account,omitempty"`
	Workspace      *Workspace              `json:"workspace,omitempty"`
	HardwareAssets []AssetAllocation       `json:"hardware_assets,omitempty"`
	Message        string                  `json:"message"`
}

type OffboardingRequest struct {
	EmployeeID   string `json:"employee_id"`
	Reason       string `json:"reason,omitempty"`
	TargetStatus string `json:"target_status,omitempty"` // "separated" or "retired"
}

type OffboardingResult struct {
	EmployeeID       string                `json:"employee_id"`
	Status           string                `json:"status"`
	IdentityRevoked  bool                  `json:"identity_revoked"`
	SessionsRevoked  bool                  `json:"sessions_revoked"`
	StorageArchived  bool                  `json:"storage_archived"`
	ArchiveVaultPath string                `json:"archive_vault_path,omitempty"`
	HardwareTickets  *AssetRecoverySummary `json:"hardware_tickets,omitempty"`
	TradingHoldState bool                  `json:"trading_hold_state"`
	CareTransition   string                `json:"care_transition,omitempty"`
	Message          string                `json:"message"`
}

type OnboardingSaga struct {
	wfSvc     wfService.Service
	publisher message.Publisher
	logger    *slog.Logger
	vcsClient VCSClient
	wsClient  WorkspaceClient
	hwClient  HardwareClient
}

func NewOnboardingSaga(wfSvc wfService.Service, publisher message.Publisher, logger *slog.Logger) *OnboardingSaga {
	return NewOnboardingSagaWithIntegrations(wfSvc, publisher, nil, nil, nil, logger)
}

func NewOnboardingSagaWithIntegrations(wfSvc wfService.Service, publisher message.Publisher, vcs VCSClient, ws WorkspaceClient, hw HardwareClient, logger *slog.Logger) *OnboardingSaga {
	if logger == nil {
		logger = slog.Default()
	}
	if vcs == nil {
		vcs = NewForgejoVCSAdapter(nil)
	}
	if ws == nil {
		ws = NewNextcloudWorkspaceAdapter(nil)
	}
	if hw == nil {
		hw = NewSnipeITHardwareAdapter(nil)
	}
	return &OnboardingSaga{
		wfSvc:     wfSvc,
		publisher: publisher,
		logger:    logger,
		vcsClient: vcs,
		wsClient:  ws,
		hwClient:  hw,
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

	cs := s.buildCareSchedule(ctx, req)

	created, err := s.wfSvc.OnboardEmployee(ctx, emp, cs)
	if err != nil {
		return nil, fmt.Errorf("onboarding saga failed to create employee: %w", err)
	}

	idpID := fmt.Sprintf("idp-forgejo-%s", created.EmployeeID)
	username, fullName, email := deriveUserCredentials(req)

	vcsAcc, ws, hwAssets := s.provisionITResources(ctx, req, created.EmployeeID, username, email, fullName)

	taskNames := s.getChecklistTasks(ctx, req, created.EmployeeID)

	storedCS, _ := s.wfSvc.GetCareSchedule(ctx, created.EmployeeID)

	result := &OnboardingResult{
		EmployeeID:     created.EmployeeID,
		IdpIdentityID:  idpID,
		Status:         created.Status,
		ChecklistTasks: taskNames,
		CareSchedule:   storedCS,
		VCSAccount:     vcsAcc,
		Workspace:      ws,
		HardwareAssets: hwAssets,
		Message:        "Employee onboarding saga completed successfully",
	}

	s.publishOnboardingEvent(created.EmployeeID, result)

	return result, nil
}

func (s *OnboardingSaga) buildCareSchedule(ctx context.Context, req OnboardingRequest) *workforce.CareSchedule {
	if req.EmployeeType != "feline" {
		return nil
	}
	dietary := req.DietaryPlan
	if dietary == "" {
		dietary = "Standard balanced feline diet"
	}
	feedingTimes := req.FeedingTimes
	if len(feedingTimes) == 0 {
		feedingTimes = []string{"08:00 AM", "05:00 PM"}
	}
	var caretakerPtr *string
	if req.CaretakerID != "" {
		caretakerPtr = &req.CaretakerID
	} else if humanEmps, err := s.wfSvc.ListEmployees(ctx, "human", "active"); err == nil && len(humanEmps) > 0 {
		id := humanEmps[0].EmployeeID
		caretakerPtr = &id
	}
	return &workforce.CareSchedule{
		DietaryPlan:          dietary,
		FeedingTimes:         feedingTimes,
		EmergencyMedicalHold: false,
		CaretakerID:          caretakerPtr,
	}
}

func deriveUserCredentials(req OnboardingRequest) (string, string, string) {
	username := strings.ToLower(req.FirstName)
	if req.LastName != "" {
		username = strings.ToLower(fmt.Sprintf("%s.%s", req.FirstName, req.LastName))
	} else if req.Email != "" {
		parts := strings.Split(req.Email, "@")
		username = parts[0]
	}
	fullName := req.FirstName
	if req.LastName != "" {
		fullName = fmt.Sprintf("%s %s", req.FirstName, req.LastName)
	}
	email := req.Email
	if email == "" {
		email = fmt.Sprintf("%s@oci.local", username)
	}
	return username, fullName, email
}

func (s *OnboardingSaga) provisionITResources(ctx context.Context, req OnboardingRequest, employeeID, username, email, fullName string) (*VCSAccount, *Workspace, []AssetAllocation) {
	if req.EmployeeType == "feline" {
		return nil, nil, nil
	}

	var vcsAcc *VCSAccount
	var ws *Workspace
	var hwAssets []AssetAllocation
	var err error

	// 1. Provision VCS Account in Forgejo
	if s.vcsClient != nil {
		vcsAcc, err = s.vcsClient.ProvisionAccount(ctx, username, email, fullName, req.SSHKey)
		if err != nil {
			s.logger.Warn("VCS provisioning warning", "error", err)
		}
	}

	// 2. Provision Storage Workspace in Nextcloud
	if s.wsClient != nil {
		ws, err = s.wsClient.ProvisionWorkspace(ctx, username, email, req.Department, req.StorageQuota)
		if err != nil {
			s.logger.Warn("Workspace provisioning warning", "error", err)
		}
	}

	// 3. Allocate Initial Hardware in Snipe-IT
	if s.hwClient != nil {
		hwAssets, err = s.hwClient.AssignInitialHardware(ctx, employeeID, req.RoleTitle, req.Department)
		if err != nil {
			s.logger.Warn("Hardware asset allocation warning", "error", err)
		}
	}

	return vcsAcc, ws, hwAssets
}

func (s *OnboardingSaga) getChecklistTasks(ctx context.Context, req OnboardingRequest, employeeID string) []string {
	tasks, err := s.wfSvc.ListOnboardingChecklist(ctx, employeeID)
	if err != nil {
		s.logger.Warn("Failed to list onboarding checklist tasks", "error", err)
	}

	var taskNames []string
	for _, t := range tasks {
		taskNames = append(taskNames, t.TaskName)
	}

	// Enrich task list with COTS provisioning verification
	if req.EmployeeType != "feline" {
		hasVCS := false
		for _, name := range taskNames {
			if strings.Contains(name, "Forgejo") {
				hasVCS = true
				break
			}
		}
		if !hasVCS {
			taskNames = append(taskNames, "Forgejo Account Provisioning & SPIFFE ID Issuance")
		}
	}

	return taskNames
}

func (s *OnboardingSaga) publishOnboardingEvent(employeeID string, result *OnboardingResult) {
	if s.publisher == nil {
		return
	}
	event := EventEnvelope[OnboardingResult]{
		EventID:       fmt.Sprintf("evt-onboard-%d", time.Now().UnixNano()),
		EventType:     "events.workforce.employee_onboarded.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: fmt.Sprintf("corr-%s", employeeID),
		Payload:       *result,
	}
	data, _ := json.Marshal(event)
	msg := message.NewMessage(event.EventID, data)
	_ = s.publisher.Publish("events.workforce.employee_onboarded.v1", msg)
}

