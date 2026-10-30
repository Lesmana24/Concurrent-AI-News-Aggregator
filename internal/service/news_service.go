package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/cache"
	"news-aggregator/internal/models"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/singleflight"
)

type Repository interface {
	GetUserByKey(apiKey string) (*models.User, error)
	UpdateUserQuota(userID uint, newQuota int) error
	DeductUserQuota(userID uint, amount int) (bool, error)
	RefundUserQuota(userID uint, amount int) error
	BatchInsertArticles(articles []models.Article) error
	GetCachedArticles(url string) ([]models.Article, error)
	GetArticleByURL(url string) (*models.Article, error)
	SaveArticleEmbedding(articleID uint, embedding []float32) error
	SearchSimilarArticles(ctx context.Context, queryVector []float32, limit int, minSimilarity float64) ([]models.SemanticSearchResult, error)
	SearchArticlesByKeyword(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error)
	HasVectorSupport() bool
}

type NewsService interface {
	SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error)
	AnswerQuestion(ctx context.Context, apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error)
	AnswerQuestionStream(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (provider string, sources []string, err error)
	SemanticSearch(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error)
	GlobalQA(ctx context.Context, apiKey string, req models.GlobalQARequest) (*models.GlobalQAResponse, error)
	CompareArticles(ctx context.Context, apiKey string, req models.NewsCompareRequest) (*models.NewsCompareResponse, error)
	GetUser(apiKey string) (*models.User, error)
	WithCache(c cache.CacheService) NewsService
	WithEmbedding(emb ai.EmbeddingService) NewsService
}

type newsService struct {
	repo         Repository
	aiService    ai.AIService
	cacheSvc     cache.CacheService
	embeddingSvc ai.EmbeddingService
	sfGroup      singleflight.Group
}

type ScrapedArticle struct {
	Title   string
	Content string
}

// NewNewsService creates a new NewsService with the specified repository and optional AI service.
func NewNewsService(repo Repository, aiServices ...ai.AIService) NewsService {
	var aiSvc ai.AIService
	if len(aiServices) > 0 && aiServices[0] != nil {
		aiSvc = aiServices[0]
	} else {
		aiSvc = ai.NewDefaultAIService()
	}

	return &newsService{
		repo:         repo,
		aiService:    aiSvc,
		embeddingSvc: ai.NewDefaultEmbeddingService(),
	}
}

func (s *newsService) WithCache(c cache.CacheService) NewsService {
	s.cacheSvc = c
	return s
}

func (s *newsService) WithEmbedding(emb ai.EmbeddingService) NewsService {
	s.embeddingSvc = emb
	return s
}

func (s *newsService) GetUser(apiKey string) (*models.User, error) {
	return s.repo.GetUserByKey(apiKey)
}

func (s *newsService) SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	validURLs, err := FilterURLs(urls, 10)
	if err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	// Atomic Quota Deduction Upfront to prevent race conditions
	deducted, err := s.repo.DeductUserQuota(user.ID, len(validURLs))
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if !deducted {
		return nil, fmt.Errorf("quota exceeded")
	}

	resultsChan := make(chan models.ArticleResponse, len(validURLs))
	errChan := make(chan error, len(validURLs))
	var wg sync.WaitGroup

	for _, url := range validURLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			res, err := s.getOrProcessArticle(ctx, u)
			if err != nil {
				errChan <- fmt.Errorf("error processing [%s]: %w", u, err)
				return
			}

			resultsChan <- *res
		}(url)
	}

	wg.Wait()
	close(resultsChan)
	close(errChan)

	var finalResults []models.ArticleResponse
	for res := range resultsChan {
		finalResults = append(finalResults, res)
	}

	// Refund quota for failed URLs
	failedCount := len(validURLs) - len(finalResults)
	if failedCount > 0 {
		if err := s.repo.RefundUserQuota(user.ID, failedCount); err != nil {
			log.Printf("Warning: failed to refund %d quota to user %d: %v", failedCount, user.ID, err)
		}
	}

	if len(finalResults) == 0 && len(errChan) > 0 {
		return nil, <-errChan
	}

	return finalResults, nil
}

func (s *newsService) AnswerQuestion(ctx context.Context, apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	validURLs, err := FilterURLs(req.URLs, 5)
	if err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	// Atomic Quota Deduction Upfront (1 quota per QA question)
	deducted, err := s.repo.DeductUserQuota(user.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if !deducted {
		return nil, fmt.Errorf("quota exceeded")
	}

	refundOnFailure := func() {
		if err := s.repo.RefundUserQuota(user.ID, 1); err != nil {
			log.Printf("Warning: failed to refund quota to user %d: %v", user.ID, err)
		}
	}

	var contextBuilder strings.Builder
	for idx, u := range validURLs {
		title, content, err := s.getOrFetchArticleContent(ctx, u)
		if err != nil {
			refundOnFailure()
			return nil, fmt.Errorf("failed to retrieve content from [%s]: %w", u, err)
		}

		// Smart Chunking: Select the most relevant paragraph chunks matching the question
		relevantContent := SmartSelectChunks(content, req.Question, 3500)
		contextBuilder.WriteString(fmt.Sprintf("\n--- ARTIKEL [%d]: %s (URL: %s) ---\n%s\n", idx+1, title, u, relevantContent))
	}

	answer, provider, err := s.aiService.AnswerQuestion(ctx, contextBuilder.String(), req.Question)
	if err != nil {
		refundOnFailure()
		return nil, fmt.Errorf("AI Q&A failed: %w", err)
	}

	return &models.NewsQAResponse{
		Question: req.Question,
		Answer:   answer,
		Sources:  validURLs,
		Source:   provider,
	}, nil
}

func (s *newsService) AnswerQuestionStream(ctx context.Context, apiKey string, req models.NewsQARequest, outChan chan<- string) (string, []string, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return "", nil, fmt.Errorf("authentication failed: %w", err)
	}

	validURLs, err := FilterURLs(req.URLs, 5)
	if err != nil {
		return "", nil, fmt.Errorf("URL validation failed: %w", err)
	}

	deducted, err := s.repo.DeductUserQuota(user.ID, 1)
	if err != nil {
		return "", nil, fmt.Errorf("database error: %w", err)
	}
	if !deducted {
		return "", nil, fmt.Errorf("quota exceeded")
	}

	refundOnFailure := func() {
		if err := s.repo.RefundUserQuota(user.ID, 1); err != nil {
			log.Printf("Warning: failed to refund quota to user %d: %v", user.ID, err)
		}
	}

	var contextBuilder strings.Builder
	for idx, u := range validURLs {
		title, content, err := s.getOrFetchArticleContent(ctx, u)
		if err != nil {
			refundOnFailure()
			return "", nil, fmt.Errorf("failed to retrieve content from [%s]: %w", u, err)
		}

		relevantContent := SmartSelectChunks(content, req.Question, 3500)
		contextBuilder.WriteString(fmt.Sprintf("\n--- ARTIKEL [%d]: %s (URL: %s) ---\n%s\n", idx+1, title, u, relevantContent))
	}

	provider, err := s.aiService.AnswerQuestionStream(ctx, contextBuilder.String(), req.Question, outChan)
	if err != nil {
		refundOnFailure()
		return "", nil, fmt.Errorf("AI streaming Q&A failed: %w", err)
	}

	return provider, validURLs, nil
}

func (s *newsService) getOrProcessArticle(ctx context.Context, url string) (*models.ArticleResponse, error) {
	cacheKey := "article:summary:" + hashURL(url)

	// 1. Check Redis L1 Cache (Fastest: ~1-2ms)
	if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
		if cachedJSON, err := s.cacheSvc.Get(ctx, cacheKey); err == nil && cachedJSON != "" {
			var res models.ArticleResponse
			if err := json.Unmarshal([]byte(cachedJSON), &res); err == nil && res.Summary != "" {
				res.Source = res.Source + " (Redis L1)"
				return &res, nil
			}
		}
	}

	// 2. Check PostgreSQL L2 Cache
	cached, err := s.repo.GetCachedArticles(url)
	if err == nil && len(cached) > 0 && strings.TrimSpace(cached[0].Summary) != "" {
		res := &models.ArticleResponse{
			Title:   cached[0].Title,
			URL:     cached[0].URL,
			Summary: cached[0].Summary,
			Source:  cached[0].AIProvider + " (Cached)",
		}

		// Warm Redis L1 Cache (TTL: 24h)
		if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
			_ = s.cacheSvc.Set(ctx, cacheKey, res, 24*time.Hour)
		}

		return res, nil
	}

	// 3. Singleflight deduplication to prevent cache stampede
	val, err, _ := s.sfGroup.Do(url, func() (interface{}, error) {
		// Double-check cache in case another flight just finished
		cached, err := s.repo.GetCachedArticles(url)
		if err == nil && len(cached) > 0 && strings.TrimSpace(cached[0].Summary) != "" {
			res := &models.ArticleResponse{
				Title:   cached[0].Title,
				URL:     cached[0].URL,
				Summary: cached[0].Summary,
				Source:  cached[0].AIProvider + " (Cached)",
			}
			if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
				_ = s.cacheSvc.Set(ctx, cacheKey, res, 24*time.Hour)
			}
			return res, nil
		}

		var title, content string
		if len(cached) > 0 && strings.TrimSpace(cached[0].Content) != "" {
			title = cached[0].Title
			content = cached[0].Content
		} else {
			scraped, err := s.scrapeArticle(ctx, url)
			if err != nil {
				return nil, err
			}
			title = scraped.Title
			content = scraped.Content
		}

		summary, provider, err := s.aiService.Summarize(ctx, content)
		if err != nil {
			return nil, fmt.Errorf("AI error [%s]: %w", url, err)
		}

		article := models.Article{
			Title:       title,
			URL:         url,
			Content:     content,
			Summary:     summary,
			AIProvider:  provider,
			PublishedAt: time.Now(),
		}
		if err := s.repo.BatchInsertArticles([]models.Article{article}); err != nil {
			log.Printf("Warning: failed to cache article %s: %v", url, err)
		} else if s.embeddingSvc != nil && s.repo.HasVectorSupport() {
			// Asynchronously generate and persist vector embedding for semantic search
			go func(t, sum, u string) {
				art, err := s.repo.GetArticleByURL(u)
				if err == nil && art != nil && art.ID > 0 {
					textToEmbed := fmt.Sprintf("%s\n\n%s", t, sum)
					if vec, err := s.embeddingSvc.GenerateEmbedding(context.Background(), textToEmbed); err == nil && len(vec) > 0 {
						_ = s.repo.SaveArticleEmbedding(art.ID, vec)
					}
				}
			}(title, summary, url)
		}

		res := &models.ArticleResponse{
			Title:   title,
			URL:     url,
			Summary: summary,
			Source:  provider,
		}

		// Cache in Redis L1
		if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
			_ = s.cacheSvc.Set(ctx, cacheKey, res, 24*time.Hour)
		}

		return res, nil
	})

	if err != nil {
		return nil, err
	}

	res := val.(*models.ArticleResponse)
	return res, nil
}

func (s *newsService) getOrFetchArticleContent(ctx context.Context, url string) (string, string, error) {
	contentKey := "article:content:" + hashURL(url)

	// 1. Check Redis L1 Cache
	if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
		if cachedJSON, err := s.cacheSvc.Get(ctx, contentKey); err == nil && cachedJSON != "" {
			var art models.Article
			if err := json.Unmarshal([]byte(cachedJSON), &art); err == nil && art.Content != "" {
				return art.Title, art.Content, nil
			}
		}
	}

	// 2. Check PostgreSQL L2 Cache
	cached, err := s.repo.GetCachedArticles(url)
	if err == nil && len(cached) > 0 && strings.TrimSpace(cached[0].Content) != "" {
		if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
			_ = s.cacheSvc.Set(ctx, contentKey, cached[0], 24*time.Hour)
		}
		return cached[0].Title, cached[0].Content, nil
	}

	// 3. Singleflight content fetch
	val, err, _ := s.sfGroup.Do("content:"+url, func() (interface{}, error) {
		cached, err := s.repo.GetCachedArticles(url)
		if err == nil && len(cached) > 0 && strings.TrimSpace(cached[0].Content) != "" {
			return cached[0], nil
		}

		scraped, err := s.scrapeArticle(ctx, url)
		if err != nil {
			return nil, err
		}

		article := models.Article{
			Title:       scraped.Title,
			URL:         url,
			Content:     scraped.Content,
			PublishedAt: time.Now(),
		}
		if err := s.repo.BatchInsertArticles([]models.Article{article}); err != nil {
			log.Printf("Warning: failed to cache article content for %s: %v", url, err)
		}

		if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
			_ = s.cacheSvc.Set(ctx, contentKey, article, 24*time.Hour)
		}

		return article, nil
	})

	if err != nil {
		return "", "", err
	}

	art := val.(models.Article)
	return art.Title, art.Content, nil
}

func (s *newsService) scrapeArticle(ctx context.Context, url string) (*ScrapedArticle, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 NewsAggregator/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// 1. Extract Real Title
	title := strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	if title == "" {
		title = strings.TrimSpace(doc.Find(`meta[name="twitter:title"]`).AttrOr("content", ""))
	}
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if title == "" {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}
	if title == "" {
		title = "Summarized Article"
	}

	// 2. Strip non-content elements
	doc.Find("script, style, nav, footer, header, aside, noscript, .advertisement, .ads, svg, form").Remove()

	var textBuilder strings.Builder
	doc.Find("p").Each(func(i int, sel *goquery.Selection) {
		t := strings.TrimSpace(sel.Text())
		if len(t) > 20 {
			textBuilder.WriteString(t)
			textBuilder.WriteString(" ")
		}
	})

	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		return nil, fmt.Errorf("no content found in page")
	}

	if len(content) < 80 {
		return nil, fmt.Errorf("insufficient readable content found (possibly paywalled or anti-bot protected)")
	}

	return &ScrapedArticle{
		Title:   title,
		Content: content,
	}, nil
}

func hashURL(u string) string {
	h := sha256.Sum256([]byte(u))
	return hex.EncodeToString(h[:])
}

// SemanticSearch queries archived news using pgvector cosine similarity,
// with resilient fallback to keyword search if vector embeddings are unpopulated.
func (s *newsService) SemanticSearch(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	var results []models.SemanticSearchResult

	// 1. Attempt Vector Similarity Search if embedding service and vector are available
	if s.embeddingSvc != nil {
		vec, err := s.embeddingSvc.GenerateEmbedding(ctx, query)
		if err == nil && len(vec) > 0 {
			vecResults, err := s.repo.SearchSimilarArticles(ctx, vec, limit, 0.20)
			if err == nil && len(vecResults) > 0 {
				results = vecResults
			}
		}
	}

	// 2. Resilient fallback to keyword search if vector search yielded no results
	if len(results) == 0 {
		kwResults, err := s.repo.SearchArticlesByKeyword(ctx, query, limit)
		if err == nil && len(kwResults) > 0 {
			results = kwResults
		}
	}

	return results, nil
}

// GlobalQA performs RAG across all archived news articles in the vector database.
func (s *newsService) GlobalQA(ctx context.Context, apiKey string, req models.GlobalQARequest) (*models.GlobalQAResponse, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Atomic Quota Deduction Upfront (1 quota per Global Q&A question)
	deducted, err := s.repo.DeductUserQuota(user.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if !deducted {
		return nil, fmt.Errorf("quota exceeded")
	}

	refundOnFailure := func() {
		if err := s.repo.RefundUserQuota(user.ID, 1); err != nil {
			log.Printf("Warning: failed to refund quota to user %d: %v", user.ID, err)
		}
	}

	// 1. Vector Search for top-3 relevant articles in the archive
	results, err := s.SemanticSearch(ctx, req.Question, 3)
	if err != nil || len(results) == 0 {
		return &models.GlobalQAResponse{
			Question: req.Question,
			Answer:   "Maaf, tidak ditemukan arsip berita yang relevan dengan pertanyaan Anda di dalam sistem saat ini.",
			Sources:  []models.SemanticSearchResult{},
			Provider: "local-archive",
		}, nil
	}

	// 2. Build RAG Context
	var contextBuilder strings.Builder
	for idx, res := range results {
		contextBuilder.WriteString(fmt.Sprintf("\n--- ARTIKEL [%d]: %s (Sumber: %s, URL: %s, Similarity: %.2f) ---\n%s\n",
			idx+1, res.Title, res.Source, res.URL, res.Similarity, res.Summary))
	}

	// 3. Query AI Service with strict facts from retrieved context
	answer, provider, err := s.aiService.AnswerQuestion(ctx, contextBuilder.String(), req.Question)
	if err != nil {
		refundOnFailure()
		return nil, fmt.Errorf("global AI Q&A failed: %w", err)
	}

	return &models.GlobalQAResponse{
		Question: req.Question,
		Answer:   answer,
		Sources:  results,
		Provider: provider,
	}, nil
}

// CompareArticles performs multi-perspective media bias and framing analysis across 2-5 news articles.
func (s *newsService) CompareArticles(ctx context.Context, apiKey string, req models.NewsCompareRequest) (*models.NewsCompareResponse, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	validURLs, err := FilterURLs(req.URLs, 5)
	if err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}
	if len(validURLs) < 2 {
		return nil, fmt.Errorf("at least 2 valid news URLs are required for comparative bias analysis")
	}

	// 1. Check Redis L1 Cache (Derived from sorted URLs)
	sortedURLs := make([]string, len(validURLs))
	copy(sortedURLs, validURLs)
	sort.Strings(sortedURLs)
	cacheKey := "article:bias:" + hashURL(strings.Join(sortedURLs, "|"))

	if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
		if cachedJSON, err := s.cacheSvc.Get(ctx, cacheKey); err == nil && cachedJSON != "" {
			var cachedReport models.NewsCompareResponse
			if err := json.Unmarshal([]byte(cachedJSON), &cachedReport); err == nil && cachedReport.Topic != "" {
				cachedReport.Provider = cachedReport.Provider + " (Redis L1)"
				return &cachedReport, nil
			}
		}
	}

	// 2. Deduct user quota (1 quota per comparison analysis)
	deducted, err := s.repo.DeductUserQuota(user.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if !deducted {
		return nil, fmt.Errorf("quota exceeded")
	}

	refundOnFailure := func() {
		if err := s.repo.RefundUserQuota(user.ID, 1); err != nil {
			log.Printf("Warning: failed to refund quota to user %d: %v", user.ID, err)
		}
	}

	// 3. Fetch/Scrape article contents concurrently
	articlesForAnalysis := make([]ai.ArticleForAnalysis, len(validURLs))
	type fetchResult struct {
		index   int
		article ai.ArticleForAnalysis
		err     error
	}

	resultsChan := make(chan fetchResult, len(validURLs))
	var wg sync.WaitGroup

	for idx, u := range validURLs {
		wg.Add(1)
		go func(i int, articleURL string) {
			defer wg.Done()
			title, content, err := s.getOrFetchArticleContent(ctx, articleURL)
			if err != nil {
				resultsChan <- fetchResult{index: i, err: err}
				return
			}
			outlet := extractOutletName(articleURL)
			contentSnippet := SmartSelectChunks(content, title, 2500)
			resultsChan <- fetchResult{
				index: i,
				article: ai.ArticleForAnalysis{
					URL:     articleURL,
					Outlet:  outlet,
					Title:   title,
					Content: contentSnippet,
				},
			}
		}(idx, u)
	}

	wg.Wait()
	close(resultsChan)

	for res := range resultsChan {
		if res.err != nil {
			refundOnFailure()
			return nil, fmt.Errorf("failed to retrieve content from article [%d]: %w", res.index+1, res.err)
		}
		articlesForAnalysis[res.index] = res.article
	}

	// 4. Call AI comparative bias analysis engine
	report, provider, err := s.aiService.AnalyzeBias(ctx, articlesForAnalysis)
	if err != nil {
		refundOnFailure()
		return nil, fmt.Errorf("media bias analysis failed: %w", err)
	}
	report.Provider = provider

	// 5. Cache in Redis L1 (TTL: 24h)
	if s.cacheSvc != nil && s.cacheSvc.IsAvailable() {
		_ = s.cacheSvc.Set(ctx, cacheKey, report, 24*time.Hour)
	}

	return report, nil
}

func extractOutletName(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "Media Online"
	}
	hostname := strings.TrimPrefix(parsed.Hostname(), "www.")
	parts := strings.Split(hostname, ".")
	if len(parts) >= 2 {
		outlet := parts[0]
		if len(parts) >= 3 && (parts[0] == "nasional" || parts[0] == "news" || parts[0] == "regional" || parts[0] == "koran") {
			outlet = parts[1]
		}
		if len(outlet) > 0 {
			return strings.ToUpper(string(outlet[0])) + outlet[1:]
		}
	}
	return hostname
}
