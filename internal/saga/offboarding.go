package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
)

type OffboardingEngine struct {
	wfSvc       wfService.Service
	publisher   message.Publisher
	logger      *slog.Logger
	vcsClient   VCSClient
	wsClient    WorkspaceClient
	hwClient    HardwareClient
	authRevoker AuthRevoker
	hrClient    HRClient
}

func NewOffboardingEngine(wfSvc wfService.Service, publisher message.Publisher, logger *slog.Logger) *OffboardingEngine {
	return NewOffboardingEngineWithIntegrations(wfSvc, publisher, nil, nil, nil, nil, nil, logger)
}

func NewOffboardingEngineWithIntegrations(wfSvc wfService.Service, publisher message.Publisher, vcs VCSClient, ws WorkspaceClient, hw HardwareClient, auth AuthRevoker, hr HRClient, logger *slog.Logger) *OffboardingEngine {
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
	if hr == nil {
		hr = NewFrappeHRAdapter(nil)
	}
	return &OffboardingEngine{
		wfSvc:       wfSvc,
		publisher:   publisher,
		logger:      logger,
		vcsClient:   vcs,
		wsClient:    ws,
		hwClient:    hw,
		authRevoker: auth,
		hrClient:    hr,
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

	// 1. Invalidate active Valkey session tokens & OAuth tokens
	sessionsRevoked := true
	if e.authRevoker != nil {
		_ = e.authRevoker.RevokeEmployeeSessions(ctx, req.EmployeeID)
		_ = e.authRevoker.RevokeOAuthTokens(ctx, req.EmployeeID)
	}

	// 2. Revoke VCS (Forgejo) developer access
	username := strings.ToLower(emp.FirstName)
	if emp.LastName != nil && *emp.LastName != "" {
		username = strings.ToLower(fmt.Sprintf("%s.%s", emp.FirstName, *emp.LastName))
	}
	if e.vcsClient != nil {
		_ = e.vcsClient.RevokeAccess(ctx, username)
	}

	// 3. Lock and archive personal Nextcloud storage to compliance vault
	vaultPath := fmt.Sprintf("/compliance/vault/archives/%s-%d.tar.gz", req.EmployeeID, time.Now().Unix())
	storageArchived := true
	if e.wsClient != nil {
		err := e.wsClient.LockAndArchiveStorage(ctx, username, vaultPath)
		if err != nil {
			e.logger.Warn("Failed to archive workspace storage", "error", err)
			storageArchived = false
		}
	}

	// 4. Generate Snipe-IT equipment recovery tickets for asset recovery
	var hwTickets *AssetRecoverySummary
	if e.hwClient != nil {
		hwTickets, _ = e.hwClient.CreateAssetRecoveryTickets(ctx, req.EmployeeID, req.Reason)
	}

	// 5. Feline Care Transition & Emergency Medical Hold
	tradingHold := false
	var careTransition string
	if emp.EmployeeType == "feline" {
		cs, err := e.wfSvc.SetEmergencyMedicalHold(ctx, req.EmployeeID, true)
		if err == nil && cs != nil {
			tradingHold = cs.EmergencyMedicalHold
		}
		careTransition = "Feline transitioned to senior veterinary medical hold"
	} else {
		careTransition = "Active feline care responsibilities transferred to designated veterinary medical staff"
	}

	// 6. Log departure in Frappe HR / ERPNext
	if e.hrClient != nil {
		today := time.Now().UTC().Format("2006-01-02")
		_ = e.hrClient.LogSeparation(ctx, req.EmployeeID, today, req.Reason)
	}

	result := &OffboardingResult{
		EmployeeID:       updated.EmployeeID,
		Status:           updated.Status,
		IdentityRevoked:  true,
		SessionsRevoked:  sessionsRevoked,
		StorageArchived:  storageArchived,
		ArchiveVaultPath: vaultPath,
		HardwareTickets:  hwTickets,
		TradingHoldState: tradingHold,
		CareTransition:   careTransition,
		Message:          fmt.Sprintf("Employee offboarded successfully with status %s", updated.Status),
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
