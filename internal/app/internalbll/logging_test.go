package internalbll

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubFailingFacilitiesClient struct {
	facilitiesv1.FacilitiesServiceClient
	err error
}

func (s *stubFailingFacilitiesClient) ListAssets(ctx context.Context, in *facilitiesv1.ListAssetsRequest) (*facilitiesv1.ListAssetsResponse, error) {
	return nil, s.err
}

type stubFailingOpsClient struct {
	opsv1.OpsServiceClient
	err error
}

func (s *stubFailingOpsClient) CreateTicket(ctx context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	return nil, s.err
}

func TestInternalBLL_LogsDomainAPIErrors(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	srv := NewServer(
		nil,
		&stubFailingFacilitiesClient{err: status.Error(codes.Unavailable, "facilities-api down")},
		&stubFailingOpsClient{err: status.Error(codes.Unavailable, "ops-api down")},
		nil,
		logger,
	)

	_, err := srv.ListAssets(context.Background(), &facilitiesv1.ListAssetsRequest{})
	require.Error(t, err)
	assert.Contains(t, logBuf.String(), "facilities-api")
	assert.Contains(t, logBuf.String(), "ListAssets")

	logBuf.Reset()
	_, err = srv.CreateTicket(context.Background(), &opsv1.CreateTicketRequest{Title: "test"})
	require.Error(t, err)
	assert.Contains(t, logBuf.String(), "ops-api")
	assert.Contains(t, logBuf.String(), "CreateTicket")
}
