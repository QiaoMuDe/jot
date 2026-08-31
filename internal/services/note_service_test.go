package services

import (
	"path/filepath"
	"strings"
	"testing"

	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TestBuildSortOrder 验证搜索/列表的 ORDER BY 子句构造逻辑
// 覆盖 3 个 sortBy 值 + 默认值 + 非法值
func TestBuildSortOrder(t *testing.T) {
	cases := []struct {
		name     string
		sortBy   string
		expected string
	}{
		{"updated_at 默认", "updated_at", "pinned DESC, updated_at DESC"},
		{"created_at", "created_at", "pinned DESC, created_at DESC"},
		{"title 升序", "title", "pinned DESC, title ASC"},
		{"空字符串回退默认", "", "pinned DESC, updated_at DESC"},
		{"非法值回退默认", "xxx_invalid", "pinned DESC, updated_at DESC"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildSortOrder(c.sortBy)
			if got != c.expected {
				t.Errorf("buildSortOrder(%q) = %q, want %q", c.sortBy, got, c.expected)
			}
		})
	}
}

// TestNextDuplicateTitle 验证副本标题生成：后缀、冲突递增序号、标题截断、空标题回退。
func TestNextDuplicateTitle(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		existing map[string]bool
		want     string
	}{
		{"首副本", "会议记录", nil, "会议记录 副本"},
		{"副本冲突递增到2", "会议记录", map[string]bool{"会议记录 副本": true}, "会议记录 副本 2"},
		{"副本2也冲突递增到3", "会议记录", map[string]bool{"会议记录 副本": true, "会议记录 副本 2": true}, "会议记录 副本 3"},
		{"首尾空格去除", "  会议记录  ", nil, "会议记录 副本"},
		{"空标题回退未命名", "   ", nil, "未命名 副本"},
		{"空标题冲突递增", "", map[string]bool{"未命名 副本": true}, "未命名 副本 2"},
		{"超长标题截断", strings.Repeat("长", 200), nil, strings.Repeat("长", 197) + " 副本"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exists := func(title string) bool { return c.existing[title] }
			got := nextDuplicateTitle(c.base, exists)
			if got != c.want {
				t.Errorf("nextDuplicateTitle(%q) = %q, want %q", c.base, got, c.want)
			}
			if len([]rune(got)) > noteTitleMaxRunes {
				t.Errorf("标题超长: %d runes > %d", len([]rune(got)), noteTitleMaxRunes)
			}
		})
	}
}

// TestTruncateTitleRunes 验证标题 rune 截断边界。
func TestTruncateTitleRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"未超限原样返回", "hello", 10, "hello"},
		{"恰好等于上限", "hello", 5, "hello"},
		{"多字节截断", "你好世界", 3, "你好世"},
		{"空串", "", 5, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := truncateTitleRunes(c.in, c.max); got != c.want {
				t.Errorf("truncateTitleRunes(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
			}
		})
	}
}

// TestBuildSearchSortOrder 验证搜索打分排序规则构造：
// 空关键词回退常规排序；非空关键词生成打分 CASE 表达式，参数顺序对应
// 完全相等 > 前缀 > 标题+内容都命中 > 仅标题 > 仅内容。
func TestBuildSearchSortOrder(t *testing.T) {
	t.Run("空关键词回退常规排序", func(t *testing.T) {
		got := buildSearchSortOrder("", "updated_at")
		s, ok := got.(string)
		if !ok {
			t.Fatalf("期望返回 string, 实际 %T", got)
		}
		if s != "pinned DESC, updated_at DESC" {
			t.Errorf("空关键词排序 = %q, want %q", s, "pinned DESC, updated_at DESC")
		}
	})

	t.Run("非空关键词返回打分表达式", func(t *testing.T) {
		got := buildSearchSortOrder("会议", "updated_at")
		orderBy, ok := got.(clause.OrderBy)
		if !ok {
			t.Fatalf("期望 clause.OrderBy, 实际 %T", got)
		}
		expr, ok := orderBy.Expression.(clause.Expr)
		if !ok {
			t.Fatalf("期望 Expression 为 clause.Expr, 实际 %T", orderBy.Expression)
		}
		for _, frag := range []string{"title LIKE", "content LIKE", "END DESC", "pinned DESC", "updated_at DESC"} {
			if !strings.Contains(expr.SQL, frag) {
				t.Errorf("SQL 缺少片段 %q: %s", frag, expr.SQL)
			}
		}
		if len(expr.Vars) != 6 {
			t.Fatalf("参数数量 = %d, want 6", len(expr.Vars))
		}
		want := []string{"会议", "会议%", "%会议%", "%会议%", "%会议%", "%会议%"}
		for i, w := range want {
			if expr.Vars[i] != w {
				t.Errorf("参数[%d] = %v, want %q", i, expr.Vars[i], w)
			}
		}
	})
}

// newSearchTestService 打开内存 SQLite（单连接）并迁移 Note/Tag，构造 NoteService
func newSearchTestService(t *testing.T) (*NoteService, *gorm.DB) {
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
	if err := db.AutoMigrate(&models.Note{}, &models.Tag{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	logger := fastlog.New(fastlog.Prod(filepath.Join(t.TempDir(), "test.log")))
	return NewNoteService(db, NewSettingService(db), logger), db
}

// createSearchTestNote 插入一条测试笔记并返回带 ID 的模型
func createSearchTestNote(t *testing.T, db *gorm.DB, title, content string) models.Note {
	t.Helper()
	n := models.Note{Title: title, Content: content}
	if err := db.Create(&n).Error; err != nil {
		t.Fatalf("插入笔记(%q)失败: %v", title, err)
	}
	return n
}

// TestSearchRelevanceOrdering 验证真实 SQLite 上搜索按相关性打分排序：
// 标题完全相等 > 标题前缀命中 > 标题+内容都命中 > 仅标题命中 > 仅内容命中，
// 不匹配的笔记不出现在结果中。
func TestSearchRelevanceOrdering(t *testing.T) {
	svc, db := newSearchTestService(t)

	// 故意打乱插入顺序，确保测试能区分"打分排序"与"插入顺序"（插入顺序不是打分顺序）
	contentOnly := createSearchTestNote(t, db, "工作总结", "提到会议讨论结果") // 10 分
	exact := createSearchTestNote(t, db, "会议", "随便内容")             // 50 分
	titleOnly := createSearchTestNote(t, db, "第一季度会议安排", "无相关内容")  // 25 分
	prefix := createSearchTestNote(t, db, "会议纪要", "随便内容")          // 40 分
	both := createSearchTestNote(t, db, "上周会议总结", "本次会议讨论了排期")     // 30 分
	createSearchTestNote(t, db, "购物清单", "牛奶面包")                    // 完全不匹配，不应出现

	notes, total, err := svc.Search("会议", 1, 20, "updated_at", "", "", nil)
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	wantOrder := []uint{exact.ID, prefix.ID, both.ID, titleOnly.ID, contentOnly.ID}
	if len(notes) != len(wantOrder) {
		t.Fatalf("返回 %d 条, want %d", len(notes), len(wantOrder))
	}
	for i, want := range wantOrder {
		if notes[i].ID != want {
			t.Errorf("顺序[%d] = %d, want %d", i, notes[i].ID, want)
		}
	}
}

// TestSearchRelevanceOrderingWithTagFilter 验证标签筛选下搜索仍按打分排序：
// 标签筛选仅做 AND 过滤（不参与排序），过滤后的结果按相关性打分排列。
func TestSearchRelevanceOrderingWithTagFilter(t *testing.T) {
	svc, db := newSearchTestService(t)

	work := models.Tag{Name: "工作"}
	personal := models.Tag{Name: "个人"}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("创建标签失败: %v", err)
	}
	if err := db.Create(&personal).Error; err != nil {
		t.Fatalf("创建标签失败: %v", err)
	}

	// 故意打乱插入顺序：both(30) 先插入，prefix(40) 最后插入
	both := createSearchTestNote(t, db, "上周会议总结", "本次会议讨论了排期")
	titleOnly := createSearchTestNote(t, db, "第一季度会议安排", "无相关内容")
	prefix := createSearchTestNote(t, db, "会议纪要", "随便内容")

	// 关联标签：prefix→工作；both→工作+个人；titleOnly→个人
	if err := db.Model(&prefix).Association("Tags").Append(&work); err != nil {
		t.Fatalf("关联标签失败: %v", err)
	}
	if err := db.Model(&both).Association("Tags").Append(&work, &personal); err != nil {
		t.Fatalf("关联标签失败: %v", err)
	}
	if err := db.Model(&titleOnly).Association("Tags").Append(&personal); err != nil {
		t.Fatalf("关联标签失败: %v", err)
	}

	// 筛选"工作"标签：仅返回包含该标签的笔记（AND 过滤），仍按打分排序
	notes, total, err := svc.Search("会议", 1, 20, "updated_at", "", "", []uint{work.ID})
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	wantOrder := []uint{prefix.ID, both.ID} // 前缀 40 > 标题+内容 30
	if len(notes) != len(wantOrder) {
		t.Fatalf("返回 %d 条, want %d", len(notes), len(wantOrder))
	}
	for i, want := range wantOrder {
		if notes[i].ID != want {
			t.Errorf("顺序[%d] = %d, want %d", i, notes[i].ID, want)
		}
	}
}

// TestSearchByNotebookRelevanceOrdering 验证笔记本内搜索同样按相关性打分排序，
// 且不影响其他笔记本的笔记
func TestSearchByNotebookRelevanceOrdering(t *testing.T) {
	svc, db := newSearchTestService(t)

	// 笔记本 1 内故意乱序插入（打分顺序 ≠ 插入顺序）
	contentOnly := createSearchTestNote(t, db, "工作总结", "提到会议讨论结果") // 10 分
	exact := createSearchTestNote(t, db, "会议", "随便内容")             // 50 分
	both := createSearchTestNote(t, db, "上周会议总结", "本次会议讨论了排期")     // 30 分
	for _, n := range []*models.Note{&contentOnly, &exact, &both} {
		if err := db.Model(n).Update("notebook_id", 1).Error; err != nil {
			t.Fatalf("设置笔记本失败: %v", err)
		}
	}
	// 笔记本 2 的干扰项：标题完全相等但不在目标笔记本内
	other := createSearchTestNote(t, db, "会议", "随便内容")
	if err := db.Model(&other).Update("notebook_id", 2).Error; err != nil {
		t.Fatalf("设置笔记本失败: %v", err)
	}

	notes, total, err := svc.SearchByNotebook("会议", 1, 20, 1, "updated_at", "", "", nil)
	if err != nil {
		t.Fatalf("SearchByNotebook 失败: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	wantOrder := []uint{exact.ID, both.ID, contentOnly.ID} // 50 > 30 > 10
	if len(notes) != len(wantOrder) {
		t.Fatalf("返回 %d 条, want %d", len(notes), len(wantOrder))
	}
	for i, want := range wantOrder {
		if notes[i].ID != want {
			t.Errorf("顺序[%d] = %d, want %d", i, notes[i].ID, want)
		}
	}
}

// TestEscapeLike 验证 LIKE 特殊字符 % _ \ 的转义
func TestEscapeLike(t *testing.T) {
	cases := []struct{ in, want string }{
		{"日志", "日志"},
		{"100%", `100\%`},
		{"a_b", `a\_b`},
		{`a\b`, `a\\b`},
		{"%_\\", `\%\_\\`},
	}
	for _, c := range cases {
		if got := escapeLike(c.in); got != c.want {
			t.Errorf("escapeLike(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSearchWildcardEscaping 验证含 % 关键词按字面匹配，不被当作 LIKE 通配符
func TestSearchWildcardEscaping(t *testing.T) {
	svc, db := newSearchTestService(t)

	literal := createSearchTestNote(t, db, "100%完成", "任务进度")
	createSearchTestNote(t, db, "100完成", "任务进度") // 无字面 %，不应匹配
	createSearchTestNote(t, db, "50%折扣", "购物信息") // 含 % 但不含 "100%"

	notes, total, err := svc.Search("100%", 1, 20, "updated_at", "", "", nil)
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(notes) != 1 || notes[0].ID != literal.ID {
		t.Errorf("应只返回字面匹配的笔记, got %+v", notes)
	}
}

// TestFindByTitleAndExt 验证按标题+后缀+笔记本查找已有笔记的逻辑
func TestFindByTitleAndExt(t *testing.T) {
	svc, db := newSearchTestService(t)

	// 创建测试笔记：标题=readme, 后缀=.md, 笔记本=1
	note := createSearchTestNote(t, db, "readme", "# Hello")
	if err := db.Model(&note).Update("notebook_id", 1).Error; err != nil {
		t.Fatalf("设置笔记本失败: %v", err)
	}
	if err := db.Model(&note).Update("file_ext", ".md").Error; err != nil {
		t.Fatalf("设置后缀失败: %v", err)
	}

	// 创建干扰项：同标题不同后缀
	干扰 := createSearchTestNote(t, db, "readme", "plain text")
	if err := db.Model(&干扰).Update("notebook_id", 1).Error; err != nil {
		t.Fatalf("设置笔记本失败: %v", err)
	}
	if err := db.Model(&干扰).Update("file_ext", ".txt").Error; err != nil {
		t.Fatalf("设置后缀失败: %v", err)
	}

	// 创建干扰项：同标题同后缀但不同笔记本
	其他笔记本 := createSearchTestNote(t, db, "readme", "# Other")
	if err := db.Model(&其他笔记本).Update("notebook_id", 2).Error; err != nil {
		t.Fatalf("设置笔记本失败: %v", err)
	}
	if err := db.Model(&其他笔记本).Update("file_ext", ".md").Error; err != nil {
		t.Fatalf("设置后缀失败: %v", err)
	}

	t.Run("有匹配", func(t *testing.T) {
		result, err := svc.FindByTitleAndExt("readme", ".md", 1)
		if err != nil {
			t.Fatalf("FindByTitleAndExt 失败: %v", err)
		}
		if result == nil {
			t.Fatal("应找到匹配笔记，结果为 nil")
		}
		if result.ID != note.ID {
			t.Errorf("匹配到错误的笔记: got ID=%d, want ID=%d", result.ID, note.ID)
		}
	})

	t.Run("不同后缀不匹配", func(t *testing.T) {
		result, err := svc.FindByTitleAndExt("readme", ".txt", 1)
		if err != nil {
			t.Fatalf("FindByTitleAndExt 失败: %v", err)
		}
		if result == nil {
			t.Fatal("应找到匹配笔记（.txt），结果为 nil")
		}
		if result.ID != 干扰.ID {
			t.Errorf("匹配到错误的笔记: got ID=%d, want ID=%d", result.ID, 干扰.ID)
		}
	})

	t.Run("不同笔记本不匹配", func(t *testing.T) {
		result, err := svc.FindByTitleAndExt("readme", ".md", 2)
		if err != nil {
			t.Fatalf("FindByTitleAndExt 失败: %v", err)
		}
		if result == nil {
			t.Fatal("应找到匹配笔记（笔记本2），结果为 nil")
		}
		if result.ID != 其他笔记本.ID {
			t.Errorf("匹配到错误的笔记: got ID=%d, want ID=%d", result.ID, 其他笔记本.ID)
		}
	})

	t.Run("无匹配返回nil", func(t *testing.T) {
		result, err := svc.FindByTitleAndExt("不存在的标题", ".md", 1)
		if err != nil {
			t.Fatalf("FindByTitleAndExt 失败: %v", err)
		}
		if result != nil {
			t.Errorf("不应找到匹配笔记，但返回了 ID=%d", result.ID)
		}
	})
}
