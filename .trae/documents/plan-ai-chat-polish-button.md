# 计划：AI 聊天输入框内嵌「优化表达」按钮

## 概要

在 AI 聊天输入框（textarea）内部右下角添加一个「优化表达」按钮。点击后将当前输入文本发送给 AI 优化表达，优化结果替换输入框内容，按钮变为「还原」可恢复原文。

## 现状分析

- 输入框结构（[index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L879-L887)）：
  ```html
  <div class="ai-chat-input-row">
      <textarea id="aiChatInput" class="ai-chat-input" ...></textarea>
      <button id="aiChatSendBtn" ...>发送</button>
      <button id="aiChatStopBtn" ...>停止</button>
  </div>
  ```
- textarea 当前 `flex:1`，发送/停止按钮在其右侧，使用 `gap:8px` 分隔
- 后端已有 `CallAI(messages)` 方法（非流式），返回 `(string, error)`
- 已有 `SKILL_PROMPTS.polish` 系统提示词（文本润色），但「优化表达」按钮使用独立的 `OPTIMIZE_EXPRESSION_PROMPT`
- 「更多技能 → 文本润色」保留不动

## 变更方案

### 1. HTML：textarea 外层包裹容器

**文件**：[index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L879-L887)

```html
<!-- 修改前 -->
<textarea id="aiChatInput" class="ai-chat-input" placeholder="输入消息..." rows="1"></textarea>
<button id="aiChatSendBtn" ...>...</button>
<button id="aiChatStopBtn" ...>...</button>

<!-- 修改后 -->
<div class="ai-chat-input-wrap">
    <textarea id="aiChatInput" class="ai-chat-input" placeholder="输入消息..." rows="1"></textarea>
    <button id="aiChatPolishBtn" class="ai-chat-polish-btn" title="优化表达" disabled>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/>
        </svg>
        <span class="ai-chat-polish-text">优化</span>
    </button>
</div>
<button id="aiChatSendBtn" ...>...</button>
<button id="aiChatStopBtn" ...>...</button>
```

- 使用 `ai-chat-input-wrap` 容器包裹 textarea，容器 `position:relative`
- 按钮 `position:absolute` 定位在容器右下角
- 按钮初始状态 `disabled`（无输入内容时不可点击）
- 按钮内包含 SVG 图标 + `<span>` 文字，文字在「优化」和「还原」之间切换

### 2. CSS：内嵌按钮样式

**文件**：[ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css) (~line 1023-1087)

新增样式：

```css
/* 输入框包裹容器 */
.ai-chat-input-wrap {
    position: relative;
    flex: 1;
}

/* 内嵌优化表达按钮 */
.ai-chat-polish-btn {
    position: absolute;
    right: 6px;
    bottom: 6px;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 3px 8px;
    font-size: 0.72rem;
    color: var(--text-muted);
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, color 0.15s, background 0.15s;
    user-select: none;
    white-space: nowrap;
    line-height: 1;
}

/* textarea 获得焦点 或 hover 到包裹容器时 显示按钮 */
.ai-chat-input-wrap:hover .ai-chat-polish-btn,
.ai-chat-input:focus ~ .ai-chat-polish-btn {
    opacity: 1;
}

.ai-chat-polish-btn:not(:disabled):hover {
    color: var(--accent);
    background: color-mix(in srgb, var(--accent) 10%, transparent);
    border-color: var(--accent);
}

.ai-chat-polish-btn:disabled {
    opacity: 0;
    cursor: not-allowed;
}

/* 还原状态 */
.ai-chat-polish-btn.is-revert {
    color: var(--warning, #e6a23c);
    border-color: var(--warning, #e6a23c);
    opacity: 1;
}
```

- `opacity: 0` → `1` 通过 hover/focus 控制，避免按钮常驻遮挡输入框
- disabled 时始终隐藏（`opacity: 0` + `cursor: not-allowed`）
- 还原状态改为警告色（橙色），始终可见

### 3. JS：按钮逻辑 + AI 调用

**文件**：[ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)

#### 3a. 新增变量 + DOM 引用（~line 410-413 附近）

```javascript
let polishBtn = null;       // #aiChatPolishBtn
let polishOriginalText = ''; // 用于还原的原文快照
```

初始化引用：
```javascript
polishBtn = document.getElementById('aiChatPolishBtn');
```

#### 3b. 新增独立系统提示词（与「文本润色」技能区分）

在 `SKILL_PROMPTS` 常量块旁边（~line 98-99），新增独立常量：

```javascript
// 优化表达提示词（输入框内嵌按钮专用，与下拉菜单的「文本润色」技能区分）
const OPTIMIZE_EXPRESSION_PROMPT = `# Role: 表达优化助手

## Core Task
对用户输入的文本进行表达优化，在保留原意的前提下使表达更清晰、自然、地道。

## Guidelines
- 保持原文的核心信息、事实和数据完全不变
- 优化句式结构，使表达更流畅易读
- 替换生硬、啰嗦或不通顺的措辞，提升自然度
- 适当调整语气，使其更符合日常交流习惯
- 专业术语和专有名词保持原样不做替换
- 只输出优化后的文本，不添加任何解释、备注或额外内容
- 如果原文已经表达得很好了，可以不做改动直接返回原文`;
```

#### 3c. 输入监听：控制按钮 disabled 状态

在输入框的 `input` 事件处理中（~line 603-610），增加按钮启用/禁用逻辑：

```javascript
// 在 input 事件中已有 adjustHeight 和 sendBtn 逻辑之后添加
if (polishBtn) {
    if (text.length > 0 && !isStreaming) {
        polishBtn.disabled = false;
    } else {
        polishBtn.disabled = true;
        polishBtn.classList.remove('is-revert');
        polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
    }
}
```

同时在 `startStreaming` 中停止按钮时禁用润色按钮，流结束后恢复。

#### 3d. 优化表达按钮点击处理（使用独立提示词）

```javascript
if (polishBtn) {
    polishBtn.addEventListener('click', async () => {
        const text = inputEl.value.trim();
        if (!text || isStreaming) return;

        // 如果当前是「还原」状态
        if (polishBtn.classList.contains('is-revert')) {
            inputEl.value = polishOriginalText;
            inputEl.style.height = 'auto';
            polishBtn.classList.remove('is-revert');
            polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
            polishBtn.disabled = false;
            return;
        }

        // 保存原文快照
        polishOriginalText = text;
        polishBtn.disabled = true;
        polishBtn.querySelector('.ai-chat-polish-text').textContent = '...';

        try {
            const messages = [
                { role: 'system', content: OPTIMIZE_EXPRESSION_PROMPT },
                { role: 'user', content: text }
            ];
            const result = await window.go.main.App.CallAI(messages);
            if (result) {
                inputEl.value = result;
                inputEl.style.height = 'auto';
                polishBtn.classList.add('is-revert');
                polishBtn.querySelector('.ai-chat-polish-text').textContent = '还原';
                polishBtn.disabled = false;
            }
        } catch (e) {
            console.warn('优化表达失败:', e);
            polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
            polishBtn.disabled = false;
        }
    });
}
```

#### 3e. 发送消息时重置润色状态

在 `onSend()` 中清空输入框后（~line 1405），重置按钮状态：

```javascript
// 在 inputEl.value = '' 之后
if (polishBtn) {
    polishBtn.classList.remove('is-revert');
    polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
    polishBtn.disabled = true;
    polishOriginalText = '';
}
```

#### 3f. 流式开始时禁用按钮

在 `startStreaming()` 中（~line 1453 附近），显示停止按钮的同时禁用按钮：

```javascript
// 在 stopBtnEl.style.display 逻辑旁添加
if (polishBtn) polishBtn.disabled = true;
```

流结束/错误时恢复启用判断：

```javascript
// 在 ai:stream-done / ai:stream-error 回调中，isStreaming = false 后
if (polishBtn) {
    polishBtn.disabled = !(inputEl.value.trim().length > 0);
}
```

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/index.html` | textarea 加包裹容器 `ai-chat-input-wrap`，插入 `aiChatPolishBtn` 按钮 |
| `frontend/src/css/components/ai-chat.css` | 新增 `.ai-chat-input-wrap`、`.ai-chat-polish-btn`、`.is-revert` 样式 |
| `frontend/src/js/ai-chat.js` | 新增 `OPTIMIZE_EXPRESSION_PROMPT` 常量 + 按钮交互逻辑 + AI 调用 + 状态管理 |

## 用户体验流程

1. 用户在 textarea 中输入内容 → 按钮出现（鼠标悬停在输入区域时显示）
2. 点击「优化」→ 按钮变为「...」→ AI 返回优化结果 → 内容替换 → 按钮变为橙色「还原」
3. 点击「还原」→ 恢复原文 → 按钮变回「优化」
4. 用户发送消息 → 按钮重置为禁用状态
5. 流式输出期间按钮不可用

## 验证

1. textarea 为空时按钮隐藏（disabled）
2. 输入内容后 hover 输入区域，按钮可见可点击
3. 点击「优化」调用 AI（使用独立提示词），内容替换成功
4. 替换后按钮变为「还原」，点击恢复原文
5. 发送消息后按钮重置
6. 流式输出期间按钮不可点击
7. 现有「更多技能 → 文本润色」不受影响