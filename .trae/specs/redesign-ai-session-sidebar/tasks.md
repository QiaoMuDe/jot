# Tasks

## 任务依赖
- 所有任务均为基础 CSS 改动，无相互依赖，可并行执行

## 任务 1：调整侧边栏宽度为 230px
- [x] 修改 `.ai-session-sidebar` 的 `width` 从 `220px` → `230px`
- [x] 修改 `.ai-session-sidebar:not(.collapsed) ~ .ai-sidebar-toggle` 的 `left` 从 `220px` → `230px`

## 任务 2：重设 Header 区域间距和样式
- [x] 修改 `.ai-session-sidebar-header` 的 padding 从 `10px 12px` → `8px 10px`
- [x] 修改 `.ai-session-sidebar-title` 的字号从 `0.8rem` → `0.82rem`
- [ ] `.ai-session-new-btn` 保持现有尺寸和 hover 样式（已符合 spec）

## 任务 3：压缩搜索框区域 padding
- [x] 修改 `.ai-session-search-wrap` 的 padding 从 `8px 10px 8px` → `6px 10px 4px`
- [x] 修改 `.ai-session-search` 的 padding 从 `5px 8px` → `4px 8px`
- [ ] 搜索框 `::placeholder` 颜色使用 `var(--text-muted)`（已有此规则，已验证）

## 任务 4：为活跃会话条目添加左侧 accent 竖条指示器
- [x] 为 `.ai-session-item.active` 添加 `::before` 伪元素：绝对定位，`left: 0`, `top: 2px`, `bottom: 2px`, `width: 2px`, `background: var(--accent)`, `border-radius: 0 1px 1px 0`
- [x] 确保 `.ai-session-item` 的 `position: relative` 已设置（已有）
- [x] 为 `.ai-session-item.active` 的 `border` 添加 `overflow: hidden` 或调整，确保伪元素不被 `border-radius` 遮盖

## 任务 5：压缩 Footer 区域 padding
- [x] 修改 `.ai-session-sidebar-footer` 的 padding 从 `8px 10px` → `6px 10px`

## 任务 6：优化空状态样式
- [x] 给 `.ai-session-empty`（或动态生成的空状态文本）添加居中样式：`text-align: center`, `color: var(--text-muted)`, `font-size: 0.82rem`, `padding-top: 24px`
- [ ] 确认 JS 中空状态 HTML 结构（现有：`<div class="ai-session-empty">暂无会话</div>`）

## 任务 7：验证
- [x] 检查所有改动在不同主题（default/light/dark/nord/monokai-pro/tokyo-night 等）下的兼容性
- [x] 确认活跃条目左侧指示器正确显示
- [x] 确认所有 padding 调整后无布局溢出
- [x] 确认删除按钮功能正常
- [x] 确认侧边栏折叠/展开交互正常
