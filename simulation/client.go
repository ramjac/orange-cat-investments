package simulation

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/internal/handler"
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
	"github.com/orange-cat-investments/oci/pkg/reporting"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

// Client wraps HTTP communication with the OCI stack (Domain API server and Employee BFF).
type Client struct {
	serverURL       string
	bffURL          string
	custBFFURL      string
	brokerURL       string
	bankURL         string
	forgejoURL      string
	mode            string
	transport       http.RoundTripper
	httpClient      *http.Client
	jarsMu          sync.RWMutex
	personaJars     map[string]http.CookieJar
	personaCSRF     map[string]string
	activePersonaID string
	requireAuth     bool

	// Mock server references when running in "mock" mode
	mockServer               *httptest.Server
	mockBFF                  *httptest.Server
	mockBroker               *httptest.Server
	mockBank                 *httptest.Server
	mockCustBFF              *httptest.Server
	mockForgejo              *httptest.Server
	mockEmployeeSessionStore sessionvalkey.EmployeeSessionStore
	mockCustomerSessionStore sessionvalkey.CustomerSessionStore
}

// NewClient constructs a new simulation client. In "mock" mode, it spins up in-memory HTTP test servers.
func NewClient(cfg *Config) (*Client, error) {
	ingressIP := os.Getenv("OCI_INGRESS_IP")
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if ingressIP != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err == nil && (strings.HasSuffix(host, ".oci.local") || host == "oci.local") {
				return dialer.DialContext(ctx, network, net.JoinHostPort(ingressIP, port))
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	c := &Client{
		serverURL:   strings.TrimSuffix(cfg.ServerURL, "/"),
		bffURL:      strings.TrimSuffix(cfg.EmployeeBFFURL, "/"),
		custBFFURL:  strings.TrimSuffix(cfg.CustomerBFFURL, "/"),
		brokerURL:   strings.TrimSuffix(cfg.BrokerURL, "/"),
		bankURL:     strings.TrimSuffix(cfg.BankURL, "/"),
		forgejoURL:  strings.TrimSuffix(cfg.ForgejoURL, "/"),
		mode:        cfg.Mode,
		transport:   transport,
		personaJars: make(map[string]http.CookieJar),
		personaCSRF: make(map[string]string),
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
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
	if c.mockForgejo != nil {
		c.mockForgejo.Close()
	}
	if c.mockEmployeeSessionStore != nil {
		_ = c.mockEmployeeSessionStore.Close()
	}
	if c.mockCustomerSessionStore != nil {
		_ = c.mockCustomerSessionStore.Close()
	}
}

func (c *Client) initMockBrokerServer() {
	brokerStore := broker_ledger.NewStore()
	mktEngine := broker_engine.NewMarketEngine(brokerStore, time.Second, broker_engine.DefaultAssets())
	broker_ledger.SeedDefaultAccounts(brokerStore, mktEngine.InitialPrices())
	brokerHandler := broker_handler.NewHandler(brokerStore, mktEngine)
	brokerMux := http.NewServeMux()
	brokerHandler.RegisterRoutes(brokerMux)
	c.mockBroker = httptest.NewServer(brokerMux)
	c.brokerURL = c.mockBroker.URL
}

func (c *Client) initMockBankServer() {
	bankStore := bank_ledger.NewLedgerStore()
	bankEng := bank_engine.NewSettlementEngine(bankStore, nil, time.Second)
	bankHandler := bank_handler.NewHandler(bankStore, bankEng)
	bankMux := http.NewServeMux()
	bankHandler.RegisterRoutes(bankMux)
	c.mockBank = httptest.NewServer(bankMux)
	c.bankURL = c.mockBank.URL
}

func (c *Client) initMockForgejoServer() {
	mock := NewMockForgejoServer()
	c.mockForgejo = httptest.NewServer(mock.Handler())
	c.forgejoURL = c.mockForgejo.URL
}

func registerFacilitiesRoutes(serverMux *http.ServeMux, facService facilities_svc.Service, opsService ops_svc.Service) {
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
}

var (
	mockGDPRCertsMu sync.RWMutex
	mockGDPRCerts   = make(map[string]*reporting.GDPRDeletionCertificate)
)

func (c *Client) registerCustomerRoutes(custMux, serverMux *http.ServeMux, bankClient *plaid.Client, brokerClient *alpaca.Client, ciService core_invest_svc.Service) {
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
			acc, err := brokerClient.GetAccount(r.Context(), portfolioID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			positions, _ := brokerClient.ListPositions(r.Context(), portfolioID)
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
			bars, _ := brokerClient.GetLatestBars(r.Context(), symbols)
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
		mux.HandleFunc("POST /api/v1/customer/auth/register", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				Name              string  `json:"name"`
				Email             string  `json:"email"`
				Password          string  `json:"password"`
				BankAccount       string  `json:"bank_account"`
				InitialDepositUSD float64 `json:"initial_deposit_usd"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			res, err := ciService.RegisterCustomer(r.Context(), core_invest_svc.CustomerRegistrationRequest{
				FullName:        req.Name,
				Email:           req.Email,
				Password:        req.Password,
				BankPublicToken: req.BankAccount,
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(res)
		})
		mux.HandleFunc("POST /api/v1/customer/account/close", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				CustomerID  string `json:"customer_id"`
				PortfolioID string `json:"portfolio_id"`
				Reason      string `json:"reason"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if req.CustomerID == "" {
				req.CustomerID = "018e0000-0000-7000-8000-000000000098"
			}
			if req.PortfolioID == "" {
				req.PortfolioID = "018f3a9a-2222-7000-8000-000000000003"
			}
			res, err := ciService.CloseAndLiquidateAccount(r.Context(), core_invest_svc.CloseAccountRequest{
				CustomerID:           req.CustomerID,
				PortfolioID:          req.PortfolioID,
				DestinationAccountID: "plaid-dest-acc-01",
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if c.mockCustomerSessionStore != nil {
				_, _ = c.mockCustomerSessionStore.PurgeCustomerSessions(req.CustomerID)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(res)
		})
		mux.HandleFunc("POST /api/v1/customer/privacy/gdpr-erasure", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				CustomerID   string `json:"customer_id"`
				Email        string `json:"email"`
				Confirmation string `json:"confirmation"`
				Reason       string `json:"reason"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if req.CustomerID == "" {
				req.CustomerID = "018e0000-0000-7000-8000-000000000098" // Chloe Spark
			}
			if req.Email == "" {
				req.Email = "chloe@oci.local"
			}
			cert, err := ciService.ExecuteGDPRErasure(r.Context(), req.CustomerID, req.Email)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			mockGDPRCertsMu.Lock()
			mockGDPRCerts[cert.CertificateID] = cert
			mockGDPRCertsMu.Unlock()

			if c.mockCustomerSessionStore != nil {
				_, _ = c.mockCustomerSessionStore.PurgeCustomerSessions(req.CustomerID)
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cert)
		})
		mux.HandleFunc("GET /api/v1/customer/privacy/gdpr-certificate/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			certID := r.PathValue("id")
			mockGDPRCertsMu.RLock()
			cert, ok := mockGDPRCerts[certID]
			mockGDPRCertsMu.RUnlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "certificate not found"})
				return
			}
			json.NewEncoder(w).Encode(cert)
		})
		mux.HandleFunc("GET /api/v1/customer/privacy/gdpr-certificate/{id}/download", func(w http.ResponseWriter, r *http.Request) {
			certID := r.PathValue("id")
			mockGDPRCertsMu.RLock()
			cert, ok := mockGDPRCerts[certID]
			mockGDPRCertsMu.RUnlock()
			if !ok {
				http.Error(w, "certificate not found", http.StatusNotFound)
				return
			}
			pdfBytes, err := reporting.GenerateGDPRCertificatePDF(cert)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gdpr-erasure-certificate-%s.pdf", cert.CertificateID))
			w.Write(pdfBytes)
		})
		mux.HandleFunc("GET /api/v1/customer/statements", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			custID := r.URL.Query().Get("customer_id")
			if custID == "" {
				custID = "018e0000-0000-7000-8000-000000000099" // Arthur Pendelton
			}
			items, err := ciService.ListStatements(r.Context(), custID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"items": items, "total": len(items)})
		})
		mux.HandleFunc("POST /api/v1/customer/statements/generate", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				CustomerID  string `json:"customer_id"`
				PortfolioID string `json:"portfolio_id"`
				Type        string `json:"type"`
				PeriodMonth int    `json:"period_month"`
				PeriodYear  int    `json:"period_year"`
				TaxYear     int    `json:"tax_year"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if req.CustomerID == "" {
				req.CustomerID = "018e0000-0000-7000-8000-000000000099"
			}
			if req.PortfolioID == "" {
				req.PortfolioID = "018f3a9a-2222-7000-8000-000000000002"
			}
			var stmt *core_invest_repo.InvestmentStatement
			var err error
			if req.Type == "1099_b" {
				yr := req.TaxYear
				if yr == 0 {
					yr = 2025
				}
				stmt, _, err = ciService.GenerateForm1099B(r.Context(), req.CustomerID, req.PortfolioID, yr)
			} else {
				m := req.PeriodMonth
				if m == 0 {
					m = 3
				}
				y := req.PeriodYear
				if y == 0 {
					y = 2026
				}
				stmt, _, err = ciService.GenerateMonthlyStatement(r.Context(), req.CustomerID, req.PortfolioID, y, m)
			}
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(stmt)
		})
		mux.HandleFunc("GET /api/v1/customer/statements/{id}/download", func(w http.ResponseWriter, r *http.Request) {
			stmtID := r.PathValue("id")
			custID := r.URL.Query().Get("customer_id")
			if custID == "" {
				custID = "018e0000-0000-7000-8000-000000000099"
			}
			stmt, pdfBytes, err := ciService.GetStatementDocument(r.Context(), custID, stmtID)
			if err != nil {
				http.Error(w, "statement not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.pdf", stmt.StatementType, stmt.Period))
			w.Write(pdfBytes)
		})
	}
	registerCustRoutes(custMux)
	registerCustRoutes(serverMux)
}

func generateMockCSRFToken(n int) string {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return "mock-csrf-token-default"
	}
	return hex.EncodeToString(b)
}

type mockAuthRevoker struct {
	store sessionvalkey.EmployeeSessionStore
}

func (r *mockAuthRevoker) RevokeEmployeeSessions(ctx context.Context, employeeID string) error {
	if r.store != nil {
		_, err := r.store.PurgeEmployeeSessions(employeeID)
		return err
	}
	return nil
}

func (r *mockAuthRevoker) RevokeOAuthTokens(ctx context.Context, employeeID string) error {
	return nil
}

func (c *Client) registerEmployeeAuthRoutes(bffMux *http.ServeMux) {
	bffMux.HandleFunc("POST /api/v1/employee/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			EmployeeID string `json:"employee_id"`
			Email      string `json:"email"`
			Name       string `json:"name"`
			Role       string `json:"role"`
			Department string `json:"department"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		if req.EmployeeID == "" {
			req.EmployeeID = "emp-test"
		}
		if req.Role == "" {
			req.Role = "staff"
		}
		sess, err := c.mockEmployeeSessionStore.Create(req.EmployeeID, req.Email, req.Name, req.Role, req.Department)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "employee_session",
			Value:    sess.SessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  sess.ExpiresAt,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-employee-session",
			Value:    sess.SessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  sess.ExpiresAt,
		})

		csrfToken := generateMockCSRFToken(16)
		http.SetCookie(w, &http.Cookie{
			Name:     "employee_csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
		w.Header().Set("X-CSRF-Token", csrfToken)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "authenticated",
			"session":    sess,
			"csrf_token": csrfToken,
		})
	})

	bffMux.HandleFunc("POST /api/v1/employee/auth/forgejo/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		empID := fmt.Sprintf("emp-%s", req.Username)
		sess, err := c.mockEmployeeSessionStore.Create(empID, fmt.Sprintf("%s@oci.local", req.Username), req.Username, "engineering", "Engineering")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "employee_session",
			Value:    sess.SessionID,
			Path:     "/",
			HttpOnly: true,
			Expires:  sess.ExpiresAt,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-employee-session",
			Value:    sess.SessionID,
			Path:     "/",
			HttpOnly: true,
			Expires:  sess.ExpiresAt,
		})
		csrfToken := generateMockCSRFToken(16)
		http.SetCookie(w, &http.Cookie{
			Name:     "employee_csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
		})
		w.Header().Set("X-CSRF-Token", csrfToken)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "authenticated",
			"session":    sess,
			"csrf_token": csrfToken,
		})
	})

	bffMux.HandleFunc("GET /api/v1/employee/csrf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		csrfToken := ""
		if ck, err := r.Cookie("employee_csrf_token"); err == nil && ck.Value != "" {
			csrfToken = ck.Value
		} else {
			csrfToken = generateMockCSRFToken(16)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "employee_csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
		})
		w.Header().Set("X-CSRF-Token", csrfToken)
		_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": csrfToken})
	})

	bffMux.HandleFunc("POST /api/v1/employee/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var token string
		if ck, err := r.Cookie("__Host-employee-session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if ck, err := r.Cookie("employee_session"); err == nil && ck.Value != "" {
			token = ck.Value
		}
		if token != "" && c.mockEmployeeSessionStore != nil {
			c.mockEmployeeSessionStore.Delete(token)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "employee_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-employee-session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
	})

	bffMux.HandleFunc("GET /api/v1/employee/auth/session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var token string
		if ck, err := r.Cookie("__Host-employee-session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if ck, err := r.Cookie("employee_session"); err == nil && ck.Value != "" {
			token = ck.Value
		}
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		if c.mockEmployeeSessionStore != nil {
			sess, ok := c.mockEmployeeSessionStore.Get(token)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "session expired"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"session": sess})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
}

func (c *Client) employeeAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/employee/auth/login" ||
			path == "/api/v1/employee/auth/forgejo/login" ||
			path == "/api/v1/employee/csrf" ||
			path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		var token string
		if ck, err := r.Cookie("__Host-employee-session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if ck, err := r.Cookie("employee_session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if authHdr := r.Header.Get("Authorization"); strings.HasPrefix(authHdr, "Bearer ") {
			token = strings.TrimPrefix(authHdr, "Bearer ")
		}

		if token != "" {
			if c.mockEmployeeSessionStore != nil {
				if _, ok := c.mockEmployeeSessionStore.Get(token); !ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "unauthorized: invalid or expired employee session",
					})
					return
				}
			}

			// Validate Double-Submit CSRF on mutating requests
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
				if path != "/api/v1/employee/auth/logout" {
					headerToken := r.Header.Get("X-CSRF-Token")
					var cookieToken string
					if ck, err := r.Cookie("employee_csrf_token"); err == nil && ck.Value != "" {
						cookieToken = ck.Value
					} else if ck, err := r.Cookie("__Host-employee-csrf-token"); err == nil && ck.Value != "" {
						cookieToken = ck.Value
					}

					if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusForbidden)
						_ = json.NewEncoder(w).Encode(map[string]string{
							"error": "forbidden: invalid or missing CSRF token",
						})
						return
					}
				}
			}
		} else if c.requireAuth {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized: valid employee session required",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *Client) registerCustomerAuthRoutes(custMux *http.ServeMux) {
	custMux.HandleFunc("POST /api/v1/customer/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Persona    string `json:"persona"`
			CustomerID string `json:"customer_id"`
			Name       string `json:"name"`
			Email      string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		custID := req.CustomerID
		if custID == "" && req.Persona != "" {
			custID = fmt.Sprintf("cust-%s", req.Persona)
		}
		if custID == "" {
			custID = "cust-anonymous"
		}
		name := req.Name
		if name == "" {
			name = custID
		}
		email := req.Email
		if email == "" {
			email = fmt.Sprintf("%s@oci.local", custID)
		}

		sess, err := c.mockCustomerSessionStore.Create(custID, name, email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "customer_session",
			Value:    sess.SessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  sess.ExpiresAt,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-customer-session",
			Value:    sess.SessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  sess.ExpiresAt,
		})

		csrfToken := generateMockCSRFToken(16)
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
		w.Header().Set("X-CSRF-Token", csrfToken)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "authenticated",
			"session":    sess,
			"csrf_token": csrfToken,
		})
	})

	custMux.HandleFunc("GET /api/v1/customer/csrf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		csrfToken := ""
		if ck, err := r.Cookie("csrf_token"); err == nil && ck.Value != "" {
			csrfToken = ck.Value
		} else {
			csrfToken = generateMockCSRFToken(16)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
		})
		w.Header().Set("X-CSRF-Token", csrfToken)
		_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": csrfToken})
	})

	custMux.HandleFunc("POST /api/v1/customer/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var token string
		if ck, err := r.Cookie("__Host-customer-session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if ck, err := r.Cookie("customer_session"); err == nil && ck.Value != "" {
			token = ck.Value
		}
		if token != "" && c.mockCustomerSessionStore != nil {
			c.mockCustomerSessionStore.Delete(token)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "customer_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-customer-session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
	})
}

func (c *Client) customerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/customer/auth/login" ||
			path == "/api/v1/customer/auth/register" ||
			path == "/api/v1/customer/csrf" ||
			path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		var token string
		if ck, err := r.Cookie("__Host-customer-session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if ck, err := r.Cookie("customer_session"); err == nil && ck.Value != "" {
			token = ck.Value
		} else if authHdr := r.Header.Get("Authorization"); strings.HasPrefix(authHdr, "Bearer ") {
			token = strings.TrimPrefix(authHdr, "Bearer ")
		}

		if token != "" {
			if c.mockCustomerSessionStore != nil {
				if _, ok := c.mockCustomerSessionStore.Get(token); !ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "unauthorized: invalid or expired customer session",
					})
					return
				}
			}

			// Validate Double-Submit CSRF on mutating requests
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
				if path != "/api/v1/customer/auth/logout" {
					headerToken := r.Header.Get("X-CSRF-Token")
					var cookieToken string
					if ck, err := r.Cookie("csrf_token"); err == nil && ck.Value != "" {
						cookieToken = ck.Value
					} else if ck, err := r.Cookie("__Host-csrf-token"); err == nil && ck.Value != "" {
						cookieToken = ck.Value
					}

					if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusForbidden)
						_ = json.NewEncoder(w).Encode(map[string]string{
							"error": "forbidden: invalid or missing CSRF token",
						})
						return
					}
				}
			}
		} else if c.requireAuth {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized: valid customer session required",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *Client) initMockEmployeeBFFServer(wfHandler *handler.WorkforceHandler, opsHandler *ops_handler.Handler) {
	bffMux := http.NewServeMux()
	wfHandler.RegisterRoutes(bffMux)
	if opsHandler != nil {
		opsHandler.RegisterRoutes(bffMux)
	}

	c.registerEmployeeAuthRoutes(bffMux)
	handler := c.employeeAuthMiddleware(bffMux)

	c.mockBFF = httptest.NewServer(handler)
	c.bffURL = c.mockBFF.URL
}

func (c *Client) initMockServers() error {
	c.initMockBrokerServer()
	c.initMockBankServer()
	c.initMockForgejoServer()

	c.mockEmployeeSessionStore = sessionvalkey.NewMemoryEmployeeSessionStore()
	c.mockCustomerSessionStore = sessionvalkey.NewMemoryCustomerSessionStore()

	// Initialize external clients
	brokerClient := alpaca.NewClient(alpaca.Config{
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
	ciService := core_invest_svc.NewServiceWithIntegrations(ciRepo, brokerClient, bankClient)
	registerMockCoreInvestRoutes(serverMux, ciService)

	opsRepo := ops_repo.NewMockRepository()
	opsService := ops_svc.NewService(opsRepo)
	opsHandler := ops_handler.NewHandler(opsService)
	opsHandler.RegisterRoutes(serverMux)

	facRepo := facilities_repo.NewMockRepository()
	facService := facilities_svc.NewService(facRepo)

	registerFacilitiesRoutes(serverMux, facService, opsService)

	wfRepo := workforce_repo.NewMockRepository()
	wfService := workforce_svc.NewService(wfRepo)
	onboardingSaga := saga.NewOnboardingSaga(wfService, nil, nil)
	authRevoker := &mockAuthRevoker{store: c.mockEmployeeSessionStore}
	offboardingEng := saga.NewOffboardingEngineWithIntegrations(wfService, nil, nil, nil, nil, authRevoker, nil, nil)
	wfHandler := handler.NewWorkforceHandler(wfService, onboardingSaga, offboardingEng)
	wfHandler.RegisterRoutes(serverMux)

	// Customer BFF routes and Mock Customer BFF server
	custMux := http.NewServeMux()
	c.registerCustomerRoutes(custMux, serverMux, bankClient, brokerClient, ciService)
	opsHandler.RegisterRoutes(custMux)

	c.registerCustomerAuthRoutes(custMux)
	custHandler := c.customerAuthMiddleware(custMux)

	c.mockCustBFF = httptest.NewServer(custHandler)
	c.custBFFURL = c.mockCustBFF.URL

	c.mockServer = httptest.NewServer(serverMux)
	c.serverURL = c.mockServer.URL

	c.initMockEmployeeBFFServer(wfHandler, opsHandler)

	return nil
}

// GetCookieJar returns the persona's cookie jar, creating one if not present.
func (c *Client) GetCookieJar(personaID string) http.CookieJar {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaJars == nil {
		c.personaJars = make(map[string]http.CookieJar)
	}
	jar, ok := c.personaJars[personaID]
	if !ok {
		jar, _ = cookiejar.New(nil)
		c.personaJars[personaID] = jar
	}
	return jar
}

// SetCookieJar associates a cookie jar with a specific persona.
func (c *Client) SetCookieJar(personaID string, jar http.CookieJar) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaJars == nil {
		c.personaJars = make(map[string]http.CookieJar)
	}
	c.personaJars[personaID] = jar
}

// GetCSRFToken returns the active cached CSRF token for a persona.
func (c *Client) GetCSRFToken(personaID string) string {
	c.jarsMu.RLock()
	defer c.jarsMu.RUnlock()
	if c.personaCSRF != nil {
		return c.personaCSRF[personaID]
	}
	return ""
}

// SetCSRFToken stores the active CSRF token for a persona.
func (c *Client) SetCSRFToken(personaID, token string) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaCSRF == nil {
		c.personaCSRF = make(map[string]string)
	}
	c.personaCSRF[personaID] = token
}

// ResetSession clears the persona's cookie jar and CSRF token cache.
func (c *Client) ResetSession(personaID string) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaJars != nil {
		jar, _ := cookiejar.New(nil)
		c.personaJars[personaID] = jar
	}
	if c.personaCSRF != nil {
		delete(c.personaCSRF, personaID)
	}
}

// SetRequireAuth configures whether the mock BFF requires authentication for all protected routes.
func (c *Client) SetRequireAuth(require bool) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	c.requireAuth = require
}

// IsRequireAuth returns true if the mock BFF enforces authentication strictly.
func (c *Client) IsRequireAuth() bool {
	c.jarsMu.RLock()
	defer c.jarsMu.RUnlock()
	return c.requireAuth
}

// SetActivePersona binds the active persona ID on the client.
func (c *Client) SetActivePersona(p *Persona) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if p != nil {
		c.activePersonaID = p.ID
	} else {
		c.activePersonaID = ""
	}
}

// WithPersona sets the active persona and returns the client instance.
func (c *Client) WithPersona(p *Persona) *Client {
	c.SetActivePersona(p)
	return c
}

// MockEmployeeSessionStore returns the internal mock employee session store.
func (c *Client) MockEmployeeSessionStore() sessionvalkey.EmployeeSessionStore {
	return c.mockEmployeeSessionStore
}

// MockCustomerSessionStore returns the internal mock customer session store.
func (c *Client) MockCustomerSessionStore() sessionvalkey.CustomerSessionStore {
	return c.mockCustomerSessionStore
}

// AuthenticatePersona executes the automated authentication handshake for the specified persona.
func (c *Client) AuthenticatePersona(ctx context.Context, p *Persona) error {
	if p == nil {
		return errors.New("cannot authenticate nil persona")
	}
	personaCtx := WithPersona(ctx, p)
	if p.Type == "customer" || strings.HasPrefix(p.ID, "cust-") {
		return c.authenticateCustomer(personaCtx, p)
	}
	return c.authenticateEmployee(personaCtx, p)
}

func (c *Client) authenticateCustomer(ctx context.Context, p *Persona) error {
	email := p.AuthEmail()
	loginPayload := map[string]string{
		"customer_id": p.ID,
		"name":        p.Name,
		"email":       email,
		"persona":     p.ID,
	}

	endpoint := "/api/v1/customer/auth/login"
	resp, respBytes, err := c.DoCustomer(ctx, "POST", endpoint, loginPayload)
	if err != nil {
		return fmt.Errorf("customer authentication handshake failed for %s (%s): %w", p.Name, p.ID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("customer authentication rejected for %s (%s): status %d body %s", p.Name, p.ID, resp.StatusCode, string(respBytes))
	}

	var res struct {
		Status    string `json:"status"`
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(respBytes, &res); err == nil && res.CSRFToken != "" {
		c.SetCSRFToken(p.ID, res.CSRFToken)
	}
	return nil
}

func (c *Client) authenticateEmployee(ctx context.Context, p *Persona) error {
	email := p.AuthEmail()
	role := p.AuthRole()

	loginPayload := map[string]string{
		"employee_id": p.ID,
		"name":        p.Name,
		"email":       email,
		"role":        role,
		"department":  p.Department,
	}

	endpoint := "/api/v1/employee/auth/login"
	resp, respBytes, err := c.Do(ctx, true, "POST", endpoint, loginPayload)
	if err == nil && resp.StatusCode == http.StatusOK {
		var res struct {
			Status    string `json:"status"`
			CSRFToken string `json:"csrf_token"`
		}
		if err := json.Unmarshal(respBytes, &res); err == nil && res.CSRFToken != "" {
			c.SetCSRFToken(p.ID, res.CSRFToken)
		}
		return nil
	}

	// In production mode, self-asserted login is disabled, so try Forgejo login
	if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound) {
		forgejoUsername := strings.ToLower(strings.Split(p.Name, " ")[0])
		forgejoPayload := map[string]string{
			"username": forgejoUsername,
			"password": "password123",
		}
		fResp, fBytes, fErr := c.Do(ctx, true, "POST", "/api/v1/employee/auth/forgejo/login", forgejoPayload)
		if fErr == nil && fResp.StatusCode == http.StatusOK {
			var res struct {
				Status    string `json:"status"`
				CSRFToken string `json:"csrf_token"`
			}
			if err := json.Unmarshal(fBytes, &res); err == nil && res.CSRFToken != "" {
				c.SetCSRFToken(p.ID, res.CSRFToken)
			}
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("employee authentication handshake failed for %s (%s): %w", p.Name, p.ID, err)
	}
	return fmt.Errorf("employee authentication rejected for %s (%s): status %d body %s", p.Name, p.ID, resp.StatusCode, string(respBytes))
}

// LogoutPersona logs out the specified persona from their active BFF session.
func (c *Client) LogoutPersona(ctx context.Context, p *Persona) error {
	if p == nil {
		return errors.New("cannot logout nil persona")
	}
	personaCtx := WithPersona(ctx, p)
	if p.Type == "customer" || strings.HasPrefix(p.ID, "cust-") {
		endpoint := "/api/v1/customer/auth/logout"
		resp, _, err := c.DoCustomer(personaCtx, "POST", endpoint, nil)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("customer logout failed: status %d", resp.StatusCode)
		}
		return nil
	}

	endpoint := "/api/v1/employee/auth/logout"
	resp, _, err := c.Do(personaCtx, true, "POST", endpoint, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("employee logout failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) doRequest(ctx context.Context, fullURL, method string, body any, headers map[string]string) (*http.Response, []byte, error) {
	var personaID string
	if p := PersonaFromContext(ctx); p != nil {
		personaID = p.ID
	} else if c.activePersonaID != "" {
		personaID = c.activePersonaID
	}

	jar := c.GetCookieJar(personaID)

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

	// Transparent Double-Submit CSRF Header Injection on mutating requests
	if (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete) && req.Header.Get("X-CSRF-Token") == "" {
		var csrfToken string
		if parsedURL, parseErr := url.Parse(fullURL); parseErr == nil && jar != nil {
			cookies := jar.Cookies(parsedURL)
			for _, ck := range cookies {
				if ck.Name == "csrf_token" || ck.Name == "employee_csrf_token" ||
					ck.Name == "__Host-csrf-token" || ck.Name == "__Host-employee-csrf-token" {
					csrfToken = ck.Value
					break
				}
			}
		}
		if csrfToken == "" && personaID != "" {
			csrfToken = c.GetCSRFToken(personaID)
		}
		if csrfToken != "" {
			req.Header.Set("X-CSRF-Token", csrfToken)

			if parsedURL, parseErr := url.Parse(fullURL); parseErr == nil && jar != nil {
				hasCookie := false
				for _, ck := range jar.Cookies(parsedURL) {
					if ck.Name == "csrf_token" || ck.Name == "employee_csrf_token" {
						hasCookie = true
						break
					}
				}
				if !hasCookie {
					cookieName := "employee_csrf_token"
					if strings.Contains(fullURL, "customer") {
						cookieName = "csrf_token"
					}
					jar.SetCookies(parsedURL, []*http.Cookie{
						{
							Name:  cookieName,
							Value: csrfToken,
							Path:  "/",
						},
					})
				}
			}
		}
	}

	tr := c.transport
	if tr == nil && c.httpClient != nil {
		tr = c.httpClient.Transport
	}
	if tr == nil {
		tr = http.DefaultTransport
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
		Jar:       jar,
	}
	if c.httpClient != nil && c.httpClient.CheckRedirect != nil {
		httpClient.CheckRedirect = c.httpClient.CheckRedirect
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed (%s %s): %w", method, fullURL, err)
	}
	defer resp.Body.Close()

	if token := resp.Header.Get("X-CSRF-Token"); token != "" && personaID != "" {
		c.SetCSRFToken(personaID, token)
	}
	for _, ck := range resp.Cookies() {
		if (ck.Name == "csrf_token" || ck.Name == "employee_csrf_token") && ck.Value != "" && personaID != "" {
			c.SetCSRFToken(personaID, ck.Value)
		}
	}

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

// DoForgejo executes an HTTP request directly against the Forgejo REST API.
func (c *Client) DoForgejo(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.forgejoURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	headers := map[string]string{
		"Authorization": "token mock-forgejo-token",
	}
	return c.doRequest(ctx, baseURL+path, method, body, headers)
}

