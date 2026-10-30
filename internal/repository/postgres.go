package repository

import (
	"context"
	"log/slog"
	"news-aggregator/internal/models"
	"strings"

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

// GetActiveTelegramSubscriptions retrieves all active subscriptions.
func (r *PostgresRepository) GetActiveTelegramSubscriptions() ([]models.TelegramSubscription, error) {
	var subs []models.TelegramSubscription
	err := r.db.Where("is_active = ?", true).Find(&subs).Error
	return subs, err
}
