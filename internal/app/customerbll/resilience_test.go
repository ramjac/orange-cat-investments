package customerbll

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resilienceStubCoreInvest struct {
	coreinvestv1.CoreInvestServiceClient
}

func (s *resilienceStubCoreInvest) GetPortfolio(ctx context.Context, in *coreinvestv1.GetPortfolioRequest) (*coreinvestv1.GetPortfolioResponse, error) {
	return &coreinvestv1.GetPortfolioResponse{
		Portfolio: &coreinvestv1.Portfolio{
			PortfolioID:     "port-cust-1",
			CustomerID:      "cust-1",
			AccountName:     "Fallback Core-Invest Portfolio",
			TotalBalanceUSD: 12345.67,
			CashBalanceUSD:  2000.00,
			Status:          "active",
		},
	}, nil
}

func (s *resilienceStubCoreInvest) ListPortfolioSnapshots(ctx context.Context, in *coreinvestv1.ListPortfolioSnapshotsRequest) (*coreinvestv1.ListPortfolioSnapshotsResponse, error) {
	return &coreinvestv1.ListPortfolioSnapshotsResponse{}, nil
}

func TestGetPortfolio_ShortCircuitsSlowBrokerTimeout(t *testing.T) {
	var brokerHits atomic.Int32
	slowBroker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		brokerHits.Add(1)
		select {
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer slowBroker.Close()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	s := NewServer(Config{
		BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: slowBroker.URL}),
		UpstreamTimeout: 150 * time.Millisecond,
		Logger:          logger,
	})
	s.SetCoreInvestClient(&resilienceStubCoreInvest{})

	start := time.Now()
	resp, err := s.GetPortfolio(context.Background(), &customerbllv1.GetPortfolioRequest{
		CustomerID: "cust-1",
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Fallback Core-Invest Portfolio", resp.AccountName)
	assert.Equal(t, int32(1), brokerHits.Load(), "GetPortfolio must short-circuit after first broker timeout instead of chaining 3 sequential timeouts")
	assert.Less(t, elapsed, 450*time.Millisecond, "GetPortfolio should fall back after a single bounded timeout")
	assert.NotContains(t, logBuf.String(), "alpaca", "BLL logs must not leak vendor name alpaca")
}
