# 行号栏宽度缩减与内容穿过修复方案

## 问题概述

1. **行号栏太宽**：CM6 编辑区左侧的行号栏（gutter）整体宽度偏大，占用过多编辑区域。
2. **水平滚动时内容穿过行号栏**：拖动水平滚动条向右时，编辑内容向左移动，穿过行号栏区域，在折叠区域的空白处显现出来。

---

## 当前状态分析

### 问题 1：宽度构成

行号栏由两个 gutter 水平并列组成。相关样式在 [cm6-syntax-highlight.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/cm6-syntax-highlight.js#L95-L110) 中定义：

| 区域 | 选择器 | 当前 padding | 内容宽度 | 总计 |
|------|--------|-------------|---------|------|
| **折叠 gutter**（左侧） | `.cm-foldGutter .cm-gutterElement` | `padding: 0 2px 0 8px`（左 8px + 右 2px） | 折叠图标 ~10px | **~20px** |
| **行号 gutter**（右侧） | `.cm-lineNumbers .cm-gutterElement` | `padding: 0 4px 0 4px`（左 4px + 右 4px） | 数字 8~24px（1~3位） | **~16~32px** |
| **总计** | | | | **~36~52px** |

折叠 gutter 的 `padding-left: 8px` 是主要冗余来源（图标左侧留白过大）。

### 问题 2：内容穿过根因

在 [editor.css#L258-L262](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css#L258-L262) 中：

```css
.cm-gutters {
  background-color: var(--card-bg) !important;
  z-index: 10;
  position: relative;    /* ← 问题行 */
}
```

- CM6 内部默认给 `.cm-gutters` 应用了 `position: sticky; left: 0`，使行号栏固定在左边缘，覆盖下方的滚动内容
- 我们的 CSS 中 `position: relative` 覆盖了 CM6 的 `position: sticky`
- 后果是：行号栏失去粘性，水平滚动时跟随内容移动，其原本位置露出下方编辑内容，造成"内容穿过"的视觉效果

---

## 修改方案

涉及两个文件，共 3 处修改。

### 修改 1：缩减折叠 gutter 左侧 padding — `cm6-syntax-highlight.js` 第 106 行

```js
// 修改前
'.cm-foldGutter .cm-gutterElement': {
    padding: '0 2px 0 8px',
},

// 修改后
'.cm-foldGutter .cm-gutterElement': {
    padding: '0 2px 0 4px',
},
```

**作用**：折叠图标左侧留白从 8px 缩减至 4px，节省约 4px。

### 修改 2：缩减行号元素左右 padding — `cm6-syntax-highlight.js` 第 100 行

```js
// 修改前
'.cm-lineNumbers .cm-gutterElement': {
    padding: '0 4px 0 4px',
},

// 修改后
'.cm-lineNumbers .cm-gutterElement': {
    padding: '0 2px 0 4px',
},
```

**作用**：行号右侧 padding 从 4px 减至 2px，节省约 2px。

> 左侧保留 4px 不动，不与折叠 gutter 争空间；右侧缩到 2px，数字后仅留少许间隙即可。

### 修改前后宽度对比

| 区域 | 修改前 | 修改后 |
|------|--------|--------|
| 折叠 gutter | 8px + ~10px + 2px = ~20px | 4px + ~10px + 2px = ~16px |
| 行号 gutter（3位数字） | 4px + ~24px + 4px = ~32px | 4px + ~24px + 2px = ~30px |
| 总计 | **~52px** | **~46px** |
| 节省 | | **~6px** |

### 修改 3：移除 `position: relative` 修复内容穿过 — `editor.css` 第 261 行

```css
/* 修改前 */
.cm-gutters {
  background-color: var(--card-bg) !important;
  z-index: 10;
  position: relative;
}

/* 修改后 */
.cm-gutters {
  background-color: var(--card-bg) !important;
  z-index: 10;
}
```

**作用**：移除 `position: relative` 后，CM6 内部默认的 `position: sticky; left: 0` 重新生效，行号栏恢复粘性定位，水平滚动时始终固定在左边缘并覆盖下方编辑内容。

> `position: sticky` 与 `z-index: 10` 配合使用，确保行号栏在叠加层之上。

---

## 不受影响的区域

| 元素 | 说明 |
|------|------|
| `.editor-textarea`、`.editor-body` | 未改动 |
| `.md-rendered`（预览区） | 未涉及 |
| `.editor-footer`、`.editor-find-bar` | 未涉及 |
| 其他页面（设置页、AI 聊天等） | 未涉及 |
| `.cm-gutters` 的背景色、边框 | 未改动，仅移除了 `position` |

---

## 验证步骤

1. 运行 `vite build` 确认构建无错误
2. 打开编辑器，查看行号栏宽度是否缩小
3. 输入长行文本触发水平滚动，拖动滚动条向右 → 确认编辑内容不再穿过行号栏
4. 确认行号始终可见，不会随水平滚动隐藏
5. 切换至预览模式，确认预览区不受影响
6. 测试全屏模式下行号栏表现
