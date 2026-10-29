package customerbll

import (
	"context"
	"errors"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestListStatements_Success(t *testing.T) {
	expectedStatements := []*coreinvestv1.CustomerStatement{
		{
			StatementID:   "stmt-101",
			CustomerID:    "cust-123",
			PortfolioID:   "port-123",
			StatementType: "monthly",
			Period:        "2026-01",
			Status:        "generated",
		},
		{
			StatementID:   "stmt-102",
			CustomerID:    "cust-123",
			PortfolioID:   "port-123",
			StatementType: "monthly",
			Period:        "2026-02",
			Status:        "generated",
		},
	}

	mockCI := &mockCoreInvestClient{
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			assert.Equal(t, "cust-123", in.CustomerID)
			return &coreinvestv1.ListStatementsResponse{
				Statements: expectedStatements,
			}, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Len(t, res.Statements, 2)
	assert.Equal(t, "stmt-101", res.Statements[0].StatementID)
	assert.Equal(t, "stmt-102", res.Statements[1].StatementID)
}

func TestListStatements_CoreInvestErrorFallback(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			return nil, errors.New("database connection failed")
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

func TestListStatements_NilResponseFromCoreInvest(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			return nil, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

func TestListStatements_NilCoreInvestClient(t *testing.T) {
	server := NewServer(Config{})

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

func TestListStatements_NilRequest(t *testing.T) {
	server := NewServer(Config{})

	res, err := server.ListStatements(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

type mockGDPRCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	executeGDPRErasureFunc func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error)
}

func (m *mockGDPRCoreInvestClient) ExecuteGDPRErasure(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
	if m.executeGDPRErasureFunc != nil {
		return m.executeGDPRErasureFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestExecuteGDPRErasure(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.ExecuteGDPRErasure(context.Background(), nil)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Nil(t, resp)
	})

	t.Run("empty customer_id", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "",
			Email:      "user@example.com",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Nil(t, resp)
	})

	t.Run("fallback mode without ciClient", func(t *testing.T) {
		s := NewServer(Config{})
		req := &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-12345",
			Email:      "test@example.com",
		}
		resp, err := s.ExecuteGDPRErasure(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Certificate)

		cert := resp.Certificate
		assert.Equal(t, "cust-12345", cert.CustomerID)
		assert.NotEmpty(t, cert.CertificateID)
		assert.NotEmpty(t, cert.CustomerIDHash)
		assert.NotEmpty(t, cert.CustomerEmailHash)
		assert.Equal(t, "erased", cert.Status)
		assert.Contains(t, cert.LegalBasis, "GDPR Article 17")
		assert.NotEmpty(t, cert.AnonymizedTables)
		assert.Equal(t, int32(1), cert.SessionsPurgedCount)
		assert.Equal(t, int32(4), cert.RecordsScrubbedCount)
	})

	t.Run("integrated mode with ciClient success", func(t *testing.T) {
		mockCI := &mockGDPRCoreInvestClient{
			executeGDPRErasureFunc: func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
				assert.Equal(t, "cust-777", in.CustomerID)
				assert.Equal(t, "alice@example.com", in.Email)
				return &coreinvestv1.ExecuteGDPRErasureResponse{
					Certificate: &coreinvestv1.GDPRErasureCertificate{
						CertificateID: "cert-ci-001",
						CustomerID:    in.CustomerID,
						Status:        "erased",
					},
				}, nil
			},
		}

		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-777",
			Email:      "alice@example.com",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Certificate)
		assert.Equal(t, "cert-ci-001", resp.Certificate.CertificateID)
		assert.Equal(t, "cust-777", resp.Certificate.CustomerID)
	})

	t.Run("integrated mode with ciClient error propagation", func(t *testing.T) {
		mockCI := &mockGDPRCoreInvestClient{
			executeGDPRErasureFunc: func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
				return nil, errors.New("database connection failed")
			},
		}

		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.ExecuteGDPRErasure(context.Background(), &customerbllv1.ExecuteGDPRErasureRequest{
			CustomerID: "cust-err",
			Email:      "err@example.com",
		})
		require.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "gdpr erasure failed")
	})
}

func TestDownloadStatement_ValidationErrors(t *testing.T) {
	srv := NewServer(Config{})
	ctx := context.Background()

	// 1. Nil request
	res, err := srv.DownloadStatement(ctx, nil)
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "statement_id is required")

	// 2. Empty statement_id
	res, err = srv.DownloadStatement(ctx, &customerbllv1.DownloadStatementRequest{
		CustomerID:  "cust-123",
		StatementID: "",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "statement_id is required")
}

func TestDownloadStatement_MockFallback(t *testing.T) {
	srv := NewServer(Config{})
	ctx := context.Background()

	res, err := srv.DownloadStatement(ctx, &customerbllv1.DownloadStatementRequest{
		CustomerID:  "cust-123",
		StatementID: "stmt-mock-99",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "application/pdf", res.ContentType)
	assert.Equal(t, "statement-stmt-mock-99.pdf", res.FileName)
	assert.Equal(t, []byte("%PDF-1.4 OCI Customer Statement Document Mock"), res.PDFBytes)
	assert.Nil(t, res.Statement)
}

func TestDownloadStatement_CoreInvestSuccess(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		getStatementDocFunc: func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
			assert.Equal(t, "cust-777", in.CustomerID)
			assert.Equal(t, "stmt-abc-123", in.StatementID)

			return &coreinvestv1.GetStatementDocumentResponse{
				Statement: &coreinvestv1.CustomerStatement{
					StatementID:   "stmt-abc-123",
					CustomerID:    "cust-777",
					PortfolioID:   "port-777",
					StatementType: "monthly",
					Period:        "2026-03",
					Status:        "generated",
				},
				PDFBytes: []byte("%PDF-1.4 Real Dynamic Statement Content"),
			}, nil
		},
	}

	srv := NewServer(Config{})
	srv.SetCoreInvestClient(mockCI)
	ctx := context.Background()

	res, err := srv.DownloadStatement(ctx, &customerbllv1.DownloadStatementRequest{
		CustomerID:  "cust-777",
		StatementID: "stmt-abc-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "application/pdf", res.ContentType)
	assert.Equal(t, "statement-stmt-abc-123.pdf", res.FileName)
	assert.Equal(t, []byte("%PDF-1.4 Real Dynamic Statement Content"), res.PDFBytes)
	require.NotNil(t, res.Statement)
	assert.Equal(t, "stmt-abc-123", res.Statement.StatementID)
	assert.Equal(t, "cust-777", res.Statement.CustomerID)
	assert.Equal(t, "monthly", res.Statement.StatementType)
}

func TestDownloadStatement_CoreInvestError(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		getStatementDocFunc: func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
			return nil, errors.New("database connection timeout")
		},
	}

	srv := NewServer(Config{})
	srv.SetCoreInvestClient(mockCI)
	ctx := context.Background()

	res, err := srv.DownloadStatement(ctx, &customerbllv1.DownloadStatementRequest{
		CustomerID:  "cust-777",
		StatementID: "stmt-nonexistent",
	})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "failed to retrieve statement: database connection timeout")
}

func TestRegisterCustomer_Lifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. Invalid input (nil request)
	srv := NewServer(Config{})
	_, err := srv.RegisterCustomer(ctx, nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// 2. Mock fallback (ciClient == nil)
	resMock, err := srv.RegisterCustomer(ctx, &customerbllv1.RegisterCustomerRequest{
		FullName: "Tabby Cat",
		Email:    "tabby@oci.local",
	})
	require.NoError(t, err)
	require.NotNil(t, resMock.Result)
	assert.Equal(t, "Tabby Cat", resMock.Result.Name)
	assert.Equal(t, "active", resMock.Result.Status)

	// 3. CoreInvest connected success
	mockCI := &mockCoreInvestClient{
		registerCustFunc: func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
			return &coreinvestv1.RegisterCustomerResponse{
				CustomerID:  "cust-registered-01",
				Name:        in.Name,
				Email:       in.Email,
				PortfolioID: "port-registered-01",
				Status:      "active",
			}, nil
		},
	}
	srv.SetCoreInvestClient(mockCI)
	resCI, err := srv.RegisterCustomer(ctx, &customerbllv1.RegisterCustomerRequest{
		FullName: "Tabby Cat",
		Email:    "tabby@oci.local",
	})
	require.NoError(t, err)
	assert.Equal(t, "cust-registered-01", resCI.Result.CustomerID)
}

func TestCloseAccount_Lifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. Invalid input (empty customer_id)
	srv := NewServer(Config{})
	_, err := srv.CloseAccount(ctx, &customerbllv1.CloseAccountRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// 2. Mock fallback
	resMock, err := srv.CloseAccount(ctx, &customerbllv1.CloseAccountRequest{
		CustomerID:           "cust-99",
		PortfolioID:          "port-99",
		DestinationAccountID: "acc-dest-99",
	})
	require.NoError(t, err)
	assert.Equal(t, "closed", resMock.Result.AccountStatus)

	// 3. CoreInvest connected
	mockCI := &mockCoreInvestClient{
		closeAccountFunc: func(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
			return &coreinvestv1.CloseAndLiquidateAccountResponse{
				CustomerID:    in.CustomerID,
				AccountStatus: "closed",
			}, nil
		},
	}
	srv.SetCoreInvestClient(mockCI)
	resCI, err := srv.CloseAccount(ctx, &customerbllv1.CloseAccountRequest{CustomerID: "cust-99"})
	require.NoError(t, err)
	assert.Equal(t, "closed", resCI.Result.AccountStatus)
}

func TestGDPRErasure_Lifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. ExecuteGDPR - Validation
	srv := NewServer(Config{})
	_, err := srv.ExecuteGDPRErasure(ctx, &customerbllv1.ExecuteGDPRErasureRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// 2. ExecuteGDPR - Mock fallback
	resMock, err := srv.ExecuteGDPRErasure(ctx, &customerbllv1.ExecuteGDPRErasureRequest{
		CustomerID: "cust-gdpr-1",
		Email:      "privacy@oci.local",
	})
	require.NoError(t, err)
	assert.Equal(t, "erased", resMock.Certificate.Status)

	// 3. GetGDPRCertificate - Validation
	_, err = srv.GetGDPRCertificate(ctx, &customerbllv1.GetGDPRCertificateRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// 4. GetGDPRCertificate - Mock fallback
	resCert, err := srv.GetGDPRCertificate(ctx, &customerbllv1.GetGDPRCertificateRequest{
		CertificateID: "cert-001",
		CustomerID:    "cust-gdpr-1",
		Email:         "privacy@oci.local",
	})
	require.NoError(t, err)
	assert.Equal(t, "cert-001", resCert.Certificate.CertificateID)
	assert.NotEmpty(t, resCert.PDFBytes)
}

func TestGenerateAndListStatements_Lifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. GenerateStatement validation
	srv := NewServer(Config{})
	_, err := srv.GenerateStatement(ctx, nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// 2. GenerateStatement - Mock fallback
	resGen, err := srv.GenerateStatement(ctx, &customerbllv1.GenerateStatementRequest{
		CustomerID:  "cust-stmt-1",
		PortfolioID: "port-stmt-1",
		Type:        "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "generated", resGen.Statement.Status)

	// 3. GenerateStatement - CoreInvest monthly & 1099b
	mockCI := &mockCoreInvestClient{
		generateMonthlyFunc: func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
			return &coreinvestv1.GenerateMonthlyStatementResponse{
				Statement: &coreinvestv1.CustomerStatement{StatementID: "stmt-m-1"},
				PDFBytes:  []byte("%PDF-monthly"),
			}, nil
		},
		generate1099BFunc: func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
			return &coreinvestv1.GenerateForm1099BResponse{
				Statement: &coreinvestv1.CustomerStatement{StatementID: "stmt-1099-1"},
				PDFBytes:  []byte("%PDF-1099b"),
			}, nil
		},
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			return &coreinvestv1.ListStatementsResponse{
				Statements: []*coreinvestv1.CustomerStatement{
					{StatementID: "stmt-m-1"},
				},
			}, nil
		},
	}
	srv.SetCoreInvestClient(mockCI)

	resM, err := srv.GenerateStatement(ctx, &customerbllv1.GenerateStatementRequest{Type: "monthly"})
	require.NoError(t, err)
	assert.Equal(t, "stmt-m-1", resM.Statement.StatementID)

	res1099, err := srv.GenerateStatement(ctx, &customerbllv1.GenerateStatementRequest{Type: "form_1099b"})
	require.NoError(t, err)
	assert.Equal(t, "stmt-1099-1", res1099.Statement.StatementID)

	resList, err := srv.ListStatements(ctx, &customerbllv1.ListStatementsRequest{CustomerID: "cust-stmt-1"})
	require.NoError(t, err)
	assert.Len(t, resList.Statements, 1)

	// 4. GenerateBatchStatements - CoreInvest nil check
	srvNoCI := NewServer(Config{})
	_, err = srvNoCI.GenerateBatchStatements(ctx, &customerbllv1.GenerateBatchStatementsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}
