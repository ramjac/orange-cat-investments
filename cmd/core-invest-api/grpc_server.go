package main

import (
	"context"
	"fmt"
	"time"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
)

type CoreInvestGRPCServer struct {
	svc  core_invest_svc.Service
	repo core_invest_repo.Repository
}

func NewCoreInvestGRPCServer(svc core_invest_svc.Service, repo core_invest_repo.Repository) *CoreInvestGRPCServer {
	return &CoreInvestGRPCServer{svc: svc, repo: repo}
}

func toProtoPortfolio(p *core_invest_repo.Portfolio) *coreinvestv1.Portfolio {
	if p == nil {
		return nil
	}
	created := ""
	if !p.CreatedAt.IsZero() {
		created = p.CreatedAt.Format(time.RFC3339)
	}
	return &coreinvestv1.Portfolio{
		PortfolioID:     p.PortfolioID,
		CustomerID:      p.CustomerID,
		AccountName:     p.AccountName,
		TotalBalanceUSD: p.TotalBalanceUSD,
		CashBalanceUSD:  p.CashBalanceUSD,
		Status:          p.Status,
		CreatedAt:       created,
	}
}

func (s *CoreInvestGRPCServer) GetPortfolio(ctx context.Context, in *coreinvestv1.GetPortfolioRequest) (*coreinvestv1.GetPortfolioResponse, error) {
	id := in.PortfolioID
	if id == "" {
		id = in.CustomerID
	}
	p, err := s.svc.GetPortfolio(ctx, id)
	if err != nil {
		return nil, err
	}

	var holdings []*coreinvestv1.PortfolioHolding
	if s.repo != nil && p != nil {
		repoHoldings, err := s.repo.GetHoldingsByPortfolioID(ctx, p.PortfolioID)
		if err == nil {
			for _, h := range repoHoldings {
				holdings = append(holdings, &coreinvestv1.PortfolioHolding{
					HoldingID:       h.HoldingID,
					PortfolioID:     h.PortfolioID,
					Symbol:          h.Symbol,
					Quantity:        h.Quantity,
					AvgPriceUSD:     h.AvgPriceUSD,
					CurrentPriceUSD: h.CurrentPriceUSD,
					MarketValueUSD:  h.MarketValueUSD,
				})
			}
		}
	}

	return &coreinvestv1.GetPortfolioResponse{
		Portfolio: toProtoPortfolio(p),
		Holdings:  holdings,
	}, nil
}

func (s *CoreInvestGRPCServer) ListHoldings(ctx context.Context, in *coreinvestv1.ListHoldingsRequest) (*coreinvestv1.ListHoldingsResponse, error) {
	var holdings []*coreinvestv1.PortfolioHolding
	if s.repo != nil {
		repoHoldings, err := s.repo.GetHoldingsByPortfolioID(ctx, in.PortfolioID)
		if err != nil {
			return nil, err
		}
		for _, h := range repoHoldings {
			holdings = append(holdings, &coreinvestv1.PortfolioHolding{
				HoldingID:       h.HoldingID,
				PortfolioID:     h.PortfolioID,
				Symbol:          h.Symbol,
				Quantity:        h.Quantity,
				AvgPriceUSD:     h.AvgPriceUSD,
				CurrentPriceUSD: h.CurrentPriceUSD,
				MarketValueUSD:  h.MarketValueUSD,
			})
		}
	}
	return &coreinvestv1.ListHoldingsResponse{Holdings: holdings}, nil
}

func (s *CoreInvestGRPCServer) UpdatePortfolioBalance(ctx context.Context, in *coreinvestv1.UpdatePortfolioBalanceRequest) (*coreinvestv1.UpdatePortfolioBalanceResponse, error) {
	p, err := s.svc.GetPortfolio(ctx, in.PortfolioID)
	if err != nil {
		return nil, err
	}

	newCash := p.CashBalanceUSD + in.DeltaUSD
	newTotal := p.TotalBalanceUSD + in.DeltaUSD

	if s.repo != nil {
		p.CashBalanceUSD = newCash
		p.TotalBalanceUSD = newTotal
		_, _ = s.repo.UpdatePortfolio(ctx, p)
	}

	return &coreinvestv1.UpdatePortfolioBalanceResponse{
		PortfolioID:            in.PortfolioID,
		UpdatedCashBalanceUSD:  newCash,
		UpdatedTotalBalanceUSD: newTotal,
		Status:                 "settled",
	}, nil
}

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
		return nil, err
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

func (s *CoreInvestGRPCServer) IngestObservation(ctx context.Context, in *coreinvestv1.IngestObservationRequest) (*coreinvestv1.IngestObservationResponse, error) {
	return &coreinvestv1.IngestObservationResponse{
		EventID: fmt.Sprintf("obs-%d", time.Now().UnixNano()),
		Status:  "ingested",
	}, nil
}

func (s *CoreInvestGRPCServer) ListObservations(ctx context.Context, in *coreinvestv1.ListObservationsRequest) (*coreinvestv1.ListObservationsResponse, error) {
	return &coreinvestv1.ListObservationsResponse{}, nil
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

func (s *CoreInvestGRPCServer) ListPortfolios(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
	ports, err := s.svc.ListPortfolios(ctx)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.Portfolio
	for _, p := range ports {
		out = append(out, toProtoPortfolio(p))
	}
	return &coreinvestv1.ListPortfoliosResponse{Portfolios: out}, nil
}

func (s *CoreInvestGRPCServer) GenerateStatement(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
	yr := int(in.Year)
	if yr == 0 {
		yr = time.Now().Year()
	}
	stmt, err := s.svc.GenerateStatement(ctx, in.PortfolioID, yr)
	if err != nil {
		return nil, err
	}
	stmtID, _ := stmt["statement_id"].(string)
	pID, _ := stmt["portfolio_id"].(string)
	docName, _ := stmt["document_name"].(string)
	return &coreinvestv1.GenerateStatementResponse{
		StatementID:   stmtID,
		PortfolioID:   pID,
		Year:          int32(yr),
		StatementType: "1099_b",
		Period:        docName,
		Status:        "generated",
	}, nil
}

func (s *CoreInvestGRPCServer) StreamPortfolioValuation(req *coreinvestv1.StreamPortfolioValuationRequest, stream coreinvestv1.CoreInvestService_StreamPortfolioValuationServer) error {
	count := req.UpdateCount
	if count <= 0 {
		count = 5
	}
	baseBalance := 10000.0
	cashBalance := 2500.0
	if req.PortfolioID != "" {
		if p, err := s.svc.GetPortfolio(stream.Context(), req.PortfolioID); err == nil && p != nil {
			baseBalance = p.TotalBalanceUSD
			cashBalance = p.CashBalanceUSD
		}
	}

	for i := 0; i < int(count); i++ {
		fluctuation := float64(i) * 12.50
		total := baseBalance + fluctuation
		invested := total - cashBalance
		changePct := (fluctuation / baseBalance) * 100.0

		err := stream.Send(&coreinvestv1.PortfolioValuationEvent{
			PortfolioID:        req.PortfolioID,
			TotalBalanceUSD:    total,
			CashBalanceUSD:     cashBalance,
			InvestedBalanceUSD: invested,
			ChangePct:          changePct,
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
