package main

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

type WorkforceGRPCServer struct {
	svc workforce_svc.Service
}

func NewWorkforceGRPCServer(svc workforce_svc.Service) *WorkforceGRPCServer {
	return &WorkforceGRPCServer{svc: svc}
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

func (s *WorkforceGRPCServer) GetCareSchedule(ctx context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	cs, err := s.svc.GetCareSchedule(ctx, in.FelineID)
	if err != nil {
		return nil, err
	}
	return &workforcev1.GetCareScheduleResponse{CareSchedule: toProtoCareSchedule(cs)}, nil
}

func (s *WorkforceGRPCServer) UpdateCareSchedule(ctx context.Context, in *workforcev1.UpdateCareScheduleRequest) (*workforcev1.UpdateCareScheduleResponse, error) {
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
	return &workforcev1.SubmitLeaveResponse{LeaveRequest: &workforcev1.LeaveRequest{
		LeaveID:    lr.LeaveID,
		EmployeeID: lr.EmployeeID,
		LeaveType:  lr.LeaveType,
		StartDate:  lr.StartDate,
		EndDate:    lr.EndDate,
		Status:     lr.Status,
		Reason:     strVal(lr.Reason),
	}}, nil
}

func (s *WorkforceGRPCServer) ListLeaveRequests(ctx context.Context, in *workforcev1.ListLeaveRequestsRequest) (*workforcev1.ListLeaveRequestsResponse, error) {
	reqs, err := s.svc.ListLeaveRequests(ctx, in.EmployeeID, in.Status)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.LeaveRequest
	for _, r := range reqs {
		out = append(out, &workforcev1.LeaveRequest{
			LeaveID:    r.LeaveID,
			EmployeeID: r.EmployeeID,
			LeaveType:  r.LeaveType,
			StartDate:  r.StartDate,
			EndDate:    r.EndDate,
			Status:     r.Status,
			Reason:     strVal(r.Reason),
		})
	}
	return &workforcev1.ListLeaveRequestsResponse{LeaveRequests: out}, nil
}

func (s *WorkforceGRPCServer) UpdateLeaveStatus(ctx context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	updated, err := s.svc.UpdateLeaveRequestStatus(ctx, in.LeaveID, in.Status)
	if err != nil {
		return nil, err
	}
	return &workforcev1.UpdateLeaveStatusResponse{LeaveRequest: &workforcev1.LeaveRequest{
		LeaveID:    updated.LeaveID,
		EmployeeID: updated.EmployeeID,
		LeaveType:  updated.LeaveType,
		Status:     updated.Status,
	}}, nil
}

func (s *WorkforceGRPCServer) CreateReviewCycle(ctx context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	rc, err := s.svc.CreateReviewCycle(ctx, workforce_svc.CreateReviewCycleParams{
		EmployeeID:   in.EmployeeID,
		ReviewType:   in.ReviewType,
		ReviewPeriod: in.ReviewPeriod,
		ScheduledFor: in.ScheduledFor,
	})
	if err != nil {
		return nil, err
	}
	return &workforcev1.CreateReviewCycleResponse{ReviewCycle: &workforcev1.ReviewCycle{
		ReviewID:     rc.ReviewID,
		EmployeeID:   rc.EmployeeID,
		ReviewType:   rc.ReviewType,
		ReviewPeriod: rc.ReviewPeriod,
		Status:       rc.Status,
	}}, nil
}

func (s *WorkforceGRPCServer) ListReviewCycles(ctx context.Context, in *workforcev1.ListReviewCyclesRequest) (*workforcev1.ListReviewCyclesResponse, error) {
	rcs, err := s.svc.ListReviewCycles(ctx, in.EmployeeID, "", "")
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.ReviewCycle
	for _, r := range rcs {
		out = append(out, &workforcev1.ReviewCycle{
			ReviewID:     r.ReviewID,
			EmployeeID:   r.EmployeeID,
			ReviewType:   r.ReviewType,
			ReviewPeriod: r.ReviewPeriod,
			Phase:        r.Phase,
			Status:       r.Status,
		})
	}
	return &workforcev1.ListReviewCyclesResponse{ReviewCycles: out}, nil
}

func (s *WorkforceGRPCServer) CreateVHRRecord(ctx context.Context, in *workforcev1.CreateVHRRecordRequest) (*workforcev1.CreateVHRRecordResponse, error) {
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
	}}, nil
}

func (s *WorkforceGRPCServer) ListVHRRecords(ctx context.Context, in *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error) {
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
	return &workforcev1.CreateIncidentResponse{Incident: &workforcev1.WorkplaceIncident{
		IncidentID:  created.IncidentID,
		Title:       created.Title,
		Category:    created.Category,
		Severity:    created.Severity,
		Description: created.Description,
		Status:      created.Status,
	}}, nil
}

func (s *WorkforceGRPCServer) ListIncidents(ctx context.Context, in *workforcev1.ListIncidentsRequest) (*workforcev1.ListIncidentsResponse, error) {
	incidents, err := s.svc.ListWorkplaceIncidents(ctx, in.Status)
	if err != nil {
		return nil, err
	}
	var out []*workforcev1.WorkplaceIncident
	for _, inc := range incidents {
		out = append(out, &workforcev1.WorkplaceIncident{
			IncidentID:  inc.IncidentID,
			Title:       inc.Title,
			Category:    inc.Category,
			Severity:    inc.Severity,
			Description: inc.Description,
			Status:      inc.Status,
		})
	}
	return &workforcev1.ListIncidentsResponse{Incidents: out}, nil
}
