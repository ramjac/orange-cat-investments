package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"os"

	"os/signal"
	"syscall"

	"github.com/orange-cat-investments/oci/pkg/telemetry"

	"github.com/orange-cat-investments/oci/internal/app/internalbll"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/orange-cat-investments/oci/pkg/mtls"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.InitProvider(ctx, "internal-bll")
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", "error", err)
	}
	if shutdownTelemetry != nil {
		defer shutdownTelemetry(ctx)
	}
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	clientCA := os.Getenv("CLIENT_CA_FILE")
	if clientCA == "" {
		clientCA = os.Getenv("TLS_CA_FILE")
	}

	var serverTLS, clientTLS *tls.Config
	if certFile != "" && keyFile != "" {
		if sTLS, err := mtls.NewServerTLSConfig(certFile, keyFile, clientCA); err == nil {
			serverTLS = sTLS
		}
		if clientCA != "" {
			if cTLS, err := mtls.NewClientTLSConfig(certFile, keyFile, clientCA); err == nil {
				clientTLS = cTLS
			}
		}
	}

	wfAddr := os.Getenv("WORKFORCE_GRPC_ADDR")
	if wfAddr == "" {
		wfAddr = os.Getenv("WORKFORCE_API_URL")
	}
	if wfAddr == "" {
		wfAddr = "workforce-api-service.oci-core.svc.cluster.local:50051"
	}
	wfConn, err := grpcutil.Dial(context.Background(), wfAddr, clientTLS)
	if err != nil {
		logger.Error("failed to dial workforce-api", "error", err)
		os.Exit(1)
	}
	defer wfConn.Close()

	facAddr := os.Getenv("FACILITIES_GRPC_ADDR")
	if facAddr == "" {
		facAddr = os.Getenv("FACILITIES_API_URL")
	}
	if facAddr == "" {
		facAddr = "facilities-api-service.oci-core.svc.cluster.local:50052"
	}
	facConn, err := grpcutil.Dial(context.Background(), facAddr, clientTLS)
	if err != nil {
		logger.Error("failed to dial facilities-api", "error", err)
		os.Exit(1)
	}
	defer facConn.Close()

	opsAddr := os.Getenv("OPS_GRPC_ADDR")
	if opsAddr == "" {
		opsAddr = os.Getenv("OPS_API_URL")
	}
	if opsAddr == "" {
		opsAddr = "ops-api-service.oci-core.svc.cluster.local:50054"
	}
	opsConn, err := grpcutil.Dial(context.Background(), opsAddr, clientTLS)
	if err != nil {
		logger.Error("failed to dial ops-api", "error", err)
		os.Exit(1)
	}
	defer opsConn.Close()

	ciAddr := os.Getenv("CORE_INVEST_GRPC_ADDR")
	if ciAddr == "" {
		ciAddr = os.Getenv("CORE_INVEST_API_URL")
	}
	if ciAddr == "" {
		ciAddr = "core-invest-api-service.oci-core.svc.cluster.local:50053"
	}
	ciConn, err := grpcutil.Dial(context.Background(), ciAddr, clientTLS)
	if err != nil {
		logger.Warn("could not connect to core-invest-api", "error", err)
	} else {
		defer ciConn.Close()
	}

	var cfg internalbll.Config
	if ciConn != nil {
		cfg.CoreInvestClient = coreinvestv1.NewCoreInvestServiceClient(ciConn)
	}
	if frappeURL := os.Getenv("FRAPPE_URL"); frappeURL != "" && os.Getenv("FRAPPE_API_KEY") != "" {
		cfg.HRClient = wfService.HRClient(frappe.NewClient(frappe.Config{
			BaseURL:   frappeURL,
			APIKey:    os.Getenv("FRAPPE_API_KEY"),
			APISecret: os.Getenv("FRAPPE_API_SECRET"),
		}))
	}
	if ncURL := os.Getenv("NEXTCLOUD_URL"); ncURL != "" && os.Getenv("NEXTCLOUD_ADMIN_USER") != "" {
		cfg.WorkspaceClient = wfService.WorkspaceClient(nextcloud.NewClient(nextcloud.Config{
			BaseURL:       ncURL,
			AdminUser:     os.Getenv("NEXTCLOUD_ADMIN_USER"),
			AdminPassword: os.Getenv("NEXTCLOUD_ADMIN_PASS"),
		}))
	}
	if fgURL := os.Getenv("FORGEJO_URL"); fgURL != "" {
		cfg.VCSClient = saga.NewForgejoVCSAdapter(forgejo.NewClient(forgejo.Config{
			BaseURL: fgURL,
			Token:   os.Getenv("FORGEJO_API_TOKEN"),
		}))
	}

	amqpURI := os.Getenv("RABBITMQ_URL")
	if amqpURI == "" {
		amqpURI = os.Getenv("AMQP_URL")
	}
	wmPub := envelope.NewPublisherWithFallback(amqpURI, logger)
	defer func() { _ = wmPub.Close() }()

	bllSrv := internalbll.NewServer(
		workforcev1.NewWorkforceServiceClient(wfConn),
		facilitiesv1.NewFacilitiesServiceClient(facConn),
		opsv1.NewOpsServiceClient(opsConn),
		internalbll.NewEventPublisherFromWatermill(wmPub),
		logger,
		cfg,
	)

	grpcSrv := grpcutil.NewServer(serverTLS)
	internalbllv1.RegisterInternalBLLServiceServer(grpcSrv, bllSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("oci.internal_bll.v1.InternalBLLService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen", "port", port, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("Starting Internal Business Logic Layer (internal-bll) gRPC server", "port", port)
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("gRPC server stopped", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down internal-bll gRPC server...")
	grpcSrv.GracefulStop()
}
