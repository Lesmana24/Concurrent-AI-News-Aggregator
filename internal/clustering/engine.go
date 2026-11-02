package clustering

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/models"
	"regexp"
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

	// For articles missing vector embeddings, generate on-the-fly via embedSvc (capped to 10 articles to avoid latency)
	if e.embedSvc != nil {
		generatedCount := 0
		for _, a := range articles {
			if generatedCount >= 10 {
				break
			}
			if len(embeddingsMap[a.ID]) == 0 {
				textToEmbed := fmt.Sprintf("%s\n\n%s", a.Title, a.Summary)
				if vec, err := e.embedSvc.GenerateEmbedding(ctx, textToEmbed); err == nil && len(vec) > 0 {
					embeddingsMap[a.ID] = vec
					generatedCount++
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
		var bestRequiredThreshold float32 = e.cfg.SimilarityThreshold

		for cIdx, c := range clusters {
			var sim float32
			var requiredThreshold float32 = e.cfg.SimilarityThreshold

			if len(artVec) > 0 && len(c.centroid) > 0 {
				sim = cosineSimilarity(artVec, c.centroid)
			} else {
				// Text similarity fallback based on title and summary token overlap
				// Threshold for Jaccard text overlap is 0.28 (0.78 is only for vector embeddings)
				requiredThreshold = 0.28
				for _, existingArt := range c.articles {
					jSim := jaccardSimilarity(art.Title+" "+art.Summary, existingArt.Title+" "+existingArt.Summary)
					if jSim > sim {
						sim = jSim
					}
				}
			}

			if sim > bestSim {
				bestSim = sim
				bestIdx = cIdx
				bestRequiredThreshold = requiredThreshold
			}
		}

		if bestIdx >= 0 && bestSim >= bestRequiredThreshold {
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

	// 2.1 Agglomerative Cluster Consolidation Pass
	// Merges any overlapping sub-clusters reporting on the same event
	merged := true
	for merged {
		merged = false
		for i := 0; i < len(clusters); i++ {
			for j := i + 1; j < len(clusters); j++ {
				shouldMerge := false

				// Check vector similarity if both centroids exist
				if len(clusters[i].centroid) > 0 && len(clusters[j].centroid) > 0 {
					sim := cosineSimilarity(clusters[i].centroid, clusters[j].centroid)
					if sim >= (e.cfg.SimilarityThreshold - 0.05) {
						shouldMerge = true
					}
				} else {
					// Check cross-article title and summary overlap between clusters
					for _, artI := range clusters[i].articles {
						for _, artJ := range clusters[j].articles {
							if jaccardSimilarity(artI.Title, artJ.Title) >= 0.30 ||
								jaccardSimilarity(artI.Title+" "+artI.Summary, artJ.Title+" "+artJ.Summary) >= 0.28 {
								shouldMerge = true
								break
							}
						}
						if shouldMerge {
							break
						}
					}
				}

				if shouldMerge {
					// Merge cluster j into cluster i
					clusters[i].articles = append(clusters[i].articles, clusters[j].articles...)

					// Deduplicate articles by URL
					seenURLs := make(map[string]bool)
					var uniqueArticles []models.Article
					for _, a := range clusters[i].articles {
						if !seenURLs[a.URL] {
							seenURLs[a.URL] = true
							uniqueArticles = append(uniqueArticles, a)
						}
					}
					clusters[i].articles = uniqueArticles

					if len(clusters[j].centroid) > 0 {
						if len(clusters[i].centroid) == 0 {
							clusters[i].centroid = clusters[j].centroid
						} else {
							clusters[i].centroid = updateCentroid(clusters[i].centroid, clusters[j].centroid, 2)
						}
					}

					// Remove cluster j
					clusters = append(clusters[:j], clusters[j+1:]...)
					merged = true
					break
				}
			}
			if merged {
				break
			}
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

			prompt := "Dari kumpulan artikel berita terkait di atas, rumuskan konsensus peristiwa dalam Bahasa Indonesia baku.\n" +
				"ATURAN MUTLAK:\n" +
				"1. HANYA berikan output persis dengan format: <Judul> || <Intisari>\n" +
				"2. <Judul>: Judul topik utama ringkas, padat, menarik (maksimal 10 kata).\n" +
				"3. <Intisari>: Ringkasan konsensus 2-3 kalimat objektif.\n" +
				"4. DILARANG KERAS menyertakan proses berpikir, bahasa Inggris ('We need to...', 'The relevant article...'), atau tag <think>/<intisari>."

			aiResp, _, err := e.aiSvc.AnswerQuestion(ctx, contextBuilder.String(), prompt)
			if err == nil {
				if cleanTitle, cleanSummary, ok := cleanClusterSynthesis(aiResp); ok {
					title = cleanTitle
					summary = cleanSummary
				}
			}
		}

		var primaryImg, secondaryImg string
		for _, a := range c.articles {
			if a.ImageURL != "" {
				if primaryImg == "" {
					primaryImg = a.ImageURL
				} else if secondaryImg == "" && a.ImageURL != primaryImg {
					secondaryImg = a.ImageURL
					break
				}
			}
		}

		combinedText := title + " " + summary
		if primaryImg == "" {
			primaryImg = ResolveThematicImage(combinedText)
		}
		if secondaryImg == "" && len(c.articles) > 1 {
			secondaryImg = ResolveSecondaryThematicImage(combinedText)
		}

		topicCluster := models.TopicCluster{
			Title:             title,
			Summary:           summary,
			ImageURL:          primaryImg,
			SecondaryImageURL: secondaryImg,
			ArticleCount:      len(c.articles),
			UniqueOutlets:     uniqueOutlets,
			Outlets:           strings.Join(outletList, ", "),
			HotnessScore:      hotness,
			LastArticleTime:   latestTime,
			Articles:          c.articles,
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

		img := c.ImageURL
		if img == "" {
			img = ResolveThematicImage(c.Title + " " + c.Summary)
		}
		secImg := c.SecondaryImageURL
		if secImg == "" && c.ArticleCount > 1 {
			secImg = ResolveSecondaryThematicImage(c.Title + " " + c.Summary)
		}

		items[i] = models.TrendingClusterItem{
			ID:                c.ID,
			Title:             c.Title,
			Summary:           c.Summary,
			ImageURL:          img,
			SecondaryImageURL: secImg,
			ArticleCount:      c.ArticleCount,
			UniqueOutlets:     c.UniqueOutlets,
			Outlets:           outlets,
			HotnessScore:      c.HotnessScore,
			LastUpdated:       c.LastArticleTime,
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

	img := cluster.ImageURL
	if img == "" {
		img = ResolveThematicImage(cluster.Title + " " + cluster.Summary)
	}
	secImg := cluster.SecondaryImageURL
	if secImg == "" && cluster.ArticleCount > 1 {
		secImg = ResolveSecondaryThematicImage(cluster.Title + " " + cluster.Summary)
	}

	articles := make([]models.ArticleResponse, len(cluster.Articles))
	for i, a := range cluster.Articles {
		artImg := a.ImageURL
		if artImg == "" {
			artImg = ResolveThematicImage(a.Title + " " + a.Summary)
		}
		articles[i] = models.ArticleResponse{
			Title:    a.Title,
			URL:      a.URL,
			Summary:  a.Summary,
			Source:   a.Source,
			ImageURL: artImg,
		}
	}

	return &models.ClusterDetailResponse{
		ID:                cluster.ID,
		Title:             cluster.Title,
		Summary:           cluster.Summary,
		ImageURL:          img,
		SecondaryImageURL: secImg,
		ArticleCount:      cluster.ArticleCount,
		UniqueOutlets:     cluster.UniqueOutlets,
		Outlets:           outlets,
		HotnessScore:      cluster.HotnessScore,
		LastUpdated:       cluster.LastArticleTime,
		Articles:          articles,
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

func ResolveThematicImage(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "ekonomi") || strings.Contains(lower, "inflasi") || strings.Contains(lower, "rupiah") || strings.Contains(lower, "saham") || strings.Contains(lower, "ihsg") || strings.Contains(lower, "pajak") || strings.Contains(lower, "anggaran"):
		return "https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "energi") || strings.Contains(lower, "minyak") || strings.Contains(lower, "batu bara") || strings.Contains(lower, "listrik") || strings.Contains(lower, "bbm") || strings.Contains(lower, "pertamina"):
		return "https://images.unsplash.com/photo-1518709268805-4e9042af9f23?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "politik") || strings.Contains(lower, "dpr") || strings.Contains(lower, "presiden") || strings.Contains(lower, "pemilu") || strings.Contains(lower, "pilkada") || strings.Contains(lower, "mk") || strings.Contains(lower, "hukum") || strings.Contains(lower, "kpk") || strings.Contains(lower, "gibran"):
		return "https://images.unsplash.com/photo-1541872703-74c5e44368f9?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "pangan") || strings.Contains(lower, "beras") || strings.Contains(lower, "pertanian") || strings.Contains(lower, "panen") || strings.Contains(lower, "petani"):
		return "https://images.unsplash.com/photo-1500937386664-56d1dfef3854?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "teknologi") || strings.Contains(lower, "ai") || strings.Contains(lower, "digital") || strings.Contains(lower, "cyber") || strings.Contains(lower, "gadget"):
		return "https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "ikn") || strings.Contains(lower, "nusantara") || strings.Contains(lower, "infrastruktur") || strings.Contains(lower, "konstruksi") || strings.Contains(lower, "jalan tol"):
		return "https://images.unsplash.com/photo-1503387762-592deb58ef4e?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "bencana") || strings.Contains(lower, "banjir") || strings.Contains(lower, "gempa") || strings.Contains(lower, "gunung") || strings.Contains(lower, "cuaca"):
		return "https://images.unsplash.com/photo-1527482797697-8795b05a13fe?auto=format&fit=crop&w=800&q=80"
	default:
		return "https://images.unsplash.com/photo-1504711434969-e33886168f5c?auto=format&fit=crop&w=800&q=80"
	}
}

func ResolveSecondaryThematicImage(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "ekonomi") || strings.Contains(lower, "inflasi") || strings.Contains(lower, "rupiah") || strings.Contains(lower, "saham") || strings.Contains(lower, "ihsg"):
		return "https://images.unsplash.com/photo-1590283603385-17ffb3a7f29f?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "energi") || strings.Contains(lower, "minyak") || strings.Contains(lower, "bbm") || strings.Contains(lower, "pertamina"):
		return "https://images.unsplash.com/photo-1473341304170-971dccb5ac1e?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "politik") || strings.Contains(lower, "presiden") || strings.Contains(lower, "dpr") || strings.Contains(lower, "mk") || strings.Contains(lower, "gibran"):
		return "https://images.unsplash.com/photo-1589829545856-d10d557cf95f?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "pangan") || strings.Contains(lower, "beras") || strings.Contains(lower, "pertanian"):
		return "https://images.unsplash.com/photo-1586771107445-d3ca888129ff?auto=format&fit=crop&w=800&q=80"
	case strings.Contains(lower, "teknologi") || strings.Contains(lower, "ai") || strings.Contains(lower, "digital"):
		return "https://images.unsplash.com/photo-1526374965328-7f61d4dc18c5?auto=format&fit=crop&w=800&q=80"
	default:
		return "https://images.unsplash.com/photo-1495020689067-958852a7765e?auto=format&fit=crop&w=800&q=80"
	}
}

func cleanClusterSynthesis(aiResp string) (string, string, bool) {
	// Strip reasoning tags or think blocks
	aiResp = regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(aiResp, "")
	aiResp = regexp.MustCompile(`(?i)</?(?:intisari|judul|summary|title)>`).ReplaceAllString(aiResp, "")
	aiResp = strings.TrimSpace(aiResp)

	if !strings.Contains(aiResp, "||") {
		return "", "", false
	}

	parts := strings.SplitN(aiResp, "||", 2)
	t := strings.TrimSpace(parts[0])
	s := strings.TrimSpace(parts[1])

	// Strip common English prompt leaks or preamble prefixes
	badPrefixes := []string{
		"we need to", "the relevant article", "here is", "based on", "according to",
		"note:", "key points:", "output:", "summary:", "title:", "intisari:", "judul:",
		"i need to", "this article is about",
	}
	tLower := strings.ToLower(t)
	sLower := strings.ToLower(s)
	for _, bp := range badPrefixes {
		if strings.HasPrefix(tLower, bp) || strings.HasPrefix(sLower, bp) {
			return "", "", false
		}
	}

	// Clean accidental quotes or markdown wrap
	t = strings.Trim(t, "\"`*[]")
	s = strings.Trim(s, "\"`*[]")

	if len(t) < 5 || len(t) > 160 || len(s) < 20 {
		return "", "", false
	}

	return t, s, true
}


