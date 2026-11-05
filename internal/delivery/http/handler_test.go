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
	streamFunc    func(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error)
	registerFunc  func(ctx context.Context, email string) (*models.User, error)
	rotateFunc    func(ctx context.Context, currentAPIKey string) (string, string, *models.User, error)
}

func (m *mockNewsService) SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
	if m.summarizeFunc != nil {
		return m.summarizeFunc(ctx, apiKey, urls)
	}
	return nil, nil
}

func (m *mockNewsService) RegisterUser(ctx context.Context, email string) (*models.User, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, email)
	}
	return nil, nil
}

func (m *mockNewsService) RotateAPIKey(ctx context.Context, currentAPIKey string) (string, string, *models.User, error) {
	if m.rotateFunc != nil {
		return m.rotateFunc(ctx, currentAPIKey)
	}
	return "", "", nil, nil
}

func (m *mockNewsService) AnswerQuestion(ctx context.Context, apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error) {
	return nil, nil
}

func (m *mockNewsService) AnswerQuestionStream(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, apiKey, req, outChan)
	}
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
	r.POST("/api/v1/auth/register", handler.RegisterUser)
	r.POST("/api/v1/user/rotate-key", handler.RotateAPIKey)
	r.POST("/api/v1/summarize", handler.Summarize)
	r.POST("/api/v1/news/qa/stream", handler.QAStream)
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

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func newCloseNotifyingRecorder() *closeNotifyingRecorder {
	return &closeNotifyingRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closed:           make(chan bool, 1),
	}
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func TestQAStream_EmitsResetEventOnFailover(t *testing.T) {
	mockSvc := &mockNewsService{
		streamFunc: func(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error) {
			assert.Equal(t, "Apa kabar?", req.Question)
			// Emit initial token, then failover reset, then fresh token
			outChan <- "Sebagian teks terpotong"
			outChan <- ai.StreamControlReset
			outChan <- "Jawaban bersih final"
			return "NVIDIA NIM", []string{"https://example.com/source"}, nil
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"question": "Apa kabar?",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/news/qa/stream", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := newCloseNotifyingRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")

	responseBody := w.Body.String()
	assert.Contains(t, responseBody, "event:token\ndata:Sebagian teks terpotong\n\n")
	assert.Contains(t, responseBody, "event:reset\ndata:{\"reason\":\"provider_failover\",\"status\":\"reset\"}\n\n")
	assert.Contains(t, responseBody, "event:token\ndata:Jawaban bersih final\n\n")
	assert.Contains(t, responseBody, "event:done\n")
	assert.Contains(t, responseBody, "NVIDIA NIM")
}

func TestQAStream_ErrorSuppressesDoneEvent(t *testing.T) {
	mockSvc := &mockNewsService{
		streamFunc: func(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error) {
			outChan <- "Beberapa token awal"
			return "", nil, errors.New("all AI providers failed: [Groq]: timeout")
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"question": "Pertanyaan gagal?",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/news/qa/stream", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := newCloseNotifyingRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	responseBody := w.Body.String()

	assert.Contains(t, responseBody, "event:error\n")
	assert.Contains(t, responseBody, "all AI providers failed")
	assert.NotContains(t, responseBody, "event:done\n", "When stream errors out, event:done must NEVER be emitted")
}



func TestRegisterUser_Returns201(t *testing.T) {
	mockSvc := &mockNewsService{
		registerFunc: func(ctx context.Context, email string) (*models.User, error) {
			assert.Equal(t, "dev@example.com", email)
			u := &models.User{
				Email:  email,
				APIKey: "na_live_test123",
				Quota:  100,
			}
			u.ID = 1
			return u, nil
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]string{"email": "dev@example.com"})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "success", resp["status"])
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "dev@example.com", data["email"])
	assert.Equal(t, "na_live_test123", data["api_key"])
	assert.Equal(t, float64(100), data["quota"])
}

func TestRegisterUser_Duplicate_Returns409(t *testing.T) {
	mockSvc := &mockNewsService{
		registerFunc: func(ctx context.Context, email string) (*models.User, error) {
			return nil, service.ErrEmailAlreadyRegistered
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]string{"email": "existing@example.com"})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already registered")
}

func TestRegisterUser_InvalidEmail_Returns400(t *testing.T) {
	mockSvc := &mockNewsService{
		registerFunc: func(ctx context.Context, email string) (*models.User, error) {
			return nil, service.ErrInvalidEmail
		},
	}

	router := setupTestRouter(mockSvc)

	body, _ := json.Marshal(map[string]string{"email": "invalid-email"})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRotateAPIKey_Returns200(t *testing.T) {
	mockSvc := &mockNewsService{
		rotateFunc: func(ctx context.Context, currentAPIKey string) (string, string, *models.User, error) {
			assert.Equal(t, "na_live_old", currentAPIKey)
			u := &models.User{
				Email:  "dev@example.com",
				APIKey: "na_live_new",
				Quota:  90,
			}
			u.ID = 1
			return "na_live_old", "na_live_new", u, nil
		},
	}

	router := setupTestRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/rotate-key", nil)
	req.Header.Set("X-API-KEY", "na_live_old")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "success", resp["status"])
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "na_live_old", data["old_api_key"])
	assert.Equal(t, "na_live_new", data["new_api_key"])
}

func TestRotateAPIKey_SystemUser_Returns403(t *testing.T) {
	mockSvc := &mockNewsService{
		rotateFunc: func(ctx context.Context, currentAPIKey string) (string, string, *models.User, error) {
			return "", "", nil, service.ErrCannotRotateSystemKey
		},
	}

	router := setupTestRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/rotate-key", nil)
	req.Header.Set("X-API-KEY", "crawler-system-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "System account")
}

func TestRotateAPIKey_MissingAuth_Returns401(t *testing.T) {
	mockSvc := &mockNewsService{}
	router := setupTestRouter(mockSvc)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/user/rotate-key", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
