# AI 侧栏折叠/展开按钮重新定位 + 样式重设计

## 概述

将 AI 侧栏切换按钮从当前的"绝对定位细条"改为整合到侧栏 header 区域的可见按钮，并重新设计样式，提高可发现性和易用性。

## 当前状态

**位置**：`.ai-sidebar-toggle` 作为 `.ai-chat-layout` 内绝对定位元素，浮在侧栏与对话区的缝隙中。

```html
<div class="ai-chat-layout">           <!-- position: relative -->
    <div class="ai-session-sidebar">    <!-- 宽 230px -->
        <div class="ai-session-sidebar-header">
            <span>会话</span>
            <button>+</button>
        </div>
        <!-- 搜索、列表、底部 -->
    </div>
    <button id="aiSidebarToggle">       <!-- 绝对定位，top:50%, left动态 -->
        <!-- 14px × 14px 箭头 -->
    </button>
    <div class="ai-chat-content">…</div>
</div>
```

**样式**：14px × 44px 细条，`background: transparent`，默认几乎不可见。

## 修改方案

### 设计思路

- 将切换按钮**视觉上整合到侧栏 header** 区域
- 为了避免折叠后按钮不可点，**DOM 上仍保持为 `.ai-chat-layout` 的兄弟元素**（不在侧栏内部）
- 通过 `position: absolute` 定位到侧栏 header 区域的右端，视觉上像 header 的一部分
- 折叠后按钮移到左侧边缘（当前 notebook sidebar 的类似模式）

### 文件 1：`frontend\index.html`

**改动**：移除旧的 `#aiSidebarToggle`，在侧栏 header 内部添加新按钮。

```html
<!-- 原独立按钮删除 -->
<!-- <button id="aiSidebarToggle" class="ai-sidebar-toggle" title="展开侧栏">...</button> -->

<!-- 在侧栏 header 内部添加 -->
<span class="ai-session-sidebar-title" id="aiSessionTitle">会话</span>
<button id="aiSidebarToggle" class="ai-sidebar-toggle-btn" title="折叠侧栏">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
</button>
<button id="aiSessionNewBtn" class="ai-session-new-btn" title="新建会话">+</button>
```

### 文件 2：`frontend\src\css\components\ai-chat.css`

**改动 A**：删除旧的 `.ai-sidebar-toggle` 全部样式（line 2237-2277）。

**改动 B**：添加新按钮 `.ai-sidebar-toggle-btn` 样式。

```css
/* 侧栏折叠/展开切换按钮 — 整合在 header 中的图标按钮 */
.ai-sidebar-toggle-btn {
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s ease, color 0.15s ease, transform 0.12s ease;
}
.ai-sidebar-toggle-btn:hover {
    background: var(--hover-bg);
    color: var(--accent);
}
.ai-sidebar-toggle-btn:active {
    transform: scale(0.92);
}
```

**改动 C**：调整 `.ai-session-sidebar-header` 的 `justify-content` 以适应三个子元素布局。

当前是 `justify-content: space-between`（2 个子元素）。加上按钮后变成 3 个，需改为：

```css
.ai-session-sidebar-header {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 8px 10px;
    border-bottom: 1px solid var(--border);
}
```

使用 `gap` 控制间距，`margin-left: auto` 将"新建会话"按钮推到右侧。

### 文件 3：`frontend\src\js\ai-chat.js`

**改动**：无需修改 DOM 获取逻辑。`document.getElementById('aiSidebarToggle')` 在新位置（sidebar header 内）仍然有效。toggle 逻辑不变。

唯一需注意：折叠后按钮随侧栏消失。**解决方式**：修改折叠状态，让侧栏折叠后保留一个极窄可视条（仅容纳按钮）。

```css
/* 折叠状态：只保留切换按钮区域，会话内容隐藏 */
.ai-session-sidebar.collapsed {
    width: 44px;
    min-width: 44px;
    border-right: 1px solid var(--border);
    overflow: hidden;
}
.ai-session-sidebar.collapsed .ai-session-sidebar-header {
    padding: 8px;
    justify-content: center;
    border-bottom: none;
}
.ai-session-sidebar.collapsed .ai-session-sidebar-title,
.ai-session-sidebar.collapsed .ai-session-new-btn,
.ai-session-sidebar.collapsed .ai-session-search-wrap,
.ai-session-sidebar.collapsed .ai-session-list,
.ai-session-sidebar.collapsed .ai-session-sidebar-footer {
    display: none;
}
```

折叠后侧栏变成一个 **44px 宽的细条**（8px padding × 2 + 28px 按钮），只显示居中放置的切换按钮。点击后展开回 230px（`transition: width 0.25s ease` 自动过渡）。

### 样式汇总

| 属性 | 旧值（细条） | 新值（图标按钮） |
|------|-------------|----------------|
| 尺寸 | 14px × 44px | 28px × 28px |
| 背景 | `transparent` | `transparent`（hover: `var(--hover-bg)`） |
| 圆角 | `0 4px 4px 0` | `6px` |
| 定位 | `position: absolute; top: 50%` | flex 正常流内 |
| hover 色 | `var(--accent)` | `var(--accent)` |
| 点击反馈 | 无 | `scale(0.92)` |

## 验证

1. 侧栏展开时，切换按钮正确显示在 header 中标题和"+"按钮之间
2. 点击按钮，侧栏折叠为 36px 窄条，按钮居中显示，箭头方向反转
3. 再次点击按钮，侧栏展开恢复完整视图
4. 会话列表重新加载（当前 `loadSessionList()` 逻辑）
5. `Ctrl+J` 快捷键仍正常工作
6. localStorage 状态保存/恢复正常
