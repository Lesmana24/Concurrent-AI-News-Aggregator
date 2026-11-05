package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMaxBodySize_AllowsRequestsUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(1024)) // 1 KB limit
	r.POST("/test", func(c *gin.Context) {
		var payload map[string]string
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	smallBody := bytes.NewBufferString(`{"message": "hello"}`)
	req, _ := http.NewRequest(http.MethodPost, "/test", smallBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaxBodySize_RejectsRequestsExceedingContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(100)) // 100 bytes limit
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	largePayload := `{"data": "` + strings.Repeat("A", 200) + `"}`
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(largePayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "too large")
}

func TestMaxBodySize_RejectsStreamingChunkedExceedingLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(100)) // 100 bytes limit
	r.POST("/test", func(c *gin.Context) {
		var payload map[string]string
		if err := c.ShouldBindJSON(&payload); err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	largePayload := `{"data": "` + strings.Repeat("B", 300) + `"}`
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(largePayload))
	req.ContentLength = -1 // Simulate chunked encoding without Content-Length
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusRequestEntityTooLarge || w.Code == http.StatusBadRequest)
}

