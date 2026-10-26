package core_invest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

func (s *coreInvestService) GenerateForm1099B(ctx context.Context, customerID, portfolioID string, taxYear int) (*core_invest.InvestmentStatement, []byte, error) {
	if customerID == "" {
		return nil, nil, errors.New("customer_id is required")
	}
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("port-%s", customerID)
	}
	if taxYear < 2000 || taxYear > 2100 {
		return nil, nil, fmt.Errorf("invalid tax year: %d", taxYear)
	}

	p, _ := s.GetPortfolio(ctx, customerID)
	customerName := customerID
	if p != nil && p.CustomerName != "" {
		customerName = p.CustomerName
	}

	lots, err := s.repo.GetTaxLotSalesByPortfolioID(ctx, portfolioID, taxYear)
	if err != nil || len(lots) == 0 {
		if lots2, err2 := s.repo.GetTaxLotSalesByPortfolioID(ctx, customerID, taxYear); err2 == nil && len(lots2) > 0 {
			lots = lots2
		}
	}

	var transactions []reporting.Form1099BTransaction
	var totalGrossProceeds, totalCostBasis, totalRealizedGain float64
	var shortTermGain, longTermGain float64

	for _, lot := range lots {
		desc := lot.Description
		if desc == "" {
			desc = fmt.Sprintf("%.0f shs %s", lot.Quantity, lot.Symbol)
		}
		gain := lot.RealizedGainLossUSD
		if gain == 0.0 && (lot.GrossProceedsUSD != 0.0 || lot.CostBasisUSD != 0.0) {
			gain = lot.GrossProceedsUSD - lot.CostBasisUSD
		}
		tx := reporting.Form1099BTransaction{
			Description:            desc,
			Symbol:                 lot.Symbol,
			Quantity:               lot.Quantity,
			DateAcquired:           lot.DateAcquired,
			DateSold:               lot.DateSold,
			GrossProceedsUSD:       lot.GrossProceedsUSD,
			CostBasisUSD:           lot.CostBasisUSD,
			WashSaleLossDisallowed: lot.WashSaleLossDisallowed,
			RealizedGainLossUSD:    gain,
			IsShortTerm:            lot.IsShortTerm,
		}
		transactions = append(transactions, tx)
		totalGrossProceeds += lot.GrossProceedsUSD
		totalCostBasis += lot.CostBasisUSD
		totalRealizedGain += gain
		if lot.IsShortTerm {
			shortTermGain += gain
		} else {
			longTermGain += gain
		}
	}

	if transactions == nil {
		transactions = []reporting.Form1099BTransaction{}
	}

	stmtID := fmt.Sprintf("stmt-%s-1099b-%d", portfolioID, taxYear)
	fileName := fmt.Sprintf("form1099b-%s-%d.pdf", portfolioID, taxYear)

	custSum := sha256.Sum256([]byte(customerID))
	custHex := hex.EncodeToString(custSum[:])
	recipientTIN := fmt.Sprintf("***-**-%s", custHex[:4])

	formStmt := &reporting.Form1099BStatement{
		StatementID:           stmtID,
		PortfolioID:           portfolioID,
		CustomerID:            customerID,
		CustomerName:          customerName,
		TaxYear:               taxYear,
		PayerName:             "Orange Cat Investments LLC",
		PayerTIN:              "12-9847120",
		RecipientTIN:          recipientTIN,
		TotalGrossProceedsUSD: totalGrossProceeds,
		TotalCostBasisUSD:     totalCostBasis,
		TotalRealizedGainUSD:  totalRealizedGain,
		ShortTermGainUSD:      shortTermGain,
		LongTermGainUSD:       longTermGain,
		Transactions:          transactions,
		CreatedAt:             time.Now().UTC(),
	}

	pdfBytes, err := reporting.GenerateForm1099BPDF(formStmt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile form 1099-b pdf: %w", err)
	}

	record := &core_invest.InvestmentStatement{
		StatementID:      stmtID,
		PortfolioID:      portfolioID,
		CustomerID:       customerID,
		StatementType:    "form_1099b",
		Period:           fmt.Sprintf("Tax Year %d", taxYear),
		TaxYear:          taxYear,
		Title:            fmt.Sprintf("IRS Form 1099-B Tax Report - %d", taxYear),
		FileName:         fileName,
		TotalProceedsUSD: formStmt.TotalGrossProceedsUSD,
		NetGainUSD:       formStmt.TotalRealizedGainUSD,
		Status:           "available",
		CreatedAt:        formStmt.CreatedAt,
	}

	created, _ := s.repo.CreateInvestmentStatement(ctx, record)
	if created == nil {
		created = record
	}

	return created, pdfBytes, nil
}
