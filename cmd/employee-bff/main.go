package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"

	"github.com/orange-cat-investments/oci/internal/app/employeebff"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/httputil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting OCI Employee BFF API Server...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	bllURL := os.Getenv("INTERNAL_BLL_URL")
	if bllURL == "" {
		bllURL = "internal-bll-service.oci-core.svc.cluster.local:8086"
	}

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
			logger.Error("employee-bff fatal: failed to initialize mTLS config for Internal BLL upstream", "error", err)
			os.Exit(1)
		}
	}

	conn, err := grpcutil.Dial(context.Background(), bllURL, clientTLS)
	if err != nil {
		logger.Error("employee-bff fatal: failed to connect to Internal BLL upstream", "addr", bllURL, "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	bllClient := internalbllv1.NewInternalBLLServiceClient(conn)
	logger.Info("connected Employee BFF to Internal BLL upstream via gRPC", "addr", bllURL)

	bff := employeebff.NewEmployeeBFF(bllClient, logger)

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	valkeyPass := os.Getenv("VALKEY_PASSWORD")

	sessionStore := sessionvalkey.NewEmployeeSessionStore(valkeyAddr, valkeyPass, 0, logger)
	cache := cachevalkey.NewCache(valkeyAddr, valkeyPass, 0, logger)
	rateLimiter := cachevalkey.NewRateLimiterWithAddr(valkeyAddr, valkeyPass, 0, logger)

	bff.SetSessionStore(sessionStore)
	bff.SetCache(cache)
	bff.SetRateLimiter(rateLimiter)

	issueTrackerURL := os.Getenv("FORGEJO_URL")
	if issueTrackerURL == "" {
		issueTrackerURL = os.Getenv("ISSUE_TRACKER_URL")
	}
	bff.SetIssueTrackerURL(issueTrackerURL)

	isProd := os.Getenv("ENV") == "production" || os.Getenv("OCI_ENV") == "production"
	bff.SetIsProd(isProd)

	enableFeline := os.Getenv("ENABLE_FELINE_WORKFORCE") != "false"
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && os.Getenv("ENABLE_FELINE_WORKFORCE") == "" {
		enableFeline = false
	}
	bff.SetEnableFelineWorkforce(enableFeline)

	meiliURL := os.Getenv("MEILISEARCH_URL")
	if meiliURL == "" {
		meiliURL = "http://localhost:7700"
	}
	meiliKey := os.Getenv("MEILI_MASTER_KEY")
	meiliClient := meilisearch.NewClient(meiliURL, meiliKey)
	_ = meiliClient.EnsureIndex(context.Background(), "employee_knowledge", "id")
	bff.SetSearchClient(meiliClient)

	// Event-driven cache invalidation listener
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL != "" {
		cachevalkey.StartAMQPInvalidationListener(context.Background(), amqpURL, "employee-bff", cache, logger)
	}

	httputil.ListenAndServeGraceful("employee-bff", port, bff.SetupRouter(), logger)
}
