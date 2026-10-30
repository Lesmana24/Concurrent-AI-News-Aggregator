package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	APIKey string `gorm:"uniqueIndex;not null" json:"api_key"`
	Quota  int    `gorm:"default:100" json:"quota"`
	Email  string `gorm:"uniqueIndex;not null" json:"email"`
}

type Article struct {
	gorm.Model
	Title       string    `gorm:"not null" json:"title"`
	Content     string    `gorm:"type:text" json:"content"`
	URL         string    `gorm:"uniqueIndex;not null" json:"url"`
	Source      string    `gorm:"index" json:"source"`
	AIProvider  string    `json:"ai_provider"`
	PublishedAt time.Time `json:"published_at"`
	Summary     string    `gorm:"type:text" json:"summary"`
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

type ArticleResponse struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Summary string `json:"summary"`
	Source  string `json:"source"`
}
