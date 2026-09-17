package tools

// 本文件覆盖 manage_note / manage_notebook / manage_tag / manage_todo 四个管理工具
// 写操作审批接入的单元测试：批准放行、拒绝不落库、Approver 缺失 fail-fast、
// 裸工具放行、create 豁免与 manage_note 的 critical 分级。
// 工具经真实 Service（内存 SQLite 单连接）构造，拒绝场景直接查库验证"不落库"。

import (
	"context"
	"errors"
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

// rejectApprover 模拟拒绝审批的审批器：记录收到的 toolName/summary/critical，
// RequestApproval 固定返回预设错误，用于验证写操作被拒绝后不落库。
type rejectApprover struct {
	gotTool     string
	gotSum      string
	gotCritical bool
	err         error
}

// RequestApproval 实现 Approver 接口：记录参数并返回预设错误。
func (m *rejectApprover) RequestApproval(_ context.Context, toolName, summary string, critical bool) error {
	m.gotTool = toolName
	m.gotSum = summary
	m.gotCritical = critical
	return m.err
}

// newApprovalTestLogger 构造测试用 fastlog Logger（写入临时目录，避免污染真实日志）。
func newApprovalTestLogger(t *testing.T) *fastlog.Logger {
	t.Helper()
	return fastlog.New(fastlog.Prod(filepath.Join(t.TempDir(), "test.log")))
}

// newApprovalTestDB 打开内存 SQLite（单连接）并迁移管理工具涉及的全部表。
func newApprovalTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&models.Note{}, &models.Tag{}, &models.Notebook{}, &models.Todo{}, &models.Setting{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// newManageNoteApprovalTool 构造 manageNoteTool（真实 NoteService/TagService/SettingService，
// 内存 SQLite 支撑），返回工具与 db 供断言落库状态。
func newManageNoteApprovalTool(t *testing.T) (*manageNoteTool, *gorm.DB) {
	t.Helper()
	db := newApprovalTestDB(t)
	logger := newApprovalTestLogger(t)
	setting := services.NewSettingService(db)
	note := services.NewNoteService(db, setting, logger)
	tag := services.NewTagService(db, logger)
	return &manageNoteTool{note: note, tag: tag, setting: setting, ctx: nil}, db
}

// seedApprovalNote 插入一条笔记并返回其 ID。
func seedApprovalNote(t *testing.T, db *gorm.DB, title, content string) uint {
	t.Helper()
	n := models.Note{Title: title, Content: content}
	if err := db.Create(&n).Error; err != nil {
		t.Fatalf("插入笔记(%q)失败: %v", title, err)
	}
	return n.ID
}

// TestManageNoteApproval 覆盖 manage_note 写操作审批接入：批准放行、拒绝不落库、
// Approver 缺失 fail-fast、裸工具放行、create 豁免与 critical 分级。
func TestManageNoteApproval(t *testing.T) {
	// 批准放行：Approver 返回 nil 时写操作正常执行并落库，返回成功文案
	t.Run("批准放行（update）", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		id := seedApprovalNote(t, db, "原标题", "内容")
		mock := &mockApprover{}
		m.ctx = &Context{Approver: mock}

		out, err := m.InvokableRun(context.Background(), `{"action":"update","ids":[`+strconv.Itoa(int(id))+`],"title":"新标题"}`)
		if err != nil {
			t.Fatalf("批准后更新失败: %v", err)
		}
		if !strings.Contains(out, "已更新") {
			t.Errorf("返回应含已更新文案，实际: %q", out)
		}
		var n models.Note
		if err := db.First(&n, id).Error; err != nil {
			t.Fatalf("读取笔记失败: %v", err)
		}
		if n.Title != "新标题" {
			t.Errorf("批准后标题 = %q, want 新标题", n.Title)
		}
		if mock.gotTool != "manage_note" {
			t.Errorf("审批 toolName = %q, want manage_note", mock.gotTool)
		}
		if !strings.Contains(mock.gotSum, "更新笔记 #"+strconv.Itoa(int(id))) {
			t.Errorf("审批摘要应含动作与笔记编号，实际: %q", mock.gotSum)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且写操作不落库
	t.Run("拒绝不落库（update）", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		id := seedApprovalNote(t, db, "原标题", "内容")
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m.ctx = &Context{Approver: rej}

		_, err := m.InvokableRun(context.Background(), `{"action":"update","ids":[`+strconv.Itoa(int(id))+`],"title":"篡改标题"}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		var n models.Note
		if err := db.First(&n, id).Error; err != nil {
			t.Fatalf("读取笔记失败: %v", err)
		}
		if n.Title != "原标题" {
			t.Errorf("拒绝后标题不应变更 = %q, want 原标题", n.Title)
		}
	})

	// Approver 缺失 fail-fast：ctx 非 nil 但 Approver 为 nil 时报"未配置审批机制"
	t.Run("Approver 缺失 fail-fast", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		id := seedApprovalNote(t, db, "原标题", "内容")
		m.ctx = &Context{} // 非 nil 但缺 Approver：装配遗漏应 fail-fast

		_, err := m.InvokableRun(context.Background(), `{"action":"update","ids":[`+strconv.Itoa(int(id))+`],"title":"x"}`)
		if err == nil {
			t.Fatal("Approver 缺失应报错")
		}
		if !strings.Contains(err.Error(), "未配置审批机制") {
			t.Errorf("错误应含未配置审批机制，实际: %v", err)
		}
		var n models.Note
		if err := db.First(&n, id).Error; err != nil {
			t.Fatalf("读取笔记失败: %v", err)
		}
		if n.Title != "原标题" {
			t.Errorf("fail-fast 后标题不应变更 = %q, want 原标题", n.Title)
		}
	})

	// 裸工具放行：ctx 为 nil（无审批机制）时写操作直接执行，不 panic、正常返回
	t.Run("裸工具放行（ctx 为 nil）", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		id := seedApprovalNote(t, db, "原标题", "内容")
		// m.ctx 保持 nil：视为放行，写操作直接执行
		out, err := m.InvokableRun(context.Background(), `{"action":"update","ids":[`+strconv.Itoa(int(id))+`],"title":"裸工具标题"}`)
		if err != nil {
			t.Fatalf("裸工具应放行，实际错误: %v", err)
		}
		if !strings.Contains(out, "已更新") {
			t.Errorf("返回应含已更新文案，实际: %q", out)
		}
		var n models.Note
		if err := db.First(&n, id).Error; err != nil {
			t.Fatalf("读取笔记失败: %v", err)
		}
		if n.Title != "裸工具标题" {
			t.Errorf("裸工具放行后标题 = %q, want 裸工具标题", n.Title)
		}
	})

	// create 豁免：action=create 时 Approver 不被调用
	t.Run("create 豁免审批", func(t *testing.T) {
		m, _ := newManageNoteApprovalTool(t)
		mock := &mockApprover{}
		m.ctx = &Context{Approver: mock}

		out, err := m.InvokableRun(context.Background(), `{"action":"create","title":"新笔记","content":"正文"}`)
		if err != nil {
			t.Fatalf("create 失败: %v", err)
		}
		if !strings.Contains(out, "已创建笔记") {
			t.Errorf("返回应含已创建文案，实际: %q", out)
		}
		if mock.gotTool != "" {
			t.Errorf("create 不应触发审批，实际 toolName = %q", mock.gotTool)
		}
	})

	// critical 分级：edit 恒 true；批量 move（ids>1）为 true；单条 pin / update 为 false
	t.Run("critical 分级", func(t *testing.T) {
		t.Run("edit 恒为 true", func(t *testing.T) {
			m, db := newManageNoteApprovalTool(t)
			id := seedApprovalNote(t, db, "标题", "这是内容")
			mock := &mockApprover{}
			m.ctx = &Context{Approver: mock}
			if _, err := m.InvokableRun(context.Background(), `{"action":"edit","ids":[`+strconv.Itoa(int(id))+`],"find":"这是","replace":"那是"}`); err != nil {
				t.Fatalf("edit 批准后执行失败: %v", err)
			}
			if !mock.gotCritical {
				t.Error("edit 审批 critical 应为 true")
			}
		})
		t.Run("批量 move（2 篇）为 true", func(t *testing.T) {
			m, db := newManageNoteApprovalTool(t)
			idA := seedApprovalNote(t, db, "A", "a")
			idB := seedApprovalNote(t, db, "B", "b")
			target := models.Notebook{Name: "目标本"}
			if err := db.Create(&target).Error; err != nil {
				t.Fatalf("创建目标笔记本失败: %v", err)
			}
			mock := &mockApprover{}
			m.ctx = &Context{Approver: mock}
			args := `{"action":"move","ids":[` + strconv.Itoa(int(idA)) + `,` + strconv.Itoa(int(idB)) + `],"notebook_id":` + strconv.Itoa(int(target.ID)) + `}`
			if _, err := m.InvokableRun(context.Background(), args); err != nil {
				t.Fatalf("批量 move 批准后执行失败: %v", err)
			}
			if !mock.gotCritical {
				t.Error("批量 move 审批 critical 应为 true")
			}
		})
		t.Run("单条 pin 为 false", func(t *testing.T) {
			m, db := newManageNoteApprovalTool(t)
			id := seedApprovalNote(t, db, "标题", "内容")
			mock := &mockApprover{}
			m.ctx = &Context{Approver: mock}
			if _, err := m.InvokableRun(context.Background(), `{"action":"pin","ids":[`+strconv.Itoa(int(id))+`]}`); err != nil {
				t.Fatalf("pin 批准后执行失败: %v", err)
			}
			if mock.gotCritical {
				t.Error("单条 pin 审批 critical 应为 false")
			}
		})
		t.Run("update 为 false", func(t *testing.T) {
			m, db := newManageNoteApprovalTool(t)
			id := seedApprovalNote(t, db, "原标题", "内容")
			mock := &mockApprover{}
			m.ctx = &Context{Approver: mock}
			if _, err := m.InvokableRun(context.Background(), `{"action":"update","ids":[`+strconv.Itoa(int(id))+`],"title":"x"}`); err != nil {
				t.Fatalf("update 批准后执行失败: %v", err)
			}
			if mock.gotCritical {
				t.Error("update 审批 critical 应为 false")
			}
		})
		t.Run("remove_tag 批量（2 篇）为 true", func(t *testing.T) {
			m, db := newManageNoteApprovalTool(t)
			idA := seedApprovalNote(t, db, "A", "a")
			idB := seedApprovalNote(t, db, "B", "b")
			tag := models.Tag{Name: "标签"}
			if err := db.Create(&tag).Error; err != nil {
				t.Fatalf("创建标签失败: %v", err)
			}
			mock := &mockApprover{}
			m.ctx = &Context{Approver: mock}
			args := `{"action":"remove_tag","ids":[` + strconv.Itoa(int(idA)) + `,` + strconv.Itoa(int(idB)) + `],"tag_id":` + strconv.Itoa(int(tag.ID)) + `}`
			if _, err := m.InvokableRun(context.Background(), args); err != nil {
				t.Fatalf("remove_tag 批量批准后执行失败: %v", err)
			}
			if !mock.gotCritical {
				t.Error("remove_tag 批量审批 critical 应为 true")
			}
		})
		t.Run("单条 move 为 false", func(t *testing.T) {
			m, db := newManageNoteApprovalTool(t)
			id := seedApprovalNote(t, db, "标题", "内容")
			target := models.Notebook{Name: "目标本"}
			if err := db.Create(&target).Error; err != nil {
				t.Fatalf("创建目标笔记本失败: %v", err)
			}
			mock := &mockApprover{}
			m.ctx = &Context{Approver: mock}
			args := `{"action":"move","ids":[` + strconv.Itoa(int(id)) + `],"notebook_id":` + strconv.Itoa(int(target.ID)) + `}`
			if _, err := m.InvokableRun(context.Background(), args); err != nil {
				t.Fatalf("单条 move 批准后执行失败: %v", err)
			}
			if mock.gotCritical {
				t.Error("单条 move 审批 critical 应为 false")
			}
		})
	})
}

// TestManageNotebookApproval 覆盖 manage_notebook 写操作审批：批准放行（rename）与 create 豁免。
func TestManageNotebookApproval(t *testing.T) {
	// 批准放行：rename 经 Approver 批准后正常执行并落库
	t.Run("批准放行（rename）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		// 先建默认笔记本（id=1，服务层禁止重命名），再建第二个用于 rename
		if err := db.Create(&models.Notebook{Name: "默认笔记本"}).Error; err != nil {
			t.Fatalf("创建默认笔记本失败: %v", err)
		}
		nb := models.Notebook{Name: "旧名"}
		if err := db.Create(&nb).Error; err != nil {
			t.Fatalf("创建笔记本失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"rename","id":`+strconv.Itoa(int(nb.ID))+`,"name":"新名"}`)
		if err != nil {
			t.Fatalf("批准后重命名失败: %v", err)
		}
		if !strings.Contains(out, "已重命名") {
			t.Errorf("返回应含已重命名文案，实际: %q", out)
		}
		var got models.Notebook
		if err := db.First(&got, nb.ID).Error; err != nil {
			t.Fatalf("读取笔记本失败: %v", err)
		}
		if got.Name != "新名" {
			t.Errorf("重命名后名称 = %q, want 新名", got.Name)
		}
		if mock.gotTool != "manage_notebook" {
			t.Errorf("审批 toolName = %q, want manage_notebook", mock.gotTool)
		}
	})

	// create 豁免：action=create 时 Approver 不被调用
	t.Run("create 豁免审批", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		mock := &mockApprover{}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"create","name":"新笔记本"}`)
		if err != nil {
			t.Fatalf("create 失败: %v", err)
		}
		if !strings.Contains(out, "已创建笔记本") {
			t.Errorf("返回应含已创建文案，实际: %q", out)
		}
		if mock.gotTool != "" {
			t.Errorf("create 不应触发审批，实际 toolName = %q", mock.gotTool)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且写操作不落库
	t.Run("拒绝不落库（rename）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		// 先建默认笔记本（id=1，服务层禁止重命名），再建第二个用于 rename
		if err := db.Create(&models.Notebook{Name: "默认笔记本"}).Error; err != nil {
			t.Fatalf("创建默认笔记本失败: %v", err)
		}
		nb := models.Notebook{Name: "旧名"}
		if err := db.Create(&nb).Error; err != nil {
			t.Fatalf("创建笔记本失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"rename","id":`+strconv.Itoa(int(nb.ID))+`,"name":"篡改名"}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		var got models.Notebook
		if err := db.First(&got, nb.ID).Error; err != nil {
			t.Fatalf("读取笔记本失败: %v", err)
		}
		if got.Name != "旧名" {
			t.Errorf("拒绝后名称不应变更 = %q, want 旧名", got.Name)
		}
	})
}

// TestManageTagApproval 覆盖 manage_tag 写操作审批：批准放行（update）与 create 豁免。
func TestManageTagApproval(t *testing.T) {
	// 批准放行：update 经 Approver 批准后正常执行并落库
	t.Run("批准放行（update）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTagService(db, logger)
		tag := models.Tag{Name: "旧标签"}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatalf("创建标签失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageTagTool{tag: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"update","id":`+strconv.Itoa(int(tag.ID))+`,"name":"新标签"}`)
		if err != nil {
			t.Fatalf("批准后更新标签失败: %v", err)
		}
		if !strings.Contains(out, "已更新标签") {
			t.Errorf("返回应含已更新文案，实际: %q", out)
		}
		var got models.Tag
		if err := db.First(&got, tag.ID).Error; err != nil {
			t.Fatalf("读取标签失败: %v", err)
		}
		if got.Name != "新标签" {
			t.Errorf("更新后标签名 = %q, want 新标签", got.Name)
		}
		if mock.gotTool != "manage_tag" {
			t.Errorf("审批 toolName = %q, want manage_tag", mock.gotTool)
		}
	})

	// create 豁免：action=create 时 Approver 不被调用
	t.Run("create 豁免审批", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTagService(db, logger)
		mock := &mockApprover{}
		m := &manageTagTool{tag: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"create","name":"新标签"}`)
		if err != nil {
			t.Fatalf("create 失败: %v", err)
		}
		if !strings.Contains(out, "已创建标签") {
			t.Errorf("返回应含已创建文案，实际: %q", out)
		}
		if mock.gotTool != "" {
			t.Errorf("create 不应触发审批，实际 toolName = %q", mock.gotTool)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且写操作不落库
	t.Run("拒绝不落库（update）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTagService(db, logger)
		tag := models.Tag{Name: "旧标签"}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatalf("创建标签失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageTagTool{tag: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"update","id":`+strconv.Itoa(int(tag.ID))+`,"name":"篡改标签"}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		var got models.Tag
		if err := db.First(&got, tag.ID).Error; err != nil {
			t.Fatalf("读取标签失败: %v", err)
		}
		if got.Name != "旧标签" {
			t.Errorf("拒绝后标签名不应变更 = %q, want 旧标签", got.Name)
		}
	})
}

// TestManageTodoApproval 覆盖 manage_todo 写操作审批：批准放行（toggle）与 create 豁免。
func TestManageTodoApproval(t *testing.T) {
	// 批准放行：toggle 经 Approver 批准后正常执行并落库
	t.Run("批准放行（toggle）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		todo := models.Todo{Text: "买牛奶"}
		if err := db.Create(&todo).Error; err != nil {
			t.Fatalf("创建待办失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"toggle","id":`+strconv.Itoa(int(todo.ID))+`}`)
		if err != nil {
			t.Fatalf("批准后勾选失败: %v", err)
		}
		if !strings.Contains(out, "已标记为完成") {
			t.Errorf("返回应含完成文案，实际: %q", out)
		}
		var got models.Todo
		if err := db.First(&got, todo.ID).Error; err != nil {
			t.Fatalf("读取待办失败: %v", err)
		}
		if !got.Done {
			t.Error("批准后待办应为完成状态")
		}
		if mock.gotTool != "manage_todo" {
			t.Errorf("审批 toolName = %q, want manage_todo", mock.gotTool)
		}
	})

	// create 豁免：action=create 时 Approver 不被调用
	t.Run("create 豁免审批", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		mock := &mockApprover{}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"create","text":"新待办"}`)
		if err != nil {
			t.Fatalf("create 失败: %v", err)
		}
		if !strings.Contains(out, "已创建待办") {
			t.Errorf("返回应含已创建文案，实际: %q", out)
		}
		if mock.gotTool != "" {
			t.Errorf("create 不应触发审批，实际 toolName = %q", mock.gotTool)
		}
	})

	// 批准放行：update 经 Approver 批准后正常执行并落库（源码先校验 text 非空）
	t.Run("批准放行（update）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		todo := models.Todo{Text: "买牛奶"}
		if err := db.Create(&todo).Error; err != nil {
			t.Fatalf("创建待办失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"update","id":`+strconv.Itoa(int(todo.ID))+`,"text":"买两瓶牛奶"}`)
		if err != nil {
			t.Fatalf("批准后更新文本失败: %v", err)
		}
		if !strings.Contains(out, "已更新待办") {
			t.Errorf("返回应含更新文案，实际: %q", out)
		}
		var got models.Todo
		if err := db.First(&got, todo.ID).Error; err != nil {
			t.Fatalf("读取待办失败: %v", err)
		}
		if got.Text != "买两瓶牛奶" {
			t.Errorf("批准后文本 = %q, want 买两瓶牛奶", got.Text)
		}
		if mock.gotTool != "manage_todo" {
			t.Errorf("审批 toolName = %q, want manage_todo", mock.gotTool)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且写操作不落库（toggle 不改 Done 状态）
	t.Run("拒绝不落库（toggle）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		todo := models.Todo{Text: "买牛奶"}
		if err := db.Create(&todo).Error; err != nil {
			t.Fatalf("创建待办失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"toggle","id":`+strconv.Itoa(int(todo.ID))+`}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		var got models.Todo
		if err := db.First(&got, todo.ID).Error; err != nil {
			t.Fatalf("读取待办失败: %v", err)
		}
		if got.Done {
			t.Error("拒绝后待办不应变为完成状态")
		}
	})
}
