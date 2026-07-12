# Tasks

## 任务列表

- [x] Task 1: 后端 — AISessionConfig 模型新增 roleplay_notes 字段
  - [x] 1.1 `internal/models/ai_session_config.go` 新增 `RoleplayNotes string` 字段（type:text, default:''）
  - [x] 1.2 `internal/services/ai_service.go` 的 `SessionConfig` 结构体新增 `RoleplayNotes` 字段
  - [x] 1.3 `CreateDefaultSessionConfig()` 中设置 `roleplay_notes` 默认值为 `[]`
  - [x] 1.4 `SaveSessionConfig()` 中读写 `RoleplayNotes`
  - [x] 1.5 `LoadSessionConfig()` 中读写 `RoleplayNotes`
  - [x] 1.6 `internal/database/db.go` 的 `initBuiltinPrompts()` 新增 `skill_roleplay` 技能提示词

- [x] Task 2: 前端 — "更多技能"菜单新增"角色扮演"条目
  - [x] 2.1 `index.html` 的更多技能菜单中添加 `data-skill="roleplay"` 条目，图标使用面具/人物 SVG
  - [x] 2.2 `ai-chat.js` 的 skill 点击处理中添加 `skill === 'roleplay'` 分支（互斥激活）
  - [x] 2.3 `renderSkillChips()` 中添加 `roleplay` 的 chip 模板渲染

- [x] Task 3: 前端 — 角色档案选择器 UI 组件
  - [x] 3.1 在 `ai-chat.js` 中新增 `roleplayNotes` 变量和 `roleplayCacheContext`
  - [x] 3.2 在工具栏区域渲染角色档案选择组件（`.roleplay-selector`），含图标和已选状态文字
  - [x] 3.3 点击组件时复用 `openNoteRefModal` 打开笔记选择器，限制最大选择 3 篇
  - [x] 3.4 选择确认后更新 `roleplayNotes`，更新组件显示（"角色档案: N 篇"），悬停 tooltip 显示标题
  - [x] 3.5 移除角色档案（单个/全部）时更新组件和 `roleplayNotes`
  - [x] 3.6 `ai-chat.css` 新增 `.roleplay-selector` 相关样式

- [x] Task 4: 前端 — 角色扮演 system message 注入
  - [x] 4.1 新增 `getRoleplayContext()` 函数，调用后端 `GetNoteRefContext` 获取笔记内容
  - [x] 4.2 在 `onSend()` 中检测角色扮演模式：若激活且有 `roleplayNotes`，以"人物设定"格式注入 system message
  - [x] 4.3 同时支持角色扮演 + 普通引用笔记共存

- [x] Task 5: 前端 — 会话配置持久化
  - [x] 5.1 `saveCurrentSessionConfig()` 中新增 `roleplay_notes` 字段保存
  - [x] 5.2 `switchSession()` 中加载 `roleplay_notes` 并恢复组件状态
  - [x] 5.3 `createSession()` 中加载默认的 `roleplay_notes`（空数组）

- [x] Task 6: `go build ./...` 编译验证
- [x] Task 7: `npx vite build` 前端构建验证

# Task Dependencies
- Task 1 → Task 3, Task 4, Task 5（后端模型先定义）
- Task 2 → Task 3（菜单项先加，才能激活）
- Task 3, Task 5 可并行