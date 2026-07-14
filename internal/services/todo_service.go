package services

import (
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

func (s *TodoService) List() ([]models.Todo, error) {
	var todos []models.Todo
	if err := s.db.Order("done ASC, created_at DESC").Find(&todos).Error; err != nil {
		s.logger.Errorw("TodoService.List 失败", fastlog.Error(err))
		return nil, err
	}
	return todos, nil
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

func (s *TodoService) DeleteCompleted() (int64, error) {
	result := s.db.Where("done = ?", true).Delete(&models.Todo{})
	if result.Error != nil {
		s.logger.Errorw("TodoService.DeleteCompleted 失败", fastlog.Error(result.Error))
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
