package main

import (
	"context"
	"fmt"
	"time"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"github.com/orange-cat-investments/oci/pkg/reporting"
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

func (s *CoreInvestGRPCServer) ExecuteGDPRErasure(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
	cert, err := s.svc.ExecuteGDPRErasure(ctx, in.CustomerID, in.Email)
	if err != nil {
		return nil, err
	}
	pdfBytes, _ := reporting.GenerateGDPRCertificatePDF(cert)
	return &coreinvestv1.ExecuteGDPRErasureResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        cert.CertificateID,
			CustomerID:           in.CustomerID,
			CustomerIDHash:       cert.CustomerIDHash,
			CustomerEmailHash:    cert.CustomerEmailHash,
			ErasedAt:             cert.ErasureTimestamp.Format(time.RFC3339),
			LegalBasis:           cert.LegalBasis,
			SupervisoryAuthority: cert.SupervisoryAuthority,
			AnonymizedTables:     cert.TablesScrubbed,
			SessionsPurgedCount:  int32(cert.SessionsPurgedCount),
			RecordsScrubbedCount: int32(cert.RecordsScrubbedCount),
			ComplianceHash:       cert.DigitalSignature,
			ComplianceStatement:  cert.ComplianceStatement,
			Status:               "erased",
		},
		PDFBytes: pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) GetGDPRErasureCertificate(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error) {
	rec, err := s.svc.GetGDPRErasureCertificate(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	cert := &reporting.GDPRDeletionCertificate{
		CertificateID:        rec.CertificateID,
		CustomerIDHash:       rec.CustomerIDHash,
		CustomerEmailHash:    rec.CustomerEmailHash,
		ErasureTimestamp:     rec.CreatedAt,
		LegalBasis:           rec.LegalBasis,
		SupervisoryAuthority: rec.SupervisoryAuthority,
		TablesScrubbed:       rec.TablesScrubbed,
		SessionsPurgedCount:  rec.SessionsPurgedCount,
		RecordsScrubbedCount: rec.RecordsScrubbedCount,
		DigitalSignature:     rec.DigitalSignature,
		ComplianceStatement:  rec.ComplianceStatement,
	}
	pdfBytes, _ := reporting.GenerateGDPRCertificatePDF(cert)
	return &coreinvestv1.GetGDPRErasureCertificateResponse{
		Certificate: &coreinvestv1.GDPRErasureCertificate{
			CertificateID:        rec.CertificateID,
			CustomerIDHash:       rec.CustomerIDHash,
			CustomerEmailHash:    rec.CustomerEmailHash,
			ErasedAt:             rec.CreatedAt.Format(time.RFC3339),
			LegalBasis:           rec.LegalBasis,
			SupervisoryAuthority: rec.SupervisoryAuthority,
			AnonymizedTables:     rec.TablesScrubbed,
			SessionsPurgedCount:  int32(rec.SessionsPurgedCount),
			RecordsScrubbedCount: int32(rec.RecordsScrubbedCount),
			ComplianceHash:       rec.DigitalSignature,
			ComplianceStatement:  rec.ComplianceStatement,
			Status:               "erased",
		},
		PDFBytes: pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) CreateAPIKey(ctx context.Context, in *coreinvestv1.CreateAPIKeyRequest) (*coreinvestv1.CreateAPIKeyResponse, error) {
	key, err := s.svc.CreateCustomerAPIKey(ctx, &core_invest_repo.CustomerAPIKey{
		CustomerID:  in.CustomerID,
		Name:        in.Label,
		Permissions: in.Scopes,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.CreateAPIKeyResponse{
		APIKey: &coreinvestv1.CustomerAPIKey{
			KeyID:      key.KeyID,
			CustomerID: key.CustomerID,
			Label:      key.Name,
			KeyPrefix:  key.KeyPrefix,
			Scopes:     key.Permissions,
			Status:     key.Status,
			CreatedAt:  key.CreatedAt.Format(time.RFC3339),
		},
		RawSecret: key.APIKey,
	}, nil
}

func (s *CoreInvestGRPCServer) ListAPIKeys(ctx context.Context, in *coreinvestv1.ListAPIKeysRequest) (*coreinvestv1.ListAPIKeysResponse, error) {
	keys, err := s.svc.ListCustomerAPIKeys(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CustomerAPIKey
	for _, k := range keys {
		out = append(out, &coreinvestv1.CustomerAPIKey{
			KeyID:      k.KeyID,
			CustomerID: k.CustomerID,
			Label:      k.Name,
			KeyPrefix:  k.KeyPrefix,
			Scopes:     k.Permissions,
			Status:     k.Status,
			CreatedAt:  k.CreatedAt.Format(time.RFC3339),
		})
	}
	return &coreinvestv1.ListAPIKeysResponse{APIKeys: out}, nil
}

func (s *CoreInvestGRPCServer) RevokeAPIKey(ctx context.Context, in *coreinvestv1.RevokeAPIKeyRequest) (*coreinvestv1.RevokeAPIKeyResponse, error) {
	if err := s.svc.RevokeCustomerAPIKey(ctx, in.CustomerID, in.KeyID); err != nil {
		return nil, err
	}
	return &coreinvestv1.RevokeAPIKeyResponse{Status: "revoked"}, nil
}

func (s *CoreInvestGRPCServer) CreateWebhook(ctx context.Context, in *coreinvestv1.CreateWebhookRequest) (*coreinvestv1.CreateWebhookResponse, error) {
	wh, err := s.svc.CreateWebhookSubscription(ctx, &core_invest_repo.WebhookSubscription{
		CustomerID: in.CustomerID,
		TargetURL:  in.TargetURL,
		Events:     in.EventTypes,
	})
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.CreateWebhookResponse{
		Webhook: &coreinvestv1.CustomerWebhook{
			WebhookID:  wh.SubscriptionID,
			CustomerID: wh.CustomerID,
			TargetURL:  wh.TargetURL,
			EventTypes: wh.Events,
			Secret:     wh.Secret,
			Status:     wh.Status,
			CreatedAt:  wh.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *CoreInvestGRPCServer) ListWebhooks(ctx context.Context, in *coreinvestv1.ListWebhooksRequest) (*coreinvestv1.ListWebhooksResponse, error) {
	whs, err := s.svc.ListWebhookSubscriptions(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CustomerWebhook
	for _, w := range whs {
		out = append(out, &coreinvestv1.CustomerWebhook{
			WebhookID:  w.SubscriptionID,
			CustomerID: w.CustomerID,
			TargetURL:  w.TargetURL,
			EventTypes: w.Events,
			Secret:     w.Secret,
			Status:     w.Status,
			CreatedAt:  w.CreatedAt.Format(time.RFC3339),
		})
	}
	return &coreinvestv1.ListWebhooksResponse{Webhooks: out}, nil
}

func (s *CoreInvestGRPCServer) DeleteWebhook(ctx context.Context, in *coreinvestv1.DeleteWebhookRequest) (*coreinvestv1.DeleteWebhookResponse, error) {
	if err := s.svc.DeleteWebhookSubscription(ctx, in.CustomerID, in.WebhookID); err != nil {
		return nil, err
	}
	return &coreinvestv1.DeleteWebhookResponse{Status: "deleted"}, nil
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

func (s *CoreInvestGRPCServer) RegisterCameraStream(ctx context.Context, in *coreinvestv1.RegisterCameraStreamRequest) (*coreinvestv1.RegisterCameraStreamResponse, error) {
	cs, err := s.svc.RegisterCameraStream(ctx, in.AssetID, in.StreamURL, in.Protocol)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.RegisterCameraStreamResponse{Stream: toProtoCameraStream(cs)}, nil
}

func (s *CoreInvestGRPCServer) GetCameraStream(ctx context.Context, in *coreinvestv1.GetCameraStreamRequest) (*coreinvestv1.GetCameraStreamResponse, error) {
	cs, err := s.svc.GetCameraStream(ctx, in.StreamID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetCameraStreamResponse{Stream: toProtoCameraStream(cs)}, nil
}

func (s *CoreInvestGRPCServer) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	streams, err := s.svc.ListCameraStreams(ctx, 50, 0)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CameraStream
	for _, cs := range streams {
		if in.Status != "" && cs.Status != in.Status {
			continue
		}
		out = append(out, toProtoCameraStream(cs))
	}
	return &coreinvestv1.ListCameraStreamsResponse{Streams: out}, nil
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

func (s *CoreInvestGRPCServer) StartBacktestRun(ctx context.Context, in *coreinvestv1.StartBacktestRunRequest) (*coreinvestv1.StartBacktestRunResponse, error) {
	start, err := time.Parse("2006-01-02", in.StartDate)
	if err != nil {
		start = time.Now().UTC().AddDate(-1, 0, 0)
	}
	end, err := time.Parse("2006-01-02", in.EndDate)
	if err != nil {
		end = time.Now().UTC()
	}
	run, err := s.svc.StartBacktestRun(ctx, in.StrategyName, start, end, fmt.Sprintf(`{"initial_capital":%f}`, in.InitialCapital))
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.StartBacktestRunResponse{Run: toProtoBacktestRun(run)}, nil
}

func (s *CoreInvestGRPCServer) GetBacktestRun(ctx context.Context, in *coreinvestv1.GetBacktestRunRequest) (*coreinvestv1.GetBacktestRunResponse, error) {
	run, err := s.svc.GetBacktestRun(ctx, in.RunID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetBacktestRunResponse{Run: toProtoBacktestRun(run)}, nil
}

func (s *CoreInvestGRPCServer) ListBacktestRuns(ctx context.Context, in *coreinvestv1.ListBacktestRunsRequest) (*coreinvestv1.ListBacktestRunsResponse, error) {
	runs, err := s.svc.ListBacktestRuns(ctx, 50, 0)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.BacktestRun
	for _, r := range runs {
		out = append(out, toProtoBacktestRun(r))
	}
	return &coreinvestv1.ListBacktestRunsResponse{Runs: out}, nil
}

func (s *CoreInvestGRPCServer) ListMLModels(ctx context.Context, in *coreinvestv1.ListMLModelsRequest) (*coreinvestv1.ListMLModelsResponse, error) {
	models, err := s.svc.ListMLModels(ctx)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.MLModelInfo
	for _, m := range models {
		id, _ := m["model_id"].(string)
		name, _ := m["name"].(string)
		ver, _ := m["version"].(string)
		fw, _ := m["framework"].(string)
		st, _ := m["status"].(string)
		acc, _ := m["accuracy"].(float64)
		out = append(out, &coreinvestv1.MLModelInfo{
			ModelID:   id,
			Name:      name,
			Version:   ver,
			Framework: fw,
			Status:    st,
			Accuracy:  acc,
		})
	}
	return &coreinvestv1.ListMLModelsResponse{Models: out}, nil
}

func (s *CoreInvestGRPCServer) LogModelDrift(ctx context.Context, in *coreinvestv1.LogModelDriftRequest) (*coreinvestv1.LogModelDriftResponse, error) {
	res, err := s.svc.LogModelDrift(ctx, map[string]interface{}{
		"model_id":       in.ModelID,
		"psi_score":      in.PSIScore,
		"accuracy_delta": in.AccuracyDelta,
		"notes":          in.Notes,
	})
	if err != nil {
		return nil, err
	}
	driftID, _ := res["drift_id"].(string)
	st, _ := res["status"].(string)
	return &coreinvestv1.LogModelDriftResponse{
		DriftID:  driftID,
		ModelID:  in.ModelID,
		Status:   st,
		LoggedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
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

func (s *CoreInvestGRPCServer) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	stmts, err := s.svc.ListStatements(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CustomerStatement
	for _, st := range stmts {
		if in.StatementType != "" && st.StatementType != in.StatementType {
			continue
		}
		out = append(out, toProtoStatement(st))
	}
	return &coreinvestv1.ListStatementsResponse{Statements: out}, nil
}

func (s *CoreInvestGRPCServer) GetStatement(ctx context.Context, in *coreinvestv1.GetStatementRequest) (*coreinvestv1.GetStatementResponse, error) {
	st, err := s.svc.GetStatement(ctx, in.StatementID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetStatementResponse{Statement: toProtoStatement(st)}, nil
}

func (s *CoreInvestGRPCServer) GenerateMonthlyStatement(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
	st, pdfBytes, err := s.svc.GenerateMonthlyStatement(ctx, in.CustomerID, in.PortfolioID, int(in.Year), int(in.Month))
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GenerateMonthlyStatementResponse{
		Statement: toProtoStatement(st),
		PDFBytes:  pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) GenerateForm1099B(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
	st, pdfBytes, err := s.svc.GenerateForm1099B(ctx, in.CustomerID, in.PortfolioID, int(in.TaxYear))
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GenerateForm1099BResponse{
		Statement: toProtoStatement(st),
		PDFBytes:  pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) GetStatementDocument(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
	st, pdfBytes, err := s.svc.GetStatementDocument(ctx, in.CustomerID, in.StatementID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetStatementDocumentResponse{
		Statement: toProtoStatement(st),
		PDFBytes:  pdfBytes,
	}, nil
}

func (s *CoreInvestGRPCServer) StreamPortfolioValuation(req *coreinvestv1.StreamPortfolioValuationRequest, stream coreinvestv1.CoreInvestService_StreamPortfolioValuationServer) error {
	if req.PortfolioID == "" {
		return status.Error(codes.InvalidArgument, "portfolio_id is required")
	}

	p, err := s.svc.GetPortfolio(stream.Context(), req.PortfolioID)
	if err != nil || p == nil {
		return status.Errorf(codes.NotFound, "portfolio %s not found: %v", req.PortfolioID, err)
	}

	count := req.UpdateCount
	if count <= 0 {
		count = 5
	}

	invested := p.TotalBalanceUSD - p.CashBalanceUSD
	changePct := 0.0
	if p.TotalBalanceUSD > 0 && invested > 0 {
		changePct = (invested / p.TotalBalanceUSD) * 100.0
	}

	for i := 0; i < int(count); i++ {
		if i > 0 {
			if latest, err := s.svc.GetPortfolio(stream.Context(), req.PortfolioID); err == nil && latest != nil {
				p = latest
				invested = p.TotalBalanceUSD - p.CashBalanceUSD
				if p.TotalBalanceUSD > 0 && invested > 0 {
					changePct = (invested / p.TotalBalanceUSD) * 100.0
				}
			}
		}

		err := stream.Send(&coreinvestv1.PortfolioValuationEvent{
			PortfolioID:        p.PortfolioID,
			TotalBalanceUSD:    p.TotalBalanceUSD,
			CashBalanceUSD:     p.CashBalanceUSD,
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
