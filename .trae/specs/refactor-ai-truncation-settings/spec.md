# 拆分引用截断设置项 Spec

## Why

现有的 `ai_ref_max_chars`（引用截断字数）同时服务于 5 个不同场景，但各场景的合理值差异大，用户无法独立调优。按照用户决策：
- **笔记引用、卡片召回、文件上传**：不再截断，全量使用（文件上传保留 10MB 大小限制兜底）
- **联网搜索结果**：新增专用截断设置项
- **大文件预览**：新增专用阈值设置项

## What Changes

- **REMOVED** `ai_ref_max_chars` 设置项（DB 默认值、SettingsConfig 字段、前端 UI、后端绑定方法）
- **REMOVED** `BuildNoteRefContext` 中的单条笔记截断逻辑（保留总长度截断，参考 `max_file_size` 配置）
- **REMOVED** `readAIChatFiles` 中的内容截断逻辑（保留 10MB 大小限制和二进制检测）
- **REMOVED** `CardRecallSearch` 中的内容截断逻辑（去除 `maxChars` 参数，全量注入）
- **REMOVED** `CardRecallSearch` 的调用方（`app.go` 两处）读取 `ai_ref_max_chars` 的代码
- **REMOVED** `GetAIRefMaxChars()` / `SetAIRefMaxChars()` 后端绑定方法
- **ADDED** `ai_web_search_max_chars` 设置项（联网搜索结果截断，默认 5000，范围 1-50000）
- **ADDED** `ai_large_file_preview_threshold` 设置项（大文件纯文本预览阈值，默认 10000，范围 1-100000）
- **MODIFIED** 联网搜索（Tavily/知乎搜索/知乎全网）调用方改为读取 `ai_web_search_max_chars`
- **MODIFIED** 前端大文件预览逻辑改为读取 `ai_large_file_preview_threshold`

## Impact

- Affected specs: 设置页、笔记引用、AI 聊天文件上传、卡片召回、联网搜索、编辑器预览
- Affected code: `app.go`, `types.go`, `note_service.go`, `recall_service.go`, `search_service.go`, `zhihu_search_service.go`, `db.go`, `index.html`, `main.js`, `models.ts`, `App.js`, `App.d.ts`

## REMOVED Requirements

### Requirement: `ai_ref_max_chars` 设置项
**Reason**: 拆分到两个专用设置项，来源场景（笔记引用/卡片召回/文件上传）不再需要截断
**Migration**: 用户在设置页看到的"引用截断"输入框将被替换为"联网搜索结果截断"和"大文件纯文本预览阈值"两个独立输入框

### Requirement: `GetAIRefMaxChars()` / `SetAIRefMaxChars()` 后端绑定
**Reason**: 不再需要独立的 Wails 绑定方法，新设置项通过 `SaveAllSettings` / `GetAllSettings` 统一管理

### Requirement: `BuildNoteRefContext` 单条笔记截断
**Reason**: 笔记引用不再截断，全量使用
**Migration**: 移除 `maxPerNote` 相关逻辑，`NoteRefInfo.Truncated` 字段始终为 `false`

### Requirement: `readAIChatFiles` 内容截断
**Reason**: 文件上传不再截断，保留 10MB 大小限制兜底
**Migration**: 移除 `GetAIRefMaxChars()` 调用和截断逻辑，`Truncated` 字段始终为 `false`

### Requirement: `CardRecallSearch` 内容截断
**Reason**: 卡片召回不再截断，全量注入
**Migration**: 移除 `maxChars` 参数，`RecallCard.Truncated` 字段始终为 `false`；`extractContext` 函数不再使用（可保留不删除）

## ADDED Requirements

### Requirement: 联网搜索结果截断设置项

系统 SHALL 新增 `ai_web_search_max_chars` 设置项，控制联网搜索（Tavily、知乎搜索、知乎全网）单条结果的截断字数。

- 默认值：5000
- 范围：1-50000
- DB 初始化：`{Key: "ai_web_search_max_chars", Value: "5000"}`
- `SettingsConfig` 新增字段 `AIWebSearchMaxChars int \`json:"ai_web_search_max_chars"\``
- `GetAllSettings()` 中读取，`SaveAllSettings()` 中写入（含范围校验）
- 前端设置页「对话与搜索」区域新增输入框，id: `aiWebSearchMaxChars`

#### Scenario: 设置页加载显示
- **WHEN** 用户打开设置页
- **THEN** 输入框显示当前 DB 中的 `ai_web_search_max_chars` 值，默认 5000

#### Scenario: 用户修改并保存
- **WHEN** 用户修改该值并保存
- **THEN** 值在 1-50000 范围内时写入 DB，超出范围时重置为默认值

#### Scenario: 联网搜索使用新设置
- **WHEN** 用户发送消息触发联网搜索（Tavily/知乎搜索/知乎全网）
- **THEN** 搜索服务读取 `ai_web_search_max_chars` 作为 `maxChars` 参数传入

### Requirement: 大文件纯文本预览阈值设置项

系统 SHALL 新增 `ai_large_file_preview_threshold` 设置项，控制 `.md` 笔记内容超过多少字符时自动切换为纯文本模式（跳过 Markdown 渲染）。

- 默认值：10000
- 范围：1-100000
- DB 初始化：`{Key: "ai_large_file_preview_threshold", Value: "10000"}`
- `SettingsConfig` 新增字段 `AILargeFilePreviewThreshold int \`json:"ai_large_file_preview_threshold"\``
- `GetAllSettings()` 中读取，`SaveAllSettings()` 中写入（含范围校验）
- 前端设置页「对话与搜索」区域新增输入框，id: `aiLargeFilePreviewThreshold`

#### Scenario: 设置页加载显示
- **WHEN** 用户打开设置页
- **THEN** 输入框显示当前 DB 中的 `ai_large_file_preview_threshold` 值，默认 10000

#### Scenario: 打开大文件 `.md` 笔记
- **WHEN** 用户打开一篇 `.md` 笔记且内容长度 > `ai_large_file_preview_threshold`
- **THEN** 自动切换为 CM6 纯文本编辑模式，显示通知"笔记内容超过纯文本预览阈值，已自动切换为纯文本模式"

## MODIFIED Requirements

### Requirement: `BuildNoteRefContext` 去除单条截断

系统 SHALL 移除 `BuildNoteRefContext` 中的 `maxPerNote` 截断逻辑，保留总长度截断（基于 `max_file_size` 配置）。

- 移除第 140-147 行读取 `ai_ref_max_chars` 的代码
- `noteText` 直接使用 `row.Content` 完整内容
- `truncated` 始终为 `false`
- 总长度截断（基于 `max_file_size`）逻辑保留

#### Scenario: 用户引用笔记
- **WHEN** 用户引用笔记发送给 AI
- **THEN** 每条笔记的完整内容注入上下文，不再截断

### Requirement: `readAIChatFiles` 去除内容截断

系统 SHALL 移除 `readAIChatFiles` 中的 `GetAIRefMaxChars()` 调用和内容截断逻辑。

- 移除第 1601-1607 行截断代码
- `result.Truncated` 始终为 `false`

#### Scenario: 用户上传文件到 AI 聊天
- **WHEN** 用户上传文本文件到 AI 聊天
- **THEN** 文件完整内容注入（10MB 大小限制和二进制检测仍保留）

### Requirement: `CardRecallSearch` 去除内容截断

系统 SHALL 移除 `CardRecallSearch` 中的 `maxChars` 参数和内容截断逻辑。

- `CardRecallSearch` 签名移除 `maxChars int` 参数
- 移除第 180-190 行截断代码
- `RecallCard.Truncated` 始终为 `false`
- `extractContext` 函数可以保留不删除（不影响功能）
- 调用方（`app.go` 两处）移除读取 `ai_ref_max_chars` 的代码，移除 `maxChars` 参数传递

#### Scenario: 卡片召回触发
- **WHEN** 用户发送消息触发卡片召回
- **THEN** 召回笔记的完整内容注入上下文，不再截断

### Requirement: 联网搜索调用方改为读取新设置

系统 SHALL 将联网搜索调用方（`app.go` 中两处）的 `searchMaxChars` 来源从 `GetAIRefMaxChars()` 改为 `Get("ai_web_search_max_chars")`。

#### Scenario: 联网搜索触发
- **WHEN** 用户发送消息触发联网搜索
- **THEN** 搜索结果的截断阈值来自 `ai_web_search_max_chars` 设置，而非旧的 `ai_ref_max_chars`

### Requirement: 前端大文件预览逻辑改为读取新设置

系统 SHALL 将前端大文件预览逻辑中的阈值来源从 `#aiRefMaxChars` 改为 `#aiLargeFilePreviewThreshold`。

- 修改 `main.js` 第 3620 行：`document.getElementById('aiLargeFilePreviewThreshold')?.value`
- 通知文本改为"笔记内容超过纯文本预览阈值，已自动切换为纯文本模式"

#### Scenario: 打开大文件 `.md` 笔记
- **WHEN** 用户打开一篇 `.md` 笔记且内容长度 > `ai_large_file_preview_threshold`
- **THEN** 自动切换为纯文本模式，通知正确