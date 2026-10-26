package core_invest

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/reporting"
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
	ExchangePublicToken(ctx context.Context, publicToken, clientUserID string) (string, error)
}

type StreamBacktestService interface {
	RegisterCameraStream(ctx context.Context, cameraAssetID, streamURL, protocol string) (*core_invest.CameraStream, error)
	GetCameraStream(ctx context.Context, id string) (*core_invest.CameraStream, error)
	ListCameraStreams(ctx context.Context, limit, offset int32) ([]*core_invest.CameraStream, error)
	StartBacktestRun(ctx context.Context, strategyID string, startDate, endDate time.Time, parameters string) (*core_invest.BacktestRun, error)
	GetBacktestRun(ctx context.Context, id string) (*core_invest.BacktestRun, error)
	ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*core_invest.BacktestRun, error)
	ListMLModels(ctx context.Context) ([]map[string]interface{}, error)
	LogModelDrift(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
}

type OrderDepositService interface {
	ExecuteBrokerageOrder(ctx context.Context, opts ExecuteBrokerageOrderOpts) (*core_invest.BrokerageOrder, error)
	GetBrokerageOrder(ctx context.Context, id string) (*core_invest.BrokerageOrder, error)
	ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*core_invest.BrokerageOrder, error)
	CreateRecurringDeposit(ctx context.Context, opts CreateRecurringDepositOpts) (*core_invest.RecurringDeposit, error)
	ListRecurringDeposits(ctx context.Context, customerID string) ([]*core_invest.RecurringDeposit, error)
	AdvanceRecurringDepositSchedule(ctx context.Context, id string, nextChargeAt time.Time) error
}

type PortfolioDeveloperService interface {
	GetPortfolio(ctx context.Context, customerOrPortfolioID string) (*core_invest.Portfolio, error)
	ListPortfolios(ctx context.Context) ([]*core_invest.Portfolio, error)
	CreatePortfolio(ctx context.Context, p *core_invest.Portfolio) (*core_invest.Portfolio, error)
	CreateCustomerAPIKey(ctx context.Context, key *core_invest.CustomerAPIKey) (*core_invest.CustomerAPIKey, error)
	ListCustomerAPIKeys(ctx context.Context, customerID string) ([]*core_invest.CustomerAPIKey, error)
	RevokeCustomerAPIKey(ctx context.Context, customerID, keyID string) error
	CreateWebhookSubscription(ctx context.Context, sub *core_invest.WebhookSubscription) (*core_invest.WebhookSubscription, error)
	ListWebhookSubscriptions(ctx context.Context, customerID string) ([]*core_invest.WebhookSubscription, error)
	DeleteWebhookSubscription(ctx context.Context, customerID, subscriptionID string) error
}

type CustomerLifecycleService interface {
	RegisterCustomer(ctx context.Context, req CustomerRegistrationRequest) (*CustomerRegistrationResult, error)
	CloseAndLiquidateAccount(ctx context.Context, req CloseAccountRequest) (*LiquidationResult, error)
	ExecuteGDPRErasure(ctx context.Context, customerID, email string) (*reporting.GDPRDeletionCertificate, error)
	GetGDPRErasureCertificate(ctx context.Context, id string) (*core_invest.GDPRCertificateRecord, error)
}

type StatementTaxService interface {
	GenerateStatement(ctx context.Context, portfolioID string, year int) (map[string]interface{}, error)
	ListStatements(ctx context.Context, customerID string) ([]*core_invest.InvestmentStatement, error)
	GetStatement(ctx context.Context, statementID string) (*core_invest.InvestmentStatement, error)
	GenerateMonthlyStatement(ctx context.Context, customerID, portfolioID string, year, month int) (*core_invest.InvestmentStatement, []byte, error)
	GenerateForm1099B(ctx context.Context, customerID, portfolioID string, taxYear int) (*core_invest.InvestmentStatement, []byte, error)
	GetStatementDocument(ctx context.Context, customerID, statementID string) (*core_invest.InvestmentStatement, []byte, error)
}

type Service interface {
	StreamBacktestService
	OrderDepositService
	PortfolioDeveloperService
	CustomerLifecycleService
	StatementTaxService
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

func NewServiceWithAll(repo core_invest.Repository, publisher EventPublisher, broker BrokerClient, bank BankClient) Service {
	return &coreInvestService{repo: repo, publisher: publisher, brokerClient: broker, bankClient: bank}
}
