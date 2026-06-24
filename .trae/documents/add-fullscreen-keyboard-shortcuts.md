# 全屏快捷键方案

## 摘要

为编辑器的 CSS 伪全屏和 Wails 窗口的 OS 全屏分别绑定不同的快捷键，并更新快捷键说明面板。

## 当前状态分析

### 编辑器的"全屏"（CSS 伪全屏）

* 函数 `toggleEditorFullscreen()`（[main.js:2862](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2862)）

* 仅切换 `.editor-panel` 的 `fullscreen` class，面板从 `560px/85vh` 变为 `100vw/calc(100vh-56px)`

* 当前只能通过点击工具栏全屏按钮触发（[main.js:3759](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3759)）

* ESC 键已支持退出全屏（[main.js:4084](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L4084)）

### Wails 窗口的"全屏"（OS 全屏）

* **完全未使用**。`WindowFullscreen()` / `WindowUnfullscreen()` / `WindowIsFullscreen()` 已在 `wailsjs/runtime/runtime.js` 中存在（约 96-106 行），但从未被 `main.js` 导入或调用

* Go 后端没有任何窗口全屏相关的绑定方法

### 键盘快捷键处理入口

* `handleKeyboardNavigation(e)` 函数（[main.js:4014](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L4014)），全局 `keydown` 监听器

* 快捷键说明列表在 `renderShortcutsPage()` 函数（[main.js:4372](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L4372)）

### 已占用键位（不能冲突）

Ctrl 组合: S, F, H, N, L, A, D, 1-7, Home, End
纯按键: Escape, PageUp, PageDown

***

## 变更方案

### 文件变更清单

#### 1. `frontend/src/main.js` — 3 处修改

##### 1a. 导入 Wails 窗口全屏 API（顶部 import 区域）

* **位置**: 第 3 行附近，现有 import 语句

* **改动**: 在已有 import 中添加 `WindowFullscreen, WindowUnfullscreen, WindowIsFullscreen`

* **原因**: Wails runtime 已提供这些 API，只需导入即可使用

* **代码**:

```javascript
import { WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit, EventsOn,
         WindowFullscreen, WindowUnfullscreen, WindowIsFullscreen } from '../wailsjs/runtime/runtime.js';
```

##### 1b. 在 `handleKeyboardNavigation()` 中添加两个快捷键处理

* **位置**: 现有快捷键处理链中，例如在 Ctrl+L 处理（约 4069）之后、Escape 处理（约 4077）之前

**快捷键 1:** **`F11`** **— 窗口 OS 全屏**

```javascript
// F11: 切换窗口 OS 全屏（与编辑器全屏独立）
if (e.key === 'F11') {
    e.preventDefault();
    // WindowIsFullscreen() 返回 Promise，用 then 回调
    WindowIsFullscreen().then(isWinFs => {
        if (isWinFs) {
            WindowUnfullscreen();
        } else {
            WindowFullscreen();
        }
    });
    return;
}
```

* **前置条件**: 无（全局可用）

* **行为**: 切换 Wails 窗口到真正的 OS 全屏模式

* **与编辑器全屏的关系**: 独立不冲突，用户可同时启用两者

**快捷键 2:** **`Ctrl + E`** **— 编辑器全屏切换**

```javascript
// Ctrl+E: 切换编辑器全屏模式（仅编辑器打开时有效）
if (e.ctrlKey && (e.key === 'e' || e.key === 'E')) {
    e.preventDefault();
    if (els.viewEditor.classList.contains('active')) {
        toggleEditorFullscreen();
    }
    return;
}
```

* **前置条件**: 编辑器必须打开 (`els.viewEditor.classList.contains('active')`)

* **行为**: 调用现有的 `toggleEditorFullscreen()`，作用与点击全屏按钮完全一致

* **键位选择理由**: Ctrl+E 未被占用；"E" 可联想 Editor/Expand

##### 1c. 更新快捷键说明面板

* **位置**: `renderShortcutsPage()` 中的 `shortcuts` 数组

* **改动**: 添加两条新条目，位置放在 Ctrl+L 之后、PgUp 之前（编辑相关分类）

```javascript
{ key: 'Ctrl + E', desc: '编辑器内切换全屏' },
{ key: 'F11', desc: '切换窗口全屏' },
```

#### 2. 无需修改的文件

* **`app.go`** **/** **`main.go`**: 不需要改，Wails runtime 已提供 WindowFullscreen API

* **`style.css`** **/** **`app.css`**: 不需要改，无样式变更

* **`index.html`**: 不需要改，无结构变更

* **快捷键说明页面**: 会自动通过 `renderShortcutsPage()` 更新

***

## 按键区分原则

| 快捷键        | 功能                   | 作用域     | 互斥性         |
| ---------- | -------------------- | ------- | ----------- |
| `Ctrl + E` | 编辑器面板全屏（CSS 伪全屏）     | 仅编辑器打开时 | 与 F11 独立    |
| `F11`      | 窗口 OS 全屏（Wails 原生全屏） | 全局      | 与 Ctrl+E 独立 |
| `Escape`   | 退出编辑器全屏（已有）          | 编辑器全屏时  | —           |

三者可任意组合、互不干扰：

* 无全屏 → `Ctrl+E` → 编辑器撑满 → `F11` → 窗口全屏 + 编辑器撑满

* 窗口全屏 → `Ctrl+E` → 编辑器撑满

* 窗口全屏 → `Escape` → 退出编辑器全屏（保留窗口全屏）

* `F11` 再次按 → 退出窗口全屏

***

## 验证步骤

1. **编辑器全屏 (`Ctrl+E`)**

   * 打开笔记编辑器 → 按 `Ctrl+E` → 编辑器面板展开为全屏

   * 再按 `Ctrl+E` → 编辑器恢复悬浮卡片态

   * 编辑器关闭时按 `Ctrl+E` → 无反应（前置条件过滤）

   * `Escape` → 退出编辑器全屏

2. **窗口全屏 (`F11`)**

   * 任意状态按 `F11` → Wails 窗口进入 OS 全屏

   * 再按 `F11` → 退出窗口全屏

   * 窗口全屏 + 编辑器全屏同时启用 → 两者独立，视觉正确

3. **快捷键说明面板**

   * 按 `Ctrl+7` 打开快捷键面板 → 看到 `Ctrl+E` 和 `F11` 新条目

