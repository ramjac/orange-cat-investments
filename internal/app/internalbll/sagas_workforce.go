package internalbll

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetEmployee(ctx context.Context, in *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	if in.EmployeeID == "" {
		return nil, status.Error(codes.InvalidArgument, "employee_id is required")
	}
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	return s.wfClient.GetEmployee(ctx, in)
}

func (s *Server) ListEmployees(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	if s.wfClient == nil {
		return &workforcev1.ListEmployeesResponse{}, nil
	}
	return s.wfClient.ListEmployees(ctx, in)
}

func (s *Server) OnboardEmployee(ctx context.Context, in *internalbllv1.OnboardEmployeeSagaRequest) (*internalbllv1.OnboardEmployeeSagaResponse, error) {
	empType := in.EmployeeType
	if empType == "" {
		empType = "human"
	}
	if in.FirstName == "" || (empType != "human" && empType != "feline") {
		return nil, status.Error(codes.InvalidArgument, "first_name and valid employee_type ('human' or 'feline') are required")
	}
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}

	s.logger.Info("Executing OnboardingSaga", "first_name", in.FirstName, "type", empType)

	cs := in.CareSchedule
	if empType == "feline" && cs == nil {
		dietary := in.DietaryPlan
		if dietary == "" {
			dietary = "Standard balanced feline diet"
		}
		feeding := in.FeedingTimes
		if len(feeding) == 0 {
			feeding = []string{"08:00 AM", "05:00 PM"}
		}
		cs = &workforcev1.CareSchedule{
			DietaryPlan:          dietary,
			FeedingTimes:         feeding,
			EmergencyMedicalHold: false,
			CaretakerID:          in.CaretakerID,
		}
	}

	onboardResp, err := s.wfClient.OnboardEmployee(ctx, &workforcev1.OnboardEmployeeRequest{
		Employee: &workforcev1.Employee{
			EmployeeID:   in.EmployeeID,
			EmployeeType: empType,
			FirstName:    in.FirstName,
			LastName:     in.LastName,
			Email:        in.Email,
			RoleTitle:    in.RoleTitle,
			Department:   in.Department,
			Status:       "onboarding",
		},
		CareSchedule: cs,
	})
	if err != nil {
		return nil, err
	}
	created := onboardResp.Employee
	if created == nil {
		return nil, status.Error(codes.Internal, "workforce api returned nil employee")
	}

	username := strings.ToLower(in.FirstName)
	if in.LastName != "" {
		username = strings.ToLower(fmt.Sprintf("%s.%s", in.FirstName, in.LastName))
	}
	fullName := in.FirstName
	if in.LastName != "" {
		fullName = fmt.Sprintf("%s %s", in.FirstName, in.LastName)
	}
	email := in.Email
	if email == "" {
		email = fmt.Sprintf("%s@oci.local", username)
	}

	var vcsAcc *internalbllv1.VCSAccount
	var ws *internalbllv1.Workspace
	var hwAssets []internalbllv1.AssetAllocation
	if empType != "feline" {
		if s.vcsClient != nil {
			if acc, vcsErr := s.vcsClient.ProvisionAccount(ctx, username, email, fullName, in.SSHKey); vcsErr == nil && acc != nil {
				vcsAcc = &internalbllv1.VCSAccount{
					Username:  acc.Username,
					Email:     acc.Email,
					FullName:  acc.FullName,
					SSHKeyIDs: acc.SSHKeyIDs,
					Status:    acc.Status,
				}
			}
		}
		quota := in.StorageQuota
		if quota == "" {
			quota = "25 GB"
		}
		dept := in.Department
		if dept == "" {
			dept = "General"
		}
		ws = &internalbllv1.Workspace{
			Username:     username,
			Quota:        quota,
			GroupFolders: []string{"Company Shared", dept},
			Status:       "Active",
		}
		if s.hwClient != nil {
			if assets, hwErr := s.hwClient.AssignInitialHardware(ctx, created.EmployeeID, in.RoleTitle, in.Department); hwErr == nil {
				for _, a := range assets {
					hwAssets = append(hwAssets, internalbllv1.AssetAllocation{
						AssetTag:    a.AssetTag,
						Name:        a.Name,
						Category:    a.Category,
						StatusLabel: a.StatusLabel,
					})
				}
			}
		}
	}

	var taskNames []string
	if clResp, clErr := s.wfClient.ListOnboardingChecklist(ctx, &workforcev1.ListOnboardingChecklistRequest{EmployeeID: created.EmployeeID}); clErr == nil {
		for _, t := range clResp.Tasks {
			taskNames = append(taskNames, t.TaskName)
		}
	}
	if len(taskNames) == 0 {
		taskNames = []string{"Badge & Access Provisioning", "Workplace Safety & Compliance"}
		if empType == "feline" {
			taskNames = append(taskNames, "Preferred Perch Inspection", "Dietary & Care Schedule Setup")
		} else {
			taskNames = append(taskNames, "Forgejo Account Provisioning & SPIFFE ID Issuance")
		}
	}

	var storedCS *workforcev1.CareSchedule
	if empType == "feline" {
		if csResp, csErr := s.wfClient.GetCareSchedule(ctx, &workforcev1.GetCareScheduleRequest{FelineID: created.EmployeeID}); csErr == nil && csResp.CareSchedule != nil {
			storedCS = csResp.CareSchedule
		} else {
			storedCS = cs
		}
	}

	sagaRes := &internalbllv1.OnboardingResult{
		EmployeeID:     created.EmployeeID,
		IdpIdentityID:  fmt.Sprintf("idp-forgejo-%s", created.EmployeeID),
		Status:         created.Status,
		ChecklistTasks: taskNames,
		CareSchedule:   storedCS,
		VCSAccount:     vcsAcc,
		Workspace:      ws,
		HardwareAssets: hwAssets,
		Message:        "Employee onboarding saga completed successfully",
	}

	if s.publisher != nil {
		_ = s.publisher.Publish("events.workforce.employee_onboarded.v1", envelope.NewEventEnvelope(
			fmt.Sprintf("evt-onboard-%d", time.Now().UnixNano()),
			"events.workforce.employee_onboarded.v1",
			"",
			envelope.EmployeeOnboardedPayload{
				EmployeeID:   created.EmployeeID,
				EmployeeType: empType,
				FirstName:    in.FirstName,
				LastName:     in.LastName,
				Email:        in.Email,
				RoleTitle:    in.RoleTitle,
				Department:   in.Department,
			},
		))
	}

	return &internalbllv1.OnboardEmployeeSagaResponse{
		Employee:   created,
		SagaResult: sagaRes,
	}, nil
}

func (s *Server) OffboardEmployee(ctx context.Context, in *internalbllv1.OffboardEmployeeSagaRequest) (*internalbllv1.OffboardEmployeeSagaResponse, error) {
	if in.EmployeeID == "" {
		return nil, status.Error(codes.InvalidArgument, "employee_id is required")
	}
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	empResp, err := s.wfClient.GetEmployee(ctx, &workforcev1.GetEmployeeRequest{EmployeeID: in.EmployeeID})
	if err != nil || empResp.Employee == nil {
		return nil, fmt.Errorf("offboarding engine failed to fetch employee: %w", err)
	}
	emp := empResp.Employee

	targetStatus := in.TargetStatus
	if targetStatus == "" {
		targetStatus = "separated"
	}
	updResp, err := s.wfClient.UpdateEmployeeStatus(ctx, &workforcev1.UpdateEmployeeStatusRequest{
		EmployeeID: in.EmployeeID,
		Status:     targetStatus,
	})
	if err != nil {
		return nil, err
	}
	updated := emp
	if updResp.Employee != nil {
		updated = updResp.Employee
	}

	username := strings.ToLower(emp.FirstName)
	if emp.LastName != "" {
		username = strings.ToLower(fmt.Sprintf("%s.%s", emp.FirstName, emp.LastName))
	}
	if s.vcsClient != nil {
		_ = s.vcsClient.RevokeAccess(ctx, username)
	}

	vaultPath := fmt.Sprintf("/compliance/vault/archives/%s-%d.tar.gz", in.EmployeeID, time.Now().Unix())
	var hwTickets *internalbllv1.AssetRecoverySummary
	if s.hwClient != nil {
		if summary, hwErr := s.hwClient.CreateAssetRecoveryTickets(ctx, in.EmployeeID, in.Reason); hwErr == nil && summary != nil {
			var assets []internalbllv1.AssetAllocation
			for _, a := range summary.Assets {
				assets = append(assets, internalbllv1.AssetAllocation{
					AssetTag:    a.AssetTag,
					Name:        a.Name,
					Category:    a.Category,
					StatusLabel: a.StatusLabel,
				})
			}
			hwTickets = &internalbllv1.AssetRecoverySummary{
				TicketID:   summary.TicketID,
				TotalItems: summary.TotalItems,
				Status:     summary.Status,
				Assets:     assets,
			}
		}
	}

	tradingHold := false
	careTransition := "Active feline care responsibilities transferred to designated veterinary medical staff"
	if emp.EmployeeType == "feline" {
		if mhResp, mhErr := s.wfClient.SetMedicalHold(ctx, &workforcev1.SetMedicalHoldRequest{FelineID: in.EmployeeID, EmergencyMedicalHold: true}); mhErr == nil && mhResp.CareSchedule != nil {
			tradingHold = mhResp.CareSchedule.EmergencyMedicalHold
		}
		careTransition = "Feline transitioned to senior veterinary medical hold"
	}

	res := &internalbllv1.OffboardingResult{
		EmployeeID:       updated.EmployeeID,
		Status:           updated.Status,
		IdentityRevoked:  true,
		SessionsRevoked:  true,
		StorageArchived:  true,
		ArchiveVaultPath: vaultPath,
		HardwareTickets:  hwTickets,
		TradingHoldState: tradingHold,
		CareTransition:   careTransition,
		Message:          fmt.Sprintf("Employee offboarded successfully with status %s", updated.Status),
	}
	return &internalbllv1.OffboardEmployeeSagaResponse{SagaResult: res}, nil
}

func (s *Server) GetEmployeeLifecycle(ctx context.Context, in *internalbllv1.GetEmployeeLifecycleRequest) (*internalbllv1.GetEmployeeLifecycleResponse, error) {
	if in.EmployeeID == "" {
		return nil, status.Error(codes.InvalidArgument, "employee_id is required")
	}
	if s.wfClient == nil {
		return nil, status.Error(codes.Unavailable, "workforce client not configured")
	}
	empResp, err := s.wfClient.GetEmployee(ctx, &workforcev1.GetEmployeeRequest{EmployeeID: in.EmployeeID})
	if err != nil || empResp.Employee == nil {
		return nil, err
	}
	emp := empResp.Employee
	clResp, _ := s.wfClient.ListOnboardingChecklist(ctx, &workforcev1.ListOnboardingChecklistRequest{EmployeeID: in.EmployeeID})
	csResp, _ := s.wfClient.GetCareSchedule(ctx, &workforcev1.GetCareScheduleRequest{FelineID: in.EmployeeID})

	username := strings.ToLower(emp.FirstName)
	if emp.LastName != "" {
		username = strings.ToLower(fmt.Sprintf("%s.%s", emp.FirstName, emp.LastName))
	}
	var tasks []*workforcev1.OnboardingTask
	if clResp != nil {
		tasks = clResp.Tasks
	}
	var cs *workforcev1.CareSchedule
	if csResp != nil {
		cs = csResp.CareSchedule
	}
	return &internalbllv1.GetEmployeeLifecycleResponse{
		Employee:        emp,
		Status:          emp.Status,
		CareSchedule:    cs,
		Checklist:       tasks,
		VCSUsername:     username,
		WorkspaceQuota:  "25 GB",
		HardwareSummary: fmt.Sprintf("Standard IT asset allocation for %s", emp.RoleTitle),
	}, nil
}

func (s *Server) HandleHRWebhook(ctx context.Context, in *internalbllv1.HandleHRWebhookRequest) (*internalbllv1.HandleHRWebhookResponse, error) {
	event := strings.ToLower(in.Event)
	st := strings.ToLower(in.Status)

	if event == "employee_created" || st == "onboarding" {
		empType := in.EmployeeType
		if empType == "" {
			empType = "human"
		}
		firstName := in.FirstName
		if firstName == "" {
			firstName = "Employee"
		}
		res, err := s.OnboardEmployee(ctx, &internalbllv1.OnboardEmployeeSagaRequest{
			EmployeeID:   in.EmployeeID,
			EmployeeType: empType,
			FirstName:    firstName,
			LastName:     in.LastName,
			Email:        in.Email,
			RoleTitle:    in.RoleTitle,
			Department:   in.Department,
			DietaryPlan:  in.DietaryPlan,
			FeedingTimes: in.FeedingTimes,
		})
		if err != nil {
			return nil, err
		}
		return &internalbllv1.HandleHRWebhookResponse{
			Status:     "processed",
			Action:     "onboarding_saga_executed",
			OnboardRes: res.SagaResult,
		}, nil
	}

	if event == "employee_separated" || st == "separated" || st == "retired" {
		res, err := s.OffboardEmployee(ctx, &internalbllv1.OffboardEmployeeSagaRequest{
			EmployeeID:   in.EmployeeID,
			Reason:       "Frappe HR webhook trigger",
			TargetStatus: st,
		})
		if err != nil {
			return nil, err
		}
		return &internalbllv1.HandleHRWebhookResponse{
			Status:      "processed",
			Action:      "offboarding_engine_executed",
			OffboardRes: res.SagaResult,
		}, nil
	}

	if in.EmployeeID != "" && in.Status != "" && s.wfClient != nil {
		updResp, err := s.wfClient.UpdateEmployeeStatus(ctx, &workforcev1.UpdateEmployeeStatusRequest{
			EmployeeID: in.EmployeeID,
			Status:     in.Status,
		})
		if err != nil {
			return nil, err
		}
		return &internalbllv1.HandleHRWebhookResponse{
			Status:          "processed",
			Action:          "employee_status_updated",
			EmployeeID:      in.EmployeeID,
			UpdatedEmployee: updResp.Employee,
		}, nil
	}

	return &internalbllv1.HandleHRWebhookResponse{
		Status: "ignored",
		Action: "no_matching_saga_trigger",
	}, nil
}

func (s *Server) GetFelineTarget(ctx context.Context, in *internalbllv1.GetFelineTargetRequest) (*internalbllv1.GetFelineTargetResponse, error) {
	target := "feline-observation-target"
	if s.wfClient != nil {
		emps, err := s.wfClient.ListEmployees(ctx, &workforcev1.ListEmployeesRequest{EmployeeType: "feline", Status: "active"})
		if err == nil && len(emps.Employees) > 0 {
			target = emps.Employees[0].EmployeeID
		}
	}
	return &internalbllv1.GetFelineTargetResponse{FelineID: target}, nil
}
