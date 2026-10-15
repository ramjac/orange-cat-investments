package reporting

import (
	"bytes"
	"encoding/csv"
	"strings"
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

func TestGenerateExecutiveCSV(t *testing.T) {
	r := sampleReport()
	csvStr, err := GenerateExecutiveCSV(r)
	require.NoError(t, err)
	assert.NotEmpty(t, csvStr)

	// Verify RFC 4180 parsing
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 20)

	// Check header
	assert.Equal(t, []string{"Category", "Metric", "Value", "Unit", "Notes"}, records[0])

	// Check specific metrics
	foundRevenue := false
	foundHeadcount := false
	for _, rec := range records {
		if rec[1] == "Total Revenue" {
			foundRevenue = true
			assert.Equal(t, "1350000.00", rec[2])
			assert.Equal(t, "USD", rec[3])
		}
		if rec[1] == "Total Headcount" {
			foundHeadcount = true
			assert.Equal(t, "12", rec[2])
			assert.Equal(t, "Staff", rec[3])
		}
	}
	assert.True(t, foundRevenue, "Total Revenue should exist in CSV")
	assert.True(t, foundHeadcount, "Total Headcount should exist in CSV")
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
