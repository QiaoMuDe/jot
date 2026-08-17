package models

import (
	"time"

	"gorm.io/gorm"
)

// Note 表示一条笔记实体
type Note struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Title      string         `gorm:"size:200" json:"title"`
	Content    string         `gorm:"type:text" json:"content"`
	FileExt    string         `gorm:"size:10;default:.txt" json:"file_ext"`
	Pinned     bool           `gorm:"default:false;index:idx_notes_sort,priority:1" json:"pinned"`
	NotebookID uint           `gorm:"default:0;index:idx_notes_notebook_deleted,priority:2" json:"notebook_id"`
	CreatedAt  time.Time      `gorm:"index:idx_notes_created" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"index:idx_notes_sort,priority:2" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index:idx_notes_notebook_deleted,priority:1" json:"deleted_at"`
	Tags       []Tag          `gorm:"many2many:note_tags;" json:"tags,omitempty"`
	Notebook   *Notebook      `gorm:"foreignKey:NotebookID" json:"notebook,omitempty"`
}
