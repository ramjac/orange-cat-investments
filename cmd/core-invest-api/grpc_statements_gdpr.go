package main

import (
	"context"
	"time"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"github.com/orange-cat-investments/oci/pkg/reporting"
)

func (s *CoreInvestGRPCServer) ExecuteGDPRErasure(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
	cert, err := s.svc.ExecuteGDPRErasure(ctx, in.CustomerID, in.Email)
	if err != nil {
		return nil, err
	}
	pdfBytes, _ := reporting.GenerateGDPRCertificatePDF(cert)
	return &coreinvestv1.ExecuteGDPRErasureResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        cert.CertificateID,
			CustomerID:           in.CustomerID,
			CustomerIDHash:       cert.CustomerIDHash,
			CustomerEmailHash:    cert.CustomerEmailHash,
			ErasedAt:             cert.ErasureTimestamp.Format(time.RFC3339),
			LegalBasis:           cert.LegalBasis,
			SupervisoryAuthority: cert.SupervisoryAuthority,
			AnonymizedTables:     cert.TablesScrubbed,
			SessionsPurgedCount:  int32(cert.SessionsPurgedCount),
			RecordsScrubbedCount: int32(cert.RecordsScrubbedCount),
			ComplianceHash:       cert.DigitalSignature,
			ComplianceStatement:  cert.ComplianceStatement,
			Status:               "erased",
		},
		PDFBytes: pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) GetGDPRErasureCertificate(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error) {
	rec, err := s.svc.GetGDPRErasureCertificate(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	cert := &reporting.GDPRDeletionCertificate{
		CertificateID:        rec.CertificateID,
		CustomerIDHash:       rec.CustomerIDHash,
		CustomerEmailHash:    rec.CustomerEmailHash,
		ErasureTimestamp:     rec.CreatedAt,
		LegalBasis:           rec.LegalBasis,
		SupervisoryAuthority: rec.SupervisoryAuthority,
		TablesScrubbed:       rec.TablesScrubbed,
		SessionsPurgedCount:  rec.SessionsPurgedCount,
		RecordsScrubbedCount: rec.RecordsScrubbedCount,
		DigitalSignature:     rec.DigitalSignature,
		ComplianceStatement:  rec.ComplianceStatement,
	}
	pdfBytes, _ := reporting.GenerateGDPRCertificatePDF(cert)
	return &coreinvestv1.GetGDPRErasureCertificateResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        rec.CertificateID,
			CustomerIDHash:       rec.CustomerIDHash,
			CustomerEmailHash:    rec.CustomerEmailHash,
			ErasedAt:             rec.CreatedAt.Format(time.RFC3339),
			LegalBasis:           rec.LegalBasis,
			SupervisoryAuthority: rec.SupervisoryAuthority,
			AnonymizedTables:     rec.TablesScrubbed,
			SessionsPurgedCount:  int32(rec.SessionsPurgedCount),
			RecordsScrubbedCount: int32(rec.RecordsScrubbedCount),
			ComplianceHash:       rec.DigitalSignature,
			ComplianceStatement:  rec.ComplianceStatement,
			Status:               "erased",
		},
		PDFBytes: pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) CreateAPIKey(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
	key, err := s.svc.CreateCustomerAPIKey(ctx, &core_invest_repo.CustomerAPIKey{
		CustomerID:  in.CustomerID,
		Name:        in.Label,
		Permissions: in.Scopes,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.CreateAPIKeyResponse{
		APIKey: &coreinvestv1.CustomerAPIKey{
			KeyID:      key.KeyID,
			CustomerID: key.CustomerID,
			Label:      key.Name,
			KeyPrefix:  key.KeyPrefix,
			Scopes:     key.Permissions,
			Status:     key.Status,
			CreatedAt:  key.CreatedAt.Format(time.RFC3339),
		},
		RawSecret: key.APIKey,
	}, nil
}

func (s *CoreInvestGRPCServer) ListAPIKeys(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error) {
	keys, err := s.svc.ListCustomerAPIKeys(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CustomerAPIKey
	for _, k := range keys {
		out = append(out, &coreinvestv1.CustomerAPIKey{
			KeyID:      k.KeyID,
			CustomerID: k.CustomerID,
			Label:      k.Name,
			KeyPrefix:  k.KeyPrefix,
			Scopes:     k.Permissions,
			Status:     k.Status,
			CreatedAt:  k.CreatedAt.Format(time.RFC3339),
		})
	}
	return &coreinvestv1.ListAPIKeysResponse{APIKeys: out}, nil
}

func (s *CoreInvestGRPCServer) RevokeAPIKey(ctx context.Context, in *coreinvestv1.RevokeAPIKeyRequest) (*coreinvestv1.RevokeAPIKeyResponse, error) {
	if err := s.svc.RevokeCustomerAPIKey(ctx, in.CustomerID, in.KeyID); err != nil {
		return nil, err
	}
	return &coreinvestv1.RevokeAPIKeyResponse{Status: "revoked"}, nil
}

func (s *CoreInvestGRPCServer) CreateWebhook(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
	wh, err := s.svc.CreateWebhookSubscription(ctx, &core_invest_repo.WebhookSubscription{
		CustomerID: in.CustomerID,
		TargetURL:  in.TargetURL,
		Events:     in.EventTypes,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.CreateWebhookResponse{
		Webhook: &coreinvestv1.CustomerWebhook{
			WebhookID:  wh.SubscriptionID,
			CustomerID: wh.CustomerID,
			TargetURL:  wh.TargetURL,
			EventTypes: wh.Events,
			Secret:     wh.Secret,
			Status:     wh.Status,
			CreatedAt:  wh.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *CoreInvestGRPCServer) ListWebhooks(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
	whs, err := s.svc.ListWebhookSubscriptions(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CustomerWebhook
	for _, w := range whs {
		out = append(out, &coreinvestv1.CustomerWebhook{
			WebhookID:  w.SubscriptionID,
			CustomerID: w.CustomerID,
			TargetURL:  w.TargetURL,
			EventTypes: w.Events,
			Secret:     w.Secret,
			Status:     w.Status,
			CreatedAt:  w.CreatedAt.Format(time.RFC3339),
		})
	}
	return &coreinvestv1.ListWebhooksResponse{Webhooks: out}, nil
}

func (s *CoreInvestGRPCServer) DeleteWebhook(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
	if err := s.svc.DeleteWebhookSubscription(ctx, in.CustomerID, in.WebhookID); err != nil {
		return nil, err
	}
	return &coreinvestv1.DeleteWebhookResponse{Status: "deleted"}, nil
}

func (s *CoreInvestGRPCServer) GenerateStatement(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
	yr := int(in.Year)
	if yr == 0 {
		yr = time.Now().Year()
	}
	stmt, err := s.svc.GenerateStatement(ctx, in.PortfolioID, yr)
	if err != nil {
		return nil, err
	}
	stmtID, _ := stmt["statement_id"].(string)
	pID, _ := stmt["portfolio_id"].(string)
	docName, _ := stmt["document_name"].(string)
	return &coreinvestv1.GenerateStatementResponse{
		StatementID:   stmtID,
		PortfolioID:   pID,
		Year:          int32(yr),
		StatementType: "1099_b",
		Period:        docName,
		Status:        "generated",
	}, nil
}

func (s *CoreInvestGRPCServer) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	stmts, err := s.svc.ListStatements(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CustomerStatement
	for _, st := range stmts {
		if in.StatementType != "" && st.StatementType != in.StatementType {
			continue
		}
		out = append(out, toProtoStatement(st))
	}
	return &coreinvestv1.ListStatementsResponse{Statements: out}, nil
}

func (s *CoreInvestGRPCServer) GetStatement(ctx context.Context, in *coreinvestv1.GetStatementRequest) (*coreinvestv1.GetStatementResponse, error) {
	st, err := s.svc.GetStatement(ctx, in.StatementID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetStatementResponse{Statement: toProtoStatement(st)}, nil
}

func (s *CoreInvestGRPCServer) GenerateMonthlyStatement(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
	st, pdfBytes, err := s.svc.GenerateMonthlyStatement(ctx, in.CustomerID, in.PortfolioID, int(in.Year), int(in.Month))
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GenerateMonthlyStatementResponse{
		Statement: toProtoStatement(st),
		PDFBytes:  pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) GenerateForm1099B(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
	st, pdfBytes, err := s.svc.GenerateForm1099B(ctx, in.CustomerID, in.PortfolioID, int(in.TaxYear))
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GenerateForm1099BResponse{
		Statement: toProtoStatement(st),
		PDFBytes:  pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) GetStatementDocument(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
	st, pdfBytes, err := s.svc.GetStatementDocument(ctx, in.CustomerID, in.StatementID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetStatementDocumentResponse{
		Statement: toProtoStatement(st),
		PDFBytes:  pdfBytes,
	}, nil
}
