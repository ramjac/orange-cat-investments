package internalbll

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/auditintent"
	"github.com/orange-cat-investments/oci/pkg/authz/policy"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	"google.golang.org/grpc/status"
)

// EventPublisher publishes domain event payloads to a topic.
type EventPublisher = wfService.EventPublisher

type watermillPublisherAdapter struct {
	pub message.Publisher
}

func (a *watermillPublisherAdapter) Publish(topic string, payload any) error {
	if a.pub == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := message.NewMessage(fmt.Sprintf("msg-%d", time.Now().UnixNano()), data)
	return a.pub.Publish(topic, msg)
}

func (a *watermillPublisherAdapter) PublishWithContext(ctx context.Context, topic string, payload any) error {
	if a.pub == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := message.NewMessage(fmt.Sprintf("msg-%d", time.Now().UnixNano()), data)
	msg.SetContext(ctx)
	return a.pub.Publish(topic, msg)
}

// NewEventPublisherFromWatermill wraps a Watermill message.Publisher as an EventPublisher.
func NewEventPublisherFromWatermill(pub message.Publisher) EventPublisher {
	if pub == nil {
		return nil
	}
	return &watermillPublisherAdapter{pub: pub}
}

// Server implements internalbllv1.InternalBLLServiceServer.
type Server struct {
	internalbllv1.UnimplementedInternalBLLServiceServer
	wfClient        workforcev1.WorkforceServiceClient
	facClient       facilitiesv1.FacilitiesServiceClient
	opsClient       opsv1.OpsServiceClient
	ciClient        coreinvestv1.CoreInvestServiceClient
	wfSvc           wfService.Service
	workspaceClient wfService.WorkspaceClient
	vcsClient       saga.VCSClient
	hwClient        saga.HardwareClient
	publisher       EventPublisher
	logger          *slog.Logger
	meiliClient     *meilisearch.Client
	policyEngine    *policy.Engine

	// auditRecorder durably spools audit intents whose synchronous ledger
	// append failed; auditRelay re-delivers them until ops-api accepts them.
	auditRecorder *auditintent.Recorder
	auditRelay    *auditintent.Relay
}

// Config configures optional COTS and saga dependencies for the Internal BLL server.
type Config struct {
	HRClient         wfService.HRClient
	WorkspaceClient  wfService.WorkspaceClient
	VCSClient        saga.VCSClient
	HardwareClient   saga.HardwareClient
	WorkforceSvc     wfService.Service
	CoreInvestClient coreinvestv1.CoreInvestServiceClient
	SearchClient     *meilisearch.Client
	PolicyEngine     *policy.Engine
	// AuditSpoolDir overrides the durable audit-intent spool location
	// (default: $OCI_AUDIT_INTENT_DIR, else ~/.oci/audit-intents/internal-bll).
	AuditSpoolDir string
}

// NewServer creates a new Internal BLL gRPC server backed by Domain API gRPC clients.
func NewServer(
	wfClient workforcev1.WorkforceServiceClient,
	facClient facilitiesv1.FacilitiesServiceClient,
	opsClient opsv1.OpsServiceClient,
	publisher EventPublisher,
	logger *slog.Logger,
	cfgs ...Config,
) *Server {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	var cfg Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	wfSvc := cfg.WorkforceSvc
	if wfSvc == nil {
		wfSvc = wfService.NewServiceWithIntegrations(nil, publisher, cfg.HRClient, cfg.WorkspaceClient)
	}

	vcs := cfg.VCSClient
	if vcs == nil {
		vcs = saga.NewForgejoVCSAdapter(nil)
	}
	hw := cfg.HardwareClient
	if hw == nil {
		hw = saga.NewSnipeITHardwareAdapter(nil)
	}

	meiliClient := cfg.SearchClient
	if meiliClient == nil {
		meiliURL := os.Getenv("MEILISEARCH_URL")
		if meiliURL == "" {
			meiliURL = "http://localhost:7700"
		}
		meiliKey := os.Getenv("MEILI_MASTER_KEY")
		meiliClient = meilisearch.NewClient(meiliURL, meiliKey)
	}

	policyEngine := cfg.PolicyEngine
	if policyEngine == nil {
		if eng, err := policy.NewEngine(context.Background()); err == nil {
			policyEngine = eng
		} else {
			logger.Error("failed to compile embedded OPA policy engine", "error", err)
		}
	}

	var auditRecorder *auditintent.Recorder
	if opsClient != nil {
		spool, spoolErr := auditintent.NewFileSpool(resolveAuditSpoolDir(cfg.AuditSpoolDir, "internal-bll"), logger)
		if spoolErr != nil {
			logger.Error("failed to initialize durable audit-intent spool; audit fallback disabled", "error", spoolErr)
		} else {
			auditRecorder = auditintent.NewRecorder(spool, logger)
			logger.Info("durable audit-intent spool active", "spool_dir", resolveAuditSpoolDir(cfg.AuditSpoolDir, "internal-bll"))
		}
	}

	s := &Server{
		wfClient:        wfClient,
		facClient:       facClient,
		opsClient:       opsClient,
		ciClient:        cfg.CoreInvestClient,
		wfSvc:           wfSvc,
		workspaceClient: cfg.WorkspaceClient,
		vcsClient:       vcs,
		hwClient:        hw,
		publisher:       publisher,
		logger:          logger,
		meiliClient:     meiliClient,
		policyEngine:    policyEngine,
		auditRecorder:   auditRecorder,
	}

	// The relay delivers spooled intents through the server's own ops-api
	// client so a single wiring point serves both paths.
	if auditRecorder != nil {
		s.auditRelay = auditintent.NewRelay(auditRecorder.Spool(), s.deliverAuditIntent, auditintent.RelayConfig{Logger: logger})
		s.auditRelay.Start()
	}

	return s
}

// resolveAuditSpoolDir determines the durable audit-intent spool directory.
func resolveAuditSpoolDir(override, service string) string {
	if override != "" {
		return override
	}
	if dir := os.Getenv("OCI_AUDIT_INTENT_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".oci", "audit-intents", service)
	}
	return filepath.Join(os.TempDir(), "oci-audit-intents", service)
}

// Close stops the audit-intent relay and releases the spool.
func (s *Server) Close() {
	if s.auditRelay != nil {
		s.auditRelay.Stop()
	}
}

// SetCoreInvestClient sets the CoreInvestServiceClient on the server.
func (s *Server) SetCoreInvestClient(c coreinvestv1.CoreInvestServiceClient) {
	s.ciClient = c
}

// SetSearchClient sets the Meilisearch client on the server.
func (s *Server) SetSearchClient(c *meilisearch.Client) {
	s.meiliClient = c
}

func (s *Server) logDomainError(domainAPI, op string, err error) error {
	if err == nil {
		return nil
	}
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("domain API gRPC call failed",
		"domain_api", domainAPI,
		"operation", op,
		"grpc_code", status.Code(err).String(),
		"error", err,
	)
	return err
}
