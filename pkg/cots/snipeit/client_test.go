package snipeit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnipeITClient_CheckoutAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hardware/10/checkout", r.URL.Path)
		assert.Equal(t, "Bearer snipe-key", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","messages":"Asset 10 checked out successfully"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "snipe-key",
	})

	asset, err := client.CheckoutAsset(context.Background(), CheckoutRequest{
		AssetID:      10,
		AssignedUser: "emp-human-rick",
		Note:         "MacBook Pro provisioned for new hire",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(10), asset.ID)
	assert.Equal(t, "emp-human-rick", asset.AssignedTo)
	assert.Equal(t, "Deployed", asset.StatusLabel)
}

func TestSnipeITClient_CheckinAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hardware/10/checkin", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","messages":"Asset checked in"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "snipe-key",
	})

	asset, err := client.CheckinAsset(context.Background(), 10, CheckinRequest{
		AssetID: 10,
		Note:    "Employee separation equipment return",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(10), asset.ID)
	assert.Equal(t, "Ready to Deploy", asset.StatusLabel)
}

func TestSnipeITClient_CreateAssetRecoveryTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total":1,"rows":[{"id":10,"asset_tag":"MBP-001","name":"MacBook Pro 16"}]}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "snipe-key",
	})

	ticket, err := client.CreateAssetRecoveryTicket(context.Background(), "emp-human-rick", "Voluntary Resignation")
	require.NoError(t, err)
	assert.NotEmpty(t, ticket.TicketID)
	assert.Equal(t, "emp-human-rick", ticket.EmployeeID)
	assert.Equal(t, "Pending Recovery", ticket.Status)
	assert.Len(t, ticket.Assets, 1)
}
