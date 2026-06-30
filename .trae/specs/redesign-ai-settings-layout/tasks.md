# Tasks

- [x] Task 1: 新增 AI 设置独立 CSS 样式
  - 在 `settings-panel.css` 中新增 `.ai-settings-group`、`.ai-setting-item`、`.ai-setting-label`、`.ai-setting-control`、`.ai-setting-hint`、`.ai-group-header` 等 scoped 类
  - 分组卡片样式：`background: var(--card-bg)`、`border-radius: var(--radius-xl)`、`padding: 20px`、`margin-bottom: 16px`
  - 设置项行样式：flex 布局，标签固定宽度 80px，控件弹性填充
  - 开关靠右对齐
  - 子项（如卡片召回条数）缩进 12px
  - 提示文字统一样式
  - SVG 图标占位样式（16×16，颜色继承）

- [x] Task 2: 重构 HTML 结构
  - 将当前 AI 设置从 `font-settings` 容器中取出，改为 3 个分组
  - **Group 1 - API 连接**：API 地址、服务商、API Key、模型
  - **Group 2 - 对话增强**：深度思考（开关）、引用截断（数字输入）、卡片召回（开关 + 缩进条数输入）
  - **Group 3 - 联网搜索**：Tavily API Key（输入 + 显示隐藏 + 测试按钮 + 提示文字）、搜索开关
  - 每个分组使用新的 CSS 类，移除所有 inline style
  - 替换 emoji 图标（👁）为 SVG 图标

- [x] Task 3: 验证 DOM 元素 ID 一致性
  - 确认所有 DOM 元素 ID 保持不变
  - 确认 CSS 类引用一致

# Task Dependencies

- [Task 1] 必须先于 [Task 2]
- [Task 3] 依赖 [Task 2] 完成
