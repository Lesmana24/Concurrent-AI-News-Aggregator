package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"os"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type NewsService interface {
	SummarizeArticles(apiKey string, urls []string) ([]models.ArticleResponse, error)
	GetUser(apiKey string) (*models.User, error)
}

type newsService struct {
	repo *repository.PostgresRepository
}

func NewNewsService(repo *repository.PostgresRepository) NewsService {
	return &newsService{repo: repo}
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

			summary, provider, err := s.summarizeWithAI(content)
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
	resp, err := http.Get(url)
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

func (s *newsService) summarizeWithAI(content string) (string, string, error) {
	apiURL := os.Getenv("OMNIROUTE_API_URL")
	apiKey := os.Getenv("OMNIROUTE_API_KEY")

	if apiURL == "" || apiKey == "" {
		return "", "", fmt.Errorf("OmniRoute configuration missing in .env")
	}

	requestBody, _ := json.Marshal(map[string]interface{}{
		"prompt": "Summarize the following news content concisely in one paragraph:\n\n" + content,
		"max_tokens": 200,
	})

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("AI API responded with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	summary, ok := result["text"].(string)
	if !ok {
		return "", "", fmt.Errorf("unexpected AI response format")
	}

	provider, _ := result["provider"].(string)
	if provider == "" {
		provider = "OmniRoute"
	}

	return summary, provider, nil
}
