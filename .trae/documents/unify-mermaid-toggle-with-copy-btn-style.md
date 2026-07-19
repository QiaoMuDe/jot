# 统一 Mermaid 渲染按钮样式与复制按钮一致

## 总结

将 `.mermaid-toggle` 的尺寸、间距、圆角、背景、布局等所有视觉属性与 `.copy-code-btn` 完全统一，仅保留 `right` 定位值和 `z-index` 差异。

## 当前状态分析

[editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css) 中两个按钮存在 10 项差异（见下方表格），导致渲染按钮偏大、偏亮、圆角不一致。

## 具体修改

仅修改 [editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css) 中 `.mermaid-toggle` 的 CSS 规则（L1328-1357）。

### 修改清单

| 属性 | 修改前 | 修改后 | 说明 |
|---|---|---|---|
| `padding` | `4px 10px` | `2px 8px` | 与复制按钮完全一致 |
| `line-height` | `1.4` | `1.6` | 与复制按钮完全一致 |
| `border-radius` | `5px` | `var(--radius-sm)` | 使用 CSS 变量，与主题联动 |
| `background` | `var(--card-bg)` | `color-mix(in srgb, var(--card-bg) 85%, transparent)` | 半透明毛玻璃效果 |
| `backdrop-filter` | 无 | `blur(4px)` | 补齐毛玻璃效果 |
| `display` | 无 | `inline-flex` | 弹性布局居中 |
| `gap` | 无 | `4px` | 图标与文字间距 |
| `min-width` | 无 | `62px` | 防宽度跳动 |
| `justify-content` | 无 | `center` | 内容居中 |
| `text-align` | 无 | `center` | 文字居中 |
| `transition` | `opacity 0.15s, transform 0.15s ease, background 0.15s, color 0.15s, border-color 0.15s` | `opacity 0.15s, background 0.15s, transform 0.15s` | 简化为与复制按钮一致，去掉不必要的 color/border-color 过渡 |
| `z-index` | `3` | `2` | 减少为比复制按钮的 `1` 稍高即可（保持点击优先级） |

### 保留不变的属性

- `position: absolute` — 相同
- `top: 8px` — 已对齐
- `right: 72px` — 定位差异，保留
- `border: 1px solid var(--border)` — 相同
- `color: var(--text-muted)` — 相同
- `cursor: pointer` — 相同
- `opacity: 0` — 相同
- `user-select: none` — 相同
- `white-space: nowrap` — 相同
- `font-size: 12px` — 相同
- `font-family: inherit` — 相同
- hover 后的 `background`, `color`, `border-color`, `transform` — 已与复制按钮一致，无需改动
- hover 显隐规则（`.pre-wrapper.has-mermaid:hover .mermaid-toggle`）— 保持不变

## 涉及文件

仅 `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\editor.css`，无 JS 变更。

## 验证步骤

1. `cd frontend && npx vite build` — 构建无错误
2. 运行应用，对比笔记预览中复制按钮和渲染按钮的视觉一致性
3. 运行应用，对比 AI 消息中两者的一致性
4. 点击渲染按钮触发渲染，检查 SVG 显示正常
