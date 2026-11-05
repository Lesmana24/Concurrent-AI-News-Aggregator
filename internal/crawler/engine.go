package crawler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"news-aggregator/internal/models"
	"news-aggregator/internal/service"
	"os"
	"strings"
	"sync"
	"time"
)

// CrawlerRepository defines repository operations required by the harvester.
type CrawlerRepository interface {
	GetArticleByURL(url string) (*models.Article, error)
	EnsureSystemUser(apiKey, email string, defaultQuota int) (*models.User, error)
}

// Config holds configuration parameters for the harvester engine.
type Config struct {
	Interval   time.Duration
	MaxPerFeed int
	Timeout    time.Duration
	Enabled    bool
	Sources    []FeedSource
	SystemKey  string
}

// HarvesterEngine periodically polls news feeds, filters new articles,
// and routes them to the AI summarization & embedding pipeline.
type HarvesterEngine struct {
	config       Config
	newsService  service.NewsService
	repo         CrawlerRepository
	httpClient   *http.Client
	stats        CrawlStats
	mu           sync.RWMutex
	systemAPIKey string
}

func generateSecureToken(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// NewHarvesterEngine creates a new autonomous news harvester.
func NewHarvesterEngine(cfg Config, newsSvc service.NewsService, repo CrawlerRepository) *HarvesterEngine {
	if cfg.Interval <= 0 {
		cfg.Interval = 15 * time.Minute
	}
	if cfg.MaxPerFeed <= 0 {
		cfg.MaxPerFeed = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if len(cfg.Sources) == 0 {
		cfg.Sources = DefaultSources()
	}

	systemKey := strings.TrimSpace(cfg.SystemKey)
	if systemKey == "" {
		systemKey = strings.TrimSpace(os.Getenv("CRAWLER_SYSTEM_KEY"))
	}
	if systemKey == "" {
		systemKey = generateSecureToken("crawler")
	}

	if repo != nil {
		_, _ = repo.EnsureSystemUser(systemKey, "crawler@newsaggregator.internal", 999999)
	}

	activeCount := 0
	for _, s := range cfg.Sources {
		if s.Enabled {
			activeCount++
		}
	}

	return &HarvesterEngine{
		config:       cfg,
		newsService:  newsSvc,
		repo:         repo,
		httpClient:   service.NewSafeHTTPClient(cfg.Timeout),
		systemAPIKey: systemKey,
		stats: CrawlStats{
			ActiveFeedCount: activeCount,
		},
	}
}

// Start launches the background polling loop. It blocks until ctx is canceled.
func (e *HarvesterEngine) Start(ctx context.Context) {
	if !e.config.Enabled {
		slog.Info("Autonomous News Harvester is disabled by configuration")
		return
	}

	slog.Info("Starting Autonomous News Harvester daemon...",
		slog.Duration("interval", e.config.Interval),
		slog.Int("active_feeds", e.stats.ActiveFeedCount),
		slog.Int("max_per_feed", e.config.MaxPerFeed),
	)

	// Run initial ingestion run shortly after startup (3 seconds delay)
	select {
	case <-time.After(3 * time.Second):
		if _, err := e.CrawlAll(ctx); err != nil {
			slog.Warn("Initial crawl cycle completed with errors", slog.String("error", err.Error()))
		}
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(e.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Autonomous News Harvester stopped gracefully")
			return
		case <-ticker.C:
			if _, err := e.CrawlAll(ctx); err != nil {
				slog.Warn("Scheduled crawl cycle completed with errors", slog.String("error", err.Error()))
			}
		}
	}
}

// CrawlAll fetches all enabled feeds, deduplicates against existing articles,
// and summarizes and indexes new articles.
func (e *HarvesterEngine) CrawlAll(ctx context.Context) (*CrawlResult, error) {
	e.mu.Lock()
	if e.stats.IsRunning {
		e.mu.Unlock()
		return nil, fmt.Errorf("crawler is already running")
	}
	e.stats.IsRunning = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.stats.IsRunning = false
		e.mu.Unlock()
	}()

	startTime := time.Now()
	res := &CrawlResult{
		StartTime: startTime,
	}

	slog.Info("Starting news harvesting cycle across active feeds...")

	// 1. Collect feed items from all active sources
	var candidateItems []FeedItem
	for _, source := range e.config.Sources {
		if !source.Enabled {
			continue
		}

		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}

		items, err := e.fetchFeed(ctx, source)
		if err != nil {
			errMsg := fmt.Sprintf("[%s] failed to fetch feed: %v", source.Name, err)
			slog.Warn(errMsg)
			res.Errors = append(res.Errors, errMsg)
			continue
		}

		// Limit items per feed
		limit := e.config.MaxPerFeed
		if len(items) > limit {
			items = items[:limit]
		}
		candidateItems = append(candidateItems, items...)
	}

	res.Fetched = len(candidateItems)

	// 2. Filter and deduplicate
	var newURLs []string
	for _, item := range candidateItems {
		cleanURL := strings.TrimSpace(item.URL)
		if cleanURL == "" {
			continue
		}

		if e.repo != nil {
			existing, err := e.repo.GetArticleByURL(cleanURL)
			if err == nil && existing != nil && existing.ID > 0 {
				res.Skipped++
				continue
			}
		}

		newURLs = append(newURLs, cleanURL)
	}

	// 3. Process new articles through news service
	if len(newURLs) > 0 && e.newsService != nil {
		slog.Info("Ingesting new articles to AI pipeline", slog.Int("count", len(newURLs)))
		
		// Ingest with concurrency safety
		for _, u := range newURLs {
			select {
			case <-ctx.Done():
				return res, ctx.Err()
			default:
			}

			summarized, err := e.newsService.SummarizeArticles(ctx, e.systemAPIKey, []string{u})
			if err != nil {
				errMsg := fmt.Sprintf("failed to process article %s: %v", u, err)
				slog.Warn(errMsg)
				res.Errors = append(res.Errors, errMsg)
				continue
			}

			if len(summarized) > 0 {
				res.Indexed++
			}

			// Polite throttle between background articles to preserve cloud AI TPM quota
			select {
			case <-ctx.Done():
				return res, ctx.Err()
			case <-time.After(1500 * time.Millisecond):
			}
		}
	}

	duration := time.Since(startTime)
	res.Duration = duration

	// Update cumulative statistics
	e.mu.Lock()
	e.stats.TotalRuns++
	e.stats.LastRunTime = startTime
	e.stats.LastDurationMs = duration.Milliseconds()
	e.stats.TotalFetched += int64(res.Fetched)
	e.stats.TotalIndexed += int64(res.Indexed)
	e.stats.TotalSkipped += int64(res.Skipped)
	e.stats.TotalErrors += int64(len(res.Errors))
	e.mu.Unlock()

	slog.Info("News harvesting cycle completed",
		slog.Int("fetched", res.Fetched),
		slog.Int("indexed", res.Indexed),
		slog.Int("skipped", res.Skipped),
		slog.Int("errors", len(res.Errors)),
		slog.Duration("duration", duration),
	)

	return res, nil
}

// GetStats returns current crawler statistics.
func (e *HarvesterEngine) GetStats() CrawlStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

func (e *HarvesterEngine) fetchFeed(ctx context.Context, source FeedSource) ([]FeedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; NewsAggregatorBot/2.0; +https://github.com/news-aggregator)")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Drain up to 64 KB of unread body so Go's Transport can return the underlying TCP connection to the idle pool
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed server returned status %d", resp.StatusCode)
	}

	// 1. Early Content-Length check to save bandwidth
	const maxFeedBytes = 5 * 1024 * 1024 // 5 MB
	if resp.ContentLength > maxFeedBytes {
		return nil, fmt.Errorf("feed content length (%d bytes) exceeds %d MB limit", resp.ContentLength, maxFeedBytes/(1024*1024))
	}

	// 2. Validate Content-Type for XML/RSS/Atom feeds
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if contentType != "" &&
		!strings.Contains(contentType, "xml") &&
		!strings.Contains(contentType, "rss") &&
		!strings.Contains(contentType, "atom") &&
		!strings.Contains(contentType, "text") &&
		!strings.Contains(contentType, "octet-stream") {
		return nil, fmt.Errorf("unsupported feed content-type '%s' (only XML/RSS/Atom accepted)", contentType)
	}

	// 3. Limit feed response body to 5 MB to prevent decompression bombs / memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
	if err != nil {
		return nil, err
	}

	return ParseFeed(body, source.Name, source.Category)
}

