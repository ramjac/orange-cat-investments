package valkey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// TTL Constants
const (
	TTLPortfolioValuation = 30 * time.Second
	TTLMarketQuotes       = 5 * time.Second
	TTLHabitatTelemetry   = 5 * time.Second
	TTLESGMetrics         = 15 * time.Minute
	TTLCareSchedule       = 2 * time.Minute
	TTLCOTSMetadata       = 60 * time.Second
	TTLActiveIncidents    = 30 * time.Second
)

// Cache Key Helpers
func KeyPortfolio(customerID string) string {
	return "cache:portfolio:" + customerID
}

func KeyPortfolioWithID(customerID, portfolioID string) string {
	if portfolioID == "" || portfolioID == "port-"+customerID {
		return KeyPortfolio(customerID)
	}
	return "cache:portfolio:" + customerID + ":" + portfolioID
}

func KeyOrders(customerID string) string {
	return "cache:orders:" + customerID
}

func KeyQuotes(symbol string) string {
	return "cache:quotes:" + strings.ToUpper(strings.TrimSpace(symbol))
}

func KeyHabitatTelemetry() string {
	return "cache:habitat:telemetry"
}

func KeyESGMetrics() string {
	return "cache:esg:metrics"
}

func KeyCareSchedule(felineID string) string {
	return "cache:care_schedule:" + felineID
}

func KeySnipeITAssets() string {
	return "cache:cots:snipeit:assets"
}

func KeyForgejoRepos() string {
	return "cache:cots:forgejo:repos"
}

func KeyFrappeLeavePolicies() string {
	return "cache:cots:frappe:leave_policies"
}

func KeyActiveIncidents() string {
	return "cache:ops:active_incidents"
}

// Cache interface defines general-purpose distributed caching operations.
type Cache interface {
	Get(ctx context.Context, key string, dest any) (bool, error)
	GetRaw(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val any, ttl time.Duration) error
	SetRaw(ctx context.Context, key string, data []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	DeletePrefix(ctx context.Context, prefix string) error
	Close() error
}

// --- In-Memory Cache Implementation ---

type memoryCacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type memoryCache struct {
	mu      sync.RWMutex
	entries map[string]memoryCacheEntry
}

// NewMemoryCache creates an in-memory cache with expiration support.
func NewMemoryCache() Cache {
	return &memoryCache{
		entries: make(map[string]memoryCacheEntry),
	}
}

func (m *memoryCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if !ok {
		return false, nil
	}

	if time.Now().After(entry.expiresAt) {
		m.mu.Lock()
		delete(m.entries, key)
		m.mu.Unlock()
		return false, nil
	}

	if err := json.Unmarshal(entry.data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached data for key %s: %w", key, err)
	}

	return true, nil
}

func (m *memoryCache) GetRaw(ctx context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if !ok {
		return nil, false, nil
	}

	if time.Now().After(entry.expiresAt) {
		m.mu.Lock()
		delete(m.entries, key)
		m.mu.Unlock()
		return nil, false, nil
	}

	return entry.data, true, nil
}

func (m *memoryCache) SetRaw(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = memoryCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (m *memoryCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
	}
	return m.SetRaw(ctx, key, data, ttl)
}

func (m *memoryCache) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.entries, k)
	}
	return nil
}

func (m *memoryCache) DeletePrefix(ctx context.Context, prefix string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.entries {
		if strings.HasPrefix(k, prefix) {
			delete(m.entries, k)
		}
	}
	return nil
}

func (m *memoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]memoryCacheEntry)
	return nil
}

// --- Valkey Cache Implementation ---

type valkeyCache struct {
	client *redis.Client
	logger *slog.Logger
}

// NewValkeyCache creates a Valkey-backed cache client.
func NewValkeyCache(client *redis.Client, logger *slog.Logger) Cache {
	if logger == nil {
		logger = slog.Default()
	}
	return &valkeyCache{
		client: client,
		logger: logger,
	}
}

func (v *valkeyCache) GetRaw(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := v.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("valkey get error for key %s: %w", key, err)
	}
	return data, true, nil
}

func (v *valkeyCache) SetRaw(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if err := v.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("valkey set error for key %s: %w", key, err)
	}
	return nil
}

func (v *valkeyCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	val, err := v.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("valkey get error for key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal valkey data for key %s: %w", key, err)
	}

	return true, nil
}

func (v *valkeyCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
	}
	return v.SetRaw(ctx, key, data, ttl)
}

func (v *valkeyCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return v.client.Del(ctx, keys...).Err()
}

func (v *valkeyCache) DeletePrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	match := prefix + "*"
	for {
		keys, nextCursor, err := v.client.Scan(ctx, cursor, match, 100).Result()
		if err != nil {
			return fmt.Errorf("valkey scan error for prefix %s: %w", prefix, err)
		}
		if len(keys) > 0 {
			if err := v.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("valkey del error for scanned prefix %s: %w", prefix, err)
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return nil
}

func (v *valkeyCache) Close() error {
	return v.client.Close()
}

// NewCache initializes a Cache client. If addr is empty or unpingable, falls back to in-memory cache.
func NewCache(addr, password string, db int, logger *slog.Logger) Cache {
	if logger == nil {
		logger = slog.Default()
	}

	if addr == "" {
		logger.Info("VALKEY_ADDR not specified, using in-memory query cache")
		return NewMemoryCache()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to ping Valkey at "+addr+", falling back to in-memory query cache", "error", err)
		_ = client.Close()
		return NewMemoryCache()
	}

	logger.Info("connected to distributed Valkey query cache", "addr", addr)
	return NewValkeyCache(client, logger)
}

// --- HTTP Conditional Validation (ETag & 304 Not Modified) ---

// ComputeETag returns a strong HTTP ETag based on SHA-256 hash of response data.
func ComputeETag(data []byte) string {
	h := sha256.Sum256(data)
	return `"` + hex.EncodeToString(h[:16]) + `"`
}

// HandleConditionalETag checks the request's If-None-Match header against the computed ETag.
// If matched, it writes a 304 Not Modified header and returns true (indicating response handled).
// If not matched, it sets the ETag response header and returns false.
func HandleConditionalETag(w http.ResponseWriter, r *http.Request, data []byte) bool {
	etag := ComputeETag(data)
	clientETag := r.Header.Get("If-None-Match")

	// Check if client sent matching ETag (supporting both strong and weak formatting)
	if clientETag != "" && (clientETag == etag || clientETag == "W/"+etag || strings.TrimPrefix(clientETag, "W/") == etag) {
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return true
	}

	w.Header().Set("ETag", etag)
	return false
}
