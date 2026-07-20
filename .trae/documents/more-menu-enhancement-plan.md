# 更多菜单 (More Menu) 视觉增强方案

## 概述

基于"方案A: 暖色精工卡"方向，对 Jot 顶部导航栏的"更多菜单"（`#moreMenu`）进行视觉增强，提升菜单的可见性、精致度和交互反馈。同时融入方案D的快捷键提示。

---

## 当前状态分析

### 现存问题
1. **按钮无 active 状态** — 菜单打开时汉堡按钮外观不变，用户无法感知"菜单已打开"
2. **菜单背景融合** — `--card-bg` + `--shadow-lg` 在白色 topbar 上对比度不足
3. **hover 反馈单一** — 仅有 `--hover-bg` 背景变化，图标无变化，视觉层级不足
4. **分组感薄弱** — div 分隔线过于隐性，区域划分不明显
5. **动画通用** — 使用与其他 modal 共享的 `menuEnter`/`modalExit` keyframes，缺少个性
6. **缺少快捷键提示** — title 属性中有快捷键信息但界面上不显示

### 当前相关文件
| 文件 | 相关行 | 作用 |
|------|--------|------|
| `frontend/index.html` | 70-96 | HTML 结构：按钮 + 菜单项 + 子菜单 |
| `frontend/src/css/components/topbar.css` | 144-244 | 样式：下拉菜单容器、菜单项、子菜单 |
| `frontend/src/css/animations.css` | 183-186 | `@keyframes menuEnter` 动画定义 |
| `frontend/src/main.js` | 5120-5140 | `openMoreMenu`/`closeMoreMenu` 函数 |
| `frontend/src/main.js` | 5177-5187 | 按钮点击事件绑定 |
| `frontend/src/css/variables.css` | 全部 | 设计系统变量（accent、shadow、anim 等） |

---

## 具体变更

### 1. `frontend/src/css/components/topbar.css` — 核心样式重构

#### 1a. 汉堡按钮 active 态
```css
/* 菜单打开时按钮视觉反馈 */
#moreMenuBtn.active {
  background: var(--hover-bg);
  color: var(--accent);
  border-color: transparent;
}
```
- 在 `#moreMenuBtn` 现有样式后追加
- 复用现有变量，无需新变量

#### 1b. 下拉菜单容器增强
将 `.dropdown-menu` 从通用样式升级为带 `#moreMenu` 专属样式：

```css
#moreMenu {
  /* 加粗阴影，增强与 topbar 的视觉分离 */
  box-shadow: var(--shadow-dropdown);
  /* 增大圆角更精致 */
  border-radius: var(--radius-xl);
  /* 顶部 accent 色腰线 */
  border-top: 3px solid var(--accent);
  /* 增大最小宽度容纳快捷键 */
  min-width: 186px;
  padding: 6px;
}
```

注意：保留 `.dropdown-menu` 作为基础样式，通过 `#moreMenu` 的 ID 选择器叠加专属样式，不破坏 `.dropdown-submenu` 等其他使用 `.dropdown-menu` 的元素。

#### 1c. 菜单项 hover 强化
```css
#moreMenu .dropdown-item:hover svg {
  opacity: 1;
  color: var(--accent);
}
#moreMenu .dropdown-item:hover .shortcut-hint {
  opacity: 1;
  color: var(--accent);
}
```

#### 1d. 快捷键提示样式
```css
.shortcut-hint {
  margin-left: auto;
  font-size: 0.7rem;
  color: var(--text-muted);
  opacity: 0.6;
  font-family: inherit;
  letter-spacing: -0.2px;
  transition: opacity 0.12s, color 0.12s;
}
```

#### 1e. 自定义入场动画（替换通用 menuEnter）
```css
#moreMenu.active {
  animation: moreMenuIn 0.2s var(--anim-easing-spring) forwards;
}

@keyframes moreMenuIn {
  from {
    opacity: 0;
    transform: scale(0.92) translateY(-4px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}
```

#### 1f. 分组区域增强
将 `.dropdown-divider` 替换为带文字的分组标签样式：
```css
#moreMenu .dropdown-group-label {
  padding: 4px 12px 2px;
  font-size: 0.65rem;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  opacity: 0.6;
}
```

---

### 2. `frontend/index.html` — HTML 结构调整

#### 2a. 快捷键提示 span
在每个 `.dropdown-item` 中追加 `<span class="shortcut-hint">`：

```html
<div class="dropdown-item" data-action="home">
  <svg>...</svg>笔记首页
  <span class="shortcut-hint">Ctrl+1</span>
</div>
```

快捷键映射表：
| data-action | 快捷键文本 |
|-------------|-----------|
| home | Ctrl+1 |
| sidebar-toggle | Ctrl+2 |
| batch-mode | Ctrl+3 |
| data | Ctrl+4 |
| trash | Ctrl+5 |
| todo | Ctrl+6 |
| settings | Ctrl+7 |
| ai-chat | Ctrl+8 |

注意：子菜单项（快捷键说明、MD 语法、关于）和"帮助参考"触发器本身不加快捷键提示。

#### 2b. 分隔线替换为分组标签
```html
<div class="dropdown-group-label">导航</div>
<!-- 笔记首页、展开侧栏、批量管理 -->
<div class="dropdown-group-label">管理</div>
<!-- 数据管理、回收站、待办清单 -->
<div class="dropdown-group-label">工具</div>
<!-- 设置 -->
<div class="dropdown-group-label">参考</div>
<!-- 帮助参考（含子菜单） -->
<div class="dropdown-group-label">快捷</div>
<!-- AI 助手 -->
```

---

### 3. `frontend/src/main.js` — 按钮 active 态切换

#### 3a. `openMoreMenu` 函数追加 active class
```javascript
function openMoreMenu(menu) {
    menu.style.animation = 'moreMenuIn 0.2s var(--anim-easing-spring) forwards';
    menu.classList.add('active');
    els.moreMenuBtn.classList.add('active');
}
```

#### 3b. `closeMoreMenu` 函数移除 active class
```javascript
function closeMoreMenu(menu) {
    if (!menu.classList.contains('active')) return;
    menu.querySelector('.dropdown-submenu-trigger')?.classList.remove('open');
    menu.style.animation = 'modalExit 0.1s ease-in forwards';
    els.moreMenuBtn.classList.remove('active');
    const onEnd = () => {
        menu.classList.remove('active');
        menu.style.animation = '';
        menu.removeEventListener('animationend', onEnd);
    };
    menu.addEventListener('animationend', onEnd);
}
```

#### 3c. 点击其他区域关闭菜单时也移除 active
```javascript
document.addEventListener('click', () => {
    closeMoreMenu(els.moreMenu);
    // 注意：closeMoreMenu 内部已处理 els.moreMenuBtn.classList.remove('active')
});
```

注意：需要确保所有调用 `closeMoreMenu(els.moreMenu)` 的地方都能正确移除按钮的 active class。由于修改后的 `closeMoreMenu` 内部已处理，外部调用无需改动。

---

### 4. `frontend/src/css/animations.css` — 动画更新

保留现有的 `@keyframes menuEnter` 和 `@keyframes modalExit` 供其他组件使用。

本次将 `openMoreMenu` 中使用的动画从 `menuEnter`（通用）改为 `moreMenuIn`（自定义），定义在 `topbar.css` 中。

---

## 依赖/影响

| 影响对象 | 影响程度 | 说明 |
|---------|---------|------|
| `.dropdown-menu` 通用样式 | 安全 — 通过 `#moreMenu` ID 选择器隔离 | 右侧 `#themeSelectDropdown` 等其他 dropdown 不受影响 |
| `.dropdown-divider` | 不删除 | 保留原有样式，HTML 中改为使用 `.dropdown-group-label` |
| `@keyframes menuEnter` | 不删除 | 其他组件仍在使用 |
| `openMoreMenu`/`closeMoreMenu` | 函数签名不变 | 外部调用无需修改 |
| 12 套主题兼容性 | 完全兼容 | 全部使用现有 CSS 变量 |

---

## 实施步骤

1. **HTML** (`index.html`)：为菜单项添加 `<span class="shortcut-hint">` 快捷键提示；将 `.dropdown-divider` 替换为 `.dropdown-group-label`
2. **CSS** (`topbar.css`)：追加 `#moreMenuBtn.active` 按钮态、`#moreMenu` 专属样式、快捷键提示样式、分组标签样式、自定义 `moreMenuIn` 动画
3. **JS** (`main.js`)：修改 `openMoreMenu`/`closeMoreMenu`，添加按钮 active class 切换
4. **验证**：构建运行，检查所有主题下显示效果，确认点击/关闭/快捷键功能正常

---

## 验证

1. 构建无报错：`cd frontend && npm run build`
2. 功能验证：
   - 点击汉堡按钮 → 菜单打开，按钮变 accent 色背景
   - 鼠标菜单项 → 图标亮起 accent 色
   - 快捷键提示正确显示
   - 分组标签清晰区分
   - 点击菜单项/菜单外/ESC → 菜单关闭，按钮恢复
   - 子菜单（帮助参考）正常展开
3. 主题切换：在 default、dark、nord、tokyo-night 等主题下检查对比度
