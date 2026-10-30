package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Repository interface {
	GetUserByKey(apiKey string) (*models.User, error)
	UpdateUserQuota(userID uint, newQuota int) error
	BatchInsertArticles(articles []models.Article) error
	GetCachedArticles(url string) ([]models.Article, error)
}

type NewsService interface {
	SummarizeArticles(apiKey string, urls []string) ([]models.ArticleResponse, error)
	AnswerQuestion(apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error)
	GetUser(apiKey string) (*models.User, error)
}

type newsService struct {
	repo      Repository
	aiService ai.AIService
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

func (s *newsService) GetUser(apiKey string) (*models.User, error) {
	return s.repo.GetUserByKey(apiKey)
}

func (s *newsService) SummarizeArticles(apiKey string, urls []string) ([]models.ArticleResponse, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if user.Quota <= 0 {
		return nil, fmt.Errorf("quota exceeded")
	}

	validURLs, err := FilterURLs(urls, 10)
	if err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	resultsChan := make(chan models.ArticleResponse, len(validURLs))
	errChan := make(chan error, len(validURLs))
	var wg sync.WaitGroup

	for _, url := range validURLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			content, err := s.getOrFetchArticleContent(u)
			if err != nil {
				errChan <- fmt.Errorf("error processing [%s]: %w", u, err)
				return
			}

			summary, provider, err := s.aiService.Summarize(context.Background(), content)
			if err != nil {
				errChan <- fmt.Errorf("AI error [%s]: %w", u, err)
				return
			}

			article := models.Article{
				URL:         u,
				Content:     content,
				Summary:     summary,
				AIProvider:  provider,
				PublishedAt: time.Now(),
			}
			if err := s.repo.BatchInsertArticles([]models.Article{article}); err != nil {
				log.Printf("Warning: failed to cache article %s: %v", u, err)
			}

			resultsChan <- models.ArticleResponse{
				Title:   "Summarized Article",
				URL:     u,
				Summary: summary,
				Source:  provider,
			}
		}(url)
	}

	wg.Wait()
	close(resultsChan)
	close(errChan)

	var finalResults []models.ArticleResponse
	for res := range resultsChan {
		finalResults = append(finalResults, res)
	}

	newQuota := user.Quota - len(validURLs)
	if newQuota < 0 {
		newQuota = 0
	}
	if err := s.repo.UpdateUserQuota(user.ID, newQuota); err != nil {
		log.Printf("Error updating quota for user %d: %v", user.ID, err)
	}

	if len(finalResults) == 0 && len(errChan) > 0 {
		return nil, <-errChan
	}

	return finalResults, nil
}

func (s *newsService) AnswerQuestion(apiKey string, req models.NewsQARequest) (*models.NewsQAResponse, error) {
	user, err := s.GetUser(apiKey)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if user.Quota <= 0 {
		return nil, fmt.Errorf("quota exceeded")
	}

	validURLs, err := FilterURLs(req.URLs, 5)
	if err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	var contextBuilder strings.Builder
	for idx, u := range validURLs {
		content, err := s.getOrFetchArticleContent(u)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve content from [%s]: %w", u, err)
		}

		// Cap content length to max 4000 characters to prevent token overflow
		if len(content) > 4000 {
			content = content[:4000] + "..."
		}

		contextBuilder.WriteString(fmt.Sprintf("\n--- ARTIKEL [%d] (URL: %s) ---\n%s\n", idx+1, u, content))
	}

	answer, provider, err := s.aiService.AnswerQuestion(context.Background(), contextBuilder.String(), req.Question)
	if err != nil {
		return nil, fmt.Errorf("AI Q&A failed: %w", err)
	}

	// Deduct 1 quota per question
	newQuota := user.Quota - 1
	if newQuota < 0 {
		newQuota = 0
	}
	if err := s.repo.UpdateUserQuota(user.ID, newQuota); err != nil {
		log.Printf("Error updating quota for user %d: %v", user.ID, err)
	}

	return &models.NewsQAResponse{
		Question: req.Question,
		Answer:   answer,
		Sources:  validURLs,
		Source:   provider,
	}, nil
}

func (s *newsService) getOrFetchArticleContent(url string) (string, error) {
	// 1. Cache-first lookup
	cached, err := s.repo.GetCachedArticles(url)
	if err == nil && len(cached) > 0 && strings.TrimSpace(cached[0].Content) != "" {
		return cached[0].Content, nil
	}

	// 2. Fetch via scraper
	content, err := s.scrapeContent(url)
	if err != nil {
		return "", err
	}

	if len(strings.TrimSpace(content)) < 80 {
		return "", fmt.Errorf("insufficient readable content found (possibly paywalled or anti-bot protected)")
	}

	return content, nil
}

func (s *newsService) scrapeContent(url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 NewsAggregator/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http error: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Strip out non-content elements
	doc.Find("script, style, nav, footer, header, aside, noscript, .advertisement, .ads").Remove()

	var textBuilder strings.Builder
	doc.Find("p").Each(func(i int, sel *goquery.Selection) {
		t := strings.TrimSpace(sel.Text())
		if len(t) > 20 {
			textBuilder.WriteString(t)
			textBuilder.WriteString(" ")
		}
	})

	result := strings.TrimSpace(textBuilder.String())
	if result == "" {
		return "", fmt.Errorf("no content found in page")
	}

	return result, nil
}
