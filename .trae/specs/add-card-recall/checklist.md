# Checklist

## 后端核心逻辑
- [ ] `recall_service.go` 存在且实现 `tokenize2Gram` 分词函数
- [ ] `recall_service.go` 定义 `RecallCard` 和 `CardRecallResult` 结构体
- [ ] `CardRecallSearch` 返回格式化的上下文文本 + 结构化卡片数据
- [ ] 2-gram 分词正确处理中文、英文、数字混合输入，去重
- [ ] `note_service.go` `SearchFull` 方法签名正确，多关键词 OR LIKE 搜索
- [ ] `SearchFull` 返回完整笔记内容（非 200 字截断），按 `updated_at DESC` 排序

## 后端集成
- [ ] `CallAIStream` 签名新增 `cardRecallEnabled bool`，前后端一致
- [ ] 卡片召回在 goroutine 内执行，不阻塞 Wails 事件循环
- [ ] 卡片召回在联网搜索之后执行（如同时启用）
- [ ] 召回格式化文本成功注入 system message
- [ ] 有匹配时通过 `runtime.EventsEmit("ai:recall-cards", ...)` 发射结构化 JSON
- [ ] 无匹配笔记时静默跳过，不报错，不发射事件
- [ ] 卡片召回错误不阻塞主 AI 流式调用
- [ ] ctx 取消可终止卡片召回搜索

## 前端设置页
- [ ] 设置页「卡片召回」开关存在（`#aiSettingCardRecallToggle`），复用 `.ai-setting-search-line` + `.ai-chat-toggle-switch` 样式
- [ ] 设置页「召回条数」输入框存在（`#aiSettingCardRecallLimit`，默认 3）
- [ ] `loadAISettings()` 正确恢复卡片召回开关状态（`localStorage.ai_card_recall_enabled`）
- [ ] `loadAISettings()` 正确恢复召回条数值（`localStorage.ai_card_recall_limit`）

## 前端工具栏
- [ ] AI 工具栏「卡片召回」开关存在（`#aiChatCardRecallToggle`），复用 `.ai-chat-search-toggle` + `.ai-chat-toggle-switch` 样式
- [ ] 工具栏开关点击切换 active class，保存 localStorage，同步设置页 toggle
- [ ] 设置页 toggle 点击同步工具栏开关

## 前端流式调用
- [ ] `startStreaming()` 中调用 `window.go.main.App.CallAIStream(..., enableCardRecall)` 传递开关状态

## 召回卡片展示
- [ ] `ai-chat.js` 新增 `recallCards` 变量和 `ai:recall-cards` 事件监听
- [ ] AI 回复完成后，有召回卡片时展示 `<details class="recall-cards">` 折叠面板
- [ ] 折叠面板显示「📄 召回笔记 (N 篇)」
- [ ] 每张卡片显示标题（可点击）和内容摘要（前 100 字，3 行截断）
- [ ] 无召回卡片时不显示折叠面板
- [ ] 流完成/错误时清空 `recallCards`

## 卡片预览浮层
- [ ] 预览浮层 HTML 存在（`#aiCardPreviewModal`），默认隐藏
- [ ] 点击卡片标题打开预览浮层，展示标题 + 完整内容
- [ ] 点击关闭按钮或遮罩层关闭浮层
- [ ] 浮层固定居中，最大高度 80vh，内容可滚动

## CSS 样式
- [ ] `.recall-cards` 系列样式复用 `.search-sources` 样式体系（不同 class name）
- [ ] `.ai-card-preview-modal` / `.ai-card-preview-overlay` / `.ai-card-preview-panel` 模态浮层样式存在
- [ ] 项目可正常编译（`go build .` 通过）
- [ ] AGENTS.md 已记录卡片召回功能
