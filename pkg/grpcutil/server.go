package grpcutil

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewServer creates a new gRPC server with either mTLS credentials or insecure settings.
func NewServer(tlsConfig *tls.Config, opts ...grpc.ServerOption) *grpc.Server {
	var allOpts []grpc.ServerOption
	if tlsConfig != nil {
		allOpts = append(allOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	allOpts = append(allOpts, opts...)
	return grpc.NewServer(allOpts...)
}
