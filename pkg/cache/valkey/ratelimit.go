package valkey

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides distributed sliding-window rate limiting.
type RateLimiter struct {
	client    *redis.Client
	logger    *slog.Logger
	mu        sync.Mutex
	memLimits map[string][]time.Time
}

// NewRateLimiter creates a RateLimiter backed by Valkey if client is non-nil,
// or falls back to an in-memory sliding window limiter.
func NewRateLimiter(client *redis.Client, logger *slog.Logger) *RateLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	return &RateLimiter{
		client:    client,
		logger:    logger,
		memLimits: make(map[string][]time.Time),
	}
}

// NewRateLimiterWithAddr initializes a RateLimiter. If addr is empty or unpingable,
// it gracefully falls back to an in-memory sliding window limiter.
func NewRateLimiterWithAddr(addr, password string, db int, logger *slog.Logger) *RateLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	if addr == "" {
		return NewRateLimiter(nil, logger)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("failed to ping Valkey for rate limiter at "+addr+", falling back to in-memory", "error", err)
		_ = client.Close()
		return NewRateLimiter(nil, logger)
	}

	return NewRateLimiter(client, logger)
}

// Allow checks if the given key is allowed under limit within the specified window.
// Returns (allowed bool, remaining int, retryAfter time.Duration, err error).
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	if rl.client == nil {
		return rl.allowMemory(key, limit, window)
	}

	redisKey := "ratelimit:" + key
	now := time.Now()
	nowNano := now.UnixNano()
	windowStartNano := now.Add(-window).UnixNano()

	pipe := rl.client.Pipeline()
	// 1. Remove entries outside the sliding window
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStartNano, 10))
	// 2. Count remaining entries in window
	cardCmd := pipe.ZCard(ctx, redisKey)
	// 3. Set expiration
	pipe.Expire(ctx, redisKey, window+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.Warn("valkey rate limiter error, falling back to memory", "error", err)
		return rl.allowMemory(key, limit, window)
	}

	currentCount := int(cardCmd.Val())
	if currentCount >= limit {
		// Calculate retryAfter
		oldest := rl.client.ZRangeWithScores(ctx, redisKey, 0, 0).Val()
		retryAfter := window
		if len(oldest) > 0 {
			oldestTime := time.Unix(0, int64(oldest[0].Score))
			retryAfter = time.Until(oldestTime.Add(window))
			if retryAfter < 0 {
				retryAfter = time.Second
			}
		}
		return false, 0, retryAfter, nil
	}

	// Add current request
	b := make([]byte, 4)
	_, _ = crand.Read(b)
	member := fmt.Sprintf("%d-%s", nowNano, hex.EncodeToString(b))

	addPipe := rl.client.Pipeline()
	addPipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(nowNano), Member: member})
	addPipe.Expire(ctx, redisKey, window+time.Second)
	_, _ = addPipe.Exec(ctx)

	remaining := limit - (currentCount + 1)
	return true, remaining, 0, nil
}

func (rl *RateLimiter) allowMemory(key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	timestamps := rl.memLimits[key]
	valid := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		oldest := valid[0]
		retryAfter := time.Until(oldest.Add(window))
		if retryAfter < 0 {
			retryAfter = time.Second
		}
		rl.memLimits[key] = valid
		return false, 0, retryAfter, nil
	}

	valid = append(valid, now)
	rl.memLimits[key] = valid
	remaining := limit - len(valid)
	return true, remaining, 0, nil
}

// ClientIP extracts the remote client IP from headers or RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// RateLimitMiddleware creates an HTTP middleware that throttles requests based on keyFunc.
func RateLimitMiddleware(limiter *RateLimiter, keyFunc func(r *http.Request) string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			allowed, remaining, retryAfter, err := limiter.Allow(r.Context(), key, limit, window)
			if err != nil {
				// Fail open on rate limiter error to preserve service availability
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				retrySec := int(retryAfter.Seconds())
				if retrySec <= 0 {
					retrySec = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySec))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please retry later"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
