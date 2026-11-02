package scheduler

import (
	"context"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"time"
)

// BriefingRepository abstracts database interactions needed by the briefing engine.
type BriefingRepository interface {
	GetActiveTopics() ([]string, error)
	GetSubscribersByTopic(topic string) ([]int64, error)
	GetRecentArticlesByTopic(ctx context.Context, topic string, hours int, limit int) ([]models.Article, error)
	DeactivateTelegramChat(chatID int64) error
	GetChatSubscriptions(chatID int64) ([]models.TelegramSubscription, error)
	GetSubscriptionStats() (*repository.SubscriptionStats, error)
}

// Config contains operational parameters for the briefing and push notification engine.
type Config struct {
	Enabled                 bool          `json:"enabled"`
	DailyHours              []string      `json:"daily_hours"`                // e.g. ["07:00", "17:00"]
	IntervalMinutes         int           `json:"interval_minutes"`           // Optional periodic run (0 = disabled, use DailyHours)
	MaxArticlesPerDigest    int           `json:"max_articles_per_digest"`    // Default: 5
	BroadcastRateLimitPerSec int          `json:"broadcast_rate_limit_per_sec"` // Default: 25 (Telegram limit is 30)
}

// DefaultConfig provides recommended production defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:                 true,
		DailyHours:              []string{"07:00", "17:00"},
		IntervalMinutes:         0,
		MaxArticlesPerDigest:    5,
		BroadcastRateLimitPerSec: 25,
	}
}

// TelegramSender is an abstraction for dispatching messages via Telegram.
type TelegramSender interface {
	SendNotification(chatID int64, htmlContent string) error
}

// BroadcastResult details the outcome of a push notification dispatch for a topic.
type BroadcastResult struct {
	TriggeredAt      time.Time     `json:"triggered_at"`
	Duration         time.Duration `json:"duration" swaggertype:"primitive,integer"`
	Topic            string        `json:"topic"`
	TotalSubscribers int           `json:"total_subscribers"`
	SuccessCount     int           `json:"success_count"`
	FailedCount      int           `json:"failed_count"`
	Errors           []string      `json:"errors,omitempty"`
}

// BroadcastStats represents cumulative health and telemetries for the scheduler.
type BroadcastStats struct {
	TotalBroadcasts int       `json:"total_broadcasts"`
	LastRunTime     time.Time `json:"last_run_time"`
	LastDurationMs  int64     `json:"last_duration_ms"`
	TotalSent       int       `json:"total_sent"`
	TotalFailed     int       `json:"total_failed"`
	IsRunning       bool      `json:"is_running"`
}

// BroadcastRequest represents the JSON payload to manually trigger a broadcast.
type BroadcastRequest struct {
	Topic   string `json:"topic"`             // Optional: specific topic (e.g. "teknologi") or empty for all
	Message string `json:"message,omitempty"` // Optional: custom announcement message; if empty, synthesizes AI briefing
}
