package customerbff

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	"github.com/orange-cat-investments/oci/pkg/session"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CustomerSession = sessionvalkey.CustomerSession
type SessionStore = sessionvalkey.CustomerSessionStore

// CustomerBFF is the presentation proxy layer for customer web and mobile frontends.
// Invariant: CustomerBFF never connects directly to database schemas (pgxpool) or Domain APIs.
// All business domain logic and persistence are delegated to Customer BLL over gRPC.
type CustomerBFF struct {
	bllClient      customerbllv1.CustomerBLLServiceClient
	sessions       SessionStore
	cache          cachevalkey.Cache
	rateLimiter    *cachevalkey.RateLimiter
	logger         *slog.Logger
	searchClient   *meilisearch.Client
	turnServerHost string
	turnAuthSecret string
}

// NewCustomerBFF initializes a new CustomerBFF instance.
// In production code, bllClient is required and cannot be nil.
func NewCustomerBFF(bllClient customerbllv1.CustomerBLLServiceClient, logger *slog.Logger) *CustomerBFF {
	if logger == nil {
		logger = slog.Default()
	}
	if bllClient == nil {
		panic("customerbff: bllClient is required and cannot be nil")
	}

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	valkeyPass := os.Getenv("VALKEY_PASSWORD")

	meiliURL := os.Getenv("MEILISEARCH_URL")
	if meiliURL == "" {
		meiliURL = "http://localhost:7700"
	}
	meiliKey := os.Getenv("MEILI_MASTER_KEY")
	sClient := meilisearch.NewClient(meiliURL, meiliKey)

	var sess SessionStore
	if valkeyAddr != "" {
		sess = sessionvalkey.NewCustomerSessionStore(valkeyAddr, valkeyPass, 0, logger)
	} else {
		sess = sessionvalkey.NewMemoryCustomerSessionStore()
	}

	turnServerHost := strings.TrimSpace(os.Getenv("TURN_SERVER_HOST"))
	if turnServerHost == "" {
		turnServerHost = "turn.oci.local:3478"
	}
	turnAuthSecret := os.Getenv("TURN_AUTH_SECRET")

	return &CustomerBFF{
		bllClient:      bllClient,
		sessions:       sess,
		cache:          cachevalkey.NewCache(valkeyAddr, valkeyPass, 0, logger),
		rateLimiter:    cachevalkey.NewRateLimiterWithAddr(valkeyAddr, valkeyPass, 0, logger),
		logger:         logger,
		searchClient:   sClient,
		turnServerHost: turnServerHost,
		turnAuthSecret: turnAuthSecret,
	}
}

// SetTURNConfig configures the STUN/TURN relay host and shared HMAC-SHA1 auth secret.
func (b *CustomerBFF) SetTURNConfig(serverHost, authSecret string) {
	if strings.TrimSpace(serverHost) != "" {
		b.turnServerHost = strings.TrimSpace(serverHost)
	}
	b.turnAuthSecret = authSecret
}

func (b *CustomerBFF) SetBLLClient(client customerbllv1.CustomerBLLServiceClient) {
	b.bllClient = client
}

func (b *CustomerBFF) BLLClient() customerbllv1.CustomerBLLServiceClient {
	return b.bllClient
}

func (b *CustomerBFF) SetSessionStore(store SessionStore) {
	b.sessions = store
}

func (b *CustomerBFF) SetCache(c cachevalkey.Cache) {
	b.cache = c
}

func (b *CustomerBFF) SetRateLimiter(rl *cachevalkey.RateLimiter) {
	b.rateLimiter = rl
}

func (b *CustomerBFF) SetSearchClient(client *meilisearch.Client) {
	b.searchClient = client
}

func (b *CustomerBFF) SearchClient() *meilisearch.Client {
	return b.searchClient
}

func CustomerFromContext(ctx context.Context) *CustomerSession {
	return session.CustomerFromContext(ctx)
}

func containsSensitiveInternalDetails(msg string) bool {
	lower := strings.ToLower(msg)
	for _, needle := range []string{
		"dial tcp",
		"connection refused",
		"transport:",
		"user=",
		"database=",
		"password=",
		"sqlstate",
		"failed to connect to",
		"rpc error:",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func sanitizeClientErrorMessage(code codes.Code, msg string) string {
	trimmed := strings.TrimSpace(msg)
	if trimmed != "" && !containsSensitiveInternalDetails(trimmed) {
		return trimmed
	}
	switch code {
	case codes.Unavailable:
		return "customer service temporarily unavailable"
	case codes.DeadlineExceeded:
		return "customer service request timed out"
	case codes.NotFound:
		return "resource not found"
	case codes.InvalidArgument:
		return "invalid request parameters"
	case codes.FailedPrecondition:
		return "request precondition failed"
	case codes.Unauthenticated:
		return "authentication required"
	case codes.PermissionDenied:
		return "permission denied"
	default:
		return "internal service error"
	}
}

func (b *CustomerBFF) log() *slog.Logger {
	if b != nil && b.logger != nil {
		return b.logger
	}
	return slog.Default()
}

func (b *CustomerBFF) handleGRPCError(w http.ResponseWriter, op string, err error, fallbackStatus int) {
	writeGRPCErrorWithLogger(b.log(), op, w, err, fallbackStatus)
}

func writeGRPCError(w http.ResponseWriter, err error, fallbackStatus int) {
	writeGRPCErrorWithLogger(slog.Default(), "customer-bll", w, err, fallbackStatus)
}

func writeGRPCErrorWithLogger(logger *slog.Logger, op string, w http.ResponseWriter, err error, fallbackStatus int) {
	st, ok := status.FromError(err)
	code := status.Code(err)
	rawMsg := err.Error()
	if ok && st.Message() != "" {
		rawMsg = st.Message()
	}

	var httpStatus int
	switch code {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.FailedPrecondition:
		httpStatus = http.StatusUnprocessableEntity
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
	default:
		httpStatus = fallbackStatus
	}

	if logger != nil && (httpStatus >= 500 || code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.Internal || code == codes.Unknown) {
		logger.Error("customer-bll gRPC call failed",
			"downstream", "customer-bll",
			"operation", op,
			"grpc_code", code.String(),
			"http_status", httpStatus,
			"error", err,
		)
	}

	clientMsg := sanitizeClientErrorMessage(code, rawMsg)
	httputil.WriteError(w, httpStatus, clientMsg)
}
