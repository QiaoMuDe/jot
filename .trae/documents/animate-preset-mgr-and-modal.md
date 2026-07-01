# 预设配置管理按钮和弹窗动画计划

## 摘要
为设置页的两个交互区域添加过渡动画：① "管理"按钮展开/收起预设管理列表的动画；② 预设弹窗（新增/编辑）打开/关闭的动画。

## 现状分析

### 1. 预设管理列表（「管理」按钮切换）
- **位置**: `frontend/src/main.js` 的 `renderPresetMgrList()` / `closePresetMgrList()`
- **当前行为**: 点击「管理」→ 直接 `insertAdjacentHTML` 插入面板；点击「关闭」→ `removeChild` 直接移除。无任何过渡。
- **DOM**: 动态创建的 `.preset-mgr-list` div，高度由异步加载的预设列表决定

### 2. 预设弹窗（新增/编辑）
- **位置**: `frontend/index.html` 第1625行 `.preset-modal-overlay`；`main.js` 的 `openAddProfileModal()` / `closePresetModal()`
- **当前行为**: `display: none` ↔ `display: flex` 直接切换。遮罩和内容同时出现/消失，无动画。
- **DOM**: 固定存在的 overlay（`display: none`），内含 `.preset-modal` 内容

## 改动方案

### 改动 1：预设管理列表展开/收起动画

**涉及文件**: [settings-panel.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css)、[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)

**CSS 改动** — 替换 `.preset-mgr-list` 当前样式：
```css
.preset-mgr-list {
    margin-bottom: 16px;
    overflow: hidden;
    animation: mgrSlideDown 250ms ease-out both;
}

.preset-mgr-list.closing {
    animation: mgrSlideUp 200ms ease-in both;
}

@keyframes mgrSlideDown {
    from {
        opacity: 0;
        transform: translateY(-8px);
        max-height: 0;
    }
    to {
        opacity: 1;
        transform: translateY(0);
        max-height: 500px;
    }
}

@keyframes mgrSlideUp {
    from {
        opacity: 1;
        transform: translateY(0);
        max-height: 500px;
    }
    to {
        opacity: 0;
        transform: translateY(-8px);
        max-height: 0;
    }
}
```

**JS 改动** — `closePresetMgrList()`：

当前逻辑：
```js
function closePresetMgrList() {
    presetMgrExpanded = false;
    if (presetMgrContainer && presetMgrContainer.parentNode) {
        presetMgrContainer.parentNode.removeChild(presetMgrContainer);
        presetMgrContainer = null;
    }
}
```

改为：
```js
function closePresetMgrList() {
    presetMgrExpanded = false;
    if (!presetMgrContainer) return;
    presetMgrContainer.classList.add('closing');
    presetMgrContainer.addEventListener('animationend', () => {
        if (presetMgrContainer && presetMgrContainer.parentNode) {
            presetMgrContainer.parentNode.removeChild(presetMgrContainer);
            presetMgrContainer = null;
        }
    }, { once: true });
}
```

### 改动 2：预设弹窗打开/关闭动画

**涉及文件**: [settings-panel.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css)、[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html)、[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)

**CSS 改动**：

将 `.preset-modal-overlay` 当前样式从 `display: flex` 改为 opacity + visibility 控制：
```css
.preset-modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    /* 动画控制 */
    opacity: 0;
    visibility: hidden;
    transition: opacity 200ms ease-out, visibility 200ms ease-out;
}

.preset-modal-overlay.visible {
    opacity: 1;
    visibility: visible;
}

.preset-modal {
    background: var(--card-bg);
    border-radius: var(--radius-lg);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    width: 420px;
    max-width: 90vw;
    max-height: 80vh;
    overflow-y: auto;
    /* 弹窗内容缩放动画 */
    transform: scale(0.92) translateY(-12px);
    transition: transform 250ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.preset-modal-overlay.visible .preset-modal {
    transform: scale(1) translateY(0);
}
```

**HTML 改动** — 移除 overlay 上硬编码的 `style="display:none"`，改为无 style（默认 CSS 控制 `opacity:0; visibility:hidden`）：

从 `index.html` 第1625行：
```html
<div class="preset-modal-overlay" id="presetModalOverlay" style="display:none;">
```
改为：
```html
<div class="preset-modal-overlay" id="presetModalOverlay">
```

**JS 改动** — 两处：

1. `openAddProfileModal()` / `openEditProfileModal()` 中：
```js
// 从：
document.getElementById('presetModalOverlay').style.display = 'flex';
// 改为：
document.getElementById('presetModalOverlay').classList.add('visible');
```

2. `closePresetModal()` 中：
```js
// 从：
document.getElementById('presetModalOverlay').style.display = 'none';
// 改为：
document.getElementById('presetModalOverlay').classList.remove('visible');
```

## 验证方式
1. 打开设置页 → 点击「管理」按钮 → 预设管理面板应滑入展开（250ms）→ 点击「关闭」滑出收起
2. 点击「+ 新增」→ 弹窗遮罩淡入 + 内容缩放弹入（200ms/250ms）→ 点击遮罩 / 关闭按钮 / 取消 → 弹窗淡出
3. 从管理列表点击「编辑」→ 弹窗同样动画打开
4. 多次快速点击「管理」→ 不应出现重复面板或动画冲突
