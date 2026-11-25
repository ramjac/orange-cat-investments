package valkey

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultOnceTTL is the retention window for settlement-deduplication keys.
// ACH retries are bounded well below this horizon, so a retained claim still
// represents an in-progress or completed settlement when a redelivery arrives.
const DefaultOnceTTL = 30 * 24 * time.Hour

// OnceStore provides durable once-only key claims (e.g. ACH settlement
// deduplication) backed by shared Valkey SET NX EX. When a Valkey client is
// configured, Claim fails closed (returns an error) if the shared store is
// unavailable so callers can never fall back to process-local state; the
// in-memory fallback is only used when no shared client was ever configured
// (local development and unit tests).
type OnceStore struct {
	client *redis.Client
	logger *slog.Logger
	ttl    time.Duration

	mu     sync.Mutex
	memory map[string]time.Time
}

// NewOnceStore constructs an OnceStore. A nil client selects local (in-memory)
// mode; a non-nil client selects shared mode with fail-closed claims.
func NewOnceStore(client *redis.Client, logger *slog.Logger, ttl time.Duration) *OnceStore {
	if logger == nil {
		logger = slog.Default()
	}
	if ttl <= 0 {
		ttl = DefaultOnceTTL
	}
	return &OnceStore{
		client: client,
		logger: logger,
		ttl:    ttl,
		memory: map[string]time.Time{},
	}
}

// NewOnceStoreWithAddr connects to Valkey at addr. Unlike the idempotency
// store it never degrades to in-memory mode when the address is configured:
// a failed ping keeps shared mode active (the redis client reconnects
// lazily) and Claim fails closed until Valkey is reachable again. An empty
// addr selects local mode for development and tests.
func NewOnceStoreWithAddr(addr, password string, db int, logger *slog.Logger, ttl time.Duration) *OnceStore {
	if addr == "" {
		return NewOnceStore(nil, logger, ttl)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to ping Valkey for once-store at "+addr+", claims will fail closed until reachable", "error", err)
	}
	return NewOnceStore(client, logger, ttl)
}

// Shared reports whether claims are backed by shared Valkey state.
func (s *OnceStore) Shared() bool {
	return s.client != nil
}

// Claim acquires the once-only key for the configured TTL. It returns
// (true, nil) when the caller owns the claim, (false, nil) when the key was
// already claimed, and (false, err) when a shared store is configured but
// unavailable (fail closed: the caller must not perform the side effect).
func (s *OnceStore) Claim(ctx context.Context, key string) (bool, error) {
	if s.client != nil {
		ok, err := s.client.SetNX(ctx, key, time.Now().UTC().Format(time.RFC3339), s.ttl).Result()
		if err != nil {
			return false, err
		}
		return ok, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if exp, exists := s.memory[key]; exists && now.Before(exp) {
		return false, nil
	}
	s.memory[key] = now.Add(s.ttl)
	return true, nil
}

// Release removes a previously acquired claim (e.g. after a failed side
// effect) so a later retry can claim it again.
func (s *OnceStore) Release(ctx context.Context, key string) {
	if s.client != nil {
		_ = s.client.Del(ctx, key).Err()
	}
	s.mu.Lock()
	delete(s.memory, key)
	s.mu.Unlock()
}
