package customerbll

import (
	"context"
	"errors"
	"testing"
	"time"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAPIKeys(t *testing.T) {
	ctx := context.Background()

	t.Run("nil ciClient returns empty keys list and no error", func(t *testing.T) {
		srv := NewServer(Config{})
		res, err := srv.ListAPIKeys(ctx, &customerbllv1.ListAPIKeysRequest{
			CustomerID: "cust-123",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Keys)
	})

	t.Run("ciClient returns error returns empty keys list and no error", func(t *testing.T) {
		mockClient := &mockCoreInvestClient{
			listAPIKeysFunc: func(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error) {
				assert.Equal(t, "cust-123", in.CustomerID)
				return nil, errors.New("gRPC connection failure")
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockClient)

		res, err := srv.ListAPIKeys(ctx, &customerbllv1.ListAPIKeysRequest{
			CustomerID: "cust-123",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Keys)
	})

	t.Run("ciClient returns keys successfully", func(t *testing.T) {
		expectedKeys := []*coreinvestv1.CustomerAPIKey{
			{
				KeyID:      "key-01",
				CustomerID: "cust-123",
				Label:      "Trading Bot",
				KeyPrefix:  "oci_live_",
				Status:     "active",
				Scopes:     []string{"read", "trade"},
				CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			},
			{
				KeyID:      "key-02",
				CustomerID: "cust-123",
				Label:      "Reporting App",
				KeyPrefix:  "oci_read_",
				Status:     "active",
				Scopes:     []string{"read"},
				CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			},
		}

		mockClient := &mockCoreInvestClient{
			listAPIKeysFunc: func(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error) {
				assert.Equal(t, "cust-123", in.CustomerID)
				return &coreinvestv1.ListAPIKeysResponse{
					APIKeys: expectedKeys,
				}, nil
			},
		}
		srv := NewServer(Config{})
		srv.SetCoreInvestClient(mockClient)

		res, err := srv.ListAPIKeys(ctx, &customerbllv1.ListAPIKeysRequest{
			CustomerID: "cust-123",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Keys, 2)
		assert.Equal(t, "key-01", res.Keys[0].KeyID)
		assert.Equal(t, "Trading Bot", res.Keys[0].Label)
		assert.Equal(t, "key-02", res.Keys[1].KeyID)
		assert.Equal(t, "Reporting App", res.Keys[1].Label)
	})
}

type mockCoreInvestForWebhook struct {
	coreinvestv1.CoreInvestServiceClient
	createWebhookFunc func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error)
	listWebhooksFunc  func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error)
	deleteWebhookFunc func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error)
}

func (m *mockCoreInvestForWebhook) CreateWebhook(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestForWebhook) ListWebhooks(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestForWebhook) DeleteWebhook(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, in)
	}
	return nil, nil
}

func TestCreateWebhook_WithCoreInvestClient_NonEmptySecret(t *testing.T) {
	ctx := context.Background()
	mockCI := &mockCoreInvestForWebhook{
		createWebhookFunc: func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
			assert.Equal(t, "cust-101", in.CustomerID)
			assert.Equal(t, "https://example.com/hooks/trade", in.TargetURL)
			assert.Equal(t, []string{"trade.executed", "order.filled"}, in.EventTypes)
			return &coreinvestv1.CreateWebhookResponse{
				Webhook: &coreinvestv1.CustomerWebhook{
					WebhookID:  "wh-grpc-001",
					CustomerID: in.CustomerID,
					TargetURL:  in.TargetURL,
					EventTypes: in.EventTypes,
					Secret:     "whsec_backend_generated_secret_12345678",
					Status:     "active",
					CreatedAt:  "2026-03-31T12:00:00Z",
				},
			}, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	req := &customerbllv1.CreateWebhookRequest{
		CustomerID: "cust-101",
		TargetURL:  "https://example.com/hooks/trade",
		Events:     []string{"trade.executed", "order.filled"},
		Secret:     "whsec_input_secret_8888",
	}

	res, err := server.CreateWebhook(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Subscription)

	assert.Equal(t, "wh-grpc-001", res.Subscription.WebhookID)
	assert.Equal(t, "cust-101", res.Subscription.CustomerID)
	assert.Equal(t, "https://example.com/hooks/trade", res.Subscription.TargetURL)
	assert.Equal(t, []string{"trade.executed", "order.filled"}, res.Subscription.EventTypes)
	assert.Equal(t, "whsec_backend_generated_secret_12345678", res.Subscription.Secret)
	assert.Equal(t, "active", res.Subscription.Status)
}

func TestCreateWebhook_WithCoreInvestClient_EmptySecretFallbackToInput(t *testing.T) {
	ctx := context.Background()
	mockCI := &mockCoreInvestForWebhook{
		createWebhookFunc: func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
			return &coreinvestv1.CreateWebhookResponse{
				Webhook: &coreinvestv1.CustomerWebhook{
					WebhookID:  "wh-grpc-002",
					CustomerID: in.CustomerID,
					TargetURL:  in.TargetURL,
					EventTypes: in.EventTypes,
					Secret:     "", // empty secret returned by gRPC client
					Status:     "active",
				},
			}, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	req := &customerbllv1.CreateWebhookRequest{
		CustomerID: "cust-102",
		TargetURL:  "https://example.com/hooks/events",
		Events:     []string{"deposit.settled"},
		Secret:     "whsec_fallback_input_secret_9999",
	}

	res, err := server.CreateWebhook(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Subscription)

	assert.Equal(t, "wh-grpc-002", res.Subscription.WebhookID)
	assert.Equal(t, "whsec_fallback_input_secret_9999", res.Subscription.Secret)
}

func TestCreateWebhook_WithCoreInvestClient_ErrorFallback(t *testing.T) {
	ctx := context.Background()
	mockCI := &mockCoreInvestForWebhook{
		createWebhookFunc: func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
			return nil, errors.New("gRPC service unavailable")
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	req := &customerbllv1.CreateWebhookRequest{
		CustomerID: "cust-103",
		TargetURL:  "https://example.com/hooks/err",
		Events:     []string{"order.canceled"},
		Secret:     "whsec_local_secret_1111",
	}

	res, err := server.CreateWebhook(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Subscription)

	assert.Contains(t, res.Subscription.WebhookID, "sub-")
	assert.Equal(t, "cust-103", res.Subscription.CustomerID)
	assert.Equal(t, "https://example.com/hooks/err", res.Subscription.TargetURL)
	assert.Equal(t, []string{"order.canceled"}, res.Subscription.EventTypes)
	assert.Equal(t, "whsec_local_secret_1111", res.Subscription.Secret)
	assert.Equal(t, "active", res.Subscription.Status)
	assert.NotEmpty(t, res.Subscription.CreatedAt)
}

func TestCreateWebhook_NilCoreInvestClient_Fallback(t *testing.T) {
	ctx := context.Background()
	server := NewServer(Config{}) // s.ciClient is nil

	req := &customerbllv1.CreateWebhookRequest{
		CustomerID: "cust-104",
		TargetURL:  "https://example.com/hooks/nil",
		Events:     []string{"account.updated"},
		Secret:     "whsec_nil_secret_2222",
	}

	res, err := server.CreateWebhook(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Subscription)

	assert.Contains(t, res.Subscription.WebhookID, "sub-")
	assert.Equal(t, "cust-104", res.Subscription.CustomerID)
	assert.Equal(t, "https://example.com/hooks/nil", res.Subscription.TargetURL)
	assert.Equal(t, []string{"account.updated"}, res.Subscription.EventTypes)
	assert.Equal(t, "whsec_nil_secret_2222", res.Subscription.Secret)
	assert.Equal(t, "active", res.Subscription.Status)
	assert.NotEmpty(t, res.Subscription.CreatedAt)
}

func TestListWebhooks(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil Client", func(t *testing.T) {
		server := NewServer(Config{})
		res, err := server.ListWebhooks(ctx, &customerbllv1.ListWebhooksRequest{CustomerID: "cust-201"})
		require.NoError(t, err)
		assert.Empty(t, res.Subscriptions)
	})

	t.Run("Success Client", func(t *testing.T) {
		mockCI := &mockCoreInvestForWebhook{
			listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
				return &coreinvestv1.ListWebhooksResponse{
					Webhooks: []*coreinvestv1.CustomerWebhook{
						{WebhookID: "sub-1", CustomerID: in.CustomerID, TargetURL: "https://example.com/1"},
						{WebhookID: "sub-2", CustomerID: in.CustomerID, TargetURL: "https://example.com/2"},
					},
				}, nil
			},
		}
		server := NewServer(Config{})
		server.SetCoreInvestClient(mockCI)

		res, err := server.ListWebhooks(ctx, &customerbllv1.ListWebhooksRequest{CustomerID: "cust-201"})
		require.NoError(t, err)
		require.Len(t, res.Subscriptions, 2)
		assert.Equal(t, "sub-1", res.Subscriptions[0].WebhookID)
		assert.Equal(t, "sub-2", res.Subscriptions[1].WebhookID)
	})

	t.Run("Error Client", func(t *testing.T) {
		mockCI := &mockCoreInvestForWebhook{
			listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
				return nil, errors.New("db error")
			},
		}
		server := NewServer(Config{})
		server.SetCoreInvestClient(mockCI)

		res, err := server.ListWebhooks(ctx, &customerbllv1.ListWebhooksRequest{CustomerID: "cust-201"})
		require.NoError(t, err)
		assert.Empty(t, res.Subscriptions)
	})
}

func TestDeleteWebhook(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil Client", func(t *testing.T) {
		server := NewServer(Config{})
		res, err := server.DeleteWebhook(ctx, &customerbllv1.DeleteWebhookRequest{CustomerID: "cust-301", WebhookID: "sub-301"})
		require.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, "deleted", res.Status)
	})

	t.Run("Success Client", func(t *testing.T) {
		mockCI := &mockCoreInvestForWebhook{
			deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
				return &coreinvestv1.DeleteWebhookResponse{Status: "deleted"}, nil
			},
		}
		server := NewServer(Config{})
		server.SetCoreInvestClient(mockCI)

		res, err := server.DeleteWebhook(ctx, &customerbllv1.DeleteWebhookRequest{CustomerID: "cust-301", WebhookID: "sub-301"})
		require.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, "deleted", res.Status)
	})

	t.Run("Error Client", func(t *testing.T) {
		mockCI := &mockCoreInvestForWebhook{
			deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
				return nil, errors.New("webhook not found")
			},
		}
		server := NewServer(Config{})
		server.SetCoreInvestClient(mockCI)

		res, err := server.DeleteWebhook(ctx, &customerbllv1.DeleteWebhookRequest{CustomerID: "cust-301", WebhookID: "sub-invalid"})
		require.Error(t, err)
		assert.Nil(t, res)
	})
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
