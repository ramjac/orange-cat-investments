package main

import (
	"crypto/tls"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/orange-cat-investments/oci/internal/app/customerbll"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Customer Business Logic Layer (Customer BLL) API (pure gRPC)...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	workforceURL := os.Getenv("WORKFORCE_API_URL")
	if workforceURL == "" {
		workforceURL = "localhost:50051"
	}
	coreInvestURL := os.Getenv("CORE_INVEST_API_URL")
	if coreInvestURL == "" {
		coreInvestURL = "localhost:50053"
	}
	opsURL := os.Getenv("OPS_API_URL")
	if opsURL == "" {
		opsURL = "localhost:50054"
	}

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	pubSub := envelope.NewPublisherWithFallback(amqpURL, logger)
	defer func() { _ = pubSub.Close() }()

	defaultFelineID := os.Getenv("DEFAULT_FELINE_ID")
	if defaultFelineID == "" {
		defaultFelineID = os.Getenv("DEFAULT_EMPLOYEE_ID")
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")

	var serverTLS, clientTLS *tls.Config
	if certFile != "" && keyFile != "" {
		if sTLS, err := mtls.NewServerTLSConfig(certFile, keyFile, clientCA); err == nil {
			serverTLS = sTLS
			logger.Info("Customer BLL configured with dynamic server mTLS")
		} else {
			logger.Error("failed to configure server mTLS in Customer BLL", "error", err)
			os.Exit(1)
		}
		if clientCA != "" {
			if cTLS, err := mtls.NewClientTLSConfig(certFile, keyFile, clientCA); err == nil {
				clientTLS = cTLS
			}
		}
	}

	bllSrv := customerbll.NewServer(customerbll.Config{
		Logger:           logger,
		Publisher:        pubSub,
		WorkforceTarget:  workforceURL,
		CoreInvestTarget: coreInvestURL,
		OpsTarget:        opsURL,
		DefaultFelineID:  defaultFelineID,
		ClientTLS:        clientTLS,
	})
	defer bllSrv.Close()

	grpcServer := grpcutil.NewServer(serverTLS)
	customerbllv1.RegisterCustomerBLLServiceServer(grpcServer, bllSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("oci.customer_bll.v1.CustomerBLLService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen on gRPC port", "port", port, "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Customer BLL listening (pure gRPC)", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Customer BLL gRPC server error", "error", err)
		}
	}()

	<-stop
	logger.Info("shutting down Customer BLL...")

	grpcServer.GracefulStop()
	logger.Info("Customer BLL stopped gracefully")
}
