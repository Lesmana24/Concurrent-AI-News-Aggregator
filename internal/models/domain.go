package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	APIKey   string `gorm:"uniqueIndex;not null" json:"api_key"`
	Quota    int    `gorm:"default:100" json:"quota"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	IsSystem bool   `gorm:"default:false" json:"is_system"`
}

type Article struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `gorm:"index:idx_articles_created_at" json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Title       string         `gorm:"not null" json:"title"`
	Content     string         `gorm:"type:text" json:"content"`
	URL         string         `gorm:"uniqueIndex;not null" json:"url"`
	Source      string         `gorm:"index" json:"source"`
	ImageURL    string         `gorm:"type:text;column:image_url" json:"image_url"`
	AIProvider  string         `gorm:"column:ai_provider" json:"ai_provider"`
	PublishedAt time.Time      `gorm:"index:idx_articles_published_at" json:"published_at"`
	Summary     string         `gorm:"type:text" json:"summary"`
}

type ArticleEmbedding struct {
	ArticleID uint            `gorm:"primaryKey" json:"article_id"`
	Embedding pgvector.Vector `gorm:"type:vector(768)" json:"-"`
	CreatedAt time.Time       `json:"created_at"`
}

// Request/Response Structs
type UserRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type UserResponse struct {
	APIKey string `json:"api_key"`
	Quota  int    `json:"quota"`
}

type RegisterUserRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type RegisterUserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	APIKey    string    `json:"api_key"`
	Quota     int       `json:"quota"`
	CreatedAt time.Time `json:"created_at"`
}

type RotateAPIKeyResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	OldAPIKey string    `json:"old_api_key"`
	NewAPIKey string    `json:"new_api_key"`
	Quota     int       `json:"quota"`
	RotatedAt time.Time `json:"rotated_at"`
}

type ArticleResponse struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	ImageURL string `json:"image_url"`
}
