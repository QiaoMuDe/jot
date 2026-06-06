package services

import (
	"errors"

	"gorm.io/gorm"
	"jot/internal/models"
)

// DraftService 封装草稿相关的业务逻辑操作
type DraftService struct {
	db *gorm.DB
}

// NewDraftService 创建一个新的 DraftService 实例
func NewDraftService(db *gorm.DB) *DraftService {
	return &DraftService{db: db}
}

// SaveDraft 保存草稿（对 ID=1 执行 upsert）
func (s *DraftService) SaveDraft(title, content string) error {
	draft := models.Draft{
		ID:      1,
		Title:   title,
		Content: content,
	}
	return s.db.Save(&draft).Error
}

// GetDraft 获取草稿（查询 ID=1 的记录），不存在时返回 nil, nil
func (s *DraftService) GetDraft() (*models.Draft, error) {
	var draft models.Draft
	err := s.db.First(&draft, 1).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &draft, nil
}

// ClearDraft 删除草稿（删除 ID=1 的记录）
func (s *DraftService) ClearDraft() error {
	return s.db.Delete(&models.Draft{}, 1).Error
}
