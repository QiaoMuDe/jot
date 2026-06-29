# 持久化 AI 会话与多会话支持 Spec

## Why
当前 AI 对话是临时的（`chatHistory` 内存数组），刷新页面或切换视图后对话丢失。需要持久化到 SQLite，支持创建多会话、切换会话、删除会话，让用户可以保留和管理多条 AI 对话记录。

## What Changes
- **NEW**: `AISession` 和 `AIMessage` 两个 GORM 模型，持久化到 `jot.db`
- **NEW**: `ai_service.go` 新增会话 CRUD + 消息保存方法
- **NEW**: `app.go` 新增 5 个 Wails 绑定（会话列表/创建/删除/重命名/加载消息）
- **MODIFIED**: `database/db.go` 新增 `AutoMigrate` 表
- **NEW**: `index.html` 在 AI 助手视图左侧新增会话侧栏
- **MODIFIED**: `ai-chat.js` 全部重写会话管理逻辑（加载/切换/新建/删除/自动存档）
- **NEW**: `ai-chat.css` 新增会话侧栏样式

## Impact
- Affected specs: `replace-quikchat-with-custom-ai-chat`（已完成，本 spec 在其基础上扩展）
- Affected code:
  - `internal/models/`：新增 `ai_session.go` + `ai_message.go`
  - `internal/database/db.go`：追加 `AutoMigrate`
  - `internal/services/ai_service.go`：新增方法
  - `app.go`：新增绑定
  - `frontend/index.html`：AI 助手视图改为左右分栏布局
  - `frontend/src/js/ai-chat.js`：重写
  - `frontend/src/css/components/ai-chat.css`：新增

## ADDED Requirements

### Requirement: 数据模型
系统 SHALL 在 SQLite 中持久化 AI 会话和消息。

#### AISession 表
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint (PK) | 自增主键 |
| Title | string(100) | 会话标题，自动生成（取首条用户消息前 30 字） |
| CreatedAt | time | 创建时间 |
| UpdatedAt | time | 更新时间（最后一条消息的时间） |

#### AIMessage 表
| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint (PK) | 自增主键 |
| SessionID | uint (FK→AISession.ID, index) | 所属会话 |
| Role | string(20) | user / assistant |
| Content | text | 消息内容 |
| ReasoningContent | text | 思考链内容（可选） |
| CreatedAt | time | 创建时间 |

### Requirement: 后端 API

#### GetAISessions() — 获取会话列表
- **WHEN** 用户打开 AI 助手页面
- **THEN** 返回所有会话按 `updated_at DESC` 排序
- **AND** 每个会话附带最后一条消息内容作为摘要

#### CreateAISession() uint — 创建新会话
- **WHEN** 用户点击「新建会话」
- **THEN** 创建空会话（title 默认为 `"新对话"`），返回会话 ID
- **AND** 前端自动切换到新会话

#### DeleteAISession(id uint) — 删除会话
- **WHEN** 用户删除某个会话
- **THEN** 级联删除该会话下的所有消息
- **AND** 如果当前激活的会话被删除，切换到最近会话或显示空状态

#### RenameAISession(id uint, title string) — 重命名会话
- **WHEN** 用户双击会话标题
- **THEN** 进入编辑状态，回车或失焦时保存新标题

#### LoadAISessionMessages(id uint) — 加载会话消息
- **WHEN** 用户切换到某个会话
- **THEN** 返回该会话所有消息（按 `created_at ASC` 排序）

### Requirement: 流式输出后自动保存
- **WHEN** AI 流式输出完成（`onDone` 回调）
- **THEN** 自动将该轮对话的 user 消息和 assistant 消息保存到数据库
- **AND** 更新会话的 `title`（如果是第一次回复，取第一条 user 消息前 30 字）
- **AND** 更新会话列表 UI（标题 + 时间）

### Requirement: 前端会话侧栏
```
┌─────────────┬──────────────────────────┐
│  会话侧栏    │                          │
│  48px 宽     │                          │
├─────────────┤    消息列表 + 输入区      │
│  [新建] 按钮  │     （现有布局）          │
│ ○ 会话 1     │                          │
│ ○ 会话 2     │                          │
│ ○ 会话 3     │                          │
│             │                          │
│  清空当前    │                          │
└─────────────┴──────────────────────────┘
```
- 左侧会话列表，右侧保持现有消息列表 + 输入区
- 每个会话项显示标题 + 删除按钮（hover 可见）
- 当前会话高亮（`.active`）
- 点击会话项 → 切换并加载消息
- 新建按钮 → 创建空会话
- 双击会话标题 → 内联编辑重命名
- 会话项按 `updated_at` 降序排列

### Requirement: 切换视图后保留状态
- **WHEN** 用户从 AI 助手切换到其他视图再切回来
- **THEN** 自动恢复最后一次激活的会话及其消息列表

## MODIFIED Requirements

### Requirement: 清空对话（现有）
- **MODIFIED**: 清空当前会话时，删除数据库中该会话的所有消息，但不删除会话本身
- 清空后会话保留在列表中（标题不变），方便重新开始对话

## REMOVED Requirements
无
