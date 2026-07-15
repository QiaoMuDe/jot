# 修复 topbar 品牌标识动画（v2 — 改用 transform）

## 摘要

当前 `padding-left` 过渡不稳定（品牌标识"立马到左侧"）。改用 `transform: translateX()` 驱动 `.topbar-left` 整体移动，GPU 加速，无布局抖动。

## 问题分析

### 当前方案（失败原因）

`#topbar` 的 `padding-left: 24px → 4px` 是**布局属性**，触发 relayout，且与 `.topbar-dropdown` 的 `margin-left: -24px → 0` 过渡时序冲突：

1. `padding-left` 开始收缩（0.35s）→ 所有子元素向左移
2. `margin-left` 从 -24px 过渡到 0（0.35s）→ 下拉按钮向右移 24px
3. 两个反向运动叠加，导致品牌标识在 flex 流中位置抖动
4. `closeEditor` 时 `editor-fullscreen` 在 setTimeout 200ms 后才移除，品牌标识从 4px 瞬间回到 24px

### 新方案

用 `transform: translateX()` 替代 `padding-left`：

- `#topbar` 的 `padding-left` 保持 24px **不变**
- `.topbar-left` 整体 `translateX(-20px)`，GPU 合成，不触发 relayout
- `.topbar-dropdown` 的 `margin-left` 保持 -24px **不变**，只 fade/shrink
- 品牌标识作为 `.topbar-left` 的子元素，随父级一起平滑移动，无抖动

## 变更清单

### 文件 1：`frontend/src/css/components/editor.css`

**删除** `#topbar.editor-fullscreen` 的 `padding-left: 4px`：
```css
#topbar.editor-fullscreen {
  /* 删除 padding-left: 4px; */
}
```

**简化** `#topbar.editor-fullscreen .topbar-dropdown` — 删除 `margin: 0` 和自定义 transition：
```css
#topbar.editor-fullscreen .topbar-dropdown {
  opacity: 0;
  transform: scale(0.8);
  width: 0;
  overflow: hidden;
  pointer-events: none;
  /* 删除 margin: 0; */
  /* 删除 transition 覆盖（继承 .topbar-dropdown 基础 transition） */
}
```

**新增** `#topbar.editor-fullscreen .topbar-left` 规则：
```css
#topbar.editor-fullscreen .topbar-left {
    transform: translateX(-20px);
}
```

### 文件 2：`frontend/src/css/components/topbar.css`

**补充** `.topbar-left` 的 `transform` 过渡（与 `#topbar` 原 `padding-left` 过渡节奏一致）：
```css
.topbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
```

**恢复** `.topbar-dropdown` 的 transition（移除冗余的 `margin`）：
```css
.topbar-dropdown {
  position: relative;
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1),
              transform 0.2s cubic-bezier(0.4, 0, 0.2, 1),
              width 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  /* 恢复：删除 margin 0.35s */
}
```

## 验证步骤

1. 打开编辑器 → 品牌标识平滑左移，无卡顿
2. 关闭编辑器 → 品牌标识平滑右移还原，无卡顿
3. 新建笔记 → 同样平滑
4. `wails build` 成功