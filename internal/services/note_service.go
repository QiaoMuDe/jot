package services

import (
	"errors"

	"gorm.io/gorm"
	"jot/internal/models"
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
func (s *NoteService) Create(title, content string) (*models.Note, error) {
	note := models.Note{
		Title:   title,
		Content: content,
	}
	if err := s.db.Create(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

// Update 更新指定 ID 的笔记的标题和内容，返回更新后的笔记对象
func (s *NoteService) Update(id uint, title, content string) (*models.Note, error) {
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

// buildSortOrder 根据 sortBy 参数构建 ORDER BY 子句
// 支持的排序方式：updated_at（默认）、created_at、title
func buildSortOrder(sortBy string) string {
	switch sortBy {
	case "created_at":
		return "pinned DESC, created_at DESC"
	case "title":
		return "pinned DESC, title ASC"
	default:
		return "pinned DESC, updated_at DESC"
	}
}

// GetAll 分页获取未删除的笔记列表，按指定排序方式排列，返回列表与总数
func (s *NoteService) GetAll(page, pageSize int, sortBy string) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	query := s.db.Model(&models.Note{}).Where("deleted_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order(buildSortOrder(sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// GetAllIDs 获取所有未删除笔记的 ID 数组
func (s *NoteService) GetAllIDs() ([]uint, error) {
	var ids []uint
	if err := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// Search 按标题或内容关键词模糊搜索未删除的笔记，支持分页
func (s *NoteService) Search(keyword string, page, pageSize int) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	likePattern := "%" + keyword + "%"
	tagSubquery := s.db.Table("note_tags").
		Select("note_id").
		Joins("JOIN tags ON tags.id = note_tags.tag_id").
		Where("tags.name LIKE ?", likePattern)

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Where(s.db.Where("title LIKE ? OR content LIKE ?", likePattern, likePattern).
			Or("id IN (?)", tagSubquery))

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
	if err := s.db.Model(note).UpdateColumn("pinned", note.Pinned).Error; err != nil {
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

// RestoreAll 批量恢复回收站中所有已软删除的笔记
func (s *NoteService) RestoreAll() error {
	result := s.db.Unscoped().Model(&models.Note{}).
		Where("deleted_at IS NOT NULL").
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// EmptyTrash 永久删除回收站中所有已软删除的笔记
func (s *NoteService) EmptyTrash() error {
	result := s.db.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Note{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// BatchDelete 批量软删除指定 ID 数组的笔记（移入回收站）
func (s *NoteService) BatchDelete(ids []uint) error {
	result := s.db.Where("id IN ?", ids).Delete(&models.Note{})
	return result.Error
}

// BatchRestore 批量从回收站恢复指定 ID 数组的笔记
func (s *NoteService) BatchRestore(ids []uint) error {
	result := s.db.Unscoped().Model(&models.Note{}).Where("id IN ?", ids).Update("deleted_at", nil)
	return result.Error
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

// GetByTag 按标签 ID 分页获取未删除的笔记列表，支持指定排序方式
func (s *NoteService) GetByTag(tagID uint, page, pageSize int, sortBy string) ([]models.Note, int64, error) {
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
	if err := query.Order(buildSortOrder(sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// GetStats 获取数据统计概览
func (s *NoteService) GetStats() (*DataStats, error) {
	var totalNotes, trashedNotes, pinnedNotes int64

	// 未删除笔记总数
	if err := s.db.Model(&models.Note{}).Where("deleted_at IS NULL").Count(&totalNotes).Error; err != nil {
		return nil, err
	}
	// 回收站笔记数
	if err := s.db.Model(&models.Note{}).Unscoped().Where("deleted_at IS NOT NULL").Count(&trashedNotes).Error; err != nil {
		return nil, err
	}
	// 置顶笔记数
	if err := s.db.Model(&models.Note{}).Where("deleted_at IS NULL AND pinned = ?", true).Count(&pinnedNotes).Error; err != nil {
		return nil, err
	}

	return &DataStats{
		TotalNotes:   totalNotes,
		TrashedNotes: trashedNotes,
		PinnedNotes:  pinnedNotes,
	}, nil
}

// ResetAll 清空所有笔记和标签数据，用于恢复出厂设置
func (s *NoteService) ResetAll() error {
	// 清空所有笔记（包括软删除，自动清理 note_tags 关联）
	if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Note{}).Error; err != nil {
		return err
	}
	// 清空所有标签（自动清理 note_tags 中残留关联）
	if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Tag{}).Error; err != nil {
		return err
	}
	return nil
}

// ExportBackup 使用 VACUUM INTO 创建数据库的压缩副本到指定路径
func (s *NoteService) ExportBackup(destPath string) error {
	return s.db.Exec("VACUUM INTO ?", destPath).Error
}
