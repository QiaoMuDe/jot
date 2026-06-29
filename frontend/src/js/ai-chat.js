/**
 * AI 对话模块 — 自实现聊天引擎，支持流式输出
 */
import { marked } from 'marked';
import hljs from 'highlight.js';

let messagesEl = null;      // #aiChatMessages
let inputEl = null;         // #aiChatInput
let sendBtnEl = null;       // #aiChatSendBtn
let emptyEl = null;         // #aiChatEmpty
let inputAreaEl = null;     // #aiChatInputArea
let clearBtnEl = null;      // #aiChatClearBtn

// 对话历史
let chatHistory = [];
// 当前流式消息的元素和累积内容
let streamingEl = null;
let streamingContent = '';

/**
 * 初始化 AI 对话页面
 */
export function initAIChat() {
    messagesEl = document.getElementById('aiChatMessages');
    inputEl = document.getElementById('aiChatInput');
    sendBtnEl = document.getElementById('aiChatSendBtn');
    emptyEl = document.getElementById('aiChatEmpty');
    inputAreaEl = document.getElementById('aiChatInputArea');
    clearBtnEl = document.getElementById('aiChatClearBtn');

    if (!messagesEl) return;

    bindEvents();
}

/**
 * 绑定所有事件
 */
function bindEvents() {
    // 返回按钮
    const backBtn = document.getElementById('aiChatBackBtn');
    if (backBtn) {
        backBtn.addEventListener('click', () => {
            window.switchView('grid');
        });
    }

    // 清空对话
    if (clearBtnEl) {
        clearBtnEl.addEventListener('click', async () => {
            if (messagesEl.children.length === 0) return;
            const confirmed = await window.showConfirmDialog('确定清空当前对话吗？');
            if (!confirmed) return;
            messagesEl.innerHTML = '';
            chatHistory = [];
            streamingEl = null;
            streamingContent = '';
            updateClearBtn();
        });
    }

    // 前往设置
    const goSettingsBtn = document.getElementById('aiChatGoSettings');
    if (goSettingsBtn) {
        goSettingsBtn.addEventListener('click', () => {
            window.switchView('settings');
        });
    }

    // 输入框事件
    if (inputEl) {
        inputEl.addEventListener('input', onInputChange);
        inputEl.addEventListener('keydown', onInputKeydown);
        inputEl.addEventListener('input', autoResizeInput);
    }

    // 发送按钮
    if (sendBtnEl) {
        sendBtnEl.addEventListener('click', onSend);
    }
}

/**
 * 输入内容变化，控制发送按钮状态
 */
function onInputChange() {
    if (!sendBtnEl) return;
    sendBtnEl.disabled = inputEl.value.trim().length === 0;
}

/**
 * 键盘事件：Enter 发送，Shift+Enter 换行
 */
function onInputKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        onSend();
    }
}

/**
 * auto-resize textarea
 */
function autoResizeInput() {
    inputEl.style.height = 'auto';
    inputEl.style.height = Math.min(inputEl.scrollHeight, 140) + 'px';
}

/**
 * 发送消息（流式输出）
 */
async function onSend() {
    const text = inputEl.value.trim();
    if (!text) return;

    // 清空输入
    inputEl.value = '';
    inputEl.style.height = 'auto';
    sendBtnEl.disabled = true;

    // 添加用户消息
    addMessage(text, 'user');
    chatHistory.push({ role: 'user', content: text });

    // 创建空的 AI 消息容器（内含打字指示器）
    streamingContent = '';
    streamingEl = document.createElement('div');
    streamingEl.className = 'ai-msg ai-msg-assistant';
    const contentDiv = document.createElement('div');
    contentDiv.className = 'msg-content';
    // 在内容区内置打字指示器，避免独立气泡
    contentDiv.appendChild(createTypingDots());
    streamingEl.appendChild(contentDiv);
    messagesEl.appendChild(streamingEl);

    // 事件监听清理函数
    let unsubs = [];
    let hasReceivedChunk = false;

    // 流式块事件
    const unsubChunk = window.runtime.EventsOn('ai:stream-chunk', chunk => {
        if (!hasReceivedChunk) {
            hasReceivedChunk = true;
            contentDiv.innerHTML = ''; // 清除打字指示器
        }
        streamingContent += chunk;
        contentDiv.textContent = streamingContent;
        scrollToBottom();
    });
    unsubs.push(unsubChunk);

    // 流结束事件
    const unsubDone = window.runtime.EventsOn('ai:stream-done', fullContent => {
        unsubs.forEach(fn => fn());
        if (!hasReceivedChunk) {
            contentDiv.innerHTML = ''; // 清除打字指示器
        }
        // 如果流期间没有收到任何块（空回复），直接用 fullContent
        if (streamingContent === '' && fullContent) {
            streamingContent = fullContent;
        }
        // 最终渲染 Markdown + 代码高亮
        renderMarkdown(contentDiv, streamingContent || fullContent);
        chatHistory.push({ role: 'assistant', content: streamingContent || fullContent });
        streamingEl = null;
        streamingContent = '';
        updateClearBtn();
        scrollToBottom();
    });
    unsubs.push(unsubDone);

    // 流错误事件
    const unsubError = window.runtime.EventsOn('ai:stream-error', err => {
        unsubs.forEach(fn => fn());
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();
        addErrorMessage(err);
        streamingEl = null;
        streamingContent = '';
    });
    unsubs.push(unsubError);

    // 发起流式调用（fire-and-forget）
    try {
        window.go.main.App.CallAIStream(chatHistory);
    } catch (e) {
        unsubs.forEach(fn => fn());
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();
        addErrorMessage('流式调用失败: ' + (e.message || e));
        streamingEl = null;
        streamingContent = '';
    }
}

/**
 * 渲染 Markdown + 代码高亮到元素
 */
function renderMarkdown(el, content) {
    marked.setOptions({
        breaks: true,
        gfm: true,
        highlight: function(code, lang) {
            if (lang && hljs.getLanguage(lang)) {
                try { return hljs.highlight(code, { language: lang }).value; } catch (_) {}
            }
            try { return hljs.highlightAuto(code).value; } catch (_) {}
            return code;
        }
    });
    el.innerHTML = marked.parse(content);
    // 代码块语言标签
    el.querySelectorAll('pre code').forEach((codeEl) => {
        const pre = codeEl.parentElement;
        const lang = codeEl.className.match(/language-(\w+)/);
        if (lang) {
            const badge = document.createElement('span');
            badge.className = 'code-lang-badge';
            badge.textContent = lang[1];
            pre.style.position = 'relative';
            pre.appendChild(badge);
        }
    });
}

/**
 * 添加消息气泡（非流式）
 */
function addMessage(content, role) {
    const el = document.createElement('div');
    el.className = 'ai-msg ' + (role === 'user' ? 'ai-msg-user' : 'ai-msg-assistant');

    if (role === 'assistant') {
        const contentEl = document.createElement('div');
        contentEl.className = 'msg-content';
        renderMarkdown(contentEl, content);
        el.appendChild(contentEl);
    } else {
        el.textContent = content;
    }

    messagesEl.appendChild(el);
    scrollToBottom();
    return el;
}

/**
 * 创建打字指示器 DOM 片段（三圆点），不追加到消息列表
 */
function createTypingDots() {
    const el = document.createElement('span');
    el.className = 'ai-msg-typing';
    for (let i = 0; i < 3; i++) {
        const dot = document.createElement('span');
        dot.className = 'ai-typing-dot';
        el.appendChild(dot);
    }
    return el;
}

/**
 * 显示错误消息
 */
function addErrorMessage(msg) {
    const el = document.createElement('div');
    el.className = 'ai-msg-error';
    el.textContent = msg;
    messagesEl.appendChild(el);
    scrollToBottom();
}

/**
 * 滚动到底部
 */
function scrollToBottom() {
    requestAnimationFrame(() => {
        messagesEl.scrollTop = messagesEl.scrollHeight;
    });
}

/**
 * 更新清空按钮状态
 */
function updateClearBtn() {
    if (!clearBtnEl) return;
    clearBtnEl.style.display = messagesEl.children.length === 0 ? 'none' : '';
}

/**
 * 视图激活时调用 — 检查配置状态
 */
export async function onAIChatViewActivated() {
    if (!messagesEl) return;

    try {
        const cfg = await window.go.main.App.GetAIConfig();
        if (!cfg.api_key || !cfg.base_url) {
            emptyEl.style.display = '';
            messagesEl.style.display = 'none';
            if (inputAreaEl) inputAreaEl.style.display = 'none';
            return;
        }

        emptyEl.style.display = 'none';
        messagesEl.style.display = '';
        if (inputAreaEl) inputAreaEl.style.display = '';
        updateClearBtn();
    } catch (e) {
        emptyEl.style.display = '';
        messagesEl.style.display = 'none';
        if (inputAreaEl) inputAreaEl.style.display = 'none';
    }
}
