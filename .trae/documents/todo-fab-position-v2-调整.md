# FAB 按钮位置调整方案 v2

## 当前状态

当前 FAB 按钮位于 `.todo-list-inner`（可滚动的列表容器）内，使用 `position: absolute; bottom: 16px; right: 16px;` 定位。这意味着：

```
.view (#viewTodo, height:100%, padding:24px 32px)
  └── .todo-container (max-width:720px, margin:0 auto, flex:1)
       ├── .todo-filter-bar
       └── .todo-list-wrap (flex:1, overflow-y:auto, 可滚动)
            └── .todo-list-inner (position:relative, min-height:100%)
                 ├── #todoList (待办条目)
                 ├── .todo-fab ◄———— 在此定位
                 └── .todo-fab-panel
```

**问题**：FAB 在可滚动列表区域内，滚动时可能被遮挡。

---

## 用户需求解读

> "放在列表的右侧的右下角的位置，和窗口框体的中间"

- **列表的右侧** → 水平方向在 `.todo-container`（列表区域）的右侧边缘
- **右下角** → 垂直方向在底部
- **和窗口框体的中间** → 垂直居中于窗口（或水平居中于容器右侧与窗口边框之间）

---

## 方案：FAB 移出滚动区域，定位到 `.todo-container` 的右下角

### 目标位置

```
.view (#viewTodo)
  └── .todo-container (max-width:720px, margin:0 auto, position:relative, flex:1)
       ├── .todo-filter-bar
       ├── .todo-list-wrap (flex:1, overflow-y:auto)
       │    └── .todo-list-inner
       │         └── #todoList
       ├── .todo-fab ◄———— 移到这层，position:absolute
       └── .todo-fab-panel ◄———— 移到这层
```

### 具体修改

#### 1. HTML 改动（index.html）

将 FAB 和面板从 `.todo-list-inner` 中移出，放在 `.todo-container` 末尾：

```html
<div class="todo-container">
    <div class="todo-filter-bar">...</div>
    <div class="todo-list-wrap">
        <div class="todo-list-inner">
            <div id="todoList" class="todo-list"></div>
        </div>
    </div>
    <!-- FAB + 输入面板移到 .todo-container 层 -->
    <button class="todo-fab" id="todoFab" title="添加待办">...</button>
    <div class="todo-fab-panel" id="todoFabPanel">
        <textarea id="todoInput" ...></textarea>
    </div>
</div>
```

#### 2. CSS 改动（todo.css）

- `.todo-container` 添加 `position: relative` 作为定位锚点
- 删除 `.todo-list-inner` 的 `position: relative`（不再需要）
- `.todo-fab` 改相对于 `.todo-container` 定位：`bottom: 20px; right: -22px;`（一半在容器内，一半突出在右侧）
- `.todo-fab-panel` 输入面板：从底部朝上展开，`bottom: 72px; right: -22px;`

**为什么不使用 `fixed` 定位？**
- 使用 `position: absolute` 相对于 `.todo-container`，FAB 会自动跟随视图切换和滚动，保持与列表容器的视觉关联。
- `position: fixed` 会脱离页面流，在视图切换或滚动时可能出现位置错乱。

**为什么 right 用负值？**
- 让 FAB 按钮一半在 `.todo-container` (720px) 内，一半突出到右侧空白区域
- 视觉上位于"列表的右侧"和"窗口框体的中间"

#### 3. JS 改动

- 无直接 JS 改动，只调整 CSS 定位即可
- 原有事件监听（FAB 点击、面板切换、外部点击关闭）依然有效

---

## 视觉示意

```
┌─────────────────────────────────────────────┐
│  ← 返回         待办清单                      │  ← .view-header
├─────────────────────────────────────────────┤
│  [全部] [待办] [已完成]           [清空]       │  ← .todo-filter-bar
├─────────────────────────────────────────────┤
│ ┌───────────────────────────────────────┐   │
│ │ 待办条目 1                            │   │
│ │ 待办条目 2                            │   │  ← .todo-list-wrap
│ │ ...                                  │   │
│ └───────────────────────────────────────┘   │
│                                          ┌──┐│
│                                          │＋││  ← FAB 半突出在右侧
│                                          └──┘│
└─────────────────────────────────────────────┘
     ^-- 容器右边缘         窗口右边缘 --^
```

---

## 涉及的修改文件

| 文件 | 修改内容 |
|------|---------|
| `frontend/index.html` | 将 FAB/面板从 `.todo-list-inner` 移到 `.todo-container` |
| `frontend/src/css/components/todo.css` | 调整 FAB/面板定位方式，`.todo-container` 加 `position: relative` |
