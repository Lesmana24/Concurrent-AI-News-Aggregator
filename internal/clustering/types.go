package clustering

import (
	"context"
	"news-aggregator/internal/models"
	"time"
)

// Config defines the operational parameters for the news clustering engine.
type Config struct {
	SimilarityThreshold      float32 `json:"similarity_threshold"`       // Minimum cosine similarity to join cluster (default: 0.78)
	HoursWindow              int     `json:"hours_window"`               // Recent time window to cluster (default: 48)
	MinArticlesForTrending   int     `json:"min_articles_for_trending"`  // Minimum articles to qualify as a cluster (default: 1)
	RecomputeIntervalMinutes int     `json:"recompute_interval_minutes"` // Periodic recalculation (default: 30)
}

// DefaultConfig provides recommended production defaults.
func DefaultConfig() Config {
	return Config{
		SimilarityThreshold:      0.78,
		HoursWindow:              48,
		MinArticlesForTrending:   1,
		RecomputeIntervalMinutes: 30,
	}
}

// ClusterStats stores telemetry and performance data for the clustering engine.
type ClusterStats struct {
	LastComputedAt         time.Time `json:"last_computed_at"`
	LastDurationMs          int64     `json:"last_duration_ms"`
	TotalClusters          int       `json:"total_clusters"`
	TotalArticlesClustered int       `json:"total_articles_clustered"`
	IsComputing            bool      `json:"is_computing"`
}

// ClusterRepository defines data access requirements for topic clustering.
type ClusterRepository interface {
	GetRecentArticles(ctx context.Context, hours int) ([]models.Article, error)
	GetArticleEmbeddings(ctx context.Context, articleIDs []uint) (map[uint][]float32, error)
	SaveClusters(ctx context.Context, clusters []models.TopicCluster) error
	GetTrendingClusters(ctx context.Context, limit int) ([]models.TopicCluster, error)
	GetClusterByID(ctx context.Context, id uint) (*models.TopicCluster, error)
}

