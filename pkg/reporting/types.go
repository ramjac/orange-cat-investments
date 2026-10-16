package reporting

import (
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
)

// WorkforceMetrics contains aggregated workforce operational metrics.
type WorkforceMetrics struct {
	Period             string  `json:"period"`
	TotalHeadcount     int     `json:"total_headcount"`
	HumanHeadcount     int     `json:"human_headcount"`
	FelineHeadcount    int     `json:"feline_headcount"`
	AttendanceRate     float64 `json:"attendance_rate"`
	TotalHoursWorked   float64 `json:"total_hours_worked"`
	OvertimeHours      float64 `json:"overtime_hours"`
	TimesheetsApproved int     `json:"timesheets_approved"`
	PendingReviews     int     `json:"pending_reviews"`
	OpenIncidents      int     `json:"open_incidents"`
}

// ExecutiveReport represents a full monthly executive management report.
type ExecutiveReport struct {
	ReportID         string                  `json:"report_id"`
	Period           string                  `json:"period"`
	Title            string                  `json:"title"`
	Status           string                  `json:"status"` // "archived", "generating", "failed"
	FinancialSummary frappe.FinancialMetrics `json:"financial_summary"`
	WorkforceSummary WorkforceMetrics        `json:"workforce_summary"`
	WebDAVPathPDF    string                  `json:"webdav_path_pdf"`
	WebDAVPathCSV    string                  `json:"webdav_path_csv"`
	CSVContent       string                  `json:"csv_content,omitempty"`
	PDFContent       []byte                  `json:"pdf_content,omitempty"`
	GeneratedBy      string                  `json:"generated_by"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}
