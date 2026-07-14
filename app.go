package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jot/internal/aicli"
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

	"gitee.com/MM-Q/fastlog"
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
	LogSvc          *services.LogService
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
	// 初始化日志服务（先创建设置读取日志级别，等 startup 再真正初始化 logger）
	logSvc := services.NewLogService()
	return &App{
		db:              db,
		noteService:     services.NewNoteService(db, settingService, logSvc.Logger),
		tagService:      services.NewTagService(db, logSvc.Logger),
		settingService:  settingService,
		notebookService: services.NewNotebookService(db, logSvc.Logger),
		aiService:       services.NewAIService(db, logSvc.Logger),
		profileService:  services.NewProfileService(db, logSvc.Logger),
		todoService:     services.NewTodoService(db, logSvc.Logger),
		LogSvc:          logSvc,
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
	// 迁移存量明文密钥为 Base64 编码格式（放在旧迁移之后，确保旧迁移逻辑读到的是明文）
	a.migrateSensitiveKeys()

	// 初始化日志服务
	logDir := filepath.Join(home, ".jot", "logs")
	logLevelStr := a.settingService.Get("log_level")
	logLevelVal := 1
	if n, err := strconv.Atoi(logLevelStr); err == nil {
		logLevelVal = n
	}
	logLevel := services.LevelFromInt(logLevelVal)
	if err := a.LogSvc.Init(logDir, logLevel); err != nil {
		// 日志初始化失败不应阻止应用启动
		println("日志初始化失败:", err.Error())
	} else {
		a.LogSvc.Logger.Infow("数据库连接成功")
		a.LogSvc.Logger.Infow("默认笔记本已就绪")
		a.LogSvc.Logger.Infow("密钥迁移完成")
		a.LogSvc.Logger.Infow("启动初始化完成",
			fastlog.String("version", verman.V.GitVersion),
		)
	}
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.LogSvc != nil {
		a.LogSvc.Close()
	}
}

// migrateSensitiveKeys 迁移存量明文密钥为 Base64 编码格式（(zk) 前缀）
func (a *App) migrateSensitiveKeys() {
	// 迁移 settings 表
	keys := []string{"ai_api_key", "tavily_api_key", "zhihu_access_secret"}
	for _, key := range keys {
		var setting models.Setting
		if err := a.db.Where("key = ?", key).First(&setting).Error; err != nil {
			continue // 无记录则跳过
		}
		val := setting.Value
		if val == "" || strings.HasPrefix(val, "(zk)") {
			continue
		}
		encoded := services.EncodeB64(val)
		a.db.Model(&setting).Update("value", encoded)
		a.LogSvc.Logger.Infow("迁移密钥已编码", fastlog.String("key", key))
	}

	// 迁移 api_profiles 表的 api_key 字段
	var profiles []models.APIProfile
	a.db.Find(&profiles)
	for _, p := range profiles {
		if p.APIKey == "" || strings.HasPrefix(p.APIKey, "(zk)") {
			continue
		}
		encoded := services.EncodeB64(p.APIKey)
		a.db.Model(&models.APIProfile{}).Where("id = ?", p.ID).Update("api_key", encoded)
		a.LogSvc.Logger.Infow("迁移预设密钥已编码", fastlog.String("profile", p.Name))
	}
}

// ==================== 图片相关方法 ====================

// SaveImage 保存图片到 ~/.jot/images/，返回可访问的 URL 路径
// name: 原始文件名, data: base64 编码的图片数据
// 返回: /images/uuid_name.ext 格式的 URL
func (a *App) SaveImage(name string, data string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("解码图片数据失败: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	imageDir := filepath.Join(home, ".jot", "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	uuid := fmt.Sprintf("%x", b)

	filename := uuid + "_" + name
	filePath := filepath.Join(imageDir, filename)
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("写入图片文件失败: %w", err)
	}

	a.LogSvc.Logger.Infow("图片保存成功",
		fastlog.String("file", filename),
	)
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
	a.LogSvc.Logger.Debugw("CleanupOrphanImages")
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
	a.LogSvc.Logger.Debugw("CreateNote", fastlog.String("title", title))
	note, err := a.noteService.CreateWithNotebook(title, content, fileExt, notebookID)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// UpdateNote 更新指定笔记的标题和内容
func (a *App) UpdateNote(id uint, title, content, fileExt string) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("UpdateNote", fastlog.Uint("id", id))
	note, err := a.noteService.Update(id, title, content, fileExt)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// UpdateNoteFileExt 更新指定笔记的文件后缀（不修改其他字段）
func (a *App) UpdateNoteFileExt(id uint, fileExt string) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("UpdateNoteFileExt", fastlog.Uint("id", id), fastlog.String("fileExt", fileExt))
	note, err := a.noteService.UpdateFileExt(id, fileExt)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateNoteFileExt 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateNoteFileExt 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// DeleteNote 软删除指定笔记（移入回收站）
func (a *App) DeleteNote(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteNote", fastlog.Uint("id", id))
	if err := a.noteService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteNote 成功", fastlog.Uint("id", id))
	return nil
}

// PermanentDeleteNote 永久删除指定笔记（从数据库彻底移除）
func (a *App) PermanentDeleteNote(id uint) error {
	a.LogSvc.Logger.Debugw("PermanentDeleteNote", fastlog.Uint("id", id))
	if err := a.noteService.PermanentDelete(id); err != nil {
		a.LogSvc.Logger.Errorw("PermanentDeleteNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("PermanentDeleteNote 成功", fastlog.Uint("id", id))
	return nil
}

// GetNote 按 ID 获取单条笔记
func (a *App) GetNote(id uint) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("GetNote", fastlog.Uint("id", id))
	note, err := a.noteService.GetByID(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNote 失败", fastlog.Error(err))
		return nil, err
	}
	return note, nil
}

// GetNoteContent 按 ID 仅获取笔记的完整 content 文本（列表查询只返回截断版本，用于编辑器按需加载）
func (a *App) GetNoteContent(id uint) (string, error) {
	a.LogSvc.Logger.Debugw("GetNoteContent", fastlog.Uint("id", id))
	content, err := a.noteService.GetNoteContent(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNoteContent 失败", fastlog.Error(err))
		return "", err
	}
	return content, nil
}

// GetNoteRefContext 构建笔记引用上下文。
// 后端一次性完成：查库 → 截断 → 拼装，返回每条笔记信息和完整 context 文本。
func (a *App) GetNoteRefContext(ids []uint) (*services.NoteRefContext, error) {
	a.LogSvc.Logger.Debugw("GetNoteRefContext", fastlog.Int("count", len(ids)))
	ctx, err := a.noteService.BuildNoteRefContext(ids)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNoteRefContext 失败", fastlog.Error(err))
		return nil, err
	}
	return ctx, nil
}

// GetNotes 分页获取未删除的笔记列表，支持指定排序方式和笔记本筛选
func (a *App) GetNotes(page, pageSize int, sortBy string, notebookID uint) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetNotes", fastlog.Int("page", page), fastlog.Int("pageSize", pageSize), fastlog.Uint("notebookID", notebookID))
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.GetAllByNotebook(page, pageSize, sortBy, notebookID)
	} else {
		notes, total, err = a.noteService.GetAll(page, pageSize, sortBy)
	}
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotes 失败", fastlog.Error(err))
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
	a.LogSvc.Logger.Debugw("GetAllNoteIDs")
	ids, err := a.noteService.GetAllIDs()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetAllNoteIDs 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// SearchNotes 按关键词搜索笔记（标题/内容），支持分页、笔记本筛选、日期范围和标签 AND 过滤
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string, tagIDs []uint) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("SearchNotes", fastlog.String("keyword", keyword), fastlog.Int("page", page), fastlog.Int("pageSize", pageSize), fastlog.Uint("notebookID", notebookID))
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.SearchByNotebook(keyword, page, pageSize, notebookID, sortBy, startDate, endDate, tagIDs)
	} else {
		notes, total, err = a.noteService.Search(keyword, page, pageSize, sortBy, startDate, endDate, tagIDs)
	}
	if err != nil {
		a.LogSvc.Logger.Errorw("SearchNotes 失败", fastlog.Error(err))
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
	a.LogSvc.Logger.Debugw("SearchNoteIDs", fastlog.String("keyword", keyword), fastlog.Uint("notebookID", notebookID), fastlog.Int("tagCount", len(tagIDs)))
	var ids []uint
	var err error
	if notebookID > 0 {
		ids, err = a.noteService.SearchNoteIDsByNotebook(keyword, notebookID, tagIDs)
	} else {
		ids, err = a.noteService.SearchNoteIDs(keyword, tagIDs)
	}
	if err != nil {
		a.LogSvc.Logger.Errorw("SearchNoteIDs 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// TogglePinNote 切换指定笔记的置顶状态
func (a *App) TogglePinNote(id uint) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("TogglePinNote", fastlog.Uint("id", id))
	note, err := a.noteService.TogglePin(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("TogglePinNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("TogglePinNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// GetTrashNotes 分页获取回收站中的笔记列表
func (a *App) GetTrashNotes(page, pageSize int) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetTrashNotes", fastlog.Int("page", page), fastlog.Int("pageSize", pageSize))
	notes, total, err := a.noteService.GetTrash(page, pageSize)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetTrashNotes 失败", fastlog.Error(err))
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
	a.LogSvc.Logger.Debugw("RestoreNote", fastlog.Uint("id", id))
	if err := a.noteService.Restore(id); err != nil {
		a.LogSvc.Logger.Errorw("RestoreNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreNote 成功", fastlog.Uint("id", id))
	return nil
}

// BatchPinNotes 批量置顶或取消置顶笔记
func (a *App) BatchPinNotes(noteIDs []uint, pin bool) error {
	a.LogSvc.Logger.Debugw("BatchPinNotes", fastlog.Int("count", len(noteIDs)), fastlog.Bool("pin", pin))
	if err := a.noteService.BatchPinNotes(noteIDs, pin); err != nil {
		a.LogSvc.Logger.Errorw("BatchPinNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchPinNotes 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// BatchDeleteNotes 批量软删除笔记
func (a *App) BatchDeleteNotes(ids []uint) error {
	a.LogSvc.Logger.Debugw("BatchDeleteNotes", fastlog.Int("count", len(ids)))
	if err := a.noteService.BatchDelete(ids); err != nil {
		a.LogSvc.Logger.Errorw("BatchDeleteNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchDeleteNotes 成功", fastlog.Int("count", len(ids)))
	return nil
}

// BatchRestoreNotes 批量从回收站恢复笔记
func (a *App) BatchRestoreNotes(ids []uint) error {
	a.LogSvc.Logger.Debugw("BatchRestoreNotes", fastlog.Int("count", len(ids)))
	if err := a.noteService.BatchRestore(ids); err != nil {
		a.LogSvc.Logger.Errorw("BatchRestoreNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchRestoreNotes 成功", fastlog.Int("count", len(ids)))
	return nil
}

// BatchAddTagToNotes 批量添加标签到笔记
func (a *App) BatchAddTagToNotes(noteIDs []uint, tagID uint) error {
	a.LogSvc.Logger.Debugw("BatchAddTagToNotes", fastlog.Int("count", len(noteIDs)), fastlog.Uint("tagID", tagID))
	if err := a.tagService.BatchAddTagToNotes(noteIDs, tagID); err != nil {
		a.LogSvc.Logger.Errorw("BatchAddTagToNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchAddTagToNotes 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// BatchRemoveTagFromNotes 批量从笔记移除标签
func (a *App) BatchRemoveTagFromNotes(noteIDs []uint, tagID uint) error {
	a.LogSvc.Logger.Debugw("BatchRemoveTagFromNotes", fastlog.Int("count", len(noteIDs)), fastlog.Uint("tagID", tagID))
	if err := a.tagService.BatchRemoveTagFromNotes(noteIDs, tagID); err != nil {
		a.LogSvc.Logger.Errorw("BatchRemoveTagFromNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchRemoveTagFromNotes 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// RestoreAllNotes 批量恢复回收站中所有笔记
func (a *App) RestoreAllNotes() error {
	a.LogSvc.Logger.Debugw("RestoreAllNotes")
	if err := a.noteService.RestoreAll(); err != nil {
		a.LogSvc.Logger.Errorw("RestoreAllNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreAllNotes 成功")
	return nil
}

// EmptyTrash 永久清空回收站中所有笔记
func (a *App) EmptyTrash() error {
	a.LogSvc.Logger.Debugw("EmptyTrash")
	if err := a.noteService.EmptyTrash(); err != nil {
		a.LogSvc.Logger.Errorw("EmptyTrash 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("EmptyTrash 成功")
	return nil
}

// GetNotesByTag 按标签分页获取笔记，支持指定排序方式（updated_at/created_at/title）
func (a *App) GetNotesByTag(tagID uint, page, pageSize int, sortBy string) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetNotesByTag", fastlog.Uint("tagID", tagID), fastlog.Int("page", page), fastlog.Int("pageSize", pageSize))
	notes, total, err := a.noteService.GetByTag(tagID, page, pageSize, sortBy)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotesByTag 失败", fastlog.Error(err))
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
	a.LogSvc.Logger.Debugw("GetDataStats")
	stats, err := a.noteService.GetStats()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetDataStats 失败", fastlog.Error(err))
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
	a.LogSvc.Logger.Debugw("VacuumDatabase")
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
		a.LogSvc.Logger.Errorw("VacuumDatabase 失败", fastlog.Error(err))
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
	result := strings.Join(parts, "，")
	a.LogSvc.Logger.Infow("存储优化完成",
		fastlog.String("result", result),
	)
	return result, nil
}

// ExportDataWithDialog 弹出保存对话框，导出 ZIP 格式备份（含数据库和图片）
func (a *App) ExportDataWithDialog() (string, error) {
	a.LogSvc.Logger.Debugw("ExportDataWithDialog")
	// 弹出保存对话框
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出数据备份",
		DefaultFilename: "jot-backup-" + time.Now().Format("2006-01-02") + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		a.LogSvc.Logger.Errorw("ExportDataWithDialog 失败", fastlog.Error(err))
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	// 调用统一导出
	if err := a.exportSnapshot(filePath); err != nil {
		a.LogSvc.Logger.Errorw("ExportDataWithDialog 失败", fastlog.Error(err))
		return "", fmt.Errorf("导出失败: %w", err)
	}

	a.LogSvc.Logger.Infow("数据导出成功")
	return "导出成功：" + filePath, nil
}

// ImportDatabaseWithDialog 弹出文件选择对话框，从 ZIP 备份文件恢复数据（含图片）
func (a *App) ImportDatabaseWithDialog() (*services.ImportResult, error) {
	a.LogSvc.Logger.Debugw("ImportDatabaseWithDialog")
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入数据备份",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		a.LogSvc.Logger.Errorw("ImportDatabaseWithDialog 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}
	if filePath == "" {
		return &services.ImportResult{Message: "已取消"}, nil
	}

	if err := a.importFromArchive(filePath); err != nil {
		a.LogSvc.Logger.Errorw("ImportDatabaseWithDialog 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}

	a.LogSvc.Logger.Infow("数据导入成功")
	return &services.ImportResult{
		Message:      "已从备份文件恢复数据库与图片",
		SuccessCount: 1,
	}, nil
}

// ==================== Tag 相关绑定方法 ====================

// CreateTag 创建一个新标签
func (a *App) CreateTag(name, color string) (*models.Tag, error) {
	a.LogSvc.Logger.Debugw("CreateTag", fastlog.String("name", name))
	tag, err := a.tagService.Create(name, color)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateTag 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateTag 成功", fastlog.Uint("id", tag.ID))
	return tag, nil
}

// UpdateTag 更新指定标签的名称和颜色
func (a *App) UpdateTag(id uint, name, color string) (*models.Tag, error) {
	a.LogSvc.Logger.Debugw("UpdateTag", fastlog.Uint("id", id))
	tag, err := a.tagService.Update(id, name, color)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateTag 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateTag 成功", fastlog.Uint("id", tag.ID))
	return tag, nil
}

// DeleteTag 删除指定标签
func (a *App) DeleteTag(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteTag", fastlog.Uint("id", id))
	if err := a.tagService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteTag 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteTag 成功", fastlog.Uint("id", id))
	return nil
}

// GetAllTags 获取所有标签列表
func (a *App) GetAllTags() ([]models.Tag, error) {
	a.LogSvc.Logger.Debugw("GetAllTags")
	tags, err := a.tagService.GetAll()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetAllTags 失败", fastlog.Error(err))
		return nil, err
	}
	return tags, nil
}

// AddTagToNote 为指定笔记添加标签
func (a *App) AddTagToNote(noteID, tagID uint) error {
	a.LogSvc.Logger.Debugw("AddTagToNote", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	if err := a.tagService.AddTagToNote(noteID, tagID); err != nil {
		a.LogSvc.Logger.Errorw("AddTagToNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("AddTagToNote 成功", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	return nil
}

// RemoveTagFromNote 为指定笔记移除标签
func (a *App) RemoveTagFromNote(noteID, tagID uint) error {
	a.LogSvc.Logger.Debugw("RemoveTagFromNote", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	if err := a.tagService.RemoveTagFromNote(noteID, tagID); err != nil {
		a.LogSvc.Logger.Errorw("RemoveTagFromNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RemoveTagFromNote 成功", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	return nil
}

// ==================== Notebook 相关绑定方法 ====================

// CreateNotebook 创建新笔记本
func (a *App) CreateNotebook(name string) (*models.Notebook, error) {
	a.LogSvc.Logger.Debugw("CreateNotebook", fastlog.String("name", name))
	notebook, err := a.notebookService.Create(name)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateNotebook 成功", fastlog.Uint("id", notebook.ID))
	return notebook, nil
}

// RenameNotebook 重命名笔记本
func (a *App) RenameNotebook(id uint, name string) (*models.Notebook, error) {
	a.LogSvc.Logger.Debugw("RenameNotebook", fastlog.Uint("id", id))
	notebook, err := a.notebookService.Update(id, name)
	if err != nil {
		a.LogSvc.Logger.Errorw("RenameNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("RenameNotebook 成功", fastlog.Uint("id", notebook.ID))
	return notebook, nil
}

// DeleteNotebook 删除笔记本，其下笔记自动迁入默认笔记本
func (a *App) DeleteNotebook(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteNotebook", fastlog.Uint("id", id))
	if err := a.notebookService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// DeleteNotebookWithNotes 删除笔记本并清空其下所有笔记
func (a *App) DeleteNotebookWithNotes(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteNotebookWithNotes", fastlog.Uint("id", id))
	if err := a.notebookService.DeleteWithNotes(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteNotebookWithNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteNotebookWithNotes 成功", fastlog.Uint("id", id))
	return nil
}

// GetTrashNotebooks 分页获取回收站中已删除的笔记本列表
func (a *App) GetTrashNotebooks(page, pageSize int) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetTrashNotebooks", fastlog.Int("page", page), fastlog.Int("pageSize", pageSize))
	notebooks, total, err := a.notebookService.GetTrash(page, pageSize)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetTrashNotebooks 失败", fastlog.Error(err))
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
	a.LogSvc.Logger.Debugw("RestoreTrashNotebook", fastlog.Uint("id", id))
	if err := a.notebookService.RestoreFromTrash(id); err != nil {
		a.LogSvc.Logger.Errorw("RestoreTrashNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreTrashNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// PermanentDeleteTrashNotebook 从回收站永久删除指定笔记本
func (a *App) PermanentDeleteTrashNotebook(id uint) error {
	a.LogSvc.Logger.Debugw("PermanentDeleteTrashNotebook", fastlog.Uint("id", id))
	if err := a.notebookService.PermanentDeleteFromTrash(id); err != nil {
		a.LogSvc.Logger.Errorw("PermanentDeleteTrashNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("PermanentDeleteTrashNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// RestoreAllTrashNotebooks 恢复回收站中所有笔记本
func (a *App) RestoreAllTrashNotebooks() error {
	a.LogSvc.Logger.Debugw("RestoreAllTrashNotebooks")
	if err := a.notebookService.RestoreAllFromTrash(); err != nil {
		a.LogSvc.Logger.Errorw("RestoreAllTrashNotebooks 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreAllTrashNotebooks 成功")
	return nil
}

// EmptyTrashNotebooks 清空回收站中所有笔记本
func (a *App) EmptyTrashNotebooks() error {
	a.LogSvc.Logger.Debugw("EmptyTrashNotebooks")
	if err := a.notebookService.EmptyTrash(); err != nil {
		a.LogSvc.Logger.Errorw("EmptyTrashNotebooks 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("EmptyTrashNotebooks 成功")
	return nil
}

// MoveNoteToNotebook 将单条笔记移动到目标笔记本
func (a *App) MoveNoteToNotebook(id, targetNotebookID uint) error {
	a.LogSvc.Logger.Debugw("MoveNoteToNotebook", fastlog.Uint("id", id), fastlog.Uint("targetNotebookID", targetNotebookID))
	if err := a.noteService.MoveToNotebook(id, targetNotebookID); err != nil {
		a.LogSvc.Logger.Errorw("MoveNoteToNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("MoveNoteToNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// BatchMoveNotesToNotebook 批量将多条笔记移动到目标笔记本
func (a *App) BatchMoveNotesToNotebook(noteIDs []uint, targetNotebookID uint) error {
	a.LogSvc.Logger.Debugw("BatchMoveNotesToNotebook", fastlog.Int("count", len(noteIDs)), fastlog.Uint("targetNotebookID", targetNotebookID))
	if err := a.noteService.BatchMoveToNotebook(noteIDs, targetNotebookID); err != nil {
		a.LogSvc.Logger.Errorw("BatchMoveNotesToNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchMoveNotesToNotebook 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// GetAllNotebooks 获取所有未删除笔记本列表
func (a *App) GetAllNotebooks() ([]models.Notebook, error) {
	a.LogSvc.Logger.Debugw("GetAllNotebooks")
	notebooks, err := a.notebookService.GetAll()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetAllNotebooks 失败", fastlog.Error(err))
		return nil, err
	}
	return notebooks, nil
}

// GetNotebookNoteCounts 获取各笔记本下笔记数量
func (a *App) GetNotebookNoteCounts() (map[uint]int, error) {
	a.LogSvc.Logger.Debugw("GetNotebookNoteCounts")
	counts, err := a.notebookService.GetAllNotesCount(a.db)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotebookNoteCounts 失败", fastlog.Error(err))
		return nil, err
	}
	return counts, nil
}

// GetNoteIDsByNotebook 获取指定笔记本中所有未删除笔记的 ID 数组
func (a *App) GetNoteIDsByNotebook(notebookID uint) ([]uint, error) {
	a.LogSvc.Logger.Debugw("GetNoteIDsByNotebook", fastlog.Uint("notebookID", notebookID))
	ids, err := a.noteService.GetAllNoteIDsByNotebook(notebookID)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNoteIDsByNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
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

// SaveAllSettings 保存全部设置项，无预设时自动创建默认配置
func (a *App) SaveAllSettings(cfg services.SettingsConfig) error {
	a.LogSvc.Logger.Debugw("SaveAllSettings")
	if err := a.settingService.SaveAllSettings(cfg); err != nil {
		a.LogSvc.Logger.Errorw("SaveAllSettings 失败", fastlog.Error(err))
		return err
	}
	// 动态调整日志级别
	if a.LogSvc != nil {
		oldLevel := a.LogSvc.Logger.Level()
		newLevel := services.LevelFromInt(cfg.LogLevel)
		a.LogSvc.SetLevel(newLevel)
		if oldLevel != newLevel {
			a.LogSvc.Logger.Infow("日志级别已变更",
				fastlog.String("from", oldLevel.String()),
				fastlog.String("to", newLevel.String()),
			)
		}
	}
	// 无预设时自动创建"默认配置"
	profiles := a.profileService.ListProfiles()
	if len(profiles) == 0 && cfg.AIBaseURL != "" && cfg.AIAPIKey != "" {
		provider := cfg.AIProvider
		if provider == "" {
			provider = "openai"
		}
		profile := a.profileService.CreateProfile("默认配置", provider, cfg.AIBaseURL, cfg.AIAPIKey, true)
		if err := a.profileService.SetActive(profile.ID); err != nil {
			a.LogSvc.Logger.Errorw("激活默认配置失败", fastlog.Error(err))
		}
	}
	return nil
}

// GetAIRefMaxChars 获取 AI 引用笔记截断字数，空值时返回默认 10000
func (a *App) GetAIRefMaxChars() int {
	a.LogSvc.Logger.Debugw("GetAIRefMaxChars")
	val := a.settingService.Get("ai_ref_max_chars")
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 10000
	}
	if n > 100000 {
		return 100000
	}
	return n
}

// SetAIRefMaxChars 设置 AI 引用笔记截断字数，含范围校验（1-100000）
func (a *App) SetAIRefMaxChars(chars int) error {
	a.LogSvc.Logger.Debugw("SetAIRefMaxChars", fastlog.Int("chars", chars))
	if chars <= 0 {
		err := fmt.Errorf("截断字数必须大于 0")
		a.LogSvc.Logger.Errorw("SetAIRefMaxChars 失败", fastlog.Error(err))
		return err
	}
	if chars > 100000 {
		err := fmt.Errorf("截断字数不能超过 100000")
		a.LogSvc.Logger.Errorw("SetAIRefMaxChars 失败", fastlog.Error(err))
		return err
	}
	if err := a.settingService.Set("ai_ref_max_chars", strconv.Itoa(chars)); err != nil {
		a.LogSvc.Logger.Errorw("SetAIRefMaxChars 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetAIRefMaxChars 成功")
	return nil
}

// GetMaxFileSize 获取最大文件限制大小（字节），空值时返回默认 1MB
func (a *App) GetMaxFileSize() int64 {
	a.LogSvc.Logger.Debugw("GetMaxFileSize")
	val := a.settingService.Get("max_file_size")
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 1 * 1024 * 1024
	}
	if n > 100 {
		n = 100
	}
	return int64(n) * 1024 * 1024
}

// GetAISearchResultLimit 获取 AI 联网搜索结果数，空值时返回默认 5
func (a *App) GetAISearchResultLimit() int {
	a.LogSvc.Logger.Debugw("GetAISearchResultLimit")
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
	a.LogSvc.Logger.Debugw("SetAISearchResultLimit", fastlog.Int("limit", limit))
	if limit < 1 {
		err := fmt.Errorf("搜索结果数必须大于 0")
		a.LogSvc.Logger.Errorw("SetAISearchResultLimit 失败", fastlog.Error(err))
		return err
	}
	if limit > 30 {
		err := fmt.Errorf("搜索结果数不能超过 30")
		a.LogSvc.Logger.Errorw("SetAISearchResultLimit 失败", fastlog.Error(err))
		return err
	}
	if err := a.settingService.Set("ai_search_result_limit", strconv.Itoa(limit)); err != nil {
		a.LogSvc.Logger.Errorw("SetAISearchResultLimit 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetAISearchResultLimit 成功")
	return nil
}

// GetAICardRecallLimit 获取 AI 卡片召回条数，空值时返回默认 5
func (a *App) GetAICardRecallLimit() int {
	a.LogSvc.Logger.Debugw("GetAICardRecallLimit")
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
	a.LogSvc.Logger.Debugw("SetAICardRecallLimit", fastlog.Int("limit", limit))
	if limit < 1 {
		err := fmt.Errorf("卡片召回条数必须大于 0")
		a.LogSvc.Logger.Errorw("SetAICardRecallLimit 失败", fastlog.Error(err))
		return err
	}
	if limit > 30 {
		err := fmt.Errorf("卡片召回条数不能超过 30")
		a.LogSvc.Logger.Errorw("SetAICardRecallLimit 失败", fastlog.Error(err))
		return err
	}
	if err := a.settingService.Set("ai_card_recall_limit", strconv.Itoa(limit)); err != nil {
		a.LogSvc.Logger.Errorw("SetAICardRecallLimit 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetAICardRecallLimit 成功")
	return nil
}

// ==================== AI 相关绑定方法 ====================

// GetAIConfig 获取 AI 服务配置
func (a *App) GetAIConfig() services.AIConfig {
	a.LogSvc.Logger.Debugw("GetAIConfig")
	return a.aiService.GetConfig()
}

// SaveAIConfig 保存 AI 服务配置，无预设时自动创建默认配置
func (a *App) SaveAIConfig(cfg services.AIConfig) error {
	a.LogSvc.Logger.Debugw("SaveAIConfig")
	if err := a.aiService.SaveConfig(cfg); err != nil {
		a.LogSvc.Logger.Errorw("SaveAIConfig 失败", fastlog.Error(err))
		return err
	}
	// 无预设时自动创建"默认配置"
	profiles := a.profileService.ListProfiles()
	if len(profiles) == 0 {
		profile := a.profileService.CreateProfile("默认配置", cfg.Provider, cfg.BaseURL, cfg.APIKey, true)
		if err := a.profileService.SetActive(profile.ID); err != nil {
			a.LogSvc.Logger.Errorw("激活默认配置失败", fastlog.Error(err))
		}
	}
	a.LogSvc.Logger.Infow("SaveAIConfig 成功")
	return nil
}

// ==================== API 配置预设绑定 ====================

// GetProfiles 获取所有 API 配置预设
func (a *App) GetProfiles() []models.APIProfile {
	a.LogSvc.Logger.Debugw("GetProfiles")
	return a.profileService.ListProfiles()
}

// CreateProfile 创建 API 配置预设
func (a *App) CreateProfile(name, provider, baseURL, apiKey string) models.APIProfile {
	a.LogSvc.Logger.Debugw("CreateProfile", fastlog.String("name", name), fastlog.String("provider", provider), fastlog.String("key", "***"))
	return a.profileService.CreateProfile(name, provider, baseURL, apiKey)
}

// UpdateProfile 更新 API 配置预设
func (a *App) UpdateProfile(id uint, name, provider, baseURL, apiKey string) error {
	a.LogSvc.Logger.Debugw("UpdateProfile", fastlog.Uint("id", id), fastlog.String("name", name), fastlog.String("provider", provider), fastlog.String("key", "***"))
	if err := a.profileService.UpdateProfile(id, name, provider, baseURL, apiKey); err != nil {
		a.LogSvc.Logger.Errorw("UpdateProfile 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("UpdateProfile 成功", fastlog.Uint("id", id))
	return nil
}

// DeleteProfile 删除 API 配置预设
func (a *App) DeleteProfile(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteProfile", fastlog.Uint("id", id))
	if err := a.profileService.DeleteProfile(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteProfile 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteProfile 成功", fastlog.Uint("id", id))
	return nil
}

// SwitchProfile 切换 API 配置预设
func (a *App) SwitchProfile(id uint) error {
	a.LogSvc.Logger.Debugw("SwitchProfile", fastlog.Uint("id", id))
	if err := a.profileService.SwitchProfile(id); err != nil {
		a.LogSvc.Logger.Errorw("SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SwitchProfile 成功", fastlog.Uint("id", id))
	return nil
}

// TestAIBaseURL 测试 AI Base URL 连通性
func (a *App) TestAIBaseURL(baseURL, apiKey string) (bool, error) {
	a.LogSvc.Logger.Debugw("TestAIBaseURL", fastlog.String("baseURL", baseURL), fastlog.String("key", "***"))
	cfg := a.aiService.GetConfig()
	cfg.BaseURL = baseURL
	cfg.APIKey = apiKey
	result, err := a.aiService.TestConnection(cfg)
	if err != nil {
		a.LogSvc.Logger.Errorw("TestAIBaseURL 失败", fastlog.Error(err))
		return false, err
	}
	a.LogSvc.Logger.Infow("TestAIBaseURL 成功")
	return result, nil
}

// TestTavilyConnection 测试 Tavily API Key 是否有效
func (a *App) TestTavilyConnection(apiKey string) (bool, error) {
	a.LogSvc.Logger.Debugw("TestTavilyConnection", fastlog.String("key", "***"))
	if apiKey == "" {
		err := fmt.Errorf("API Key 不能为空")
		a.LogSvc.Logger.Errorw("TestTavilyConnection 失败", fastlog.Error(err))
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := services.SearchWeb(ctx, "test", apiKey, 1)
	if err != nil {
		a.LogSvc.Logger.Errorw("TestTavilyConnection 失败", fastlog.Error(err))
		return false, fmt.Errorf("连接失败: %v", err)
	}
	if result == nil {
		err := fmt.Errorf("连接失败，请检查 API Key 是否正确")
		a.LogSvc.Logger.Errorw("TestTavilyConnection 失败", fastlog.Error(err))
		return false, err
	}
	a.LogSvc.Logger.Infow("TestTavilyConnection 成功")
	return true, nil
}

// TestZhihuConnection 测试知乎 Access Secret 是否有效
func (a *App) TestZhihuConnection(accessSecret string) (bool, error) {
	a.LogSvc.Logger.Debugw("TestZhihuConnection", fastlog.String("key", "***"))
	if accessSecret == "" {
		err := fmt.Errorf("access Secret 不能为空")
		a.LogSvc.Logger.Errorw("TestZhihuConnection 失败", fastlog.Error(err))
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := services.SearchZhihuContent(ctx, "test", accessSecret, 1)
	if err != nil {
		a.LogSvc.Logger.Errorw("TestZhihuConnection 失败", fastlog.Error(err))
		return false, fmt.Errorf("连接失败: %v", err)
	}
	if result == nil {
		err := fmt.Errorf("连接失败，请检查 Access Secret 是否正确")
		a.LogSvc.Logger.Errorw("TestZhihuConnection 失败", fastlog.Error(err))
		return false, err
	}
	a.LogSvc.Logger.Infow("TestZhihuConnection 成功")
	return true, nil
}

// FetchAIModels 获取可用模型列表
func (a *App) FetchAIModels(baseURL, apiKey string) ([]string, error) {
	a.LogSvc.Logger.Debugw("FetchAIModels", fastlog.String("baseURL", baseURL), fastlog.String("key", "***"))
	cfg := a.aiService.GetConfig()
	cfg.BaseURL = baseURL
	cfg.APIKey = apiKey
	models, err := a.aiService.FetchModels(cfg)
	if err != nil {
		a.LogSvc.Logger.Errorw("FetchAIModels 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("FetchAIModels 成功", fastlog.Int("count", len(models)))
	return models, nil
}

// SelectAIChatFiles 打开文件对话框选择文本文件，校验并读取内容返回给 AI 聊天使用
func (a *App) SelectAIChatFiles() ([]AIChatFileResult, error) {
	a.LogSvc.Logger.Debugw("SelectAIChatFiles")
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           "选择要上传的文本文件",
		ShowHiddenFiles: false,
	})
	if err != nil {
		a.LogSvc.Logger.Errorw("SelectAIChatFiles 失败", fastlog.Error(err))
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if len(paths) == 0 {
		return []AIChatFileResult{}, nil // 用户取消
	}

	return a.readAIChatFiles(paths), nil
}

// ReadAIChatFiles 直接根据文件路径列表校验并读取内容（拖拽上传用）
func (a *App) ReadAIChatFiles(paths []string) []AIChatFileResult {
	a.LogSvc.Logger.Debugw("ReadAIChatFiles", fastlog.Int("file_count", len(paths)))
	return a.readAIChatFiles(paths)
}

// readAIChatFiles 内部方法：校验、读取、截断一组文件（按钮上传和拖拽上传共用）
func (a *App) readAIChatFiles(paths []string) []AIChatFileResult {
	a.LogSvc.Logger.Debugw("readAIChatFiles", fastlog.Int("file_count", len(paths)))
	maxSize := a.GetMaxFileSize()

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
			maxSizeMB := maxSize / (1024 * 1024)
			result.Error = fmt.Sprintf("文件过大（超过 %dMB），请选择小于 %dMB 的文件", maxSizeMB, maxSizeMB)
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

	return results
}

// CallAI 调用 AI 对话接口
func (a *App) CallAI(messages []services.Message) (string, error) {
	return a.aiService.CallAI(messages)
}

// CallAIStream 流式调用 AI 对话接口（通过 EventsEmit 推送逐块内容）
func (a *App) CallAIStream(streamGen int, messages []services.Message, thinkingEnabled bool, searchSources []string, cardRecallEnabled bool, sessionID uint, isRegenerate bool, skillIds []string, referencedNoteIDs []uint, roleplayNoteIDs []uint, followUpRefContent string, uploadedFiles []AIChatFileResult) {
	ctx, cancel := context.WithCancel(context.Background())
	a.aiStreamCancel = cancel

	var fullThinking strings.Builder
	var searchSourcesJSON, recallCardsJSON string

	// 在主 goroutine 发射搜索状态事件，确保前端能立即收到
	var searching bool
	if len(searchSources) > 0 {
		searching = true
		runtime.EventsEmit(a.ctx, "ai:search-status", "refining")
		a.LogSvc.Logger.Infow("AI 联网搜索启动", fastlog.Int("source_count", len(searchSources)))
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

		// ── 步骤 2: 角色扮演上下文注入 ──
		hasRoleplay := false
		for _, sid := range skillIds {
			if sid == "skill_roleplay" {
				hasRoleplay = true
				break
			}
		}
		var roleplayContext string
		if hasRoleplay && len(roleplayNoteIDs) > 0 {
			refCtx, err := a.noteService.BuildNoteRefContext(roleplayNoteIDs)
			if err == nil && refCtx != nil && refCtx.Context != "" {
				roleplayContext = refCtx.Context
				roleplayText := "以下是用户提供的人物设定笔记内容：\n\n" + refCtx.Context
				found := false
				for i := range messages {
					if messages[i].Role == "system" {
						messages[i].Content = messages[i].Content + "\n\n" + roleplayText
						found = true
						break
					}
				}
				if !found {
					messages = append([]services.Message{{Role: "system", Content: roleplayText}}, messages...)
				}
			}
		}

		// ── 步骤 3: 笔记引用上下文注入 ──
		if len(referencedNoteIDs) > 0 {
			refCtx, err := a.noteService.BuildNoteRefContext(referencedNoteIDs)
			if err == nil && refCtx != nil && refCtx.Context != "" {
				found := false
				for i := range messages {
					if messages[i].Role == "system" {
						messages[i].Content = messages[i].Content + "\n\n" + refCtx.Context
						found = true
						break
					}
				}
				if !found {
					messages = append([]services.Message{{Role: "system", Content: refCtx.Context}}, messages...)
				}
			}
		}

		// ── 步骤 4: 追问引用内容注入 ──
		if followUpRefContent != "" {
			refText := "用户正在追问以下内容：\n" + followUpRefContent
			if len([]rune(followUpRefContent)) > 500 {
				refText = "用户正在追问以下内容：\n" + string([]rune(followUpRefContent)[:500])
			}
			found := false
			for i := range messages {
				if messages[i].Role == "system" {
					messages[i].Content = messages[i].Content + "\n\n" + refText
					found = true
					break
				}
			}
			if !found {
				messages = append([]services.Message{{Role: "system", Content: refText}}, messages...)
			}
		}

		// ── 步骤 5: 上传文件内容注入 ──
		if len(uploadedFiles) > 0 {
			var b strings.Builder
			b.WriteString("用户上传了以下文件内容，请基于这些内容回答用户的提问：\n")
			for _, f := range uploadedFiles {
				if f.Error != "" || f.Content == "" {
					continue
				}
				sizeStr := formatFileSize(f.Size)
				fmt.Fprintf(&b, "\n--- 文件: %s (%s) ---\n%s\n---", f.Name, sizeStr, f.Content)
			}
			if b.Len() > 0 {
				found := false
				for i := range messages {
					if messages[i].Role == "system" {
						messages[i].Content = messages[i].Content + "\n\n" + b.String()
						found = true
						break
					}
				}
				if !found {
					messages = append([]services.Message{{Role: "system", Content: b.String()}}, messages...)
				}
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
					var aiErr *aicli.AIErrorWrapper
					if errors.As(err, &aiErr) {
						runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, aiErr.Err.ToJSON())
					} else {
						ae := aicli.NewAIError(aicli.CategoryUnknown, "搜索关键词精炼失败: "+err.Error())
						runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, ae.ToJSON())
					}
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
			a.LogSvc.Logger.Debugw("AI 搜索结果汇总", fastlog.Int("sources_count", len(searchSources)))
		}

		// 通知前端搜索完成，关闭搜索动画
		if searching {
			runtime.EventsEmit(a.ctx, "ai:search-status", "done")
		}

		// 卡片召回（在联网搜索之后执行）
		if cardRecallEnabled {
			a.LogSvc.Logger.Infow("AI 卡片召回启动")
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
				// 读取引用截断阈值（默认 10000）
				maxChars := 10000
				if a.settingService != nil {
					if val := a.settingService.Get("ai_ref_max_chars"); val != "" {
						if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 100000 {
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
						a.LogSvc.Logger.Debugw("AI 卡片召回结果", fastlog.Int("cards_count", len(recallResult.Cards)))
					} else {
						a.LogSvc.Logger.Debugw("AI 卡片召回无结果")
					}
				}
			}
		}

		// 技能提示词注入（在搜索和卡片召回之后执行）
		if len(skillIds) > 0 {
			a.LogSvc.Logger.Infow("AI 技能注入启动", fastlog.Int("skill_count", len(skillIds)))
			skillPrompt, err := a.aiService.GetSkillPrompts(skillIds)
			if err == nil && skillPrompt != "" {
				// 替换角色扮演占位符
				if hasRoleplay && roleplayContext != "" {
					skillPrompt = strings.ReplaceAll(skillPrompt, "{roleplay_context}", roleplayContext)
				}
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
				a.LogSvc.Logger.Errorw("获取技能提示词失败", fastlog.Error(err))
			}
		}

		// 如果已被用户取消（停止按钮），不再继续调用 LLM，避免白调用
		if ctx.Err() != nil {
			runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, "", 0.0, 0.0, 0, 0, 0)
			return
		}

		a.LogSvc.Logger.Debugw("AI 流开始",
			fastlog.Int("message_count", len(messages)),
		)
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
				a.LogSvc.Logger.Infow("AI 流完成",
					fastlog.Int("total_tokens", totalTokens),
					fastlog.Float64("elapsed_total", elapsedTotal),
				)
				runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, content, elapsedThinking, elapsedTotal, totalTokens, userTokens, assistantTokens)
				if thinkingEnabled && fullThinking.Len() > 0 {
					runtime.EventsEmit(a.ctx, "ai:stream-thinking-done", fullThinking.String())
				}
			},
			func(err string) {
				a.LogSvc.Logger.Errorw("AI 流错误",
					fastlog.String("error", err),
				)
				runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, err)
			},
		)
	}()
}

// formatFileSize 将字节数转为人类可读的文件大小字符串
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
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
	a.LogSvc.Logger.Debugw("CancelAIStream")
	if a.aiStreamCancel != nil {
		a.aiStreamCancel()
		a.aiStreamCancel = nil
	}
}

// GetAISessions 获取 AI 会话列表
func (a *App) GetAISessions() []services.AISessionSummary {
	a.LogSvc.Logger.Debugw("GetAISessions")
	return a.aiService.GetAISessions()
}

// TogglePinAISession 切换会话置顶状态
func (a *App) TogglePinAISession(id uint) error {
	a.LogSvc.Logger.Debugw("TogglePinAISession", fastlog.Uint("id", id))
	if err := a.aiService.TogglePinAISession(id); err != nil {
		a.LogSvc.Logger.Errorw("TogglePinAISession 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("TogglePinAISession 成功", fastlog.Uint("id", id))
	return nil
}

// CreateAISession 创建新 AI 会话，返回会话 ID
func (a *App) CreateAISession() uint {
	a.LogSvc.Logger.Debugw("CreateAISession")
	id := a.aiService.CreateAISession()
	a.LogSvc.Logger.Infow("CreateAISession 成功", fastlog.Uint("id", id))
	return id
}

// DeleteAISession 删除 AI 会话及所有消息
func (a *App) DeleteAISession(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteAISession", fastlog.Uint("id", id))
	if err := a.aiService.DeleteAISession(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteAISession 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteAISession 成功", fastlog.Uint("id", id))
	return nil
}

// RenameAISession 重命名 AI 会话
func (a *App) RenameAISession(id uint, title string) error {
	a.LogSvc.Logger.Debugw("RenameAISession", fastlog.Uint("id", id))
	if err := a.aiService.RenameAISession(id, title); err != nil {
		a.LogSvc.Logger.Errorw("RenameAISession 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RenameAISession 成功", fastlog.Uint("id", id))
	return nil
}

// LoadAISessionMessages 加载 AI 会话的所有消息
func (a *App) LoadAISessionMessages(id uint) []services.Message {
	a.LogSvc.Logger.Debugw("LoadAISessionMessages", fastlog.Uint("id", id))
	return a.aiService.LoadAISessionMessages(id)
}

// SaveAIMessages 保存一轮 AI 对话消息到指定会话
func (a *App) SaveAIMessages(sessionID uint, messages []services.Message) error {
	a.LogSvc.Logger.Debugw("SaveAIMessages", fastlog.Uint("sessionID", sessionID), fastlog.Int("message_count", len(messages)))
	if err := a.aiService.SaveAIMessages(sessionID, messages); err != nil {
		a.LogSvc.Logger.Errorw("SaveAIMessages 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SaveAIMessages 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// ClearAISessionMessages 清空 AI 会话的所有消息（不删会话）
func (a *App) ClearAISessionMessages(sessionID uint) error {
	a.LogSvc.Logger.Debugw("ClearAISessionMessages", fastlog.Uint("sessionID", sessionID))
	if err := a.aiService.ClearAISessionMessages(sessionID); err != nil {
		a.LogSvc.Logger.Errorw("ClearAISessionMessages 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("ClearAISessionMessages 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// UpdateAIMessageContent 更新指定 AI 消息的内容
func (a *App) UpdateAIMessageContent(id uint, content string) error {
	a.LogSvc.Logger.Debugw("UpdateAIMessageContent", fastlog.Uint("id", id))
	if err := a.aiService.UpdateAIMessageContent(id, content); err != nil {
		a.LogSvc.Logger.Errorw("UpdateAIMessageContent 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("UpdateAIMessageContent 成功", fastlog.Uint("id", id))
	return nil
}

// DeleteAIMessage 按 ID 删除单条 AI 消息
func (a *App) DeleteAIMessage(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteAIMessage", fastlog.Uint("id", id))
	if err := a.aiService.DeleteAIMessage(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteAIMessage 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteAIMessage 成功", fastlog.Uint("id", id))
	return nil
}

// DeleteAIMessagesAfter 删除指定会话中在指定消息之后的所有消息
func (a *App) DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error) {
	a.LogSvc.Logger.Debugw("DeleteAIMessagesAfter", fastlog.Uint("sessionID", sessionID), fastlog.Uint("messageID", messageID))
	count, err := a.aiService.DeleteAIMessagesAfter(sessionID, messageID)
	if err != nil {
		a.LogSvc.Logger.Errorw("DeleteAIMessagesAfter 失败", fastlog.Error(err))
		return 0, err
	}
	a.LogSvc.Logger.Infow("DeleteAIMessagesAfter 成功", fastlog.Int64("count", count))
	return count, nil
}

// UpdateSessionContextTokens 更新会话的上下文 Token 数
func (a *App) UpdateSessionContextTokens(sessionID uint, tokens int) error {
	a.LogSvc.Logger.Debugw("UpdateSessionContextTokens", fastlog.Uint("sessionID", sessionID), fastlog.Int("tokens", tokens))
	if err := a.aiService.UpdateSessionContextTokens(sessionID, tokens); err != nil {
		a.LogSvc.Logger.Errorw("UpdateSessionContextTokens 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// SaveSessionConfig 保存会话操作栏配置
func (a *App) SaveSessionConfig(sessionID uint, cfg services.SessionConfig) error {
	a.LogSvc.Logger.Debugw("SaveSessionConfig", fastlog.Uint("sessionID", sessionID))
	if err := a.aiService.SaveSessionConfig(sessionID, cfg); err != nil {
		a.LogSvc.Logger.Errorw("SaveSessionConfig 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SaveSessionConfig 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// LoadSessionConfig 加载会话操作栏配置
func (a *App) LoadSessionConfig(sessionID uint) services.SessionConfig {
	a.LogSvc.Logger.Debugw("LoadSessionConfig", fastlog.Uint("sessionID", sessionID))
	return a.aiService.LoadSessionConfig(sessionID)
}

// ClearAllAISessions 清空所有 AI 会话及消息
func (a *App) ClearAllAISessions() error {
	a.LogSvc.Logger.Debugw("ClearAllAISessions")
	if err := a.aiService.ClearAllAISessions(); err != nil {
		a.LogSvc.Logger.Errorw("ClearAllAISessions 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("ClearAllAISessions 成功")
	return nil
}

// UpdateLastUserMessageTokens 更新指定会话中最后一条用户消息的 tokens
func (a *App) UpdateLastUserMessageTokens(sessionID uint, tokens int) error {
	a.LogSvc.Logger.Debugw("UpdateLastUserMessageTokens", fastlog.Uint("sessionID", sessionID), fastlog.Int("tokens", tokens))
	if err := a.aiService.UpdateLastUserMessageTokens(sessionID, tokens); err != nil {
		a.LogSvc.Logger.Errorw("UpdateLastUserMessageTokens 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// SaveAIMessageAsNote 将 AI 消息内容保存为笔记（归入默认笔记本）
func (a *App) SaveAIMessageAsNote(content string) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("SaveAIMessageAsNote")
	if strings.TrimSpace(content) == "" {
		err := fmt.Errorf("内容不能为空")
		a.LogSvc.Logger.Errorw("SaveAIMessageAsNote 失败", fastlog.Error(err))
		return nil, err
	}
	// 自动生成标题：取第一行，截断到 50 字符
	title := generateNoteTitle(content)
	// 保存到默认笔记本（id=1）
	note, err := a.noteService.CreateWithNotebook(title, content, ".md", 1)
	if err != nil {
		a.LogSvc.Logger.Errorw("SaveAIMessageAsNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("SaveAIMessageAsNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
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
	a.LogSvc.Logger.Debugw("GetSortOrder")
	return a.settingService.Get("sort_order")
}

// SetSortOrder 保存排序方式设置
func (a *App) SetSortOrder(order string) error {
	a.LogSvc.Logger.Debugw("SetSortOrder", fastlog.String("order", order))
	if err := a.settingService.Set("sort_order", order); err != nil {
		a.LogSvc.Logger.Errorw("SetSortOrder 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetSortOrder 成功")
	return nil
}

// GetPageSize 获取分页大小设置
func (a *App) GetPageSize() int {
	a.LogSvc.Logger.Debugw("GetPageSize")
	size := a.settingService.Get("page_size")
	n, err := strconv.Atoi(size)
	if err != nil || n < 20 || n > 100 {
		return 20
	}
	return n
}

// SetPageSize 保存分页大小设置
func (a *App) SetPageSize(size int) error {
	a.LogSvc.Logger.Debugw("SetPageSize", fastlog.Int("size", size))
	if err := a.settingService.Set("page_size", strconv.Itoa(size)); err != nil {
		a.LogSvc.Logger.Errorw("SetPageSize 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetPageSize 成功")
	return nil
}

// ==================== 版本与链接绑定方法 ====================

// GetVersion 获取应用版本号
func (a *App) GetVersion() string {
	a.LogSvc.Logger.Debugw("GetVersion")
	return verman.V.GitVersion
}

// ExportNoteAsMarkdown 导出单条笔记为 Markdown 文件，弹出保存对话框让用户选择路径
func (a *App) ExportNoteAsMarkdown(id uint) (string, error) {
	a.LogSvc.Logger.Debugw("ExportNoteAsMarkdown", fastlog.Uint("id", id))
	note, err := a.noteService.GetByID(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("ExportNoteAsMarkdown 失败", fastlog.Error(err))
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
		a.LogSvc.Logger.Errorw("ExportNoteAsMarkdown 失败", fastlog.Error(err))
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	a.LogSvc.Logger.Infow("ExportNoteAsMarkdown 成功")
	return "导出成功：" + filePath, nil
}

// ExportAISessionAsMarkdown 导出 AI 对话为 Markdown 文件
func (a *App) ExportAISessionAsMarkdown(sessionID uint) (string, error) {
	a.LogSvc.Logger.Debugw("ExportAISessionAsMarkdown", fastlog.Uint("sessionID", sessionID))
	// 获取会话标题
	var session models.AISession
	if err := a.db.First(&session, sessionID).Error; err != nil {
		a.LogSvc.Logger.Errorw("ExportAISessionAsMarkdown 失败", fastlog.Error(err))
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
		a.LogSvc.Logger.Errorw("ExportAISessionAsMarkdown 失败", fastlog.Error(err))
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	a.LogSvc.Logger.Infow("ExportAISessionAsMarkdown 成功")
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
	a.LogSvc.Logger.Debugw("OpenDataDir")
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		a.LogSvc.Logger.Errorw("OpenDataDir 失败", fastlog.Error(err))
		return err
	}
	dir := filepath.Dir(dbPath)
	cmd := exec.Command("explorer", dir)
	if err := cmd.Start(); err != nil {
		a.LogSvc.Logger.Errorw("OpenDataDir 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("OpenDataDir 成功")
	return nil
}

// OpenLogDir 在文件管理器中打开日志目录
func (a *App) OpenLogDir() error {
	a.LogSvc.Logger.Debugw("OpenLogDir")
	logDir := a.LogSvc.LogDir()
	if logDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			a.LogSvc.Logger.Errorw("OpenLogDir 失败", fastlog.Error(err))
			return err
		}
		logDir = filepath.Join(homeDir, ".jot", "logs")
	}
	cmd := exec.Command("explorer", logDir)
	if err := cmd.Start(); err != nil {
		a.LogSvc.Logger.Errorw("OpenLogDir 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("OpenLogDir 成功")
	return nil
}

// OpenProjectURL 在默认浏览器中打开项目地址
func (a *App) OpenProjectURL(url string) string {
	a.LogSvc.Logger.Debugw("OpenProjectURL", fastlog.String("url", url))
	runtime.BrowserOpenURL(a.ctx, url)
	return "已打开浏览器"
}

// exportSnapshot 统一导出：VACUUM INTO → ZIP 打包 {jot-backup.db, images/} → 清理
func (a *App) exportSnapshot(destZipPath string) error {
	a.LogSvc.Logger.Debugw("exportSnapshot", fastlog.String("dest", destZipPath))
	// 1. VACUUM INTO 临时 .db
	tempDB := destZipPath + ".tmpdb"
	defer func() { _ = os.Remove(tempDB) }()
	if err := a.noteService.ExportBackup(tempDB); err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("VACUUM INTO 失败: %w", err)
	}

	// 2. 获取图片目录
	imgDir, err := a.imageDirPath()
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return err
	}

	// 3. 创建 ZIP 文件
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("创建 ZIP 失败: %w", err)
	}
	defer func() { _ = zipFile.Close() }()

	zw := zip.NewWriter(zipFile)
	defer func() { _ = zw.Close() }()

	// 3a. 添加 db 文件（不压缩，SQLite 已是压缩状态）
	dbFile, err := os.Open(tempDB)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("打开临时 db 失败: %w", err)
	}
	defer func() { _ = dbFile.Close() }()

	dbInfo, _ := dbFile.Stat()
	dbHeader, err := zip.FileInfoHeader(dbInfo)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("创建 db header 失败: %w", err)
	}
	dbHeader.Name = "jot-backup.db"
	dbHeader.Method = zip.Store
	dbWriter, err := zw.CreateHeader(dbHeader)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("创建 db ZIP entry 失败: %w", err)
	}
	if _, err := io.Copy(dbWriter, dbFile); err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("写入 db 到 ZIP 失败: %w", err)
	}

	// 3b. 添加 images/ 目录中的文件
	entries, err := os.ReadDir(imgDir)
	if err != nil && !os.IsNotExist(err) {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
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

	a.LogSvc.Logger.Infow("exportSnapshot 成功")
	return nil
}

// replaceDatabase 统一替换：备份当前 db → 关闭连接 → 替换 db + images → 重连 → 重建服务
func (a *App) replaceDatabase(srcDBPath, srcImagesDir string) error {
	a.LogSvc.Logger.Debugw("replaceDatabase", fastlog.String("srcDB", srcDBPath))
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("获取数据库路径失败: %w", err)
	}

	// Step 1: 备份当前数据库
	backupPath := dbPath + ".bak"
	if err := fs.CopyEx(dbPath, backupPath, true); err != nil {
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
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
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	_ = sqlDB.Close()

	// Step 3: 复制 db 文件
	if err := fs.CopyEx(srcDBPath, dbPath, true); err != nil {
		rollback()
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
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
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("数据库重连失败: %w", err)
	}

	// Step 6: 重建服务
	a.db = newDB
	a.rebuildServices(newDB)

	// Step 7: 清理备份
	_ = os.Remove(backupPath)

	a.LogSvc.Logger.Infow("replaceDatabase 成功")
	return nil
}

// importFromArchive 统一导入：解压 ZIP → 提取 db + images → replaceDatabase
func (a *App) importFromArchive(srcZipPath string) error {
	a.LogSvc.Logger.Debugw("importFromArchive", fastlog.String("src", srcZipPath))
	// 解压到临时目录
	tmpDir := filepath.Join(os.TempDir(), "jot-restore-"+fmt.Sprintf("%x", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		a.LogSvc.Logger.Errorw("importFromArchive 失败", fastlog.Error(err))
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	reader, err := zip.OpenReader(srcZipPath)
	if err != nil {
		a.LogSvc.Logger.Errorw("importFromArchive 失败", fastlog.Error(err))
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
		err := fmt.Errorf("ZIP 文件中未找到 jot-backup.db")
		a.LogSvc.Logger.Errorw("importFromArchive 失败", fastlog.Error(err))
		return err
	}

	a.LogSvc.Logger.Infow("importFromArchive 成功")
	return a.replaceDatabase(dbSrc, imagesDir)
}

// ==================== 一键备份与还原绑定方法 ====================

// BackupToDir 一键备份到 ~/.jot/backup/ 目录，固定文件名 jot-backup.zip（覆盖旧备份）
func (a *App) BackupToDir() (string, error) {
	a.LogSvc.Logger.Debugw("BackupToDir")
	backupDir, err := database.BackupDir()
	if err != nil {
		a.LogSvc.Logger.Errorw("BackupToDir 失败", fastlog.Error(err))
		return "", fmt.Errorf("获取备份目录失败: %w", err)
	}
	if err := database.EnsureBackupDir(); err != nil {
		a.LogSvc.Logger.Errorw("BackupToDir 失败", fastlog.Error(err))
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	zipPath := filepath.Join(backupDir, "jot-backup.zip")

	// 先删除旧备份
	_ = os.Remove(zipPath)

	if err := a.exportSnapshot(zipPath); err != nil {
		a.LogSvc.Logger.Errorw("BackupToDir 失败", fastlog.Error(err))
		return "", fmt.Errorf("备份失败: %w", err)
	}

	a.LogSvc.Logger.Infow("备份成功")
	return "备份成功：jot-backup.zip", nil
}

// RestoreFromDir 从 backup 目录的 jot-backup.zip 还原备份（含图片）
func (a *App) RestoreFromDir() (*services.ImportResult, error) {
	a.LogSvc.Logger.Debugw("RestoreFromDir")
	backupDir, err := database.BackupDir()
	if err != nil {
		a.LogSvc.Logger.Errorw("RestoreFromDir 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "获取备份目录失败：" + err.Error()}, nil
	}

	zipPath := filepath.Join(backupDir, "jot-backup.zip")

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return &services.ImportResult{Message: "暂无可用备份"}, nil
	} else if err != nil {
		a.LogSvc.Logger.Errorw("RestoreFromDir 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "读取备份文件失败：" + err.Error()}, nil
	}

	if err := a.importFromArchive(zipPath); err != nil {
		a.LogSvc.Logger.Errorw("RestoreFromDir 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "还原失败：" + err.Error()}, nil
	}

	a.LogSvc.Logger.Infow("还原成功")
	return &services.ImportResult{
		Message:      "已从备份文件恢复：jot-backup.zip",
		SuccessCount: 1,
	}, nil
}

// GetBackupInfo 获取备份文件信息（文件名、修改时间、文件大小），无备份时返回空值
func (a *App) GetBackupInfo() (map[string]string, error) {
	a.LogSvc.Logger.Debugw("GetBackupInfo")
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
	a.LogSvc.Logger.Debugw("ImportFiles", fastlog.Int("file_count", len(paths)), fastlog.Uint("notebookID", notebookID))
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		return []FileImportResult{{
			Path:    "",
			Success: false,
			Error:   "获取默认笔记本失败: " + err.Error(),
		}}
	}

	maxSize := a.GetMaxFileSize()
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
			maxSizeMB := maxSize / (1024 * 1024)
			result.Error = fmt.Sprintf("文件过大（超过 %dMB），无法导入", maxSizeMB)
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
	a.LogSvc.Logger.Debugw("ResetDatabase")
	// 1. 删除所有表（自动处理外键依赖顺序）
	tables := []interface{}{
		&models.AIMessage{},
		&models.AISessionConfig{},
		&models.AISession{},
		&models.AIPrompt{},
		&models.APIProfile{},
		&models.Todo{},
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
		&models.Notebook{}, &models.AISession{}, &models.AIMessage{}, &models.AISessionConfig{},
		&models.APIProfile{}, &models.AIPrompt{}, &models.Todo{}); err != nil {
		return err
	}

	// 3. 重新初始化内置技能提示词
	if err := database.InitBuiltinPrompts(a.db); err != nil {
		return fmt.Errorf("初始化内置提示词失败: %w", err)
	}

	// 4. 重新初始化默认标签
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

	a.LogSvc.Logger.Infow("数据库重置完成")
	return nil
}

// rebuildServices 使用新的数据库连接重建所有服务实例
func (a *App) rebuildServices(db *gorm.DB) {
	a.LogSvc.Logger.Debugw("rebuildServices")
	a.settingService = services.NewSettingService(db)
	a.noteService = services.NewNoteService(db, a.settingService, a.LogSvc.Logger)
	a.tagService = services.NewTagService(db, a.LogSvc.Logger)
	a.notebookService = services.NewNotebookService(db, a.LogSvc.Logger)
	a.aiService = services.NewAIService(db, a.LogSvc.Logger)
	a.profileService = services.NewProfileService(db, a.LogSvc.Logger)
	a.todoService = services.NewTodoService(db, a.LogSvc.Logger)
	// 重建日志服务
	a.LogSvc = services.NewLogService()
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".jot", "logs")
	logLevelStr := a.settingService.Get("log_level")
	logLevelVal := 1
	if n, err := strconv.Atoi(logLevelStr); err == nil {
		logLevelVal = n
	}
	logLevel := services.LevelFromInt(logLevelVal)
	if err := a.LogSvc.Init(logDir, logLevel); err != nil {
		a.LogSvc.Logger.Errorw("日志重新初始化失败", fastlog.Error(err))
	}
	a.LogSvc.Logger.Infow("rebuildServices 成功")
}

// ==================== Todo 相关绑定方法 ====================

func (a *App) CreateTodo(text string) (*models.Todo, error) {
	a.LogSvc.Logger.Debugw("CreateTodo")
	todo, err := a.todoService.Create(text)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateTodo 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateTodo 成功", fastlog.Uint("id", todo.ID))
	return todo, nil
}

func (a *App) ListTodos() ([]models.Todo, error) {
	a.LogSvc.Logger.Debugw("ListTodos")
	todos, err := a.todoService.List()
	if err != nil {
		a.LogSvc.Logger.Errorw("ListTodos 失败", fastlog.Error(err))
		return nil, err
	}
	return todos, nil
}

func (a *App) ToggleTodo(id uint) (*models.Todo, error) {
	a.LogSvc.Logger.Debugw("ToggleTodo", fastlog.Uint("id", id))
	todo, err := a.todoService.Toggle(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("ToggleTodo 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("ToggleTodo 成功", fastlog.Uint("id", id))
	return todo, nil
}

func (a *App) DeleteTodo(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteTodo", fastlog.Uint("id", id))
	if err := a.todoService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteTodo 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteTodo 成功", fastlog.Uint("id", id))
	return nil
}

func (a *App) UpdateTodo(id uint, text string) (*models.Todo, error) {
	a.LogSvc.Logger.Debugw("UpdateTodo", fastlog.Uint("id", id))
	todo, err := a.todoService.Update(id, text)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateTodo 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateTodo 成功", fastlog.Uint("id", id))
	return todo, nil
}

func (a *App) ClearCompletedTodos() (string, error) {
	a.LogSvc.Logger.Debugw("ClearCompletedTodos")
	count, err := a.todoService.DeleteCompleted()
	if err != nil {
		a.LogSvc.Logger.Errorw("ClearCompletedTodos 失败", fastlog.Error(err))
		return "", err
	}
	a.LogSvc.Logger.Infow("ClearCompletedTodos 成功", fastlog.Int64("count", count))
	return fmt.Sprintf("已清空 %d 个已完成待办事项", count), nil
}

// reconnectDB 重新连接数据库（用于导入失败后的恢复）
func (a *App) reconnectDB(dbPath string) error {
	a.LogSvc.Logger.Debugw("reconnectDB", fastlog.String("dbPath", dbPath))
	// 关闭旧连接
	if sqlDB, err := a.db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		a.LogSvc.Logger.Errorw("reconnectDB 失败", fastlog.Error(err))
		return fmt.Errorf("数据库重连失败: %w", err)
	}
	a.db = db
	a.rebuildServices(db)
	a.LogSvc.Logger.Infow("reconnectDB 成功")
	return nil
}
