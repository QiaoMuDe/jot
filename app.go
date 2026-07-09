package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"jot/internal/database"
	"jot/internal/fontutil"
	"jot/internal/models"
	"jot/internal/services"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gitee.com/MM-Q/go-kit/fs"
	"gitee.com/MM-Q/verman"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// App struct
type App struct {
	ctx             context.Context
	db              *gorm.DB
	noteService     *services.NoteService
	tagService      *services.TagService
	settingService  *services.SettingService
	notebookService *services.NotebookService
	aiService       *services.AIService
	profileService  *services.ProfileService
	todoService     *services.TodoService
	aiStreamCancel  context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		panic(err)
	}
	db, err := database.InitDB(dbPath)
	if err != nil {
		panic(err)
	}
	settingService := services.NewSettingService(db)
	return &App{
		db:              db,
		noteService:     services.NewNoteService(db, settingService),
		tagService:      services.NewTagService(db),
		settingService:  settingService,
		notebookService: services.NewNotebookService(db),
		aiService:       services.NewAIService(db),
		profileService:  services.NewProfileService(db),
		todoService:     services.NewTodoService(db),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 确保图片存储目录存在
	home, _ := os.UserHomeDir()
	imageDir := filepath.Join(home, ".jot", "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		fmt.Printf("创建图片目录失败: %v\n", err)
	}

	// 确保默认笔记本存在（首次启动自动创建）
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		fmt.Printf("初始化默认笔记本失败: %v\n", err)
	}
	// 迁移：已有配置但无预设时，自动创建"默认配置"
	profiles := a.profileService.ListProfiles()
	if len(profiles) == 0 {
		baseURL := a.settingService.Get("ai_base_url")
		if baseURL != "" {
			provider := a.settingService.Get("ai_provider")
			apiKey := a.settingService.Get("ai_api_key")
			if provider == "" {
				provider = "openai"
			}
			profile := a.profileService.CreateProfile("默认配置", provider, baseURL, apiKey, true)
			// 标记为激活
			if err := a.profileService.SwitchProfile(profile.ID); err != nil {
				fmt.Printf("迁移警告：激活默认配置失败: %v\n", err)
			}
			fmt.Println("迁移完成：已从现有配置创建'默认配置'预设")
		}
	} else {
		// 旧数据迁移：如有预设但无一激活，值匹配补标
		hasActive := false
		for _, p := range profiles {
			if p.IsActive {
				hasActive = true
				break
			}
		}
		if !hasActive {
			baseURL := a.settingService.Get("ai_base_url")
			apiKey := a.settingService.Get("ai_api_key")
			if baseURL != "" {
				for _, p := range profiles {
					if p.BaseURL == baseURL && p.APIKey == apiKey {
						if err := a.profileService.SwitchProfile(p.ID); err != nil {
							fmt.Printf("迁移警告：标记激活预设失败: %v\n", err)
						}
						fmt.Println("迁移完成：已标记匹配预设为激活")
						break
					}
				}
			}
		}
	}
}

// ==================== 图片相关方法 ====================

// SaveImage 保存图片到 ~/.jot/images/，返回可访问的 URL 路径
// name: 原始文件名, data: base64 编码的图片数据
// 返回: /images/uuid_name.ext 格式的 URL
func (a *App) SaveImage(name string, data string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("解码图片数据失败: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	imageDir := filepath.Join(home, ".jot", "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	uuid := fmt.Sprintf("%x", b)

	filename := uuid + "_" + name
	filePath := filepath.Join(imageDir, filename)
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		return "", fmt.Errorf("写入图片文件失败: %w", err)
	}

	return "/images/" + filename, nil
}

// SaveImageFromPath 从本地路径复制图片到 ~/.jot/images/，返回可访问的 URL 路径
// localPath: 本地文件绝对路径
// 返回: /images/uuid_name.ext 格式的 URL
func (a *App) SaveImageFromPath(localPath string) (string, error) {
	bytes, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("读取本地图片失败: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	imageDir := filepath.Join(home, ".jot", "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	uuid := fmt.Sprintf("%x", b)

	filename := uuid + "_" + filepath.Base(localPath)
	filePath := filepath.Join(imageDir, filename)
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		return "", fmt.Errorf("写入图片文件失败: %w", err)
	}

	return "/images/" + filename, nil
}

// ReadTextFile 读取文本文件内容，若为二进制文件则返回错误
// localPath: 本地文件绝对路径
func (a *App) ReadTextFile(localPath string) (string, error) {
	if fs.IsBinaryPath(localPath) {
		return "", fmt.Errorf("不支持二进制文件")
	}
	content, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	return string(content), nil
}

// CleanupOrphanImages 清理 ~/.jot/images/ 中未被任何笔记引用的孤儿图片
// 扫描所有笔记（含回收站）的 content，删除未引用的图片文件
// 返回删除的文件数量
func (a *App) CleanupOrphanImages() int {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0
	}
	imageDir := filepath.Join(home, ".jot", "images")

	// 读取图片目录
	entries, err := os.ReadDir(imageDir)
	if err != nil {
		// 目录不存在或无法读取，视为无孤儿图片
		return 0
	}

	// 查询所有笔记（含软删除/回收站）的 content
	var contents []string
	a.db.Model(&models.Note{}).Unscoped().Pluck("content", &contents)

	// 构建引用集合
	referenced := make(map[string]bool)
	for _, content := range contents {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filename := entry.Name()
			if strings.Contains(content, "/images/"+filename) {
				referenced[filename] = true
			}
		}
	}

	// 删除未被引用的图片
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !referenced[filename] {
			if err := os.Remove(filepath.Join(imageDir, filename)); err == nil {
				deleted++
			}
		}
	}

	return deleted
}

// imageDirPath 返回 ~/.jot/images 目录路径
func (a *App) imageDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, ".jot", "images"), nil
}

// ==================== Note 相关绑定方法 ====================

// CreateNote 创建一条新笔记，归入指定笔记本
func (a *App) CreateNote(title, content, fileExt string, notebookID uint) (*models.Note, error) {
	return a.noteService.CreateWithNotebook(title, content, fileExt, notebookID)
}

// UpdateNote 更新指定笔记的标题和内容
func (a *App) UpdateNote(id uint, title, content, fileExt string) (*models.Note, error) {
	return a.noteService.Update(id, title, content, fileExt)
}

// UpdateNoteFileExt 更新指定笔记的文件后缀（不修改其他字段）
func (a *App) UpdateNoteFileExt(id uint, fileExt string) (*models.Note, error) {
	return a.noteService.UpdateFileExt(id, fileExt)
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

// GetNoteContent 按 ID 仅获取笔记的完整 content 文本（列表查询只返回截断版本，用于编辑器按需加载）
func (a *App) GetNoteContent(id uint) (string, error) {
	return a.noteService.GetNoteContent(id)
}

// GetNoteRefContext 构建笔记引用上下文。
// 后端一次性完成：查库 → 截断 → 拼装，返回每条笔记信息和完整 context 文本。
func (a *App) GetNoteRefContext(ids []uint) (*services.NoteRefContext, error) {
	return a.noteService.BuildNoteRefContext(ids)
}

// GetNotes 分页获取未删除的笔记列表，支持指定排序方式和笔记本筛选
func (a *App) GetNotes(page, pageSize int, sortBy string, notebookID uint) (*services.PaginatedResult, error) {
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.GetAllByNotebook(page, pageSize, sortBy, notebookID)
	} else {
		notes, total, err = a.noteService.GetAll(page, pageSize, sortBy)
	}
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

// GetAllNoteIDs 获取所有未删除笔记的 ID 数组
func (a *App) GetAllNoteIDs() ([]uint, error) {
	return a.noteService.GetAllIDs()
}

// SearchNotes 按关键词搜索笔记（标题/内容），支持分页、笔记本筛选、日期范围和标签 AND 过滤
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string, tagIDs []uint) (*services.PaginatedResult, error) {
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.SearchByNotebook(keyword, page, pageSize, notebookID, sortBy, startDate, endDate, tagIDs)
	} else {
		notes, total, err = a.noteService.Search(keyword, page, pageSize, sortBy, startDate, endDate, tagIDs)
	}
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

// SearchNoteIDs 按筛选条件返回所有匹配笔记 ID（不分页），用于全选功能
func (a *App) SearchNoteIDs(keyword string, notebookID uint, tagIDs []uint) ([]uint, error) {
	if notebookID > 0 {
		return a.noteService.SearchNoteIDsByNotebook(keyword, notebookID, tagIDs)
	}
	return a.noteService.SearchNoteIDs(keyword, tagIDs)
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

// BatchPinNotes 批量置顶或取消置顶笔记
func (a *App) BatchPinNotes(noteIDs []uint, pin bool) error {
	return a.noteService.BatchPinNotes(noteIDs, pin)
}

// BatchDeleteNotes 批量软删除笔记
func (a *App) BatchDeleteNotes(ids []uint) error {
	return a.noteService.BatchDelete(ids)
}

// BatchRestoreNotes 批量从回收站恢复笔记
func (a *App) BatchRestoreNotes(ids []uint) error {
	return a.noteService.BatchRestore(ids)
}

// BatchAddTagToNotes 批量添加标签到笔记
func (a *App) BatchAddTagToNotes(noteIDs []uint, tagID uint) error {
	return a.tagService.BatchAddTagToNotes(noteIDs, tagID)
}

// BatchRemoveTagFromNotes 批量从笔记移除标签
func (a *App) BatchRemoveTagFromNotes(noteIDs []uint, tagID uint) error {
	return a.tagService.BatchRemoveTagFromNotes(noteIDs, tagID)
}

// RestoreAllNotes 批量恢复回收站中所有笔记
func (a *App) RestoreAllNotes() error {
	return a.noteService.RestoreAll()
}

// EmptyTrash 永久清空回收站中所有笔记
func (a *App) EmptyTrash() error {
	return a.noteService.EmptyTrash()
}

// GetNotesByTag 按标签分页获取笔记，支持指定排序方式（updated_at/created_at/title）
func (a *App) GetNotesByTag(tagID uint, page, pageSize int, sortBy string) (*services.PaginatedResult, error) {
	notes, total, err := a.noteService.GetByTag(tagID, page, pageSize, sortBy)
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

// GetDataStats 获取数据统计概览
func (a *App) GetDataStats() (*services.DataStats, error) {
	stats, err := a.noteService.GetStats()
	if err != nil {
		return nil, err
	}
	tagCount, err := a.tagService.Count()
	if err != nil {
		return nil, err
	}
	stats.TotalTags = tagCount
	// 获取数据库文件大小
	dbPath, pathErr := database.DefaultDBPath()
	if pathErr == nil {
		if fi, statErr := os.Stat(dbPath); statErr == nil {
			size := fi.Size()
			stats.DBSize = size
			switch {
			case size < 1024:
				stats.DBSizeStr = fmt.Sprintf("%d B", size)
			case size < 1024*1024:
				stats.DBSizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
			default:
				stats.DBSizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
			}
		}
	}
	// AI 会话/消息统计
	aiSessions, _ := a.aiService.CountSessions()
	aiMessages, _ := a.aiService.CountMessages()
	stats.AISessions = aiSessions
	stats.AIMessages = aiMessages

	// AI 性能统计
	totalTokens, _ := a.aiService.SumTokens()
	avgResponseTime, _ := a.aiService.AvgResponseTime()
	avgThinkingTime, _ := a.aiService.AvgThinkingTime()
	maxResponseTime, _ := a.aiService.MaxResponseTime()
	stats.TotalTokens = totalTokens
	stats.AvgResponseTime = avgResponseTime
	stats.AvgThinkingTime = avgThinkingTime
	stats.MaxResponseTime = maxResponseTime

	// 待办统计
	totalTodos, _ := a.todoService.Count()
	completedTodos, _ := a.todoService.CountCompleted()
	stats.TotalTodos = totalTodos
	stats.CompletedTodos = completedTodos

	return stats, nil
}

// VacuumDatabase 执行存储优化操作：清理无效数据后执行 VACUUM，返回释放的空间大小
func (a *App) VacuumDatabase() (string, error) {
	// 读取回收站自动清理天数设置
	daysStr := a.settingService.Get("trash_cleanup_retention_days")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	// 1. 清理空 AI 会话
	deletedSessions := a.aiService.DeleteEmptyAISessions()

	// 2. 清理孤儿 AI 消息
	deletedOrphanMsgs := a.aiService.DeleteOrphanMessages()

	// 3. 清理过期回收站笔记（超过 N 天）
	deletedNotes := a.noteService.CleanExpiredTrash(days)

	// 4. 清理过期回收站笔记本（超过 N 天）
	deletedNotebooks := a.notebookService.CleanExpiredTrash(days)

	// 5. 迁移指向不存在笔记本的笔记到默认笔记本
	migratedNotes := a.noteService.MigrateOrphanNotes()

	// 6. 清空已完成待办
	deletedTodos, _ := a.todoService.DeleteCompleted()

	// 7. 清理未引用的图片文件
	deletedImages := a.CleanupOrphanImages()

	// 获取瘦身前数据库文件大小
	dbPath, _ := database.DefaultDBPath()
	var beforeSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		beforeSize = fi.Size()
	}

	if err := a.noteService.Vacuum(); err != nil {
		return "", fmt.Errorf("存储优化失败: %w", err)
	}

	// 获取瘦身后数据库文件大小
	var afterSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		afterSize = fi.Size()
	}

	saved := beforeSize - afterSize
	if saved < 0 {
		saved = 0
	}
	var savedStr string
	switch {
	case saved < 1024:
		savedStr = fmt.Sprintf("%d B", saved)
	case saved < 1024*1024:
		savedStr = fmt.Sprintf("%.1f KB", float64(saved)/1024)
	default:
		savedStr = fmt.Sprintf("%.1f MB", float64(saved)/(1024*1024))
	}

	// 组装结果消息
	var parts []string
	parts = append(parts, fmt.Sprintf("释放了 %s 空间", savedStr))
	if deletedSessions > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 个空 AI 会话", deletedSessions))
	}
	if deletedOrphanMsgs > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 条孤儿 AI 消息", deletedOrphanMsgs))
	}
	if deletedNotes > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 条过期回收站笔记", deletedNotes))
	}
	if deletedNotebooks > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 个过期回收站笔记本", deletedNotebooks))
	}
	if migratedNotes > 0 {
		parts = append(parts, fmt.Sprintf("迁移了 %d 条孤儿笔记到默认笔记本", migratedNotes))
	}
	if deletedTodos > 0 {
		parts = append(parts, fmt.Sprintf("清空了 %d 个已完成待办", deletedTodos))
	}
	if deletedImages > 0 {
		parts = append(parts, fmt.Sprintf("删除了 %d 张未引用图片", deletedImages))
	}
	return strings.Join(parts, "，"), nil
}

// ExportDataWithDialog 弹出保存对话框，导出 ZIP 格式备份（含数据库和图片）
func (a *App) ExportDataWithDialog() (string, error) {
	// 弹出保存对话框
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出数据备份",
		DefaultFilename: "jot-backup-" + time.Now().Format("2006-01-02") + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	// 调用统一导出
	if err := a.exportSnapshot(filePath); err != nil {
		return "", fmt.Errorf("导出失败: %w", err)
	}

	return "导出成功：" + filePath, nil
}

// ImportDatabaseWithDialog 弹出文件选择对话框，从 ZIP 备份文件恢复数据（含图片）
func (a *App) ImportDatabaseWithDialog() (*services.ImportResult, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入数据备份",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}
	if filePath == "" {
		return &services.ImportResult{Message: "已取消"}, nil
	}

	if err := a.importFromArchive(filePath); err != nil {
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}

	return &services.ImportResult{
		Message:      "已从备份文件恢复数据库与图片",
		SuccessCount: 1,
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

// ==================== Notebook 相关绑定方法 ====================

// CreateNotebook 创建新笔记本
func (a *App) CreateNotebook(name string) (*models.Notebook, error) {
	return a.notebookService.Create(name)
}

// RenameNotebook 重命名笔记本
func (a *App) RenameNotebook(id uint, name string) (*models.Notebook, error) {
	return a.notebookService.Update(id, name)
}

// DeleteNotebook 删除笔记本，其下笔记自动迁入默认笔记本
func (a *App) DeleteNotebook(id uint) error {
	return a.notebookService.Delete(id)
}

// DeleteNotebookWithNotes 删除笔记本并清空其下所有笔记
func (a *App) DeleteNotebookWithNotes(id uint) error {
	return a.notebookService.DeleteWithNotes(id)
}

// GetTrashNotebooks 分页获取回收站中已删除的笔记本列表
func (a *App) GetTrashNotebooks(page, pageSize int) (*services.PaginatedResult, error) {
	notebooks, total, err := a.notebookService.GetTrash(page, pageSize)
	if err != nil {
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notebooks,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// RestoreTrashNotebook 从回收站恢复指定笔记本
func (a *App) RestoreTrashNotebook(id uint) error {
	return a.notebookService.RestoreFromTrash(id)
}

// PermanentDeleteTrashNotebook 从回收站永久删除指定笔记本
func (a *App) PermanentDeleteTrashNotebook(id uint) error {
	return a.notebookService.PermanentDeleteFromTrash(id)
}

// RestoreAllTrashNotebooks 恢复回收站中所有笔记本
func (a *App) RestoreAllTrashNotebooks() error {
	return a.notebookService.RestoreAllFromTrash()
}

// EmptyTrashNotebooks 清空回收站中所有笔记本
func (a *App) EmptyTrashNotebooks() error {
	return a.notebookService.EmptyTrash()
}

// MoveNoteToNotebook 将单条笔记移动到目标笔记本
func (a *App) MoveNoteToNotebook(id, targetNotebookID uint) error {
	return a.noteService.MoveToNotebook(id, targetNotebookID)
}

// BatchMoveNotesToNotebook 批量将多条笔记移动到目标笔记本
func (a *App) BatchMoveNotesToNotebook(noteIDs []uint, targetNotebookID uint) error {
	return a.noteService.BatchMoveToNotebook(noteIDs, targetNotebookID)
}

// GetAllNotebooks 获取所有未删除笔记本列表
func (a *App) GetAllNotebooks() ([]models.Notebook, error) {
	return a.notebookService.GetAll()
}

// GetNotebookNoteCounts 获取各笔记本下笔记数量
func (a *App) GetNotebookNoteCounts() (map[uint]int, error) {
	return a.notebookService.GetAllNotesCount(a.db)
}

// GetNoteIDsByNotebook 获取指定笔记本中所有未删除笔记的 ID 数组
func (a *App) GetNoteIDsByNotebook(notebookID uint) ([]uint, error) {
	return a.noteService.GetAllNoteIDsByNotebook(notebookID)
}

// ==================== Setting 相关绑定方法 ====================

// GetSetting 获取指定 key 的配置值
func (a *App) GetSetting(key string) string {
	return a.settingService.Get(key)
}

// SetSetting 设置指定 key 的配置值
func (a *App) SetSetting(key, value string) error {
	return a.settingService.Set(key, value)
}

// GetAllSettings 获取全部设置项
func (a *App) GetAllSettings() services.SettingsConfig {
	return a.settingService.GetAllSettings()
}

// SaveAllSettings 保存全部设置项
func (a *App) SaveAllSettings(cfg services.SettingsConfig) error {
	return a.settingService.SaveAllSettings(cfg)
}

// GetAIRefMaxChars 获取 AI 引用笔记截断字数，空值时返回默认 5000
func (a *App) GetAIRefMaxChars() int {
	val := a.settingService.Get("ai_ref_max_chars")
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 5000
	}
	if n > 50000 {
		return 50000
	}
	return n
}

// SetAIRefMaxChars 设置 AI 引用笔记截断字数，含范围校验（1-50000）
func (a *App) SetAIRefMaxChars(chars int) error {
	if chars <= 0 {
		return fmt.Errorf("截断字数必须大于 0")
	}
	if chars > 50000 {
		return fmt.Errorf("截断字数不能超过 50000")
	}
	return a.settingService.Set("ai_ref_max_chars", strconv.Itoa(chars))
}

// GetAISearchResultLimit 获取 AI 联网搜索结果数，空值时返回默认 5
func (a *App) GetAISearchResultLimit() int {
	val := a.settingService.Get("ai_search_result_limit")
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return 5
	}
	if n > 30 {
		return 30
	}
	return n
}

// SetAISearchResultLimit 设置 AI 联网搜索结果数，含范围校验（1-20）
func (a *App) SetAISearchResultLimit(limit int) error {
	if limit < 1 {
		return fmt.Errorf("搜索结果数必须大于 0")
	}
	if limit > 30 {
		return fmt.Errorf("搜索结果数不能超过 30")
	}
	return a.settingService.Set("ai_search_result_limit", strconv.Itoa(limit))
}

// GetAICardRecallLimit 获取 AI 卡片召回条数，空值时返回默认 5
func (a *App) GetAICardRecallLimit() int {
	val := a.settingService.Get("ai_card_recall_limit")
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return 5
	}
	if n > 30 {
		return 30
	}
	return n
}

// SetAICardRecallLimit 设置 AI 卡片召回条数，含范围校验（1-30）
func (a *App) SetAICardRecallLimit(limit int) error {
	if limit < 1 {
		return fmt.Errorf("卡片召回条数必须大于 0")
	}
	if limit > 30 {
		return fmt.Errorf("卡片召回条数不能超过 30")
	}
	return a.settingService.Set("ai_card_recall_limit", strconv.Itoa(limit))
}

// ==================== AI 相关绑定方法 ====================

// GetAIConfig 获取 AI 服务配置
func (a *App) GetAIConfig() services.AIConfig {
	return a.aiService.GetConfig()
}

// SaveAIConfig 保存 AI 服务配置，无预设时自动创建默认配置
func (a *App) SaveAIConfig(cfg services.AIConfig) error {
	if err := a.aiService.SaveConfig(cfg); err != nil {
		return err
	}
	// 无预设时自动创建"默认配置"
	profiles := a.profileService.ListProfiles()
	if len(profiles) == 0 {
		profile := a.profileService.CreateProfile("默认配置", cfg.Provider, cfg.BaseURL, cfg.APIKey, true)
		if err := a.profileService.SetActive(profile.ID); err != nil {
			fmt.Printf("警告：激活默认配置失败: %v\n", err)
		}
	}
	return nil
}

// ==================== API 配置预设绑定 ====================

// GetProfiles 获取所有 API 配置预设
func (a *App) GetProfiles() []models.APIProfile {
	return a.profileService.ListProfiles()
}

// CreateProfile 创建 API 配置预设
func (a *App) CreateProfile(name, provider, baseURL, apiKey string) models.APIProfile {
	return a.profileService.CreateProfile(name, provider, baseURL, apiKey)
}

// UpdateProfile 更新 API 配置预设
func (a *App) UpdateProfile(id uint, name, provider, baseURL, apiKey string) error {
	return a.profileService.UpdateProfile(id, name, provider, baseURL, apiKey)
}

// DeleteProfile 删除 API 配置预设
func (a *App) DeleteProfile(id uint) error {
	return a.profileService.DeleteProfile(id)
}

// SwitchProfile 切换 API 配置预设
func (a *App) SwitchProfile(id uint) error {
	return a.profileService.SwitchProfile(id)
}

// TestAIBaseURL 测试 AI Base URL 连通性
func (a *App) TestAIBaseURL(baseURL, apiKey string) (bool, error) {
	cfg := a.aiService.GetConfig()
	cfg.BaseURL = baseURL
	cfg.APIKey = apiKey
	return a.aiService.TestConnection(cfg)
}

// TestTavilyConnection 测试 Tavily API Key 是否有效
func (a *App) TestTavilyConnection(apiKey string) (bool, error) {
	if apiKey == "" {
		return false, fmt.Errorf("API Key 不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := services.SearchWeb(ctx, "test", apiKey, 1)
	if err != nil {
		return false, fmt.Errorf("连接失败: %v", err)
	}
	if result == nil {
		return false, fmt.Errorf("连接失败，请检查 API Key 是否正确")
	}
	return true, nil
}

// TestZhihuConnection 测试知乎 Access Secret 是否有效
func (a *App) TestZhihuConnection(accessSecret string) (bool, error) {
	if accessSecret == "" {
		return false, fmt.Errorf("access Secret 不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := services.SearchZhihuContent(ctx, "test", accessSecret, 1)
	if err != nil {
		return false, fmt.Errorf("连接失败: %v", err)
	}
	if result == nil {
		return false, fmt.Errorf("连接失败，请检查 Access Secret 是否正确")
	}
	return true, nil
}

// FetchAIModels 获取可用模型列表
func (a *App) FetchAIModels(baseURL, apiKey string) ([]string, error) {
	cfg := a.aiService.GetConfig()
	cfg.BaseURL = baseURL
	cfg.APIKey = apiKey
	return a.aiService.FetchModels(cfg)
}

// SelectAIChatFiles 打开文件对话框选择文本文件，校验并读取内容返回给 AI 聊天使用
func (a *App) SelectAIChatFiles() ([]AIChatFileResult, error) {
	const maxSize int64 = 10 * 1024 * 1024 // 10MB

	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           "选择要上传的文本文件",
		ShowHiddenFiles: false,
	})
	if err != nil {
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if len(paths) == 0 {
		return []AIChatFileResult{}, nil // 用户取消
	}

	var results []AIChatFileResult

	for _, p := range paths {
		result := AIChatFileResult{Path: p, Name: filepath.Base(p)}

		// 检查路径
		info, err := os.Stat(p)
		if err != nil {
			result.Error = "无法访问文件: " + err.Error()
			results = append(results, result)
			continue
		}

		// 拒绝目录
		if info.IsDir() {
			result.Error = "不支持上传目录，请选择文件"
			results = append(results, result)
			continue
		}

		// 文件大小限制
		if info.Size() > maxSize {
			result.Error = "文件过大（超过 10MB），请选择小于 10MB 的文件"
			results = append(results, result)
			continue
		}
		result.Size = info.Size()

		// 二进制文件检测
		if fs.IsBinaryPath(p) {
			result.Error = "不支持二进制文件，请选择文本文件"
			results = append(results, result)
			continue
		}

		// 读取文件内容
		content, err := os.ReadFile(p)
		if err != nil {
			result.Error = "读取文件失败: " + err.Error()
			results = append(results, result)
			continue
		}
		contentStr := string(content)

		// 调用 GetAIRefMaxChars() 获取截断阈值（每次实时从 DB 读取）
		maxChars := a.GetAIRefMaxChars()
		if len(contentStr) > maxChars {
			truncMsg := fmt.Sprintf("\n\n...(内容已截断，完整文件共 %d 字)", len(contentStr))
			contentStr = contentStr[:maxChars] + truncMsg
			result.Truncated = true
		}

		result.Content = contentStr
		results = append(results, result)
	}

	return results, nil
}

// CallAI 调用 AI 对话接口
func (a *App) CallAI(messages []services.Message) (string, error) {
	return a.aiService.CallAI(messages)
}

// CallAIStream 流式调用 AI 对话接口（通过 EventsEmit 推送逐块内容）
func (a *App) CallAIStream(streamGen int, messages []services.Message, thinkingEnabled bool, searchSources []string, cardRecallEnabled bool, sessionID uint, isRegenerate bool, skillIds []string) {
	ctx, cancel := context.WithCancel(context.Background())
	a.aiStreamCancel = cancel

	var fullThinking strings.Builder
	var searchSourcesJSON, recallCardsJSON string

	// 在主 goroutine 发射搜索状态事件，确保前端能立即收到
	var searching bool
	if len(searchSources) > 0 {
		searching = true
		runtime.EventsEmit(a.ctx, "ai:search-status", "refining")
	}

	// 搜索 + 流式调用放进 goroutine，避免阻塞 Wails 事件循环
	go func() {
		// 注入基础身份提示词（仅在无笔记引用且无技能时）
		if len(skillIds) == 0 {
			hasSystem := false
			for i := range messages {
				if messages[i].Role == "system" {
					hasSystem = true
					break
				}
			}
			if !hasSystem {
				messages = append([]services.Message{
					{Role: "system", Content: "你是 Jot 智能助手，一款轻量级本地笔记应用的内置 AI。你可以帮助用户写作、编程、翻译、总结、答疑以及完成其他文本处理任务。请根据用户的提问提供准确、有用的回答。"},
				}, messages...)
			}
		}

		// 搜索源并行执行
		if len(searchSources) > 0 {
			cfg := a.aiService.GetConfig()

			var query string
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role == "user" {
					query = messages[i].Content
					break
				}
			}

			if query != "" {
				// 精炼 query
				refinedQuery, err := services.RefineSearchQuery(query, a.aiService)
				if err != nil {
					runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, "搜索关键词精炼失败: "+err.Error())
					return
				}
				if refinedQuery != "" {
					query = refinedQuery
				}
				runtime.EventsEmit(a.ctx, "ai:refined-keywords", query)

				// 发射 searching 状态，前端切换显示为"正在联网搜索..."
				runtime.EventsEmit(a.ctx, "ai:search-status", "searching")

				// 为每个搜索源发射 searching 状态
				for _, source := range searchSources {
					sourceStatus := map[string]interface{}{
						"source": source,
						"status": "searching",
					}
					statusJSON, _ := json.Marshal(sourceStatus)
					runtime.EventsEmit(a.ctx, "ai:search-source-status", string(statusJSON))
				}

				searchResultLimit := a.GetAISearchResultLimit()
				type searchResult struct {
					source string
					result *services.SearchWebResult
					err    error
				}
				resultCh := make(chan searchResult, len(searchSources))

				// 并行执行所有搜索源
				for _, source := range searchSources {
					go func(src string) {
						var r searchResult
						r.source = src
						switch src {
						case "tavily":
							r.result, r.err = services.SearchWeb(ctx, query, cfg.TavilyAPIKey, searchResultLimit)
						case "zhihu_search":
							r.result, r.err = services.SearchZhihuContent(ctx, query, cfg.ZhihuAccessSecret, searchResultLimit)
						case "zhihu_global":
							r.result, r.err = services.SearchGlobalContent(ctx, query, cfg.ZhihuAccessSecret, searchResultLimit)
						default:
							r.err = fmt.Errorf("未知搜索源: %s", src)
						}
						resultCh <- r
					}(source)
				}

				// 收集结果
				for i := 0; i < len(searchSources); i++ {
					r := <-resultCh
					if r.err != nil {
						// 发射错误事件给前端
						errEvent := map[string]interface{}{
							"source": r.source,
							"error":  r.err.Error(),
						}
						errJSON, _ := json.Marshal(errEvent)
						runtime.EventsEmit(a.ctx, "ai:search-error", string(errJSON))
					} else if r.result != nil {
						// 注入搜索结果到 system message
						found := false
						for i := range messages {
							if messages[i].Role == "system" {
								messages[i].Content = messages[i].Content + "\n\n" + r.result.FormattedText
								found = true
								break
							}
						}
						if !found {
							messages = append([]services.Message{{Role: "system", Content: r.result.FormattedText}}, messages...)
						}

						// 发射来源状态 success
						sourceStatus := map[string]interface{}{
							"source": r.source,
							"status": "success",
							"count":  len(r.result.Sources),
						}
						statusJSON, _ := json.Marshal(sourceStatus)
						runtime.EventsEmit(a.ctx, "ai:search-source-status", string(statusJSON))

						// 累积所有来源数据
						if len(r.result.Sources) > 0 {
							if searchSourcesJSON == "" {
								sJSON, _ := json.Marshal(r.result.Sources)
								searchSourcesJSON = string(sJSON)
							} else {
								var existing []services.SearchSource
								json.Unmarshal([]byte(searchSourcesJSON), &existing) //nolint:errcheck
								existing = append(existing, r.result.Sources...)
								sJSON, _ := json.Marshal(existing)
								searchSourcesJSON = string(sJSON)
							}
						}
					} else {
						// 无结果但也没错误
						sourceStatus := map[string]interface{}{
							"source": r.source,
							"status": "success",
							"count":  0,
						}
						statusJSON, _ := json.Marshal(sourceStatus)
						runtime.EventsEmit(a.ctx, "ai:search-source-status", string(statusJSON))
					}
				}
				close(resultCh)
			}
		}

		// 发射最终的 search-sources 事件
		if searchSourcesJSON != "" {
			runtime.EventsEmit(a.ctx, "ai:search-sources", searchSourcesJSON)
		}

		// 通知前端搜索完成，关闭搜索动画
		if searching {
			runtime.EventsEmit(a.ctx, "ai:search-status", "done")
		}

		// 卡片召回（在联网搜索之后执行）
		if cardRecallEnabled {
			var query string
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role == "user" {
					query = messages[i].Content
					break
				}
			}
			if query != "" {
				// 从 localStorage 读取召回条数（默认 3）
				recallLimit := 5
				if a.settingService != nil {
					if val := a.settingService.Get("ai_card_recall_limit"); val != "" {
						if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 30 {
							recallLimit = n
						}
					}
				}
				// 读取引用截断阈值（默认 5000）
				maxChars := 5000
				if a.settingService != nil {
					if val := a.settingService.Get("ai_ref_max_chars"); val != "" {
						if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 50000 {
							maxChars = n
						}
					}
				}
				recallResult := services.CardRecallSearch(ctx, query, recallLimit, maxChars, a.noteService)
				if recallResult != nil {
					// 注入格式化文本到 system role
					found := false
					for i := range messages {
						if messages[i].Role == "system" {
							messages[i].Content = messages[i].Content + "\n\n" + recallResult.FormattedText
							found = true
							break
						}
					}
					if !found {
						messages = append([]services.Message{{Role: "system", Content: recallResult.FormattedText}}, messages...)
					}

					// 发射结构化卡片数据给前端，并缓存用于持久化
					if len(recallResult.Cards) > 0 {
						cardsJSON, _ := json.Marshal(recallResult.Cards)
						recallCardsJSON = string(cardsJSON)
						runtime.EventsEmit(a.ctx, "ai:recall-cards", string(cardsJSON))
					}
				}
			}
		}

		// 技能提示词注入（在搜索和卡片召回之后执行）
		if len(skillIds) > 0 {
			skillPrompt, err := a.aiService.GetSkillPrompts(skillIds)
			if err == nil && skillPrompt != "" {
				found := false
				for i := range messages {
					if messages[i].Role == "system" {
						messages[i].Content = messages[i].Content + "\n\n" + skillPrompt
						found = true
						break
					}
				}
				if !found {
					messages = append([]services.Message{{Role: "system", Content: skillPrompt}}, messages...)
				}
			} else if err != nil {
				fmt.Printf("[SkillPrompt] 获取技能提示词失败: %v\n", err)
			}
		}

		// 如果已被用户取消（停止按钮），不再继续调用 LLM，避免白调用
		if ctx.Err() != nil {
			runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, "", 0.0, 0.0, 0, 0, 0)
			return
		}

		a.aiService.CallAIStream(ctx, messages, thinkingEnabled,
			func(chunk string) {
				runtime.EventsEmit(a.ctx, "ai:stream-chunk", streamGen, chunk)
			},
			func(thinking string) {
				fullThinking.WriteString(thinking)
				runtime.EventsEmit(a.ctx, "ai:stream-thinking", streamGen, thinking)
			},
			func(content string, elapsedThinking, elapsedTotal float64) {
				// 在后端统一计算 tokens
				userTokens := 0
				assistantTokens := estimateTokens(content)
				// 仅统计本轮 system 上下文（skill prompts、笔记引用、搜索注入、卡片召回）
				for _, msg := range messages {
					if msg.Role == "system" {
						userTokens += estimateTokens(msg.Content)
					}
				}
				// 仅统计最后一条用户消息（当前轮次的输入），不累加历史
				for i := len(messages) - 1; i >= 0; i-- {
					if messages[i].Role == "user" {
						userTokens += estimateTokens(messages[i].Content)
						break
					}
				}
				totalTokens := userTokens + assistantTokens

				// 由后端直接保存消息到数据库（含 tokens）
				assistantMsg := services.Message{
					Role:    "assistant",
					Content: content,
					ReasoningContent: func() string {
						if !thinkingEnabled {
							return ""
						}
						return fullThinking.String()
					}(),
					ThinkingElapsed: func() float64 {
						if !thinkingEnabled {
							return 0
						}
						return elapsedThinking
					}(),
					TotalElapsed:  elapsedTotal,
					Tokens:        assistantTokens,
					SearchSources: searchSourcesJSON,
					RecallCards:   recallCardsJSON,
				}
				if isRegenerate {
					// 再生模式只存 assistant
					_ = a.aiService.SaveAIMessages(sessionID, []services.Message{assistantMsg})
				} else {
					// 正常模式：提取最后一条 user 消息一同保存
					var userContent string
					for i := len(messages) - 1; i >= 0; i-- {
						if messages[i].Role == "user" {
							userContent = messages[i].Content
							break
						}
					}
					_ = a.aiService.SaveAIMessages(sessionID, []services.Message{
						{Role: "user", Content: userContent, Tokens: userTokens},
						assistantMsg,
					})
				}

				// 持久化会话的 context_tokens
				_ = a.aiService.UpdateSessionContextTokens(sessionID, totalTokens)

				// 通过 stream-done 一并返回 token 数据
				runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, content, elapsedThinking, elapsedTotal, totalTokens, userTokens, assistantTokens)
				if thinkingEnabled && fullThinking.Len() > 0 {
					runtime.EventsEmit(a.ctx, "ai:stream-thinking-done", fullThinking.String())
				}
			},
			func(err string) {
				runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, err)
			},
		)
	}()
}

// estimateTokens 估算文本的 token 数（与前端 estimateTokens 算法一致）
func estimateTokens(text string) int {
	chineseCount := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}
	runes := []rune(text)
	otherCount := len(runes) - chineseCount
	return int(math.Ceil(float64(chineseCount)/1.5 + float64(otherCount)/4))
}

// CancelAIStream 取消当前正在进行的 AI 流式调用
func (a *App) CancelAIStream() {
	if a.aiStreamCancel != nil {
		a.aiStreamCancel()
		a.aiStreamCancel = nil
	}
}

// GetAISessions 获取 AI 会话列表
func (a *App) GetAISessions() []services.AISessionSummary {
	return a.aiService.GetAISessions()
}

// TogglePinAISession 切换会话置顶状态
func (a *App) TogglePinAISession(id uint) error {
	return a.aiService.TogglePinAISession(id)
}

// CreateAISession 创建新 AI 会话，返回会话 ID
func (a *App) CreateAISession() uint {
	return a.aiService.CreateAISession()
}

// DeleteAISession 删除 AI 会话及所有消息
func (a *App) DeleteAISession(id uint) error {
	return a.aiService.DeleteAISession(id)
}

// RenameAISession 重命名 AI 会话
func (a *App) RenameAISession(id uint, title string) error {
	return a.aiService.RenameAISession(id, title)
}

// LoadAISessionMessages 加载 AI 会话的所有消息
func (a *App) LoadAISessionMessages(id uint) []services.Message {
	return a.aiService.LoadAISessionMessages(id)
}

// SaveAIMessages 保存一轮 AI 对话消息到指定会话
func (a *App) SaveAIMessages(sessionID uint, messages []services.Message) error {
	return a.aiService.SaveAIMessages(sessionID, messages)
}

// ClearAISessionMessages 清空 AI 会话的所有消息（不删会话）
func (a *App) ClearAISessionMessages(sessionID uint) error {
	return a.aiService.ClearAISessionMessages(sessionID)
}

// UpdateAIMessageContent 更新指定 AI 消息的内容
func (a *App) UpdateAIMessageContent(id uint, content string) error {
	return a.aiService.UpdateAIMessageContent(id, content)
}

// DeleteAIMessage 按 ID 删除单条 AI 消息
func (a *App) DeleteAIMessage(id uint) error {
	return a.aiService.DeleteAIMessage(id)
}

// DeleteAIMessagesAfter 删除指定会话中在指定消息之后的所有消息
func (a *App) DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error) {
	return a.aiService.DeleteAIMessagesAfter(sessionID, messageID)
}

// UpdateSessionContextTokens 更新会话的上下文 Token 数
func (a *App) UpdateSessionContextTokens(sessionID uint, tokens int) error {
	return a.aiService.UpdateSessionContextTokens(sessionID, tokens)
}

// ClearAllAISessions 清空所有 AI 会话及消息
func (a *App) ClearAllAISessions() error {
	return a.aiService.ClearAllAISessions()
}

// UpdateLastUserMessageTokens 更新指定会话中最后一条用户消息的 tokens
func (a *App) UpdateLastUserMessageTokens(sessionID uint, tokens int) error {
	return a.aiService.UpdateLastUserMessageTokens(sessionID, tokens)
}

// SaveAIMessageAsNote 将 AI 消息内容保存为笔记（归入默认笔记本）
func (a *App) SaveAIMessageAsNote(content string) (*models.Note, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("内容不能为空")
	}
	// 自动生成标题：取第一行，截断到 50 字符
	title := generateNoteTitle(content)
	// 保存到默认笔记本（id=1）
	return a.noteService.CreateWithNotebook(title, content, ".md", 1)
}

// generateNoteTitle 从内容中自动生成笔记标题
func generateNoteTitle(content string) string {
	// 取第一个有内容的行，去掉首尾空白
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			// 截断到 50 字符
			runes := []rune(trimmed)
			if len(runes) > 50 {
				return string(runes[:50]) + "..."
			}
			return trimmed
		}
	}
	return "AI 回复"
}

// GetSystemFonts 获取系统已安装的字体族列表
func (a *App) GetSystemFonts() []string {
	return fontutil.GetFonts()
}

// ==================== 排序与分页设置绑定方法 ====================

// GetSortOrder 获取排序方式设置
func (a *App) GetSortOrder() string {
	return a.settingService.Get("sort_order")
}

// SetSortOrder 保存排序方式设置
func (a *App) SetSortOrder(order string) error {
	return a.settingService.Set("sort_order", order)
}

// GetPageSize 获取分页大小设置
func (a *App) GetPageSize() int {
	size := a.settingService.Get("page_size")
	n, err := strconv.Atoi(size)
	if err != nil || n < 20 || n > 100 {
		return 20
	}
	return n
}

// SetPageSize 保存分页大小设置
func (a *App) SetPageSize(size int) error {
	return a.settingService.Set("page_size", strconv.Itoa(size))
}

// ==================== 版本与链接绑定方法 ====================

// GetVersion 获取应用版本号
func (a *App) GetVersion() string {
	return verman.V.GitVersion
}

// ExportNoteAsMarkdown 导出单条笔记为 Markdown 文件，弹出保存对话框让用户选择路径
func (a *App) ExportNoteAsMarkdown(id uint) (string, error) {
	note, err := a.noteService.GetByID(id)
	if err != nil {
		return "", fmt.Errorf("笔记不存在: %w", err)
	}

	defaultName := sanitizeFilename(note.Title) + note.FileExt
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出笔记",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "笔记文件 (*" + note.FileExt + ")", Pattern: "*" + note.FileExt},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	if err := os.WriteFile(filePath, []byte(note.Content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return "导出成功：" + filePath, nil
}

// ExportAISessionAsMarkdown 导出 AI 对话为 Markdown 文件
func (a *App) ExportAISessionAsMarkdown(sessionID uint) (string, error) {
	// 获取会话标题
	var session models.AISession
	if err := a.db.First(&session, sessionID).Error; err != nil {
		return "", fmt.Errorf("对话不存在: %w", err)
	}

	// 获取会话消息
	messages := a.aiService.LoadAISessionMessages(sessionID)

	// 构建 Markdown 内容
	var buf strings.Builder
	buf.WriteString("# " + session.Title + "\n\n---\n\n")

	for i, msg := range messages {
		if i > 0 {
			buf.WriteString("---\n\n")
		}

		switch msg.Role {
		case "user":
			buf.WriteString("**User**:\n" + msg.Content + "\n\n")
		case "assistant":
			buf.WriteString("**AI Assistant**:\n")
			if msg.ReasoningContent != "" {
				buf.WriteString("> 思考过程：\n")
				for _, line := range strings.Split(msg.ReasoningContent, "\n") {
					buf.WriteString("> " + line + "\n")
				}
				buf.WriteString("\n")
			}
			buf.WriteString(msg.Content + "\n\n")
		}
	}

	// 弹出保存对话框
	defaultName := sanitizeFilename(session.Title) + ".md"
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出对话",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown 文件 (*.md)", Pattern: "*.md"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	// 写入文件
	if err := os.WriteFile(filePath, []byte(buf.String()), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return "导出成功：" + filePath, nil
}

// sanitizeFilename 清理笔记标题，生成安全的文件名
// 白名单策略：仅保留英文字母、数字、中文、中文标点及安全符号，其余字符（emoji、特殊符号等）全部移除
func sanitizeFilename(title string) string {
	// step1: 白名单过滤 — 只保留英文、中文、数字、安全符号
	var b strings.Builder
	b.Grow(len(title))
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r >= 0x4e00 && r <= 0x9fff, // CJK 统一表意文字
			r >= 0x3000 && r <= 0x303f, // CJK 符号和标点
			r == '-' || r == '_' || r == '.' || r == '(' || r == ')' ||
				r == '[' || r == ']' || r == '{' || r == '}' || r == ',' ||
				r == ';' || r == '!' || r == '?' || r == '+' || r == '=' ||
				r == '~' || r == '@' || r == '#' || r == '&' || r == ' ':
			b.WriteRune(r)
		}
	}
	name := b.String()

	// step2: 原有清洗流程
	name = strings.TrimSpace(name)
	if name == "" {
		return "untitled"
	}
	// 替换无效文件名字符和空白为下划线
	re := regexp.MustCompile(`[\\/:*?"<>|\s]+`)
	name = re.ReplaceAllString(name, "_")
	// 合并连续下划线
	re2 := regexp.MustCompile(`_+`)
	name = re2.ReplaceAllString(name, "_")
	// 去掉首尾下划线
	name = strings.Trim(name, "_")
	if name == "" {
		return "untitled"
	}
	return name
}

// OpenDataDir 在文件管理器中打开数据库目录
func (a *App) OpenDataDir() error {
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(dbPath)
	cmd := exec.Command("explorer", dir)
	return cmd.Start()
}

// OpenProjectURL 在默认浏览器中打开项目地址
func (a *App) OpenProjectURL(url string) string {
	runtime.BrowserOpenURL(a.ctx, url)
	return "已打开浏览器"
}

// exportSnapshot 统一导出：VACUUM INTO → ZIP 打包 {jot-backup.db, images/} → 清理
func (a *App) exportSnapshot(destZipPath string) error {
	// 1. VACUUM INTO 临时 .db
	tempDB := destZipPath + ".tmpdb"
	defer func() { _ = os.Remove(tempDB) }()
	if err := a.noteService.ExportBackup(tempDB); err != nil {
		return fmt.Errorf("VACUUM INTO 失败: %w", err)
	}

	// 2. 获取图片目录
	imgDir, err := a.imageDirPath()
	if err != nil {
		return err
	}

	// 3. 创建 ZIP 文件
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("创建 ZIP 失败: %w", err)
	}
	defer func() { _ = zipFile.Close() }()

	zw := zip.NewWriter(zipFile)
	defer func() { _ = zw.Close() }()

	// 3a. 添加 db 文件（不压缩，SQLite 已是压缩状态）
	dbFile, err := os.Open(tempDB)
	if err != nil {
		return fmt.Errorf("打开临时 db 失败: %w", err)
	}
	defer func() { _ = dbFile.Close() }()

	dbInfo, _ := dbFile.Stat()
	dbHeader, err := zip.FileInfoHeader(dbInfo)
	if err != nil {
		return fmt.Errorf("创建 db header 失败: %w", err)
	}
	dbHeader.Name = "jot-backup.db"
	dbHeader.Method = zip.Store
	dbWriter, err := zw.CreateHeader(dbHeader)
	if err != nil {
		return fmt.Errorf("创建 db ZIP entry 失败: %w", err)
	}
	if _, err := io.Copy(dbWriter, dbFile); err != nil {
		return fmt.Errorf("写入 db 到 ZIP 失败: %w", err)
	}

	// 3b. 添加 images/ 目录中的文件
	entries, err := os.ReadDir(imgDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取图片目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		imgFile, err := os.Open(filepath.Join(imgDir, entry.Name()))
		if err != nil {
			continue
		}
		imgInfo, _ := imgFile.Stat()
		imgHeader, err := zip.FileInfoHeader(imgInfo)
		if err != nil {
			_ = imgFile.Close()
			continue
		}
		imgHeader.Name = "images/" + entry.Name()
		imgHeader.Method = zip.Deflate
		imgWriter, err := zw.CreateHeader(imgHeader)
		if err != nil {
			_ = imgFile.Close()
			continue
		}
		_, _ = io.Copy(imgWriter, imgFile)
		_ = imgFile.Close()
	}

	return nil
}

// replaceDatabase 统一替换：备份当前 db → 关闭连接 → 替换 db + images → 重连 → 重建服务
func (a *App) replaceDatabase(srcDBPath, srcImagesDir string) error {
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("获取数据库路径失败: %w", err)
	}

	// Step 1: 备份当前数据库
	backupPath := dbPath + ".bak"
	if err := fs.CopyEx(dbPath, backupPath, true); err != nil {
		return fmt.Errorf("备份当前数据库失败: %w", err)
	}

	// 失败的回滚
	rollback := func() {
		_ = fs.CopyEx(backupPath, dbPath, true)
		_ = a.reconnectDB(dbPath)
	}

	// Step 2: 关闭旧连接
	sqlDB, err := a.db.DB()
	if err != nil {
		_ = os.Remove(backupPath)
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	_ = sqlDB.Close()

	// Step 3: 复制 db 文件
	if err := fs.CopyEx(srcDBPath, dbPath, true); err != nil {
		rollback()
		return fmt.Errorf("复制数据库文件失败: %w", err)
	}

	// Step 4: 替换 images/ 目录
	if srcImagesDir != "" {
		imgDir, err := a.imageDirPath()
		if err == nil {
			_ = os.RemoveAll(imgDir)
			_ = os.MkdirAll(imgDir, 0755)
			entries, _ := os.ReadDir(srcImagesDir)
			for _, entry := range entries {
				if !entry.IsDir() {
					src := filepath.Join(srcImagesDir, entry.Name())
					dst := filepath.Join(imgDir, entry.Name())
					_ = fs.CopyEx(src, dst, true)
				}
			}
		}
	}

	// Step 5: 重新初始化数据库
	newDB, err := database.InitDB(dbPath)
	if err != nil {
		rollback()
		return fmt.Errorf("数据库重连失败: %w", err)
	}

	// Step 6: 重建服务
	a.db = newDB
	a.rebuildServices(newDB)

	// Step 7: 清理备份
	_ = os.Remove(backupPath)

	return nil
}

// importFromArchive 统一导入：解压 ZIP → 提取 db + images → replaceDatabase
func (a *App) importFromArchive(srcZipPath string) error {
	// 解压到临时目录
	tmpDir := filepath.Join(os.TempDir(), "jot-restore-"+fmt.Sprintf("%x", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	reader, err := zip.OpenReader(srcZipPath)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer func() { _ = reader.Close() }()

	var dbSrc string
	var imagesDir string

	for _, f := range reader.File {
		destPath := filepath.Join(tmpDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)

		rc, err := f.Open()
		if err != nil {
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			_ = rc.Close()
			continue
		}

		_, _ = io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()

		if f.Name == "jot-backup.db" {
			dbSrc = destPath
		} else if strings.HasPrefix(f.Name, "images/") {
			imagesDir = filepath.Dir(destPath)
		}
	}

	if dbSrc == "" {
		return fmt.Errorf("ZIP 文件中未找到 jot-backup.db")
	}

	return a.replaceDatabase(dbSrc, imagesDir)
}

// ==================== 一键备份与还原绑定方法 ====================

// BackupToDir 一键备份到 ~/.jot/backup/ 目录，固定文件名 jot-backup.zip（覆盖旧备份）
func (a *App) BackupToDir() (string, error) {
	backupDir, err := database.BackupDir()
	if err != nil {
		return "", fmt.Errorf("获取备份目录失败: %w", err)
	}
	if err := database.EnsureBackupDir(); err != nil {
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	zipPath := filepath.Join(backupDir, "jot-backup.zip")

	// 先删除旧备份
	_ = os.Remove(zipPath)

	if err := a.exportSnapshot(zipPath); err != nil {
		return "", fmt.Errorf("备份失败: %w", err)
	}

	return "备份成功：jot-backup.zip", nil
}

// RestoreFromDir 从 backup 目录的 jot-backup.zip 还原备份（含图片）
func (a *App) RestoreFromDir() (*services.ImportResult, error) {
	backupDir, err := database.BackupDir()
	if err != nil {
		return &services.ImportResult{Message: "获取备份目录失败：" + err.Error()}, nil
	}

	zipPath := filepath.Join(backupDir, "jot-backup.zip")

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return &services.ImportResult{Message: "暂无可用备份"}, nil
	} else if err != nil {
		return &services.ImportResult{Message: "读取备份文件失败：" + err.Error()}, nil
	}

	if err := a.importFromArchive(zipPath); err != nil {
		return &services.ImportResult{Message: "还原失败：" + err.Error()}, nil
	}

	return &services.ImportResult{
		Message:      "已从备份文件恢复：jot-backup.zip",
		SuccessCount: 1,
	}, nil
}

// GetBackupInfo 获取备份文件信息（文件名、修改时间、文件大小），无备份时返回空值
func (a *App) GetBackupInfo() (map[string]string, error) {
	backupDir, err := database.BackupDir()
	if err != nil {
		return map[string]string{"file_name": "", "file_time": "", "file_size": ""}, nil
	}

	filePath := filepath.Join(backupDir, "jot-backup.zip")
	fi, err := os.Stat(filePath)
	if err != nil {
		return map[string]string{"file_name": "", "file_time": "", "file_size": ""}, nil
	}

	size := fi.Size()
	var sizeStr string
	switch {
	case size < 1024:
		sizeStr = fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}

	return map[string]string{
		"file_name": "jot-backup.zip",
		"file_time": fi.ModTime().Format("2006-01-02 15:04"),
		"file_size": sizeStr,
	}, nil
}

// FileImportResult 单个文件导入结果
type FileImportResult struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	NoteID  uint   `json:"note_id"`
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// AIChatFileResult AI 聊天上传文件的处理结果
type AIChatFileResult struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}

// ImportFiles 批量导入拖拽文件为笔记（归入指定笔记本）
func (a *App) ImportFiles(paths []string, notebookID uint) []FileImportResult {
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		return []FileImportResult{{
			Path:    "",
			Success: false,
			Error:   "获取默认笔记本失败: " + err.Error(),
		}}
	}

	const maxSize int64 = 10 * 1024 * 1024 // 10MB
	var results []FileImportResult

	for _, p := range paths {
		result := FileImportResult{Path: p}

		// 检查路径
		info, err := os.Stat(p)
		if err != nil {
			result.Error = "无法访问文件: " + err.Error()
			results = append(results, result)
			continue
		}

		// 拒绝目录
		if info.IsDir() {
			result.Error = "不支持导入目录，请选择文件"
			results = append(results, result)
			continue
		}

		// 文件大小限制
		if info.Size() > maxSize {
			result.Error = "文件过大（超过 10MB），无法导入"
			results = append(results, result)
			continue
		}

		// 二进制文件检测（go-kit/fs 读取前 8000 字节检查是否包含空字符）
		if fs.IsBinaryPath(p) {
			result.Error = "不支持导入二进制文件，请选择文本文件后重试"
			results = append(results, result)
			continue
		}

		// 读取文件内容
		content, err := os.ReadFile(p)
		if err != nil {
			result.Error = "读取文件失败: " + err.Error()
			results = append(results, result)
			continue
		}

		// 提取文件名（去后缀）作标题
		name := filepath.Base(p)
		ext := filepath.Ext(name)
		title := strings.TrimSuffix(name, ext)
		if title == "" {
			title = "untitled"
		}

		// 确定文件后缀：.md 文件保持 .md，其他文件按原始后缀处理
		fileExt := ext
		if fileExt == "" {
			fileExt = ".txt"
		}

		// 创建笔记（归入指定笔记本）
		note, err := a.noteService.CreateWithNotebook(title, string(content), fileExt, notebookID)
		if err != nil {
			result.Error = "创建笔记失败: " + err.Error()
			results = append(results, result)
			continue
		}

		result.Title = title
		result.NoteID = note.ID
		result.Success = true
		results = append(results, result)
	}

	return results
}

// ResetDatabase 清空所有数据，恢复出厂状态（删表重建）
func (a *App) ResetDatabase() error {
	// 1. 删除所有表（自动处理外键依赖顺序）
	tables := []interface{}{
		&models.AIMessage{},
		&models.AISession{},
		&models.APIProfile{},
		&models.Setting{},
		&models.Note{},
		&models.Tag{},
		&models.Notebook{},
	}
	for _, table := range tables {
		if err := a.db.Migrator().DropTable(table); err != nil {
			return err
		}
	}

	// 2. 重新 AutoMigrate（与 InitDB 保持同步）
	if err := a.db.AutoMigrate(&models.Note{}, &models.Tag{}, &models.Setting{},
		&models.Notebook{}, &models.AISession{}, &models.AIMessage{}, &models.APIProfile{}); err != nil {
		return err
	}

	// 3. 重新初始化默认标签
	if err := services.InitDefaultTags(a.db); err != nil {
		return err
	}

	// 4. 重新初始化默认设置
	if err := database.InitDefaultSettings(a.db); err != nil {
		return err
	}

	// 5. 确保默认笔记本存在
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		return err
	}

	// 6. 重建数据库连接（DropTable 后 glebarez/sqlite 驱动连接可能失效）
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("获取数据库路径失败: %w", err)
	}
	if err := a.reconnectDB(dbPath); err != nil {
		return fmt.Errorf("重置后重连失败: %w", err)
	}

	// 7. 清空图片目录
	imgDir, err := a.imageDirPath()
	if err == nil {
		_ = os.RemoveAll(imgDir)
		_ = os.MkdirAll(imgDir, 0755)
	}

	return nil
}

// rebuildServices 使用新的数据库连接重建所有服务实例
func (a *App) rebuildServices(db *gorm.DB) {
	a.settingService = services.NewSettingService(db)
	a.noteService = services.NewNoteService(db, a.settingService)
	a.tagService = services.NewTagService(db)
	a.notebookService = services.NewNotebookService(db)
	a.aiService = services.NewAIService(db)
	a.profileService = services.NewProfileService(db)
	a.todoService = services.NewTodoService(db)
}

// ==================== Todo 相关绑定方法 ====================

func (a *App) CreateTodo(text string) (*models.Todo, error) {
	return a.todoService.Create(text)
}

func (a *App) ListTodos() ([]models.Todo, error) {
	return a.todoService.List()
}

func (a *App) ToggleTodo(id uint) (*models.Todo, error) {
	return a.todoService.Toggle(id)
}

func (a *App) DeleteTodo(id uint) error {
	return a.todoService.Delete(id)
}

func (a *App) UpdateTodo(id uint, text string) (*models.Todo, error) {
	return a.todoService.Update(id, text)
}

func (a *App) ClearCompletedTodos() (string, error) {
	count, err := a.todoService.DeleteCompleted()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已清空 %d 个已完成待办事项", count), nil
}

// reconnectDB 重新连接数据库（用于导入失败后的恢复）
func (a *App) reconnectDB(dbPath string) error {
	// 关闭旧连接
	if sqlDB, err := a.db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("数据库重连失败: %w", err)
	}
	a.db = db
	a.rebuildServices(db)
	return nil
}
