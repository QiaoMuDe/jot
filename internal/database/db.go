package database

import (
	"fmt"
	"os"
	"path/filepath"

	"jot/internal/config"
	"jot/internal/models"
	"jot/internal/services"

	// blank import 注册 sqlite-vec 扩展（sqlite3_auto_extension，对新打开的连接自动生效）
	_ "modernc.org/sqlite/vec"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// DefaultDBPath 返回默认数据库路径: ~/.jot/data/jot.db
func DefaultDBPath() (string, error) {
	dir, err := config.SubDir(config.DirData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "jot.db"), nil
}

// InitDB 初始化 SQLite 数据库连接并执行自动迁移
// dbPath 为数据库文件路径, 默认为 ~/.jot/data/jot.db
func InitDB(dbPath string) (*gorm.DB, error) {
	// 确保数据库文件所在目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 打开 SQLite 连接 (使用纯 Go 实现的 glebarez/sqlite 驱动, 免 cgo)
	// CreateBatchSize: 批量 Create(&切片) 时按此值分批 INSERT，避免单条 SQL 参数数超过
	// SQLite 的 SQLITE_MAX_VARIABLE_NUMBER 限制（旧版 999）。NoteVector 每行 8 个参数，
	// 100×8=800 留安全余量；大文档嵌入时一篇笔记可切出上百块，不分批会触发 too many SQL variables
	// SkipDefaultTransaction: 跳过 GORM 对单条 Create/Update/Delete 的隐式事务包裹，
	// SQLite 自动提交模式下单语句本就原子，省掉冗余 BEGIN/COMMIT 往返；不影响显式 db.Transaction
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		SkipDefaultTransaction: true,
		CreateBatchSize:        100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 配置连接池: SQLite 仅支持单连接写入
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// 配置 SQLite 优化 PRAGMA，提升并发读写性能
	// WAL 模式：允许并发读取，写入不阻塞读取
	_ = db.Exec("PRAGMA journal_mode=WAL").Error
	// busy_timeout：忙等待超时 5 秒，避免 "database is locked" 错误
	_ = db.Exec("PRAGMA busy_timeout=5000").Error
	// synchronous=NORMAL：WAL 模式下安全且性能更好（比 FULL 快得多）
	_ = db.Exec("PRAGMA synchronous=NORMAL").Error
	// cache_size：8MB 页面缓存（负值表示 KB 单位）
	_ = db.Exec("PRAGMA cache_size=-8000").Error

	// 自动迁移数据模型（模型注册见 models.go 的 AllModels）
	if err := db.AutoMigrate(AllModels...); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 统一清理历史遗留孤儿列与残留设置键
	//（后续新增"列/键移除"只需在 cleanupOrphanedData 内的清单追加条目）
	if err := cleanupOrphanedData(db); err != nil {
		return nil, fmt.Errorf("清理历史遗留数据失败: %w", err)
	}

	// 初始化内置技能提示词
	if err := InitBuiltinPrompts(db); err != nil {
		return nil, fmt.Errorf("初始化内置提示词失败: %w", err)
	}

	// 初始化默认标签
	if err := services.InitDefaultTags(db); err != nil {
		return nil, fmt.Errorf("初始化默认标签失败: %w", err)
	}

	// 初始化默认设置（仅插入表中不存在的 key）
	if err := InitDefaultSettings(db); err != nil {
		return nil, fmt.Errorf("初始化默认设置失败: %w", err)
	}

	// 初始化内置 API 服务商预设（仅插入不存在的）
	if err := InitBuiltinProfiles(db); err != nil {
		return nil, fmt.Errorf("初始化内置 API 预设失败: %w", err)
	}

	// 初始化内置 MCP 服务器（仅插入不存在的，默认禁用待用户填密钥后启用）
	if err := InitBuiltinMCPServers(db); err != nil {
		return nil, fmt.Errorf("初始化内置 MCP 服务器失败: %w", err)
	}

	return db, nil
}

// BackupDir 返回备份目录路径 ~/.jot/backup/
func BackupDir() (string, error) {
	return config.SubDir(config.DirBackup)
}

// EnsureBackupDir 确保备份目录存在, 不存在则创建
func EnsureBackupDir() error {
	dir, err := BackupDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

// cleanupOrphanedData 统一清理数据库中的历史遗留数据：
//   - 孤儿列：已从代码模型移除的字段对应的列（GORM AutoMigrate 只增不删，存量表会残留旧列）
//   - 孤儿设置键：已从种子/模型移除的设置项（InitDefaultSettings 只插缺失键，存量键残留）
//
// 后续新增"移除列 / 移除设置键"的改动时，只需在本函数对应的清单
// （orphanColumnSpecs 的 cols、orphanSettingKeys）追加条目，无需新增函数。
//
// 幂等：HasColumn 为 false（列不存在或全新库）或 DELETE 未命中时无副作用，无需迁移标记；
// 失败返回 error 由 InitDB 中止启动，避免结构不一致被静默掩盖。
func cleanupOrphanedData(db *gorm.DB) error {
	// ── 孤儿列清单：{模型, 已移除字段对应的历史列名} ──
	type columnSpec struct {
		model interface{}
		cols  []string
	}
	orphanColumnSpecs := []columnSpec{
		// api_profiles：
		//   is_default —— IsDefault→IsBuiltin 字段改名前的旧列（改名时未删除，SQLite 保留为孤儿列）
		//   is_builtin —— is_builtin 字段移除后 AutoMigrate 为存量表新增的列（同样不再被代码引用）
		{model: &models.APIProfile{}, cols: []string{"is_default", "is_builtin"}},
		// ai_session_config：
		//   zhihu_search_enabled / zhihu_global_search_enabled / tavily_search_enabled ——
		//     内置搜索（Tavily/知乎/全网）整体移除（MCP 迁移）时从代码模型删除
		//   agent_enabled —— Agent 成为唯一对话模式（Chat 问答模式移除）后不再需要模式标记
		//   enable_card_recall —— 卡片召回开关移除（是否召回由 Agent 自主判断）
		//   plan_mode —— 迁移到 mode（chat/agent/plan 单字段）后删除（迁移逻辑见 db.go 顶部，须先迁移后删列）
		// 注：AIMessage.search_sources 列保留（存量历史消息数据，仅前端不再展示）
		{model: &models.AISessionConfig{}, cols: []string{
			"zhihu_search_enabled", "zhihu_global_search_enabled", "tavily_search_enabled",
			"agent_enabled", "enable_card_recall", "plan_mode",
		}},
		// ai_sessions：
		//   summary_msg_count —— 摘要触发从条数口径改为 token 预算（SummaryUpToMsgID）后废弃的旧列
		{model: &models.AISession{}, cols: []string{"summary_msg_count"}},
	}
	m := db.Migrator()
	for _, spec := range orphanColumnSpecs {
		for _, col := range spec.cols {
			if m.HasColumn(spec.model, col) {
				if err := m.DropColumn(spec.model, col); err != nil {
					return err
				}
			}
		}
	}

	// ── 孤儿设置键清单：已从种子/模型移除的键 ──
	orphanSettingKeys := []string{
		// 内置搜索（Tavily/知乎/全网）整体移除（含更早的旧键 ai_web_search_enabled，
		// 它曾迁移到 tavily_search_enabled，目标键亦已移除；
		// ai_web_search_max_chars 是 read_url 截断键改名前的旧键，现由 ai_read_url_max_chars 取代，
		// 后者已在 InitDefaultSettings 种子初始化，无前端 UI）。
		"tavily_api_key", "zhihu_access_secret", "zhihu_search_enabled",
		"zhihu_global_search_enabled", "tavily_search_enabled",
		"ai_web_search_max_chars", "ai_search_result_limit", "ai_web_search_enabled",
		// 卡片召回开关移除（是否召回由 Agent 自主判断）
		"ai_card_recall_enabled",
		// 摘要触发从条数窗口改为 token 预算（ai_context_token_budget）后废弃的旧键
		"ai_context_window_size",
	}
	if err := db.Where("key IN ?", orphanSettingKeys).Delete(&models.Setting{}).Error; err != nil {
		return err
	}
	return nil
}

// InitDefaultSettings 增量插入默认设置（仅插入缺失的 key）
func InitDefaultSettings(db *gorm.DB) error {
	var existingKeys []string
	db.Model(&models.Setting{}).Pluck("key", &existingKeys)
	existing := make(map[string]bool, len(existingKeys))
	for _, k := range existingKeys {
		existing[k] = true
	}

	defaults := []models.Setting{
		{Key: "theme", Value: "default"},
		{Key: "font_family", Value: ""},
		{Key: "font_size", Value: "16"},
		{Key: "code_highlight_theme", Value: "monokai-dimmed"},
		{Key: "note_open_fullscreen", Value: "false"},
		{Key: "sort_order", Value: "updated_at"},
		{Key: "page_size", Value: "20"},
		{Key: "cm_syntax_highlight", Value: "true"},
		{Key: "ai_base_url", Value: ""},
		{Key: "ai_api_key", Value: ""},
		{Key: "ai_model", Value: ""},
		{Key: "ai_embed_base_url", Value: ""},
		{Key: "ai_embed_api_key", Value: ""},
		{Key: "ai_embed_model", Value: ""},
		{Key: "ai_thinking_enabled", Value: "false"},
		{Key: "ai_card_recall_limit", Value: "5"},
		{Key: "max_file_size", Value: "1"},
		{Key: "ai_large_file_preview_threshold", Value: "10000"},
		// read_url 工具网页正文截断上限（无前端 UI，仅初始化默认值，由 read_url 直接读取）
		{Key: "ai_read_url_max_chars", Value: "5000"},
		// http_request 工具响应体截断上限（无前端 UI，仅初始化默认值，由 http_request 直接读取）
		{Key: "ai_http_max_chars", Value: "5000"},
		{Key: "ai_agent_tools_disabled", Value: ""},
		{Key: "ai_agent_max_iterations", Value: "20"},
		{Key: "trash_cleanup_retention_days", Value: "30"},
		{Key: "log_level", Value: "1"},
		{Key: "screen_lock_enabled", Value: "false"},
		{Key: "screen_lock_password", Value: ""},
		{Key: "editor_word_wrap", Value: "false"},
		// AI 上下文 token 预算（默认 128K），tail 达触发比例时摘要压缩
		{Key: "ai_context_token_budget", Value: "131072"},
		// 摘要压缩触发比例（无前端 UI，tail 达预算该比例时触发，调小便于测试）
		{Key: "ai_context_summary_trigger_ratio", Value: "0.8"},
	}

	var toInsert []models.Setting
	for _, s := range defaults {
		if !existing[s.Key] {
			toInsert = append(toInsert, s)
		}
	}
	if len(toInsert) == 0 {
		return nil
	}
	if err := db.Create(&toInsert).Error; err != nil {
		return err
	}
	return nil
}
