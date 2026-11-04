package http

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/stretchr/testify/require"
)

type mockNewsService struct {
	summarizeFunc func(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error)
}

func (m *mockNewsService) SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
	if m.summarizeFunc != nil {
		return m.summarizeFunc(ctx, apiKey, urls)
	}
	return nil, nil
}

func (m *mockNewsService) AnswerQuestion(ctx context.Context, apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error) {
	return nil, nil
}

func (m *mockNewsService) AnswerQuestionStream(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error) {
	return "", nil, nil
}

func (m *mockNewsService) SemanticSearch(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error) {
	return nil, nil
}

func (m *mockNewsService) GlobalQA(ctx context.Context, apiKey string, req models.GlobalQARequest) (*models.GlobalQAResponse, error) {
	return nil, nil
}

func (m *mockNewsService) CompareArticles(ctx context.Context, apiKey string, req models.NewsCompareRequest) (*models.NewsCompareResponse, error) {
	return nil, nil
}

func (m *mockNewsService) GetUser(apiKey string) (*models.User, error) {
	return nil, nil
}

func (m *mockNewsService) WithCache(c cache.CacheService) service.NewsService {
	return m
}

func (m *mockNewsService) WithEmbedding(emb ai.EmbeddingService) service.NewsService {
	return m
}

func setupTestRouter(svc service.NewsService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewNewsHandler(svc)
	r.POST("/api/v1/summarize", handler.Summarize)
	return r
}

func TestSummarize_PluralURLs_Success(t *testing.T) {
	mockSvc := &mockNewsService{
		summarizeFunc: func(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
			assert.Equal(t, "test-api-key", apiKey)
			assert.Equal(t, []string{"https://example.com/berita-1", "https://example.com/berita-2"}, urls)
			return []models.ArticleResponse{
				{
					Title:   "Berita 1",
					URL:     "https://example.com/berita-1",
					Summary: "Ringkasan berita 1.",
					Source:  "Example",
				},
				{
					Title:   "Berita 2",
					URL:     "https://example.com/berita-2",
					Summary: "Ringkasan berita 2.",
					Source:  "Example",
				},
			}, nil
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"urls": []string{"https://example.com/berita-1", "https://example.com/berita-2"},
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/summarize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "test-api-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []models.ArticleResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, "Berita 1", resp[0].Title)
}

func TestSummarize_SingularURL_Success(t *testing.T) {
	mockSvc := &mockNewsService{
		summarizeFunc: func(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
			assert.Equal(t, []string{"https://example.com/single-news"}, urls)
			return []models.ArticleResponse{
				{
					Title:    "Single Article",
					URL:      "https://example.com/single-news",
					Summary:  "Ringkasan artikel tunggal.",
					Source:   "Example",
					ImageURL: "https://example.com/image.jpg",
				},
			}, nil
		},
	}

	router := setupTestRouter(mockSvc)

	// Frontend sends singular { "url": "..." }
	body, _ := json.Marshal(map[string]interface{}{
		"url": "https://example.com/single-news",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/summarize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "test-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []models.ArticleResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, "Single Article", resp[0].Title)
	assert.Equal(t, "https://example.com/image.jpg", resp[0].ImageURL)
}

func TestSummarize_EmptyBody_Returns400(t *testing.T) {
	mockSvc := &mockNewsService{}
	router := setupTestRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/summarize", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSummarize_InvalidURL_Returns400(t *testing.T) {
	mockSvc := &mockNewsService{}
	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"url": "not-a-valid-url-format",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/summarize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSummarize_QuotaExceeded_Returns403(t *testing.T) {
	mockSvc := &mockNewsService{
		summarizeFunc: func(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
			return nil, errors.New("quota exceeded")
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"urls": []string{"https://example.com/news"},
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/summarize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSummarize_InternalServerError_Returns500(t *testing.T) {
	mockSvc := &mockNewsService{
		summarizeFunc: func(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
			return nil, errors.New("failed to connect to database")
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"urls": []string{"https://example.com/news"},
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/summarize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

