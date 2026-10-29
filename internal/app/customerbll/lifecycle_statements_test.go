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
