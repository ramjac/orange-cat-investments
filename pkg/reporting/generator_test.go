package reporting

import (
	"bytes"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport() *ExecutiveReport {
	return &ExecutiveReport{
		ReportID: "rep-2026-10",
		Period:   "2026-10",
		Title:    "OCI Executive Management Report - October 2026",
		Status:   "archived",
		FinancialSummary: frappe.FinancialMetrics{
			Period:            "2026-10",
			Currency:          "USD",
			TotalRevenue:      1350000.0,
			OperatingExpenses: 320000.0,
			PayrollExpenses:   430000.0,
			NetProfit:         600000.0,
			OperationalBurn:   750000.0,
			CashReserves:      9800000.0,
			RunwayMonths:      13.06,
		},
		WorkforceSummary: WorkforceMetrics{
			Period:             "2026-10",
			TotalHeadcount:     12,
			HumanHeadcount:     10,
			FelineHeadcount:    2,
			AttendanceRate:     99.4,
			TotalHoursWorked:   1860.5,
			OvertimeHours:      44.0,
			TimesheetsApproved: 19,
			PendingReviews:     1,
			OpenIncidents:      0,
		},
		WebDAVPathPDF: "/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_report_2026-10.pdf",
		WebDAVPathCSV: "/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_report_2026-10.csv",
		GeneratedBy:   "system-reporting-worker",
		CreatedAt:     time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestGenerateExecutivePDF(t *testing.T) {
	r := sampleReport()
	pdfBytes, err := GenerateExecutivePDF(r)
	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)

	// Verify PDF Header
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4\n")), "PDF must begin with %%PDF-1.4")

	// Verify PDF Trailer and EOF
	assert.True(t, bytes.HasSuffix(pdfBytes, []byte("%%EOF\n")), "PDF must end with %%%%EOF")

	// Verify internal PDF structure
	pdfStr := string(pdfBytes)
	assert.Contains(t, pdfStr, "/Type /Catalog")
	assert.Contains(t, pdfStr, "/Type /Pages")
	assert.Contains(t, pdfStr, "/Type /Page")
	assert.Contains(t, pdfStr, "ORANGE CAT INVESTMENTS")
	assert.Contains(t, pdfStr, "1350000.00")
	assert.Contains(t, pdfStr, "Executive/Reports/2026-10")
	assert.Contains(t, pdfStr, "xref")
	assert.Contains(t, pdfStr, "trailer")
	assert.Contains(t, pdfStr, "startxref")
}
