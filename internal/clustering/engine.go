package clustering

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
	"sort"
	"strings"
	"sync"
	"time"
)

// ClusterEngine orchestrates topic detection and hotness ranking across multi-outlet news.
type ClusterEngine struct {
	cfg       Config
	repo      ClusterRepository
	aiSvc     ai.AIService
	embedSvc  ai.EmbeddingService
	stats     ClusterStats
	mu        sync.RWMutex
	computing bool
}

// NewClusterEngine creates a new instance of ClusterEngine.
func NewClusterEngine(cfg Config, repo ClusterRepository, aiSvc ai.AIService, embedSvc ai.EmbeddingService) *ClusterEngine {
	if cfg.SimilarityThreshold <= 0 || cfg.SimilarityThreshold > 1 {
		cfg.SimilarityThreshold = 0.78
	}
	if cfg.HoursWindow <= 0 {
		cfg.HoursWindow = 48
	}
	if cfg.MinArticlesForTrending <= 0 {
		cfg.MinArticlesForTrending = 1
	}
	if cfg.RecomputeIntervalMinutes <= 0 {
		cfg.RecomputeIntervalMinutes = 30
	}

	return &ClusterEngine{
		cfg:      cfg,
		repo:     repo,
		aiSvc:    aiSvc,
		embedSvc: embedSvc,
	}
}

// Start launches the periodic background clustering worker.
func (e *ClusterEngine) Start(ctx context.Context) {
	slog.Info("Starting Topic Clustering Engine daemon...",
		slog.Float64("threshold", float64(e.cfg.SimilarityThreshold)),
		slog.Int("window_hours", e.cfg.HoursWindow),
		slog.Int("interval_minutes", e.cfg.RecomputeIntervalMinutes),
	)

	// Run initial clustering upon startup
	go func() {
		time.Sleep(5 * time.Second)
		if _, err := e.Recompute(context.Background()); err != nil {
			slog.Warn("Initial clustering cycle error", slog.String("error", err.Error()))
		}
	}()

	ticker := time.NewTicker(time.Duration(e.cfg.RecomputeIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Topic Clustering Engine stopped gracefully")
			return
		case <-ticker.C:
			if _, err := e.Recompute(ctx); err != nil {
				slog.Warn("Periodic clustering cycle error", slog.String("error", err.Error()))
			}
		}
	}
}

type rawCluster struct {
	articles []models.Article
	centroid []float32
}

// Recompute recalculates topic clusters from recent articles in the database.
func (e *ClusterEngine) Recompute(ctx context.Context) ([]models.TopicCluster, error) {
	e.mu.Lock()
	if e.computing {
		e.mu.Unlock()
		return nil, fmt.Errorf("clustering computation is already in progress")
	}
	e.computing = true
	e.stats.IsComputing = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.computing = false
		e.stats.IsComputing = false
		e.mu.Unlock()
	}()

	startTime := time.Now()

	if e.repo == nil {
		return nil, fmt.Errorf("cluster repository is not configured")
	}

	articles, err := e.repo.GetRecentArticles(ctx, e.cfg.HoursWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent articles: %w", err)
	}

	if len(articles) == 0 {
		return nil, nil
	}

	// 1. Gather vector embeddings for articles
	articleIDs := make([]uint, len(articles))
	for i, a := range articles {
		articleIDs[i] = a.ID
	}

	embeddingsMap, _ := e.repo.GetArticleEmbeddings(ctx, articleIDs)
	if embeddingsMap == nil {
		embeddingsMap = make(map[uint][]float32)
	}

	// For articles missing vector embeddings, generate on-the-fly via embedSvc
	if e.embedSvc != nil {
		for _, a := range articles {
			if len(embeddingsMap[a.ID]) == 0 {
				textToEmbed := fmt.Sprintf("%s\n\n%s", a.Title, a.Summary)
				if vec, err := e.embedSvc.GenerateEmbedding(ctx, textToEmbed); err == nil && len(vec) > 0 {
					embeddingsMap[a.ID] = vec
				}
			}
		}
	}

	// 2. Single-Pass Leader-Follower Clustering
	var clusters []rawCluster

	for _, art := range articles {
		artVec := embeddingsMap[art.ID]

		bestIdx := -1
		var bestSim float32 = -1.0

		for cIdx, c := range clusters {
			var sim float32
			if len(artVec) > 0 && len(c.centroid) > 0 {
				sim = cosineSimilarity(artVec, c.centroid)
			} else {
				// Text similarity fallback based on title token overlap
				sim = jaccardSimilarity(art.Title+" "+art.Summary, c.articles[0].Title+" "+c.articles[0].Summary)
			}

			if sim > bestSim {
				bestSim = sim
				bestIdx = cIdx
			}
		}

		if bestIdx >= 0 && bestSim >= e.cfg.SimilarityThreshold {
			// Join existing cluster and update centroid
			target := &clusters[bestIdx]
			target.articles = append(target.articles, art)
			if len(artVec) > 0 && len(target.centroid) > 0 {
				target.centroid = updateCentroid(target.centroid, artVec, len(target.articles))
			}
		} else {
			// Form a new cluster with this article as leader
			newC := rawCluster{
				articles: []models.Article{art},
				centroid: artVec,
			}
			clusters = append(clusters, newC)
		}
	}

	// 3. Score and Synthesize Topic Clusters
	var resultClusters []models.TopicCluster

	for _, c := range clusters {
		if len(c.articles) < e.cfg.MinArticlesForTrending {
			continue
		}

		// Calculate outlets and recency
		outletSet := make(map[string]bool)
		var outletList []string
		var latestTime time.Time

		for _, a := range c.articles {
			src := strings.TrimSpace(a.Source)
			if src == "" {
				src = extractOutletFromURL(a.URL)
			}
			if src != "" && !outletSet[strings.ToLower(src)] {
				outletSet[strings.ToLower(src)] = true
				outletList = append(outletList, src)
			}
			if a.PublishedAt.After(latestTime) {
				latestTime = a.PublishedAt
			}
			if a.CreatedAt.After(latestTime) {
				latestTime = a.CreatedAt
			}
		}

		if latestTime.IsZero() {
			latestTime = time.Now()
		}

		// Calculate Hotness Score
		// Formula: (Articles * 2.0) + (UniqueOutlets * 3.0) + RecencyBonus
		uniqueOutlets := len(outletList)
		if uniqueOutlets == 0 {
			uniqueOutlets = 1
			outletList = []string{"Media Nasional"}
		}

		hotness := float64(len(c.articles)*2) + float64(uniqueOutlets*3)
		hoursAgo := time.Since(latestTime).Hours()
		if hoursAgo <= 6 {
			hotness += 5.0
		} else if hoursAgo <= 12 {
			hotness += 3.0
		} else if hoursAgo <= 24 {
			hotness += 1.5
		}

		// Synthesize Title and Consensus Summary
		title := c.articles[0].Title
		summary := c.articles[0].Summary

		if len(c.articles) > 1 && e.aiSvc != nil {
			var contextBuilder strings.Builder
			for i, a := range c.articles {
				contextBuilder.WriteString(fmt.Sprintf("[%d] %s (%s)\nRingkasan: %s\n\n", i+1, a.Title, a.Source, a.Summary))
			}

			prompt := "Dari kumpulan artikel berita terkait di atas, rumuskan:\n" +
				"1. Judul isu utama yang singkat, padat, dan menarik (maksimal 10 kata).\n" +
				"2. Intisari konsensus peristiwa (2-3 kalimat objektif).\n" +
				"Format: <Judul> || <Intisari>"

			aiResp, _, err := e.aiSvc.AnswerQuestion(ctx, contextBuilder.String(), prompt)
			if err == nil && strings.Contains(aiResp, "||") {
				parts := strings.SplitN(aiResp, "||", 2)
				t := strings.TrimSpace(parts[0])
				s := strings.TrimSpace(parts[1])
				if len(t) > 5 && len(t) < 150 {
					title = t
				}
				if len(s) > 20 {
					summary = s
				}
			}
		}

		topicCluster := models.TopicCluster{
			Title:           title,
			Summary:         summary,
			ArticleCount:    len(c.articles),
			UniqueOutlets:   uniqueOutlets,
			Outlets:         strings.Join(outletList, ", "),
			HotnessScore:    hotness,
			LastArticleTime: latestTime,
			Articles:        c.articles,
		}

		resultClusters = append(resultClusters, topicCluster)
	}

	// Sort by Hotness Score descending
	sort.Slice(resultClusters, func(i, j int) bool {
		return resultClusters[i].HotnessScore > resultClusters[j].HotnessScore
	})

	// Persist to database
	if err := e.repo.SaveClusters(ctx, resultClusters); err != nil {
		slog.Warn("Failed to persist topic clusters", slog.String("error", err.Error()))
	}

	duration := time.Since(startTime)

	// Update telemetry stats
	e.mu.Lock()
	e.stats.LastComputedAt = startTime
	e.stats.LastDurationMs = duration.Milliseconds()
	e.stats.TotalClusters = len(resultClusters)
	e.stats.TotalArticlesClustered = len(articles)
	e.mu.Unlock()

	slog.Info("Topic clustering cycle completed",
		slog.Int("total_clusters", len(resultClusters)),
		slog.Int("articles_processed", len(articles)),
		slog.Duration("duration", duration),
	)

	return resultClusters, nil
}

// GetTrending returns the top trending clusters formatted for consumer API.
func (e *ClusterEngine) GetTrending(ctx context.Context, limit int) ([]models.TrendingClusterItem, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	clusters, err := e.repo.GetTrendingClusters(ctx, limit)
	if err != nil {
		return nil, err
	}

	items := make([]models.TrendingClusterItem, len(clusters))
	for i, c := range clusters {
		var outlets []string
		if c.Outlets != "" {
			for _, o := range strings.Split(c.Outlets, ",") {
				outlets = append(outlets, strings.TrimSpace(o))
			}
		}
		items[i] = models.TrendingClusterItem{
			ID:            c.ID,
			Title:         c.Title,
			Summary:       c.Summary,
			ArticleCount:  c.ArticleCount,
			UniqueOutlets: c.UniqueOutlets,
			Outlets:       outlets,
			HotnessScore:  c.HotnessScore,
			LastUpdated:   c.LastArticleTime,
		}
	}

	return items, nil
}

// GetClusterDetail returns full details and all associated articles for a cluster.
func (e *ClusterEngine) GetClusterDetail(ctx context.Context, id uint) (*models.ClusterDetailResponse, error) {
	cluster, err := e.repo.GetClusterByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, fmt.Errorf("cluster not found")
	}

	var outlets []string
	if cluster.Outlets != "" {
		for _, o := range strings.Split(cluster.Outlets, ",") {
			outlets = append(outlets, strings.TrimSpace(o))
		}
	}

	articles := make([]models.ArticleResponse, len(cluster.Articles))
	for i, a := range cluster.Articles {
		articles[i] = models.ArticleResponse{
			Title:   a.Title,
			URL:     a.URL,
			Summary: a.Summary,
			Source:  a.Source,
		}
	}

	return &models.ClusterDetailResponse{
		ID:            cluster.ID,
		Title:         cluster.Title,
		Summary:       cluster.Summary,
		ArticleCount:  cluster.ArticleCount,
		UniqueOutlets: cluster.UniqueOutlets,
		Outlets:       outlets,
		HotnessScore:  cluster.HotnessScore,
		LastUpdated:   cluster.LastArticleTime,
		Articles:      articles,
	}, nil
}

// GetStats returns current clustering engine performance metrics.
func (e *ClusterEngine) GetStats() ClusterStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func updateCentroid(current []float32, newVec []float32, count int) []float32 {
	if len(current) != len(newVec) || len(current) == 0 {
		return current
	}
	res := make([]float32, len(current))
	n := float32(count)
	for i := range current {
		res[i] = ((current[i] * (n - 1)) + newVec[i]) / n
	}
	return res
}

func jaccardSimilarity(textA, textB string) float32 {
	setA := tokenizeWords(textA)
	setB := tokenizeWords(textB)

	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}

	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

func tokenizeWords(text string) map[string]bool {
	clean := strings.Map(func(r rune) rune {
		if strings.ContainsRune("?!.,:;()[]\"'{}<>/\\|@#$%^&*~`+=_-", r) {
			return ' '
		}
		return r
	}, strings.ToLower(text))

	tokens := strings.Fields(clean)
	m := make(map[string]bool)
	for _, t := range tokens {
		if len(t) >= 4 {
			m[t] = true
		}
	}
	return m
}

func extractOutletFromURL(u string) string {
	lower := strings.ToLower(u)
	switch {
	case strings.Contains(lower, "antaranews"):
		return "Antara News"
	case strings.Contains(lower, "cnnindonesia"):
		return "CNN Indonesia"
	case strings.Contains(lower, "cnbcindonesia"):
		return "CNBC Indonesia"
	case strings.Contains(lower, "republika"):
		return "Republika"
	case strings.Contains(lower, "kompas"):
		return "Kompas"
	case strings.Contains(lower, "detik"):
		return "Detik"
	case strings.Contains(lower, "tempo"):
		return "Tempo"
	default:
		return "Media Nasional"
	}
}

