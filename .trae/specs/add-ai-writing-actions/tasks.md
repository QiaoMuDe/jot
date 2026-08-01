# Tasks

- [ ] Task 1: 后端新增 `AITextOperation` 绑定
  - 在 `app.go` 中新增 `AITextOperation(text, operation string) (string, error)` 方法
  - 根据 operation 构造对应的 system prompt（polish/continue/expand/condense/proofread/rewrite/translate/translate-en）
  - 加载当前 AI 配置，调用 `aiService.CallAI(ctx, messages)`
  - 返回处理结果或错误
  - 前置检查：AI 未配置时返回友好错误提示

- [ ] Task 2: 前端 executeAction 改为 async
  - 将 `executeAction` 改为 `async function`
  - 用 `await handler(sourceText)` 调用 handler
  - 错误处理增加 AI 分支：`actionType === 'ai'` 时显示"AI 处理失败"而非"不是合法的"
  - 验证同步操作（格式化/文本转换等）行为不变

- [ ] Task 3: 新建 ai-writing.js 定义 AI 写作操作项
  - 创建 `frontend/src/js/editor-actions/ai-writing.js`
  - 定义 8 个操作项（润色/续写/扩写/缩写/校对/改写/翻译成中文/翻译成英文）
  - 每个操作项 `type: 'ai'`，handler 异步调用 `window.go.main.App.AITextOperation`
  - 无选中文本时抛出错误提示"请先选择要处理的文本"
  - 默认导出 `AI_WRITING_ACTIONS`

- [ ] Task 4: 导入并注册 AI 写作操作项
  - 在 `editor-actions.js` 中导入 `AI_WRITING_ACTIONS`
  - 在 `EDITOR_ACTIONS` 末尾展开 `...AI_WRITING_ACTIONS`

- [ ] Task 5: 实现 AI 处理状态栏 `#aiStatusBar`（CSS + DOM）
  - 在 `editor.css` 中新增 `#aiStatusBar` 样式
  - 位置：`.editor-header` 下方，编辑器内容区上方，与编辑器面板同宽（`padding: 0 12px`）
  - 布局：flex 行，包含 spinner SVG（旋转动画）、文字"AI 处理中..."、flex 占位、取消按钮
  - 取消按钮样式：小型按钮，与现有 `.editor-header-btn` 风格一致
  - 默认隐藏，AI 处理中通过 JS 动态创建/移除

- [ ] Task 6: 实现状态栏的创建/移除和取消逻辑
  - 创建 `createAIStatusBar()` 函数：创建 DOM 元素，追加到 `.editor-header` 之后
  - 创建 `removeAIStatusBar()` 函数：移除 DOM 元素
  - 取消按钮绑定 `window.go.main.App.CancelAIStream()`
  - 在 `executeAction` 的 AI 分支中调用 `createAIStatusBar`
  - 在 Promise `.then()` / `.catch()` / 取消回调中调用 `removeAIStatusBar`
  - 按钮禁用/恢复与 `#aiStatusBar` 同步

# Task Dependencies

- [Task 1] 独立，可并行
- [Task 2] 独立，可并行
- [Task 3] 依赖于 [Task 1]（需确认后端 API 签名）
- [Task 4] 依赖于 [Task 2][Task 3]
- [Task 5] 依赖于 [Task 6]（需确认 DOM 结构）
- [Task 6] 依赖于 [Task 2]（需在 executeAction 中集成）