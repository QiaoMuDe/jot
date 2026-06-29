/**
 * AI 对话模块 — 持久化多会话支持
 */
import { marked } from 'marked';
import hljs from 'highlight.js';

let messagesEl = null;        // #aiChatMessages
let inputEl = null;           // #aiChatInput
let sendBtnEl = null;         // #aiChatSendBtn
let emptyEl = null;           // #aiChatEmpty
let inputAreaEl = null;       // #aiChatInputArea
let clearBtnEl = null;        // #aiChatClearBtn
let sessionListEl = null;     // #aiSessionList
let sessionNewBtnEl = null;   // #aiSessionNewBtn

// 状态
let chatHistory = [];          // 当前会话的消息（发送给模型用）
let activeSessionId = null;    // null = 新会话尚未保存
let sessions = [];             // 侧栏会话列表
let isStreaming = false;       // 正在流式输出时禁止切换/发送

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
    sessionListEl = document.getElementById('aiSessionList');
    sessionNewBtnEl = document.getElementById('aiSessionNewBtn');

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
        backBtn.addEventListener('click', () => window.switchView('grid'));
    }

    // 清空当前对话
    if (clearBtnEl) {
        clearBtnEl.addEventListener('click', async () => {
            if (activeSessionId === null || messagesEl.children.length === 0) return;
            const confirmed = await window.showConfirmDialog('确定清空当前对话吗？');
            if (!confirmed) return;

            try {
                await window.go.main.App.ClearAISessionMessages(activeSessionId);
            } catch (_) { /* 忽略 */ }

            messagesEl.innerHTML = '';
            chatHistory = [];
            updateClearBtn();
            scrollToBottom();
        });
    }

    // 新建会话
    if (sessionNewBtnEl) {
        sessionNewBtnEl.addEventListener('click', createSession);
    }

    // 前往设置
    const goSettingsBtn = document.getElementById('aiChatGoSettings');
    if (goSettingsBtn) {
        goSettingsBtn.addEventListener('click', () => window.switchView('settings'));
    }

    // 输入框事件
    if (inputEl) {
        inputEl.addEventListener('input', () => {
            sendBtnEl.disabled = inputEl.value.trim().length === 0;
        });
        inputEl.addEventListener('keydown', onInputKeydown);
        inputEl.addEventListener('input', autoResizeInput);
    }

    // 发送按钮
    if (sendBtnEl) {
        sendBtnEl.addEventListener('click', onSend);
    }

    // 侧栏折叠/展开
    const toggleBtn = document.getElementById('aiSidebarToggle');
    const sidebar = document.querySelector('.ai-session-sidebar');
    if (toggleBtn && sidebar) {
        const chevronLeft = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>';
        const chevronRight = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';

        // 恢复保存的状态（默认折叠）
        const saved = localStorage.getItem('ai_sidebar_collapsed');
        if (saved === 'false') {
            sidebar.classList.remove('collapsed');
            toggleBtn.innerHTML = chevronRight;
            toggleBtn.title = '折叠侧栏';
        } else {
            sidebar.classList.add('collapsed');
            toggleBtn.innerHTML = chevronLeft;
            toggleBtn.title = '展开侧栏';
        }

        toggleBtn.addEventListener('click', () => {
            const isCollapsed = sidebar.classList.toggle('collapsed');
            toggleBtn.innerHTML = isCollapsed ? chevronLeft : chevronRight;
            toggleBtn.title = isCollapsed ? '展开侧栏' : '折叠侧栏';
            localStorage.setItem('ai_sidebar_collapsed', String(!isCollapsed));
        });
    }
}

/* ── 会话侧栏管理 ── */

/**
 * 加载会话列表并渲染侧栏
 */
async function loadSessionList() {
    try {
        sessions = await window.go.main.App.GetAISessions() || [];
    } catch (_) {
        sessions = [];
    }
    renderSessionList();
}

/**
 * 渲染侧栏会话项
 */
function renderSessionList() {
    if (!sessionListEl) return;

    if (sessions.length === 0) {
        sessionListEl.innerHTML = '<div class="ai-session-empty">暂无会话</div>';
        return;
    }

    sessionListEl.innerHTML = '';
    sessions.forEach(s => {
        const item = document.createElement('div');
        item.className = 'ai-session-item' + (s.id === activeSessionId ? ' active' : '');
        item.dataset.id = s.id;

        const title = document.createElement('span');
        title.className = 'ai-session-item-title';
        title.textContent = s.title;
        title.title = s.title;
        item.appendChild(title);

        const delBtn = document.createElement('button');
        delBtn.className = 'ai-session-item-delete';
        delBtn.textContent = '✕';
        delBtn.title = '删除会话';
        item.appendChild(delBtn);

        // 点击切换会话
        item.addEventListener('click', (e) => {
            if (e.target === delBtn) return;
            switchSession(s.id);
        });

        // 双击内联编辑
        title.addEventListener('dblclick', () => {
            if (isStreaming) return;
            const orig = title.textContent;
            title.contentEditable = 'true';
            title.focus();
            // 全选
            const range = document.createRange();
            range.selectNodeContents(title);
            const sel = window.getSelection();
            sel.removeAllRanges();
            sel.addRange(range);

            const finish = async () => {
                title.contentEditable = 'false';
                const newTitle = title.textContent.trim();
                if (newTitle && newTitle !== orig) {
                    try {
                        await window.go.main.App.RenameAISession(s.id, newTitle);
                        s.title = newTitle;
                        title.title = newTitle;
                    } catch (_) {
                        title.textContent = orig;
                    }
                } else {
                    title.textContent = orig;
                }
            };

            title.addEventListener('blur', finish, { once: true });
            title.addEventListener('keydown', (ke) => {
                if (ke.key === 'Enter') {
                    ke.preventDefault();
                    title.blur();
                }
                if (ke.key === 'Escape') {
                    title.textContent = orig;
                    title.contentEditable = 'false';
                }
            }, { once: true });
        });

        // 删除会话
        delBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            if (isStreaming) return;
            const confirmed = await window.showConfirmDialog('确定删除此会话吗？');
            if (!confirmed) return;

            try {
                await window.go.main.App.DeleteAISession(s.id);
            } catch (_) { /* 忽略 */ }

            // 如果删除的是当前会话，切换到最近会话或新建
            if (s.id === activeSessionId) {
                activeSessionId = null;
                chatHistory = [];
                messagesEl.innerHTML = '';
                updateClearBtn();
            }

            await loadSessionList();
            // 如果删除后没有会话了，自动新建一个
            if (sessions.length === 0) {
                await createSession();
            } else if (activeSessionId === null) {
                // 切换到最近一个会话
                switchSession(sessions[0].id);
            }
        });

        sessionListEl.appendChild(item);
    });
}

/**
 * 切换会话
 */
async function switchSession(id) {
    if (isStreaming || id === activeSessionId) return;

    try {
        activeSessionId = id;
        const msgs = await window.go.main.App.LoadAISessionMessages(id);

        // 重建 chatHistory（只保留 role/content 供 API 使用）
        chatHistory = msgs ? msgs.map(msg => ({ role: msg.role, content: msg.content })) : [];

        // 清空消息列表
        messagesEl.innerHTML = '';

        if (!msgs || msgs.length === 0) {
            updateClearBtn();
            renderSessionList();
            scrollToBottom();
            return;
        }

        // 按原始顺序渲染，role 直接从服务端返回
        msgs.forEach(msg => {
            if (msg.role === 'user') {
                addMessage(msg.content, 'user');
                const userMsgEl = messagesEl.lastElementChild;
                if (userMsgEl) userMsgEl.appendChild(createMsgActions(msg.content, 'user'));
            } else if (msg.role === 'assistant') {
                const el = addMessage(msg.content, 'assistant', msg.reasoning_content || '');
                el.appendChild(createMsgActions(msg.content, 'assistant'));
            }
        });

        renderSessionList();
        updateClearBtn();
        scrollToBottom();
    } catch (_) { /* 静默失败 */ }
}

/**
 * 创建新会话
 */
async function createSession() {
    if (isStreaming) return;

    let id;
    try {
        id = await window.go.main.App.CreateAISession();
    } catch (_) {
        return;
    }

    // 清空当前状态
    activeSessionId = id;
    chatHistory = [];
    messagesEl.innerHTML = '';
    updateClearBtn();
    hideEmptyState();

    await loadSessionList();
    scrollToBottom();
}

/* ── 对话管理 ── */

/**
 * 输入框键盘事件
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
 * 发送消息
 */
async function onSend() {
    const text = inputEl.value.trim();
    if (!text || isStreaming) return;

    // 如果没有激活的会话，自动创建
    if (activeSessionId === null) {
        await createSession();
        if (activeSessionId === null) return;
    }

    inputEl.value = '';
    inputEl.style.height = 'auto';
    sendBtnEl.disabled = true;

    addMessage(text, 'user');
    const userMsgEl = messagesEl.lastElementChild;
    if (userMsgEl) userMsgEl.appendChild(createMsgActions(text, 'user'));
    chatHistory.push({ role: 'user', content: text });

    startStreaming();
}

/**
 * 启动流式输出
 */
function startStreaming() {
    if (isStreaming) return;
    isStreaming = true;

    let streamingContent = '';
    let streamingThinking = '';
    let hasReceivedChunk = false;

    const streamingEl = document.createElement('div');
    streamingEl.className = 'ai-msg ai-msg-assistant';
    const contentDiv = document.createElement('div');
    contentDiv.className = 'msg-content';
    contentDiv.appendChild(createTypingDots());
    streamingEl.appendChild(contentDiv);
    messagesEl.appendChild(streamingEl);

    let thinkingDetails = null;
    let thinkingContentEl = null;
    let unsubs = [];

    const unsubThinking = window.runtime.EventsOn('ai:stream-thinking', chunk => {
        if (!thinkingDetails) {
            thinkingDetails = document.createElement('details');
            thinkingDetails.className = 'thinking-details';
            thinkingDetails.open = true;
            const summary = document.createElement('summary');
            summary.className = 'thinking-summary';
            summary.textContent = '💭 思考过程';
            thinkingDetails.appendChild(summary);
            thinkingContentEl = document.createElement('div');
            thinkingContentEl.className = 'thinking-content';
            thinkingDetails.appendChild(thinkingContentEl);
            streamingEl.insertBefore(thinkingDetails, contentDiv);
        }
        streamingThinking += chunk;
        thinkingContentEl.textContent = streamingThinking;
        scrollToBottom();
    });
    unsubs.push(unsubThinking);

    const unsubChunk = window.runtime.EventsOn('ai:stream-chunk', chunk => {
        if (!hasReceivedChunk) {
            hasReceivedChunk = true;
            contentDiv.innerHTML = '';
        }
        streamingContent += chunk;
        contentDiv.textContent = streamingContent;
        scrollToBottom();
    });
    unsubs.push(unsubChunk);

    const unsubDone = window.runtime.EventsOn('ai:stream-done', fullContent => {
        unsubs.forEach(fn => fn());
        isStreaming = false;

        if (!hasReceivedChunk) contentDiv.innerHTML = '';
        if (streamingContent === '' && fullContent) streamingContent = fullContent;

        const finalContent = streamingContent || fullContent;
        renderMarkdown(contentDiv, finalContent);

        if (thinkingDetails && thinkingContentEl && streamingThinking) {
            const summary = thinkingDetails.querySelector('.thinking-summary');
            if (summary) summary.textContent = '💭 已思考';
            renderMarkdown(thinkingContentEl, streamingThinking);
        }

        chatHistory.push({ role: 'assistant', content: finalContent });
        streamingEl.appendChild(createMsgActions(finalContent, 'assistant'));

        // 自动保存消息到数据库
        saveSessionMessages([{ role: 'user', content: chatHistory[chatHistory.length - 2].content }, { role: 'assistant', content: finalContent, reasoning_content: streamingThinking || '' }]);

        updateClearBtn();
        scrollToBottom();
    });
    unsubs.push(unsubDone);

    const unsubError = window.runtime.EventsOn('ai:stream-error', err => {
        unsubs.forEach(fn => fn());
        isStreaming = false;
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();
        addErrorMessage(err);
    });
    unsubs.push(unsubError);

    try {
        window.go.main.App.CallAIStream(chatHistory);
    } catch (e) {
        unsubs.forEach(fn => fn());
        isStreaming = false;
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();
        addErrorMessage('流式调用失败: ' + (e.message || e));
    }
}

/**
 * 保存一轮对话消息并刷新侧栏
 */
async function saveSessionMessages(roundMessages) {
    if (activeSessionId === null) return;
    try {
        await window.go.main.App.SaveAIMessages(activeSessionId, roundMessages);
        // 更新侧栏标题和顺序
        await loadSessionList();
    } catch (_) { /* 静默 */ }
}

/* ── 渲染与 UI ── */

/**
 * 渲染 Markdown + 代码高亮
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
 * 添加消息气泡（不含操作按钮，调用方自行添加）
 * @param {string} content - 消息内容
 * @param {'user'|'assistant'} role - 角色
 * @param {string} [reasoningContent] - 思维链内容（可选）
 */
function addMessage(content, role, reasoningContent) {
    const el = document.createElement('div');
    el.className = 'ai-msg ' + (role === 'user' ? 'ai-msg-user' : 'ai-msg-assistant');

    // 如果有思维链内容，先渲染可折叠思考区域
    if (role === 'assistant' && reasoningContent) {
        const details = document.createElement('details');
        details.className = 'thinking-details';
        details.open = true;
        const summary = document.createElement('summary');
        summary.className = 'thinking-summary';
        summary.textContent = '💭 已思考';
        details.appendChild(summary);
        const thinkingEl = document.createElement('div');
        thinkingEl.className = 'thinking-content';
        renderMarkdown(thinkingEl, reasoningContent);
        details.appendChild(thinkingEl);
        el.appendChild(details);
    }

    const contentEl = document.createElement('div');
    contentEl.className = 'msg-content';
    if (role === 'assistant') {
        renderMarkdown(contentEl, content);
    } else {
        contentEl.textContent = content;
    }
    el.appendChild(contentEl);

    messagesEl.appendChild(el);
    scrollToBottom();
    return el;
}

/**
 * 创建打字指示器 DOM 片段
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
 * 视图激活时调用
 */
export async function onAIChatViewActivated() {
    if (!messagesEl) return;

    try {
        const cfg = await window.go.main.App.GetAIConfig();
        if (!cfg.api_key || !cfg.base_url) {
            showEmptyState();
            return;
        }
        hideEmptyState();
        await loadSessionList();

        // 仅在未激活会话时才自动加载第一个
        if (activeSessionId === null) {
            if (sessions.length > 0) {
                await switchSession(sessions[0].id);
            } else {
                await createSession();
            }
        }
    } catch (_) {
        showEmptyState();
    }
}

function showEmptyState() {
    if (!emptyEl) return;
    emptyEl.style.display = '';
    if (messagesEl) messagesEl.style.display = 'none';
    if (inputAreaEl) inputAreaEl.style.display = 'none';
    if (clearBtnEl) clearBtnEl.style.display = 'none';
    // 侧栏仍可见但禁用操作
    if (sessionNewBtnEl) sessionNewBtnEl.style.display = 'none';
}

function hideEmptyState() {
    if (!emptyEl) return;
    emptyEl.style.display = 'none';
    if (messagesEl) messagesEl.style.display = '';
    if (inputAreaEl) inputAreaEl.style.display = '';
    if (sessionNewBtnEl) sessionNewBtnEl.style.display = '';
    updateClearBtn();
}

/* ── SVG 图标 ── */
const COPY_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
const REGEN_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>';
const CHECK_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';

/**
 * 创建消息气泡操作按钮
 */
function createMsgActions(content, role) {
    const container = document.createElement('div');
    container.className = 'ai-msg-actions';

    const copyBtn = document.createElement('button');
    copyBtn.innerHTML = COPY_ICON;
    copyBtn.title = '复制';
    copyBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        handleCopy(content, copyBtn);
    });
    container.appendChild(copyBtn);

    if (role === 'assistant') {
        const regenBtn = document.createElement('button');
        regenBtn.innerHTML = REGEN_ICON;
        regenBtn.title = '重新生成';
        regenBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            handleRegenerate(container.parentElement);
        });
        container.appendChild(regenBtn);
    }

    return container;
}

function handleCopy(text, btn) {
    navigator.clipboard.writeText(text).then(() => {
        btn.classList.add('copied');
        btn.innerHTML = CHECK_ICON;
        setTimeout(() => {
            btn.classList.remove('copied');
            btn.innerHTML = COPY_ICON;
        }, 500);
    }).catch(() => {});
}

function handleRegenerate(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const children = Array.from(messagesEl.children);
    const idx = children.indexOf(msgEl);
    if (idx === -1) return;

    chatHistory.splice(idx);
    children.slice(idx).forEach(el => el.remove());

    startStreaming();
}
