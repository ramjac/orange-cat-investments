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
)

// Client wraps HTTP communication with the OCI stack (Domain API server and Employee BFF).
type Client struct {
	serverURL string
	bffURL    string
	mode      string
	httpClient *http.Client

	// Mock server references when running in "mock" mode
	mockServer *httptest.Server
	mockBFF    *httptest.Server
}

// NewClient constructs a new simulation client. In "mock" mode, it spins up in-memory HTTP test servers.
func NewClient(cfg *Config) (*Client, error) {
	c := &Client{
		serverURL:  strings.TrimSuffix(cfg.ServerURL, "/"),
		bffURL:     strings.TrimSuffix(cfg.EmployeeBFFURL, "/"),
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
}

func (c *Client) initMockServers() error {
	// Initialize Domain API Server Mux
	serverMux := http.NewServeMux()

	ciRepo := core_invest_repo.NewMockRepository()
	ciService := core_invest_svc.NewService(ciRepo)
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

// Request executes an HTTP request against either the Server or Employee BFF target URL.
func (c *Client) Do(ctx context.Context, useBFF bool, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.serverURL
	if useBFF {
		baseURL = c.bffURL
	}

	url := baseURL + path
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal JSON request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed (%s %s): %w", method, url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	return resp, respBytes, nil
}
