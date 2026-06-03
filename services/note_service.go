package services

import (
	"errors"

	"gorm.io/gorm"
	"jot/models"
)

// NoteService 封装笔记相关的业务逻辑操作
type NoteService struct {
	db *gorm.DB
}

// NewNoteService 创建一个新的 NoteService 实例
func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{db: db}
}

// Create 创建一条新笔记，返回创建后的笔记对象
func (s *NoteService) Create(title, content, color string) (*models.Note, error) {
	if color == "" {
		color = "#ffffff"
	}
	note := models.Note{
		Title:   title,
		Content: content,
		Color:   color,
	}
	if err := s.db.Create(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

// Update 更新指定 ID 的笔记的标题、内容和颜色，返回更新后的笔记对象
func (s *NoteService) Update(id uint, title, content, color string) (*models.Note, error) {
	note, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if title != "" {
		note.Title = title
	}
	if content != "" {
		note.Content = content
	}
	if color != "" {
		note.Color = color
	}
	if err := s.db.Save(note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

// Delete 软删除指定 ID 的笔记（移入回收站）
func (s *NoteService) Delete(id uint) error {
	result := s.db.Delete(&models.Note{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("note not found")
	}
	return nil
}

// PermanentDelete 永久删除指定 ID 的笔记（从数据库中彻底移除）
func (s *NoteService) PermanentDelete(id uint) error {
	result := s.db.Unscoped().Delete(&models.Note{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("note not found")
	}
	return nil
}

// GetByID 按 ID 获取单条笔记，预加载关联的标签
func (s *NoteService) GetByID(id uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.Preload("Tags").First(&note, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found")
		}
		return nil, err
	}
	return &note, nil
}

// GetAll 分页获取未删除的笔记列表，按置顶状态和更新时间降序排列，返回列表与总数
func (s *NoteService) GetAll(page, pageSize int) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	query := s.db.Model(&models.Note{}).Where("deleted_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("pinned DESC, updated_at DESC").
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// Search 按标题或内容关键词模糊搜索未删除的笔记，支持分页
func (s *NoteService) Search(keyword string, page, pageSize int) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	likePattern := "%" + keyword + "%"
	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Where("title LIKE ? OR content LIKE ?", likePattern, likePattern)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("pinned DESC, updated_at DESC").
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// TogglePin 切换指定笔记的置顶状态，返回更新后的笔记对象
func (s *NoteService) TogglePin(id uint) (*models.Note, error) {
	note, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	note.Pinned = !note.Pinned
	if err := s.db.Save(note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

// GetTrash 分页获取回收站中已软删除的笔记列表
func (s *NoteService) GetTrash(page, pageSize int) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	query := s.db.Model(&models.Note{}).Unscoped().Where("deleted_at IS NOT NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// Restore 从回收站恢复指定 ID 的笔记（取消软删除）
func (s *NoteService) Restore(id uint) error {
	result := s.db.Unscoped().Model(&models.Note{}).Where("id = ?", id).Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("note not found")
	}
	return nil
}

// GetByTag 按标签 ID 分页获取未删除的笔记列表
func (s *NoteService) GetByTag(tagID uint, page, pageSize int) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	// 查询与该标签关联且未删除的笔记数量
	query := s.db.Model(&models.Note{}).
		Joins("JOIN note_tags ON note_tags.note_id = notes.id").
		Where("note_tags.tag_id = ? AND notes.deleted_at IS NULL", tagID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("pinned DESC, updated_at DESC").
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}
