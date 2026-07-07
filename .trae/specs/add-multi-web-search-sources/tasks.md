# Tasks

## Task 1: 引入依赖并新增知乎搜索服务层 ✅

- [x] 运行 `go get gitee.com/MM-Q/zhihu-go` 引入依赖
- [x] 创建 `internal/services/zhihu_search_service.go`，封装 `SearchZhihuContent` 和 `SearchGlobalContent` 两个函数
  - 接收 `ctx`, `query`, `accessSecret`, `maxResults` 参数
  - 调用 zhihu-go 的 `SearchZhihu` / `SearchGlobal` API
  - 返回与 `SearchWebResult` 一致的格式（`FormattedText` + `Sources`），但 Sources 中标注来源为 `"zhihu_search"` / `"zhihu_global"`
  - 超时控制 5s，失败时返回 error 信息（不再静默返回 nil）

## Task 2: 修改后端数据结构 ✅

- [x] 修改 `internal/services/types.go` — `SettingsConfig` 新增字段：
  - `ZhihuAccessSecret string json:"zhihu_access_secret"`
  - `ZhihuSearchEnabled bool json:"zhihu_search_enabled"`
  - `ZhihuGlobalSearchEnabled bool json:"zhihu_global_search_enabled"`
  - `TavilySearchEnabled bool json:"tavily_search_enabled"`（替代 `AIWebSearchEnabled`）
- [x] 更新 `GetAllSettings()` 读取新字段
- [x] 更新 `SaveAllSettings()` 写入新字段并添加值校验
- [x] 修改 `internal/services/ai_service.go` — `AIConfig` 新增 `ZhihuAccessSecret` 字段
- [x] 更新 `GetConfig()` / `SaveConfig()` 读写 `zhihu_access_secret`
- [x] 修改 `internal/database/db.go` — `InitDefaultSettings` 新增默认设置项：
  - `zhihu_access_secret = ""`
  - `zhihu_search_enabled = "false"`
  - `zhihu_global_search_enabled = "false"`
  - `tavily_search_enabled = "false"`（替代 `ai_web_search_enabled`）

## Task 3: 修改后端核心搜索与 AI 流式逻辑 ✅

- [x] 修改 `internal/services/search_service.go` — `SearchSource` 新增 `SourceLabel string json:"source_label"` 字段
- [x] 修改 `internal/services/search_service.go` — `SearchWeb` 在 Sources 中填充 `SourceLabel: "tavily"`，失败时返回 error（不再静默返回 nil）
- [x] 修改 `app.go` — `CallAIStream` 签名：`searchEnabled bool` → `searchSources []string`
- [x] 修改 `app.go` — 当 `searchSources` 非空时，并行（goroutine）执行所有启用的搜索：
  - 每个搜索源发射 `ai:search-source-status` 事件（`{source, status: "searching"}`）
  - 搜索成功：发射 `ai:search-source-status`（`{source, status: "success", count}`）
  - 搜索失败：发射 `ai:search-error`（`{source, error: "失败原因"}`）
  - 所有搜索源完成后发射 `ai:search-status → "done"`
  - 结果注入 system message 时标注来源
- [x] 修改 `app.go` — 新增 `TestZhihuConnection(accessSecret string)` 绑定方法
- [x] 所有搜索结果统一使用 `ai_search_result_limit` 作为 maxResults

## Task 4: 修改前端设置页 ✅

- [x] 修改 `frontend/index.html`
  - 在「对话与搜索」区域新增「知乎 Access Secret」密码输入框（`id="aiZhihuAccessSecret"`）+ 显示/隐藏按钮 + 测试连接按钮（`id="aiTestZhihuBtn"`）
  - 将原来的「默认开启」替换为三个独立开关：
    - 「知乎搜索」（`id="aiSettingZhihuSearchToggle"`）
    - 「全网搜索」（`id="aiSettingZhihuGlobalSearchToggle"`）
    - 「Tavily搜索」（`id="aiSettingTavilySearchToggle"`）
- [x] 修改 `frontend/src/main.js` — `loadSettings()` 读取新设置项并同步 UI
- [x] 修改 `frontend/src/main.js` — `saveSettings()` 收集新设置项

## Task 5: 修改前端 AI 聊天栏多源搜索 UI ✅

- [x] 修改 `frontend/index.html`
  - 将 `#aiChatWebSearchToggle` 从 toggle 控件改为按钮 `#aiChatSearchSourcesBtn`
  - 新增下拉菜单 `#aiChatSearchSourcesDropdown`，内含三个复选框：
    - 「知乎搜索」`id="aiChatZhihuSearch"`
    - 「全网搜索」`id="aiChatZhihuGlobalSearch"`
    - 「Tavily搜索」`id="aiChatTavilySearch"`
- [x] 修改 `frontend/src/js/ai-chat.js`
  - `enableWebSearch` 替换为 `searchSources` Set（如 `new Set(["zhihu_search", "tavily"])`）
  - 复选框点击切换状态，同步设置页对应开关
  - 按钮激活状态 = 至少一个搜索源被选中
  - `onSend()` 中传递 `searchSources` 数组给 `CallAIStream`
  - `syncToolbarState()` 从 DOM 读取三个复选框状态
- [x] 修改 CSS 样式（`frontend/src/css/components/ai-chat.css` 或内联样式）— 添加下拉菜单样式

## Task 6: 重新设计前端搜索动画（多源分阶段） ✅

- [x] 修改 `frontend/src/js/ai-chat.js` — `createAdvancedSearchIndicator()` 重构为多源版本：
  - 创建 `createMultiSourceSearchIndicator(sources)` 新函数
  - 精炼阶段：显示「正在优化搜索词...」（与现有一致）
  - 搜索阶段：以列表形式展示每个源的独立状态行
    - 每行包含：来源图标 + 来源名称 + 状态图标（旋转/✓/✗）+ 状态文字
    - 支持通过 `ai:search-source-status` 事件更新单个源的状态
    - 支持通过 `ai:search-error` 事件显示单个源的错误信息
  - 完成阶段：所有源完成后切换到打字点动画
  - 点击可展开详情面板显示各源结果数/错误
- [x] 新增搜索源状态管理（`sourceStates` Map）：跟踪每个搜索源的状态、结果数、错误信息
- [x] 新增 `ai:search-source-status` 事件监听处理
- [x] 新增 `ai:search-error` 事件监听处理，在动画中显示红色错误状态
- [x] 修改 `frontend/src/css/components/ai-chat.css` — 新增多源搜索动画样式：
  - `.ai-multi-search-indicator` — 容器样式
  - `.ai-search-source-row` — 单源状态行
  - `.ai-search-source-row.success` — 成功状态（绿色）
  - `.ai-search-source-row.error` — 失败状态（红色）
  - `.ai-search-source-row.searching` — 搜索中（旋转动画）
  - 各状态图标样式（✓/✗/旋转）

## Task 7: 验证与清理 ✅

- [x] 更新 `wailsjs` 绑定（手动更新 `models.ts`）
- [x] 验证设置可正常保存和加载
- [x] 验证多源搜索动画正常工作（每个源独立状态展示）
- [x] 验证搜索失败时前端显示错误提示
- [x] 验证所有搜索源均失败时，AI 仍能正常回复
- [x] 验证搜索结果在 Sources 中标注来源标签
- [x] 验证搜索来源折叠面板按来源分组展示
- [x] 验证后端的设置页和 AI 聊天栏 toggle 双向同步
- [x] 验证后端兼容性（旧数据中 `ai_web_search_enabled` 的迁移逻辑）

## Task Dependencies

- Task 1 (zhihu-go SDK) 是 Task 3 的前置依赖
- Task 2 (数据结构修改) 是 Task 3 和 Task 4 的前置依赖
- Task 4 和 Task 5 可并行开发
- Task 6 依赖 Task 3（后端事件流）
- Task 7 在所有其他任务完成后执行
