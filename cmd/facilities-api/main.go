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
	facilities_repo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting OCI Facilities Domain API microservice (pure gRPC)...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "50052"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pool *pgxpool.Pool
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(ctx, dbURL)
		if err != nil {
			logger.Warn("failed to connect to database pool, falling back to mock persistence", "error", err)
		} else {
			defer pool.Close()
			logger.Info("connected Facilities Domain API to PostgreSQL pool successfully")
		}
	} else {
		logger.Info("DATABASE_URL not set, running Facilities Domain API with in-memory persistence fallback")
	}

	var repo facilities_repo.Repository
	if pool != nil {
		repo = facilities_repo.NewRepository(pool)
	} else {
		repo = facilities_repo.NewMockRepository()
	}

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

	grpcServer := grpcutil.NewServer(tlsConfig)
	grpcSrv := NewFacilitiesGRPCServer(svc)
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

	healthServer := mtls.StartHealthListener(os.Getenv("HEALTH_PORT"), port, logger, "facilities-api")

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if healthServer != nil {
		_ = healthServer.Shutdown(shutdownCtx)
	}

	grpcServer.GracefulStop()
	logger.Info("Facilities Domain API stopped gracefully")
}
