package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"news-aggregator/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetUserByKey(apiKey string) (*models.User, error) {
	args := m.Called(apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) UpdateUserQuota(userID uint, newQuota int) error {
	args := m.Called(userID, newQuota)
	return args.Error(0)
}

func (m *MockRepository) BatchInsertArticles(articles []models.Article) error {
	args := m.Called(articles)
	return args.Error(0)
}

func (m *MockRepository) GetCachedArticles(url string) ([]models.Article, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Article), args.Error(1)
}

type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) Summarize(ctx context.Context, text string) (string, string, error) {
	args := m.Called(ctx, text)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAIService) AnswerQuestion(ctx context.Context, contextArticles, question string) (string, string, error) {
	args := m.Called(ctx, contextArticles, question)
	return args.String(0), args.String(1), args.Error(2)
}

func TestSummarizeArticles_QuotaExceeded(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo)

	user := &models.User{}
	user.Quota = 0
	mockRepo.On("GetUserByKey", "test-key").Return(user, nil)

	urls := []string{"https://example.com/1"}
	res, err := svc.SummarizeArticles("test-key", urls)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestSummarizeArticles_InvalidKey(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo)

	mockRepo.On("GetUserByKey", "invalid-key").Return(nil, fmt.Errorf("user not found"))

	urls := []string{"https://example.com/1"}
	res, err := svc.SummarizeArticles("invalid-key", urls)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestSummarizeArticles_SuccessWithAIService(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>Ini adalah berita pertama yang penting dan memiliki lebih dari seratus karakter agar lolos validasi scraping berita secara utuh.</p></body></html>`))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 10}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("GetCachedArticles", ts.URL).Return([]models.Article{}, nil)
	mockAI.On("Summarize", mock.Anything, mock.Anything).Return("Ringkasan artikel.", "Groq", nil)
	mockRepo.On("BatchInsertArticles", mock.Anything).Return(nil)
	mockRepo.On("UpdateUserQuota", uint(1), 9).Return(nil)

	res, err := svc.SummarizeArticles("valid-key", []string{ts.URL})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "Ringkasan artikel.", res[0].Summary)
	assert.Equal(t, "Groq", res[0].Source)
}

func TestAnswerQuestion_SuccessWithCacheHit(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 5}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)

	cachedArticle := models.Article{
		URL:     "https://example.com/demo-jakarta",
		Content: "Polda Metro Jaya menyiapkan ribuan personel untuk mengamankan 61 titik demo di Jakarta hari ini.",
	}
	mockRepo.On("GetCachedArticles", "https://example.com/demo-jakarta").Return([]models.Article{cachedArticle}, nil)
	mockAI.On("AnswerQuestion", mock.Anything, mock.Anything, "Berapa titik demo?").Return("Ada 61 titik demo yang diamankan oleh kepolisian.", "Groq", nil)
	mockRepo.On("UpdateUserQuota", uint(1), 4).Return(nil)

	req := models.NewsQARequest{
		URLs:     []string{"https://example.com/demo-jakarta"},
		Question: "Berapa titik demo?",
	}

	res, err := svc.AnswerQuestion("valid-key", req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Ada 61 titik demo yang diamankan oleh kepolisian.", res.Answer)
	assert.Equal(t, "Groq", res.Source)
}

func TestAnswerQuestion_QuotaExceeded(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 0}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)

	req := models.NewsQARequest{
		URLs:     []string{"https://example.com/demo-jakarta"},
		Question: "Berapa titik demo?",
	}

	res, err := svc.AnswerQuestion("valid-key", req)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "quota exceeded")
}
