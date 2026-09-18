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

// TestManageNoteDeleteAction 覆盖 manage_note delete 动作：审批参数（critical=true）、
// 拒绝不落库、批准后软删落库（deleted_at 置位，回收站可查）与批量删除。
func TestManageNoteDeleteAction(t *testing.T) {
	// 批准放行：单条 delete 经 Approver 批准后软删，deleted_at 置位（Unscoped 可查）
	t.Run("批准放行（单条软删）", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		id := seedApprovalNote(t, db, "待删笔记", "内容")
		mock := &mockApprover{}
		m.ctx = &Context{Approver: mock}

		out, err := m.InvokableRun(context.Background(), `{"action":"delete","ids":[`+strconv.Itoa(int(id))+`]}`)
		if err != nil {
			t.Fatalf("批准后删除失败: %v", err)
		}
		if !strings.Contains(out, "已删除") || !strings.Contains(out, "回收站") {
			t.Errorf("返回应含删除与回收站文案，实际: %q", out)
		}
		// 普通查询不可见（软删），Unscoped 查询可见且 deleted_at 置位
		if err := db.First(&models.Note{}, id).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("软删后普通查询应不可见，实际错误: %v", err)
		}
		var n models.Note
		if err := db.Unscoped().First(&n, id).Error; err != nil {
			t.Fatalf("Unscoped 读取笔记失败: %v", err)
		}
		if !n.DeletedAt.Valid {
			t.Error("批准后 deleted_at 应置位（回收站可见）")
		}
		if mock.gotTool != "manage_note" {
			t.Errorf("审批 toolName = %q, want manage_note", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("delete 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "删除笔记 #"+strconv.Itoa(int(id))) {
			t.Errorf("审批摘要应含删除动作与笔记编号，实际: %q", mock.gotSum)
		}
	})

	// 批准放行：批量 delete 一次删两篇，全部 deleted_at 置位，摘要注明批量
	t.Run("批准放行（批量软删）", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		idA := seedApprovalNote(t, db, "A", "a")
		idB := seedApprovalNote(t, db, "B", "b")
		mock := &mockApprover{}
		m.ctx = &Context{Approver: mock}

		args := `{"action":"delete","ids":[` + strconv.Itoa(int(idA)) + `,` + strconv.Itoa(int(idB)) + `]}`
		out, err := m.InvokableRun(context.Background(), args)
		if err != nil {
			t.Fatalf("批准后批量删除失败: %v", err)
		}
		if !strings.Contains(out, "2 篇") {
			t.Errorf("返回应含删除条数，实际: %q", out)
		}
		if !mock.gotCritical {
			t.Error("批量 delete 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "批量删除 2 篇") {
			t.Errorf("审批摘要应含批量删除条数，实际: %q", mock.gotSum)
		}
		for _, id := range []uint{idA, idB} {
			var n models.Note
			if err := db.Unscoped().First(&n, id).Error; err != nil {
				t.Fatalf("Unscoped 读取笔记失败: %v", err)
			}
			if !n.DeletedAt.Valid {
				t.Errorf("笔记 #%d 的 deleted_at 应置位", id)
			}
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且笔记不落库（deleted_at 不置位）
	t.Run("拒绝不落库（delete）", func(t *testing.T) {
		m, db := newManageNoteApprovalTool(t)
		id := seedApprovalNote(t, db, "保留笔记", "内容")
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m.ctx = &Context{Approver: rej}

		_, err := m.InvokableRun(context.Background(), `{"action":"delete","ids":[`+strconv.Itoa(int(id))+`]}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		var n models.Note
		if err := db.Unscoped().First(&n, id).Error; err != nil {
			t.Fatalf("Unscoped 读取笔记失败: %v", err)
		}
		if n.DeletedAt.Valid {
			t.Error("拒绝后 deleted_at 不应置位")
		}
	})
}

// TestManageNotebookDeleteAction 覆盖 manage_notebook delete 动作：审批参数（critical=true）、
// 拒绝不落库、温和删除（其下笔记迁默认笔记本 id=1）、连带删除（with_notes=true 进回收站）
// 与默认笔记本保护（id=1 报错且不触发审批）。
func TestManageNotebookDeleteAction(t *testing.T) {
	// 批准放行（温和删除）：笔记本软删，其下笔记 notebook_id 迁为默认笔记本 1
	t.Run("批准放行（笔记迁默认笔记本）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		// 种子：默认笔记本（id=1）+ 待删笔记本 + 其下笔记
		if err := db.Create(&models.Notebook{Name: "默认笔记本"}).Error; err != nil {
			t.Fatalf("创建默认笔记本失败: %v", err)
		}
		nb := models.Notebook{Name: "待删本"}
		if err := db.Create(&nb).Error; err != nil {
			t.Fatalf("创建笔记本失败: %v", err)
		}
		note := models.Note{Title: "归属笔记", Content: "内容", NotebookID: nb.ID}
		if err := db.Create(&note).Error; err != nil {
			t.Fatalf("创建笔记失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(nb.ID))+`}`)
		if err != nil {
			t.Fatalf("批准后删除笔记本失败: %v", err)
		}
		if !strings.Contains(out, "已删除笔记本") {
			t.Errorf("返回应含已删除文案，实际: %q", out)
		}
		// 笔记本软删（deleted_at 置位）
		var gotNb models.Notebook
		if err := db.Unscoped().First(&gotNb, nb.ID).Error; err != nil {
			t.Fatalf("Unscoped 读取笔记本失败: %v", err)
		}
		if !gotNb.DeletedAt.Valid {
			t.Error("批准后笔记本应软删（deleted_at 置位）")
		}
		// 笔记本身未删，notebook_id 迁为默认笔记本 1
		var gotNote models.Note
		if err := db.First(&gotNote, note.ID).Error; err != nil {
			t.Fatalf("读取笔记失败: %v", err)
		}
		if gotNote.NotebookID != 1 {
			t.Errorf("笔记 notebook_id = %d, want 1（默认笔记本）", gotNote.NotebookID)
		}
		if mock.gotTool != "manage_notebook" {
			t.Errorf("审批 toolName = %q, want manage_notebook", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("delete 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "迁入默认笔记本") {
			t.Errorf("审批摘要应注明笔记去向，实际: %q", mock.gotSum)
		}
	})

	// 批准放行（连带删除）：with_notes=true 时笔记本软删，其下笔记移入回收站
	t.Run("批准放行（with_notes=true 进回收站）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		if err := db.Create(&models.Notebook{Name: "默认笔记本"}).Error; err != nil {
			t.Fatalf("创建默认笔记本失败: %v", err)
		}
		nb := models.Notebook{Name: "待删本"}
		if err := db.Create(&nb).Error; err != nil {
			t.Fatalf("创建笔记本失败: %v", err)
		}
		note := models.Note{Title: "连带笔记", Content: "内容", NotebookID: nb.ID}
		if err := db.Create(&note).Error; err != nil {
			t.Fatalf("创建笔记失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(nb.ID))+`,"with_notes":true}`)
		if err != nil {
			t.Fatalf("批准后连带删除失败: %v", err)
		}
		if !strings.Contains(out, "回收站") {
			t.Errorf("返回应含回收站文案，实际: %q", out)
		}
		// 笔记本软删
		var gotNb models.Notebook
		if err := db.Unscoped().First(&gotNb, nb.ID).Error; err != nil {
			t.Fatalf("Unscoped 读取笔记本失败: %v", err)
		}
		if !gotNb.DeletedAt.Valid {
			t.Error("批准后笔记本应软删（deleted_at 置位）")
		}
		// 笔记移入回收站（deleted_at 置位）
		var gotNote models.Note
		if err := db.Unscoped().First(&gotNote, note.ID).Error; err != nil {
			t.Fatalf("Unscoped 读取笔记失败: %v", err)
		}
		if !gotNote.DeletedAt.Valid {
			t.Error("with_notes=true 后笔记应进回收站（deleted_at 置位）")
		}
		if !mock.gotCritical {
			t.Error("delete 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "回收站") {
			t.Errorf("审批摘要应注明笔记去向（回收站），实际: %q", mock.gotSum)
		}
	})

	// 默认笔记本保护：id=1 删除直接报错，Approver 不被调用（不弹无效审批窗）
	t.Run("默认笔记本保护（id=1）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		if err := db.Create(&models.Notebook{Name: "默认笔记本"}).Error; err != nil {
			t.Fatalf("创建默认笔记本失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: mock}}

		_, err := m.InvokableRun(context.Background(), `{"action":"delete","id":1}`)
		if err == nil {
			t.Fatal("删除默认笔记本应报错")
		}
		if !strings.Contains(err.Error(), "默认笔记本不可删除") {
			t.Errorf("错误应含默认笔记本不可删除，实际: %v", err)
		}
		if mock.gotTool != "" {
			t.Errorf("默认笔记本删除不应触发审批，实际 toolName = %q", mock.gotTool)
		}
		// 默认笔记本不应被删除
		var gotNb models.Notebook
		if err := db.First(&gotNb, 1).Error; err != nil {
			t.Fatalf("默认笔记本应仍然存在: %v", err)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且笔记本与笔记均不落库
	t.Run("拒绝不落库（delete）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewNotebookService(db, logger)
		if err := db.Create(&models.Notebook{Name: "默认笔记本"}).Error; err != nil {
			t.Fatalf("创建默认笔记本失败: %v", err)
		}
		nb := models.Notebook{Name: "保留本"}
		if err := db.Create(&nb).Error; err != nil {
			t.Fatalf("创建笔记本失败: %v", err)
		}
		note := models.Note{Title: "归属笔记", Content: "内容", NotebookID: nb.ID}
		if err := db.Create(&note).Error; err != nil {
			t.Fatalf("创建笔记失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageNotebookTool{notebook: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(nb.ID))+`}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		// 笔记本仍在（deleted_at 未置位）、笔记归属未变
		var gotNb models.Notebook
		if err := db.First(&gotNb, nb.ID).Error; err != nil {
			t.Fatalf("读取笔记本失败: %v", err)
		}
		var gotNote models.Note
		if err := db.First(&gotNote, note.ID).Error; err != nil {
			t.Fatalf("读取笔记失败: %v", err)
		}
		if gotNote.NotebookID != nb.ID {
			t.Errorf("拒绝后笔记 notebook_id 不应变更 = %d, want %d", gotNote.NotebookID, nb.ID)
		}
	})
}

// TestManageTagDeleteAction 覆盖 manage_tag delete 动作：审批参数（critical=true）、
// 拒绝不落库、批准后真实删除（标签硬删，笔记内容不受影响）。
func TestManageTagDeleteAction(t *testing.T) {
	// 批准放行：delete 经 Approver 批准后真实删除标签（Tag 无软删字段，物理删除）
	t.Run("批准放行（delete）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTagService(db, logger)
		tag := models.Tag{Name: "待删标签"}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatalf("创建标签失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageTagTool{tag: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(tag.ID))+`}`)
		if err != nil {
			t.Fatalf("批准后删除标签失败: %v", err)
		}
		if !strings.Contains(out, "已删除标签") {
			t.Errorf("返回应含已删除文案，实际: %q", out)
		}
		// 标签已删除（查询不可见）
		if err := db.First(&models.Tag{}, tag.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("删除后查询标签应不可见，实际错误: %v", err)
		}
		if mock.gotTool != "manage_tag" {
			t.Errorf("审批 toolName = %q, want manage_tag", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("delete 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "删除标签 #"+strconv.Itoa(int(tag.ID))) {
			t.Errorf("审批摘要应含删除动作与标签编号，实际: %q", mock.gotSum)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且标签不落库
	t.Run("拒绝不落库（delete）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTagService(db, logger)
		tag := models.Tag{Name: "保留标签"}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatalf("创建标签失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageTagTool{tag: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(tag.ID))+`}`)
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
		if got.Name != "保留标签" {
			t.Errorf("拒绝后标签不应被删除，实际名称 = %q", got.Name)
		}
	})
}

// TestManageTodoDeleteAndClearAction 覆盖 manage_todo delete / clear 动作：审批参数（critical=true、
// 摘要注明不可恢复）、拒绝不落库、批准后硬删落库，clear 仅清已完成并返回清理条数。
func TestManageTodoDeleteAndClearAction(t *testing.T) {
	// 批准放行：delete 经 Approver 批准后硬删单条待办
	t.Run("批准放行（delete 单条）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		todo := models.Todo{Text: "买牛奶"}
		if err := db.Create(&todo).Error; err != nil {
			t.Fatalf("创建待办失败: %v", err)
		}
		mock := &mockApprover{}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(todo.ID))+`}`)
		if err != nil {
			t.Fatalf("批准后删除待办失败: %v", err)
		}
		if !strings.Contains(out, "已删除待办") {
			t.Errorf("返回应含已删除文案，实际: %q", out)
		}
		// 硬删：查询不可见
		if err := db.First(&models.Todo{}, todo.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("删除后查询待办应不可见，实际错误: %v", err)
		}
		if mock.gotTool != "manage_todo" {
			t.Errorf("审批 toolName = %q, want manage_todo", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("delete 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "不可恢复") {
			t.Errorf("审批摘要应注明不可恢复，实际: %q", mock.gotSum)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，且待办不落库
	t.Run("拒绝不落库（delete）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		todo := models.Todo{Text: "买牛奶"}
		if err := db.Create(&todo).Error; err != nil {
			t.Fatalf("创建待办失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"delete","id":`+strconv.Itoa(int(todo.ID))+`}`)
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
		if got.Text != "买牛奶" {
			t.Errorf("拒绝后待办不应被删除，实际文本 = %q", got.Text)
		}
	})

	// 批准放行：clear 仅清空已完成待办并返回清理条数，未完成待办不受影响
	t.Run("批准放行（clear 仅清已完成）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		// 种子：2 条已完成 + 2 条未完成
		for _, txt := range []string{"已完成A", "已完成B"} {
			if err := db.Create(&models.Todo{Text: txt, Done: true}).Error; err != nil {
				t.Fatalf("创建已完成待办失败: %v", err)
			}
		}
		for _, txt := range []string{"未完成A", "未完成B"} {
			if err := db.Create(&models.Todo{Text: txt, Done: false}).Error; err != nil {
				t.Fatalf("创建未完成待办失败: %v", err)
			}
		}
		mock := &mockApprover{}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: mock}}

		out, err := m.InvokableRun(context.Background(), `{"action":"clear"}`)
		if err != nil {
			t.Fatalf("批准后清空失败: %v", err)
		}
		if !strings.Contains(out, "2") {
			t.Errorf("返回应含清理条数 2，实际: %q", out)
		}
		// 已完成待办被硬删，未完成待办保留
		var doneCnt, activeCnt int64
		if err := db.Model(&models.Todo{}).Where("done = ?", true).Count(&doneCnt).Error; err != nil {
			t.Fatalf("统计已完成待办失败: %v", err)
		}
		if err := db.Model(&models.Todo{}).Where("done = ?", false).Count(&activeCnt).Error; err != nil {
			t.Fatalf("统计未完成待办失败: %v", err)
		}
		if doneCnt != 0 {
			t.Errorf("清空后已完成待办数 = %d, want 0", doneCnt)
		}
		if activeCnt != 2 {
			t.Errorf("清空后未完成待办数 = %d, want 2（不受影响）", activeCnt)
		}
		if mock.gotTool != "manage_todo" {
			t.Errorf("审批 toolName = %q, want manage_todo", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("clear 审批 critical 应为 true")
		}
		if !strings.Contains(mock.gotSum, "不可恢复") {
			t.Errorf("审批摘要应注明不可恢复，实际: %q", mock.gotSum)
		}
	})

	// 拒绝：Approver 返回错误时工具返回该错误，已完成与未完成待办均保留
	t.Run("拒绝不落库（clear）", func(t *testing.T) {
		db := newApprovalTestDB(t)
		logger := newApprovalTestLogger(t)
		svc := services.NewTodoService(db, logger)
		if err := db.Create(&models.Todo{Text: "已完成A", Done: true}).Error; err != nil {
			t.Fatalf("创建已完成待办失败: %v", err)
		}
		if err := db.Create(&models.Todo{Text: "未完成A", Done: false}).Error; err != nil {
			t.Fatalf("创建未完成待办失败: %v", err)
		}
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		m := &manageTodoTool{todo: svc, ctx: &Context{Approver: rej}}

		_, err := m.InvokableRun(context.Background(), `{"action":"clear"}`)
		if err == nil {
			t.Fatal("拒绝后应返回错误")
		}
		if !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Errorf("错误应为审批拒绝错误，实际: %v", err)
		}
		var cnt int64
		if err := db.Model(&models.Todo{}).Count(&cnt).Error; err != nil {
			t.Fatalf("统计待办失败: %v", err)
		}
		if cnt != 2 {
			t.Errorf("拒绝后待办总数 = %d, want 2（均保留）", cnt)
		}
	})
}
