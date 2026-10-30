package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func (m *MockRepository) DeductUserQuota(userID uint, amount int) (bool, error) {
	args := m.Called(userID, amount)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) RefundUserQuota(userID uint, amount int) error {
	args := m.Called(userID, amount)
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

func (m *MockRepository) GetArticleByURL(url string) (*models.Article, error) {
	for _, call := range m.ExpectedCalls {
		if call.Method == "GetArticleByURL" {
			args := m.Called(url)
			if args.Get(0) == nil {
				return nil, args.Error(1)
			}
			return args.Get(0).(*models.Article), args.Error(1)
		}
	}
	return nil, nil
}

func (m *MockRepository) SaveArticleEmbedding(articleID uint, embedding []float32) error {
	for _, call := range m.ExpectedCalls {
		if call.Method == "SaveArticleEmbedding" {
			args := m.Called(articleID, embedding)
			return args.Error(0)
		}
	}
	return nil
}

func (m *MockRepository) SearchSimilarArticles(ctx context.Context, queryVector []float32, limit int, minSimilarity float64) ([]models.SemanticSearchResult, error) {
	args := m.Called(ctx, queryVector, limit, minSimilarity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SemanticSearchResult), args.Error(1)
}

func (m *MockRepository) SearchArticlesByKeyword(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error) {
	for _, call := range m.ExpectedCalls {
		if call.Method == "SearchArticlesByKeyword" {
			args := m.Called(ctx, query, limit)
			if args.Get(0) == nil {
				return nil, args.Error(1)
			}
			return args.Get(0).([]models.SemanticSearchResult), args.Error(1)
		}
	}
	return nil, nil
}

func (m *MockRepository) HasVectorSupport() bool {
	for _, call := range m.ExpectedCalls {
		if call.Method == "HasVectorSupport" {
			args := m.Called()
			return args.Bool(0)
		}
	}
	return false
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

func (m *MockAIService) AnswerQuestionStream(ctx context.Context, contextArticles, question string, outChan chan<- string) (string, error) {
	args := m.Called(ctx, contextArticles, question, outChan)
	if outChan != nil && args.Error(1) == nil {
		outChan <- "Streamed answer token."
	}
	return args.String(0), args.Error(1)
}

func (m *MockAIService) AnalyzeBias(ctx context.Context, articles []ai.ArticleForAnalysis) (*models.NewsCompareResponse, string, error) {
	args := m.Called(ctx, articles)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*models.NewsCompareResponse), args.String(1), args.Error(2)
}

func TestSummarizeArticles_QuotaExceeded(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo)

	user := &models.User{Quota: 0}
	user.ID = 1
	mockRepo.On("GetUserByKey", "test-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(false, nil)

	urls := []string{"https://example.com/1"}
	res, err := svc.SummarizeArticles(context.Background(), "test-key", urls)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestSummarizeArticles_InvalidKey(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo)

	mockRepo.On("GetUserByKey", "invalid-key").Return(nil, fmt.Errorf("user not found"))

	urls := []string{"https://example.com/1"}
	res, err := svc.SummarizeArticles(context.Background(), "invalid-key", urls)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestSummarizeArticles_SuccessWithRealTitleExtraction(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
			<head>
				<title>Default Fallback Title</title>
				<meta property="og:title" content="Polisi Siapkan Pengamanan 61 Titik Demo" />
			</head>
			<body>
				<h1>Judul di H1 Tag</h1>
				<p>Ini adalah berita pertama yang penting dan memiliki lebih dari seratus karakter agar lolos validasi scraping berita secara utuh dan komprehensif.</p>
			</body>
		</html>`))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 10}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)
	mockRepo.On("GetCachedArticles", ts.URL).Return([]models.Article{}, nil)
	mockAI.On("Summarize", mock.Anything, mock.Anything).Return("Ringkasan artikel lengkap.", "Groq", nil)
	mockRepo.On("BatchInsertArticles", mock.MatchedBy(func(articles []models.Article) bool {
		return len(articles) == 1 && articles[0].Title == "Polisi Siapkan Pengamanan 61 Titik Demo"
	})).Return(nil)

	res, err := svc.SummarizeArticles(context.Background(), "valid-key", []string{ts.URL})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "Polisi Siapkan Pengamanan 61 Titik Demo", res[0].Title)
	assert.Equal(t, "Ringkasan artikel lengkap.", res[0].Summary)
	assert.Equal(t, "Groq", res[0].Source)
}

func TestSingleflight_AntiCacheStampede(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	var serverHits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverHits, 1)
		time.Sleep(50 * time.Millisecond) // Simulate slow fetch
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
			<head><meta property="og:title" content="Berita Viral Terkini"/></head>
			<body><p>Konten berita viral yang sangat panjang melebihi delapan puluh karakter untuk pengujian singleflight cache stampede.</p></body>
		</html>`))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 100}
	user.ID = 1
	mockRepo.On("GetUserByKey", "concurrent-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)
	mockRepo.On("GetCachedArticles", ts.URL).Return([]models.Article{}, nil)
	mockRepo.On("BatchInsertArticles", mock.Anything).Return(nil)

	var aiCalls int32
	mockAI.On("Summarize", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		atomic.AddInt32(&aiCalls, 1)
		time.Sleep(50 * time.Millisecond)
	}).Return("Ringkasan bersama.", "NVIDIA NIM", nil)

	// Launch 5 concurrent calls requesting the EXACT SAME url
	concurrentCount := 5
	var wg sync.WaitGroup
	results := make([]*models.ArticleResponse, concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := svc.SummarizeArticles(context.Background(), "concurrent-key", []string{ts.URL})
			if assert.NoError(t, err) && len(res) > 0 {
				results[idx] = &res[0]
			}
		}(i)
	}

	wg.Wait()

	// Singleflight should ensure AI was called only ONCE for all concurrent requests!
	assert.Equal(t, int32(1), atomic.LoadInt32(&aiCalls), "AI Summarize should only be executed once via singleflight")
	for i := 0; i < concurrentCount; i++ {
		assert.NotNil(t, results[i])
		assert.Equal(t, "Berita Viral Terkini", results[i].Title)
		assert.Equal(t, "Ringkasan bersama.", results[i].Summary)
	}
}

func TestAnswerQuestion_SuccessWithCacheHit(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 5}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)

	cachedArticle := models.Article{
		Title:   "Demo Jakarta Hari Ini",
		URL:     "https://example.com/demo-jakarta",
		Content: "Polda Metro Jaya menyiapkan ribuan personel untuk mengamankan 61 titik demo di Jakarta hari ini.",
	}
	mockRepo.On("GetCachedArticles", "https://example.com/demo-jakarta").Return([]models.Article{cachedArticle}, nil)
	mockAI.On("AnswerQuestion", mock.Anything, mock.Anything, "Berapa titik demo?").Return("Ada 61 titik demo yang diamankan oleh kepolisian.", "Groq", nil)

	req := models.NewsQARequest{
		URLs:     []string{"https://example.com/demo-jakarta"},
		Question: "Berapa titik demo?",
	}

	res, err := svc.AnswerQuestion(context.Background(), "valid-key", req)
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
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(false, nil)

	req := models.NewsQARequest{
		URLs:     []string{"https://example.com/demo-jakarta"},
		Question: "Berapa titik demo?",
	}

	res, err := svc.AnswerQuestion(context.Background(), "valid-key", req)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestAnswerQuestionStream_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 5}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)

	cachedArticle := models.Article{
		Title:   "Demo Jakarta Hari Ini",
		URL:     "https://example.com/demo-jakarta",
		Content: "Polda Metro Jaya menyiapkan ribuan personel untuk mengamankan 61 titik demo di Jakarta hari ini.",
	}
	mockRepo.On("GetCachedArticles", "https://example.com/demo-jakarta").Return([]models.Article{cachedArticle}, nil)
	mockAI.On("AnswerQuestionStream", mock.Anything, mock.Anything, "Berapa titik demo?", mock.Anything).Return("Groq", nil)

	req := models.NewsQARequest{
		URLs:     []string{"https://example.com/demo-jakarta"},
		Question: "Berapa titik demo?",
	}

	tokenChan := make(chan string, 10)
	provider, sources, err := svc.AnswerQuestionStream(context.Background(), "valid-key", req, tokenChan)
	assert.NoError(t, err)
	assert.Equal(t, "Groq", provider)
	assert.Len(t, sources, 1)

	var received string
	close(tokenChan)
	for tkn := range tokenChan {
		received += tkn
	}
	assert.Equal(t, "Streamed answer token.", received)
}

func TestSemanticSearch_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	expectedResults := []models.SemanticSearchResult{
		{
			ArticleID:  10,
			Title:      "Demo 61 Titik Jakarta",
			URL:        "https://example.com/demo",
			Summary:    "Polisi siapkan ribuan personel.",
			Source:     "Kompas",
			Similarity: 0.88,
		},
	}

	mockRepo.On("SearchSimilarArticles", mock.Anything, mock.Anything, 5, 0.20).Return(expectedResults, nil)

	results, err := svc.SemanticSearch(context.Background(), "pengamanan demo", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Demo 61 Titik Jakarta", results[0].Title)
	assert.Equal(t, 0.88, results[0].Similarity)
}

func TestGlobalQA_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 10}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)

	matchingArticles := []models.SemanticSearchResult{
		{
			ArticleID:  1,
			Title:      "Demo 61 Titik Jakarta",
			URL:        "https://example.com/demo",
			Summary:    "Polisi siapkan ribuan personel di 61 titik demo.",
			Source:     "Detik",
			Similarity: 0.85,
		},
	}
	mockRepo.On("SearchSimilarArticles", mock.Anything, mock.Anything, 3, 0.20).Return(matchingArticles, nil)
	mockAI.On("AnswerQuestion", mock.Anything, mock.Anything, "Berapa titik demo?").Return("Terdapat 61 titik demo di Jakarta.", "Groq", nil)

	res, err := svc.GlobalQA(context.Background(), "valid-key", models.GlobalQARequest{Question: "Berapa titik demo?"})
	assert.NoError(t, err)
	assert.Equal(t, "Terdapat 61 titik demo di Jakarta.", res.Answer)
	assert.Equal(t, "Groq", res.Provider)
	assert.Len(t, res.Sources, 1)
}

func TestGlobalQA_NoRelevantArticlesGracefulNotice(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 10}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)

	mockRepo.On("SearchSimilarArticles", mock.Anything, mock.Anything, 3, 0.20).Return([]models.SemanticSearchResult{}, nil)

	res, err := svc.GlobalQA(context.Background(), "valid-key", models.GlobalQARequest{Question: "Bagaimana resep nasi goreng?"})
	assert.NoError(t, err)
	assert.Contains(t, res.Answer, "tidak ditemukan arsip berita yang relevan")
	assert.Empty(t, res.Sources)
}

func TestCompareArticles_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 10}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)
	mockRepo.On("DeductUserQuota", uint(1), 1).Return(true, nil)

	art1 := models.Article{Title: "Demo 61 Titik", URL: "https://kompas.com/demo", Content: "Polisi amankan 61 titik demo di Jakarta."}
	art2 := models.Article{Title: "Unjuk Rasa Jakarta", URL: "https://detik.com/demo", Content: "Ribuan massa padati kawasan patung kuda."}
	mockRepo.On("GetCachedArticles", "https://kompas.com/demo").Return([]models.Article{art1}, nil)
	mockRepo.On("GetCachedArticles", "https://detik.com/demo").Return([]models.Article{art2}, nil)

	expectedReport := &models.NewsCompareResponse{
		Topic:          "Aksi Unjuk Rasa di Jakarta",
		ConsensusFacts: []string{"Aksi demonstrasi terjadi di wilayah Jakarta."},
		MediaComparisons: []models.MediaFraming{
			{SourceURL: "https://kompas.com/demo", OutletName: "Kompas", Sentiment: "Netral", SentimentScore: 0.1},
			{SourceURL: "https://detik.com/demo", OutletName: "Detik", Sentiment: "Kritis", SentimentScore: -0.2},
		},
		ObjectiveSummary: "Kedua media meliput peristiwa unjuk rasa dengan sudut pandang pengamanan dan situasi lapangan.",
	}
	mockAI.On("AnalyzeBias", mock.Anything, mock.Anything).Return(expectedReport, "Groq", nil)

	req := models.NewsCompareRequest{
		URLs: []string{"https://kompas.com/demo", "https://detik.com/demo"},
	}

	report, err := svc.CompareArticles(context.Background(), "valid-key", req)
	assert.NoError(t, err)
	assert.Equal(t, "Aksi Unjuk Rasa di Jakarta", report.Topic)
	assert.Equal(t, "Groq", report.Provider)
	assert.Len(t, report.MediaComparisons, 2)
}

func TestCompareArticles_RejectsLessThanTwoURLs(t *testing.T) {
	mockRepo := new(MockRepository)
	mockAI := new(MockAIService)
	svc := NewNewsService(mockRepo, mockAI)

	user := &models.User{Quota: 10}
	user.ID = 1
	mockRepo.On("GetUserByKey", "valid-key").Return(user, nil)

	req := models.NewsCompareRequest{
		URLs: []string{"https://kompas.com/demo"},
	}

	report, err := svc.CompareArticles(context.Background(), "valid-key", req)
	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "at least 2 valid news URLs")
}
