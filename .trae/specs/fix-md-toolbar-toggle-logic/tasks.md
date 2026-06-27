# Tasks

- [x] Task 1: 修复标题下拉面板透明 — 检查并修正 `.heading-dropdown-panel` 的 CSS 背景/边框/阴影，确保在所有 6 个主题下正确可见
  - [x] 定位到 `editor.css:209` 查看 `.heading-dropdown-panel` 现有样式
  - [x] 确认 `var(--bg-primary)` 在所有主题下均非透明
  - [x] 确保 `.heading-dropdown-item` 文字和 hover 效果正确显示
  - [x] 用 `npm run build` 验证构建无错误

- [x] Task 2: 重写格式切换逻辑 — `formatBold()`/`formatItalic()`/`formatStrikethrough()`/`formatInlineCode()` 4 个函数改为智能切换（已包裹则 unwrap，否则 wrap）
  - [x] 重写 `formatBold()` — 检测 `**` 包裹，切换 wrap/unwrap
  - [x] 重写 `formatItalic()` — 检测 `*` 包裹，切换 wrap/unwrap
  - [x] 重写 `formatStrikethrough()` — 检测 `~~` 包裹，切换 wrap/unwrap
  - [x] 重写 `formatInlineCode()` — 检测 `` ` `` 包裹，切换 wrap/unwrap
  - [x] 用 `npm run build` 验证构建无错误

# Task Dependencies

- Task 1 和 Task 2 互不依赖，可并行执行
