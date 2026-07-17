# 主题下拉菜单自动化生成方案

## 概述

将系统主题和代码高亮主题两个下拉菜单的菜单项，从硬编码 HTML 改为 JS 动态生成。今后新增主题只需修改 JS 数据，菜单自动更新。

***

## 当前状态

两处下拉菜单有**三个**数据定义位置，需要同时修改：

| 菜单   | 数据源 (JS)                                                   | HTML 硬编码             |
| ---- | ---------------------------------------------------------- | -------------------- |
| 系统主题 | `themeLabels` (main.js L1367)                              | index.html L320-L333 |
| 代码高亮 | `codeHighlightThemeLabels` (cm6-syntax-highlight.js L1214) | index.html L404-L414 |

初始化函数 `initThemeSettings()` 和 `initCodeHighlightThemeSettings()` 都用 `querySelectorAll('.theme-select-item')` 绑定事件，但不负责生成 DOM。

项目中已有动态生成的先例（`addModelDropdownItem()` 生成 AI 模型列表），模式可直接复用。

***

## 改动方案

### 文件 1: `frontend/index.html`

**做什么**：删除两处硬编码的 `.theme-select-item` 列表，仅保留空的 `themeDropdown` 容器。

**系统主题菜单** (L319-L334)：

* `<div class="theme-select-dropdown" id="themeDropdown">` — 容器保留，删除内部所有 `<div class="theme-select-item">`，留空

**代码高亮主题菜单** (L400-L413)：

* `<div class="theme-select-dropdown" id="codeHighlightThemeDropdown">` — 同上，容器保留，留空

**为什么**：把菜单项渲染职责从 HTML 转移到 JS，新增主题时不再需要碰 HTML。

***

### 文件 2: `frontend/src/main.js`

#### 2a. 新增 `buildThemeDropdown()` 函数

**做什么**：读取 `themeLabels` 对象，遍历生成 `.theme-select-item` 元素插入到 `themeDropdown`。

**逻辑**：

```js
function buildThemeDropdown() {
    const dropdown = els.themeDropdown;
    if (!dropdown) return;
    // 清空（保留容器）
    dropdown.innerHTML = '';
    const currentTheme = getCurrentTheme();
    for (const [key, label] of Object.entries(themeLabels)) {
        const item = document.createElement('div');
        item.className = 'theme-select-item' + (key === currentTheme ? ' active' : '');
        item.dataset.themeValue = key;
        item.textContent = label;
        item.addEventListener('click', () => {
            applyTheme(key);
            // 将配置持久化到 localStorage/后端
            saveSettings();
        });
        dropdown.appendChild(item);
    }
}
```

**为什么**：

* `Object.entries(themeLabels)` 保证遍历顺序，菜单顺序与 `themeLabels` 定义顺序一致

* 初始化时一次性绑定 click 事件，不需要 `initThemeSettings` 再二次绑定

#### 2b. 新增 `buildCodeHighlightThemeDropdown()` 函数

**做什么**：类似地，读取 `codeHighlightThemeLabels`（已从 cm6-syntax-highlight.js 导入），生成代码高亮主题菜单项。

**逻辑**：

```js
function buildCodeHighlightThemeDropdown() {
    const dropdown = document.getElementById('codeHighlightThemeDropdown');
    if (!dropdown) return;
    dropdown.innerHTML = '';
    const currentTheme = codeHighlightTheme; // 已有全局变量
    for (const [key, label] of Object.entries(codeHighlightThemeLabels)) {
        const item = document.createElement('div');
        item.className = 'theme-select-item' + (key === currentTheme ? ' active' : '');
        item.dataset.themeValue = key;
        item.textContent = label;
        item.addEventListener('click', () => {
            applyCodeHighlightTheme(key);
            saveSettings();
        });
        dropdown.appendChild(item);
    }
}
```

**为什么**：与系统主题对称，保持一致的代码风格。

#### 2c. 简化 `initThemeSettings()` 和 `initCodeHighlightThemeSettings()`

**做什么**：这两个函数目前负责绑定 click 事件到硬编码的 item 上。改为动态生成后，click 事件已在 `buildXxxDropdown()` 中绑定，这两个函数只需处理**触发按钮的 toggle** 和**外部点击关闭**逻辑。

**`initThemeSettings()`** **简化后**：

```js
function initThemeSettings() {
    if (_themeInited) return;
    _themeInited = true;
    const trigger = els.themeTrigger;
    const dropdown = els.themeDropdown;
    if (!trigger || !dropdown) return;

    // 点击触发按钮切换下拉
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
    });

    // 点击外部关闭
    document.addEventListener('click', (e) => {
        if (!trigger.contains(e.target) && !dropdown.contains(e.target)) {
            trigger.classList.remove('open');
            dropdown.classList.remove('open');
        }
    });
}
```

**`initCodeHighlightThemeSettings()`** **同理简化**。

#### 2d. 调整初始化调用顺序

**做什么**：在 `initThemeSettings()` 和 `initCodeHighlightThemeSettings()` 被调用之前，先调用对应的 `buildXxxDropdown()`。

在当前调用点（大约 L3100-3200 区域的 `loadSettings().then(...)` 流程中）：

```js
buildThemeDropdown();
buildCodeHighlightThemeDropdown();
initThemeSettings();
initCodeHighlightThemeSettings();
```

#### 2e. 新增主题后的注册流程调整

**做什么**：`codeHighlightThemePairing` 和 `themeLabels` 仍是手动维护的映射，这个不受影响。新增主题时仍需在 `codeHighlightThemePairing` 和 `themeLabels` 中加条目，但**不再需要改 HTML**。

***

### 文件 3: `frontend/src/js/cm6-syntax-highlight.js`（可选优化）

**做什么**：`codeHighlightThemeLabels` 已在适当位置定义并导出。无需修改。

***

## 不需要做的改动

| 内容                                  | 理由                                                                |
| ----------------------------------- | ----------------------------------------------------------------- |
| CSS 样式                              | `.theme-select-item` 样式已完整，新生成的元素 class 相同                        |
| `applyTheme()` 更新标签逻辑               | 仍通过 `els.themeLabel.textContent = themeLabels[themeName]` 更新，无需改动 |
| `applyCodeHighlightThemeUI()`       | 同理，通过标签元素 textContent 更新                                          |
| `updateCodeHighlightThemePairing()` | 依旧通过 `querySelectorAll('.theme-select-item')` 遍历，动态 DOM 不影响       |
| 字体下拉菜单                              | 已有自己的动态生成逻辑，不在此计划范围                                               |

***

## 异常处理

1. **主题数据为空**：如果 `themeLabels` 或 `codeHighlightThemeLabels` 为空对象，dropdown 为空菜单，触发按钮点击后无展开效果
2. **设置缺省值**：如果 `getCurrentTheme()` 返回 null（`data-theme` 属性未设置），`buildThemeDropdown()` 中 items 都不会有 `active` 类，此时应取 `'default'` 作为后备

***

## 验证步骤

1. 构建项目，打开设置页
2. 系统主题下拉菜单应显示所有主题（顺序与 `themeLabels` 定义一致），当前主题高亮
3. 切换系统主题 → 菜单收起 → 标签更新 → 页面主题变色
4. 代码高亮主题下拉菜单同理验证
5. 新增一个测试主题（在 `themeLabels` + `codeHighlightThemePairing` 中添加条目 + `variables.css` 中添加变量块）→ 不修改 HTML → 重启后菜单自动多一条

