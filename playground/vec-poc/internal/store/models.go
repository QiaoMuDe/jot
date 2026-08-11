// Package store 提供向量存储与检索实现：
// 基于 sqlite-vec 的加速实现与纯 Go 暴力检索回退实现。
package store

import "time"

// Document 表示一个被索引的文档（数据库中的 documents 表）。
type Document struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:255" json:"name"` // 文件名
	SourcePath string    `gorm:"size:1024" json:"source_path"`
	Content    string    `gorm:"type:text" json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// Chunk 表示文档切块及其 embedding 向量（数据库中的 chunks 表）。
type Chunk struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DocID      uint      `gorm:"index" json:"doc_id"`
	ChunkIndex int       `gorm:"index" json:"chunk_index"`
	Text       string    `gorm:"type:text" json:"text"`
	Embedding  []byte    `gorm:"type:blob" json:"-"` // float32 小端 BLOB
	Dim        int       `json:"dim"`
	Model      string    `gorm:"size:128" json:"model"`
	CreatedAt  time.Time `json:"created_at"`
}
