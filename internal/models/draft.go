package models

import "time"

// Draft 表示一条未保存的笔记草稿（仅存储 1 行）
type Draft struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:500" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
