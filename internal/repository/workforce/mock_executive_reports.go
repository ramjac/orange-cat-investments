package workforce

import (
	"context"
	"fmt"
	"sort"
	"time"
)

func (r *mockRepository) seedExecutiveReports(now time.Time) {
	pdfPath8 := "/remote.php/dav/files/admin/Executive/Reports/2026-08/executive_summary_report_2026-08.pdf"
	csvPath8 := "/remote.php/dav/files/admin/Executive/Reports/2026-08/executive_summary_report_2026-08.csv"
	r.executiveReports["rep-2026-08"] = &ExecutiveReport{
		ReportID:           "rep-2026-08",
		Period:             "2026-08",
		Title:              "OCI Executive Management Report - August 2026",
		Status:             "archived",
		TotalRevenue:       1180000.00,
		OperatingExpenses:  310000.00,
		PayrollExpenses:    395000.00,
		NetProfit:          475000.00,
		OperationalBurn:    705000.00,
		CashReserves:       9200000.00,
		RunwayMonths:       13.05,
		TotalHeadcount:     11,
		HumanHeadcount:     9,
		FelineHeadcount:    2,
		AttendanceRate:     98.80,
		OvertimeHours:      38.00,
		TimesheetsApproved: 16,
		WebDAVPathPDF:      &pdfPath8,
		WebDAVPathCSV:      &csvPath8,
		GeneratedBy:        "system-reporting-worker",
		CreatedAt:          now.Add(-60 * 24 * time.Hour),
		UpdatedAt:          now.Add(-60 * 24 * time.Hour),
	}

	pdfPath9 := "/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_report_2026-09.pdf"
	csvPath9 := "/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_report_2026-09.csv"
	r.executiveReports["rep-2026-09"] = &ExecutiveReport{
		ReportID:           "rep-2026-09",
		Period:             "2026-09",
		Title:              "OCI Executive Management Report - September 2026",
		Status:             "archived",
		TotalRevenue:       1240000.00,
		OperatingExpenses:  335000.00,
		PayrollExpenses:    410000.00,
		NetProfit:          495000.00,
		OperationalBurn:    745000.00,
		CashReserves:       9650000.00,
		RunwayMonths:       12.95,
		TotalHeadcount:     12,
		HumanHeadcount:     10,
		FelineHeadcount:    2,
		AttendanceRate:     99.10,
		OvertimeHours:      41.50,
		TimesheetsApproved: 18,
		WebDAVPathPDF:      &pdfPath9,
		WebDAVPathCSV:      &csvPath9,
		GeneratedBy:        "system-reporting-worker",
		CreatedAt:          now.Add(-30 * 24 * time.Hour),
		UpdatedAt:          now.Add(-30 * 24 * time.Hour),
	}
}

func (r *mockRepository) CreateExecutiveReport(ctx context.Context, report *ExecutiveReport) (*ExecutiveReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if report.ReportID == "" {
		report.ReportID = fmt.Sprintf("rep-%s", report.Period)
	}
	now := time.Now().UTC()
	if report.CreatedAt.IsZero() {
		report.CreatedAt = now
	}
	if report.UpdatedAt.IsZero() {
		report.UpdatedAt = now
	}
	if report.Status == "" {
		report.Status = "archived"
	}
	if report.GeneratedBy == "" {
		report.GeneratedBy = "system-reporting-worker"
	}

	r.executiveReports[report.ReportID] = report
	return report, nil
}

func (r *mockRepository) GetExecutiveReportByID(ctx context.Context, id string) (*ExecutiveReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rep, ok := r.executiveReports[id]
	if !ok {
		return nil, fmt.Errorf("executive report with id %s not found", id)
	}
	return rep, nil
}

func (r *mockRepository) GetExecutiveReportByPeriod(ctx context.Context, period string) (*ExecutiveReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched *ExecutiveReport
	for _, rep := range r.executiveReports {
		if rep.Period == period {
			if matched == nil || rep.CreatedAt.After(matched.CreatedAt) {
				matched = rep
			}
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("executive report for period %s not found", period)
	}
	return matched, nil
}

func (r *mockRepository) ListExecutiveReports(ctx context.Context, limit, offset int) ([]*ExecutiveReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*ExecutiveReport
	for _, rep := range r.executiveReports {
		list = append(list, rep)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Period != list[j].Period {
			return list[i].Period > list[j].Period
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if offset >= len(list) {
		return []*ExecutiveReport{}, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (r *mockRepository) UpdateExecutiveReportStatus(ctx context.Context, id, status string, webdavPDF, webdavCSV *string) (*ExecutiveReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rep, ok := r.executiveReports[id]
	if !ok {
		return nil, fmt.Errorf("executive report with id %s not found", id)
	}
	rep.Status = status
	if webdavPDF != nil {
		rep.WebDAVPathPDF = webdavPDF
	}
	if webdavCSV != nil {
		rep.WebDAVPathCSV = webdavCSV
	}
	rep.UpdatedAt = time.Now().UTC()
	return rep, nil
}
