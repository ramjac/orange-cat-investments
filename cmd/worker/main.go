package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"

	"os/signal"
	"syscall"
	"time"

	"github.com/orange-cat-investments/oci/pkg/telemetry"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/internal/app/worker"
	"github.com/orange-cat-investments/oci/internal/service/pki"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
)

func getWorkerTLSConfig() *tls.Config {
	wCert := os.Getenv("TLS_CERT_FILE")
	wKey := os.Getenv("TLS_KEY_FILE")
	wCA := os.Getenv("CLIENT_CA_FILE")
	if wCA == "" {
		wCA = os.Getenv("CA_CERT_FILE")
	}
	if wCA == "" {
		wCA = os.Getenv("CA_CERT_PATH")
	}
	if wCert != "" && wKey != "" && wCA != "" {
		if c, err := mtls.NewClientTLSConfig(wCert, wKey, wCA); err == nil {
			return c
		}
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "worker")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		defer shutdownTelemetry(ctx)
	}
	logger.Info("starting OCI Background Worker with Watermill Engine (5-Tier BLL Architecture)")

	healthServer := mtls.StartHealthListener(os.Getenv("HEALTH_PORT"), "", logger, "worker")
	if healthServer != nil {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			_ = healthServer.Shutdown(shutdownCtx)
		}()
	}

	watermillLogger := watermill.NewStdLogger(false, false)
	tlsConfig := getWorkerTLSConfig()

	customerBLLURL := os.Getenv("CUSTOMER_BLL_URL")
	if customerBLLURL == "" {
		customerBLLURL = "localhost:8085"
	}
	connCust, err := grpcutil.Dial(context.Background(), grpcutil.CleanTarget(customerBLLURL), tlsConfig)
	if err != nil {
		logger.Error("worker fatal: failed to dial Customer BLL service", "url", customerBLLURL, "error", err)
		os.Exit(1)
	}
	defer connCust.Close()
	customerBLL := customerbllv1.NewCustomerBLLServiceClient(connCust)
	logger.Info("worker connected to Customer BLL service via gRPC", "url", customerBLLURL)

	internalBLLURL := os.Getenv("INTERNAL_BLL_URL")
	if internalBLLURL == "" {
		internalBLLURL = "localhost:8086"
	}
	connInt, err := grpcutil.Dial(context.Background(), grpcutil.CleanTarget(internalBLLURL), tlsConfig)
	if err != nil {
		logger.Error("worker fatal: failed to dial Internal BLL service", "url", internalBLLURL, "error", err)
		os.Exit(1)
	}
	defer connInt.Close()
	internalBLL := internalbllv1.NewInternalBLLServiceClient(connInt)
	logger.Info("worker connected to Internal BLL service via gRPC", "url", internalBLLURL)

	var pubSub worker.PubSub
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL != "" {
		ps, err := worker.NewAMQPPubSub(amqpURL, watermillLogger)
		if err != nil {
			logger.Error("failed to create RabbitMQ pub/sub in worker", "error", err)
			os.Exit(1)
		}
		pubSub = ps
		logger.Info("worker connected to RabbitMQ AMQP broker successfully")
	} else {
		pubSub = gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)
		logger.Info("worker running with in-memory GoChannel publisher (local test mode)")
	}

	batchMode := os.Getenv("BATCH_MODE")
	if batchMode == "ach-recurring" {
		logger.Info("executing one-shot ACH recurring deposits batch run via Customer BLL")
		if err := worker.RunACHRecurringBatch(context.Background(), customerBLL, pubSub, logger); err != nil {
			logger.Error("ACH recurring batch run failed", "error", err)
			os.Exit(1)
		}
		logger.Info("ACH recurring batch completed successfully")
		return
	}
	if batchMode == "eom-statements" {
		logger.Info("executing one-shot End-of-Month statements batch run via Customer BLL")
		if err := worker.RunEOMStatementBatch(context.Background(), customerBLL, pubSub, logger, time.Now().UTC()); err != nil {
			logger.Error("EOM statements batch run failed", "error", err)
			os.Exit(1)
		}
		logger.Info("EOM statements batch completed successfully")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	employeeBFFURL := os.Getenv("EMPLOYEE_BFF_URL")
	pebbleProxyURL := os.Getenv("PEBBLE_PROXY_URL")
	httpClient := &http.Client{Timeout: 10 * time.Second}

	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, customerBLL, internalBLL, employeeBFFURL, pebbleProxyURL, httpClient)
	if err != nil {
		logger.Error("failed to create watermill router", "error", err)
		os.Exit(1)
	}

	caCertPath := os.Getenv("CA_CERT_PATH")
	if caCertPath == "" {
		if _, err := os.Stat("/etc/oci/tls/ca.crt"); err == nil {
			caCertPath = "/etc/oci/tls/ca.crt"
		} else if _, err := os.Stat("k8s/ha/lan/tls/oci-lan-root-ca.crt"); err == nil {
			caCertPath = "k8s/ha/lan/tls/oci-lan-root-ca.crt"
		}
	}
	caKeyPath := os.Getenv("CA_KEY_PATH")
	if caKeyPath == "" {
		if _, err := os.Stat("/etc/oci/tls/ca.key"); err == nil {
			caKeyPath = "/etc/oci/tls/ca.key"
		} else if _, err := os.Stat("k8s/ha/lan/tls/oci-lan-root-ca.key"); err == nil {
			caKeyPath = "k8s/ha/lan/tls/oci-lan-root-ca.key"
		}
	}

	caManager := pki.NewCARotationManager(pki.CARotationConfig{
		RootCAName:           "oci-root-ca",
		IntermediateCAName:   "oci-intermediate-ca",
		RenewalThresholdDays: 30,
		IssuerNamespace:      "cert-manager",
		CACertFile:           caCertPath,
		CAKeyFile:            caKeyPath,
	})

	go func() {
		if err := router.Run(ctx); err != nil {
			logger.Error("watermill router error", "error", err)
		}
	}()

	worker.StartCARotationWorker(ctx, router, pubSub, caManager, logger)
	worker.StartExecutiveReportingWorker(ctx, router, pubSub, internalBLL, logger)

	if worker.IsFeatureEnabled("ENABLE_CORE_INVEST") {
		worker.StartRecurringACHProcessor(ctx, router, pubSub, customerBLL, logger)
		worker.StartEOMStatementBatchGenerator(ctx, router, pubSub, customerBLL, logger)
	}

	if worker.IsFeatureEnabled("ENABLE_FELINE_WORKFORCE") && worker.IsFeatureEnabled("ENABLE_CORE_INVEST") {
		worker.StartCatSpottedPublisher(ctx, router, pubSub, internalBLL)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down Watermill Background Worker...")
	cancel()
	_ = router.Close()
	logger.Info("OCI Background Worker stopped gracefully")
}
