package customerbff

import (
	"context"
	"log/slog"
	"os"

	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
	"github.com/orange-cat-investments/oci/pkg/session"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

type CustomerSession = sessionvalkey.CustomerSession
type SessionStore = sessionvalkey.CustomerSessionStore

// CustomerBFF is the presentation proxy layer for customer web and mobile frontends.
// Invariant: CustomerBFF never connects directly to database schemas (pgxpool) or Domain APIs.
// All business domain logic and persistence are delegated to Customer BLL over gRPC.
type CustomerBFF struct {
	bllClient    customerbllv1.CustomerBLLServiceClient
	sessions     SessionStore
	cache        cachevalkey.Cache
	rateLimiter  *cachevalkey.RateLimiter
	logger       *slog.Logger
	searchClient *meilisearch.Client
}

// NewCustomerBFF initializes a new CustomerBFF instance.
// In production code, bllClient is required and cannot be nil.
func NewCustomerBFF(bllClient customerbllv1.CustomerBLLServiceClient, logger *slog.Logger) *CustomerBFF {
	if logger == nil {
		logger = slog.Default()
	}
	if bllClient == nil {
		panic("customerbff: bllClient is required and cannot be nil")
	}

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	valkeyPass := os.Getenv("VALKEY_PASSWORD")

	meiliURL := os.Getenv("MEILISEARCH_URL")
	if meiliURL == "" {
		meiliURL = "http://localhost:7700"
	}
	meiliKey := os.Getenv("MEILI_MASTER_KEY")
	sClient := meilisearch.NewClient(meiliURL, meiliKey)

	var sess SessionStore
	if valkeyAddr != "" {
		sess = sessionvalkey.NewCustomerSessionStore(valkeyAddr, valkeyPass, 0, logger)
	} else {
		sess = sessionvalkey.NewMemoryCustomerSessionStore()
	}

	return &CustomerBFF{
		bllClient:    bllClient,
		sessions:     sess,
		cache:        cachevalkey.NewCache(valkeyAddr, valkeyPass, 0, logger),
		rateLimiter:  cachevalkey.NewRateLimiterWithAddr(valkeyAddr, valkeyPass, 0, logger),
		logger:       logger,
		searchClient: sClient,
	}
}

func (b *CustomerBFF) SetBLLClient(client customerbllv1.CustomerBLLServiceClient) {
	b.bllClient = client
}

func (b *CustomerBFF) BLLClient() customerbllv1.CustomerBLLServiceClient {
	return b.bllClient
}

func (b *CustomerBFF) SetSessionStore(store SessionStore) {
	b.sessions = store
}

func (b *CustomerBFF) SetCache(c cachevalkey.Cache) {
	b.cache = c
}

func (b *CustomerBFF) SetRateLimiter(rl *cachevalkey.RateLimiter) {
	b.rateLimiter = rl
}

func (b *CustomerBFF) SetSearchClient(client *meilisearch.Client) {
	b.searchClient = client
}

func (b *CustomerBFF) SearchClient() *meilisearch.Client {
	return b.searchClient
}

func CustomerFromContext(ctx context.Context) *CustomerSession {
	return session.CustomerFromContext(ctx)
}
