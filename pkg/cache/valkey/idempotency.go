package valkey

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// IdempotencyHeader is the required HTTP header containing a client-generated UUID.
	IdempotencyHeader = "Idempotency-Key"
	// IdempotencyReplayedHeader marks responses served from the idempotency cache.
	IdempotencyReplayedHeader = "X-Idempotent-Replayed"
	// DefaultIdempotencyTTL matches the 24-hour (86400s) lock & replay window.
	DefaultIdempotencyTTL = 24 * time.Hour

	idempotencyStateInFlight  = "in_flight"
	idempotencyStateCompleted = "completed"
)

type idempotencyRecord struct {
	State       string `json:"state"`
	StatusCode  int    `json:"status_code,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Body        []byte `json:"body,omitempty"`
}

type memIdempotencyEntry struct {
	rec       idempotencyRecord
	expiresAt time.Time
}

// IdempotencyStore enforces atomic Valkey SET NX EX 86400 locks with an in-memory fallback.
type IdempotencyStore struct {
	client *redis.Client
	logger *slog.Logger
	ttl    time.Duration

	mu         sync.Mutex
	memRecords map[string]memIdempotencyEntry
}

// NewIdempotencyStore constructs a Valkey-backed IdempotencyStore (or in-memory if client == nil).
func NewIdempotencyStore(client *redis.Client, logger *slog.Logger) *IdempotencyStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &IdempotencyStore{
		client:     client,
		logger:     logger,
		ttl:        DefaultIdempotencyTTL,
		memRecords: make(map[string]memIdempotencyEntry),
	}
}

// NewIdempotencyStoreWithAddr connects to Valkey at addr. An empty addr
// selects local (in-memory) mode for development and tests. When an address
// is configured the store stays in shared mode even if the initial ping
// fails: the redis client reconnects lazily and Acquire fails closed until
// Valkey is reachable, so financial reservations can never silently degrade
// to process-local state.
func NewIdempotencyStoreWithAddr(addr, password string, db int, logger *slog.Logger) *IdempotencyStore {
	if logger == nil {
		logger = slog.Default()
	}
	if addr == "" {
		return NewIdempotencyStore(nil, logger)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to ping Valkey for idempotency store at "+addr+", store will fail closed until reachable", "error", err)
	}
	return NewIdempotencyStore(client, logger)
}

// IdempotencyKey builds the canonical Valkey key: idempotency:{customer_id}:{key}.
func IdempotencyKey(customerID, key string) string {
	return "idempotency:" + customerID + ":" + key
}

// Acquire atomically reserves the idempotency key as in-flight via SET NX EX 86400.
// Returns (acquired=true, nil, nil) if this caller owns execution, or (acquired=false, existing, nil)
// if a concurrent or previously completed record already exists.
func (s *IdempotencyStore) Acquire(ctx context.Context, redisKey string) (bool, *idempotencyRecord, error) {
	inFlight := idempotencyRecord{State: idempotencyStateInFlight}

	if s.client != nil {
		raw, err := json.Marshal(inFlight)
		if err != nil {
			return false, nil, err
		}
		ok, err := s.client.SetNX(ctx, redisKey, raw, s.ttl).Result()
		if err == nil {
			if ok {
				return true, nil, nil
			}
			existingRaw, getErr := s.client.Get(ctx, redisKey).Bytes()
			if getErr != nil {
				if errors.Is(getErr, redis.Nil) {
					// Key expired right between SetNX and Get; retry once.
					ok2, err2 := s.client.SetNX(ctx, redisKey, raw, s.ttl).Result()
					if err2 == nil && ok2 {
						return true, nil, nil
					}
					if err2 != nil {
						// Fail closed: with a shared store configured we must
						// never fall back to process-local reservations, which
						// would let two instances execute the same key.
						s.logger.Error("valkey idempotency retry SetNX failed; failing closed", "error", err2)
						return false, nil, err2
					}
				}
				return false, &inFlight, nil
			}
			var existing idempotencyRecord
			if unmarshalErr := json.Unmarshal(existingRaw, &existing); unmarshalErr != nil {
				return false, &inFlight, nil
			}
			return false, &existing, nil
		}
		// Fail closed: a shared reservation that cannot be obtained must not
		// silently degrade to in-memory state (which is not shared across
		// instances and would allow the same financial key to execute twice).
		s.logger.Error("valkey idempotency SetNX failed; failing closed", "error", err)
		return false, nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if entry, exists := s.memRecords[redisKey]; exists && now.Before(entry.expiresAt) {
		cp := entry.rec
		cp.Body = append([]byte(nil), entry.rec.Body...)
		return false, &cp, nil
	}
	s.memRecords[redisKey] = memIdempotencyEntry{
		rec:       inFlight,
		expiresAt: now.Add(s.ttl),
	}
	return true, nil, nil
}

// Complete stores the deterministic response status/body under the idempotency key for 24h.
func (s *IdempotencyStore) Complete(ctx context.Context, redisKey string, statusCode int, contentType string, body []byte) error {
	rec := idempotencyRecord{
		State:       idempotencyStateCompleted,
		StatusCode:  statusCode,
		ContentType: contentType,
		Body:        append([]byte(nil), body...),
	}
	if s.client != nil {
		raw, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		setErr := s.client.Set(ctx, redisKey, raw, s.ttl).Err()
		// Keep a local copy for diagnostics, but report the shared-store
		// failure: a completion that only this instance can replay is not a
		// shared success, so the caller must not treat the key as durably
		// recorded (it may release the reservation for a clean retry).
		s.mu.Lock()
		s.memRecords[redisKey] = memIdempotencyEntry{
			rec:       rec,
			expiresAt: time.Now().Add(s.ttl),
		}
		s.mu.Unlock()
		if setErr != nil {
			s.logger.Error("valkey idempotency Complete failed; shared replay state not recorded", "error", setErr)
			return setErr
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.memRecords[redisKey] = memIdempotencyEntry{
		rec:       rec,
		expiresAt: time.Now().Add(s.ttl),
	}
	return nil
}

// Release deletes an in-flight lock (e.g. on 5xx failure or panic) so retries can proceed.
func (s *IdempotencyStore) Release(ctx context.Context, redisKey string) {
	if s.client != nil {
		_ = s.client.Del(ctx, redisKey).Err()
	}
	s.mu.Lock()
	delete(s.memRecords, redisKey)
	s.mu.Unlock()
}

type responseCaptureWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	wroteHdr   bool
}

func (w *responseCaptureWriter) WriteHeader(code int) {
	if !w.wroteHdr {
		w.statusCode = code
		w.wroteHdr = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCaptureWriter) Write(p []byte) (int, error) {
	if !w.wroteHdr {
		w.statusCode = http.StatusOK
		w.wroteHdr = true
	}
	_, _ = w.body.Write(p)
	return w.ResponseWriter.Write(p)
}

// IdempotencyMiddleware requires a valid UUID Idempotency-Key header, deduplicates retries
// using atomic Valkey SET NX EX 86400 locks, replays cached responses on duplicate retries,
// and returns 409 Conflict if an identical key is currently in-flight.
func IdempotencyMiddleware(store *IdempotencyStore, customerIDFunc func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if store == nil {
				next.ServeHTTP(w, r)
				return
			}

			rawKey := strings.TrimSpace(r.Header.Get(IdempotencyHeader))
			parsedUUID, err := uuid.Parse(rawKey)
			if rawKey == "" || err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"missing or invalid Idempotency-Key UUID header"}`))
				return
			}

			customerID := ""
			if customerIDFunc != nil {
				customerID = strings.TrimSpace(customerIDFunc(r))
			}
			if customerID == "" {
				customerID = ClientIP(r)
			}

			redisKey := IdempotencyKey(customerID, parsedUUID.String())
			acquired, existing, err := store.Acquire(r.Context(), redisKey)
			if err != nil {
				// Fail closed: executing a state-changing financial request
				// without a shared reservation could run the same key twice
				// across instances.
				store.logger.Error("idempotency acquire error; failing closed", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"idempotency service unavailable; retry with the same Idempotency-Key"}`))
				return
			}

			if !acquired {
				if existing == nil || existing.State == idempotencyStateInFlight {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_, _ = w.Write([]byte(`{"error":"conflict: request with this Idempotency-Key is already in progress"}`))
					return
				}

				ct := existing.ContentType
				if ct == "" {
					ct = "application/json"
				}
				w.Header().Set("Content-Type", ct)
				w.Header().Set(IdempotencyReplayedHeader, "true")
				statusCode := existing.StatusCode
				if statusCode == 0 {
					statusCode = http.StatusOK
				}
				w.WriteHeader(statusCode)
				_, _ = w.Write(existing.Body)
				return
			}

			capture := &responseCaptureWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			completed := false
			defer func() {
				if !completed {
					store.Release(r.Context(), redisKey)
				}
			}()

			next.ServeHTTP(capture, r)

			if capture.statusCode >= http.StatusInternalServerError {
				store.Release(r.Context(), redisKey)
				completed = true
				return
			}

			if completeErr := store.Complete(r.Context(), redisKey, capture.statusCode, capture.Header().Get("Content-Type"), capture.body.Bytes()); completeErr != nil {
				// The shared replay state was not recorded. Release the
				// reservation so the client's retry re-reserves cleanly
				// instead of being pinned to a 409 conflict for the TTL.
				store.logger.Error("idempotency completion failed; releasing reservation for retry", "error", completeErr)
				store.Release(r.Context(), redisKey)
				completed = true
				return
			}
			completed = true
		})
	}
}
