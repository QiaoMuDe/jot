package store

import (
	"context"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// SearchHit 表示一次向量检索返回的一个命中块。
type SearchHit struct {
	DocID      uint    // 所属文档 ID
	DocName    string  // 所属文档文件名
	ChunkIndex int     // 块在文档内的序号
	Text       string  // 块文本
	Distance   float64 // 距离（余弦距离，越小越相关）
}

// Embedder 是外部注入的文本向量化函数（内部调用 Ollama）。
type Embedder func(texts []string) ([][]float32, error)

// VectorStore 定义向量检索存储的统一接口。
type VectorStore interface {
	// AddDocument 添加一个文档：写入 Document 记录，对 chunks 批量向量化后写入 Chunk 记录；返回写入的块数。
	AddDocument(ctx context.Context, name, sourcePath, content string, chunks []string, embedder Embedder) (int, error)
	// Rebuild 清空所有 chunks 后，对全部 documents 重新切块并向量化；返回重建的块数。
	Rebuild(ctx context.Context, embedder Embedder) (int, error)
	// Search 用查询向量做 topK 检索，返回按距离升序（相关度降序）的命中块。
	Search(ctx context.Context, queryVec []float32, topK int) ([]SearchHit, error)
	// ListDocs 列出全部文档记录。
	ListDocs(ctx context.Context) ([]Document, error)
	// Status 返回实现名与文档数、块数统计。
	Status(ctx context.Context) (implName string, docCount int, chunkCount int, err error)
	// SetProgress 设置批量操作（index/rebuild）的文档级进度回调。
	SetProgress(p ProgressFunc)
}

// NewStore 按 useVec 选择实现：true 使用 sqlite-vec 加速实现，false 使用纯 Go 暴力检索。
func NewStore(db *gorm.DB, embedModel string, useVec bool) VectorStore {
	base := baseStore{db: db, embedModel: embedModel}
	if useVec {
		return &vecStore{baseStore: base}
	}
	return &bruteStore{baseStore: base}
}

// OpenDB 打开（或创建）SQLite 数据库并完成初始化：
// 设置 WAL 模式与 busy_timeout，AutoMigrate 建表。
func OpenDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库 %s 失败: %w", dbPath, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	// glebarez/go-sqlite 驱动不支持并发连接，强制单连接
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// WAL 模式与忙等待超时
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return nil, fmt.Errorf("设置 WAL 模式失败: %w", err)
	}
	if err := db.Exec("PRAGMA busy_timeout=5000;").Error; err != nil {
		return nil, fmt.Errorf("设置 busy_timeout 失败: %w", err)
	}

	// 自动建表
	if err := db.AutoMigrate(&Document{}, &Chunk{}); err != nil {
		return nil, fmt.Errorf("AutoMigrate 失败: %w", err)
	}
	return db, nil
}
