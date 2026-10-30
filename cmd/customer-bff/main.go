package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"

	"github.com/orange-cat-investments/oci/pkg/telemetry"

	"github.com/orange-cat-investments/oci/internal/app/customerbff"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/session"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

// Backwards compatibility aliases for package main
type CustomerBFF = customerbff.CustomerBFF
type CustomerSession = customerbff.CustomerSession
type SessionStore = customerbff.SessionStore

var (
	NewCustomerBFF      = customerbff.NewCustomerBFF
	CustomerFromContext = customerbff.CustomerFromContext
)

func generateRandomToken(n int) (string, error) {
	return session.GenerateRandomToken(n)
}

func NewSessionStore() SessionStore {
	return sessionvalkey.NewMemoryCustomerSessionStore()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "customer-bff")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		defer shutdownTelemetry(ctx)
	}
	logger.Info("Starting OCI Customer BFF API Server...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	bllAddr := os.Getenv("CUSTOMER_BLL_URL")
	if bllAddr == "" {
		bllAddr = "customer-bll-service.oci-core.svc.cluster.local:8085"
	}

	var bllClient customerbllv1.CustomerBLLServiceClient
	tlsCert := os.Getenv("TLS_CERT_FILE")
	tlsKey := os.Getenv("TLS_KEY_FILE")
	caCert := os.Getenv("CA_CERT_FILE")
	if caCert == "" {
		caCert = os.Getenv("CLIENT_CA_FILE")
	}

	var clientTLS *tls.Config
	if tlsCert != "" && tlsKey != "" && caCert != "" {
		var err error
		clientTLS, err = mtls.NewClientTLSConfig(tlsCert, tlsKey, caCert)
		if err != nil {
			logger.Warn("failed to initialize mTLS config for Customer BLL upstream; proceeding with fallback", "error", err)
		}
	}

	conn, err := grpcutil.Dial(context.Background(), bllAddr, clientTLS)
	if err != nil {
		logger.Error("failed to connect to Customer BLL upstream", "addr", bllAddr, "error", err)
		os.Exit(1)
	}
	bllClient = customerbllv1.NewCustomerBLLServiceClient(conn)
	logger.Info("connected Customer BFF to Customer BLL upstream via gRPC", "addr", bllAddr)

	custBFF := customerbff.NewCustomerBFF(bllClient, logger)

	// Event-driven cache invalidation listener
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL != "" {
		valkeyAddr := os.Getenv("VALKEY_ADDR")
		valkeyPass := os.Getenv("VALKEY_PASSWORD")
		cacheInstance := cachevalkey.NewCache(valkeyAddr, valkeyPass, 0, logger)
		cachevalkey.StartAMQPInvalidationListener(context.Background(), amqpURL, "customer-bff", cacheInstance, logger)
	}

	httputil.ListenAndServeGraceful("customer-bff", port, httputil.InstrumentHandler(custBFF.Routes(), "customer-bff"), logger)
}
