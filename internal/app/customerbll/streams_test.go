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
	listCameraStreamsFunc func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error)
	listStatementsFunc    func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error)
}

func (m *mockCoreInvestClient) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	if m.listCameraStreamsFunc != nil {
		return m.listCameraStreamsFunc(ctx, in)
	}
	return &coreinvestv1.ListCameraStreamsResponse{}, nil
}

func (m *mockCoreInvestClient) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	if m.listStatementsFunc != nil {
		return m.listStatementsFunc(ctx, in)
	}
	return nil, nil
}

func TestListCameraStreams(t *testing.T) {
	t.Run("nil ciClient returns empty streams response without error", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.ListCameraStreams(context.Background(), &customerbllv1.ListCameraStreamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Streams)
	})

	t.Run("ciClient returning error fallback to empty streams response without error", func(t *testing.T) {
		mockClient := &mockCoreInvestClient{
			listCameraStreamsFunc: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
				return nil, errors.New("gRPC connection failed")
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockClient)

		resp, err := s.ListCameraStreams(context.Background(), &customerbllv1.ListCameraStreamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Streams)
	})

	t.Run("ciClient returning camera streams success", func(t *testing.T) {
		expectedStreams := []*coreinvestv1.CameraStream{
			{
				StreamID:  "stream-001",
				AssetID:   "asset-cam-01",
				FelineID:  "fel-barnaby-01",
				StreamURL: "rtsp://live.oci.local/lounge1",
				Status:    "online",
			},
			{
				StreamID:  "stream-002",
				AssetID:   "asset-cam-02",
				FelineID:  "fel-cleo-02",
				StreamURL: "rtsp://live.oci.local/perch2",
				Status:    "online",
			},
		}

		mockClient := &mockCoreInvestClient{
			listCameraStreamsFunc: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
				return &coreinvestv1.ListCameraStreamsResponse{
					Streams: expectedStreams,
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockClient)

		resp, err := s.ListCameraStreams(context.Background(), &customerbllv1.ListCameraStreamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Streams, 2)
		assert.Equal(t, "stream-001", resp.Streams[0].StreamID)
		assert.Equal(t, "stream-002", resp.Streams[1].StreamID)
	})
}
