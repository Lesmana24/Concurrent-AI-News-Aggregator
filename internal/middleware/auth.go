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

		if user.IsSystem {
			c.JSON(http.StatusForbidden, gin.H{"error": "System API keys cannot be used for public API access"})
			c.Abort()
			return
		}

		if user.Quota <= 0 {
			msg := "Quota exceeded"
			if user.IsGuest {
				msg = "Batas 5 token harian Anda telah habis. Kuota akan otomatis tereset kembali menjadi 5 pada pukul 00:00 WIB. Daftarkan email developer untuk kuota lebih tinggi."
			}
			c.JSON(http.StatusForbidden, gin.H{
				"error":       msg,
				"quota":       0,
				"daily_limit": user.DailyLimit,
				"is_guest":    user.IsGuest,
			})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
