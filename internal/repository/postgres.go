package repository

import (
	"news-aggregator/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
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
