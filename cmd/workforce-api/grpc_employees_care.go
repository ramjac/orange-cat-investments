package main

import (
	"context"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
