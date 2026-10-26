package customerbll

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc"
)

type Config struct {
	Logger          *slog.Logger
	Publisher       message.Publisher
	WorkforceTarget string
	CoreInvestTarget string
	OpsTarget       string
	DefaultFelineID string
	ClientTLS       *tls.Config
	BrokerClient    *alpaca.Client
	BankClient      *plaid.Client
}

type Server struct {
	customerbllv1.UnimplementedCustomerBLLServiceServer

	logger          *slog.Logger
	publisher       message.Publisher
	brokerClient    *alpaca.Client
	bankClient      *plaid.Client
	defaultFelineID string

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
	s := &Server{
		logger:          logger,
		publisher:       cfg.Publisher,
		brokerClient:    cfg.BrokerClient,
		bankClient:      cfg.BankClient,
		defaultFelineID: cfg.DefaultFelineID,
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

	if s.brokerClient == nil {
		brokerURL := os.Getenv("BROKER_API_BASE_URL")
		if brokerURL == "" {
			brokerURL = os.Getenv("ALPACA_BASE_URL")
		}
		if brokerURL != "" {
			apiKey := os.Getenv("APCA_API_KEY_ID")
			if apiKey == "" {
				apiKey = "CK_OCI_TEST_KEY_01"
			}
			apiSecret := os.Getenv("APCA_API_SECRET_KEY")
			if apiSecret == "" {
				apiSecret = "SK_OCI_TEST_SECRET_01"
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
				secret = "sec_oci_sim_test_8829"
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
	for _, conn := range s.conns {
		_ = conn.Close()
	}
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

func (s *Server) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}
