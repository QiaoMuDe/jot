package services

import (
	"gorm.io/gorm"
	"jot/internal/models"
)

type TodoService struct {
	db *gorm.DB
}

func NewTodoService(db *gorm.DB) *TodoService {
	return &TodoService{db: db}
}

func (s *TodoService) Create(text string) (*models.Todo, error) {
	todo := &models.Todo{Text: text}
	if err := s.db.Create(todo).Error; err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *TodoService) List() ([]models.Todo, error) {
	var todos []models.Todo
	if err := s.db.Order("done ASC, created_at DESC").Find(&todos).Error; err != nil {
		return nil, err
	}
	return todos, nil
}

func (s *TodoService) Toggle(id uint) (*models.Todo, error) {
	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return nil, err
	}
	todo.Done = !todo.Done
	if err := s.db.Save(&todo).Error; err != nil {
		return nil, err
	}
	return &todo, nil
}

func (s *TodoService) Delete(id uint) error {
	return s.db.Delete(&models.Todo{}, id).Error
}

func (s *TodoService) Update(id uint, text string) (*models.Todo, error) {
	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return nil, err
	}
	todo.Text = text
	if err := s.db.Save(&todo).Error; err != nil {
		return nil, err
	}
	return &todo, nil
}
