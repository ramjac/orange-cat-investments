package handler

import (
	"context"
	"fmt"
	"time"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

func (m *mockInternalBLLClient) GetEmployee(ctx context.Context, in *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	emp, ok := m.employees[in.EmployeeID]
	if !ok {
		return &workforcev1.GetEmployeeResponse{
			Employee: &workforcev1.Employee{
				EmployeeID:   in.EmployeeID,
				FirstName:    "Test",
				EmployeeType: "human",
				Status:       "active",
			},
		}, nil
	}
	return &workforcev1.GetEmployeeResponse{Employee: emp}, nil
}

func (m *mockInternalBLLClient) ListEmployees(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*workforcev1.Employee
	for _, emp := range m.employees {
		list = append(list, emp)
	}
	return &workforcev1.ListEmployeesResponse{Employees: list}, nil
}

func (m *mockInternalBLLClient) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs, ok := m.careSchedules[in.FelineID]
	if !ok {
		return nil, fmt.Errorf("care schedule for %s not found", in.FelineID)
	}
	return &workforcev1.GetCareScheduleResponse{CareSchedule: cs}, nil
}

func (m *mockInternalBLLClient) UpdateCareSchedule(ctx context.Context, in *workforcev1.UpdateCareScheduleRequest) (*workforcev1.UpdateCareScheduleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs := &workforcev1.CareSchedule{
		FelineID:             in.FelineID,
		DietaryPlan:          in.DietaryPlan,
		FeedingTimes:         in.FeedingTimes,
		SpecialMedicalNeeds:  in.SpecialMedicalNeeds,
		PreferredPerchZone:   in.PreferredPerchZone,
		EmergencyMedicalHold: in.EmergencyMedicalHold,
		CaretakerID:          in.CaretakerID,
	}
	m.careSchedules[in.FelineID] = cs
	return &workforcev1.UpdateCareScheduleResponse{CareSchedule: cs}, nil
}

func (m *mockInternalBLLClient) SetMedicalHold(ctx context.Context, in *workforcev1.SetMedicalHoldRequest) (*workforcev1.SetMedicalHoldResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs, ok := m.careSchedules[in.FelineID]
	if !ok {
		cs = &workforcev1.CareSchedule{FelineID: in.FelineID}
		m.careSchedules[in.FelineID] = cs
	}
	cs.EmergencyMedicalHold = in.EmergencyMedicalHold
	return &workforcev1.SetMedicalHoldResponse{CareSchedule: cs}, nil
}

func (m *mockInternalBLLClient) HandleHRWebhook(ctx context.Context, in *internalbllv1.HandleHRWebhookRequest) (*internalbllv1.HandleHRWebhookResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	action := "onboarding_saga_executed"
	if in.Event == "employee_separated" || in.Status == "separated" {
		action = "offboarding_engine_executed"
	}
	return &internalbllv1.HandleHRWebhookResponse{
		Status: "processed",
		Action: action,
		OnboardRes: &internalbllv1.OnboardingResult{
			Status: "completed",
		},
	}, nil
}

func (m *mockInternalBLLClient) OnboardEmployee(ctx context.Context, in *internalbllv1.OnboardEmployeeSagaRequest) (*internalbllv1.OnboardEmployeeSagaResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	empID := in.EmployeeID
	if empID == "" {
		empID = fmt.Sprintf("emp-human-%d", time.Now().UnixNano())
	}
	m.employees[empID] = &workforcev1.Employee{
		EmployeeID:   empID,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		EmployeeType: in.EmployeeType,
		RoleTitle:    in.RoleTitle,
		Department:   in.Department,
		Status:       "onboarding",
	}
	return &internalbllv1.OnboardEmployeeSagaResponse{
		SagaResult: &internalbllv1.OnboardingResult{
			EmployeeID:     empID,
			Status:         "onboarding",
			VCSAccount:     &internalbllv1.VCSAccount{Username: "nala.simba", Email: in.Email},
			Workspace:      &internalbllv1.Workspace{Username: "nala.simba", Quota: "25 GB", Status: "active"},
			HardwareAssets: []internalbllv1.AssetAllocation{{AssetTag: "asset-laptop-01"}},
		},
	}, nil
}

func (m *mockInternalBLLClient) OffboardEmployee(ctx context.Context, in *internalbllv1.OffboardEmployeeSagaRequest) (*internalbllv1.OffboardEmployeeSagaResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if emp, ok := m.employees[in.EmployeeID]; ok {
		emp.Status = in.TargetStatus
	}
	return &internalbllv1.OffboardEmployeeSagaResponse{
		SagaResult: &internalbllv1.OffboardingResult{
			EmployeeID:      in.EmployeeID,
			Status:          "separated",
			IdentityRevoked: true,
			SessionsRevoked: true,
			StorageArchived: true,
			HardwareTickets: &internalbllv1.AssetRecoverySummary{
				TicketID: "ticket-hw-01",
				Status:   "recovered",
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) GetEmployeeLifecycle(ctx context.Context, in *internalbllv1.GetEmployeeLifecycleRequest) (*internalbllv1.GetEmployeeLifecycleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	emp, ok := m.employees[in.EmployeeID]
	if !ok {
		emp = &workforcev1.Employee{
			EmployeeID: in.EmployeeID,
			FirstName:  "Nala",
			RoleTitle:  "Junior Facilities Engineer",
			Status:     "onboarding",
		}
	}
	return &internalbllv1.GetEmployeeLifecycleResponse{
		Employee:        emp,
		Status:          emp.Status,
		CareSchedule:    m.careSchedules[in.EmployeeID],
		VCSUsername:     "nala.simba",
		WorkspaceQuota:  "25 GB",
		HardwareSummary: "Standard laptop allocation",
	}, nil
}

func (m *mockInternalBLLClient) ListLeaveRequests(ctx context.Context, in *workforcev1.ListLeaveRequestsRequest) (*workforcev1.ListLeaveRequestsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*workforcev1.LeaveRequest
	for _, lr := range m.leaveRequests {
		if in.EmployeeID != "" && lr.EmployeeID != in.EmployeeID {
			continue
		}
		if in.Status != "" && lr.Status != in.Status {
			continue
		}
		list = append(list, lr)
	}
	return &workforcev1.ListLeaveRequestsResponse{LeaveRequests: list}, nil
}

func (m *mockInternalBLLClient) SubmitLeaveRequest(ctx context.Context, in *workforcev1.SubmitLeaveRequestMessage) (*workforcev1.SubmitLeaveResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	validTypes := map[string]bool{
		"annual":       true,
		"sick":         true,
		"catnip_break": true,
		"feline_care":  true,
		"training":     true,
	}
	if !validTypes[in.LeaveType] {
		return nil, fmt.Errorf("invalid leave_type %q", in.LeaveType)
	}
	id := fmt.Sprintf("leave-%d", len(m.leaveRequests)+1)
	lr := &workforcev1.LeaveRequest{
		LeaveID:    id,
		EmployeeID: in.EmployeeID,
		LeaveType:  in.LeaveType,
		StartDate:  in.StartDate,
		EndDate:    in.EndDate,
		Reason:     in.Reason,
		Status:     "pending",
	}
	m.leaveRequests[id] = lr
	return &workforcev1.SubmitLeaveResponse{LeaveRequest: lr}, nil
}

func (m *mockInternalBLLClient) GetLeaveRequest(ctx context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lr, ok := m.leaveRequests[in.LeaveID]
	if !ok {
		return nil, fmt.Errorf("leave request %s not found", in.LeaveID)
	}
	return &workforcev1.GetLeaveRequestResponse{LeaveRequest: lr}, nil
}

func (m *mockInternalBLLClient) UpdateLeaveStatus(ctx context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lr, ok := m.leaveRequests[in.LeaveID]
	if !ok {
		return nil, fmt.Errorf("leave request %s not found", in.LeaveID)
	}
	lr.Status = in.Status
	return &workforcev1.UpdateLeaveStatusResponse{LeaveRequest: lr}, nil
}

func (m *mockInternalBLLClient) ListReviewCycles(ctx context.Context, in *workforcev1.ListReviewCyclesRequest) (*workforcev1.ListReviewCyclesResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*workforcev1.ReviewCycle
	for _, rc := range m.reviewCycles {
		if in.EmployeeID != "" && rc.EmployeeID != in.EmployeeID {
			continue
		}
		if in.Status != "" && rc.Status != in.Status {
			continue
		}
		list = append(list, rc)
	}
	return &workforcev1.ListReviewCyclesResponse{ReviewCycles: list}, nil
}

func (m *mockInternalBLLClient) CreateReviewCycle(ctx context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	validTypes := map[string]bool{
		"feline_health_assessment": true,
		"performance":              true,
		"probationary":             true,
		"annual":                   true,
	}
	if !validTypes[in.ReviewType] {
		return nil, fmt.Errorf("invalid review_type %q", in.ReviewType)
	}
	id := fmt.Sprintf("rev-%d", len(m.reviewCycles)+1)
	rc := &workforcev1.ReviewCycle{
		ReviewID:           id,
		EmployeeID:         in.EmployeeID,
		ReviewType:         in.ReviewType,
		ReviewPeriod:       in.ReviewPeriod,
		ScheduledFor:       in.ScheduledFor,
		Phase:              in.Phase,
		ReviewerID:         in.ReviewerID,
		Status:             "scheduled",
		Notes:              in.Notes,
		SelfAppraisal:      in.SelfAppraisal,
		SelfScore:          in.SelfScore,
		WellnessAuditScore: in.WellnessAuditScore,
	}
	m.reviewCycles[id] = rc
	return &workforcev1.CreateReviewCycleResponse{ReviewCycle: rc}, nil
}

func (m *mockInternalBLLClient) GetReviewCycle(ctx context.Context, in *workforcev1.GetReviewCycleRequest) (*workforcev1.GetReviewCycleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rc, ok := m.reviewCycles[in.ReviewID]
	if !ok {
		return nil, fmt.Errorf("review cycle %s not found", in.ReviewID)
	}
	return &workforcev1.GetReviewCycleResponse{ReviewCycle: rc}, nil
}

func (m *mockInternalBLLClient) UpdateReviewCycle(ctx context.Context, in *workforcev1.UpdateReviewCycleRequest) (*workforcev1.UpdateReviewCycleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if in.HasScore && (in.Score > 5.0 || in.Score < 0.0) {
		return nil, fmt.Errorf("score must be between 0.0 and 5.0")
	}
	rc, ok := m.reviewCycles[in.ReviewID]
	if !ok {
		rc = &workforcev1.ReviewCycle{ReviewID: in.ReviewID}
		m.reviewCycles[in.ReviewID] = rc
	}
	if in.Status != "" {
		rc.Status = in.Status
	}
	if in.Phase != "" {
		rc.Phase = in.Phase
	}
	if in.HasScore {
		rc.Score = in.Score
	}
	if in.Notes != "" {
		rc.Notes = in.Notes
	}
	rc.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	return &workforcev1.UpdateReviewCycleResponse{ReviewCycle: rc}, nil
}

func (m *mockInternalBLLClient) SubmitSelfAppraisal(ctx context.Context, in *workforcev1.SubmitSelfAppraisalRequest) (*workforcev1.SubmitSelfAppraisalResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rc, ok := m.reviewCycles[in.ReviewID]
	if !ok {
		rc = &workforcev1.ReviewCycle{ReviewID: in.ReviewID}
		m.reviewCycles[in.ReviewID] = rc
	}
	rc.SelfAppraisal = in.Appraisal
	rc.SelfScore = in.Score
	rc.Phase = "peer_feedback"
	return &workforcev1.SubmitSelfAppraisalResponse{ReviewCycle: rc}, nil
}

func (m *mockInternalBLLClient) SubmitPeerFeedback(ctx context.Context, in *workforcev1.SubmitPeerFeedbackRequest) (*workforcev1.SubmitPeerFeedbackResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rc, ok := m.reviewCycles[in.ReviewID]
	if !ok {
		rc = &workforcev1.ReviewCycle{ReviewID: in.ReviewID}
		m.reviewCycles[in.ReviewID] = rc
	}
	rc.PeerFeedback = in.Feedback
	rc.PeerScore = in.Score
	rc.PeerReviewerID = in.PeerReviewerID
	rc.Phase = "manager_review"
	return &workforcev1.SubmitPeerFeedbackResponse{ReviewCycle: rc}, nil
}

func (m *mockInternalBLLClient) SubmitManagerAssessment(ctx context.Context, in *workforcev1.SubmitManagerAssessmentRequest) (*workforcev1.SubmitManagerAssessmentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rc, ok := m.reviewCycles[in.ReviewID]
	if !ok {
		rc = &workforcev1.ReviewCycle{ReviewID: in.ReviewID}
		m.reviewCycles[in.ReviewID] = rc
	}
	rc.ManagerAssessment = in.Assessment
	rc.ManagerScore = in.Score
	rc.Score = in.Score
	rc.KPIRatings = in.KPIRatings
	rc.Phase = "completed"
	rc.Status = "completed"
	return &workforcev1.SubmitManagerAssessmentResponse{ReviewCycle: rc}, nil
}

func (m *mockInternalBLLClient) SubmitVeterinaryAudit(ctx context.Context, in *workforcev1.SubmitVeterinaryAuditRequest) (*workforcev1.SubmitVeterinaryAuditResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rc, ok := m.reviewCycles[in.ReviewID]
	if !ok {
		rc = &workforcev1.ReviewCycle{ReviewID: in.ReviewID}
		m.reviewCycles[in.ReviewID] = rc
	}
	rc.WellnessAuditScore = in.WellnessScore
	rc.HabitatRecommendations = in.Recommendations
	rc.ClinicalAuditorID = in.AuditorID
	rc.VHRRecordID = in.VHRRecordID
	return &workforcev1.SubmitVeterinaryAuditResponse{ReviewCycle: rc}, nil
}
