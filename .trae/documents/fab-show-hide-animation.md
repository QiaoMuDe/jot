# FAB 按钮显隐过渡动画方案

## 概述

为笔记首页的「新建/批量管理」fab-group 和待办清单页的 todo-fab/todo-fab-panel 添加平滑的显隐过渡动画，替换原有的 `display: none/''` 瞬时切换。

## 当前状态分析

### 控制位置

| 元素 | JS 控制行 (main.js) | 当前控制方式 |
|------|-------------------|-------------|
| `fab-group`（笔记首页 FAB） | L587 `switchView` | `style.display = view==='grid' ? '' : 'none'` |
| `todo-fab`（待办 FAB） | L591 `switchView` | `style.display = view==='todo' ? '' : 'none'` |
| `todo-fab-panel`（待办面板） | L598 `switchView` | `style.display = view==='todo' ? '' : 'none'` |
| `todo-fab` + `todo-fab-panel` | L7477-7478 `init()` | `style.display = 'none'` |

### 现有 CSS 过渡

- `fab-group`: 已有 `transition: bottom 0.25s ease`（滚动偏移）
- `todo-fab`: 已有 `transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease`
- `todo-fab-panel`: 已有 `transition: opacity 0.2s ease, transform 0.25s var(--anim-easing-out)`（配合 `.open` 类）

### 问题

`display: none/''` 不可动画，切换时瞬间消失/出现，缺乏丝滑感。

## 方案设计

### 核心思路

用 CSS class `fab-hidden` + `opacity/transform/visibility/pointer-events` 替代 `style.display`，让浏览器驱动过渡动画。

- **隐藏态**：`opacity: 0; transform: scale(0.8); pointer-events: none; visibility: hidden;`
- **显示态**：移除 `fab-hidden`，恢复 CSS 默认值，`transition` 驱动平滑过渡
- **`visibility`**：解决 `pointer-events` 无法阻止 Tab 键聚焦的问题；transition 结束时再切换，避免突然消失

### 具体改动

#### 1. `frontend/src/css/components/main-content.css` — fab-group 过渡

在 `.fab-group` 声明之后添加：

```css
/* fab-group 显隐过渡 */
.fab-group {
    transition: opacity 0.25s ease, transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1),
                visibility 0.25s ease, bottom 0.25s ease;
}

.fab-group.fab-hidden {
    opacity: 0;
    transform: scale(0.8);
    pointer-events: none;
    visibility: hidden;
}
```

- 使用 `cubic-bezier(0.34, 1.56, 0.64, 1)` 让显现时有弹性感，与项目中 `var(--anim-easing-spring)` 一致
- `visibility` 延迟过渡，确保 opacity 动画完成后才真正隐藏元素

#### 2. `frontend/src/css/components/todo.css` — todo-fab / todo-fab-panel 过渡

**todo-fab**：
```css
.todo-fab {
    transition: opacity 0.25s ease, transform 0.2s ease,
                box-shadow 0.2s ease, background 0.2s ease,
                visibility 0.25s ease;
}

.todo-fab.fab-hidden {
    opacity: 0;
    transform: scale(0.8);
    pointer-events: none;
    visibility: hidden;
}
```

**todo-fab-panel**（在 `.todo-fab-panel.open` 定义之后添加）：
```css
.todo-fab-panel.fab-hidden {
    opacity: 0;
    transform: translateY(16px);
    pointer-events: none;
    visibility: hidden;
}
```

> 注：`.todo-fab-panel.fab-hidden` 放在 `.open` 之后，确保当 `.fab-hidden` 和 `.open` 同时存在时（切离待办页时），`.fab-hidden` 生效覆盖 `.open`。

#### 3. `frontend/src/main.js` — 替换 display 操作为 class 切换

**`switchView` 函数 (L586-599)**：

```js
// 旧代码（L586-599）
els.fabGroup.style.display = view === 'grid' ? '' : 'none';
if (els.todoFab) els.todoFab.style.display = view === 'todo' ? '' : 'none';
if (els.todoFabPanel) {
    if (view !== 'todo') {
        els.todoFabPanel.classList.remove('open');
        els.todoFab?.classList.remove('open');
    }
    els.todoFabPanel.style.display = view === 'todo' ? '' : 'none';
}

// 新代码
els.fabGroup.classList.toggle('fab-hidden', view !== 'grid');
if (els.todoFab) els.todoFab.classList.toggle('fab-hidden', view !== 'todo');
if (els.todoFabPanel) {
    if (view !== 'todo') {
        els.todoFabPanel.classList.remove('open');
        els.todoFab?.classList.remove('open');
    }
    els.todoFabPanel.classList.toggle('fab-hidden', view !== 'todo');
}
```

**`init` 函数 (L7476-7478)**：

```js
// 旧代码
if (els.todoFab) els.todoFab.style.display = 'none';
if (els.todoFabPanel) els.todoFabPanel.style.display = 'none';

// 新代码
if (els.todoFab) els.todoFab.classList.add('fab-hidden');
if (els.todoFabPanel) els.todoFabPanel.classList.add('fab-hidden');
```

### 动画流程示意

```
切换至待办页:
  todo-fab:  ──[opacity 0→1, scale 0.8→1, 0.25s]──→ 显示
  todo-fab-panel: (视觉无变化: 从 fab-hidden 到 base, 两者 opacity/transform 相同)
                  → 100ms 后 .open 添加 → [opacity 0→1, translateY 16→0, 0.25s]

切换离待办页:
  todo-fab-panel: .open 移除 (面板回到 base 隐藏态)
                  → 添加 fab-hidden (覆盖 .open, 防止重新进入时闪烁)
  todo-fab:  ──[opacity 1→0, scale 1→0.8, 0.25s]──→ 隐藏

切换至笔记首页:
  fab-group: ──[opacity 0→1, scale 0.8→1, 0.25s spring]──→ 显示

切换离笔记首页:
  fab-group: ──[opacity 1→0, scale 1→0.8, 0.25s]──→ 隐藏
```

### 不涉及变更的代码

- `.todo-fab-panel` 的 `.open` / base 状态 CSS 不变
- `.fab-group` 的 `scrolled`（滚动偏移）类逻辑不变
- `openTodoInputPanel()` / `closeTodoInputPanel()` 函数不变

### 假设 & 决策

1. `.fab-hidden` 作为全局约定名称（当前项目无重名），不与其他样式冲突
2. `.todo-fab-panel.fab-hidden` 放在 `.open` 之后确保覆盖优先级
3. `visibility` 的过渡时间与 opacity 一致（0.25s），确保它们同时完成
4. `fab-group` 使用 spring easing 增强显现弹性感

## 验证步骤

1. 构建：运行 `wails build`（或 dev 模式）确认无编译错误
2. 笔记首页切换：
   - 从笔记首页切换到设置页 → fab-group 缩小消隐（非瞬消）
   - 从设置页切回笔记首页 → fab-group 弹性显现
3. 待办页切换：
   - 从待办页切换到笔记首页 → todo-fab 缩小消隐，面板先关闭再消隐
   - 从笔记首页切到待办页 → todo-fab 显现 → 面板自动展开
4. 边缘情况：
   - 重复快速切换视图 → 动画不冲突无残留
   - 待办面板打开时切换离待办页 → 面板先关闭，再整体隐藏
   - 初始加载（笔记首页）→ fab-group 正常显示，todo-fab/面板隐藏
