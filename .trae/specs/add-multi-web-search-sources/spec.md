# 多源联网搜索功能 Spec

## Why

目前仅支持 Tavily 单一联网搜索源。知乎开放平台提供了「知乎搜索」和「全网搜索」两个 API，结合已有 Tavily 搜索，为用户提供更多样化的信息检索渠道，提升 AI 助手的实时信息获取能力。

同时，现有的搜索动画仅适配单一来源，且搜索失败时后端静默跳过、前端无任何反馈，用户体验不佳。需要：
- 搜索动画改为多源分阶段展示，每个搜索源有独立的状态（进行中/成功/失败）
- 后端搜索失败时向前端发送错误事件，前端展示错误提示

## What Changes

- **后端新增依赖** — 引入 `gitee.com/MM-Q/zhihu-go` SDK（知乎开放平台 Go SDK，零外部依赖）
- **后端新增** `zhihu_search_service.go` — 封装知乎搜索和全网搜索 API 调用
- **后端修改** `search_service.go` — `SearchSource` 新增 `SourceLabel` 字段，统一搜索源接口
- **后端修改** `types.go` — `SettingsConfig` 新增 `ZhihuAccessSecret`、`ZhihuSearchEnabled`、`ZhihuGlobalSearchEnabled`、`TavilySearchEnabled` 字段
- **后端修改** `ai_service.go` — `CallAIStream` 新增 `searchSources []string` 参数，支持多源并行搜索
- **后端修改** `app.go` — `CallAIStream` 签名修改，新增测试知乎连接的方法，新增搜索源绑定
- **后端修改** `app.go` — 搜索失败时通过 `ai:search-error` 事件向前端推送错误信息（含来源标识和错误原因）
- **后端修改** `db.go` — 默认设置新增知乎相关配置项
- **前端设置页** — 新增「知乎 Access Secret」输入框、三个独立开关（知乎搜索/知乎全网搜索/Tavily搜索）
- **前端 AI 聊天栏** — 「联网搜索」按钮改为下拉菜单，内含三个复选框（知乎搜索/全网搜索/Tavily搜索）
- **前端搜索动画** — 重新设计为多源分阶段动画：精炼关键词 → 各源依次/并行搜索（每源显示独立状态）→ 汇总完成
- **前端错误处理** — 监听 `ai:search-error` 事件，在搜索动画区域显示哪个源失败及原因，不阻塞其他源
- **前端 JS** — `saveSettings`/`loadSettings` 同步新设置项，`ai-chat.js` 支持多搜索源选择和错误展示

## Impact

- Affected specs: AI 助手（联网搜索）、设置页
- Affected code: `internal/services/`（新增 + 修改）、`internal/database/db.go`、`app.go`、`frontend/index.html`、`frontend/src/main.js`、`frontend/src/js/ai-chat.js`、`frontend/src/css/components/ai-chat.css`、`go.mod`

## ADDED Requirements

### Requirement: 知乎搜索后端服务

The system SHALL provide a search service that queries Zhihu's open platform APIs.

#### Scenario: 知乎搜索成功
- **WHEN** 用户启用了知乎搜索并发送消息，且 Zhihu Access Secret 已配置
- **THEN** 后端调用 zhihu-go 的 `SearchZhihu` API，将搜索结果注入 system message
- **AND** 后端发射 `ai:search-source-status` 事件携带来源标识 `zhihu_search` 和状态 `success`

#### Scenario: 知乎全网搜索成功
- **WHEN** 用户启用了知乎全网搜索并发送消息，且 Zhihu Access Secret 已配置
- **THEN** 后端调用 zhihu-go 的 `SearchGlobal` API，将搜索结果注入 system message
- **AND** 后端发射 `ai:search-source-status` 事件携带来源标识 `zhihu_global` 和状态 `success`

#### Scenario: Zhihu Access Secret 未配置
- **WHEN** 用户启用了知乎搜索或知乎全网搜索，但 Access Secret 为空
- **THEN** 后端发射 `ai:search-error` 事件，携带来源标识和错误信息「知乎 Access Secret 未配置」
- **AND** 该搜索源静默跳过，不阻塞对话和其他搜索源

#### Scenario: 知乎 API 调用失败
- **WHEN** Zhihu API 调用超时或返回错误
- **THEN** 后端发射 `ai:search-error` 事件，携带来源标识和具体错误信息
- **AND** 该搜索源跳过，不阻塞对话和其他搜索源

### Requirement: 搜索错误通知

The system SHALL propagate search errors to the frontend for user visibility.

#### Scenario: 单个搜索源失败
- **WHEN** 某个搜索源（如 Tavily）调用失败
- **THEN** 后端通过 `ai:search-error` 事件发射 `{source: "tavily", error: "Tavily 搜索失败: API 返回 401"}` 
- **AND** 前端在搜索动画区域显示该源失败状态（红色的 × 图标 + 错误简短提示）
- **AND** 其他搜索源继续执行，不受影响

#### Scenario: 所有搜索源均失败
- **WHEN** 所有启用的搜索源均调用失败
- **THEN** 前端搜索动画显示「所有搜索源均失败，将仅使用 AI 自身知识回复」
- **AND** AI 流式调用继续执行（不阻塞）

### Requirement: 多源并行搜索

The system SHALL support multiple search sources running concurrently.

#### Scenario: 启用多个搜索源
- **WHEN** 用户同时启用了知乎搜索和 Tavily 搜索
- **THEN** 多个搜索源并行执行，所有结果汇总后一并注入 system message
- **AND** 每个搜索源的结果分别标注来源标签（zhihu_search/zhihu_global/tavily）

#### Scenario: 搜索结果显示来源
- **WHEN** 多个搜索源返回结果
- **THEN** 前端展示搜索结果时每条结果标注来源（知乎搜索/全网搜索/Tavily搜索）
- **AND** 搜索来源折叠面板按来源分组展示

### Requirement: 搜索结果数限制统一

The system SHALL apply the same search result limit to all search methods.

#### Scenario: 搜索结果数限制
- **WHEN** 用户设置了搜索结果数为 N
- **THEN** Tavily 搜索结果数限制为 N 条
- **AND** 知乎搜索限制为 N 条
- **AND** 知乎全网搜索限制为 N 条

### Requirement: 多源搜索动画

The system SHALL display a multi-stage search animation that shows per-source status.

#### Scenario: 搜索动画流程
- **WHEN** 用户发送消息且启用了多个搜索源
- **THEN** 搜索动画按以下阶段展示：
  1. **精炼阶段** — 地球图标 + 「正在优化搜索词...」（与现有一致）
  2. **搜索阶段** — 以列表形式展示每个搜索源的独立状态：
     - 知乎搜索: [旋转图标] 正在搜索...
     - 全网搜索: [旋转图标] 正在搜索...
     - Tavily搜索: [旋转图标] 正在搜索...
  3. **完成阶段** — 各源状态更新为：
     - 知乎搜索: [绿色✓] 搜索完成 (X 条结果)
     - 全网搜索: [红色✗] 搜索失败: 原因
     - Tavily搜索: [绿色✓] 搜索完成 (X 条结果)
  4. **汇总** — 所有搜索源完成后，切换为打字点动画等待 LLM 响应

#### Scenario: 点击搜索动画展开详情
- **WHEN** 用户点击搜索动画区域
- **THEN** 下拉展开详情面板，显示每个搜索源的状态、结果数、错误信息（如有）

### Requirement: 前端多搜索源选择 UI

The system SHALL replace the single web search toggle with a multi-source selection dropdown.

#### Scenario: 点击联网搜索按钮
- **WHEN** 用户点击 AI 聊天工具栏的「联网搜索」按钮
- **THEN** 弹出下拉菜单，包含三个复选框：「知乎搜索」「全网搜索」「Tavily搜索」
- **AND** 复选框独立选择，至少选一个时才认为联网搜索已启用
- **AND** 任意一个被选中时，按钮显示为激活状态

#### Scenario: 设置页同步
- **WHEN** 用户在设置页中开启/关闭某个搜索源
- **THEN** AI 聊天栏的复选框状态同步更新
- **AND** 反之亦然

### Requirement: 设置页知乎配置

The system SHALL provide Zhihu-related configuration in the settings page.

#### Scenario: 知乎 Access Secret 配置
- **WHEN** 用户打开设置页的「对话与搜索」区域
- **THEN** 显示「知乎 Access Secret」密码输入框，支持显示/隐藏切换
- **AND** 显示「测试连接」按钮验证 Secret 有效性

#### Scenario: 搜索源独立开关
- **WHEN** 用户在设置页中操作
- **THEN** 看到三个独立开关：「知乎搜索」「全网搜索」「Tavily搜索」
- **AND** 每个开关独立控制对应搜索源的默认启用状态

## MODIFIED Requirements

### Requirement: CallAIStream 流式调用

- `CallAIStream` 签名中 `searchEnabled bool` 改为 `searchSources []string`（数组，包含启用的搜索源标识：`"zhihu_search"`, `"zhihu_global"`, `"tavily"`）
- 当数组非空时，并行执行所有启用的搜索
- 每个搜索源独立执行，失败时不阻塞其他源
- 搜索失败时通过 `ai:search-error` 事件发射 `{source: string, error: string}` 给前端
- 每个搜索源完成时通过 `ai:search-source-status` 事件发射 `{source: string, status: string, count: number}` 给前端
- 搜索结果统一注入 system message

### Requirement: SettingsConfig 配置结构

`SettingsConfig` 结构体新增字段：
- `ZhihuAccessSecret string json:"zhihu_access_secret"`
- `ZhihuSearchEnabled bool json:"zhihu_search_enabled"`
- `ZhihuGlobalSearchEnabled bool json:"zhihu_global_search_enabled"`
- `TavilySearchEnabled bool json:"tavily_search_enabled"`（替代原有的 `AIWebSearchEnabled`）

### Requirement: AIConfig 配置结构

`AIConfig` 结构体新增 `ZhihuAccessSecret string json:"zhihu_access_secret"` 字段。

### Requirement: 搜索状态事件流

后端事件流从：
```
ai:search-status → "refining" → "searching" → "done"
```
改为：
```
ai:search-status → "refining"
ai:refined-keywords → "关键词"
ai:search-source-status → {source: "zhihu_search", status: "searching"}
ai:search-source-status → {source: "tavily", status: "searching"}
ai:search-source-status → {source: "zhihu_search", status: "success", count: 5}
ai:search-error → {source: "tavily", error: "Tavily 搜索失败: 401 Unauthorized"}
ai:search-status → "done"
```

## REMOVED Requirements

### Requirement: AIWebSearchEnabled 单一开关

**Reason**: 单一「联网搜索」开关被拆分为三个独立的搜索源开关（知乎搜索、全网搜索、Tavily搜索），需要更细粒度的控制。
**Migration**: `AIWebSearchEnabled` 字段由 `TavilySearchEnabled` + 新的两个开关替代。前端 UI 从单个 toggle 改为按钮+下拉复选框菜单。

### Requirement: 搜索静默失败

**Reason**: 用户要求搜索失败时不能静默处理，必须在前端展示错误提示。
**Migration**: 新增 `ai:search-error` 事件，后端搜索失败时向前端推送错误信息。前端新增错误展示 UI。
