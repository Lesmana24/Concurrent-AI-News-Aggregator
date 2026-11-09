package clustering

import (
	"context"
	"fmt"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockClusterRepo struct {
	mu          sync.Mutex
	articles    []models.Article
	embeddings  map[uint][]float32
	clusters    []models.TopicCluster
	nextClustID uint
}

func newMockClusterRepo() *mockClusterRepo {
	return &mockClusterRepo{
		embeddings:  make(map[uint][]float32),
		nextClustID: 1,
	}
}

func (m *mockClusterRepo) GetRecentArticles(ctx context.Context, hours int) ([]models.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.articles, nil
}

func (m *mockClusterRepo) GetArticleEmbeddings(ctx context.Context, articleIDs []uint) (map[uint][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[uint][]float32)
	for _, id := range articleIDs {
		if v, ok := m.embeddings[id]; ok {
			res[id] = v
		}
	}
	return res, nil
}

func (m *mockClusterRepo) SaveClusters(ctx context.Context, clusters []models.TopicCluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range clusters {
		if clusters[i].ID == 0 {
			clusters[i].ID = m.nextClustID
			m.nextClustID++
		}
	}
	m.clusters = clusters
	return nil
}

func (m *mockClusterRepo) GetTrendingClusters(ctx context.Context, limit int) ([]models.TopicCluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit > len(m.clusters) {
		limit = len(m.clusters)
	}
	return m.clusters[:limit], nil
}

func (m *mockClusterRepo) GetClusterByID(ctx context.Context, id uint) (*models.TopicCluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.clusters {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}

type mockAIService struct{}

func (m *mockAIService) Summarize(ctx context.Context, content string) (string, string, error) {
	return "Summary", "mock", nil
}

func (m *mockAIService) AnswerQuestion(ctx context.Context, contextText, question string) (string, string, error) {
	return "Sorotan Pajak Kripto RI || Penerimaan pajak dari aset kripto dan fintech digital tembus rekor baru.", "mock", nil
}

func (m *mockAIService) AnswerQuestionStream(ctx context.Context, contextText, question string, ch chan<- string) (string, error) {
	return "stream", nil
}

func (m *mockAIService) AnalyzeBias(ctx context.Context, articles []ai.ArticleForAnalysis) (*models.NewsCompareResponse, string, error) {
	return nil, "mock", nil
}

func TestClusterEngine_CosineClustering_GroupsSimilarArticles(t *testing.T) {
	repo := newMockClusterRepo()

	// Vector 1: Crypto Tax (CNN)
	vecCrypto1 := make([]float32, 768)
	vecCrypto1[0] = 0.95
	vecCrypto1[1] = 0.20

	// Vector 2: Crypto Tax (CNBC) - High Cosine Similarity to vecCrypto1 (~0.99)
	vecCrypto2 := make([]float32, 768)
	vecCrypto2[0] = 0.94
	vecCrypto2[1] = 0.22

	// Vector 3: Andrea Bocelli Concert (Antara) - Orthogonal/low similarity
	vecConcert := make([]float32, 768)
	vecConcert[10] = 0.99

	art1 := models.Article{
		Title:       "Pajak Kripto Tembus Rp 2,18 Triliun",
		Summary:     "Penerimaan pajak sektor kripto meningkat pesat.",
		Source:      "CNN Indonesia",
		URL:         "https://cnnindonesia.com/kripto",
		PublishedAt: time.Now(),
	}
	art1.ID = 1

	art2 := models.Article{
		Title:       "Penerimaan Fintech dan Pajak Kripto Melonjak",
		Summary:     "Sektor ekonomi digital menyumbang triliunan rupiah penerimaan.",
		Source:      "CNBC Indonesia",
		URL:         "https://cnbcindonesia.com/fintech",
		PublishedAt: time.Now(),
	}
	art2.ID = 2

	art3 := models.Article{
		Title:       "Konser Andrea Bocelli Digelar di Jakarta",
		Summary:     "Penyanyi tenor legendaris akan menyapa penggemar di Indonesia.",
		Source:      "Antara News",
		URL:         "https://antaranews.com/bocelli",
		PublishedAt: time.Now(),
	}
	art3.ID = 3

	repo.articles = []models.Article{art1, art2, art3}
	repo.embeddings[1] = vecCrypto1
	repo.embeddings[2] = vecCrypto2
	repo.embeddings[3] = vecConcert

	engine := NewClusterEngine(DefaultConfig(), repo, &mockAIService{}, nil)

	clusters, err := engine.Recompute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during Recompute: %v", err)
	}

	if len(clusters) != 2 {
		t.Fatalf("expected 2 distinct clusters, got %d", len(clusters))
	}

	// Top cluster should be Crypto Tax because it has 2 articles and 2 outlets
	topCluster := clusters[0]
	if topCluster.ArticleCount != 2 {
		t.Errorf("expected top cluster to have 2 articles, got %d", topCluster.ArticleCount)
	}
	if topCluster.UniqueOutlets != 2 {
		t.Errorf("expected top cluster to have 2 unique outlets, got %d", topCluster.UniqueOutlets)
	}
	if topCluster.HotnessScore <= clusters[1].HotnessScore {
		t.Errorf("expected top cluster hotness (%.2f) to be strictly greater than second cluster (%.2f)",
			topCluster.HotnessScore, clusters[1].HotnessScore)
	}
}

func TestClusterEngine_TextFallback_Jaccard(t *testing.T) {
	repo := newMockClusterRepo()

	art1 := models.Article{
		Title:   "Presiden Prabowo Resmikan Batalyon Baru TNI",
		Summary: "Penguatan pertahanan nasional dengan pembentukan satuan baru.",
		Source:  "Antara News",
		URL:     "https://antaranews.com/prabowo-tni",
	}
	art1.ID = 10

	art2 := models.Article{
		Title:   "Prabowo Resmikan Satuan Batalyon Baru TNI",
		Summary: "Sebanyak 204 satuan baru TNI diresmikan oleh Presiden Prabowo.",
		Source:  "Detik News",
		URL:     "https://detik.com/prabowo-tni",
	}
	art2.ID = 11

	art3 := models.Article{
		Title:   "Tips Merawat Rem Motor Agar Tidak Berdecit",
		Summary: "Penyebab rem cakram motor bersuara dan cara mudah mengatasinya.",
		Source:  "Otomotif Media",
		URL:     "https://otomotif.com/rem-motor",
	}
	art3.ID = 12

	repo.articles = []models.Article{art1, art2, art3}
	// No vector embeddings provided -> forces Jaccard text similarity fallback

	cfg := DefaultConfig()
	cfg.SimilarityThreshold = 0.35 // lower threshold for word overlap

	engine := NewClusterEngine(cfg, repo, &mockAIService{}, nil)

	clusters, err := engine.Recompute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters via text fallback, got %d", len(clusters))
	}

	if clusters[0].ArticleCount != 2 {
		t.Errorf("expected TNI cluster to merge 2 articles, got %d", clusters[0].ArticleCount)
	}
}

func TestClusterEngine_GetTrendingAndDetail(t *testing.T) {
	repo := newMockClusterRepo()
	repo.clusters = []models.TopicCluster{
		{
			ID:            1,
			Title:         "Isu Trending 1",
			Summary:       "Ringkasan 1",
			ArticleCount:  3,
			UniqueOutlets: 2,
			Outlets:       "CNN, Antara",
			HotnessScore:  15.5,
			Articles: []models.Article{
				{Title: "Berita A", URL: "https://cnn.com/a"},
			},
		},
	}

	engine := NewClusterEngine(DefaultConfig(), repo, nil, nil)

	trending, err := engine.GetTrending(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trending) != 1 {
		t.Fatalf("expected 1 trending item, got %d", len(trending))
	}
	if trending[0].HotnessScore != 15.5 {
		t.Errorf("expected hotness score 15.5, got %.2f", trending[0].HotnessScore)
	}

	detail, err := engine.GetClusterDetail(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected detail error: %v", err)
	}
	if detail.Title != "Isu Trending 1" {
		t.Errorf("expected title 'Isu Trending 1', got %s", detail.Title)
	}
	if len(detail.Articles) != 1 {
		t.Errorf("expected 1 article in detail, got %d", len(detail.Articles))
	}
}

func TestClusterEngine_PreventConcurrentRecompute(t *testing.T) {
	repo := newMockClusterRepo()
	engine := NewClusterEngine(DefaultConfig(), repo, nil, nil)
	engine.computing = true

	_, err := engine.Recompute(context.Background())
	if err == nil {
		t.Fatalf("expected error when recomputing concurrently, got nil")
	}
}

type concurrentTrackingAIService struct {
	currentActive int32
	maxConcurrent int32
	delay         time.Duration
}

func (m *concurrentTrackingAIService) Summarize(ctx context.Context, content string) (string, string, error) {
	return "Summary", "mock", nil
}

func (m *concurrentTrackingAIService) AnswerQuestion(ctx context.Context, contextText, question string) (string, string, error) {
	active := atomic.AddInt32(&m.currentActive, 1)
	defer atomic.AddInt32(&m.currentActive, -1)

	for {
		max := atomic.LoadInt32(&m.maxConcurrent)
		if active <= max || atomic.CompareAndSwapInt32(&m.maxConcurrent, max, active) {
			break
		}
	}

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case <-time.After(m.delay):
	}

	return "Judul Sintesis Paralel || Ringkasan konsensus berita hasil paralel.", "mock", nil
}

func (m *concurrentTrackingAIService) AnswerQuestionStream(ctx context.Context, contextText, question string, ch chan<- string) (string, error) {
	return "stream", nil
}

func (m *concurrentTrackingAIService) AnalyzeBias(ctx context.Context, articles []ai.ArticleForAnalysis) (*models.NewsCompareResponse, string, error) {
	return nil, "mock", nil
}

func TestClusterEngine_ParallelSynthesis_WorkerPoolLimits(t *testing.T) {
	repo := newMockClusterRepo()

	// Build 3 clusters of 2 articles each (6 articles total)
	var articles []models.Article
	for c := 0; c < 3; c++ {
		for a := 0; a < 2; a++ {
			id := uint(c*2 + a + 1)
			art := models.Article{
				Title:       fmt.Sprintf("Cluster %d Article %d", c+1, a+1),
				Summary:     fmt.Sprintf("Summary of %d-%d", c+1, a+1),
				Source:      fmt.Sprintf("Outlet %d", a+1),
				URL:         fmt.Sprintf("https://outlet%d.com/art%d", a+1, id),
				PublishedAt: time.Now(),
			}
			art.ID = id
			articles = append(articles, art)

			vec := make([]float32, 768)
			vec[c] = 0.99
			repo.embeddings[id] = vec
		}
	}
	repo.articles = articles

	mockAI := &concurrentTrackingAIService{
		delay: 60 * time.Millisecond,
	}

	cfg := DefaultConfig()
	cfg.SynthesisWorkers = 3
	engine := NewClusterEngine(cfg, repo, mockAI, nil)

	start := time.Now()
	clusters, err := engine.Recompute(context.Background())
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(clusters))

	// All 3 clusters should be synthesized in parallel within ~60-150ms instead of sequentially (~180ms+)
	assert.Less(t, duration, 170*time.Millisecond, "Parallel synthesis should be significantly faster than sequential")

	// Max concurrency must not exceed configured worker limit
	assert.LessOrEqual(t, mockAI.maxConcurrent, int32(3), "Concurrency must never exceed SynthesisWorkers limit")
	assert.GreaterOrEqual(t, mockAI.maxConcurrent, int32(2), "At least 2 workers should have executed concurrently")

	for _, cl := range clusters {
		assert.Equal(t, "Judul Sintesis Paralel", cl.Title)
		assert.Equal(t, "Ringkasan konsensus berita hasil paralel.", cl.Summary)
	}
}

func TestClusterEngine_ContextCancellation_AbortsImmediately(t *testing.T) {
	repo := newMockClusterRepo()
	for i := uint(1); i <= 4; i++ {
		art := models.Article{Title: "Article", URL: fmt.Sprintf("https://ex.com/%d", i)}
		art.ID = i
		repo.articles = append(repo.articles, art)
		vec := make([]float32, 768)
		vec[0] = 0.99
		repo.embeddings[i] = vec
	}

	mockAI := &concurrentTrackingAIService{
		delay: 1 * time.Second,
	}

	engine := NewClusterEngine(DefaultConfig(), repo, mockAI, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := engine.Recompute(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}



func TestCleanClusterSynthesis(t *testing.T) {
	raw := `<think>evaluating cluster</think><summary>Kebijakan Baru Transportasi Publik</summary> || Pemerintah daerah meluncurkan armada bus listrik baru.`
	title, summary, ok := cleanClusterSynthesis(raw)
	assert.True(t, ok)
	assert.Equal(t, "Kebijakan Baru Transportasi Publik", title)
	assert.Equal(t, "Pemerintah daerah meluncurkan armada bus listrik baru.", summary)
}

func BenchmarkCleanClusterSynthesis(b *testing.B) {
	raw := `<think>thinking...</think><summary>Judul Berita Utama</summary> || Ringkasan berita terkini.`
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = cleanClusterSynthesis(raw)
	}
}
