package models

import (
	"time"

	"gorm.io/gorm"
)

// AISession 表示一次 AI 对话会话
type AISession struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:100;default:新对话" json:"title"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
