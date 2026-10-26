package main

import (
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
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
