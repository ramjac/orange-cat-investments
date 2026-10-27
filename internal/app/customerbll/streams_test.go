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
	completeBacktestRunFn func(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error)
	listCameraStreamsFn   func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error)
}

func (m *mockCoreInvestClient) CompleteBacktestRun(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error) {
	if m.completeBacktestRunFn != nil {
		return m.completeBacktestRunFn(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	if m.listCameraStreamsFn != nil {
		return m.listCameraStreamsFn(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func TestCompleteBacktestRun_UninitializedClient(t *testing.T) {
	s := &Server{ciClient: nil}
	ctx := context.Background()

	req := &customerbllv1.CompleteBacktestRunRequest{
		RunID:   "run-001",
		Status:  "completed",
		Results: `{"return_pct": 12.5}`,
	}

	resp, err := s.CompleteBacktestRun(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Contains(t, st.Message(), "core invest domain client is not initialized")
}

func TestCompleteBacktestRun_ClientError(t *testing.T) {
	mockErr := status.Error(codes.NotFound, "backtest run not found")
	mockClient := &mockCoreInvestClient{
		completeBacktestRunFn: func(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error) {
			return nil, mockErr
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	req := &customerbllv1.CompleteBacktestRunRequest{
		RunID:   "run-invalid",
		Status:  "failed",
		Results: `{}`,
	}

	resp, err := s.CompleteBacktestRun(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, mockErr, err)
}

func TestCompleteBacktestRun_Success(t *testing.T) {
	expectedRun := &coreinvestv1.BacktestRun{
		RunID:          "run-123",
		ResearcherID:   "fel-01",
		StrategyName:   "Alpha Cat Strategy",
		Status:         "COMPLETED",
		InitialCapital: 100000.0,
		FinalCapital:   125000.0,
		SharpeRatio:    1.85,
		MaxDrawdown:    0.05,
		CreatedAt:      "2025-01-01T00:00:00Z",
	}

	mockClient := &mockCoreInvestClient{
		completeBacktestRunFn: func(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error) {
			assert.Equal(t, "run-123", in.RunID)
			assert.Equal(t, "COMPLETED", in.Status)
			assert.Equal(t, `{"sharpe": 1.85, "alpha": 0.05}`, in.Results)
			return &coreinvestv1.CompleteBacktestRunResponse{
				Run: expectedRun,
			}, nil
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	req := &customerbllv1.CompleteBacktestRunRequest{
		RunID:   "run-123",
		Status:  "COMPLETED",
		Results: `{"sharpe": 1.85, "alpha": 0.05}`,
	}

	resp, err := s.CompleteBacktestRun(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedRun, resp.Run)
}

func TestListCameraStreams_UninitializedClient(t *testing.T) {
	s := &Server{ciClient: nil}
	ctx := context.Background()

	resp, err := s.ListCameraStreams(ctx, &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Streams)
}

func TestListCameraStreams_ClientError(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		listCameraStreamsFn: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
			return nil, errors.New("rpc connection failed")
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	resp, err := s.ListCameraStreams(ctx, &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Streams)
}

func TestListCameraStreams_Success(t *testing.T) {
	mockStreams := []*coreinvestv1.CameraStream{
		{StreamID: "cam-01", StreamURL: "rtsp://camera1.local/live", Status: "active"},
		{StreamID: "cam-02", StreamURL: "rtsp://camera2.local/live", Status: "active"},
	}

	mockClient := &mockCoreInvestClient{
		listCameraStreamsFn: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
			return &coreinvestv1.ListCameraStreamsResponse{Streams: mockStreams}, nil
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	resp, err := s.ListCameraStreams(ctx, &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Streams, 2)
	assert.Equal(t, mockStreams, resp.Streams)
}
