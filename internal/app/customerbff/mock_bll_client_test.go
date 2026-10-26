package customerbff

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCustomerBLLClient struct {
	customerbllv1.CustomerBLLServiceClient
	mu           sync.Mutex
	portfolios   map[string]*customerbllv1.GetPortfolioResponse
	orders       map[string][]*coreinvestv1.BrokerageOrder
	deposits     map[string][]*coreinvestv1.RecurringDeposit
	bankAccounts map[string][]*customerbllv1.LinkedBankAccount
	apiKeys      map[string][]*coreinvestv1.CustomerAPIKey
	webhooks     map[string][]*coreinvestv1.CustomerWebhook
	tickets      map[string]*opsv1.ITTicket
	comments     map[string][]*opsv1.TicketComment
	statements   map[string][]*coreinvestv1.CustomerStatement
	streams      []*coreinvestv1.CameraStream
}

func newMockCustomerBLLClient() *mockCustomerBLLClient {
	m := &mockCustomerBLLClient{
		portfolios:   make(map[string]*customerbllv1.GetPortfolioResponse),
		orders:       make(map[string][]*coreinvestv1.BrokerageOrder),
		deposits:     make(map[string][]*coreinvestv1.RecurringDeposit),
		bankAccounts: make(map[string][]*customerbllv1.LinkedBankAccount),
		apiKeys:      make(map[string][]*coreinvestv1.CustomerAPIKey),
		webhooks:     make(map[string][]*coreinvestv1.CustomerWebhook),
		tickets:      make(map[string]*opsv1.ITTicket),
		comments:     make(map[string][]*opsv1.TicketComment),
		statements:   make(map[string][]*coreinvestv1.CustomerStatement),
		streams: []*coreinvestv1.CameraStream{
			{StreamID: "stream-01", AssetID: "asset-cam-01", FelineID: "fel-01", StreamURL: "rtsp://live.oci.local/h1", Status: "online"},
		},
	}
	m.statements["cust-longterm-arthur"] = []*coreinvestv1.CustomerStatement{
		{
			StatementID:   "stmt-018f-2026-02",
			CustomerID:    "cust-longterm-arthur",
			PortfolioID:   "port-cust-longterm-arthur",
			StatementType: "monthly_summary",
			Period:        "2026-02",
		},
	}
	m.webhooks["cust-longterm-arthur"] = []*coreinvestv1.CustomerWebhook{
		{
			WebhookID:  "sub-018f-1111",
			CustomerID: "cust-longterm-arthur",
			TargetURL:  "https://example.com/wh",
			EventTypes: []string{"events.trade.executed.v1"},
			Status:     "active",
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		},
	}
	return m
}

func (m *mockCustomerBLLClient) GetPortfolio(ctx context.Context, in *customerbllv1.GetPortfolioRequest) (*customerbllv1.GetPortfolioResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	custID := in.CustomerID
	if custID == "" {
		custID = in.PortfolioID
	}

	if p, ok := m.portfolios[custID]; ok {
		return p, nil
	}

	var p *customerbllv1.GetPortfolioResponse
	if custID == "cust-longterm-arthur" {
		p = &customerbllv1.GetPortfolioResponse{
			PortfolioID:     "port-" + custID,
			CustomerID:      custID,
			CustomerName:    "Arthur Pendelton",
			AccountName:     "OCI Investment Portfolio",
			TotalBalanceUSD: 154200.50,
			CashBalanceUSD:  24500.00,
			Status:          "active",
			Holdings: []*coreinvestv1.PortfolioHolding{
				{Symbol: "NVDA", Quantity: 50, AvgPriceUSD: 120.00, CurrentPriceUSD: 135.00, MarketValueUSD: 6750.00},
				{Symbol: "AAPL", Quantity: 100, AvgPriceUSD: 210.00, CurrentPriceUSD: 228.00, MarketValueUSD: 22800.00},
			},
			HistoricalPerformance: []*customerbllv1.ValuationPoint{
				{Date: "2026-01-01", Valuation: 120000.0},
				{Date: "2026-04-01", Valuation: 154200.50},
			},
		}
	} else if custID == "cust-active-chloe" {
		p = &customerbllv1.GetPortfolioResponse{
			PortfolioID:     "port-" + custID,
			CustomerID:      custID,
			CustomerName:    "Chloe Spark",
			AccountName:     "OCI Active Portfolio",
			TotalBalanceUSD: 45000.00,
			CashBalanceUSD:  12000.00,
			Status:          "active",
			Holdings:        []*coreinvestv1.PortfolioHolding{},
		}
	} else {
		name := custID
		p = &customerbllv1.GetPortfolioResponse{
			PortfolioID:     "port-" + custID,
			CustomerID:      custID,
			CustomerName:    name,
			AccountName:     "OCI Investment Portfolio",
			TotalBalanceUSD: 0.0,
			CashBalanceUSD:  0.0,
			Status:          "active",
			Holdings:        []*coreinvestv1.PortfolioHolding{},
		}
	}

	m.portfolios[custID] = p
	return p, nil
}

func (m *mockCustomerBLLClient) GetPortfolioMetrics(ctx context.Context, in *customerbllv1.GetPortfolioMetricsRequest) (*customerbllv1.GetPortfolioMetricsResponse, error) {
	return &customerbllv1.GetPortfolioMetricsResponse{
		PortfolioID:     in.PortfolioID,
		TotalValue:      154200.50,
		CashBalance:     24500.00,
		InvestedBalance: 129700.50,
		SharpeRatio:     1.85,
		SortinoRatio:    2.10,
		YTDReturnPct:    18.4,
		BetaToSP500:     0.92,
	}, nil
}

func (m *mockCustomerBLLClient) StreamPortfolioValuation(ctx context.Context, in *coreinvestv1.StreamPortfolioValuationRequest) (customerbllv1.CustomerBLLService_StreamPortfolioValuationClient, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented in mock")
}

func (m *mockCustomerBLLClient) RegisterCustomer(ctx context.Context, in *customerbllv1.RegisterCustomerRequest) (*customerbllv1.RegisterCustomerResponse, error) {
	if len(in.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters long")
	}
	custID := "cust-" + uuid.NewString()[:8]
	portID := "port-" + custID
	return &customerbllv1.RegisterCustomerResponse{
		Result: &coreinvestv1.RegisterCustomerResponse{
			CustomerID:  custID,
			Name:        in.FullName,
			Email:       in.Email,
			PortfolioID: portID,
			Status:      "active",
		},
	}, nil
}

func (m *mockCustomerBLLClient) CloseAccount(ctx context.Context, in *customerbllv1.CloseAccountRequest) (*customerbllv1.CloseAccountResponse, error) {
	return &customerbllv1.CloseAccountResponse{
		Result: &coreinvestv1.CloseAndLiquidateAccountResponse{
			CustomerID:            in.CustomerID,
			PortfolioID:           in.PortfolioID,
			AccountStatus:         "closed",
			ACHWithdrawalTransfer: "ach_tx_" + uuid.NewString()[:8],
		},
	}, nil
}

func (m *mockCustomerBLLClient) ExecuteGDPRErasure(ctx context.Context, in *customerbllv1.ExecuteGDPRErasureRequest) (*customerbllv1.ExecuteGDPRErasureResponse, error) {
	certID := "gdpr-" + uuid.NewString()[:8]
	return &customerbllv1.ExecuteGDPRErasureResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        certID,
			CustomerID:           in.CustomerID,
			CustomerIDHash:       fmt.Sprintf("hash-%s", in.CustomerID),
			CustomerEmailHash:    fmt.Sprintf("hash-%s", in.Email),
			ErasedAt:             time.Now().UTC().Format(time.RFC3339),
			LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
			SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
			AnonymizedTables:     []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions", "core_invest.recurring_deposits"},
			SessionsPurgedCount:  1,
			RecordsScrubbedCount: 4,
			ComplianceHash:       "hmac-sha256-mock-signature",
			ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
			Status:               "completed",
		},
	}, nil
}

func (m *mockCustomerBLLClient) GetGDPRCertificate(ctx context.Context, in *customerbllv1.GetGDPRCertificateRequest) (*customerbllv1.GetGDPRCertificateResponse, error) {
	return &customerbllv1.GetGDPRCertificateResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        in.CertificateID,
			CustomerIDHash:       "hash-customer",
			CustomerEmailHash:    "hash-email",
			ErasedAt:             time.Now().UTC().Format(time.RFC3339),
			LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
			SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
			AnonymizedTables:     []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions", "core_invest.recurring_deposits"},
			SessionsPurgedCount:  1,
			RecordsScrubbedCount: 4,
			ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
			Status:               "completed",
		},
		PDFBytes: []byte("%PDF-1.4 mock pdf"),
	}, nil
}

func (m *mockCustomerBLLClient) ValidateAndExecuteOrder(ctx context.Context, in *customerbllv1.ValidateAndExecuteOrderRequest) (*customerbllv1.ValidateAndExecuteOrderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if in.Symbol == "" || in.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid order parameters")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	orderID := "ord-" + uuid.NewString()[:8]
	bo := &coreinvestv1.BrokerageOrder{
		OrderID:     orderID,
		PortfolioID: in.PortfolioID,
		Symbol:      in.Symbol,
		Side:        in.Side,
		Quantity:    in.Quantity,
		Price:       in.LimitPrice,
		Status:      "executed",
		ExecutedAt:  now,
	}
	m.orders[in.CustomerID] = append(m.orders[in.CustomerID], bo)

	return &customerbllv1.ValidateAndExecuteOrderResponse{
		OrderID:     orderID,
		CustomerID:  in.CustomerID,
		PortfolioID: in.PortfolioID,
		Symbol:      in.Symbol,
		Side:        in.Side,
		Quantity:    in.Quantity,
		Price:       in.LimitPrice,
		Status:      "executed",
		ExecutedAt:  now,
	}, nil
}

func (m *mockCustomerBLLClient) ListTradeOrders(ctx context.Context, in *customerbllv1.ListTradeOrdersRequest) (*customerbllv1.ListTradeOrdersResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ords := m.orders[in.CustomerID]
	if ords == nil {
		ords = []*coreinvestv1.BrokerageOrder{}
	}
	return &customerbllv1.ListTradeOrdersResponse{Orders: ords}, nil
}

func (m *mockCustomerBLLClient) GetTickerSnapshot(ctx context.Context, in *customerbllv1.GetTickerSnapshotRequest) (*customerbllv1.GetTickerSnapshotResponse, error) {
	quotes := []*customerbllv1.TickerQuote{
		{Symbol: "AAPL", Price: 228.50, ChangePct: 1.45, Volume: 15400200},
		{Symbol: "NVDA", Price: 135.20, ChangePct: 3.12, Volume: 42100800},
		{Symbol: "MSFT", Price: 442.80, ChangePct: -0.25, Volume: 9800400},
		{Symbol: "TSLA", Price: 262.10, ChangePct: 2.80, Volume: 18200300},
	}
	return &customerbllv1.GetTickerSnapshotResponse{Quotes: quotes}, nil
}

func (m *mockCustomerBLLClient) GetZoomieIndex(ctx context.Context, in *customerbllv1.GetZoomieIndexRequest) (*customerbllv1.GetZoomieIndexResponse, error) {
	return &customerbllv1.GetZoomieIndexResponse{
		ZoomieIndexScore:      88.5,
		MomentumLevel:         "HIGH_VELOCITY_3AM_ZOOMIES",
		FelineLeader:          "Barnaby",
		ActivePouncesCnt:      14,
		CurrentActivity:       "zooming",
		ConfidenceScore:       0.965,
		TradingSignal:         "TACTICAL_BUY_BULLISH",
		LastUpdated:           time.Now().UTC().Format(time.RFC3339),
		RecommendedMultiplier: 1.75,
	}, nil
}

func (m *mockCustomerBLLClient) InitiateACHDeposit(ctx context.Context, in *customerbllv1.InitiateACHDepositRequest) (*customerbllv1.InitiateACHDepositResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dep := &coreinvestv1.RecurringDeposit{
		DepositID:   "dep-" + uuid.NewString()[:8],
		PortfolioID: in.PortfolioID,
		AmountUSD:   in.AmountUSD,
		Frequency:   in.Frequency,
		Status:      "pending",
	}
	m.deposits[in.CustomerID] = append(m.deposits[in.CustomerID], dep)
	return &customerbllv1.InitiateACHDepositResponse{
		TransferID:  dep.DepositID,
		CustomerID:  in.CustomerID,
		PortfolioID: in.PortfolioID,
		AmountUSD:   in.AmountUSD,
		Frequency:   in.Frequency,
		Status:      "pending",
	}, nil
}

func (m *mockCustomerBLLClient) ListDeposits(ctx context.Context, in *customerbllv1.ListDepositsRequest) (*customerbllv1.ListDepositsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	deps := m.deposits[in.CustomerID]
	if deps == nil {
		deps = []*coreinvestv1.RecurringDeposit{}
	}
	return &customerbllv1.ListDepositsResponse{Deposits: deps}, nil
}

func (m *mockCustomerBLLClient) ListBankAccounts(ctx context.Context, in *customerbllv1.ListBankAccountsRequest) (*customerbllv1.ListBankAccountsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	accs := m.bankAccounts[in.CustomerID]
	if accs == nil {
		accs = []*customerbllv1.LinkedBankAccount{
			{AccountID: "acc-bank-primary", InstitutionName: "First Feline Federal", AccountNumber: "•••4242", AvailableBalance: 52000.00},
		}
	}
	return &customerbllv1.ListBankAccountsResponse{Accounts: accs}, nil
}

func (m *mockCustomerBLLClient) SettleACHTransfer(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
	return &customerbllv1.SettleACHTransferResponse{
		TransferID: in.TransferID,
		Status:     "settled",
		SettledAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (m *mockCustomerBLLClient) ClaimDueRecurringDeposits(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error) {
	return &customerbllv1.ClaimDueRecurringDepositsResponse{DueDeposits: []*customerbllv1.RecurringDepositSummary{}}, nil
}

func (m *mockCustomerBLLClient) ListAPIKeys(ctx context.Context, in *customerbllv1.ListAPIKeysRequest) (*customerbllv1.ListAPIKeysResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := m.apiKeys[in.CustomerID]
	if keys == nil {
		keys = []*coreinvestv1.CustomerAPIKey{}
	}
	return &customerbllv1.ListAPIKeysResponse{Keys: keys}, nil
}

func (m *mockCustomerBLLClient) CreateAPIKey(ctx context.Context, in *customerbllv1.CreateAPIKeyRequest) (*customerbllv1.CreateAPIKeyResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keyID := "key-" + uuid.NewString()[:8]
	key := &coreinvestv1.CustomerAPIKey{
		KeyID:      keyID,
		CustomerID: in.CustomerID,
		Label:      in.Name,
		KeyPrefix:  "oci_live_",
		Scopes:     in.Permissions,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	m.apiKeys[in.CustomerID] = append(m.apiKeys[in.CustomerID], key)
	return &customerbllv1.CreateAPIKeyResponse{
		Key:       key,
		RawSecret: "oci_live_" + uuid.NewString(),
	}, nil
}

func (m *mockCustomerBLLClient) RevokeAPIKey(ctx context.Context, in *customerbllv1.RevokeAPIKeyRequest) (*customerbllv1.RevokeAPIKeyResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.apiKeys, in.CustomerID)
	return &customerbllv1.RevokeAPIKeyResponse{Status: "revoked"}, nil
}
