package internalbll

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/internal/saga"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
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
}

// Config configures optional COTS and saga dependencies for the Internal BLL server.
type Config struct {
	HRClient         wfService.HRClient
	WorkspaceClient  wfService.WorkspaceClient
	VCSClient        saga.VCSClient
	HardwareClient   saga.HardwareClient
	WorkforceSvc     wfService.Service
	CoreInvestClient coreinvestv1.CoreInvestServiceClient
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

	return &Server{
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
	}
}

// SetCoreInvestClient sets the CoreInvestServiceClient on the server.
func (s *Server) SetCoreInvestClient(c coreinvestv1.CoreInvestServiceClient) {
	s.ciClient = c
}
