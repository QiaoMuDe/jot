# Tasks

- [x] Task 1: DB 初始化时插入默认值
  - 在 `internal/database/db.go` 中新增 `InitDefaultSettings(db)` 函数，定义默认值清单
  - 使用与 `initBuiltinPrompts` 相同的模式：遍历默认值，对表中不存在的 key 执行 INSERT
  - 在 `InitDB()` 中的 `AutoMigrate` 之后调用
  - 在 `ResetDatabase()` 中也调用该函数

- [x] Task 2: 简化后端 Getter（移除硬编码 fallback）
  - `GetSortOrder()` → 直接返回 `a.settingService.Get("sort_order")`
  - `GetPageSize()` → 直接返回 `GetSetting("page_size")` 转 int
  - `GetAIRefMaxChars()` → 直接返回 `GetSetting("ai_ref_max_chars")` 转 int
  - `GetAISearchResultLimit()` → 直接返回 `GetSetting("ai_search_result_limit")` 转 int
  - `GetAICardRecallLimit()` → 直接返回 `GetSetting("ai_card_recall_limit")` 转 int
  - `ai_service.go` 中的 `GetConfig()` → 移除 provider fallback，`SaveConfig()` 同理

- [x] Task 3: 简化前端 load* 函数（移除硬编码 fallback）
  - `loadThemeSetting()` — 移除 localStorage fallback 分支
  - `loadFontSettings()` — 移除 localStorage fallback 分支
  - `loadSortSettings()` — 直接使用 GetSortOrder() 返回值
  - `loadPageSizeSetting()` — 直接使用 GetPageSize() 返回值
  - `loadQuickNoteSetting()` — 移除 localStorage fallback 分支
  - `loadSyntaxHighlightSetting()` — 从 `val !== 'false'` 改为 `val === 'true'`
  - `loadNoteOpenFullscreenSetting()` — 移除 localStorage fallback 分支
  - `loadCodeHighlightThemeSetting()` — 移除 localStorage fallback 分支
  - `loadAISettings()` — 深度思考/联网搜索/卡片召回开关移除 false 初始化
  - `updateProviderUI()` — 添加注释说明默认 URL 仅用于 UI 提示

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1]
