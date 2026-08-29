package services

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"gitee.com/MM-Q/fastlog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"jot/internal/models"
)

// newVectorTestDB 打开内存 SQLite（单连接）并迁移 Note/Tag/NoteVector（含 note_tags 关联表）
func newVectorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1) // 内存库必须单连接，否则各连接库相互独立
	if err := db.AutoMigrate(&models.Note{}, &models.Tag{}, &models.NoteVector{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// newVectorTestService 构造 VectorService（日志写入测试临时目录）
func newVectorTestService(t *testing.T, db *gorm.DB) *VectorService {
	t.Helper()
	logger := fastlog.New(fastlog.Prod(filepath.Join(t.TempDir(), "test.log")))
	return NewVectorService(db, logger)
}

// indexNoteForTest 按生产 IndexNotes 同一切块口径（标签排序 + ChunkContent 600）为笔记写入向量记录
func indexNoteForTest(t *testing.T, db *gorm.DB, note models.Note) {
	t.Helper()
	names := make([]string, 0, len(note.Tags))
	for _, tg := range note.Tags {
		names = append(names, tg.Name)
	}
	sort.Strings(names)
	meta := ChunkMeta{Title: note.Title, Tags: names, CreatedAt: note.CreatedAt}
	chunks := ChunkContent(note.Content, 600, meta)
	rows := make([]models.NoteVector, 0, len(chunks))
	for i, c := range chunks {
		rows = append(rows, models.NoteVector{
			NoteID:     note.ID,
			ChunkIndex: i,
			ChunkText:  c,
			Model:      "test-embed",
		})
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("插入向量记录失败: %v", err)
	}
}

// createNoteForTest 创建笔记（支持附加标签），返回含 ID/CreatedAt 的实体
func createNoteForTest(t *testing.T, db *gorm.DB, title, content string, tagNames ...string) models.Note {
	t.Helper()
	note := models.Note{Title: title, Content: content}
	if err := db.Create(&note).Error; err != nil {
		t.Fatalf("创建笔记失败: %v", err)
	}
	for _, name := range tagNames {
		tag := models.Tag{Name: name}
		if err := db.Where("name = ?", name).FirstOrCreate(&tag).Error; err != nil {
			t.Fatalf("创建标签失败: %v", err)
		}
		if err := db.Model(&note).Association("Tags").Append(&tag); err != nil {
			t.Fatalf("关联标签失败: %v", err)
		}
	}
	// 重新读取以填充 Tags 关联（分类时 Preload 使用）
	if err := db.Preload("Tags").First(&note, note.ID).Error; err != nil {
		t.Fatalf("重读笔记失败: %v", err)
	}
	return note
}

// TestClassifyVectorNotesEmpty 空库：全部分类均为 0，无错误
func TestClassifyVectorNotesEmpty(t *testing.T) {
	db := newVectorTestDB(t)
	svc := newVectorTestService(t, db)

	status, err := svc.classifyVectorNotes(nilCtx())
	if err != nil {
		t.Fatalf("classifyVectorNotes 空库不应报错: %v", err)
	}
	if status.TotalNotes != 0 || status.IndexedNotes != 0 ||
		len(status.UnindexedIDs) != 0 || len(status.StaleIDs) != 0 || len(status.UpToDateIDs) != 0 {
		t.Fatalf("空库分类结果应为全 0, got %+v", status)
	}
}

// TestClassifyVectorNotesMixed 混合场景：
//   - n1 已嵌入且内容未变 → upToDate
//   - n2 已嵌入但内容已改 → stale
//   - n3 未嵌入 → unindexed
func TestClassifyVectorNotesMixed(t *testing.T) {
	db := newVectorTestDB(t)
	svc := newVectorTestService(t, db)

	n1 := createNoteForTest(t, db, "已嵌入未变", "第一章内容", "工作")
	indexNoteForTest(t, db, n1)

	n2 := createNoteForTest(t, db, "已嵌入已变", "原始正文")
	indexNoteForTest(t, db, n2)
	// 嵌入后编辑内容
	if err := db.Model(&models.Note{}).Where("id = ?", n2.ID).Update("content", "修改后的正文内容").Error; err != nil {
		t.Fatalf("更新笔记内容失败: %v", err)
	}

	n3 := createNoteForTest(t, db, "未嵌入", "从未嵌入过")

	status, err := svc.classifyVectorNotes(nilCtx())
	if err != nil {
		t.Fatalf("classifyVectorNotes 失败: %v", err)
	}
	if status.TotalNotes != 3 {
		t.Errorf("TotalNotes = %d, want 3", status.TotalNotes)
	}
	if status.IndexedNotes != 2 {
		t.Errorf("IndexedNotes = %d, want 2", status.IndexedNotes)
	}
	if !containsID(status.UpToDateIDs, n1.ID) {
		t.Errorf("n1 应归类为 upToDate, got UpToDateIDs=%v", status.UpToDateIDs)
	}
	if !containsID(status.StaleIDs, n2.ID) {
		t.Errorf("n2 应归类为 stale（内容已变化）, got StaleIDs=%v", status.StaleIDs)
	}
	if !containsID(status.UnindexedIDs, n3.ID) {
		t.Errorf("n3 应归类为 unindexed, got UnindexedIDs=%v", status.UnindexedIDs)
	}

	// 导出方法口径一致
	total, unindexed, stale, uptodate, err := svc.GetVectorNoteOverview()
	if err != nil {
		t.Fatalf("GetVectorNoteOverview 失败: %v", err)
	}
	if total != 3 || unindexed != 1 || stale != 1 || uptodate != 1 {
		t.Errorf("GetVectorNoteOverview = (%d,%d,%d,%d), want (3,1,1,1)", total, unindexed, stale, uptodate)
	}
	unidxIDs, err := svc.GetUnindexedNoteIDs()
	if err != nil || !containsID(unidxIDs, n3.ID) || len(unidxIDs) != 1 {
		t.Errorf("GetUnindexedNoteIDs = %v, err=%v, want [%d]", unidxIDs, err, n3.ID)
	}
	staleIDs, err := svc.GetStaleNoteIDs()
	if err != nil || !containsID(staleIDs, n2.ID) || len(staleIDs) != 1 {
		t.Errorf("GetStaleNoteIDs = %v, err=%v, want [%d]", staleIDs, err, n2.ID)
	}
}

// TestClassifyVectorNotesTitleChange 仅修改标题也应判定为需重新嵌入
func TestClassifyVectorNotesTitleChange(t *testing.T) {
	db := newVectorTestDB(t)
	svc := newVectorTestService(t, db)

	n := createNoteForTest(t, db, "旧标题", "正文保持不变")
	indexNoteForTest(t, db, n)
	if err := db.Model(&models.Note{}).Where("id = ?", n.ID).Update("title", "新标题").Error; err != nil {
		t.Fatalf("更新笔记标题失败: %v", err)
	}

	status, err := svc.classifyVectorNotes(nilCtx())
	if err != nil {
		t.Fatalf("classifyVectorNotes 失败: %v", err)
	}
	if len(status.UpToDateIDs) != 0 || len(status.StaleIDs) != 1 {
		t.Errorf("标题修改后应判为 stale, got UpToDate=%v Stale=%v", status.UpToDateIDs, status.StaleIDs)
	}
}

// TestClassifyVectorNotesSoftDeleted 软删（回收站）笔记不参与计数；其残留向量不计入已嵌入
func TestClassifyVectorNotesSoftDeleted(t *testing.T) {
	db := newVectorTestDB(t)
	svc := newVectorTestService(t, db)

	nLive := createNoteForTest(t, db, "正常笔记", "内容")
	indexNoteForTest(t, db, nLive)

	nTrash := createNoteForTest(t, db, "回收站笔记", "内容")
	indexNoteForTest(t, db, nTrash)
	if err := db.Delete(&models.Note{}, nTrash.ID).Error; err != nil {
		t.Fatalf("软删笔记失败: %v", err)
	}

	status, err := svc.classifyVectorNotes(nilCtx())
	if err != nil {
		t.Fatalf("classifyVectorNotes 失败: %v", err)
	}
	if status.TotalNotes != 1 {
		t.Errorf("TotalNotes = %d, want 1（排除软删笔记）", status.TotalNotes)
	}
	if status.IndexedNotes != 1 || len(status.UpToDateIDs) != 1 || len(status.StaleIDs) != 0 {
		t.Errorf("软删笔记的向量不应计入, got %+v", status)
	}
}

// TestClassifyVectorNotesEmptiedContent 已嵌入笔记内容被清空 → 存储块 >0 而重新切块为 0 → stale
func TestClassifyVectorNotesEmptiedContent(t *testing.T) {
	db := newVectorTestDB(t)
	svc := newVectorTestService(t, db)

	n := createNoteForTest(t, db, "有内容", "这是一段会被清空的内容")
	indexNoteForTest(t, db, n)
	if err := db.Model(&models.Note{}).Where("id = ?", n.ID).Update("content", "").Error; err != nil {
		t.Fatalf("清空笔记内容失败: %v", err)
	}

	status, err := svc.classifyVectorNotes(nilCtx())
	if err != nil {
		t.Fatalf("classifyVectorNotes 失败: %v", err)
	}
	if len(status.StaleIDs) != 1 || len(status.UpToDateIDs) != 0 {
		t.Errorf("内容清空应判为 stale, got UpToDate=%v Stale=%v", status.UpToDateIDs, status.StaleIDs)
	}
}

// containsID 判断 ID 切片中是否包含指定 ID
func containsID(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// nilCtx 返回 context.Background() 的别名（保持测试断言简洁）
func nilCtx() context.Context {
	return context.Background()
}
