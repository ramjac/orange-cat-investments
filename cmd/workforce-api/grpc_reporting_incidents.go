package main

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
