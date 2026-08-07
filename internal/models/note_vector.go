package models

import "time"

// NoteVector 表示笔记切块后的向量索引记录
// 一条笔记对应多个 ChunkIndex 递增的向量块；Embedding 为 float32 小端字节序序列化后的 BLOB
type NoteVector struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NoteID     uint      `gorm:"index" json:"note_id"`
	ChunkIndex int       `gorm:"index" json:"chunk_index"`
	ChunkText  string    `gorm:"type:text" json:"chunk_text"`
	Embedding  []byte    `gorm:"type:blob" json:"-"`
	Dim        int       `json:"dim"`
	Model      string    `gorm:"size:128" json:"model"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
