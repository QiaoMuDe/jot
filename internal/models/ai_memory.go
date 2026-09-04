package models

import (
	"time"

	"gorm.io/gorm"
)

// AIMemory 表示一条全局显式长期记忆。
// 跨会话可写，供 Agent 工具 manage_memory 写入及提问时注入系统提示词使用。
// Summary 提供简短描述（供注入），Content 保存详情；Summary 唯一约束用于去重。
type AIMemory struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Summary   string         `gorm:"size:200;uniqueIndex" json:"summary"`
	Content   string         `gorm:"type:text" json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
