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

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	listAPIKeysFunc func(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error)
}

func (m *mockCoreInvestClient) ListAPIKeys(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error) {
	if m.listAPIKeysFunc != nil {
		return m.listAPIKeysFunc(ctx, in)
	}
	return &coreinvestv1.ListAPIKeysResponse{APIKeys: []*coreinvestv1.CustomerAPIKey{}}, nil
}

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
