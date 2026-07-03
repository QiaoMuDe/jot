package models

import (
	"time"
)

// AIPrompt 系统提示词表
func (AIPrompt) TableName() string {
	return "ai_prompts"
}

type AIPrompt struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:64" json:"key"`
	Name      string    `gorm:"size:64" json:"name"`
	Category  string    `gorm:"size:32;index" json:"category"`
	Content   string    `gorm:"type:text" json:"content"`
	IsBuiltin bool      `gorm:"default:true" json:"is_builtin"`
	UpdatedAt time.Time `json:"updated_at"`
}
