cat << 'PATCH' > brokerage_test.patch
<<<<<<< SEARCH
		res, err := server.ListWebhooks(ctx, &customerbllv1.ListWebhooksRequest{CustomerID: "cust-err"})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Subscriptions)
	})
}

func TestDeleteWebhook(t *testing.T) {
=======
		res, err := server.ListWebhooks(ctx, &customerbllv1.ListWebhooksRequest{CustomerID: "cust-err"})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Subscriptions)
	})

	t.Run("Nil Request", func(t *testing.T) {
		mockCI := &mockCoreInvestForWebhook{
			listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
				return &coreinvestv1.ListWebhooksResponse{Webhooks: []*coreinvestv1.CustomerWebhook{}}, nil
			},
		}
		server := NewServer(Config{})
		server.SetCoreInvestClient(mockCI)

		res, err := server.ListWebhooks(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Subscriptions)
	})

	t.Run("Nil Response from Client", func(t *testing.T) {
		mockCI := &mockCoreInvestForWebhook{
			listWebhooksFunc: func(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
				return nil, nil // Nil response, no error
			},
		}
		server := NewServer(Config{})
		server.SetCoreInvestClient(mockCI)

		res, err := server.ListWebhooks(ctx, &customerbllv1.ListWebhooksRequest{CustomerID: "cust-nil-res"})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Subscriptions)
	})
}

func TestDeleteWebhook(t *testing.T) {
PATCH

patch internal/app/customerbll/brokerage_test.go < brokerage_test.patch
