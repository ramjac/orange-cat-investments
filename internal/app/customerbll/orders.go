package customerbll

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/authz/policy"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *Server) ValidateAndExecuteOrder(ctx context.Context, in *customerbllv1.ValidateAndExecuteOrderRequest) (*customerbllv1.ValidateAndExecuteOrderResponse, error) {
	if in == nil || in.PortfolioID == "" || in.Symbol == "" || in.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "portfolio_id, symbol, and quantity > 0 are required")
	}

	if in.Side != "buy" && in.Side != "sell" {
		return nil, status.Error(codes.InvalidArgument, "side must be 'buy' or 'sell'")
	}

	orderType := in.OrderType
	if orderType == "" {
		orderType = "market"
	}
	if orderType != "market" && orderType != "limit" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported order type '%s'; supported types: 'market', 'limit'", orderType)
	}

	var limitPricePtr *float64
	if orderType == "limit" {
		if in.LimitPrice <= 0 {
			return nil, status.Error(codes.InvalidArgument, "limit order requires limit_price > 0")
		}
		limitPricePtr = &in.LimitPrice
	}
	if in.StopLoss > 0 || in.TakeProfit > 0 {
		return nil, status.Error(codes.InvalidArgument, "stop loss and take profit not supported by current broker integration")
	}

	// 1. Business Rule: Check feline medical hold from workforce domain
	targetFeline, err := s.resolveTargetFeline(ctx, in.FelineID)
	if err != nil || targetFeline == "" {
		s.log().Error("feline executive resolution failed", "error", err)
		return nil, status.Error(codes.Unavailable, "unable to resolve feline executive for required medical hold verification")
	}

	medicalHoldActive, err := s.checkFelineMedicalHold(ctx, targetFeline)
	if err != nil {
		s.log().Error("feline medical hold verification failed; failing closed", "error", err, "feline_id", targetFeline)
		return nil, status.Error(codes.Unavailable, "feline medical hold safety check unavailable; failing closed")
	}

	// 1.5 Embedded OPA Policy-as-Code Pre-Trade Authorization (Phase 10 / 7.42)
	// Account state must come from a successful, nonempty core-invest lookup;
	// failing open would let blocked accounts trade when the domain is down.
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest-api is not configured; account state unavailable for pre-trade authorization")
	}
	portResp, pErr := s.ciClient.GetPortfolio(ctx, &coreinvestv1.GetPortfolioRequest{PortfolioID: in.PortfolioID})
	if pErr != nil || portResp == nil || portResp.Portfolio == nil || portResp.Portfolio.Status == "" {
		s.log().Error("pre-trade portfolio lookup failed; failing closed", "error", pErr, "portfolio_id", in.PortfolioID)
		return nil, status.Error(codes.Unavailable, "account state unavailable for pre-trade authorization; failing closed")
	}
	accountStatus := portResp.Portfolio.Status
	buyingPower := portResp.Portfolio.CashBalanceUSD
	if buyingPower < 0 {
		buyingPower = 0
	}

	// Estimate the order notional without trusting caller-supplied prices.
	// The broker ignores LimitPrice for market orders, so market orders must
	// be sized from a trusted live broker quote (fail closed when it is
	// unavailable); limit orders use the validated limit price.
	var referencePrice float64
	var estimatedNotional float64
	switch orderType {
	case "limit":
		referencePrice = in.LimitPrice
		estimatedNotional = in.Quantity * in.LimitPrice
	case "market":
		quote, qErr := s.trustedMarketQuote(ctx, in.Symbol, in.Side)
		if qErr != nil || quote <= 0 {
			s.log().Error("trusted market quote unavailable; failing closed", "error", qErr, "symbol", in.Symbol)
			return nil, status.Error(codes.Unavailable, "trusted market quote unavailable; market order notional cannot be estimated")
		}
		referencePrice = quote
		estimatedNotional = in.Quantity * quote
	}
	if s.policyEngine == nil {
		return nil, status.Error(codes.PermissionDenied, "trading policy engine not initialized")
	}
	tradeDec, policyErr := s.policyEngine.EvaluateTrading(ctx, policy.TradingInput{
		Actor: policy.Actor{
			ID:   in.CustomerID,
			Role: "customer",
		},
		Account: policy.AccountState{
			Status:         accountStatus,
			BuyingPowerUSD: buyingPower,
		},
		Feline: policy.FelineState{
			FelineID:             targetFeline,
			EmergencyMedicalHold: medicalHoldActive,
		},
		Order: policy.OrderRequest{
			Symbol:               in.Symbol,
			Side:                 in.Side,
			Quantity:             in.Quantity,
			OrderType:            orderType,
			EstimatedNotionalUSD: estimatedNotional,
		},
	})
	if policyErr != nil || !tradeDec.Allow {
		reason := tradeDec.ReasonSummary()
		if policyErr != nil && reason == "" {
			reason = policyErr.Error()
		}
		s.log().Warn("OPA trading policy denied order",
			"customer_id", in.CustomerID,
			"portfolio_id", in.PortfolioID,
			"feline_id", targetFeline,
			"medical_hold", medicalHoldActive,
			"reason", reason,
		)
		s.recordAudit(ctx, in.CustomerID, "customer", "authz.policy.denied", "trading_order", in.PortfolioID, map[string]any{
			"policy":                 tradeDec.Policy,
			"symbol":                 in.Symbol,
			"side":                   in.Side,
			"quantity":               in.Quantity,
			"account_status":         accountStatus,
			"emergency_medical_hold": medicalHoldActive,
			"reasons":                tradeDec.Reasons,
		})
		if medicalHoldActive {
			return nil, status.Error(codes.FailedPrecondition, reason)
		}
		return nil, status.Error(codes.PermissionDenied, reason)
	}

	// 2. Execute order with broker
	execStart := time.Now()
	if s.brokerClient == nil {
		if s.ciClient != nil {
			res, err := s.ciClient.CreateOrder(ctx, &coreinvestv1.CreateOrderRequest{
				PortfolioID: in.PortfolioID,
				Symbol:      in.Symbol,
				Side:        in.Side,
				Quantity:    in.Quantity,
				Price:       referencePrice,
				OrderType:   orderType,
			})
			if err == nil && res != nil && res.Order != nil {
				telemetry.RecordOrderExecutionLatency(ctx, float64(time.Since(execStart).Microseconds())/1000.0)
				return &customerbllv1.ValidateAndExecuteOrderResponse{
					OrderID:     res.Order.OrderID,
					CustomerID:  in.CustomerID,
					PortfolioID: res.Order.PortfolioID,
					BrokerName:  res.Order.BrokerName,
					Symbol:      res.Order.Symbol,
					Side:        res.Order.Side,
					Quantity:    res.Order.Quantity,
					Price:       res.Order.Price,
					Status:      res.Order.Status,
					ExecutedAt:  res.Order.ExecutedAt,
				}, nil
			}
		}
		s.log().Error("broker client not configured")
		return nil, status.Error(codes.Unavailable, "broker integration not configured")
	}

	clientOrderID := "ord-" + watermill.NewUUID()[:8]
	brokerCtx, brokerCancel := s.withUpstreamTimeout(ctx)
	brokerResp, err := s.brokerClient.CreateOrder(brokerCtx, in.PortfolioID, alpaca.CreateOrderRequest{
		Symbol:        in.Symbol,
		Qty:           in.Quantity,
		Side:          in.Side,
		Type:          orderType,
		TimeInForce:   "day",
		LimitPrice:    limitPricePtr,
		ClientOrderID: clientOrderID,
	})
	brokerCancel()
	if err != nil {
		s.log().Error("broker order execution failed", "error", sanitizeVendorNames(err.Error()), "portfolio_id", in.PortfolioID)
		return nil, mapUpstreamError("broker", err)
	}
	recordUpstreamSuccess(ctx, "broker")
	if brokerResp == nil {
		return nil, status.Error(codes.Unavailable, "empty response from broker")
	}

	executedPrice := 0.0
	if brokerResp.FilledAvgPrice != nil {
		executedPrice = *brokerResp.FilledAvgPrice
	} else if in.LimitPrice > 0 {
		executedPrice = in.LimitPrice
	}

	orderStatus := brokerResp.Status
	if orderStatus == "" {
		orderStatus = "submitted"
	}

	telemetry.RecordOrderExecutionLatency(ctx, float64(time.Since(execStart).Microseconds())/1000.0)

	orderID := brokerResp.ID
	if orderID == "" {
		orderID = clientOrderID
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	result := &customerbllv1.ValidateAndExecuteOrderResponse{
		OrderID:     orderID,
		CustomerID:  in.CustomerID,
		PortfolioID: in.PortfolioID,
		BrokerName:  "OCI Smart Execution Router",
		Symbol:      in.Symbol,
		Side:        in.Side,
		Quantity:    in.Quantity,
		Price:       executedPrice,
		Status:      orderStatus,
		ExecutedAt:  nowStr,
	}

	// 3. Persist order through core-invest-api via gRPC (atomically writes core_invest.outbox_events in same pgx.Tx)
	persistedViaOutbox := false
	if s.ciClient != nil {
		_, err := s.ciClient.CreateOrder(ctx, &coreinvestv1.CreateOrderRequest{
			PortfolioID: in.PortfolioID,
			Symbol:      in.Symbol,
			Side:        in.Side,
			Quantity:    in.Quantity,
			Price:       executedPrice,
			OrderType:   orderType,
		})
		if err != nil {
			s.log().Warn("failed to persist order to core-invest-api via gRPC", "error", err)
		} else {
			persistedViaOutbox = true
		}
	}

	// 4. Fallback direct event publish only when core-invest-api outbox persistence was unavailable
	if !persistedViaOutbox && s.publisher != nil {
		eventID := watermill.NewUUID()
		event := envelope.NewEventEnvelope(ctx, eventID, envelope.TopicOrderPlaced, orderID, result)
		eventBytes, _ := json.Marshal(event)
		_ = s.publisher.Publish(envelope.TopicOrderPlaced, message.NewMessage(event.EventID, eventBytes))
	}

	// 5. Publish transactional trade_executed notification event for async SMTP confirmation (Phase 11 / 7.45)
	if s.publisher != nil {
		customerEmail := resolveOrderCustomerEmail(ctx, portResp.Portfolio, in.CustomerID)
		if customerEmail != "" {
			txEventID := watermill.NewUUID()
			txEvent := envelope.NewEventEnvelope(ctx, txEventID, envelope.TopicTradeExecuted, orderID, envelope.TradeExecutedPayload{
				OrderID:       orderID,
				CustomerID:    in.CustomerID,
				CustomerEmail: customerEmail,
				PortfolioID:   in.PortfolioID,
				Symbol:        in.Symbol,
				Side:          in.Side,
				Quantity:      in.Quantity,
				Price:         executedPrice,
				Status:        orderStatus,
				ExecutedAt:    nowStr,
			})
			if txBytes, mErr := json.Marshal(txEvent); mErr == nil {
				_ = s.publisher.Publish(envelope.TopicTradeExecuted, message.NewMessage(txEvent.EventID, txBytes))
			}
		}
	}

	return result, nil
}

func resolveOrderCustomerEmail(ctx context.Context, _ *coreinvestv1.Portfolio, customerID string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-customer-email"); len(vals) > 0 && strings.Contains(vals[0], "@") {
			return strings.TrimSpace(vals[0])
		}
	}
	if customerID != "" {
		return strings.ToLower(strings.TrimSpace(customerID)) + "@oci.local"
	}
	return ""
}

// trustedMarketQuote returns a conservative per-share reference price used to
// estimate market-order notionals: the live ask for buys and bid for sells,
// sourced from the broker (the execution counterparty) rather than the
// caller. It fails closed when no trusted market data source is configured.
func (s *Server) trustedMarketQuote(ctx context.Context, symbol, side string) (float64, error) {
	if s.brokerClient == nil {
		return 0, errors.New("no trusted market data source configured for market order estimation")
	}
	qCtx, cancel := s.withUpstreamTimeout(ctx)
	defer cancel()
	quote, err := s.brokerClient.GetLatestQuote(qCtx, symbol)
	if err != nil {
		return 0, err
	}
	if quote == nil {
		return 0, errors.New("empty quote response from broker")
	}
	if side == "buy" {
		return quote.AskPrice, nil
	}
	return quote.BidPrice, nil
}

func (s *Server) checkFelineMedicalHold(ctx context.Context, felineID string) (bool, error) {
	if s.wfClient == nil {
		return false, errors.New("workforce gRPC client is not configured")
	}
	res, err := s.wfClient.GetCareSchedule(ctx, &workforcev1.GetCareScheduleRequest{FelineID: felineID})
	if err != nil {
		return false, err
	}
	if res != nil && res.CareSchedule != nil {
		return res.CareSchedule.EmergencyMedicalHold, nil
	}
	return false, nil
}

func (s *Server) resolveTargetFeline(ctx context.Context, requestedID string) (string, error) {
	if requestedID != "" {
		return requestedID, nil
	}
	if s.defaultFelineID != "" {
		return s.defaultFelineID, nil
	}
	felineID, err := s.fetchFirstActiveFeline(ctx)
	if err != nil {
		s.log().Warn("could not fetch default feline employee from workforce domain API", "error", err)
		return "", err
	}
	if felineID != "" {
		return felineID, nil
	}
	return "", errors.New("no feline executive available or configured")
}

func (s *Server) fetchFirstActiveFeline(ctx context.Context) (string, error) {
	if s.wfClient == nil {
		return "", errors.New("workforce gRPC client is not configured")
	}
	res, err := s.wfClient.ListEmployees(ctx, &workforcev1.ListEmployeesRequest{
		EmployeeType: "feline",
		Status:       "active",
	})
	if err != nil {
		return "", err
	}
	if res != nil && len(res.Employees) > 0 && res.Employees[0].EmployeeID != "" {
		return res.Employees[0].EmployeeID, nil
	}
	return "", errors.New("no active feline employees found")
}

func (s *Server) ListTradeOrders(ctx context.Context, in *customerbllv1.ListTradeOrdersRequest) (*customerbllv1.ListTradeOrdersResponse, error) {
	if s.ciClient == nil {
		return &customerbllv1.ListTradeOrdersResponse{Orders: []*coreinvestv1.BrokerageOrder{}}, nil
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	res, err := s.ciClient.ListOrders(ctx, &coreinvestv1.ListOrdersRequest{
		PortfolioID: in.PortfolioID,
		Limit:       limit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}
	return &customerbllv1.ListTradeOrdersResponse{Orders: res.Orders}, nil
}

func (s *Server) GetTickerSnapshot(ctx context.Context, in *customerbllv1.GetTickerSnapshotRequest) (*customerbllv1.GetTickerSnapshotResponse, error) {
	if s.brokerClient == nil {
		return nil, status.Error(codes.Unavailable, "market data broker service unavailable")
	}
	symbols := in.Symbols
	if len(symbols) == 0 {
		symbols = []string{"AAPL", "NVDA", "MSFT", "TSLA"}
	}
	barsCtx, barsCancel := s.withUpstreamTimeout(ctx)
	bars, err := s.brokerClient.GetLatestBars(barsCtx, symbols)
	barsCancel()
	if err != nil {
		s.log().Error("failed to fetch market data bars from broker", "error", sanitizeVendorNames(err.Error()))
		return nil, mapUpstreamError("broker", err)
	}
	recordUpstreamSuccess(ctx, "broker")
	quotesCtx, quotesCancel := s.withUpstreamTimeout(ctx)
	quotes, _ := s.brokerClient.GetLatestQuotes(quotesCtx, symbols)
	quotesCancel()

	out := make([]*customerbllv1.TickerQuote, 0, len(symbols))
	for _, sym := range symbols {
		if bar, ok := bars[sym]; ok && bar.Close > 0 {
			price := bar.Close
			if quote, ok := quotes[sym]; ok && quote.AskPrice > 0 {
				price = (quote.BidPrice + quote.AskPrice) / 2.0
			}
			changePct := 0.0
			if bar.Open > 0 {
				changePct = math.Round(((price-bar.Open)/bar.Open)*10000.0) / 100.0
			}
			out = append(out, &customerbllv1.TickerQuote{
				Symbol:    sym,
				Price:     math.Round(price*100.0) / 100.0,
				ChangePct: changePct,
				Volume:    bar.Volume,
			})
		}
	}
	return &customerbllv1.GetTickerSnapshotResponse{Quotes: out}, nil
}

func (s *Server) GetZoomieIndex(ctx context.Context, in *customerbllv1.GetZoomieIndexRequest) (*customerbllv1.GetZoomieIndexResponse, error) {
	var felineCount int32
	var vhrCount int32
	leaderName := "Feline Leader"

	emps, _ := s.getActiveFelines(ctx)
	if len(emps) > 0 {
		felineCount = int32(len(emps))
		if emps[0].FirstName != "" {
			leaderName = emps[0].FirstName
		}
	}
	if s.wfClient != nil {
		vhrRes, err := s.wfClient.ListVHRRecords(ctx, &workforcev1.ListVHRRecordsRequest{})
		if err == nil && vhrRes != nil {
			vhrCount = int32(len(vhrRes.Records))
		}
	}
	idx := 88.5
	if felineCount > 0 {
		idx = 50.0 + float64(felineCount)*5.5 + float64(vhrCount)*1.2
		if idx > 99.5 {
			idx = 98.4
		}
	}
	nowStr := time.Now().UTC().Format(time.RFC3339)
	return &customerbllv1.GetZoomieIndexResponse{
		ZoomieIndexScore:      idx,
		ZoomieIndex:           idx,
		MomentumLevel:         "HIGH_VELOCITY_3AM_ZOOMIES",
		FelineLeader:          leaderName,
		ActivePouncesCnt:      14,
		CurrentActivity:       "zooming",
		ConfidenceScore:       0.965,
		TradingSignal:         "TACTICAL_BUY_BULLISH",
		LastUpdated:           nowStr,
		RecommendedMultiplier: 1.75,
		FelinesActive:         felineCount,
		VHRSampleCount:        vhrCount,
		Status:                "nominal_high_activity",
		CalculatedAt:          nowStr,
	}, nil
}
