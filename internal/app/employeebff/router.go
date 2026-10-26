package employeebff

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

// SetupRouter creates and configures the production chi.Router for Employee BFF.
func (b *EmployeeBFF) SetupRouter() http.Handler {
	mux := http.NewServeMux()
	b.RegisterRoutes(mux)

	r := chi.NewRouter()
	r.Use(versioning.Middleware(versioning.DefaultConfig()))
	r.Mount("/", mux)

	return r
}

// RegisterRoutes registers all Employee BFF HTTP endpoints on the provided ServeMux.
func (b *EmployeeBFF) RegisterRoutes(mux *http.ServeMux) {
	// 1. Health check endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// 2. Auth and CSRF endpoints
	if b.authHandler != nil {
		b.authHandler.RegisterAuthRoutes(mux)
	}

	// 3. Workforce domain routes
	if b.wfHandler != nil {
		b.wfHandler.RegisterRoutes(mux)
	}

	// 4. Operations domain routes
	if b.opsHandler != nil {
		RegisterProtectedOpsRoutes(mux, b.opsHandler, b.sessionStore, b.rateLimiter)
	}

	// 5. Incident SSE Stream & Broadcast Hub
	if b.incidentBroker != nil {
		sessionAuth := SessionAuthMiddleware(b.sessionStore)
		serviceAuth := InternalServiceAuthMiddleware()

		mux.Handle("GET /api/v1/employee/incidents/stream", sessionAuth(http.HandlerFunc(b.incidentBroker.HandleStream)))
		mux.Handle("GET /employee/incidents/stream", sessionAuth(http.HandlerFunc(b.incidentBroker.HandleStream)))
		mux.Handle("POST /api/v1/employee/incidents/broadcast", serviceAuth(http.HandlerFunc(b.incidentBroker.HandleBroadcast)))
		mux.Handle("POST /employee/incidents/broadcast", serviceAuth(http.HandlerFunc(b.incidentBroker.HandleBroadcast)))
	}

	// 6. Search API
	sessionAuth := SessionAuthMiddleware(b.sessionStore)
	searchHandler := func(w http.ResponseWriter, r *http.Request) {
		if b.searchClient == nil {
			http.Error(w, `{"error":"search service unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		q := r.URL.Query().Get("q")
		category := r.URL.Query().Get("category")
		filter := ""
		if category != "" {
			filter = fmt.Sprintf("category = '%s'", category)
		}
		res, err := b.searchClient.Search(r.Context(), "employee_knowledge", q, meilisearch.SearchOptions{
			Filter: filter,
			Limit:  25,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"search failed: %v"}`, err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}

	mux.Handle("GET /api/v1/employee/search", sessionAuth(http.HandlerFunc(searchHandler)))
	mux.Handle("GET /employee/search", sessionAuth(http.HandlerFunc(searchHandler)))
}
