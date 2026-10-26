package employeebff

import (
	"log/slog"

	"github.com/orange-cat-investments/oci/internal/handler"
	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

// EmployeeBFF is the presentation proxy layer for corporate employee, caretaker, and engineering portals.
// Invariant: EmployeeBFF never connects directly to database schemas (pgxpool) or Domain APIs.
// All business domain logic and persistence are delegated to Internal BLL over gRPC.
type EmployeeBFF struct {
	bllClient             internalbllv1.InternalBLLServiceClient
	sessionStore          SessionStore
	cache                 cachevalkey.Cache
	rateLimiter           *cachevalkey.RateLimiter
	searchClient          *meilisearch.Client
	logger                *slog.Logger
	incidentBroker        *IncidentBroker
	authHandler           *AuthHandler
	wfHandler             *handler.WorkforceHandler
	opsHandler            *opshandler.Handler
	issueTrackerURL       string
	isProd                bool
	enableFelineWorkforce bool
}

// NewEmployeeBFF creates an EmployeeBFF instance with default in-memory fallbacks.
func NewEmployeeBFF(bllClient internalbllv1.InternalBLLServiceClient, logger *slog.Logger) *EmployeeBFF {
	if logger == nil {
		logger = slog.Default()
	}

	b := &EmployeeBFF{
		bllClient:             bllClient,
		sessionStore:          sessionvalkey.NewMemoryEmployeeSessionStore(),
		logger:                logger,
		incidentBroker:        NewIncidentBroker(logger),
		enableFelineWorkforce: true,
	}

	b.authHandler = NewAuthHandler(b.sessionStore, "", false)
	b.wfHandler = handler.NewWorkforceHandlerWithBLL(bllClient)
	b.wfHandler.EnableFelineWorkforce = true
	b.opsHandler = opshandler.NewHandlerWithBLL(bllClient)

	return b
}

func (b *EmployeeBFF) SetSessionStore(store SessionStore) {
	b.sessionStore = store
	if b.authHandler != nil {
		b.authHandler.store = store
	}
}

func (b *EmployeeBFF) SetCache(c cachevalkey.Cache) {
	b.cache = c
	if b.wfHandler != nil {
		b.wfHandler.SetCache(c)
	}
}

func (b *EmployeeBFF) SetRateLimiter(r *cachevalkey.RateLimiter) {
	b.rateLimiter = r
}

func (b *EmployeeBFF) SetSearchClient(s *meilisearch.Client) {
	b.searchClient = s
}

func (b *EmployeeBFF) SetIssueTrackerURL(url string) {
	b.issueTrackerURL = url
	if b.authHandler != nil {
		b.authHandler.forgejoURL = url
	}
}

func (b *EmployeeBFF) SetIsProd(isProd bool) {
	b.isProd = isProd
	if b.authHandler != nil {
		b.authHandler.isProd = isProd
	}
}

func (b *EmployeeBFF) SetEnableFelineWorkforce(enable bool) {
	b.enableFelineWorkforce = enable
	if b.wfHandler != nil {
		b.wfHandler.EnableFelineWorkforce = enable
	}
}

func (b *EmployeeBFF) SetWorkforceHandler(h *handler.WorkforceHandler) {
	b.wfHandler = h
}

func (b *EmployeeBFF) SetOpsHandler(h *opshandler.Handler) {
	b.opsHandler = h
}

func (b *EmployeeBFF) WorkforceHandler() *handler.WorkforceHandler {
	return b.wfHandler
}

func (b *EmployeeBFF) OpsHandler() *opshandler.Handler {
	return b.opsHandler
}

func (b *EmployeeBFF) IncidentBroker() *IncidentBroker {
	return b.incidentBroker
}

func (b *EmployeeBFF) SessionStore() SessionStore {
	return b.sessionStore
}

func (b *EmployeeBFF) BLLClient() internalbllv1.InternalBLLServiceClient {
	return b.bllClient
}
