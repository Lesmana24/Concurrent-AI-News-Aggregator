package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/cache"
	"news-aggregator/internal/models"
	"news-aggregator/internal/service"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCrawlerRepo struct {
	mock.Mock
}

func (m *mockCrawlerRepo) GetArticleByURL(url string) (*models.Article, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *mockCrawlerRepo) EnsureSystemUser(apiKey, email string, defaultQuota int) (*models.User, error) {
	args := m.Called(apiKey, email, defaultQuota)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

type mockNewsService struct {
	mock.Mock
}

func (m *mockNewsService) SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
	args := m.Called(ctx, apiKey, urls)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ArticleResponse), args.Error(1)
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

func (m *mockNewsService) RegisterUser(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}

func (m *mockNewsService) RotateAPIKey(ctx context.Context, currentAPIKey string) (string, string, *models.User, error) {
	return "", "", nil, nil
}

func (m *mockNewsService) WithCache(c cache.CacheService) service.NewsService {
	return m
}

func (m *mockNewsService) WithEmbedding(e ai.EmbeddingService) service.NewsService {
	return m
}

func TestHarvesterEngine_DeduplicationAndIndexing(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	// Create mock RSS server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Media Nasional</title>
    <item>
      <title>Berita Lama yang Sudah Ada di Database</title>
      <link>https://contoh.com/berita-lama</link>
      <pubDate>Mon, 02 Nov 2026 08:00:00 +0700</pubDate>
    </item>
    <item>
      <title>Berita Baru Segar Hari Ini</title>
      <link>https://contoh.com/berita-baru</link>
      <pubDate>Mon, 02 Nov 2026 09:00:00 +0700</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer ts.Close()

	mockRepo := new(mockCrawlerRepo)
	systemUser := &models.User{}
	systemUser.ID = 1
	mockRepo.On("EnsureSystemUser", "crawler-system-key", "crawler@newsaggregator.internal", 999999).Return(systemUser, nil)

	// Old article exists in DB (ID > 0)
	oldArt := &models.Article{Title: "Berita Lama", URL: "https://contoh.com/berita-lama"}
	oldArt.ID = 10
	mockRepo.On("GetArticleByURL", "https://contoh.com/berita-lama").Return(oldArt, nil)

	// New article does NOT exist in DB
	mockRepo.On("GetArticleByURL", "https://contoh.com/berita-baru").Return(nil, nil)

	mockNews := new(mockNewsService)
	// SummarizeArticles should ONLY be called for the new article!
	mockNews.On("SummarizeArticles", mock.Anything, "crawler-system-key", []string{"https://contoh.com/berita-baru"}).Return([]models.ArticleResponse{
		{
			Title:   "Berita Baru Segar Hari Ini",
			URL:     "https://contoh.com/berita-baru",
			Summary: "Ringkasan berita baru.",
		},
	}, nil)

	cfg := Config{
		Interval:   time.Hour,
		MaxPerFeed: 5,
		Timeout:    5 * time.Second,
		Enabled:    true,
		SystemKey:  "crawler-system-key",
		Sources: []FeedSource{
			{
				ID:       "test-feed",
				Name:     "Test Media",
				Category: "nasional",
				URL:      ts.URL,
				Enabled:  true,
			},
		},
	}

	engine := NewHarvesterEngine(cfg, mockNews, mockRepo)
	result, err := engine.CrawlAll(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Fetched)
	assert.Equal(t, 1, result.Skipped, "Expected 1 article to be skipped by deduplication")
	assert.Equal(t, 1, result.Indexed, "Expected 1 new article to be indexed")
	assert.Empty(t, result.Errors)

	stats := engine.GetStats()
	assert.Equal(t, int64(1), stats.TotalRuns)
	assert.Equal(t, int64(2), stats.TotalFetched)
	assert.Equal(t, int64(1), stats.TotalIndexed)
	assert.Equal(t, int64(1), stats.TotalSkipped)
	assert.Equal(t, 1, stats.ActiveFeedCount)
	assert.False(t, stats.IsRunning)

	mockRepo.AssertExpectations(t)
	mockNews.AssertExpectations(t)
}

func TestHarvesterEngine_FeedServerFailureGraceful(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	// Create server that returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	mockRepo := new(mockCrawlerRepo)
	systemUser := &models.User{}
	systemUser.ID = 1
	mockRepo.On("EnsureSystemUser", mock.Anything, mock.Anything, mock.Anything).Return(systemUser, nil)
	mockNews := new(mockNewsService)

	cfg := Config{
		Interval: time.Hour,
		Sources: []FeedSource{
			{ID: "failing-feed", Name: "Fail Media", URL: ts.URL, Enabled: true},
		},
	}

	engine := NewHarvesterEngine(cfg, mockNews, mockRepo)
	result, err := engine.CrawlAll(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Fetched)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "status 500")

	stats := engine.GetStats()
	assert.Equal(t, int64(1), stats.TotalRuns)
	assert.Equal(t, int64(1), stats.TotalErrors)
}

func TestHarvesterEngine_BlocksPrivateAndRedirectFeeds(t *testing.T) {
	// Ensure internal URLs are strictly disallowed
	t.Setenv("ALLOW_INTERNAL_URLS", "false")

	// Server that redirects to AWS/cloud metadata IP
	tsRedirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer tsRedirect.Close()

	mockRepo := new(mockCrawlerRepo)
	systemUser := &models.User{}
	systemUser.ID = 1
	mockRepo.On("EnsureSystemUser", mock.Anything, mock.Anything, mock.Anything).Return(systemUser, nil)
	mockNews := new(mockNewsService)

	cfg := Config{
		Interval: time.Hour,
		Sources: []FeedSource{
			{ID: "direct-private", Name: "Private Feed", URL: "http://127.0.0.1:5432/feed", Enabled: true},
			{ID: "redirect-metadata", Name: "Redirect Feed", URL: tsRedirect.URL, Enabled: true},
		},
	}

	engine := NewHarvesterEngine(cfg, mockNews, mockRepo)
	result, err := engine.CrawlAll(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Fetched)
	// Both sources must fail and record error
	assert.Len(t, result.Errors, 2)
	assert.Contains(t, result.Errors[0], "blocked")
	assert.Contains(t, result.Errors[1], "blocked")
}

func TestHarvesterEngine_RejectsOversizedFeed(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Server reports Content-Length of 10 MB
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Length", "10485760") // 10 MB
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss></rss>"))
	}))
	defer ts.Close()

	mockRepo := new(mockCrawlerRepo)
	systemUser := &models.User{}
	systemUser.ID = 1
	mockRepo.On("EnsureSystemUser", mock.Anything, mock.Anything, mock.Anything).Return(systemUser, nil)
	mockNews := new(mockNewsService)

	cfg := Config{
		Interval: time.Hour,
		Sources: []FeedSource{
			{ID: "oversized", Name: "Oversized Feed", URL: ts.URL, Enabled: true},
		},
	}

	engine := NewHarvesterEngine(cfg, mockNews, mockRepo)
	result, err := engine.CrawlAll(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Fetched)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "exceeds 5 MB limit")
}

func TestHarvesterEngine_RejectsNonXMLContentType(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Server reports Content-Type of video/mp4
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\x00\x00\x00 ftypisom"))
	}))
	defer ts.Close()

	mockRepo := new(mockCrawlerRepo)
	systemUser := &models.User{}
	systemUser.ID = 1
	mockRepo.On("EnsureSystemUser", mock.Anything, mock.Anything, mock.Anything).Return(systemUser, nil)
	mockNews := new(mockNewsService)

	cfg := Config{
		Interval: time.Hour,
		Sources: []FeedSource{
			{ID: "video", Name: "Video Feed", URL: ts.URL, Enabled: true},
		},
	}

	engine := NewHarvesterEngine(cfg, mockNews, mockRepo)
	result, err := engine.CrawlAll(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Fetched)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "unsupported feed content-type")
}
