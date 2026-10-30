package models

import "time"

// TelegramSubscription stores subscribed Telegram chat IDs for morning briefings and push notifications
type TelegramSubscription struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ChatID    int64     `gorm:"uniqueIndex:idx_chat_topic;not null" json:"chat_id"`
	Topic     string    `gorm:"uniqueIndex:idx_chat_topic;not null;default:'berita'" json:"topic"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

