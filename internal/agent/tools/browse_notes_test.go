package tools

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"jot/internal/models"
	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newBrowseNotesTestService 打开内存 SQLite（单连接）、迁移 Note/Tag，
// 构造 browseNotesTool（note + setting），并默认把大文件预览阈值设为 10，
// 使 view 的首段截断在短内容上即可确定性触发。
func newBrowseNotesTestService(t *testing.T) (*browseNotesTool, *gorm.DB) {
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
	if err := db.AutoMigrate(&models.Note{}, &models.Tag{}, &models.Setting{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	setting := services.NewSettingService(db)
	if err := setting.Set("ai_large_file_preview_threshold", "10"); err != nil {
		t.Fatalf("设置预览阈值失败: %v", err)
	}
	svc := services.NewNoteService(db, setting, fastlog.New(fastlog.Prod(filepath.Join(t.TempDir(), "test.log"))))
	return &browseNotesTool{note: svc, setting: setting, ctx: nil}, db
}

// seedBrowseNote 插入一条笔记并返回其 ID。
func seedBrowseNote(t *testing.T, db *gorm.DB, title, content string) uint {
	t.Helper()
	n := models.Note{Title: title, Content: content}
	if err := db.Create(&n).Error; err != nil {
		t.Fatalf("插入笔记(%q)失败: %v", title, err)
	}
	return n.ID
}

// TestBrowseNotesViewSegmentTruncate 验证 view 首段截断 + 续读：
// offset=0 且缺省 length 时按阈值截取首段并给续读指引，续读 offset 后取后续段不再带提示。
func TestBrowseNotesViewSegmentTruncate(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	id := seedBrowseNote(t, db, "segment", "1234567890ABCDEFGHIJ") // 20 字符，阈值 10

	first, err := m.viewNote([]float64{float64(id)}, false, 0, 0)
	if err != nil {
		t.Fatalf("首段 view 意外错误: %v", err)
	}
	if !strings.Contains(first, "笔记 #"+strconv.Itoa(int(id))+" 内容：") {
		t.Errorf("缺少笔记编号前缀: %v", first)
	}
	if !strings.Contains(first, "1234567890") {
		t.Errorf("首段应包含前 10 字符: %v", first)
	}
	if !strings.Contains(first, "内容共 20 字符 / 1 行，已显示前 10 字符 / 1 行") {
		t.Errorf("截断提示应标识总字符/行与已显示: %v", first)
	}
	if !strings.Contains(first, "offset=10") {
		t.Errorf("截断提示应给出续读 offset=10: %v", first)
	}

	second, err := m.viewNote([]float64{float64(id)}, false, 10, 0)
	if err != nil {
		t.Fatalf("续读 view 意外错误: %v", err)
	}
	if !strings.Contains(second, "ABCDEFGHIJ") {
		t.Errorf("续读段应包含后 10 字符: %v", second)
	}
	if strings.Contains(second, "（内容共") {
		t.Errorf("续读到末尾不应再带截断提示: %v", second)
	}
}

// TestBrowseNotesViewExplicitLength 验证显式 length 分段读取的边界。
func TestBrowseNotesViewExplicitLength(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	id := seedBrowseNote(t, db, "len", "1234567890ABCDEFGHIJ")

	r1, err := m.viewNote([]float64{float64(id)}, false, 0, 5)
	if err != nil {
		t.Fatalf("length=5 首段错误: %v", err)
	}
	if !strings.Contains(r1, "12345") || !strings.Contains(r1, "已显示前 5 字符") {
		t.Errorf("显式 length 首段错误: %v", r1)
	}
	r2, err := m.viewNote([]float64{float64(id)}, false, 5, 5)
	if err != nil {
		t.Fatalf("length=5 第二段错误: %v", err)
	}
	if !strings.Contains(r2, "67890") {
		t.Errorf("显式 length 第二段应为 56789 之后: %v", r2)
	}
}

// TestBrowseNotesViewOffsetOutOfRange 验证 offset 越界返回「已全部读取完毕」。
func TestBrowseNotesViewOffsetOutOfRange(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	id := seedBrowseNote(t, db, "bound", "abcdefghij") // 10 字符
	for _, off := range []int{10, 50} {
		_, err := m.viewNote([]float64{float64(id)}, false, off, 0)
		if err == nil || !strings.Contains(err.Error(), "已全部读取完毕") {
			t.Errorf("offset=%d 应报越界错误, got: %v", off, err)
		}
	}
}

// TestBrowseNotesViewEmpty 验证空笔记：offset=0 正常返回空内容，offset>0 报越界（回归复现 panic 修复）。
func TestBrowseNotesViewEmpty(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	id := seedBrowseNote(t, db, "空笔记", "")

	r, err := m.viewNote([]float64{float64(id)}, false, 0, 0)
	if err != nil {
		t.Fatalf("空笔记 offset=0 应成功, got: %v", err)
	}
	if !strings.Contains(r, "笔记 #"+strconv.Itoa(int(id))+" 内容：") {
		t.Errorf("空笔记应返回笔记头: %v", r)
	}
	if _, err := m.viewNote([]float64{float64(id)}, false, 5, 0); err == nil ||
		!strings.Contains(err.Error(), "已全部读取完毕") {
		t.Errorf("空笔记 offset>0 应报越界, got: %v", err)
	}
}

// TestBrowseNotesViewLineNumberContinuity 验证 line_numbers 分段续读的行号全局连续：
// 首段 offset=0 起始行 1，续读 offset=4 时应从其行号继续（而非始终从 1 开始）。
func TestBrowseNotesViewLineNumberContinuity(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	id := seedBrowseNote(t, db, "line", "a\nb\nc\nd") // 7 字符，4 行

	first, err := m.viewNote([]float64{float64(id)}, true, 0, 4)
	if err != nil {
		t.Fatalf("行号首段错误: %v", err)
	}
	if !strings.Contains(first, "行 1: a") || !strings.Contains(first, "行 2: b") {
		t.Errorf("首段应含行 1/2 前缀: %v", first)
	}

	second, err := m.viewNote([]float64{float64(id)}, true, 4, 3)
	if err != nil {
		t.Fatalf("行号续读错误: %v", err)
	}
	if !strings.Contains(second, "行 3: c") || !strings.Contains(second, "行 4: d") {
		t.Errorf("续读应含行 3/4（见读取到行 1/2 之后）: %v", second)
	}
}

// TestBrowseNotesViewInvalidIDs 验证 ids 缺失 / 多 id 的拒绝分支。
func TestBrowseNotesViewInvalidIDs(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	idA := seedBrowseNote(t, db, "A", "a")
	_ = seedBrowseNote(t, db, "B", "b")

	if _, err := m.viewNote([]float64{}, false, 0, 0); err == nil ||
		!strings.Contains(err.Error(), "缺少有效的 ids") {
		t.Errorf("空 ids 应报缺少 ids: %v", err)
	}
	if _, err := m.viewNote([]float64{float64(idA), float64(idA + 1)}, false, 0, 0); err == nil ||
		!strings.Contains(err.Error(), "只支持单条") {
		t.Errorf("多 ids 应报只支持单条: %v", err)
	}
}

// TestBrowseNotesListKeywordFilter 验证 list 关键字过滤与计数。
func TestBrowseNotesListKeywordFilter(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	idA := seedBrowseNote(t, db, "会议安排", "讨论排期")
	seedBrowseNote(t, db, "购物清单", "牛奶鸡蛋")

	r, err := m.listNotes("会议", 1, 10, 0, "updated_at", "", "", nil)
	if err != nil {
		t.Fatalf("list 错误: %v", err)
	}
	if !strings.Contains(r, "找到包含「会议」的笔记 1 条") {
		t.Errorf("应标识命中 1 条: %v", r)
	}
	if !strings.Contains(r, "["+strconv.Itoa(int(idA))+"] 会议安排") {
		t.Errorf("应包含[%d] 会议安排: %v", idA, r)
	}
	if strings.Contains(r, "购物清单") {
		t.Errorf("不应包含无关笔记: %v", r)
	}
}

// TestBrowseNotesListPagination 验证分页计数 / 页码超出。
func TestBrowseNotesListPagination(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	for i := 1; i <= 6; i++ {
		seedBrowseNote(t, db, "笔记"+strconv.Itoa(i), "")
	}

	r1, err := m.listNotes("", 1, 4, 0, "updated_at", "", "", nil)
	if err != nil || !strings.Contains(r1, "笔记共 6 条，第 1/2 页，本页 4 条") {
		t.Errorf("第1页计数错误 (%v): %v", err, r1)
	}
	r2, _ := m.listNotes("", 2, 4, 0, "updated_at", "", "", nil)
	if !strings.Contains(r2, "第 2/2 页，本页 2 条") {
		t.Errorf("第2页计数错误: %v", r2)
	}
	r3, _ := m.listNotes("", 3, 4, 0, "updated_at", "", "", nil)
	if !strings.Contains(r3, "第 3 页超出范围") {
		t.Errorf("页码超出应提示: %v", r3)
	}
}

// TestBrowseNotesListInvalidSort 验证非法 sort_by 校验。
func TestBrowseNotesListInvalidSort(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	seedBrowseNote(t, db, "x", "y")
	if _, err := m.listNotes("", 1, 10, 0, "bogus", "", "", nil); err == nil ||
		!strings.Contains(err.Error(), "参数非法 sort_by") {
		t.Errorf("非法 sort_by 应报错: %v", err)
	}
}

// TestBrowseNotesListPinnedAndTagFilter 验证置顶标识展示与标签 AND 过滤。
func TestBrowseNotesListPinnedAndTagFilter(t *testing.T) {
	m, db := newBrowseNotesTestService(t)
	idA := seedBrowseNote(t, db, "待办A", "内容A")
	idB := seedBrowseNote(t, db, "待办B", "内容B")

	// 置顶 A，list 应展示 📌
	var a models.Note
	if err := db.First(&a, idA).Error; err != nil {
		t.Fatalf("读取笔记A失败: %v", err)
	}
	if err := db.Model(&a).Update("pinned", true).Error; err != nil {
		t.Fatalf("置顶失败: %v", err)
	}
	r, err := m.listNotes("待办", 1, 10, 0, "updated_at", "", "", nil)
	if err != nil || !strings.Contains(r, "📌") {
		t.Errorf("置顶笔记应带📌 (err=%v): %v", err, r)
	}

	// 标签 AND 过滤：仅给 A 打标签 work，过滤后只出 A
	tag := models.Tag{Name: "work"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("创建标签失败: %v", err)
	}
	var a2 models.Note
	if err := db.First(&a2, idA).Error; err != nil {
		t.Fatalf("读取A失败: %v", err)
	}
	if err := db.Model(&a2).Association("Tags").Append(&tag); err != nil {
		t.Fatalf("打标签失败: %v", err)
	}
	rt, err := m.listNotes("", 1, 10, 0, "updated_at", "", "", []float64{float64(tag.ID)})
	if err != nil {
		t.Fatalf("标签过滤 list 错误: %v", err)
	}
	if !strings.Contains(rt, "待办A") || !strings.Contains(rt, "标签 work") {
		t.Errorf("标签过滤应包含待办A 及其标签: %v", rt)
	}
	if strings.Contains(rt, "["+strconv.Itoa(int(idB))+"]") {
		t.Errorf("标签过滤不应包含未打标签的待办B: %v", rt)
	}
}
