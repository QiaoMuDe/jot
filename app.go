package main

import (
	"context"
	"fmt"
	"jot/internal/database"
	"jot/internal/fontutil"
	"jot/internal/models"
	"jot/internal/services"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	return &App{
		db:              db,
		noteService:     services.NewNoteService(db),
		tagService:      services.NewTagService(db),
		settingService:  services.NewSettingService(db),
		notebookService: services.NewNotebookService(db),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 确保默认笔记本存在（首次启动自动创建）
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		fmt.Printf("初始化默认笔记本失败: %v\n", err)
	}
}

// ==================== Note 相关绑定方法 ====================

// CreateNote 创建一条新笔记，归入指定笔记本
func (a *App) CreateNote(title, content, noteType string, notebookID uint) (*models.Note, error) {
	return a.noteService.CreateWithNotebook(title, content, noteType, notebookID)
}

// UpdateNote 更新指定笔记的标题和内容
func (a *App) UpdateNote(id uint, title, content, noteType string) (*models.Note, error) {
	return a.noteService.Update(id, title, content, noteType)
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

// SearchNotes 按关键词搜索笔记（标题/内容），支持分页、笔记本筛选和日期范围
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, startDate, endDate string) (*services.PaginatedResult, error) {
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.SearchByNotebook(keyword, page, pageSize, notebookID, startDate, endDate)
	} else {
		notes, total, err = a.noteService.Search(keyword, page, pageSize, startDate, endDate)
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
	return stats, nil
}

// ExportDataWithDialog 弹出保存对话框，使用 VACUUM INTO 创建数据库压缩副本到用户选择的位置
func (a *App) ExportDataWithDialog() (string, error) {
	// 创建临时路径用于 VACUUM INTO 输出
	tempPath := filepath.Join(os.TempDir(), "jot-backup-"+time.Now().Format("2006-01-02")+".db")

	// 使用 VACUUM INTO 创建压缩副本
	if err := a.noteService.ExportBackup(tempPath); err != nil {
		return "", fmt.Errorf("数据库备份失败: %w", err)
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()

	// 弹出保存对话框让用户选择保存位置
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出数据库备份",
		DefaultFilename: "jot-backup-" + time.Now().Format("2006-01-02") + ".db",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite 数据库 (*.db)", Pattern: "*.db"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	// 将临时文件复制到用户选择的路径
	if err := fs.CopyEx(tempPath, filePath, true); err != nil {
		return "", err
	}

	return "导出成功：" + filePath, nil
}

// ImportDatabaseWithDialog 弹出文件选择对话框，从数据库备份文件恢复数据
func (a *App) ImportDatabaseWithDialog() (*services.ImportResult, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入数据库备份",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite 数据库 (*.db)", Pattern: "*.db"},
		},
	})
	if err != nil {
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}
	if filePath == "" {
		return &services.ImportResult{Message: "已取消"}, nil
	}

	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return &services.ImportResult{Message: "获取数据库路径失败：" + err.Error()}, nil
	}

	// Step 1: 备份当前数据库
	backupPath := dbPath + ".bak"
	if err := fs.CopyEx(dbPath, backupPath, true); err != nil {
		return &services.ImportResult{Message: "备份当前数据库失败：" + err.Error()}, nil
	}

	// Step 2: 关闭旧连接
	sqlDB, err := a.db.DB()
	if err != nil {
		_ = os.Remove(backupPath)
		return &services.ImportResult{Message: "获取数据库连接失败：" + err.Error()}, nil
	}
	_ = sqlDB.Close()

	// Step 3: 复制选定文件到数据库路径
	if err := fs.CopyEx(filePath, dbPath, true); err != nil {
		// 恢复备份
		_ = fs.CopyEx(backupPath, dbPath, true)
		if rerr := a.reconnectDB(dbPath); rerr != nil {
			return &services.ImportResult{Message: "恢复失败：" + err.Error() + "；重连也失败：" + rerr.Error()}, nil
		}
		return &services.ImportResult{Message: "复制备份文件失败：" + err.Error()}, nil
	}

	// Step 4: 重新初始化数据库
	newDB, err := database.InitDB(dbPath)
	if err != nil {
		// 恢复备份
		_ = fs.CopyEx(backupPath, dbPath, true)
		if rerr := a.reconnectDB(dbPath); rerr != nil {
			return &services.ImportResult{Message: "恢复失败：" + err.Error() + "；重连也失败：" + rerr.Error()}, nil
		}
		return &services.ImportResult{Message: "数据库重连失败：" + err.Error()}, nil
	}

	// Step 5: 重建服务
	a.db = newDB
	a.noteService = services.NewNoteService(newDB)
	a.tagService = services.NewTagService(newDB)
	a.settingService = services.NewSettingService(newDB)

	// Step 6: 清理备份
	_ = os.Remove(backupPath)

	return &services.ImportResult{Message: "已从备份文件恢复数据库", SuccessCount: 1}, nil
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

// DeleteNotebookWithNotes 删除笔记本并永久删除其下所有笔记
func (a *App) DeleteNotebookWithNotes(id uint) error {
	return a.notebookService.DeleteWithNotes(id)
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

// GetSystemFonts 获取系统已安装的字体族列表
func (a *App) GetSystemFonts() []string {
	return fontutil.GetFonts()
}

// ==================== 排序与分页设置绑定方法 ====================

// GetSortOrder 获取排序方式设置
func (a *App) GetSortOrder() string {
	order := a.settingService.Get("sort_order")
	if order == "" {
		return "updated_at"
	}
	return order
}

// SetSortOrder 保存排序方式设置
func (a *App) SetSortOrder(order string) error {
	return a.settingService.Set("sort_order", order)
}

// GetPageSize 获取分页大小设置
func (a *App) GetPageSize() int {
	size := a.settingService.Get("page_size")
	if size == "" {
		return 20
	}
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

	defaultName := sanitizeFilename(note.Title) + ".md"
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出笔记为 Markdown",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown (*.md)", Pattern: "*.md"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	content := fmt.Sprintf("# %s\n\n%s", note.Title, note.Content)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return "导出成功：" + filePath, nil
}

// sanitizeFilename 将笔记标题中的特殊符号和空白替换为下划线，生成安全的文件名
func sanitizeFilename(title string) string {
	name := strings.TrimSpace(title)
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

// ==================== 一键备份与还原绑定方法 ====================

// BackupToDir 一键备份到 ~/.jot/backup/ 目录，固定文件名 jot-backup.db（覆盖旧备份）
func (a *App) BackupToDir() (string, error) {
	backupDir, err := database.BackupDir()
	if err != nil {
		return "", fmt.Errorf("获取备份目录失败: %w", err)
	}
	if err := database.EnsureBackupDir(); err != nil {
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	filePath := filepath.Join(backupDir, "jot-backup.db")

	// VACUUM INTO 要求目标文件不存在，先删除旧备份（文件不存在也忽略）
	_ = os.Remove(filePath)

	if err := a.noteService.ExportBackup(filePath); err != nil {
		return "", fmt.Errorf("备份失败: %w", err)
	}

	return "备份成功：jot-backup.db", nil
}

// RestoreFromDir 从 backup 目录的 jot-backup.db 还原备份
func (a *App) RestoreFromDir() (*services.ImportResult, error) {
	backupDir, err := database.BackupDir()
	if err != nil {
		return &services.ImportResult{Message: "获取备份目录失败：" + err.Error()}, nil
	}

	filePath := filepath.Join(backupDir, "jot-backup.db")

	// 检查备份文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &services.ImportResult{Message: "暂无可用备份"}, nil
	} else if err != nil {
		return &services.ImportResult{Message: "读取备份文件失败：" + err.Error()}, nil
	}

	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return &services.ImportResult{Message: "获取数据库路径失败：" + err.Error()}, nil
	}

	// Step 1: 备份当前数据库
	backupPath := dbPath + ".bak"
	if err := fs.CopyEx(dbPath, backupPath, true); err != nil {
		return &services.ImportResult{Message: "备份当前数据库失败：" + err.Error()}, nil
	}

	// Step 2: 关闭旧连接
	sqlDB, err := a.db.DB()
	if err != nil {
		_ = os.Remove(backupPath)
		return &services.ImportResult{Message: "获取数据库连接失败：" + err.Error()}, nil
	}
	_ = sqlDB.Close()

	// Step 3: 复制备份文件到数据库路径
	if err := fs.CopyEx(filePath, dbPath, true); err != nil {
		_ = fs.CopyEx(backupPath, dbPath, true)
		if rerr := a.reconnectDB(dbPath); rerr != nil {
			return &services.ImportResult{Message: "恢复失败：" + err.Error() + "；重连也失败：" + rerr.Error()}, nil
		}
		return &services.ImportResult{Message: "复制备份文件失败：" + err.Error()}, nil
	}

	// Step 4: 重新初始化数据库
	newDB, err := database.InitDB(dbPath)
	if err != nil {
		_ = fs.CopyEx(backupPath, dbPath, true)
		if rerr := a.reconnectDB(dbPath); rerr != nil {
			return &services.ImportResult{Message: "恢复失败：" + err.Error() + "；重连也失败：" + rerr.Error()}, nil
		}
		return &services.ImportResult{Message: "数据库重连失败：" + err.Error()}, nil
	}

	// Step 5: 重建服务
	a.db = newDB
	a.noteService = services.NewNoteService(newDB)
	a.tagService = services.NewTagService(newDB)
	a.settingService = services.NewSettingService(newDB)

	// Step 6: 清理备份
	_ = os.Remove(backupPath)

	return &services.ImportResult{Message: "已从备份文件恢复：jot-backup.db", SuccessCount: 1}, nil
}

// GetBackupInfo 获取备份文件信息（文件名、修改时间、文件大小），无备份时返回空值
func (a *App) GetBackupInfo() (map[string]string, error) {
	backupDir, err := database.BackupDir()
	if err != nil {
		return map[string]string{"file_name": "", "file_time": "", "file_size": ""}, nil
	}

	filePath := filepath.Join(backupDir, "jot-backup.db")
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
		"file_name": "jot-backup.db",
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

		// 判断笔记类型：.md → markdown，其他 → text
		noteType := "text"
		if strings.EqualFold(ext, ".md") {
			noteType = "markdown"
		}

		// 创建笔记（归入指定笔记本）
		note, err := a.noteService.CreateWithNotebook(title, string(content), noteType, notebookID)
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

// ResetDatabase 清空所有数据（笔记/标签/设置），重新初始化默认标签，恢复出厂状态
func (a *App) ResetDatabase() error {
	// 1. 清空所有笔记和标签
	if err := a.noteService.ResetAll(); err != nil {
		return err
	}
	// 2. 清空所有设置
	if err := a.settingService.DeleteAll(); err != nil {
		return err
	}
	// 3. 清空所有笔记本，重建默认笔记本
	if err := a.notebookService.ResetAll(); err != nil {
		return err
	}
	// 4. 重新初始化默认标签
	if err := services.InitDefaultTags(a.db); err != nil {
		return err
	}
	return nil
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
	a.noteService = services.NewNoteService(db)
	a.tagService = services.NewTagService(db)
	a.settingService = services.NewSettingService(db)
	return nil
}
