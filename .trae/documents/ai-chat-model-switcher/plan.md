# 模型切换组件设计方案（修订版）

## 总结

在 AI 对话页面的输入框上方添加一个紧凑型模型切换栏，用户无需离开对话即可快速切换 AI 模型。切换后自动保存并同步到设置页。该位置为后续添加其他快捷控件预留空间。

***

## 放置位置

**输入框上方左侧**，紧贴 `.ai-chat-input-area` 顶部，留 4-6px 间距。

布局示意：

```
┌─────────────────────────────────────────┐
│             消息列表                      │
│                                         │
├─────────────────────────────────────────┤
│ [模型: gpt-4o ▼] ❮预留更多控件❯          │   ← 新增工具栏行
│ ─────────────────────────────────────── │   ← 分隔线
│ 输入消息...                      [发送]  │   ← 输入区域
└─────────────────────────────────────────┘
```

理由：

* 输入框上方是最自然的操作区域（用户正在输入时切换模型）

* 靠近左侧为后续添加更多快捷控件（如上下文长度、温度等）预留了横向空间

* 不依赖侧栏折叠状态，始终可见

***

## 当前状态

### HTML 结构（[index.html#L653-L661](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L653-L661)）

```html
<div class="ai-chat-content">
    <div id="aiChatEmpty" class="ai-chat-empty">...</div>
    <div id="aiChatMessages" class="ai-chat-messages" style="display:none;"></div>
    <!-- 输入区 -->
    <div id="aiChatInputArea" class="ai-chat-input-area" style="display:none;">
        <textarea id="aiChatInput" class="ai-chat-input" placeholder="输入消息..." rows="1"></textarea>
        <button id="aiChatSendBtn" class="ai-chat-send-btn" disabled>...</button>
    </div>
</div>
```

### 输入区域样式（[ai-chat.css#L668+](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)）

`.ai-chat-input-area` 当前 layout：

* `display: flex` 水平排列 textarea + 发送按钮

* `padding: 0 16px 16px` 底部间距

* `max-width: 900px; margin: 0 auto` 居中约束

***

## 变更方案

### 1. [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html) — 新增工具栏行

在 `.ai-chat-input-area` 内部、textarea 上方新增工具栏行：

```html
<div id="aiChatInputArea" class="ai-chat-input-area" style="display:none;">
    <div class="ai-chat-toolbar">
        <div class="ai-chat-model-select">
            <div class="ai-chat-model-trigger" id="aiChatModelTrigger">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
                <span id="aiChatModelLabel">--</span>
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
            </div>
            <div class="ai-chat-model-dropdown" id="aiChatModelDropdown"></div>
        </div>
    </div>
    <div class="ai-chat-input-row">
        <textarea id="aiChatInput" class="ai-chat-input" placeholder="输入消息..." rows="1"></textarea>
        <button id="aiChatSendBtn" class="ai-chat-send-btn" disabled><!-- SVG --></button>
    </div>
</div>
```

结构说明：

* `.ai-chat-toolbar` — 工具栏行，flex 水平布局，左对齐

* `.ai-chat-model-select` — 模型选择器 widget，包含 trigger + dropdown

* `.ai-chat-input-row` — 原有的输入框 + 发送按钮，保持原有 flex 布局

* trigger 带齿轮图标表示"设置"含义（暗示切换模型）

### 2. [ai-chat.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css) — 新增工具栏样式

```css
/* 工具栏行 */
.ai-chat-toolbar {
    display: flex;
    align-items: center;
    padding: 0 0 6px 0;
    gap: 8px;
}

/* 模型选择器容器 */
.ai-chat-model-select {
    position: relative;
}

/* trigger 按钮 — 紧凑标签样式 */
.ai-chat-model-trigger {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px 8px;
    font-size: 0.78rem;
    color: var(--text-muted);
    border-radius: var(--radius-sm);
    cursor: pointer;
    user-select: none;
    transition: all 0.15s;
}

.ai-chat-model-trigger:hover {
    color: var(--accent);
    background: var(--hover-bg);
}

.ai-chat-model-trigger svg {
    flex-shrink: 0;
}

/* 下拉面板 — 从 trigger 下方弹出 */
.ai-chat-model-dropdown {
    position: absolute;
    bottom: 100%;
    left: 0;
    margin-bottom: 4px;
    min-width: 160px;
    max-height: 240px;
    overflow-y: auto;
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 4px 12px var(--shadow);
    display: none;
    z-index: 20;
}

.ai-chat-model-dropdown.open {
    display: block;
}

.ai-chat-model-dropdown .theme-select-item {
    padding: 6px 12px;
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.1s;
}

.ai-chat-model-dropdown .theme-select-item:hover {
    background: var(--hover-bg);
    color: var(--accent);
}

.ai-chat-model-dropdown .theme-select-item.active {
    color: var(--accent);
    font-weight: 600;
    background: color-mix(in srgb, var(--accent) 8%, transparent);
}

/* 输入框行 — 把 textarea+发送按钮包起来 */
.ai-chat-input-row {
    display: flex;
    gap: 8px;
    align-items: flex-end;
}

/* 原有 .ai-chat-input-area 调整 */
.ai-chat-input-area {
    padding: 8px 16px 16px;
    /* 其他原有样式不变 */
}
```

关键样式决策：

* 下拉从底部弹出（`bottom: 100%`），从下往上展开，不遮挡输入框

* trigger 小而轻：无背景/边框，hover 时才出现背景

* 齿轮图标暗示"设置/配置"语义

* 工具栏行 `padding: 0 0 6px 0` 与输入框行保持 6px 间距

### 3. [ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js) — 交互逻辑

**在** **`initAIChat()`** **中新增：**

```javascript
// ── 模型选择器 ──
const modelTrigger = document.getElementById('aiChatModelTrigger');
const modelDropdown = document.getElementById('aiChatModelDropdown');
const modelLabel = document.getElementById('aiChatModelLabel');
let modelList = []; // 缓存可用模型列表

// 加载模型配置
async function loadModelSelector(cfg) {
    const model = cfg.model || '--';
    modelLabel.textContent = model;
}

// 打开下拉并填充模型列表
async function openModelDropdown() {
    if (modelList.length === 0) {
        try {
            const cfg = await window.go.main.App.GetAIConfig();
            if (cfg.base_url && cfg.api_key) {
                modelList = await window.go.main.App.FetchAIModels(cfg.base_url, cfg.api_key);
            }
        } catch (_) {}
    }
    renderModelDropdown();
    modelDropdown.classList.add('open');
}

function renderModelDropdown() {
    const current = modelLabel.textContent;
    modelDropdown.innerHTML = modelList.map(m =>
        `<div class="theme-select-item${m === current ? ' active' : ''}" data-model="${m}">${m}</div>`
    ).join('');
}

// 切换模型
async function switchModel(model) {
    try {
        const cfg = await window.go.main.App.GetAIConfig();
        cfg.model = model;
        await window.go.main.App.SaveAIConfig(cfg);
        modelLabel.textContent = model;
        // 同步设置页
        const settingsLabel = document.getElementById('aiModelLabel');
        if (settingsLabel) settingsLabel.textContent = model;
        modelDropdown.classList.remove('open');
    } catch (_) {}
}

// 事件绑定
modelTrigger.addEventListener('click', (e) => {
    e.stopPropagation();
    if (modelDropdown.classList.contains('open')) {
        modelDropdown.classList.remove('open');
    } else {
        openModelDropdown();
    }
});

modelDropdown.addEventListener('click', (e) => {
    const item = e.target.closest('.theme-select-item');
    if (item) switchModel(item.dataset.model);
});

// 点击外部关闭
document.addEventListener('click', () => modelDropdown.classList.remove('open'));
```

**入口调用**：在 `onAIChatViewActivated()` 中 `cfg` 加载后调用 `loadModelSelector(cfg)`。

### 4. 同步机制

* 调用 `App.SaveAIConfig({..., model: newModel})` 持久化

* 同时设置 `els.aiModelLabel.textContent = newModel`（这个元素在设置页，属于 `main.js` 的 `els` 对象）

* 因为 `ai-chat.js` 是独立模块，通过 `document.getElementById('aiModelLabel')` 直接设置

***

## 实现步骤

1. 修改 `index.html` — 新增 `.ai-chat-toolbar` + `.ai-chat-input-row` 结构
2. 修改 `ai-chat.css` — 新增工具栏/模型选择器/下拉样式，调整输入区布局
3. 修改 `ai-chat.js` — 实现 `loadModelSelector()` + `openModelDropdown()` + `switchModel()`
4. 构建验证（Vite build）

## 验证

1. Vite build 零错误
2. 输入框上方显示模型名 + 齿轮图标 + 小箭头
3. 点击 → 弹出模型下拉列表
4. 点击模型 → 切换 → 设置页模型同步更新
5. 下拉从底部弹出，不遮挡输入框
6. 侧栏折叠时也显示，因为位置在输入区上方

