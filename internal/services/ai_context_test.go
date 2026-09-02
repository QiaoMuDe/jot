package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/MM-Q/fastlog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"jot/internal/models"
)

// newAIContextTestDB 打开内存 SQLite（单连接）并迁移 AI 会话相关表与设置表
func newAIContextTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.AISession{}, &models.AIMessage{}, &models.Setting{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// newAIContextTestService 构造 AIService（日志写入测试临时目录）
func newAIContextTestService(t *testing.T, db *gorm.DB) *AIService {
	t.Helper()
	logger := fastlog.New(fastlog.Prod(filepath.Join(t.TempDir(), "test.log")))
	return NewAIService(db, logger)
}

// asciiTokens 生成估算 token 数为 n 的英文内容（4 字符 ≈ 1 token）
func asciiTokens(n int) string {
	return strings.Repeat("a", n*4)
}

// msg 构造带 ID 的消息
func msg(id uint, role string, tokens int) Message {
	return Message{ID: id, Role: role, Content: asciiTokens(tokens)}
}

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{"两个中文字", "你好", 2},         // 2/1.5 = 1.34 → ceil 2
		{"四个英文字符", "abcd", 1},      // 4/4 = 1
		{"中英混合", "你好ab", 2},        // 1.34 + 0.5 = 1.84 → ceil 2
		{"空文本", "", 0},             //
		{"十个中文字", "一二三四五六七八九十", 7}, // 10/1.5 = 6.67 → ceil 7
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EstimateTokens(c.text); got != c.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", c.text, got, c.want)
			}
		})
	}
}

func TestSelectTailByTokenBudget(t *testing.T) {
	t.Run("预算内全保留", func(t *testing.T) {
		msgs := []Message{msg(1, "user", 10), msg(2, "assistant", 10), msg(3, "user", 10)}
		tail, start := SelectTailByTokenBudget(msgs, 100)
		if start != 0 || len(tail) != 3 {
			t.Fatalf("start=%d len(tail)=%d, want 0/3", start, len(tail))
		}
	})

	t.Run("超预算截断且轮次对齐 user", func(t *testing.T) {
		// 每条 10 token，预算 30：从尾部累计 a2(10)+u2(10)+a1(10)=30，
		// u1 再加 40>30 停止 → 边界落在 a1（assistant），对齐后从 u2 开始
		msgs := []Message{
			msg(1, "user", 10), msg(2, "assistant", 10),
			msg(3, "user", 10), msg(4, "assistant", 10),
		}
		tail, start := SelectTailByTokenBudget(msgs, 30)
		if start != 2 {
			t.Fatalf("start=%d, want 2", start)
		}
		if tail[0].Role != "user" {
			t.Fatalf("tail 首条 role=%s, want user", tail[0].Role)
		}
		if len(tail) != 2 || tail[0].ID != 3 || tail[1].ID != 4 {
			t.Fatalf("tail=%v, want [3 4]", tail)
		}
	})

	t.Run("单条超预算仍保留最后一条", func(t *testing.T) {
		msgs := []Message{msg(1, "user", 5), msg(2, "user", 100)}
		tail, start := SelectTailByTokenBudget(msgs, 10)
		if start != 1 || len(tail) != 1 || tail[0].ID != 2 {
			t.Fatalf("start=%d tail=%v, want 单条保留 ID=2", start, tail)
		}
	})
}

func TestSelectKeepTailByTokenBudget(t *testing.T) {
	// 6 条 × 10 token，keepRatio 0.5、预算 100 → keepBudget=50：
	// 从尾部累计 5 条恰好 50，第 6 条超限 → 边界落在 assistant（下标 1），对齐后保留下标 2 起的 4 条
	tail := []Message{
		msg(1, "user", 10), msg(2, "assistant", 10),
		msg(3, "user", 10), msg(4, "assistant", 10),
		msg(5, "user", 10), msg(6, "assistant", 10),
	}
	kept := SelectKeepTailByTokenBudget(tail, 100, CompactKeepRatio)
	if len(kept) != 4 || kept[0].ID != 3 || kept[0].Role != "user" {
		t.Fatalf("kept=%v, want 从 ID=3（user）起保留 4 条", kept)
	}
}

func TestLimitRegionByTokens(t *testing.T) {
	// 5 条 × 10 token，上限 25：末尾累计 10、20，第 3 条 30>25 停止 → 保留末尾 2 条
	region := []Message{
		msg(1, "user", 10), msg(2, "assistant", 10), msg(3, "user", 10),
		msg(4, "assistant", 10), msg(5, "user", 10),
	}
	got := limitRegionByTokens(region, 25)
	if len(got) != 2 || got[0].ID != 4 || got[1].ID != 5 {
		t.Fatalf("got=%v, want 末尾 2 条 [4 5]", got)
	}
}

// newSummaryMockServer 启动模拟 OpenAI 兼容 /chat/completions 端点，
// 按调用次数依次返回 contents 中的内容（循环使用）
func newSummaryMockServer(t *testing.T, contents []string) *httptest.Server {
	t.Helper()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		content := contents[callCount%len(contents)]
		callCount++
		resp := `{"id":"1","object":"chat.completion","created":1,"model":"test-model",` +
			`"choices":[{"index":0,"message":{"role":"assistant","content":"` + content + `"},"finish_reason":"stop"}]}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(server.Close)
	return server
}

// setupSummarySession 建会话 + 消息，并将 AI 配置指向 mock 服务
func setupSummarySession(t *testing.T, db *gorm.DB, serverURL string, msgs []Message) uint {
	t.Helper()
	session := models.AISession{Title: "测试会话"}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	for i, m := range msgs {
		row := models.AIMessage{
			SessionID: session.ID,
			Role:      m.Role,
			Content:   m.Content,
			Tokens:    0,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("创建消息失败: %v", err)
		}
		msgs[i].ID = row.ID
	}
	svc := NewSettingService(db)
	for k, v := range map[string]string{
		"ai_base_url": serverURL,
		"ai_api_key":  "test-key",
		"ai_model":    "test-model",
	} {
		if err := svc.Set(k, v); err != nil {
			t.Fatalf("写入设置 %s 失败: %v", k, err)
		}
	}
	return session.ID
}

func TestCompactSessionSummary(t *testing.T) {
	t.Run("首次摘要并持久化边界", func(t *testing.T) {
		db := newAIContextTestDB(t)
		server := newSummaryMockServer(t, []string{"摘要v1"})
		msgs := []Message{
			{Role: "user", Content: asciiTokens(10)},
			{Role: "assistant", Content: asciiTokens(10)},
			{Role: "user", Content: asciiTokens(10)},
		}
		sessionID := setupSummarySession(t, db, server.URL, msgs)

		svc := newAIContextTestService(t, db)
		boundary := msgs[2].ID
		summary, ok := svc.CompactSessionSummary(context.Background(), sessionID, msgs, boundary)
		if !ok || summary != "摘要v1" {
			t.Fatalf("ok=%v summary=%q, want true/摘要v1", ok, summary)
		}
		var session models.AISession
		if err := db.First(&session, sessionID).Error; err != nil {
			t.Fatalf("加载会话失败: %v", err)
		}
		if session.SummaryContent != "摘要v1" {
			t.Errorf("SummaryContent=%q, want 摘要v1", session.SummaryContent)
		}
		if session.SummaryUpToMsgID != boundary {
			t.Errorf("SummaryUpToMsgID=%d, want %d", session.SummaryUpToMsgID, boundary)
		}
	})

	t.Run("增量压缩推进边界", func(t *testing.T) {
		db := newAIContextTestDB(t)
		server := newSummaryMockServer(t, []string{"摘要v1", "摘要v2"})
		msgs := []Message{
			{Role: "user", Content: asciiTokens(10)},
			{Role: "assistant", Content: asciiTokens(10)},
			{Role: "user", Content: asciiTokens(10)},
			{Role: "assistant", Content: asciiTokens(10)},
		}
		sessionID := setupSummarySession(t, db, server.URL, msgs)

		svc := newAIContextTestService(t, db)

		// 第一次：摘要 [0..3)，边界 = msgs[2].ID
		firstBoundary := msgs[2].ID
		if _, ok := svc.CompactSessionSummary(context.Background(), sessionID, msgs[:3], firstBoundary); !ok {
			t.Fatal("首次压缩应成功")
		}

		// 第二次：摘要 [3..4)，边界推进到 msgs[3].ID
		secondBoundary := msgs[3].ID
		summary, ok := svc.CompactSessionSummary(context.Background(), sessionID, msgs[3:], secondBoundary)
		if !ok || summary != "摘要v2" {
			t.Fatalf("ok=%v summary=%q, want true/摘要v2", ok, summary)
		}

		var session models.AISession
		if err := db.First(&session, sessionID).Error; err != nil {
			t.Fatalf("加载会话失败: %v", err)
		}
		if session.SummaryContent != "摘要v2" {
			t.Errorf("SummaryContent=%q, want 摘要v2", session.SummaryContent)
		}
		if session.SummaryUpToMsgID != secondBoundary {
			t.Errorf("SummaryUpToMsgID=%d, want %d", session.SummaryUpToMsgID, secondBoundary)
		}
	})

	t.Run("AI 调用失败时沿用旧摘要", func(t *testing.T) {
		db := newAIContextTestDB(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		msgs := []Message{
			{Role: "user", Content: asciiTokens(10)},
			{Role: "assistant", Content: asciiTokens(10)},
		}
		sessionID := setupSummarySession(t, db, server.URL, msgs)
		// 预置旧摘要
		if err := db.Model(&models.AISession{}).Where("id = ?", sessionID).
			Updates(map[string]interface{}{"summary_content": "旧摘要", "summary_up_to_msg_id": msgs[0].ID}).Error; err != nil {
			t.Fatalf("预置旧摘要失败: %v", err)
		}

		svc := newAIContextTestService(t, db)
		summary, ok := svc.CompactSessionSummary(context.Background(), sessionID, msgs[1:], msgs[1].ID)
		if ok || summary != "" {
			t.Fatalf("ok=%v summary=%q, want false/空", ok, summary)
		}
		var session models.AISession
		if err := db.First(&session, sessionID).Error; err != nil {
			t.Fatalf("加载会话失败: %v", err)
		}
		if session.SummaryContent != "旧摘要" || session.SummaryUpToMsgID != msgs[0].ID {
			t.Errorf("旧摘要被篡改: content=%q boundary=%d", session.SummaryContent, session.SummaryUpToMsgID)
		}
	})
}
