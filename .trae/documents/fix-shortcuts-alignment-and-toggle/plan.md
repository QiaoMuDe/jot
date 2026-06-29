# 修复快捷键说明页面对齐 + 增加 Ctrl+7 切换开关

## 总结

修复快捷键说明对话框的两个问题：1) 描述文字与快捷键键位不在同一行（说明比快捷键低一层）；2) 按 Ctrl+7 时如果快捷键页面已打开则关闭它（当前只开不关）。

***

## 当前状态分析

### 问题 1：对齐问题

`renderShortcutsPage()` ([main.js#L4175-L4180](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L4175-L4180)) 生成的结构：

```html
<div class="shortcut-row">
    <div class="shortcut-key"><kbd>Ctrl</kbd> + <kbd>N</kbd></div>
    <div class="shortcut-desc">新建笔记</div>
</div>
```

CSS 中：

* `.shortcut-row` — **没有 flex 规则**，只有 `:last-child` 的 border-bottom 控制（[modals.css#L465](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/modals.css#L465)）。div 是块级元素，key 和 desc 垂直堆叠。

* `.shortcuts-section` — 定义了 `display: flex` 样式（[modals.css#L456](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/modals.css#L456)），但渲染用的是 `shortcut-row` class，这个 section 样式从未被使用。

* `.shortcut-desc` — 有 `margin-left: 24px`，但垂直布局下 margin-left 无效果。

### 问题 2：Ctrl+7 无 toggle

键盘处理 ([main.js#L3937-L3940](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3937-L3940))：

```javascript
case '7':
    e.preventDefault();
    openShortcuts();
    return;
```

总是调用 `openShortcuts()`，不检查当前是否已打开。已定义 `closeShortcuts()` 函数，但从未通过键盘触发过。

***

## 变更方案

### 1. [modals.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/modals.css) — 修复对齐

**改动 A**：`.shortcut-row` 增加 `display: flex` 使其与 desc 水平排列。

```css
.shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 0;
    border-bottom: 1px solid var(--divider);
}

.shortcut-row:last-child {
    border-bottom: none;
}
```

**改动 B**：删除未使用的 `.shortcuts-section` 规则（已被 `.shortcut-row` 取代）。

**改动 C**：`.shortcut-desc` 移除 `margin-left: 24px`（flex 布局下 space-between 已处理间距）。

### 2. [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js) — Ctrl+7 切换

**改动**：`case '7'` 改为判断 shortcuts 当前是否可见：

```javascript
case '7':
    e.preventDefault();
    if (els.shortcutsView.style.display !== 'none') {
        closeShortcuts();
    } else {
        openShortcuts();
    }
    return;
```

***

## 假设与决策

* 其他关闭方式（点击遮罩、Escape、关闭按钮）不受影响

* 通过菜单"快捷键说明"打开的，考虑到菜单位于更多菜单中，点击菜单时快捷键说明对话框是关闭状态。即使对话框已打开（通过键盘），点击菜单项也会再次打开（等于重新渲染），但不影响功能

* `.shortcuts-section` 从未在 JS 中使用，安全删除

## 验证

1. Vite build 零错误
2. 打开快捷键页面，确认每行快捷键和说明在同一行水平对齐
3. 按 Ctrl+7 打开 → 再按 Ctrl+7 关闭 → 再按 Ctrl+7 打开
4. 按 Ctrl+7 打开 → Escape 关闭 → Ctrl+7 打开
5. 菜单路径：更多菜单 → 快捷键说明 → 打开
6. 点击遮罩关闭 → 再次打开

