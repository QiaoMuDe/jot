# 卡片召回按笔记本筛选 Spec

## Why
当前卡片召回功能搜索所有笔记本中的笔记，用户需要能在特定笔记本范围内召回，而非全局搜索。通过在 AI 会话页面提供一个笔记本选择菜单，用户可以精确控制召回范围。

## What Changes
- **模型层**: `AISessionConfig` 新增 `RecallNotebookIDs` 字段（JSON 字符串），`SessionConfig` 同步新增
- **后端服务层**: `SearchFull` 增加 notebook ID 过滤参数；`CardRecallSearch` 接收并传递 notebookID 参数；`SaveSessionConfig`/`LoadSessionConfig`/`CreateDefaultSessionConfig` 同步新字段
- **Wails 绑定层**: `CallAIStream` 新增 `recallNotebookIDs []uint` 参数
- **前端 UI**: 将卡片召回简单 toggle 改为弹出菜单（参考联网搜索多源菜单位置和样式），内置各笔记本复选框
- **前端逻辑**: 选中/取消笔记本时自动保存配置；设置页开关联动全选/全取消

## Impact
- Affected specs: AI 会话配置持久化、卡片召回核心流程
- Affected code: 见下方完整文件清单

## ADDED Requirements

### Requirement: 笔记本选择菜单
The system SHALL provide a pop-up menu on the AI chat toolbar for selecting which notebooks to recall cards from.

#### Scenario: 打开菜单
- **WHEN** user clicks the card recall toggle's dropdown arrow area
- **THEN** a pop-up menu appears above the toggle (similar to 联网搜索 multi-source menu), listing all notebooks with checkboxes

#### Scenario: 单独选中笔记本
- **WHEN** user checks/unchecks a notebook checkbox in the menu
- **THEN** the selection persists via `SaveSessionConfig`; the toggle's global active state updates accordingly

#### Scenario: 开关全选/全取消
- **WHEN** user clicks the toggle switch (right knob area)
  - If turning ON: all notebook checkboxes become checked
  - If turning OFF: all notebook checkboxes become unchecked
- **THEN** the change persists and the settings page toggle syncs

#### Scenario: 设置页联动
- **WHEN** user toggles card recall ON in settings page
- **THEN** current session's menu selects all notebooks and toggle turns active
- **WHEN** user toggles card recall OFF in settings page
- **THEN** current session's menu deselects all notebooks and toggle turns inactive

#### Scenario: 新建会话默认值
- **WHEN** a new AI session is created
- **THEN** if global `ai_card_recall_enabled` is true, all notebooks are selected by default; otherwise none are selected

#### Scenario: 后端过滤召回
- **WHEN** `CardRecallSearch` is called with non-empty `notebookIDs`
- **THEN** only notes belonging to those notebooks are included in the search results
- **WHEN** `notebookIDs` is nil/empty (legacy data)
- **THEN** all notebooks are searched (backward compatible)

### Requirement: 数据持久化
The system SHALL persist selected notebook IDs per AI session.

#### Scenario: 保存配置
- **WHEN** `SaveSessionConfig` is called
- **THEN** `recall_notebook_ids` field is saved as JSON array string (e.g. `"[1,2,5]"`) in `ai_session_configs` table

#### Scenario: 加载配置
- **WHEN** session config is loaded (session switch / page load)
- **THEN** `recall_notebook_ids` is deserialized and applied to the menu checkboxes; if empty/null, defaults to "all notebooks selected" when card recall is enabled

## MODIFIED Requirements

### Requirement: 卡片召回核心流程（修改）
- `CardRecallSearch` 新增 `notebookIDs ...uint` 参数
- `SearchFull` 新增 `notebookIDs ...uint` 参数，非空时加 `WHERE notebook_id IN (?)` 过滤
- `CallAIStream` 新增 `recallNotebookIDs []uint` 参数，从 `AISessionConfig.RecallNotebookIDs` 读取后传入

## REMOVED Requirements
无

## 涉及的文件清单

| 层级 | 文件 | 改动量 | 说明 |
|---|---|---|---|
| 模型 | `internal/models/ai_session_config.go` | +1 行 | 新增 `RecallNotebookIDs` 字段 |
| 服务 | `internal/services/ai_service.go` | ~8 行 | `SessionConfig` + 3 个 CURD 函数同步 |
| 服务 | `internal/services/recall_service.go` | ~5 行 | `CardRecallSearch` 参数 + 传递给 `SearchFull` |
| 服务 | `internal/services/note_service.go` | ~6 行 | `SearchFull` 加 notebookID IN 过滤 |
| 绑定 | `app.go` | ~5 行 | `CallAIStream` 参数 + 向下传递 |
| HTML | `frontend/index.html` | ~20 行 | 弹出菜单 HTML 结构 |
| JS | `frontend/src/js/ai-chat.js` | ~60 行 | 菜单交互 + 配置读写 |
| CSS | `frontend/src/css/components/ai-chat.css` | ~20 行 | 菜单样式（参照联网搜索菜单复用） |
