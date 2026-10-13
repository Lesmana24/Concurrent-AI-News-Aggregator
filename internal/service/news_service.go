package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"io"
	"news-aggregator/internal/models"
	"os"
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
	repo Repository
}

func NewNewsService(repo Repository) NewsService {
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

	payload := map[string]interface{}{
		"model": "best-template",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a professional news summarizer. Provide a concise, engaging summary of the article in 2-3 sentences. Focus on the main point."},
			{"role": "user", "content": content},
		},
		"temperature": 0.7,
		"max_tokens":  300,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("OmniRoute error: status=%d url=%s body=%s", resp.StatusCode, apiURL, string(b))
		return "", "", fmt.Errorf("AI API responded with status: %d", resp.StatusCode)
	}

	var aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", "", err
	}

	if len(aiResp.Choices) == 0 {
		if aiResp.Error != nil && aiResp.Error.Message != "" {
			return "", "", fmt.Errorf("AI returned error: %s", aiResp.Error.Message)
		}
		return "", "", fmt.Errorf("AI returned no response")
	}

	return aiResp.Choices[0].Message.Content, "OmniRoute", nil
}
