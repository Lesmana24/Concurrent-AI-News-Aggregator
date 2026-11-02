package repository

import (
	"context"
	"log/slog"
	"news-aggregator/internal/models"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresRepository struct {
	db        *gorm.DB
	hasVector bool
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	repo := &PostgresRepository{db: db}
	repo.InitVectorExtension()
	return repo
}

// InitVectorExtension attempts to enable pgvector and auto-migrate the ArticleEmbedding schema.
func (r *PostgresRepository) InitVectorExtension() (bool, error) {
	if r.db == nil {
		return false, nil
	}

	err := r.db.Exec("CREATE EXTENSION IF NOT EXISTS vector;").Error
	if err != nil {
		slog.Warn("pgvector extension could not be enabled (semantic vector search will be disabled)", slog.String("error", err.Error()))
		r.hasVector = false
		return false, err
	}

	if err := r.db.AutoMigrate(&models.ArticleEmbedding{}); err != nil {
		slog.Warn("ArticleEmbedding migration failed", slog.String("error", err.Error()))
		r.hasVector = false
		return false, err
	}

	// Create HNSW cosine similarity index for sub-millisecond vector lookups
	_ = r.db.Exec("CREATE INDEX IF NOT EXISTS idx_article_embeddings_hnsw ON article_embeddings USING hnsw (embedding vector_cosine_ops);").Error

	r.hasVector = true
	slog.Info("pgvector extension and HNSW index successfully initialized")
	return true, nil
}

func (r *PostgresRepository) HasVectorSupport() bool {
	return r.hasVector
}

func (r *PostgresRepository) GetUserByKey(apiKey string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("api_key = ?", apiKey).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) UpdateUserQuota(userID uint, newQuota int) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("quota", newQuota).Error
}

// DeductUserQuota atomically decrements the user's quota only if current quota >= amount.
// Returns (true, nil) if deducted, (false, nil) if quota is insufficient.
func (r *PostgresRepository) DeductUserQuota(userID uint, amount int) (bool, error) {
	if amount <= 0 {
		return true, nil
	}
	res := r.db.Model(&models.User{}).
		Where("id = ? AND quota >= ?", userID, amount).
		Update("quota", gorm.Expr("quota - ?", amount))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RefundUserQuota atomically increments the user's quota in case of processing failure.
func (r *PostgresRepository) RefundUserQuota(userID uint, amount int) error {
	if amount <= 0 {
		return nil
	}
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("quota", gorm.Expr("quota + ?", amount)).Error
}

func (r *PostgresRepository) BatchInsertArticles(articles []models.Article) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "content", "summary", "ai_provider"}),
	}).Create(&articles).Error
}

func (r *PostgresRepository) GetCachedArticles(url string) ([]models.Article, error) {
	var articles []models.Article
	err := r.db.Where("url = ?", url).Find(&articles).Error
	return articles, err
}

func (r *PostgresRepository) GetArticleByURL(url string) (*models.Article, error) {
	var article models.Article
	if err := r.db.Where("url = ?", url).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// SaveArticleEmbedding upserts dense vector embedding for an article.
func (r *PostgresRepository) SaveArticleEmbedding(articleID uint, embedding []float32) error {
	if !r.hasVector || len(embedding) == 0 {
		return nil
	}
	vec := pgvector.NewVector(embedding)
	record := models.ArticleEmbedding{
		ArticleID: articleID,
		Embedding: vec,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "article_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"embedding"}),
	}).Create(&record).Error
}

// SearchSimilarArticles queries the top-K most similar articles using cosine distance <=> in pgvector.
func (r *PostgresRepository) SearchSimilarArticles(ctx context.Context, queryVector []float32, limit int, minSimilarity float64) ([]models.SemanticSearchResult, error) {
	if !r.hasVector || len(queryVector) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}

	vecStr := pgvector.NewVector(queryVector).String()
	var results []models.SemanticSearchResult

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			a.id AS article_id,
			a.title,
			a.url,
			a.summary,
			a.source,
			(1 - (e.embedding <=> ?)) AS similarity
		FROM articles a
		JOIN article_embeddings e ON a.id = e.article_id
		WHERE (1 - (e.embedding <=> ?)) >= ?
		ORDER BY similarity DESC
		LIMIT ?
	`, vecStr, vecStr, minSimilarity, limit).Scan(&results).Error

	return results, err
}

var idStopWords = map[string]struct{}{
	"apa": {}, "siapa": {}, "kapan": {}, "dimana": {}, "dmana": {}, "bagaimana": {}, "bgmn": {},
	"mengapa": {}, "kenapa": {}, "knp": {}, "apakah": {}, "berita": {}, "kabar": {}, "terbaru": {},
	"tentang": {}, "ttg": {}, "mengenai": {}, "seputar": {}, "terkait": {}, "adalah": {}, "yaitu": {},
	"atau": {}, "dan": {}, "yang": {}, "yg": {}, "ini": {}, "itu": {}, "dari": {}, "pada": {},
	"untuk": {}, "utk": {}, "bisa": {}, "ada": {}, "oleh": {}, "para": {}, "hari": {}, "kemarin": {},
	"besok": {}, "lalu": {}, "info": {}, "informasi": {}, "tolong": {}, "coba": {}, "jelaskan": {},
	"dong": {}, "saja": {}, "kah": {}, "lah": {}, "pun": {},
}

// SearchArticlesByKeyword searches articles in PostgreSQL using smart keyword matching as a fallback.
func (r *PostgresRepository) SearchArticlesByKeyword(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	// Clean punctuation from query
	cleanQuery := strings.Map(func(r rune) rune {
		if strings.ContainsRune("?!.,:;()[]\"'{}<>/\\|@#$%^&*~`+=_-", r) {
			return ' '
		}
		return r
	}, strings.ToLower(strings.TrimSpace(query)))

	rawWords := strings.Fields(cleanQuery)
	if len(rawWords) == 0 {
		return nil, nil
	}

	// Filter out Indonesian stop words and short particles
	var meaningfulWords []string
	for _, w := range rawWords {
		if _, isStop := idStopWords[w]; !isStop && len(w) >= 3 {
			meaningfulWords = append(meaningfulWords, w)
			// Add common Indonesian root/stem variants (e.g., demonstrasi / demonstran -> demo)
			if strings.HasPrefix(w, "demonstra") {
				meaningfulWords = append(meaningfulWords, "demo")
			}
		}
	}

	// If everything was stripped by stop words, retain words with length >= 3, or original words
	if len(meaningfulWords) == 0 {
		for _, w := range rawWords {
			if len(w) >= 3 {
				meaningfulWords = append(meaningfulWords, w)
			}
		}
	}
	if len(meaningfulWords) == 0 {
		meaningfulWords = rawWords
	}

	var articles []models.Article

	// Strategy 1: Attempt strict AND match across all meaningful words
	if len(meaningfulWords) > 1 {
		qAnd := r.db.WithContext(ctx).Model(&models.Article{})
		for _, w := range meaningfulWords {
			pattern := "%" + w + "%"
			qAnd = qAnd.Where("LOWER(title) LIKE ? OR LOWER(summary) LIKE ?", pattern, pattern)
		}
		_ = qAnd.Order("id DESC").Limit(limit).Find(&articles).Error
	}

	// Strategy 2: If strict match found nothing or single keyword, search with OR
	if len(articles) == 0 {
		var orConditions []string
		var orArgs []interface{}
		for _, w := range meaningfulWords {
			pattern := "%" + w + "%"
			orConditions = append(orConditions, "LOWER(title) LIKE ? OR LOWER(summary) LIKE ?")
			orArgs = append(orArgs, pattern, pattern)
		}

		if len(orConditions) > 0 {
			clause := strings.Join(orConditions, " OR ")
			_ = r.db.WithContext(ctx).Model(&models.Article{}).
				Where(clause, orArgs...).
				Order("id DESC").
				Limit(limit).
				Find(&articles).Error
		}
	}

	// Strategy 3: If still empty (e.g. generic query), return latest articles
	if len(articles) == 0 {
		_ = r.db.WithContext(ctx).Model(&models.Article{}).
			Order("id DESC").
			Limit(limit).
			Find(&articles).Error
	}

	results := make([]models.SemanticSearchResult, len(articles))
	for i, a := range articles {
		results[i] = models.SemanticSearchResult{
			ArticleID:  a.ID,
			Title:      a.Title,
			URL:        a.URL,
			Summary:    a.Summary,
			Source:     a.Source,
			Similarity: 0.80,
		}
	}

	return results, nil
}

// EnsureSystemUser creates or retrieves a system user for background/bot operations.
func (r *PostgresRepository) EnsureSystemUser(apiKey, email string, defaultQuota int) (*models.User, error) {
	var user models.User
	err := r.db.Where("api_key = ?", apiKey).First(&user).Error
	if err == nil {
		if user.Quota < 100 {
			_ = r.db.Model(&user).Update("quota", defaultQuota).Error
			user.Quota = defaultQuota
		}
		return &user, nil
	}

	user = models.User{
		APIKey: apiKey,
		Email:  email,
		Quota:  defaultQuota,
	}
	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// SubscribeTelegram activates or adds a telegram chat subscription for a topic.
func (r *PostgresRepository) SubscribeTelegram(chatID int64, topic string) error {
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic == "" {
		topic = "berita"
	}

	var sub models.TelegramSubscription
	err := r.db.Where("chat_id = ? AND topic = ?", chatID, topic).First(&sub).Error
	if err == nil {
		return r.db.Model(&sub).Update("is_active", true).Error
	}

	sub = models.TelegramSubscription{
		ChatID:   chatID,
		Topic:    topic,
		IsActive: true,
	}
	return r.db.Create(&sub).Error
}

// UnsubscribeTelegram deactivates all subscriptions for a chat ID.
func (r *PostgresRepository) UnsubscribeTelegram(chatID int64) error {
	return r.db.Model(&models.TelegramSubscription{}).
		Where("chat_id = ?", chatID).
		Update("is_active", false).Error
}

// UnsubscribeTelegramTopic deactivates a specific subscription topic for a chat ID.
// If topic is empty or "all", it deactivates all subscriptions for that chat.
func (r *PostgresRepository) UnsubscribeTelegramTopic(chatID int64, topic string) error {
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic == "" || topic == "all" || topic == "semua" {
		return r.UnsubscribeTelegram(chatID)
	}

	return r.db.Model(&models.TelegramSubscription{}).
		Where("chat_id = ? AND topic = ?", chatID, topic).
		Update("is_active", false).Error
}

// GetChatSubscriptions retrieves active subscriptions for a specific chat ID.
func (r *PostgresRepository) GetChatSubscriptions(chatID int64) ([]models.TelegramSubscription, error) {
	var subs []models.TelegramSubscription
	err := r.db.Where("chat_id = ? AND is_active = ?", chatID, true).Order("topic ASC").Find(&subs).Error
	return subs, err
}

// GetActiveTelegramSubscriptions retrieves all active subscriptions.
func (r *PostgresRepository) GetActiveTelegramSubscriptions() ([]models.TelegramSubscription, error) {
	var subs []models.TelegramSubscription
	err := r.db.Where("is_active = ?", true).Find(&subs).Error
	return subs, err
}

// GetSubscribersByTopic retrieves unique active chat IDs subscribed to a given topic.
func (r *PostgresRepository) GetSubscribersByTopic(topic string) ([]int64, error) {
	topic = strings.TrimSpace(strings.ToLower(topic))
	var chatIDs []int64
	err := r.db.Model(&models.TelegramSubscription{}).
		Where("topic = ? AND is_active = ?", topic, true).
		Distinct("chat_id").
		Pluck("chat_id", &chatIDs).Error
	return chatIDs, err
}

// GetActiveTopics retrieves distinct active topics currently subscribed to.
func (r *PostgresRepository) GetActiveTopics() ([]string, error) {
	var topics []string
	err := r.db.Model(&models.TelegramSubscription{}).
		Where("is_active = ?", true).
		Distinct("topic").
		Pluck("topic", &topics).Error
	return topics, err
}

// DeactivateTelegramChat deactivates all subscriptions for a chat ID (e.g. when user blocked bot).
func (r *PostgresRepository) DeactivateTelegramChat(chatID int64) error {
	return r.db.Model(&models.TelegramSubscription{}).
		Where("chat_id = ?", chatID).
		Update("is_active", false).Error
}

// GetRecentArticlesByTopic retrieves recent articles (default within hours, with fallback) for a topic.
func (r *PostgresRepository) GetRecentArticlesByTopic(ctx context.Context, topic string, hours int, limit int) ([]models.Article, error) {
	if limit <= 0 {
		limit = 5
	}
	if hours <= 0 {
		hours = 24
	}

	topic = strings.TrimSpace(strings.ToLower(topic))
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

	var articles []models.Article

	isGeneric := topic == "" || topic == "berita" || topic == "umum" || topic == "berita umum" || topic == "semua"
	if isGeneric {
		// Try fetching within time window
		err := r.db.WithContext(ctx).Model(&models.Article{}).
			Where("created_at >= ?", cutoff).
			Order("id DESC").
			Limit(limit).
			Find(&articles).Error
		if err == nil && len(articles) > 0 {
			return articles, nil
		}

		// Fallback to latest articles regardless of window
		err = r.db.WithContext(ctx).Model(&models.Article{}).
			Order("id DESC").
			Limit(limit).
			Find(&articles).Error
		return articles, err
	}

	// Topic-specific search
	pattern := "%" + topic + "%"
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("(LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(source) LIKE ?) AND created_at >= ?", pattern, pattern, pattern, cutoff).
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	if err == nil && len(articles) > 0 {
		return articles, nil
	}

	// Fallback without cutoff
	err = r.db.WithContext(ctx).Model(&models.Article{}).
		Where("LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(source) LIKE ?", pattern, pattern, pattern).
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	if err == nil && len(articles) > 0 {
		return articles, nil
	}

	// If still empty for niche topic, fallback to latest articles
	err = r.db.WithContext(ctx).Model(&models.Article{}).
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	return articles, err
}

// SubscriptionStats stores statistics about active subscribers.
type SubscriptionStats struct {
	TotalActive   int64            `json:"total_active"`
	TotalInactive int64            `json:"total_inactive"`
	UniqueUsers   int64            `json:"unique_users"`
	TopicCounts   map[string]int64 `json:"topic_counts"`
}

// GetSubscriptionStats retrieves subscriber counts and breakdown by topic.
func (r *PostgresRepository) GetSubscriptionStats() (*SubscriptionStats, error) {
	var totalActive int64
	var totalInactive int64
	var uniqueUsers int64

	_ = r.db.Model(&models.TelegramSubscription{}).Where("is_active = ?", true).Count(&totalActive).Error
	_ = r.db.Model(&models.TelegramSubscription{}).Where("is_active = ?", false).Count(&totalInactive).Error
	_ = r.db.Model(&models.TelegramSubscription{}).Where("is_active = ?", true).Distinct("chat_id").Count(&uniqueUsers).Error

	type topicRow struct {
		Topic string
		Count int64
	}
	var rows []topicRow
	_ = r.db.Model(&models.TelegramSubscription{}).
		Select("topic, count(*) as count").
		Where("is_active = ?", true).
		Group("topic").
		Scan(&rows).Error

	topicCounts := make(map[string]int64)
	for _, row := range rows {
		topicCounts[row.Topic] = row.Count
	}

	return &SubscriptionStats{
		TotalActive:   totalActive,
		TotalInactive: totalInactive,
		UniqueUsers:   uniqueUsers,
		TopicCounts:   topicCounts,
	}, nil
}
