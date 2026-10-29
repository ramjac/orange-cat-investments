package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"

	"github.com/orange-cat-investments/oci/pkg/telemetry"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "pebble-proxy")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		defer shutdownTelemetry(ctx)
	}
	logger.Info("starting OCI Pebble Companion App Gateway Proxy")

	enablePebbleGateway := os.Getenv("ENABLE_PEBBLE_GATEWAY") != "false"
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && os.Getenv("ENABLE_PEBBLE_GATEWAY") == "" {
		enablePebbleGateway = false
	}
	if !enablePebbleGateway {
		logger.Info("Pebble companion gateway proxy is disabled via configuration (ENABLE_PEBBLE_GATEWAY=false)")
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	internalBLLURL := os.Getenv("INTERNAL_BLL_URL")
	if internalBLLURL == "" {
		internalBLLURL = "internal-bll-service.oci-core.svc.cluster.local:8086"
	}

	var tlsConfig *tls.Config
	tlsCert := os.Getenv("TLS_CERT_FILE")
	tlsKey := os.Getenv("TLS_KEY_FILE")
	caCert := os.Getenv("CLIENT_CA_FILE")
	if caCert == "" {
		caCert = os.Getenv("CA_CERT_FILE")
	}
	if tlsCert != "" && tlsKey != "" && caCert != "" {
		var err error
		tlsConfig, err = mtls.NewClientTLSConfig(tlsCert, tlsKey, caCert)
		if err != nil {
			logger.Error("pebble-proxy fatal: could not load client mTLS config for internal-bll", "error", err)
			os.Exit(1)
		}
	}

	bllConn, err := grpcutil.Dial(ctx, internalBLLURL, tlsConfig)
	if err != nil {
		logger.Error("pebble-proxy fatal: could not connect to internal-bll gRPC service", "url", internalBLLURL, "error", err)
		os.Exit(1)
	}
	defer bllConn.Close()
	logger.Info("connected to internal-bll gRPC service", "url", internalBLLURL)

	bllClient := internalbllv1.NewInternalBLLServiceClient(bllConn)

	router := NewRouter(bllClient, logger)
	httputil.ListenAndServeGraceful("Pebble Companion Gateway Proxy", port, router, logger)
}
