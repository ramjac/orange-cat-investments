package customerbll

import (
	"context"
	"errors"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	revokeFunc    func(ctx context.Context, in *coreinvestv1.RevokeAPIKeyRequest) (*coreinvestv1.RevokeAPIKeyResponse, error)
	lastRevokeReq *coreinvestv1.RevokeAPIKeyRequest
}

func (m *mockCoreInvestClient) RevokeAPIKey(ctx context.Context, in *coreinvestv1.RevokeAPIKeyRequest) (*coreinvestv1.RevokeAPIKeyResponse, error) {
	m.lastRevokeReq = in
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, in)
	}
	return &coreinvestv1.RevokeAPIKeyResponse{Status: "revoked"}, nil
}

func TestRevokeAPIKey_WithCoreInvestClient(t *testing.T) {
	mockCI := &mockCoreInvestClient{}
	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	ctx := context.Background()
	req := &customerbllv1.RevokeAPIKeyRequest{
		CustomerID: "cust-123",
		KeyID:      "key-456",
	}

	res, err := server.RevokeAPIKey(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "revoked", res.Status)

	require.NotNil(t, mockCI.lastRevokeReq)
	assert.Equal(t, "cust-123", mockCI.lastRevokeReq.CustomerID)
	assert.Equal(t, "key-456", mockCI.lastRevokeReq.KeyID)
}

func TestRevokeAPIKey_NilCoreInvestClient(t *testing.T) {
	server := NewServer(Config{})
	// Ensure ciClient is nil
	server.SetCoreInvestClient(nil)

	ctx := context.Background()
	req := &customerbllv1.RevokeAPIKeyRequest{
		CustomerID: "cust-789",
		KeyID:      "key-999",
	}

	res, err := server.RevokeAPIKey(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "revoked", res.Status)
}

func TestRevokeAPIKey_CoreInvestErrorHandled(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		revokeFunc: func(ctx context.Context, in *coreinvestv1.RevokeAPIKeyRequest) (*coreinvestv1.RevokeAPIKeyResponse, error) {
			return nil, errors.New("upstream gRPC connection failed")
		},
	}
	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	ctx := context.Background()
	req := &customerbllv1.RevokeAPIKeyRequest{
		CustomerID: "cust-err-1",
		KeyID:      "key-err-1",
	}

	res, err := server.RevokeAPIKey(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "revoked", res.Status)

	require.NotNil(t, mockCI.lastRevokeReq)
	assert.Equal(t, "cust-err-1", mockCI.lastRevokeReq.CustomerID)
	assert.Equal(t, "key-err-1", mockCI.lastRevokeReq.KeyID)
}

func TestRevokeAPIKey_EmptyRequest(t *testing.T) {
	mockCI := &mockCoreInvestClient{}
	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	ctx := context.Background()
	req := &customerbllv1.RevokeAPIKeyRequest{}

	res, err := server.RevokeAPIKey(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "revoked", res.Status)

	require.NotNil(t, mockCI.lastRevokeReq)
	assert.Empty(t, mockCI.lastRevokeReq.CustomerID)
	assert.Empty(t, mockCI.lastRevokeReq.KeyID)
}
