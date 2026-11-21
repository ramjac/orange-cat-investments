package grpcutil

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// WithTelemetryDialOption returns a grpc.DialOption that configures OpenTelemetry tracing.
func WithTelemetryDialOption() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}

// WithTelemetryServerOption returns a grpc.ServerOption that configures OpenTelemetry tracing.
func WithTelemetryServerOption() grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler())
}

// ExtractPeerSPIFFEIDs extracts all spiffe:// URI SANs from the authenticated TLS peer
// certificate in the incoming gRPC context.
func ExtractPeerSPIFFEIDs(ctx context.Context) ([]string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok || p == nil || p.AuthInfo == nil {
		return nil, fmt.Errorf("missing peer auth info in gRPC context")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("peer auth info is not TLSInfo")
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates presented")
	}
	var ids []string
	for _, u := range tlsInfo.State.PeerCertificates[0].URIs {
		if u != nil && u.Scheme == "spiffe" {
			ids = append(ids, u.String())
		}
	}
	return ids, nil
}

func buildAllowedSPIFFESet(allowedIDs []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(allowedIDs))
	for _, id := range allowedIDs {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	return allowed
}

func authorizeSPIFFEContext(ctx context.Context, allowed map[string]struct{}) error {
	ids, err := ExtractPeerSPIFFEIDs(ctx)
	if err != nil {
		return status.Errorf(codes.PermissionDenied, "peer SPIFFE identity verification failed: %v", err)
	}
	if len(ids) == 0 {
		return status.Error(codes.PermissionDenied, "peer certificate does not contain a SPIFFE URI SAN")
	}
	for _, id := range ids {
		if _, ok := allowed[id]; ok {
			return nil
		}
	}
	return status.Errorf(codes.PermissionDenied, "caller SPIFFE ID %q is not authorized", strings.Join(ids, ","))
}

// UnarySPIFFEAuthInterceptor enforces that incoming unary RPC callers present a verified mTLS
// client certificate containing at least one SPIFFE URI SAN present in allowedIDs.
func UnarySPIFFEAuthInterceptor(allowedIDs ...string) grpc.UnaryServerInterceptor {
	allowed := buildAllowedSPIFFESet(allowedIDs)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := authorizeSPIFFEContext(ctx, allowed); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamSPIFFEAuthInterceptor enforces that incoming streaming RPC callers present a verified mTLS
// client certificate containing at least one SPIFFE URI SAN present in allowedIDs.
func StreamSPIFFEAuthInterceptor(allowedIDs ...string) grpc.StreamServerInterceptor {
	allowed := buildAllowedSPIFFESet(allowedIDs)
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := authorizeSPIFFEContext(ss.Context(), allowed); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

// WithAllowedPeerSPIFFEIDs returns gRPC unary and stream server interceptors restricting access
// to callers presenting one of the specified SPIFFE URI SANs.
func WithAllowedPeerSPIFFEIDs(allowedIDs ...string) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(UnarySPIFFEAuthInterceptor(allowedIDs...)),
		grpc.ChainStreamInterceptor(StreamSPIFFEAuthInterceptor(allowedIDs...)),
	}
}

// ResolveNamespace reads POD_NAMESPACE, trims whitespace, and defaults to "oci-core".
func ResolveNamespace() string {
	if ns := strings.TrimSpace(os.Getenv("POD_NAMESPACE")); ns != "" {
		return ns
	}
	return "oci-core"
}

func resolveAllowedCallers(defaults []string) []string {
	if raw := strings.TrimSpace(os.Getenv("ALLOWED_PEER_SPIFFE_IDS")); raw != "" {
		var overrides []string
		for _, part := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				overrides = append(overrides, trimmed)
			}
		}
		if len(overrides) > 0 {
			return overrides
		}
	}
	return defaults
}

func normalizeNamespace(namespace string) string {
	if ns := strings.TrimSpace(namespace); ns != "" {
		return ns
	}
	return ResolveNamespace()
}

func spiffeID(namespace, serviceAccount string) string {
	return fmt.Sprintf("spiffe://oci.local/ns/%s/sa/%s", namespace, serviceAccount)
}

// AllowedDomainAPICallers returns the SPIFFE URI SANs permitted to invoke Domain APIs
// in the given namespace, honoring the ALLOWED_PEER_SPIFFE_IDS override when non-empty.
func AllowedDomainAPICallers(namespace string) []string {
	ns := normalizeNamespace(namespace)
	return resolveAllowedCallers([]string{
		spiffeID(ns, "customer-bll"),
		spiffeID(ns, "internal-bll"),
		spiffeID(ns, "oci-worker"),
	})
}

// AllowedCustomerBLLCallers returns the SPIFFE URI SANs permitted to invoke customer-bll
// in the given namespace, honoring the ALLOWED_PEER_SPIFFE_IDS override when non-empty.
func AllowedCustomerBLLCallers(namespace string) []string {
	ns := normalizeNamespace(namespace)
	return resolveAllowedCallers([]string{
		spiffeID(ns, "oci-customer-bff"),
		spiffeID(ns, "oci-worker"),
	})
}

// AllowedInternalBLLCallers returns the SPIFFE URI SANs permitted to invoke internal-bll
// in the given namespace, honoring the ALLOWED_PEER_SPIFFE_IDS override when non-empty.
func AllowedInternalBLLCallers(namespace string) []string {
	ns := normalizeNamespace(namespace)
	mockHardwareID := "spiffe://oci.local/ns/external-sim/sa/mock-hardware"
	if ns == "oci-dev" {
		mockHardwareID = "spiffe://oci.local/ns/external-sim/sa/mock-hardware-dev"
	}
	return resolveAllowedCallers([]string{
		spiffeID(ns, "oci-employee-bff"),
		spiffeID(ns, "oci-pebble-proxy"),
		spiffeID(ns, "oci-worker"),
		mockHardwareID,
	})
}
