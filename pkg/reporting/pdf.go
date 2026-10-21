package reporting

import (
	"bytes"
	"fmt"
	"strings"
)

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}

// GenerateExecutivePDF creates a valid PDF 1.4 document containing the executive management report.
func GenerateExecutivePDF(r *ExecutiveReport) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("report is nil")
	}

	var streamBuf bytes.Buffer

	// Stream graphics & text commands
	sb := &streamBuf

	// Draw top banner (OCI Navy/Orange theme)
	// Navy banner: x=40, y=710, w=532, h=50
	sb.WriteString("0.1 0.15 0.25 rg\n") // Dark Navy
	sb.WriteString("40 715 532 50 re f\n")

	// Orange accent strip below banner
	sb.WriteString("0.96 0.5 0.15 rg\n") // OCI Orange
	sb.WriteString("40 710 532 5 re f\n")

	// Header Text
	sb.WriteString("BT\n")
	sb.WriteString("/F1 16 Tf\n")
	sb.WriteString("1 1 1 rg\n") // White text
	sb.WriteString("55 742 Td\n")
	sb.WriteString("(ORANGE CAT INVESTMENTS  |  EXECUTIVE MANAGEMENT & BI REPORT) Tj\n")
	sb.WriteString("ET\n")

	sb.WriteString("BT\n")
	sb.WriteString("/F2 9 Tf\n")
	sb.WriteString("0.85 0.85 0.9 rg\n")
	sb.WriteString("55 725 Td\n")
	sb.WriteString(fmt.Sprintf("(Period: %s   |   Status: %s   |   Generated: %s) Tj\n",
		escapePDF(r.Period), escapePDF(strings.ToUpper(r.Status)), r.CreatedAt.Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString("ET\n")

	// Section 1: Financial Performance (ERPNext General Ledger)
	sb.WriteString("0.96 0.5 0.15 rg\n")
	sb.WriteString("40 680 532 18 re f\n") // Section header box
	sb.WriteString("BT\n/F1 11 Tf\n1 1 1 rg\n50 685 Td\n(1. FINANCIAL PERFORMANCE & CASH RUNWAY (ERPNext)) Tj\nET\n")

	fin := r.FinancialSummary
	finItems := []struct {
		Label string
		Value string
		Note  string
	}{
		{"Total Gross Revenue", fmt.Sprintf("$%.2f %s", fin.TotalRevenue, fin.Currency), "Advisory fees & feline alpha trading returns"},
		{"Operating Expenses", fmt.Sprintf("$%.2f %s", fin.OperatingExpenses, fin.Currency), "Facilities, hardware, cloud infra & catnip"},
		{"Workforce Payroll", fmt.Sprintf("$%.2f %s", fin.PayrollExpenses, fin.Currency), "Human staff salaries & feline allocations"},
		{"Net Operating Profit", fmt.Sprintf("$%.2f %s", fin.NetProfit, fin.Currency), "Net margin after operating and payroll expenses"},
		{"Monthly Operational Burn", fmt.Sprintf("$%.2f %s", fin.OperationalBurn, fin.Currency), "Total operating cash outflow"},
		{"Liquid Cash Reserves", fmt.Sprintf("$%.2f %s", fin.CashReserves, fin.Currency), "Total commercial bank liquidity"},
		{"Cash Runway Buffer", fmt.Sprintf("%.1f Months", fin.RunwayMonths), "Reserves divided by monthly operational burn"},
	}

	y := 660
	for i, item := range finItems {
		// Alternate row background
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.95 0.96 0.98 rg\n40 %d 532 18 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 9 Tf\n0.15 0.15 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("50 %d Td\n(%s) Tj\nET\n", y, escapePDF(item.Label)))

		sb.WriteString("BT\n/F1 9 Tf\n0.1 0.35 0.1 rg\n") // Green-ish dark for value
		sb.WriteString(fmt.Sprintf("210 %d Td\n(%s) Tj\nET\n", y, escapePDF(item.Value)))

		sb.WriteString("BT\n/F2 8 Tf\n0.4 0.45 0.5 rg\n")
		sb.WriteString(fmt.Sprintf("340 %d Td\n(%s) Tj\nET\n", y, escapePDF(item.Note)))

		y -= 20
	}

	// Section 2: Workforce Operations & Feline Telemetry
	y -= 10
	sb.WriteString(fmt.Sprintf("0.1 0.15 0.25 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 11 Tf\n1 1 1 rg\n50 %d Td\n(2. WORKFORCE OPERATIONS & FELINE TELEMETRY) Tj\nET\n", y+5))

	y -= 20
	wf := r.WorkforceSummary
	wfItems := []struct {
		Label string
		Value string
		Note  string
	}{
		{"Total Active Headcount", fmt.Sprintf("%d Personnel", wf.TotalHeadcount), fmt.Sprintf("%d Humans, %d Feline Executives", wf.HumanHeadcount, wf.FelineHeadcount)},
		{"Shift Attendance Rate", fmt.Sprintf("%.2f%%", wf.AttendanceRate), "Verified via RFID badge and timesheet logs"},
		{"Total Logged Work Hours", fmt.Sprintf("%.1f Hours", wf.TotalHoursWorked), "Combined duty and observation shifts"},
		{"Overtime & On-Call Hours", fmt.Sprintf("%.1f Hours", wf.OvertimeHours), "Additional shift & edge incident resolution"},
		{"Approved Timesheets", fmt.Sprintf("%d Submitted", wf.TimesheetsApproved), "Reviewed and authorized by management"},
		{"Performance Appraisals", fmt.Sprintf("%d Pending", wf.PendingReviews), "Quarterly reviews & veterinary wellness audits"},
		{"Workplace Safety Incidents", fmt.Sprintf("%d Open", wf.OpenIncidents), "Safety tickets logged across habitats & labs"},
	}

	for i, item := range wfItems {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.95 0.96 0.98 rg\n40 %d 532 18 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 9 Tf\n0.15 0.15 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("50 %d Td\n(%s) Tj\nET\n", y, escapePDF(item.Label)))

		sb.WriteString("BT\n/F1 9 Tf\n0.15 0.2 0.35 rg\n")
		sb.WriteString(fmt.Sprintf("210 %d Td\n(%s) Tj\nET\n", y, escapePDF(item.Value)))

		sb.WriteString("BT\n/F2 8 Tf\n0.4 0.45 0.5 rg\n")
		sb.WriteString(fmt.Sprintf("340 %d Td\n(%s) Tj\nET\n", y, escapePDF(item.Note)))

		y -= 20
	}

	// Section 3: Nextcloud Compliance Archival & Audit Trail
	y -= 10
	sb.WriteString(fmt.Sprintf("0.2 0.45 0.35 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 11 Tf\n1 1 1 rg\n50 %d Td\n(3. COMPLIANCE ARCHIVE & AUDIT PRESERVATION (Nextcloud)) Tj\nET\n", y+5))

	y -= 22
	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(Nextcloud WebDAV PDF Path: %s) Tj\nET\n", y, escapePDF(r.WebDAVPathPDF)))

	y -= 16
	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(Nextcloud WebDAV CSV Path: %s) Tj\nET\n", y, escapePDF(r.WebDAVPathCSV)))

	y -= 16
	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(Archived Under Policy: Executive/Reports/YYYY-MM/  |  Classification: STRICTLY CONFIDENTIAL) Tj\nET\n", y))

	// Footer separator & note
	sb.WriteString("0.8 0.8 0.85 rg\n40 45 532 1 re f\n")
	sb.WriteString("BT\n/F2 8 Tf\n0.5 0.5 0.55 rg\n")
	sb.WriteString("50 32 Td\n(CONFIDENTIAL // ORANGE CAT INVESTMENTS INTERNAL BI // AUTOMATED MONTHLY REPORT) Tj\nET\n")

	streamBytes := streamBuf.Bytes()

	// Assemble PDF 1.4 objects
	var out bytes.Buffer
	offsets := make([]int, 8)

	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	// Object 1: Catalog
	offsets[1] = out.Len()
	out.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	offsets[2] = out.Len()
	out.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Object 3: Page
	offsets[3] = out.Len()
	out.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R /F2 6 0 R /F3 7 0 R >> >> >>\nendobj\n")

	// Object 4: Contents Stream
	offsets[4] = out.Len()
	fmt.Fprintf(&out, "4 0 obj\n<< /Length %d >>\nstream\n", len(streamBytes))
	out.Write(streamBytes)
	out.WriteString("\nendstream\nendobj\n")

	// Object 5: Font F1 (Helvetica-Bold)
	offsets[5] = out.Len()
	out.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>\nendobj\n")

	// Object 6: Font F2 (Helvetica)
	offsets[6] = out.Len()
	out.WriteString("6 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// Object 7: Font F3 (Courier)
	offsets[7] = out.Len()
	out.WriteString("7 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>\nendobj\n")

	// Cross-reference table
	xrefStart := out.Len()
	out.WriteString("xref\n0 8\n0000000000 65535 f \n")
	for i := 1; i <= 7; i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}

	// Trailer
	out.WriteString("trailer\n<< /Size 8 /Root 1 0 R >>\nstartxref\n")
	fmt.Fprintf(&out, "%d\n%%%%EOF\n", xrefStart)

	return out.Bytes(), nil
}
