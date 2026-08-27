package models

import (
	"time"

	"gorm.io/gorm"
)

// PasswordRecord 表示一条密码记录
type PasswordRecord struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:200;not null" json:"name"`     // 名称（如 GitHub、淘宝）
	Username  string         `gorm:"size:200;not null" json:"username"` // 用户名
	Password  string         `gorm:"size:500;not null" json:"password"` // 密码（存储为 Base64 编码值）
	URL       string         `gorm:"size:500" json:"url"`               // 网址（可选）
	Note      string         `json:"note"`                              // 备注（可选，text 类型不设长度限制）
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"` // 软删除时间
}
