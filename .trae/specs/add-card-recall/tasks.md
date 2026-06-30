# Tasks

- [x] Task 1: 实现 2-gram 分词 + 卡片召回后端服务
  - [ ] 新建 `internal/services/recall_service.go`
  - [ ] 定义 `RecallCard` struct（ID, Title, Content）
  - [ ] 定义 `CardRecallResult` struct（FormattedText string, Cards []RecallCard）
  - [ ] 实现 `tokenize2Gram(text string) []string` — 中文 2-gram 分词，英文/数字按空格/标点切分，去重
  - [ ] 实现 `CardRecallSearch(ctx, query string, limit int) *CardRecallResult` — 分词 → 搜索 → 格式化上下文 + 结构化卡片数据
  - [ ] 格式化输出：`"以下是用户笔记库中与问题相关的笔记：\n\n--- 📄 《标题》 ---\n内容\n--- 📄 《标题2》 ---\n内容2\n\n请基于以上笔记内容..."`

- [ ] Task 2: NoteService 新增 `SearchFull` 方法
  - [ ] `SearchFull(keywords []string, limit int) ([]models.Note, error)` — 多关键词 OR LIKE 搜索标题+内容
  - [ ] 返回完整 Content（不截断），Limit 控制返回条数
  - [ ] 按 `updated_at DESC` 排序
  - [ ] 不需要分页、日期过滤等无关参数——简洁够用

- [ ] Task 3: 修改后端 `CallAIStream` — 新增 cardRecallEnabled 参数 + 发射召回数据事件
  - [ ] `app.go`: `CallAIStream` 签名新增 `cardRecallEnabled bool`
  - [ ] 在 goroutine 内、联网搜索之后插入卡片召回块：
    - [ ] 如果 `cardRecallEnabled` 开启 → 提取最后一条 user message → 调 `CardRecallSearch` → 非空时注入 system message
    - [ ] 同时 `json.Marshal` 结构化卡片数据 → `runtime.EventsEmit("ai:recall-cards", cardsJSON)`
    - [ ] 无匹配时静默跳过
  - [ ] 前端 `window.go.main.App.CallAIStream` 调用处同步新增 `enableCardRecall` 参数
  - [ ] 取消逻辑不需要改动（ctx 取消已可终止搜索）

- [ ] Task 4: 前端设置页 — 卡片召回开关 + 召回条数配置
  - [ ] `index.html`: 在联网搜索设置之后添加「卡片召回」配置区，复用 `.ai-setting-search-line` + `.ai-setting-search-desc` 样式：
    - [ ] 行 1：标签 + 描述文字「发送消息时自动召回相关笔记」+ toggle `#aiSettingCardRecallToggle`
    - [ ] 行 2：标签 + 输入框 `#aiSettingCardRecallLimit`（数字，min=1 max=10，默认 3）+ 「条/次」
  - [ ] `main.js` `loadAISettings()`: 从 localStorage 恢复 `ai_card_recall_enabled`（toggle active class）和 `ai_card_recall_limit`（输入框 value）
  - [ ] `main.js`: toggle 和 limit 输入框的 change 事件 → 存入 localStorage

- [ ] Task 5: 前端 AI 聊天工具栏 — 卡片召回开关
  - [ ] `index.html`: 在「联网搜索」开关后新增「卡片召回」开关 `#aiChatCardRecallToggle`：
    - [ ] 复用 `.ai-chat-search-toggle` 样式
    - [ ] SVG 图标：使用文档/笔记图标（`<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>`）
    - [ ] 文字标签为「卡片召回」
    - [ ] 复用 `.ai-chat-toggle-switch` + `.ai-chat-toggle-knob`
  - [ ] `ai-chat.js`:
    - [ ] 新增 `enableCardRecall` 变量，从 localStorage `ai_card_recall_enabled` 初始化
    - [ ] 点击事件：切换 `active` class → 保存 localStorage → 同步设置页 toggle
    - [ ] `startStreaming()`: `CallAIStream` 调用新增 `enableCardRecall` 参数

- [ ] Task 6: 前端 AI 回复后展示召回卡片 + 卡片预览浮层
  - [ ] `ai-chat.js`:
    - [ ] 新增 `recallCards` 变量（`null` 初始值）
    - [ ] 新增 `ai:recall-cards` 事件监听（EventsOn），解析 JSON 存入 `recallCards`
    - [ ] 在 `ai:stream-done` 回调中，`recallCards` 非空时插入 `<details class="recall-cards">` 折叠面板到消息气泡底部（操作按钮上方）
      - [ ] `<summary class="recall-cards-summary">` → `📄 召回笔记 (N 篇)`
      - [ ] 每张卡片：`<div class="recall-cards-item">` → 标题（可点击 `<a>`）+ 内容摘要 `<p>`（前 100 字，3 行截断）
      - [ ] 点击 `<a>` 触发 `openCardPreview(card)` 函数
    - [ ] 流完成/错误时清空 `recallCards = null`
    - [ ] 实现 `openCardPreview(card)` 函数：动态创建/显示预览浮层
  - [ ] `index.html`: 新增预览浮层 HTML（始终隐藏）：
    - [ ] `<div id="aiCardPreviewModal" class="ai-card-preview-modal" style="display:none">`
    - [ ] 浮层包含：遮罩层 `.ai-card-preview-overlay` + 面板 `.ai-card-preview-panel`
    - [ ] 面板结构：关闭按钮 `<button class="ai-card-preview-close">` + 标题 `<h2>` + 内容 `<div class="ai-card-preview-content">`
  - [ ] `ai-chat.css`:
    - [ ] `.recall-cards` / `.recall-cards-summary` / `.recall-cards-item` / `.recall-cards-snippet` — 复用 `.search-sources` 样式体系（边框、圆角、hover、3 行截断等），不同 class name
    - [ ] `.ai-card-preview-modal` / `.ai-card-preview-overlay` / `.ai-card-preview-panel` — 模态浮层样式（固定定位、居中、最大高度 80vh、可滚动、圆角阴影）

- [ ] Task 7: AGENTS.md 更新记忆

## Task Dependencies

- [Task 1] → [Task 3]
- [Task 2] → [Task 1]（recall_service.go 调用 NoteService.SearchFull）
- [Task 4] 与 [Task 5] 可并行
- [Task 3] + [Task 4] + [Task 5] + [Task 6] 都完成后功能闭环
- [Task 7] 最后执行
