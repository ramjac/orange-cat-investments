package core_invest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
)

type EventPublisher interface {
	Publish(topic string, payload any) error
}

type BrokerClient interface {
	CreateOrder(ctx context.Context, accountID string, req alpaca.CreateOrderRequest) (*alpaca.Order, error)
	GetAccount(ctx context.Context, accountID string) (*alpaca.Account, error)
	ListPositions(ctx context.Context, accountID string) ([]alpaca.Position, error)
	ListOrders(ctx context.Context, accountID string, status string, limit int) ([]alpaca.Order, error)
}

type BankClient interface {
	CreateTransfer(ctx context.Context, req plaid.CreateTransferRequest) (*plaid.Transfer, error)
	GetTransfer(ctx context.Context, transferID string) (*plaid.Transfer, error)
	AuthorizeTransfer(ctx context.Context, accountID string, amount float64, transferType string) (*plaid.TransferAuthorization, error)
	GetBalances(ctx context.Context, accessToken string, accountIDs []string) ([]plaid.AccountWithBalance, error)
	GetAccounts(ctx context.Context, ownerID string) ([]plaid.Account, error)
}

type Service interface {
	// Camera Stream Ingestion Pipeline
	RegisterCameraStream(ctx context.Context, cameraAssetID, streamURL, protocol string) (*core_invest.CameraStream, error)
	GetCameraStream(ctx context.Context, id string) (*core_invest.CameraStream, error)
	ListCameraStreams(ctx context.Context, limit, offset int32) ([]*core_invest.CameraStream, error)

	// Automated Brokerage Execution
	ExecuteBrokerageOrder(ctx context.Context, opts ExecuteBrokerageOrderOpts) (*core_invest.BrokerageOrder, error)
	GetBrokerageOrder(ctx context.Context, id string) (*core_invest.BrokerageOrder, error)
	ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*core_invest.BrokerageOrder, error)

	// Bank & ACH Deposits
	CreateRecurringDeposit(ctx context.Context, opts CreateRecurringDepositOpts) (*core_invest.RecurringDeposit, error)
	ListRecurringDeposits(ctx context.Context, customerID string) ([]*core_invest.RecurringDeposit, error)

	// Dynamic Backtesting Workbench
	StartBacktestRun(ctx context.Context, strategyID string, startDate, endDate time.Time, parameters string) (*core_invest.BacktestRun, error)
	GetBacktestRun(ctx context.Context, id string) (*core_invest.BacktestRun, error)
	ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*core_invest.BacktestRun, error)

	// ML Models & Advanced Analytics
	ListMLModels(ctx context.Context) ([]map[string]interface{}, error)
	LogModelDrift(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
	ClassifyAcoustics(ctx context.Context, felineID string) (map[string]interface{}, error)
	GenerateStatement(ctx context.Context, portfolioID string, year int) (map[string]interface{}, error)
}

type CreateRecurringDepositOpts struct {
	CustomerID  string
	PortfolioID string
	AmountUSD   float64
	Frequency   string
	BankAccount string
	AccountID   string // External Bank Account ID
}

type coreInvestService struct {
	repo         core_invest.Repository
	publisher    EventPublisher
	brokerClient BrokerClient
	bankClient   BankClient
}

func NewService(repo core_invest.Repository) Service {
	return &coreInvestService{repo: repo}
}

func NewServiceWithPublisher(repo core_invest.Repository, publisher EventPublisher) Service {
	return &coreInvestService{repo: repo, publisher: publisher}
}

func NewServiceWithBroker(repo core_invest.Repository, broker BrokerClient) Service {
	return &coreInvestService{repo: repo, brokerClient: broker}
}

func NewServiceWithIntegrations(repo core_invest.Repository, broker BrokerClient, bank BankClient) Service {
	return &coreInvestService{repo: repo, brokerClient: broker, bankClient: bank}
}

func (s *coreInvestService) RegisterCameraStream(ctx context.Context, cameraAssetID, streamURL, protocol string) (*core_invest.CameraStream, error) {
	if cameraAssetID == "" || streamURL == "" {
		return nil, errors.New("camera_asset_id and stream_url are required")
	}
	if protocol != "rtsp" && protocol != "webrtc" {
		return nil, errors.New("protocol must be rtsp or webrtc")
	}

	stream := &core_invest.CameraStream{
		CameraAssetID:  cameraAssetID,
		StreamURL:      streamURL,
		Protocol:       protocol,
		Status:         "active",
		IngestSettings: "{}",
	}

	return s.repo.CreateCameraStream(ctx, stream)
}

func (s *coreInvestService) GetCameraStream(ctx context.Context, id string) (*core_invest.CameraStream, error) {
	if id == "" {
		return nil, errors.New("stream id cannot be empty")
	}
	return s.repo.GetCameraStreamByID(ctx, id)
}

func (s *coreInvestService) ListCameraStreams(ctx context.Context, limit, offset int32) ([]*core_invest.CameraStream, error) {
	return s.repo.ListCameraStreams(ctx, limit, offset)
}

type ExecuteBrokerageOrderOpts struct {
	PortfolioID string
	BrokerName  string
	Symbol      string
	Side        string
	Quantity    float64
	Price       float64
	StrategyID  *string
	EventID     *string
}

func (s *coreInvestService) ExecuteBrokerageOrder(ctx context.Context, opts ExecuteBrokerageOrderOpts) (*core_invest.BrokerageOrder, error) {
	if opts.PortfolioID == "" || opts.BrokerName == "" || opts.Symbol == "" {
		return nil, errors.New("portfolio_id, broker_name, and symbol are required")
	}
	if opts.Side != "buy" && opts.Side != "sell" {
		return nil, errors.New("side must be buy or sell")
	}
	if opts.Quantity <= 0 || opts.Price <= 0 {
		return nil, errors.New("quantity and price must be greater than zero")
	}

	order := &core_invest.BrokerageOrder{
		PortfolioID: opts.PortfolioID,
		StrategyID:  opts.StrategyID,
		EventID:     opts.EventID,
		BrokerName:  opts.BrokerName,
		Symbol:      opts.Symbol,
		Side:        opts.Side,
		Quantity:    opts.Quantity,
		Price:       opts.Price,
		Status:      "pending",
	}

	createdOrder, err := s.repo.CreateBrokerageOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Route through external Broker API if brokerClient is configured
	if s.brokerClient != nil {
		brokerOrd, err := s.brokerClient.CreateOrder(ctx, opts.PortfolioID, alpaca.CreateOrderRequest{
			Symbol:        opts.Symbol,
			Qty:           opts.Quantity,
			Side:          opts.Side,
			Type:          "market",
			ClientOrderID: createdOrder.OrderID,
		})
		if err == nil && brokerOrd != nil {
			fillPrice := opts.Price
			if brokerOrd.FilledAvgPrice != nil && *brokerOrd.FilledAvgPrice > 0 {
				fillPrice = *brokerOrd.FilledAvgPrice
			}
			now := time.Now().UTC()
			if brokerOrd.FilledAt != nil {
				now = *brokerOrd.FilledAt
			}
			status := "executed"
			if brokerOrd.Status != "" {
				status = brokerOrd.Status
			}
			resOrder, updateErr := s.repo.UpdateBrokerageOrderStatus(ctx, createdOrder.OrderID, status, &now)
			if updateErr == nil && resOrder != nil {
				resOrder.Price = fillPrice
			}
			return resOrder, updateErr
		}
	}

	// Fallback to internal execution simulation
	now := time.Now().UTC()
	return s.repo.UpdateBrokerageOrderStatus(ctx, createdOrder.OrderID, "executed", &now)
}

func (s *coreInvestService) GetBrokerageOrder(ctx context.Context, id string) (*core_invest.BrokerageOrder, error) {
	if id == "" {
		return nil, errors.New("order id cannot be empty")
	}
	return s.repo.GetBrokerageOrderByID(ctx, id)
}

func (s *coreInvestService) ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*core_invest.BrokerageOrder, error) {
	return s.repo.ListBrokerageOrders(ctx, limit, offset)
}

func (s *coreInvestService) CreateRecurringDeposit(ctx context.Context, opts CreateRecurringDepositOpts) (*core_invest.RecurringDeposit, error) {
	if opts.CustomerID == "" || opts.PortfolioID == "" {
		return nil, errors.New("customer_id and portfolio_id are required")
	}
	if opts.AmountUSD <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if opts.Frequency == "" {
		opts.Frequency = "monthly"
	}

	// If bank client is configured, check available balance via authorization
	if s.bankClient != nil {
		accountID := opts.AccountID
		accs, _ := s.bankClient.GetAccounts(ctx, opts.CustomerID)
		if accountID == "" && len(accs) > 0 {
			accountID = accs[0].AccountID
		}

		if opts.BankAccount == "" && len(accs) > 0 {
			targetAcc := accs[0]
			for _, a := range accs {
				if a.AccountID == accountID {
					targetAcc = a
					break
				}
			}
			mask := targetAcc.AccountNumber
			if len(mask) > 4 {
				mask = mask[len(mask)-4:]
			}
			if mask != "" {
				opts.BankAccount = fmt.Sprintf("%s (****%s)", targetAcc.Name, mask)
			} else {
				opts.BankAccount = targetAcc.Name
			}
		}

		if accountID != "" {
			auth, err := s.bankClient.AuthorizeTransfer(ctx, accountID, opts.AmountUSD, "debit")
			if err != nil {
				return nil, fmt.Errorf("bank authorization failed: %w", err)
			}
			if auth.Decision == "declined" {
				desc := "insufficient funds"
				if auth.DecisionRationale != nil && auth.DecisionRationale.Description != "" {
					desc = auth.DecisionRationale.Description
				}
				return nil, fmt.Errorf("bank declined deposit authorization: %s", desc)
			}

			// Also initiate the first transfer through Bank API
			_, _ = s.bankClient.CreateTransfer(ctx, plaid.CreateTransferRequest{
				AccountID:   accountID,
				Amount:      opts.AmountUSD,
				Direction:   "debit",
				Description: fmt.Sprintf("Recurring deposit for portfolio %s", opts.PortfolioID),
			})
		}
	}

	if opts.BankAccount == "" {
		opts.BankAccount = "Primary Linked Account"
	}

	nextCharge := time.Now().AddDate(0, 1, 0)
	if opts.Frequency == "weekly" {
		nextCharge = time.Now().AddDate(0, 0, 7)
	} else if opts.Frequency == "biweekly" {
		nextCharge = time.Now().AddDate(0, 0, 14)
	}

	dep := &core_invest.RecurringDeposit{
		CustomerID:   opts.CustomerID,
		PortfolioID:  opts.PortfolioID,
		AmountUSD:    opts.AmountUSD,
		Frequency:    opts.Frequency,
		BankAccount:  opts.BankAccount,
		Status:       "active",
		NextChargeAt: nextCharge,
	}

	return s.repo.CreateRecurringDeposit(ctx, dep)
}

func (s *coreInvestService) ListRecurringDeposits(ctx context.Context, customerID string) ([]*core_invest.RecurringDeposit, error) {
	return s.repo.ListRecurringDeposits(ctx, customerID)
}

func (s *coreInvestService) StartBacktestRun(ctx context.Context, strategyID string, startDate, endDate time.Time, parameters string) (*core_invest.BacktestRun, error) {
	if strategyID == "" {
		return nil, errors.New("strategy_id is required")
	}
	if endDate.Before(startDate) {
		return nil, errors.New("end_date must be after start_date")
	}
	if parameters == "" {
		parameters = "{}"
	}

	run := &core_invest.BacktestRun{
		StrategyID: strategyID,
		StartDate:  startDate,
		EndDate:    endDate,
		Parameters: parameters,
	}

	createdRun, err := s.repo.CreateBacktestRun(ctx, run)
	if err != nil {
		return nil, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish("events.investment.backtest_requested.v1", map[string]any{
			"backtest_id": createdRun.BacktestID,
			"strategy_id": createdRun.StrategyID,
		})
	}

	return createdRun, nil
}

func (s *coreInvestService) GetBacktestRun(ctx context.Context, id string) (*core_invest.BacktestRun, error) {
	if id == "" {
		return nil, errors.New("backtest id cannot be empty")
	}
	return s.repo.GetBacktestRunByID(ctx, id)
}

func (s *coreInvestService) ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*core_invest.BacktestRun, error) {
	return s.repo.ListBacktestRuns(ctx, limit, offset)
}

func (s *coreInvestService) ListMLModels(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{
			"model_id":   "mdl-yolov8-cat-pose-v3",
			"name":       "CatPose-YOLOv8-Alpha",
			"version":    "v3.2.0",
			"accuracy":   0.9845,
			"drift":      0.012,
			"status":     "active",
			"created_at": time.Now().AddDate(0, -2, 0).Format(time.RFC3339),
		},
		{
			"model_id":   "mdl-whisker-audio-v1",
			"name":       "PurrAcoustic-Classifier-V1",
			"version":    "v1.0.4",
			"accuracy":   0.9620,
			"drift":      0.008,
			"status":     "active",
			"created_at": time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		},
	}, nil
}

func (s *coreInvestService) LogModelDrift(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"drift_id":         "drift-018f-99",
		"status":           "recorded",
		"confidence_drift": 0.012,
		"evaluated_at":     time.Now().Format(time.RFC3339),
		"payload":          payload,
	}, nil
}

func (s *coreInvestService) ClassifyAcoustics(ctx context.Context, felineID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"acoustic_id":       "ac-018f-777",
		"feline_id":         felineID,
		"vocalization_type": "purring",
		"frequency_hz":      28.5,
		"decibel_level":     88.2,
		"confidence":        0.982,
	}, nil
}

func (s *coreInvestService) GenerateStatement(ctx context.Context, portfolioID string, year int) (map[string]interface{}, error) {
	if portfolioID == "" {
		return nil, errors.New("portfolio_id is required")
	}
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("invalid tax statement year %d: year must be between 2000 and 2100", year)
	}

	statementID := fmt.Sprintf("stmt-%s-%d", portfolioID, year)
	fileName := fmt.Sprintf("form1099b-%s-%d.pdf", portfolioID, year)
	documentURL := fmt.Sprintf("https://statements.oci.local/documents/%s/%d/%s", portfolioID, year, fileName)

	accountName := "OCI Alpha Growth Portfolio"
	totalVal := 128450.75

	return map[string]interface{}{
		"statement_id":        statementID,
		"portfolio_id":        portfolioID,
		"account_name":        accountName,
		"year":                year,
		"tax_year":            year,
		"document_name":       fileName,
		"form_1099b_url":      documentURL,
		"total_proceeds_usd":  totalVal,
		"sharpe_ratio":        2.14,
		"sortino_ratio":       3.08,
		"performance_summary": fmt.Sprintf("Feline behavioral alpha tax statement for portfolio %s, tax year %d", portfolioID, year),
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
	}, nil
}

