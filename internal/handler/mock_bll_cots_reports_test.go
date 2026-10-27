package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

func (m *mockInternalBLLClient) ListTimesheets(ctx context.Context, in *internalbllv1.ListTimesheetsRequest) (*internalbllv1.ListTimesheetsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*frappe.Timesheet
	for _, ts := range m.timesheets {
		if in.EmployeeID != "" && ts.EmployeeID != in.EmployeeID {
			continue
		}
		list = append(list, ts)
	}
	return &internalbllv1.ListTimesheetsResponse{Timesheets: list}, nil
}

func (m *mockInternalBLLClient) SubmitTimesheet(ctx context.Context, in *internalbllv1.SubmitTimesheetRequest) (*internalbllv1.SubmitTimesheetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ts := in.Timesheet
	validCodes := map[string]bool{
		"habitat_maintenance": true,
		"feline_feeding":      true,
		"veterinary_assist":   true,
		"admin":               true,
		"invest_ops":          true,
	}
	if !validCodes[ts.TaskCode] {
		return nil, fmt.Errorf("invalid task_code %q", ts.TaskCode)
	}
	if ts.TimesheetID == "" {
		ts.TimesheetID = fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	ts.HoursWorked = 8.0
	ts.Status = "submitted"
	if ts.TaskCode == "habitat_maintenance" {
		ts.CostCenter = "Facilities Operations"
	}
	m.timesheets[ts.TimesheetID] = ts
	return &internalbllv1.SubmitTimesheetResponse{Timesheet: ts}, nil
}

func (m *mockInternalBLLClient) GetTimesheet(ctx context.Context, in *internalbllv1.GetTimesheetRequest) (*internalbllv1.GetTimesheetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ts, ok := m.timesheets[in.TimesheetID]
	if !ok {
		return nil, fmt.Errorf("timesheet %s not found", in.TimesheetID)
	}
	return &internalbllv1.GetTimesheetResponse{Timesheet: ts}, nil
}

func (m *mockInternalBLLClient) ApproveTimesheet(ctx context.Context, in *internalbllv1.ApproveTimesheetRequest) (*internalbllv1.ApproveTimesheetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ts, ok := m.timesheets[in.TimesheetID]
	if !ok {
		ts = &frappe.Timesheet{TimesheetID: in.TimesheetID}
		m.timesheets[in.TimesheetID] = ts
	}
	ts.Status = "approved"
	ts.ApprovedBy = &in.ApproverID
	return &internalbllv1.ApproveTimesheetResponse{Timesheet: ts}, nil
}

func (m *mockInternalBLLClient) RejectTimesheet(ctx context.Context, in *internalbllv1.RejectTimesheetRequest) (*internalbllv1.RejectTimesheetResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ts, ok := m.timesheets[in.TimesheetID]
	if !ok {
		ts = &frappe.Timesheet{TimesheetID: in.TimesheetID}
		m.timesheets[in.TimesheetID] = ts
	}
	ts.Status = "rejected"
	return &internalbllv1.RejectTimesheetResponse{Timesheet: ts}, nil
}

func (m *mockInternalBLLClient) ListSalarySlips(ctx context.Context, in *internalbllv1.ListSalarySlipsRequest) (*internalbllv1.ListSalarySlipsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []frappe.SalarySlip
	for _, s := range m.salarySlips {
		if in.EmployeeID != "" && s.EmployeeID != in.EmployeeID {
			continue
		}
		list = append(list, *s)
	}
	return &internalbllv1.ListSalarySlipsResponse{SalarySlips: list}, nil
}

func (m *mockInternalBLLClient) GetSalarySlip(ctx context.Context, in *internalbllv1.GetSalarySlipRequest) (*internalbllv1.GetSalarySlipResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.salarySlips[in.SlipID]
	if !ok {
		return nil, fmt.Errorf("salary slip %s not found", in.SlipID)
	}
	return &internalbllv1.GetSalarySlipResponse{SalarySlip: s}, nil
}

func (m *mockInternalBLLClient) ProcessPayrollRun(ctx context.Context, in *internalbllv1.ProcessPayrollRunRequest) (*internalbllv1.ProcessPayrollRunResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pe := in.PayrollEntry
	pe.PayrollEntryID = fmt.Sprintf("pe-%d", time.Now().UnixNano())
	pe.SalarySlipsCreated = 2
	return &internalbllv1.ProcessPayrollRunResponse{PayrollEntry: pe}, nil
}

func (m *mockInternalBLLClient) ListGroupFolders(ctx context.Context, in *internalbllv1.ListGroupFoldersRequest) (*internalbllv1.ListGroupFoldersResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &internalbllv1.ListGroupFoldersResponse{GroupFolders: m.folders}, nil
}

func (m *mockInternalBLLClient) CreateGroupFolder(ctx context.Context, in *internalbllv1.CreateGroupFolderRequest) (*internalbllv1.CreateGroupFolderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := nextcloud.GroupFolder{
		ID:         int64(len(m.folders) + 1),
		MountPoint: in.MountPoint,
		Groups:     in.Groups,
		Quota:      in.Quota,
	}
	m.folders = append(m.folders, f)
	return &internalbllv1.CreateGroupFolderResponse{GroupFolder: &f}, nil
}

func (m *mockInternalBLLClient) ListDocuments(ctx context.Context, in *internalbllv1.ListDocumentsRequest) (*internalbllv1.ListDocumentsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []nextcloud.Document
	for _, doc := range m.documents {
		list = append(list, *doc)
	}
	return &internalbllv1.ListDocumentsResponse{Documents: list}, nil
}

func (m *mockInternalBLLClient) GetDocument(ctx context.Context, in *internalbllv1.GetDocumentRequest) (*internalbllv1.GetDocumentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := m.documents[in.DocumentID]
	if !ok {
		return nil, fmt.Errorf("document %s not found", in.DocumentID)
	}
	return &internalbllv1.GetDocumentResponse{Document: doc}, nil
}

func (m *mockInternalBLLClient) PublishDocument(ctx context.Context, in *internalbllv1.PublishDocumentRequest) (*internalbllv1.PublishDocumentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc := in.Document
	doc.DocumentID = fmt.Sprintf("doc-%d", len(m.documents)+1)
	doc.WebDAVPath = "/remote.php/dav/files/admin/" + doc.Title
	m.documents[doc.DocumentID] = &doc
	return &internalbllv1.PublishDocumentResponse{Document: &doc}, nil
}

func (m *mockInternalBLLClient) ListCalendarEvents(ctx context.Context, in *internalbllv1.ListCalendarEventsRequest) (*internalbllv1.ListCalendarEventsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &internalbllv1.ListCalendarEventsResponse{Events: m.events}, nil
}

func (m *mockInternalBLLClient) CreateCalendarEvent(ctx context.Context, in *internalbllv1.CreateCalendarEventRequest) (*internalbllv1.CreateCalendarEventResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ev := in.Event
	ev.EventID = fmt.Sprintf("ev-%d", len(m.events)+1)
	m.events = append(m.events, ev)
	return &internalbllv1.CreateCalendarEventResponse{Event: &ev}, nil
}

func (m *mockInternalBLLClient) GetBIMetrics(ctx context.Context, in *internalbllv1.GetBIMetricsRequest) (*internalbllv1.GetBIMetricsResponse, error) {
	return &internalbllv1.GetBIMetricsResponse{
		Period: in.Period,
		FinancialMetrics: &frappe.FinancialMetrics{
			Period:            in.Period,
			Currency:          "USD",
			TotalRevenue:      1500000.0,
			OperatingExpenses: 500000.0,
			PayrollExpenses:   200000.0,
		},
		WorkforceMetrics: &reporting.WorkforceMetrics{
			Period:          in.Period,
			TotalHeadcount:  42,
			HumanHeadcount:  22,
			FelineHeadcount: 20,
			AttendanceRate:  98.5,
		},
	}, nil
}

func (m *mockInternalBLLClient) ListExecutiveReports(ctx context.Context, in *workforcev1.ListExecutiveReportsRequest) (*workforcev1.ListExecutiveReportsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*workforcev1.ExecutiveReport
	for _, r := range m.reports {
		list = append(list, r)
	}
	return &workforcev1.ListExecutiveReportsResponse{Reports: list}, nil
}

func (m *mockInternalBLLClient) GenerateMonthlyReport(ctx context.Context, in *internalbllv1.GenerateMonthlyReportRequest) (*internalbllv1.GenerateMonthlyReportResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rep := &workforcev1.ExecutiveReport{
		ReportID:    fmt.Sprintf("rep-%s", in.Period),
		Period:      in.Period,
		GeneratedBy: in.GeneratedBy,
		Status:      "archived",
		CSVContent:  "Total Revenue,1600000\nMRR,120000",
		PDFContent:  []byte("%PDF-1.4 generated report"),
	}
	m.reports[rep.ReportID] = rep
	return &internalbllv1.GenerateMonthlyReportResponse{
		Report: &internalbllv1.ExecutiveReportSummary{
			ReportID: rep.ReportID,
			Period:   rep.Period,
			Title:    fmt.Sprintf("Executive Summary Report %s", rep.Period),
			Status:   rep.Status,
		},
	}, nil
}

func (m *mockInternalBLLClient) GetExecutiveReport(ctx context.Context, in *workforcev1.GetExecutiveReportRequest) (*workforcev1.GetExecutiveReportResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rep, ok := m.reports[in.ReportID]
	if !ok {
		return nil, fmt.Errorf("executive report %s not found", in.ReportID)
	}
	return &workforcev1.GetExecutiveReportResponse{Report: rep}, nil
}

func (m *mockInternalBLLClient) DownloadExecutiveReport(ctx context.Context, in *internalbllv1.DownloadExecutiveReportRequest) (*internalbllv1.DownloadExecutiveReportResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rep, ok := m.reports[in.ReportID]
	if !ok {
		return nil, fmt.Errorf("executive report %s not found", in.ReportID)
	}
	if in.Format == "csv" {
		return &internalbllv1.DownloadExecutiveReportResponse{
			ContentType: "text/csv; charset=utf-8",
			Filename:    fmt.Sprintf("executive_summary_report_%s.csv", rep.Period),
			Content:     []byte(rep.CSVContent),
		}, nil
	}
	return &internalbllv1.DownloadExecutiveReportResponse{
		ContentType: "application/pdf",
		Filename:    fmt.Sprintf("executive_summary_report_%s.pdf", rep.Period),
		Content:     rep.PDFContent,
	}, nil
}
