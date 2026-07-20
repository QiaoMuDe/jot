# 更多菜单 UI 重构计划

## 问题陈述

当前 `#moreMenu` 展开后与周围背景（顶栏/笔记内容）视觉分离度不足：
- **默认主题**下 `--topbar-bg: #FFFFFF`、`--card-bg: #FFFFFF`，菜单纯白背景与顶栏完全一致
- 阴影 `--shadow-dropdown: 0 4px 16px rgba(45,42,36,0.10)` 太浅，菜单像贴在背景上的平面而不是悬浮
- 内容内部图标 opacity 0.5 偏暗，快捷键提示看不清
- 入场/离场动画不连贯

## 设计目标

**"悬浮玻璃卡"** — 让菜单从页面中"浮"出来，有深度感、通透感、弹跳感。

---

## 改动清单

### 文件 1: `frontend/src/css/components/topbar.css`

#### 1.1 增强阴影 — 双层阴影制造悬浮感

```css
#moreMenu {
    box-shadow:
        0 12px 48px rgba(0,0,0,0.10),   /* 远层环境阴影 → 大偏移制造悬浮 */
        0 2px 8px rgba(0,0,0,0.06);      /* 近层接触阴影 → 小偏移增加真实感 */
}
```

暗色主题下阴影自动继承 CSS 变量，无需额外处理。

#### 1.2 毛玻璃背景（最关键改动）

```css
#moreMenu {
    background: rgba(255,255,255,0.82);
    backdrop-filter: blur(24px) saturate(1.2);
    -webkit-backdrop-filter: blur(24px) saturate(1.2);
}
```

效果：
- 半透明背景让菜单"看到"下方的笔记内容，产生层次
- 与纯白顶栏自然区分
- 暗色主题下 `rgba(26,26,26,0.85)` + blur 同样生效

暗色主题覆盖需在 `[data-theme="dark"]` 等块中追加：

```css
[data-theme="dark"] #moreMenu,
[data-theme="tokyo-night"] #moreMenu,
[data-theme="dracula"] #moreMenu,
[data-theme="one-dark-pro"] #moreMenu {
    background: rgba(26,26,26,0.88);
}
```

#### 1.3 重写入场动画 — 三段式 overshoot

替换现有的 `moreMenuIn` keyframes：

```css
@keyframes moreMenuIn {
    0%   { opacity: 0; transform: scale(0.88) translateY(12px); }
    65%  { opacity: 1; transform: scale(1.015) translateY(-2px); }
    85%  { transform: scale(0.995) translateY(0); }
    100% { opacity: 1; transform: scale(1) translateY(0); }
}
```

时长从 0.2s → 0.3s，保持 `var(--anim-easing-spring)`。

#### 1.4 新增离场动画 — 与入场对称

```css
@keyframes moreMenuOut {
    0%   { opacity: 1; transform: scale(1) translateY(0); }
    100% { opacity: 0; transform: scale(0.9) translateY(6px); }
}
```

时长 0.15s，ease-in。

#### 1.5 条目 stagger 入场动画

新增 keyframes：

```css
@keyframes menuItemFadeIn {
    from { opacity: 0; transform: translateY(6px) scale(0.97); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
}
```

菜单 `.active` 时，每条 `.dropdown-item` 依次播放（CSS 变量控制 delay），仅首次打开时。

#### 1.6 分组标签增强

```css
#moreMenu .dropdown-group-label {
    position: relative;
    padding: 6px 12px 4px 16px;  /* 左侧留出装饰条空间 */
    font-size: 0.675rem;
    font-weight: 600;
    letter-spacing: 0.8px;
    opacity: 0.65;
}

#moreMenu .dropdown-group-label::before {
    content: '';
    position: absolute;
    left: 6px;
    top: 50%;
    transform: translateY(-50%);
    width: 2px;
    height: 10px;
    border-radius: 2px;
    background: var(--accent);
    opacity: 0.5;
}
```

#### 1.7 快捷键提示 — KBD 风格

```css
#moreMenu .shortcut-hint {
    background: var(--hover-bg);
    padding: 1px 6px;
    border-radius: 4px;
    font-size: 0.65rem;
    opacity: 0.7;
    letter-spacing: -0.1px;
}

#moreMenu .dropdown-item:hover .shortcut-hint {
    background: var(--accent-lighter);
    color: var(--accent);
    opacity: 1;
}
```

#### 1.8 图标增强

```css
#moreMenu .dropdown-item svg {
    opacity: 0.65;          /* 0.5 → 0.65 */
    transition: opacity 0.12s, transform 0.12s, color 0.12s;
}

#moreMenu .dropdown-item:hover svg {
    opacity: 1;
    color: var(--accent);
    transform: translateX(2px);  /* hover 时图标轻微右移 */
}
```

#### 1.9 hover 微交互

```css
#moreMenu .dropdown-item {
    transition: background 0.1s, transform 0.12s;
    transform: translateY(0);
}

#moreMenu .dropdown-item:hover {
    background: var(--hover-bg);
    transform: translateY(-1px);  /* 轻微上浮 */
}
```

#### 1.10 子菜单 — 添加动效

```css
.dropdown-submenu {
    display: block;          /* 从 display: none → 用 opacity/transform 控制 */
    opacity: 0;
    transform: translateX(-4px);
    pointer-events: none;
    transition: opacity 0.12s ease-out, transform 0.12s ease-out;
}

.dropdown-submenu-trigger.open .dropdown-submenu {
    opacity: 1;
    transform: translateX(0);
    pointer-events: auto;
}
```

---

### 文件 2: `frontend/index.html`

改动最小，仅：
1. 在 `#moreMenu` 容器内 `.dropdown-item` 上添加 `style="--item-index: N"` 属性，用于 stagger 动画 delay

```html
<div class="dropdown-item" data-action="home" style="--item-index: 0" ...>
<div class="dropdown-item" data-action="sidebar-toggle" style="--item-index: 1" ...>
...
```

2. **可选**：在 `#moreMenu` 外部包裹一层用作 backdrop 遮罩（如果需要极淡 overlay）：

```html
<div class="more-menu-overlay" id="moreMenuOverlay"></div>
```

如果加 overlay，需要 CSS 和 JS 配合。**先不加，试效果后决定。**

---

### 文件 3: `frontend/src/main.js`

#### 3.1 `openMoreMenu` — 改用 CSS class 驱动

```javascript
function openMoreMenu(menu) {
    // 清除可能残留的离场标记
    menu.classList.remove('exiting');
    // 使用 CSS class 触发入场
    menu.classList.add('active');
    // 添加 stagger 入场标记
    menu.classList.add('stagger-enter');
    els.moreMenuBtn.classList.add('active');
}
```

#### 3.2 `closeMoreMenu` — 改用 CSS class 驱动

```javascript
function closeMoreMenu(menu) {
    if (!menu.classList.contains('active')) return;
    menu.querySelector('.dropdown-submenu-trigger')?.classList.remove('open');
    menu.classList.remove('stagger-enter');
    // 清除所有条目的 stagger 延迟
    menu.querySelectorAll('.dropdown-item').forEach(item => {
        item.style.animation = '';
    });
    menu.classList.add('exiting');
    els.moreMenuBtn.classList.remove('active');
    const onEnd = () => {
        menu.classList.remove('active', 'exiting');
        menu.removeEventListener('animationend', onEnd);
    };
    menu.addEventListener('animationend', onEnd);
}
```

#### 3.3 移除内联 `style.animation`

- `openMoreMenu` 中不再设置 `menu.style.animation`
- `closeMoreMenu` 中不再设置 `menu.style.animation`
- 两函数统一由 CSS class 控制动画

#### 3.4 stagger 入场逻辑

在 `.stagger-enter` 类被添加到 `#moreMenu` 后，用 JS 给每个 `.dropdown-item` 设置 `animation-delay`（因为 CSS 无法在不同元素上根据索引自动计算 delay）：

```javascript
function applyStaggerAnimation(menu) {
    const items = menu.querySelectorAll('.dropdown-item');
    items.forEach((item, i) => {
        // 移除之前的动画
        item.style.animation = '';
        // 强制 reflow
        void item.offsetWidth;
        // 设置 stagger delay
        item.style.animation = `menuItemFadeIn 0.25s var(--anim-easing-spring) ${i * 0.035}s forwards`;
    });
}
```

在 `openMoreMenu` 结尾调用 `requestAnimationFrame(() => applyStaggerAnimation(menu))`。

---

### 文件 4: `frontend/src/css/animations.css`

添加新 keyframes：

```css
/* More menu item stagger entrance */
@keyframes menuItemFadeIn {
    from { opacity: 0; transform: translateY(6px) scale(0.97); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
}
```

---

## 实施顺序

| 步骤 | 文件 | 改动 | 优先级 | 预期效果 |
|------|------|------|--------|----------|
| 1 | topbar.css | 阴影增强 + 毛玻璃背景 | P0 | 菜单从背景中浮出 |
| 2 | topbar.css | 重写入场/离场动画 | P0 | 弹跳感，视觉愉悦 |
| 3 | main.js | open/closeMoreMenu 重构 | P0 | 配合新动画体系 |
| 4 | topbar.css | 图标/快捷键/分组标签增强 | P1 | 内部可读性提升 |
| 5 | topbar.css + animations.css | 条目 stagger + hover 微交互 | P1 | 精致感 |
| 6 | topbar.css | 子菜单过渡动画 | P2 | 子菜单不突兀 |
| 7 | index.html | 添加 `--item-index` 属性 | P1 | stagger 的前提 |

---

## 不做的事

- ❌ 不改 HTML 结构（不增减菜单项、不重新分组）
- ❌ 不改按键绑定、data-action 逻辑
- ❌ 不改 `updateSidebarMenuItem` 函数
- ❌ 不新增字体/图标库
- ❌ 不加 overlay 遮罩（先看阴影+毛玻璃能否解决问题）

## 验证方式

1. 在默认主题下打开/关闭菜单，观察：
   - 菜单是否明显从背景中"浮"出来
   - 入场动画是否有 overshoot 弹跳
   - 离场动画是否自然
2. 在所有12个主题下检查毛玻璃效果和阴影表现
3. 检查子菜单 hover 交互是否正常
4. 检查所有 data-action 菜单项点击后是否正确关闭菜单并执行操作
5. 检查侧栏折叠/展开后 `updateSidebarMenuItem` 是否正确更新
