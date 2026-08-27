# 计划：密码管理页面添加「随机密码生成器」对话框

## 概述

在密码管理页面工具栏添加一个「生成密码」按钮，点击打开一个独立的对话框，支持配置密码规则、批量生成多个密码，并可一键复制。需要集成 ESC 关闭拦截。

***

## 当前状态分析

### 工具栏结构

文件：[index.html#L1931-L1945](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1931-L1945)

当前工具栏包含三个元素：搜索框（flex:1）、添加按钮、批量操作按钮。新按钮将插入在批量操作按钮之后。

### 现有对话框模式

文件：[password-manager.css#L467-L534](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/password-manager.css#L467-L534)

使用 `.pm-overlay`（fixed 全屏遮罩）+ `.pm-dialog`（居中卡片）的固定模式，z-index 为 2000。新对话框将复用此模式。

### ESC 处理链

文件：[main.js#L6589-L6591](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L6589-L6591)

全局 ESC 链在第 6589 行调用 `window.pmHandleEscape()`，PM 模块内部自行判断关闭优先级。新对话框的 ESC 拦截应加入此函数的判断链中。

### PM 模块结构

文件：[password-manager.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/password-manager.js)

纯原生 JS 模块，无框架依赖。通过 Wails 绑定与后端通信（但随机密码生成不需要后端参与）。

***

## 修改方案

### 1. HTML — 添加按钮 + 对话框结构

**文件：** [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html)

**改动 A — 工具栏新增按钮（\~行 1945 后）**

在 `pmBatchToggleBtn` 后面添加：

```html
<button class="pm-gen-btn" id="pmGenBtn">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
    </svg>
    生成密码
</button>
```

使用钥匙图标（类似 1Password / Bitwarden 风格），与密码管理语义匹配。

**改动 B — 对话框 HTML（在** **`pmDetailOverlay`** **后面添加）**

```html
<!-- 密码生成器对话框 -->
<div id="pmGenOverlay" class="pm-overlay" style="display:none;">
    <div class="pm-dialog pm-gen-dialog" role="dialog">
        <div class="pm-dialog-header">
            <h3>随机密码生成器</h3>
            <button id="pmGenCloseBtn" class="pm-close-btn">×</button>
        </div>
        <div class="pm-gen-body">
            <!-- 选项区域 -->
            <div class="pm-gen-options">
                <div class="pm-gen-opt-row">
                    <label>密码长度</label>
                    <div class="pm-gen-length-wrap">
                        <input type="range" id="pmGenLengthRange" min="6" max="64" value="16" />
                        <span id="pmGenLengthVal" class="pm-gen-length-val">16</span>
                    </div>
                </div>
                <div class="pm-gen-opt-row">
                    <label>生成数量</label>
                    <div class="pm-gen-count-wrap">
                        <button type="button" class="pm-gen-count-btn" data-delta="-1">−</button>
                        <span id="pmGenCountVal" class="pm-gen-count-val">5</span>
                        <button type="button" class="pm-gen-count-btn" data-delta="1">+</button>
                    </div>
                </div>
                <div class="pm-gen-opt-row pm-gen-chars">
                    <label>字符类型</label>
                    <div class="pm-gen-char-checks">
                        <label class="pm-gen-check"><input type="checkbox" id="pmGenUpper" checked /> 大写 A-Z</label>
                        <label class="pm-gen-check"><input type="checkbox" id="pmGenLower" checked /> 小写 a-z</label>
                        <label class="pm-gen-check"><input type="checkbox" id="pmGenDigits" checked /> 数字 0-9</label>
                        <label class="pm-gen-check"><input type="checkbox" id="pmGenSymbols" checked /> 符号 !@#$</label>
                    </div>
                </div>
                <div class="pm-gen-opt-row">
                    <label class="pm-gen-check"><input type="checkbox" id="pmGenExcludeAmbiguous" /> 排除易混淆字符 (l, I, 1, O, 0)</label>
                </div>
            </div>

            <!-- 生成按钮 -->
            <button class="pm-btn primary pm-gen-action-btn" id="pmGenGenerateBtn">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
                </svg>
                生成密码
            </button>

            <!-- 结果列表 -->
            <div id="pmGenResults" class="pm-gen-results"></div>
        </div>
    </div>
</div>
```

### 2. CSS — 新增按钮和对话框样式

**文件：** [password-manager.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/password-manager.css)

在文件末尾添加：

```css
/* ==================== 生成密码按钮 ==================== */
.pm-gen-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 6px 12px;
    border-radius: var(--radius-md);
    font-size: 0.78rem;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.15s ease;
    flex-shrink: 0;
    border: 1px solid var(--border);
    background: var(--card-bg);
    color: var(--text-secondary);
}
.pm-gen-btn:hover {
    color: var(--text-primary);
    border-color: var(--accent);
}
.pm-gen-btn:active { transform: scale(0.97); }

/* ==================== 密码生成器对话框 ==================== */
.pm-gen-dialog {
    width: min(520px, calc(100vw - 48px));
}

.pm-gen-body {
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    overflow-y: auto;
}

/* 选项行 */
.pm-gen-options {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
}

.pm-gen-opt-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
}
.pm-gen-opt-row > label:first-child {
    font-size: 0.82rem;
    color: var(--text-secondary);
    white-space: nowrap;
    flex-shrink: 0;
}

/* 长度滑块 */
.pm-gen-length-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
}
.pm-gen-length-wrap input[type="range"] {
    width: 160px;
    accent-color: var(--accent);
}
.pm-gen-length-val {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    min-width: 24px;
    text-align: center;
}

/* 数量 +/- 按钮 */
.pm-gen-count-wrap {
    display: flex;
    align-items: center;
    gap: 4px;
}
.pm-gen-count-btn {
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background: var(--card-bg);
    color: var(--text-secondary);
    font-size: 1rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
}
.pm-gen-count-btn:hover {
    border-color: var(--accent);
    color: var(--accent);
}
.pm-gen-count-val {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    min-width: 20px;
    text-align: center;
}

/* 字符类型复选框组 */
.pm-gen-chars {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
}
.pm-gen-char-checks {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-4);
}
.pm-gen-check {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 0.8rem;
    color: var(--text-secondary);
    cursor: pointer;
}
.pm-gen-check input[type="checkbox"] {
    accent-color: var(--accent);
}

/* 生成按钮 */
.pm-gen-action-btn {
    align-self: center;
    padding: 8px 24px;
}

/* 结果列表 */
.pm-gen-results {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 240px;
    overflow-y: auto;
}

.pm-gen-result-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    background: var(--input-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    transition: border-color 0.15s;
}
.pm-gen-result-item:hover {
    border-color: var(--accent);
}

.pm-gen-result-pwd {
    flex: 1;
    font-family: 'Consolas', 'Menlo', 'Courier New', monospace;
    font-size: 0.82rem;
    color: var(--text-primary);
    letter-spacing: 0.5px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    user-select: all;
}

.pm-gen-copy-btn {
    flex-shrink: 0;
    padding: 4px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background: var(--card-bg);
    color: var(--text-muted);
    font-size: 0.75rem;
    font-family: inherit;
    cursor: pointer;
    transition: all 0.15s;
}
.pm-gen-copy-btn:hover {
    color: var(--accent);
    border-color: var(--accent);
}
.pm-gen-copy-btn:active { transform: scale(0.95); }
.pm-gen-copy-btn.copied {
    color: var(--success, #22c55e);
    border-color: var(--success, #22c55e);
}

/* 无结果提示 */
.pm-gen-empty {
    text-align: center;
    color: var(--text-muted);
    font-size: 0.82rem;
    padding: var(--space-4) 0;
}
```

### 3. JavaScript — 生成逻辑和事件绑定

**文件：** [password-manager.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/password-manager.js)

#### 3a. 新增 DOM 引用变量（在现有变量声明区域 \~行 13-23 追加）

```javascript
let pmGenOverlay, pmGenCloseBtn;
let pmGenLengthRange, pmGenLengthVal;
let pmGenCountVal, pmGenCountBtns;
let pmGenUpper, pmGenLower, pmGenDigits, pmGenSymbols, pmGenExcludeAmbiguous;
let pmGenGenerateBtn, pmGenResultsEl;
let pmGenCount = 5; // 当前生成数量
```

#### 3b. 新增密码生成核心函数

```javascript
/** 密码字符池定义 */
const PM_GEN_CHARS = {
    upper:   'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
    lower:   'abcdefghijklmnopqrstuvwxyz',
    digits:  '0123456789',
    symbols: '!@#$%^&*()_+-=[]{}|;:,.<>?',
};

/** 易混淆字符 */
const PM_GEN_AMBIGUOUS = 'lI1O0';

/**
 * 生成随机密码（密码学安全随机）
 * @param {object} opts
 * @param {number} opts.length - 密码长度
 * @param {boolean} opts.upper - 包含大写
 * @param {boolean} opts.lower - 包含小写
 * @param {boolean} opts.digits - 包含数字
 * @param {boolean} opts.symbols - 包含符号
 * @param {boolean} opts.excludeAmbiguous - 排除易混淆字符
 * @returns {string}
 */
function pmGeneratePassword(opts) {
    let pool = '';
    if (opts.upper) pool += PM_GEN_CHARS.upper;
    if (opts.lower) pool += PM_GEN_CHARS.lower;
    if (opts.digits) pool += PM_GEN_CHARS.digits;
    if (opts.symbols) pool += PM_GEN_CHARS.symbols;
    if (pool.length === 0) pool = PM_GEN_CHARS.lower; // 至少包含小写
    if (opts.excludeAmbiguous) {
        pool = pool.split('').filter(c => !PM_GEN_AMBIGUOUS.includes(c)).join('');
    }
    const arr = new Uint32Array(opts.length);
    crypto.getRandomValues(arr);
    return Array.from(arr, v => pool[v % pool.length]).join('');
}

/** 获取当前选项面板的配置 */
function pmGetGenOptions() {
    return {
        length: Number(pmGenLengthRange.value),
        upper: pmGenUpper.checked,
        lower: pmGenLower.checked,
        digits: pmGenDigits.checked,
        symbols: pmGenSymbols.checked,
        excludeAmbiguous: pmGenExcludeAmbiguous.checked,
    };
}

/** 执行生成并渲染结果 */
function pmDoGenerate() {
    const opts = pmGetGenOptions();
    const count = pmGenCount;
    pmGenResultsEl.innerHTML = '';
    for (let i = 0; i < count; i++) {
        const pwd = pmGeneratePassword(opts);
        const item = document.createElement('div');
        item.className = 'pm-gen-result-item';

        const pwdEl = document.createElement('span');
        pwdEl.className = 'pm-gen-result-pwd';
        pwdEl.textContent = pwd;

        const copyBtn = document.createElement('button');
        copyBtn.type = 'button';
        copyBtn.className = 'pm-gen-copy-btn';
        copyBtn.textContent = '复制';
        copyBtn.addEventListener('click', async () => {
            const ok = await pmCopyText(pwd, '密码已复制');
            if (ok) {
                copyBtn.textContent = '已复制';
                copyBtn.classList.add('copied');
                setTimeout(() => { copyBtn.textContent = '复制'; copyBtn.classList.remove('copied'); }, 1200);
            }
        });

        item.appendChild(pwdEl);
        item.appendChild(copyBtn);
        pmGenResultsEl.appendChild(item);
    }
}
```

#### 3c. 对话框打开/关闭函数

```javascript
/** 打开密码生成器对话框 */
function openPmGenDialog() {
    pmGenCount = 5;
    pmGenCountVal.textContent = '5';
    pmGenLengthRange.value = 16;
    pmGenLengthVal.textContent = '16';
    pmGenUpper.checked = true;
    pmGenLower.checked = true;
    pmGenDigits.checked = true;
    pmGenSymbols.checked = true;
    pmGenExcludeAmbiguous.checked = false;
    pmGenResultsEl.innerHTML = '<div class="pm-gen-empty">点击「生成密码」开始</div>';
    pmGenOverlay.style.display = 'flex';
}

/** 关闭密码生成器对话框 */
function closePmGenDialog() {
    pmGenOverlay.style.display = 'none';
}
```

#### 3d. 在 `initPasswordManager()` 中添加 DOM 引用和事件绑定

在现有的 DOM 引用区域（\~行 816 附近 `pmDetailEditBtn` 后面）添加：

```javascript
pmGenOverlay = document.getElementById('pmGenOverlay');
pmGenCloseBtn = document.getElementById('pmGenCloseBtn');
pmGenLengthRange = document.getElementById('pmGenLengthRange');
pmGenLengthVal = document.getElementById('pmGenLengthVal');
pmGenCountVal = document.getElementById('pmGenCountVal');
pmGenCountBtns = document.querySelectorAll('.pm-gen-count-btn');
pmGenUpper = document.getElementById('pmGenUpper');
pmGenLower = document.getElementById('pmGenLower');
pmGenDigits = document.getElementById('pmGenDigits');
pmGenSymbols = document.getElementById('pmGenSymbols');
pmGenExcludeAmbiguous = document.getElementById('pmGenExcludeAmbiguous');
pmGenGenerateBtn = document.getElementById('pmGenGenerateBtn');
pmGenResultsEl = document.getElementById('pmGenResults');
```

在事件绑定区域（\~行 847 附近 `pmBatchToggleBtn` 事件绑定后）添加：

```javascript
// 生成密码按钮
document.getElementById('pmGenBtn')?.addEventListener('click', openPmGenDialog);
pmGenCloseBtn.addEventListener('click', closePmGenDialog);
pmGenOverlay.addEventListener('mousedown', (e) => {
    if (e.target === pmGenOverlay) closePmGenDialog();
});
pmGenGenerateBtn.addEventListener('click', pmDoGenerate);

// 长度滑块
pmGenLengthRange.addEventListener('input', () => {
    pmGenLengthVal.textContent = pmGenLengthRange.value;
});

// 数量 +/- 按钮
pmGenCountBtns.forEach(btn => {
    btn.addEventListener('click', () => {
        const delta = Number(btn.dataset.delta);
        pmGenCount = Math.max(1, Math.min(20, pmGenCount + delta));
        pmGenCountVal.textContent = String(pmGenCount);
    });
});
```

#### 3e. ESC 拦截集成

修改现有的 `window.pmHandleEscape` 函数（行 967-979），在详情对话框检查**之后**添加密码生成器的检查：

```javascript
window.pmHandleEscape = function () {
    if (document.getElementById('confirmDialog')?.classList.contains('visible')) return false;
    if (isContextMenuVisible()) { hideContextMenu(); return true; }
    if (pmEditOverlay && pmEditOverlay.style.display !== 'none') { closePmEditDialog(); return true; }
    if (pmDetailOverlay && pmDetailOverlay.style.display !== 'none') { closePmDetailDialog(); return true; }
    if (pmGenOverlay && pmGenOverlay.style.display !== 'none') { closePmGenDialog(); return true; }  // ← 新增
    return false;
};
```

优先级放在详情对话框之后是合理的：如果用户同时打开了详情和生成器（不太可能但理论上），ESC 应先关闭最上层的。

***

## 设计决策

| 决策项      | 选择                              | 理由                           |
| -------- | ------------------------------- | ---------------------------- |
| 密码生成位置   | 纯前端 `crypto.getRandomValues`    | 密码不需要持久化存储，纯工具功能，无需后端参与      |
| 对话框模式    | 复用 `.pm-overlay` + `.pm-dialog` | 与现有编辑/详情对话框风格一致              |
| 默认密码长度   | 16                              | 兼顾安全性和可读性                    |
| 默认生成数量   | 5                               | 提供多种选择，一次性生成不多不少             |
| 默认字符类型   | 全选（大写+小写+数字+符号）                 | 最安全的默认值，用户可按需减少              |
| 生成数量范围   | 1-20                            | 1 个太少不过瘾，20 个是合理上限           |
| 按钮在工具栏位置 | 搜索框右侧最末                         | 添加和批量是数据操作，生成密码是独立工具，放最后     |
| 结果复制     | 复用现有 `pmCopyText` 工具函数          | 保持一致的复制体验和提示风格               |
| 选项持久化    | 不持久化                            | 每次打开重置为安全默认值，避免用户上次的配置影响新场景  |
| 对话框宽度    | min(520px, 100vw-48px)          | 比编辑对话框(480px)稍宽，容纳复选框行和长密码展示 |

***

## 需要修改的文件清单

| 文件                                                                                                       | 改动类型    | 说明                            |
| -------------------------------------------------------------------------------------------------------- | ------- | ----------------------------- |
| [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html)                                        | 追加 HTML | 工具栏按钮 + 对话框结构                 |
| [password-manager.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/password-manager.css) | 追加 CSS  | 按钮样式 + 对话框 + 结果列表样式           |
| [password-manager.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/password-manager.js)               | 追加逻辑    | DOM 引用 + 生成函数 + 事件绑定 + ESC 拦截 |

***

## 验证步骤

1. **基本功能**：打开密码管理页 → 点击「生成密码」按钮 → 对话框打开 → 点击「生成密码」→ 5 个随机密码出现
2. **选项调节**：拖动长度滑块 → 数字实时更新 → 再次生成确认长度变化
3. **数量调节**：点 +/- 按钮 → 数量在 1-20 范围内变化 → 生成对应数量密码
4. **字符类型**：取消勾选「符号」→ 生成的密码不应包含符号字符
5. **排除易混淆**：勾选后 → 生成的密码不应包含 l, I, 1, O, 0
6. **复制功能**：点击「复制」按钮 → 按钮变为「已复制」 → 粘贴验证内容正确
7. **ESC 关闭**：打开对话框后按 ESC → 对话框关闭
8. **遮罩关闭**：点击对话框外部遮罩区域 → 对话框关闭
9. **层级 ESC**：同时打开编辑对话框 + 生成器 → ESC 应只关闭最上层的生成器（因为生成器优先级低于编辑对话框）
10. **空字符类型**：取消所有字符类型勾选 → 生成应 fallback 到小写字母（防止空池报错）

