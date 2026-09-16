package services

import (
	"path/filepath"
	"testing"

	"gitee.com/MM-Q/fastlog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"jot/internal/models"
)

// newSessionConfigTestDB 打开内存 SQLite（单连接）并迁移会话配置表
func newSessionConfigTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&models.AISessionConfig{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

func newSessionConfigTestService(t *testing.T, db *gorm.DB) *AIService {
	t.Helper()
	logger := fastlog.New(fastlog.Prod(filepath.Join(t.TempDir(), "test.log")))
	return NewAIService(db, logger)
}

func TestApprovalModeOrDefault(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"confirm_every 合法", "confirm_every", "confirm_every"},
		{"auto 合法", "auto", "auto"},
		{"review 合法", "review", "review"},
		{"空值回退", "", "confirm_every"},
		{"非法值回退", "always", "confirm_every"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := approvalModeOrDefault(c.in); got != c.want {
				t.Fatalf("approvalModeOrDefault(%q) = %q, 期望 %q", c.in, got, c.want)
			}
		})
	}
}

func TestLoadSessionConfigApprovalMode(t *testing.T) {
	db := newSessionConfigTestDB(t)
	svc := newSessionConfigTestService(t, db)

	// 默认会话：approval_mode 为空（模拟存量/未设置），加载应兜底为 confirm_every
	if err := db.Create(&models.AISessionConfig{SessionID: 1}).Error; err != nil {
		t.Fatalf("插入默认会话配置失败: %v", err)
	}
	if got := svc.LoadSessionConfig(1).ApprovalMode; got != "confirm_every" {
		t.Fatalf("空 approval_mode 加载后 = %q, 期望 confirm_every", got)
	}

	// 已选 review：加载应原样保留
	if err := db.Create(&models.AISessionConfig{SessionID: 2, ApprovalMode: "review"}).Error; err != nil {
		t.Fatalf("插入 review 会话配置失败: %v", err)
	}
	if got := svc.LoadSessionConfig(2).ApprovalMode; got != "review" {
		t.Fatalf("review 会话加载后 = %q, 期望 review", got)
	}
}
