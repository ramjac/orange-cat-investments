package customerbll

import (
	"context"
	"errors"
	"strings"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockLifecycleCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient

	registerCustomerFunc          func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error)
	closeAndLiquidateAccountFunc  func(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error)
	executeGDPRErasureFunc        func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error)
	getGDPRErasureCertificateFunc func(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error)
	listStatementsFunc            func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error)
	generateMonthlyStatementFunc  func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error)
	generateForm1099BFunc         func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error)
	getStatementDocumentFunc      func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error)
	listPortfoliosFunc            func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error)
	generateStatementFunc         func(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error)
}

func (m *mockLifecycleCoreInvestClient) RegisterCustomer(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
	if m.registerCustomerFunc != nil {
		return m.registerCustomerFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) CloseAndLiquidateAccount(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
	if m.closeAndLiquidateAccountFunc != nil {
		return m.closeAndLiquidateAccountFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) ExecuteGDPRErasure(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
	if m.executeGDPRErasureFunc != nil {
		return m.executeGDPRErasureFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) GetGDPRErasureCertificate(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error) {
	if m.getGDPRErasureCertificateFunc != nil {
		return m.getGDPRErasureCertificateFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	if m.listStatementsFunc != nil {
		return m.listStatementsFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) GenerateMonthlyStatement(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
	if m.generateMonthlyStatementFunc != nil {
		return m.generateMonthlyStatementFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) GenerateForm1099B(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
	if m.generateForm1099BFunc != nil {
		return m.generateForm1099BFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) GetStatementDocument(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
	if m.getStatementDocumentFunc != nil {
		return m.getStatementDocumentFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) ListPortfolios(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
	if m.listPortfoliosFunc != nil {
		return m.listPortfoliosFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockLifecycleCoreInvestClient) GenerateStatement(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
	if m.generateStatementFunc != nil {
		return m.generateStatementFunc(ctx, in)
	}
	return nil, errors.New("unimplemented")
}

func TestRegisterCustomer_NilInput(t *testing.T) {
	s := NewServer(Config{})
	res, err := s.RegisterCustomer(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "request is required", st.Message())
}

func TestRegisterCustomer_Fallback_WithoutCiClient(t *testing.T) {
	s := NewServer(Config{})
	req := &customerbllv1.RegisterCustomerRequest{
		FullName:        "Oliver Cat",
		Email:           "oliver@orangecat.io",
		Password:        "MeowSecret123!",
		BankPublicToken: "public-sandbox-token-123",
		TargetFund:      "fund-growth-01",
	}

	res, err := s.RegisterCustomer(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Result)

	assert.True(t, strings.HasPrefix(res.Result.CustomerID, "cust-"), "expected customer_id to start with cust-")
	assert.Equal(t, "Oliver Cat", res.Result.Name)
	assert.Equal(t, "oliver@orangecat.io", res.Result.Email)
	assert.Equal(t, "port-"+res.Result.CustomerID, res.Result.PortfolioID)
	assert.Equal(t, "acc-mock-1111", res.Result.LinkedBankAccID)
	assert.Equal(t, "active", res.Result.Status)
	assert.NotEmpty(t, res.Result.CreatedAt)
}

func TestRegisterCustomer_WithCiClient_Success(t *testing.T) {
	mockCi := &mockLifecycleCoreInvestClient{
		registerCustomerFunc: func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
			assert.Equal(t, "Whiskers Orange", in.Name)
			assert.Equal(t, "whiskers@orangecat.io", in.Email)
			assert.Equal(t, "Pass1234!", in.Password)
			assert.Equal(t, "public-token-abc", in.BankPublicToken)
			assert.Equal(t, "fund-aggr-99", in.TargetFund)

			return &coreinvestv1.RegisterCustomerResponse{
				CustomerID:      "cust-ci-100",
				Name:            in.Name,
				Email:           in.Email,
				PortfolioID:     "port-ci-100",
				LinkedBankAccID: "acc-ci-100",
				Status:          "active",
				CreatedAt:       "2026-04-01T10:00:00Z",
			}, nil
		},
	}

	s := NewServer(Config{})
	s.SetCoreInvestClient(mockCi)

	req := &customerbllv1.RegisterCustomerRequest{
		FullName:        "Whiskers Orange",
		Email:           "whiskers@orangecat.io",
		Password:        "Pass1234!",
		BankPublicToken: "public-token-abc",
		TargetFund:      "fund-aggr-99",
	}

	res, err := s.RegisterCustomer(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Result)

	assert.Equal(t, "cust-ci-100", res.Result.CustomerID)
	assert.Equal(t, "Whiskers Orange", res.Result.Name)
	assert.Equal(t, "whiskers@orangecat.io", res.Result.Email)
	assert.Equal(t, "port-ci-100", res.Result.PortfolioID)
	assert.Equal(t, "acc-ci-100", res.Result.LinkedBankAccID)
	assert.Equal(t, "active", res.Result.Status)
}

func TestRegisterCustomer_WithCiClient_Error(t *testing.T) {
	mockCi := &mockLifecycleCoreInvestClient{
		registerCustomerFunc: func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
			return nil, errors.New("email already registered")
		},
	}

	s := NewServer(Config{})
	s.SetCoreInvestClient(mockCi)

	req := &customerbllv1.RegisterCustomerRequest{
		FullName: "Duplicate Cat",
		Email:    "existing@orangecat.io",
	}

	res, err := s.RegisterCustomer(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "email already registered")
}

func TestCloseAccount(t *testing.T) {
	t.Run("invalid input", func(t *testing.T) {
		s := NewServer(Config{})
		_, err := s.CloseAccount(context.Background(), nil)
		require.Error(t, err)
		_, err = s.CloseAccount(context.Background(), &customerbllv1.CloseAccountRequest{})
		require.Error(t, err)
	})

	t.Run("fallback without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		res, err := s.CloseAccount(context.Background(), &customerbllv1.CloseAccountRequest{
			CustomerID:           "cust-001",
			PortfolioID:          "port-001",
			DestinationAccountID: "acc-dest-1",
		})
		require.NoError(t, err)
		assert.Equal(t, "cust-001", res.Result.CustomerID)
		assert.Equal(t, "closed", res.Result.AccountStatus)
	})

	t.Run("with ciClient success", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			closeAndLiquidateAccountFunc: func(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
				return &coreinvestv1.CloseAndLiquidateAccountResponse{
					CustomerID:    in.CustomerID,
					AccountStatus: "closed",
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.CloseAccount(context.Background(), &customerbllv1.CloseAccountRequest{
			CustomerID: "cust-002",
		})
		require.NoError(t, err)
		assert.Equal(t, "cust-002", res.Result.CustomerID)
	})
}

func TestExecuteGDPRErasure(t *testing.T) {
	t.Run("invalid input", func(t *testing.T) {
		s := NewServer(Config{})
		_, err := s.ExecuteGDPRErasure(context.Background(), nil)
		require.Error(t, err)
		_, err = s.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{})
		require.Error(t, err)
	})

	t.Run("fallback without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		res, err := s.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-gdpr-1",
			Email:      "user@gdpr.com",
		})
		require.NoError(t, err)
		require.NotNil(t, res.Certificate)
		assert.Equal(t, "cust-gdpr-1", res.Certificate.CustomerID)
		assert.Equal(t, "erased", res.Certificate.Status)
	})

	t.Run("with ciClient success", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			executeGDPRErasureFunc: func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
				return &coreinvestv1.ExecuteGDPRErasureResponse{
					Certificate: &coreinvestv1.GDPRErasureCertificate{
						CustomerID: in.CustomerID,
						Status:     "erased",
					},
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-gdpr-2",
		})
		require.NoError(t, err)
		assert.Equal(t, "cust-gdpr-2", res.Certificate.CustomerID)
	})
}

func TestGetGDPRCertificate(t *testing.T) {
	t.Run("invalid input", func(t *testing.T) {
		s := NewServer(Config{})
		_, err := s.GetGDPRCertificate(context.Background(), nil)
		require.Error(t, err)
		_, err = s.GetGDPRCertificate(context.Background(), &customerbllv1.GetGDPRCertificateRequest{})
		require.Error(t, err)
	})

	t.Run("fallback without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		res, err := s.GetGDPRCertificate(context.Background(), &customerbllv1.GetGDPRCertificateRequest{
			CertificateID: "cert-001",
			CustomerID:    "cust-001",
		})
		require.NoError(t, err)
		assert.Equal(t, "cert-001", res.Certificate.CertificateID)
		assert.NotEmpty(t, res.PDFBytes)
	})
}

func TestListStatements(t *testing.T) {
	t.Run("without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		res, err := s.ListStatements(context.Background(), &customerbllv1.ListStatementsRequest{
			CustomerID: "cust-001",
		})
		require.NoError(t, err)
		assert.Empty(t, res.Statements)
	})

	t.Run("with ciClient", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
				return &coreinvestv1.ListStatementsResponse{
					Statements: []*coreinvestv1.CustomerStatement{
						{StatementID: "stmt-100", CustomerID: in.CustomerID},
					},
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.ListStatements(context.Background(), &customerbllv1.ListStatementsRequest{
			CustomerID: "cust-001",
		})
		require.NoError(t, err)
		require.Len(t, res.Statements, 1)
		assert.Equal(t, "stmt-100", res.Statements[0].StatementID)
	})
}

func TestGenerateStatement(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		s := NewServer(Config{})
		_, err := s.GenerateStatement(context.Background(), nil)
		require.Error(t, err)
	})

	t.Run("fallback without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		res, err := s.GenerateStatement(context.Background(), &customerbllv1.GenerateStatementRequest{
			CustomerID:  "cust-1",
			PortfolioID: "port-1",
			Type:        "monthly_summary",
			Year:        2026,
			Month:       3,
		})
		require.NoError(t, err)
		assert.Equal(t, "cust-1", res.Statement.CustomerID)
		assert.Equal(t, "2026-03", res.Statement.Period)
	})

	t.Run("with ciClient monthly statement", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			generateMonthlyStatementFunc: func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
				return &coreinvestv1.GenerateMonthlyStatementResponse{
					Statement: &coreinvestv1.CustomerStatement{StatementID: "stmt-m-1"},
					PDFBytes:  []byte("pdf-data"),
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.GenerateStatement(context.Background(), &customerbllv1.GenerateStatementRequest{
			CustomerID: "cust-1",
			Type:       "monthly_summary",
		})
		require.NoError(t, err)
		assert.Equal(t, "stmt-m-1", res.Statement.StatementID)
	})

	t.Run("with ciClient 1099b", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			generateForm1099BFunc: func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
				return &coreinvestv1.GenerateForm1099BResponse{
					Statement: &coreinvestv1.CustomerStatement{StatementID: "stmt-1099-1"},
					PDFBytes:  []byte("pdf-1099"),
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.GenerateStatement(context.Background(), &customerbllv1.GenerateStatementRequest{
			CustomerID: "cust-1",
			Type:       "form_1099b",
		})
		require.NoError(t, err)
		assert.Equal(t, "stmt-1099-1", res.Statement.StatementID)
	})
}

func TestDownloadStatement(t *testing.T) {
	t.Run("invalid input", func(t *testing.T) {
		s := NewServer(Config{})
		_, err := s.DownloadStatement(context.Background(), nil)
		require.Error(t, err)
		_, err = s.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{})
		require.Error(t, err)
	})

	t.Run("fallback without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		res, err := s.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{
			StatementID: "stmt-99",
		})
		require.NoError(t, err)
		assert.Equal(t, "application/pdf", res.ContentType)
		assert.Equal(t, "statement-stmt-99.pdf", res.FileName)
	})

	t.Run("with ciClient", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			getStatementDocumentFunc: func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
				return &coreinvestv1.GetStatementDocumentResponse{
					Statement: &coreinvestv1.CustomerStatement{StatementID: in.StatementID},
					PDFBytes:  []byte("doc-pdf"),
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.DownloadStatement(context.Background(), &customerbllv1.DownloadStatementRequest{
			CustomerID:  "cust-1",
			StatementID: "stmt-99",
		})
		require.NoError(t, err)
		assert.Equal(t, "stmt-99", res.Statement.StatementID)
		assert.Equal(t, []byte("doc-pdf"), res.PDFBytes)
	})
}

func TestGenerateBatchStatements(t *testing.T) {
	t.Run("without ciClient unavailable", func(t *testing.T) {
		s := NewServer(Config{})
		_, err := s.GenerateBatchStatements(context.Background(), &customerbllv1.GenerateBatchStatementsRequest{})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
	})

	t.Run("with ciClient success", func(t *testing.T) {
		mockCi := &mockLifecycleCoreInvestClient{
			listPortfoliosFunc: func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
				return &coreinvestv1.ListPortfoliosResponse{
					Portfolios: []*coreinvestv1.Portfolio{
						{PortfolioID: "p1"},
						{PortfolioID: "p2"},
					},
				}, nil
			},
			generateStatementFunc: func(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
				return &coreinvestv1.GenerateStatementResponse{}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCi)

		res, err := s.GenerateBatchStatements(context.Background(), &customerbllv1.GenerateBatchStatementsRequest{
			Year:   2026,
			Period: "2026-03",
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), res.TotalPortfolios)
		assert.Equal(t, "completed", res.Status)
	})
}
