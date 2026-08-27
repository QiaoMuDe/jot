package services

import (
	"strings"

	"gitee.com/MM-Q/fastlog"
	"gorm.io/gorm"
	"jot/internal/models"
)

type TodoService struct {
	db     *gorm.DB
	logger *fastlog.Logger
}

func NewTodoService(db *gorm.DB, logger *fastlog.Logger) *TodoService {
	return &TodoService{db: db, logger: logger}
}

func (s *TodoService) Create(text string) (*models.Todo, error) {
	todo := &models.Todo{Text: text}
	if err := s.db.Create(todo).Error; err != nil {
		s.logger.Errorw("TodoService.Create 失败", fastlog.Error(err))
		return nil, err
	}
	return todo, nil
}

// todoOrder 待办统一排序：未完成在前，未完成按创建时间倒序、已完成按更新时间倒序，
// 追加 id DESC 保证唯一键，使分页跨页时顺序稳定、不重复不遗漏。
const todoOrder = "done ASC, CASE WHEN done = 1 THEN updated_at ELSE created_at END DESC, id DESC"

// List 全量列出待办（供前端页面一次性渲染使用）。
func (s *TodoService) List() ([]models.Todo, error) {
	var todos []models.Todo
	if err := s.db.Order(todoOrder).Find(&todos).Error; err != nil {
		s.logger.Errorw("TodoService.List 失败", fastlog.Error(err))
		return nil, err
	}
	return todos, nil
}

// ListPaged 分页列出待办，返回当前页条目与满足过滤条件的总数。
// done 为 nil 表示全部；非 nil 表示按完成状态过滤（false=未完成，true=已完成）。
// page 从 1 开始（小于 1 视为 1），pageSize 小于等于 0 时默认 20。
// 排序与 List 一致（见 todoOrder），保证跨页稳定。
func (s *TodoService) ListPaged(done *bool, page, pageSize int) ([]models.Todo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	base := s.db.Model(&models.Todo{})
	if done != nil {
		base = base.Where("done = ?", *done)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		s.logger.Errorw("TodoService.ListPaged 计数失败", fastlog.Error(err))
		return nil, 0, err
	}

	var todos []models.Todo
	if err := base.
		Order(todoOrder).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&todos).Error; err != nil {
		s.logger.Errorw("TodoService.ListPaged 失败", fastlog.Error(err))
		return nil, 0, err
	}
	return todos, total, nil
}

// Search 按关键字模糊搜索待办内容，支持按完成状态过滤与分页。
// keyword trim 后为空时等价于 ListPaged（全量）；done 为 nil 表示全部；
// page 从 1 开始（小于 1 视为 1），pageSize 小于等于 0 时默认 20。
// 排序与 List 一致（见 todoOrder），保证跨页稳定。
func (s *TodoService) Search(keyword string, done *bool, page, pageSize int) ([]models.Todo, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	base := s.db.Model(&models.Todo{})
	if done != nil {
		base = base.Where("done = ?", *done)
	}
	if keyword != "" {
		base = base.Where("text LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		s.logger.Errorw("TodoService.Search 计数失败", fastlog.Error(err))
		return nil, 0, err
	}

	var todos []models.Todo
	if err := base.
		Order(todoOrder).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&todos).Error; err != nil {
		s.logger.Errorw("TodoService.Search 失败", fastlog.Error(err))
		return nil, 0, err
	}
	return todos, total, nil
}

func (s *TodoService) Toggle(id uint) (*models.Todo, error) {
	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		s.logger.Errorw("TodoService.Toggle 失败", fastlog.Error(err))
		return nil, err
	}
	todo.Done = !todo.Done
	if err := s.db.Save(&todo).Error; err != nil {
		s.logger.Errorw("TodoService.Toggle 失败", fastlog.Error(err))
		return nil, err
	}
	return &todo, nil
}

func (s *TodoService) Delete(id uint) error {
	err := s.db.Delete(&models.Todo{}, id).Error
	if err != nil {
		s.logger.Errorw("TodoService.Delete 失败", fastlog.Error(err))
	}
	return err
}

func (s *TodoService) Update(id uint, text string) (*models.Todo, error) {
	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		s.logger.Errorw("TodoService.Update 失败", fastlog.Error(err))
		return nil, err
	}
	todo.Text = text
	if err := s.db.Save(&todo).Error; err != nil {
		s.logger.Errorw("TodoService.Update 失败", fastlog.Error(err))
		return nil, err
	}
	return &todo, nil
}

func (s *TodoService) Count() (int64, error) {
	var count int64
	if err := s.db.Model(&models.Todo{}).Count(&count).Error; err != nil {
		s.logger.Errorw("TodoService.Count 失败", fastlog.Error(err))
		return 0, err
	}
	return count, nil
}

func (s *TodoService) CountCompleted() (int64, error) {
	var count int64
	if err := s.db.Model(&models.Todo{}).Where("done = ?", true).Count(&count).Error; err != nil {
		s.logger.Errorw("TodoService.CountCompleted 失败", fastlog.Error(err))
		return 0, err
	}
	return count, nil
}

// CountUnfinished 统计未完成待办数量
func (s *TodoService) CountUnfinished() (int64, error) {
	var count int64
	if err := s.db.Model(&models.Todo{}).Where("done = ?", false).Count(&count).Error; err != nil {
		s.logger.Errorw("TodoService.CountUnfinished 失败", fastlog.Error(err))
		return 0, err
	}
	return count, nil
}

func (s *TodoService) DeleteCompleted() (int64, error) {
	result := s.db.Where("done = ?", true).Delete(&models.Todo{})
	if result.Error != nil {
		s.logger.Errorw("TodoService.DeleteCompleted 失败", fastlog.Error(result.Error))
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// DeleteUnfinished 删除所有未完成的待办事项，返回删除条数
func (s *TodoService) DeleteUnfinished() (int64, error) {
	result := s.db.Where("done = ?", false).Delete(&models.Todo{})
	if result.Error != nil {
		s.logger.Errorw("TodoService.DeleteUnfinished 失败", fastlog.Error(result.Error))
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// DeleteAll 删除所有待办事项，返回删除条数
func (s *TodoService) DeleteAll() (int64, error) {
	result := s.db.Where("1 = 1").Delete(&models.Todo{})
	if result.Error != nil {
		s.logger.Errorw("TodoService.DeleteAll 失败", fastlog.Error(result.Error))
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
