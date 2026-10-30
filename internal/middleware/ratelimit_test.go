package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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

