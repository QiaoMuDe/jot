# Tasks

- [x] Task 1: 后端 AIMessage 模型新增 Meta 字段
  - [x] SubTask 1.1: 在 `internal/models/ai_message.go` 的 `AIMessage` 结构体中新增 `Meta string \`gorm:"type:text" json:"meta"\`` 字段
  - [x] SubTask 1.2: 确认 `internal/database/models.go` 中 `AIMessage` 已在 `AllModels` 列表（已确认），GORM AutoMigrate 启动时自动加列
  - [x] SubTask 1.3: 启动应用或调用 `InitDB` 触发 AutoMigrate，验证 SQLite 中 `ai_messages` 表新增 `meta` 列（TEXT，可空）

- [x] Task 2: 后端 services 层 Message 结构体透传 Meta
  - [x] SubTask 2.1: 在 `internal/services/ai_service.go` 第 22-33 行的 `Message` 结构体中新增 `Meta string \`json:"meta"\`` 字段（位置放在 `ToolCalls` 之后）
  - [x] SubTask 2.2: 在 `AIService.SaveAIMessage` 第 711-725 行构造 `models.AIMessage{}` 时增加一行 `Meta: msg.Meta,`
  - [x] SubTask 2.3: 在 `AIService.SaveAIMessages`（第 757 行起）同样位置增加 `Meta: msg.Meta,`（如存在该函数）

- [x] Task 3: 后端 App.SaveAIMessage Wails 绑定增加 meta 参数
  - [x] SubTask 3.1: 修改 `app.go` 第 3190 行 `App.SaveAIMessage` 签名，从 `(sessionID uint, content string, role string)` 改为 `(sessionID uint, content string, role string, meta string)`
  - [x] SubTask 3.2: 构造 `services.Message{}` 时增加 `Meta: meta`
  - [x] SubTask 3.3: 日志中增加 meta 长度打印（仅长度，不打内容，避免日志膨胀）
  - [x] SubTask 3.4: 验证 `go vet ./...` 和 `go build` 通过

- [x] Task 4: 前端 sendUserText 拼装 meta JSON 并发送
  - [x] SubTask 4.1: 在 `frontend/src/js/ai-chat.js` 的 `sendUserText` 函数（第 2250 行起）中，从 `referencedNotes` / `uploadedFiles` / `activeSkills` / `roleplayNotes` 拼装 meta 数组
  - [x] SubTask 4.2: 数组元素格式：`{type:"ref"|"file"|"skill"|"roleplay", ...}`，字段遵循 spec.md 中定义的 JSON Schema
  - [x] SubTask 4.3: 调用 `App.SaveAIMessage` 时新增第 4 个参数 `metaJson`（空数组时传 `""` 或 `"[]"`，后端按需处理）
  - [x] SubTask 4.4: 添加 `buildUserMessageMeta()` 纯函数，逻辑可独立测试；**该函数在 Task 8 中会被复用**

- [x] Task 8: 重新生成 / 编辑时同步更新 Meta 字段
  - [x] SubTask 8.1: 后端新增 Wails 绑定 `App.UpdateAIMessageMeta(msgID uint, meta string) error`，仅更新 `ai_messages.meta` 字段，其他字段不动
  - [x] SubTask 8.2: 前端在 `handleRegenerate` 触发后、调用 `App.CallAIStreamRegenerate` 之前，调 `buildUserMessageMeta()` 派生新 meta，调 `App.UpdateAIMessageMeta(userMsgID, newMeta)` 写回 DB
  - [x] SubTask 8.3: 前端在 `handleResend` / 编辑消息保存路径中，同样在更新 Content 之后/之前调 `buildUserMessageMeta()` + `App.UpdateAIMessageMeta` 同步 Meta
  - [x] SubTask 8.4: 当前工具栏为空时，meta 传 `""`（或 `"[]"`）以实现"清空"语义
  - [x] SubTask 8.5: 更新成功后，**同步更新前端 DOM**：找到对应消息元素，重新调 `renderUserMessageWithChips` 渲染新 chip 列表（避免切走切回才发现 chip 变了）
  - [x] SubTask 8.6: 更新失败时（如 msgID 无效）记录 warn 日志，不阻塞主流程（不影响 LLM 调用）

- [x] Task 5: 前端 addMessage 用户分支渲染 chip
  - [x] SubTask 5.1: 在 `addMessage` 函数（第 3120 行）用户分支中，调用新增的 `renderUserMessageWithChips(contentEl, content, metaJson)`
  - [x] SubTask 5.2: 实现 `renderUserMessageWithChips(el, content, metaJson)`：原文本段用 `createTextNode` 写入；meta 段解析 JSON 后逐项创建 `<span class="ai-msg-chip ai-msg-chip-{type}">` 元素
  - [x] SubTask 5.3: 实现 `createChipElement(metaItem)`：根据 `type` 渲染对应 SVG 图标 + 文本标签；过长文本截断到 20 字符 + `…`
  - [x] SubTask 5.4: meta 解析失败时降级为纯文本展示，记录 `console.warn` 便于排查
  - [x] SubTask 5.5: 历史消息加载路径（`switchSession` 和滚动加载更多）确保把 `msg.meta` 透传给 `addMessage`

- [x] Task 6: CSS 适配 accent 底色的 chip 样式
  - [x] SubTask 6.1: 在 `frontend/src/css/components/ai-chat.css` 第 99-104 行附近新增 `.ai-msg-user .ai-msg-chip` 基础样式：`display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; border-radius: 10px; background: rgba(255,255,255,0.16); border: 1px solid rgba(255,255,255,0.25); color: #fff; font-size: 0.85em; vertical-align: baseline;`
  - [x] SubTask 6.2: 新增 `.ai-msg-chip-icon` / `.ai-msg-chip-label` 子元素样式
  - [x] SubTask 6.3: 新增 `.ai-msg-chip-ref` / `.ai-msg-chip-file` / `.ai-msg-chip-skill` 类型微调（如 file chip 用 paperclip svg，ref chip 用 file-text svg）
  - [x] SubTask 6.4: 用户消息 `.msg-content` 改为 `display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px 6px;`（允许文本和 chip 混排 + 自动折行）
  - [x] SubTask 6.5: 验证 14 个主题下 chip 可读性（重点是浅色 + 深色 + accent 强对比主题）

- [ ] Task 7: 编译与回归验证
  - [x] SubTask 7.1: 后端 `go vet ./...` + `go build` 通过
  - [x] SubTask 7.2: 前端 `npm run build` 通过
  - [x] SubTask 7.3: `wails build` 完整编译通过（`jot.exe` 已产出）
  - [ ] SubTask 7.4: 启动应用，发送带引用的消息，验证气泡末尾出现 chip
  - [ ] SubTask 7.5: 切换会话再切回，验证 chip 仍存在
  - [ ] SubTask 7.6: 对该消息点击"重新生成"，检查后端日志确认 LLM 输入不含 meta JSON，且 DB 中该消息的 Meta 字段已更新
  - [ ] SubTask 7.7: 编辑该消息，textarea 只显示纯文本，保存后 chip 反映当前工具栏状态
  - [ ] SubTask 7.8: 在编辑时清空所有引用 → 保存后该消息的 chip 消失，DB Meta 字段为空
  - [ ] SubTask 7.9: 不带任何引用的消息行为与现状完全一致（无 chip 视觉变化）

# Task Dependencies
- Task 2 depends on Task 1
- Task 3 depends on Task 2
- Task 4 depends on Task 3
- Task 5 depends on Task 4
- Task 6 depends on Task 5（chip 渲染需要样式才能看到效果）
- Task 7 depends on Task 6
- Task 8 depends on Task 4（复用 `buildUserMessageMeta`）
- Task 7 验证项 7.6/7.7/7.8 depends on Task 8
