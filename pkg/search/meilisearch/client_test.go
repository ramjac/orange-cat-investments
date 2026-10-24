package meilisearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeilisearchClient_InMemoryFallback(t *testing.T) {
	client := NewClient("http://unreachable-meili.local:7700", "test-key")

	ctx := context.Background()
	err := client.EnsureIndex(ctx, "employee_knowledge", "id")
	require.NoError(t, err)

	docs := []map[string]interface{}{
		{
			"id":       "sop-001",
			"title":    "Feline Laser Pointer Calibration SOP",
			"category": "sop",
			"content":  "Standard procedure for calibrating Class 2 visual stimulation emitters.",
		},
		{
			"id":       "vhr-garfield-01",
			"title":    "VHR Clinical Log: Garfield Lasagna Intake",
			"category": "vhr",
			"content":  "Subject consumed excess carbohydrates. Glucose levels elevated.",
		},
		{
			"id":       "ticket-ops-42",
			"title":    "Switch reboot needed in Habitat Perch Zone Alpha",
			"category": "ticket",
			"content":  "Cat fur in fan exhaust requires compressed air cleaning.",
		},
	}

	err = client.IndexDocuments(ctx, "employee_knowledge", docs)
	require.NoError(t, err)

	// Search for laser
	res, err := client.Search(ctx, "employee_knowledge", "Laser", SearchOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, res.TotalHits)
	assert.Equal(t, "sop-001", res.Hits[0]["id"])

	// Filter search
	res, err = client.Search(ctx, "employee_knowledge", "", SearchOptions{
		Filter: "category = 'vhr'",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, res.TotalHits)
	assert.Equal(t, "vhr-garfield-01", res.Hits[0]["id"])
}

func TestMeilisearchClient_MockHTTPServer(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"available"}`))
			return
		}
		if r.URL.Path == "/indexes/customer_catalog/search" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"hits": [
					{"id":"fund-zoomie","symbol":"ZOOM","title":"OCI Feline Behavioral Alpha Fund"}
				],
				"query": "alpha",
				"totalHits": 1,
				"processingTimeMs": 2
			}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL, "secret-key")
	ctx := context.Background()

	ok, err := client.Health(ctx)
	require.NoError(t, err)
	assert.True(t, ok)

	res, err := client.Search(ctx, "customer_catalog", "alpha", SearchOptions{Limit: 5})
	require.NoError(t, err)
	assert.Equal(t, 1, res.TotalHits)
	assert.Equal(t, "fund-zoomie", res.Hits[0]["id"])
}
