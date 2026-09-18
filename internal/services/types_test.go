package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"jot/internal/models"
)

// newSettingTestDB 打开内存 SQLite 并迁移 Setting 表，供 SaveAllSettings clamp 测试使用
func newSettingTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestSaveAllSettingsChunkClamp 验证 SaveAllSettings 对向量切块区间的 clamp 行为：
//   - target > max：max 保持，target 夹到 max（保证 max>=target）
//   - target 越界（<1）：夹到 1；max 越界（<100）：夹到 100
//   - clamp 后写库，GetAllSettings 读回的值与落库值一致（口径一致）
func TestSaveAllSettingsChunkClamp(t *testing.T) {
	db := newSettingTestDB(t)
	svc := NewSettingService(db)

	// 场景1：target=5000, max=200 → target 夹到 max(200)，max 不变
	if err := svc.SaveAllSettings(SettingsConfig{AIChunkTargetRunes: 5000, AIChunkMaxRunes: 200}); err != nil {
		t.Fatalf("SaveAllSettings 失败: %v", err)
	}
	cfg := svc.GetAllSettings()
	if cfg.AIChunkMaxRunes != 200 || cfg.AIChunkTargetRunes != 200 {
		t.Errorf("场景1 期望 (target=200, max=200)，实际 (target=%d, max=%d)",
			cfg.AIChunkTargetRunes, cfg.AIChunkMaxRunes)
	}

	// 场景2：target=-1, max=50 → max 夹到 100，target 夹到 1
	if err := svc.SaveAllSettings(SettingsConfig{AIChunkTargetRunes: -1, AIChunkMaxRunes: 50}); err != nil {
		t.Fatalf("SaveAllSettings 失败: %v", err)
	}
	cfg = svc.GetAllSettings()
	if cfg.AIChunkMaxRunes != 100 || cfg.AIChunkTargetRunes != 1 {
		t.Errorf("场景2 期望 (target=1, max=100)，实际 (target=%d, max=%d)",
			cfg.AIChunkTargetRunes, cfg.AIChunkMaxRunes)
	}

	// 场景3：正常值 target=600, max=1500 原样保存并读回
	if err := svc.SaveAllSettings(SettingsConfig{AIChunkTargetRunes: 600, AIChunkMaxRunes: 1500}); err != nil {
		t.Fatalf("SaveAllSettings 失败: %v", err)
	}
	cfg = svc.GetAllSettings()
	if cfg.AIChunkMaxRunes != 1500 || cfg.AIChunkTargetRunes != 600 {
		t.Errorf("场景3 期望 (target=600, max=1500)，实际 (target=%d, max=%d)",
			cfg.AIChunkTargetRunes, cfg.AIChunkMaxRunes)
	}

	// 场景4：max 越上界 20000 → 夹到 10000；target 在范围内
	if err := svc.SaveAllSettings(SettingsConfig{AIChunkTargetRunes: 500, AIChunkMaxRunes: 20000}); err != nil {
		t.Fatalf("SaveAllSettings 失败: %v", err)
	}
	cfg = svc.GetAllSettings()
	if cfg.AIChunkMaxRunes != 10000 || cfg.AIChunkTargetRunes != 500 {
		t.Errorf("场景4 期望 (target=500, max=10000)，实际 (target=%d, max=%d)",
			cfg.AIChunkTargetRunes, cfg.AIChunkMaxRunes)
	}
}

// TestClampChunkSizes 直接验证 clampChunkSizes 纯函数：
// 覆盖 target/max 越界与恒等场景，与 SaveAllSettings 共用同一函数保证口径一致
func TestClampChunkSizes(t *testing.T) {
	cases := []struct {
		name        string
		target, max int
		wantTarget  int
		wantMax     int
	}{
		{"边界内恒等", 600, 1500, 600, 1500},
		{"target大于max", 5000, 200, 200, 200},
		{"target小于1", -1, 50, 1, 100},
		{"max小于100", 0, 10, 1, 100},
		{"max大于10000", 500, 20000, 500, 10000},
		{"max等于10000上限保持", 10000, 10000, 10000, 10000},
	}
	for _, c := range cases {
		gotTarget, gotMax := clampChunkSizes(c.target, c.max)
		if gotTarget != c.wantTarget || gotMax != c.wantMax {
			t.Errorf("%s: clampChunkSizes(%d,%d) = (%d,%d)，期望 (%d,%d)",
				c.name, c.target, c.max, gotTarget, gotMax, c.wantTarget, c.wantMax)
		}
	}
}
