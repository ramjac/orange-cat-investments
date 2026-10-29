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
		cfg := tlsConfig.Clone()
		cfg.NextProtos = []string{"h2"}
		if origGetConfig := tlsConfig.GetConfigForClient; origGetConfig != nil {
			cfg.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
				c, err := origGetConfig(hello)
				if err != nil {
					return nil, err
				}
				if c != nil {
					cCopy := c.Clone()
					if len(cCopy.NextProtos) == 0 {
						cCopy.NextProtos = []string{"h2"}
					}
					return cCopy, nil
				}
				return nil, nil
			}
		}
		allOpts = append(allOpts, grpc.Creds(credentials.NewTLS(cfg)))
	}
	allOpts = append(allOpts, opts...)
	return grpc.NewServer(append(allOpts, WithTelemetryServerOption())...)
}
