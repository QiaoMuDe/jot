package models

import (
	"time"

	"gorm.io/gorm"
)

// AISession 表示一次 AI 对话会话
type AISession struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Title           string         `gorm:"size:100;default:新对话" json:"title"`
	ContextTokens   int            `gorm:"default:0" json:"context_tokens"`
	IsPinned        bool           `gorm:"default:false" json:"is_pinned"`
	SummaryContent  string         `gorm:"type:text" json:"summary_content"`   // 会话摘要文本
	SummaryMsgCount int            `gorm:"default:0" json:"summary_msg_count"` // 已摘要的非 system 消息数
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
