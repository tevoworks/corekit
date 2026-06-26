// Package redisstore provides an optional Redis-backed session revocation store.
// The store is designed to be a pure acceleration layer — Redis failure NEVER
// affects correctness; the DB remains the canonical source of truth.
package redisstore

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RevocationStore wraps an optional Redis client with graceful fallback semantics.
// When Redis is unavailable the store silently no-ops all writes and returns
// false (not revoked) for reads — callers fall back to DB verification.
type RevocationStore struct {
	client    *redis.Client
	enabled   int32 // 1 = connected; 0 = disabled. Updated atomically.
	stopCh    chan struct{}
	closeOnce sync.Once
}

const (
	revocationKeyPrefix = "revoked:sid:"
	pingTimeout         = 1 * time.Second
	healthCheckInterval = 10 * time.Second
)

// NewRevocationStore creates a RevocationStore. If redisURL is empty or Redis
// cannot be reached, a disabled (no-op) store is returned — no panic.
func NewRevocationStore(redisURL string) *RevocationStore {
	if redisURL == "" {
		slog.Info("No REDIS_URL configured — revocation store disabled (Mode B)")
		return &RevocationStore{}
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Warn("Invalid REDIS_URL — revocation store disabled (Mode B)", "error", err)
		return &RevocationStore{}
	}

	client := redis.NewClient(opts)

	store := &RevocationStore{
		client: client,
		stopCh: make(chan struct{}),
	}

	// Probe connectivity. Non-fatal: failure just means Mode B at startup.
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("Ping failed — starting in Mode B; will retry in background", "error", err)
	} else {
		atomic.StoreInt32(&store.enabled, 1)
		slog.Info("Connected — Mode A (fast revocation path active)")
	}

	// Background health checker — auto-switches mode on recovery or failure.
	go store.healthLoop()

	return store
}

// IsEnabled performs a lightweight liveness check. Mode can switch per-request.
func (s *RevocationStore) IsEnabled() bool {
	if s.client == nil {
		return false
	}
	return atomic.LoadInt32(&s.enabled) == 1
}

// MarkRevoked writes the revocation marker for tokenID with a TTL matching
// the remaining session lifetime. Fire-and-forget safe: errors are logged only.
func (s *RevocationStore) MarkRevoked(tokenID string, expiresAt time.Time) {
	if !s.IsEnabled() {
		return
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour // safety floor — already expired sessions still block
	}

	key := revocationKeyPrefix + tokenID
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := s.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		slog.Error("MarkRevoked failed", "token_id", tokenID, "error", err)
		s.markDegraded()
	}
}

// IsRevoked checks whether tokenID has a Redis revocation marker.
// Returns (false, nil) when Redis is unavailable — callers must fall back to DB.
func (s *RevocationStore) IsRevoked(tokenID string) (bool, error) {
	if !s.IsEnabled() {
		return false, nil // Mode B: let DB decide
	}

	key := revocationKeyPrefix + tokenID
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	val, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil // key not present → not revoked
	}
	if err != nil {
		slog.Error("IsRevoked error", "token_id", tokenID, "error", err)
		s.markDegraded()
		return false, nil // degrade to DB fallback, not an error for callers
	}

	return val == "1", nil
}

// MarkAllRevoked bulk-revokes a slice of tokenIDs using a pipeline for efficiency.
func (s *RevocationStore) MarkAllRevoked(tokenIDs []string, expiresAt time.Time) {
	if !s.IsEnabled() || len(tokenIDs) == 0 {
		return
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pipe := s.client.Pipeline()
	for _, id := range tokenIDs {
		pipe.Set(ctx, revocationKeyPrefix+id, "1", ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Error("MarkAllRevoked pipeline error", "error", err)
		s.markDegraded()
	}
}

// ModeLabel returns a human-readable mode string for logging.
func (s *RevocationStore) ModeLabel() string {
	if s.IsEnabled() {
		return "MODE_A (Redis ON)"
	}
	return "MODE_B (Redis OFF)"
}

// Close cleanly shuts down the Redis client connection and stops the health loop.
func (s *RevocationStore) Close() {
	s.closeOnce.Do(func() {
		close(s.stopCh)
		if s.client != nil {
			_ = s.client.Close()
		}
	})
}

// ─── internal ────────────────────────────────────────────────────────────────

func (s *RevocationStore) markDegraded() {
	if atomic.CompareAndSwapInt32(&s.enabled, 1, 0) {
		slog.Warn("Connection degraded — switching to Mode B")
	}
}

func (s *RevocationStore) healthLoop() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if s.client == nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
			err := s.client.Ping(ctx).Err()
			cancel()

			if err != nil {
				if atomic.CompareAndSwapInt32(&s.enabled, 1, 0) {
					slog.Warn("Health check failed — switching to Mode B", "error", err)
				}
			} else {
				if atomic.CompareAndSwapInt32(&s.enabled, 0, 1) {
					slog.Info("Reconnected — switching back to Mode A")
				}
			}
		}
	}
}
