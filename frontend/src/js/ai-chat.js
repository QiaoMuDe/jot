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
let stopBtnEl = null;         // #aiChatStopBtn
let sessionListEl = null;     // #aiSessionList
let sessionNewBtnEl = null;   // #aiSessionNewBtn

// 状态
let chatHistory = [];          // 当前会话的消息（发送给模型用）
let activeSessionId = null;    // null = 新会话尚未保存
let sessions = [];             // 侧栏会话列表
let sessionSearchQuery = '';
let sessionSearchEl = null;
let isStreaming = false;       // 正在流式输出时禁止切换/发送
let sessionContextMenu = null; // AI 会话右键菜单
let _contextSessionId = null;  // 右键菜单当前会话 ID
let _contextTitleEl = null;    // 右键菜单当前会话标题元素
let _aiStreamGen = 0;          // 流式 generation 计数器，跨流防串扰

// 模型选择器状态
let modelTrigger = null;
let modelDropdown = null;
let modelLabel = null;
let modelList = [];

// 深度思考状态
let searchToggle = null;
let enableThinking = false;

// 笔记引用状态
let referencedNotes = [];       // { id, title, notebook_name }

// 笔记引用选择浮层 DOM
let refBtn = null;              // #aiChatRefBtn
let refBar = null;              // #aiChatRefBar
let refChips = null;            // #aiChatRefChips
let refModal = null;            // #aiNoteRefModal
let refOverlay = null;          // #aiNoteRefOverlay
let refSearch = null;           // #aiNoteRefSearch
let refNotebook = null;         // #aiNoteRefNotebook
let refList = null;             // #aiNoteRefList
let refCount = null;            // #aiNoteRefCount
let refConfirm = null;          // #aiNoteRefConfirm
let refCancel = null;           // #aiNoteRefCancel
let refClose = null;            // #aiNoteRefClose
let refSkeleton = null;         // #aiNoteRefSkeleton
let refLoadingOverlay = null;   // #aiNoteRefLoadingOverlay
let refSearchClear = null;      // #aiNoteRefSearchClear
let _refSearchTimer = null;     // 搜索 debounce 定时器
let _refTempSelected = {};      // 浮层中临时选中状态 { [id]: true }
let _refListLoaded = false;     // 是否已加载过列表（首次用骨架屏，后续用 overlay）
let _refCurrentPage = 1;        // 当前页码
let _refTotalItems = 0;         // 匹配总数
let _refPageSize = 20;          // 每页条数（从设置读取）
let _refLoading = false;        // 是否正在加载中

/**
 * 加载模型配置到选择器 UI
 */
async function loadModelSelector(cfg) {
    const model = cfg?.model || '--';
    if (modelLabel) modelLabel.textContent = model;
}

/**
 * 打开下拉并填充模型列表
 */
async function openModelDropdown() {
    if (!modelDropdown) return;
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
    if (!modelDropdown || !modelLabel) return;
    const current = modelLabel.textContent;
    modelDropdown.innerHTML = modelList.map(m =>
        `<div class="theme-select-item${m === current ? ' active' : ''}" data-model="${m}">${m}</div>`
    ).join('');
}

/**
 * 切换模型并保存
 */
async function switchModel(model) {
    if (!modelLabel || !modelDropdown || !model) return;
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
    stopBtnEl = document.getElementById('aiChatStopBtn');
    sessionListEl = document.getElementById('aiSessionList');
    sessionNewBtnEl = document.getElementById('aiSessionNewBtn');
    sessionSearchEl = document.getElementById('aiSessionSearch');

    // 模型选择器
    modelTrigger = document.getElementById('aiChatModelTrigger');
    modelDropdown = document.getElementById('aiChatModelDropdown');
    modelLabel = document.getElementById('aiChatModelLabel');
    modelList = [];

    // 深度思考
    searchToggle = document.getElementById('aiChatSearchToggle');
    enableThinking = localStorage.getItem('ai_thinking_enabled') === 'true';

    // 笔记引用
    refBtn = document.getElementById('aiChatRefBtn');
    refBar = document.getElementById('aiChatRefBar');
    refChips = document.getElementById('aiChatRefChips');
    refModal = document.getElementById('aiNoteRefModal');
    refOverlay = document.getElementById('aiNoteRefOverlay');
    refSearch = document.getElementById('aiNoteRefSearch');
    refNotebook = document.getElementById('aiNoteRefNotebook');
    refList = document.getElementById('aiNoteRefList');
    refCount = document.getElementById('aiNoteRefCount');
    refConfirm = document.getElementById('aiNoteRefConfirm');
    refCancel = document.getElementById('aiNoteRefCancel');
    refClose = document.getElementById('aiNoteRefClose');
    refSkeleton = document.getElementById('aiNoteRefSkeleton');
    refLoadingOverlay = document.getElementById('aiNoteRefLoadingOverlay');
    refSearchClear = document.getElementById('aiNoteRefSearchClear');
    _refListLoaded = false;

    if (!messagesEl) return;

    sessionContextMenu = document.getElementById('aiSessionContextMenu');

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
            if (activeSessionId === null || messagesEl.children.length === 0) {
                window.showNotification?.('当前没有对话可以清空', 'info');
                return;
            }
            const confirmed = await window.showConfirmDialog('确定清空当前对话吗？');
            if (!confirmed) return;

            try {
                await window.go.main.App.ClearAISessionMessages(activeSessionId);
            } catch (_) { /* 静态失败 */ }

            messagesEl.innerHTML = '';
            chatHistory = [];
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

    // 停止生成
    if (stopBtnEl) {
        stopBtnEl.addEventListener('click', async () => {
            try {
                await window.go.main.App.CancelAIStream();
            } catch (_) {}
        });
    }

    // 侧栏折叠/展开
    const toggleBtn = document.getElementById('aiSidebarToggle');
    const sidebar = document.querySelector('.ai-session-sidebar');
    if (toggleBtn && sidebar) {
        const chevronLeft = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>';
        const chevronRight = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';

        // 恢复保存的状态（默认展开）
        const saved = localStorage.getItem('ai_sidebar_collapsed');
        if (saved === 'false') {
            sidebar.classList.add('collapsed');
            toggleBtn.innerHTML = chevronLeft;
            toggleBtn.title = '展开侧栏';
        } else {
            sidebar.classList.remove('collapsed');
            toggleBtn.innerHTML = chevronRight;
            toggleBtn.title = '折叠侧栏';
        }

        toggleBtn.addEventListener('click', () => {
            const isCollapsed = sidebar.classList.toggle('collapsed');
            toggleBtn.innerHTML = isCollapsed ? chevronLeft : chevronRight;
            toggleBtn.title = isCollapsed ? '展开侧栏' : '折叠侧栏';
            localStorage.setItem('ai_sidebar_collapsed', String(!isCollapsed));
        });
    }

    // 对话搜索
    if (sessionSearchEl) {
        sessionSearchEl.addEventListener('input', () => {
            sessionSearchQuery = sessionSearchEl.value.trim().toLowerCase();
            renderSessionList();
        });
    }

    // ── 模型选择器事件 ──
    if (modelTrigger && modelDropdown) {
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

        document.addEventListener('click', () => modelDropdown.classList.remove('open'));
    }

    // ── 深度思考切换 ──
    if (searchToggle) {
        if (enableThinking) searchToggle.classList.add('active');
        searchToggle.addEventListener('click', () => {
            enableThinking = searchToggle.classList.toggle('active');
            localStorage.setItem('ai_thinking_enabled', String(enableThinking));
            // 同步设置页 toggle
            const settingToggle = document.getElementById('aiSettingSearchToggle');
            if (settingToggle) {
                settingToggle.classList.toggle('active', enableThinking);
            }
        });
    }

    // ── 笔记引用按钮 ──
    if (refBtn) {
        refBtn.addEventListener('click', openNoteRefModal);
    }

    // ── 笔记引用浮层 ──
    if (refOverlay) {
        refOverlay.addEventListener('click', closeNoteRefModal);
    }
    if (refClose) {
        refClose.addEventListener('click', closeNoteRefModal);
    }
    if (refCancel) {
        refCancel.addEventListener('click', closeNoteRefModal);
    }
    if (refConfirm) {
        refConfirm.addEventListener('click', confirmNoteSelection);
    }
    if (refSearch) {
        refSearch.addEventListener('input', () => {
            clearTimeout(_refSearchTimer);
            _refSearchTimer = setTimeout(loadNoteList, 300);
        });
        // Enter 键触发搜索
        refSearch.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                clearTimeout(_refSearchTimer);
                loadNoteList();
            }
        });
    }
    if (refNotebook) {
        refNotebook.addEventListener('change', () => loadNoteList(false));
    }
    if (refSearchClear) {
        refSearchClear.addEventListener('click', () => {
            if (refSearch) {
                refSearch.value = '';
                refSearch.focus();
            }
            refSearchClear.classList.remove('visible');
            loadNoteList();
        });
    }
    // 搜索输入时切换清除按钮可见性
    if (refSearch) {
        refSearch.addEventListener('input', () => {
            refSearchClear?.classList.toggle('visible', refSearch.value.length > 0);
        });
    }

    // 浮层列表点击切换选中
    if (refList) {
        refList.addEventListener('click', (e) => {
            const loadMore = e.target.closest('.ai-note-ref-load-more');
            if (loadMore) {
                loadNoteList(true);
                return;
            }
            const item = e.target.closest('.ai-note-ref-item');
            if (item) toggleNoteSelection(item.dataset.id);
        });
    }

    // 点击菜单外区域关闭右键菜单
    document.addEventListener('click', (e) => {
        if (sessionContextMenu && !sessionContextMenu.contains(e.target)) {
            closeSessionContextMenu();
        }
        // 关闭笔记引用浮层（点击 overlay 外部不关闭）
    });

    // Escape 关闭右键菜单 & 笔记引用浮层
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            closeSessionContextMenu();
            if (refModal && refModal.style.display !== 'none') {
                closeNoteRefModal();
            }
        }
    });

    // 右键菜单项点击
    if (sessionContextMenu) {
        sessionContextMenu.addEventListener('click', async (e) => {
            const item = e.target.closest('.context-menu-item');
            if (!item) return;
            const action = item.dataset.action;
            const sessionId = _contextSessionId;
            const titleEl = _contextTitleEl;

            closeSessionContextMenu();

            if (action === 'rename') {
                // 触发内联编辑
                startInlineEdit(titleEl, sessionId);
            } else if (action === 'export') {
                try {
                    const result = await window.go.main.App.ExportAISessionAsMarkdown(sessionId);
                    if (result && result !== '已取消') {
                        window.showNotification?.(result, 'success');
                    }
                } catch (e) {
                    window.showNotification?.('导出失败: ' + (e.message || e), 'error');
                }
            }
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
 * 启动会话标题内联编辑
 * @param {HTMLElement} titleEl - 标题元素
 * @param {number} sessionId - 会话 ID
 */
function startInlineEdit(titleEl, sessionId) {
    if (isStreaming) return;
    if (!titleEl) return;
    const orig = titleEl.textContent;
    titleEl.contentEditable = 'true';
    titleEl.focus();
    // 全选
    const range = document.createRange();
    range.selectNodeContents(titleEl);
    const sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);

    const finish = async () => {
        titleEl.contentEditable = 'false';
        const newTitle = titleEl.textContent.trim();
        if (newTitle && newTitle !== orig) {
            try {
                await window.go.main.App.RenameAISession(sessionId, newTitle);
                // 也更新本地 sessions 数组中的标题
                const s = sessions.find(s => s.id === sessionId);
                if (s) s.title = newTitle;
                titleEl.title = newTitle;
            } catch (_) {
                titleEl.textContent = orig;
            }
        } else {
            titleEl.textContent = orig;
        }
    };

    titleEl.addEventListener('blur', finish, { once: true });
    titleEl.addEventListener('keydown', (ke) => {
        if (ke.key === 'Enter') {
            ke.preventDefault();
            titleEl.blur();
        }
        if (ke.key === 'Escape') {
            titleEl.textContent = orig;
            titleEl.contentEditable = 'false';
        }
    }, { once: true });
}

/**
 * 渲染侧栏会话项
 */
function renderSessionList() {
    if (!sessionListEl) return;

    // 对话搜索过滤
    let filteredSessions = sessions;
    if (sessionSearchQuery) {
        filteredSessions = sessions.filter(s => s.title.toLowerCase().includes(sessionSearchQuery));
    }

    if (filteredSessions.length === 0) {
        sessionListEl.innerHTML = '<div class="ai-session-empty">暂无会话</div>';
        return;
    }

    sessionListEl.innerHTML = '';
    filteredSessions.forEach(s => {
        const item = document.createElement('div');
        item.className = 'ai-session-item' + (s.id === activeSessionId ? ' active' : '');
        item.dataset.id = s.id;

        const title = document.createElement('span');
        title.className = 'ai-session-item-title';
        // 搜索关键词高亮
        if (sessionSearchQuery && s.title.toLowerCase().includes(sessionSearchQuery)) {
            const idx = s.title.toLowerCase().indexOf(sessionSearchQuery);
            const before = s.title.substring(0, idx);
            const match = s.title.substring(idx, idx + sessionSearchQuery.length);
            const after = s.title.substring(idx + sessionSearchQuery.length);
            title.innerHTML = before + '<mark class="ai-search-highlight">' + match + '</mark>' + after;
        } else {
            title.textContent = s.title;
        }
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
        title.addEventListener('dblclick', () => startInlineEdit(title, s.id));

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

        // 右键菜单
        item.addEventListener('contextmenu', showSessionContextMenu);

        sessionListEl.appendChild(item);
    });
}

/**
 * 切换会话
 */
async function switchSession(id) {
    if (isStreaming || id === activeSessionId) return;

    // 切换会话时清空笔记引用
    referencedNotes = [];
    cachedRefContext = '';
    updateRefChips();

    try {
        activeSessionId = id;
        const msgs = await window.go.main.App.LoadAISessionMessages(id);

        // 重建 chatHistory（只保留 role/content 供 API 使用）
        chatHistory = msgs ? msgs.map(msg => ({ role: msg.role, content: msg.content })) : [];

        // 清空消息列表
        messagesEl.innerHTML = '';

        if (!msgs || msgs.length === 0) {
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
    hideEmptyState();
    referencedNotes = [];
    cachedRefContext = '';
    updateRefChips();

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

    // 构建笔记引用上下文（后端已处理截断）
    let systemContext = '';
    if (referencedNotes.length > 0) {
        systemContext = await getNoteContext();
    }

    startStreaming(false, systemContext);
}

/**
 * 启动流式输出
 * @param {boolean} isRegenerate - 是否再生
 * @param {string} systemContext - 可选的 system prompt / 笔记上下文，拼入 messages 开头但不存库
 */
function startStreaming(isRegenerate = false, systemContext = '') {
    if (isStreaming) return;
    isStreaming = true;

    // 递增 generation，后续事件回调据此判断是否属于当前流
    _aiStreamGen++;
    const myGen = _aiStreamGen;

    // 清除该事件名下所有旧监听器，防止残留
    window.runtime.EventsOff('ai:stream-done', 'ai:stream-error', 'ai:stream-chunk', 'ai:stream-thinking');

    // 显示停止按钮，隐藏发送按钮
    if (stopBtnEl) stopBtnEl.style.display = '';
    if (sendBtnEl) sendBtnEl.style.display = 'none';

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

    const unsubThinking = window.runtime.EventsOn('ai:stream-thinking', (streamGen, chunk) => {
        if (streamGen !== myGen) return; // 属于旧流，丢弃
        if (!thinkingDetails) {
            thinkingDetails = document.createElement('details');
            thinkingDetails.className = 'thinking-details';
            thinkingDetails.open = localStorage.getItem('ai_cot_collapsed') !== 'true';
            thinkingDetails.addEventListener('toggle', () => {
                localStorage.setItem('ai_cot_collapsed', thinkingDetails.open ? 'false' : 'true');
            });
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

    const unsubChunk = window.runtime.EventsOn('ai:stream-chunk', (streamGen, chunk) => {
        if (streamGen !== myGen) return; // 属于旧流，丢弃
        if (!hasReceivedChunk) {
            hasReceivedChunk = true;
            contentDiv.innerHTML = '';
        }
        streamingContent += chunk;
        contentDiv.textContent = streamingContent;
        scrollToBottom();
    });
    unsubs.push(unsubChunk);

    const unsubDone = window.runtime.EventsOn('ai:stream-done', (streamGen, fullContent) => {
        if (streamGen !== myGen) return; // 属于旧流，丢弃
        unsubs.forEach(fn => fn());
        isStreaming = false;

        // 恢复发送按钮，隐藏停止按钮
        if (stopBtnEl) stopBtnEl.style.display = 'none';
        if (sendBtnEl) sendBtnEl.style.display = '';

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
        if (isRegenerate) {
            // 再生模式：user 消息已在 handleRegenerate 中重保存，只存 assistant
            saveSessionMessages([{ role: 'assistant', content: finalContent, reasoning_content: streamingThinking || '' }]);
        } else {
            saveSessionMessages([{ role: 'user', content: chatHistory[chatHistory.length - 2].content }, { role: 'assistant', content: finalContent, reasoning_content: streamingThinking || '' }]);
        }

        scrollToBottom();
    });
    unsubs.push(unsubDone);

    const unsubError = window.runtime.EventsOn('ai:stream-error', (streamGen, err) => {
        if (streamGen !== myGen) return; // 属于旧流，丢弃
        unsubs.forEach(fn => fn());
        isStreaming = false;
        // 恢复发送按钮，隐藏停止按钮
        if (stopBtnEl) stopBtnEl.style.display = 'none';
        if (sendBtnEl) sendBtnEl.style.display = '';
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();
        addErrorMessage(err);
    });
    unsubs.push(unsubError);

    // 构建发送给 API 的消息列表（system 上下文 + 历史对话）
    let messages = chatHistory;
    if (systemContext) {
        messages = [{ role: 'system', content: systemContext }, ...chatHistory];
    }

    try {
        window.go.main.App.CallAIStream(myGen, messages, enableThinking);
    } catch (e) {
        unsubs.forEach(fn => fn());
        isStreaming = false;
        // 恢复发送按钮，隐藏停止按钮
        if (stopBtnEl) stopBtnEl.style.display = 'none';
        if (sendBtnEl) sendBtnEl.style.display = '';
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
        details.open = localStorage.getItem('ai_cot_collapsed') !== 'true';
        details.addEventListener('toggle', () => {
            localStorage.setItem('ai_cot_collapsed', details.open ? 'false' : 'true');
        });
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
        loadModelSelector(cfg);
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

/**
 * 关闭 AI 会话右键菜单
 */
function closeSessionContextMenu() {
    if (sessionContextMenu) sessionContextMenu.classList.remove('active');
}

/**
 * 右键菜单显示/定位
 */
function showSessionContextMenu(e) {
    e.preventDefault();
    e.stopPropagation();
    if (!sessionContextMenu) return;

    // 定位到鼠标位置
    const x = e.clientX;
    const y = e.clientY;
    sessionContextMenu.style.left = x + 'px';
    sessionContextMenu.style.top = y + 'px';

    // 保存当前右键的会话 ID 和标题元素
    const item = e.currentTarget;
    _contextSessionId = parseInt(item.dataset.id);
    _contextTitleEl = item.querySelector('.ai-session-item-title');

    sessionContextMenu.classList.add('active');
}

function hideEmptyState() {
    if (!emptyEl) return;
    emptyEl.style.display = 'none';
    if (messagesEl) messagesEl.style.display = '';
    if (inputAreaEl) inputAreaEl.style.display = '';
    if (sessionNewBtnEl) sessionNewBtnEl.style.display = '';
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
        // 保存为笔记（仅 AI 回复）
        const saveBtn = document.createElement('button');
        saveBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>';
        saveBtn.title = '保存为笔记';
        saveBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            try {
                const note = await window.go.main.App.SaveAIMessageAsNote(content);
                if (note && note.id) {
                    window.showActionNotification?.('笔记已创建', 'success', [
                        { text: '查看', callback: async () => { window.switchView('grid'); await window.loadNotes(); window.openEditor(note.id, true); } }
                    ]);
                }
            } catch (e) {
                window.showNotification?.('保存失败: ' + (e.message || e), 'error');
            }
        });
        container.appendChild(saveBtn);

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

async function handleRegenerate(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const children = Array.from(messagesEl.children);
    const idx = children.indexOf(msgEl);
    if (idx === -1) return;

    chatHistory.splice(idx);
    children.slice(idx).forEach(el => el.remove());

    // 清空该会话的 DB 消息，重新保存截断后的 chatHistory，避免再生导致 user 消息重复
    try {
        await window.go.main.App.ClearAISessionMessages(activeSessionId);
        if (chatHistory.length > 0) {
            await window.go.main.App.SaveAIMessages(activeSessionId, chatHistory);
        }
    } catch (_) { /* 静默 */ }

    startStreaming(true);
}

/* ── 笔记引用 ═══════════════════════════════════════════════════ */

/** 缓存的引用上下文（后端已拼装好） */
let cachedRefContext = '';
const DOC_ICON = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>';
const CHECK_SVG = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';

/**
 * 打开笔记引用选择浮层
 */
async function openNoteRefModal() {
    if (!refModal) return;
    _refTempSelected = {};
    // 以已有引用笔记预填选中状态
    referencedNotes.forEach(n => { _refTempSelected[n.id] = true; });

    refModal.style.display = 'flex';

    // 重置搜索/筛选
    if (refSearch) refSearch.value = '';
    if (refSearchClear) refSearchClear.classList.remove('visible');
    if (refNotebook) refNotebook.value = '0';

    // 读取分页设置
    await loadRefPageSize();

    _refListLoaded = false;
    _refCurrentPage = 1;
    _refTotalItems = 0;
    _refLoading = false;
    // 骨架屏默认可见
    if (refSkeleton) refSkeleton.classList.remove('hidden');
    if (refLoadingOverlay) refLoadingOverlay.classList.remove('active');

    // 并行加载笔记本选项和笔记列表
    await Promise.all([
        loadAllNotebooks(),
        loadNoteList()
    ]);

    // 焦点到搜索框
    setTimeout(() => refSearch?.focus(), 150);
}

/**
 * 从设置中读取分页大小
 */
async function loadRefPageSize() {
    _refPageSize = 20;
    try {
        if (window.go?.main?.App?.GetPageSize) {
            const saved = await window.go.main.App.GetPageSize();
            if (saved && saved >= 10 && saved <= 100) {
                _refPageSize = saved;
            }
        }
    } catch (_) { /* 使用默认值 */ }
}

/**
 * 关闭笔记引用选择浮层
 */
function closeNoteRefModal() {
    if (!refModal) return;
    refModal.style.display = 'none';
    _refTempSelected = {};
}

/**
 * 加载所有笔记本到筛选下拉框
 */
async function loadAllNotebooks() {
    if (!refNotebook) return;
    try {
        const notebooks = await window.go.main.App.GetAllNotebooks() || [];
        const currentVal = refNotebook.value;
        refNotebook.innerHTML = '<option value="0">全部笔记本</option>' +
            notebooks.map(n => `<option value="${n.id}">${n.name}</option>`).join('');
        refNotebook.value = currentVal;
    } catch (_) {
        /* 静默 */
    }
}

/**
 * 加载笔记列表（根据当前搜索关键词和笔记本筛选）
 * 加载策略：
 *   - 首次加载：显示骨架屏，数据到达后替换
 *   - 二次加载：保留旧列表，显示半透明 overlay + 旋转环，数据到达后替换
 *   - 点击加载更多: 追加到列表末尾
 */
async function loadNoteList(append = false) {
    if (!refList || _refLoading) return;
    _refLoading = true;

    const query = refSearch?.value.trim() || '';
    const notebookId = parseInt(refNotebook?.value || '0');
    const page = append ? _refCurrentPage + 1 : 1;

    // 首次 → 骨架屏；二次刷新 → overlay；追加不显示 loading
    if (!append) {
        if (_refListLoaded && refLoadingOverlay) {
            refLoadingOverlay.classList.add('active');
        }
    }

    try {
        let result;
        if (query || notebookId > 0) {
            result = await window.go.main.App.SearchNotes(query, page, _refPageSize, notebookId, 'updated_at', '', '');
        } else {
            result = await window.go.main.App.GetNotes(page, _refPageSize, 'updated_at', 0);
        }
        const notes = result?.items || [];
        _refTotalItems = result?.total || 0;

        if (append) {
            _refCurrentPage = page;
            appendToList(notes);
        } else {
             // 隐藏骨架屏 / overlay
             if (refSkeleton) refSkeleton.classList.add('hidden');
             if (refLoadingOverlay) refLoadingOverlay.classList.remove('active');

             _refCurrentPage = 1;
             renderNoteList(notes);
             _refListLoaded = true;
         }
    } catch (e) {
        if (!append) {
            if (refSkeleton) refSkeleton.classList.add('hidden');
            if (refLoadingOverlay) refLoadingOverlay.classList.remove('active');
            if (!_refListLoaded) {
                refList.innerHTML = '<div class="ai-note-ref-empty">加载失败</div>';
            }
        }
    } finally {
        _refLoading = false;
    }
}

/**
 * 渲染笔记列表
 * @param {Array} notes - 笔记列表
 */
function renderNoteList(notes) {
    if (!refList) return;
    if (!notes || notes.length === 0) {
        refList.innerHTML = '<div class="ai-note-ref-empty">暂无匹配的笔记</div>';
        updateRefCount();
        return;
    }

    // 高亮搜索关键词
    const query = (refSearch?.value.trim() || '').toLowerCase();

    let html = notes.map((note, idx) => {
        const isSelected = !!_refTempSelected[note.id];
        let title = note.title || '无标题';
        if (query && title.toLowerCase().includes(query)) {
            const i = title.toLowerCase().indexOf(query);
            title = title.substring(0, i) + '<span class="highlight">' + title.substring(i, i + query.length) + '</span>' + title.substring(i + query.length);
        }
        const date = note.updated_at ? formatDate(note.updated_at) : '';
        return `<div class="ai-note-ref-item${isSelected ? ' selected' : ''}" data-id="${note.id}" style="--i:${idx}">
            <div class="ai-note-ref-item-check">${CHECK_SVG}</div>
            <div class="ai-note-ref-item-info">
                <div class="ai-note-ref-item-title">${title}</div>
                <div class="ai-note-ref-item-meta">
                    ${date ? `<span>🕐 ${date}</span>` : ''}
                </div>
            </div>
        </div>`;
    }).join('');

    // 底部加载更多按钮
    if (_refTotalItems > _refPageSize) {
        html += buildLoadMoreBtn();
    }

    refList.innerHTML = html;
    updateRefCount();
}

/**
 * 追加笔记到列表末尾（加载更多）
 * @param {Array} notes - 新加载的笔记列表
 */
function appendToList(notes) {
    if (!refList || !notes || notes.length === 0) {
        // 没有更多数据，移除加载更多按钮
        const loadMore = refList.querySelector('.ai-note-ref-load-more');
        if (loadMore) loadMore.remove();
        return;
    }

    // 移除已有的加载更多按钮
    const oldBtn = refList.querySelector('.ai-note-ref-load-more');
    if (oldBtn) oldBtn.remove();

    const query = (refSearch?.value.trim() || '').toLowerCase();
    const startIdx = (_refCurrentPage - 1) * _refPageSize; // items already shown

    const fragment = document.createElement('div');
    fragment.innerHTML = notes.map((note, idx) => {
        const isSelected = !!_refTempSelected[note.id];
        let title = note.title || '无标题';
        if (query && title.toLowerCase().includes(query)) {
            const i = title.toLowerCase().indexOf(query);
            title = title.substring(0, i) + '<span class="highlight">' + title.substring(i, i + query.length) + '</span>' + title.substring(i + query.length);
        }
        const date = note.updated_at ? formatDate(note.updated_at) : '';
        return `<div class="ai-note-ref-item${isSelected ? ' selected' : ''}" data-id="${note.id}" style="--i:${startIdx + idx}">
            <div class="ai-note-ref-item-check">${CHECK_SVG}</div>
            <div class="ai-note-ref-item-info">
                <div class="ai-note-ref-item-title">${title}</div>
                <div class="ai-note-ref-item-meta">
                    ${date ? `<span>🕐 ${date}</span>` : ''}
                </div>
            </div>
        </div>`;
    }).join('');

    // 追加新条目
    while (fragment.firstChild) {
        refList.appendChild(fragment.firstChild);
    }

    // 底部加载更多按钮（如果还有更多）
    if (_refCurrentPage * _refPageSize < _refTotalItems) {
        refList.insertAdjacentHTML('beforeend', buildLoadMoreBtn());
    }

    updateRefCount();
}

/**
 * 构建「加载更多」按钮的 HTML
 */
function buildLoadMoreBtn() {
    const remaining = _refTotalItems - (_refCurrentPage * _refPageSize);
    return `<div class="ai-note-ref-load-more">
        <span class="ai-note-ref-load-more-text">加载更多（剩余 ${remaining} 条）</span>
    </div>`;
}

/**
 * 切换笔记选中状态
 * @param {string|number} id - 笔记 ID
 */
function toggleNoteSelection(id) {
    if (!id) return;
    if (_refTempSelected[id]) {
        delete _refTempSelected[id];
    } else {
        _refTempSelected[id] = true;
    }
    // 仅更新选中态（不重新加载列表，保留滚动位置）
    const items = refList.querySelectorAll('.ai-note-ref-item');
    items.forEach(item => {
        if (item.dataset.id === id) {
            item.classList.toggle('selected');
        }
    });
    updateRefCount();
}

/**
 * 更新浮层底部的已选计数
 */
function updateRefCount() {
    if (!refCount) return;
    const count = Object.keys(_refTempSelected).length;
    refCount.textContent = `已选 ${count} 篇`;
    if (refConfirm) {
        refConfirm.disabled = count === 0;
        refConfirm.style.opacity = count === 0 ? '0.5' : '1';
    }
}

/**
 * 确认笔记选择，更新 chips
 */
async function confirmNoteSelection() {
    const selectedIds = Object.keys(_refTempSelected);
    if (selectedIds.length === 0) return;

    try {
        const ids = selectedIds.map(id => parseInt(id));
        // 后端一次性完成查询、截断、拼装
        const refContext = await window.go.main.App.GetNoteRefContext(ids);

        if (!refContext) {
            console.error('confirmNoteSelection: refContext is null');
            return;
        }

        // 合并：保留未取消的旧引用 + 新增的
        const newIds = new Set(ids);
        const keepNotes = referencedNotes.filter(n => !newIds.has(n.id));
        referencedNotes = [...keepNotes, ...refContext.notes];
        cachedRefContext = refContext.context;
    } catch (e) {
        console.error('confirmNoteSelection 失败:', e);
        return;
    }

    closeNoteRefModal();
    updateRefChips();
}

/**
 * 更新引用笔记 chips 显示
 */
function updateRefChips() {
    if (!refChips || !refBar || !refBtn) return;

    if (referencedNotes.length === 0) {
        refBar.style.display = 'none';
        refBtn.classList.remove('has-ref');
        return;
    }

    refBar.style.display = '';
    refBtn.classList.add('has-ref');
    refChips.innerHTML = referencedNotes.map(n => {
        const title = n.title || '无标题';
        const truncTip = n.truncated ? '<span class="ai-chat-ref-chip-trunc">(内容已截断)</span>' : '';
        return `<div class="ai-chat-ref-chip" data-id="${n.id}">
            <span class="ai-chat-ref-chip-icon">${DOC_ICON}</span>
            <span class="ai-chat-ref-chip-title" title="${title.replace(/"/g, '&quot;')}">${title}</span>
            ${truncTip}
            <button class="ai-chat-ref-chip-remove" data-id="${n.id}" title="移除引用">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>`;
    }).join('');

    // 绑定移除事件
    refChips.querySelectorAll('.ai-chat-ref-chip-remove').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            removeRefNote(btn.dataset.id);
        });
    });

    // chips 区域已包含后端返回的截断状态，无需异步刷新
}

/**
 * 移除单条引用笔记
 * @param {string|number} id - 笔记 ID
 */
function removeRefNote(id) {
    referencedNotes = referencedNotes.filter(n => String(n.id) !== String(id));
    cachedRefContext = ''; // 清除缓存
    updateRefChips();
}

/**
 * 获取笔记引用上下文（直接使用后端拼装好的结果）
 * @returns {Promise<string>} 拼装后的上下文内容，无引用时返回空字符串
 */
async function getNoteContext() {
    if (referencedNotes.length === 0) return '';

    if (cachedRefContext) return cachedRefContext;

    // 缓存不存在（如之前清除过），重新从后端获取
    const ids = referencedNotes.map(n => n.id);
    try {
        const refContext = await window.go.main.App.GetNoteRefContext(ids);
        referencedNotes = refContext.notes;
        cachedRefContext = refContext.context;
        updateRefChips();
        return refContext.context;
    } catch (_) {
        return '';
    }
}

/**
 * 格式化日期为简短显示
 * @param {string} dateStr - ISO 日期字符串
 * @returns {string}
 */
function formatDate(dateStr) {
    if (!dateStr) return '';
    try {
        const d = new Date(dateStr);
        const now = new Date();
        const diff = (now - d) / 1000;
        if (diff < 86400) return '今天';
        if (diff < 172800) return '昨天';
        return `${d.getMonth() + 1}/${d.getDate()}`;
    } catch (_) {
        return '';
    }
}
