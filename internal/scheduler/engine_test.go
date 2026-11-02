package scheduler

import (
	"context"
	"fmt"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"sync"
	"testing"
)

// mockBriefingRepo provides an in-memory implementation of BriefingRepository for unit tests.
type mockBriefingRepo struct {
	mu            sync.Mutex
	subscriptions map[int64][]string // chatID -> list of active topics
	articles      map[string][]models.Article
	deactivated   map[int64]bool
}

func newMockBriefingRepo() *mockBriefingRepo {
	return &mockBriefingRepo{
		subscriptions: make(map[int64][]string),
		articles:      make(map[string][]models.Article),
		deactivated:   make(map[int64]bool),
	}
}

func (m *mockBriefingRepo) Subscribe(chatID int64, topic string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[chatID] = append(m.subscriptions[chatID], topic)
	delete(m.deactivated, chatID)
}

func (m *mockBriefingRepo) GetActiveTopics() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	topicSet := make(map[string]bool)
	for chatID, topics := range m.subscriptions {
		if !m.deactivated[chatID] {
			for _, t := range topics {
				topicSet[t] = true
			}
		}
	}

	var result []string
	for t := range topicSet {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockBriefingRepo) GetSubscribersByTopic(topic string) ([]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []int64
	for chatID, topics := range m.subscriptions {
		if !m.deactivated[chatID] {
			for _, t := range topics {
				if t == topic {
					result = append(result, chatID)
					break
				}
			}
		}
	}
	return result, nil
}

func (m *mockBriefingRepo) GetRecentArticlesByTopic(ctx context.Context, topic string, hours int, limit int) ([]models.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.articles[topic], nil
}

func (m *mockBriefingRepo) DeactivateTelegramChat(chatID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deactivated[chatID] = true
	return nil
}

func (m *mockBriefingRepo) GetChatSubscriptions(chatID int64) ([]models.TelegramSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deactivated[chatID] {
		return nil, nil
	}
	var res []models.TelegramSubscription
	for _, t := range m.subscriptions[chatID] {
		res = append(res, models.TelegramSubscription{
			ChatID:   chatID,
			Topic:    t,
			IsActive: true,
		})
	}
	return res, nil
}

func (m *mockBriefingRepo) GetSubscriptionStats() (*repository.SubscriptionStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &repository.SubscriptionStats{
		TotalActive: int64(len(m.subscriptions)),
	}, nil
}

// mockTelegramSender records dispatched messages for testing.
type mockTelegramSender struct {
	mu           sync.Mutex
	sentMessages map[int64][]string
	failChats    map[int64]error
}

func newMockTelegramSender() *mockTelegramSender {
	return &mockTelegramSender{
		sentMessages: make(map[int64][]string),
		failChats:    make(map[int64]error),
	}
}

func (m *mockTelegramSender) SendNotification(chatID int64, htmlContent string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, shouldFail := m.failChats[chatID]; shouldFail {
		return err
	}

	m.sentMessages[chatID] = append(m.sentMessages[chatID], htmlContent)
	return nil
}

// mockAIService simulates an AI synthesis response.
type mockAIService struct {
	mockResponse string
}

func (m *mockAIService) Summarize(ctx context.Context, content string) (string, string, error) {
	return "Summary", "mock-ai", nil
}

func (m *mockAIService) AnswerQuestion(ctx context.Context, contextText, question string) (string, string, error) {
	if m.mockResponse != "" {
		return m.mockResponse, "mock-ai", nil
	}
	return "1. Poin penting pertama.\n2. Poin penting kedua.\nKesimpulan: Kondisi pasar stabil.", "mock-ai", nil
}

func (m *mockAIService) AnswerQuestionStream(ctx context.Context, contextText, question string, ch chan<- string) (string, error) {
	return "mock-stream", nil
}

func (m *mockAIService) AnalyzeBias(ctx context.Context, articles []ai.ArticleForAnalysis) (*models.NewsCompareResponse, string, error) {
	return nil, "mock-ai", nil
}

func TestBriefingEngine_TriggerAll_Success(t *testing.T) {
	repo := newMockBriefingRepo()
	repo.Subscribe(101, "teknologi")
	repo.Subscribe(102, "teknologi")
	repo.Subscribe(201, "ekonomi")

	repo.articles["teknologi"] = []models.Article{
		{
			Title:   "Perkembangan AI dan Chip Indonesia",
			Summary: "Pemerintah mendorong riset semikonduktor di universitas.",
			Source:  "Antara",
			URL:     "https://antara.com/chip",
		},
	}
	repo.articles["ekonomi"] = []models.Article{
		{
			Title:   "Pajak Kripto Mengalami Peningkatan",
			Summary: "Penerimaan pajak sektor digital mencapai rekor baru.",
			Source:  "CNBC",
			URL:     "https://cnbc.com/pajak",
		},
	}

	sender := newMockTelegramSender()
	aiSvc := &mockAIService{mockResponse: "1. Riset AI chip melesat.\n2. Pajak kripto meningkat."}

	cfg := Config{
		Enabled:                  true,
		BroadcastRateLimitPerSec: 100, // fast for testing
		MaxArticlesPerDigest:     3,
	}

	engine := NewBriefingEngine(cfg, repo, aiSvc, sender)

	results, err := engine.TriggerAll(context.Background(), "", "")
	if err != nil {
		t.Fatalf("unexpected error from TriggerAll: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 topic results, got %d", len(results))
	}

	stats := engine.GetStats()
	if stats.TotalBroadcasts != 1 {
		t.Errorf("expected 1 total broadcast, got %d", stats.TotalBroadcasts)
	}
	if stats.TotalSent != 3 {
		t.Errorf("expected 3 total sent messages, got %d", stats.TotalSent)
	}

	// Verify messages delivered
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.sentMessages[101]) == 0 {
		t.Errorf("expected chat 101 to receive briefing")
	}
	if len(sender.sentMessages[102]) == 0 {
		t.Errorf("expected chat 102 to receive briefing")
	}
	if len(sender.sentMessages[201]) == 0 {
		t.Errorf("expected chat 201 to receive briefing")
	}
}

func TestBriefingEngine_AutoDeactivateBlockedUser(t *testing.T) {
	repo := newMockBriefingRepo()
	repo.Subscribe(999, "teknologi")
	repo.Subscribe(888, "teknologi")

	sender := newMockTelegramSender()
	sender.failChats[999] = fmt.Errorf("Forbidden: bot was blocked by the user")

	cfg := Config{
		Enabled:                  true,
		BroadcastRateLimitPerSec: 100,
		MaxArticlesPerDigest:     3,
	}

	engine := NewBriefingEngine(cfg, repo, &mockAIService{}, sender)

	results, err := engine.TriggerAll(context.Background(), "teknologi", "Pesan Khusus Pengumuman")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", res.SuccessCount)
	}
	if res.FailedCount != 1 {
		t.Errorf("expected 1 failed, got %d", res.FailedCount)
	}

	// Verify chat 999 is now deactivated in mock repo
	subs, err := repo.GetChatSubscriptions(999)
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected chat 999 to be deactivated, but found active: %v", subs)
	}
}

func TestBriefingEngine_PreventConcurrentBroadcast(t *testing.T) {
	repo := newMockBriefingRepo()
	engine := NewBriefingEngine(DefaultConfig(), repo, &mockAIService{}, newMockTelegramSender())
	engine.running = true

	_, err := engine.TriggerAll(context.Background(), "", "")
	if err == nil {
		t.Fatalf("expected error when triggering while already running, got nil")
	}
}
