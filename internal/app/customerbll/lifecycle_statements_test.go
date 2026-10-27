package customerbll

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGenerateStatement_NilRequest(t *testing.T) {
	srv := NewServer(Config{})
	res, err := srv.GenerateStatement(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument status error, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil response, got %v", res)
	}
}

func TestGenerateStatement_FallbackMode(t *testing.T) {
	srv := NewServer(Config{}) // ciClient is nil

	t.Run("default zero year and month", func(t *testing.T) {
		req := &customerbllv1.GenerateStatementRequest{
			CustomerID:  "cust-123",
			PortfolioID: "port-123",
			Type:        "monthly_summary",
		}
		res, err := srv.GenerateStatement(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil || res.Statement == nil {
			t.Fatal("expected non-nil response and statement")
		}
		expectedYear := time.Now().Year()
		expectedMonth := int(time.Now().Month())
		expectedPeriod := fmt.Sprintf("%d-%02d", expectedYear, expectedMonth)

		if res.Statement.Period != expectedPeriod {
			t.Errorf("expected period %s, got %s", expectedPeriod, res.Statement.Period)
		}
		if res.Statement.CustomerID != req.CustomerID {
			t.Errorf("expected customer ID %s, got %s", req.CustomerID, res.Statement.CustomerID)
		}
		if res.Statement.PortfolioID != req.PortfolioID {
			t.Errorf("expected portfolio ID %s, got %s", req.PortfolioID, res.Statement.PortfolioID)
		}
		if res.Statement.StatementType != req.Type {
			t.Errorf("expected statement type %s, got %s", req.Type, res.Statement.StatementType)
		}
		if res.Statement.Status != "generated" {
			t.Errorf("expected status 'generated', got %s", res.Statement.Status)
		}
		if !strings.HasPrefix(res.Statement.StatementID, "stmt-") {
			t.Errorf("expected statement ID to start with stmt-, got %s", res.Statement.StatementID)
		}
	})

	t.Run("explicit year and month", func(t *testing.T) {
		req := &customerbllv1.GenerateStatementRequest{
			CustomerID:  "cust-456",
			PortfolioID: "port-456",
			Type:        "monthly_summary",
			Year:        2025,
			Month:       11,
		}
		res, err := srv.GenerateStatement(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Statement.Period != "2025-11" {
			t.Errorf("expected period 2025-11, got %s", res.Statement.Period)
		}
	})
}

func TestGenerateStatement_ServiceMode_Monthly(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		generateMonthlyStatementFn: func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
			if in.CustomerID != "cust-789" || in.Year != 2026 || in.Month != 3 {
				t.Errorf("unexpected arguments in GenerateMonthlyStatement: %+v", in)
			}
			return &coreinvestv1.GenerateMonthlyStatementResponse{
				Statement: &coreinvestv1.CustomerStatement{
					StatementID:   "stmt-mock-monthly",
					CustomerID:    in.CustomerID,
					PortfolioID:   in.PortfolioID,
					StatementType: "monthly",
					Period:        "2026-03",
					Status:        "ready",
				},
				PDFBytes: []byte("mock-monthly-pdf"),
			}, nil
		},
	}

	srv := NewServer(Config{})
	srv.SetCoreInvestClient(mockCI)

	req := &customerbllv1.GenerateStatementRequest{
		CustomerID:  "cust-789",
		PortfolioID: "port-789",
		Type:        "monthly",
		Year:        2026,
		Month:       3,
	}

	res, err := srv.GenerateStatement(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Statement.StatementID != "stmt-mock-monthly" {
		t.Errorf("expected stmt-mock-monthly, got %s", res.Statement.StatementID)
	}
	if string(res.PDFBytes) != "mock-monthly-pdf" {
		t.Errorf("expected 'mock-monthly-pdf', got %s", string(res.PDFBytes))
	}
}

func TestGenerateStatement_ServiceMode_Form1099B(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		generateForm1099BFn: func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
			if in.CustomerID != "cust-789" || in.TaxYear != 2025 {
				t.Errorf("unexpected arguments in GenerateForm1099B: %+v", in)
			}
			return &coreinvestv1.GenerateForm1099BResponse{
				Statement: &coreinvestv1.CustomerStatement{
					StatementID:   "stmt-mock-1099b",
					CustomerID:    in.CustomerID,
					PortfolioID:   in.PortfolioID,
					StatementType: "form_1099b",
					Period:        "2025",
					Status:        "ready",
				},
				PDFBytes: []byte("mock-1099b-pdf"),
			}, nil
		},
	}

	srv := NewServer(Config{})
	srv.SetCoreInvestClient(mockCI)

	req := &customerbllv1.GenerateStatementRequest{
		CustomerID:  "cust-789",
		PortfolioID: "port-789",
		Type:        "form_1099b",
		Year:        2025,
	}

	res, err := srv.GenerateStatement(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Statement.StatementID != "stmt-mock-1099b" {
		t.Errorf("expected stmt-mock-1099b, got %s", res.Statement.StatementID)
	}
	if string(res.PDFBytes) != "mock-1099b-pdf" {
		t.Errorf("expected 'mock-1099b-pdf', got %s", string(res.PDFBytes))
	}
}

func TestGenerateStatement_ServiceMode_Error(t *testing.T) {
	t.Run("form_1099b error", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			generateForm1099BFn: func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
				return nil, errors.New("core-invest tax error")
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		_, err := srv.GenerateStatement(context.Background(), &customerbllv1.GenerateStatementRequest{
			CustomerID: "cust-1",
			Type:       "form_1099b",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Internal {
			t.Fatalf("expected Internal error, got %v", err)
		}
	})

	t.Run("monthly statement error", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			generateMonthlyStatementFn: func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
				return nil, errors.New("core-invest monthly error")
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		_, err := srv.GenerateStatement(context.Background(), &customerbllv1.GenerateStatementRequest{
			CustomerID: "cust-1",
			Type:       "monthly",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Internal {
			t.Fatalf("expected Internal error, got %v", err)
		}
	})
}

func TestListStatements(t *testing.T) {
	t.Run("fallback mode", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.ListStatements(context.Background(), &customerbllv1.ListStatementsRequest{CustomerID: "cust-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil || len(res.Statements) != 0 {
			t.Fatalf("expected empty statement list, got %v", res)
		}
	})

	t.Run("service mode success", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			listStatementsFn: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
				return &coreinvestv1.ListStatementsResponse{
					Statements: []*coreinvestv1.CustomerStatement{
						{StatementID: "stmt-01", CustomerID: in.CustomerID},
					},
				}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		res, err := srv.ListStatements(context.Background(), &customerbllv1.ListStatementsRequest{CustomerID: "cust-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Statements) != 1 || res.Statements[0].StatementID != "stmt-01" {
			t.Fatalf("unexpected statements response: %v", res)
		}
	})
}

func TestDownloadStatement(t *testing.T) {
	t.Run("nil or empty statement ID error", func(t *testing.T) {
		srv := NewServer(Config{})
		_, err := srv.DownloadStatement(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
		_, err = srv.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{StatementID: ""})
		if err == nil {
			t.Fatal("expected error for empty statement_id")
		}
	})

	t.Run("fallback mode", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{
			CustomerID:  "cust-1",
			StatementID: "stmt-99",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ContentType != "application/pdf" {
			t.Errorf("expected application/pdf, got %s", res.ContentType)
		}
		if res.FileName != "statement-stmt-99.pdf" {
			t.Errorf("expected statement-stmt-99.pdf, got %s", res.FileName)
		}
	})

	t.Run("service mode", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			getStatementDocumentFn: func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
				return &coreinvestv1.GetStatementDocumentResponse{
					Statement: &coreinvestv1.CustomerStatement{StatementID: in.StatementID},
					PDFBytes:  []byte("ci-pdf-data"),
				}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		res, err := srv.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{
			CustomerID:  "cust-1",
			StatementID: "stmt-99",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(res.PDFBytes) != "ci-pdf-data" {
			t.Errorf("expected ci-pdf-data, got %s", string(res.PDFBytes))
		}
	})

	t.Run("service mode error", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			getStatementDocumentFn: func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
				return nil, errors.New("doc error")
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		_, err := srv.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{
			CustomerID:  "cust-1",
			StatementID: "stmt-99",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Internal {
			t.Fatalf("expected Internal status code, got %v", err)
		}
	})
}

func TestGenerateBatchStatements(t *testing.T) {
	t.Run("ciClient unavailable", func(t *testing.T) {
		srv := NewServer(Config{})
		_, err := srv.GenerateBatchStatements(context.Background(), &customerbllv1.GenerateBatchStatementsRequest{})
		if err == nil {
			t.Fatal("expected error when ciClient is nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Unavailable {
			t.Fatalf("expected Unavailable status error, got %v", err)
		}
	})

	t.Run("ciClient list portfolios error", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			listPortfoliosFn: func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
				return nil, errors.New("db error")
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		_, err := srv.GenerateBatchStatements(context.Background(), &customerbllv1.GenerateBatchStatementsRequest{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success batch generation", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			listPortfoliosFn: func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
				return &coreinvestv1.ListPortfoliosResponse{
					Portfolios: []*coreinvestv1.Portfolio{
						{PortfolioID: "p1"},
						{PortfolioID: "p2"},
					},
				}, nil
			},
			generateStatementFn: func(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
				if in.PortfolioID == "p2" {
					return nil, errors.New("generation failed for p2")
				}
				return &coreinvestv1.GenerateStatementResponse{}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		res, err := srv.GenerateBatchStatements(context.Background(), &customerbllv1.GenerateBatchStatementsRequest{
			Year:   2026,
			Period: "2026-02",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TotalPortfolios != 1 {
			t.Errorf("expected 1 success portfolio, got %d", res.TotalPortfolios)
		}
		if res.Period != "2026-02" {
			t.Errorf("expected period 2026-02, got %s", res.Period)
		}
	})
}

func TestRegisterCustomer(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		srv := NewServer(Config{})
		_, err := srv.RegisterCustomer(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("fallback mode", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.RegisterCustomer(context.Background(), &customerbllv1.RegisterCustomerRequest{
			FullName: "Tabby Cat",
			Email:    "tabby@oci.local",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Result.Name != "Tabby Cat" || res.Result.Email != "tabby@oci.local" {
			t.Errorf("unexpected result: %+v", res.Result)
		}
	})

	t.Run("service mode error and success", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			registerCustomerFn: func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
				if in.Email == "error@oci.local" {
					return nil, errors.New("email exists")
				}
				return &coreinvestv1.RegisterCustomerResponse{CustomerID: "cust-ci-123"}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		_, err := srv.RegisterCustomer(context.Background(), &customerbllv1.RegisterCustomerRequest{Email: "error@oci.local"})
		if err == nil {
			t.Fatal("expected error")
		}

		res, err := srv.RegisterCustomer(context.Background(), &customerbllv1.RegisterCustomerRequest{Email: "valid@oci.local"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Result.CustomerID != "cust-ci-123" {
			t.Errorf("expected cust-ci-123, got %s", res.Result.CustomerID)
		}
	})
}

func TestCloseAccount(t *testing.T) {
	t.Run("invalid request", func(t *testing.T) {
		srv := NewServer(Config{})
		_, err := srv.CloseAccount(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
		_, err = srv.CloseAccount(context.Background(), &customerbllv1.CloseAccountRequest{CustomerID: ""})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("fallback mode", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.CloseAccount(context.Background(), &customerbllv1.CloseAccountRequest{CustomerID: "cust-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Result.AccountStatus != "closed" {
			t.Errorf("expected closed, got %s", res.Result.AccountStatus)
		}
	})

	t.Run("service mode", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			closeAndLiquidateAccountFn: func(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
				return &coreinvestv1.CloseAndLiquidateAccountResponse{
					CustomerID:    in.CustomerID,
					AccountStatus: "liquidated_and_closed",
				}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		res, err := srv.CloseAccount(context.Background(), &customerbllv1.CloseAccountRequest{CustomerID: "cust-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Result.AccountStatus != "liquidated_and_closed" {
			t.Errorf("expected liquidated_and_closed, got %s", res.Result.AccountStatus)
		}
	})
}

func TestExecuteGDPRErasure(t *testing.T) {
	t.Run("invalid request", func(t *testing.T) {
		srv := NewServer(Config{})
		_, err := srv.ExecuteGDPRErasure(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("fallback mode", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-gdpr-1",
			Email:      "gdpr@oci.local",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Certificate.CustomerID != "cust-gdpr-1" || res.Certificate.Status != "erased" {
			t.Errorf("unexpected certificate: %+v", res.Certificate)
		}
	})

	t.Run("service mode", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			executeGDPRErasureFn: func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
				return &coreinvestv1.ExecuteGDPRErasureResponse{
					Certificate: &coreinvestv1.GDPRErasureCertificate{
						CertificateID: "cert-ci-123",
						Status:        "erased",
					},
				}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		res, err := srv.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-gdpr-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Certificate.CertificateID != "cert-ci-123" {
			t.Errorf("expected cert-ci-123, got %s", res.Certificate.CertificateID)
		}
	})
}

func TestGetGDPRCertificate(t *testing.T) {
	t.Run("invalid request", func(t *testing.T) {
		srv := NewServer(Config{})
		_, err := srv.GetGDPRCertificate(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("fallback mode", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.GetGDPRCertificate(context.Background(), &customerbllv1.GetGDPRCertificateRequest{
			CertificateID: "cert-1",
			CustomerID:    "cust-1",
			Email:         "cust1@oci.local",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Certificate.CertificateID != "cert-1" {
			t.Errorf("expected cert-1, got %s", res.Certificate.CertificateID)
		}
		if len(res.PDFBytes) == 0 {
			t.Error("expected non-empty PDF bytes")
		}
	})

	t.Run("service mode", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			getGDPRErasureCertificateFn: func(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error) {
				return &coreinvestv1.GetGDPRErasureCertificateResponse{
					Certificate: &coreinvestv1.GDPRErasureCertificate{CertificateID: "cert-ci"},
					PDFBytes:    []byte("gdpr-pdf"),
				}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockCI)

		res, err := srv.GetGDPRCertificate(context.Background(), &customerbllv1.GetGDPRCertificateRequest{
			CertificateID: "cert-ci",
			CustomerID:    "cust-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Certificate.CertificateID != "cert-ci" || string(res.PDFBytes) != "gdpr-pdf" {
			t.Errorf("unexpected response: %+v", res)
		}
	})
}
