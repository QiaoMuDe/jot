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
	ThinkingElapsed  float64   `gorm:"default:0" json:"thinking_elapsed"`
	TotalElapsed     float64   `gorm:"default:0" json:"total_elapsed"`
	Tokens           int       `gorm:"default:0" json:"tokens"`
	SearchSources    string    `gorm:"type:text" json:"search_sources"`
	RecallCards      string    `gorm:"type:text" json:"recall_cards"`
	ToolCalls        string    `gorm:"type:text" json:"tool_calls"` // Agent 模式工具调用链 JSON（[]toolCallRecord）
	Meta             string    `gorm:"type:text" json:"meta"`       // 用户消息附加上下文 JSON（引用笔记/上传文件/技能等，不流向 LLM）
	CreatedAt        time.Time `json:"created_at"`
}
