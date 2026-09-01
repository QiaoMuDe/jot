package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NoteService 封装笔记相关的业务逻辑操作
type NoteService struct {
	db             *gorm.DB
	settingService *SettingService
	logger         *fastlog.Logger
}

// NewNoteService 创建一个新的 NoteService 实例
func NewNoteService(db *gorm.DB, settingService *SettingService, logger *fastlog.Logger) *NoteService {
	return &NoteService{db: db, settingService: settingService, logger: logger}
}

// Create 创建一条新笔记，返回创建后的笔记对象
func (s *NoteService) Create(title, content, fileExt string) (*models.Note, error) {
	note := models.Note{
		Title:   title,
		Content: content,
		FileExt: fileExt,
	}
	if err := s.db.Create(&note).Error; err != nil {
		s.logger.Errorw("NoteService.Create 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.Update 失败", fastlog.Error(err))
		return nil, err
	}
	return note, nil
}

// Delete 软删除指定 ID 的笔记（移入回收站）
func (s *NoteService) Delete(id uint) error {
	result := s.db.Delete(&models.Note{}, id)
	if result.Error != nil {
		s.logger.Errorw("NoteService.Delete 失败", fastlog.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("note not found")
	}
	return nil
}

// PermanentDelete 永久删除指定 ID 的笔记（从数据库中彻底移除），并联动清理该笔记的向量索引，避免孤儿向量残留
func (s *NoteService) PermanentDelete(id uint) error {
	result := s.db.Unscoped().Delete(&models.Note{}, id)
	if result.Error != nil {
		s.logger.Errorw("NoteService.PermanentDelete 失败", fastlog.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("note not found")
	}
	// 清理向量索引：附属数据，删除失败不阻断笔记删除，仅记日志
	if err := s.db.Where("note_id = ?", id).Delete(&models.NoteVector{}).Error; err != nil {
		s.logger.Errorw("NoteService.PermanentDelete 清理向量失败", fastlog.Uint("note_id", id), fastlog.Error(err))
	}
	return nil
}

// GetByID 按 ID 获取单条笔记，预加载关联的标签
func (s *NoteService) GetByID(id uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.Preload("Tags").First(&note, id).Error; err != nil {
		s.logger.Errorw("NoteService.GetByID 失败", fastlog.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found")
		}
		return nil, err
	}
	return &note, nil
}

// GetNoteWithRelations 按 ID 获取单条笔记，预加载标签与笔记本（Unscoped 使回收站笔记也可查，用于属性展示）
func (s *NoteService) GetNoteWithRelations(id uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.Unscoped().Preload("Tags").Preload("Notebook").First(&note, id).Error; err != nil {
		s.logger.Errorw("NoteService.GetNoteWithRelations 失败", fastlog.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found")
		}
		return nil, err
	}
	return &note, nil
}

// FindByTitleAndExt 按标题+后缀+笔记本查找已有笔记，用于导入时判断是否重复
func (s *NoteService) FindByTitleAndExt(title, fileExt string, notebookID uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.Where("title = ? AND file_ext = ? AND notebook_id = ? AND deleted_at IS NULL", title, fileExt, notebookID).
		First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		s.logger.Errorw("NoteService.FindByTitleAndExt 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.GetNoteContent 失败", fastlog.Error(err))
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

	notes := make([]NoteRefInfo, 0, len(rows))
	var parts []string

	for _, row := range rows {
		block := fmt.Sprintf("--- 📄 《%s》 ---\n%s", row.Title, row.Content)

		parts = append(parts, block)
		notes = append(notes, NoteRefInfo{
			ID:           row.ID,
			Title:        row.Title,
			Truncated:    false,
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

// escapeLike 转义 LIKE 模式中的特殊字符 % _ \，须配合 ESCAPE '\' 使用，
// 保证关键词按字面匹配而不是被当作通配符
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// buildSearchSortOrder 构建搜索时的排序规则
// 有关键词时采用相关性打分:标题完全相等 > 标题前缀命中 > 标题+内容都命中 > 仅标题命中 > 仅内容命中,同分按 pinned DESC, updated_at DESC 兜底
// 无关键词时(纯筛选场景)回退到 buildSortOrder(sortBy) 的常规排序
// 注意:必须返回 clause.OrderBy 而非 gorm.Expr——GORM v1.31.1 的 Order() 只处理
// clause.OrderBy / clause.OrderByColumn / string 三种类型,gorm.Expr 会被静默丢弃导致 ORDER BY 丢失
func buildSearchSortOrder(keyword string, sortBy string) interface{} {
	if strings.TrimSpace(keyword) == "" {
		return buildSortOrder(sortBy)
	}
	// LIKE 不带通配符时等价于不区分大小写的完全相等(SQLite 对 ASCII 不区分大小写)
	esc := escapeLike(keyword)
	return clause.OrderBy{
		Expression: clause.Expr{
			SQL: `CASE
			WHEN title LIKE ? ESCAPE '\' THEN 50
			WHEN title LIKE ? ESCAPE '\' THEN 40
			WHEN title LIKE ? ESCAPE '\' AND content LIKE ? ESCAPE '\' THEN 30
			WHEN title LIKE ? ESCAPE '\' THEN 25
			WHEN content LIKE ? ESCAPE '\' THEN 10
			ELSE 0
		END DESC, pinned DESC, updated_at DESC`,
			Vars: []interface{}{
				esc,
				esc + "%",
				"%" + esc + "%",
				"%" + esc + "%",
				"%" + esc + "%",
				"%" + esc + "%",
			},
		},
	}
}

// noteThinSelect 列表/搜索查询时使用的 Select，排除全量 Content，替换为约 80 字符用于卡片预览
// 有 keyword 时围绕关键词首次出现位置截取，无 keyword 时取前 200 字符
func noteThinSelect(keyword ...string) string {
	const base = "id, title, %s AS content, file_ext, pinned, notebook_id, created_at, updated_at"
	if len(keyword) > 0 && keyword[0] != "" {
		// 围绕关键词首次出现位置截取约 80 字符：关键词前 40 + 后 40
		// INSTR 返回 1-based 位置；MAX(1, pos-40) 确保起始位置不越界
		escaped := strings.ReplaceAll(keyword[0], "'", "''")
		return fmt.Sprintf(base,
			fmt.Sprintf("COALESCE(SUBSTR(content, MAX(1, INSTR(content, '%s') - 40), 80), SUBSTR(content, 1, 80))", escaped))
	}
	return fmt.Sprintf(base, "SUBSTR(content, 1, 200)")
}

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
		s.logger.Errorw("NoteService.GetAllByNotebook 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select(noteThinSelect()).
		Order(buildSortOrder(sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		s.logger.Errorw("NoteService.GetAllByNotebook 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.GetAllIDs 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// Search 按标题或内容关键词模糊搜索未删除的笔记，支持分页、日期范围筛选和标签 AND 过滤
func (s *NoteService) Search(keyword string, page, pageSize int, sortBy string, startDate, endDate string, tagIDs []uint) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	likePattern := "%" + escapeLike(keyword) + "%"

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Where("title LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\'", likePattern, likePattern)

	// 日期范围过滤
	if startDate != "" && endDate != "" {
		query = query.Where("updated_at BETWEEN ? AND ?",
			startDate+" 00:00:00", endDate+" 23:59:59")
	}

	// 标签 AND 过滤：使用子查询，确保笔记包含所有选中标签
	if len(tagIDs) > 0 {
		subQuery := s.db.Table("note_tags").
			Select("note_id").
			Where("tag_id IN ?", tagIDs).
			Group("note_id").
			Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs))
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorw("NoteService.Search 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select(noteThinSelect(keyword)).
		Order(buildSearchSortOrder(keyword, sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		s.logger.Errorw("NoteService.Search 失败", fastlog.Error(err))
		return nil, 0, err
	}

	return notes, total, nil
}

// SearchByNotebook 在指定笔记本范围内按关键词搜索，支持分页、日期范围筛选和标签 AND 过滤
func (s *NoteService) SearchByNotebook(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string, tagIDs []uint) ([]models.Note, int64, error) {
	var notes []models.Note
	var total int64

	likePattern := "%" + escapeLike(keyword) + "%"

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL AND notebook_id = ?", notebookID).
		Where("title LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\'", likePattern, likePattern)

	if startDate != "" && endDate != "" {
		query = query.Where("updated_at BETWEEN ? AND ?",
			startDate+" 00:00:00", endDate+" 23:59:59")
	}

	// 标签 AND 过滤：使用子查询，确保笔记包含所有选中标签
	if len(tagIDs) > 0 {
		subQuery := s.db.Table("note_tags").
			Select("note_id").
			Where("tag_id IN ?", tagIDs).
			Group("note_id").
			Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs))
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorw("NoteService.SearchByNotebook 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Select(noteThinSelect(keyword)).
		Order(buildSearchSortOrder(keyword, sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		s.logger.Errorw("NoteService.SearchByNotebook 失败", fastlog.Error(err))
		return nil, 0, err
	}

	return notes, total, nil
}

// SearchNoteIDs 按关键词/标签搜索并返回所有匹配笔记 ID（不分页）
func (s *NoteService) SearchNoteIDs(keyword string, tagIDs []uint) ([]uint, error) {
	var ids []uint

	likePattern := "%" + escapeLike(keyword) + "%"

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Where("title LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\'", likePattern, likePattern)

	// 标签 AND 过滤：使用子查询，确保笔记包含所有选中标签
	if len(tagIDs) > 0 {
		subQuery := s.db.Table("note_tags").
			Select("note_id").
			Where("tag_id IN ?", tagIDs).
			Group("note_id").
			Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs))
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Pluck("id", &ids).Error; err != nil {
		s.logger.Errorw("NoteService.SearchNoteIDs 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// SearchNoteIDsByNotebook 在指定笔记本中按关键词/标签搜索并返回所有匹配笔记 ID（不分页）
func (s *NoteService) SearchNoteIDsByNotebook(keyword string, notebookID uint, tagIDs []uint) ([]uint, error) {
	var ids []uint

	likePattern := "%" + escapeLike(keyword) + "%"

	query := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL AND notebook_id = ?", notebookID).
		Where("title LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\'", likePattern, likePattern)

	// 标签 AND 过滤：使用子查询，确保笔记包含所有选中标签
	if len(tagIDs) > 0 {
		subQuery := s.db.Table("note_tags").
			Select("note_id").
			Where("tag_id IN ?", tagIDs).
			Group("note_id").
			Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs))
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Pluck("id", &ids).Error; err != nil {
		s.logger.Errorw("NoteService.SearchNoteIDsByNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// TogglePin 切换指定笔记的置顶状态，返回更新后的笔记对象
func (s *NoteService) TogglePin(id uint) (*models.Note, error) {
	note, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	note.Pinned = !note.Pinned
	if err := s.db.Model(note).UpdateColumn("pinned", note.Pinned).Error; err != nil {
		s.logger.Errorw("NoteService.TogglePin 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.GetTrash 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		s.logger.Errorw("NoteService.GetTrash 失败", fastlog.Error(err))
		return nil, 0, err
	}

	return notes, total, nil
}

// RestoreAll 批量恢复回收站中所有已软删除的笔记。
// 按笔记所属笔记本的状态分三种场景处理:
//  1. 父笔记本在回收站（软删除）→ 先恢复该笔记本, 笔记回到原 notebook_id
//  2. 父笔记本存活 → 笔记直接回到原 notebook_id
//  3. 父笔记本已被永久删除/不存在 → 笔记迁到默认笔记本 (id=1) 后恢复
//
// 默认笔记本 (id=1) 因 Delete/DeleteWithNotes/RestoreFromTrash 都有 id==1 守卫,
// 永远不会被软删除, 因此不在恢复范围。
func (s *NoteService) RestoreAll() error {
	// Stage 1: 恢复回收站笔记引用的、且本身在回收站的非默认笔记本
	if err := s.db.Unscoped().Exec(`
		UPDATE notebooks
		SET deleted_at = NULL
		WHERE deleted_at IS NOT NULL
		  AND id IN (
		      SELECT DISTINCT notebook_id
		      FROM notes
		      WHERE deleted_at IS NOT NULL
		        AND notebook_id != 0
		        AND notebook_id != 1
		  )
	`).Error; err != nil {
		s.logger.Errorw("NoteService.RestoreAll 失败(恢复关联笔记本)", fastlog.Error(err))
		return err
	}

	// Stage 2: 父笔记本已永久删除或不存在 → 迁到默认笔记本
	if err := s.db.Unscoped().Model(&models.Note{}).
		Where("deleted_at IS NOT NULL AND notebook_id != 1").
		Where("notebook_id NOT IN (SELECT id FROM notebooks)").
		Update("notebook_id", 1).Error; err != nil {
		s.logger.Errorw("NoteService.RestoreAll 失败(迁默认笔记本)", fastlog.Error(err))
		return err
	}

	// Stage 3: 取消所有回收站笔记的 deleted_at
	if err := s.db.Unscoped().Model(&models.Note{}).
		Where("deleted_at IS NOT NULL").
		Update("deleted_at", nil).Error; err != nil {
		s.logger.Errorw("NoteService.RestoreAll 失败(恢复笔记)", fastlog.Error(err))
		return err
	}
	return nil
}

// EmptyTrash 永久删除回收站中所有已软删除的笔记，并联动清理这些笔记的向量索引，避免孤儿向量残留
func (s *NoteService) EmptyTrash() error {
	// 先收集回收站笔记 ID，用于删除后清理对应向量
	var ids []uint
	if err := s.db.Unscoped().Model(&models.Note{}).
		Where("deleted_at IS NOT NULL").Pluck("id", &ids).Error; err != nil {
		s.logger.Errorw("NoteService.EmptyTrash 查询回收站笔记失败", fastlog.Error(err))
		return err
	}

	result := s.db.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Note{})
	if result.Error != nil {
		s.logger.Errorw("NoteService.EmptyTrash 失败", fastlog.Error(result.Error))
		return result.Error
	}

	// 清理向量索引：附属数据，删除失败不阻断清空，仅记日志
	if len(ids) > 0 {
		if err := s.db.Where("note_id IN ?", ids).Delete(&models.NoteVector{}).Error; err != nil {
			s.logger.Errorw("NoteService.EmptyTrash 清理向量失败", fastlog.Error(err))
		}
	}
	return nil
}

// CleanExpiredTrash 永久删除回收站中超过指定天数的笔记，并联动清理这些笔记的向量索引
func (s *NoteService) CleanExpiredTrash(days int) int64 {
	// 先收集过期笔记 ID，用于删除后清理对应向量
	var ids []uint
	_ = s.db.Unscoped().Model(&models.Note{}).
		Where(fmt.Sprintf("deleted_at IS NOT NULL AND deleted_at < datetime('now', '-%d days')", days)).
		Pluck("id", &ids).Error

	result := s.db.Unscoped().Where(fmt.Sprintf("deleted_at IS NOT NULL AND deleted_at < datetime('now', '-%d days')", days)).Delete(&models.Note{})

	// 清理向量索引：附属数据，删除失败不影响整体，仅记日志
	if result.Error == nil && len(ids) > 0 {
		if err := s.db.Where("note_id IN ?", ids).Delete(&models.NoteVector{}).Error; err != nil {
			s.logger.Errorw("NoteService.CleanExpiredTrash 清理向量失败", fastlog.Error(err))
		}
	}
	return result.RowsAffected
}

// MigrateOrphanNotes 将所有指向不存在笔记本的笔记迁移到默认笔记本
func (s *NoteService) MigrateOrphanNotes() int64 {
	result := s.db.Model(&models.Note{}).
		Where("notebook_id NOT IN (SELECT id FROM notebooks) AND notebook_id != 0").
		Update("notebook_id", 1)
	return result.RowsAffected
}

// BatchPinNotes 批量置顶或取消置顶指定 ID 数组的笔记
func (s *NoteService) BatchPinNotes(ids []uint, pin bool) error {
	err := s.db.Model(&models.Note{}).Where("id IN ?", ids).UpdateColumn("pinned", pin).Error
	if err != nil {
		s.logger.Errorw("NoteService.BatchPinNotes 失败", fastlog.Error(err))
	}
	return err
}

// BatchDelete 批量软删除指定 ID 数组的笔记（移入回收站）
func (s *NoteService) BatchDelete(ids []uint) error {
	err := s.db.Where("id IN ?", ids).Delete(&models.Note{}).Error
	if err != nil {
		s.logger.Errorw("NoteService.BatchDelete 失败", fastlog.Error(err))
	}
	return err
}

// BatchRestore 批量从回收站恢复指定 ID 数组的笔记。
// 对每条笔记按所属笔记本状态分三场景处理:
//  1. 父笔记本在回收站（软删除）→ 先恢复该笔记本, 笔记回到原 notebook_id
//  2. 父笔记本存活 → 笔记直接回到原 notebook_id
//  3. 父笔记本已被永久删除/不存在 → 笔记迁到默认笔记本 (id=1) 后恢复
func (s *NoteService) BatchRestore(ids []uint) error {
	// Stage 1: 恢复这些笔记引用的、且本身在回收站的非默认笔记本
	if err := s.db.Unscoped().Exec(`
		UPDATE notebooks
		SET deleted_at = NULL
		WHERE deleted_at IS NOT NULL
		  AND id IN (
		      SELECT DISTINCT notebook_id
		      FROM notes
		      WHERE id IN ?
		        AND deleted_at IS NOT NULL
		        AND notebook_id != 0
		        AND notebook_id != 1
		  )
	`, ids).Error; err != nil {
		s.logger.Errorw("NoteService.BatchRestore 失败(恢复关联笔记本)", fastlog.Error(err))
		return err
	}

	// Stage 2: 父笔记本已永久删除或不存在 → 迁到默认笔记本
	if err := s.db.Unscoped().Model(&models.Note{}).
		Where("id IN ? AND deleted_at IS NOT NULL AND notebook_id != 1", ids).
		Where("notebook_id NOT IN (SELECT id FROM notebooks)").
		Update("notebook_id", 1).Error; err != nil {
		s.logger.Errorw("NoteService.BatchRestore 失败(迁默认笔记本)", fastlog.Error(err))
		return err
	}

	// Stage 3: 取消这些笔记的 deleted_at
	if err := s.db.Unscoped().Model(&models.Note{}).
		Where("id IN ? AND deleted_at IS NOT NULL", ids).
		Update("deleted_at", nil).Error; err != nil {
		s.logger.Errorw("NoteService.BatchRestore 失败(恢复笔记)", fastlog.Error(err))
		return err
	}
	return nil
}

// Restore 从回收站恢复指定 ID 的笔记（取消软删除）。
// 按笔记所属笔记本状态分三场景处理:
//  1. 父笔记本在回收站（软删除）→ 先恢复该笔记本, 笔记回到原 notebook_id
//  2. 父笔记本存活 → 笔记直接回到原 notebook_id
//  3. 父笔记本已被永久删除/不存在 → 笔记迁到默认笔记本 (id=1) 后恢复
func (s *NoteService) Restore(id uint) error {
	// 先获取笔记信息（含软删除）
	var note models.Note
	if err := s.db.Unscoped().First(&note, id).Error; err != nil {
		s.logger.Errorw("NoteService.Restore 失败", fastlog.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("note not found")
		}
		return err
	}

	// 笔记本 id=0 (历史脏数据) 或 id=1 (默认, 永不在 trash) 无需处理
	if note.NotebookID != 0 && note.NotebookID != 1 {
		// Unscoped 查询以包含软删除记录, 然后按状态分支处理
		var notebook models.Notebook
		err := s.db.Unscoped().First(&notebook, note.NotebookID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 笔记本已被永久删除/不存在 → 迁到默认笔记本
			if err := s.db.Unscoped().Model(&note).Update("notebook_id", 1).Error; err != nil {
				s.logger.Errorw("NoteService.Restore 失败(迁默认)", fastlog.Error(err))
				return err
			}
		} else if err != nil {
			s.logger.Errorw("NoteService.Restore 失败", fastlog.Error(err))
			return err
		} else if notebook.DeletedAt.Valid {
			// 笔记本在回收站 (软删除) → 恢复笔记本, 笔记保持原 notebook_id
			if err := s.db.Unscoped().Model(&notebook).Update("deleted_at", nil).Error; err != nil {
				s.logger.Errorw("NoteService.Restore 失败(恢复笔记本)", fastlog.Error(err))
				return err
			}
		}
		// 笔记本存活时: 无需任何处理, 笔记保持原 notebook_id
	}

	// 恢复笔记
	result := s.db.Unscoped().Model(&models.Note{}).Where("id = ?", id).Update("deleted_at", nil)
	if result.Error != nil {
		s.logger.Errorw("NoteService.Restore 失败", fastlog.Error(result.Error))
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
		s.logger.Errorw("NoteService.GetByTag 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order(buildSortOrder(sortBy)).
		Preload("Tags").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error; err != nil {
		s.logger.Errorw("NoteService.GetByTag 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.GetStats 失败", fastlog.Error(err))
		return nil, err
	}
	// 回收站笔记数
	if err := s.db.Model(&models.Note{}).Unscoped().Where("deleted_at IS NOT NULL").Count(&trashedNotes).Error; err != nil {
		s.logger.Errorw("NoteService.GetStats 失败", fastlog.Error(err))
		return nil, err
	}
	// 置顶笔记数
	if err := s.db.Model(&models.Note{}).Where("deleted_at IS NULL AND pinned = ?", true).Count(&pinnedNotes).Error; err != nil {
		s.logger.Errorw("NoteService.GetStats 失败", fastlog.Error(err))
		return nil, err
	}
	// 笔记本数（包含软删除的保留计数，不统计已删除的笔记本）
	if err := s.db.Model(&models.Notebook{}).Where("deleted_at IS NULL").Count(&totalNotebooks).Error; err != nil {
		s.logger.Errorw("NoteService.GetStats 失败", fastlog.Error(err))
		return nil, err
	}

	return &DataStats{
		TotalNotes:     totalNotes,
		TrashedNotes:   trashedNotes,
		PinnedNotes:    pinnedNotes,
		TotalNotebooks: totalNotebooks,
	}, nil
}

// ResetAll 清空所有笔记、标签和待办数据，用于恢复出厂设置
func (s *NoteService) ResetAll() error {
	// 清空所有笔记（包括软删除，自动清理 note_tags 关联）
	if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Note{}).Error; err != nil {
		s.logger.Errorw("NoteService.ResetAll 失败", fastlog.Error(err))
		return err
	}
	// 清空所有标签（自动清理 note_tags 中残留关联）
	if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Tag{}).Error; err != nil {
		s.logger.Errorw("NoteService.ResetAll 失败", fastlog.Error(err))
		return err
	}
	// 清空所有待办
	if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Todo{}).Error; err != nil {
		s.logger.Errorw("NoteService.ResetAll 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// ExportBackup 使用 VACUUM INTO 创建数据库的压缩副本到指定路径
func (s *NoteService) ExportBackup(destPath string) error {
	err := s.db.Exec("VACUUM INTO ?", destPath).Error
	if err != nil {
		s.logger.Errorw("NoteService.ExportBackup 失败", fastlog.Error(err))
	}
	return err
}

// Vacuum 执行 SQLite VACUUM 命令，重建数据库文件以回收已删除数据占用的磁盘空间
func (s *NoteService) Vacuum() error {
	err := s.db.Exec("VACUUM").Error
	if err != nil {
		s.logger.Errorw("NoteService.Vacuum 失败", fastlog.Error(err))
	}
	return err
}

// GetAllNoteIDsByNotebook 获取指定笔记本中所有未删除笔记的 ID 数组
func (s *NoteService) GetAllNoteIDsByNotebook(notebookID uint) ([]uint, error) {
	var ids []uint
	if err := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL AND notebook_id = ?", notebookID).
		Pluck("id", &ids).Error; err != nil {
		s.logger.Errorw("NoteService.GetAllNoteIDsByNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// MigrateOrphanNotesToDefault 将 notebook_id=0 的存量笔记迁移到默认笔记本（id=1）
func (s *NoteService) MigrateOrphanNotesToDefault() error {
	err := s.db.Model(&models.Note{}).Where("notebook_id = ?", 0).Update("notebook_id", 1).Error
	if err != nil {
		s.logger.Errorw("NoteService.MigrateOrphanNotesToDefault 失败", fastlog.Error(err))
	}
	return err
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
		s.logger.Errorw("NoteService.MoveToNotebook 失败", fastlog.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("notebook not found")
		}
		return err
	}

	// 使用 UpdateColumn 以避免修改 UpdatedAt
	if err := s.db.Model(&models.Note{}).Where("id = ?", noteID).UpdateColumn("notebook_id", targetNotebookID).Error; err != nil {
		s.logger.Errorw("NoteService.MoveToNotebook 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// BatchMoveToNotebook 批量将多条笔记移动到目标笔记本
func (s *NoteService) BatchMoveToNotebook(noteIDs []uint, targetNotebookID uint) error {
	// 先检查目标笔记本是否存在
	var notebook models.Notebook
	if err := s.db.First(&notebook, targetNotebookID).Error; err != nil {
		s.logger.Errorw("NoteService.BatchMoveToNotebook 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.UpdateFileExt 失败", fastlog.Error(err))
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
		s.logger.Errorw("NoteService.CreateWithNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	return &note, nil
}

// CreateWithNotebookAt 创建一条新笔记并指定所属笔记本，同时把 CreatedAt/UpdatedAt
// 对齐为指定时间（导入场景下为文件的修改时间），使时间戳成为文件同步的对比基准。
// GORM 对时间字段仅在零值时自动填充，预设非零值会被保留。
func (s *NoteService) CreateWithNotebookAt(title, content, fileExt string, notebookID uint, t time.Time) (*models.Note, error) {
	note := models.Note{
		Title:      title,
		Content:    content,
		FileExt:    fileExt,
		NotebookID: notebookID,
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	if err := s.db.Create(&note).Error; err != nil {
		s.logger.Errorw("NoteService.CreateWithNotebookAt 失败", fastlog.Error(err))
		return nil, err
	}
	// GORM 若仍覆盖了预设时间戳，则用 UpdateColumns 修正（一次额外写入，仅在需要时执行）
	if !note.CreatedAt.Equal(t) || !note.UpdatedAt.Equal(t) {
		if err := s.db.Model(&models.Note{}).Where("id = ?", note.ID).UpdateColumns(map[string]interface{}{
			"created_at": t,
			"updated_at": t,
		}).Error; err != nil {
			s.logger.Errorw("NoteService.CreateWithNotebookAt 修正时间戳失败", fastlog.Error(err))
			return nil, err
		}
		note.CreatedAt = t
		note.UpdatedAt = t
	}
	return &note, nil
}

// UpdateWithTime 更新指定 ID 的笔记的标题、内容、后缀，并把 UpdatedAt 对齐为指定时间
// （导入覆盖场景下为文件的修改时间）。使用 UpdateColumns 绕过 GORM 自动刷新 UpdatedAt。
func (s *NoteService) UpdateWithTime(id uint, title, content, fileExt string, t time.Time) (*models.Note, error) {
	if _, err := s.GetByID(id); err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.Note{}).Where("id = ?", id).UpdateColumns(map[string]interface{}{
		"title":      title,
		"content":    content,
		"file_ext":   fileExt,
		"updated_at": t,
	}).Error; err != nil {
		s.logger.Errorw("NoteService.UpdateWithTime 失败", fastlog.Error(err))
		return nil, err
	}
	return s.GetByID(id)
}

// noteTitleMaxRunes 笔记标题上限（models.Note.Title 为 size:200 的 VARCHAR，按 rune 计保护多字节标题）。
const noteTitleMaxRunes = 200

// nextDuplicateTitle 生成"标题 副本"形式的副本标题：先试 base+" 副本"，
// 若 titleExists 判定已存在则依次递增序号（"副本 2"、"副本 3"…）取第一个不冲突的；
// 超长时只截断 base 前缀、保留"副本"后缀（总长按 rune 不超过 noteTitleMaxRunes）。
// titleExists 由调用方注入（DB 查询或测试闭包），本函数不触达存储。
func nextDuplicateTitle(base string, titleExists func(string) bool) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "未命名"
	}
	for n := 0; ; n++ {
		suffix := " 副本"
		if n > 0 {
			suffix = fmt.Sprintf(" 副本 %d", n+1) // 副本 2 / 副本 3 …
		}
		// 只截断 base 前缀，保证" 副本"后缀完整保留
		baseTrunc := truncateTitleRunes(base, noteTitleMaxRunes-len([]rune(suffix)))
		candidate := baseTrunc + suffix
		if !titleExists(candidate) {
			return candidate
		}
	}
}

// truncateTitleRunes 将标题截断到 maxRunes 个 rune（从末尾截断，保留前缀；不产生孤立代理对）。
func truncateTitleRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

// NextDuplicateTitle 为原标题生成不冲突的"副本"标题（供创建副本使用）。
// titleExists 通过 DB 精确匹配未删除笔记判断；原标题为空时回退"未命名"。
func (s *NoteService) NextDuplicateTitle(origTitle string) (string, error) {
	exists := func(title string) bool {
		var count int64
		if err := s.db.Model(&models.Note{}).
			Where("title = ? AND deleted_at IS NULL", title).
			Count(&count).Error; err != nil {
			s.logger.Errorw("NoteService.NextDuplicateTitle 查重失败", fastlog.Error(err))
			return true // 查询失败时保守视为已存在，避免重名
		}
		return count > 0
	}
	return nextDuplicateTitle(origTitle, exists), nil
}

// GetByDate 按创建日期查询非删除笔记，使用 noteThinSelect 避免加载大文本内容
func (s *NoteService) GetByDate(date string) ([]models.Note, error) {
	var notes []models.Note
	start := date + " 00:00:00"
	end := date + " 23:59:59"
	if err := s.db.Model(&models.Note{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Where("deleted_at IS NULL").
		Select(noteThinSelect()).
		Order("created_at DESC").
		Preload("Tags").
		Preload("Notebook").
		Find(&notes).Error; err != nil {
		s.logger.Errorw("NoteService.GetByDate 失败", fastlog.Error(err))
		return nil, err
	}
	return notes, nil
}

// GetMonthCounts 按月统计某月每天创建的非删除笔记数量，返回 map[int]int（key 为日期数字 1-31）
func (s *NoteService) GetMonthCounts(year, month int) (map[int]int, error) {
	// 使用范围查询（月初 → 下月初）替代 strftime 函数过滤，命中 idx_notes_created 索引，
	// 避免大库下每次进日历/启动渲染当月时的全表扫描
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	type dayCount struct {
		Day   int `gorm:"column:day"`
		Count int `gorm:"column:count"`
	}
	var results []dayCount
	if err := s.db.Model(&models.Note{}).
		Select("strftime('%d', created_at) as day, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", start, end).
		Group("strftime('%d', created_at)").
		Find(&results).Error; err != nil {
		s.logger.Errorw("NoteService.GetMonthCounts 失败", fastlog.Error(err))
		return nil, err
	}

	counts := make(map[int]int, len(results))
	for _, r := range results {
		counts[r.Day] = r.Count
	}
	return counts, nil
}

// SearchByNotebook 在指定笔记本范围内按标题或内容关键词
