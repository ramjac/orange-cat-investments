package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"os"

	"os/signal"
	"syscall"

	facilities_repo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	"github.com/orange-cat-investments/oci/pkg/dbpool"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "facilities-api")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		defer func() { _ = shutdownTelemetry() }()
	}
	logger.Info("starting OCI Facilities Domain API microservice (pure gRPC)...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50052"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	dbURLFile := os.Getenv("DATABASE_URL_FILE")
	if dbURL == "" && dbURLFile == "" {
		logger.Error("DATABASE_URL or DATABASE_URL_FILE environment variable is required for Facilities Domain API")
		os.Exit(1)
	}

	pool, err := dbpool.NewDynamicPool(ctx, dbURL, dbURLFile)
	if err != nil {
		logger.Error("failed to connect to database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected Facilities Domain API to PostgreSQL pool successfully")

	repo := facilities_repo.NewRepository(pool)
	svc := facilities_svc.NewService(repo)

	var tlsConfig *tls.Config
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if certFile != "" && keyFile != "" {
		var err error
		tlsConfig, err = mtls.NewServerTLSConfig(certFile, keyFile, clientCA)
		if err != nil {
			logger.Error("failed to configure mTLS", "error", err)
			os.Exit(1)
		}
		logger.Info("Facilities Domain API configured with dynamic mTLS")
	}

	var serverOpts []grpc.ServerOption
	if tlsConfig != nil {
		serverOpts = append(serverOpts, grpcutil.WithAllowedPeerSPIFFEIDs(grpcutil.AllowedDomainAPICallers(grpcutil.ResolveNamespace())...)...)
	}
	grpcServer := grpcutil.NewServer(tlsConfig, serverOpts...)
	grpcSrv := NewFacilitiesGRPCServerWithRepo(svc, repo)
	facilitiesv1.RegisterFacilitiesServiceServer(grpcServer, grpcSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("oci.facilities.v1.FacilitiesService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen on gRPC port", "port", port, "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Facilities Domain API listening (pure gRPC)", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Facilities Domain API gRPC server error", "error", err)
		}
	}()

	<-stop
	logger.Info("shutting down Facilities Domain API...")

	grpcServer.GracefulStop()
	logger.Info("Facilities Domain API stopped gracefully")
}
