package models

import "time"

// Tag 表示一个标签实体
type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:50" json:"name"`
	Color     string    `gorm:"size:20;default:'#3b82f6'" json:"color"`
	CreatedAt time.Time `json:"created_at"`
	Notes     []Note    `gorm:"many2many:note_tags;" json:"notes,omitempty"`
}
