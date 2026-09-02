package models

import (
	"time"

	"gorm.io/gorm"
)

// AISession 表示一次 AI 对话会话
type AISession struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Title            string         `gorm:"size:100;default:新对话" json:"title"`
	ContextTokens    int            `gorm:"default:0" json:"context_tokens"`
	IsPinned         bool           `gorm:"default:false" json:"is_pinned"`
	SummaryContent   string         `gorm:"type:text" json:"summary_content"`      // 会话摘要文本
	SummaryUpToMsgID uint           `gorm:"default:0" json:"summary_up_to_msg_id"` // 摘要已覆盖到的最后一条消息 ID（不含）；0 表示未摘要或存量旧数据
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
