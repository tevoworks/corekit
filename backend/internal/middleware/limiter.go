package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/tevoworks/corekit/backend/pkg/httputil"
)

var tokenBucketLua = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local fill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local data = redis.call('HMGET', key, 'tokens', 'last_check')
local tokens = tonumber(data[1])
local last_check = tonumber(data[2])

if not tokens then
    tokens = limit
    last_check = now
else
    local elapsed = now - last_check
    if elapsed > 0 then
        tokens = tokens + (elapsed * fill_rate)
        if tokens > limit then
            tokens = limit
        end
        last_check = now
    end
end

if tokens >= 1.0 then
    tokens = tokens - 1.0
    redis.call('HSET', key, 'tokens', tokens, 'last_check', last_check)
    redis.call('EXPIRE', key, 600)
    return 1
else
    redis.call('HSET', key, 'tokens', tokens, 'last_check', last_check)
    redis.call('EXPIRE', key, 600)
    return 0
end
`

type rateLimitEntry struct {
	mu        sync.Mutex
	tokens    float64
	lastCheck time.Time
}

type RateLimiter struct {
	window      time.Duration
	limits      sync.Map
	maxEntries  int
	count       int64
	redisClient *redis.Client
	redisLuaSHA string
}

func NewRateLimiter(window time.Duration, redisURL string) *RateLimiter {
	rl := &RateLimiter{
		window:     window,
		maxEntries: 50000,
	}

	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			rl.redisClient = redis.NewClient(opt)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			sha, err := rl.redisClient.ScriptLoad(ctx, tokenBucketLua).Result()
			cancel()
			if err == nil {
				rl.redisLuaSHA = sha
			}
		}
	}

	if rl.redisClient == nil {
		go rl.startCleanupLoop()
		slog.Warn("Redis not configured, using in-memory rate limiter (limits NOT shared across instances)")
	}

	return rl
}

func (rl *RateLimiter) startCleanupLoop() {
	ticker := time.NewTicker(rl.window * 2)
	for range ticker.C {
		now := time.Now()
		rl.limits.Range(func(k, v any) bool {
			entry := v.(*rateLimitEntry)
			entry.mu.Lock()
			expired := now.Sub(entry.lastCheck) > rl.window
			entry.mu.Unlock()

			if expired {
				if _, deleted := rl.limits.LoadAndDelete(k); deleted {
					atomic.AddInt64(&rl.count, -1)
				}
			}
			return true
		})
	}
}

func (rl *RateLimiter) evictOne() interface{} {
	var oldestKey interface{}
	var oldestTime time.Time
	rl.limits.Range(func(k, v any) bool {
		entry := v.(*rateLimitEntry)
		entry.mu.Lock()
		if oldestKey == nil || entry.lastCheck.Before(oldestTime) {
			oldestKey = k
			oldestTime = entry.lastCheck
		}
		entry.mu.Unlock()
		return true
	})
	return oldestKey
}

func (rl *RateLimiter) Allow(key string, limit int) bool {
	if rl.redisClient != nil && rl.redisLuaSHA != "" {
		nowMs := time.Now().UnixNano() / int64(time.Millisecond)
		fillRate := float64(limit) / float64(rl.window.Milliseconds())

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		res, err := rl.redisClient.EvalSha(ctx, rl.redisLuaSHA, []string{key}, limit, fillRate, nowMs).Result()
		if err == nil {
			if val, ok := res.(int64); ok {
				return val == 1
			}
		}
	}

	var entry *rateLimitEntry
	val, exists := rl.limits.Load(key)
	if !exists {
		if atomic.LoadInt64(&rl.count) >= int64(rl.maxEntries) {
			rl.limits.Delete(rl.evictOne())
			atomic.AddInt64(&rl.count, -1)
		}
		newEntry := &rateLimitEntry{
			tokens:    float64(limit),
			lastCheck: time.Now(),
		}
		val, loaded := rl.limits.LoadOrStore(key, newEntry)
		if !loaded {
			atomic.AddInt64(&rl.count, 1)
		}
		entry = val.(*rateLimitEntry)
	} else {
		entry = val.(*rateLimitEntry)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(entry.lastCheck).Seconds()
	fillRate := float64(limit) / rl.window.Seconds()
	entry.tokens = entry.tokens + elapsed*fillRate
	if entry.tokens > float64(limit) {
		entry.tokens = float64(limit)
	}
	entry.lastCheck = now

	if entry.tokens >= 1.0 {
		entry.tokens -= 1.0
		return true
	}

	return false
}

var (
	IPRateLimiter    *RateLimiter
	EmailRateLimiter *RateLimiter
)

func InitRateLimiters(redisURL string, appEnv string) {
	if appEnv == "production" && redisURL == "" {
		slog.Error("REDIS_URL is required for rate limiting in production")
		os.Exit(1)
	}
	IPRateLimiter = NewRateLimiter(1*time.Minute, redisURL)
	EmailRateLimiter = NewRateLimiter(1*time.Minute, redisURL)
}

func LimitIP(limit int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if IPRateLimiter == nil {
				return next(c)
			}
			ip := c.RealIP()
			if !IPRateLimiter.Allow(ip, limit) {
				return httputil.TooManyRequests(c, "Rate limit exceeded. Please try again later.")
			}
			return next(c)
		}
	}
}

func LimitEmail(limit int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if EmailRateLimiter == nil {
				return next(c)
			}
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return httputil.BadRequest(c, "Invalid request body")
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))

			var req struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(body, &req); err == nil && req.Email != "" {
				key := "email:" + HashString(strings.ToLower(strings.TrimSpace(req.Email)))
				if !EmailRateLimiter.Allow(key, limit) {
					return httputil.TooManyRequests(c, "Too many attempts for this account. Please try again later.")
				}
			}
			return next(c)
		}
	}
}

func HashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
