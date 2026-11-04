package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/cache"
	"news-aggregator/internal/models"
	"news-aggregator/internal/service"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockAuthNewsService struct {
	getUserFunc func(apiKey string) (*models.User, error)
}

func (m *mockAuthNewsService) SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
	return nil, nil
}
func (m *mockAuthNewsService) AnswerQuestion(ctx context.Context, apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error) {
	return nil, nil
}
func (m *mockAuthNewsService) AnswerQuestionStream(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error) {
	return "", nil, nil
}
func (m *mockAuthNewsService) SemanticSearch(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error) {
	return nil, nil
}
func (m *mockAuthNewsService) GlobalQA(ctx context.Context, apiKey string, req models.GlobalQARequest) (*models.GlobalQAResponse, error) {
	return nil, nil
}
func (m *mockAuthNewsService) CompareArticles(ctx context.Context, apiKey string, req models.NewsCompareRequest) (*models.NewsCompareResponse, error) {
	return nil, nil
}
func (m *mockAuthNewsService) GetUser(apiKey string) (*models.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(apiKey)
	}
	return nil, errors.New("user not found")
}
func (m *mockAuthNewsService) WithCache(c cache.CacheService) service.NewsService {
	return m
}
func (m *mockAuthNewsService) WithEmbedding(emb ai.EmbeddingService) service.NewsService {
	return m
}

func setupAuthRouter(svc service.NewsService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIKeyAuth(svc))
	r.GET("/protected", func(c *gin.Context) {
		user, _ := c.Get("user")
		c.JSON(http.StatusOK, gin.H{"status": "ok", "user": user})
	})
	return r
}

func TestAPIKeyAuth_MissingAPIKey_Returns401(t *testing.T) {
	mockSvc := &mockAuthNewsService{}
	router := setupAuthRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "API Key is missing")
}

func TestAPIKeyAuth_InvalidAPIKey_Returns401(t *testing.T) {
	mockSvc := &mockAuthNewsService{
		getUserFunc: func(apiKey string) (*models.User, error) {
			return nil, errors.New("invalid key")
		},
	}
	router := setupAuthRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-KEY", "wrong-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API Key")
}

func TestAPIKeyAuth_SystemUserBlocked_Returns403(t *testing.T) {
	mockSvc := &mockAuthNewsService{
		getUserFunc: func(apiKey string) (*models.User, error) {
			return &models.User{
				APIKey:   apiKey,
				Email:    "crawler@newsaggregator.internal",
				Quota:    999999,
				IsSystem: true, // System account
			}, nil
		},
	}
	router := setupAuthRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-KEY", "crawler-secret-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "System API keys cannot be used for public API access")
}

func TestAPIKeyAuth_QuotaExceeded_Returns403(t *testing.T) {
	mockSvc := &mockAuthNewsService{
		getUserFunc: func(apiKey string) (*models.User, error) {
			return &models.User{
				APIKey:   apiKey,
				Email:    "user@example.com",
				Quota:    0,
				IsSystem: false,
			}, nil
		},
	}
	router := setupAuthRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-KEY", "valid-key-no-quota")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Quota exceeded")
}

func TestAPIKeyAuth_ValidUser_Returns200AndSetsContext(t *testing.T) {
	mockSvc := &mockAuthNewsService{
		getUserFunc: func(apiKey string) (*models.User, error) {
			return &models.User{
				APIKey:   apiKey,
				Email:    "developer@example.com",
				Quota:    50,
				IsSystem: false,
			}, nil
		},
	}
	router := setupAuthRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-KEY", "valid-user-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "developer@example.com")
}

