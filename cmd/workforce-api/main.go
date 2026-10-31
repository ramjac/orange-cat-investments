package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"os"

	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/orange-cat-investments/oci/pkg/telemetry"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "workforce-api")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		defer shutdownTelemetry(ctx)
	}
	logger.Info("starting OCI Workforce Domain API microservice (pure gRPC)...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL environment variable is required for Workforce Domain API")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected Workforce Domain API to PostgreSQL pool successfully")

	repo := workforce.NewRepository(pool)
	svc := workforce_svc.NewService(repo)

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
		logger.Info("Workforce Domain API configured with dynamic mTLS")
	}

	enableFeline := os.Getenv("ENABLE_FELINE_WORKFORCE") != "false"
	if os.Getenv("ENABLE_SMALL_BUSINESS_MODE") == "true" && os.Getenv("ENABLE_FELINE_WORKFORCE") == "" {
		enableFeline = false
	}

	grpcServer := grpcutil.NewServer(tlsConfig)
	grpcSrv := NewWorkforceGRPCServerWithRepo(svc, repo, enableFeline)
	workforcev1.RegisterWorkforceServiceServer(grpcServer, grpcSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("oci.workforce.v1.WorkforceService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen on gRPC port", "port", port, "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Workforce Domain API listening (pure gRPC)", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Workforce Domain API gRPC server error", "error", err)
		}
	}()

	<-stop
	logger.Info("shutting down Workforce Domain API...")

	grpcServer.GracefulStop()
	logger.Info("Workforce Domain API stopped gracefully")
}
