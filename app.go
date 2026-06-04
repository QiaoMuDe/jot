package main

import (
	"context"
	"jot/database"
	"jot/models"
	"jot/services"
)

// App struct
type App struct {
	ctx         context.Context
	noteService *services.NoteService
	tagService  *services.TagService
}

// NewApp creates a new App application struct
func NewApp() *App {
	db := database.InitDB("data/jot.db")
	return &App{
		noteService: services.NewNoteService(db),
		tagService:  services.NewTagService(db),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ==================== Note 相关绑定方法 ====================

// CreateNote 创建一条新笔记
func (a *App) CreateNote(title, content string) (*models.Note, error) {
	return a.noteService.Create(title, content)
}

// UpdateNote 更新指定笔记的标题和内容
func (a *App) UpdateNote(id uint, title, content string) (*models.Note, error) {
	return a.noteService.Update(id, title, content)
}

// DeleteNote 软删除指定笔记（移入回收站）
func (a *App) DeleteNote(id uint) error {
	return a.noteService.Delete(id)
}

// PermanentDeleteNote 永久删除指定笔记（从数据库彻底移除）
func (a *App) PermanentDeleteNote(id uint) error {
	return a.noteService.PermanentDelete(id)
}

// GetNote 按 ID 获取单条笔记
func (a *App) GetNote(id uint) (*models.Note, error) {
	return a.noteService.GetByID(id)
}

// GetNotes 分页获取未删除的笔记列表
func (a *App) GetNotes(page, pageSize int) (*services.PaginatedResult, error) {
	notes, total, err := a.noteService.GetAll(page, pageSize)
	if err != nil {
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// SearchNotes 按关键词搜索笔记（标题/内容），支持分页
func (a *App) SearchNotes(keyword string, page, pageSize int) (*services.PaginatedResult, error) {
	notes, total, err := a.noteService.Search(keyword, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// TogglePinNote 切换指定笔记的置顶状态
func (a *App) TogglePinNote(id uint) (*models.Note, error) {
	return a.noteService.TogglePin(id)
}

// GetTrashNotes 分页获取回收站中的笔记列表
func (a *App) GetTrashNotes(page, pageSize int) (*services.PaginatedResult, error) {
	notes, total, err := a.noteService.GetTrash(page, pageSize)
	if err != nil {
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// RestoreNote 从回收站恢复指定笔记
func (a *App) RestoreNote(id uint) error {
	return a.noteService.Restore(id)
}

// GetNotesByTag 按标签分页获取笔记
func (a *App) GetNotesByTag(tagID uint, page, pageSize int) (*services.PaginatedResult, error) {
	notes, total, err := a.noteService.GetByTag(tagID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ==================== Tag 相关绑定方法 ====================

// CreateTag 创建一个新标签
func (a *App) CreateTag(name, color string) (*models.Tag, error) {
	return a.tagService.Create(name, color)
}

// UpdateTag 更新指定标签的名称和颜色
func (a *App) UpdateTag(id uint, name, color string) (*models.Tag, error) {
	return a.tagService.Update(id, name, color)
}

// DeleteTag 删除指定标签
func (a *App) DeleteTag(id uint) error {
	return a.tagService.Delete(id)
}

// GetAllTags 获取所有标签列表
func (a *App) GetAllTags() ([]models.Tag, error) {
	return a.tagService.GetAll()
}

// AddTagToNote 为指定笔记添加标签
func (a *App) AddTagToNote(noteID, tagID uint) error {
	return a.tagService.AddTagToNote(noteID, tagID)
}

// RemoveTagFromNote 为指定笔记移除标签
func (a *App) RemoveTagFromNote(noteID, tagID uint) error {
	return a.tagService.RemoveTagFromNote(noteID, tagID)
}
