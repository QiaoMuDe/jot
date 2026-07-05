# Tasks

- [ ] Task 1: 后端定义 `SettingsConfig` 结构体（`types.go`）
  - 结构体字段覆盖全部 20 项设置（参考 `InitDefaultSettings`）
  - 字段类型：string / bool / int
  - JSON 标签与现有 DB key 名对齐
  - 加入 `GetAllSettings()` 方法（从 SettingService 读取）和 `SaveAllSettings()` 方法（写入 SettingService + 范围校验）

- [ ] Task 2: 后端 `app.go` 新增 Wails 绑定 `GetAllSettings` / `SaveAllSettings`
  - `GetAllSettings()` 调 service 层方法返回 `SettingsConfig`
  - `SaveAllSettings(cfg)` 调 service 层方法写入，含范围校验（page_size 1-100, ai_card_recall_limit 1-30, ai_ref_max_chars 1-50000, ai_search_result_limit 1-30, font_size 8-72）
  - 调用方统一写 `settings` 表

- [ ] Task 3: 前端创建 `loadSettings()` 单一加载函数（`main.js`）
  - 调用 `GetAllSettings()` 一次获取全部设置
  - 填充所有设置页 DOM 元素（主题下拉、字体输入、排序分段、开关、AI 输入框等）
  - 应用视觉设置（theme → `applyTheme()`，font → `applyFontFamily/applyFontSize`，syntax highlight toggle，code highlight theme）
  - 替换 `loadThemeSetting` / `loadFontSettings` / `loadSortSettings` / `loadPageSizeSetting` / `loadQuickNoteSetting` / `loadSyntaxHighlightSetting` / `loadCodeHighlightThemeSetting` / `loadNoteOpenFullscreenSetting` / `loadAISettings` 共 9 个函数的功能
  - 设置页入口 `showTargetView('settings')` 调用 `loadSettings()` 替代 8 个独立调用
  - init 流程（约 6809 行）调用 `loadSettings()` 替代 7 个独立调用
  - `window.loadSettings = loadSettings` 导出，删除 9 个 `window.loadXxxSetting` 导出
  - 删除 9 个废弃的 load 函数定义
  - 删除 `init()` 末尾 5 个被注释的 load 调用

- [ ] Task 4: 前端创建 `saveSettings()` 单一保存函数（`main.js`）
  - 从所有设置页 DOM 元素收集当前值
  - 构建完整的 `SettingsConfig` 对象
  - 调用 `SaveAllSettings(cfg)` 一次性保存
  - 对 AI 工具栏（`ai-chat.js`）中的 3 个 toggle 保存点也要收集
  - 保留各个控件的即时保存体验（onchange/onclick 触发 `saveSettings()`）

- [ ] Task 5: 迁移各保存触发器指向 `saveSettings()`
  - 主题选择器 `change` → 调用 `saveSettings()` 而非 `SetSetting('theme')`
  - 字体族/字号 `change` → 调用 `saveSettings()` 而非 `saveFontSetting()`
  - 排序分段控件 `click` → 调用 `saveSettings()` 而非 `SetSetting('sort_order')`
  - 分页大小 `change` → 调用 `saveSettings()` 而非 `SetSetting('page_size')`
  - 快速笔记 checkbox `change` → 调用 `saveSettings()`
  - 语法高亮 checkbox `change` → 调用 `saveSettings()`
  - 全屏打开 checkbox `change` → 调用 `saveSettings()`
  - 代码高亮主题选择器 `change` → 调用 `saveSettings()`
  - 深度思考/联网搜索/卡片召回 toggle（设置页） → 调用 `saveSettings()`
  - AI URL/Key/Tavily Key 输入框 `change` → 调用 `saveSettings()`
  - 引用截断/搜索结果数/召回条数输入框 `change` → 调用 `saveSettings()`
  - 移除上述各点中独立的 `SetSetting()` / `SaveAIConfig()` 调用

- [ ] Task 6: 简化 `reloadSettings()`（`data-management.js`）
  - `reloadSettings()` 改为只调用 `window.loadSettings()`
  - 删除 9 行单独的 load 函数调用

- [ ] Task 7: 清理 `ai-chat.js` AI 工具栏设置加载与保存
  - 3 个 AI toggle（thinking/web_search/card_recall）的初始化加载从 `GetSetting('key') + localStorage` fallback 改为从 `loadSettings()` 的结果读取
  - 3 个 AI toolbar toggle 的保存从 `SetSetting('key', val)` + `localStorage.setItem` 改为调用 `saveSettings()`
  - 移除 3 处 `localStorage.getItem` 和 `localStorage.setItem` 调用
  - 移除 `initAIToolbar()` 中 3 处独立 `GetSetting` 调用

# Task Dependencies

- Task 1 (后端 struct) 和 Task 2 (后端绑定) 是 Task 3-7 的前置依赖
- Task 3 (loadSettings) 和 Task 4 (saveSettings) 可以并行开发
- Task 5 (迁移保存点) 依赖 Task 4
- Task 6 (reloadSettings) 依赖 Task 3
- Task 7 (ai-chat.js) 依赖 Task 3 和 Task 4
