package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"news-aggregator/internal/cache"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type limiterShard struct {
	mu      sync.Mutex
	records map[string][]time.Time
	maxSize int
}

func (s *limiterShard) allow(key string, limit int, window time.Duration, now time.Time) (bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := now.Add(-window)
	timestamps, exists := s.records[key]

	var valid []time.Time
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) == 0 && exists {
		// All timestamps expired: delete key to prevent unbounded memory leak
		delete(s.records, key)
	}

	if len(valid) >= limit {
		s.records[key] = valid
		return false, 0
	}

	// Capacity cap enforcement: if adding a new key causes the shard to reach maxSize,
	// run eviction of expired or oldest entries
	if !exists && len(s.records) >= s.maxSize {
		s.evictExpiredOrOldest(cutoff)
	}

	valid = append(valid, now)
	s.records[key] = valid

	remaining := limit - len(valid)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining
}

func (s *limiterShard) evictExpiredOrOldest(cutoff time.Time) {
	// Step 1: Evict any entry that has no active timestamps
	for k, ts := range s.records {
		hasValid := false
		for _, t := range ts {
			if t.After(cutoff) {
				hasValid = true
				break
			}
		}
		if !hasValid {
			delete(s.records, k)
		}
	}

	// Step 2: If still at capacity, evict ~10% of entries to prevent OOM
	if len(s.records) >= s.maxSize {
		toEvict := s.maxSize / 10
		if toEvict <= 0 {
			toEvict = 1
		}
		count := 0
		for k := range s.records {
			delete(s.records, k)
			count++
			if count >= toEvict {
				break
			}
		}
	}
}

func (s *limiterShard) cleanupExpired(cutoff time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, ts := range s.records {
		var valid []time.Time
		for _, t := range ts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(s.records, k)
		} else {
			s.records[k] = valid
		}
	}
}

// ShardedMemoryLimiter provides a bounded, concurrent sliding-window rate limiter
// that splits keys across multiple shards to eliminate mutex contention.
type ShardedMemoryLimiter struct {
	shards    []*limiterShard
	numShards int
	stopChan  chan struct{}
	once      sync.Once
}

// NewShardedMemoryLimiter initializes a sharded in-memory limiter with auto-eviction and periodic janitor.
func NewShardedMemoryLimiter(numShards, maxPerShard int, cleanupInterval time.Duration) *ShardedMemoryLimiter {
	if numShards <= 0 {
		numShards = 16
	}
	if maxPerShard <= 0 {
		maxPerShard = 1000
	}

	shards := make([]*limiterShard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &limiterShard{
			records: make(map[string][]time.Time),
			maxSize: maxPerShard,
		}
	}

	limiter := &ShardedMemoryLimiter{
		shards:    shards,
		numShards: numShards,
		stopChan:  make(chan struct{}),
	}

	if cleanupInterval > 0 {
		go limiter.startJanitor(cleanupInterval)
	}

	return limiter
}

func (l *ShardedMemoryLimiter) Allow(key string, limit int, window time.Duration, now time.Time) (bool, int) {
	idx := getShardIndex(key, l.numShards)
	return l.shards[idx].allow(key, limit, window, now)
}

func (l *ShardedMemoryLimiter) cleanup(now time.Time, window time.Duration) {
	cutoff := now.Add(-window)
	for _, s := range l.shards {
		s.cleanupExpired(cutoff)
	}
}

func (l *ShardedMemoryLimiter) startJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup(time.Now(), 1*time.Minute)
		case <-l.stopChan:
			return
		}
	}
}

// Close stops the background janitor goroutine.
func (l *ShardedMemoryLimiter) Close() {
	l.once.Do(func() {
		close(l.stopChan)
	})
}

// CountActiveKeys counts the total number of keys actively stored across all shards.
func (l *ShardedMemoryLimiter) CountActiveKeys() int {
	total := 0
	for _, s := range l.shards {
		s.mu.Lock()
		total += len(s.records)
		s.mu.Unlock()
	}
	return total
}

func getShardIndex(key string, numShards int) int {
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return int(h % uint32(numShards))
}

func sanitizeClientKey(rawKey string) string {
	trimmed := strings.TrimSpace(rawKey)
	if trimmed == "" {
		return "anonymous"
	}
	if len(trimmed) > 64 {
		h := sha256.Sum256([]byte(trimmed))
		return hex.EncodeToString(h[:16])
	}
	return trimmed
}

// Package-level default in-memory limiter with 16 shards (16,000 max capacity) and 1-minute janitor sweep.
var defaultMemoryLimiter = NewShardedMemoryLimiter(16, 1000, 1*time.Minute)

// RateLimit creates a rate limiting middleware that restricts requests per minute (RPM).
// Uses Redis atomic Lua script when available for distributed rate limiting, falling back
// to a sharded, bounded in-memory sliding window limiter to prevent race conditions and memory leaks.
func RateLimit(cacheSvc cache.CacheService, rpm int) gin.HandlerFunc {
	if rpm <= 0 {
		rpm = 60 // Default: 60 requests per minute
	}

	window := 1 * time.Minute

	return func(c *gin.Context) {
		// Identify client by API Key first, fallback to client IP
		clientKey := c.GetHeader("X-API-KEY")
		if clientKey == "" {
			clientKey = c.ClientIP()
		}

		allowed, remaining, err := checkRateLimit(c.Request.Context(), cacheSvc, clientKey, rpm, window)
		if err != nil {
			// Fail open on unexpected error so service doesn't break
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rpm))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded. Please slow down and try again later.",
				"limit":   rpm,
				"window":  "1m",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func checkRateLimit(ctx context.Context, cacheSvc cache.CacheService, rawKey string, limit int, window time.Duration) (bool, int, error) {
	key := sanitizeClientKey(rawKey)

	// 1. If Redis is available, execute single atomic round-trip Lua script (INCR + EXPIRE)
	if cacheSvc != nil && cacheSvc.IsAvailable() {
		redisKey := fmt.Sprintf("ratelimit:%s:%d", key, time.Now().Unix()/60)
		count, err := cacheSvc.Incr(ctx, redisKey, window+5*time.Second)
		if err == nil {
			if count > int64(limit) {
				return false, 0, nil
			}
			remaining := limit - int(count)
			if remaining < 0 {
				remaining = 0
			}
			return true, remaining, nil
		}
		// On transient Redis error, gracefully fall through to in-memory limiter for defense-in-depth
	}

	// 2. In-memory fallback (sharded, bounded, auto-cleaning sliding window)
	allowed, remaining := defaultMemoryLimiter.Allow(key, limit, window, time.Now())
	return allowed, remaining, nil
}
