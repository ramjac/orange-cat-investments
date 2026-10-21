package reporting

import (
	"bytes"
	"fmt"
	"strings"
)

func buildPDF(streamBytes []byte) []byte {
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

	return out.Bytes()
}

// GenerateMonthlyStatementPDF generates an authentic PDF 1.4 statement for a customer portfolio.
func GenerateMonthlyStatementPDF(stmt *CustomerMonthlyStatement) ([]byte, error) {
	var streamBuf bytes.Buffer
	sb := &streamBuf

	// Header Banner (Navy + Orange)
	sb.WriteString("0.1 0.15 0.25 rg\n40 715 532 50 re f\n")
	sb.WriteString("0.96 0.5 0.15 rg\n40 710 532 5 re f\n")

	// Header Text
	sb.WriteString("BT\n/F1 16 Tf\n1 1 1 rg\n55 742 Td\n")
	sb.WriteString("(ORANGE CAT INVESTMENTS  |  MONTHLY PORTFOLIO STATEMENT) Tj\nET\n")

	sb.WriteString("BT\n/F2 9 Tf\n0.85 0.85 0.9 rg\n55 725 Td\n")
	sb.WriteString(fmt.Sprintf("(Period: %s   |   Account: %s   |   Statement ID: %s) Tj\nET\n",
		escapePDF(stmt.Period), escapePDF(stmt.AccountName), escapePDF(stmt.StatementID)))

	// Customer Info Box
	sb.WriteString("0.95 0.96 0.98 rg\n40 665 532 35 re f\n")
	sb.WriteString("BT\n/F1 10 Tf\n0.15 0.15 0.2 rg\n55 683 Td\n")
	sb.WriteString(fmt.Sprintf("(Client: %s) Tj\nET\n", escapePDF(stmt.CustomerName)))
	sb.WriteString("BT\n/F2 8.5 Tf\n0.4 0.45 0.5 rg\n55 671 Td\n")
	sb.WriteString(fmt.Sprintf("(Customer Ref: %s   |   Portfolio Ref: %s   |   Date Generated: %s) Tj\nET\n",
		escapePDF(stmt.CustomerID), escapePDF(stmt.PortfolioID), stmt.CreatedAt.Format("2006-01-02 15:04:05 UTC")))

	// Section 1: Portfolio Performance & Balances
	y := 635
	sb.WriteString(fmt.Sprintf("0.96 0.5 0.15 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 11 Tf\n1 1 1 rg\n50 %d Td\n(1. PORTFOLIO VALUATION & PERFORMANCE SUMMARY) Tj\nET\n", y+5))

	items := []struct {
		Label string
		Value string
		Note  string
	}{
		{"Ending Portfolio Value", fmt.Sprintf("$%.2f USD", stmt.EndingBalanceUSD), "Total equity market value plus cash reserves"},
		{"Cash Balance (Available)", fmt.Sprintf("$%.2f USD", stmt.CashBalanceUSD), "Liquid unallocated brokerage cash reserves"},
		{"Net Monthly Contributions", fmt.Sprintf("$%.2f USD", stmt.NetContributionsUSD), "Recurring ACH bank deposits less distributions"},
		{"Monthly Gross Return", fmt.Sprintf("%+.2f%%", stmt.GrossReturnPct), "Time-weighted asset performance curve"},
		{"Feline Alpha Attribution", fmt.Sprintf("$%.2f USD", stmt.ZoomieAlphaAttributionUSD), "Excess alpha derived from feline biometric triggers"},
		{"Sharpe Ratio (Annualized)", fmt.Sprintf("%.2f", stmt.SharpeRatio), "Risk-adjusted performance over risk-free rate"},
		{"Sortino Ratio", fmt.Sprintf("%.2f", stmt.SortinoRatio), "Downside risk-adjusted alpha efficiency metric"},
	}

	y -= 20
	for i, it := range items {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.95 0.96 0.98 rg\n40 %d 532 18 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 9 Tf\n0.15 0.15 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("50 %d Td\n(%s) Tj\nET\n", y, escapePDF(it.Label)))

		sb.WriteString("BT\n/F1 9 Tf\n0.1 0.35 0.1 rg\n")
		sb.WriteString(fmt.Sprintf("220 %d Td\n(%s) Tj\nET\n", y, escapePDF(it.Value)))

		sb.WriteString("BT\n/F2 8 Tf\n0.4 0.45 0.5 rg\n")
		sb.WriteString(fmt.Sprintf("340 %d Td\n(%s) Tj\nET\n", y, escapePDF(it.Note)))

		y -= 20
	}

	// Section 2: Asset Allocation & Holdings
	y -= 10
	sb.WriteString(fmt.Sprintf("0.1 0.15 0.25 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 11 Tf\n1 1 1 rg\n50 %d Td\n(2. ASSET ALLOCATION & OPEN EQUITY POSITIONS) Tj\nET\n", y+5))

	y -= 18
	// Table Headers
	sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n0.3 0.35 0.4 rg\n50 %d Td\n(SYMBOL) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n0.3 0.35 0.4 rg\n130 %d Td\n(QUANTITY) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n0.3 0.35 0.4 rg\n210 %d Td\n(COST BASIS) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n0.3 0.35 0.4 rg\n300 %d Td\n(CURRENT PRICE) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n0.3 0.35 0.4 rg\n400 %d Td\n(MARKET VALUE) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n0.3 0.35 0.4 rg\n490 %d Td\n(UNREALIZED) Tj\nET\n", y))

	y -= 16
	for i, h := range stmt.Holdings {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.95 0.96 0.98 rg\n40 %d 532 16 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 8.5 Tf\n0.15 0.15 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("50 %d Td\n(%s) Tj\nET\n", y, escapePDF(h.Symbol)))

		sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("130 %d Td\n(%.2f) Tj\nET\n", y, h.Quantity))

		sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("210 %d Td\n($%.2f) Tj\nET\n", y, h.CostBasisUSD))

		sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("300 %d Td\n($%.2f) Tj\nET\n", y, h.CurrentPriceUSD))

		sb.WriteString("BT\n/F1 8.5 Tf\n0.1 0.2 0.4 rg\n")
		sb.WriteString(fmt.Sprintf("400 %d Td\n($%.2f) Tj\nET\n", y, h.MarketValueUSD))

		gainColor := "0.1 0.45 0.1" // green
		if h.UnrealizedGainUSD < 0 {
			gainColor = "0.75 0.15 0.1" // red
		}
		sb.WriteString(fmt.Sprintf("BT\n/F1 8.5 Tf\n%s rg\n", gainColor))
		sb.WriteString(fmt.Sprintf("490 %d Td\n(%+.2f%%) Tj\nET\n", y, h.UnrealizedGainPct))

		y -= 18
	}

	// Section 3: Legal Disclosures & Custody Audit
	y -= 8
	sb.WriteString(fmt.Sprintf("0.2 0.45 0.35 rg\n40 %d 532 16 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 10 Tf\n1 1 1 rg\n50 %d Td\n(3. REGULATORY DISCLOSURES & CUSTODIAL SAFEGUARDS) Tj\nET\n", y+4))

	y -= 18
	sb.WriteString("BT\n/F2 7.5 Tf\n0.35 0.4 0.45 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(Securities offered through Orange Cat Investments Brokerage Services LLC. Member FINRA / SIPC.) Tj\nET\n", y))

	y -= 12
	sb.WriteString("BT\n/F2 7.5 Tf\n0.35 0.4 0.45 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(Feline-driven algorithmic allocation signals do not guarantee future investment returns. Assets held in custody.) Tj\nET\n", y))

	// Footer
	sb.WriteString("0.8 0.8 0.85 rg\n40 40 532 1 re f\n")
	sb.WriteString("BT\n/F2 8 Tf\n0.5 0.5 0.55 rg\n50 28 Td\n")
	sb.WriteString("(CONFIDENTIAL // PREPARED FOR THE DESIGNATED ACCOUNT HOLDER ONLY // NOT FOR REDISTRIBUTION) Tj\nET\n")

	return buildPDF(sb.Bytes()), nil
}

// GenerateForm1099BPDF generates IRS Form 1099-B tax document for a customer.
func GenerateForm1099BPDF(stmt *Form1099BStatement) ([]byte, error) {
	var streamBuf bytes.Buffer
	sb := &streamBuf

	// IRS 1099-B Top Header Banner
	sb.WriteString("0.1 0.15 0.25 rg\n40 715 532 50 re f\n")
	sb.WriteString("0.96 0.5 0.15 rg\n40 710 532 5 re f\n")

	sb.WriteString("BT\n/F1 15 Tf\n1 1 1 rg\n55 742 Td\n")
	sb.WriteString("(FORM 1099-B  |  PROCEEDS FROM BROKER AND BARTER TRANSACTIONS) Tj\nET\n")

	sb.WriteString("BT\n/F2 9 Tf\n0.85 0.85 0.9 rg\n55 725 Td\n")
	sb.WriteString(fmt.Sprintf("(Tax Year: %d   |   OMB No. 1545-0715   |   Copy B for Recipient) Tj\nET\n", stmt.TaxYear))

	// Payer & Recipient Split Box
	sb.WriteString("0.95 0.96 0.98 rg\n40 645 260 55 re f\n")
	sb.WriteString("0.95 0.96 0.98 rg\n312 645 260 55 re f\n")

	// Payer Info (Left)
	sb.WriteString("BT\n/F1 9 Tf\n0.15 0.15 0.2 rg\n50 685 Td\n(PAYER'S name, address, and ZIP code:) Tj\nET\n")
	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n50 672 Td\n")
	sb.WriteString(fmt.Sprintf("(%s) Tj\nET\n", escapePDF(stmt.PayerName)))
	sb.WriteString("BT\n/F2 8 Tf\n0.35 0.35 0.4 rg\n50 660 Td\n")
	sb.WriteString(fmt.Sprintf("(Payer TIN: %s   |   Custody Brokerage) Tj\nET\n", escapePDF(stmt.PayerTIN)))

	// Recipient Info (Right)
	sb.WriteString("BT\n/F1 9 Tf\n0.15 0.15 0.2 rg\n322 685 Td\n(RECIPIENT'S name and identifying number:) Tj\nET\n")
	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.2 0.2 rg\n322 672 Td\n")
	sb.WriteString(fmt.Sprintf("(%s) Tj\nET\n", escapePDF(stmt.CustomerName)))
	sb.WriteString("BT\n/F2 8 Tf\n0.35 0.35 0.4 rg\n322 660 Td\n")
	sb.WriteString(fmt.Sprintf("(Recipient TIN: %s   |   Account ID: %s) Tj\nET\n", escapePDF(stmt.RecipientTIN), escapePDF(stmt.PortfolioID)))

	// Tax Totals Summary Box
	y := 615
	sb.WriteString(fmt.Sprintf("0.96 0.5 0.15 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 10 Tf\n1 1 1 rg\n50 %d Td\n(TAX YEAR %d CAPITAL GAINS & PROCEEDS SUMMARY) Tj\nET\n", stmt.TaxYear, y+5))

	y -= 20
	taxItems := []struct {
		Box   string
		Label string
		Value string
	}{
		{"Box 1d", "Total Gross Proceeds from Transactions", fmt.Sprintf("$%.2f USD", stmt.TotalGrossProceedsUSD)},
		{"Box 1e", "Total Cost or Other Basis", fmt.Sprintf("$%.2f USD", stmt.TotalCostBasisUSD)},
		{"Box 2", "Total Realized Net Capital Gain / (Loss)", fmt.Sprintf("$%.2f USD", stmt.TotalRealizedGainUSD)},
		{"Part I", "Short-Term Transactions (Held 1 Year or Less)", fmt.Sprintf("$%.2f USD", stmt.ShortTermGainUSD)},
		{"Part II", "Long-Term Transactions (Held Over 1 Year)", fmt.Sprintf("$%.2f USD", stmt.LongTermGainUSD)},
	}

	for i, it := range taxItems {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.95 0.96 0.98 rg\n40 %d 532 17 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 8.5 Tf\n0.96 0.4 0.1 rg\n")
		sb.WriteString(fmt.Sprintf("50 %d Td\n([%s]) Tj\nET\n", y, escapePDF(it.Box)))

		sb.WriteString("BT\n/F1 8.5 Tf\n0.15 0.15 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("110 %d Td\n(%s) Tj\nET\n", y, escapePDF(it.Label)))

		sb.WriteString("BT\n/F1 8.5 Tf\n0.1 0.2 0.35 rg\n")
		sb.WriteString(fmt.Sprintf("440 %d Td\n(%s) Tj\nET\n", y, escapePDF(it.Value)))

		y -= 19
	}

	// Itemized Transactions Header
	y -= 10
	sb.WriteString(fmt.Sprintf("0.1 0.15 0.25 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 10 Tf\n1 1 1 rg\n50 %d Td\n(ITEMIZED BROKER DISPOSITIONS & WASH SALE ADJUSTMENTS) Tj\nET\n", y+5))

	y -= 18
	// Table Headers
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n45 %d Td\n(DESCRIPTION) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n140 %d Td\n(ACQUIRED) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n205 %d Td\n(SOLD) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n270 %d Td\n(PROCEEDS) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n345 %d Td\n(COST BASIS) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n425 %d Td\n(WASH SALE) Tj\nET\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 7.5 Tf\n0.3 0.35 0.4 rg\n495 %d Td\n(GAIN/LOSS) Tj\nET\n", y))

	y -= 15
	for i, tx := range stmt.Transactions {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.95 0.96 0.98 rg\n40 %d 532 15 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 8 Tf\n0.15 0.15 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("45 %d Td\n(%s) Tj\nET\n", y, escapePDF(tx.Description)))

		sb.WriteString("BT\n/F2 8 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("140 %d Td\n(%s) Tj\nET\n", y, escapePDF(tx.DateAcquired)))

		sb.WriteString("BT\n/F2 8 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("205 %d Td\n(%s) Tj\nET\n", y, escapePDF(tx.DateSold)))

		sb.WriteString("BT\n/F2 8 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("270 %d Td\n($%.2f) Tj\nET\n", y, tx.GrossProceedsUSD))

		sb.WriteString("BT\n/F2 8 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("345 %d Td\n($%.2f) Tj\nET\n", y, tx.CostBasisUSD))

		sb.WriteString("BT\n/F2 8 Tf\n0.2 0.2 0.2 rg\n")
		sb.WriteString(fmt.Sprintf("425 %d Td\n($%.2f) Tj\nET\n", y, tx.WashSaleLossDisallowed))

		gainColor := "0.1 0.4 0.1"
		if tx.RealizedGainLossUSD < 0 {
			gainColor = "0.7 0.1 0.1"
		}
		sb.WriteString(fmt.Sprintf("BT\n/F1 8 Tf\n%s rg\n", gainColor))
		sb.WriteString(fmt.Sprintf("495 %d Td\n($%+.2f) Tj\nET\n", y, tx.RealizedGainLossUSD))

		y -= 16
	}

	// IRS Footer text
	sb.WriteString("0.8 0.8 0.85 rg\n40 45 532 1 re f\n")
	sb.WriteString("BT\n/F2 7.5 Tf\n0.4 0.45 0.5 rg\n50 32 Td\n")
	sb.WriteString("(This is important tax information and is being furnished to the Internal Revenue Service. If you are required to file a return, a negligence penalty may be imposed.) Tj\nET\n")

	return buildPDF(sb.Bytes()), nil
}

// GenerateGDPRCertificatePDF generates a cryptographic data erasure certificate under GDPR Article 17.
func GenerateGDPRCertificatePDF(cert *GDPRDeletionCertificate) ([]byte, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate data is nil")
	}

	var streamBuf bytes.Buffer
	sb := &streamBuf

	// Header Banner: Emerald Green theme representing verified compliance & clean slate
	sb.WriteString("0.1 0.3 0.25 rg\n40 715 532 50 re f\n")
	sb.WriteString("0.2 0.65 0.4 rg\n40 710 532 5 re f\n")

	sb.WriteString("BT\n/F1 15 Tf\n1 1 1 rg\n55 742 Td\n")
	sb.WriteString("(CERTIFICATE OF PERMANENT DATA ERASURE // GDPR ARTICLE 17) Tj\nET\n")

	sb.WriteString("BT\n/F2 9 Tf\n0.85 0.95 0.9 rg\n55 725 Td\n")
	sb.WriteString(fmt.Sprintf("(Certificate ID: %s   |   Authority: %s) Tj\nET\n",
		escapePDF(cert.CertificateID), escapePDF(cert.SupervisoryAuthority)))

	// Main Legal Attestation Box
	sb.WriteString("0.96 0.98 0.96 rg\n40 625 532 75 re f\n")
	sb.WriteString("0.2 0.65 0.4 rg\n40 625 4 75 re f\n") // Green left border

	sb.WriteString("BT\n/F1 10.5 Tf\n0.1 0.25 0.15 rg\n55 683 Td\n")
	sb.WriteString("(OFFICIAL VERIFICATION OF RIGHT TO ERASURE / RIGHT TO BE FORGOTTEN) Tj\nET\n")

	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.25 0.2 rg\n55 667 Td\n")
	sb.WriteString("(This document certifies that Orange Cat Investments has completed the permanent, irrecoverable erasure) Tj\nET\n")
	sb.WriteString("BT\n/F2 8.5 Tf\n0.2 0.25 0.2 rg\n55 655 Td\n")
	sb.WriteString("(of all Personally Identifiable Information (PII), active session tokens, and identity links associated with the subject.) Tj\nET\n")
	sb.WriteString("BT\n/F2 8 Tf\n0.35 0.45 0.35 rg\n55 640 Td\n")
	sb.WriteString(fmt.Sprintf("(Legal Basis: %s   |   Executed At: %s) Tj\nET\n",
		escapePDF(cert.LegalBasis), cert.ErasureTimestamp.Format("2006-01-02 15:04:05 UTC")))

	// Erasure Verification Details
	y := 595
	sb.WriteString(fmt.Sprintf("0.1 0.3 0.25 rg\n40 %d 532 18 re f\n", y))
	sb.WriteString(fmt.Sprintf("BT\n/F1 10 Tf\n1 1 1 rg\n50 %d Td\n(ERASURE ENGINE AUDIT SUMMARY) Tj\nET\n", y+5))

	y -= 20
	details := []struct {
		Label string
		Value string
		Note  string
	}{
		{"Customer ID Pseudonym Hash", cert.CustomerIDHash[:24] + "...", "SHA-256 one-way cryptographic hash of subject ID"},
		{"Customer Email Hash", cert.CustomerEmailHash[:24] + "...", "SHA-256 one-way cryptographic hash of subject email"},
		{"Active Sessions Invalidated", fmt.Sprintf("%d Session(s)", cert.SessionsPurgedCount), "Revoked from Valkey & in-memory session caches"},
		{"Total Records Scrubbed/Purged", fmt.Sprintf("%d Record(s)", cert.RecordsScrubbedCount), "Scrubbed from customer_api_keys, webhooks, deposits"},
		{"Relational Tables Scrubbed", strings.Join(cert.TablesScrubbed, ", "), "Core relational schemas sanitized of PII"},
		{"Transaction Ledger Anonymization", "Verified Anonymized", "Accounting ledgers decoupled from subject identity"},
	}

	for i, d := range details {
		if i%2 == 0 {
			sb.WriteString(fmt.Sprintf("0.96 0.98 0.96 rg\n40 %d 532 18 re f\n", y-3))
		}
		sb.WriteString("BT\n/F1 8.5 Tf\n0.15 0.2 0.15 rg\n")
		sb.WriteString(fmt.Sprintf("50 %d Td\n(%s) Tj\nET\n", y, escapePDF(d.Label)))

		sb.WriteString("BT\n/F1 8.5 Tf\n0.1 0.3 0.4 rg\n")
		sb.WriteString(fmt.Sprintf("230 %d Td\n(%s) Tj\nET\n", y, escapePDF(d.Value)))

		sb.WriteString("BT\n/F2 8 Tf\n0.4 0.45 0.4 rg\n")
		sb.WriteString(fmt.Sprintf("360 %d Td\n(%s) Tj\nET\n", y, escapePDF(d.Note)))

		y -= 20
	}

	// Cryptographic Digital Seal & Signature Box
	y -= 15
	sb.WriteString(fmt.Sprintf("0.92 0.95 0.93 rg\n40 %d 532 60 re f\n", y-42))
	sb.WriteString("BT\n/F1 9.5 Tf\n0.1 0.25 0.15 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(CRYPTOGRAPHIC DIGITAL SEAL & TAMPER-EVIDENT SIGNATURE) Tj\nET\n", y+2))

	y -= 14
	sb.WriteString("BT\n/F3 8 Tf\n0.1 0.35 0.2 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(HMAC-SHA256 Signature: %s) Tj\nET\n", y, escapePDF(cert.DigitalSignature)))

	y -= 14
	sb.WriteString("BT\n/F2 8 Tf\n0.3 0.35 0.3 rg\n")
	sb.WriteString(fmt.Sprintf("50 %d Td\n(Signed by: %s Cryptographic Key Infrastructure) Tj\nET\n", y, escapePDF(cert.SupervisoryAuthority)))

	// Footer
	sb.WriteString("0.7 0.8 0.75 rg\n40 45 532 1 re f\n")
	sb.WriteString("BT\n/F2 8 Tf\n0.4 0.5 0.4 rg\n50 32 Td\n")
	sb.WriteString("(OFFICIAL GDPR ARTICLE 17 COMPLIANCE RECORD // PRESERVE FOR REGULATORY VERIFICATION) Tj\nET\n")

	return buildPDF(sb.Bytes()), nil
}
