package reporting

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMonthlyStatementPDF(t *testing.T) {
	stmt := &CustomerMonthlyStatement{
		StatementID:               "stmt-test-001",
		PortfolioID:               "port-test-001",
		CustomerID:                "cust-test-001",
		CustomerName:              "Arthur Pendelton",
		AccountName:               "OCI Alpha Growth Fund",
		Period:                    "March 2026",
		TaxYear:                   2026,
		OpeningBalanceUSD:         100000.00,
		EndingBalanceUSD:          128450.75,
		CashBalanceUSD:            32100.25,
		NetContributionsUSD:       5000.00,
		TotalRealizedGainUSD:      12450.00,
		TotalUnrealizedGainUSD:    11000.75,
		GrossReturnPct:            28.45,
		SharpeRatio:               2.14,
		SortinoRatio:              3.08,
		ZoomieAlphaAttributionUSD: 8420.50,
		Holdings: []StatementHolding{
			{
				Symbol:            "AAPL",
				Quantity:          120.0,
				CostBasisUSD:      21420.00,
				CurrentPriceUSD:   224.30,
				MarketValueUSD:    26916.00,
				UnrealizedGainUSD: 5496.00,
				UnrealizedGainPct: 25.66,
			},
			{
				Symbol:            "NVDA",
				Quantity:          85.0,
				CostBasisUSD:      9367.00,
				CurrentPriceUSD:   138.80,
				MarketValueUSD:    11798.00,
				UnrealizedGainUSD: 2431.00,
				UnrealizedGainPct: 25.95,
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	pdfBytes, err := GenerateMonthlyStatementPDF(stmt)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("xref")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("%%EOF")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("ORANGE CAT INVESTMENTS")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("MONTHLY PORTFOLIO STATEMENT")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("AAPL")))
}

func TestGenerateForm1099BPDF(t *testing.T) {
	stmt := &Form1099BStatement{
		StatementID:           "1099b-test-2025",
		PortfolioID:           "port-test-001",
		CustomerID:            "cust-test-001",
		CustomerName:          "Chloe Spark",
		TaxYear:               2025,
		PayerName:             "Orange Cat Investments LLC",
		PayerTIN:              "12-3456789",
		RecipientTIN:          "***-**-8842",
		TotalGrossProceedsUSD: 75400.00,
		TotalCostBasisUSD:     58200.00,
		TotalRealizedGainUSD:  17200.00,
		ShortTermGainUSD:      12400.00,
		LongTermGainUSD:       4800.00,
		Transactions: []Form1099BTransaction{
			{
				Description:            "50 shs NVDA",
				Symbol:                 "NVDA",
				Quantity:               50.0,
				DateAcquired:           "2025-02-10",
				DateSold:               "2025-08-14",
				GrossProceedsUSD:       6940.00,
				CostBasisUSD:           5500.00,
				WashSaleLossDisallowed: 0.00,
				RealizedGainLossUSD:    1440.00,
				IsShortTerm:            true,
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	pdfBytes, err := GenerateForm1099BPDF(stmt)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("FORM 1099-B")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("Orange Cat Investments LLC")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("NVDA")))
}

func TestGenerateGDPRCertificatePDF(t *testing.T) {
	cert := &GDPRDeletionCertificate{
		CertificateID:        "cert-gdpr-12345",
		CustomerIDHash:       "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		CustomerEmailHash:    "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
		ErasureTimestamp:     time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC),
		LegalBasis:           "GDPR Article 17 - Right to Erasure",
		SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
		TablesScrubbed:       []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions"},
		SessionsPurgedCount:  3,
		RecordsScrubbedCount: 7,
		DigitalSignature:     "hmac-sha256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069",
		ComplianceStatement:  "Permanent erasure completed under GDPR Article 17.",
	}

	pdfBytes, err := GenerateGDPRCertificatePDF(cert)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("xref")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("%%EOF")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("CERTIFICATE OF PERMANENT DATA ERASURE")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("GDPR ARTICLE 17")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("cert-gdpr-12345")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("Orange Cat Investments Data Protection Office")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("GDPR Article 17 - Right to Erasure")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("e3b0c44298fc1c149afbf4c8...")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("ca978112ca1bbdcafac231b3...")))
	assert.True(t, bytes.Contains(pdfBytes, []byte(`3 Session\(s\)`)))
	assert.True(t, bytes.Contains(pdfBytes, []byte(`7 Record\(s\)`)))
	assert.True(t, bytes.Contains(pdfBytes, []byte("core_invest.investment_portfolios, core_invest.customer_api_keys, core_invest.webhook_subscriptions")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("hmac-sha256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069")))
}

func TestGenerateGDPRCertificatePDF_Nil(t *testing.T) {
	pdfBytes, err := GenerateGDPRCertificatePDF(nil)
	assert.Error(t, err)
	assert.Nil(t, pdfBytes)
	assert.EqualError(t, err, "certificate data is nil")
}
