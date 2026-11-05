package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize returns a Gin middleware that enforces a maximum request body size (in bytes)
// to prevent Denial of Service (DoS) attacks via memory exhaustion.
// If the request exceeds the limit, it returns HTTP 413 (StatusRequestEntityTooLarge).
func MaxBodySize(limitBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil || c.Request.Body == nil || limitBytes <= 0 {
			c.Next()
			return
		}

		// Fast path: check Content-Length header if present
		if c.Request.ContentLength > limitBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"status":  "error",
				"message": "request entity too large",
			})
			return
		}

		// Wrap body with MaxBytesReader to protect against chunked requests or spoofed Content-Length
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limitBytes)
		c.Next()

		// If an unhandled MaxBytesError occurred during handler execution
		for _, err := range c.Errors {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err.Err, &maxBytesErr) {
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
						"status":  "error",
						"message": "request entity too large",
					})
				}
				return
			}
		}
	}
}

