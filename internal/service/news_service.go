package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/cache"
	"news-aggregator/internal/models"
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
}

type NewsService interface {
	SummarizeArticles(ctx context.Context, apiKey string, urls []string) ([]models.ArticleResponse, error)
	AnswerQuestion(ctx context.Context, apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error)
	GetUser(apiKey string) (*models.User, error)
	WithCache(c cache.CacheService) NewsService
}

type newsService struct {
	repo      Repository
	aiService ai.AIService
	cacheSvc  cache.CacheService
	sfGroup   singleflight.Group
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
		repo:      repo,
		aiService: aiSvc,
	}
}

func (s *newsService) WithCache(c cache.CacheService) NewsService {
	s.cacheSvc = c
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

		// Cap content length to max 4000 characters to prevent token overflow
		if len(content) > 4000 {
			content = content[:4000] + "..."
		}

		contextBuilder.WriteString(fmt.Sprintf("\n--- ARTIKEL [%d]: %s (URL: %s) ---\n%s\n", idx+1, title, u, content))
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
