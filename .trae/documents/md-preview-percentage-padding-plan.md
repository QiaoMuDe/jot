# MD 预览模式百分比留白方案

## 概述

为 `.md` 后缀笔记的预览模式（`.md-rendered`）添加百分比左右留白，使阅读时内容不贴边框，且普通模式和全屏模式享受等比例舒适的阅读效果。

## 当前状态

`.md-rendered` 的 padding 当前为固定值：

```css
.md-rendered {
  padding: 0.5em 0.5rem 1rem 1.25rem;
}
```

* `right: 0.5rem`（\~8px）— 右边距偏小

* `left: 1.25rem`（\~20px）— 左边距稍大

* 左右不对称，且均为固定值，全屏模式下不缩放

## 修改方案

### 文件：`frontend/src/css/components/editor.css`

**位置：** `.md-rendered` 的 `padding` 属性（第 1009 行）

**修改：** 将左右 padding 从固定值改为百分比 `8%`

```css
/* 修改前 */
padding: 0.5em 0.5rem 1rem 1.25rem;

/* 修改后 */
padding: 0.5em 8% 1rem;
```

效果：

* `top: 0.5em` — 保持不变

* `left/right: 8%` — 等比例留白，相对父容器宽度

* `bottom: 1rem` — 保持不变

### 百分比 8% 的取值依据

| 面板宽度             | 单侧留白    | 内容区宽度    | 体验      |
| ---------------- | ------- | -------- | ------- |
| 560px（普通模式）      | \~45px  | \~470px  | 舒适阅读宽度  |
| 100vw（全屏 1920px） | \~154px | \~1612px | 宽屏留白自然  |
| 90vw（窄屏 360px）   | \~29px  | \~302px  | 小屏不浪费空间 |

### 兼容性说明

* `.editor-overlay[data-mode="preview"] .md-rendered` 仅覆盖 `padding-top: 0.1em`，左右不受影响

* `.editor-panel.fullscreen` 只改变面板尺寸，百分比 padding 自动适配

* TOC 侧栏展开时（`.toc-sidebar.toc-visible`），`.md-rendered` 宽度自适应减少，padding 百分比相对剩余空间计算，表现自然

## 验证步骤

1. 构建前端：`cd frontend && npm run build`
2. 运行应用，打开 `.md` 笔记
3. 切换到预览模式，确认左右留白对称且舒适
4. 切换到全屏模式，确认留白等比例放大
5. 切换到普通模式，确认留白等比例缩小

