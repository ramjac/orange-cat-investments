package customerbll

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/auditintent"
	"github.com/orange-cat-investments/oci/pkg/authz/policy"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
)

type Config struct {
	Logger           *slog.Logger
	Publisher        message.Publisher
	WorkforceTarget  string
	CoreInvestTarget string
	OpsTarget        string
	DefaultFelineID  string
	ClientTLS        *tls.Config
	BrokerClient     *alpaca.Client
	BankClient       *plaid.Client
	UpstreamTimeout  time.Duration
	PolicyEngine     *policy.Engine
	// SettlementDedup provides durable ACH settlement deduplication keyed by
	// transfer ID. When nil, NewServer builds one from VALKEY_ADDR (shared,
	// fail-closed) or falls back to local mode when unset.
	SettlementDedup *cachevalkey.OnceStore
	// AuditSpoolDir overrides the durable audit-intent spool location
	// (default: $OCI_AUDIT_INTENT_DIR, else ~/.oci/audit-intents/customer-bll).
	AuditSpoolDir string
}

type Server struct {
	customerbllv1.UnimplementedCustomerBLLServiceServer

	logger          *slog.Logger
	publisher       message.Publisher
	brokerClient    *alpaca.Client
	bankClient      *plaid.Client
	defaultFelineID string
	upstreamTimeout time.Duration
	policyEngine    *policy.Engine
	settlementDedup *cachevalkey.OnceStore

	// auditRecorder durably spools audit intents whose synchronous ledger
	// append failed; auditRelay re-delivers them until ops-api accepts them.
	auditRecorder *auditintent.Recorder
	auditRelay    *auditintent.Relay

	wfClient  workforcev1.WorkforceServiceClient
	ciClient  coreinvestv1.CoreInvestServiceClient
	opsClient opsv1.OpsServiceClient
	conns     []*grpc.ClientConn

	felineCacheMu sync.RWMutex
	felinesCached []*workforcev1.Employee
	felinesAt     time.Time

	vhrCacheMu sync.RWMutex
	vhrCached  map[string][]*workforcev1.VHRRecord
	vhrAt      time.Time
}

func NewServer(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	upstreamTimeout := cfg.UpstreamTimeout
	if upstreamTimeout <= 0 {
		if raw := os.Getenv("UPSTREAM_TIMEOUT"); raw != "" {
			if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
				upstreamTimeout = parsed
			}
		}
	}
	if upstreamTimeout <= 0 {
		upstreamTimeout = defaultUpstreamTimeout
	}
	policyEngine := cfg.PolicyEngine
	if policyEngine == nil {
		if eng, err := policy.NewEngine(context.Background()); err == nil {
			policyEngine = eng
		} else {
			logger.Error("failed to compile embedded OPA policy engine", "error", err)
		}
	}
	settlementDedup := cfg.SettlementDedup
	if settlementDedup == nil {
		valkeyAddr := os.Getenv("VALKEY_ADDR")
		if valkeyAddr != "" {
			settlementDedup = cachevalkey.NewOnceStoreWithAddr(valkeyAddr, os.Getenv("VALKEY_PASSWORD"), 0, logger, cachevalkey.DefaultOnceTTL)
			logger.Info("ACH settlement deduplication backed by shared Valkey (fail-closed claims)", "addr", valkeyAddr)
		}
	}
	s := &Server{
		logger:          logger,
		publisher:       cfg.Publisher,
		brokerClient:    cfg.BrokerClient,
		bankClient:      cfg.BankClient,
		defaultFelineID: cfg.DefaultFelineID,
		upstreamTimeout: upstreamTimeout,
		policyEngine:    policyEngine,
		settlementDedup: settlementDedup,
	}

	if cfg.WorkforceTarget != "" {
		target := grpcutil.CleanTarget(cfg.WorkforceTarget)
		if conn, err := grpcutil.Dial(context.Background(), target, cfg.ClientTLS); err == nil && conn != nil {
			s.wfClient = workforcev1.NewWorkforceServiceClient(conn)
			s.conns = append(s.conns, conn)
			logger.Info("Customer BLL connected to workforce domain API via gRPC", "target", target)
		}
	}
	if cfg.CoreInvestTarget != "" {
		target := grpcutil.CleanTarget(cfg.CoreInvestTarget)
		if conn, err := grpcutil.Dial(context.Background(), target, cfg.ClientTLS); err == nil && conn != nil {
			s.ciClient = coreinvestv1.NewCoreInvestServiceClient(conn)
			s.conns = append(s.conns, conn)
			logger.Info("Customer BLL connected to core-invest domain API via gRPC", "target", target)
		}
	}
	if cfg.OpsTarget != "" {
		target := grpcutil.CleanTarget(cfg.OpsTarget)
		if conn, err := grpcutil.Dial(context.Background(), target, cfg.ClientTLS); err == nil && conn != nil {
			s.opsClient = opsv1.NewOpsServiceClient(conn)
			s.conns = append(s.conns, conn)
			logger.Info("Customer BLL connected to ops domain API via gRPC", "target", target)
		}
	}

	if s.opsClient != nil {
		spool, spoolErr := auditintent.NewFileSpool(resolveCustomerAuditSpoolDir(cfg.AuditSpoolDir), logger)
		if spoolErr != nil {
			logger.Error("failed to initialize durable audit-intent spool; audit fallback disabled", "error", spoolErr)
		} else {
			s.auditRecorder = auditintent.NewRecorder(spool, logger)
			s.auditRelay = auditintent.NewRelay(spool, s.deliverAuditIntent, auditintent.RelayConfig{Logger: logger})
			s.auditRelay.Start()
			logger.Info("durable audit-intent spool active", "spool_dir", resolveCustomerAuditSpoolDir(cfg.AuditSpoolDir))
		}
	}

	if s.brokerClient == nil {
		brokerURL := os.Getenv("BROKER_API_BASE_URL")
		if brokerURL == "" {
			brokerURL = os.Getenv("ALPACA_BASE_URL")
		}
		if brokerURL != "" {
			apiKey := os.Getenv("APCA_API_KEY_ID")
			if apiKey == "" {
				apiKey = "CK_OCI_TEST_KEY_01" // #nosec G101 -- test mock fallback overridden by K8s Secret
			}
			apiSecret := os.Getenv("APCA_API_SECRET_KEY")
			if apiSecret == "" {
				apiSecret = "SK_OCI_TEST_SECRET_01" // #nosec G101 -- test mock fallback overridden by K8s Secret
			}
			s.brokerClient = alpaca.NewClient(alpaca.Config{
				BaseURL:   brokerURL,
				DataURL:   os.Getenv("ALPACA_DATA_URL"),
				APIKey:    apiKey,
				APISecret: apiSecret,
			})
			logger.Info("Customer BLL connected to broker", "broker_url", brokerURL)
		}
	}

	if s.bankClient == nil {
		bankURL := os.Getenv("BANK_API_BASE_URL")
		if bankURL == "" {
			bankURL = os.Getenv("PLAID_BASE_URL")
		}
		if bankURL != "" {
			clientID := os.Getenv("PLAID_CLIENT_ID")
			if clientID == "" {
				clientID = "client_oci_sim_01"
			}
			secret := os.Getenv("PLAID_SECRET")
			if secret == "" {
				secret = "sec_oci_sim_test_8829" // #nosec G101 -- test mock fallback overridden by K8s Secret
			}
			s.bankClient = plaid.NewClient(plaid.Config{
				BaseURL:  bankURL,
				ClientID: clientID,
				Secret:   secret,
			})
			logger.Info("Customer BLL connected to bank", "bank_url", bankURL)
		}
	}

	return s
}

func (s *Server) Close() {
	if s.auditRelay != nil {
		s.auditRelay.Stop()
	}
	for _, conn := range s.conns {
		_ = conn.Close()
	}
}

// resolveCustomerAuditSpoolDir determines the durable audit-intent spool directory.
func resolveCustomerAuditSpoolDir(override string) string {
	if override != "" {
		return override
	}
	if dir := os.Getenv("OCI_AUDIT_INTENT_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".oci", "audit-intents", "customer-bll")
	}
	return filepath.Join(os.TempDir(), "oci-audit-intents", "customer-bll")
}

func (s *Server) SetWorkforceClient(c workforcev1.WorkforceServiceClient) {
	s.wfClient = c
}

func (s *Server) SetCoreInvestClient(c coreinvestv1.CoreInvestServiceClient) {
	s.ciClient = c
}

func (s *Server) SetOpsClient(c opsv1.OpsServiceClient) {
	s.opsClient = c
}

func (s *Server) SetBrokerClient(c *alpaca.Client) {
	s.brokerClient = c
}

func (s *Server) SetBankClient(c *plaid.Client) {
	s.bankClient = c
}

func (s *Server) SetDefaultFelineID(id string) {
	s.defaultFelineID = id
}

func (s *Server) SetPublisher(p message.Publisher) {
	s.publisher = p
}

func (s *Server) SetUpstreamTimeout(d time.Duration) {
	s.upstreamTimeout = d
}

func (s *Server) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}
