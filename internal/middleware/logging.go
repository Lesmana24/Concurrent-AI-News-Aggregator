package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestIDAndLogger middleware attaches a unique X-Request-ID and records structured logs via log/slog.
func RequestIDAndLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			bytes := make([]byte, 8)
			if _, err := rand.Read(bytes); err == nil {
				reqID = hex.EncodeToString(bytes)
			} else {
				reqID = "req-" + time.Now().Format("20060102150405.000")
			}
		}

		c.Header("X-Request-ID", reqID)
		c.Set("request_id", reqID)

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		attrs := []any{
			slog.String("request_id", reqID),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Duration("latency", latency),
			slog.String("client_ip", clientIP),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.String()))
			slog.Error("HTTP request finished with errors", attrs...)
		} else if statusCode >= 500 {
			slog.Error("HTTP internal server error", attrs...)
		} else if statusCode >= 400 {
			slog.Warn("HTTP request rejected", attrs...)
		} else {
			slog.Info("HTTP request completed", attrs...)
		}
	}
}

