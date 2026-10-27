package customerbll

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPortfolio_DynamicHistoricalPerformance(t *testing.T) {
	// Setup mock HTTP server to simulate mock-broker / alpaca client
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/accounts/acc-123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "acc-123",
				"account_number": "OCI-000001",
				"status": "ACTIVE",
				"currency": "USD",
				"cash": "1000.00",
				"portfolio_value": "5000.00",
				"customer_name": "Test Cat"
			}`))
		case "/v1/trading/accounts/acc-123/positions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case "/v1/trading/accounts/acc-123/portfolio/history":
			w.Header().Set("Content-Type", "application/json")
			// Return timestamp and equity history
			_, _ = w.Write([]byte(`{
				"timestamp": [1767225600, 1769817600],
				"equity": [4000.5, 5000.0],
				"profit_loss": [0, 999.5],
				"profit_loss_pct": [0, 0.2498],
				"base_value": 4000.5,
				"timeframe": "1M"
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	brokerClient := alpaca.NewClient(alpaca.Config{
		BaseURL: ts.URL,
	})

	server := NewServer(Config{
		BrokerClient: brokerClient,
	})

	resp, err := server.GetPortfolio(context.Background(), &customerbllv1.GetPortfolioRequest{
		PortfolioID: "acc-123",
		CustomerID:  "cust-123",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "acc-123", resp.PortfolioID)
	assert.Equal(t, 5000.0, resp.TotalBalanceUSD)

	require.Len(t, resp.HistoricalPerformance, 2)
	assert.Equal(t, "2026-01-01", resp.HistoricalPerformance[0].Date)
	assert.Equal(t, 4000.5, resp.HistoricalPerformance[0].Valuation)
	assert.Equal(t, "2026-01-31", resp.HistoricalPerformance[1].Date)
	assert.Equal(t, 5000.0, resp.HistoricalPerformance[1].Valuation)
}

func TestGetPortfolio_HistoricalPerformanceFallback(t *testing.T) {
	// Setup mock HTTP server where portfolio history returns 404/error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/accounts/acc-456":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "acc-456",
				"account_number": "OCI-000002",
				"status": "ACTIVE",
				"currency": "USD",
				"cash": "2000.00",
				"portfolio_value": "10000.00",
				"customer_name": "Fallback Cat"
			}`))
		case "/v1/trading/accounts/acc-456/positions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case "/v1/trading/accounts/acc-456/portfolio/history":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "portfolio history unavailable"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	brokerClient := alpaca.NewClient(alpaca.Config{
		BaseURL: ts.URL,
	})

	server := NewServer(Config{
		BrokerClient: brokerClient,
	})

	resp, err := server.GetPortfolio(context.Background(), &customerbllv1.GetPortfolioRequest{
		PortfolioID: "acc-456",
		CustomerID:  "cust-456",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "acc-456", resp.PortfolioID)
	assert.Equal(t, 10000.0, resp.TotalBalanceUSD)

	// Should fallback to default 4 points calculated from current balance
	require.Len(t, resp.HistoricalPerformance, 4)
	assert.Equal(t, 7800.0, resp.HistoricalPerformance[0].Valuation)
	assert.Equal(t, 8500.0, resp.HistoricalPerformance[1].Valuation)
	assert.Equal(t, 9200.0, resp.HistoricalPerformance[2].Valuation)
	assert.Equal(t, 10000.0, resp.HistoricalPerformance[3].Valuation)
}
