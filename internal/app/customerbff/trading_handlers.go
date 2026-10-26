package customerbff

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/httputil"
)

func (b *CustomerBFF) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		PortfolioID string  `json:"portfolio_id"`
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"` // "buy" or "sell"
		Quantity    float64 `json:"quantity"`
		OrderType   string  `json:"order_type"` // "market" or "limit"
		Price       float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	req.Side = strings.ToLower(strings.TrimSpace(req.Side))
	if req.Symbol == "" || (req.Side != "buy" && req.Side != "sell") || req.Quantity <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "symbol, side (buy/sell), and positive quantity are required")
		return
	}
	if req.PortfolioID == "" {
		req.PortfolioID = fmt.Sprintf("port-%s", session.CustomerID)
	}
	if req.Price <= 0 {
		req.Price = 224.30
	}
	if req.OrderType == "" {
		req.OrderType = "market"
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	orderResp, err := b.bllClient.ValidateAndExecuteOrder(r.Context(), &customerbllv1.ValidateAndExecuteOrderRequest{
		CustomerID:  session.CustomerID,
		PortfolioID: req.PortfolioID,
		Symbol:      req.Symbol,
		Side:        req.Side,
		Quantity:    req.Quantity,
		OrderType:   req.OrderType,
		LimitPrice:  req.Price,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Cache invalidation for customer portfolio and active orders
	if b.cache != nil && session.CustomerID != "" {
		_ = b.cache.Delete(r.Context(), cachevalkey.KeyPortfolio(session.CustomerID), cachevalkey.KeyOrders(session.CustomerID))
		_ = b.cache.DeletePrefix(r.Context(), "cache:portfolio:"+session.CustomerID)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"order_id":     orderResp.OrderID,
		"portfolio_id": orderResp.PortfolioID,
		"symbol":       orderResp.Symbol,
		"side":         orderResp.Side,
		"quantity":     orderResp.Quantity,
		"price":        orderResp.Price,
		"status":       orderResp.Status,
		"executed_at":  orderResp.ExecutedAt,
	})
}

func (b *CustomerBFF) handleGetTradeLogs(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListTradeOrders(r.Context(), &customerbllv1.ListTradeOrdersRequest{
		CustomerID: session.CustomerID,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, []*coreinvestv1.BrokerageOrder{})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp.Orders)
}

func (b *CustomerBFF) handleTickerSSE(w http.ResponseWriter, r *http.Request) {
	if b.cache != nil {
		var cachedTicker []map[string]any
		if ok, _ := b.cache.Get(r.Context(), cachevalkey.KeyQuotes("TICKER_ALL"), &cachedTicker); ok && len(cachedTicker) > 0 {
			httputil.WriteJSON(w, http.StatusOK, cachedTicker)
			return
		}
	}

	symbols := []string{"AAPL", "NVDA", "MSFT", "TSLA"}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetTickerSnapshot(r.Context(), &customerbllv1.GetTickerSnapshotRequest{
		Symbols: symbols,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, fmt.Sprintf("failed to fetch market quotes: %v", err))
		return
	}

	tickerData := make([]map[string]any, 0, len(resp.Quotes))
	for _, q := range resp.Quotes {
		tickerData = append(tickerData, map[string]any{
			"symbol":     q.Symbol,
			"price":      q.Price,
			"change_pct": q.ChangePct,
			"volume":     q.Volume,
		})
	}

	if b.cache != nil && len(tickerData) > 0 {
		_ = b.cache.Set(r.Context(), cachevalkey.KeyQuotes("TICKER_ALL"), tickerData, cachevalkey.TTLMarketQuotes)
		for _, item := range tickerData {
			if sym, ok := item["symbol"].(string); ok {
				_ = b.cache.Set(r.Context(), cachevalkey.KeyQuotes(sym), item, cachevalkey.TTLMarketQuotes)
			}
		}
	}

	httputil.WriteJSON(w, http.StatusOK, tickerData)
}

func (b *CustomerBFF) handleGetZoomieIndex(w http.ResponseWriter, r *http.Request) {
	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.GetZoomieIndex(r.Context(), &customerbllv1.GetZoomieIndexRequest{})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get zoomie index")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"zoomie_index_score":     resp.ZoomieIndexScore,
		"momentum_level":         resp.MomentumLevel,
		"feline_leader":          resp.FelineLeader,
		"active_pounces_cnt":     resp.ActivePouncesCnt,
		"current_activity":       resp.CurrentActivity,
		"confidence_score":       resp.ConfidenceScore,
		"trading_signal":         resp.TradingSignal,
		"last_updated":           resp.LastUpdated,
		"recommended_multiplier": resp.RecommendedMultiplier,
	})
}

func (b *CustomerBFF) handleGetAPIKeys(w http.ResponseWriter, r *http.Request) {
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}
	if custID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: customer identity required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListAPIKeys(r.Context(), &customerbllv1.ListAPIKeysRequest{
		CustomerID: custID,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, []any{})
		return
	}

	list := make([]map[string]any, 0, len(resp.Keys))
	for _, k := range resp.Keys {
		list = append(list, map[string]any{
			"key_id":      k.KeyID,
			"customer_id": k.CustomerID,
			"name":        k.Label,
			"key_name":    k.Label,
			"key_prefix":  k.KeyPrefix,
			"permissions": k.Scopes,
			"scopes":      k.Scopes,
			"created_at":  k.CreatedAt,
		})
	}

	httputil.WriteJSON(w, http.StatusOK, list)
}

func (b *CustomerBFF) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		Name        string   `json:"name"`
		KeyName     string   `json:"key_name"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		req.Name = req.KeyName
	}
	if req.Name == "" {
		req.Name = "Default API Key"
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.CreateAPIKey(r.Context(), &customerbllv1.CreateAPIKeyRequest{
		CustomerID:  session.CustomerID,
		Name:        req.Name,
		Permissions: req.Permissions,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	key := resp.Key
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"key_id":      key.KeyID,
		"customer_id": key.CustomerID,
		"name":        key.Label,
		"key_name":    key.Label,
		"key_prefix":  key.KeyPrefix,
		"permissions": key.Scopes,
		"scopes":      key.Scopes,
		"created_at":  key.CreatedAt,
		"secret":      resp.RawSecret,
	})
}

func (b *CustomerBFF) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}
	keyID := chi.URLParam(r, "id")
	if keyID == "" {
		keyID = r.PathValue("id")
	}
	if keyID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "key id is required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.RevokeAPIKey(r.Context(), &customerbllv1.RevokeAPIKeyRequest{
		CustomerID: session.CustomerID,
		KeyID:      keyID,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (b *CustomerBFF) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}
	if custID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: customer identity required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	resp, err := b.bllClient.ListWebhooks(r.Context(), &customerbllv1.ListWebhooksRequest{
		CustomerID: custID,
	})
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, []any{})
		return
	}

	list := make([]map[string]any, 0, len(resp.Subscriptions))
	for _, wh := range resp.Subscriptions {
		list = append(list, map[string]any{
			"subscription_id": wh.WebhookID,
			"webhook_id":      wh.WebhookID,
			"customer_id":     wh.CustomerID,
			"target_url":      wh.TargetURL,
			"url":             wh.TargetURL,
			"events":          wh.EventTypes,
			"status":          wh.Status,
			"created_at":      wh.CreatedAt,
		})
	}

	httputil.WriteJSON(w, http.StatusOK, list)
}

func (b *CustomerBFF) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	session := CustomerFromContext(r.Context())
	if session == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized: valid customer session required")
		return
	}

	var req struct {
		URL       string   `json:"url"`
		TargetURL string   `json:"target_url"`
		Events    []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TargetURL == "" {
		req.TargetURL = req.URL
	}
	if req.TargetURL == "" {
		httputil.WriteError(w, http.StatusBadRequest, "target_url or url is required")
		return
	}

	if b.bllClient == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "customer bll service unavailable")
		return
	}

	tokBytes := make([]byte, 32)
	_, _ = crand.Read(tokBytes)
	secret := "whsec_" + hex.EncodeToString(tokBytes)

	resp, err := b.bllClient.CreateWebhook(r.Context(), &customerbllv1.CreateWebhookRequest{
		CustomerID: session.CustomerID,
		TargetURL:  req.TargetURL,
		Events:     req.Events,
		Secret:     secret,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	wh := resp.Subscription
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"subscription_id": wh.WebhookID,
		"webhook_id":      wh.WebhookID,
		"customer_id":     wh.CustomerID,
		"target_url":      wh.TargetURL,
		"url":             wh.TargetURL,
		"events":          wh.EventTypes,
		"secret":          secret,
		"status":          wh.Status,
		"created_at":      wh.CreatedAt,
	})
}
