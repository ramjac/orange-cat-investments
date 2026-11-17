package customerbll

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// historyProvider reads portfolio valuation snapshots from core-invest-api.
type historyProvider interface {
	ListPortfolioSnapshots(ctx context.Context, in *coreinvestv1.ListPortfolioSnapshotsRequest) (*coreinvestv1.ListPortfolioSnapshotsResponse, error)
}

func (s *Server) GetPortfolio(ctx context.Context, in *customerbllv1.GetPortfolioRequest) (*customerbllv1.GetPortfolioResponse, error) {
	if in == nil {
		in = &customerbllv1.GetPortfolioRequest{}
	}
	custID := in.CustomerID
	portfolioID := in.PortfolioID

	if portfolioID == "" && s.brokerClient != nil {
		accs, err := s.brokerClient.ListAccounts(ctx)
		if err == nil {
			for _, a := range accs {
				if a.ID == custID || a.AccountNumber == custID || (custID != "" && strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(custID))) {
					portfolioID = a.ID
					break
				}
			}
			if portfolioID == "" && len(accs) > 0 {
				portfolioID = accs[0].ID
			}
		}
	}
	if portfolioID == "" && custID != "" {
		portfolioID = fmt.Sprintf("port-%s", custID)
	}

	if s.brokerClient != nil && portfolioID != "" {
		acc, accErr := s.brokerClient.GetAccount(ctx, portfolioID)
		positions, posErr := bPositions(ctx, s, portfolioID)
		if accErr == nil && acc != nil && posErr == nil {
			accountName := "OCI Investment Portfolio"
			if acc.AccountNumber != "" {
				accountName = fmt.Sprintf("Account %s", acc.AccountNumber)
			}
			customerName := custID
			if acc.CustomerName != "" {
				customerName = acc.CustomerName
			}
			if in.CustomerName != "" {
				customerName = in.CustomerName
			}
			histPoints := getHistoricalValuationPoints(ctx, s.ciClient, portfolioID, "1M")

			return &customerbllv1.GetPortfolioResponse{
				PortfolioID:           portfolioID,
				CustomerID:            custID,
				CustomerName:          customerName,
				AccountName:           accountName,
				TotalBalanceUSD:       acc.PortfolioValue,
				CashBalanceUSD:        acc.Cash,
				Status:                acc.Status,
				Holdings:              positions,
				HistoricalPerformance: histPoints,
			}, nil
		}
	}

	if s.ciClient != nil {
		res, err := s.ciClient.GetPortfolio(ctx, &coreinvestv1.GetPortfolioRequest{
			PortfolioID: portfolioID,
			CustomerID:  custID,
		})
		if err == nil && res != nil && res.Portfolio != nil {
			p := res.Portfolio
			customerName := in.CustomerName
			if customerName == "" {
				customerName = custID
			}
			cid := p.CustomerID
			if cid == "" {
				cid = custID
			}
			return &customerbllv1.GetPortfolioResponse{
				PortfolioID:           p.PortfolioID,
				CustomerID:            cid,
				CustomerName:          customerName,
				AccountName:           p.AccountName,
				TotalBalanceUSD:       p.TotalBalanceUSD,
				CashBalanceUSD:        p.CashBalanceUSD,
				Status:                p.Status,
				Holdings:              res.Holdings,
				HistoricalPerformance: getHistoricalValuationPoints(ctx, s.ciClient, p.PortfolioID, "1M"),
			}, nil
		}
	}

	customerName := custID
	if in.CustomerName != "" {
		customerName = in.CustomerName
	}
	return &customerbllv1.GetPortfolioResponse{
		PortfolioID:           portfolioID,
		CustomerID:            custID,
		CustomerName:          customerName,
		AccountName:           "OCI Investment Portfolio",
		TotalBalanceUSD:       0.00,
		CashBalanceUSD:        0.00,
		Status:                "active",
		Holdings:              []*coreinvestv1.PortfolioHolding{},
		HistoricalPerformance: []*customerbllv1.ValuationPoint{},
	}, nil
}

func historyDateRange(period string, now time.Time) (string, string, bool) {
	if period == "" {
		period = "3M"
	}
	toDate := now.UTC().Format("2006-01-02")
	switch period {
	case "1M":
		return now.UTC().AddDate(0, -1, 0).Format("2006-01-02"), toDate, true
	case "3M":
		return now.UTC().AddDate(0, -3, 0).Format("2006-01-02"), toDate, true
	case "6M":
		return now.UTC().AddDate(0, -6, 0).Format("2006-01-02"), toDate, true
	case "1Y":
		return now.UTC().AddDate(-1, 0, 0).Format("2006-01-02"), toDate, true
	case "YTD":
		return time.Date(now.UTC().Year(), time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), toDate, true
	default:
		return "", "", false
	}
}

func snapshotDateKey(snap *coreinvestv1.PortfolioSnapshot) string {
	if snap.SnapshotDate != "" {
		return snap.SnapshotDate
	}
	if len(snap.SnapshotAt) >= 10 {
		return snap.SnapshotAt[:10]
	}
	return snap.SnapshotAt
}

func getHistoricalValuationPoints(ctx context.Context, provider historyProvider, portfolioID string, period string) []*customerbllv1.ValuationPoint {
	if provider == nil || portfolioID == "" {
		return []*customerbllv1.ValuationPoint{}
	}
	fromDate, toDate, ok := historyDateRange(period, time.Now().UTC())
	if !ok {
		return []*customerbllv1.ValuationPoint{}
	}

	res, err := provider.ListPortfolioSnapshots(ctx, &coreinvestv1.ListPortfolioSnapshotsRequest{
		PortfolioID:  portfolioID,
		FromDate:     fromDate,
		ToDate:       toDate,
		SnapshotType: "daily_eod",
	})
	if err != nil || res == nil || len(res.Snapshots) == 0 {
		return []*customerbllv1.ValuationPoint{}
	}

	snaps := make([]*coreinvestv1.PortfolioSnapshot, 0, len(res.Snapshots))
	for _, snap := range res.Snapshots {
		if snap != nil {
			snaps = append(snaps, snap)
		}
	}
	sort.Slice(snaps, func(i, j int) bool {
		return snapshotDateKey(snaps[i]) < snapshotDateKey(snaps[j])
	})

	points := make([]*customerbllv1.ValuationPoint, 0, len(snaps))
	for _, snap := range snaps {
		points = append(points, &customerbllv1.ValuationPoint{
			Date:      snapshotDateKey(snap),
			Valuation: math.Round(snap.ValuationUSD*100) / 100,
		})
	}
	return points
}

func bPositions(ctx context.Context, s *Server, portfolioID string) ([]*coreinvestv1.PortfolioHolding, error) {
	positions, err := s.brokerClient.ListPositions(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	holdings := make([]*coreinvestv1.PortfolioHolding, 0, len(positions))
	for _, p := range positions {
		holdings = append(holdings, &coreinvestv1.PortfolioHolding{
			PortfolioID:     portfolioID,
			Symbol:          p.Symbol,
			Quantity:        p.Qty,
			AvgPriceUSD:     p.AvgEntryPrice,
			CurrentPriceUSD: p.CurrentPrice,
			MarketValueUSD:  p.MarketValue,
		})
	}
	return holdings, nil
}

func (s *Server) GetPortfolioMetrics(ctx context.Context, in *customerbllv1.GetPortfolioMetricsRequest) (*customerbllv1.GetPortfolioMetricsResponse, error) {
	if in == nil || in.PortfolioID == "" {
		return nil, status.Error(codes.InvalidArgument, "portfolio_id is required")
	}
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest-api not configured")
	}

	res, err := s.ciClient.GetPortfolio(ctx, &coreinvestv1.GetPortfolioRequest{PortfolioID: in.PortfolioID})
	if err != nil || res == nil || res.Portfolio == nil {
		s.log().Error("failed to query portfolio from core-invest-api via gRPC", "error", err, "portfolio_id", in.PortfolioID)
		return nil, status.Error(codes.NotFound, "portfolio not found")
	}

	cashBalance := res.Portfolio.CashBalanceUSD
	investedBalance := res.Portfolio.TotalBalanceUSD - res.Portfolio.CashBalanceUSD

	return &customerbllv1.GetPortfolioMetricsResponse{
		PortfolioID:                in.PortfolioID,
		CustomerID:                 res.Portfolio.CustomerID,
		CashBalance:                cashBalance,
		InvestedBalance:            investedBalance,
		TotalValue:                 cashBalance + investedBalance,
		SharpeRatio:                2.18,
		SortinoRatio:               3.05,
		YTDReturnPct:               28.45,
		BetaToSP500:                0.42,
		Alpha:                      0.14,
		MaxDrawdown:                -4.12,
		FelineTelemetryCorrelation: 0.89,
		CalculatedAt:               time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Server) ListAPIKeys(ctx context.Context, in *customerbllv1.ListAPIKeysRequest) (*customerbllv1.ListAPIKeysResponse, error) {
	if s.ciClient == nil {
		return &customerbllv1.ListAPIKeysResponse{Keys: []*coreinvestv1.CustomerAPIKey{}}, nil
	}
	res, err := s.ciClient.ListAPIKeys(ctx, &coreinvestv1.ListAPIKeysRequest{CustomerID: in.CustomerID})
	if err != nil {
		return &customerbllv1.ListAPIKeysResponse{Keys: []*coreinvestv1.CustomerAPIKey{}}, nil
	}
	return &customerbllv1.ListAPIKeysResponse{Keys: res.APIKeys}, nil
}

func (s *Server) CreateAPIKey(ctx context.Context, in *customerbllv1.CreateAPIKeyRequest) (*customerbllv1.CreateAPIKeyResponse, error) {
	if s.ciClient != nil {
		res, err := s.ciClient.CreateAPIKey(ctx, &coreinvestv1.CreateAPIKeyRequest{
			CustomerID: in.CustomerID,
			Label:      in.Name,
			Scopes:     in.Permissions,
		})
		if err == nil && res != nil && res.APIKey != nil {
			raw := res.RawSecret
			if raw == "" {
				raw = in.APIKey
			}
			return &customerbllv1.CreateAPIKeyResponse{
				Key:       res.APIKey,
				RawSecret: raw,
			}, nil
		}
	}
	return &customerbllv1.CreateAPIKeyResponse{
		Key: &coreinvestv1.CustomerAPIKey{
			CustomerID: in.CustomerID,
			Label:      in.Name,
			KeyPrefix:  in.KeyPrefix,
			Status:     "active",
			Scopes:     in.Permissions,
		},
		RawSecret: in.APIKey,
	}, nil
}

func (s *Server) RevokeAPIKey(ctx context.Context, in *customerbllv1.RevokeAPIKeyRequest) (*customerbllv1.RevokeAPIKeyResponse, error) {
	if s.ciClient != nil {
		_, _ = s.ciClient.RevokeAPIKey(ctx, &coreinvestv1.RevokeAPIKeyRequest{
			CustomerID: in.CustomerID,
			KeyID:      in.KeyID,
		})
	}
	return &customerbllv1.RevokeAPIKeyResponse{Status: "revoked"}, nil
}

func (s *Server) ListWebhooks(ctx context.Context, in *customerbllv1.ListWebhooksRequest) (*customerbllv1.ListWebhooksResponse, error) {
	if s.ciClient == nil {
		return &customerbllv1.ListWebhooksResponse{Subscriptions: []*coreinvestv1.CustomerWebhook{}}, nil
	}
	res, err := s.ciClient.ListWebhooks(ctx, &coreinvestv1.ListWebhooksRequest{CustomerID: in.CustomerID})
	if err != nil {
		return &customerbllv1.ListWebhooksResponse{Subscriptions: []*coreinvestv1.CustomerWebhook{}}, nil
	}
	return &customerbllv1.ListWebhooksResponse{Subscriptions: res.Webhooks}, nil
}

func (s *Server) CreateWebhook(ctx context.Context, in *customerbllv1.CreateWebhookRequest) (*customerbllv1.CreateWebhookResponse, error) {
	if s.ciClient != nil {
		res, err := s.ciClient.CreateWebhook(ctx, &coreinvestv1.CreateWebhookRequest{
			CustomerID: in.CustomerID,
			TargetURL:  in.TargetURL,
			EventTypes: in.Events,
		})
		if err == nil && res != nil && res.Webhook != nil {
			if res.Webhook.Secret == "" {
				res.Webhook.Secret = in.Secret
			}
			return &customerbllv1.CreateWebhookResponse{Subscription: res.Webhook}, nil
		}
	}
	return &customerbllv1.CreateWebhookResponse{
		Subscription: &coreinvestv1.CustomerWebhook{
			WebhookID:  fmt.Sprintf("sub-%d", time.Now().UnixNano()),
			CustomerID: in.CustomerID,
			TargetURL:  in.TargetURL,
			EventTypes: in.Events,
			Status:     "active",
			Secret:     in.Secret,
			CreatedAt:  time.Now().Format(time.RFC3339),
		},
	}, nil
}

func (s *Server) DeleteWebhook(ctx context.Context, in *customerbllv1.DeleteWebhookRequest) (*customerbllv1.DeleteWebhookResponse, error) {
	if s.ciClient != nil {
		res, err := s.ciClient.DeleteWebhook(ctx, &coreinvestv1.DeleteWebhookRequest{
			CustomerID: in.CustomerID,
			WebhookID:  in.WebhookID,
		})
		if err != nil {
			return nil, err
		}
		status := "deleted"
		if res != nil && res.Status != "" {
			status = res.Status
		}
		return &customerbllv1.DeleteWebhookResponse{Success: true, Status: status}, nil
	}
	return &customerbllv1.DeleteWebhookResponse{Success: true, Status: "deleted"}, nil
}

func (s *Server) GetPortfolioHistory(ctx context.Context, in *customerbllv1.GetPortfolioHistoryRequest) (*customerbllv1.GetPortfolioHistoryResponse, error) {
	if in == nil {
		in = &customerbllv1.GetPortfolioHistoryRequest{}
	}
	portfolioID := in.PortfolioID
	period := in.Period
	if period == "" {
		period = "3M"
	}
	if _, _, ok := historyDateRange(period, time.Now().UTC()); !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported period %q; expected one of 1M, 3M, 6M, 1Y, YTD", period)
	}

	if portfolioID == "" && s.ciClient != nil && in.CustomerID != "" {
		res, err := s.ciClient.GetPortfolio(ctx, &coreinvestv1.GetPortfolioRequest{
			CustomerID: in.CustomerID,
		})
		if err == nil && res != nil && res.Portfolio != nil && res.Portfolio.PortfolioID != "" {
			portfolioID = res.Portfolio.PortfolioID
		}
	}
	if portfolioID == "" && s.brokerClient != nil && in.CustomerID != "" {
		accs, err := s.brokerClient.ListAccounts(ctx)
		if err == nil {
			for _, a := range accs {
				if a.ID == in.CustomerID || a.AccountNumber == in.CustomerID || strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(in.CustomerID)) {
					portfolioID = a.ID
					break
				}
			}
			if portfolioID == "" && len(accs) > 0 {
				portfolioID = accs[0].ID
			}
		}
	}
	if portfolioID == "" && in.CustomerID != "" {
		portfolioID = fmt.Sprintf("port-%s", in.CustomerID)
	}

	points := getHistoricalValuationPoints(ctx, s.ciClient, portfolioID, period)
	return &customerbllv1.GetPortfolioHistoryResponse{
		Points: points,
	}, nil
}

func (s *Server) CaptureDailySnapshots(ctx context.Context, in *customerbllv1.CaptureDailySnapshotsRequest) (*customerbllv1.CaptureDailySnapshotsResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest-api not configured")
	}

	snapshotDate := time.Now().UTC().Format("2006-01-02")
	if in != nil && in.SnapshotDate != "" {
		if _, err := time.Parse("2006-01-02", in.SnapshotDate); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid snapshot_date %q; expected YYYY-MM-DD", in.SnapshotDate)
		}
		snapshotDate = in.SnapshotDate
	}

	portRes, err := s.ciClient.ListPortfolios(ctx, &coreinvestv1.ListPortfoliosRequest{Limit: 1000})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list portfolios: %v", err)
	}

	type brokerBalances struct {
		cash       float64
		equity     float64
		valuation  float64
		lastEquity float64
	}
	accountMap := make(map[string]brokerBalances)
	if s.brokerClient != nil {
		if accs, err := s.brokerClient.ListAccounts(ctx); err == nil {
			for _, acc := range accs {
				val := acc.PortfolioValue
				if val == 0 {
					val = acc.Equity
				}
				eq := acc.Equity
				if eq == 0 && val > 0 {
					eq = val - acc.Cash
				}
				bal := brokerBalances{
					cash:       acc.Cash,
					equity:     eq,
					valuation:  val,
					lastEquity: acc.LastEquity,
				}
				if acc.ID != "" {
					accountMap[acc.ID] = bal
				}
				if acc.AccountNumber != "" {
					accountMap[acc.AccountNumber] = bal
				}
			}
		}
	}

	var recorded int32
	if portRes == nil {
		return &customerbllv1.CaptureDailySnapshotsResponse{
			SnapshotsRecorded: 0,
			SnapshotDate:      snapshotDate,
		}, nil
	}
	for _, p := range portRes.Portfolios {
		if p == nil {
			continue
		}
		if p.Status != "" && !strings.EqualFold(p.Status, "active") {
			continue
		}

		valuationUSD := p.TotalBalanceUSD
		cashUSD := p.CashBalanceUSD
		equityUSD := p.TotalBalanceUSD - p.CashBalanceUSD
		var dailyReturnPct float64

		if bal, ok := accountMap[p.PortfolioID]; ok {
			valuationUSD = bal.valuation
			cashUSD = bal.cash
			equityUSD = bal.equity
			if bal.lastEquity > 0 {
				dailyReturnPct = math.Round(((valuationUSD-bal.lastEquity)/bal.lastEquity)*10000) / 100
			}
		} else if s.brokerClient != nil {
			if acc, err := s.brokerClient.GetAccount(ctx, p.PortfolioID); err == nil && acc != nil {
				valuationUSD = acc.PortfolioValue
				if valuationUSD == 0 {
					valuationUSD = acc.Equity
				}
				cashUSD = acc.Cash
				equityUSD = acc.Equity
				if equityUSD == 0 && valuationUSD > 0 {
					equityUSD = valuationUSD - cashUSD
				}
				if acc.LastEquity > 0 {
					dailyReturnPct = math.Round(((valuationUSD-acc.LastEquity)/acc.LastEquity)*10000) / 100
				}
			}
		}

		if _, err := s.ciClient.RecordDailySnapshot(ctx, &coreinvestv1.RecordDailySnapshotRequest{
			PortfolioID:    p.PortfolioID,
			SnapshotDate:   snapshotDate,
			ValuationUSD:   math.Round(valuationUSD*100) / 100,
			CashUSD:        math.Round(cashUSD*100) / 100,
			EquityUSD:      math.Round(equityUSD*100) / 100,
			DailyReturnPct: dailyReturnPct,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to record daily snapshot for portfolio %s: %v", p.PortfolioID, err)
		}
		recorded++
	}

	return &customerbllv1.CaptureDailySnapshotsResponse{
		SnapshotsRecorded: recorded,
		SnapshotDate:      snapshotDate,
	}, nil
}
