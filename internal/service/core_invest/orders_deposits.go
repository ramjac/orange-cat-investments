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

type CreateRecurringDepositOpts struct {
	CustomerID  string
	PortfolioID string
	AmountUSD   float64
	Frequency   string
	BankAccount string
	AccountID   string // External Bank Account ID
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

	if s.brokerClient == nil {
		return nil, errors.New("broker client is not configured: order execution unavailable")
	}

	brokerOrd, err := s.brokerClient.CreateOrder(ctx, opts.PortfolioID, alpaca.CreateOrderRequest{
		Symbol:        opts.Symbol,
		Qty:           opts.Quantity,
		Side:          opts.Side,
		Type:          "market",
		ClientOrderID: createdOrder.OrderID,
	})
	if err != nil {
		return nil, fmt.Errorf("broker order creation failed: %w", err)
	}

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

func (s *coreInvestService) AdvanceRecurringDepositSchedule(ctx context.Context, id string, nextChargeAt time.Time) error {
	return s.repo.AdvanceRecurringDepositSchedule(ctx, id, nextChargeAt)
}
