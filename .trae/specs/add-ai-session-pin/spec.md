# AI 会话侧栏置顶功能 Spec

## Why

AI 会话侧栏目前不支持会话置顶，用户无法将常用会话固定在列表顶部以便快速访问。需要增加置顶功能，提升高频会话的使用效率。

## What Changes

- 会话表 `ai_sessions` 新增 `is_pinned` 布尔字段
- 后端 `GetAISessions()` 排序逻辑改为：置顶会话按标题升序排在最前，其余按 `updated_at DESC`
- 后端新增 `TogglePinAISession(id)` API，用于切换置顶状态
- 前端每个会话条目的删除按钮（✕）替换为"更多"按钮（⋯），点击弹出下拉菜单，包含"删除会话"和"置顶/取消置顶"两个选项
- 前端渲染时对置顶会话显示置顶图标（📌 或 pin 图标），且视觉上与其他会话区分
- 前端下拉菜单需要定位在按钮附近，避免超出侧栏边界

## Impact

- Affected specs: AI assistant module (会话管理)
- Affected code:
  - `internal/models/ai_session.go` — 新增 `IsPinned` 字段
  - `internal/services/ai_service.go` — 修改 `GetAISessions()` 排序逻辑，新增 `TogglePinAISession()` 方法，`AISessionSummary` 新增 `IsPinned` 字段
  - `app.go` — 新增 `TogglePinAISession()` 绑定方法
  - `frontend/src/js/ai-chat.js` — 修改 `renderSessionList()`，替换删除按钮为下拉菜单，处理置顶/取消置顶操作
  - `frontend/src/css/components/ai-chat.css` — 新增下拉菜单样式、置顶状态样式

## ADDED Requirements

### Requirement: 会话置顶功能

系统 SHALL 支持用户将会话置顶，置顶后的会话始终显示在侧栏列表最前方。

#### Scenario: 置顶会话
- **WHEN** 用户点击会话条目的"更多"按钮，选择"置顶"
- **THEN** 该会话被标记为置顶
- **AND** 该会话立即移动到列表顶部置顶区域
- **AND** 按钮文字变为"取消置顶"

#### Scenario: 取消置顶
- **WHEN** 用户点击已置顶会话的"更多"按钮，选择"取消置顶"
- **THEN** 该会话取消置顶标记
- **AND** 该会话回到按 `updated_at DESC` 排序的正常位置

#### Scenario: 置顶会话排序
- **WHEN** 存在多个置顶会话
- **THEN** 置顶会话之间按标题（`title`）升序排列
- **AND** 置顶会话与普通会话之间有明显分隔

### Requirement: 下拉菜单

系统 SHALL 将会话条目的删除按钮替换为"更多"按钮，点击弹出下拉菜单。

#### Scenario: 下拉菜单交互
- **WHEN** 用户点击"更多"按钮（⋯）
- **THEN** 显示包含"置顶/取消置顶"和"删除会话"两个选项的下拉菜单
- **AND** 点击菜单项后菜单关闭
- **AND** 点击菜单外部区域关闭菜单
- **AND** 菜单定位在按钮附近，不超出侧栏边界

## MODIFIED Requirements

### Requirement: 会话列表排序

**修改前**: 所有会话按 `updated_at DESC` 排序。

**修改后**: 置顶会话优先显示（按标题 `title` ASC 排序），其余会话按 `updated_at DESC` 排序。

### Requirement: 会话列表数据模型

**修改前**: `AISession` 模型无置顶字段，`AISessionSummary` 无置顶字段。

**修改后**: `AISession` 模型新增 `IsPinned bool` 字段，`AISessionSummary` 新增 `IsPinned bool` 字段。
