# 重构数据统计卡片并增加 AI 性能统计 Spec

## Why

当前数据管理页面将 7 张不分类别的统计卡片平铺在同一网格中(笔记/标签/回收站/笔记本/AI 会话/AI 消息/数据库大小)，缺乏视觉分组，且缺少 AI 对话性能数据（Token 消耗、响应时间、思考时间）的统计展示。用户需要更清晰的分组结构和更全面的 AI 使用指标。

## What Changes

### 后端 — DataStats 新增 4 个 AI 性能字段

- `internal/services/types.go` 中 `DataStats` 结构体新增字段：
  - `TotalTokens int64` — 所有 AI 消息 Token 消耗总和（JSON: `total_tokens`）
  - `AvgResponseTime float64` — 平均响应总耗时（秒，JSON: `avg_response_time`）
  - `AvgThinkingTime float64` — 平均思维链耗时（秒，JSON: `avg_thinking_time`）
  - `MaxResponseTime float64` — 最长响应总耗时（秒，JSON: `max_response_time`）

### 后端 — AIService 新增聚合查询方法

- `internal/services/ai_service.go` 新增 4 个方法：
  - `SumTokens() (int64, error)` — `SELECT COALESCE(SUM(tokens), 0) FROM ai_messages`
  - `AvgResponseTime() (float64, error)` — `SELECT COALESCE(AVG(total_elapsed), 0) FROM ai_messages WHERE total_elapsed > 0`
  - `AvgThinkingTime() (float64, error)` — `SELECT COALESCE(AVG(thinking_elapsed), 0) FROM ai_messages WHERE thinking_elapsed > 0`
  - `MaxResponseTime() (float64, error)` — `SELECT COALESCE(MAX(total_elapsed), 0) FROM ai_messages`

### 后端 — GetDataStats 接入 AI 性能统计

- `app.go` 中 `GetDataStats()` 在 AI 会话/消息统计之后新增对上述 4 个方法的调用

### 前端 — 统计卡片按分类分组

- 将现有 7 张 + 新增 4 张 = 11 张统计卡片按功能分组为 2 个 Section：
  - **笔记与存储**（5 张）：笔记总数、标签总数、回收站、笔记本数、数据库大小
  - **AI 统计数据**（6 张）：AI 会话、AI 消息、Token 总消耗、平均响应时间、平均思考时间、最长响应时间
- 每个 Section 使用 `.data-section-title` 标题（左侧 accent 竖条装饰）
- 笔记与存储网格：`grid-template-columns: repeat(5, 1fr)`
- AI 统计数据网格：`grid-template-columns: repeat(3, 1fr)`（6 张卡片分 2 行）

### 前端 — 新增 4 张 AI 性能卡片

| 卡片 ID | 标签文字 | 数值格式 |
|---------|---------|---------|
| `statTotalTokens` | Token 总消耗 | 数字，千分位 |
| `statAvgResponseTime` | 平均响应时间 | `X.Xs` |
| `statAvgThinkingTime` | 平均思考时间 | `X.Xs` |
| `statMaxResponseTime` | 最长响应时间 | `X.Xs` |

- 时间值保留 1 位小数
- Token 值使用 `toLocaleString()` 添加千分位逗号

### 前端 — 入场动画适配

- 两个 Section 的卡片各自独立触发交错入场动画（`cardEnter`，80ms 步进延迟）
- 数字递增动画（`animateCountUp`）统一在所有卡片入场完成后启动

## Impact

- 修改模型: `internal/services/types.go`
- 修改服务: `internal/services/ai_service.go`
- 修改绑定: `app.go`
- 修改 HTML: `frontend/index.html`（viewData 区域）
- 修改 CSS: `frontend/src/css/components/data-view.css`
- 修改 JS: `frontend/src/js/data-management.js`、`frontend/src/main.js`
- 不涉及 DB schema 变更（字段已存在）

## ADDED Requirements

### Requirement: 后端 AI 性能聚合查询

系统 SHALL 从 `ai_messages` 表中聚合查询 Token 总和与响应时间统计。

#### Scenario: 无数据时返回零值
- **WHEN** `ai_messages` 表为空
- **THEN** `SumTokens()` 返回 0
- **AND** `AvgResponseTime()` 返回 0
- **AND** `AvgThinkingTime()` 返回 0
- **AND** `MaxResponseTime()` 返回 0

#### Scenario: 有数据时正确聚合
- **WHEN** `ai_messages` 表中有消息记录
- **THEN** `SumTokens()` 返回所有消息 tokens 字段的总和
- **AND** `AvgResponseTime()` 返回所有 `total_elapsed > 0` 消息的平均值
- **AND** `AvgThinkingTime()` 返回所有 `thinking_elapsed > 0` 消息的平均值
- **AND** `MaxResponseTime()` 返回所有消息中 `total_elapsed` 的最大值

### Requirement: 前端按分组展示统计卡片

系统 SHALL 将统计卡片按"笔记与存储"和"AI 统计数据"两组展示，每组有独立标题和网格布局。

#### Scenario: 页面渲染
- **WHEN** 用户打开数据管理页面
- **THEN** 第一个 Section 标题为"笔记与存储"，包含 5 张卡片，5 列网格
- **AND** 第二个 Section 标题为"AI 统计数据"，包含 6 张卡片，3 列网格
- **AND** 两组卡片各自独立执行交错入场动画
- **AND** 数字递增动画在所有卡片入场后统一启动

### Requirement: Token 和时间的格式化展示

#### Scenario: Token 数值展示
- **WHEN** `stats.total_tokens` 值较大（如 1234567）
- **THEN** 卡片中显示为 `1,234,567`（千分位格式）

#### Scenario: 时间数值展示
- **WHEN** `avg_response_time` 为 5.678
- **THEN** 卡片中显示为 `5.7s`
- **WHEN** `avg_thinking_time` 为 2.345
- **THEN** 卡片中显示为 `2.3s`
- **WHEN** `max_response_time` 为 30.912
- **THEN** 卡片中显示为 `30.9s`

## MODIFIED Requirements

### Requirement: 统计卡片布局重构

- **修改前**: 7 张卡片平铺在单一 `.data-stats` 网格（`repeat(7, 1fr)`）
- **修改后**: 按分组拆分为两个独立的网格区域，每组使用 `data-section` + `data-stats` 结构，各自独立网格列数

### Requirement: loadDataStats 函数适配

- **修改前**: 读取 7 个 DOM 引用（`statTotalNotes` 等）
- **修改后**: 读取 11 个 DOM 引用，其中时间类卡片使用 `animateCountUp` 但以格式化字符串显示（需特殊处理时间格式的 count-up）

## REMOVED Requirements

无