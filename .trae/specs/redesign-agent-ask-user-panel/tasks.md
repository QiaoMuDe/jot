# Tasks

- [x] Task 1: 后端 selection 协议扩展（ask_user.go）
  - `internal/agent/tools/ask_user.go`：
    - Schema ParamsOneOf 新增 `selection`（string 枚举 `single`/`multiple`，缺省 "single"，说明"single=单选（用户点选项即答）；multiple=多选（用户勾选多项后确认）"）
    - 解析结构体新增 `Selection string`；规范化（非 "multiple" 一律视为 "single"）
    - `ai:ask-user` 事件负载 map 增加 `"selection": sel`
    - Desc 补充：需要多选决策时用 selection="multiple"，选项仍 2-6 个
  - 运行 `go build ./...` 验证编译

- [x] Task 2: 前端 HTML 容器（index.html）
  - `frontend/index.html` 的 `#aiChatInputArea` 内、`#aiChatFollowUpBar` 之前插入静态容器：
    `<div id="aiAskPanel" class="ai-ask-panel" style="display:none;"></div>`
  - 面板内容由 JS 动态填充（不写死选项）

- [x] Task 3: 前端 JS 面板实现（ai-chat.js）
  - **移除** `renderAskCard` 函数及其调用（`ai:ask-user` 监听回调改为调用新函数 `showAskPanel`）
  - 模块级新增面板元素引用 `askPanelEl`（init 时 querySelector `#aiAskPanel`）
  - 新增 `showAskPanel(question, options, selection)`：填充面板 DOM（问句标题 + 选项区 + 自定义输入行 + 多选时确认按钮），display 显示；同会话新问题到达时先清空再填充（替换）
  - 单选模式：选项按钮点击即发 `sendUserText(opt)`，然后 `hideAskPanel()`
  - 多选模式：选项按钮切换勾选态（维护 Set 选中集合 + 对勾图标），"确认提交"按钮发送 `我选择：${selected.join('、')}`（未选且输入为空时轻微提示不发送），然后 `hideAskPanel()`
  - 自定义输入行：Enter/提交发送输入文本，然后 `hideAskPanel()`
  - 新增 `hideAskPanel()`：清空面板内容 + `display:none`
  - **生命周期挂钩**：
    - `startStreaming` 开头调用 `hideAskPanel()`（绕答/新流覆盖）
    - `switchSession` 调用 `hideAskPanel()`
    - 清空会话（clearSession/重置）相关函数调用 `hideAskPanel()`
    - `ai:ask-user` 监听仍注册在 startStreaming 的 unsubs 内、仍只在 isAgentFlow 下生效（保持现状）
  - 历史回放不渲染面板（无改动，正文文本 + 工具折叠已有）
  - 注意：`ai:ask-user` 事件监听位置不变（startStreaming 内），但回调不再依赖 streamingEl/contentDiv

- [x] Task 4: 前端 CSS 面板样式（ai-chat.css）
  - **移除** `.ai-ask-card` 系列样式（含 `ai-ask-card-in` keyframes 与 prefers-reduced-motion 块）
  - 新增 `.ai-ask-panel` 系列：
    - 面板容器：`--card-bg` 底、1px `--border` 边框、圆角 12px、内边距 14px、下边距 10px、accent 左缘 3px 指示条（`border-left` 或伪元素）、淡入动画 0.2s ease-out（`prefers-reduced-motion` 关闭）
    - `.ai-ask-question`：问句标题，`--text-primary` 加粗，下边距 10px
    - `.ai-ask-options`：flex 换行 gap 8px
    - `.ai-ask-option`：选项 chip（`--bg` 底、1px 边框、圆角 8px、内边距 6px 12px）；hover 淡 accent 边框/文字；**选中态** accent 实底 + 白字 + 对勾 SVG（多选模式）
    - `.ai-ask-confirm`：多选确认按钮（`--accent` 底白字、圆角 8px、hover 加深、active 按下反馈），多选模式下显示
    - `.ai-ask-input-row` / `.ai-ask-input` / `.ai-ask-submit`：自定义输入行（flex，输入框 `--bg` 底聚焦 accent 描边，提交按钮 `--accent` 底白字）
  - 主题自适应（复用现有 CSS 变量），14 主题下均可读

- [x] Task 5: 文档更新
  - `internal/agent/TOOLS.md` §7.1：更新为面板交互描述——`ai:ask-user` 负载含 `selection`；面板位于输入区上方 `#aiAskPanel`；单选即发/多选确认/自定义输入；回答后、绕答、切会话时隐藏；历史回放靠正文文本兜底不重现面板
  - `AGENTS.md`：滚动窗口（删除记忆点 1，原 2-10 顺移为 1-9，追加新记忆点 10）——新记忆点内容：ask_user 面板重设计（方案B 输入区上方 + selection 单选多选 + 完成后隐藏 + renderAskCard 移除 + 生命周期规则）

- [x] Task 6: 构建与验证
  - `go build ./...` 通过
  - 前端 `npm run build` 通过
  - 检查：`renderAskCard`/`.ai-ask-card` 已无残留引用；事件名清理列表仍含 `ai:ask-user`；面板显示/隐藏生命周期挂钩完整

# Task Dependencies
- [Task 1] 定稿事件负载结构（selection 字段）
- [Task 3] 依赖 [Task 1]（解析 selection）；[Task 2] 提供容器
- [Task 4] 依赖 [Task 3] 的 DOM 结构与类名约定；可与 [Task 2]/[Task 3] 并行（类名先行约定）
- [Task 5] 依赖 [Task 1]（协议）与 [Task 3]（生命周期规则定稿）
- [Task 6] 依赖 [Task 1]-[Task 5]
