package authverify

import (
	"sync"
	"sync/atomic"
	"time"
)

// ttlByAction defines the maximum cache lifetime per action type.
// CRITICAL is intentionally absent — it is NEVER cached.
var ttlByAction = map[ActionType]time.Duration{
	ActionREAD:  60 * time.Second,
	ActionWRITE: 30 * time.Second,
}

type cachedEntry struct {
	response  *IntrospectResponse
	expiresAt time.Time
}

// IntrospectionCache is a lightweight in-memory TTL cache keyed by
// (tokenID + actionType). It is goroutine-safe via sync.Map and bounded.
type IntrospectionCache struct {
	store sync.Map
	count int64
}

// NewIntrospectionCache creates a ready-to-use cache.
func NewIntrospectionCache() *IntrospectionCache {
	c := &IntrospectionCache{}
	go c.periodicPurge()
	return c
}

// cacheKey builds the composite lookup key.
func cacheKey(tokenID string, action ActionType) string {
	return tokenID + ":" + string(action)
}

func bypass(action ActionType) bool {
	return action == ActionCRITICAL
}

// Get retrieves a cached introspection result. Returns (nil, false) on miss or expiry.
func (c *IntrospectionCache) Get(tokenID string, action ActionType) (*IntrospectResponse, bool) {
	if bypass(action) {
		return nil, false // CRITICAL always bypasses cache
	}

	key := cacheKey(tokenID, action)
	raw, ok := c.store.Load(key)
	if !ok {
		return nil, false
	}

	entry := raw.(cachedEntry)
	if time.Now().After(entry.expiresAt) {
		if _, deleted := c.store.LoadAndDelete(key); deleted {
			atomic.AddInt64(&c.count, -1)
		}
		return nil, false
	}

	return entry.response, true
}

// Set stores a response in cache with the appropriate TTL for the action type.
// CRITICAL responses are silently ignored (never cached).
func (c *IntrospectionCache) Set(tokenID string, action ActionType, resp *IntrospectResponse) {
	if bypass(action) || resp == nil {
		return
	}

	ttl, ok := ttlByAction[action]
	if !ok || ttl <= 0 {
		return
	}

	key := cacheKey(tokenID, action)

	if atomic.LoadInt64(&c.count) >= 50000 {
		c.store.Range(func(k, v any) bool {
			entry := v.(cachedEntry)
			if time.Now().After(entry.expiresAt) {
				if _, deleted := c.store.LoadAndDelete(k); deleted {
					atomic.AddInt64(&c.count, -1)
				}
			}
			return true
		})
		// Evict oldest 10% if still over capacity
		if atomic.LoadInt64(&c.count) >= 50000 {
			var oldestKey any
			var oldestTime time.Time
			evictTarget := atomic.LoadInt64(&c.count) - 45000
			var evicted int64
			c.store.Range(func(k, v any) bool {
				if evicted >= evictTarget {
					return false
				}
				entry := v.(cachedEntry)
				if oldestKey == nil || entry.expiresAt.Before(oldestTime) {
					oldestKey = k
					oldestTime = entry.expiresAt
				}
				if _, deleted := c.store.LoadAndDelete(k); deleted {
					atomic.AddInt64(&c.count, -1)
					evicted++
				}
				return true
			})
		}
	}

	_, loaded := c.store.LoadOrStore(key, cachedEntry{
		response:  resp,
		expiresAt: time.Now().Add(ttl),
	})
	if !loaded {
		atomic.AddInt64(&c.count, 1)
	}
}

// Invalidate removes all cached entries for a tokenID across all action types.
// Must be called on session revocation, logout, and user lock/suspend.
func (c *IntrospectionCache) Invalidate(tokenID string) {
	for _, action := range []ActionType{ActionREAD, ActionWRITE} {
		key := cacheKey(tokenID, action)
		if _, deleted := c.store.LoadAndDelete(key); deleted {
			atomic.AddInt64(&c.count, -1)
		}
	}
}

// periodicPurge evicts expired entries every 2 minutes to prevent unbounded growth.
func (c *IntrospectionCache) periodicPurge() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.store.Range(func(k, v any) bool {
			entry := v.(cachedEntry)
			if now.After(entry.expiresAt) {
				if _, deleted := c.store.LoadAndDelete(k); deleted {
					atomic.AddInt64(&c.count, -1)
				}
			}
			return true
		})
	}
}
