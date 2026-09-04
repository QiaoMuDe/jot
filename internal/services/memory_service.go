package services

import (
	"errors"
	"strings"
	"time"

	"gitee.com/MM-Q/fastlog"
	"gorm.io/gorm"
	"jot/internal/models"
)

// ErrMemoryExists 表示尝试创建一条 summary 已存在的记忆（唯一约束去重）。
var ErrMemoryExists = errors.New("记忆已存在")

// ErrMemorySummaryConflict 表示更新记忆时，新 summary 与另一条记忆的唯一约束冲突。
var ErrMemorySummaryConflict = errors.New("新描述与另一条记忆重复，请换一个描述")

// MemoryService 全局记忆数据服务，提供 AIMemory 的增删改查。
type MemoryService struct {
	db     *gorm.DB
	logger *fastlog.Logger
}

func NewMemoryService(db *gorm.DB, logger *fastlog.Logger) *MemoryService {
	return &MemoryService{db: db, logger: logger}
}

// Create 写入一条新记忆。summary 唯一去重：已存在时返回哨兵错误 ErrMemoryExists；
// 若命中已软删除记录（同摘要曾删除，重建场景）则复活该记录并更新内容，保证原摘要可再创建。
func (s *MemoryService) Create(summary, content string) (*models.AIMemory, error) {
	var exist int64
	if err := s.db.Model(&models.AIMemory{}).Where("summary = ?", summary).Count(&exist).Error; err != nil {
		s.logger.Errorw("MemoryService.Create 查重失败", fastlog.Error(err))
		return nil, err
	}
	if exist > 0 {
		s.logger.Warnw("MemoryService.Create 记忆已存在", fastlog.String("summary", summary))
		return nil, ErrMemoryExists
	}

	memory := &models.AIMemory{Summary: summary, Content: content}
	if err := s.db.Create(memory).Error; err != nil {
		// 唯一约束冲突：Count 预检查未拦截（并发重复 或 已软删记录占用唯一索引），
		// 统一分流到 reviveOrExists：软删记录复活重建，活跃重复返回"已存在"。
		if isUniqueConstraintError(err) {
			return s.reviveOrExists(summary, content)
		}
		s.logger.Errorw("MemoryService.Create 失败", fastlog.Error(err))
		return nil, err
	}
	s.logger.Infow("MemoryService.Create 成功",
		fastlog.Uint("id", memory.ID),
		fastlog.String("summary", summary))
	return memory, nil
}

// reviveOrExists 唯一约束冲突分流：命中已软删记录则复活并更新内容，否则视为已存在。
func (s *MemoryService) reviveOrExists(summary, content string) (*models.AIMemory, error) {
	var existing models.AIMemory
	if err := s.db.Unscoped().Where("summary = ?", summary).First(&existing).Error; err != nil {
		s.logger.Warnw("MemoryService.Create 唯一冲突但未找到既有记录",
			fastlog.String("summary", summary), fastlog.Error(err))
		return nil, ErrMemoryExists
	}
	if existing.DeletedAt.Valid {
		// 已软删记录：复活（清除删除标记）并更新内容，UNIQUE 索引随之可用
		existing.DeletedAt = gorm.DeletedAt{}
		existing.Content = content
		existing.UpdatedAt = time.Now()
		if err := s.db.Unscoped().Save(&existing).Error; err != nil {
			s.logger.Warnw("MemoryService.Create 复活软删记忆失败",
				fastlog.Uint("id", existing.ID), fastlog.Error(err))
			return nil, err
		}
		s.logger.Infow("MemoryService.Create 复活已删除记忆",
			fastlog.Uint("id", existing.ID),
			fastlog.String("summary", summary))
		return &existing, nil
	}
	// 活跃重复（并发下 Count 未拦截）
	s.logger.Warnw("MemoryService.Create 记忆已存在", fastlog.String("summary", summary))
	return nil, ErrMemoryExists
}

// isUniqueConstraintError 判断错误是否为唯一约束冲突（兼容 sqlite / mysql 错误文本）。
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "duplicate entry")
}

// Update 按 id 更新记忆的 summary 与 content。summary 变更可能触发唯一冲突，冲突时返回 error。
func (s *MemoryService) Update(id uint, summary, content string) (*models.AIMemory, error) {
	if err := s.db.Model(&models.AIMemory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"summary":    summary,
			"content":    content,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error; err != nil {
		if isUniqueConstraintError(err) {
			s.logger.Warnw("MemoryService.Update 新 summary 与另一条记忆重复", fastlog.Uint("id", id), fastlog.Error(err))
			return nil, ErrMemorySummaryConflict
		}
		s.logger.Errorw("MemoryService.Update 失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	return s.Get(id)
}

// Delete 按 id 软删除一条记忆。
func (s *MemoryService) Delete(id uint) error {
	result := s.db.Delete(&models.AIMemory{}, id)
	if result.Error != nil {
		s.logger.Errorw("MemoryService.Delete 失败", fastlog.Uint("id", id), fastlog.Error(result.Error))
		return result.Error
	}
	s.logger.Infow("MemoryService.Delete 成功", fastlog.Uint("id", id))
	return nil
}

// Get 按 id 查询记忆，不存在时返回错误。
func (s *MemoryService) Get(id uint) (*models.AIMemory, error) {
	var memory models.AIMemory
	if err := s.db.First(&memory, id).Error; err != nil {
		s.logger.Warnw("MemoryService.Get 失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	return &memory, nil
}

// List 按创建时间倒序返回全部记忆。
func (s *MemoryService) List() ([]models.AIMemory, error) {
	var memories []models.AIMemory
	if err := s.db.Order("created_at DESC, id DESC").Find(&memories).Error; err != nil {
		s.logger.Errorw("MemoryService.List 失败", fastlog.Error(err))
		return nil, err
	}
	return memories, nil
}
