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
