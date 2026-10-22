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

var slidingWindowLuaScript = redis.NewScript(`
local key = KEYS[1]
local windowStartNano = ARGV[1]
local nowNano = ARGV[2]
local member = ARGV[3]
local limit = tonumber(ARGV[4])
local expireSec = tonumber(ARGV[5])

redis.call('ZREMRANGEBYSCORE', key, '0', windowStartNano)
local count = redis.call('ZCARD', key)

if count >= limit then
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local oldestScore = '0'
    if #oldest >= 2 then
        oldestScore = tostring(oldest[2])
    end
    return {0, count, oldestScore}
end

redis.call('ZADD', key, nowNano, member)
redis.call('EXPIRE', key, expireSec)

return {1, count + 1, '0'}
`)

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
	expireSec := int64((window + time.Second).Seconds())
	if expireSec <= 0 {
		expireSec = 1
	}

	b := make([]byte, 4)
	_, _ = crand.Read(b)
	member := fmt.Sprintf("%d-%s", nowNano, hex.EncodeToString(b))

	// Execute pruning, counting, admission, and insertion atomically in Lua to prevent race conditions during bursts
	res, err := slidingWindowLuaScript.Run(ctx, rl.client, []string{redisKey},
		strconv.FormatInt(windowStartNano, 10),
		strconv.FormatInt(nowNano, 10),
		member,
		strconv.Itoa(limit),
		strconv.FormatInt(expireSec, 10),
	).Slice()

	if err != nil {
		rl.logger.Warn("valkey atomic rate limiter error, falling back to memory", "error", err)
		return rl.allowMemory(key, limit, window)
	}

	if len(res) < 3 {
		return rl.allowMemory(key, limit, window)
	}

	allowedInt, _ := res[0].(int64)
	countInt, _ := res[1].(int64)
	oldestScoreStr, _ := res[2].(string)

	if allowedInt == 0 {
		retryAfter := window
		if oldestScoreStr != "" && oldestScoreStr != "0" {
			if oldestScoreNano, parseErr := strconv.ParseInt(oldestScoreStr, 10, 64); parseErr == nil && oldestScoreNano > 0 {
				oldestTime := time.Unix(0, oldestScoreNano)
				retryAfter = time.Until(oldestTime.Add(window))
				if retryAfter < 0 {
					retryAfter = time.Second
				}
			}
		}
		return false, 0, retryAfter, nil
	}

	remaining := limit - int(countInt)
	if remaining < 0 {
		remaining = 0
	}
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
