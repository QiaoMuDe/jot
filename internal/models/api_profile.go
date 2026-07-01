package models

import "time"

// APIProfile 表示一组 API 配置预设
type APIProfile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Provider  string    `gorm:"size:20;not null" json:"provider"`
	BaseURL   string    `gorm:"size:200;not null" json:"base_url"`
	APIKey    string    `gorm:"size:200;not null" json:"api_key"`
	IsDefault bool      `gorm:"default:false" json:"is_default"`
	IsActive  bool      `gorm:"default:false" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
