package customerbll

import (
	"context"
	"errors"
	"io"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	listCameraStreamsFunc func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error)
}

func (m *mockCoreInvestClient) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	if m.listCameraStreamsFunc != nil {
		return m.listCameraStreamsFunc(ctx, in)
	}
	return &coreinvestv1.ListCameraStreamsResponse{}, nil
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

// Mocks for StreamPortfolioValuation tests

var _ coreinvestv1.CoreInvestServiceClient = (*mockValuationCoreInvestClient)(nil)

type mockValuationServer struct {
	grpc.ServerStream
	ctx          context.Context
	sentEvents   []*coreinvestv1.PortfolioValuationEvent
	sendErr      error
	cancelOnSend bool
	cancelFunc   context.CancelFunc
}

func (m *mockValuationServer) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *mockValuationServer) Send(evt *coreinvestv1.PortfolioValuationEvent) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentEvents = append(m.sentEvents, evt)
	if m.cancelOnSend && m.cancelFunc != nil {
		m.cancelFunc()
	}
	return nil
}

type mockValuationCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	streamClient coreinvestv1.CoreInvestService_StreamPortfolioValuationClient
	streamErr    error
}

func (m *mockValuationCoreInvestClient) StreamPortfolioValuation(ctx context.Context, in *coreinvestv1.StreamPortfolioValuationRequest) (coreinvestv1.CoreInvestService_StreamPortfolioValuationClient, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return m.streamClient, nil
}

type mockCoreInvestStreamClient struct {
	grpc.ClientStream
	events []*coreinvestv1.PortfolioValuationEvent
	errs   []error
	index  int
}

func (m *mockCoreInvestStreamClient) Recv() (*coreinvestv1.PortfolioValuationEvent, error) {
	if m.index < len(m.events) {
		evt := m.events[m.index]
		m.index++
		return evt, nil
	}
	errIdx := m.index - len(m.events)
	if errIdx < len(m.errs) {
		err := m.errs[errIdx]
		m.index++
		return nil, err
	}
	return nil, io.EOF
}

func TestStreamPortfolioValuation_InvalidArgument(t *testing.T) {
	s := &Server{}
	stream := &mockValuationServer{}

	// Case 1: nil request
	err := s.StreamPortfolioValuation(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "portfolio_id required")

	// Case 2: empty portfolio_id
	err = s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: ""}, stream)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "portfolio_id required")
}

func TestStreamPortfolioValuation_UpstreamProxy(t *testing.T) {
	t.Run("successful proxying", func(t *testing.T) {
		events := []*coreinvestv1.PortfolioValuationEvent{
			{PortfolioID: "port-101", TotalBalanceUSD: 10000.0, CashBalanceUSD: 1000.0, InvestedBalanceUSD: 9000.0},
			{PortfolioID: "port-101", TotalBalanceUSD: 10050.0, CashBalanceUSD: 1000.0, InvestedBalanceUSD: 9050.0},
		}

		streamClient := &mockCoreInvestStreamClient{
			events: events,
			errs:   []error{io.EOF},
		}
		ciClient := &mockValuationCoreInvestClient{streamClient: streamClient}

		s := &Server{}
		s.SetCoreInvestClient(ciClient)

		serverStream := &mockValuationServer{}

		err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-101"}, serverStream)
		require.NoError(t, err)
		assert.Len(t, serverStream.sentEvents, 2)
		assert.Equal(t, "port-101", serverStream.sentEvents[0].PortfolioID)
		assert.Equal(t, 10000.0, serverStream.sentEvents[0].TotalBalanceUSD)
		assert.Equal(t, 10050.0, serverStream.sentEvents[1].TotalBalanceUSD)
	})

	t.Run("send error during proxying", func(t *testing.T) {
		events := []*coreinvestv1.PortfolioValuationEvent{
			{PortfolioID: "port-101", TotalBalanceUSD: 10000.0},
		}

		streamClient := &mockCoreInvestStreamClient{
			events: events,
		}
		ciClient := &mockValuationCoreInvestClient{streamClient: streamClient}

		s := &Server{}
		s.SetCoreInvestClient(ciClient)

		expectedErr := errors.New("client connection closed")
		serverStream := &mockValuationServer{sendErr: expectedErr}

		err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-101"}, serverStream)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestStreamPortfolioValuation_FallbackSimulation(t *testing.T) {
	t.Run("fallback when ciClient is nil", func(t *testing.T) {
		s := &Server{} // ciClient is nil

		serverStream := &mockValuationServer{}

		err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-fallback"}, serverStream)
		require.NoError(t, err)
		assert.Len(t, serverStream.sentEvents, 5)

		for i, evt := range serverStream.sentEvents {
			assert.Equal(t, "port-fallback", evt.PortfolioID)
			expectedVal := 154200.50 + float64(i)*2.75
			assert.InDelta(t, expectedVal, evt.TotalBalanceUSD, 0.001)
			assert.Equal(t, 15000.00, evt.CashBalanceUSD)
			assert.InDelta(t, expectedVal-15000.00, evt.InvestedBalanceUSD, 0.001)
			assert.Equal(t, 1.42, evt.ChangePct)
			assert.NotEmpty(t, evt.Timestamp)
		}
	})

	t.Run("fallback when upstream returns error on stream creation", func(t *testing.T) {
		ciClient := &mockValuationCoreInvestClient{
			streamErr: status.Error(codes.Unavailable, "core-invest stream unavailable"),
		}

		s := &Server{}
		s.SetCoreInvestClient(ciClient)

		serverStream := &mockValuationServer{}

		err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-fallback-2"}, serverStream)
		require.NoError(t, err)
		assert.Len(t, serverStream.sentEvents, 5)
	})

	t.Run("context cancelled during fallback simulation", func(t *testing.T) {
		s := &Server{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // pre-cancelled context

		serverStream := &mockValuationServer{ctx: ctx}

		err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-fallback-cancel"}, serverStream)
		require.NoError(t, err)
		assert.Empty(t, serverStream.sentEvents)
	})

	t.Run("send error during fallback simulation", func(t *testing.T) {
		s := &Server{}

		expectedErr := errors.New("stream write failure")
		serverStream := &mockValuationServer{sendErr: expectedErr}

		err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-fallback-senderr"}, serverStream)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestStreamPortfolioValuation_ContextCancelMidStream(t *testing.T) {
	s := &Server{}

	ctx, cancel := context.WithCancel(context.Background())
	serverStream := &mockValuationServer{
		ctx:          ctx,
		cancelOnSend: true,
		cancelFunc:   cancel,
	}

	err := s.StreamPortfolioValuation(&coreinvestv1.StreamPortfolioValuationRequest{PortfolioID: "port-cancel-mid"}, serverStream)
	require.NoError(t, err)
	assert.Equal(t, 1, len(serverStream.sentEvents))
}
