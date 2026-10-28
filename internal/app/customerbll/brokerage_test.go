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
