package crawler

import (
	"time"
)

// FeedSource represents a registered news feed source.
type FeedSource struct {
	ID       string `json:"id"`
	Name     string `json:"name"`     // e.g. "Antara News"
	Category string `json:"category"` // e.g. "nasional", "ekonomi", "politik", "teknologi"
	URL      string `json:"url"`      // RSS/Atom Feed URL
	Enabled  bool   `json:"enabled"`
}

// FeedItem represents a single news item extracted from an RSS/Atom feed.
type FeedItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	PublishedAt time.Time `json:"published_at"`
	Source      string    `json:"source"`
	Category    string    `json:"category"`
}

// CrawlStats tracks the execution statistics of the harvester.
type CrawlStats struct {
	TotalRuns       int64     `json:"total_runs"`
	LastRunTime     time.Time `json:"last_run_time"`
	LastDurationMs  int64     `json:"last_duration_ms"`
	TotalFetched    int64     `json:"total_fetched"`
	TotalIndexed    int64     `json:"total_indexed"`
	TotalSkipped    int64     `json:"total_skipped"`
	TotalErrors     int64     `json:"total_errors"`
	ActiveFeedCount int       `json:"active_feed_count"`
	IsRunning       bool      `json:"is_running"`
}

// CrawlResult details the outcome of an individual crawl execution.
type CrawlResult struct {
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration" swaggertype:"primitive,integer"`
	Fetched   int           `json:"fetched"`
	Indexed   int           `json:"indexed"`
	Skipped   int           `json:"skipped"`
	Errors    []string      `json:"errors,omitempty"`
}
