package customerbll

import (
	"bytes"
	"log/slog"
	"net"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockPublisher struct{}

func (m *mockPublisher) Publish(topic string, messages ...interface{}) error {
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

type mockWorkforceClient struct {
	workforcev1.WorkforceServiceClient
}

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
}

type mockOpsClient struct {
	opsv1.OpsServiceClient
}

func TestNewServer_Defaults(t *testing.T) {
	// Clear any environment variables that might trigger client initialization
	t.Setenv("BROKER_API_BASE_URL", "")
	t.Setenv("ALPACA_BASE_URL", "")
	t.Setenv("BANK_API_BASE_URL", "")
	t.Setenv("PLAID_BASE_URL", "")

	srv := NewServer(Config{})
	require.NotNil(t, srv)
	defer srv.Close()

	assert.NotNil(t, srv.logger)
	assert.Nil(t, srv.publisher)
	assert.Nil(t, srv.brokerClient)
	assert.Nil(t, srv.bankClient)
	assert.Empty(t, srv.defaultFelineID)
	assert.Nil(t, srv.wfClient)
	assert.Nil(t, srv.ciClient)
	assert.Nil(t, srv.opsClient)

	// Verify log() helper fallback and behavior
	assert.NotNil(t, srv.log())
}

func TestNewServer_ExplicitConfig(t *testing.T) {
	buf := new(bytes.Buffer)
	customLogger := slog.New(slog.NewTextHandler(buf, nil))
	customBroker := alpaca.NewClient(alpaca.Config{BaseURL: "http://localhost:8080"})
	customBank := plaid.NewClient(plaid.Config{BaseURL: "http://localhost:8081"})
	customFelineID := "feline-99"

	cfg := Config{
		Logger:          customLogger,
		BrokerClient:    customBroker,
		BankClient:      customBank,
		DefaultFelineID: customFelineID,
	}

	srv := NewServer(cfg)
	require.NotNil(t, srv)
	defer srv.Close()

	assert.Equal(t, customLogger, srv.logger)
	assert.Equal(t, customBroker, srv.brokerClient)
	assert.Equal(t, customBank, srv.bankClient)
	assert.Equal(t, customFelineID, srv.defaultFelineID)
	assert.Equal(t, customLogger, srv.log())
}

func TestNewServer_BrokerClient_EnvVars(t *testing.T) {
	t.Run("BROKER_API_BASE_URL with explicit keys", func(t *testing.T) {
		t.Setenv("BROKER_API_BASE_URL", "http://mock-broker:8080")
		t.Setenv("ALPACA_DATA_URL", "http://mock-broker-data:8080")
		t.Setenv("APCA_API_KEY_ID", "CUSTOM_KEY")
		t.Setenv("APCA_API_SECRET_KEY", "CUSTOM_SECRET")
		t.Setenv("BANK_API_BASE_URL", "")
		t.Setenv("PLAID_BASE_URL", "")

		srv := NewServer(Config{})
		require.NotNil(t, srv)
		defer srv.Close()

		assert.NotNil(t, srv.brokerClient)
	})

	t.Run("ALPACA_BASE_URL with default keys", func(t *testing.T) {
		t.Setenv("BROKER_API_BASE_URL", "")
		t.Setenv("ALPACA_BASE_URL", "http://alpaca-mock:8080")
		t.Setenv("APCA_API_KEY_ID", "")
		t.Setenv("APCA_API_SECRET_KEY", "")
		t.Setenv("BANK_API_BASE_URL", "")
		t.Setenv("PLAID_BASE_URL", "")

		srv := NewServer(Config{})
		require.NotNil(t, srv)
		defer srv.Close()

		assert.NotNil(t, srv.brokerClient)
	})
}

func TestNewServer_BankClient_EnvVars(t *testing.T) {
	t.Run("BANK_API_BASE_URL with explicit keys", func(t *testing.T) {
		t.Setenv("BROKER_API_BASE_URL", "")
		t.Setenv("ALPACA_BASE_URL", "")
		t.Setenv("BANK_API_BASE_URL", "http://mock-bank:8081")
		t.Setenv("PLAID_CLIENT_ID", "CUSTOM_CLIENT_ID")
		t.Setenv("PLAID_SECRET", "CUSTOM_SECRET")

		srv := NewServer(Config{})
		require.NotNil(t, srv)
		defer srv.Close()

		assert.NotNil(t, srv.bankClient)
	})

	t.Run("PLAID_BASE_URL with default keys", func(t *testing.T) {
		t.Setenv("BROKER_API_BASE_URL", "")
		t.Setenv("ALPACA_BASE_URL", "")
		t.Setenv("BANK_API_BASE_URL", "")
		t.Setenv("PLAID_BASE_URL", "http://plaid-mock:8081")
		t.Setenv("PLAID_CLIENT_ID", "")
		t.Setenv("PLAID_SECRET", "")

		srv := NewServer(Config{})
		require.NotNil(t, srv)
		defer srv.Close()

		assert.NotNil(t, srv.bankClient)
	})
}

func TestNewServer_gRPC_Targets(t *testing.T) {
	t.Setenv("BROKER_API_BASE_URL", "")
	t.Setenv("ALPACA_BASE_URL", "")
	t.Setenv("BANK_API_BASE_URL", "")
	t.Setenv("PLAID_BASE_URL", "")

	// Create test listeners for workforce, core-invest, and ops gRPC targets
	lis1, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis1.Close()

	lis2, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis2.Close()

	lis3, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis3.Close()

	grpcSrv1 := grpc.NewServer()
	go func() { _ = grpcSrv1.Serve(lis1) }()
	defer grpcSrv1.Stop()

	grpcSrv2 := grpc.NewServer()
	go func() { _ = grpcSrv2.Serve(lis2) }()
	defer grpcSrv2.Stop()

	grpcSrv3 := grpc.NewServer()
	go func() { _ = grpcSrv3.Serve(lis3) }()
	defer grpcSrv3.Stop()

	cfg := Config{
		WorkforceTarget:  lis1.Addr().String(),
		CoreInvestTarget: lis2.Addr().String(),
		OpsTarget:        lis3.Addr().String(),
	}

	srv := NewServer(cfg)
	require.NotNil(t, srv)
	assert.Len(t, srv.conns, 3)
	assert.NotNil(t, srv.wfClient)
	assert.NotNil(t, srv.ciClient)
	assert.NotNil(t, srv.opsClient)

	// Test Close() closes all gRPC connections
	srv.Close()
}

func TestServer_SettersAndLog(t *testing.T) {
	srv := &Server{}

	// Test log() when logger is nil vs set
	assert.NotNil(t, srv.log())
	customLogger := slog.Default()
	srv.logger = customLogger
	assert.Equal(t, customLogger, srv.log())

	// Test Setters
	wfClient := &mockWorkforceClient{}
	ciClient := &mockCoreInvestClient{}
	opsClient := &mockOpsClient{}
	brokerClient := alpaca.NewClient(alpaca.Config{BaseURL: "http://localhost:8080"})
	bankClient := plaid.NewClient(plaid.Config{BaseURL: "http://localhost:8081"})

	srv.SetWorkforceClient(wfClient)
	assert.Equal(t, wfClient, srv.wfClient)

	srv.SetCoreInvestClient(ciClient)
	assert.Equal(t, ciClient, srv.ciClient)

	srv.SetOpsClient(opsClient)
	assert.Equal(t, opsClient, srv.opsClient)

	srv.SetBrokerClient(brokerClient)
	assert.Equal(t, brokerClient, srv.brokerClient)

	srv.SetBankClient(bankClient)
	assert.Equal(t, bankClient, srv.bankClient)

	srv.SetDefaultFelineID("feline-123")
	assert.Equal(t, "feline-123", srv.defaultFelineID)

	srv.SetPublisher(nil)
	assert.Nil(t, srv.publisher)
}
