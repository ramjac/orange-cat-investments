package alpaca

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client handles communication with the Alpaca Broker API.
type Client struct {
	baseURL    string
	dataURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// Config holds configuration parameters for the Alpaca client.
type Config struct {
	BaseURL    string
	DataURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
}

// NewClient creates a new Alpaca Broker API client.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8089"
	}

	dataURL := strings.TrimRight(cfg.DataURL, "/")
	if dataURL == "" {
		dataURL = baseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &Client{
		baseURL:    baseURL,
		dataURL:    dataURL,
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
		httpClient: httpClient,
	}
}

func (c *Client) do(ctx context.Context, method, fullURL string, body any, target any) error {
	var bodyReader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
	}
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("APCA-API-KEY-ID", c.apiKey)
	}
	if c.apiSecret != "" {
		req.Header.Set("APCA-API-SECRET-KEY", c.apiSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("alpaca api error (%d): %s", resp.StatusCode, msg)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// GetAccount retrieves account details and balances.
func (c *Client) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	url := fmt.Sprintf("%s/v1/accounts/%s", c.baseURL, accountID)
	var acc Account
	if err := c.do(ctx, "GET", url, nil, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

// ListPositions returns all open equity positions for an account.
func (c *Client) ListPositions(ctx context.Context, accountID string) ([]Position, error) {
	url := fmt.Sprintf("%s/v1/trading/accounts/%s/positions", c.baseURL, accountID)
	var positions []Position
	if err := c.do(ctx, "GET", url, nil, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

// GetPosition returns a single equity position for an account.
func (c *Client) GetPosition(ctx context.Context, accountID, symbol string) (*Position, error) {
	url := fmt.Sprintf("%s/v1/trading/accounts/%s/positions/%s", c.baseURL, accountID, strings.ToUpper(symbol))
	var pos Position
	if err := c.do(ctx, "GET", url, nil, &pos); err != nil {
		return nil, err
	}
	return &pos, nil
}

// CreateOrder submits a market or limit order for an account.
func (c *Client) CreateOrder(ctx context.Context, accountID string, req CreateOrderRequest) (*Order, error) {
	url := fmt.Sprintf("%s/v1/trading/accounts/%s/orders", c.baseURL, accountID)
	var ord Order
	if err := c.do(ctx, "POST", url, req, &ord); err != nil {
		return nil, err
	}
	return &ord, nil
}

// GetOrder retrieves a single order by ID.
func (c *Client) GetOrder(ctx context.Context, accountID, orderID string) (*Order, error) {
	url := fmt.Sprintf("%s/v1/trading/accounts/%s/orders/%s", c.baseURL, accountID, orderID)
	var ord Order
	if err := c.do(ctx, "GET", url, nil, &ord); err != nil {
		return nil, err
	}
	return &ord, nil
}

// ListOrders retrieves order history for an account.
func (c *Client) ListOrders(ctx context.Context, accountID string, status string, limit int) ([]Order, error) {
	u, err := url.Parse(fmt.Sprintf("%s/v1/trading/accounts/%s/orders", c.baseURL, accountID))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	var orders []Order
	if err := c.do(ctx, "GET", u.String(), nil, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// GetLatestBars returns latest 1-min OHLCV bars for given symbols.
func (c *Client) GetLatestBars(ctx context.Context, symbols []string) (map[string]Bar, error) {
	symbolsParam := strings.Join(symbols, ",")
	url := fmt.Sprintf("%s/v2/stocks/bars/latest?symbols=%s", c.dataURL, url.QueryEscape(symbolsParam))

	var resp struct {
		Bars map[string]Bar `json:"bars"`
	}
	if err := c.do(ctx, "GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Bars, nil
}

// GetLatestQuote returns the latest quote for a symbol.
func (c *Client) GetLatestQuote(ctx context.Context, symbol string) (*Quote, error) {
	url := fmt.Sprintf("%s/v2/stocks/%s/quotes/latest", c.dataURL, strings.ToUpper(symbol))

	var resp struct {
		Symbol string `json:"symbol"`
		Quote  Quote  `json:"quote"`
	}
	if err := c.do(ctx, "GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Quote, nil
}

// GetPortfolioHistory returns historical performance curve.
func (c *Client) GetPortfolioHistory(ctx context.Context, accountID, period string) (*PortfolioHistory, error) {
	url := fmt.Sprintf("%s/v1/trading/accounts/%s/portfolio/history?period=%s", c.baseURL, accountID, url.QueryEscape(period))
	var history PortfolioHistory
	if err := c.do(ctx, "GET", url, nil, &history); err != nil {
		return nil, err
	}
	return &history, nil
}

// ListAccounts returns all brokerage accounts.
func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	url := fmt.Sprintf("%s/v1/accounts", c.baseURL)
	var accounts []Account
	if err := c.do(ctx, "GET", url, nil, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}
