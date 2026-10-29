package grpcutil

import (
	"context"
	"crypto/tls"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// CleanTarget cleans target addresses by removing http://, https://, and trailing slashes.
func CleanTarget(target string) string {
	target = strings.TrimSpace(target)
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimRight(target, "/")
	return target
}

// Dial creates a new gRPC client connection to the target with either mutual TLS or insecure transport.
func Dial(ctx context.Context, target string, tlsConfig *tls.Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	cleaned := CleanTarget(target)

	var credOption grpc.DialOption
	if tlsConfig != nil {
		credOption = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		credOption = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	allOpts := append([]grpc.DialOption{
		credOption,
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	}, append(opts, WithTelemetryDialOption())...)
	return grpc.NewClient(cleaned, allOpts...)
}
