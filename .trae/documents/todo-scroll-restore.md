# 待办清单页面内层滚动恢复方案

## 概述

禁用待办清单页面的外层 `#mainContent` 滚动，恢复 `.todo-list-wrap` 内层独立滚动。使顶栏（返回按钮、筛选栏）固定不动，仅待办条目列表区域滚动。

## 当前状态

```
#mainContent    overflow-y: auto   ← 整个页面滚动（顶栏 + 筛选栏也跟着消失）
  └── #viewTodo  height: 100%
       ├── .view-header
       └── .todo-container  flex:1
            ├── .todo-filter-bar
            └── .todo-list-wrap  flex:1   ← 无 overflow，不滚动
```

## 目标状态

```
#mainContent    overflow-y: hidden  （待办视图激活时）
  └── #viewTodo  height: 100%
       ├── .view-header          ← 固定顶部
       └── .todo-container flex:1
            ├── .todo-filter-bar ← 固定
            └── .todo-list-wrap  flex:1, overflow-y: auto   ← 只有这里滚动，且 scrollbar-gutter: stable 预留空间
```

## 实施步骤

### 1. `frontend/src/css/components/todo.css` — 恢复 `.todo-list-wrap` 内层滚动

将当前空的 `.todo-list-wrap` 恢复为带 overflow 的滚动容器：

```css
.todo-list-wrap {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    min-height: 0;
    scrollbar-gutter: stable;
    scrollbar-width: thin;
    scrollbar-color: var(--scrollbar-thumb) transparent;
    padding-right: var(--space-1);
}

.todo-list-wrap::-webkit-scrollbar {
    width: 5px;
}

.todo-list-wrap::-webkit-scrollbar-track {
    background: transparent;
}

.todo-list-wrap::-webkit-scrollbar-thumb {
    background: var(--scrollbar-thumb);
    border-radius: 3px;
}

.todo-list-wrap::-webkit-scrollbar-thumb:hover {
    background: var(--scrollbar-thumb-hover);
}
```

要点：
- `min-height: 0` — 关键：flex 子项有 `overflow: auto` 时，浏览器默认 `min-height: auto` 会导致无法收缩到内容高度以下，`min-height: 0` 显式修正
- `scrollbar-gutter: stable` — 始终预留滚动条空间，避免出现/消失时条目宽度闪变
- `padding-right: var(--space-1)` — 滚动条与条目之间的呼吸间距（5+4=9px）

### 2. `frontend/src/css/components/todo.css` — 禁用外层滚动

参照日历视图的现成模式（`calendar.css L11-14`），在 todo.css 末尾添加：

```css
/* 待办视图下，#mainContent 无需滚动，由 .todo-list-wrap 内部滚动 */
#mainContent:has(#viewTodo.active) {
    scrollbar-gutter: auto;
    overflow-y: hidden;
}
```

### 3. 删除冗余 `.todo-list-inner`

`.todo-list-inner` 之前作为 FAB 的定位容器而引入。FAB 现已移到 body 层级，`min-height: 100%` 在此处没有实际作用，可以移除。

在 `index.html` 中删除 `div.todo-list-inner` 包裹层，让 `#todoList` 直接作为 `.todo-list-wrap` 的子元素。

**修改 HTML**：

```html
<!-- 之前 -->
<div class="todo-list-wrap">
    <div class="todo-list-inner">
        <div id="todoList" class="todo-list">
            <!-- 动态渲染 -->
        </div>
    </div>
</div>

<!-- 之后 -->
<div class="todo-list-wrap">
    <div id="todoList" class="todo-list">
        <!-- 动态渲染 -->
    </div>
</div>
```

同时删除 todo.css 中的 `.todo-list-inner` 样式块。

### 不涉及变更

- `#mainContent` 的全局样式不受影响（仅 `:has(#viewTodo.active)` 时修改）
- 其他视图（笔记网格、设置、AI 对话、日历等）的外层滚动逻辑不变
- `.view.active` 的 flex 布局不变

## 验证

1. 进入待办清单页 → 外层滚动条消失，顶栏/筛选栏固定
2. 添加 10+ 条待办 → 只有条目列表内部滚动，滚动条在条目区域右侧
3. 切换回笔记首页 → 外层滚动条恢复，正常滚动
4. 滚动条出现/消失时 → 条目宽度不闪变（`scrollbar-gutter: stable` 保证）
