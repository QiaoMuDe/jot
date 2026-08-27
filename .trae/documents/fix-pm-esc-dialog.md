# 修复：密码管理弹层未纳入全局 ESC 处理链，按 ESC 连同页面一同退出

## 摘要

在密码管理页打开"添加/编辑"或"详情"对话框（或条目右键菜单）时按 ESC，弹层关闭的同时页面也退回笔记首页。根因是 password-manager.js 自带一个独立的 `document` keydown ESC 监听，与 main.js 的全局 `handleKeyboardNavigation` 并行触发、互不感知；全局链不认识密码弹层，最终落入"非 grid 视图 → `switchView('grid')`"的兜底退出逻辑。

修复方式：遵循本代码库已有的"全局 ESC 链集中处理"惯例——密码模块暴露统一的 `window.pmHandleEscape()`，从自身监听中移除 ESC 逻辑；main.js 全局链在兜底退出之前调用它，命中即 `return`。

## 现状分析

* [main.js L6237](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js) 注册全局键盘监听 `handleKeyboardNavigation`（定义于 L6426）。其 Escape 分支（L6556-6660）按优先级链式检查各浮层：量化弹窗 → 引用选择器 → 启动器 → 搜索弹窗 → 待办输入面板 → 应用确认框 → MCP 表单/导入 → 预设弹窗 → pwdModal → 关于页 → 灯箱 → 编辑器全屏/搜索条/编辑器 → 快捷键页，任何命中即 `return`；全部未命中时，若 `state.currentView !== 'grid'` 则 `switchView('grid')` + `loadNotes()`（L6654-6658）退回首页。分支开头已统一 `e.preventDefault()`（L6557）。

* [password-manager.js L834-841](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/password-manager.js) 另行注册了一个 document keydown 监听处理 ESC：确认框打开时跳过；否则按"右键菜单 → 编辑对话框 → 详情对话框"的层级顺序只关一个。该监听与全局监听同挂在 `document` 上，两个都会执行，且没有任何一方阻止对方。

* 实际时序：ESC 按下 → pm 监听关闭弹层（正常）→ 全局链所有检查均不命中 → 落入 L6654 兜底 → `switchView('grid')`，页面一同退出。右键菜单打开时按 ESC 同理。

* 弹层元素 `pmEditOverlay` / `pmDetailOverlay` / `pmContextMenu` 均为模块级变量，在 `initPasswordManager()`（main.js L8366 启动时调用）中赋值；`closePmEditDialog` / `closePmDetailDialog` / `hideContextMenu` / `isContextMenuVisible` 均为模块级函数，具备从模块暴露统一处理函数的条件。

* `editFromDetail()`（L553-556）先关详情再开编辑，两对话框不存在叠放，ESC 无需处理多层对话框嵌套，但仍按"菜单 → 编辑 → 详情"顺序只关最上层一个（与旧行为一致）。

## 修改方案

### 1. frontend/src/js/password-manager.js

**(a) 删除 L830-841 中的独立 ESC keydown 监听**（保留 L831-833 右键菜单"点击外部关闭"的 mousedown 监听，仅删 keydown 部分）：

```js
// 删除以下监听：
document.addEventListener('keydown', (e) => {
    if (e.key !== 'Escape') return;
    // 应用级确认框打开时由其自身处理，避免联动关闭底层弹层
    if (document.getElementById('confirmDialog')?.classList.contains('visible')) return;
    if (isContextMenuVisible()) hideContextMenu();
    else if (pmEditOverlay.style.display !== 'none') closePmEditDialog();
    else if (pmDetailOverlay.style.display !== 'none') closePmDetailDialog();
});
```

**(b) 在文件尾部** **`window.refreshPasswordManagerView`** **定义之后新增统一 ESC 出口**：

```js
/**
 * 统一 ESC 出口：由 main.js 全局 Escape 分支调用
 * 按 右键菜单 → 编辑对话框 → 详情对话框 的层级只关闭最上层一个
 * @returns {boolean} 是否关闭了某个弹层（true = 本次 ESC 已被消费）
 */
window.pmHandleEscape = function () {
    // 应用级确认框打开时交给其自身处理，不消费本次 ESC
    if (document.getElementById('confirmDialog')?.classList.contains('visible')) return false;
    if (isContextMenuVisible()) { hideContextMenu(); return true; }
    if (pmEditOverlay && pmEditOverlay.style.display !== 'none') { closePmEditDialog(); return true; }
    if (pmDetailOverlay && pmDetailOverlay.style.display !== 'none') { closePmDetailDialog(); return true; }
    return false;
};
```

### 2. frontend/src/main.js — `handleKeyboardNavigation` 的 Escape 分支

在应用确认框检查（L6586-6588）之后、MCP 表单检查之前插入：

```js
// 密码管理：右键菜单/编辑/详情弹层打开时只关闭最上层弹层（不继续执行导航逻辑）
if (typeof window.pmHandleEscape === 'function' && window.pmHandleEscape()) {
    return;
}
```

**插入位置理由**：

* 与旧 pm 监听相同的优先级语义——应用级确认框优先于密码弹层；

* 必须早于 L6646 起的兜底"退出子视图"逻辑，否则页面仍会被切走；

* 与链中 MCP 表单、预设弹窗、pwdModal 等十几处既有检查的写法与风格一致。

## 假设与决策

* **决策**：采用"全局链集中处理 + 模块暴露 `pmHandleEscape`"，而非在 pm 自有监听里 `stopImmediatePropagation()`——后者依赖两个监听的注册顺序（谁先注册谁先执行），时序脆弱；且代码库既有惯例就是全局链集中处理各浮层。

* **决策**：`pmHandleEscape` 返回布尔值表示"是否消费了本次 ESC"，让全局链能自然落到后续检查或兜底退出，无弹层时行为不变。

* **决策**：保留 confirmDialog 守卫在 `pmHandleEscape` 内部（虽全局链在更早处已检查），作为防御性冗余，代价极小。

* **假设**：`initPasswordManager` 在应用启动时调用（main.js L8366），早于任何用户 ESC 操作；对 overlay 变量仅做非空守卫，不做额外时序防御。

## 验证步骤

1. `npm run build`（在 frontend 目录）构建通过。
2. 代码级行为推演核对：

   * 密码页打开"详情"或"添加/编辑"对话框，按 ESC → 仅关闭弹层，停留在密码页；

   * 从详情点"编辑"（详情先关、编辑后开）后按 ESC → 关编辑框；再按 ESC → 此时无弹层，退回首页（符合"无弹层时原行为"）；

   * 右键菜单打开时按 ESC → 仅关菜单，页面不退出；

   * 应用确认框（如删除确认）打开时按 ESC → 走确认框自身逻辑，不联动关闭密码弹层；

   * 密码页无任何弹层时按 ESC → 退回笔记首页（原行为保持）；

   * 其他视图（设置、数据管理等）按 ESC 退回首页的行为不受影响（插入的检查对它们透明——`pmHandleEscape` 返回 false）。

