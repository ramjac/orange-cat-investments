package customerbll

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultUpstreamTimeout = 5 * time.Second

// upstreamHTTPError is a consumer-side interface implemented by broker and bank
// HTTP driver errors so customerbll can map status codes without coupling to
// vendor-specific error types or leaking vendor names.
type upstreamHTTPError interface {
	error
	HTTPStatus() int
	UpstreamMessage() string
}

var vendorNameSanitizers = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)\balpaca\b`), "broker"},
	{regexp.MustCompile(`(?i)\bplaid\b`), "bank"},
}

func (s *Server) upstreamDeadline() time.Duration {
	if s != nil && s.upstreamTimeout > 0 {
		return s.upstreamTimeout
	}
	return defaultUpstreamTimeout
}

// withUpstreamTimeout derives a child context bounded by the server's upstream
// timeout (while preserving any shorter deadline already set on ctx).
func (s *Server) withUpstreamTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.upstreamDeadline())
}

// mapUpstreamError converts broker/bank HTTP or transport errors into gRPC
// status errors (Unavailable, DeadlineExceeded, FailedPrecondition, NotFound)
// without leaking upstream vendor names or raw network URLs, and records the
// outcome in the oci.upstream.requests.total OTel counter.
func mapUpstreamError(service string, err error) error {
	if err == nil {
		telemetry.RecordUpstreamRequest(context.Background(), service, "ok")
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		telemetry.RecordUpstreamRequest(context.Background(), service, "timeout")
		return status.Errorf(codes.DeadlineExceeded, "%s request timed out", service)
	}
	if errors.Is(err, context.Canceled) {
		telemetry.RecordUpstreamRequest(context.Background(), service, "unavailable")
		return status.Errorf(codes.Canceled, "%s request canceled", service)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		telemetry.RecordUpstreamRequest(context.Background(), service, "timeout")
		return status.Errorf(codes.DeadlineExceeded, "%s request timed out", service)
	}

	var httpErr upstreamHTTPError
	if errors.As(err, &httpErr) {
		code := httpErr.HTTPStatus()
		msg := sanitizeVendorNames(strings.TrimSpace(httpErr.UpstreamMessage()))
		switch {
		case code == http.StatusBadRequest ||
			code == http.StatusForbidden ||
			code == http.StatusPreconditionFailed ||
			code == http.StatusUnprocessableEntity:
			telemetry.RecordUpstreamRequest(context.Background(), service, "rejected")
			if msg == "" {
				return status.Errorf(codes.FailedPrecondition, "%s rejected request", service)
			}
			return status.Errorf(codes.FailedPrecondition, "%s rejected request: %s", service, msg)
		case code == http.StatusNotFound:
			telemetry.RecordUpstreamRequest(context.Background(), service, "rejected")
			return status.Errorf(codes.NotFound, "%s resource not found", service)
		case code == http.StatusRequestTimeout || code == http.StatusGatewayTimeout:
			telemetry.RecordUpstreamRequest(context.Background(), service, "timeout")
			return status.Errorf(codes.DeadlineExceeded, "%s request timed out", service)
		default:
			telemetry.RecordUpstreamRequest(context.Background(), service, "unavailable")
			return status.Errorf(codes.Unavailable, "%s service unavailable", service)
		}
	}

	telemetry.RecordUpstreamRequest(context.Background(), service, "unavailable")
	return status.Errorf(codes.Unavailable, "%s service unavailable", service)
}

func recordUpstreamSuccess(ctx context.Context, service string) {
	telemetry.RecordUpstreamRequest(ctx, service, "ok")
}

func sanitizeVendorNames(msg string) string {
	out := msg
	for _, s := range vendorNameSanitizers {
		out = s.pattern.ReplaceAllString(out, s.replacement)
	}
	return out
}
