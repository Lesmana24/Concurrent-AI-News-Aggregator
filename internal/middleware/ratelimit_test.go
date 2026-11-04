package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockAtomicCache struct {
	mu      sync.Mutex
	store   map[string]int64
	failErr error
}

func newMockAtomicCache() *mockAtomicCache {
	return &mockAtomicCache{
		store: make(map[string]int64),
	}
}

func (m *mockAtomicCache) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockAtomicCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (m *mockAtomicCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockAtomicCache) Incr(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failErr != nil {
		return 0, m.failErr
	}
	m.store[key]++
	return m.store[key], nil
}

func (m *mockAtomicCache) IsAvailable() bool {
	return true
}

func (m *mockAtomicCache) Close() error {
	return nil
}

func TestRateLimit_UnderLimitAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(nil, 5))

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set("X-API-KEY", "key-allowed")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimit_ExceededBlockedWith429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(nil, 2))

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 1st request -> OK
	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.Header.Set("X-API-KEY", "key-blocked")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2nd request -> OK
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.Header.Set("X-API-KEY", "key-blocked")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 3rd request -> Blocked with 429
	req3 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req3.Header.Set("X-API-KEY", "key-blocked")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Equal(t, "60", w3.Header().Get("Retry-After"))
	assert.Contains(t, w3.Body.String(), "Rate limit exceeded")
}

func TestRateLimit_RedisAtomic_UnderLimitAndBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cacheMock := newMockAtomicCache()
	r := gin.New()
	r.Use(RateLimit(cacheMock, 3))

	r.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, "data-ok")
	})

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/data", nil)
		req.Header.Set("X-API-KEY", "redis-client")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
		expectedRemaining := fmt.Sprintf("%d", 3-i)
		assert.Equal(t, expectedRemaining, w.Header().Get("X-RateLimit-Remaining"))
	}

	// 4th request must be blocked
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("X-API-KEY", "redis-client")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "60", w.Header().Get("Retry-After"))
}

func TestRateLimit_RedisConcurrentRequests_Atomic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cacheMock := newMockAtomicCache()
	r := gin.New()
	limit := 10
	r.Use(RateLimit(cacheMock, limit))

	r.GET("/resource", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	totalRequests := 25
	var passCount int32
	var blockCount int32
	var wg sync.WaitGroup

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/resource", nil)
			req.Header.Set("X-API-KEY", "concurrent-client")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				atomic.AddInt32(&passCount, 1)
			} else if w.Code == http.StatusTooManyRequests {
				atomic.AddInt32(&blockCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(limit), passCount, "Exactly limit requests must pass without race condition leakage")
	assert.Equal(t, int32(totalRequests-limit), blockCount, "Remaining requests must be blocked with 429")
}

func TestRateLimit_RedisError_FallbackToInMemory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cacheMock := newMockAtomicCache()
	cacheMock.failErr = errors.New("redis connection timeout")

	r := gin.New()
	r.Use(RateLimit(cacheMock, 2))

	r.GET("/fallback", func(c *gin.Context) {
		c.String(http.StatusOK, "fallback-ok")
	})

	// 1st request -> OK via in-memory fallback
	req1 := httptest.NewRequest(http.MethodGet, "/fallback", nil)
	req1.Header.Set("X-API-KEY", "client-fallback")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2nd request -> OK via in-memory fallback
	req2 := httptest.NewRequest(http.MethodGet, "/fallback", nil)
	req2.Header.Set("X-API-KEY", "client-fallback")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 3rd request -> Blocked via in-memory fallback
	req3 := httptest.NewRequest(http.MethodGet, "/fallback", nil)
	req3.Header.Set("X-API-KEY", "client-fallback")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
}

func TestRateLimit_InMemory_CleanupAndNoLeak(t *testing.T) {
	limiter := NewShardedMemoryLimiter(4, 50, 0)
	defer limiter.Close()

	now := time.Now()
	window := 100 * time.Millisecond

	allowed1, _ := limiter.Allow("user-alpha", 5, window, now)
	allowed2, _ := limiter.Allow("user-beta", 5, window, now)
	assert.True(t, allowed1)
	assert.True(t, allowed2)
	assert.Equal(t, 2, limiter.CountActiveKeys())

	// Time has moved forward past window
	future := now.Add(200 * time.Millisecond)
	limiter.cleanup(future, window)

	assert.Equal(t, 0, limiter.CountActiveKeys(), "Expired keys must be pruned from map to prevent memory leak")
}

func TestRateLimit_InMemory_CapacityCapEviction(t *testing.T) {
	// Small capacity: 2 shards, max 3 per shard => total capacity ~6
	limiter := NewShardedMemoryLimiter(2, 3, 0)
	defer limiter.Close()

	now := time.Now()
	window := 1 * time.Minute

	for i := 0; i < 30; i++ {
		key := fmt.Sprintf("flood-user-%d", i)
		limiter.Allow(key, 10, window, now)
	}

	// Active keys must never explode beyond shard capacity
	activeKeys := limiter.CountActiveKeys()
	assert.LessOrEqual(t, activeKeys, 6, "Total active keys must be strictly bounded by capacity cap")
}

func TestRateLimit_InMemory_ConcurrentSafety(t *testing.T) {
	limiter := NewShardedMemoryLimiter(8, 100, 0)
	defer limiter.Close()

	now := time.Now()
	window := 1 * time.Minute
	limit := 15

	var wg sync.WaitGroup
	var passed int32
	var blocked int32

	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _ := limiter.Allow("concurrent-user", limit, window, now)
			if allowed {
				atomic.AddInt32(&passed, 1)
			} else {
				atomic.AddInt32(&blocked, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(limit), passed)
	assert.Equal(t, int32(40-limit), blocked)
}
