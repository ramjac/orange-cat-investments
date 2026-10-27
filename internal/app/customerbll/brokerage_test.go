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

	listWebhooksFunc   func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error)
	createWebhookFunc func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error)
	deleteWebhookFunc func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error)
}

func (m *mockCoreInvestClient) ListWebhooks(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc(ctx, in)
	}
	return &coreinvestv1.ListWebhooksResponse{}, nil
}

func (m *mockCoreInvestClient) CreateWebhook(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, in)
	}
	return &coreinvestv1.CreateWebhookResponse{}, nil
}

func (m *mockCoreInvestClient) DeleteWebhook(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, in)
	}
	return &coreinvestv1.DeleteWebhookResponse{}, nil
}

func TestListWebhooks_NilClient(t *testing.T) {
	s := &Server{}
	req := &customerbllv1.ListWebhooksRequest{CustomerID: "cust-123"}

	res, err := s.ListWebhooks(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Subscriptions)
}

func TestListWebhooks_ClientError(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
			assert.Equal(t, "cust-123", in.CustomerID)
			return nil, errors.New("gRPC core-invest service unavailable")
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)

	req := &customerbllv1.ListWebhooksRequest{CustomerID: "cust-123"}
	res, err := s.ListWebhooks(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Subscriptions)
}

func TestListWebhooks_Success(t *testing.T) {
	expectedWebhooks := []*coreinvestv1.CustomerWebhook{
		{
			WebhookID:  "sub-1001",
			CustomerID: "cust-123",
			TargetURL:  "https://example.com/webhook",
			EventTypes: []string{"events.trade.executed.v1", "events.order.created.v1"},
			Status:     "active",
			Secret:     "whsec_0123456789abcdef",
			CreatedAt:  "2026-03-30T10:00:00Z",
		},
		{
			WebhookID:  "sub-1002",
			CustomerID: "cust-123",
			TargetURL:  "https://example.com/webhook2",
			EventTypes: []string{"events.deposit.settled.v1"},
			Status:     "active",
			Secret:     "whsec_fedcba9876543210",
			CreatedAt:  "2026-03-30T11:00:00Z",
		},
	}

	mockClient := &mockCoreInvestClient{
		listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
			assert.Equal(t, "cust-123", in.CustomerID)
			return &coreinvestv1.ListWebhooksResponse{Webhooks: expectedWebhooks}, nil
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)

	req := &customerbllv1.ListWebhooksRequest{CustomerID: "cust-123"}
	res, err := s.ListWebhooks(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Subscriptions, 2)
	assert.Equal(t, "sub-1001", res.Subscriptions[0].WebhookID)
	assert.Equal(t, "sub-1002", res.Subscriptions[1].WebhookID)
	assert.Equal(t, "https://example.com/webhook", res.Subscriptions[0].TargetURL)
	assert.Equal(t, []string{"events.trade.executed.v1", "events.order.created.v1"}, res.Subscriptions[0].EventTypes)
}

func TestCreateWebhook(t *testing.T) {
	t.Run("nil ciClient fallback", func(t *testing.T) {
		s := &Server{}
		req := &customerbllv1.CreateWebhookRequest{
			CustomerID: "cust-123",
			TargetURL:  "https://example.com/hook",
			Events:     []string{"events.trade.executed.v1"},
			Secret:     "whsec_test123",
		}

		res, err := s.CreateWebhook(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Subscription)
		assert.Equal(t, "cust-123", res.Subscription.CustomerID)
		assert.Equal(t, "https://example.com/hook", res.Subscription.TargetURL)
		assert.Equal(t, []string{"events.trade.executed.v1"}, res.Subscription.EventTypes)
		assert.Equal(t, "active", res.Subscription.Status)
		assert.Equal(t, "whsec_test123", res.Subscription.Secret)
		assert.NotEmpty(t, res.Subscription.WebhookID)
	})

	t.Run("configured ciClient success", func(t *testing.T) {
		mockClient := &mockCoreInvestClient{
			createWebhookFunc: func(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
				assert.Equal(t, "cust-123", in.CustomerID)
				assert.Equal(t, "https://example.com/hook", in.TargetURL)
				assert.Equal(t, []string{"events.trade.executed.v1"}, in.EventTypes)
				return &coreinvestv1.CreateWebhookResponse{
					Webhook: &coreinvestv1.CustomerWebhook{
						WebhookID:  "sub-999",
						CustomerID: in.CustomerID,
						TargetURL:  in.TargetURL,
						EventTypes: in.EventTypes,
						Status:     "active",
						Secret:     "whsec_from_ci",
					},
				}, nil
			},
		}

		s := &Server{}
		s.SetCoreInvestClient(mockClient)

		req := &customerbllv1.CreateWebhookRequest{
			CustomerID: "cust-123",
			TargetURL:  "https://example.com/hook",
			Events:     []string{"events.trade.executed.v1"},
			Secret:     "whsec_test123",
		}

		res, err := s.CreateWebhook(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Subscription)
		assert.Equal(t, "sub-999", res.Subscription.WebhookID)
		assert.Equal(t, "whsec_from_ci", res.Subscription.Secret)
	})
}

func TestDeleteWebhook(t *testing.T) {
	t.Run("nil ciClient fallback", func(t *testing.T) {
		s := &Server{}
		req := &customerbllv1.DeleteWebhookRequest{
			CustomerID: "cust-123",
			WebhookID:  "sub-1001",
		}

		res, err := s.DeleteWebhook(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Success)
		assert.Equal(t, "deleted", res.Status)
	})

	t.Run("configured ciClient success", func(t *testing.T) {
		mockClient := &mockCoreInvestClient{
			deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
				assert.Equal(t, "cust-123", in.CustomerID)
				assert.Equal(t, "sub-1001", in.WebhookID)
				return &coreinvestv1.DeleteWebhookResponse{
					Status: "deleted",
				}, nil
			},
		}

		s := &Server{}
		s.SetCoreInvestClient(mockClient)

		req := &customerbllv1.DeleteWebhookRequest{
			CustomerID: "cust-123",
			WebhookID:  "sub-1001",
		}

		res, err := s.DeleteWebhook(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Success)
		assert.Equal(t, "deleted", res.Status)
	})

	t.Run("configured ciClient error", func(t *testing.T) {
		mockClient := &mockCoreInvestClient{
			deleteWebhookFunc: func(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
				return nil, errors.New("webhook not found")
			},
		}

		s := &Server{}
		s.SetCoreInvestClient(mockClient)

		req := &customerbllv1.DeleteWebhookRequest{
			CustomerID: "cust-123",
			WebhookID:  "sub-invalid",
		}

		res, err := s.DeleteWebhook(context.Background(), req)
		require.Error(t, err)
		assert.Nil(t, res)
	})
}
