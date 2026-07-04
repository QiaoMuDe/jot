# 搜索状态 UI 增强 Spec

## Why

新增的 query 精炼环节在后台执行，前端无感知，用户不知道正在"优化搜索词"，会疑惑为什么点了发送没反应。需要将搜索过程拆分为两个阶段展示（精炼 → 联网搜索），并让搜索指示器可点击弹出关键词详情，提升用户对搜索过程的感知和掌控感。

## What Changes

- **后端修改** `app.go` — 新增 `ai:refined-keywords` 事件传递精炼后的关键词；搜索状态从两态（searching/done）扩展为三态（refining/searching/done）
- **前端修改** `ai-chat.js` — 搜索指示器改为两阶段展示，支持 click-to-dropdown 显示搜索关键词
- **前端修改** `ai-chat.css` — 新增下拉菜单样式

## Impact

- Affected specs: 搜索 Query 精炼功能、联网搜索（Tavily）功能
- Affected code: `app.go`、`frontend/src/js/ai-chat.js`、`frontend/src/css/components/ai-chat.css`

## ADDED Requirements

### Requirement: 三态搜索状态

The system SHALL emit three search statuses instead of two, distinguishing between query refinement and actual web search.

#### Scenario: 精炼阶段
- **WHEN** 用户发送消息且联网搜索开启
- **THEN** 后端立即发射 `ai:search-status` = `"refining"` 事件
- **AND** 前端显示"正在优化搜索词..."动画（旋转图标 + 文字）

#### Scenario: 搜索阶段
- **WHEN** query 精炼完成且即将开始 Tavily 搜索
- **THEN** 后端发射 `ai:refined-keywords` 事件，payload 为精炼后的关键词字符串
- **AND** 后端发射 `ai:search-status` = `"searching"` 事件
- **AND** 前端将动画切换为"正在联网搜索..."，搜索词显示在下拉菜单中

#### Scenario: 搜索完成
- **WHEN** Tavily 搜索完成
- **THEN** 后端发射 `ai:search-status` = `"done"` 事件（已有逻辑不变）
- **AND** 前端将动画替换为打字点（已有逻辑不变）

### Requirement: 搜索关键词下拉菜单

The system SHALL provide a clickable dropdown on the search indicator that displays the current search keywords.

#### Scenario: 点击搜索指示器
- **WHEN** 用户点击搜索指示器（精炼阶段或搜索阶段）
- **THEN** 弹出一个下拉菜单，显示当前搜索关键词
- **AND** 精炼阶段显示"正在提取搜索关键词..." + 旋转动画
- **AND** 搜索阶段显示关键词列表，每个关键词带标签样式

#### Scenario: 收起下拉菜单
- **WHEN** 用户再次点击搜索指示器或点击页面其他区域
- **THEN** 下拉菜单收起

### Requirement: 搜索状态动效

The system SHALL provide distinct visual states for the search phases.

- 精炼阶段：旋转图标（与搜索阶段相同）+ 文字"正在优化搜索词..."
- 搜索阶段：旋转图标 + 文字"正在联网搜索...N 个关键词"
- 点击指示器时下拉菜单以轻微动画展开
- 当 `ai:stream-done` 或首条 `ai:stream-chunk` 到达时，所有搜索状态指示器自动移除

## MODIFIED Requirements

### Requirement: 搜索状态事件

`ai:search-status` 事件的值从 `"searching"` / `"done"` 两态扩展为 `"refining"` / `"searching"` / `"done"` 三态。

新增 `ai:refined-keywords` 事件，在精炼完成后、搜索开始前发射，payload 为纯文本关键词字符串。

### Requirement: createSearchIndicator

`createSearchIndicator` 函数变更为接受状态参数，根据 `refining` / `searching` 状态显示不同文案和交互行为。
