# 重构：顶部按钮移入更多菜单

## 摘要

将顶部 topbar 中的 `+`（新建笔记）和 `✓`（批量选择）按钮移除：

* `+` 改由底部 FAB 按钮承担（已有），去掉顶部的冗余按钮

* `✓` 移入 `☰` 更多菜单中，命名为"批量管理"

* 同步更新键盘快捷键顺序、快捷键说明页面

## 当前状态分析

### 右上角 topbar-actions（当前有 6 个按钮）

```
+  ✓  ☰  ─  □  ✕
↑     ↑    ↑窗口控制
新建  批量  更多菜单
```

### 右下角 FAB 组（已有 2 个按钮）

```
↑  (回到顶部，滚动后显示)
+  (新建笔记)
```

### 当前菜单项与快捷键映射

| 菜单项  | data-action    | 当前数字键 | 功能        |
| ---- | -------------- | ----- | --------- |
| 笔记首页 | home           | 1     | 切换到笔记网格视图 |
| 展开侧栏 | sidebar-toggle | 2     | 切换侧栏展开/收起 |
| —    | —              | —     | 分隔线       |
| 数据管理 | data           | 3     | 切换到数据管理视图 |
| 回收站  | trash          | 4     | 切换到回收站视图  |
| —    | —              | —     | 分隔线       |
| 设置   | settings       | 5     | 切换到设置视图   |
| —    | —              | —     | 分隔线       |
| 帮助   | help           | 6     | 打开快捷键说明   |

### 当前快捷键说明页

* 最后一行：`1 2 3 4 5 6` → `快速切换视图 / 切换侧栏`

## 改动清单

### 1. [`frontend/index.html`](file:///d:%5C峡谷%5CDev%5C本地项目%5Cjot%5Cfrontend%5Cindex.html) — 移除按钮 + 添加菜单项

**删除：`newNoteBtn`** **元素**

```
<button id="newNoteBtn" class="topbar-btn" title="新建笔记">+</button>
```

理由：底部 FAB 已有 `+` 按钮，去掉顶部冗余副本。

**删除：`batchModeBtn`** **元素**

```
<button id="batchModeBtn" class="topbar-btn" title="批量选择">✓</button>
```

**新增：菜单项「批量管理」**
在「展开侧栏」后面、第一个分隔线之前插入：

```html
<div class="dropdown-item" data-action="batch-mode" title="按 3 切换">批量管理</div>
```

**更新后的菜单顺序（6 项→7 项）：**

| 顺序 | 菜单项      | data-action    | 数字键   | 说明     |
| -- | -------- | -------------- | ----- | ------ |
| 1  | 笔记首页     | home           | 1     | 不变     |
| 2  | 展开侧栏     | sidebar-toggle | 2     | 不变     |
| 3  | **批量管理** | **batch-mode** | **3** | **新增** |
| —  | 分隔线      | —              | —     | —      |
| 4  | 数据管理     | data           | 4     | 原 3→4  |
| 5  | 回收站      | trash          | 5     | 原 4→5  |
| —  | 分隔线      | —              | —     | —      |
| 6  | 设置       | settings       | 6     | 原 5→6  |
| —  | 分隔线      | —              | —     | —      |
| 7  | 帮助       | help           | 7     | 原 6→7  |

### 2. [`frontend/src/main.js`](file:///d:%5C峡谷%5CDev%5C本地项目%5Cjot%5Cfrontend%5Csrc%5Cmain.js) — 多处联动修改

#### a) 删除 `newNoteBtn` 元素引用

* 位置：`els` 对象中 `newNoteBtn: $('newNoteBtn'),`

* 理由：HTML 元素已删除

#### b) 删除 `batchModeBtn` 元素引用

* 位置：`els` 对象中 `batchModeBtn: $('batchModeBtn'),`

* 理由：HTML 元素已删除

#### c) 删除 `newNoteBtn` + `batchModeBtn` 视图切换逻辑

* 位置：`switchView()` 函数中 line 428-448

* 删除 `newNoteBtn` 和 `batchModeBtn` 的 display/opacity/pointerEvents 控制

* 保留 FAB 组的显示逻辑（line 449-450）

#### d) 菜单点击处理器新增 `batch-mode` 分支

* 位置：`els.moreMenu.addEventListener('click', ...)` 回调中（line 3209-3227）

* 新增：

```js
} else if (item.dataset.action === 'batch-mode') {
    switchView('grid');  // 确保在网格视图
    toggleBatchMode();
```

* 其他已有分支保持不变

#### e) 删除 `newNoteBtn` 点击事件

* 位置：`els.newNoteBtn.addEventListener('click', () => { openEditor(null); });` (line 3179)

* 理由：按钮已移除，底部 FAB 已有相同功能

#### f) 删除 `batchModeBtn` 点击事件

* 位置：`els.batchModeBtn.addEventListener('click', toggleBatchMode);` (line 3339)

* 理由：按钮已移除，改为菜单触发

#### g) 简化 `toggleBatchMode()` 函数

* 位置：line 2686

* 删除 `els.batchModeBtn.classList.toggle('active', state.batchMode);`

* 理由：`batchModeBtn` DOM 元素已不存在

#### h) 更新键盘快捷键数字键映射

* 位置：`handleKeyboardNavigation` 中的 switch-case（line 3568-3595）

* 新映射：

| 当前                                 | 改为                           |
| ---------------------------------- | ---------------------------- |
| case '1': → grid                   | 不变                           |
| case '2': → toggleSidebar          | 不变                           |
| case '3': → switchView('data')     | **→ toggleBatchMode()**      |
| case '4': → switchView('trash')    | **→ switchView('data')**     |
| case '5': → switchView('settings') | **→ switchView('trash')**    |
| case '6': → openShortcuts()        | **→ switchView('settings')** |
| **新增 case '7'**                    | **→ openShortcuts()**        |

#### i) 更新快捷键说明页面

* 位置：`renderShortcutsPage()` 函数（line 3797-3815）

* 将最后一行 `{ key: '1 2 3 4 5 6', desc: '快速切换视图 / 切换侧栏' }` 修改为：

```js
{ key: '1 2 3 4 5 6 7', desc: '快速切换视图 / 批量管理 / 切换侧栏' }
```

### 3. `style.css` — 无需改动

没有需要修改的 CSS，只是移除了 DOM 元素，样式体系不变。

## 涉及文件

| 文件                     | 改动类型                                  |
| ---------------------- | ------------------------------------- |
| `frontend/index.html`  | 删除 2 个 button 元素，新增 1 个 dropdown-item |
| `frontend/src/main.js` | 约 8 处改动（元素引用、事件绑定、快捷键、函数体、说明页）        |

## 注意事项

* 通过菜单进入批量模式时，需要确保当前在 grid 视图（`switchView('grid')`），因为批量模式仅在网格视图有效

* `toggleBatchMode()` 中的 `els.batchModeBtn.classList.toggle(...)` 行删除后不影响功能，仅用于视觉反馈（按钮高亮），现在由菜单触发不需要按钮高亮

* 快捷键顺序与菜单显示顺序一致（用户要求）

## 验证步骤

1. `wails build` 编译无报错
2. 运行后确认右上角没有 `+` 和 `✓` 按钮
3. 点击 ☰ 菜单，确认「批量管理」出现在「展开侧栏」下方第三个位置
4. 在网格视图按 `3` 键，进入批量模式
5. 在快捷键说明页（按 `7`）确认数字键说明已更新
6. `golangci-lint fmt ./... && golangci-lint run ./...` 零警告

