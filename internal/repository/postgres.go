package repository

import (
	"news-aggregator/internal/models"
	"gorm.io/gorm"
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

func (r *PostgresRepository) BatchInsertArticles(articles []models.Article) error {
	return r.db.Create(&articles).Error
}

func (r *PostgresRepository) GetCachedArticles(url string) ([]models.Article, error) {
	var articles []models.Article
	err := r.db.Where("url = ?", url).Find(&articles).Error
	return articles, err
}
