package middleware

import (
	"net/http"
	"news-aggregator/internal/service"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(svc service.NewsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-KEY")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key is missing"})
			c.Abort()
			return
		}

		user, err := svc.GetUser(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			c.Abort()
			return
		}

		if user.Quota <= 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Quota exceeded"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
