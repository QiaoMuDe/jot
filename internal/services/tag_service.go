package services

import (
	"errors"

	"gorm.io/gorm"
	"jot/internal/models"
)

// TagService 封装标签相关的业务逻辑操作
type TagService struct {
	db *gorm.DB
}

// NewTagService 创建一个新的 TagService 实例
func NewTagService(db *gorm.DB) *TagService {
	return &TagService{db: db}
}

// Create 创建一个新标签，返回创建后的标签对象
func (s *TagService) Create(name, color string) (*models.Tag, error) {
	if color == "" {
		color = "#3b82f6"
	}
	tag := models.Tag{
		Name:  name,
		Color: color,
	}
	if err := s.db.Create(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// Update 更新指定 ID 的标签名称和颜色，返回更新后的标签对象
func (s *TagService) Update(id uint, name, color string) (*models.Tag, error) {
	var tag models.Tag
	if err := s.db.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	if name != "" {
		tag.Name = name
	}
	if color != "" {
		tag.Color = color
	}
	if err := s.db.Save(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// Delete 删除指定 ID 的标签，同时解除所有笔记与该标签的关联
func (s *TagService) Delete(id uint) error {
	result := s.db.Select("Notes").Delete(&models.Tag{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("tag not found")
	}
	return nil
}

// GetAll 获取所有标签列表，按创建时间升序排列
func (s *TagService) GetAll() ([]models.Tag, error) {
	var tags []models.Tag
	if err := s.db.Order("created_at ASC").Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// Count 获取标签总数
func (s *TagService) Count() (int64, error) {
	var total int64
	if err := s.db.Model(&models.Tag{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// BatchAddTagToNotes 批量将指定标签添加到多篇笔记
func (s *TagService) BatchAddTagToNotes(noteIDs []uint, tagID uint) error {
	tag, err := s.getTagByID(tagID)
	if err != nil {
		return err
	}
	for _, noteID := range noteIDs {
		note, err := s.getNoteByID(noteID)
		if err != nil {
			continue // 跳过不存在的笔记
		}
		_ = s.db.Model(note).Association("Tags").Append(tag)
	}
	return nil
}

// AddTagToNote 为指定笔记添加一个标签（多对多关联）
func (s *TagService) AddTagToNote(noteID, tagID uint) error {
	note, err := s.getNoteByID(noteID)
	if err != nil {
		return err
	}
	tag, err := s.getTagByID(tagID)
	if err != nil {
		return err
	}
	return s.db.Model(note).Association("Tags").Append(tag)
}

// RemoveTagFromNote 为指定笔记移除一个标签（解除多对多关联）
func (s *TagService) RemoveTagFromNote(noteID, tagID uint) error {
	note, err := s.getNoteByID(noteID)
	if err != nil {
		return err
	}
	tag, err := s.getTagByID(tagID)
	if err != nil {
		return err
	}
	return s.db.Model(note).Association("Tags").Delete(tag)
}

// InitDefaultTags 初始化默认标签（仅在标签表为空时插入）
func InitDefaultTags(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Tag{}).Count(&count).Error; err != nil {
		return err
	}
	// 标签表已有记录，跳过初始化
	if count > 0 {
		return nil
	}

	// 默认标签列表（名称 + 颜色）
	defaultTags := []struct {
		name  string
		color string
	}{
		{"待办", "#F59E0B"}, // 琥珀色
		{"工作", "#3B82F6"}, // 蓝色
		{"生活", "#10B981"}, // 绿色
		{"个人", "#8B5CF6"}, // 紫色
		{"学习", "#EC4899"}, // 粉色
		{"重要", "#EF4444"}, // 红色
	}
	for _, dt := range defaultTags {
		tag := models.Tag{Name: dt.name, Color: dt.color}
		if err := db.Create(&tag).Error; err != nil {
			return err
		}
	}
	return nil
}

// getNoteByID 内部辅助方法，按 ID 获取笔记
func (s *TagService) getNoteByID(id uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.First(&note, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found")
		}
		return nil, err
	}
	return &note, nil
}

// getTagByID 内部辅助方法，按 ID 获取标签
func (s *TagService) getTagByID(id uint) (*models.Tag, error) {
	var tag models.Tag
	if err := s.db.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}
