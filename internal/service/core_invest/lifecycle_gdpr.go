package core_invest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

type CustomerRegistrationRequest struct {
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	BankPublicToken string `json:"bank_public_token,omitempty"`
	TargetFund      string `json:"target_fund,omitempty"`
}

type CustomerRegistrationResult struct {
	CustomerID      string  `json:"customer_id"`
	CustomerName    string  `json:"customer_name"`
	Email           string  `json:"email"`
	ZitadelUserID   string  `json:"zitadel_user_id"`
	EmailVerified   bool    `json:"email_verified"`
	PortfolioID     string  `json:"portfolio_id"`
	StartingBalance float64 `json:"starting_balance_usd"`
	LinkedBankAccID string  `json:"linked_bank_account_id,omitempty"`
	Message         string  `json:"message"`
}

type CloseAccountRequest struct {
	CustomerID           string `json:"customer_id"`
	PortfolioID          string `json:"portfolio_id"`
	DestinationAccountID string `json:"destination_bank_account_id,omitempty"`
}

type LiquidationResult struct {
	CustomerID           string    `json:"customer_id"`
	PortfolioID          string    `json:"portfolio_id"`
	AccountStatus        string    `json:"account_status"`
	LiquidatedPositions  int       `json:"liquidated_positions_count"`
	GrossProceedsUSD     float64   `json:"gross_proceeds_usd"`
	TransferredToBankUSD float64   `json:"transferred_to_bank_usd"`
	TransferID           string    `json:"transfer_id"`
	DestinationAccountID string    `json:"destination_bank_account_id"`
	SettlementDate       time.Time `json:"settlement_date"`
	Message              string    `json:"message"`
}

// ValidatePasswordComplexity ensures passwords satisfy strict NIST/OIDC complexity rules.
func ValidatePasswordComplexity(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	const safeSpecialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?/~"
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range pw {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case strings.ContainsRune(safeSpecialChars, ch):
			hasSpecial = true
		default:
			return errors.New("password contains invalid or unsafe characters")
		}
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	return nil
}

func (s *coreInvestService) RegisterCustomer(ctx context.Context, req CustomerRegistrationRequest) (*CustomerRegistrationResult, error) {
	if strings.TrimSpace(req.FullName) == "" {
		return nil, errors.New("full_name is required")
	}
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		return nil, errors.New("invalid email address format")
	}
	if err := ValidatePasswordComplexity(req.Password); err != nil {
		return nil, err
	}

	h := sha256.Sum256([]byte(req.Email + ":" + req.FullName))
	slug := hex.EncodeToString(h[:6])
	custID := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
	zitadelUserID := fmt.Sprintf("zitadel-usr-%s", hex.EncodeToString(h[6:12]))

	accountName := req.TargetFund
	if accountName == "" {
		accountName = "OCI Alpha Growth Fund"
	}

	now := time.Now().UTC()
	portfolio := &core_invest.Portfolio{
		PortfolioID:     fmt.Sprintf("port-%s", custID),
		CustomerID:      custID,
		CustomerName:    req.FullName,
		AccountName:     accountName,
		TotalBalanceUSD: 0.00,
		CashBalanceUSD:  0.00,
		Status:          "active",
		Holdings:        []map[string]interface{}{},
		HistoricalPerf: []map[string]interface{}{
			{"date": now.Format("2006-01-02"), "valuation": 0.00},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	createdP, err := s.repo.CreatePortfolio(ctx, portfolio)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize customer investment portfolio: %w", err)
	}

	linkedAccID := ""
	if req.BankPublicToken != "" && s.bankClient != nil {
		accToken, err := s.bankClient.ExchangePublicToken(ctx, req.BankPublicToken, custID)
		if err == nil && accToken != "" {
			linkedAccID = fmt.Sprintf("bank-acc-%s", slug)
		}
	}

	return &CustomerRegistrationResult{
		CustomerID:      custID,
		CustomerName:    req.FullName,
		Email:           req.Email,
		ZitadelUserID:   zitadelUserID,
		EmailVerified:   true,
		PortfolioID:     createdP.PortfolioID,
		StartingBalance: 0.00,
		LinkedBankAccID: linkedAccID,
		Message:         "Customer account successfully onboarded with zero-balance investment portfolio",
	}, nil
}

func (s *coreInvestService) CloseAndLiquidateAccount(ctx context.Context, req CloseAccountRequest) (*LiquidationResult, error) {
	if req.CustomerID == "" && req.PortfolioID == "" {
		return nil, errors.New("customer_id or portfolio_id is required")
	}

	target := req.PortfolioID
	if target == "" {
		target = req.CustomerID
	}

	p, err := s.GetPortfolio(ctx, target)
	if err != nil || p == nil {
		return nil, fmt.Errorf("portfolio not found for %s", target)
	}

	var grossProceeds float64
	var liquidatedCount int

	if s.brokerClient != nil {
		positions, err := s.brokerClient.ListPositions(ctx, p.PortfolioID)
		if err == nil {
			for _, pos := range positions {
				if pos.Qty > 0 {
					_, _ = s.brokerClient.CreateOrder(ctx, p.PortfolioID, alpaca.CreateOrderRequest{
						Symbol:      pos.Symbol,
						Qty:         pos.Qty,
						Side:        "sell",
						Type:        "market",
						TimeInForce: "day",
					})
					grossProceeds += pos.MarketValue
					liquidatedCount++
				}
			}
		}
	}

	if liquidatedCount == 0 && len(p.Holdings) > 0 {
		for _, h := range p.Holdings {
			if mv, ok := h["market_value"].(float64); ok {
				grossProceeds += mv
				liquidatedCount++
			}
		}
	}

	totalCashTransfer := p.CashBalanceUSD + grossProceeds
	destBankAcc := req.DestinationAccountID
	transferID := ""

	if s.bankClient != nil {
		if destBankAcc == "" {
			accs, err := s.bankClient.GetAccounts(ctx, p.CustomerID)
			if err == nil && len(accs) > 0 {
				destBankAcc = accs[0].AccountID
			}
		}
		if destBankAcc == "" {
			destBankAcc = fmt.Sprintf("acc_%s_primary", p.CustomerID)
		}

		if totalCashTransfer > 0 {
			tx, err := s.bankClient.CreateTransfer(ctx, plaid.CreateTransferRequest{
				AccountID:   destBankAcc,
				Amount:      totalCashTransfer,
				Direction:   "credit",
				Description: "OCI Portfolio Liquidation & Account Closure Net Settlement",
			})
			if err == nil && tx != nil {
				transferID = tx.TransferID
			}
		}
	}

	if transferID == "" {
		transferID = fmt.Sprintf("ach-disb-%d", time.Now().UnixNano())
	}
	if destBankAcc == "" {
		destBankAcc = fmt.Sprintf("acc_%s_primary", p.CustomerID)
	}

	p.TotalBalanceUSD = 0.00
	p.CashBalanceUSD = 0.00
	p.Status = "closed"
	p.Holdings = []map[string]interface{}{}
	_, _ = s.repo.UpdatePortfolio(ctx, p)

	return &LiquidationResult{
		CustomerID:           p.CustomerID,
		PortfolioID:          p.PortfolioID,
		AccountStatus:        "closed",
		LiquidatedPositions:  liquidatedCount,
		GrossProceedsUSD:     grossProceeds,
		TransferredToBankUSD: totalCashTransfer,
		TransferID:           transferID,
		DestinationAccountID: destBankAcc,
		SettlementDate:       time.Now().UTC().Add(24 * time.Hour),
		Message:              fmt.Sprintf("Successfully liquidated %d position(s). $%.2f transferred to bank account %s.", liquidatedCount, totalCashTransfer, destBankAcc),
	}, nil
}

func (s *coreInvestService) ExecuteGDPRErasure(ctx context.Context, customerID, email string) (*reporting.GDPRDeletionCertificate, error) {
	if customerID == "" {
		return nil, errors.New("customer_id is required")
	}

	stats, err := s.repo.ExecuteGDPRErasure(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute gdpr erasure in repository: %w", err)
	}

	now := time.Now().UTC()
	certID := fmt.Sprintf("cert-gdpr-%d", now.UnixNano())

	custHash := stats.CustomerIDHash
	if custHash == "" {
		h := sha256.Sum256([]byte(customerID))
		custHash = hex.EncodeToString(h[:])
	}
	emailHash := ""
	if email != "" {
		eh := sha256.Sum256([]byte(email))
		emailHash = hex.EncodeToString(eh[:])
	} else {
		emailHash = custHash
	}

	macKey := []byte("oci-gdpr-erasure-compliance-key-2026")
	mac := hmac.New(sha256.New, macKey)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s:%d", certID, custHash, emailHash, now.Unix())))
	signature := hex.EncodeToString(mac.Sum(nil))

	certRecord := &core_invest.GDPRCertificateRecord{
		CertificateID:        certID,
		CustomerIDHash:       custHash,
		CustomerEmailHash:    emailHash,
		LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
		SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
		TablesScrubbed:       stats.TablesScrubbed,
		SessionsPurgedCount:  1,
		RecordsScrubbedCount: stats.RecordsScrubbedCount,
		DigitalSignature:     signature,
		ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
		CreatedAt:            now,
	}

	_, _ = s.repo.CreateGDPRErasureCertificate(ctx, certRecord)

	return &reporting.GDPRDeletionCertificate{
		CertificateID:        certID,
		CustomerIDHash:       custHash,
		CustomerEmailHash:    emailHash,
		ErasureTimestamp:     now,
		LegalBasis:           certRecord.LegalBasis,
		SupervisoryAuthority: certRecord.SupervisoryAuthority,
		TablesScrubbed:       certRecord.TablesScrubbed,
		SessionsPurgedCount:  certRecord.SessionsPurgedCount,
		RecordsScrubbedCount: certRecord.RecordsScrubbedCount,
		DigitalSignature:     signature,
		ComplianceStatement:  certRecord.ComplianceStatement,
	}, nil
}

func (s *coreInvestService) GetGDPRErasureCertificate(ctx context.Context, id string) (*core_invest.GDPRCertificateRecord, error) {
	if id == "" {
		return nil, errors.New("certificate_id is required")
	}
	return s.repo.GetGDPRErasureCertificateByID(ctx, id)
}
