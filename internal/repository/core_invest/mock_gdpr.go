package core_invest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func (m *mockRepository) CreateGDPRErasureCertificate(ctx context.Context, cert *GDPRCertificateRecord) (*GDPRCertificateRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if cert.CertificateID == "" {
		cert.CertificateID = fmt.Sprintf("cert-gdpr-%d", now.UnixNano())
	}
	cert.CreatedAt = now
	m.certificates[cert.CertificateID] = cert
	return cert, nil
}

func (m *mockRepository) GetGDPRErasureCertificateByID(ctx context.Context, id string) (*GDPRCertificateRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cert, ok := m.certificates[id]; ok {
		return cert, nil
	}
	return nil, fmt.Errorf("gdpr certificate not found: %s", id)
}

func (m *mockRepository) ExecuteGDPRErasure(ctx context.Context, customerID string) (*GDPRErasureStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h := sha256.Sum256([]byte(customerID))
	custHash := hex.EncodeToString(h[:])

	recordsPurged := 0
	tables := []string{}

	if keys, ok := m.apiKeys[customerID]; ok {
		recordsPurged += len(keys)
		delete(m.apiKeys, customerID)
		tables = append(tables, "core_invest.customer_api_keys")
	}

	if subs, ok := m.webhooks[customerID]; ok {
		recordsPurged += len(subs)
		delete(m.webhooks, customerID)
		tables = append(tables, "core_invest.webhook_subscriptions")
	}

	depCount := 0
	for id, dep := range m.deposits {
		if dep.CustomerID == customerID {
			delete(m.deposits, id)
			depCount++
		}
	}
	if depCount > 0 {
		recordsPurged += depCount
		tables = append(tables, "core_invest.recurring_deposits")
	}

	for id, p := range m.portfolios {
		if id == customerID || p.CustomerID == customerID || p.PortfolioID == customerID {
			p.AccountName = fmt.Sprintf("Anonymized Investor (%s)", custHash[:8])
			p.CustomerName = "Anonymized Subject"
			p.Status = "closed"
			p.TotalBalanceUSD = 0.00
			p.CashBalanceUSD = 0.00
			p.Holdings = []map[string]interface{}{}
			p.UpdatedAt = time.Now().UTC()
			delete(m.holdings, p.PortfolioID)
			delete(m.holdings, p.CustomerID)
			delete(m.holdings, id)
			delete(m.taxLots, p.PortfolioID)
			delete(m.taxLots, p.CustomerID)
			delete(m.taxLots, id)
			tables = append(tables, "core_invest.investment_portfolios", "core_invest.portfolio_holdings", "core_invest.tax_lot_sales")
		}
	}

	tables = append(tables, "ops.it_tickets", "ops.ticket_comments")

	return &GDPRErasureStats{
		CustomerIDHash:       custHash,
		CustomerEmailHash:    custHash,
		TablesScrubbed:       tables,
		RecordsScrubbedCount: recordsPurged,
	}, nil
}
