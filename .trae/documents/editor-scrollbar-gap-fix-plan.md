# 编辑器滚动条右侧留白修复方案

## 问题概述

笔记编辑器（纯文本编辑模式和预览模式）中，滚动条距面板右侧边框有一段留白（编辑模式约 20px，预览模式约 36px），视觉上滚动条"悬空"，未紧贴右侧边框。

***

## 当前布局分析

### DOM 结构

```
.editor-panel (overflow: hidden)
  ├── .editor-header
  ├── .editor-body (padding: 0 20px 10px)    ← 左右各 20px 内边距
  │   ├── #editorNoteTitle.editor-input      ← 标题输入框
  │   ├── .editor-section                    ← 标签选择器区域
  │   └── .editor-panes (overflow: hidden)   ← 内容区域
  │       ├── #editorNoteContent.editor-textarea  ← CM6 容器（编辑模式）
  │       │   └── .cm-editor → .cm-scroller (padding-right: 1px) ← 滚动条在此
  │       ├── .toc-sidebar                   ← TOC 侧栏（预览模式，可选）
  │       └── #mdRendered.md-rendered        ← 预览区（预览模式）
  │           (padding: 0.5em 1rem 1rem 1.5rem, overflow-y: auto) ← 滚动条在此
  ├── .editor-find-bar                       ← 查找替换条
  └── .editor-footer (padding: 6px 24px)    ← 底部状态栏
```

### 根因

**[`.editor-body`](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css#L132)** 的 `padding: 0 20px 10px` 设置了 **左右各 20px** 的内边距。

所有可滚动元素（CM6 的 `.cm-scroller` 和预览区的 `.md-rendered`）都是 `.editor-body` 的后代，因此它们的滚动条位置被推入面板内部，距右侧边框距离如下：

| 模式        | 滚动容器                             | 距离面板右边                                       |
| --------- | -------------------------------- | -------------------------------------------- |
| **纯文本编辑** | `.cm-scroller`（CM6 内部自管理滚动）      | \~20px（来自 .editor-body padding-right）        |
| **预览**    | `.md-rendered`（overflow-y: auto） | \~36px（20px + 自身 padding-right: 1rem ≈ 16px） |

***

## 修复方案

### 核心思路

将 `.editor-body` 的右内边距转移到其**非滚动子元素**上，使可滚动容器（`.editor-panes` 及其后代）能延伸到面板右边缘，滚动条自然地贴边显示。

### 具体修改（仅修改一个文件）

**文件：** `frontend/src/css/components/editor.css`

#### 修改 1：`.editor-body` — 移除右内边距

```css
/* 修改前 */
.editor-body {
  padding: 0 20px 10px;
  ...
}

/* 修改后 */
.editor-body {
  padding: 0 0 10px 20px;   /* 移除右侧 20px，保留左侧 */
  ...
}
```

**原因：** 移除右 padding 后，`.editor-panes` 可以扩展到面板右边缘，其内部的滚动容器（CM6 scroller 和 `.md-rendered`）的滚动条即可贴边。

#### 修改 2：`.editor-input` — 添加右内边距

当前第 178-190 行：

```css
/* 修改前 */
.editor-input {
  width: 100%;
  padding: 2px 0;            /* 无左右 padding，依赖父容器 */
  ...
}

/* 修改后 */
.editor-input {
  width: 100%;
  padding: 2px 20px;         /* 左右各 20px，弥补父容器移除的 padding */
  ...
}
```

**原因：** 标题输入框需要左右留白。左留白继承自 `.editor-body`，右留白需自行添加。

#### 修改 3：`.editor-section` — 添加右内边距

当前第 302-303 行：

```css
/* 修改前 */
.editor-section {
  margin: 2px 0 12px;
  /* 无 padding，标签区域右端会顶到面板边缘 */
}

/* 修改后 */
.editor-section {
  margin: 2px 0 12px;
  padding-right: 20px;       /* 标签区域右侧留白 */
}
```

**原因：** 标签选择器（`.tag-selector`）区域需要右侧留白，避免标签紧贴边缘。

#### 修改 4：`.md-rendered` — 减小右内边距（可选优化）

当前第 983 行：

```css
/* 修改前 */
.md-rendered {
  ...
  padding: 0.5em 1rem 1rem 1.5rem;
  ...
}

/* 修改后 */
.md-rendered {
  ...
  padding: 0.5em 0.5rem 1rem 1.5rem;  /* 右 padding 从 1rem 减为 0.5rem */
  ...
}
```

**原因：** 移除 `.editor-body` 的右 padding 后，`.md-rendered` 的右边缘已贴边。其自身的 `padding-right: 1rem` 会保留内容与滚动条之间的间距，视觉尚可。但此间距略大（内容距滚动条约 16px），可选减小到 `0.5rem`（约 8px），让内容更舒展。

**如不做此修改，`.md-rendered`** **当前** **`padding-right: 1rem`** **也可以接受，不会导致留白问题。** 建议实测后再决定。

### 不受影响的元素

| 元素                               | 原因                                             |
| -------------------------------- | ---------------------------------------------- |
| `.editor-header`                 | 有自己的 `padding: 4px 12px 0`，不依赖 `.editor-body`  |
| `.editor-find-bar`               | 是 `.editor-body` 的兄弟元素，不在其内部                   |
| `.editor-footer`                 | 是 `.editor-body` 的兄弟元素，有独立 `padding: 6px 24px` |
| `.cm-scroller`                   | `padding-right: 1px` 保持不变，仅提供微量缓冲              |
| `.toc-sidebar`                   | 作为 `.editor-panes` 的子元素，不受父容器 padding 影响       |
| 全屏模式（`.editor-panel.fullscreen`） | 布局结构不变，仅 panel 本身尺寸变化                          |

***

## 验证步骤

1. 保存修改后，在开发环境下查看编辑器页面
2. 分别测试 **纯文本编辑模式** 和 **预览模式**，确认滚动条紧贴右侧边框
3. 测试标题输入框和标签选择器的左右留白是否正常
4. 测试 **全屏模式** 下滚动条位置
5. 测试 **TOC 侧栏展开** 时预览模式滚动条位置
6. 测试 **查找替换条** 展开/收起时滚动条位置不变
7. 切换不同主题，确认滚动条颜色样式不受影响

