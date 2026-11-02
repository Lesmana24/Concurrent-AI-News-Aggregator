package models

import "time"

// TopicCluster groups related news articles reporting on the same event across media outlets.
type TopicCluster struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Title             string    `gorm:"not null" json:"title"`              // Synthesized topic title (e.g. "Putusan MK & Status Wapres Gibran")
	Summary           string    `gorm:"type:text" json:"summary"`           // Consensus facts and overview
	ImageURL          string    `gorm:"type:text;column:image_url" json:"image_url"`
	SecondaryImageURL string    `gorm:"type:text;column:secondary_image_url" json:"secondary_image_url"`
	ArticleCount      int       `json:"article_count"`                     // Total articles in this cluster
	UniqueOutlets     int       `json:"unique_outlets"`                    // Number of distinct media outlets (e.g. 4)
	Outlets           string    `json:"outlets"`                           // CSV list of outlets (e.g. "Antara, CNN, CNBC")
	HotnessScore      float64   `gorm:"index" json:"hotness_score"`        // Trending score: (Articles*2) + (Outlets*3) + recency
	Centroid          string    `gorm:"type:text" json:"-"`                // Serialized centroid vector
	LastArticleTime   time.Time `json:"last_article_time"`                 // Timestamp of the most recent article in cluster
	Articles          []Article `gorm:"many2many:cluster_articles;" json:"articles,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TrendingClusterItem represents a compact summary item in the trending topics list.
type TrendingClusterItem struct {
	ID                uint      `json:"id"`
	Title             string    `json:"title"`
	Summary           string    `json:"summary"`
	ImageURL          string    `json:"image_url"`
	SecondaryImageURL string    `json:"secondary_image_url"`
	ArticleCount      int       `json:"article_count"`
	UniqueOutlets     int       `json:"unique_outlets"`
	Outlets           []string  `json:"outlets"`
	HotnessScore      float64   `json:"hotness_score"`
	LastUpdated       time.Time `json:"last_updated"`
}

// TrendingClusterResponse represents the API response for trending news topics.
type TrendingClusterResponse struct {
	TotalClusters int                   `json:"total_clusters"`
	ComputedAt    time.Time             `json:"computed_at"`
	Topics        []TrendingClusterItem `json:"topics"`
}

// ClusterDetailResponse represents the detailed API response for a single cluster.
type ClusterDetailResponse struct {
	ID                uint              `json:"id"`
	Title             string            `json:"title"`
	Summary           string            `json:"summary"`
	ImageURL          string            `json:"image_url"`
	SecondaryImageURL string            `json:"secondary_image_url"`
	ArticleCount      int               `json:"article_count"`
	UniqueOutlets     int               `json:"unique_outlets"`
	Outlets           []string          `json:"outlets"`
	HotnessScore      float64           `json:"hotness_score"`
	LastUpdated       time.Time         `json:"last_updated"`
	Articles          []ArticleResponse `json:"articles"`
}

