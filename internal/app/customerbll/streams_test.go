package customerbll

import (
	"context"
	"errors"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	getStatementDocFunc         func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error)
	listStatementsFunc          func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error)
	listAPIKeysFunc             func(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error)
	createAPIKeyFn              func(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error)
	listCameraStreamsFunc       func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error)
	generateMonthlyFunc         func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error)
	generate1099BFunc           func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error)
	listPortfoliosFunc          func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error)
	generateStmtFunc            func(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error)
	registerCustFunc            func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error)
	closeAccountFunc            func(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error)
	executeGDPRFunc             func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error)
	getGDPRCertFunc             func(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error)
	createRecurringDepositFunc  func(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error)
	listRecurringDepositsFunc   func(ctx context.Context, in *coreinvestv1.ListRecurringDepositsRequest) (*coreinvestv1.ListRecurringDepositsResponse, error)
	updatePortfolioBalanceFunc  func(ctx context.Context, in *coreinvestv1.UpdatePortfolioBalanceRequest) (*coreinvestv1.UpdatePortfolioBalanceResponse, error)
	advanceRecurringDepositFunc func(ctx context.Context, in *coreinvestv1.AdvanceRecurringDepositRequest) (*coreinvestv1.AdvanceRecurringDepositResponse, error)
	completeBacktestRunFn       func(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error)
}

func (m *mockCoreInvestClient) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	if m.listCameraStreamsFunc != nil {
		return m.listCameraStreamsFunc(ctx, in)
	}
	return &coreinvestv1.ListCameraStreamsResponse{}, nil
}

func (m *mockCoreInvestClient) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	if m.listStatementsFunc != nil {
		return m.listStatementsFunc(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) ListAPIKeys(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error) {
	if m.listAPIKeysFunc != nil {
		return m.listAPIKeysFunc(ctx, in)
	}
	return &coreinvestv1.ListAPIKeysResponse{APIKeys: []*coreinvestv1.CustomerAPIKey{}}, nil
}

func (m *mockCoreInvestClient) CreateAPIKey(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
	if m.createAPIKeyFn != nil {
		return m.createAPIKeyFn(ctx, in)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCoreInvestClient) GetStatementDocument(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
	if m.getStatementDocFunc != nil {
		return m.getStatementDocFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) GenerateMonthlyStatement(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
	if m.generateMonthlyFunc != nil {
		return m.generateMonthlyFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) GenerateForm1099B(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
	if m.generate1099BFunc != nil {
		return m.generate1099BFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) ListPortfolios(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
	if m.listPortfoliosFunc != nil {
		return m.listPortfoliosFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) GenerateStatement(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
	if m.generateStmtFunc != nil {
		return m.generateStmtFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) RegisterCustomer(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
	if m.registerCustFunc != nil {
		return m.registerCustFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) CloseAndLiquidateAccount(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
	if m.closeAccountFunc != nil {
		return m.closeAccountFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) ExecuteGDPRErasure(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
	if m.executeGDPRFunc != nil {
		return m.executeGDPRFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) GetGDPRErasureCertificate(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error) {
	if m.getGDPRCertFunc != nil {
		return m.getGDPRCertFunc(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *mockCoreInvestClient) CreateRecurringDeposit(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error) {
	if m.createRecurringDepositFunc != nil {
		return m.createRecurringDepositFunc(ctx, in)
	}
	return &coreinvestv1.CreateRecurringDepositResponse{
		Deposit: &coreinvestv1.RecurringDeposit{
			DepositID:   "dep-mock-01",
			CustomerID:  in.CustomerID,
			PortfolioID: in.PortfolioID,
			AmountUSD:   in.AmountUSD,
			Frequency:   in.Frequency,
			BankAccount: in.BankAccount,
			Status:      "active",
		},
	}, nil
}

func (m *mockCoreInvestClient) ListRecurringDeposits(ctx context.Context, in *coreinvestv1.ListRecurringDepositsRequest) (*coreinvestv1.ListRecurringDepositsResponse, error) {
	if m.listRecurringDepositsFunc != nil {
		return m.listRecurringDepositsFunc(ctx, in)
	}
	return &coreinvestv1.ListRecurringDepositsResponse{}, nil
}

func (m *mockCoreInvestClient) UpdatePortfolioBalance(ctx context.Context, in *coreinvestv1.UpdatePortfolioBalanceRequest) (*coreinvestv1.UpdatePortfolioBalanceResponse, error) {
	if m.updatePortfolioBalanceFunc != nil {
		return m.updatePortfolioBalanceFunc(ctx, in)
	}
	return &coreinvestv1.UpdatePortfolioBalanceResponse{}, nil
}

func (m *mockCoreInvestClient) AdvanceRecurringDeposit(ctx context.Context, in *coreinvestv1.AdvanceRecurringDepositRequest) (*coreinvestv1.AdvanceRecurringDepositResponse, error) {
	if m.advanceRecurringDepositFunc != nil {
		return m.advanceRecurringDepositFunc(ctx, in)
	}
	return &coreinvestv1.AdvanceRecurringDepositResponse{}, nil
}

func (m *mockCoreInvestClient) CompleteBacktestRun(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error) {
	if m.completeBacktestRunFn != nil {
		return m.completeBacktestRunFn(ctx, in)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func TestListCameraStreams(t *testing.T) {
	t.Run("nil ciClient returns empty streams response without error", func(t *testing.T) {
		s := NewServer(Config{})
		resp, err := s.ListCameraStreams(context.Background(), &customerbllv1.ListCameraStreamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Streams)
	})

	t.Run("ciClient returning error fallback to empty streams response without error", func(t *testing.T) {
		mockClient := &mockCoreInvestClient{
			listCameraStreamsFunc: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
				return nil, errors.New("gRPC connection failed")
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockClient)

		resp, err := s.ListCameraStreams(context.Background(), &customerbllv1.ListCameraStreamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Streams)
	})

	t.Run("ciClient returning camera streams success", func(t *testing.T) {
		expectedStreams := []*coreinvestv1.CameraStream{
			{
				StreamID:  "stream-001",
				AssetID:   "asset-cam-01",
				FelineID:  "fel-barnaby-01",
				StreamURL: "rtsp://live.oci.local/lounge1",
				Status:    "online",
			},
			{
				StreamID:  "stream-002",
				AssetID:   "asset-cam-02",
				FelineID:  "fel-cleo-02",
				StreamURL: "rtsp://live.oci.local/perch2",
				Status:    "online",
			},
		}

		mockClient := &mockCoreInvestClient{
			listCameraStreamsFunc: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
				return &coreinvestv1.ListCameraStreamsResponse{
					Streams: expectedStreams,
				}, nil
			},
		}
		s := NewServer(Config{})
		s.SetCoreInvestClient(mockClient)

		resp, err := s.ListCameraStreams(context.Background(), &customerbllv1.ListCameraStreamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Streams, 2)
		assert.Equal(t, "stream-001", resp.Streams[0].StreamID)
		assert.Equal(t, "stream-002", resp.Streams[1].StreamID)
	})
}

func TestCompleteBacktestRun_UninitializedClient(t *testing.T) {
	s := &Server{ciClient: nil}
	ctx := context.Background()

	req := &customerbllv1.CompleteBacktestRunRequest{
		RunID:   "run-001",
		Status:  "completed",
		Results: `{"return_pct": 12.5}`,
	}

	resp, err := s.CompleteBacktestRun(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Contains(t, st.Message(), "core invest domain client is not initialized")
}

func TestCompleteBacktestRun_ClientError(t *testing.T) {
	mockErr := status.Error(codes.NotFound, "backtest run not found")
	mockClient := &mockCoreInvestClient{
		completeBacktestRunFn: func(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error) {
			return nil, mockErr
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	req := &customerbllv1.CompleteBacktestRunRequest{
		RunID:   "run-invalid",
		Status:  "failed",
		Results: `{}`,
	}

	resp, err := s.CompleteBacktestRun(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, mockErr, err)
}

func TestCompleteBacktestRun_Success(t *testing.T) {
	expectedRun := &coreinvestv1.BacktestRun{
		RunID:          "run-123",
		ResearcherID:   "fel-01",
		StrategyName:   "Alpha Cat Strategy",
		Status:         "COMPLETED",
		InitialCapital: 100000.0,
		FinalCapital:   125000.0,
		SharpeRatio:    1.85,
		MaxDrawdown:    0.05,
		CreatedAt:      "2025-01-01T00:00:00Z",
	}

	mockClient := &mockCoreInvestClient{
		completeBacktestRunFn: func(ctx context.Context, in *coreinvestv1.CompleteBacktestRunRequest) (*coreinvestv1.CompleteBacktestRunResponse, error) {
			assert.Equal(t, "run-123", in.RunID)
			assert.Equal(t, "COMPLETED", in.Status)
			assert.Equal(t, `{"sharpe": 1.85, "alpha": 0.05}`, in.Results)
			return &coreinvestv1.CompleteBacktestRunResponse{
				Run: expectedRun,
			}, nil
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	req := &customerbllv1.CompleteBacktestRunRequest{
		RunID:   "run-123",
		Status:  "COMPLETED",
		Results: `{"sharpe": 1.85, "alpha": 0.05}`,
	}

	resp, err := s.CompleteBacktestRun(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, expectedRun, resp.Run)
}

func TestListCameraStreams_UninitializedClient(t *testing.T) {
	s := &Server{ciClient: nil}
	ctx := context.Background()

	resp, err := s.ListCameraStreams(ctx, &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Streams)
}

func TestListCameraStreams_ClientError(t *testing.T) {
	mockClient := &mockCoreInvestClient{
		listCameraStreamsFunc: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
			return nil, errors.New("rpc connection failed")
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	resp, err := s.ListCameraStreams(ctx, &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Streams)
}

func TestListCameraStreams_Success(t *testing.T) {
	mockStreams := []*coreinvestv1.CameraStream{
		{StreamID: "cam-01", StreamURL: "rtsp://camera1.local/live", Status: "active"},
		{StreamID: "cam-02", StreamURL: "rtsp://camera2.local/live", Status: "active"},
	}

	mockClient := &mockCoreInvestClient{
		listCameraStreamsFunc: func(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
			return &coreinvestv1.ListCameraStreamsResponse{Streams: mockStreams}, nil
		},
	}

	s := &Server{}
	s.SetCoreInvestClient(mockClient)
	ctx := context.Background()

	resp, err := s.ListCameraStreams(ctx, &customerbllv1.ListCameraStreamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Streams, 2)
	assert.Equal(t, mockStreams, resp.Streams)
}
