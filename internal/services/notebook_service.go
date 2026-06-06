package services

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"jot/internal/models"
)

// NotebookService 封装笔记本相关的业务逻辑操作
type NotebookService struct {
	db *gorm.DB
}

// NewNotebookService 创建一个新的 NotebookService 实例
func NewNotebookService(db *gorm.DB) *NotebookService {
	return &NotebookService{db: db}
}

// 检查笔记本名称是否已被占用（排除指定ID的笔记本，用于重命名时的自身排除）
func (s *NotebookService) isNameTaken(name string, excludeID uint) (bool, error) {
	var count int64
	query := s.db.Model(&models.Notebook{}).
		Where("LOWER(name) = LOWER(?) AND deleted_at IS NULL", name)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create 创建一个新笔记本，返回创建后的笔记本对象
func (s *NotebookService) Create(name string) (*models.Notebook, error) {
	// 检查名称是否已存在
	taken, err := s.isNameTaken(name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("笔记本「%s」已存在", name)
	}

	notebook := models.Notebook{
		Name: name,
	}
	if err := s.db.Create(&notebook).Error; err != nil {
		return nil, err
	}
	return &notebook, nil
}

// Update 重命名指定 ID 的笔记本，返回更新后的笔记本对象
// 默认笔记本（id=1）不可重命名
func (s *NotebookService) Update(id uint, name string) (*models.Notebook, error) {
	if id == 1 {
		return nil, errors.New("默认笔记本不可重命名")
	}

	var notebook models.Notebook
	if err := s.db.First(&notebook, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("notebook not found")
		}
		return nil, err
	}

	// 检查新名称是否被其他笔记本占用
	taken, err := s.isNameTaken(name, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("笔记本「%s」已存在", name)
	}

	notebook.Name = name
	if err := s.db.Save(&notebook).Error; err != nil {
		return nil, err
	}
	return &notebook, nil
}

// Delete 删除笔记本，其下笔记自动迁入默认笔记本（id=1）
// 默认笔记本（id=1）不可删除
func (s *NotebookService) Delete(id uint) error {
	if id == 1 {
		return errors.New("默认笔记本不可删除")
	}

	// 检查笔记本是否存在
	var notebook models.Notebook
	if err := s.db.First(&notebook, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notebook not found")
		}
		return err
	}

	// 将其下所有笔记的 notebook_id 改为 1（默认笔记本）
	if err := s.db.Model(&models.Note{}).
		Where("notebook_id = ?", id).
		Update("notebook_id", 1).Error; err != nil {
		return err
	}

	// 软删除笔记本
	if err := s.db.Delete(&notebook).Error; err != nil {
		return err
	}
	return nil
}

// DeleteWithNotes 删除笔记本并永久删除其下所有笔记
func (s *NotebookService) DeleteWithNotes(id uint) error {
	if id == 1 {
		return errors.New("默认笔记本不可删除")
	}

	// 检查笔记本是否存在
	var notebook models.Notebook
	if err := s.db.First(&notebook, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notebook not found")
		}
		return err
	}

	// 永久删除笔记本下的所有笔记（硬删除，不进回收站）
	if err := s.db.Unscoped().Where("notebook_id = ?", id).Delete(&models.Note{}).Error; err != nil {
		return err
	}

	// 软删除笔记本
	if err := s.db.Delete(&notebook).Error; err != nil {
		return err
	}
	return nil
}

// GetAll 获取所有未删除的笔记本，按 sort_order ASC, id ASC 排序
func (s *NotebookService) GetAll() ([]models.Notebook, error) {
	var notebooks []models.Notebook
	if err := s.db.Where("deleted_at IS NULL").
		Order("sort_order ASC, id ASC").
		Find(&notebooks).Error; err != nil {
		return nil, err
	}
	return notebooks, nil
}

// GetAllNotesCount 获取每个笔记本下未删除的笔记数量，返回 map[notebook_id]count
func (s *NotebookService) GetAllNotesCount(db *gorm.DB) (map[uint]int, error) {
	type result struct {
		NotebookID uint
		Count      int
	}
	var rows []result

	if err := db.Model(&models.Note{}).
		Select("notebook_id, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("notebook_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	countMap := make(map[uint]int, len(rows))
	for _, r := range rows {
		countMap[r.NotebookID] = r.Count
	}
	return countMap, nil
}

// EnsureDefaultNotebook 确保默认笔记本存在
// 当 notebooks 表为空时自动创建名为「默认笔记本」的笔记本（id=1），
// 并将所有 notebook_id=0 的存量笔记迁移到该默认笔记本
func (s *NotebookService) EnsureDefaultNotebook() error {
	var notebook models.Notebook
	result := s.db.First(&notebook)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}

		// 表为空，创建默认笔记本
		defaultNotebook := models.Notebook{
			Name: "默认笔记本",
		}
		if err := s.db.Create(&defaultNotebook).Error; err != nil {
			return err
		}

		// 将所有 notebook_id=0 的存量笔记迁移到默认笔记本
		if err := s.db.Model(&models.Note{}).
			Where("notebook_id = ?", 0).
			Update("notebook_id", defaultNotebook.ID).Error; err != nil {
			return err
		}
	}
	return nil
}
