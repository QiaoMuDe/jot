# 卡片召回（Card Recall）功能 Spec

## Why

用户提问时，AI 助手当前只能基于对话历史回答，无法自动关联用户已有的笔记内容。添加卡片召回后，系统会根据用户输入自动检索相关笔记，将匹配的笔记内容作为上下文注入 AI 提示词，使回答能够参考用户的知识库，提升回答的个性化和准确性。

## What Changes

- **后端新增** `recall_service.go` — 2-gram 分词 + 多关键词 OR 搜索笔记 + 取全文 + 格式化为上下文
- **后端新增** `RecallCard` struct — ID/Title/Content 结构化召回卡片数据，用于前端展示
- **后端修改** `app.go` — `CallAIStream` 新增 `cardRecallEnabled bool` 参数，在搜索阶段调用卡片召回
- **后端修改** `app.go` — 召回结果通过 `ai:recall-cards` 事件发射结构化数据到前端
- **前端设置页** — AI 助手设置区新增「卡片召回」开关 + 召回条数配置（默认 3 条），复用 `.ai-setting-search-line` 样式
- **前端 AI 聊天** — 工具栏新增「卡片召回」开关（复用 `.ai-chat-search-toggle` + `.ai-chat-toggle-switch` CSS）
- **前端 AI 回复** — AI 回复完成后，在消息气泡底部展示召回卡片折叠面板
- **前端卡片预览** — 点击召回卡片打开轻量预览浮层（modal），展示标题和完整内容
- **NoteService** — 新增 `SearchFull(keywords []string, limit int)` 查询方法（取全文而非 200 字截断）

## Impact

- Affected specs: AI 助手（流式回复/联网搜索/笔记引用）
- Affected code: `internal/services/`（新增 recall_service.go + 修改 note_service.go）、`app.go`、`frontend/src/main.js`、`frontend/src/js/ai-chat.js`、`frontend/index.html`、`frontend/src/css/components/ai-chat.css`

## ADDED Requirements

### Requirement: 卡片召回后端服务

The system SHALL provide a card recall service that:
1. Takes a user query string and a limit count
2. Splits the query into 2-gram tokens
3. Searches notes using multiple LIKE OR conditions
4. Fetches full content of matched notes
5. Formats results into AI-readable context text
6. Returns both formatted text (for system injection) and structured card data (for frontend display)

#### Scenario: 用户启用卡片召回并提问
- **WHEN** 用户在 AI 对话中启用了卡片召回开关并发送消息
- **THEN** 后端对用户输入做 2-gram 分词 → 多关键词 OR 搜索笔记 → 最多返回 N 条匹配笔记 → 格式化后注入 system message → 通过 `ai:recall-cards` 事件发射卡片数据 → 再调用 AI 流式回复

#### Scenario: 无匹配笔记
- **WHEN** 卡片召回执行后没有匹配到任何笔记
- **THEN** 静默跳过，不发射 `ai:recall-cards` 事件，不注入任何内容，AI 正常回复

#### Scenario: 2-gram 分词逻辑
- **WHEN** 用户输入 "数据库优化方案"
- **THEN** 生成 2-gram 集：["数据库", "据库优", "库优化", "优化方", "化方案"] → 去重后用于 OR 搜索
- 英文/数字按空格/标点分词后与 2-gram 混合使用

### Requirement: 前端卡片召回开关

The system SHALL provide a toggle in both the AI chat toolbar and the settings page to enable/disable card recall. 样式完全复用现有模式：
- 工具栏开关复用 `.ai-chat-search-toggle` + `.ai-chat-toggle-switch` + `.ai-chat-toggle-knob`
- 设置页开关复用 `.ai-setting-search-line` + `.ai-setting-search-desc` + `.ai-chat-toggle-switch`

#### Scenario: AI 工具栏开关
- **WHEN** 用户点击 AI 聊天工具栏的「卡片召回」开关
- **THEN** 开关 `active` class 切换，localStorage 保存状态（key: `ai_card_recall_enabled`），设置页同步状态

#### Scenario: 设置页开关
- **WHEN** 用户在设置页点击「卡片召回」行上的 toggle
- **THEN** 开关 `active` class 切换，localStorage 保存状态，工具栏同步状态

### Requirement: 召回上下文注入

The system SHALL format recalled notes and inject them into the AI's system message.

#### Scenario: 召回结果注入
- **WHEN** 卡片召回有结果返回
- **THEN** 结果按「标题 + 全文内容」格式化为文本，以 `--- 📄 《标题》 ---\n内容` 格式拼接，前后加上引导说明和约束提示，追加到 system message 中

#### Scenario: 与联网搜索共存
- **WHEN** 同时启用了卡片召回和联网搜索
- **THEN** 先执行联网搜索（注入搜索来源），再执行卡片召回（注入笔记上下文），两者追加到同一个 system message

### Requirement: AI 回复后展示召回卡片

The system SHALL display recalled cards in a collapsible panel below the AI's reply, similar to the web search sources pattern.

#### Scenario: 召回卡片折叠面板
- **WHEN** AI 流式回复完成（`ai:stream-done`），且本次有召回卡片
- **THEN** 在消息气泡底部（操作按钮上方）插入 `<details class="recall-cards">` 折叠面板
  - `<summary>` 显示「📄 召回笔记 (N 篇)」
  - 每张卡片显示标题（可点击）+ 内容摘要（前 100 字截断，3 行省略）
  - 点击卡片标题触发预览浮层

#### Scenario: 无召回卡片
- **WHEN** AI 流式回复完成，但本次没有召回卡片
- **THEN** 不显示任何折叠面板

### Requirement: 卡片预览浮层

The system SHALL provide a lightweight preview modal when clicking a recalled card.

#### Scenario: 点击卡片查看预览
- **WHEN** 用户在召回卡片折叠面板中点击某张卡片的标题
- **THEN** 打开预览浮层，展示：
  - 标题（顶部，字号稍大加粗）
  - 完整内容（可滚动，保持原始排版）
  - 右上角关闭按钮（X）
  - 点击浮层外部区域关闭

## MODIFIED Requirements

### Requirement: CallAIStream 流式调用

The system SHALL support an optional card recall phase alongside the web search phase.

- `CallAIStream` 签名新增 `cardRecallEnabled bool` 参数
- 当 `cardRecallEnabled = true` 时，在搜索阶段（联网搜索之后）执行卡片召回
- 召回后通过 `runtime.EventsEmit(a.ctx, "ai:recall-cards", cardsJSON)` 发射结构化数据
- 召回错误不阻塞主流程

### Requirement: NoteService 新增 SearchFull 方法

`NoteService` SHALL provide a `SearchFull(keywords []string, limit int) ([]models.Note, error)` method that:
- Takes a slice of keywords and a limit
- Builds a multi-keyword OR LIKE query against title and content
- Returns full content (not truncated to 200 chars)
- Applies LIMIT
- Results ordered by `updated_at DESC`

## REMOVED Requirements

None.

## Assumptions & Decisions

1. **分词采用 2-gram**：不需要引入外部分词库，纯算法实现，零依赖，对中文短文本召回效果够用
2. **去重**：2-gram 使用 `map[string]struct{}` 去重，避免重复关键词
3. **最多召回 N 条**（可配置，默认 3）：召回太多笔记会撑大 token 消耗，限制数量控制成本
4. **全量内容召回**：区别于列表搜索（前 200 字），卡片召回需要完整笔记内容作为 AI 参考
5. **localStorage 存开关状态**：与联网搜索、深度思考保持一致
6. **在联网搜索之后执行**：如果同时启用，按「联网搜索 → 卡片召回」顺序注入，卡片召回更贴近用户私有知识
7. **卡片预览为轻量浮层**：不跳转到编辑页，在当前对话上下文中快速查看笔记全文
8. **卡片样式与联网搜索来源一致**：折叠面板、边框圆角、hover 效果复用 `.search-sources` 样式体系，仅 class name 不同以方便区分
