package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"jot/internal/models"
	"jot/internal/services"
)

// InitDB 初始化 SQLite 数据库连接并执行自动迁移
// dbPath 为数据库文件路径，默认为 data/jot.db
func InitDB(dbPath string) (*gorm.DB, error) {
	// 确保数据库文件所在目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 打开 SQLite 连接（使用纯 Go 实现的 glebarez/sqlite 驱动，免 cgo）
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 配置连接池：SQLite 仅支持单连接写入
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// 自动迁移数据模型
	if err := db.AutoMigrate(&models.Note{}, &models.Tag{}, &models.Setting{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 初始化默认标签
	if err := services.InitDefaultTags(db); err != nil {
		return nil, fmt.Errorf("初始化默认标签失败: %w", err)
	}

	return db, nil
}
