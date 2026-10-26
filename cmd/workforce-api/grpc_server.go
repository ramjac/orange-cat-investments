package main

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WorkforceGRPCServer struct {
	workforcev1.UnimplementedWorkforceServiceServer
	svc          workforce_svc.Service
	repo         workforce.Repository
	enableFeline bool
}

func NewWorkforceGRPCServer(svc workforce_svc.Service, enableFeline ...bool) *WorkforceGRPCServer {
	enabled := true
	if len(enableFeline) > 0 {
		enabled = enableFeline[0]
	}
	return &WorkforceGRPCServer{svc: svc, enableFeline: enabled}
}

func NewWorkforceGRPCServerWithRepo(svc workforce_svc.Service, repo workforce.Repository, enableFeline ...bool) *WorkforceGRPCServer {
	srv := NewWorkforceGRPCServer(svc, enableFeline...)
	srv.repo = repo
	return srv
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func floatVal(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func toProtoEmployee(emp *workforce.Employee) *workforcev1.Employee {
	if emp == nil {
		return nil
	}
	hiredAt := ""
	if !emp.HiredAt.IsZero() {
		hiredAt = emp.HiredAt.Format(time.RFC3339)
	}
	createdAt := ""
	if !emp.CreatedAt.IsZero() {
		createdAt = emp.CreatedAt.Format(time.RFC3339)
	}
	updatedAt := ""
	if !emp.UpdatedAt.IsZero() {
		updatedAt = emp.UpdatedAt.Format(time.RFC3339)
	}
	return &workforcev1.Employee{
		EmployeeID:   emp.EmployeeID,
		EmployeeType: emp.EmployeeType,
		FirstName:    emp.FirstName,
		LastName:     strVal(emp.LastName),
		Email:        strVal(emp.Email),
		RoleTitle:    emp.RoleTitle,
		Department:   emp.Department,
		Status:       emp.Status,
		HiredAt:      hiredAt,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func toProtoOnboardingTask(t *workforce.OnboardingTask) *workforcev1.OnboardingTask {
	if t == nil {
		return nil
	}
	compAt := ""
	if t.CompletedAt != nil {
		compAt = t.CompletedAt.Format(time.RFC3339)
	}
	created := ""
	if !t.CreatedAt.IsZero() {
		created = t.CreatedAt.Format(time.RFC3339)
	}
	updated := ""
	if !t.UpdatedAt.IsZero() {
		updated = t.UpdatedAt.Format(time.RFC3339)
	}
	return &workforcev1.OnboardingTask{
		ChecklistID: t.ChecklistID,
		EmployeeID:  t.EmployeeID,
		TaskName:    t.TaskName,
		Category:    t.Category,
		IsCompleted: t.IsCompleted,
		CompletedAt: compAt,
		AssignedTo:  strVal(t.AssignedTo),
		CreatedAt:   created,
		UpdatedAt:   updated,
	}
}

func toProtoCareSchedule(cs *workforce.CareSchedule) *workforcev1.CareSchedule {
	if cs == nil {
		return nil
	}
	createdAt := ""
	if !cs.CreatedAt.IsZero() {
		createdAt = cs.CreatedAt.Format(time.RFC3339)
	}
	updatedAt := ""
	if !cs.UpdatedAt.IsZero() {
		updatedAt = cs.UpdatedAt.Format(time.RFC3339)
	}
	return &workforcev1.CareSchedule{
		ScheduleID:           cs.ScheduleID,
		FelineID:             cs.FelineID,
		DietaryPlan:          cs.DietaryPlan,
		FeedingTimes:         cs.FeedingTimes,
		SpecialMedicalNeeds:  strVal(cs.SpecialMedicalNeeds),
		PreferredPerchZone:   strVal(cs.PreferredPerchZone),
		EmergencyMedicalHold: cs.EmergencyMedicalHold,
		CaretakerID:          strVal(cs.CaretakerID),
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}
}

func toProtoLeaveRequest(lr *workforce.LeaveRequest) *workforcev1.LeaveRequest {
	if lr == nil {
		return nil
	}
	created := ""
	if !lr.CreatedAt.IsZero() {
		created = lr.CreatedAt.Format(time.RFC3339)
	}
	return &workforcev1.LeaveRequest{
		LeaveID:    lr.LeaveID,
		EmployeeID: lr.EmployeeID,
		LeaveType:  lr.LeaveType,
		StartDate:  lr.StartDate,
		EndDate:    lr.EndDate,
		Status:     lr.Status,
		Reason:     strVal(lr.Reason),
		CreatedAt:  created,
	}
}

func toProtoReviewCycle(rc *workforce.ReviewCycle) *workforcev1.ReviewCycle {
	if rc == nil {
		return nil
	}
	compAt := ""
	if rc.CompletedAt != nil {
		compAt = rc.CompletedAt.Format(time.RFC3339)
	}
	created := ""
	if !rc.CreatedAt.IsZero() {
		created = rc.CreatedAt.Format(time.RFC3339)
	}
	updated := ""
	if !rc.UpdatedAt.IsZero() {
		updated = rc.UpdatedAt.Format(time.RFC3339)
	}
	return &workforcev1.ReviewCycle{
		ReviewID:               rc.ReviewID,
		EmployeeID:             rc.EmployeeID,
		ReviewType:             rc.ReviewType,
		ReviewPeriod:           rc.ReviewPeriod,
		Phase:                  rc.Phase,
		ScheduledFor:           rc.ScheduledFor,
		Status:                 rc.Status,
		ReviewerID:             strVal(rc.ReviewerID),
		Score:                  floatVal(rc.Score),
		Notes:                  strVal(rc.Notes),
		SelfAppraisal:          strVal(rc.SelfAppraisal),
		SelfScore:              floatVal(rc.SelfScore),
		PeerFeedback:           strVal(rc.PeerFeedback),
		PeerScore:              floatVal(rc.PeerScore),
		PeerReviewerID:         strVal(rc.PeerReviewerID),
		ManagerAssessment:      strVal(rc.ManagerAssessment),
		ManagerScore:           floatVal(rc.ManagerScore),
		KPIRatings:             strVal(rc.KPIRatings),
		WellnessAuditScore:     floatVal(rc.WellnessAuditScore),
		HabitatRecommendations: strVal(rc.HabitatRecommendations),
		ClinicalAuditorID:      strVal(rc.ClinicalAuditorID),
		VHRRecordID:            strVal(rc.VHRRecordID),
		CompletedAt:            compAt,
		CreatedAt:              created,
		UpdatedAt:              updated,
	}
}

func toProtoIncident(inc *workforce.WorkplaceIncident) *workforcev1.WorkplaceIncident {
	if inc == nil {
		return nil
	}
	reported := ""
	if !inc.ReportedAt.IsZero() {
		reported = inc.ReportedAt.Format(time.RFC3339)
	}
	resolved := ""
	if inc.ResolvedAt != nil {
		resolved = inc.ResolvedAt.Format(time.RFC3339)
	}
	return &workforcev1.WorkplaceIncident{
		IncidentID:       inc.IncidentID,
		Title:            inc.Title,
		Category:         inc.Category,
		InvolvedFelineID: strVal(inc.InvolvedFelineID),
		InvolvedHumanID:  strVal(inc.InvolvedHumanID),
		Severity:         inc.Severity,
		Status:           inc.Status,
		Description:      inc.Description,
		ResolutionNotes:  strVal(inc.ResolutionNotes),
		ReportedAt:       reported,
		ResolvedAt:       resolved,
	}
}

func toProtoExecutiveReport(r *workforce.ExecutiveReport) *workforcev1.ExecutiveReport {
	if r == nil {
		return nil
	}
	created := ""
	if !r.CreatedAt.IsZero() {
		created = r.CreatedAt.Format(time.RFC3339)
	}
	updated := ""
	if !r.UpdatedAt.IsZero() {
		updated = r.UpdatedAt.Format(time.RFC3339)
	}
	return &workforcev1.ExecutiveReport{
		ReportID:           r.ReportID,
		Period:             r.Period,
		Title:              r.Title,
		Status:             r.Status,
		TotalRevenue:       r.TotalRevenue,
		OperatingExpenses:  r.OperatingExpenses,
		PayrollExpenses:    r.PayrollExpenses,
		NetProfit:          r.NetProfit,
		OperationalBurn:    r.OperationalBurn,
		CashReserves:       r.CashReserves,
		RunwayMonths:       r.RunwayMonths,
		TotalHeadcount:     int32(r.TotalHeadcount),
		HumanHeadcount:     int32(r.HumanHeadcount),
		FelineHeadcount:    int32(r.FelineHeadcount),
		AttendanceRate:     r.AttendanceRate,
		OvertimeHours:      r.OvertimeHours,
		TimesheetsApproved: int32(r.TimesheetsApproved),
		WebDAVPathPDF:      strVal(r.WebDAVPathPDF),
		WebDAVPathCSV:      strVal(r.WebDAVPathCSV),
		CSVContent:         strVal(r.CSVContent),
		PDFContent:         r.PDFContent,
		GeneratedBy:        r.GeneratedBy,
		CreatedAt:          created,
		UpdatedAt:          updated,
	}
}

func (s *WorkforceGRPCServer) GetEmployee(ctx context.Context, in *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	emp, err := s.svc.GetEmployee(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetEmployeeResponse{Employee: toProtoEmployee(emp)}, nil
}

func (s *WorkforceGRPCServer) ListEmployees(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	emps, err := s.svc.ListEmployees(ctx, in.EmployeeType, in.Status)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.Employee
	for _, e := range emps {
		out = append(out, toProtoEmployee(e))
	}
	return &workforcev1.ListEmployeesResponse{Employees: out}, nil
}

func (s *WorkforceGRPCServer) CreateEmployee(ctx context.Context, in *workforcev1.CreateEmployeeRequest) (*workforcev1.CreateEmployeeResponse, error) {
	emp := &workforce.Employee{
		EmployeeType: in.EmployeeType,
		FirstName:    in.FirstName,
		LastName:     strPtr(in.LastName),
		Email:        strPtr(in.Email),
		RoleTitle:    in.RoleTitle,
		Department:   in.Department,
	}
	created, err := s.svc.OnboardEmployee(ctx, emp, nil)
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateEmployeeResponse{Employee: toProtoEmployee(created)}, nil
}

func (s *WorkforceGRPCServer) OnboardEmployee(ctx context.Context, in *workforcev1.OnboardEmployeeRequest) (*workforcev1.OnboardEmployeeResponse, error) {
	if in.Employee == nil {
		return nil, status.Error(codes.InvalidArgument, "employee is required")
	}
	emp := &workforce.Employee{
		EmployeeID:   in.Employee.EmployeeID,
		EmployeeType: in.Employee.EmployeeType,
		FirstName:    in.Employee.FirstName,
		LastName:     strPtr(in.Employee.LastName),
		Email:        strPtr(in.Employee.Email),
		RoleTitle:    in.Employee.RoleTitle,
		Department:   in.Employee.Department,
		Status:       in.Employee.Status,
	}
	var cs *workforce.CareSchedule
	if in.CareSchedule != nil {
		cs = &workforce.CareSchedule{
			FelineID:             in.CareSchedule.FelineID,
			DietaryPlan:          in.CareSchedule.DietaryPlan,
			FeedingTimes:         in.CareSchedule.FeedingTimes,
			SpecialMedicalNeeds:  strPtr(in.CareSchedule.SpecialMedicalNeeds),
			PreferredPerchZone:   strPtr(in.CareSchedule.PreferredPerchZone),
			EmergencyMedicalHold: in.CareSchedule.EmergencyMedicalHold,
			CaretakerID:          strPtr(in.CareSchedule.CaretakerID),
		}
	}
	created, err := s.svc.OnboardEmployee(ctx, emp, cs)
	if err != nil {
		return nil, err
	}
	return &workforcev1.OnboardEmployeeResponse{Employee: toProtoEmployee(created)}, nil
}

func (s *WorkforceGRPCServer) UpdateEmployeeStatus(ctx context.Context, in *workforcev1.UpdateEmployeeStatusRequest) (*workforcev1.UpdateEmployeeStatusResponse, error) {
	emp, err := s.svc.UpdateEmployeeStatus(ctx, in.EmployeeID, in.Status)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateEmployeeStatusResponse{Employee: toProtoEmployee(emp)}, nil
}

func (s *WorkforceGRPCServer) ListOnboardingChecklist(ctx context.Context, in *workforcev1.ListOnboardingChecklistRequest) (*workforcev1.ListOnboardingChecklistResponse, error) {
	tasks, err := s.svc.ListOnboardingChecklist(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.OnboardingTask
	for _, t := range tasks {
		out = append(out, toProtoOnboardingTask(t))
	}
	return &workforcev1.ListOnboardingChecklistResponse{Tasks: out}, nil
}

func (s *WorkforceGRPCServer) UpdateOnboardingTask(ctx context.Context, in *workforcev1.UpdateOnboardingTaskRequest) (*workforcev1.UpdateOnboardingTaskResponse, error) {
	t, err := s.svc.UpdateOnboardingTask(ctx, in.ChecklistID, in.IsCompleted)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateOnboardingTaskResponse{Task: toProtoOnboardingTask(t)}, nil
}

func (s *WorkforceGRPCServer) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	if !s.enableFeline {
		return nil, status.Error(codes.FailedPrecondition, "feline workforce domain feature is disabled")
	}
	cs, err := s.svc.GetCareSchedule(ctx, in.FelineID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetCareScheduleResponse{CareSchedule: toProtoCareSchedule(cs)}, nil
}

func (s *WorkforceGRPCServer) UpdateCareSchedule(ctx context.Context, in *workforcev1.UpdateCareScheduleRequest) (*workforcev1.UpdateCareScheduleResponse, error) {
	if !s.enableFeline {
		return nil, status.Error(codes.FailedPrecondition, "feline workforce domain feature is disabled")
	}
	cs := &workforce.CareSchedule{
		FelineID:             in.FelineID,
		DietaryPlan:          in.DietaryPlan,
		FeedingTimes:         in.FeedingTimes,
		SpecialMedicalNeeds:  strPtr(in.SpecialMedicalNeeds),
		PreferredPerchZone:   strPtr(in.PreferredPerchZone),
		EmergencyMedicalHold: in.EmergencyMedicalHold,
		CaretakerID:          strPtr(in.CaretakerID),
	}
	updated, err := s.svc.UpdateCareSchedule(ctx, cs)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateCareScheduleResponse{CareSchedule: toProtoCareSchedule(updated)}, nil
}

func (s *WorkforceGRPCServer) SetMedicalHold(ctx context.Context, in *workforcev1.SetMedicalHoldRequest) (*workforcev1.SetMedicalHoldResponse, error) {
	if !s.enableFeline {
		return nil, status.Error(codes.FailedPrecondition, "feline workforce domain feature is disabled")
	}
	cs, err := s.svc.SetEmergencyMedicalHold(ctx, in.FelineID, in.EmergencyMedicalHold)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SetMedicalHoldResponse{CareSchedule: toProtoCareSchedule(cs)}, nil
}

func (s *WorkforceGRPCServer) SubmitLeaveRequest(ctx context.Context, in *workforcev1.SubmitLeaveRequestMessage) (*workforcev1.SubmitLeaveResponse, error) {
	lr, err := s.svc.CreateLeaveRequest(ctx, workforce_svc.CreateLeaveRequestParams{
		EmployeeID: in.EmployeeID,
		LeaveType:  in.LeaveType,
		StartDate:  in.StartDate,
		EndDate:    in.EndDate,
		Reason:     strPtr(in.Reason),
	})
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitLeaveResponse{LeaveRequest: toProtoLeaveRequest(lr)}, nil
}

func (s *WorkforceGRPCServer) GetLeaveRequest(ctx context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	lr, err := s.svc.GetLeaveRequest(ctx, in.LeaveID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetLeaveRequestResponse{LeaveRequest: toProtoLeaveRequest(lr)}, nil
}

func (s *WorkforceGRPCServer) ListLeaveRequests(ctx context.Context, in *workforcev1.ListLeaveRequestsRequest) (*workforcev1.ListLeaveRequestsResponse, error) {
	reqs, err := s.svc.ListLeaveRequests(ctx, in.EmployeeID, in.Status)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.LeaveRequest
	for _, r := range reqs {
		out = append(out, toProtoLeaveRequest(r))
	}
	return &workforcev1.ListLeaveRequestsResponse{LeaveRequests: out}, nil
}

func (s *WorkforceGRPCServer) UpdateLeaveStatus(ctx context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	updated, err := s.svc.UpdateLeaveRequestStatus(ctx, in.LeaveID, in.Status)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateLeaveStatusResponse{LeaveRequest: toProtoLeaveRequest(updated)}, nil
}

func (s *WorkforceGRPCServer) CreateReviewCycle(ctx context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	params := workforce_svc.CreateReviewCycleParams{
		EmployeeID:             in.EmployeeID,
		ReviewType:             in.ReviewType,
		ReviewPeriod:           in.ReviewPeriod,
		Phase:                  in.Phase,
		ScheduledFor:           in.ScheduledFor,
		ReviewerID:             strPtr(in.ReviewerID),
		Notes:                  strPtr(in.Notes),
		SelfAppraisal:          strPtr(in.SelfAppraisal),
		PeerFeedback:           strPtr(in.PeerFeedback),
		PeerReviewerID:         strPtr(in.PeerReviewerID),
		ManagerAssessment:      strPtr(in.ManagerAssessment),
		KPIRatings:             strPtr(in.KPIRatings),
		HabitatRecommendations: strPtr(in.HabitatRecommendations),
		ClinicalAuditorID:      strPtr(in.ClinicalAuditorID),
		VHRRecordID:            strPtr(in.VHRRecordID),
	}
	if in.HasSelfScore {
		v := in.SelfScore
		params.SelfScore = &v
	}
	if in.HasPeerScore {
		v := in.PeerScore
		params.PeerScore = &v
	}
	if in.HasManagerScore {
		v := in.ManagerScore
		params.ManagerScore = &v
	}
	if in.HasWellnessScore {
		v := in.WellnessAuditScore
		params.WellnessAuditScore = &v
	}
	rc, err := s.svc.CreateReviewCycle(ctx, params)
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateReviewCycleResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) GetReviewCycle(ctx context.Context, in *workforcev1.GetReviewCycleRequest) (*workforcev1.GetReviewCycleResponse, error) {
	rc, err := s.svc.GetReviewCycle(ctx, in.ReviewID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetReviewCycleResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) ListReviewCycles(ctx context.Context, in *workforcev1.ListReviewCyclesRequest) (*workforcev1.ListReviewCyclesResponse, error) {
	rcs, err := s.svc.ListReviewCycles(ctx, in.EmployeeID, in.Status, in.ReviewType)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.ReviewCycle
	for _, r := range rcs {
		out = append(out, toProtoReviewCycle(r))
	}
	return &workforcev1.ListReviewCyclesResponse{ReviewCycles: out}, nil
}

func (s *WorkforceGRPCServer) UpdateReviewCycle(ctx context.Context, in *workforcev1.UpdateReviewCycleRequest) (*workforcev1.UpdateReviewCycleResponse, error) {
	params := workforce_svc.UpdateReviewCycleParams{
		ID:                     in.ReviewID,
		ReviewPeriod:           strPtr(in.ReviewPeriod),
		Phase:                  strPtr(in.Phase),
		Status:                 strPtr(in.Status),
		ReviewerID:             strPtr(in.ReviewerID),
		Notes:                  strPtr(in.Notes),
		SelfAppraisal:          strPtr(in.SelfAppraisal),
		PeerFeedback:           strPtr(in.PeerFeedback),
		PeerReviewerID:         strPtr(in.PeerReviewerID),
		ManagerAssessment:      strPtr(in.ManagerAssessment),
		KPIRatings:             strPtr(in.KPIRatings),
		HabitatRecommendations: strPtr(in.HabitatRecommendations),
		ClinicalAuditorID:      strPtr(in.ClinicalAuditorID),
		VHRRecordID:            strPtr(in.VHRRecordID),
	}
	if in.HasScore {
		v := in.Score
		params.Score = &v
	}
	if in.HasSelfScore {
		v := in.SelfScore
		params.SelfScore = &v
	}
	if in.HasPeerScore {
		v := in.PeerScore
		params.PeerScore = &v
	}
	if in.HasManagerScore {
		v := in.ManagerScore
		params.ManagerScore = &v
	}
	if in.HasWellnessScore {
		v := in.WellnessAuditScore
		params.WellnessAuditScore = &v
	}
	rc, err := s.svc.UpdateReviewCycle(ctx, params)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateReviewCycleResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitSelfAppraisal(ctx context.Context, in *workforcev1.SubmitSelfAppraisalRequest) (*workforcev1.SubmitSelfAppraisalResponse, error) {
	rc, err := s.svc.SubmitSelfAppraisal(ctx, in.ReviewID, in.Appraisal, in.Score)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitSelfAppraisalResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitPeerFeedback(ctx context.Context, in *workforcev1.SubmitPeerFeedbackRequest) (*workforcev1.SubmitPeerFeedbackResponse, error) {
	rc, err := s.svc.SubmitPeerFeedback(ctx, in.ReviewID, in.PeerReviewerID, in.Feedback, in.Score)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitPeerFeedbackResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitManagerAssessment(ctx context.Context, in *workforcev1.SubmitManagerAssessmentRequest) (*workforcev1.SubmitManagerAssessmentResponse, error) {
	rc, err := s.svc.SubmitManagerAssessment(ctx, in.ReviewID, in.ManagerID, in.Assessment, in.Score, in.KPIRatings)
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitManagerAssessmentResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) SubmitVeterinaryAudit(ctx context.Context, in *workforcev1.SubmitVeterinaryAuditRequest) (*workforcev1.SubmitVeterinaryAuditResponse, error) {
	rc, err := s.svc.SubmitVeterinaryAudit(ctx, in.ReviewID, in.AuditorID, in.WellnessScore, in.Recommendations, strPtr(in.VHRRecordID))
	if err != nil {
		return nil, err
	}
	return &workforcev1.SubmitVeterinaryAuditResponse{ReviewCycle: toProtoReviewCycle(rc)}, nil
}

func (s *WorkforceGRPCServer) CreateVHRRecord(ctx context.Context, in *workforcev1.CreateVHRRecordRequest) (*workforcev1.CreateVHRRecordResponse, error) {
	if !s.enableFeline {
		return nil, status.Error(codes.FailedPrecondition, "feline workforce domain feature is disabled")
	}
	rec := &workforce.VHRRecord{
		FelineID:          in.FelineID,
		VeterinarianID:    strPtr(in.VeterinarianID),
		VisitDate:         in.VisitDate,
		WeightKG:          in.WeightKG,
		DentalScore:       int(in.DentalScore),
		VaccinationStatus: in.VaccinationStatus,
		Prescriptions:     strPtr(in.Prescriptions),
		ClinicalNotes:     in.ClinicalNotes,
	}
	created, err := s.svc.CreateVHRRecord(ctx, rec)
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateVHRRecordResponse{Record: &workforcev1.VHRRecord{
		RecordID:          created.RecordID,
		FelineID:          created.FelineID,
		VeterinarianID:    strVal(created.VeterinarianID),
		VisitDate:         created.VisitDate,
		WeightKG:          created.WeightKG,
		DentalScore:       int32(created.DentalScore),
		VaccinationStatus: created.VaccinationStatus,
		Prescriptions:     strVal(created.Prescriptions),
		ClinicalNotes:     created.ClinicalNotes,
		CreatedAt:         created.CreatedAt.Format(time.RFC3339),
	}}, nil
}

func (s *WorkforceGRPCServer) ListVHRRecords(ctx context.Context, in *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error) {
	if !s.enableFeline {
		return nil, status.Error(codes.FailedPrecondition, "feline workforce domain feature is disabled")
	}
	records, err := s.svc.ListVHRRecords(ctx, in.FelineID)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.VHRRecord
	for _, r := range records {
		out = append(out, &workforcev1.VHRRecord{
			RecordID:          r.RecordID,
			FelineID:          r.FelineID,
			VeterinarianID:    strVal(r.VeterinarianID),
			VisitDate:         r.VisitDate,
			WeightKG:          r.WeightKG,
			DentalScore:       int32(r.DentalScore),
			VaccinationStatus: r.VaccinationStatus,
			Prescriptions:     strVal(r.Prescriptions),
			ClinicalNotes:     r.ClinicalNotes,
			CreatedAt:         r.CreatedAt.Format(time.RFC3339),
		})
	}
	return &workforcev1.ListVHRRecordsResponse{Records: out}, nil
}

func (s *WorkforceGRPCServer) CreateIncident(ctx context.Context, in *workforcev1.CreateIncidentRequest) (*workforcev1.CreateIncidentResponse, error) {
	inc := &workforce.WorkplaceIncident{
		Title:            in.Title,
		Category:         in.Category,
		InvolvedFelineID: strPtr(in.InvolvedFelineID),
		InvolvedHumanID:  strPtr(in.InvolvedHumanID),
		Severity:         in.Severity,
		Description:      in.Description,
	}
	created, err := s.svc.CreateWorkplaceIncident(ctx, inc)
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateIncidentResponse{Incident: toProtoIncident(created)}, nil
}

func (s *WorkforceGRPCServer) ListIncidents(ctx context.Context, in *workforcev1.ListIncidentsRequest) (*workforcev1.ListIncidentsResponse, error) {
	incidents, err := s.svc.ListWorkplaceIncidents(ctx, in.Status)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.WorkplaceIncident
	for _, inc := range incidents {
		if in.Severity != "" && inc.Severity != in.Severity {
			continue
		}
		out = append(out, toProtoIncident(inc))
	}
	return &workforcev1.ListIncidentsResponse{Incidents: out}, nil
}

func (s *WorkforceGRPCServer) UpdateIncidentStatus(ctx context.Context, in *workforcev1.UpdateIncidentStatusRequest) (*workforcev1.UpdateIncidentStatusResponse, error) {
	updated, err := s.svc.UpdateWorkplaceIncidentStatus(ctx, in.IncidentID, in.Status, in.ResolutionNotes)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateIncidentStatusResponse{Incident: toProtoIncident(updated)}, nil
}

func (s *WorkforceGRPCServer) CreateExecutiveReport(ctx context.Context, in *workforcev1.CreateExecutiveReportRequest) (*workforcev1.CreateExecutiveReportResponse, error) {
	if in.Report == nil {
		return nil, status.Error(codes.InvalidArgument, "report is required")
	}
	if s.repo == nil {
		rep, err := s.svc.GenerateMonthlyReport(ctx, in.Report.Period, in.Report.GeneratedBy)
		if err != nil {
			return nil, err
		}
		return &workforcev1.CreateExecutiveReportResponse{Report: toProtoExecutiveReport(rep)}, nil
	}
	created, err := s.repo.CreateExecutiveReport(ctx, &workforce.ExecutiveReport{
		ReportID:           in.Report.ReportID,
		Period:             in.Report.Period,
		Title:              in.Report.Title,
		Status:             in.Report.Status,
		TotalRevenue:       in.Report.TotalRevenue,
		OperatingExpenses:  in.Report.OperatingExpenses,
		PayrollExpenses:    in.Report.PayrollExpenses,
		NetProfit:          in.Report.NetProfit,
		OperationalBurn:    in.Report.OperationalBurn,
		CashReserves:       in.Report.CashReserves,
		RunwayMonths:       in.Report.RunwayMonths,
		TotalHeadcount:     int(in.Report.TotalHeadcount),
		HumanHeadcount:     int(in.Report.HumanHeadcount),
		FelineHeadcount:    int(in.Report.FelineHeadcount),
		AttendanceRate:     in.Report.AttendanceRate,
		OvertimeHours:      in.Report.OvertimeHours,
		TimesheetsApproved: int(in.Report.TimesheetsApproved),
		WebDAVPathPDF:      strPtr(in.Report.WebDAVPathPDF),
		WebDAVPathCSV:      strPtr(in.Report.WebDAVPathCSV),
		CSVContent:         strPtr(in.Report.CSVContent),
		PDFContent:         in.Report.PDFContent,
		GeneratedBy:        in.Report.GeneratedBy,
	})
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateExecutiveReportResponse{Report: toProtoExecutiveReport(created)}, nil
}

func (s *WorkforceGRPCServer) GetExecutiveReport(ctx context.Context, in *workforcev1.GetExecutiveReportRequest) (*workforcev1.GetExecutiveReportResponse, error) {
	rep, err := s.svc.GetExecutiveReport(ctx, in.ReportID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetExecutiveReportResponse{Report: toProtoExecutiveReport(rep)}, nil
}

func (s *WorkforceGRPCServer) GetExecutiveReportByPeriod(ctx context.Context, in *workforcev1.GetExecutiveReportByPeriodRequest) (*workforcev1.GetExecutiveReportByPeriodResponse, error) {
	rep, err := s.svc.GetExecutiveReportByPeriod(ctx, in.Period)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetExecutiveReportByPeriodResponse{Report: toProtoExecutiveReport(rep)}, nil
}

func (s *WorkforceGRPCServer) ListExecutiveReports(ctx context.Context, in *workforcev1.ListExecutiveReportsRequest) (*workforcev1.ListExecutiveReportsResponse, error) {
	limit := int(in.Limit)
	if limit <= 0 {
		limit = 20
	}
	reps, err := s.svc.ListExecutiveReports(ctx, limit, int(in.Offset))
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.ExecutiveReport
	for _, r := range reps {
		out = append(out, toProtoExecutiveReport(r))
	}
	return &workforcev1.ListExecutiveReportsResponse{Reports: out}, nil
}

func (s *WorkforceGRPCServer) UpdateExecutiveReportStatus(ctx context.Context, in *workforcev1.UpdateExecutiveReportStatusRequest) (*workforcev1.UpdateExecutiveReportStatusResponse, error) {
	if s.repo == nil {
		rep, err := s.svc.GetExecutiveReport(ctx, in.ReportID)
		if err != nil {
			return nil, err
		}
		return &workforcev1.UpdateExecutiveReportStatusResponse{Report: toProtoExecutiveReport(rep)}, nil
	}
	updated, err := s.repo.UpdateExecutiveReportStatus(ctx, in.ReportID, in.Status, strPtr(in.WebDAVPathPDF), strPtr(in.WebDAVPathCSV))
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateExecutiveReportStatusResponse{Report: toProtoExecutiveReport(updated)}, nil
}
