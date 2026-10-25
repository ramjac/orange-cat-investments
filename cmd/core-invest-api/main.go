package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Core Invest Domain API microservice (pure gRPC)...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL environment variable is required for Core Invest Domain API")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected Core Invest Domain API to PostgreSQL pool successfully")

	repo := core_invest_repo.NewRepository(pool)
	svc := core_invest_svc.NewService(repo)

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
		logger.Info("Core Invest Domain API configured with dynamic mTLS")
	}

	grpcServer := grpcutil.NewServer(tlsConfig)
	grpcSrv := NewCoreInvestGRPCServer(svc, repo)
	coreinvestv1.RegisterCoreInvestServiceServer(grpcServer, grpcSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("oci.core_invest.v1.CoreInvestService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen on gRPC port", "port", port, "error", err)
		os.Exit(1)
	}

	healthServer := mtls.StartHealthListener(os.Getenv("HEALTH_PORT"), port, logger, "core-invest-api")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Core Invest Domain API listening (pure gRPC)", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Core Invest Domain API gRPC server error", "error", err)
		}
	}()

	<-stop
	logger.Info("shutting down Core Invest Domain API...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if healthServer != nil {
		_ = healthServer.Shutdown(shutdownCtx)
	}

	grpcServer.GracefulStop()
	logger.Info("Core Invest Domain API stopped gracefully")
}
