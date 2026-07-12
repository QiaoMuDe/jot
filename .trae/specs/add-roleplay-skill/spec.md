# 角色扮演技能 Spec

## Why
用户希望 AI 以特定人物的身份回答问题。通过「角色扮演」技能，用户可以选择 1~3 篇笔记作为角色设定材料，AI 将严格遵循笔记中的人物设定（性格、背景、说话风格等）来回答，提供沉浸式的角色扮演体验。

## What Changes
- **新增 `skill_roleplay` 技能**：在"更多技能"菜单中新增"角色扮演"条目，点击激活
- **新增角色档案选择器**：激活技能后，在工具栏旁出现独立的角色档案选择组件，点击打开笔记选择器
- **新增 `roleplayNotes` 数据流**：独立于现有的 `referencedNotes`，选择 1~3 篇笔记作为角色设定
- **AISessionConfig 新增 `roleplay_notes` 字段**：持久化角色档案引用，切换会话时恢复
- **system message 注入**：角色扮演模式下，笔记内容以"人物设定"格式注入，而非普通引用上下文的格式
- 与现有引用笔记系统完全解耦，两者可同时使用

## Impact
- Affected code:
  - `internal/models/ai_session_config.go` — 新增 `RoleplayNotes` 字段
  - `internal/database/db.go` — `initBuiltinPrompts()` 新增 `skill_roleplay` 技能提示词
  - `frontend/index.html` — "更多技能"菜单新增"角色扮演"条目
  - `frontend/src/js/ai-chat.js` — 新增 roleplay 相关的变量、UI 组件、逻辑、持久化
  - `frontend/src/css/components/ai-chat.css` — 新增角色档案选择器样式

## ADDED Requirements
### Requirement: 角色扮演技能激活与管理
- **WHEN** 用户在"更多技能"菜单中点击"角色扮演"
- **THEN** 激活该技能，清空其他已激活技能（互斥，与现有技能行为一致）
- **AND** 在工具栏旁显示角色档案选择组件
- **AND** 取消技能时，角色档案选择器隐藏

### Requirement: 角色档案笔记选择
- **WHEN** 用户点击角色档案选择组件
- **THEN** 打开笔记选择器，复用现有选择器 UI（可搜索、可选择笔记本）
- **AND** 限制选择 1~3 篇笔记，超过时提示"最多选择 3 篇角色档案"
- **AND** 选择确认后，组件显示已选笔记标题（溢出省略）

### Requirement: 角色档案持久化
- **WHEN** 用户选择/移除角色档案笔记
- **THEN** 自动保存到会话配置（`roleplay_notes` 字段）
- **AND** 切换会话时，从会话配置恢复角色档案选择状态
- **AND** 新建会话时，角色档案为空

### Requirement: 角色扮演模式下 system message 注入
- **WHEN** 用户发送消息且「角色扮演」技能已激活且有角色档案笔记
- **THEN** system message 中注入的格式为：
  ```
  你正在扮演以下人物角色。请严格遵循人物设定中的性格、背景、知识和说话风格来回答问题。
  如果人物设定不足以应对当前问题，基于设定进行合理推断，但不要编造与设定矛盾的信息。

  === 人物设定 ===

  --- 📄 《笔记标题1》 ---
  笔记内容...

  --- 📄 《笔记标题2》 ---
  笔记内容...

  === 设定结束 ===

  请以该人物的身份回答问题，保持角色的一致性。
  ```
- **AND** 此格式替代原本的引用上下文格式，两者互不干扰
- **AND** 若同时有角色扮演 + 普通引用笔记，system message 中先放角色设定，再放引用上下文

### Requirement: 角色档案选择器 UI
- **WHEN** 「角色扮演」技能激活
- **THEN** 在 AI 聊天工具栏区域（技能 chip 行旁边或下方）显示一个角色档案选择组件
- **AND** 组件包含：
  - 图标（人物头像 SVG）
  - 文字："角色档案: N 篇"（N 为已选数量，0 时显示"未设置"）
  - 点击区域打开笔记选择器
  - 若有已选笔记，鼠标悬停显示笔记标题列表 tooltip

## MODIFIED Requirements
### Requirement: AISessionConfig 模型
`internal/models/ai_session_config.go` 新增字段：
```go
RoleplayNotes string `gorm:"type:text;default:''" json:"roleplay_notes"`
```
- 存储格式：`[{"id":1,"title":"张三设定","notebook_name":"角色"},...]` JSON 数组
- 默认值：空数组 `[]`

### Requirement: SessionConfig 结构体
`internal/services/ai_service.go` 的 `SessionConfig` 结构体新增：
```go
RoleplayNotes string `json:"roleplay_notes"`
```

## REMOVED Requirements
无