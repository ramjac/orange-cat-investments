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
	createAPIKeyFn func(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error)
}

func (m *mockCoreInvestClient) CreateAPIKey(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
	if m.createAPIKeyFn != nil {
		return m.createAPIKeyFn(ctx, in)
	}
	return nil, errors.New("not implemented")
}

func TestCreateAPIKey_SuccessWithRawSecret(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		createAPIKeyFn: func(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
			assert.Equal(t, "cust-100", in.CustomerID)
			assert.Equal(t, "test-key-label", in.Label)
			assert.Equal(t, []string{"read", "trade"}, in.Scopes)
			return &coreinvestv1.CreateAPIKeyResponse{
				APIKey: &coreinvestv1.CustomerAPIKey{
					KeyID:      "key-grpc-123",
					CustomerID: in.CustomerID,
					Label:      in.Label,
					KeyPrefix:  "oci_live_",
					Status:     "active",
					Scopes:     in.Scopes,
				},
				RawSecret: "secret_grpc_raw_456",
			}, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockClient)

	req := &customerbllv1.CreateAPIKeyRequest{
		CustomerID:  "cust-100",
		Name:        "test-key-label",
		Permissions: []string{"read", "trade"},
		KeyPrefix:   "oci_live_",
		APIKey:      "raw_input_key",
	}

	res, err := server.CreateAPIKey(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Key)
	assert.Equal(t, "key-grpc-123", res.Key.KeyID)
	assert.Equal(t, "cust-100", res.Key.CustomerID)
	assert.Equal(t, "test-key-label", res.Key.Label)
	assert.Equal(t, "secret_grpc_raw_456", res.RawSecret)
}

func TestCreateAPIKey_SuccessWithEmptyRawSecretFallback(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		createAPIKeyFn: func(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
			return &coreinvestv1.CreateAPIKeyResponse{
				APIKey: &coreinvestv1.CustomerAPIKey{
					KeyID:      "key-grpc-456",
					CustomerID: in.CustomerID,
					Label:      in.Label,
					KeyPrefix:  "oci_live_",
					Status:     "active",
					Scopes:     in.Scopes,
				},
				RawSecret: "", // Empty raw secret from gRPC
			}, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockClient)

	req := &customerbllv1.CreateAPIKeyRequest{
		CustomerID:  "cust-101",
		Name:        "test-key-fallback-raw",
		Permissions: []string{"read"},
		KeyPrefix:   "oci_live_",
		APIKey:      "fallback_input_raw_secret",
	}

	res, err := server.CreateAPIKey(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Key)
	assert.Equal(t, "key-grpc-456", res.Key.KeyID)
	assert.Equal(t, "fallback_input_raw_secret", res.RawSecret)
}

func TestCreateAPIKey_ClientErrorFallback(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		createAPIKeyFn: func(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
			return nil, errors.New("gRPC service unavailable")
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockClient)

	req := &customerbllv1.CreateAPIKeyRequest{
		CustomerID:  "cust-102",
		Name:        "test-key-err-fallback",
		Permissions: []string{"trade"},
		KeyPrefix:   "oci_sim_",
		APIKey:      "fallback_secret_123",
	}

	res, err := server.CreateAPIKey(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Key)
	assert.Equal(t, "cust-102", res.Key.CustomerID)
	assert.Equal(t, "test-key-err-fallback", res.Key.Label)
	assert.Equal(t, "oci_sim_", res.Key.KeyPrefix)
	assert.Equal(t, "active", res.Key.Status)
	assert.Equal(t, []string{"trade"}, res.Key.Scopes)
	assert.Equal(t, "fallback_secret_123", res.RawSecret)
}

func TestCreateAPIKey_NilResponseFallback(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		createAPIKeyFn: func(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
			return nil, nil // No error but nil response
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockClient)

	req := &customerbllv1.CreateAPIKeyRequest{
		CustomerID:  "cust-103",
		Name:        "test-key-nil-res",
		Permissions: []string{"read"},
		KeyPrefix:   "oci_sim_",
		APIKey:      "fallback_secret_456",
	}

	res, err := server.CreateAPIKey(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Key)
	assert.Equal(t, "cust-103", res.Key.CustomerID)
	assert.Equal(t, "test-key-nil-res", res.Key.Label)
	assert.Equal(t, "fallback_secret_456", res.RawSecret)
}

func TestCreateAPIKey_NilClientFallback(t *testing.T) {
	server := NewServer(Config{}) // ciClient is nil

	req := &customerbllv1.CreateAPIKeyRequest{
		CustomerID:  "cust-104",
		Name:        "test-key-nil-client",
		Permissions: []string{"admin"},
		KeyPrefix:   "oci_test_",
		APIKey:      "local_secret_789",
	}

	res, err := server.CreateAPIKey(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Key)
	assert.Equal(t, "cust-104", res.Key.CustomerID)
	assert.Equal(t, "test-key-nil-client", res.Key.Label)
	assert.Equal(t, "oci_test_", res.Key.KeyPrefix)
	assert.Equal(t, "active", res.Key.Status)
	assert.Equal(t, []string{"admin"}, res.Key.Scopes)
	assert.Equal(t, "local_secret_789", res.RawSecret)
}
