package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/internal/handler"
	core_invest_handler "github.com/orange-cat-investments/oci/internal/handler/core_invest"
	ops_handler "github.com/orange-cat-investments/oci/internal/handler/ops"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	facilities_repo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	ops_repo "github.com/orange-cat-investments/oci/internal/repository/ops"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/internal/saga"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	bank_engine "github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	bank_handler "github.com/orange-cat-investments/oci/cmd/mock-bank/handler"
	bank_ledger "github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
	broker_engine "github.com/orange-cat-investments/oci/cmd/mock-broker/engine"
	broker_handler "github.com/orange-cat-investments/oci/cmd/mock-broker/handler"
	broker_ledger "github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
)

// Client wraps HTTP communication with the OCI stack (Domain API server and Employee BFF).
type Client struct {
	serverURL  string
	bffURL     string
	custBFFURL string
	brokerURL  string
	bankURL    string
	mode       string
	httpClient *http.Client

	// Mock server references when running in "mock" mode
	mockServer  *httptest.Server
	mockBFF     *httptest.Server
	mockBroker  *httptest.Server
	mockBank    *httptest.Server
	mockCustBFF *httptest.Server
}

// NewClient constructs a new simulation client. In "mock" mode, it spins up in-memory HTTP test servers.
func NewClient(cfg *Config) (*Client, error) {
	c := &Client{
		serverURL:  strings.TrimSuffix(cfg.ServerURL, "/"),
		bffURL:     strings.TrimSuffix(cfg.EmployeeBFFURL, "/"),
		custBFFURL: strings.TrimSuffix(cfg.CustomerBFFURL, "/"),
		brokerURL:  strings.TrimSuffix(cfg.BrokerURL, "/"),
		bankURL:    strings.TrimSuffix(cfg.BankURL, "/"),
		mode:       cfg.Mode,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	if cfg.Mode == "mock" || (cfg.Mode == "step" && cfg.ServerURL == "") {
		if err := c.initMockServers(); err != nil {
			return nil, fmt.Errorf("failed to initialize mock servers: %w", err)
		}
	}

	return c, nil
}

// Close cleans up resources such as mock HTTP servers.
func (c *Client) Close() {
	if c.mockServer != nil {
		c.mockServer.Close()
	}
	if c.mockBFF != nil {
		c.mockBFF.Close()
	}
	if c.mockBroker != nil {
		c.mockBroker.Close()
	}
	if c.mockBank != nil {
		c.mockBank.Close()
	}
	if c.mockCustBFF != nil {
		c.mockCustBFF.Close()
	}
}

func (c *Client) initMockServers() error {
	// Initialize Mock Broker server
	brokerStore := broker_ledger.NewStore()
	mktEngine := broker_engine.NewMarketEngine(brokerStore, time.Second, broker_engine.DefaultAssets())
	broker_ledger.SeedDefaultAccounts(brokerStore, mktEngine.InitialPrices())
	brokerHandler := broker_handler.NewHandler(brokerStore, mktEngine)
	brokerMux := http.NewServeMux()
	brokerHandler.RegisterRoutes(brokerMux)
	c.mockBroker = httptest.NewServer(brokerMux)
	c.brokerURL = c.mockBroker.URL

	// Initialize Mock Bank server
	bankStore := bank_ledger.NewLedgerStore()
	bankEng := bank_engine.NewSettlementEngine(bankStore, nil, time.Second)
	bankHandler := bank_handler.NewHandler(bankStore, bankEng)
	bankMux := http.NewServeMux()
	bankHandler.RegisterRoutes(bankMux)
	c.mockBank = httptest.NewServer(bankMux)
	c.bankURL = c.mockBank.URL

	// Initialize external clients
	alpacaClient := alpaca.NewClient(alpaca.Config{
		BaseURL:   c.brokerURL,
		DataURL:   c.brokerURL,
		APIKey:    "MOCK_ALPACAKEY",
		APISecret: "MOCK_SECRET",
	})
	bankClient := plaid.NewClient(plaid.Config{
		BaseURL:  c.bankURL,
		ClientID: "mock-client-id",
		Secret:   "mock-secret",
	})

	// Initialize Domain API Server Mux
	serverMux := http.NewServeMux()

	ciRepo := core_invest_repo.NewMockRepository()
	ciService := core_invest_svc.NewServiceWithIntegrations(ciRepo, alpacaClient, bankClient)
	ciHandler := core_invest_handler.NewHandler(ciService)
	ciHandler.RegisterRoutes(serverMux)

	opsRepo := ops_repo.NewRepository(nil)
	opsService := ops_svc.NewService(opsRepo)
	opsHandler := ops_handler.NewHandler(opsService)
	opsHandler.RegisterRoutes(serverMux)

	facRepo := facilities_repo.NewMockRepository()
	facService := facilities_svc.NewService(facRepo)

	// Register Facilities HTTP Endpoints on serverMux
	listAssetsHandler := func(w http.ResponseWriter, r *http.Request) {
		var assetTypePtr *string
		if assetType := r.URL.Query().Get("asset_type"); assetType != "" {
			assetTypePtr = &assetType
		}
		items, err := facService.ListAssets(r.Context(), 20, nil, nil, assetTypePtr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"items": items, "total": len(items)})
	}
	serverMux.HandleFunc("GET /facilities/assets", listAssetsHandler)
	serverMux.HandleFunc("GET /api/v1/facilities/assets", listAssetsHandler)

	createAssetHandler := func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SerialNumber       string `json:"serial_number"`
			AssetType          string `json:"asset_type"`
			Model              string `json:"model"`
			ZoneID             string `json:"zone_id"`
			AssignedTo         string `json:"assigned_to"`
			AssignedEmployeeID string `json:"assigned_employee_id"`
			Notes              string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		asset, err := facService.CreateAsset(r.Context(), req.SerialNumber, req.AssetType, req.Model, req.ZoneID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		assigned := req.AssignedTo
		if assigned == "" {
			assigned = req.AssignedEmployeeID
		}

		respMap := map[string]any{
			"asset_id":      asset.AssetID,
			"serial_number": asset.SerialNumber,
			"asset_type":    asset.AssetType,
			"model":         asset.Model,
			"zone_id":       asset.ZoneID,
			"status":        asset.Status,
			"created_at":    asset.CreatedAt,
		}
		if assigned != "" {
			respMap["assigned_to"] = assigned
			respMap["assigned_employee_id"] = assigned
		}
		if req.Notes != "" {
			respMap["notes"] = req.Notes
		}
		json.NewEncoder(w).Encode(respMap)
	}
	serverMux.HandleFunc("POST /facilities/assets", createAssetHandler)
	serverMux.HandleFunc("POST /api/v1/facilities/assets", createAssetHandler)

	// Nextcloud document and chat endpoints for simulator workflows
	createDocHandler := func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title     string   `json:"title"`
			Content   string   `json:"content"`
			Author    string   `json:"author"`
			ShareWith []string `json:"share_with"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"document_id": fmt.Sprintf("doc-%d", time.Now().UnixNano()),
			"title":       req.Title,
			"author":      req.Author,
			"status":      "published",
			"shared_with": req.ShareWith,
			"created_at":  time.Now().UTC().Format(time.RFC3339),
		})
	}
	serverMux.HandleFunc("POST /nextcloud/api/v1/documents", createDocHandler)
	serverMux.HandleFunc("POST /api/v1/nextcloud/documents", createDocHandler)

	listDocsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"documents": []map[string]any{
				{
					"document_id": "doc-001",
					"title":       "OCI Employee & Feline Onboarding Handbook",
					"author":      "emp-human-alice",
				},
				{
					"document_id": "doc-002",
					"title":       "Zero-Trust Microservices & PKI Security Standard",
					"author":      "emp-human-frank",
				},
			},
		})
	}
	serverMux.HandleFunc("GET /nextcloud/api/v1/documents", listDocsHandler)
	serverMux.HandleFunc("GET /api/v1/nextcloud/documents", listDocsHandler)

	chatHandler := func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Sender    string `json:"sender"`
			Recipient string `json:"recipient"`
			Room      string `json:"room"`
			Message   string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message_id": fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			"sender":     req.Sender,
			"recipient":  req.Recipient,
			"room":       req.Room,
			"status":     "delivered",
			"sent_at":    time.Now().UTC().Format(time.RFC3339),
		})
	}
	serverMux.HandleFunc("POST /nextcloud/api/v1/chat/messages", chatHandler)
	serverMux.HandleFunc("POST /api/v1/nextcloud/chat/messages", chatHandler)

	serverMux.HandleFunc("POST /facilities/assets/{id}/maintenance", func(w http.ResponseWriter, r *http.Request) {
		assetID := r.PathValue("id")
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ticket, err := opsService.CreateITTicket(r.Context(), &ops_svc.CreateITTicketRequest{
			ForgejoRepo:    "facilities/assets",
			Title:          req.Title,
			Body:           fmt.Sprintf("Asset %s maintenance: %s", assetID, req.Description),
			AuthorUsername: "emp-human-bob",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ticket)
	})

	serverMux.HandleFunc("POST /facilities/sync/maintenance-logs", func(w http.ResponseWriter, r *http.Request) {
		var logs []*facilities_repo.MaintenanceLog
		if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		syncedLogs, err := facService.SyncMaintenanceLogs(r.Context(), logs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"synced_count": len(syncedLogs),
			"items":        syncedLogs,
		})
	})

	// Customer BFF routes and Mock Customer BFF server
	custMux := http.NewServeMux()
	registerCustRoutes := func(mux *http.ServeMux) {
		mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","service":"customer-bff"}`))
		})
		mux.HandleFunc("GET /api/v1/customer/bank-accounts", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			customerID := r.URL.Query().Get("customer_id")
			if customerID == "" {
				customerID = "cust-longterm-arthur"
			}
			accs, err := bankClient.GetAccounts(r.Context(), customerID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"accounts": accs})
		})
		mux.HandleFunc("GET /api/v1/customer/portfolio", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			custID := r.URL.Query().Get("customer_id")
			portfolioID := "018f3a9a-2222-7000-8000-000000000002"
			if custID == "cust-active-chloe" || custID == "chloe" {
				portfolioID = "018f3a9a-2222-7000-8000-000000000003"
			}
			acc, err := alpacaClient.GetAccount(r.Context(), portfolioID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			positions, _ := alpacaClient.ListPositions(r.Context(), portfolioID)
			holdings := make([]map[string]any, 0, len(positions))
			for _, p := range positions {
				holdings = append(holdings, map[string]any{
					"symbol":        p.Symbol,
					"quantity":      p.Qty,
					"avg_price":     p.AvgEntryPrice,
					"current_price": p.CurrentPrice,
					"market_value":  p.MarketValue,
				})
			}
			json.NewEncoder(w).Encode(map[string]any{
				"portfolio_id":      portfolioID,
				"customer_id":       custID,
				"total_balance_usd": acc.PortfolioValue,
				"cash_balance_usd":  acc.Cash,
				"status":            acc.Status,
				"holdings":          holdings,
			})
		})
		mux.HandleFunc("GET /api/v1/customer/ticker", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			symbols := []string{"AAPL", "NVDA", "MSFT", "TSLA"}
			bars, _ := alpacaClient.GetLatestBars(r.Context(), symbols)
			tickerData := make([]map[string]any, 0, len(symbols))
			for _, sym := range symbols {
				price := 150.0
				vol := int64(100000)
				if b, ok := bars[sym]; ok && b.Close > 0 {
					price = b.Close
					vol = b.Volume
				}
				tickerData = append(tickerData, map[string]any{
					"symbol": sym,
					"price":  price,
					"volume": vol,
				})
			}
			json.NewEncoder(w).Encode(tickerData)
		})
		mux.HandleFunc("POST /api/v1/customer/deposits", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				CustomerID  string  `json:"customer_id"`
				PortfolioID string  `json:"portfolio_id"`
				AmountUSD   float64 `json:"amount_usd"`
				Frequency   string  `json:"frequency"`
				BankAccount string  `json:"bank_account"`
				AccountID   string  `json:"account_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if req.CustomerID == "" {
				req.CustomerID = "cust-longterm-arthur"
			}
			if req.PortfolioID == "" {
				if req.CustomerID == "cust-active-chloe" {
					req.PortfolioID = "018f3a9a-2222-7000-8000-000000000003"
				} else {
					req.PortfolioID = "018f3a9a-2222-7000-8000-000000000002"
				}
			}
			dep, err := ciService.CreateRecurringDeposit(r.Context(), core_invest_svc.CreateRecurringDepositOpts{
				CustomerID:  req.CustomerID,
				PortfolioID: req.PortfolioID,
				AmountUSD:   req.AmountUSD,
				Frequency:   req.Frequency,
				BankAccount: req.BankAccount,
				AccountID:   req.AccountID,
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(dep)
		})
		mux.HandleFunc("POST /api/v1/customer/trading/orders", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				PortfolioID string  `json:"portfolio_id"`
				Symbol      string  `json:"symbol"`
				Side        string  `json:"side"`
				Quantity    float64 `json:"quantity"`
				Price       float64 `json:"price"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if req.PortfolioID == "" {
				req.PortfolioID = "018f3a9a-2222-7000-8000-000000000002"
			}
			ord, err := ciService.ExecuteBrokerageOrder(r.Context(), core_invest_svc.ExecuteBrokerageOrderOpts{
				PortfolioID: req.PortfolioID,
				BrokerName:  "OCI Smart Execution Router",
				Symbol:      req.Symbol,
				Side:        req.Side,
				Quantity:    req.Quantity,
				Price:       req.Price,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(ord)
		})
		mux.HandleFunc("GET /api/v1/customer/trade-logs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			orders, err := ciService.ListBrokerageOrders(r.Context(), 50, 0)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(orders)
		})
	}
	registerCustRoutes(custMux)
	registerCustRoutes(serverMux)

	c.mockCustBFF = httptest.NewServer(custMux)
	c.custBFFURL = c.mockCustBFF.URL

	c.mockServer = httptest.NewServer(serverMux)
	c.serverURL = c.mockServer.URL

	// Initialize Employee BFF Mux
	bffMux := http.NewServeMux()

	wfRepo := workforce_repo.NewRepository(nil)
	wfService := workforce_svc.NewService(wfRepo)
	onboardingSaga := saga.NewOnboardingSaga(wfService, nil, nil)
	offboardingEng := saga.NewOffboardingEngine(wfService, nil, nil)

	wfHandler := handler.NewWorkforceHandler(wfService, onboardingSaga, offboardingEng)
	wfHandler.RegisterRoutes(bffMux)

	c.mockBFF = httptest.NewServer(bffMux)
	c.bffURL = c.mockBFF.URL

	return nil
}

func (c *Client) doRequest(ctx context.Context, fullURL, method string, body any, headers map[string]string) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, nil, fmt.Errorf("failed to encode JSON request body: %w", err)
		}
		bodyReader = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed (%s %s): %w", method, fullURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	return resp, respBytes, nil
}

// Do executes an HTTP request against either the Server or Employee BFF target URL.
func (c *Client) Do(ctx context.Context, useBFF bool, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.serverURL
	if useBFF {
		baseURL = c.bffURL
	}
	return c.doRequest(ctx, baseURL+path, method, body, nil)
}

// DoCustomer executes an HTTP request against the Customer BFF target URL.
func (c *Client) DoCustomer(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.custBFFURL
	if baseURL == "" {
		baseURL = c.serverURL
	}
	return c.doRequest(ctx, baseURL+path, method, body, nil)
}

// DoBroker executes an HTTP request directly against the Mock Broker API.
func (c *Client) DoBroker(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.brokerURL
	if baseURL == "" {
		baseURL = "http://localhost:8089"
	}
	headers := map[string]string{
		"APCA-API-KEY-ID":     "MOCK_ALPACAKEY",
		"APCA-API-SECRET-KEY": "MOCK_SECRET",
	}
	return c.doRequest(ctx, baseURL+path, method, body, headers)
}

// DoBank executes an HTTP request directly against the Mock Bank API.
func (c *Client) DoBank(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.bankURL
	if baseURL == "" {
		baseURL = "http://localhost:8088"
	}
	headers := map[string]string{
		"PLAID-CLIENT-ID": "mock-client-id",
		"PLAID-SECRET":    "mock-secret",
	}
	return c.doRequest(ctx, baseURL+path, method, body, headers)
}
