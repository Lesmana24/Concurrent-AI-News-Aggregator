package http

import (
	"net/http"
	"news-aggregator/internal/models"
	"news-aggregator/internal/service"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: urls must be a list of valid URLs"})
		return
	}

	apiKey := c.GetHeader("X-API-KEY")
	res, err := h.service.SummarizeArticles(apiKey, req.URLs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
