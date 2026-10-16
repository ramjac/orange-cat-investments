package core_invest_test

import (
	"context"
	"testing"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePasswordComplexity(t *testing.T) {
	// Too short
	assert.Error(t, core_invest_svc.ValidatePasswordComplexity("Pass1!"))
	// Missing uppercase
	assert.Error(t, core_invest_svc.ValidatePasswordComplexity("password123!"))
	// Missing lowercase
	assert.Error(t, core_invest_svc.ValidatePasswordComplexity("PASSWORD123!"))
	// Missing number
	assert.Error(t, core_invest_svc.ValidatePasswordComplexity("Password!@#"))
	// Missing special char
	assert.Error(t, core_invest_svc.ValidatePasswordComplexity("Password123"))
	// Valid password
	assert.NoError(t, core_invest_svc.ValidatePasswordComplexity("OrangeCat2026!"))
}

func TestCustomerLifecycle_Register_And_Onboard(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)

	// Invalid registration: bad email
	_, err := svc.RegisterCustomer(context.Background(), core_invest_svc.CustomerRegistrationRequest{
		FullName: "Arthur Test",
		Email:    "bad-email",
		Password: "ValidPass123!",
	})
	assert.Error(t, err)

	// Invalid registration: weak password
	_, err = svc.RegisterCustomer(context.Background(), core_invest_svc.CustomerRegistrationRequest{
		FullName: "Arthur Test",
		Email:    "arthur.test@example.com",
		Password: "weak",
	})
	assert.Error(t, err)

	// Valid registration
	res, err := svc.RegisterCustomer(context.Background(), core_invest_svc.CustomerRegistrationRequest{
		FullName:   "Jane Doe",
		Email:      "jane.doe@example.com",
		Password:   "OrangeCat2026!",
		TargetFund: "OCI Alpha Growth Fund",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.CustomerID)
	assert.NotEmpty(t, res.PortfolioID)
	assert.True(t, res.EmailVerified)
	assert.Equal(t, 0.00, res.StartingBalance)

	// Verify portfolio created in repo
	p, err := svc.GetPortfolio(context.Background(), res.CustomerID)
	require.NoError(t, err)
	assert.Equal(t, 0.00, p.TotalBalanceUSD)
	assert.Equal(t, 0.00, p.CashBalanceUSD)
	assert.Equal(t, "active", p.Status)
}

func TestCustomerLifecycle_CloseAndLiquidateAccount(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	broker := &core_invest_svc.MockBrokerClient{}
	bank := &core_invest_svc.MockBankClient{AllowTransfer: true}
	svc := core_invest_svc.NewServiceWithIntegrations(repo, broker, bank)

	// Liquidate Arthur's seeded portfolio
	res, err := svc.CloseAndLiquidateAccount(context.Background(), core_invest_svc.CloseAccountRequest{
		CustomerID: "cust-longterm-arthur",
	})
	require.NoError(t, err)
	assert.Equal(t, "closed", res.AccountStatus)
	assert.True(t, res.TransferredToBankUSD > 0)
	assert.NotEmpty(t, res.TransferID)

	// Verify portfolio status updated to closed with 0 balance
	p, err := svc.GetPortfolio(context.Background(), "cust-longterm-arthur")
	require.NoError(t, err)
	assert.Equal(t, "closed", p.Status)
	assert.Equal(t, 0.00, p.TotalBalanceUSD)
	assert.Equal(t, 0.00, p.CashBalanceUSD)
}

func TestCustomerLifecycle_GDPRErasure(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)

	cert, err := svc.ExecuteGDPRErasure(context.Background(), "cust-active-chloe", "chloe@oci.local")
	require.NoError(t, err)
	assert.NotEmpty(t, cert.CertificateID)
	assert.NotEmpty(t, cert.DigitalSignature)
	assert.NotEmpty(t, cert.CustomerIDHash)
	assert.Contains(t, cert.LegalBasis, "GDPR Article 17")
	assert.Contains(t, cert.TablesScrubbed, "core_invest.investment_portfolios")

	// Verify portfolio anonymized and closed
	p, err := svc.GetPortfolio(context.Background(), "cust-active-chloe")
	require.NoError(t, err)
	assert.Equal(t, "closed", p.Status)
	assert.Contains(t, p.AccountName, "Anonymized Investor")
}

func TestCustomerLifecycle_StatementsAndTaxDocuments(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)

	// List statements
	stmts, err := svc.ListStatements(context.Background(), "cust-longterm-arthur")
	require.NoError(t, err)
	assert.NotEmpty(t, stmts)

	// Generate Monthly Statement
	created, pdfBytes, err := svc.GenerateMonthlyStatement(context.Background(), "cust-longterm-arthur", "", 2026, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.Equal(t, "monthly_summary", created.StatementType)

	// Generate Form 1099-B
	tax1099, pdf1099, err := svc.GenerateForm1099B(context.Background(), "cust-longterm-arthur", "", 2025)
	require.NoError(t, err)
	assert.NotEmpty(t, pdf1099)
	assert.Equal(t, "form_1099b", tax1099.StatementType)

	// Retrieve statement document
	retrieved, docBytes, err := svc.GetStatementDocument(context.Background(), "cust-longterm-arthur", created.StatementID)
	require.NoError(t, err)
	assert.NotEmpty(t, docBytes)
	assert.Equal(t, created.StatementID, retrieved.StatementID)
}

func TestCustomerLifecycle_GenerateMonthlyStatement_HoldingsFromDatabase(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)
	ctx := context.Background()

	// 1. Verify holdings pulled from repo for Arthur's portfolio
	stmt, pdfBytes, err := svc.GenerateMonthlyStatement(ctx, "cust-longterm-arthur", "port-cust-longterm-arthur", 2026, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.Equal(t, "monthly_summary", stmt.StatementType)

	// Retrieve Arthur's holdings directly from repo to compare
	dbHoldings, err := repo.GetHoldingsByPortfolioID(ctx, "port-cust-longterm-arthur")
	require.NoError(t, err)
	assert.Len(t, dbHoldings, 4)

	// 2. Custom portfolio with upserted database holdings
	customPortID := "port-custom-investor-01"
	customCustID := "cust-custom-01"
	_, err = repo.CreatePortfolio(ctx, &core_invest_repo.Portfolio{
		PortfolioID:     customPortID,
		CustomerID:      customCustID,
		AccountName:     "Custom Alpha Hedge",
		TotalBalanceUSD: 50000.00,
		CashBalanceUSD:  10000.00,
		Status:          "active",
	})
	require.NoError(t, err)

	_, err = repo.UpsertHolding(ctx, &core_invest_repo.PortfolioHolding{
		PortfolioID:       customPortID,
		Symbol:            "GOOGL",
		Quantity:          50.0,
		CostBasisUSD:      8000.00,
		AvgPriceUSD:       160.00,
		CurrentPriceUSD:   180.00,
		MarketValueUSD:    9000.00,
		UnrealizedGainUSD: 1000.00,
		UnrealizedGainPct: 12.50,
	})
	require.NoError(t, err)

	_, err = repo.UpsertHolding(ctx, &core_invest_repo.PortfolioHolding{
		PortfolioID:       customPortID,
		Symbol:            "AMZN",
		Quantity:          40.0,
		CostBasisUSD:      7200.00,
		AvgPriceUSD:       180.00,
		CurrentPriceUSD:   200.00,
		MarketValueUSD:    8000.00,
		UnrealizedGainUSD: 800.00,
		UnrealizedGainPct: 11.11,
	})
	require.NoError(t, err)

	customStmt, customPDF, err := svc.GenerateMonthlyStatement(ctx, customCustID, customPortID, 2026, 4)
	require.NoError(t, err)
	assert.NotEmpty(t, customPDF)
	assert.Equal(t, "monthly_summary", customStmt.StatementType)

	// Verify custom holdings exist in the database for the portfolio
	pulledHoldings, err := repo.GetHoldingsByPortfolioID(ctx, customPortID)
	require.NoError(t, err)
	require.Len(t, pulledHoldings, 2)
	assert.Equal(t, "GOOGL", pulledHoldings[0].Symbol)
	assert.Equal(t, 50.0, pulledHoldings[0].Quantity)
	assert.Equal(t, "AMZN", pulledHoldings[1].Symbol)
	assert.Equal(t, 40.0, pulledHoldings[1].Quantity)

	// 3. Portfolio with zero holdings
	emptyPortID := "port-empty-investor"
	emptyCustID := "cust-empty-01"
	_, err = repo.CreatePortfolio(ctx, &core_invest_repo.Portfolio{
		PortfolioID:     emptyPortID,
		CustomerID:      emptyCustID,
		AccountName:     "Zero Balance Account",
		TotalBalanceUSD: 0.00,
		CashBalanceUSD:  0.00,
		Status:          "active",
	})
	require.NoError(t, err)

	emptyStmt, emptyPDF, err := svc.GenerateMonthlyStatement(ctx, emptyCustID, emptyPortID, 2026, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, emptyPDF)
	assert.Equal(t, "monthly_summary", emptyStmt.StatementType)

	emptyHoldings, err := repo.GetHoldingsByPortfolioID(ctx, emptyPortID)
	require.NoError(t, err)
	assert.Empty(t, emptyHoldings)
}

func TestCustomerLifecycle_GenerateForm1099B_TaxLotsFromDatabase(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)
	ctx := context.Background()

	// 1. Arthur's seeded tax lots for 2025
	stmt, pdfBytes, err := svc.GenerateForm1099B(ctx, "cust-longterm-arthur", "port-cust-longterm-arthur", 2025)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.True(t, len(pdfBytes) > 100)
	assert.Equal(t, "%PDF-1.4", string(pdfBytes[:8]))
	assert.Equal(t, "form_1099b", stmt.StatementType)
	assert.Equal(t, 2025, stmt.TaxYear)
	assert.Equal(t, 23750.00, stmt.TotalProceedsUSD)
	assert.Equal(t, 4950.00, stmt.NetGainUSD)

	// Verify tax lots directly in repo
	lots, err := repo.GetTaxLotSalesByPortfolioID(ctx, "port-cust-longterm-arthur", 2025)
	require.NoError(t, err)
	require.Len(t, lots, 3)
	assert.Equal(t, "NVDA", lots[0].Symbol)
	assert.Equal(t, 50.0, lots[0].Quantity)
	assert.Equal(t, 6940.00, lots[0].GrossProceedsUSD)
	assert.Equal(t, "AAPL", lots[1].Symbol)
	assert.Equal(t, 35.0, lots[1].Quantity)
	assert.Equal(t, 7850.00, lots[1].GrossProceedsUSD)
	assert.Equal(t, "MSFT", lots[2].Symbol)
	assert.Equal(t, 20.0, lots[2].Quantity)
	assert.Equal(t, 8960.00, lots[2].GrossProceedsUSD)

	// 2. Custom portfolio with newly created tax lots
	customPortID := "port-dynamic-tax-cust-01"
	customCustID := "cust-dynamic-tax-01"
	_, err = repo.CreatePortfolio(ctx, &core_invest_repo.Portfolio{
		PortfolioID:     customPortID,
		CustomerID:      customCustID,
		AccountName:     "Dynamic Tax Alpha Fund",
		TotalBalanceUSD: 100000.00,
		CashBalanceUSD:  20000.00,
		Status:          "active",
	})
	require.NoError(t, err)

	_, err = repo.CreateTaxLotSale(ctx, &core_invest_repo.TaxLotSale{
		PortfolioID:            customPortID,
		Symbol:                 "AMZN",
		Description:            "25 shs AMZN",
		Quantity:               25.0,
		DateAcquired:           "2025-01-15",
		DateSold:               "2025-06-20",
		GrossProceedsUSD:       5000.00,
		CostBasisUSD:           4000.00,
		WashSaleLossDisallowed: 0.00,
		RealizedGainLossUSD:    1000.00,
		IsShortTerm:            true,
		TaxYear:                2025,
	})
	require.NoError(t, err)

	_, err = repo.CreateTaxLotSale(ctx, &core_invest_repo.TaxLotSale{
		PortfolioID:            customPortID,
		Symbol:                 "GOOGL",
		Description:            "15 shs GOOGL",
		Quantity:               15.0,
		DateAcquired:           "2024-02-10",
		DateSold:               "2025-07-18",
		GrossProceedsUSD:       3000.00,
		CostBasisUSD:           2200.00,
		WashSaleLossDisallowed: 0.00,
		RealizedGainLossUSD:    800.00,
		IsShortTerm:            false,
		TaxYear:                2025,
	})
	require.NoError(t, err)

	customStmt, customPDF, err := svc.GenerateForm1099B(ctx, customCustID, customPortID, 2025)
	require.NoError(t, err)
	assert.NotEmpty(t, customPDF)
	assert.Equal(t, "%PDF-1.4", string(customPDF[:8]))
	assert.Equal(t, 8000.00, customStmt.TotalProceedsUSD)
	assert.Equal(t, 1800.00, customStmt.NetGainUSD)

	// 3. Tax year with zero tax lots
	emptyStmt, emptyPDF, err := svc.GenerateForm1099B(ctx, "cust-longterm-arthur", "port-cust-longterm-arthur", 2024)
	require.NoError(t, err)
	assert.NotEmpty(t, emptyPDF)
	assert.Equal(t, "%PDF-1.4", string(emptyPDF[:8]))
	assert.Equal(t, 0.00, emptyStmt.TotalProceedsUSD)
	assert.Equal(t, 0.00, emptyStmt.NetGainUSD)

	// 4. Chloe's seeded tax lots for 2025
	chloeStmt, chloePDF, err := svc.GenerateForm1099B(ctx, "cust-active-chloe", "port-cust-active-chloe", 2025)
	require.NoError(t, err)
	assert.NotEmpty(t, chloePDF)
	assert.Equal(t, 18212.80, chloeStmt.TotalProceedsUSD)
	assert.Equal(t, 4048.80, chloeStmt.NetGainUSD)
}

