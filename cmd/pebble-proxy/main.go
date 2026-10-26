package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"

	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
			logger.Warn("could not load client mTLS config for internal-bll, proceeding without TLS", "error", err)
		}
	}

	bllConn, err := grpcutil.Dial(ctx, internalBLLURL, tlsConfig)
	if err != nil {
		logger.Warn("could not connect to internal-bll gRPC service", "url", internalBLLURL, "error", err)
	} else {
		defer bllConn.Close()
		logger.Info("connected to internal-bll gRPC service", "url", internalBLLURL)
	}

	var bllClient internalbllv1.InternalBLLServiceClient
	if bllConn != nil {
		bllClient = internalbllv1.NewInternalBLLServiceClient(bllConn)
	}

	router := NewRouter(bllClient, logger)
	httputil.ListenAndServeGraceful("Pebble Companion Gateway Proxy", port, router, logger)
}
