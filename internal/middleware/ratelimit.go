package middleware

import (
	"context"
	"fmt"
	"net/http"
	"news-aggregator/internal/cache"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type inMemoryLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
}

var memoryLimiter = &inMemoryLimiter{
	records: make(map[string][]time.Time),
}

// RateLimit creates a rate limiting middleware that restricts requests per minute (RPM).
// Uses Redis when available for distributed rate limiting, falling back to in-memory window tracking.
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

func checkRateLimit(ctx context.Context, cacheSvc cache.CacheService, key string, limit int, window time.Duration) (bool, int, error) {
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, time.Now().Unix()/60)

	// 1. If Redis is available, use Redis atomic counter
	if cacheSvc != nil && cacheSvc.IsAvailable() {
		valStr, err := cacheSvc.Get(ctx, redisKey)
		count := 0
		if err == nil {
			count, _ = strconv.Atoi(valStr)
		}

		if count >= limit {
			return false, 0, nil
		}

		_ = cacheSvc.Set(ctx, redisKey, count+1, window+5*time.Second)
		remaining := limit - (count + 1)
		if remaining < 0 {
			remaining = 0
		}
		return true, remaining, nil
	}

	// 2. In-memory fallback
	memoryLimiter.mu.Lock()
	defer memoryLimiter.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	timestamps := memoryLimiter.records[key]
	var valid []time.Time
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		memoryLimiter.records[key] = valid
		return false, 0, nil
	}

	valid = append(valid, now)
	memoryLimiter.records[key] = valid

	remaining := limit - len(valid)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, nil
}

