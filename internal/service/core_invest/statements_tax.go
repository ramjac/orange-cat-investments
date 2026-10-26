package core_invest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

func (s *coreInvestService) GenerateStatement(ctx context.Context, portfolioID string, year int) (map[string]interface{}, error) {
	if portfolioID == "" {
		return nil, errors.New("portfolio_id is required")
	}
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("invalid tax statement year %d: year must be between 2000 and 2100", year)
	}

	statementID := fmt.Sprintf("stmt-%s-%d", portfolioID, year)
	fileName := fmt.Sprintf("form1099b-%s-%d.pdf", portfolioID, year)
	documentURL := fmt.Sprintf("https://statements.oci.local/documents/%s/%d/%s", portfolioID, year, fileName)

	accountName := "OCI Alpha Growth Portfolio"
	totalVal := 128450.75

	return map[string]interface{}{
		"statement_id":        statementID,
		"portfolio_id":        portfolioID,
		"account_name":        accountName,
		"year":                year,
		"tax_year":            year,
		"document_name":       fileName,
		"form_1099b_url":      documentURL,
		"total_proceeds_usd":  totalVal,
		"sharpe_ratio":        2.14,
		"sortino_ratio":       3.08,
		"performance_summary": fmt.Sprintf("Feline behavioral alpha tax statement for portfolio %s, tax year %d", portfolioID, year),
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *coreInvestService) ListStatements(ctx context.Context, customerID string) ([]*core_invest.InvestmentStatement, error) {
	if customerID == "" {
		return nil, errors.New("customer_id is required")
	}
	return s.repo.ListInvestmentStatements(ctx, customerID)
}

func (s *coreInvestService) GetStatement(ctx context.Context, statementID string) (*core_invest.InvestmentStatement, error) {
	if statementID == "" {
		return nil, errors.New("statement_id is required")
	}
	return s.repo.GetInvestmentStatementByID(ctx, statementID)
}

type portfolioAccountInfo struct {
	portfolioID  string
	customerName string
	accountName  string
	totalBal     float64
	cashBal      float64
	portfolio    *core_invest.Portfolio
}

func (s *coreInvestService) resolvePortfolioAndAccountInfo(ctx context.Context, customerID, portfolioID string) portfolioAccountInfo {
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("port-%s", customerID)
	}

	p, _ := s.GetPortfolio(ctx, customerID)
	if p == nil && portfolioID != "" {
		p, _ = s.repo.GetPortfolioByID(ctx, portfolioID)
	}
	if p != nil && p.PortfolioID != "" {
		portfolioID = p.PortfolioID
	}

	customerName := customerID
	accountName := "OCI Investment Portfolio"
	var totalBal float64
	var cashBal float64
	if p != nil {
		if p.CustomerName != "" {
			customerName = p.CustomerName
		}
		if p.AccountName != "" {
			accountName = p.AccountName
		}
		totalBal = p.TotalBalanceUSD
		cashBal = p.CashBalanceUSD
	}

	return portfolioAccountInfo{
		portfolioID:  portfolioID,
		customerName: customerName,
		accountName:  accountName,
		totalBal:     totalBal,
		cashBal:      cashBal,
		portfolio:    p,
	}
}

type statementHoldingsSummary struct {
	holdings            []reporting.StatementHolding
	totalMarketValue    float64
	totalCostBasis      float64
	totalUnrealizedGain float64
}

func (s *coreInvestService) fetchAndBuildStatementHoldings(ctx context.Context, portfolioID string, p *core_invest.Portfolio) statementHoldingsSummary {
	dbHoldings, err := s.repo.GetHoldingsByPortfolioID(ctx, portfolioID)
	if (err != nil || len(dbHoldings) == 0) && p != nil && p.CustomerID != "" && p.CustomerID != portfolioID {
		dbHoldings, _ = s.repo.GetHoldingsByPortfolioID(ctx, p.CustomerID)
	}

	statementHoldings := make([]reporting.StatementHolding, 0)
	var totalHoldingsMarketValue float64
	var totalHoldingsCostBasis float64
	var totalHoldingsUnrealizedGain float64

	if len(dbHoldings) > 0 {
		for _, h := range dbHoldings {
			costBasis := h.CostBasisUSD
			if costBasis == 0 && h.AvgPriceUSD > 0 {
				costBasis = h.Quantity * h.AvgPriceUSD
			}
			mv := h.MarketValueUSD
			if mv == 0 && h.CurrentPriceUSD > 0 {
				mv = h.Quantity * h.CurrentPriceUSD
			}
			gain := h.UnrealizedGainUSD
			if gain == 0 && mv > 0 && costBasis > 0 {
				gain = mv - costBasis
			}
			gainPct := h.UnrealizedGainPct
			if gainPct == 0 && costBasis > 0 {
				gainPct = (gain / costBasis) * 100.0
			}

			statementHoldings = append(statementHoldings, reporting.StatementHolding{
				Symbol:            h.Symbol,
				Quantity:          h.Quantity,
				CostBasisUSD:      costBasis,
				CurrentPriceUSD:   h.CurrentPriceUSD,
				MarketValueUSD:    mv,
				UnrealizedGainUSD: gain,
				UnrealizedGainPct: gainPct,
			})
			totalHoldingsMarketValue += mv
			totalHoldingsCostBasis += costBasis
			totalHoldingsUnrealizedGain += gain
		}
	} else if p != nil && len(p.Holdings) > 0 {
		for _, h := range p.Holdings {
			symbol, _ := h["symbol"].(string)
			if symbol == "" {
				continue
			}
			qty := parseNumeric(h["quantity"])
			avgPrice := parseNumeric(h["avg_price"])
			currPrice := parseNumeric(h["current_price"])
			mv := parseNumeric(h["market_value"])
			if mv == 0 && currPrice > 0 {
				mv = qty * currPrice
			}
			costBasis := parseNumeric(h["cost_basis"])
			if costBasis == 0 && avgPrice > 0 {
				costBasis = qty * avgPrice
			}
			gain := parseNumeric(h["unrealized_gain"])
			if gain == 0 && mv > 0 && costBasis > 0 {
				gain = mv - costBasis
			}
			gainPct := parseNumeric(h["unrealized_gain_pct"])
			if gainPct == 0 && costBasis > 0 {
				gainPct = (gain / costBasis) * 100.0
			}

			statementHoldings = append(statementHoldings, reporting.StatementHolding{
				Symbol:            symbol,
				Quantity:          qty,
				CostBasisUSD:      costBasis,
				CurrentPriceUSD:   currPrice,
				MarketValueUSD:    mv,
				UnrealizedGainUSD: gain,
				UnrealizedGainPct: gainPct,
			})
			totalHoldingsMarketValue += mv
			totalHoldingsCostBasis += costBasis
			totalHoldingsUnrealizedGain += gain
		}
	}

	return statementHoldingsSummary{
		holdings:            statementHoldings,
		totalMarketValue:    totalHoldingsMarketValue,
		totalCostBasis:      totalHoldingsCostBasis,
		totalUnrealizedGain: totalHoldingsUnrealizedGain,
	}
}

type monthlyStatementMetrics struct {
	openingBal     float64
	unrealizedGain float64
	grossReturnPct float64
}

func calculateMonthlyMetrics(totalBal, totalHoldingsUnrealizedGain, totalHoldingsCostBasis float64) monthlyStatementMetrics {
	openingBal := totalBal * 0.85
	if totalBal > totalHoldingsUnrealizedGain && totalHoldingsUnrealizedGain > 0 {
		openingBal = totalBal - totalHoldingsUnrealizedGain
	}
	unrealizedGain := totalHoldingsUnrealizedGain
	if unrealizedGain == 0 && totalBal > 0 {
		unrealizedGain = totalBal * 0.15
	}
	grossReturnPct := 28.45
	if totalHoldingsCostBasis > 0 {
		grossReturnPct = (totalHoldingsUnrealizedGain / totalHoldingsCostBasis) * 100.0
	} else if openingBal > 0 {
		grossReturnPct = ((totalBal - openingBal) / openingBal) * 100.0
	}

	return monthlyStatementMetrics{
		openingBal:     openingBal,
		unrealizedGain: unrealizedGain,
		grossReturnPct: grossReturnPct,
	}
}

func (s *coreInvestService) GenerateMonthlyStatement(ctx context.Context, customerID, portfolioID string, year, month int) (*core_invest.InvestmentStatement, []byte, error) {
	if customerID == "" {
		return nil, nil, errors.New("customer_id is required")
	}
	if year < 2000 || year > 2100 {
		return nil, nil, fmt.Errorf("invalid year: %d", year)
	}
	if month < 1 || month > 12 {
		return nil, nil, fmt.Errorf("invalid month: %d", month)
	}

	info := s.resolvePortfolioAndAccountInfo(ctx, customerID, portfolioID)
	portfolioID = info.portfolioID
	customerName := info.customerName
	accountName := info.accountName
	totalBal := info.totalBal
	cashBal := info.cashBal
	p := info.portfolio

	holdingsSummary := s.fetchAndBuildStatementHoldings(ctx, portfolioID, p)
	statementHoldings := holdingsSummary.holdings
	totalHoldingsMarketValue := holdingsSummary.totalMarketValue
	totalHoldingsCostBasis := holdingsSummary.totalCostBasis
	totalHoldingsUnrealizedGain := holdingsSummary.totalUnrealizedGain

	if totalBal == 0 && (totalHoldingsMarketValue > 0 || cashBal > 0) {
		totalBal = totalHoldingsMarketValue + cashBal
	}

	monthNames := []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	periodStr := fmt.Sprintf("%s %d", monthNames[month], year)
	stmtID := fmt.Sprintf("stmt-%s-%d-%02d", portfolioID, year, month)
	fileName := fmt.Sprintf("monthly-statement-%s-%d-%02d.pdf", portfolioID, year, month)

	metrics := calculateMonthlyMetrics(totalBal, totalHoldingsUnrealizedGain, totalHoldingsCostBasis)

	monthlyReport := &reporting.CustomerMonthlyStatement{
		StatementID:               stmtID,
		PortfolioID:               portfolioID,
		CustomerID:                customerID,
		CustomerName:              customerName,
		AccountName:               accountName,
		Period:                    periodStr,
		TaxYear:                   year,
		OpeningBalanceUSD:         metrics.openingBal,
		EndingBalanceUSD:          totalBal,
		CashBalanceUSD:            cashBal,
		NetContributionsUSD:       5000.00,
		TotalRealizedGainUSD:      12450.00,
		TotalUnrealizedGainUSD:    metrics.unrealizedGain,
		GrossReturnPct:            metrics.grossReturnPct,
		SharpeRatio:               2.14,
		SortinoRatio:              3.08,
		ZoomieAlphaAttributionUSD: 8420.50,
		Holdings:                  statementHoldings,
		CreatedAt:                 time.Now().UTC(),
	}

	pdfBytes, err := reporting.GenerateMonthlyStatementPDF(monthlyReport)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile monthly statement pdf: %w", err)
	}

	record := &core_invest.InvestmentStatement{
		StatementID:      stmtID,
		PortfolioID:      portfolioID,
		CustomerID:       customerID,
		StatementType:    "monthly_summary",
		Period:           periodStr,
		TaxYear:          year,
		Title:            fmt.Sprintf("Monthly Portfolio Summary - %s", periodStr),
		FileName:         fileName,
		TotalProceedsUSD: totalBal,
		NetGainUSD:       monthlyReport.TotalRealizedGainUSD,
		Status:           "available",
		CreatedAt:        monthlyReport.CreatedAt,
	}

	created, _ := s.repo.CreateInvestmentStatement(ctx, record)
	if created == nil {
		created = record
	}

	return created, pdfBytes, nil
}

func (s *coreInvestService) GetStatementDocument(ctx context.Context, customerID, statementID string) (*core_invest.InvestmentStatement, []byte, error) {
	if statementID == "" {
		return nil, nil, errors.New("statement_id is required")
	}

	stmt, err := s.repo.GetInvestmentStatementByID(ctx, statementID)
	if err != nil || stmt == nil {
		return s.GenerateMonthlyStatement(ctx, customerID, "", 2026, 3)
	}

	if stmt.StatementType == "form_1099b" {
		year := stmt.TaxYear
		if year == 0 {
			year = 2025
		}
		_, pdfBytes, err := s.GenerateForm1099B(ctx, customerID, stmt.PortfolioID, year)
		return stmt, pdfBytes, err
	}

	year := stmt.TaxYear
	if year == 0 {
		year = 2026
	}
	_, pdfBytes, err := s.GenerateMonthlyStatement(ctx, customerID, stmt.PortfolioID, year, 3)
	return stmt, pdfBytes, err
}

func parseNumeric(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
