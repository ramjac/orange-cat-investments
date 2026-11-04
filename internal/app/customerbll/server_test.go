package customerbll

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
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
	listEmployeesFunc  func(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error)
	listVHRRecordsFunc func(ctx context.Context, in *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error)
	listIncidentsFunc  func(ctx context.Context, in *workforcev1.ListIncidentsRequest) (*workforcev1.ListIncidentsResponse, error)
}

func (m *mockWorkforceClient) ListEmployees(ctx context.Context, in *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	if m.listEmployeesFunc != nil {
		return m.listEmployeesFunc(ctx, in)
	}
	return &workforcev1.ListEmployeesResponse{}, nil
}

func (m *mockWorkforceClient) ListVHRRecords(ctx context.Context, in *workforcev1.ListVHRRecordsRequest) (*workforcev1.ListVHRRecordsResponse, error) {
	if m.listVHRRecordsFunc != nil {
		return m.listVHRRecordsFunc(ctx, in)
	}
	return &workforcev1.ListVHRRecordsResponse{}, nil
}

func (m *mockWorkforceClient) ListIncidents(ctx context.Context, in *workforcev1.ListIncidentsRequest) (*workforcev1.ListIncidentsResponse, error) {
	if m.listIncidentsFunc != nil {
		return m.listIncidentsFunc(ctx, in)
	}
	return &workforcev1.ListIncidentsResponse{}, nil
}

type mockOpsClient struct {
	opsv1.OpsServiceClient
}

func TestNewServer(t *testing.T) {
	buf := new(bytes.Buffer)
	customLogger := slog.New(slog.NewTextHandler(buf, nil))
	customBroker := alpaca.NewClient(alpaca.Config{BaseURL: "http://localhost:8080"})
	customBank := plaid.NewClient(plaid.Config{BaseURL: "http://localhost:8081"})

	// Setup for gRPC target testing
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

	tests := []struct {
		name   string
		env    map[string]string
		cfg    Config
		verify func(t *testing.T, srv *Server)
	}{
		{
			name: "Defaults",
			env: map[string]string{
				"BROKER_API_BASE_URL": "",
				"ALPACA_BASE_URL":     "",
				"BANK_API_BASE_URL":   "",
				"PLAID_BASE_URL":      "",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.logger)
				assert.Nil(t, srv.publisher)
				assert.Nil(t, srv.brokerClient)
				assert.Nil(t, srv.bankClient)
				assert.Empty(t, srv.defaultFelineID)
				assert.Nil(t, srv.wfClient)
				assert.Nil(t, srv.ciClient)
				assert.Nil(t, srv.opsClient)
				assert.NotNil(t, srv.log())
			},
		},
		{
			name: "Explicit Config",
			env:  map[string]string{},
			cfg: Config{
				Logger:          customLogger,
				BrokerClient:    customBroker,
				BankClient:      customBank,
				DefaultFelineID: "feline-99",
			},
			verify: func(t *testing.T, srv *Server) {
				assert.Equal(t, customLogger, srv.logger)
				assert.Equal(t, customBroker, srv.brokerClient)
				assert.Equal(t, customBank, srv.bankClient)
				assert.Equal(t, "feline-99", srv.defaultFelineID)
				assert.Equal(t, customLogger, srv.log())
			},
		},
		{
			name: "BrokerClient via BROKER_API_BASE_URL with explicit keys",
			env: map[string]string{
				"BROKER_API_BASE_URL": "http://mock-broker:8080",
				"ALPACA_DATA_URL":     "http://mock-broker-data:8080",
				"APCA_API_KEY_ID":     "CUSTOM_KEY",
				"APCA_API_SECRET_KEY": "CUSTOM_SECRET",
				"BANK_API_BASE_URL":   "",
				"PLAID_BASE_URL":      "",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.brokerClient)
			},
		},
		{
			name: "BrokerClient via ALPACA_BASE_URL with default keys",
			env: map[string]string{
				"BROKER_API_BASE_URL": "",
				"ALPACA_BASE_URL":     "http://alpaca-mock:8080",
				"APCA_API_KEY_ID":     "",
				"APCA_API_SECRET_KEY": "",
				"BANK_API_BASE_URL":   "",
				"PLAID_BASE_URL":      "",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.brokerClient)
			},
		},
		{
			name: "BrokerClient EnvVars Overrides",
			env: map[string]string{
				"BROKER_API_BASE_URL": "http://mock-broker:8080",
				"ALPACA_DATA_URL":     "http://mock-broker-data:8080",
				"APCA_API_KEY_ID":     "CUSTOM_KEY_OVERRIDE",
				"APCA_API_SECRET_KEY": "CUSTOM_SECRET_OVERRIDE",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.brokerClient)
			},
		},
		{
			name: "BankClient via BANK_API_BASE_URL with explicit keys",
			env: map[string]string{
				"BROKER_API_BASE_URL": "",
				"ALPACA_BASE_URL":     "",
				"BANK_API_BASE_URL":   "http://mock-bank:8081",
				"PLAID_CLIENT_ID":     "CUSTOM_CLIENT_ID",
				"PLAID_SECRET":        "CUSTOM_SECRET",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.bankClient)
			},
		},
		{
			name: "BankClient via PLAID_BASE_URL with default keys",
			env: map[string]string{
				"BROKER_API_BASE_URL": "",
				"ALPACA_BASE_URL":     "",
				"BANK_API_BASE_URL":   "",
				"PLAID_BASE_URL":      "http://plaid-mock:8081",
				"PLAID_CLIENT_ID":     "",
				"PLAID_SECRET":        "",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.bankClient)
			},
		},
		{
			name: "BankClient EnvVars Overrides",
			env: map[string]string{
				"BANK_API_BASE_URL": "http://mock-bank:8081",
				"PLAID_CLIENT_ID":   "CUSTOM_CLIENT_ID_OVERRIDE",
				"PLAID_SECRET":      "CUSTOM_SECRET_OVERRIDE",
			},
			cfg: Config{},
			verify: func(t *testing.T, srv *Server) {
				assert.NotNil(t, srv.bankClient)
			},
		},
		{
			name: "gRPC Targets Success",
			env: map[string]string{
				"BROKER_API_BASE_URL": "",
				"ALPACA_BASE_URL":     "",
				"BANK_API_BASE_URL":   "",
				"PLAID_BASE_URL":      "",
			},
			cfg: Config{
				WorkforceTarget:  lis1.Addr().String(),
				CoreInvestTarget: lis2.Addr().String(),
				OpsTarget:        lis3.Addr().String(),
			},
			verify: func(t *testing.T, srv *Server) {
				assert.Len(t, srv.conns, 3)
				assert.NotNil(t, srv.wfClient)
				assert.NotNil(t, srv.ciClient)
				assert.NotNil(t, srv.opsClient)
			},
		},
		{
			name: "gRPC Targets DialError",
			env:  map[string]string{},
			cfg: Config{
				WorkforceTarget:  "invalid://\x00",
				CoreInvestTarget: "invalid://\x00",
				OpsTarget:        "invalid://\x00",
			},
			verify: func(t *testing.T, srv *Server) {
				assert.Nil(t, srv.wfClient)
				assert.Nil(t, srv.ciClient)
				assert.Nil(t, srv.opsClient)
				assert.Empty(t, srv.conns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			srv := NewServer(tt.cfg)
			require.NotNil(t, srv)
			defer srv.Close()
			tt.verify(t, srv)
		})
	}
}

func TestNewServer_ClientTLS(t *testing.T) {
	t.Setenv("BROKER_API_BASE_URL", "")
	t.Setenv("ALPACA_BASE_URL", "")
	t.Setenv("BANK_API_BASE_URL", "")
	t.Setenv("PLAID_BASE_URL", "")

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	grpcSrv := grpc.NewServer()
	go func() { _ = grpcSrv.Serve(lis) }()
	defer grpcSrv.Stop()

	cfg := Config{
		WorkforceTarget: lis.Addr().String(),
		ClientTLS:       nil,
	}

	srv := NewServer(cfg)
	require.NotNil(t, srv)
	defer srv.Close()

	assert.Len(t, srv.conns, 1)
	assert.NotNil(t, srv.wfClient)
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

func TestNewServer_LoggerOutput(t *testing.T) {
	t.Setenv("BROKER_API_BASE_URL", "")
	t.Setenv("ALPACA_BASE_URL", "")
	t.Setenv("BANK_API_BASE_URL", "")
	t.Setenv("PLAID_BASE_URL", "")

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

	buf := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(buf, nil))

	cfg := Config{
		Logger:           logger,
		WorkforceTarget:  lis1.Addr().String(),
		CoreInvestTarget: lis2.Addr().String(),
		OpsTarget:        lis3.Addr().String(),
	}

	t.Setenv("BROKER_API_BASE_URL", "http://mock-broker:8080")
	t.Setenv("BANK_API_BASE_URL", "http://mock-bank:8081")

	srv := NewServer(cfg)
	require.NotNil(t, srv)
	defer srv.Close()

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Customer BLL connected to workforce domain API via gRPC")
	assert.Contains(t, logOutput, lis1.Addr().String())
	assert.Contains(t, logOutput, "Customer BLL connected to core-invest domain API via gRPC")
	assert.Contains(t, logOutput, lis2.Addr().String())
	assert.Contains(t, logOutput, "Customer BLL connected to ops domain API via gRPC")
	assert.Contains(t, logOutput, lis3.Addr().String())
	assert.Contains(t, logOutput, "Customer BLL connected to broker")
	assert.Contains(t, logOutput, "http://mock-broker:8080")
	assert.Contains(t, logOutput, "Customer BLL connected to bank")
	assert.Contains(t, logOutput, "http://mock-bank:8081")
}
