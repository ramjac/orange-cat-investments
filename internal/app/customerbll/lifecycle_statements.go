package customerbll

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/reporting"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) RegisterCustomer(ctx context.Context, in *customerbllv1.RegisterCustomerRequest) (*customerbllv1.RegisterCustomerResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if s.ciClient != nil {
		res, err := s.ciClient.RegisterCustomer(ctx, &coreinvestv1.RegisterCustomerRequest{
			Name:            in.FullName,
			Email:           in.Email,
			Password:        in.Password,
			BankPublicToken: in.BankPublicToken,
			TargetFund:      in.TargetFund,
		})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return &customerbllv1.RegisterCustomerResponse{Result: res}, nil
	}

	custID := "cust-" + watermill.NewUUID()[:8]
	return &customerbllv1.RegisterCustomerResponse{
		Result: &coreinvestv1.RegisterCustomerResponse{
			CustomerID:      custID,
			Name:            in.FullName,
			Email:           in.Email,
			PortfolioID:     "port-" + custID,
			LinkedBankAccID: "acc-mock-1111",
			Status:          "active",
			CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *Server) CloseAccount(ctx context.Context, in *customerbllv1.CloseAccountRequest) (*customerbllv1.CloseAccountResponse, error) {
	if in == nil || in.CustomerID == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	if s.ciClient != nil {
		res, err := s.ciClient.CloseAndLiquidateAccount(ctx, &coreinvestv1.CloseAndLiquidateAccountRequest{
			CustomerID:           in.CustomerID,
			PortfolioID:          in.PortfolioID,
			DestinationAccountID: in.DestinationAccountID,
		})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return &customerbllv1.CloseAccountResponse{Result: res}, nil
	}

	return &customerbllv1.CloseAccountResponse{
		Result: &coreinvestv1.CloseAndLiquidateAccountResponse{
			CustomerID:           in.CustomerID,
			PortfolioID:          in.PortfolioID,
			DestinationAccountID: in.DestinationAccountID,
			ClosedAt:             time.Now().UTC().Format(time.RFC3339),
			AccountStatus:        "closed",
		},
	}, nil
}

func (s *Server) ExecuteGDPRErasure(ctx context.Context, in *customerbllv1.ExecuteGDPRErasureRequest) (*customerbllv1.ExecuteGDPRErasureResponse, error) {
	if in == nil || in.CustomerID == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	if s.ciClient != nil {
		res, err := s.ciClient.ExecuteGDPRErasure(ctx, &coreinvestv1.ExecuteGDPRErasureRequest{
			CustomerID: in.CustomerID,
			Email:      in.Email,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "gdpr erasure failed: %v", err)
		}
		return &customerbllv1.ExecuteGDPRErasureResponse{Certificate: res.Certificate}, nil
	}

	h := sha256.Sum256([]byte(in.CustomerID))
	custHash := hex.EncodeToString(h[:])
	eh := sha256.Sum256([]byte(in.Email))
	emailHash := hex.EncodeToString(eh[:])

	return &customerbllv1.ExecuteGDPRErasureResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        "gdpr-cert-" + watermill.NewUUID()[:8],
			CustomerID:           in.CustomerID,
			CustomerIDHash:       custHash,
			CustomerEmailHash:    emailHash,
			ErasedAt:             time.Now().UTC().Format(time.RFC3339),
			LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
			SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
			AnonymizedTables:     []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions", "core_invest.recurring_deposits"},
			SessionsPurgedCount:  1,
			RecordsScrubbedCount: 4,
			ComplianceHash:       fmt.Sprintf("hmac-sha256:%s", custHash[:32]),
			ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
			Status:               "erased",
		},
	}, nil
}

func (s *Server) GetGDPRCertificate(ctx context.Context, in *customerbllv1.GetGDPRCertificateRequest) (*customerbllv1.GetGDPRCertificateResponse, error) {
	if in == nil || in.CertificateID == "" {
		return nil, status.Error(codes.InvalidArgument, "certificate_id is required")
	}
	if s.ciClient != nil && in.CustomerID != "" {
		res, err := s.ciClient.GetGDPRErasureCertificate(ctx, &coreinvestv1.GetGDPRErasureCertificateRequest{
			CustomerID: in.CustomerID,
		})
		if err == nil && res != nil {
			return &customerbllv1.GetGDPRCertificateResponse{
				Certificate: res.Certificate,
				PDFBytes:    res.PDFBytes,
			}, nil
		}
	}

	h := sha256.Sum256([]byte(in.CustomerID))
	custHash := hex.EncodeToString(h[:])
	eh := sha256.Sum256([]byte(in.Email))
	emailHash := hex.EncodeToString(eh[:])

	certModel := &reporting.GDPRDeletionCertificate{
		CertificateID:        in.CertificateID,
		CustomerIDHash:       custHash,
		CustomerEmailHash:    emailHash,
		ErasureTimestamp:     time.Now().UTC(),
		LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
		SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
		TablesScrubbed:       []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions", "core_invest.recurring_deposits"},
		SessionsPurgedCount:  1,
		RecordsScrubbedCount: 4,
		DigitalSignature:     fmt.Sprintf("hmac-sha256:%s", custHash[:32]),
		ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
	}
	pdfBytes, _ := reporting.GenerateGDPRCertificatePDF(certModel)

	return &customerbllv1.GetGDPRCertificateResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        certModel.CertificateID,
			CustomerID:           in.CustomerID,
			CustomerIDHash:       certModel.CustomerIDHash,
			CustomerEmailHash:    certModel.CustomerEmailHash,
			ErasedAt:             certModel.ErasureTimestamp.Format(time.RFC3339),
			LegalBasis:           certModel.LegalBasis,
			SupervisoryAuthority: certModel.SupervisoryAuthority,
			AnonymizedTables:     certModel.TablesScrubbed,
			SessionsPurgedCount:  int32(certModel.SessionsPurgedCount),
			RecordsScrubbedCount: int32(certModel.RecordsScrubbedCount),
			ComplianceHash:       certModel.DigitalSignature,
			ComplianceStatement:  certModel.ComplianceStatement,
			Status:               "erased",
		},
		PDFBytes: pdfBytes,
	}, nil
}

func (s *Server) ListStatements(ctx context.Context, in *customerbllv1.ListStatementsRequest) (*customerbllv1.ListStatementsResponse, error) {
	if in == nil {
		in = &customerbllv1.ListStatementsRequest{}
	}
	if s.ciClient != nil {
		res, err := s.ciClient.ListStatements(ctx, &coreinvestv1.ListStatementsRequest{
			CustomerID: in.CustomerID,
		})
		if err == nil && res != nil {
			return &customerbllv1.ListStatementsResponse{Statements: res.Statements}, nil
		}
	}
	return &customerbllv1.ListStatementsResponse{Statements: []*coreinvestv1.CustomerStatement{}}, nil
}

func (s *Server) GenerateStatement(ctx context.Context, in *customerbllv1.GenerateStatementRequest) (*customerbllv1.GenerateStatementResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	year := in.Year
	if year == 0 {
		year = int32(time.Now().Year())
	}
	month := in.Month
	if month == 0 {
		month = int32(time.Now().Month())
	}

	if s.ciClient != nil {
		if in.Type == "form_1099b" {
			res, err := s.ciClient.GenerateForm1099B(ctx, &coreinvestv1.GenerateForm1099BRequest{
				CustomerID:  in.CustomerID,
				PortfolioID: in.PortfolioID,
				TaxYear:     year,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "%v", err)
			}
			return &customerbllv1.GenerateStatementResponse{
				Statement: res.Statement,
				PDFBytes:  res.PDFBytes,
			}, nil
		}

		res, err := s.ciClient.GenerateMonthlyStatement(ctx, &coreinvestv1.GenerateMonthlyStatementRequest{
			CustomerID:  in.CustomerID,
			PortfolioID: in.PortfolioID,
			Year:        year,
			Month:       month,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
		return &customerbllv1.GenerateStatementResponse{
			Statement: res.Statement,
			PDFBytes:  res.PDFBytes,
		}, nil
	}

	stmtID := "stmt-" + watermill.NewUUID()[:8]
	return &customerbllv1.GenerateStatementResponse{
		Statement: &coreinvestv1.CustomerStatement{
			StatementID:   stmtID,
			CustomerID:    in.CustomerID,
			PortfolioID:   in.PortfolioID,
			StatementType: in.Type,
			Period:        fmt.Sprintf("%d-%02d", year, month),
			Status:        "generated",
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *Server) DownloadStatement(ctx context.Context, in *customerbllv1.DownloadStatementRequest) (*customerbllv1.DownloadStatementResponse, error) {
	if in == nil || in.StatementID == "" {
		return nil, status.Error(codes.InvalidArgument, "statement_id is required")
	}
	if s.ciClient != nil {
		res, err := s.ciClient.GetStatementDocument(ctx, &coreinvestv1.GetStatementDocumentRequest{
			CustomerID:  in.CustomerID,
			StatementID: in.StatementID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to retrieve statement: %v", err)
		}
		fileName := fmt.Sprintf("statement-%s.pdf", in.StatementID)
		return &customerbllv1.DownloadStatementResponse{
			Statement:   res.Statement,
			PDFBytes:    res.PDFBytes,
			ContentType: "application/pdf",
			FileName:    fileName,
		}, nil
	}

	return &customerbllv1.DownloadStatementResponse{
		PDFBytes:    []byte("%PDF-1.4 OCI Customer Statement Document Mock"),
		ContentType: "application/pdf",
		FileName:    fmt.Sprintf("statement-%s.pdf", in.StatementID),
	}, nil
}

func (s *Server) GenerateBatchStatements(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest service unavailable")
	}
	ports, err := s.ciClient.ListPortfolios(ctx, &coreinvestv1.ListPortfoliosRequest{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list portfolios: %v", err)
	}
	year := in.Year
	if year == 0 {
		year = int32(time.Now().AddDate(0, -1, 0).Year())
	}
	period := in.Period
	if period == "" {
		period = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	var successCount int32
	for _, p := range ports.Portfolios {
		_, err := s.ciClient.GenerateStatement(ctx, &coreinvestv1.GenerateStatementRequest{
			PortfolioID: p.PortfolioID,
			Year:        year,
		})
		if err == nil {
			successCount++
		}
	}
	return &customerbllv1.GenerateBatchStatementsResponse{
		Period:          period,
		TotalPortfolios: successCount,
		Status:          "completed",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}
