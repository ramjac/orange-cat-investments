package grpcutil

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type echoRequest struct {
	Message string `json:"message"`
}

type echoResponse struct {
	Message string `json:"message"`
}

type echoService interface {
	Echo(ctx context.Context, in *echoRequest) (*echoResponse, error)
}

type echoServiceImpl struct{}

func (s *echoServiceImpl) Echo(ctx context.Context, in *echoRequest) (*echoResponse, error) {
	return &echoResponse{Message: "echo:" + in.Message}, nil
}

func _Echo_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(echoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(echoService).Echo(ctx, in)
}

var echoServiceDesc = grpc.ServiceDesc{
	ServiceName: "EchoService",
	HandlerType: (*echoService)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _Echo_Handler,
		},
	},
}

func TestCleanTarget(t *testing.T) {
	assert.Equal(t, "localhost:50051", CleanTarget("http://localhost:50051"))
	assert.Equal(t, "workforce-api-service:50051", CleanTarget("https://workforce-api-service:50051/"))
	assert.Equal(t, "127.0.0.1:8080", CleanTarget("127.0.0.1:8080"))
}

func TestGRPCUtil_RoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := NewServer(nil)
	srv.RegisterService(&echoServiceDesc, &echoServiceImpl{})
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	conn, err := Dial(context.Background(), "http://"+lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := new(echoResponse)
	err = conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "test"}, resp)
	require.NoError(t, err)
	assert.Equal(t, "echo:test", resp.Message)
}
