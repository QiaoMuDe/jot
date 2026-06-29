package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"jot/internal/models"
)

// NoteService 封装笔记相关的业务逻辑操作
type NoteService struct {
	db             *gorm.DB
	settingService *SettingService
}

// NewNoteService 创建一个新的 NoteService 实例
func NewNoteService(db *gorm.DB, settingService *SettingService) *NoteService {
	return &NoteService{db: db, settingService: settingService}
}

// Create 创建一条新笔记，返回创建后的笔记对象
func (s *NoteService) Create(title, content, fileExt string) (*models.Note, error) {
	note := models.Note{
		Title:   title,
		Content: content,
		FileExt: fileExt,
	}
	if err := s.db.Create(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

// Update 更新指定 ID 的笔记的标题和内容，返回更新后的笔记对象
func (s *NoteService) Update(id uint, title, content, fileExt string) (*models.Note, error) {
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
	if fileExt != "" {
		note.FileExt = fileExt
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

// GetNoteContent 按 ID 仅获取笔记的完整 content 文本（列表查询只返回截断版本，用于按需加载）
func (s *NoteService) GetNoteContent(id uint) (string, error) {
	var content string
	if err := s.db.Model(&models.Note{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Select("content").
		Take(&content).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("note not found")
		}
		return "", err
	}
	return content, nil
}

// BuildNoteRefContext 构建笔记引用上下文。
// 后端一次性完成：查库 → 逐条截断(4000字/条) → 总长度截断(8000字) → 拼装 context 字符串。
// 返回每条笔记的引用信息(含截断状态)和拼装好的上下文文本。
func (s *NoteService) BuildNoteRefContext(ids []uint) (*NoteRefContext, error) {
	if len(ids) == 0 {
		return &NoteRefContext{Notes: []NoteRefInfo{}, Context: ""}, nil
	}

	// 联表查询：笔记 + 笔记本名称
	type noteRow struct {
		ID           uint
		Title        string
		Content      string
		NotebookName string
	}
	var rows []noteRow
	if err := s.db.Table("notes").
		Select("notes.id, notes.title, notes.content, COALESCE(notebooks.name, '') as notebook_name").
		Joins("LEFT JOIN notebooks ON notes.notebook_id = notebooks.id").
		Where("notes.id IN ? AND notes.deleted_at IS NULL", ids).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query note ref rows: %w", err)
	}

	// 从设置动态读取单条笔记截断字数，默认 1000
	maxPerNote := 1000
	if s.settingService != nil {
		val := s.settingService.Get("ai_ref_max_chars")
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			maxPerNote = n
		}
	}
	const maxTotalChars = 8000 // 总上下文最大字符数

	notes := make([]NoteRefInfo, 0, len(rows))
	var parts []string
	totalLen := 0

	for _, row := range rows {
		// 截断单条笔记内容
		noteText := row.Content
		truncated := len(noteText) > maxPerNote
		if truncated {
			noteText = noteText[:maxPerNote] + "\n...(内容已截断)"
		}

		block := fmt.Sprintf("--- 📄 《%s》 ---\n%s", row.Title, noteText)

		// 总长度截断
		if totalLen+len(block) > maxTotalChars {
			parts = append(parts, fmt.Sprintf("--- 📄 《%s》 ---\n...(内容已截断，超出上下文长度限制)", row.Title))
			notes = append(notes, NoteRefInfo{
				ID:           row.ID,
				Title:        row.Title,
				Truncated:    true,
				NotebookName: row.NotebookName,
			})
			// 剩余笔记不再处理
			break
		}

		parts = append(parts, block)
		totalLen += len(block)
		notes = append(notes, NoteRefInfo{
			ID:           row.ID,
			Title:        row.Title,
			Truncated:    truncated,
			NotebookName: row.NotebookName,
		})
	}

	header := "以下是用户引用的笔记，请作为回答的参考上下文：\n\n"
	footer := "\n\n请基于以上笔记内容回答用户的问题。如果笔记内容不足以回答，请如实说明。"
	contextStr := header + strings.Join(parts, "\n\n") + footer

	return &NoteRefContext{
		Notes:   notes,
		Context: contextStr,
	}, nil
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

// noteThinSelect 列表/搜索查询时使用的 Select，排除全量 Content，替换为前 200 字符用于卡片预览
const noteThinSelect = "id, title, SUBSTR(content, 1, 200) AS content, file_ext, pinned, notebook_id, created_at, updated_at"

// GetAll 分页获取未删除的笔记列表（不过滤 notebook_id），按指定排序方式排列，返回列表与总数
func (s *NoteService) GetAll(page, pageSize int, sortBy string) ([]models.Note, int64, error) {
	return s.GetAllByNotebook(page, pageSize, sortBy, 0)
}

// GetAllByNotebook 按 notebook_id 筛选分页获取未删除的笔记列表，支持指定排序方式并预加载标签
// 当 notebookID 为 0 时，不过滤笔记本，返回所有未删除笔记
func (s *NoteService) GetAllByNotebook(page, pageSize int, sortBy string, notebookID uint) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	query := s.db.Model(&models.Note{}).Where("deleted_at IS NULL")
	if notebookID > 0 {
		query = query.Where("notebook_id = ?", notebookID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select(noteThinSelect).
		Order(buildSortOrder(sortBy)).
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

// Search 按标题或内容关键词模糊搜索未删除的笔记，支持分页和日期范围筛选
func (s *NoteService) Search(keyword string, page, pageSize int, sortBy string, startDate, endDate string) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	likePattern := "%" + keyword + "%"

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Where("title LIKE ? OR content LIKE ?", likePattern, likePattern)

	// 日期范围过滤
	if startDate != "" && endDate != "" {
		query = query.Where("updated_at BETWEEN ? AND ?",
			startDate+" 00:00:00", endDate+" 23:59:59")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select(noteThinSelect).
		Order(buildSortOrder(sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// SearchByNotebook 在指定笔记本范围内按关键词搜索，支持分页和日期范围筛选
func (s *NoteService) SearchByNotebook(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	likePattern := "%" + keyword + "%"

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL AND notebook_id = ?", notebookID).
		Where("title LIKE ? OR content LIKE ?", likePattern, likePattern)

	if startDate != "" && endDate != "" {
		query = query.Where("updated_at BETWEEN ? AND ?",
			startDate+" 00:00:00", endDate+" 23:59:59")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select(noteThinSelect).
		Order(buildSortOrder(sortBy)).
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

// BatchPinNotes 批量置顶或取消置顶指定 ID 数组的笔记
func (s *NoteService) BatchPinNotes(ids []uint, pin bool) error {
	result := s.db.Model(&models.Note{}).Where("id IN ?", ids).UpdateColumn("pinned", pin)
	return result.Error
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
// GetStats 获取数据统计概览（笔记数/标签数/笔记本数/数据库大小）
func (s *NoteService) GetStats() (*DataStats, error) {
	var totalNotes, trashedNotes, pinnedNotes, totalNotebooks int64

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
	// 笔记本数（包含软删除的保留计数，不统计已删除的笔记本）
	if err := s.db.Model(&models.Notebook{}).Where("deleted_at IS NULL").Count(&totalNotebooks).Error; err != nil {
		return nil, err
	}

	return &DataStats{
		TotalNotes:     totalNotes,
		TrashedNotes:   trashedNotes,
		PinnedNotes:    pinnedNotes,
		TotalNotebooks: totalNotebooks,
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

// Vacuum 执行 SQLite VACUUM 命令，重建数据库文件以回收已删除数据占用的磁盘空间
func (s *NoteService) Vacuum() error {
	return s.db.Exec("VACUUM").Error
}

// GetAllNoteIDsByNotebook 获取指定笔记本中所有未删除笔记的 ID 数组
func (s *NoteService) GetAllNoteIDsByNotebook(notebookID uint) ([]uint, error) {
	var ids []uint
	if err := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL AND notebook_id = ?", notebookID).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// MigrateOrphanNotesToDefault 将 notebook_id=0 的存量笔记迁移到默认笔记本（id=1）
func (s *NoteService) MigrateOrphanNotesToDefault() error {
	return s.db.Model(&models.Note{}).Where("notebook_id = ?", 0).Update("notebook_id", 1).Error
}

// MoveToNotebook 将单条笔记移动到目标笔记本
func (s *NoteService) MoveToNotebook(noteID uint, targetNotebookID uint) error {
	// 检查笔记是否存在
	if _, err := s.GetByID(noteID); err != nil {
		return err
	}

	// 检查目标笔记本是否存在
	var notebook models.Notebook
	if err := s.db.First(&notebook, targetNotebookID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notebook not found")
		}
		return err
	}

	// 使用 UpdateColumn 以避免修改 UpdatedAt
	return s.db.Model(&models.Note{}).Where("id = ?", noteID).UpdateColumn("notebook_id", targetNotebookID).Error
}

// BatchMoveToNotebook 批量将多条笔记移动到目标笔记本
func (s *NoteService) BatchMoveToNotebook(noteIDs []uint, targetNotebookID uint) error {
	// 先检查目标笔记本是否存在
	var notebook models.Notebook
	if err := s.db.First(&notebook, targetNotebookID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notebook not found")
		}
		return err
	}

	// 遍历笔记 ID 逐个迁移，遇到错误不中断，继续迁移剩余笔记
	var errs []error
	for _, noteID := range noteIDs {
		if err := s.MoveToNotebook(noteID, targetNotebookID); err != nil {
			errs = append(errs, fmt.Errorf("note %d: %w", noteID, err))
		}
	}

	// 如果有错误，合并返回
	if len(errs) > 0 {
		combined := "batch move errors: "
		for i, e := range errs {
			if i > 0 {
				combined += "; "
			}
			combined += e.Error()
		}
		return errors.New(combined)
	}

	return nil
}

// UpdateFileExt 更新指定笔记的文件后缀
func (s *NoteService) UpdateFileExt(id uint, fileExt string) (*models.Note, error) {
	note, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	note.FileExt = fileExt
	if err := s.db.Save(note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

// CreateWithNotebook 创建一条新笔记并指定所属笔记本，返回创建后的笔记对象
func (s *NoteService) CreateWithNotebook(title, content, fileExt string, notebookID uint) (*models.Note, error) {
	note := models.Note{
		Title:      title,
		Content:    content,
		FileExt:    fileExt,
		NotebookID: notebookID,
	}
	if err := s.db.Create(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

// SearchByNotebook 在指定笔记本范围内按标题或内容关键词
