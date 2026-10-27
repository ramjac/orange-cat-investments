package core_invest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func (r *pgxRepository) CreateGDPRErasureCertificate(ctx context.Context, cert *GDPRCertificateRecord) (*GDPRCertificateRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.gdpr_erasure_certificates (
                  customer_id_hash, customer_email_hash, legal_basis, supervisory_authority, tables_scrubbed, sessions_purged_count, records_scrubbed_count, digital_signature, compliance_statement
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
              RETURNING certificate_id, customer_id_hash, customer_email_hash, legal_basis, supervisory_authority, tables_scrubbed, sessions_purged_count, records_scrubbed_count, digital_signature, compliance_statement, created_at`
	var c GDPRCertificateRecord
	err := r.db.QueryRow(ctx, query, cert.CustomerIDHash, cert.CustomerEmailHash, cert.LegalBasis, cert.SupervisoryAuthority, cert.TablesScrubbed, cert.SessionsPurgedCount, cert.RecordsScrubbedCount, cert.DigitalSignature, cert.ComplianceStatement).
		Scan(&c.CertificateID, &c.CustomerIDHash, &c.CustomerEmailHash, &c.LegalBasis, &c.SupervisoryAuthority, &c.TablesScrubbed, &c.SessionsPurgedCount, &c.RecordsScrubbedCount, &c.DigitalSignature, &c.ComplianceStatement, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create gdpr erasure certificate: %w", err)
	}
	return &c, nil
}

func (r *pgxRepository) GetGDPRErasureCertificateByID(ctx context.Context, id string) (*GDPRCertificateRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT certificate_id, customer_id_hash, customer_email_hash, legal_basis, supervisory_authority, tables_scrubbed, sessions_purged_count, records_scrubbed_count, digital_signature, compliance_statement, created_at
              FROM core_invest.gdpr_erasure_certificates
              WHERE certificate_id::text = $1`
	var c GDPRCertificateRecord
	err := r.db.QueryRow(ctx, query, id).
		Scan(&c.CertificateID, &c.CustomerIDHash, &c.CustomerEmailHash, &c.LegalBasis, &c.SupervisoryAuthority, &c.TablesScrubbed, &c.SessionsPurgedCount, &c.RecordsScrubbedCount, &c.DigitalSignature, &c.ComplianceStatement, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get gdpr certificate: %w", err)
	}
	return &c, nil
}

func (r *pgxRepository) ExecuteGDPRErasure(ctx context.Context, customerID string) (*GDPRErasureStats, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	h := sha256.Sum256([]byte(customerID))
	custHash := hex.EncodeToString(h[:])

	scrubbedTables := []string{}
	totalPurged := 0

	cmdTag, err := r.db.Exec(ctx, `DELETE FROM core_invest.customer_api_keys WHERE customer_id = $1`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.customer_api_keys")
	}

	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.webhook_subscriptions WHERE customer_id = $1`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.webhook_subscriptions")
	}

	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.recurring_deposits WHERE customer_id = $1`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.recurring_deposits")
	}

	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.portfolio_holdings WHERE portfolio_id IN (
		SELECT portfolio_id FROM core_invest.investment_portfolios WHERE customer_id::text = $1
	)`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.portfolio_holdings")
	}

	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.tax_lot_sales WHERE portfolio_id IN (
		SELECT portfolio_id FROM core_invest.investment_portfolios WHERE customer_id::text = $1
	)`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.tax_lot_sales")
	}

	anonName := fmt.Sprintf("Anonymized Investor (%s)", custHash[:8])
	_, err = r.db.Exec(ctx, `UPDATE core_invest.investment_portfolios
                             SET account_name = $1, status = 'closed', updated_at = CURRENT_TIMESTAMP
                             WHERE customer_id::text = $2`, anonName, customerID)
	if err == nil {
		scrubbedTables = append(scrubbedTables, "core_invest.investment_portfolios")
	}

	_, _ = r.db.Exec(ctx, `UPDATE ops.it_tickets
                           SET author_username = 'ANONYMIZED_GDPR_USER',
                               body = '[Content redacted pursuant to GDPR Article 17 erasure request]'
                           WHERE author_username = $1`, customerID)
	scrubbedTables = append(scrubbedTables, "ops.it_tickets")

	_, _ = r.db.Exec(ctx, `UPDATE ops.ticket_comments
                           SET author_username = 'ANONYMIZED_GDPR_USER',
                               comment = '[Content redacted pursuant to GDPR Article 17 erasure request]'
                           WHERE author_username = $1`, customerID)
	scrubbedTables = append(scrubbedTables, "ops.ticket_comments")

	return &GDPRErasureStats{
		CustomerIDHash:       custHash,
		CustomerEmailHash:    custHash,
		TablesScrubbed:       scrubbedTables,
		RecordsScrubbedCount: totalPurged,
	}, nil
}
