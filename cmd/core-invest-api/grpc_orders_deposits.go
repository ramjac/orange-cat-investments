package main

import (
	"context"
	"time"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
)

func (s *CoreInvestGRPCServer) CreateOrder(ctx context.Context, in *coreinvestv1.CreateOrderRequest) (*coreinvestv1.CreateOrderResponse, error) {
	order, err := s.svc.ExecuteBrokerageOrder(ctx, core_invest_svc.ExecuteBrokerageOrderOpts{
		PortfolioID: in.PortfolioID,
		BrokerName:  "OCI Smart Execution Router",
		Symbol:      in.Symbol,
		Side:        in.Side,
		Quantity:    in.Quantity,
		Price:       in.Price,
	})
	if err != nil {
		// When core-invest-api runs as a pure persistence Domain API without an external brokerClient,
		// customer-bll has already executed the order against the broker and calls CreateOrder to persist it.
		if s.repo != nil {
			created, repoErr := s.repo.CreateBrokerageOrder(ctx, &core_invest_repo.BrokerageOrder{
				PortfolioID: in.PortfolioID,
				BrokerName:  "OCI Smart Execution Router",
				Symbol:      in.Symbol,
				Side:        in.Side,
				Quantity:    in.Quantity,
				Price:       in.Price,
				Status:      "filled",
			})
			if repoErr != nil {
				return nil, repoErr
			}
			now := time.Now().UTC()
			if updated, uErr := s.repo.UpdateBrokerageOrderStatus(ctx, created.OrderID, "filled", &now); uErr == nil && updated != nil {
				created = updated
			}
			order = created
		} else {
			return nil, err
		}
	}
	executed := ""
	if order.ExecutedAt != nil {
		executed = order.ExecutedAt.Format(time.RFC3339)
	}
	return &coreinvestv1.CreateOrderResponse{
		Order: &coreinvestv1.BrokerageOrder{
			OrderID:     order.OrderID,
			PortfolioID: order.PortfolioID,
			Symbol:      order.Symbol,
			Side:        order.Side,
			Quantity:    order.Quantity,
			Price:       order.Price,
			Status:      order.Status,
			BrokerName:  order.BrokerName,
			ExecutedAt:  executed,
		},
	}, nil
}

func (s *CoreInvestGRPCServer) GetOrder(ctx context.Context, in *coreinvestv1.GetOrderRequest) (*coreinvestv1.GetOrderResponse, error) {
	order, err := s.svc.GetBrokerageOrder(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	executed := ""
	if order.ExecutedAt != nil {
		executed = order.ExecutedAt.Format(time.RFC3339)
	}
	return &coreinvestv1.GetOrderResponse{
		Order: &coreinvestv1.BrokerageOrder{
			OrderID:     order.OrderID,
			PortfolioID: order.PortfolioID,
			Symbol:      order.Symbol,
			Side:        order.Side,
			Quantity:    order.Quantity,
			Price:       order.Price,
			Status:      order.Status,
			BrokerName:  order.BrokerName,
			ExecutedAt:  executed,
		},
	}, nil
}

func (s *CoreInvestGRPCServer) ListOrders(ctx context.Context, in *coreinvestv1.ListOrdersRequest) (*coreinvestv1.ListOrdersResponse, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	orders, err := s.svc.ListBrokerageOrders(ctx, limit, 0)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.BrokerageOrder
	for _, o := range orders {
		if in.PortfolioID != "" && o.PortfolioID != in.PortfolioID {
			continue
		}
		executed := ""
		if o.ExecutedAt != nil {
			executed = o.ExecutedAt.Format(time.RFC3339)
		}
		out = append(out, &coreinvestv1.BrokerageOrder{
			OrderID:     o.OrderID,
			PortfolioID: o.PortfolioID,
			Symbol:      o.Symbol,
			Side:        o.Side,
			Quantity:    o.Quantity,
			Price:       o.Price,
			Status:      o.Status,
			BrokerName:  o.BrokerName,
			ExecutedAt:  executed,
		})
	}
	return &coreinvestv1.ListOrdersResponse{Orders: out}, nil
}

func (s *CoreInvestGRPCServer) CreateRecurringDeposit(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error) {
	dep, err := s.svc.CreateRecurringDeposit(ctx, core_invest_svc.CreateRecurringDepositOpts{
		CustomerID:  in.CustomerID,
		PortfolioID: in.PortfolioID,
		AmountUSD:   in.AmountUSD,
		Frequency:   in.Frequency,
		BankAccount: in.BankAccount,
	})
	if err != nil {
		return nil, err
	}
	nextCharge := ""
	if !dep.NextChargeAt.IsZero() {
		nextCharge = dep.NextChargeAt.Format(time.RFC3339)
	}
	return &coreinvestv1.CreateRecurringDepositResponse{
		Deposit: &coreinvestv1.RecurringDeposit{
			DepositID:    dep.DepositID,
			CustomerID:   dep.CustomerID,
			PortfolioID:  dep.PortfolioID,
			AmountUSD:    dep.AmountUSD,
			Frequency:    dep.Frequency,
			BankAccount:  dep.BankAccount,
			Status:       dep.Status,
			NextChargeAt: nextCharge,
		},
	}, nil
}

func (s *CoreInvestGRPCServer) ListRecurringDeposits(ctx context.Context, in *coreinvestv1.ListRecurringDepositsRequest) (*coreinvestv1.ListRecurringDepositsResponse, error) {
	deps, err := s.svc.ListRecurringDeposits(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.RecurringDeposit
	for _, d := range deps {
		nextCharge := ""
		if !d.NextChargeAt.IsZero() {
			nextCharge = d.NextChargeAt.Format(time.RFC3339)
		}
		out = append(out, &coreinvestv1.RecurringDeposit{
			DepositID:    d.DepositID,
			CustomerID:   d.CustomerID,
			PortfolioID:  d.PortfolioID,
			AmountUSD:    d.AmountUSD,
			Frequency:    d.Frequency,
			BankAccount:  d.BankAccount,
			Status:       d.Status,
			NextChargeAt: nextCharge,
		})
	}
	return &coreinvestv1.ListRecurringDepositsResponse{Deposits: out}, nil
}

func (s *CoreInvestGRPCServer) AdvanceRecurringDeposit(ctx context.Context, in *coreinvestv1.AdvanceRecurringDepositRequest) (*coreinvestv1.AdvanceRecurringDepositResponse, error) {
	nextTime, err := time.Parse(time.RFC3339, in.NextChargeAt)
	if err != nil {
		nextTime = time.Now().AddDate(0, 1, 0)
	}
	if err := s.svc.AdvanceRecurringDepositSchedule(ctx, in.DepositID, nextTime); err != nil {
		return nil, err
	}
	return &coreinvestv1.AdvanceRecurringDepositResponse{Status: "advanced"}, nil
}

func (s *CoreInvestGRPCServer) ClaimDueRecurringDeposits(ctx context.Context, in *coreinvestv1.ClaimDueRecurringDepositsRequest) (*coreinvestv1.ClaimDueRecurringDepositsResponse, error) {
	deps, err := s.svc.ListRecurringDeposits(ctx, "")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var due []*coreinvestv1.RecurringDeposit
	for _, d := range deps {
		if d.Status != "active" && d.Status != "" {
			continue
		}
		if !d.NextChargeAt.IsZero() && d.NextChargeAt.After(now) {
			continue
		}
		nextCharge := now.AddDate(0, 1, 0)
		if d.Frequency == "weekly" {
			nextCharge = now.AddDate(0, 0, 7)
		} else if d.Frequency == "biweekly" {
			nextCharge = now.AddDate(0, 0, 14)
		}
		_ = s.svc.AdvanceRecurringDepositSchedule(ctx, d.DepositID, nextCharge)
		due = append(due, &coreinvestv1.RecurringDeposit{
			DepositID:    d.DepositID,
			CustomerID:   d.CustomerID,
			PortfolioID:  d.PortfolioID,
			AmountUSD:    d.AmountUSD,
			Frequency:    d.Frequency,
			BankAccount:  d.BankAccount,
			Status:       d.Status,
			NextChargeAt: nextCharge.Format(time.RFC3339),
		})
		if in.BatchSize > 0 && int32(len(due)) >= in.BatchSize {
			break
		}
	}
	return &coreinvestv1.ClaimDueRecurringDepositsResponse{Deposits: due}, nil
}
