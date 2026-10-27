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

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient

	deleteWebhookFunc func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error)
	listWebhooksFunc   func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error)
	createWebhookFunc func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error)
}

func (m *mockCoreInvestClient) DeleteWebhook(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockCoreInvestClient) ListWebhooks(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockCoreInvestClient) CreateWebhook(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestDeleteWebhook_NilCoreInvestClient(t *testing.T) {
	s := NewServer(Config{})
	// Ensure ciClient is nil
	ctx := context.Background()

	req := &customerbllv1.DeleteWebhookRequest{
		CustomerID: "cust-101",
		WebhookID:  "wh-sub-101",
	}

	resp, err := s.DeleteWebhook(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "deleted", resp.Status)
}

func TestDeleteWebhook_CoreInvestSuccessWithStatus(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
			assert.Equal(t, "cust-202", in.CustomerID)
			assert.Equal(t, "wh-sub-202", in.WebhookID)
			return &coreinvestv1.DeleteWebhookResponse{
				Status: "deactivated",
			}, nil
		},
	}

	s := NewServer(Config{})
	s.SetCoreInvestClient(mockCI)

	ctx := context.Background()
	req := &customerbllv1.DeleteWebhookRequest{
		CustomerID: "cust-202",
		WebhookID:  "wh-sub-202",
	}

	resp, err := s.DeleteWebhook(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "deactivated", resp.Status)
}

func TestDeleteWebhook_CoreInvestSuccessWithDefaultStatus(t *testing.T) {
	t.Run("nil response from coreinvest", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
				return nil, nil
			},
		}

		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.DeleteWebhook(context.Background(), &customerbllv1.DeleteWebhookRequest{
			CustomerID: "cust-303",
			WebhookID:  "wh-sub-303",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "deleted", resp.Status)
	})

	t.Run("empty status string from coreinvest", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
				return &coreinvestv1.DeleteWebhookResponse{Status: ""}, nil
			},
		}

		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.DeleteWebhook(context.Background(), &customerbllv1.DeleteWebhookRequest{
			CustomerID: "cust-303",
			WebhookID:  "wh-sub-303",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "deleted", resp.Status)
	})
}

func TestDeleteWebhook_CoreInvestErrorPropagation(t *testing.T) {
	expectedErr := status.Error(codes.NotFound, "webhook subscription not found")

	mockCI := &mockCoreInvestClient{
		deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
			return nil, expectedErr
		},
	}

	s := NewServer(Config{})
	s.SetCoreInvestClient(mockCI)

	resp, err := s.DeleteWebhook(context.Background(), &customerbllv1.DeleteWebhookRequest{
		CustomerID: "cust-404",
		WebhookID:  "wh-nonexistent",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
}

func TestListWebhooks(t *testing.T) {
	t.Run("nil ciClient returns empty slice", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.ListWebhooks(context.Background(), &customerbllv1.ListWebhooksRequest{CustomerID: "cust-1"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Subscriptions)
	})

	t.Run("ciClient error returns empty slice", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
				return nil, errors.New("database connection failed")
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.ListWebhooks(context.Background(), &customerbllv1.ListWebhooksRequest{CustomerID: "cust-1"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Subscriptions)
	})

	t.Run("ciClient success returns webhooks", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
				return &coreinvestv1.ListWebhooksResponse{
					Webhooks: []*coreinvestv1.CustomerWebhook{
						{WebhookID: "sub-1", TargetURL: "https://example.com/wh1"},
					},
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.ListWebhooks(context.Background(), &customerbllv1.ListWebhooksRequest{CustomerID: "cust-1"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Subscriptions, 1)
		assert.Equal(t, "sub-1", resp.Subscriptions[0].WebhookID)
	})
}

func TestCreateWebhook(t *testing.T) {
	t.Run("nil ciClient fallback", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.CreateWebhook(context.Background(), &customerbllv1.CreateWebhookRequest{
			CustomerID: "cust-1",
			TargetURL:  "https://example.com/hooks",
			Events:     []string{"order.created"},
			Secret:     "secret-123",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Subscription)
		assert.Equal(t, "cust-1", resp.Subscription.CustomerID)
		assert.Equal(t, "https://example.com/hooks", resp.Subscription.TargetURL)
		assert.Equal(t, "secret-123", resp.Subscription.Secret)
		assert.Equal(t, "active", resp.Subscription.Status)
	})

	t.Run("ciClient success with empty secret fallback", func(t *testing.T) {
		mockCI := &mockCoreInvestClient{
			createWebhookFunc: func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
				return &coreinvestv1.CreateWebhookResponse{
					Webhook: &coreinvestv1.CustomerWebhook{
						WebhookID:  "sub-from-ci",
						CustomerID: in.CustomerID,
						TargetURL:  in.TargetURL,
						EventTypes: in.EventTypes,
						Status:     "active",
						Secret:     "", // Empty secret from ci
					},
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockCI)

		resp, err := s.CreateWebhook(context.Background(), &customerbllv1.CreateWebhookRequest{
			CustomerID: "cust-1",
			TargetURL:  "https://example.com/hooks",
			Events:     []string{"order.created"},
			Secret:     "provided-secret",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Subscription)
		assert.Equal(t, "sub-from-ci", resp.Subscription.WebhookID)
		assert.Equal(t, "provided-secret", resp.Subscription.Secret)
	})
}
