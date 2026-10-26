package workforce

import (
	"context"
	"fmt"
)

func (r *pgxRepository) CreateExecutiveReport(ctx context.Context, report *ExecutiveReport) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO workforce.executive_reports (
	              report_id, period, title, status,
	              total_revenue, operating_expenses, payroll_expenses, net_profit,
	              operational_burn, cash_reserves, runway_months,
	              total_headcount, human_headcount, feline_headcount,
	              attendance_rate, overtime_hours, timesheets_approved,
	              webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by
	          ) VALUES (
	              $1, $2, $3, COALESCE($4, 'archived'),
	              $5, $6, $7, $8,
	              $9, $10, $11,
	              $12, $13, $14,
	              $15, $16, $17,
	              $18, $19, $20, $21, COALESCE($22, 'system-reporting-worker')
	          ) RETURNING report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at`

	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query,
		report.ReportID, report.Period, report.Title, report.Status,
		report.TotalRevenue, report.OperatingExpenses, report.PayrollExpenses, report.NetProfit,
		report.OperationalBurn, report.CashReserves, report.RunwayMonths,
		report.TotalHeadcount, report.HumanHeadcount, report.FelineHeadcount,
		report.AttendanceRate, report.OvertimeHours, report.TimesheetsApproved,
		report.WebDAVPathPDF, report.WebDAVPathCSV, report.CSVContent, report.PDFContent, report.GeneratedBy,
	).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) GetExecutiveReportByID(ctx context.Context, id string) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at
	          FROM workforce.executive_reports WHERE report_id = $1`
	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) GetExecutiveReportByPeriod(ctx context.Context, period string) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at
	          FROM workforce.executive_reports WHERE period = $1 ORDER BY created_at DESC LIMIT 1`
	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query, period).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *pgxRepository) ListExecutiveReports(ctx context.Context, limit, offset int) ([]*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at
	          FROM workforce.executive_reports
	          ORDER BY period DESC, created_at DESC
	          LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ExecutiveReport
	for rows.Next() {
		var res ExecutiveReport
		err := rows.Scan(
			&res.ReportID, &res.Period, &res.Title, &res.Status,
			&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
			&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
			&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
			&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
			&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
			&res.CreatedAt, &res.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, &res)
	}
	if list == nil {
		list = []*ExecutiveReport{}
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpdateExecutiveReportStatus(ctx context.Context, id, status string, webdavPDF, webdavCSV *string) (*ExecutiveReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE workforce.executive_reports
	          SET status = $2,
	              webdav_path_pdf = COALESCE($3, webdav_path_pdf),
	              webdav_path_csv = COALESCE($4, webdav_path_csv),
	              updated_at = CURRENT_TIMESTAMP
	          WHERE report_id = $1
	          RETURNING report_id, period, title, status, total_revenue, operating_expenses, payroll_expenses, net_profit, operational_burn, cash_reserves, runway_months, total_headcount, human_headcount, feline_headcount, attendance_rate, overtime_hours, timesheets_approved, webdav_path_pdf, webdav_path_csv, csv_content, pdf_content, generated_by, created_at, updated_at`

	var res ExecutiveReport
	err := r.db.QueryRow(ctx, query, id, status, webdavPDF, webdavCSV).Scan(
		&res.ReportID, &res.Period, &res.Title, &res.Status,
		&res.TotalRevenue, &res.OperatingExpenses, &res.PayrollExpenses, &res.NetProfit,
		&res.OperationalBurn, &res.CashReserves, &res.RunwayMonths,
		&res.TotalHeadcount, &res.HumanHeadcount, &res.FelineHeadcount,
		&res.AttendanceRate, &res.OvertimeHours, &res.TimesheetsApproved,
		&res.WebDAVPathPDF, &res.WebDAVPathCSV, &res.CSVContent, &res.PDFContent, &res.GeneratedBy,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
