package grpcutil

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// WithTelemetryDialOption returns a grpc.DialOption that configures OpenTelemetry tracing.
func WithTelemetryDialOption() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}

// WithTelemetryServerOption returns a grpc.ServerOption that configures OpenTelemetry tracing.
func WithTelemetryServerOption() grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler())
}
