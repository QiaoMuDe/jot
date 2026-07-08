package models

import "time"

type Todo struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Text      string    `gorm:"size:500;not null" json:"text"`
	Done      bool      `gorm:"default:false" json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
