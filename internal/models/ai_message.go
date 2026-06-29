package models

import (
	"time"
)

// AIMessage 表示 AI 对话中的一条消息
type AIMessage struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	SessionID        uint      `gorm:"index" json:"session_id"`
	Role             string    `gorm:"size:20" json:"role"`
	Content          string    `gorm:"type:text" json:"content"`
	ReasoningContent string    `gorm:"type:text" json:"reasoning_content"`
	CreatedAt        time.Time `json:"created_at"`
}
