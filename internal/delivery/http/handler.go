package http

import (
	"fmt"
	"net/http"
	"news-aggregator/internal/models"
	"news-aggregator/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

type NewsHandler struct {
	service service.NewsService
}

func NewNewsHandler(s service.NewsService) *NewsHandler {
	return &NewsHandler{service: s}
}

func (h *NewsHandler) Summarize(c *gin.Context) {
	var req models.SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: urls must be a non-empty list of valid URLs"})
		return
	}

	apiKey := c.GetHeader("X-API-KEY")
	res, err := h.service.SummarizeArticles(apiKey, req.URLs)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") || strings.Contains(err.Error(), "no valid") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "quota exceeded") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *NewsHandler) QA(c *gin.Context) {
	var req models.NewsQARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	apiKey := c.GetHeader("X-API-KEY")
	res, err := h.service.AnswerQuestion(apiKey, req)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") || strings.Contains(err.Error(), "no valid") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "quota exceeded") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
