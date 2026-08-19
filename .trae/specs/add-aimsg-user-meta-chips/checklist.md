# Checklist

## 后端数据层
- [x] `AIMessage` 模型新增 `Meta string` 字段（`gorm:"type:text" json:"meta"`）
- [x] `services.Message` 结构体新增 `Meta string` 字段
- [x] `AIService.SaveAIMessage` 构造 models.AIMessage 时透传 `msg.Meta`
- [x] `AIService.SaveAIMessages` 同样位置透传 Meta
- [ ] SQLite `ai_messages` 表已存在 `meta` 列（AutoMigrate 验证）
- [ ] 旧消息 `meta` 列为 NULL 不影响功能

## 后端 Wails 绑定
- [x] `App.SaveAIMessage` 签名增加 `meta string` 第 4 参数
- [x] `services.Message` 构造时透传 meta
- [x] 后端日志记录 meta 长度（不记录内容）
- [x] `App.UpdateAIMessageMeta(msgID uint, meta string) error` 新增绑定，仅更新 meta 字段
- [x] `go vet ./...` 通过
- [x] `go build` 通过

## 前端发送
- [x] `sendUserText` 拼装 `buildUserMessageMeta()` 返回 JSON 字符串
- [x] meta 数组元素符合 spec.md Schema：`{type, ...}`
- [x] 包含 referencedNotes（ref 类型）
- [x] 包含 uploadedFiles（file 类型，含 truncated 字段）
- [x] 包含 activeSkills（skill 类型，含 source/target 翻译配置）
- [x] 包含 roleplayNotes（roleplay 类型，可选）
- [x] 空 meta 时传 `""` 或 `"[]"`，不污染 Content
- [x] 调用 `App.SaveAIMessage` 新增第 4 参数

## 前端渲染
- [x] `addMessage` 用户分支调用 `renderUserMessageWithChips`
- [x] 文本段使用 `createTextNode`（XSS 安全）
- [x] meta 段解析为 `<span class="ai-msg-chip">` DOM
- [x] chip 内含 SVG 图标 + label 标签
- [x] 标签过长截断到 20 字 + `…`
- [x] meta 解析失败降级为纯文本 + warn
- [x] `switchSession` 加载历史消息时透传 `msg.meta`
- [x] 滚动加载更多消息时透传 `msg.meta`

## CSS 样式
- [x] `.ai-msg-user .ai-msg-chip` 基础样式（半透明白底 + 白字 + 圆角）
- [x] `.ai-msg-chip-icon` / `.ai-msg-chip-label` 子元素样式
- [x] 类型微调：ref 用 file-text 图标，file 用 paperclip 图标，skill 用 zap 图标
- [x] `.ai-msg-user .msg-content` 支持 flex 混排（文本 + chip 折行）
- [ ] 14 个主题下 chip 可读性验证

## Meta 字段同步（Task 8）
- [x] `handleRegenerate` 触发后调用 `buildUserMessageMeta()` + `App.UpdateAIMessageMeta`
- [x] `handleResend` / 编辑消息保存路径同样调用 `buildUserMessageMeta()` + `App.UpdateAIMessageMeta`
- [x] 当前工具栏为空时 meta 传 `""`（清空语义）
- [x] 更新成功后同步重渲染该消息 DOM（不需切走切回才看到）
- [x] 更新失败时 warn 日志，不阻塞主流程

## 回归验证
- [x] 前端 `npm run build` 通过
- [x] 后端 `go vet ./...` 通过
- [x] 后端 `go build` 通过
- [x] `wails build` 通过
- [ ] 发带引用消息 → 气泡末尾出现 chip
- [ ] 切换会话再切回 → chip 仍存在
- [ ] 重新生成 → 后端日志确认 LLM 输入不含 meta JSON，且 DB 中该消息 Meta 已更新
- [ ] 编辑消息 → textarea 只有纯文本，保存后 chip 反映当前工具栏状态
- [ ] 编辑时清空所有引用 → 保存后 chip 消失，DB Meta 为空
- [ ] 不带引用消息 → 行为完全等同现状（无视觉变化）
- [ ] 旧消息（meta=NULL）→ 仅显示纯文本，无 chip
- [ ] 多 chip 自动折行布局
