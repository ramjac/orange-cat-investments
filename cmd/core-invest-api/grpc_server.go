package main

import (
	"context"
	"time"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoreInvestGRPCServer struct {
	coreinvestv1.UnimplementedCoreInvestServiceServer
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

func toProtoHolding(h *core_invest_repo.PortfolioHolding) *coreinvestv1.PortfolioHolding {
	if h == nil {
		return nil
	}
	return &coreinvestv1.PortfolioHolding{
		HoldingID:       h.HoldingID,
		PortfolioID:     h.PortfolioID,
		Symbol:          h.Symbol,
		Quantity:        h.Quantity,
		AvgPriceUSD:     h.AvgPriceUSD,
		CurrentPriceUSD: h.CurrentPriceUSD,
		MarketValueUSD:  h.MarketValueUSD,
	}
}

func toProtoStatement(st *core_invest_repo.InvestmentStatement) *coreinvestv1.CustomerStatement {
	if st == nil {
		return nil
	}
	gen := ""
	if !st.CreatedAt.IsZero() {
		gen = st.CreatedAt.Format(time.RFC3339)
	}
	return &coreinvestv1.CustomerStatement{
		StatementID:         st.StatementID,
		CustomerID:          st.CustomerID,
		PortfolioID:         st.PortfolioID,
		StatementType:       st.StatementType,
		Period:              st.Period,
		EndingBalanceUSD:    st.TotalProceedsUSD,
		RealizedGainLossUSD: st.NetGainUSD,
		Status:              st.Status,
		GeneratedAt:         gen,
	}
}

func toProtoCameraStream(cs *core_invest_repo.CameraStream) *coreinvestv1.CameraStream {
	if cs == nil {
		return nil
	}
	created := ""
	if !cs.CreatedAt.IsZero() {
		created = cs.CreatedAt.Format(time.RFC3339)
	}
	return &coreinvestv1.CameraStream{
		StreamID:  cs.StreamID,
		AssetID:   cs.CameraAssetID,
		StreamURL: cs.StreamURL,
		Protocol:  cs.Protocol,
		Status:    cs.Status,
		CreatedAt: created,
	}
}

func toProtoBacktestRun(run *core_invest_repo.BacktestRun) *coreinvestv1.BacktestRun {
	if run == nil {
		return nil
	}
	created := ""
	if !run.CreatedAt.IsZero() {
		created = run.CreatedAt.Format(time.RFC3339)
	}
	return &coreinvestv1.BacktestRun{
		RunID:        run.BacktestID,
		StrategyName: run.StrategyID,
		StartDate:    run.StartDate.Format("2006-01-02"),
		EndDate:      run.EndDate.Format("2006-01-02"),
		Status:       run.Status,
		CreatedAt:    created,
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
				holdings = append(holdings, toProtoHolding(h))
			}
		}
	}

	return &coreinvestv1.GetPortfolioResponse{
		Portfolio: toProtoPortfolio(p),
		Holdings:  holdings,
	}, nil
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

func (s *CoreInvestGRPCServer) CreatePortfolio(ctx context.Context, in *coreinvestv1.CreatePortfolioRequest) (*coreinvestv1.CreatePortfolioResponse, error) {
	if in.Portfolio == nil {
		return nil, status.Error(codes.InvalidArgument, "portfolio is required")
	}
	created, err := s.svc.CreatePortfolio(ctx, &core_invest_repo.Portfolio{
		PortfolioID:     in.Portfolio.PortfolioID,
		CustomerID:      in.Portfolio.CustomerID,
		AccountName:     in.Portfolio.AccountName,
		TotalBalanceUSD: in.Portfolio.TotalBalanceUSD,
		CashBalanceUSD:  in.Portfolio.CashBalanceUSD,
		Status:          in.Portfolio.Status,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.CreatePortfolioResponse{Portfolio: toProtoPortfolio(created)}, nil
}

func (s *CoreInvestGRPCServer) UpdatePortfolio(ctx context.Context, in *coreinvestv1.UpdatePortfolioRequest) (*coreinvestv1.UpdatePortfolioResponse, error) {
	if in.Portfolio == nil {
		return nil, status.Error(codes.InvalidArgument, "portfolio is required")
	}
	if s.repo == nil {
		return nil, status.Error(codes.Unavailable, "repository not configured")
	}
	updated, err := s.repo.UpdatePortfolio(ctx, &core_invest_repo.Portfolio{
		PortfolioID:     in.Portfolio.PortfolioID,
		CustomerID:      in.Portfolio.CustomerID,
		AccountName:     in.Portfolio.AccountName,
		TotalBalanceUSD: in.Portfolio.TotalBalanceUSD,
		CashBalanceUSD:  in.Portfolio.CashBalanceUSD,
		Status:          in.Portfolio.Status,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.UpdatePortfolioResponse{Portfolio: toProtoPortfolio(updated)}, nil
}

func (s *CoreInvestGRPCServer) ListHoldings(ctx context.Context, in *coreinvestv1.ListHoldingsRequest) (*coreinvestv1.ListHoldingsResponse, error) {
	var holdings []*coreinvestv1.PortfolioHolding
	if s.repo != nil {
		repoHoldings, err := s.repo.GetHoldingsByPortfolioID(ctx, in.PortfolioID)
		if err != nil {
			return nil, err
		}
		for _, h := range repoHoldings {
			holdings = append(holdings, toProtoHolding(h))
		}
	}
	return &coreinvestv1.ListHoldingsResponse{Holdings: holdings}, nil
}

func (s *CoreInvestGRPCServer) UpsertHolding(ctx context.Context, in *coreinvestv1.UpsertHoldingRequest) (*coreinvestv1.UpsertHoldingResponse, error) {
	if in.Holding == nil {
		return nil, status.Error(codes.InvalidArgument, "holding is required")
	}
	if s.repo == nil {
		return nil, status.Error(codes.Unavailable, "repository not configured")
	}
	upserted, err := s.repo.UpsertHolding(ctx, &core_invest_repo.PortfolioHolding{
		HoldingID:       in.Holding.HoldingID,
		PortfolioID:     in.Holding.PortfolioID,
		Symbol:          in.Holding.Symbol,
		Quantity:        in.Holding.Quantity,
		AvgPriceUSD:     in.Holding.AvgPriceUSD,
		CurrentPriceUSD: in.Holding.CurrentPriceUSD,
		MarketValueUSD:  in.Holding.MarketValueUSD,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.UpsertHoldingResponse{Holding: toProtoHolding(upserted)}, nil
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

func (s *CoreInvestGRPCServer) RegisterCustomer(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
	res, err := s.svc.RegisterCustomer(ctx, core_invest_svc.CustomerRegistrationRequest{
		FullName:        in.Name,
		Email:           in.Email,
		Password:        in.Password,
		BankPublicToken: in.BankPublicToken,
		TargetFund:      in.TargetFund,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.RegisterCustomerResponse{
		CustomerID:      res.CustomerID,
		Email:           res.Email,
		Name:            res.CustomerName,
		ZitadelUserID:   res.ZitadelUserID,
		PortfolioID:     res.PortfolioID,
		LinkedBankAccID: res.LinkedBankAccID,
		CashBalance:     res.StartingBalance,
		Status:          "active",
		Message:         res.Message,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *CoreInvestGRPCServer) CloseAndLiquidateAccount(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
	res, err := s.svc.CloseAndLiquidateAccount(ctx, core_invest_svc.CloseAccountRequest{
		CustomerID:           in.CustomerID,
		PortfolioID:          in.PortfolioID,
		DestinationAccountID: in.DestinationAccountID,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.CloseAndLiquidateAccountResponse{
		CustomerID:            res.CustomerID,
		PortfolioID:           res.PortfolioID,
		LiquidatedOrders:      int32(res.LiquidatedPositions),
		LiquidatedProceeds:    res.GrossProceedsUSD,
		FinalCashWithdrawn:    res.TransferredToBankUSD,
		ACHWithdrawalTransfer: res.TransferID,
		DestinationAccountID:  res.DestinationAccountID,
		AccountStatus:         res.AccountStatus,
		Message:               res.Message,
		ClosedAt:              res.SettlementDate.Format(time.RFC3339),
	}, nil
}
