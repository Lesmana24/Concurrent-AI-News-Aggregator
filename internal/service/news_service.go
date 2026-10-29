package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
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
	GetUser(apiKey string) (*models.User, error)
}

type newsService struct {
	repo         Repository
	aiSummarizer ai.Summarizer
}

// NewNewsService creates a new NewsService with the specified repository and optional AI summarizer.
// If no summarizer is provided, the default multi-provider FallbackChain is used.
func NewNewsService(repo Repository, summarizers ...ai.Summarizer) NewsService {
	var summarizer ai.Summarizer
	if len(summarizers) > 0 && summarizers[0] != nil {
		summarizer = summarizers[0]
	} else {
		summarizer = ai.NewDefaultSummarizer()
	}

	return &newsService{
		repo:         repo,
		aiSummarizer: summarizer,
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

	resultsChan := make(chan models.ArticleResponse, len(urls))
	errChan := make(chan error, len(urls))
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			content, err := s.scrapeContent(u)
			if err != nil {
				errChan <- fmt.Errorf("scrape error [%s]: %w", u, err)
				return
			}

			summary, provider, err := s.aiSummarizer.Summarize(context.Background(), content)
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

	newQuota := user.Quota - len(urls)
	if err := s.repo.UpdateUserQuota(user.ID, newQuota); err != nil {
		log.Printf("Error updating quota for user %d: %v", user.ID, err)
	}

	if len(finalResults) == 0 && len(errChan) > 0 {
		return nil, <-errChan
	}

	return finalResults, nil
}

func (s *newsService) scrapeContent(url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NewsAggregator/1.0")

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

	var text string
	doc.Find("p").Each(func(i int, sel *goquery.Selection) {
		t := sel.Text()
		text += t + " "
	})

	if text == "" {
		return "", fmt.Errorf("no content found in page")
	}

	return text, nil
}
