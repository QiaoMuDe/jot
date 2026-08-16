/**
 * AI 对话模块 — 持久化多会话支持
 */
import hljs from 'highlight.js';
import { marked } from 'marked';

let messagesEl = null;        // #aiChatMessages
let messagesInnerEl = null;   // .ai-chat-messages-inner
let inputEl = null;           // #aiChatInput
let sendBtnEl = null;         // #aiChatSendBtn
let emptyEl = null;           // #aiChatEmpty
let welcomeEl = null;         // #aiChatWelcome
let inputAreaEl = null;       // #aiChatInputArea
let clearBtnEl = null;        // #aiChatClearBtn
let stopBtnEl = null;         // #aiChatStopBtn
let askPanelEl = null;        // #aiAskPanel（Agent 反问面板）
let sessionListEl = null;     // #aiSessionList
let sessionNewBtnEl = null;   // #aiSessionNewBtn
let contextSizeEl = null;     // #aiChatContextSize
let polishBtn = null;         // #aiChatPolishBtn
let polishOriginalText = '';  // 优化表达原文快照（用于还原）
let isPolishOptimizing = false; // 正在优化中，供停止按钮和 catch 块判断取消状态

let addBtn = null;            // #aiChatAddBtn
let addDropdown = null;       // #aiChatAddDropdown
let fileBar = null;           // #aiChatFileBar
let fileChips = null;         // #aiChatFileChips

// 状态
let chatHistory = []; // 仅渲染缓冲区，不再发送给后端 
let activeSessionId = null;    // null = 新会话尚未保存
let sessions = [];             // 侧栏会话列表
let sessionSearchQuery = '';
let sessionSearchEl = null;
let isStreaming = false;       // 正在流式输出时禁止切换/发送
// 窗口级标志，与 isStreaming 同步，供 main.js 全局拖拽系统读取
window.__aiStreaming = false;
let aiMsgContextMenu = null;   // AI 消息右键菜单
let _contextMsgContent = '';  // 右键消息内容
let _contextMsgRole = '';     // 右键消息角色
let _contextMsgEl = null;     // 右键消息元素
let _aiStreamGen = 0;          // 流式 generation 计数器, 跨流防串扰
let _loadingMore = false;      // 加载更多消息防重复
let _oldestMsgId = 0;          // 当前已加载的最旧消息 ID（用于分页）
let _scrollHandler = null;     // 滚动加载更多句柄，避免 switchSession 累积重复绑定

// 更多操作下拉菜单
let sessionMoreMenu = null;    // 更多操作下拉菜单元素
let sessionMoreMenuTarget = null; // 当前打开的会话项

// 模型选择器状态
let modelTrigger = null;
let modelDropdown = null;
let modelLabel = null;
let modelList = [];

// 深度思考状态
let searchToggle = null;
let enableThinking = false;

// 多源搜索状态
let searchSourcesBtn = null;
let searchSourcesDropdown = null;
let searchSources = new Set();

// 卡片召回状态
let cardRecallToggle = null;
let enableCardRecall = false;

// 笔记引用状态
let referencedNotes = [];       // { id, title, notebook_name }

// 角色扮演状态
let roleplayNotes = [];         // { id, title, notebook_name } — 角色档案笔记

// 追问引用
let followUpRef = '';           // 被追问的 AI 回复完整内容

// 上传文件列表
let uploadedFiles = [];  // 每项: { name, content, size, truncated }

// 拖拽上传
let _aiDragCounter = 0;
let aiChatContent = null;       // .ai-chat-content
let aiChatDropOverlay = null;   // #aiChatDropOverlay

// 笔记引用选择浮层 DOM
let refBar = null;              // #aiChatRefBar
let refChips = null;            // #aiChatRefChips
let refModal = null;            // #aiNoteRefModal
let refOverlay = null;          // #aiNoteRefOverlay
let refSearch = null;           // #aiNoteRefSearch
let refNotebookFilter = null;   // #aiNoteRefNotebookFilter
let refNotebookBtn = null;      // #aiNoteRefNotebookBtn
let refNotebookLabel = null;    // #aiNoteRefNotebookLabel
let refNotebookDropdown = null; // #aiNoteRefNotebookDropdown
let refList = null;             // #aiNoteRefList
let refCount = null;            // #aiNoteRefCount
let refConfirm = null;          // #aiNoteRefConfirm
let refCancel = null;           // #aiNoteRefCancel
let refClose = null;            // #aiNoteRefClose
let refSkeleton = null;         // #aiNoteRefSkeleton
let refLoadingOverlay = null;   // #aiNoteRefLoadingOverlay
let refSearchClear = null;      // #aiNoteRefSearchClear
let refListWrap = null;         // .ai-note-ref-list-wrap (滚动容器) 
let refTagBtn = null;           // #aiNoteRefTagBtn
let refTagLabel = null;         // #aiNoteRefTagLabel
let refTagDropdown = null;      // #aiNoteRefTagDropdown
let refTagFilter = null;        // #aiNoteRefTagFilter
let _refSearchTimer = null;     // 搜索 debounce 定时器
let _refTempSelected = {};      // 浮层中临时选中状态 { [id]: true }
let _refSelectAll = false;      // 全选模式标志
let _currentNotebookId = 0;    // 当前选中的笔记本 ID (0=全部)
let _refListLoaded = false;     // 是否已加载过列表 (首次用骨架屏, 后续用 overlay) 
let _refCurrentPage = 1;        // 当前页码
let _refTotalItems = 0;         // 匹配总数
let _refPageSize = 20;          // 每页条数 (从设置读取) 
let _refLoading = false;        // 是否正在加载中
let _refPendingRefresh = false; // 待刷新标志 (笔记本/搜索切换时若被 _refLoading 阻塞, 设此标志待重试) 
let _pageSizeLoaded = false;    // 分页大小是否已缓存
let _notebooksCache = null;     // 笔记本下拉选项缓存
let _refTagIds = new Set();     // 已选标签 ID 集合
let _tagsCache = null;          // 标签列表缓存

// 卡片召回笔记本选择状态
let recallNotebookIds = new Set();  // 选中的笔记本 ID 集合
let cardRecallDropdown = null;     // #aiChatRecallDropdown
let recallWrap = null;             // .ai-chat-recall-wrap
// 更多技能状态
let activeSkills = {};           // 当前激活的技能 { skillId: { config } }
let skillsBtn = null;            // #aiChatMoreSkillsBtn
let skillsDropdown = null;       // #aiChatSkillsDropdown
let skillBar = null;             // #aiChatSkillBar
let skillChips = null;           // #aiChatSkillChips

// 问答 / Agent 模式状态
let agentEnabled = false;        // 当前会话是否为 Agent 模式（后端 SessionConfig.agent_enabled）
let agentModeQaBtn = null;       // #aiChatModeQa
let agentModeAgentBtn = null;    // #aiChatModeAgent


// 优化表达提示词（输入框内嵌按钮专用，与下拉菜单的「文本润色」技能区分）
// 仅保留身份设定，优化指令嵌入到 user 消息中，避免模型将用户输入当作问题回答
const OPTIMIZE_EXPRESSION_PROMPT = `你是专业的文本表达优化师，负责优化用户提供的文本。`;

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
    // 聚焦搜索框（仅在可见时）
    const search = modelDropdown.querySelector('.ai-model-search');
    if (search && search.offsetParent !== null) setTimeout(() => search.focus(), 50);
}

function renderModelDropdown() {
    if (!modelDropdown || !modelLabel) return;
    const current = modelLabel.textContent;
    // 仅移除现有列表项，保留搜索框等非列表元素
    modelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
    // 用 DOM 方式追加列表项
    const searchWrap = modelDropdown.querySelector('.ai-model-search-wrap');
    modelList.forEach(m => {
        const div = document.createElement('div');
        div.className = 'theme-select-item' + (m === current ? ' active' : '');
        div.dataset.model = m;
        div.textContent = m;
        if (searchWrap) {
            searchWrap.after(div);
        } else {
            modelDropdown.appendChild(div);
        }
    });
    // 仅当模型 ≥2 个时显示搜索框
    if (searchWrap) {
        searchWrap.style.display = modelList.length > 1 ? '' : 'none';
    }
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
        await saveCurrentSessionConfig();
    } catch (_) {}
}

/**
 * 初始化 AI 对话页面
 */
export async function initAIChat() {
    messagesEl = document.getElementById('aiChatMessages');
    messagesInnerEl = messagesEl.querySelector('.ai-chat-messages-inner');
    inputEl = document.getElementById('aiChatInput');
    sendBtnEl = document.getElementById('aiChatSendBtn');
    emptyEl = document.getElementById('aiChatEmpty');
    welcomeEl = document.getElementById('aiChatWelcome');
    inputAreaEl = document.getElementById('aiChatInputArea');
    clearBtnEl = document.getElementById('aiChatClearBtn');
    stopBtnEl = document.getElementById('aiChatStopBtn');
    askPanelEl = document.getElementById('aiAskPanel');
    sessionListEl = document.getElementById('aiSessionList');
    sessionNewBtnEl = document.getElementById('aiSessionNewBtn');
    sessionSearchEl = document.getElementById('aiSessionSearch');
    contextSizeEl = document.getElementById('aiChatContextSize');
    polishBtn = document.getElementById('aiChatPolishBtn');

    // 模型选择器
    modelTrigger = document.getElementById('aiChatModelTrigger');
    modelDropdown = document.getElementById('aiChatModelDropdown');
    modelLabel = document.getElementById('aiChatModelLabel');
    modelList = [];

    // 问答 / Agent 模式切换
    agentModeQaBtn = document.getElementById('aiChatModeQa');
    agentModeAgentBtn = document.getElementById('aiChatModeAgent');
    agentEnabled = false;

    // 深度思考
    searchToggle = document.getElementById('aiChatSearchToggle');
    enableThinking = searchToggle?.classList.contains('active') || false;

    // 多源搜索
    searchSourcesBtn = document.getElementById('aiChatSearchSourcesBtn');
    searchSourcesDropdown = document.getElementById('aiChatSearchSourcesDropdown');

    // 卡片召回
    cardRecallToggle = document.getElementById('aiChatCardRecallToggle');
    enableCardRecall = cardRecallToggle?.classList.contains('active') || false;
    cardRecallDropdown = document.getElementById('aiChatRecallDropdown');
    recallWrap = document.querySelector('.ai-chat-recall-wrap');
    recallNotebookIds = new Set();

    // 笔记引用
    refBar = document.getElementById('aiChatRefBar');
    refChips = document.getElementById('aiChatRefChips');
    refModal = document.getElementById('aiNoteRefModal');
    refOverlay = document.getElementById('aiNoteRefOverlay');
    refSearch = document.getElementById('aiNoteRefSearch');
    refNotebookFilter = document.getElementById('aiNoteRefNotebookFilter');
    refNotebookBtn = document.getElementById('aiNoteRefNotebookBtn');
    refNotebookLabel = document.getElementById('aiNoteRefNotebookLabel');
    refNotebookDropdown = document.getElementById('aiNoteRefNotebookDropdown');
    refList = document.getElementById('aiNoteRefList');
    refListWrap = document.querySelector('.ai-note-ref-list-wrap');
    refCount = document.getElementById('aiNoteRefCount');
    refConfirm = document.getElementById('aiNoteRefConfirm');
    refCancel = document.getElementById('aiNoteRefCancel');
    refClose = document.getElementById('aiNoteRefClose');
    refSkeleton = document.getElementById('aiNoteRefSkeleton');
    refLoadingOverlay = document.getElementById('aiNoteRefLoadingOverlay');
    refSearchClear = document.getElementById('aiNoteRefSearchClear');
    refTagBtn = document.getElementById('aiNoteRefTagBtn');
    refTagLabel = document.getElementById('aiNoteRefTagLabel');
    refTagDropdown = document.getElementById('aiNoteRefTagDropdown');
    refTagFilter = document.getElementById('aiNoteRefTagFilter');
    _refListLoaded = false;

    // 更多技能
    skillsBtn = document.getElementById('aiChatMoreSkillsBtn');
    skillsDropdown = document.getElementById('aiChatSkillsDropdown');
    skillBar = document.getElementById('aiChatSkillBar');
    skillChips = document.getElementById('aiChatSkillChips');

    // 角色档案选择器

    // 添加菜单
    addBtn = document.getElementById('aiChatAddBtn');
    addDropdown = document.getElementById('aiChatAddDropdown');
    fileBar = document.getElementById('aiChatFileBar');
    fileChips = document.getElementById('aiChatFileChips');

    // 拖拽遮罩
    aiChatContent = document.querySelector('.ai-chat-content');
    aiChatDropOverlay = document.getElementById('aiChatDropOverlay');

    if (!messagesEl) return;

    aiMsgContextMenu = document.getElementById('aiMsgContextMenu');

    // 更多操作下拉菜单 - 动态创建
    sessionMoreMenu = document.createElement('div');
    sessionMoreMenu.className = 'ai-session-more-menu';
    document.body.appendChild(sessionMoreMenu);

    bindEvents();

    // 一次性初始化 Marked 选项 (高亮在 renderMarkdown 中用 hljs.highlightElement 后处理) 
    marked.setOptions({
        breaks: true,
        gfm: true
    });

    // 初始化拖拽上传
    initAiChatFileDrop();
}

/* ── 上下文大小 ── */

/** 格式化 token 数为可读字符串 */
function formatTokens(count) {
    if (count >= 1000) return (count / 1000).toFixed(1) + 'K';
    return String(count);
}

/** 更新上下文大小指示器（从后端读取会话缓存 tokens） */
async function updateContextSize() {
    if (!contextSizeEl) return;
    if (!activeSessionId) {
        contextSizeEl.textContent = '';
        contextSizeEl.style.display = 'none';
        return;
    }
    try {
        const tokens = await window.go.main.App.GetSessionContextTokens(activeSessionId);
        if (tokens > 0) {
            contextSizeEl.textContent = formatTokens(tokens) + ' tokens';
            contextSizeEl.style.display = '';
        } else {
            contextSizeEl.textContent = '';
            contextSizeEl.style.display = 'none';
        }
    } catch (_) {
        contextSizeEl.textContent = '';
        contextSizeEl.style.display = 'none';
    }
}

/**
 * 绑定所有事件
 */
function bindEvents() {
    // 清空当前对话
    if (clearBtnEl) {
        clearBtnEl.addEventListener('click', async () => {
            if (activeSessionId === null || messagesInnerEl.children.length === 0) {
                window.showNotification?.('当前没有对话可以清空', 'info');
                return;
            }
            const confirmed = await window.showConfirmDialog('确定清空当前对话吗？');
            if (!confirmed) return;

            // 流式进行中（含 Agent 反问等待）：先取消当前 run，避免清空后悬挂
            if (isStreaming) {
                if (stopBtnEl) stopBtnEl.click();
                else await window.go.main.App.CancelAIStream();
            }

            try {
                await window.go.main.App.ClearAISessionMessages(activeSessionId);
            } catch (_) { /* 静态失败 */ }

            messagesInnerEl.innerHTML = '';
            chatHistory = [];
            hideAskPanel(); // 清空会话时收起 Agent 反问面板
            updateContextSize();
            scrollToBottom();
        });
    }

    // 新建会话
    if (sessionNewBtnEl) {
        sessionNewBtnEl.addEventListener('click', () => {
            triggerPulseFeedback(sessionNewBtnEl);
            createSession();
        });
    }

    // 双击标题栏标题重命名当前会话（内联编辑，同侧栏条目重命名）
    const aiChatTitleEl = document.getElementById('aiChatTitle');
    if (aiChatTitleEl) {
        aiChatTitleEl.addEventListener('dblclick', () => {
            if (activeSessionId === null) return;
            triggerPulseFeedback(aiChatTitleEl);
            startInlineEdit(aiChatTitleEl, activeSessionId);
        });
    }

    // 前往设置
    const goSettingsBtn = document.getElementById('aiChatGoSettings');
    if (goSettingsBtn) {
        goSettingsBtn.addEventListener('click', () => window.switchView('settings'));
    }

    // 输入框事件
    if (inputEl) {
        inputEl.addEventListener('input', () => {
            const val = inputEl.value.trim();
            sendBtnEl.disabled = val.length === 0;
            if (polishBtn) {
                if (val.length > 0 && !isStreaming) {
                    polishBtn.disabled = false;
                } else {
                    polishBtn.disabled = true;
                    polishBtn.classList.remove('is-revert');
                    const txt = polishBtn.querySelector('.ai-chat-polish-text');
                    if (txt) txt.textContent = '优化';
                }
            }
        });
        inputEl.addEventListener('keydown', onInputKeydown);
        inputEl.addEventListener('input', autoResizeInput);
    }

    // 发送按钮
    if (sendBtnEl) {
        sendBtnEl.addEventListener('click', onSend);
    }

    // 停止生成 / 取消优化
    if (stopBtnEl) {
        stopBtnEl.addEventListener('click', async () => {
            // 优化中取消分支
            if (isPolishOptimizing) {
                isPolishOptimizing = false;
                stopBtnEl.style.display = 'none';
                if (sendBtnEl) sendBtnEl.style.display = '';
                // 恢复输入框内容为原文
                inputEl.value = polishOriginalText;
                inputEl.style.height = 'auto';
                sendBtnEl.disabled = false;
                // 恢复优化按钮状态
                if (polishBtn) {
                    const wrap = polishBtn.closest('.ai-chat-input-wrap');
                    if (wrap) wrap.classList.remove('is-loading');
                    polishBtn.classList.remove('is-loading');
                    polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
                    polishBtn.disabled = false;
                }
                polishOriginalText = '';
                try {
                    await window.go.main.App.CancelAIStream();
                } catch (_) {}
                return;
            }

            // 流式生成取消逻辑
            stopBtnEl.style.display = 'none';
            if (sendBtnEl) sendBtnEl.style.display = '';
            isStreaming = false;
            window.__aiStreaming = false;
            hideAskPanel();
            // 立即移除当前 streaming 气泡（无论处于搜索还是 LLM 阶段）
            const streamingBubble = messagesInnerEl.querySelector('.ai-msg-assistant:last-child');
            if (streamingBubble) {
                streamingBubble.remove();
            }
            // 停止后恢复润色按钮状态
            if (polishBtn) {
                polishBtn.disabled = !(inputEl && inputEl.value.trim().length > 0);
            }
            try {
                await window.go.main.App.CancelAIStream();
            } catch (_) {}
        });
    }

    // 优化表达按钮
    if (polishBtn) {
        polishBtn.addEventListener('click', async () => {
            const text = inputEl.value.trim();
            if (!text || isStreaming || isPolishOptimizing) return;

            // 校验 AI 配置完整性与模型选择
            if (!(await ensureAIReady('优化表达'))) return;

            // 还原模式：恢复原文
            if (polishBtn.classList.contains('is-revert')) {
                inputEl.value = polishOriginalText;
                inputEl.style.height = 'auto';
                sendBtnEl.disabled = false;
                polishBtn.classList.remove('is-revert');
                polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
                return;
            }

            // 保存原文快照
            polishOriginalText = text;

            // 进入加载态：遮罩禁用输入 + 按钮旋转动画
            const wrap = polishBtn.closest('.ai-chat-input-wrap');
            if (wrap) wrap.classList.add('is-loading');
            polishBtn.classList.add('is-loading');
            polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化中';
            inputEl.blur();

            // 禁用发送按钮，显示停止按钮以支持取消
            sendBtnEl.disabled = true;
            sendBtnEl.style.display = 'none';
            stopBtnEl.style.display = '';
            isPolishOptimizing = true;

            try {
                const result = await window.go.main.App.CallAI([
                    { role: 'system', content: OPTIMIZE_EXPRESSION_PROMPT },
                    { role: 'user', content: '请优化以下文本，只输出优化后的结果，不要回答任何问题。\n\n【规则】\n1. 100%保留用户的核心意思和观点，绝不改变原意\n2. 理顺逻辑结构，去掉冗余口语、重复表述，让表达更凝练\n3. 保持自然的中文表达习惯，不生硬、不堆砌辞藻\n4. 根据内容自动适配语气：日常交流保持平实，正式内容保持严谨\n\n【输出要求】\n- 只输出优化后的文本，不添加任何额外解释、说明、开头语或结尾语\n- 绝对不要对用户输入的内容进行回答、评论或补充\n\n以下是需要优化的文本：\n\n"""\n' + text + '\n"""' }
                ]);
                if (result) {
                    // 恢复按钮显示
                    if (isPolishOptimizing) {
                        isPolishOptimizing = false;
                        stopBtnEl.style.display = 'none';
                        sendBtnEl.style.display = '';
                    }
                    // 清除加载态
                    if (wrap) wrap.classList.remove('is-loading');
                    polishBtn.classList.remove('is-loading');
                    polishBtn.querySelector('.ai-chat-polish-text').textContent = '还原';
                    polishBtn.classList.add('is-revert');
                    polishBtn.disabled = true;

                    // 打字机效果逐字输出
                    if (wrap) wrap.classList.add('is-typing');
                    inputEl.value = '';
                    inputEl.style.height = 'auto';
                    await typewriterText(inputEl, result, 12);
                    if (wrap) wrap.classList.remove('is-typing');

                    polishBtn.disabled = false;
                } else {
                    // 结果为空，恢复
                    if (isPolishOptimizing) {
                        isPolishOptimizing = false;
                        stopBtnEl.style.display = 'none';
                        sendBtnEl.style.display = '';
                        sendBtnEl.disabled = false;
                    }
                    if (wrap) wrap.classList.remove('is-loading');
                    polishBtn.classList.remove('is-loading');
                    polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
                    polishBtn.disabled = false;
                }
            } catch (e) {
                // 如果已被停止按钮取消处理，不再重复恢复 UI 和报错
                if (!isPolishOptimizing) return;
                isPolishOptimizing = false;
                // 恢复按钮显示
                stopBtnEl.style.display = 'none';
                sendBtnEl.style.display = '';
                sendBtnEl.disabled = false;
                // 恢复输入框内容为原文
                inputEl.value = polishOriginalText;
                inputEl.style.height = 'auto';
                polishOriginalText = '';
                // 恢复优化按钮状态
                if (wrap) wrap.classList.remove('is-loading');
                polishBtn.classList.remove('is-loading');
                polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
                polishBtn.disabled = false;
                // 尝试解析结构化错误
                try {
                    const errData = JSON.parse(e.message || e);
                    if (errData.user_msg) {
                        window.showNotification(errData.user_msg, 'error', 5000);
                    } else {
                        window.showNotification('AI 调用失败: ' + (e.message || e), 'error', 5000);
                    }
                } catch (_) {
                    window.showNotification('AI 调用失败: ' + (e.message || e), 'error', 5000);
                }
            }
        });
    }

    /**
     * 打字机效果 - 逐字填入 textarea
     * @param {HTMLTextAreaElement} el
     * @param {string} text
     * @param {number} msPerChar - 每字间隔(ms)
     */
    function typewriterText(el, text, msPerChar = 12) {
        return new Promise(resolve => {
            let idx = 0;
            const len = text.length;
            let finished = false;
            function tick() {
                if (finished) return;
                if (idx >= len) {
                    finished = true;
                    el.dispatchEvent(new Event('input', { bubbles: true }));
                    resolve();
                    return;
                }
                // 每次写入 1~3 个字符，看起来更自然
                const step = Math.min(len - idx, 1 + Math.floor(Math.random() * 2));
                idx += step;
                el.value = text.slice(0, idx);
                el.style.height = 'auto';
                requestAnimationFrame(() => {
                    setTimeout(tick, msPerChar + Math.random() * 6);
                });
            }
            tick();
            // 兜底：最长 6 秒强制完成
            setTimeout(() => {
                if (!finished) {
                    finished = true;
                    el.value = text;
                    el.style.height = 'auto';
                    el.dispatchEvent(new Event('input', { bubbles: true }));
                    resolve();
                }
            }, 6000);
        });
    }

    // 侧栏折叠/展开
    const toggleBtn = document.getElementById('aiSidebarToggle');
    const sidebar = document.querySelector('.ai-session-sidebar');
    // 侧栏图标：面板图标（不带方向箭头，折叠/展开共用）
    const panelOpenIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/></svg>';
    const panelCloseIcon = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/></svg>';

    if (toggleBtn && sidebar) {
        // 恢复保存的状态 (默认展开) 
        const saved = localStorage.getItem('ai_sidebar_collapsed');
        if (saved === 'false') {
            sidebar.classList.add('collapsed');
            toggleBtn.innerHTML = panelOpenIcon;
            toggleBtn.title = '展开侧栏';
        } else {
            sidebar.classList.remove('collapsed');
            toggleBtn.innerHTML = panelCloseIcon;
            toggleBtn.title = '折叠侧栏';
        }
    }

    // 导出为全局函数，供 Ctrl+J 快捷键使用
    window.toggleAISessionSidebar = function() {
        if (!toggleBtn || !sidebar) return;
        const wasCollapsed = sidebar.classList.contains('collapsed');
        const isCollapsed = sidebar.classList.toggle('collapsed');
        toggleBtn.innerHTML = isCollapsed ? panelOpenIcon : panelCloseIcon;
        toggleBtn.title = isCollapsed ? '展开侧栏' : '折叠侧栏';
        localStorage.setItem('ai_sidebar_collapsed', String(!isCollapsed));
        // 从折叠变为展开时，刷新会话列表
        if (wasCollapsed && !isCollapsed) {
            loadSessionList();
        }
    };

    if (toggleBtn) {
        toggleBtn.addEventListener('click', window.toggleAISessionSidebar);
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
                clearModelSearch();
            } else {
                openModelDropdown();
            }
        });

        modelDropdown.addEventListener('click', (e) => {
            const item = e.target.closest('.theme-select-item');
            if (item) switchModel(item.dataset.model);
        });

        document.addEventListener('click', () => {
            modelDropdown.classList.remove('open');
            clearModelSearch();
        });

        // 模型搜索过滤 + 关键字高亮
        const modelSearch = document.getElementById('aiChatModelSearch');
        if (modelSearch) {
            modelSearch.addEventListener('input', () => {
                const query = modelSearch.value.trim();
                modelDropdown.querySelectorAll('.theme-select-item').forEach(item => {
                    const model = item.dataset.model;
                    if (!query) {
                        // 清空搜索 → 恢复 textContent，全部显示
                        item.textContent = model;
                        item.style.display = '';
                        return;
                    }
                    const lowerModel = model.toLowerCase();
                    const lowerQuery = query.toLowerCase();
                    const idx = lowerModel.indexOf(lowerQuery);
                    if (idx !== -1) {
                        // 匹配 → 高亮 + 显示
                        const before = model.substring(0, idx);
                        const match = model.substring(idx, idx + query.length);
                        const after = model.substring(idx + query.length);
                        item.innerHTML = before + '<mark class="ai-search-highlight">' + match + '</mark>' + after;
                        item.style.display = '';
                    } else {
                        // 不匹配 → 恢复 textContent + 隐藏
                        item.textContent = model;
                        item.style.display = 'none';
                    }
                });
            });
            modelSearch.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    modelDropdown.classList.remove('open');
                    clearModelSearch();
                }
                if (e.key === 'Enter') e.preventDefault();
            });
        }

        // 监听预设切换事件，重置模型缓存
        document.addEventListener('profile-switched', () => {
            modelList = [];
            if (modelLabel) modelLabel.textContent = '--';
            // 清空已有列表项
            modelDropdown?.querySelectorAll('.theme-select-item').forEach(el => el.remove());
        });
    }

    function clearModelSearch() {
        const search = modelDropdown?.querySelector('.ai-model-search');
        if (search) {
            search.value = '';
            search.dispatchEvent(new Event('input'));
        }
    }

    // ── 问答 / Agent 模式切换 ──
    if (agentModeQaBtn) {
        agentModeQaBtn.addEventListener('click', () => switchAgentMode(false));
    }
    if (agentModeAgentBtn) {
        agentModeAgentBtn.addEventListener('click', () => switchAgentMode(true));
    }

    // ── 深度思考切换 ──
    if (searchToggle) {
        if (enableThinking) searchToggle.classList.add('active');
        searchToggle.addEventListener('click', async () => {
            enableThinking = searchToggle.classList.toggle('active');
            await saveCurrentSessionConfig();
        });
    }

    // ── 多源搜索按钮：左 2/3=批量开关，右 1/3=搜索源菜单 ──
    const sourceItemIds = ['aiChatZhihuSearch', 'aiChatZhihuGlobalSearch', 'aiChatTavilySearch'];
    if (searchSourcesBtn && searchSourcesDropdown) {
        const sourcesWrap = document.querySelector('.ai-chat-sources-wrap');
        // 计算右侧菜单区宽度（至少 30px，按钮较窄时保证可点）
        const menuZoneWidth = (el) => Math.max(30, el.getBoundingClientRect().width * 0.33);
        const inMenuZone = (el, clientX) => {
            const rect = el.getBoundingClientRect();
            return (clientX - rect.left) >= rect.width - menuZoneWidth(el);
        };

        searchSourcesBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            // 右侧 1/3：展开/收起搜索源菜单
            if (inMenuZone(searchSourcesBtn, e.clientX)) {
                const open = searchSourcesDropdown.classList.toggle('open');
                if (sourcesWrap) sourcesWrap.classList.toggle('menu-open', open);
                return;
            }
            // 左侧 2/3：批量开启/关闭全部搜索源
            const allChecked = sourceItemIds.every(id => {
                const cb = document.getElementById(id);
                return cb && cb.checked;
            });
            const newState = !allChecked;
            sourceItemIds.forEach(id => {
                const cb = document.getElementById(id);
                if (cb && !cb.disabled) {
                    cb.checked = newState;
                    const source = cb.dataset.source;
                    if (newState) {
                        searchSources.add(source);
                    } else {
                        searchSources.delete(source);
                    }
                }
            });
            if (searchSourcesBtn) {
                searchSourcesBtn.classList.toggle('active', newState);
            }
            // 开关时顺手收起菜单
            searchSourcesDropdown.classList.remove('open');
            if (sourcesWrap) sourcesWrap.classList.remove('menu-open');
            await saveCurrentSessionConfig();
        });

        // 悬停右侧菜单区时箭头高亮提示（离开按钮时移除）
        searchSourcesBtn.addEventListener('mousemove', (e) => {
            searchSourcesBtn.classList.toggle('arrow-hover', inMenuZone(searchSourcesBtn, e.clientX));
        });
        searchSourcesBtn.addEventListener('mouseleave', () => {
            searchSourcesBtn.classList.remove('arrow-hover');
        });

        // 点击外部关闭下拉菜单
        document.addEventListener('click', () => {
            if (searchSourcesDropdown) {
                searchSourcesDropdown.classList.remove('open');
                if (sourcesWrap) sourcesWrap.classList.remove('menu-open');
            }
        });

        if (searchSourcesDropdown) {
            searchSourcesDropdown.addEventListener('click', (e) => {
                e.stopPropagation();
            });
        }
    }
    
    // ── 搜索源菜单项点击切换 ──
    // 点击整个菜单项区域（不限于复选框）都能切换选中状态
    const sourceItems = searchSourcesDropdown?.querySelectorAll('.ai-chat-search-source-item');
    if (sourceItems) {
        sourceItems.forEach(item => {
            const checkbox = item.querySelector('input[type="checkbox"]');
            if (!checkbox) return;
            item.addEventListener('click', (e) => {
                if (e.target.tagName === 'INPUT') return; // 让 checkbox 自身的 change 事件处理
                checkbox.checked = !checkbox.checked;
                checkbox.dispatchEvent(new Event('change'));
            });
        });
    }
    // 复选框 change 事件（处理复选框自身点击和程序化触发）
    sourceItemIds.forEach(id => {
        const checkbox = document.getElementById(id);
        if (checkbox) {
            checkbox.addEventListener('change', async () => {
                if (checkbox.disabled) return;
                const source = checkbox.dataset.source;
                if (checkbox.checked) {
                    searchSources.add(source);
                } else {
                    searchSources.delete(source);
                }
                // 更新按钮激活状态
                if (searchSourcesBtn) {
                    searchSourcesBtn.classList.toggle('active', searchSources.size > 0);
                }
                await saveCurrentSessionConfig();
            });
        }
    });

    // ── 卡片召回按钮：左 2/3=批量开关，右 1/3=笔记本菜单 ──
    if (cardRecallToggle) {
        if (enableCardRecall) cardRecallToggle.classList.add('active');
        // 计算右侧菜单区宽度（至少 30px，按钮较窄时保证可点）
        const recallMenuZone = () => Math.max(30, cardRecallToggle.getBoundingClientRect().width * 0.33);
        const recallInMenuZone = (clientX) => {
            const rect = cardRecallToggle.getBoundingClientRect();
            return (clientX - rect.left) >= rect.width - recallMenuZone();
        };

        cardRecallToggle.addEventListener('click', async (e) => {
            e.stopPropagation();
            // 右侧 1/3：展开/收起笔记本菜单
            if (recallInMenuZone(e.clientX)) {
                const dropdown = cardRecallDropdown;
                if (!dropdown) return;
                const isCurrentlyOpen = dropdown.classList.contains('open');
                if (isCurrentlyOpen) {
                    // 关闭菜单
                    dropdown.classList.remove('open');
                    if (recallWrap) recallWrap.classList.remove('menu-open');
                } else {
                    // 打开菜单：先构建子项（隐藏态），再添加 .open 触发入场动画
                    await loadRecallNotebookMenu();
                    dropdown.classList.add('open');
                    if (recallWrap) recallWrap.classList.add('menu-open');
                    dropdown.scrollTop = 0;
                    // 点击外部关闭菜单
                    const closeHandler = (ev) => {
                        if (!recallWrap || recallWrap.contains(ev.target)) return;
                        dropdown.classList.remove('open');
                        if (recallWrap) recallWrap.classList.remove('menu-open');
                        document.removeEventListener('click', closeHandler);
                    };
                    setTimeout(() => document.addEventListener('click', closeHandler), 10);
                }
                return;
            }
            // 左侧 2/3：批量开启/关闭卡片召回（开→全选所有笔记本，关→全取消）
            enableCardRecall = cardRecallToggle.classList.toggle('active');
            if (enableCardRecall) {
                // 开启前校验量化配置与量化表
                try {
                    const check = await window.go.main.App.ValidateCardRecall();
                    if (!check.ok) {
                        // 校验失败：回滚开关状态
                        enableCardRecall = false;
                        cardRecallToggle.classList.toggle('active');
                        nm.show(check.message || '卡片召回校验未通过', 'warning');
                        return;
                    }
                } catch (e) {
                    enableCardRecall = false;
                    cardRecallToggle.classList.toggle('active');
                    nm.show('卡片召回校验失败: ' + e, 'error');
                    return;
                }
                // 开→全选所有笔记本（加载菜单后勾选全部）
                try {
                    const notebooks = await window.go.main.App.GetAllNotebooks();
                    if (notebooks) {
                        notebooks.forEach(nb => recallNotebookIds.add(nb.id));
                    }
                } catch (_) {}
                // 更新菜单复选框状态
                if (cardRecallDropdown) {
                    cardRecallDropdown.querySelectorAll('.ai-chat-recall-item input[type="checkbox"]').forEach(cb => {
                        cb.checked = true;
                    });
                }
            } else {
                // 关→全取消
                recallNotebookIds.clear();
                if (cardRecallDropdown) {
                    cardRecallDropdown.querySelectorAll('.ai-chat-recall-item input[type="checkbox"]').forEach(cb => {
                        cb.checked = false;
                    });
                }
            }
            // 开关时顺手收起菜单
            if (cardRecallDropdown) cardRecallDropdown.classList.remove('open');
            if (recallWrap) recallWrap.classList.remove('menu-open');
            await saveCurrentSessionConfig();
        });

        // 悬停右侧菜单区时箭头高亮提示（离开按钮时移除）
        cardRecallToggle.addEventListener('mousemove', (e) => {
            cardRecallToggle.classList.toggle('arrow-hover', recallInMenuZone(e.clientX));
        });
        cardRecallToggle.addEventListener('mouseleave', () => {
            cardRecallToggle.classList.remove('arrow-hover');
        });
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

    // 全选按钮
    const refSelectAll = document.getElementById('aiNoteRefSelectAll');
    if (refSelectAll) {
        refSelectAll.addEventListener('click', toggleRefSelectAll);
    }

    // ── 追问引用栏关闭按钮 ──
    const followUpClose = document.getElementById('aiChatFollowUpClose');
    if (followUpClose) {
        followUpClose.addEventListener('click', () => {
            followUpRef = '';
            const bar = document.getElementById('aiChatFollowUpBar');
            if (bar) bar.style.display = 'none';
        });
    }
    if (refSearch) {
        refSearch.addEventListener('input', () => {
            clearTimeout(_refSearchTimer);
            _refSearchTimer = setTimeout(loadNoteList, 200);
        });
        // Enter 键触发搜索
        refSearch.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                clearTimeout(_refSearchTimer);
                loadNoteList();
            }
        });
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

    // 引用笔记选择器 Enter 键确认（capture 阶段拦截，不受焦点元素限制，且阻止搜索框 Enter 搜索干扰）
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && refModal && refModal.style.display !== 'none') {
            const count = Object.keys(_refTempSelected).length;
            if (count > 0) {
                e.preventDefault();
                e.stopPropagation();
                if (refConfirm) refConfirm.click();
            }
        }
    }, { capture: true });

    // 浮层列表点击切换选中 + 滚动到底部自动加载更多
    if (refList) {
        refList.addEventListener('click', (e) => {
            const item = e.target.closest('.ai-note-ref-item');
            if (item) toggleNoteSelection(item.dataset.id);
        });
    }
    if (refListWrap) {
        refListWrap.addEventListener('scroll', () => {
            const el = refListWrap;
            if (el.scrollTop + el.clientHeight >= el.scrollHeight - 400) {
                if (!_refLoading && _refCurrentPage * _refPageSize < _refTotalItems) {
                    loadNoteList(true);
                }
            }
        });
    }

    // ── 更多技能 ──
    if (skillsBtn && skillsDropdown) {
        skillsBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (skillsDropdown.classList.contains('open')) {
                closeSkillsDropdown();
            } else {
                updateSkillsMenuActiveState();
                skillsDropdown.classList.add('open');
                skillsDropdown.scrollTop = 0;
            }
        });

        // 点击技能菜单项 (toggle 模式: 已激活则取消, 未激活则激活)
        skillsDropdown.addEventListener('click', (e) => {
            const item = e.target.closest('.ai-chat-skills-item');
            if (item) {
                const skill = item.dataset.skill;

                if (activeSkills[skill]) {
                    // toggle off: 取消激活
                    delete activeSkills[skill];
                } else {
                    // toggle on: 激活（互斥，先清空其他技能）
                    activeSkills = {};
                    if (skill === 'translate') {
                        activeSkills.translate = { source: 'english', target: 'chinese' };
                    } else {
                        activeSkills[skill] = true;
                    }
                }

                renderSkillChips();
                updateSkillsMenuActiveState();
                saveCurrentSessionConfig();

                // 点击任意菜单项后关闭菜单
                closeSkillsDropdown();
                return;
            }
        });
    }

    // 点击外部关闭技能菜单 / 添加菜单
    document.addEventListener('click', (e) => {
        if (skillsDropdown && skillsBtn && !skillsBtn.contains(e.target) && !skillsDropdown.contains(e.target)) {
            closeSkillsDropdown();
        }
        if (addDropdown && addBtn && !addBtn.contains(e.target) && !addDropdown.contains(e.target)) {
            addDropdown.classList.remove('open');
        }
    });

    // 点击菜单外区域关闭右键菜单
    document.addEventListener('click', (e) => {
        if (aiMsgContextMenu && !aiMsgContextMenu.contains(e.target)) {
            closeAiMsgContextMenu();
        }
        // 关闭笔记引用浮层 (点击 overlay 外部不关闭) 
    });

    // Escape 关闭右键菜单 & 笔记引用浮层
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            if (sessionMoreMenu) sessionMoreMenu.classList.remove('active');
            sessionMoreMenuTarget = null;
            closeAiMsgContextMenu();
            if (refModal && refModal.style.display !== 'none') {
                closeNoteRefModal();
            }
        }
    });


    // AI 消息右键菜单项点击
    if (aiMsgContextMenu) {
        aiMsgContextMenu.addEventListener('click', (e) => {
            const item = e.target.closest('.context-menu-item');
            if (!item) return;
            const action = item.dataset.action;
            const content = _contextMsgContent;
            const msgEl = _contextMsgEl;

            closeAiMsgContextMenu();

            if (action === 'copy') {
                navigator.clipboard.writeText(content).then(() => {
                    window.showNotification?.('已复制', 'success');
                }).catch(() => {});
            } else if (action === 'save') {
                (async () => {
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
                })();
            } else if (action === 'regen') {
                handleRegenerate(msgEl);
            } else if (action === 'edit') {
                if (isStreaming) return;
                if (msgEl) enterEditMode(msgEl, content);
            } else if (action === 'resend') {
                if (isStreaming) return;
                if (msgEl) handleResend(msgEl);
            } else if (action === 'delete') {
                if (isStreaming) return;
                if (msgEl) handleDeleteMsg(msgEl);
            } else if (action === 'followUp') {
                try {
                    const safeContent = String(content || '');
                    const excerpt = safeContent.replace(/\s+/g, ' ').trim().slice(0, 80);
                    followUpRef = safeContent;
                    const bar = document.getElementById('aiChatFollowUpBar');
                    const text = document.getElementById('aiChatFollowUpText');
                    if (bar && text) {
                        text.textContent = '引用: ' + excerpt + (safeContent.length > 80 ? '…' : '');
                        bar.style.display = 'flex';
                    }
                } catch (_) {}
            }
        });
    }

    // ── 笔记引用浮层标签筛选 ──
    if (refTagBtn) {
        refTagBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            // 关闭笔记本下拉（互斥）
            if (refNotebookFilter) refNotebookFilter.classList.remove('open');
            refTagFilter.classList.toggle('open');
            if (refTagFilter.classList.contains('open')) {
                renderRefTagDropdown();
            }
        });
    }
    // ── 笔记引用浮层笔记本下拉 ──
    if (refNotebookBtn) {
        refNotebookBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            // 关闭标签下拉（互斥）
            if (refTagFilter) refTagFilter.classList.remove('open');
            refNotebookFilter.classList.toggle('open');
            if (refNotebookFilter.classList.contains('open')) {
                rebuildRefNotebookOptions();
            }
        });
    }
    // 点击外部关闭标签下拉和笔记本下拉
    document.addEventListener('click', (e) => {
        if (refTagFilter && !refTagFilter.contains(e.target)) {
            refTagFilter.classList.remove('open');
        }
        if (refNotebookFilter && !refNotebookFilter.contains(e.target)) {
            refNotebookFilter.classList.remove('open');
        }
    });

    // 点击下拉菜单外部关闭
    document.addEventListener('click', (e) => {
        if (sessionMoreMenu && !sessionMoreMenu.contains(e.target)) {
            sessionMoreMenu.classList.remove('active');
            sessionMoreMenuTarget = null;
        }
    });

    // ── 添加菜单（引用笔记 + 上传文件） ──
    if (addBtn && addDropdown) {
        addBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            addDropdown.classList.toggle('open');
        });

        // 点击菜单项
        addDropdown.addEventListener('click', async (e) => {
            const item = e.target.closest('.ai-chat-add-item');
            if (!item) return;

            const action = item.dataset.action;
            addDropdown.classList.remove('open');

            if (action === 'ref') {
                openNoteRefModal();
            } else if (action === 'upload') {
                // 上传文件
                addBtn.disabled = true;
                try {
                    const results = await window.go.main.App.SelectAIChatFiles();
                    if (!results || results.length === 0) return;

                    for (const r of results) {
                        if (r.error) {
                            window.showNotification?.(r.error, 'error');
                        } else {
                            uploadedFiles.push({
                                name: r.name,
                                content: r.content,
                                size: r.size,
                                truncated: r.truncated,
                            });
                        }
                    }
                    renderFileChips();
                } catch (e) {
                    window.showNotification?.('上传文件失败: ' + (e.message || e), 'error');
                } finally {
                    addBtn.disabled = false;
                }
            }
        });
    }


}

/* ── 问答 / Agent 模式 ── */

/**
 * 切换问答 / Agent 模式（持久化到当前会话配置）
 * @param {boolean} enabled - true=Agent 模式, false=问答模式
 */
async function switchAgentMode(enabled) {
    if (isStreaming || agentEnabled === enabled) return;
    agentEnabled = enabled;
    updateAgentModeUI();
    await saveCurrentSessionConfig();
}

/**
 * 根据当前会话的 Agent 模式状态刷新工具栏 UI：
 * - 高亮模式切换控件（问答 / Agent）
 * - Agent 模式隐藏 联网搜索 / 卡片召回 两个开关（深度思考开关保留可用；
 *   仅隐藏不修改开关值，切回时原值保留）
 */
function updateAgentModeUI() {
    if (agentModeQaBtn) agentModeQaBtn.classList.toggle('active', !agentEnabled);
    if (agentModeAgentBtn) agentModeAgentBtn.classList.toggle('active', agentEnabled);
    // 隐藏/显示问答模式专属开关容器（限定在 AI 对话工具栏内）
    const toolbarEl = document.querySelector('.ai-chat-toolbar');
    const sourcesWrapEl = toolbarEl?.querySelector('.ai-chat-sources-wrap');
    const recallWrapEl = toolbarEl?.querySelector('.ai-chat-recall-wrap');
    if (sourcesWrapEl) sourcesWrapEl.classList.toggle('ai-chat-mode-hidden', agentEnabled);
    if (recallWrapEl) recallWrapEl.classList.toggle('ai-chat-mode-hidden', agentEnabled);
}

/* ── 会话侧栏管理 ── */

/**
 * 加载会话列表并渲染侧栏
 */
async function loadSessionList() {
    try {
        sessions = await window.go.main.App.GetAISessions() || [];
        // 构建会话 Token 数查找表，供切换会话时直接显示
        window._sessionTokens = {};
        sessions.forEach(s => { window._sessionTokens[s.id] = s.context_tokens || 0; });
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
                updateChatTitle();
                renderSessionList();
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

    // 找出置顶到非置顶的过渡索引（后端已排序: 置顶优先）
    let pinDividerIndex = -1;
    for (let i = 0; i < filteredSessions.length - 1; i++) {
        if (filteredSessions[i].is_pinned && !filteredSessions[i + 1].is_pinned) {
            pinDividerIndex = i;
            break;
        }
    }

    filteredSessions.forEach((s, index) => {
        const item = document.createElement('div');
        item.className = 'ai-session-item' + (s.id === activeSessionId ? ' active' : '');
        if (s.is_pinned) item.classList.add('pinned');
        item.dataset.id = s.id;

        // 置顶图标
        if (s.is_pinned) {
            const pinIcon = document.createElement('span');
            pinIcon.className = 'ai-session-item-pin-icon';
            pinIcon.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="17" x2="12" y2="22"/><path d="M5 17h14v-1.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1v4.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24Z"/></svg>';
            item.appendChild(pinIcon);
        }

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

        // 更多按钮 (替换原来的删除按钮)
        const moreBtn = document.createElement('button');
        moreBtn.className = 'ai-session-item-more';
        moreBtn.textContent = '⋯';
        moreBtn.title = '更多操作';
        item.appendChild(moreBtn);

        // 点击切换会话
        item.addEventListener('click', (e) => {
            if (e.target === moreBtn || moreBtn.contains(e.target)) return;
            switchSession(s.id);
        });

        // 双击内联编辑
        title.addEventListener('dblclick', () => startInlineEdit(title, s.id));

        // 更多按钮点击 - 切换统一下拉菜单（已打开则关闭）
        moreBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (isStreaming) return;
            // 如果菜单已打开且是同一个会话条目，则关闭
            if (sessionMoreMenu.classList.contains('active') && sessionMoreMenuTarget === item) {
                sessionMoreMenu.classList.remove('active');
                sessionMoreMenuTarget = null;
                return;
            }
            // 定位到按钮旁边
            const rect = moreBtn.getBoundingClientRect();
            let left = rect.right + 4;
            let top = rect.top;
            if (left + 160 > window.innerWidth) {
                left = rect.left - 160 - 4;
            }
            if (top + 80 > window.innerHeight) {
                top = window.innerHeight - 80 - 8;
            }
            if (left < 4) left = 4;
            if (top < 4) top = 4;
            showSessionMenu(s, item, left, top);
        });

        // 右键弹出统一菜单
        item.addEventListener('contextmenu', (e) => {
            e.preventDefault();
            e.stopPropagation();
            if (isStreaming) return;
            showSessionMenu(s, item, e.clientX, e.clientY);
        });

        sessionListEl.appendChild(item);

        // 在置顶会话和普通会话之间插入分隔线
        if (index === pinDividerIndex) {
            const divider = document.createElement('div');
            divider.className = 'ai-session-pin-divider';
            sessionListEl.appendChild(divider);
        }
    });

    // 滚动到激活的会话条目
    const activeItem = sessionListEl.querySelector('.ai-session-item.active');
    if (activeItem) {
        activeItem.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
}

/**
 * 切换会话
 */
async function switchSession(id) {
    if (isStreaming || id === activeSessionId) return;

    // 切换会话时收起 Agent 反问面板
    hideAskPanel();

    // 切换会话时清空笔记引用和技能
    referencedNotes = [];
    updateRefChips();
    activeSkills = {};
    renderSkillChips();

    try {
        activeSessionId = id;
        _loadingMore = false;
        const msgs = await window.go.main.App.LoadAISessionMessagesPaginated(id, 6, 0);

        // 加载会话配置并恢复操作栏状态
        try {
            const config = await window.go.main.App.LoadSessionConfig(id);
            if (config) {
                if (config.model_name && modelLabel) modelLabel.textContent = config.model_name;
                // 恢复问答 / Agent 模式状态并刷新工具栏 UI
                agentEnabled = !!config.agent_enabled;
                updateAgentModeUI();
                enableThinking = !!config.enable_thinking;
                if (searchToggle) searchToggle.classList.toggle('active', enableThinking);
                // 恢复搜索源
                searchSources = new Set();
                ['aiChatZhihuSearch', 'aiChatZhihuGlobalSearch', 'aiChatTavilySearch'].forEach(sid => {
                    const cb = document.getElementById(sid);
                    if (!cb || cb.disabled) return;
                    const source = cb.dataset.source;
                    let enabled = false;
                    if (source === 'zhihu_search') enabled = !!config.zhihu_search_enabled;
                    else if (source === 'zhihu_global') enabled = !!config.zhihu_global_search_enabled;
                    else if (source === 'tavily') enabled = !!config.tavily_search_enabled;
                    cb.checked = enabled;
                    if (enabled) searchSources.add(source);
                });
                if (searchSourcesBtn) {
                    searchSourcesBtn.classList.toggle('active', searchSources.size > 0);
                }
                enableCardRecall = !!config.enable_card_recall;
                if (cardRecallToggle) cardRecallToggle.classList.toggle('active', enableCardRecall);
                // 加载卡片召回笔记本选择
                recallNotebookIds = new Set();
                const savedIds = JSON.parse(config.recall_notebook_ids || '[]');
                if (savedIds.length > 0) {
                    savedIds.forEach(id => recallNotebookIds.add(id));
                } else if (enableCardRecall) {
                    // 旧数据/默认：卡片召回开启时默认全选
                    try {
                        const notebooks = await window.go.main.App.GetAllNotebooks();
                        if (notebooks) {
                            notebooks.forEach(nb => recallNotebookIds.add(nb.id));
                        }
                    } catch (_) {}
                }
                // 重置菜单内容，下次打开时重新加载
                if (cardRecallDropdown) cardRecallDropdown.innerHTML = '';
                referencedNotes = JSON.parse(config.referenced_notes || '[]');
                updateRefChips();
                roleplayNotes = JSON.parse(config.roleplay_notes || '[]');
                // renderSkillChips() 会在后面被调用，不需要单独 updateRoleplaySelector
                activeSkills = JSON.parse(config.enabled_skills || '{}');
                // 兼容旧格式：translate.direction → translate.source + translate.target
                if (activeSkills.translate && activeSkills.translate.direction && !activeSkills.translate.source) {
                    const dir = activeSkills.translate.direction;
                    activeSkills.translate = {
                        source: dir === 'to_chinese' ? 'english' : 'chinese',
                        target: dir === 'to_chinese' ? 'chinese' : 'english',
                    };
                }
                renderSkillChips();
            }
        } catch (_) {}

        // 重建 chatHistory (仅渲染缓冲区)
        chatHistory = msgs ? msgs.map(msg => ({ id: msg.id, role: msg.role, content: msg.content, tokens: msg.tokens || 0, reasoning_content: msg.reasoning_content || '', thinking_elapsed: msg.thinking_elapsed || 0, total_elapsed: msg.total_elapsed || 0, search_sources: msg.search_sources || null, recall_cards: msg.recall_cards || null, tool_calls: msg.tool_calls || null })) : [];
        // 记录最旧消息 ID 用于分页加载
        _oldestMsgId = msgs && msgs.length > 0 ? msgs[0].id : 0;

        // 清空消息列表
        messagesInnerEl.innerHTML = '';

        if (!msgs || msgs.length === 0) {
            // 更新侧栏高亮
            document.querySelectorAll('.ai-session-item.active').forEach(el => el.classList.remove('active'));
            const currentItem = document.querySelector(`.ai-session-item[data-id="${id}"]`);
            if (currentItem) currentItem.classList.add('active');
            updateChatTitle();
            showWelcome();
            updateContextSize();
            scrollToBottom();
            return;
        }

        // 有消息时隐藏欢迎语
        hideWelcome();
        // 一次性渲染所有消息（不 yield，避免浏览器的中间状态绘制导致视觉跳跃）
        for (let i = 0; i < msgs.length; i += 5) {
            const chunk = msgs.slice(i, i + 5);
            for (const msg of chunk) {
                if (msg.role === 'user') {
                    addMessage(msg.content, 'user', undefined, undefined, undefined, msg.tokens || 0, msg.id, undefined, undefined, undefined, true, true);
                    const userMsgEl = messagesInnerEl.lastElementChild;
                    if (userMsgEl) {
                        userMsgEl.appendChild(createMsgActions(msg.content, 'user', undefined, msg.tokens || 0));
                        bindMsgContextMenu(userMsgEl, msg.content, 'user');
                    }
                } else if (msg.role === 'assistant') {
                    const el = addMessage(msg.content, 'assistant', msg.reasoning_content || '', msg.thinking_elapsed || 0, msg.total_elapsed || 0, msg.tokens || 0, msg.id, msg.search_sources, msg.recall_cards, msg.tool_calls, true, true);
                    el.appendChild(createMsgActions(msg.content, 'assistant', 0, msg.tokens || 0));
                    bindMsgContextMenu(el, msg.content, 'assistant');
                }
            }
        }

        // 所有消息渲染完成后，同步滚动到底部（不用 rAF，确保在浏览器绘制前完成）
        messagesEl.style.scrollBehavior = 'auto';
        messagesEl.scrollTop = messagesEl.scrollHeight;
        messagesEl.style.scrollBehavior = '';

        // 仅更新侧栏会话的高亮状态（比全量 renderSessionList + loadSessionList 轻量得多）
        document.querySelectorAll('.ai-session-item.active').forEach(el => el.classList.remove('active'));
        const currentItem = document.querySelector(`.ai-session-item[data-id="${id}"]`);
        if (currentItem) currentItem.classList.add('active');
        updateChatTitle();

        // 直接从 chatHistory 的消息 tokens 汇总显示（避免依赖缓存）
        updateContextSize();

        // 添加滚动加载更多（移除旧监听器防止重复绑定）
        if (_scrollHandler) {
            messagesEl.removeEventListener('scroll', _scrollHandler);
        }
        _scrollHandler = async () => {
            if (_loadingMore || !activeSessionId) return;
            if (messagesEl.scrollTop > 0) return;
            if (_oldestMsgId <= 0) return;
            _loadingMore = true;
            const prevScrollHeight = messagesEl.scrollHeight;
            try {
                const olderMsgs = await window.go.main.App.LoadAISessionMessagesPaginated(activeSessionId, 6, _oldestMsgId);
                if (!olderMsgs || olderMsgs.length === 0) {
                    _oldestMsgId = 0;
                    _loadingMore = false;
                    return;
                }
                _oldestMsgId = olderMsgs[0].id;
                // 记录当前第一个子元素作为锚点
                const firstChild = messagesInnerEl.firstChild;
                // 渲染新消息（addMessage 会追加到末尾）
                for (const msg of olderMsgs) {
                    if (msg.role === 'user') {
                        addMessage(msg.content, 'user', undefined, undefined, undefined, msg.tokens || 0, msg.id, undefined, undefined, undefined, true, true);
                        const userMsgEl = messagesInnerEl.lastElementChild;
                        if (userMsgEl) {
                            userMsgEl.appendChild(createMsgActions(msg.content, 'user', undefined, msg.tokens || 0));
                            bindMsgContextMenu(userMsgEl, msg.content, 'user');
                        }
                    } else if (msg.role === 'assistant') {
                        const el = addMessage(msg.content, 'assistant', msg.reasoning_content || '', msg.thinking_elapsed || 0, msg.total_elapsed || 0, msg.tokens || 0, msg.id, msg.search_sources, msg.recall_cards, msg.tool_calls, true, true);
                        el.appendChild(createMsgActions(msg.content, 'assistant', 0, msg.tokens || 0));
                        bindMsgContextMenu(el, msg.content, 'assistant');
                    }
                }
                // 将新消息按原有 ASC 顺序移到列表顶部
                const firstNewIdx = messagesInnerEl.children.length - olderMsgs.length;
                const newChildren = Array.from(messagesInnerEl.children).slice(firstNewIdx);
                for (const el of newChildren) {
                    messagesInnerEl.insertBefore(el, firstChild);
                }
                // 恢复滚动位置
                messagesEl.style.scrollBehavior = 'auto';
                messagesEl.scrollTop = messagesEl.scrollHeight - prevScrollHeight;
                messagesEl.style.scrollBehavior = '';
                // 更新 chatHistory 缓冲区
                chatHistory.unshift(...olderMsgs.map(msg => ({ id: msg.id, role: msg.role, content: msg.content, tokens: msg.tokens || 0, reasoning_content: msg.reasoning_content || '', thinking_elapsed: msg.thinking_elapsed || 0, total_elapsed: msg.total_elapsed || 0, search_sources: msg.search_sources || null, recall_cards: msg.recall_cards || null, tool_calls: msg.tool_calls || null })));
            } catch (_) { /* 静默 */ }
            _loadingMore = false;
        };
        messagesEl.addEventListener('scroll', _scrollHandler, { passive: true });

        inputEl?.focus();
    } catch (_) { /* 静默失败 */ }
}

/**
 * 触发按钮脉冲反馈动画
 */
function triggerPulseFeedback(el) {
    if (!el) return;
    el.classList.remove('anim-pulse');
    void el.offsetWidth; // 强制回流以重播动画
    el.classList.add('anim-pulse');
}

/**
 * 创建新会话
 */
async function createSession() {
    if (isStreaming) return;

    // 当前会话为空 (无消息) 时不允许新建, 避免空会话堆积
    if (activeSessionId !== null && chatHistory.length === 0) return;

    let id;
    try {
        id = await window.go.main.App.CreateAISession();
    } catch (_) {
        return;
    }

    // 清空当前状态
    activeSessionId = id;

    // 加载默认会话配置
    try {
        const defaultCfg = await window.go.main.App.LoadSessionConfig(id);
        if (defaultCfg) {
            if (defaultCfg.model_name) modelLabel.textContent = defaultCfg.model_name;
            // 恢复问答 / Agent 模式状态并刷新工具栏 UI
            agentEnabled = !!defaultCfg.agent_enabled;
            updateAgentModeUI();
            enableThinking = !!defaultCfg.enable_thinking;
            if (searchToggle) searchToggle.classList.toggle('active', enableThinking);
            searchSources = new Set();
            ['aiChatZhihuSearch', 'aiChatZhihuGlobalSearch', 'aiChatTavilySearch'].forEach(sid => {
                const cb = document.getElementById(sid);
                if (!cb || cb.disabled) return;
                const source = cb.dataset.source;
                let enabled = false;
                if (source === 'zhihu_search') enabled = !!defaultCfg.zhihu_search_enabled;
                else if (source === 'zhihu_global') enabled = !!defaultCfg.zhihu_global_search_enabled;
                else if (source === 'tavily') enabled = !!defaultCfg.tavily_search_enabled;
                cb.checked = enabled;
                if (enabled) searchSources.add(source);
            });
            if (searchSourcesBtn) {
                searchSourcesBtn.classList.toggle('active', searchSources.size > 0);
            }
            enableCardRecall = !!defaultCfg.enable_card_recall;
            if (cardRecallToggle) cardRecallToggle.classList.toggle('active', enableCardRecall);
            // 加载卡片召回笔记本选择
            recallNotebookIds = new Set();
            const savedIds = JSON.parse(defaultCfg.recall_notebook_ids || '[]');
            if (savedIds.length > 0) {
                savedIds.forEach(id => recallNotebookIds.add(id));
            } else if (enableCardRecall) {
                // 旧数据/默认：卡片召回开启时默认全选
                try {
                    const notebooks = await window.go.main.App.GetAllNotebooks();
                    if (notebooks) {
                        notebooks.forEach(nb => recallNotebookIds.add(nb.id));
                    }
                } catch (_) {}
            }
            // 重置菜单内容，下次打开时重新加载
            if (cardRecallDropdown) cardRecallDropdown.innerHTML = '';
            referencedNotes = JSON.parse(defaultCfg.referenced_notes || '[]');
            updateRefChips();
            roleplayNotes = JSON.parse(defaultCfg.roleplay_notes || '[]');
            // renderSkillChips() 会在后面被调用，不需要单独 updateRoleplaySelector
            activeSkills = JSON.parse(defaultCfg.enabled_skills || '{}');
            // 兼容旧格式：translate.direction → translate.source + translate.target
            if (activeSkills.translate && activeSkills.translate.direction && !activeSkills.translate.source) {
                const dir = activeSkills.translate.direction;
                activeSkills.translate = {
                    source: dir === 'to_chinese' ? 'english' : 'chinese',
                    target: dir === 'to_chinese' ? 'chinese' : 'english',
                };
            }
            renderSkillChips();
        }
    } catch (_) {}

    chatHistory = [];
    updateContextSize();
    messagesInnerEl.innerHTML = '';
    hideEmptyState();
    showWelcome();

    await loadSessionList();

    // 为新条目添加入场动画（通过 data-id 精确查找新建的会话）
    const newItem = sessionListEl.querySelector(`.ai-session-item[data-id="${id}"]`);
    if (newItem) {
        newItem.classList.add('anim-slide-in');
    }

    scrollToBottom();
}

/* ── 对话管理 ── */

/**
 * 输入框键盘事件
 */
function onInputKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey && !e.ctrlKey) {
        e.preventDefault();
        onSend();
    }
    if (e.key === 'Enter' && e.ctrlKey) {
        e.preventDefault();
        // 在光标位置插入换行
        const start = inputEl.selectionStart;
        const end = inputEl.selectionEnd;
        inputEl.value = inputEl.value.substring(0, start) + '\n' + inputEl.value.substring(end);
        inputEl.selectionStart = inputEl.selectionEnd = start + 1;
        // 触发 input 事件使发送按钮状态更新
        inputEl.dispatchEvent(new Event('input', { bubbles: true }));
    }
}

/**
 * auto-resize textarea
 */
function autoResizeInput() {
    inputEl.style.height = 'auto';
    inputEl.style.height = Math.min(inputEl.scrollHeight, 140) + 'px';

    // 检测单行/多行，切换按钮居中位置
    const wrap = document.getElementById('aiChatPolishBtn')?.closest('.ai-chat-input-wrap');
    if (wrap) {
        const h = parseInt(inputEl.style.height) || 0;
        wrap.classList.toggle('is-single-line', h <= 45);
    }
}

/* ── 更多技能 ── */

/**
 * 渲染技能 chip 指示器
 */
function renderSkillChips() {
    if (!skillBar || !skillChips) return;

    // 如果取消了角色扮演技能，清空 roleplayNotes（必须在 keys 为空提前返回之前执行）
    if (!activeSkills.roleplay) {
        clearRoleplayNotes();
    }

    const keys = Object.keys(activeSkills);
    if (keys.length === 0) {
        skillBar.style.display = 'none';
        if (skillsBtn) {
            skillsBtn.disabled = false;
            skillsBtn.classList.remove('is-disabled');
        }
        return;
    }
    skillBar.style.display = '';
    if (skillsBtn) {
        skillsBtn.disabled = true;
        skillsBtn.classList.add('is-disabled');
    }
    skillChips.innerHTML = keys.map(skillId => {
        const config = activeSkills[skillId];
        if (skillId === 'translate') {
            const sourceLang = getLanguageDisplayName(config.source || 'english');
            const targetLang = getLanguageDisplayName(config.target || 'chinese');
            return `<div class="ai-chat-skill-chip ai-chat-skill-chip-translate" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-translate-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M8 11l3 3 5-5"/></svg></span>
                <span class="ai-chat-skill-chip-translate-lang" data-side="source">${sourceLang}</span>
                <span class="ai-chat-skill-chip-translate-arrow">⇄</span>
                <span class="ai-chat-skill-chip-translate-lang" data-side="target">${targetLang}</span>
                <button class="ai-chat-skill-chip-translate-remove" title="取消翻译" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'coding') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg></span>
                <span class="ai-chat-skill-chip-label">编程开发</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'writing') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></span>
                <span class="ai-chat-skill-chip-label">创意写作</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'tutor') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>
                <span class="ai-chat-skill-chip-label">解题答疑</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'reqspec') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg></span>
                <span class="ai-chat-skill-chip-label">需求规格</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'polish') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg></span>
                <span class="ai-chat-skill-chip-label">文本润色</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'summary') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></span>
                <span class="ai-chat-skill-chip-label">内容摘要</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'copywriting') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></span>
                <span class="ai-chat-skill-chip-label">文案生成</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'report') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg></span>
                <span class="ai-chat-skill-chip-label">工作总结</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'promptgen') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg></span>
                <span class="ai-chat-skill-chip-label">提示词生成</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'character') {
            return `<div class="ai-chat-skill-chip" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg></span>
                <span class="ai-chat-skill-chip-label">人物档案</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>`;
        } else if (skillId === 'roleplay') {
            const count = roleplayNotes.length;
            const label = count > 0 ? count + ' 篇' : '未设置';
            const countTitle = count > 0 ? roleplayNotes.map(n => n.title || '无标题').join(' · ') : '';
            return `<div class="ai-chat-skill-chip ai-chat-skill-chip-roleplay" data-skill="${skillId}">
                <span class="ai-chat-skill-chip-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a4 4 0 0 1 4 4v2a4 4 0 0 1-8 0V6a4 4 0 0 1 4-4z"/><path d="M16 14h2a4 4 0 0 1 4 4v2H2v-2a4 4 0 0 1 4-4h2"/><circle cx="12" cy="14" r="3"/></svg></span>
                <span class="ai-chat-skill-chip-label" title="${countTitle.replace(/"/g, '&quot;')}">角色扮演: ${label}</span>
                <button class="ai-chat-skill-chip-remove" title="取消技能" data-skill="${skillId}">${SVGS.windowClose}</button>
            </div>`;
        }
        return '';
    }).join('');

    // 绑定 chip 叉号点击事件（非翻译技能）
    skillChips.querySelectorAll('.ai-chat-skill-chip-remove').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.stopPropagation();
            const skill = btn.dataset.skill;
            delete activeSkills[skill];
            renderSkillChips();
            await saveCurrentSessionConfig();
        });
    });
    
    // 绑定翻译 chip 移除按钮点击事件
    skillChips.querySelectorAll('.ai-chat-skill-chip-translate-remove').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.stopPropagation();
            const skill = btn.dataset.skill;
            delete activeSkills[skill];
            renderSkillChips();
            await saveCurrentSessionConfig();
        });
    });
    
    // 绑定翻译 chip 语言标签点击事件：打开语言选择浮层
    const translateChip = skillChips.querySelector('.ai-chat-skill-chip-translate');
    if (translateChip) {
        translateChip.addEventListener('click', (e) => {
            const langLabel = e.target.closest('.ai-chat-skill-chip-translate-lang');
            if (langLabel && activeSkills.translate) {
                const side = langLabel.dataset.side; // 'source' or 'target'
                openLangPicker(langLabel, side);
                return;
            }
        });
    }
    
    // 绑定角色扮演 chip 点击事件：打开角色档案选择器
    const roleplayChip = skillChips.querySelector('.ai-chat-skill-chip-roleplay');
    if (roleplayChip) {
        roleplayChip.addEventListener('click', (e) => {
            if (e.target.closest('.ai-chat-skill-chip-remove')) return;
            openRoleplaySelector();
        });
    }
}

/**
 * 更新更多技能菜单中的选中状态（✓ + accent 高亮）
 */
function updateSkillsMenuActiveState() {
    if (!skillsDropdown) return;
    const items = skillsDropdown.querySelectorAll('.ai-chat-skills-item');
    items.forEach(item => {
        const skill = item.dataset.skill;
        item.classList.toggle('active', !!activeSkills[skill]);
    });
}

/**
 * 带离场动画关闭技能菜单
 */
function closeSkillsDropdown() {
    if (!skillsDropdown || !skillsDropdown.classList.contains('open')) return;
    skillsDropdown.classList.add('closing');
    // 子项最大关闭时长 = 0.24s(延迟) + 0.1s(动画) = 0.34s，加一点余量
    const CLOSE_DURATION = 360;
    setTimeout(() => {
        skillsDropdown.classList.remove('open', 'closing');
    }, CLOSE_DURATION);
}

/**
 * 语言显示名称映射
 */
const LANG_NAMES = {
    'english': 'English',
    'chinese': '中文',
    'japanese': '日本語',
    'french': 'Français',
    'german': 'Deutsch',
    'spanish': 'Español',
};

/**
 * 语言代码列表（保持显示顺序）
 */
const LANG_CODES = Object.keys(LANG_NAMES);

/**
 * 获取语言的显示名称
 */
function getLanguageDisplayName(code) {
    return LANG_NAMES[code] || code;
}

/**
 * 清空角色档案笔记
 */
function clearRoleplayNotes() {
    roleplayNotes = [];
}

/**
 * 打开笔记选择器选择角色档案笔记
 */
function openRoleplaySelector() {
    if (!refModal) return;
    
    // 重置临时选中状态
    _refTempSelected = {};
    _refSelectAll = false;
    // 以当前角色档案笔记预填
    roleplayNotes.forEach(n => { _refTempSelected[n.id] = true; });
    
    refModal.style.display = 'flex';
    
    // 重置搜索/筛选
    if (refSearch) refSearch.value = '';
    if (refSearchClear) refSearchClear.classList.remove('visible');
    _currentNotebookId = 0;
    if (refNotebookLabel) refNotebookLabel.textContent = '全部笔记本';
    if (refNotebookBtn) refNotebookBtn.classList.remove('active');
    if (refNotebookFilter) refNotebookFilter.classList.remove('open');
    
    // 移除常规确认处理函数，防止冲突
    if (refConfirm) {
        refConfirm.removeEventListener('click', confirmNoteSelection);
        
        // 设置角色档案专用的确认 handler，立即生效
        refConfirm.onclick = async () => {
            const selectedIds = Object.keys(_refTempSelected);
            if (selectedIds.length === 0) {
                roleplayNotes = [];
                closeNoteRefModal();
                renderSkillChips();
                await saveCurrentSessionConfig();
                return;
            }
            
            // 限制最大 3 篇
            if (selectedIds.length > 3) {
                window.showNotification?.('角色档案最多选择 3 篇笔记', 'warning');
                return;
            }
            
            try {
                const ids = selectedIds.map(id => parseInt(id));
                // 直接构建 roleplayNotes（不调用 confirmNoteSelection）
                const notes = await getSelectedNotesInfo(ids);
                if (notes && notes.length > 0) {
                    roleplayNotes = notes;
                }
            } catch (_) {}
            
            closeNoteRefModal();
            renderSkillChips();
            await saveCurrentSessionConfig();
        };
    }
    
    // 读取分页设置
    (async () => {
        await loadRefPageSize?.();
        _refListLoaded = false;
        _refCurrentPage = 1;
        _refTotalItems = 0;
        _refLoading = false;
        _refPendingRefresh = false;
        _refTagIds.clear();
        if (refTagLabel) refTagLabel.textContent = '标签';
        if (refTagBtn) refTagBtn.classList.remove('active');
        if (refListWrap) refListWrap.scrollTop = 0;
        
        // 并行加载笔记本选项和笔记列表
        await Promise.all([
            loadAllNotebooks(),
            loadNoteList()
        ]);
    })();
}

/**
 * 获取选中笔记的信息（id, title, notebook_name）
 * @param {number[]} ids
 * @returns {Promise<Array>}
 */
async function getSelectedNotesInfo(ids) {
    if (!ids || ids.length === 0) return [];
    try {
        const refContext = await window.go.main.App.GetNoteRefContext(ids);
        if (refContext && refContext.notes) {
            return refContext.notes.map(n => ({ id: n.id, title: n.title, notebook_name: n.notebook_name }));
        }
    } catch (_) {}
    return [];
}

/**
 * 校验 AI 配置完整性：API 地址 / API Key / 模型三要素齐全才允许发起 AI 调用
 * @param {string} actionLabel 动作描述（用于提示文案），如 '开始对话'
 * @returns {Promise<boolean>} true 表示配置完整可继续
 */
async function ensureAIReady(actionLabel) {
    let cfg = null;
    try {
        cfg = await window.go.main.App.GetAIConfig();
    } catch (_) {}
    const hasConfig = !!(cfg && cfg.base_url && cfg.base_url.trim() && cfg.api_key && cfg.api_key.trim());
    if (!hasConfig) {
        window.showNotification?.('请先在设置中完整配置 AI 服务（API 地址 / API Key / 模型），再' + actionLabel + '。', 'warning');
        return false;
    }
    const currentModel = modelLabel?.textContent;
    if (!currentModel || currentModel === '--') {
        window.showNotification?.('请先在模型选择下拉列表中选一个模型，再' + actionLabel + '。', 'warning');
        return false;
    }
    return true;
}

/**
 * 发送消息
 */
async function onSend() {
    const text = inputEl.value.trim();
    if (!text || isStreaming) return;

    // 校验 AI 配置完整性与模型选择
    if (!(await ensureAIReady('开始对话'))) return;

    // 如果没有激活的会话, 自动创建
    if (activeSessionId === null) {
        await createSession();
        if (activeSessionId === null) return;
    }

    inputEl.value = '';
    inputEl.style.height = 'auto';
    sendBtnEl.disabled = true;
    // 重置优化表达状态
    if (polishBtn) {
        polishBtn.classList.remove('is-revert');
        polishBtn.querySelector('.ai-chat-polish-text').textContent = '优化';
        polishBtn.disabled = true;
        polishOriginalText = '';
    }

    hideWelcome();

    // 保存消息 → 渲染用户气泡 → 启动流式（公共发送逻辑）
    await sendUserText(text);

    // 发送后清空上传文件列表
    uploadedFiles = [];
    renderFileChips();
}

/**
 * 发送用户消息（onSend 调用；Agent 反问面板的答案不走此函数，
 * 而是经 AnswerAskUser 同轮投递，见 showAskPanel 的 doSend）
 * 保存消息到数据库 → 渲染用户气泡 → 启动流式。
 * onSend 已做过 ensureAIReady / createSession 时重复执行无害（幂等）。
 */
async function sendUserText(text) {
    if (!text) return false;
    if (isStreaming) {
        // 当前回复仍在生成：不静默丢弃，提示后保留反问面板
        window.showNotification?.('请等待当前回复完成后再选择', 'info');
        return false;
    }
    if (!(await ensureAIReady('开始对话'))) return false; // 未配置：保留面板，方便配置后重试
    if (activeSessionId === null) {
        await createSession();
        if (activeSessionId === null) return false;
    }
    // 先保存用户消息到数据库，确保前端能立即拿到 msgId 和 token 数
    let userMsgId = 0;
    let userTokens = 0;
    try {
        const result = await window.go.main.App.SaveAIMessage(activeSessionId, text, 'user');
        userMsgId = result?.msgID || 0;
        userTokens = result?.tokens || 0;
    } catch (_) { /* 静默失败，后续流程继续 */ }

    addMessage(text, 'user', undefined, undefined, undefined, userTokens, userMsgId || undefined);
    const userMsgEl = messagesInnerEl.lastElementChild;
    if (userMsgEl) {
        userMsgEl.appendChild(createMsgActions(text, 'user', undefined, userTokens));
        bindMsgContextMenu(userMsgEl, text, 'user');
    }
    startStreaming(text, false, userMsgId);
    return true;
}

/**
 * 启动流式输出
 * @param {boolean} isRegenerate - 是否再生
 * @param {number} userMsgID - 已保存的用户消息 ID（由 onSend 或 handleResend 提前获取）
 */
async function startStreaming(userText, isRegenerate, userMsgID) {
    if (isStreaming) return;
    isStreaming = true;
    window.__aiStreaming = true;

    // 递增 generation, 后续事件回调据此判断是否属于当前流
    _aiStreamGen++;
    const myGen = _aiStreamGen;

    // 记录发送时的模式快照：Agent 流的事件参数形态与问答流不同（chunk 为单参数），
    // 用快照区分，避免发送过程中切换模式导致解析错乱
    const isAgentFlow = agentEnabled;

    // 清除该事件名下所有旧监听器, 防止残留
    // （Wails v2 EventsOff 每次只接受一个事件名，逐个清除）
    ['ai:stream-done', 'ai:stream-error', 'ai:stream-chunk', 'ai:stream-thinking', 'ai:search-status', 'ai:search-sources', 'ai:search-source-status', 'ai:search-error', 'ai:recall-cards', 'ai:recall-status', 'ai:refined-keywords', 'ai:tool-status', 'ai:agent-result', 'ai:ask-user'].forEach(function(name) {
        window.runtime.EventsOff(name);
    });

    // 显示停止按钮, 隐藏发送按钮
    if (stopBtnEl) stopBtnEl.style.display = '';
    if (sendBtnEl) sendBtnEl.style.display = 'none';
    if (polishBtn) polishBtn.disabled = true;

    let streamingContent = '';
    let streamingThinking = '';
    let hasReceivedChunk = false;
    let streamSearchSources = null;
    let searchSourceStates = {};
    let recallCards = null;
    let recallIndicatorStart = 0;   // 召回指示器开始展示的时间戳，用于保证最短展示时长
    let recallSwapTimer = null;     // 召回指示器最小展示时长的延迟切换句柄（thinking 到达时立即打断）
    let recallPendingStatus = '';   // 延迟切换等待中的收尾状态（'done'/'error'）
    let recallPendingDetail = '';   // 延迟切换等待中的错误详情
    let refinedKeywords = '';

    // 新一轮输出开始：收起 Agent 反问面板、清除反问等待状态
    //（提交回答后由 AnswerAskUser 触达此处，防御性重置）
    hideAskPanel();

    const streamingEl = document.createElement('div');
    streamingEl.className = 'ai-msg ai-msg-assistant';
    const contentDiv = document.createElement('div');
    contentDiv.className = 'msg-content';
    contentDiv.appendChild(createTypingDots());
    streamingEl.appendChild(contentDiv);
    messagesInnerEl.appendChild(streamingEl);

    let thinkingDetails = null;
    let thinkingContentEl = null;
    let _thinkingStartedAt = 0;
    let _thinkingTimer = null;
    let unsubs = [];

    /** 更新思维链实时计时摘要 */
    function updateThinkingTimer() {
        if (_thinkingStartedAt <= 0) return;
        const elapsed = (Date.now() - _thinkingStartedAt) / 1000;
        const summary = thinkingDetails?.querySelector('.thinking-summary');
        const text = summary?.querySelector('.thinking-text');
        if (text) text.textContent = '思考中 ' + elapsed.toFixed(1) + ' 秒';
    }

    /** 停止实时计时, 设为最终态 */
    function stopThinkingTimer(finalElapsed) {
        if (_thinkingTimer) {
            clearInterval(_thinkingTimer);
            _thinkingTimer = null;
        }
        if (thinkingDetails) {
            const summary = thinkingDetails.querySelector('.thinking-summary');
            const text = summary?.querySelector('.thinking-text');
            if (text && finalElapsed > 0) text.textContent = '已思考 ' + finalElapsed.toFixed(1) + ' 秒';
            if (summary) summary.classList.remove('is-thinking');
        }
    }

    /** 追加思维链分片：懒创建 thinkingDetails 折叠区并流式填充（问答/Agent 两模式共用） */
    const appendThinkingChunk = (chunk) => {
        if (!enableThinking) return; // 深度思考关闭时跳过展示思维链
        if (!thinkingDetails) {
            // 召回指示器若还在最小展示时长延迟中，立即完成收尾（切回打字点），
            // 避免思考动画（thinkingDetails 插在 contentDiv 上方）与检索指示器上下重叠
            if (recallSwapTimer) {
                clearTimeout(recallSwapTimer);
                recallSwapTimer = null;
                finishRecallIndicator(recallPendingStatus, recallPendingDetail);
            }
            _thinkingStartedAt = Date.now();
            thinkingDetails = document.createElement('details');
            thinkingDetails.className = 'thinking-details';
            thinkingDetails.open = localStorage.getItem('ai_cot_collapsed') !== 'true';
            thinkingDetails.addEventListener('toggle', () => {
                localStorage.setItem('ai_cot_collapsed', thinkingDetails.open ? 'false' : 'true');
            });
            const summary = document.createElement('summary');
            summary.className = 'thinking-summary is-thinking';
            summary.innerHTML = svgIcon('brain') + '<span class="thinking-text">思考中</span>';
            thinkingDetails.appendChild(summary);
            thinkingContentEl = document.createElement('div');
            thinkingContentEl.className = 'thinking-content';
            thinkingDetails.appendChild(thinkingContentEl);
            streamingEl.insertBefore(thinkingDetails, contentDiv);
            // 启动实时计时器 (每 200ms 更新) 
            _thinkingTimer = setInterval(updateThinkingTimer, 200);
        }
        streamingThinking += chunk;
        thinkingContentEl.textContent = streamingThinking;
        scrollToBottom();
    };

    const unsubThinking = window.runtime.EventsOn('ai:stream-thinking', (streamGen, chunk) => {
        if (streamGen !== myGen) return; // 属于旧流, 丢弃（问答/Agent 模式后端均携带 gen）
        appendThinkingChunk(chunk);
    });
    unsubs.push(unsubThinking);

    // 多源搜索状态管理
    searchSourceStates = {};

    // 多源搜索状态：精炼阶段 → 搜索阶段 → 完成
    const unsubSearch = window.runtime.EventsOn('ai:search-status', (status) => {
        if (status === 'refining') {
            contentDiv.innerHTML = '';
            contentDiv.appendChild(createSearchIndicator('refining'));
        } else if (status === 'searching') {
            contentDiv.innerHTML = '';
            contentDiv.appendChild(createSearchIndicator('searching', refinedKeywords));
        } else if (status === 'done') {
            // 仅在尚未收到 stream chunk 时替换为打字点
            if (!hasReceivedChunk) {
                contentDiv.innerHTML = '';
                contentDiv.appendChild(createTypingDots());
            }
        }
    });
    unsubs.push(unsubSearch);

    // 卡片召回状态：检索中/完成/失败（独立事件，与联网搜索 ai:search-status 解耦）
    // 本地检索耗时可能极短，指示器保持最小展示时长避免闪烁
    const MIN_RECALL_INDICATOR_MS = 800;
    // 召回指示器收尾：恢复打字点（期间若已收到 chunk 则跳过），error 时弹提示
    // 供召回 done/error 与 thinking 打断共用，保证错误通知不丢失
    const finishRecallIndicator = (status, detail) => {
        recallSwapTimer = null;
        if (!hasReceivedChunk) {
            contentDiv.innerHTML = '';
            contentDiv.appendChild(createTypingDots());
        }
        if (status === 'error' && nm && typeof nm.show === 'function') {
            nm.show('卡片召回失败: ' + (detail || '未知错误'), 'warning');
        }
    };
    const unsubRecallStatus = window.runtime.EventsOn('ai:recall-status', (status, detail) => {
        if (status === 'searching') {
            // 仅当尚未收到 stream chunk 时替换为召回指示器
            if (!hasReceivedChunk) {
                recallIndicatorStart = Date.now();
                contentDiv.innerHTML = '';
                contentDiv.appendChild(createRecallIndicator());
            }
        } else if (status === 'done' || status === 'error') {
            // 保证召回指示器至少展示 MIN_RECALL_INDICATOR_MS，避免本地检索过快导致闪烁；
            // 延迟期间记录 pending 状态，thinking 到达时可立即打断并保留错误提示
            const elapsed = Date.now() - recallIndicatorStart;
            if (recallIndicatorStart > 0 && elapsed < MIN_RECALL_INDICATOR_MS) {
                recallPendingStatus = status;
                recallPendingDetail = detail || '';
                recallSwapTimer = setTimeout(() => finishRecallIndicator(status, detail), MIN_RECALL_INDICATOR_MS - elapsed);
            } else {
                finishRecallIndicator(status, detail);
            }
        }
    });
    unsubs.push(unsubRecallStatus);

    // 搜索源状态更新（仅记录状态，不更新 UI 动画）
    const unsubSourceStatus = window.runtime.EventsOn('ai:search-source-status', (statusJSON) => {
        try {
            const data = typeof statusJSON === 'string' ? JSON.parse(statusJSON) : statusJSON;
            searchSourceStates[data.source] = data;
        } catch (_) {}
    });
    unsubs.push(unsubSourceStatus);

    // 搜索源错误事件 → 弹通知
    const unsubSearchError = window.runtime.EventsOn('ai:search-error', (errJSON) => {
        try {
            const data = typeof errJSON === 'string' ? JSON.parse(errJSON) : errJSON;
            searchSourceStates[data.source] = { source: data.source, status: 'error', error: data.error };
            // 流已停止（用户主动取消），不弹错误通知
            if (!isStreaming) return;
            window.showNotification?.('联网搜索失败 (' + (sourceLabels[data.source] || data.source) + '): ' + data.error, 'error', 5000);
        } catch (_) {}
    });
    unsubs.push(unsubSearchError);

    // 精炼后的搜索关键词
    const unsubKeywords = window.runtime.EventsOn('ai:refined-keywords', (keywords) => {
        refinedKeywords = keywords || '';
    });
    unsubs.push(unsubKeywords);

    // 联网搜索来源数据（结构化来源列表，AI 回复结束后展示）
    const unsubSources = window.runtime.EventsOn('ai:search-sources', (sourcesJSON) => {
        try {
            streamSearchSources = JSON.parse(sourcesJSON);
        } catch (_) {}
    });
    unsubs.push(unsubSources);

    // 卡片召回数据（结构化卡片列表，AI 回复结束后展示）
    const unsubRecall = window.runtime.EventsOn('ai:recall-cards', (cardsJSON) => {
        try {
            recallCards = JSON.parse(cardsJSON);
        } catch (_) {}
    });
    unsubs.push(unsubRecall);

    /** 处理 AI 流式正文 chunk（问答 / Agent 模式共用） */
    const handleStreamChunk = (chunk) => {
        if (!chunk) return;
        if (!hasReceivedChunk) {
            hasReceivedChunk = true;
            contentDiv.innerHTML = '';
            // 首个正文 chunk 到达 → 思考结束, 停止计时并更新摘要
            if (streamingThinking && _thinkingStartedAt > 0) {
                stopThinkingTimer((Date.now() - _thinkingStartedAt) / 1000);
            }
            // Agent 模式：ReAct 循环中工具调用均发生在正文输出前，首个正文 token 即
            // 已全部终态，此时移除上方实时工具条（完成态折叠摘要仍在 stream-done 渲染）
            if (isAgentFlow && toolStatusListEl && toolStatusListEl.parentNode) {
                toolStatusListEl.parentNode.removeChild(toolStatusListEl);
            }
        }
        streamingContent += chunk;
        contentDiv.innerHTML = marked.parse(streamingContent);
        scrollToBottom();
    };

    const unsubChunk = window.runtime.EventsOn('ai:stream-chunk', (streamGen, chunk) => {
        if (streamGen !== myGen) return; // 属于旧流, 丢弃（问答/Agent 模式后端均携带 gen）
        handleStreamChunk(chunk);
    });
    unsubs.push(unsubChunk);

    // ── Agent 模式工具调用过程展示（ai:tool-status） ──
    // 工具状态条渲染在消息气泡正文（contentDiv）上方，样式与搜索/召回指示器保持一致观感
    let toolStatusListEl = null;   // 工具状态容器
    let toolStatusItems = {};      // { [name]: {itemEl, iconEl, textEl} }
    let streamToolRecords = [];    // 本轮流的工具调用记录（落库 tool_calls，历史回放）

    /** 创建工具状态容器（懒创建，插入到正文 contentDiv 上方） */
    const ensureToolStatusList = () => {
        if (toolStatusListEl) return toolStatusListEl;
        toolStatusListEl = document.createElement('div');
        toolStatusListEl.className = 'ai-tool-status-list';
        streamingEl.insertBefore(toolStatusListEl, contentDiv);
        return toolStatusListEl;
    };

    /** 展示工具调用开始状态（tool_start） */
    const showToolStatusStart = (payload) => {
        const name = payload.name || 'tool';
        let action = payload.action_text || '执行';
        let label = '调用「' + getToolLabel(name) + '」工具：' + action;
        let item = toolStatusItems[name];
        if (!item) {
            const list = ensureToolStatusList();
            item = { el: null, iconEl: null, textEl: null };
            item.el = document.createElement('div');
            item.el.className = 'ai-tool-status-item';
            item.el.dataset.name = name;
            item.iconEl = document.createElement('span');
            item.iconEl.className = 'ai-tool-status-icon';
            item.iconEl.innerHTML = svgIcon('search');
            item.textEl = document.createElement('span');
            item.textEl.className = 'ai-tool-status-text';
            item.el.appendChild(item.iconEl);
            item.el.appendChild(item.textEl);
            list.appendChild(item.el);
            toolStatusItems[name] = item;
        }
        item.el.classList.remove('is-done');
        item.el.classList.remove('is-error');
        item.el.classList.remove('is-warning');
        item.el.classList.add('is-active');
        item.iconEl.innerHTML = svgIcon('search');
        item.textEl.textContent = label;
        scrollToBottom();
    };

    /** 展示工具调用完成状态（tool_result） */
    const showToolStatusDone = (payload) => {
        const name = payload.name || 'tool';
        const item = toolStatusItems[name];
        if (!item) return; // 未展示过开始状态，忽略
        const label = '「' + getToolLabel(name) + '」：已完成';
        item.el.classList.add('is-done');
        item.el.classList.remove('is-active');
        item.el.classList.remove('is-error');
        item.el.classList.remove('is-warning');
        item.iconEl.innerHTML = svgIcon('check');
        item.textEl.textContent = label;
        scrollToBottom();
    };

    /** 展示工具调用失败状态（tool_error） */
    const showToolStatusError = (payload) => {
        const name = payload.name || 'tool';
        let item = toolStatusItems[name];
        if (!item) {
            const list = ensureToolStatusList();
            item = { el: null, iconEl: null, textEl: null };
            item.el = document.createElement('div');
            item.el.className = 'ai-tool-status-item';
            item.el.dataset.name = name;
            item.iconEl = document.createElement('span');
            item.iconEl.className = 'ai-tool-status-icon';
            item.textEl = document.createElement('span');
            item.textEl.className = 'ai-tool-status-text';
            item.el.appendChild(item.iconEl);
            item.el.appendChild(item.textEl);
            list.appendChild(item.el);
            toolStatusItems[name] = item;
        }
        const label = '「' + getToolLabel(name) + '」：失败';
        const reason = payload.result ? String(payload.result) : '';
        const fullLabel = reason ? label + '：' + (reason.length > 40 ? reason.slice(0, 40) + '…' : reason) : label;
        item.el.classList.remove('is-done');
        item.el.classList.add('is-error');
        item.el.classList.remove('is-active');
        item.el.classList.remove('is-warning');
        item.iconEl.innerHTML = svgIcon('x');
        item.textEl.textContent = fullLabel;
        scrollToBottom();
    };

    /** 展示部分来源失败状态（tool_partial，仅 web_search：成功但部分来源失败） */
    const showToolStatusPartial = (payload) => {
        const name = payload.name || 'tool';
        const item = toolStatusItems[name];
        if (!item) return; // 未展示过开始状态，忽略
        const label = '「' + getToolLabel(name) + '」：部分来源失败';
        const parts = payload.result ? String(payload.result) : '';
        const fullLabel = parts ? label + '：' + parts : label;
        item.el.classList.remove('is-done');
        item.el.classList.remove('is-error');
        item.el.classList.add('is-warning');
        item.el.classList.remove('is-active');
        item.iconEl.innerHTML = svgIcon('alert');
        item.textEl.textContent = fullLabel;
        scrollToBottom();
    };

    const unsubToolStatus = window.runtime.EventsOn('ai:tool-status', (streamGen, data) => {
        if (!isAgentFlow) return; // 仅 Agent 流展示工具状态
        if (streamGen !== myGen) return; // 属于旧流, 丢弃
        let payload = null;
        try {
            payload = typeof data === 'string' ? JSON.parse(data) : data;
        } catch (_) { return; }
        if (!payload || !payload.action) return;
        streamToolRecords.push(payload); // 收集工具调用记录（tool_start / tool_result），供落库 tool_calls
        if (payload.action === 'tool_start') {
            showToolStatusStart(payload);
        } else if (payload.action === 'tool_result') {
            showToolStatusDone(payload);
        } else if (payload.action === 'tool_error') {
            showToolStatusError(payload);
        } else if (payload.action === 'tool_partial') {
            showToolStatusPartial(payload);
        }
    });
    unsubs.push(unsubToolStatus);

    // ── Agent 反向提问（ai:ask-user） ──
    // 模型调用 ask_user 工具时后端发射该事件（tool_start 之后），
    // 负载为 JSON 字符串 {"question": "...", "options": [...], "selection": "single"|"multiple"}。
    // 本轮 ReAct 循环在 ask_user 工具处阻塞等待：AI 消息不结束、气泡保持流式状态，
    // 在输入区上方渲染反问面板；用户点击选择/输入答案后经 AnswerAskUser 同轮投递，
    // 同一气泡继续流式输出直至最终回答（不落库为新用户消息、不开新流）。
    const unsubAskUser = window.runtime.EventsOn('ai:ask-user', (streamGen, data) => {
        if (!isAgentFlow) return;
        if (streamGen !== myGen) return; // 属于旧流, 丢弃
        let payload = null;
        try { payload = typeof data === 'string' ? JSON.parse(data) : data; } catch (_) { return; }
        if (!payload || !payload.question) return;
        // 模型未在正文输出问句（正文为空）时，气泡内展示等待提示
        if (!hasReceivedChunk) {
            contentDiv.innerHTML = '';
            contentDiv.appendChild(createWaitingHint());
        }
        showAskPanel(payload.question, Array.isArray(payload.options) ? payload.options : [], payload.selection);
    });
    unsubs.push(unsubAskUser);

    // ── Agent 模式结构化结果回传（ai:agent-result，先于 stream-done 到达） ──
    // 后端在流结束后把搜索来源 / 召回卡片 / 工具调用链 / 思考链 一并回传，
    // 填充 streamSearchSources / recallCards / streamToolRecords，
    // 使 stream-done 的 chatHistory.push 与气泡渲染拿到与历史加载一致的数据。
    const unsubAgentResult = window.runtime.EventsOn('ai:agent-result', (evtGen, searchSourcesJSON, recallCardsJSON, toolCallsJSON, reasoningContent) => {
        if (evtGen !== myGen) return; // 属于旧流, 丢弃
        if (searchSourcesJSON) {
            try { streamSearchSources = JSON.parse(searchSourcesJSON); } catch (_) {}
        }
        if (recallCardsJSON) {
            try { recallCards = JSON.parse(recallCardsJSON); } catch (_) {}
        }
        if (toolCallsJSON) {
            try { streamToolRecords = JSON.parse(toolCallsJSON); } catch (_) {}
        }
        // 思考链兜底：流式 ai:stream-thinking 未触发时，用后端汇总的完整思考链补渲染
        if (reasoningContent && !streamingThinking) {
            streamingThinking = reasoningContent;
            appendThinkingChunk(reasoningContent);
        }
    });
    unsubs.push(unsubAgentResult);

    const unsubDone = window.runtime.EventsOn('ai:stream-done', async (streamGen, fullContent, elapsedThinking, elapsedTotal, totalTokens, userTokens, assistantTokens, userMsgID, assistantMsgID) => {
        if (streamGen !== myGen) return; // 属于旧流, 丢弃
        stopThinkingTimer(0); // 清理计时器, 摘要已在 chunk 中更新
        unsubs.forEach(fn => fn());
        isStreaming = false;
        window.__aiStreaming = false;
        hideAskPanel(); // 防御性收起反问面板（正常流程面板已在提交答案时收起）

        // 恢复发送按钮, 隐藏停止按钮
        if (stopBtnEl) stopBtnEl.style.display = 'none';
        if (sendBtnEl) sendBtnEl.style.display = '';
        if (polishBtn) polishBtn.disabled = !(inputEl && inputEl.value.trim().length > 0);

        if (!hasReceivedChunk) contentDiv.innerHTML = '';
        if (streamingContent === '' && fullContent) streamingContent = fullContent;

        let finalContent = streamingContent || fullContent;
        renderMarkdown(contentDiv, finalContent);

        // 后端是否真正保存了 assistant 消息：停止按钮/会话释放等取消路径不落库
        //（assistantMsgID=0）。据此区分"取消"与"正常完成（含空回复）"，避免把
        // 取消轮次的半截文本写入 chatHistory 造成幽灵条目/误弹历史记录。
        const cancelled = !assistantMsgID;

        // 空回复：通知用户，不保存到数据库（用户主动取消不弹通知）
        const isEmptyMsg = !finalContent || !finalContent.trim();
        if (isEmptyMsg) {
            if (isStreaming) {
                window.showNotification('AI 未返回内容，请尝试重新生成', 'warning');
            }
        }

        if (thinkingDetails && thinkingContentEl && streamingThinking) {
            const summary = thinkingDetails.querySelector('.thinking-summary');
            if (summary) {
                const text = summary.querySelector('.thinking-text');
                if (text) text.textContent = elapsedThinking > 0 ? '已思考 ' + elapsedThinking.toFixed(1) + ' 秒' : '已思考';
                summary.classList.remove('is-thinking');
            }
            renderMarkdown(thinkingContentEl, streamingThinking);
        }

        // 同步更新 DOM 中的用户消息 token 显示
        if (userMsgID) {
            streamingEl.dataset.msgId = assistantMsgID;
        }
        // 给最后一条用户消息设置 data-msg-id
        if (userMsgID) {
            const userMsgs = messagesEl.querySelectorAll('.ai-msg-user');
            const lastUserEl = userMsgs.length > 0 ? userMsgs[userMsgs.length - 1] : null;
            if (lastUserEl) {
                lastUserEl.dataset.msgId = userMsgID;
                const tokensEl = lastUserEl.querySelector('.user-tokens');
                if (tokensEl) {
                    tokensEl.textContent = userTokens > 0 ? formatTokens(userTokens) + ' tokens' : '';
                    tokensEl.style.display = '';
                }
            }
        }

        if (cancelled) {
            // 本轮被取消（停止/会话释放）：不写入 chatHistory（避免幽灵条目）、
            // 不渲染完成态操作栏；气泡通常已由停止逻辑移除，兜底移除仍挂在
            // DOM 中的流式气泡（如后端经 ReleaseSession 独立取消时）
            if (streamingEl && streamingEl.parentNode) {
                streamingEl.parentNode.removeChild(streamingEl);
            }
        } else {
            chatHistory.push({ id: assistantMsgID, role: 'assistant', content: finalContent, tokens: assistantTokens, reasoning_content: streamingThinking || '', thinking_elapsed: elapsedThinking || 0, total_elapsed: elapsedTotal || 0, search_sources: streamSearchSources ? JSON.stringify(streamSearchSources) : null, recall_cards: recallCards ? JSON.stringify(recallCards) : null, tool_calls: streamToolRecords.length > 0 ? JSON.stringify(streamToolRecords) : null });
            updateContextSize();
            streamingEl.appendChild(createMsgActions(finalContent, 'assistant', elapsedTotal, assistantTokens));
            bindMsgContextMenu(streamingEl, finalContent, 'assistant');

            // 空回复：回退 chatHistory，移除 DOM 气泡
            if (isEmptyMsg) {
                chatHistory.pop();
                if (streamingEl && streamingEl.parentNode) {
                    streamingEl.parentNode.removeChild(streamingEl);
                }
            }

            // 展示搜索来源折叠面板
            if (streamSearchSources && streamSearchSources.length > 0) {
                var actionsEl = streamingEl.querySelector('.ai-msg-actions');
                if (actionsEl) {
                    renderSearchSources(streamingEl, streamSearchSources);
                    streamingEl.insertBefore(streamingEl.lastChild, actionsEl);
                } else {
                    renderSearchSources(streamingEl, streamSearchSources);
                }
            }

            // 展示卡片召回折叠面板
            if (recallCards && recallCards.length > 0) {
                var actionsEl = streamingEl.querySelector('.ai-msg-actions');
                if (actionsEl) {
                    renderRecallCards(streamingEl, recallCards);
                    streamingEl.insertBefore(streamingEl.lastChild, actionsEl);
                } else {
                    renderRecallCards(streamingEl, recallCards);
                }
            }

            // 工具调用链：移除上方实时工具条，折叠为正文下方附录（与来源/卡片一致插到 actions 前）
            if (streamToolRecords.length > 0) {
                if (toolStatusListEl && toolStatusListEl.parentNode) {
                    toolStatusListEl.parentNode.removeChild(toolStatusListEl);
                }
                renderToolCalls(streamingEl, streamToolRecords);
                var actionsEl = streamingEl.querySelector('.ai-msg-actions');
                if (actionsEl) streamingEl.insertBefore(streamingEl.lastChild, actionsEl);
            }
        }

        scrollToBottom();

        // 发送完成, 清理追问引用
        followUpRef = '';
        const followUpBar = document.getElementById('aiChatFollowUpBar');
        if (followUpBar) followUpBar.style.display = 'none';

        // 流式完成后刷新会话列表（更新标题等）
        await loadSessionList();
        updateChatTitle();
    });
    unsubs.push(unsubDone);

    const unsubError = window.runtime.EventsOn('ai:stream-error', (streamGen, err, userTokens) => {
        if (streamGen !== myGen) return; // 属于旧流, 丢弃
        stopThinkingTimer(0); // 清理计时器
        unsubs.forEach(fn => fn());
        isStreaming = false;
        window.__aiStreaming = false;
        hideAskPanel();
        // 恢复发送按钮, 隐藏停止按钮
        if (stopBtnEl) stopBtnEl.style.display = 'none';
        if (sendBtnEl) sendBtnEl.style.display = '';
        if (polishBtn) polishBtn.disabled = !(inputEl && inputEl.value.trim().length > 0);
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();

        // 尝试解析结构化错误 JSON，仅通过通知展示，不再创建无菜单的错误气泡
        try {
            const errData = JSON.parse(err);
            if (errData.user_msg) {
                window.showNotification(errData.user_msg, 'error', 5000);
            } else {
                window.showNotification(err, 'error', 5000);
            }
        } catch (_) {
            window.showNotification(err, 'error', 5000);
        }

        // 出错也清理追问引用
        followUpRef = '';
        const fb = document.getElementById('aiChatFollowUpBar');
        if (fb) fb.style.display = 'none';

        // 更新用户消息的 token 显示（后端已计算并保存到 DB，通过 userTokens 返回）
        if (userTokens !== undefined) {
            const lastUserMsgEl = messagesInnerEl.querySelector('.ai-msg-user:last-child');
            if (lastUserMsgEl) {
                const tokensEl = lastUserMsgEl.querySelector('.user-tokens');
                if (tokensEl && userTokens > 0) {
                    tokensEl.textContent = formatTokens(userTokens) + ' tokens';
                }
            }
        }
        // 刷新会话 token 显示
        updateContextSize();
    });
    unsubs.push(unsubError);

    try {
        const skillIds = Object.entries(activeSkills).map(([id, config]) => {
            if (id === 'translate') {
                return `skill_translate:${config.source || 'english'}:${config.target || 'chinese'}`;
            }
            return 'skill_' + id;
        });
        const searchSourcesArray = Array.from(searchSources);
        const refNoteIDs = referencedNotes.map(n => n.id);
        const roleNoteIDs = roleplayNotes.map(n => n.id);
        if (isAgentFlow) {
            // Agent 模式：调用 Agent 流式接口（联网搜索/卡片召回由 Agent 工具执行，无对应开关参数）
            // 参数顺序按后端签名：streamGen, sessionID, userText, thinkingEnabled, skillIds,
            // referencedNoteIDs, roleplayNoteIDs, followUpRefContent, uploadedFiles, recallNotebookIDs, userMsgID
            window.go.main.App.CallAIAgentStream(myGen, activeSessionId, userText, enableThinking, skillIds, refNoteIDs, roleNoteIDs, followUpRef, uploadedFiles, Array.from(recallNotebookIds), userMsgID);
        } else if (isRegenerate) {
            window.go.main.App.CallAIStreamRegenerate(myGen, activeSessionId, enableThinking, searchSourcesArray, enableCardRecall, Array.from(recallNotebookIds), skillIds, refNoteIDs, roleNoteIDs, followUpRef, uploadedFiles);
        } else {
            window.go.main.App.CallAIStream(myGen, activeSessionId, userText, enableThinking, searchSourcesArray, enableCardRecall, Array.from(recallNotebookIds), skillIds, refNoteIDs, roleNoteIDs, followUpRef, uploadedFiles, userMsgID);
        }
    } catch (e) {
        unsubs.forEach(fn => fn());
        isStreaming = false;
        window.__aiStreaming = false;
        // 恢复发送按钮, 隐藏停止按钮
        if (stopBtnEl) stopBtnEl.style.display = 'none';
        if (sendBtnEl) sendBtnEl.style.display = '';
        if (polishBtn) polishBtn.disabled = !(inputEl && inputEl.value.trim().length > 0);
        if (streamingEl && streamingEl.parentNode) streamingEl.remove();
        try {
            const errData = JSON.parse(e.message || e);
            if (errData.user_msg) {
                window.showNotification(errData.user_msg, 'error', 5000);
            } else {
                window.showNotification('流式调用失败: ' + (e.message || e), 'error', 5000);
            }
        } catch (_) {
            window.showNotification('流式调用失败: ' + (e.message || e), 'error', 5000);
        }
    }
}

/* ── 渲染与 UI ── */

/**
 * 渐进式延迟处理代码块语法高亮，每批最多 8ms 后 yield
 */
function deferHighlightBlocks(el) {
    const blocks = [...el.querySelectorAll('pre code[class*="language-"]')].filter(
        b => !b.classList.contains('language-mermaid')
    );
    if (!blocks.length) return;
    let index = 0;
    const schedule = () => {
        const deadline = Date.now() + 8;
        while (index < blocks.length && Date.now() < deadline) {
            try { hljs.highlightElement(blocks[index]); } catch (_) {}
            index++;
        }
        if (index < blocks.length) {
            requestIdleCallback(schedule, { timeout: 100 });
        }
    };
    requestIdleCallback(schedule, { timeout: 100 });
}

/**
 * 渲染 Markdown + 代码高亮
 */
function renderMarkdown(el, content, deferHighlight) {
    el.innerHTML = marked.parse(content);

    if (deferHighlight) {
        // 延迟高亮：渲染后渐进式处理代码高亮，不阻塞首次渲染
        deferHighlightBlocks(el);
    } else {
        // 立即高亮（流式回复等需要即时展示的场景，跳过 Mermaid 代码块）
        el.querySelectorAll('pre code[class*="language-"]').forEach((block) => {
            if (block.classList.contains('language-mermaid')) return;
            try { hljs.highlightElement(block); } catch (_) {}
        });
    }

    el.querySelectorAll('pre').forEach((pre) => {
        // 避免重复包装
        if (pre.parentNode.classList.contains('pre-wrapper')) return;

        // 阻止代码块滚动事件冒泡到父容器 (防止触发 .ai-chat-messages 的滚动条自动显隐) 
        pre.addEventListener('scroll', (e) => e.stopPropagation(), { passive: true });

        const code = pre.querySelector('code');
        if (!code) return;

        // 语言信息
        const langClass = Array.from(code.classList).find(cls => cls.startsWith('language-'));
        const lang = langClass ? langClass.replace('language-', '') : '';

        // 始终用 pre-wrapper 包裹，为绝对定位的复制按钮提供非滚动定位容器
        const wrapper = document.createElement('div');
        wrapper.className = 'pre-wrapper';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);

        // 复制按钮 (放在 pre-wrapper 内、pre 外部，避免随 pre 滚动)
        const copyBtn = document.createElement('button');
        const isSingleLine = code && !code.textContent.trim().includes('\n');
        copyBtn.className = 'copy-code-btn' + (isSingleLine ? ' copy-code-btn--single' : '');
        copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> 复制`;
        copyBtn.title = '复制代码';
        copyBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(code.textContent);
                // 先触发渲染按钮滑出动画，再变"已复制"
                wrapper.classList.add('copying');
                await new Promise(r => setTimeout(r, 200));
                copyBtn.classList.add('copied');
                copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg> 已复制`;
                wrapper.classList.remove('copying');
                setTimeout(() => {
                    copyBtn.classList.remove('copied');
                    copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> 复制`;
                }, 1500);
            } catch (_) {
                copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg> 失败`;
                setTimeout(() => {
                    copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> 复制`;
                }, 1500);
            }
        });
        wrapper.appendChild(copyBtn);

        // 语言标签
        if (lang) {
            const badge = document.createElement('span');
            badge.className = 'code-lang-badge';
            badge.textContent = lang.charAt(0).toUpperCase() + lang.slice(1);
            wrapper.appendChild(badge);
        }
    });

    // 表格复制按钮
    el.querySelectorAll('table').forEach((table) => {
        const lastTh = table.querySelector('tr:first-child th:last-child');
        if (!lastTh) return;
        if (lastTh.querySelector('.table-copy-btn')) return;

        const copyBtn = document.createElement('button');
        copyBtn.className = 'table-copy-btn';
        copyBtn.innerHTML = SVGS.copy + ' 复制';
        copyBtn.title = '复制表格';
        copyBtn.addEventListener('click', async () => {
            try {
                const md = tableToMarkdown(table);
                await navigator.clipboard.writeText(md);
                copyBtn.classList.add('copied');
                copyBtn.innerHTML = SVGS.checkmark + ' 已复制';
                setTimeout(() => {
                    copyBtn.classList.remove('copied');
                    copyBtn.innerHTML = SVGS.copy + ' 复制';
                }, 1500);
            } catch (_) {
                copyBtn.innerHTML = SVGS.xmark + ' 复制失败';
                setTimeout(() => { copyBtn.innerHTML = SVGS.copy + ' 复制'; }, 1000);
            }
        });
        lastTh.appendChild(copyBtn);
    });

    // 为 Mermaid 代码块设置交互结构（默认显示源码，不自动渲染）
    if (window.renderMermaidBlocks) {
        window.renderMermaidBlocks(el);
    }
}

// ── AI 消息内嵌 SVG 图标（Feather 线条风格，随 currentColor 着色） ──
function svgIcon(name, size = 14) {
    const paths = {
        search: '<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>',
        check: '<polyline points="20 6 9 17 4 12"/>',
        x: '<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>',
        alert: '<path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>',
        brain: '<path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 4.44-2.04z"/><path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-4.44-2.04z"/>',
        wrench: '<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>',
    };
    const p = paths[name];
    if (!p) return '';
    return '<svg class="ai-inline-icon' + (name === 'brain' ? ' ai-brain-icon' : '') + '" width="' + size + '" height="' + size + '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' + p + '</svg>';
}

/**
 * 添加消息气泡 (不含操作按钮, 调用方自行添加) 
 * @param {string} content - 消息内容
 * @param {'user'|'assistant'} role - 角色
 * @param {string} [reasoningContent] - 思维链内容 (可选) 
 * @param {number} [thinkingElapsed] - 思考耗时
 * @param {number} [totalElapsed] - 总耗时
 */
function addMessage(content, role, reasoningContent, thinkingElapsed, totalElapsed, tokens, msgId, searchSources, recallCards, toolCalls, skipScroll = false, deferHighlight = false) {
    const el = document.createElement('div');
    el.className = 'ai-msg ' + (role === 'user' ? 'ai-msg-user' : 'ai-msg-assistant');
    // 仅流式消息播放入场动画，切换历史会话时跳过
    if (!skipScroll) el.classList.add('ai-msg-enter-anim');
    if (msgId) el.dataset.msgId = msgId;

    // 如果有思维链内容, 先渲染可折叠思考区域
    if (role === 'assistant' && reasoningContent) {
        const details = document.createElement('details');
        details.className = 'thinking-details';
        details.open = localStorage.getItem('ai_cot_collapsed') !== 'true';
        details.addEventListener('toggle', () => {
            localStorage.setItem('ai_cot_collapsed', details.open ? 'false' : 'true');
        });
        const summary = document.createElement('summary');
        summary.className = 'thinking-summary';
        summary.innerHTML = svgIcon('brain') + '<span class="thinking-text">' + (thinkingElapsed > 0 ? '已思考 ' + thinkingElapsed.toFixed(1) + ' 秒' : '已思考') + '</span>';
        details.appendChild(summary);
        const thinkingEl = document.createElement('div');
        thinkingEl.className = 'thinking-content';
        renderMarkdown(thinkingEl, reasoningContent, deferHighlight);
        details.appendChild(thinkingEl);
        el.appendChild(details);
    }

    const contentEl = document.createElement('div');
    contentEl.className = 'msg-content';

    if (role === 'assistant') {
        renderMarkdown(contentEl, content, deferHighlight);
    } else {
        contentEl.textContent = content;
    }
    el.appendChild(contentEl);

    // 显示总耗时 + token 数（仅历史 AI 消息）
    if (totalElapsed > 0) {
        const actionsEl = document.createElement('div');
        actionsEl.className = 'ai-msg-actions';
        const timeEl = document.createElement('span');
        timeEl.className = 'ai-msg-time';
        let timeText = '⏱ ' + totalElapsed.toFixed(1) + ' 秒';
        if (tokens > 0) timeText += ' · ' + formatTokens(tokens) + ' tokens';
        timeEl.textContent = timeText;
        actionsEl.appendChild(timeEl);
        el.appendChild(actionsEl);
    }

    // 解析 JSON 字符串参数
    if (typeof searchSources === 'string') {
        try { searchSources = JSON.parse(searchSources); } catch (_) { searchSources = null; }
    }
    if (typeof recallCards === 'string') {
        try { recallCards = JSON.parse(recallCards); } catch (_) { recallCards = null; }
    }
    if (typeof toolCalls === 'string') {
        try { toolCalls = JSON.parse(toolCalls); } catch (_) { toolCalls = null; }
    }

    // 渲染搜索来源（单一来源→卡片，多个来源→折叠面板）
    if (role === 'assistant' && searchSources && searchSources.length > 0) {
        renderSearchSources(el, searchSources);
    }

    // 渲染召回卡片折叠面板
    if (role === 'assistant' && recallCards && recallCards.length > 0) {
        renderRecallCards(el, recallCards);
    }

    // 渲染 Agent 工具调用链折叠面板（历史消息回放，复用工具状态样式）
    if (role === 'assistant' && toolCalls && toolCalls.length > 0) {
        renderToolCalls(el, toolCalls);
    }

    messagesInnerEl.appendChild(el);
    if (!skipScroll) scrollToBottom();
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
 * 创建 Agent 反问等待提示：模型触发 ask_user 后、用户回答前，
 * 在气泡正文区展示"等待你的回答…"（同轮续答期间 AI 消息不结束）。
 */
function createWaitingHint() {
    const el = document.createElement('span');
    el.className = 'ai-msg-waiting-hint';
    el.innerHTML = '<span class="ai-msg-waiting-dot"></span>等待你的回答…';
    return el;
}

/**
 * 来源名称映射
 */
const sourceLabels = {
    'tavily': 'Tavily搜索',
    'zhihu_search': '知乎搜索',
    'zhihu_global': '全网搜索',
};

/**
 * 创建卡片召回状态指示器：放大镜图标 + 文案（复用搜索指示器布局，语义=检索本地笔记/知识库）
 * 放大镜不参与旋转动画（与联网搜索地球图标区分），见 ai-chat.css 中 data-status="recall" 覆盖规则
 * @returns {HTMLDivElement}
 */
function createRecallIndicator() {
    // 放大镜 SVG（Lucide Search 风格，静止展示不旋转）
    var searchSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>';
    var el = document.createElement('div');
    el.className = 'ai-search-indicator';
    el.dataset.status = 'recall';
    var bar = document.createElement('div');
    bar.className = 'ai-search-bar';
    bar.innerHTML = searchSvg + '<span class="ai-search-text">正在检索本地笔记...</span>';
    el.appendChild(bar);
    return el;
}

/**
 * 创建联网搜索状态指示器（带关键词下拉菜单）
 * @param {'refining'|'searching'} status - 搜索阶段
 * @param {string} [keywords=''] - 精炼后的搜索关键词
 * @returns {HTMLDivElement}
 */
function createSearchIndicator(status, keywords) {
    // 地球 SVG（与简易版一致）
    var earthSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>';

    // 下拉箭头 SVG
    var arrowSvg = '<svg class="ai-search-arrow" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>';

    var el = document.createElement('div');
    el.className = 'ai-search-indicator';
    el.dataset.status = status;

    if (status === 'refining') {
        // 精炼阶段：简洁展示，不可点击，无下拉
        var bar = document.createElement('div');
        bar.className = 'ai-search-bar';
        bar.innerHTML = earthSvg + '<span class="ai-search-text">正在优化输入...</span>';
        el.appendChild(bar);
        return el;
    }

    // 搜索阶段：可点击 bar + 下拉菜单
    el.dataset.open = 'false';

    var bar = document.createElement('div');
    bar.className = 'ai-search-bar';

    bar.innerHTML = earthSvg + '<span class="ai-search-text">联网搜索中</span>' + arrowSvg;
    el.appendChild(bar);

    // 下拉菜单
    var dropdown = document.createElement('div');
    dropdown.className = 'ai-search-dropdown';

    if (keywords) {
        var kwList = keywords.split(/\s+/).filter(Boolean);
        if (kwList.length > 0) {
            var label = document.createElement('div');
            label.className = 'ai-search-dropdown-label';
            label.textContent = '精炼搜索词';
            dropdown.appendChild(label);

            var kwContainer = document.createElement('div');
            kwContainer.className = 'ai-search-keywords';
            kwList.forEach(function (kw) {
                var tag = document.createElement('span');
                tag.className = 'ai-search-keyword-tag';
                tag.textContent = kw;
                kwContainer.appendChild(tag);
            });
            dropdown.appendChild(kwContainer);
        } else {
            dropdown.innerHTML = '<div class="ai-search-empty-keywords">暂无精炼关键词</div>';
        }
    } else {
        dropdown.innerHTML = '<div class="ai-search-empty-keywords">暂无精炼关键词</div>';
    }

    el.appendChild(dropdown);

    // 点击 bar 切换展开/收起
    bar.addEventListener('click', function (e) {
        e.stopPropagation();
        var isOpen = el.dataset.open === 'true';
        el.dataset.open = isOpen ? 'false' : 'true';
    });

    // 点击外部收起（自清理：元素脱离 DOM 后自动移除监听器）
    var closeHandler = function (e) {
        if (!el.isConnected) {
            document.removeEventListener('click', closeHandler);
            return;
        }
        if (!el.contains(e.target)) {
            el.dataset.open = 'false';
        }
    };
    document.addEventListener('click', closeHandler);

    return el;
}

/**
 * 滚动到底部（临时禁用 smooth scroll 避免动画）
 */
function scrollToBottom() {
    requestAnimationFrame(() => {
        const orig = messagesEl.style.scrollBehavior;
        messagesEl.style.scrollBehavior = 'auto';
        messagesEl.scrollTop = messagesEl.scrollHeight;
        messagesEl.style.scrollBehavior = orig;
    });
}

/**
 * 重置 AI 聊天前端状态（恢复出厂/还原备份后调用）
 * 数据库已被替换，旧的会话缓存（activeSessionId / sessions / chatHistory）已失效，
 * 必须全部清空并清掉消息 DOM，下次激活视图时才会从后端重新加载。
 * 注意：恢复出厂流程用 innerHTML='' 清空 #aiChatMessages 会连带删除
 * .ai-chat-messages-inner 子元素，导致 messagesInnerEl 引用悬空（消息渲染进
 * 孤儿节点而不可见），因此这里必须重新查询/重建该引用。
 */
export function resetAIChatState() {
    chatHistory = [];
    sessions = [];
    activeSessionId = null;
    _oldestMsgId = 0;
    _loadingMore = false;
    hideAskPanel(); // 重置状态时收起 Agent 反问面板
    if (messagesEl) {
        messagesInnerEl = messagesEl.querySelector('.ai-chat-messages-inner');
        if (!messagesInnerEl) {
            messagesInnerEl = document.createElement('div');
            messagesInnerEl.className = 'ai-chat-messages-inner';
            messagesEl.appendChild(messagesInnerEl);
        }
        messagesInnerEl.innerHTML = '';
    }
    if (sessionListEl) sessionListEl.innerHTML = '';
}

/**
 * 视图激活时调用
 */
export async function onAIChatViewActivated() {
    if (!messagesEl) return;

    // 从 DOM 重新同步工具栏状态变量（用户可能在设置页更改了这些开关）
    syncToolbarState();

    try {
        const cfg = await window.go.main.App.GetAIConfig();
        const hasRequired = !!(cfg.base_url && cfg.api_key && cfg.model);
        if (!hasRequired) {
            showEmptyState();
            return;
        }
        hideEmptyState();
        loadModelSelector(cfg);
        await loadSessionList();

        // 没有激活会话时，恢复上次使用的会话；无历史会话时才新建
        if (activeSessionId === null) {
            if (sessions.length > 0) {
                // 在所有会话中选 updated_at 最新的（忽略置顶优先），确保加载最后使用的会话
                const mostRecent = sessions.reduce((a, b) =>
                    new Date(a.updated_at) > new Date(b.updated_at) ? a : b
                );
                await switchSession(mostRecent.id);
            } else {
                await createSession();
            }
        } else if (chatHistory.length === 0) {
            showWelcome();
        }

        // 视图入场动画完成后聚焦输入框
        setTimeout(() => inputEl?.focus(), 100);
    } catch (_) {
        showEmptyState();
    }
}

/**
 * 从 DOM 读取工具栏 toggle 状态，更新模块级变量
 */
function syncToolbarState() {
    enableThinking = document.getElementById('aiChatSearchToggle')?.classList.contains('active') || false;
    // 获取密钥配置状态（从设置页的 input 读取）
    const zhihuSecret = document.getElementById('aiZhihuAccessSecret')?.value || '';
    const tavilyKey = document.getElementById('aiTavilyApiKey')?.value || '';
    const hasZhihuSecret = !!(zhihuSecret && zhihuSecret.trim());
    const hasTavilyKey = !!(tavilyKey && tavilyKey.trim());
    
    // 从复选框读取搜索源状态
    searchSources = new Set();
    ['aiChatZhihuSearch', 'aiChatZhihuGlobalSearch', 'aiChatTavilySearch'].forEach(id => {
        const cb = document.getElementById(id);
        if (!cb) return;
        const label = cb.closest('.ai-chat-search-source-item');
        // 判断是否需要禁用
        let needsDisabled = false;
        if (cb.dataset.source === 'zhihu_search' || cb.dataset.source === 'zhihu_global') {
            needsDisabled = !hasZhihuSecret;
        } else if (cb.dataset.source === 'tavily') {
            needsDisabled = !hasTavilyKey;
        }
        if (needsDisabled) {
            cb.disabled = true;
            if (label) label.classList.add('disabled');
            cb.checked = false;
        } else {
            cb.disabled = false;
            if (label) label.classList.remove('disabled');
            if (cb.checked) {
                searchSources.add(cb.dataset.source);
            }
        }
    });
    if (searchSourcesBtn) {
        searchSourcesBtn.classList.toggle('active', searchSources.size > 0);
    }
    enableCardRecall = document.getElementById('aiChatCardRecallToggle')?.classList.contains('active') || false;
}

function showEmptyState() {
    if (!emptyEl) return;
    emptyEl.style.display = '';
    if (messagesEl) messagesEl.style.display = 'none';
    if (inputAreaEl) inputAreaEl.style.display = 'none';
    if (clearBtnEl) clearBtnEl.style.display = 'none';
    // 侧栏仍可见但禁用操作
    if (sessionNewBtnEl) sessionNewBtnEl.style.display = 'none';
    // 重置标题和 Token 显示
    const titleEl = document.getElementById('aiChatTitle');
    if (titleEl) titleEl.textContent = 'AI 助手';
    if (contextSizeEl) {
        contextSizeEl.textContent = '';
        contextSizeEl.style.display = 'none';
    }
    // 清空内存中的旧会话数据（数据库已重置，这些数据已失效）
    chatHistory = [];
    sessions = [];
    activeSessionId = null;
}

/**
 * 构建并显示统一的会话操作菜单（右击/更多按钮共用）
 * @param {Object} s - 会话对象
 * @param {HTMLElement} item - 会话条目元素
 * @param {number} left - 菜单位置 left
 * @param {number} top - 菜单位置 top
 */
function showSessionMenu(s, item, left, top) {
    // 关闭其他菜单
    if (sessionMoreMenuTarget && sessionMoreMenuTarget !== item) {
        sessionMoreMenu.classList.remove('active');
    }

    // 动态构建菜单内容
    sessionMoreMenu.innerHTML = '';

    // 置顶/取消置顶
    const pinItem = document.createElement('div');
    pinItem.className = 'ai-session-more-menu-item';
    pinItem.dataset.action = 'pin';
    pinItem.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="17" x2="12" y2="22"/><path d="M5 17h14v-1.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1v4.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24Z"/></svg>'
        + (s.is_pinned ? ' 取消置顶' : ' 置顶');
    sessionMoreMenu.appendChild(pinItem);

    // 重命名
    const renameItem = document.createElement('div');
    renameItem.className = 'ai-session-more-menu-item';
    renameItem.dataset.action = 'rename';
    renameItem.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg> 重命名';
    sessionMoreMenu.appendChild(renameItem);

    // 导出
    const exportItem = document.createElement('div');
    exportItem.className = 'ai-session-more-menu-item';
    exportItem.dataset.action = 'export';
    exportItem.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg> 导出';
    sessionMoreMenu.appendChild(exportItem);

    // 分隔线
    const divider = document.createElement('div');
    divider.className = 'ai-session-more-menu-divider';
    sessionMoreMenu.appendChild(divider);

    // 删除会话
    const delItem = document.createElement('div');
    delItem.className = 'ai-session-more-menu-item danger';
    delItem.dataset.action = 'delete';
    delItem.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg> 删除会话';
    sessionMoreMenu.appendChild(delItem);

    // 菜单项点击事件
    pinItem.addEventListener('click', async (pe) => {
        pe.stopPropagation();
        sessionMoreMenu.classList.remove('active');
        sessionMoreMenuTarget = null;
        try {
            await window.go.main.App.TogglePinAISession(s.id);
        } catch (_) { /* 忽略 */ }
        await loadSessionList();
    });

    renameItem.addEventListener('click', (re) => {
        re.stopPropagation();
        sessionMoreMenu.classList.remove('active');
        sessionMoreMenuTarget = null;
        // 关闭菜单后触发该会话标题的内联编辑
        const titleEl = item.querySelector('.ai-session-item-title');
        if (titleEl) startInlineEdit(titleEl, s.id);
    });

    exportItem.addEventListener('click', async (ex) => {
        ex.stopPropagation();
        sessionMoreMenu.classList.remove('active');
        sessionMoreMenuTarget = null;
        try {
            const result = await window.go.main.App.ExportAISessionAsMarkdown(s.id);
            if (result && result !== '已取消') {
                window.showNotification?.(result, 'success');
            }
        } catch (e) {
            window.showNotification?.('导出失败: ' + (e.message || e), 'error');
        }
    });

    delItem.addEventListener('click', async (de) => {
        de.stopPropagation();
        sessionMoreMenu.classList.remove('active');
        sessionMoreMenuTarget = null;
        if (isStreaming) return;
        const confirmed = await window.showConfirmDialog('确定删除此会话吗？');
        if (!confirmed) return;

        try {
            await window.go.main.App.DeleteAISession(s.id);
        } catch (_) { /* 忽略 */ }

        // 如果删除的是当前会话, 切换到最近会话或新建
        if (s.id === activeSessionId) {
            activeSessionId = null;
            chatHistory = [];
            messagesInnerEl.innerHTML = '';
        }

        await loadSessionList();
        // 如果删除后没有会话了, 自动新建一个
        if (sessions.length === 0) {
            await createSession();
        } else if (activeSessionId === null) {
            // 切换到最近一个会话
            switchSession(sessions[0].id);
        }
    });

    // 定位菜单
    const menuWidth = 160;
    const menuHeight = 80; // 估算
    if (left + menuWidth > window.innerWidth) {
        left = window.innerWidth - menuWidth - 8;
    }
    if (top + menuHeight > window.innerHeight) {
        top = window.innerHeight - menuHeight - 8;
    }
    if (left < 4) left = 4;
    if (top < 4) top = 4;
    sessionMoreMenu.style.left = left + 'px';
    sessionMoreMenu.style.top = top + 'px';

    sessionMoreMenu.classList.add('active');
    sessionMoreMenuTarget = item;
}

/**
 * 关闭 AI 消息右键菜单
 */
function closeAiMsgContextMenu() {
    if (aiMsgContextMenu) {
        aiMsgContextMenu.classList.remove('active');
        aiMsgContextMenu.innerHTML = '';
    }
    _contextMsgContent = '';
    _contextMsgRole = '';
    _contextMsgEl = null;
}

/**
 * 显示 AI 消息右键菜单 — 根据角色动态生成菜单项
 */
function showAiMsgContextMenu(event, content, role, msgEl) {
    event.preventDefault();
    event.stopPropagation();
    if (!aiMsgContextMenu) return;

    // 先关闭之前的菜单
    closeAiMsgContextMenu();

    // 保存上下文
    _contextMsgContent = content;
    _contextMsgRole = role;
    _contextMsgEl = msgEl;

    // 动态构建菜单项
    const items = [];

    // 复制（所有消息共有）
    items.push({ action: 'copy', label: '复制' });

    if (role === 'user') {
        items.push({ type: 'divider' });
        items.push({ action: 'edit', label: '编辑' });
        items.push({ action: 'resend', label: '重新发送' });
    }

    if (role === 'assistant') {
        items.push({ type: 'divider' });
        items.push({ action: 'save', label: '保存为笔记' });
        items.push({ action: 'regen', label: '重新生成' });
        items.push({ type: 'divider' });
        items.push({ action: 'followUp', label: '追问此条回复' });
    }

    // 删除（所有消息共有）
    items.push({ type: 'divider' });
    items.push({ action: 'delete', label: '删除' });

    // 图标映射
    const actionIcons = {
        copy: COPY_ICON,
        edit: EDIT_ICON,
        resend: RESEND_ICON,
        save: SAVE_ICON,
        regen: REGEN_ICON,
        followUp: FOLLOWUP_ICON,
        delete: DELETE_ICON,
    };

    // 渲染菜单项
    items.forEach(item => {
        if (item.type === 'divider') {
            const divider = document.createElement('div');
            divider.className = 'context-menu-divider';
            aiMsgContextMenu.appendChild(divider);
        } else {
            const menuItem = document.createElement('div');
            menuItem.className = 'context-menu-item';
            menuItem.dataset.action = item.action;
            menuItem.innerHTML = `<span class="ctx-item-icon">${actionIcons[item.action] || ''}</span><span class="ctx-item-label">${item.label}</span>`;
            aiMsgContextMenu.appendChild(menuItem);
        }
    });

    // 定位到鼠标位置，防止溢出视口
    let x = event.clientX;
    let y = event.clientY;
    const menuW = 180;
    const menuH = items.length * 36 + 16;
    if (x + menuW > window.innerWidth) x = window.innerWidth - menuW - 8;
    if (y + menuH > window.innerHeight) y = window.innerHeight - menuH - 8;
    if (x < 8) x = 8;
    if (y < 8) y = 8;
    aiMsgContextMenu.style.left = x + 'px';
    aiMsgContextMenu.style.top = y + 'px';

    aiMsgContextMenu.classList.add('active');
}

/**
 * 为消息气泡绑定右键菜单事件
 */
function bindMsgContextMenu(msgEl, content, role) {
    msgEl.addEventListener('contextmenu', (e) => {
        showAiMsgContextMenu(e, content, role, msgEl);
    });
}

function hideEmptyState() {
    if (!emptyEl) return;
    emptyEl.style.display = 'none';
    if (messagesEl) messagesEl.style.display = '';
    if (inputAreaEl) inputAreaEl.style.display = '';
    if (sessionNewBtnEl) sessionNewBtnEl.style.display = '';
    hideWelcome();
}

let typewriterTimer = null;

/**
 * 显示空对话欢迎语
 */
/**
 * 更新 AI 对话页顶部标题
 */
function updateChatTitle() {
    const titleEl = document.getElementById('aiChatTitle');
    if (!titleEl) return;
    if (activeSessionId !== null) {
        const s = sessions.find(s => s.id === activeSessionId);
        titleEl.textContent = s ? s.title : 'AI 助手';
    } else {
        titleEl.textContent = 'AI 助手';
    }
}

function showWelcome() {
    if (!welcomeEl) return;
    if (emptyEl) emptyEl.style.display = 'none';
    welcomeEl.style.display = '';
    if (messagesEl) messagesEl.style.display = 'none';
    if (inputAreaEl) inputAreaEl.style.display = '';
    updateChatTitle();
    startTypewriter();
}

/**
 * 隐藏空对话欢迎语
 */
function hideWelcome() {
    if (!welcomeEl) return;
    welcomeEl.style.display = 'none';
    if (messagesEl) messagesEl.style.display = '';
    stopTypewriter();
}

/**
 * 打字机效果 :逐字打印 → 暂停 → 逐字擦除 → 循环
 */
function startTypewriter() {
    const el = welcomeEl?.querySelector('.ai-chat-welcome-text');
    if (!el) return;

    const MESSAGES = [
        '有什么我能帮你的吗？',
        '今天想写点什么？',
        '有什么想法，随时告诉我',
        '开始记录你的灵感吧',
        '准备好了就告诉我',
        '随便聊聊也可以',
        // ── 引导对话 ──
        '想聊点什么？',
        '最近有什么新鲜事？',
        '有什么难题需要我帮忙吗？',
        '来吧，说出你的想法',
        '我在听，慢慢说',
        '今天过得怎么样？',
        '从随便一句话开始也行',
        '想到什么就问什么',
        '别客气，尽管开口',
        // ── 引导写作 / 记录 ──
        '把今天的想法记下来吧',
        '有没有想记录的心情？',
        '随手记下此刻的灵感',
        '灵感稍纵即逝，快记下来',
        '给今天留点文字吧',
        '写什么都行，先写下来',
        // ── 欢迎 / 期待 ──
        '很高兴见到你',
        '新的一天，新的记录',
        '想写什么就写什么',
    ];

    let text = MESSAGES[Math.floor(Math.random() * MESSAGES.length)];
    let i = 0, erasing = false;

    function tick() {
        if (!erasing) {
            if (i < text.length) {
                el.textContent = text.substring(0, ++i);
                typewriterTimer = setTimeout(tick, 90);
            } else {
                erasing = true;
                typewriterTimer = setTimeout(tick, 2500);
            }
        } else {
            if (i > 0) {
                el.textContent = text.substring(0, --i);
                typewriterTimer = setTimeout(tick, 40);
            } else {
                // 擦完重新选一条随机消息
                text = MESSAGES[Math.floor(Math.random() * MESSAGES.length)];
                erasing = false;
                typewriterTimer = setTimeout(tick, 1500);
            }
        }
    }

    el.textContent = '';
    i = 0; erasing = false;
    typewriterTimer = setTimeout(tick, 400);
}

function stopTypewriter() {
    if (typewriterTimer !== null) {
        clearTimeout(typewriterTimer);
        typewriterTimer = null;
    }
    const el = welcomeEl?.querySelector('.ai-chat-welcome-text');
    if (el) el.textContent = '';
}

/* ── SVG 图标 ── */
const COPY_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
const REGEN_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>';
const RESEND_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>';
const EDIT_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>';
const SAVE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>';
const FOLLOWUP_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';
const DELETE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>';

// ── 搜索来源 SVG 图标 ──
const SEARCH_SOURCE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/><path d="M11 7v8"/><path d="M7 11h8"/></svg>';
const SATELLITE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h10l-4 8"/><path d="M17 7l4 4"/><path d="M20 11l2 2"/></svg>';
const BOOK_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>';
const GLOBE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>';
const EXTERNAL_ICON = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>';
const CHEVRON_RIGHT_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';

// ── 召回笔记 SVG 图标 ──
const NOTE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>';

/**
 * 创建消息气泡操作按钮
 */
function createMsgActions(content, role, elapsedTotal, tokens) {
    const container = document.createElement('div');
    container.className = 'ai-msg-actions';

    // 耗时标签 - 左侧（仅 AI 消息）
    if (elapsedTotal > 0) {
        const timeEl = document.createElement('span');
        timeEl.className = 'ai-msg-time';
        let timeText = '⏱ ' + elapsedTotal.toFixed(1) + ' 秒';
        if (tokens > 0) timeText += ' · ' + formatTokens(tokens) + ' tokens';
        timeEl.textContent = timeText;
        container.appendChild(timeEl);
    }

    // 用户消息 token 计数（左侧）
    if (role === 'user') {
        const tokensEl = document.createElement('span');
        tokensEl.className = 'user-tokens';
        if (tokens > 0) tokensEl.textContent = formatTokens(tokens) + ' tokens';
        container.appendChild(tokensEl);
    }

    return container;
}

// ── 搜索来源渲染 ──

/**
 * 搜索来源 SVG 图标映射
 */
function getSourceIcon(label) {
    switch (label) {
        case 'tavily': return SATELLITE_ICON;
        case 'zhihu_search': return BOOK_ICON;
        case 'zhihu_global': return GLOBE_ICON;
        default: return SEARCH_SOURCE_ICON;
    }
}

/**
 * 来源标签显示名称映射
 */
function getSourceLabel(label) {
    switch (label) {
        case 'tavily': return 'Tavily';
        case 'zhihu_search': return '知乎搜索';
        case 'zhihu_global': return '全网搜索';
        default: return label;
    }
}

/**
 * 来源分组排序（固定顺序）
 */
var SOURCE_GROUP_ORDER = ['tavily', 'zhihu_search', 'zhihu_global'];

/**
 * 从 URL 中提取域名 + 路径（短路径）用于展示
 */
function extractDomain(urlStr) {
    try {
        var u = new URL(urlStr);
        var path = u.pathname;
        if (path === '/') path = '';
        // 只保留前两段路径
        var parts = path.split('/').filter(Boolean);
        if (parts.length > 2) parts = parts.slice(0, 2);
        var shortPath = parts.length > 0 ? '/' + parts.join('/') : '';
        return u.hostname + shortPath;
    } catch (_) {
        return urlStr;
    }
}

/**
 * 渲染搜索来源 — 统一使用折叠面板展示
 * @param {HTMLElement} el - 插入位置（AI 消息气泡容器）
 * @param {Array} sources - 搜索来源数组 [{title, url, content, source_label}]
 */
function renderSearchSources(el, sources) {
    if (!sources || sources.length === 0) return;
    renderMultiSourcesPanel(el, sources);
}

/**
 * 多个来源 — 自定义折叠面板
 */
function renderMultiSourcesPanel(el, sources) {
    var panel = document.createElement('div');
    panel.className = 'search-sources-panel';

    var header = document.createElement('button');
    header.className = 'search-sources-header';

    var headerIcon = document.createElement('span');
    headerIcon.className = 'search-sources-header-icon';
    headerIcon.innerHTML = SEARCH_SOURCE_ICON;
    header.appendChild(headerIcon);

    var headerText = document.createElement('span');
    headerText.className = 'search-sources-header-text';

    // 计算来源类型数量
    var typeSet = {};
    sources.forEach(function (s) { typeSet[s.source_label || 'unknown'] = true; });
    var typeCount = Object.keys(typeSet).length;
    headerText.textContent = '\u6765\u81EA ' + typeCount + ' \u4E2A\u6765\u6E90\u00B7' + sources.length + ' \u6761\u7ED3\u679C';
    header.appendChild(headerText);

    var arrow = document.createElement('span');
    arrow.className = 'search-sources-header-arrow';
    arrow.innerHTML = CHEVRON_RIGHT_ICON;
    header.appendChild(arrow);

    panel.appendChild(header);

    var body = document.createElement('div');
    body.className = 'search-sources-body';

    // 按 source_label 分组
    var groups = {};
    sources.forEach(function (src) {
        var label = src.source_label || 'unknown';
        if (!groups[label]) groups[label] = [];
        groups[label].push(src);
    });

    var globalIndex = 0;
    SOURCE_GROUP_ORDER.forEach(function (labelKey) {
        var items = groups[labelKey];
        if (!items || items.length === 0) return;

        var group = document.createElement('div');
        group.className = 'search-sources-group';

        var groupHeader = document.createElement('div');
        groupHeader.className = 'search-sources-group-header';

        var groupIcon = document.createElement('span');
        groupIcon.className = 'search-sources-group-icon';
        groupIcon.innerHTML = getSourceIcon(labelKey);
        groupHeader.appendChild(groupIcon);

        var groupLabel = document.createElement('span');
        groupLabel.className = 'search-sources-group-label';
        groupLabel.textContent = getSourceLabel(labelKey);
        groupHeader.appendChild(groupLabel);

        var groupCount = document.createElement('span');
        groupCount.className = 'search-sources-group-count';
        groupCount.textContent = items.length + ' \u6761';
        groupHeader.appendChild(groupCount);

        group.appendChild(groupHeader);

        var itemList = document.createElement('div');
        itemList.className = 'search-sources-group-items';

        items.forEach(function (src) {
            globalIndex++;
            var item = document.createElement('div');
            item.className = 'search-source-group-item';
            item.addEventListener('click', function () {
                window.runtime.BrowserOpenURL(src.url);
            });

            var num = document.createElement('span');
            num.className = 'search-source-item-num';
            num.textContent = globalIndex + '.';
            item.appendChild(num);

            var bodyWrap = document.createElement('div');
            bodyWrap.className = 'search-source-item-body';

            var top = document.createElement('div');
            top.className = 'search-source-item-top';

            var itemTitle = document.createElement('span');
            itemTitle.className = 'search-source-item-title';
            itemTitle.textContent = src.title;
            top.appendChild(itemTitle);

            var itemDomain = document.createElement('span');
            itemDomain.className = 'search-source-item-domain';
            itemDomain.textContent = extractDomain(src.url);
            top.appendChild(itemDomain);

            var itemLink = document.createElement('span');
            itemLink.className = 'search-source-item-link';
            itemLink.innerHTML = EXTERNAL_ICON;
            top.appendChild(itemLink);

            bodyWrap.appendChild(top);

            if (src.content) {
                var snippet = document.createElement('div');
                snippet.className = 'search-source-item-snippet';
                snippet.textContent = src.content;
                bodyWrap.appendChild(snippet);
            }

            item.appendChild(bodyWrap);
            itemList.appendChild(item);
        });

        group.appendChild(itemList);
        body.appendChild(group);
    });

    panel.appendChild(body);
    el.appendChild(panel);

    // 点击 header 切换展开/收起
    header.addEventListener('click', function () {
        panel.classList.toggle('open');
    });
}

/**
 * 渲染召回笔记折叠面板
 */
function renderRecallCards(el, cards) {
    if (!cards || cards.length === 0) return;

    var panel = document.createElement('div');
    panel.className = 'recall-cards-panel';

    var header = document.createElement('button');
    header.className = 'recall-cards-header';
    header.setAttribute('aria-expanded', 'false');
    header.innerHTML = '<span class="recall-cards-header-icon">' + NOTE_ICON + '</span>'
        + '<span class="recall-cards-header-text">召回笔记 (' + cards.length + ' 篇)</span>'
        + '<span class="recall-cards-header-arrow">' + CHEVRON_RIGHT_ICON + '</span>';

    var body = document.createElement('div');
    body.className = 'recall-cards-body';

    cards.forEach(function(card) {
        var item = document.createElement('div');
        item.className = 'recall-cards-item';
        item.addEventListener('click', function() {
            window.openEditor(card.id, true, false, true);
        });

        var titleRow = document.createElement('div');
        titleRow.className = 'recall-cards-item-title';
        var iconSpan = document.createElement('span');
        iconSpan.className = 'recall-cards-item-icon';
        iconSpan.innerHTML = NOTE_ICON.replace('width="14"', 'width="12"').replace('height="14"', 'height="12"');
        titleRow.appendChild(iconSpan);
        var textSpan = document.createElement('span');
        textSpan.className = 'recall-cards-item-text';
        textSpan.textContent = card.title;
        titleRow.appendChild(textSpan);
        if (card.file_ext) {
            var extSpan = document.createElement('span');
            extSpan.className = 'recall-cards-item-ext';
            extSpan.textContent = card.file_ext;
            titleRow.appendChild(extSpan);
        }
        item.appendChild(titleRow);

        if (card.content) {
            var snippet = document.createElement('div');
            snippet.className = 'recall-cards-snippet';
            snippet.textContent = card.content;
            item.appendChild(snippet);
        }

        body.appendChild(item);
    });

    panel.appendChild(header);
    panel.appendChild(body);
    el.appendChild(panel);

    header.addEventListener('click', function() {
        var isOpen = panel.classList.toggle('open');
        header.setAttribute('aria-expanded', isOpen);
    });
}

// 工具展示名：直接展示后端下发的英文工具名（web_search / recall_notes / ...），
// 不维护中文映射，避免新增工具时前端遗漏同步；如需本地化，可在此加映射兜底。
var getToolLabel = function(name) { return name || '工具'; };

/**
 * 显示 Agent 反问面板（ai:ask-user）
 * 渲染在输入区上方：问句标题 + 选项（单选/多选）+ 输入行（输入框 + 唯一按钮）。
 * 单选：点击选项即发送；多选：勾选后点「确认提交」汇总发送（勾选 + 自定义输入可拼接）。
 * 提交 = 同轮回答：经 AnswerAskUser 投递给后端等待中的 ask_user 工具，
 * 本轮 ReAct 循环继续、同一 AI 气泡续答（不新建用户消息、不开新流）。
 * 右上角关闭 = 取消本轮（与停止按钮一致），避免等待中的 run 悬挂。
 */
function showAskPanel(question, options, selection) {
    if (!askPanelEl) return;
    askPanelEl.innerHTML = '';
    const isMultiple = selection === 'multiple'; // 非 "multiple" 一律按单选处理

    // 问句标题行（标题 + 右上角关闭按钮，关闭 = 取消本轮）
    const header = document.createElement('div');
    header.className = 'ai-ask-header';
    const title = document.createElement('div');
    title.className = 'ai-ask-question';
    title.textContent = question;
    const closeBtn = document.createElement('button');
    closeBtn.type = 'button';
    closeBtn.className = 'ai-ask-close';
    closeBtn.title = '取消本轮提问';
    closeBtn.textContent = '×';
    closeBtn.addEventListener('click', () => {
        // 复用停止按钮逻辑：收起面板、移除流式气泡、取消后端 run（防止悬挂）
        if (stopBtnEl) stopBtnEl.click();
        else hideAskPanel();
    });
    header.appendChild(title);
    header.appendChild(closeBtn);
    askPanelEl.appendChild(header);

    // 输入行元素（先创建：唯一按钮回调与 Enter 需读取输入值）
    const inp = document.createElement('input');
    inp.type = 'text';
    inp.className = 'ai-ask-input';
    inp.placeholder = '输入你的答案…';
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'ai-ask-submit';
    btn.textContent = isMultiple ? '确认提交' : '发送'; // 统一布局：输入框 + 唯一按钮同排
    const shake = () => {
        inp.classList.add('error');
        setTimeout(() => inp.classList.remove('error'), 600);
    };
    // 同轮回答：投递答案给后端等待中的 ask_user 工具（AnswerAskUser），
    // 成功才隐藏面板；失败（run 已结束/取消等）保留面板便于重试
    let submitting = false; // 防重复提交（双重点击/快速 Enter）
    const doSend = async (text) => {
        if (submitting) return;
        submitting = true;
        try {
            await window.go.main.App.AnswerAskUser(activeSessionId, text);
            hideAskPanel();
        } catch (e) {
            window.showNotification?.('回答提交失败: ' + (e.message || e), 'error');
            submitting = false; // 保留面板，允许重试
        }
    };

    let doPrimary; // 唯一按钮与 Enter 共用的提交逻辑
    if (isMultiple) {
        // 多选：勾选多个选项，唯一按钮汇总提交（勾选优先，自定义输入作为补充拼接）
        const selected = new Set();
        if (options && options.length > 0) {
            const opts = document.createElement('div');
            opts.className = 'ai-ask-options';
            options.forEach(opt => {
                const optBtn = document.createElement('button');
                optBtn.type = 'button';
                optBtn.className = 'ai-ask-option';
                // 对勾图标 + 文本节点（textContent 会覆盖子元素，故用文本节点追加）
                const check = document.createElement('span');
                check.className = 'ai-ask-check';
                check.textContent = '✓';
                optBtn.appendChild(check);
                optBtn.appendChild(document.createTextNode(opt));
                optBtn.addEventListener('click', () => {
                    if (selected.has(opt)) {
                        selected.delete(opt);
                        optBtn.classList.remove('selected');
                    } else {
                        selected.add(opt);
                        optBtn.classList.add('selected');
                    }
                });
                opts.appendChild(optBtn);
            });
            askPanelEl.appendChild(opts);
        }
        doPrimary = () => {
            const v = inp.value.trim();
            if (selected.size > 0) {
                let text = '我选择：' + [...selected].join('、');
                if (v) text += '。补充说明：' + v; // 勾选 + 自定义输入拼接
                doSend(text);
            } else if (v) {
                doSend(v);
            } else {
                shake(); // 均空：不发送，输入框加轻微抖动提示
            }
        };
    } else {
        // 单选：点击选项即发送；唯一按钮发送自定义输入
        if (options && options.length > 0) {
            const opts = document.createElement('div');
            opts.className = 'ai-ask-options';
            options.forEach(opt => {
                const optBtn = document.createElement('button');
                optBtn.type = 'button';
                optBtn.className = 'ai-ask-option';
                optBtn.textContent = opt;
                optBtn.addEventListener('click', () => {
                    optBtn.classList.add('selected'); // 点过的按钮临时选中反馈
                    doSend(opt);
                });
                opts.appendChild(optBtn);
            });
            askPanelEl.appendChild(opts);
        }
        doPrimary = () => {
            const v = inp.value.trim();
            if (!v) { shake(); return; } // 空输入不发送，抖动提示
            doSend(v);
        };
    }

    // 输入行：输入框 + 唯一按钮（Enter 与按钮走同一逻辑）
    btn.addEventListener('click', doPrimary);
    inp.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); doPrimary(); } });
    const row = document.createElement('div');
    row.className = 'ai-ask-input-row';
    row.appendChild(inp);
    row.appendChild(btn);
    askPanelEl.appendChild(row);

    // 显示面板；反问等待期间输入框改为"等待你的选择…"禁用态（面板收起时恢复）
    askPanelEl.style.display = '';
    setAskInputWaiting(true);
}

/**
 * 隐藏并清空 Agent 反问面板
 */
function hideAskPanel() {
    if (!askPanelEl) return;
    askPanelEl.innerHTML = '';
    askPanelEl.style.display = 'none';
    // 面板收起（回答提交/取消本轮/结束）时恢复输入框可用态
    setAskInputWaiting(false);
}

/**
 * 反问等待期间切换输入框状态：
 * 弹出反问面板时 → 输入框禁用 + 提示"等待你的选择…"，
 * 面板收起后恢复（placeholder 还原、可输入）。
 */
function setAskInputWaiting(waiting) {
    if (!inputEl) return;
    const wrap = inputEl.closest('.ai-chat-input-wrap');
    if (waiting) {
        inputEl.disabled = true;
        if (!inputEl.dataset.normalPlaceholder) {
            inputEl.dataset.normalPlaceholder = inputEl.placeholder || '';
        }
        inputEl.placeholder = '等待你的选择…';
        if (wrap) wrap.classList.add('is-ask-waiting');
    } else {
        inputEl.disabled = false;
        if (inputEl.dataset.normalPlaceholder) {
            inputEl.placeholder = inputEl.dataset.normalPlaceholder;
            delete inputEl.dataset.normalPlaceholder;
        }
        if (wrap) wrap.classList.remove('is-ask-waiting');
    }
}

/**
 * 渲染 Agent 工具调用链（历史消息回放 / 实时完成态）。
 * 统一为默认折叠的工具摘要组件：一行「已调用 N 个工具」（+失败/部分失败徽标），
 * 点击展开明细（每个工具一条最终态记录，✓/❌/⚠️ + 文案），追加到消息末尾（正文下方证据区）。
 * toolCalls: [{ action: 'tool_start'|'tool_result', name, args, result }]
 */
function renderToolCalls(el, toolCalls) {
    if (!toolCalls || toolCalls.length === 0) return;

    // 失败/部分失败名集合（存原因文本）：tool_error、tool_partial 记录优先渲染对应状态
    var failedDetails = {};
    var partialDetails = {};
    var total = 0;
    for (var i = 0; i < toolCalls.length; i++) {
        if (toolCalls[i].action === 'tool_start') {
            total++;
        } else if (toolCalls[i].action === 'tool_error' && toolCalls[i].name) {
            failedDetails[toolCalls[i].name] = toolCalls[i].result ? String(toolCalls[i].result) : '';
        } else if (toolCalls[i].action === 'tool_partial' && toolCalls[i].name) {
            partialDetails[toolCalls[i].name] = toolCalls[i].result ? String(toolCalls[i].result) : '';
        }
    }
    if (total === 0) return;

    var list = document.createElement('div');
    list.className = 'ai-tool-status-list';

    for (var i = 0; i < toolCalls.length; i++) {
        var rec = toolCalls[i];
        if (rec.action !== 'tool_start') continue; // 只渲染工具开始记录（结果已并入最终态）
        var name = rec.name || 'tool';

        var item = document.createElement('div');
        var iconEl = document.createElement('span');
        iconEl.className = 'ai-tool-status-icon';
        var nameEl = document.createElement('span');
        nameEl.className = 'ai-tool-status-name';
        nameEl.textContent = '「' + getToolLabel(name) + '」';
        var textEl = document.createElement('span');
        textEl.className = 'ai-tool-status-text';

        var detail = function(text) {
            if (!text) return '';
            return text.length > 40 ? text.slice(0, 40) + '…' : text;
        };

        if (failedDetails.hasOwnProperty(name)) {
            // 失败态：❌ + 工具名 + 失败原因（截断 40 字符）
            item.className = 'ai-tool-status-item is-error';
            iconEl.innerHTML = svgIcon('x');
            textEl.textContent = '：失败' + (failedDetails[name] ? '：' + detail(failedDetails[name]) : '');
        } else if (partialDetails.hasOwnProperty(name)) {
            // 部分失败态：⚠️（仅 web_search 可能）
            item.className = 'ai-tool-status-item is-warning';
            iconEl.innerHTML = svgIcon('alert');
            textEl.textContent = '：部分来源失败' + (partialDetails[name] ? '：' + detail(partialDetails[name]) : '');
        } else {
            // 完成态：✓ + 工具名 + 已完成
            item.className = 'ai-tool-status-item is-done';
            iconEl.innerHTML = svgIcon('check');
            textEl.textContent = '：已完成';
        }
        item.appendChild(iconEl);
        item.appendChild(nameEl);
        item.appendChild(textEl);
        list.appendChild(item);
    }

    // 折叠摘要组件：结构与样式与来源/卡片面板一致（icon | 文本 | 徽标 | 箭头），
    // 追加到消息末尾（正文下方证据区）
    var uid = 'ai-tool-detail-' + Date.now();
    var summary = document.createElement('div');
    summary.className = 'ai-tool-summary';

    var header = document.createElement('button');
    header.type = 'button';
    header.className = 'ai-tool-summary-header';
    header.setAttribute('aria-expanded', 'false');
    header.setAttribute('aria-controls', uid);

    var iconSpan = document.createElement('span');
    iconSpan.className = 'ai-tool-summary-header-icon';
    iconSpan.innerHTML = svgIcon('wrench');
    header.appendChild(iconSpan);

    var textSpan = document.createElement('span');
    textSpan.className = 'ai-tool-summary-header-text';
    textSpan.textContent = '已调用 ' + total + ' 个工具';
    header.appendChild(textSpan);

    var failCount = Object.keys(failedDetails).length;
    var partialCount = Object.keys(partialDetails).length;
    if (failCount > 0) {
        var failBadge = document.createElement('span');
        failBadge.className = 'ai-tool-summary-status is-error';
        failBadge.textContent = failCount + ' 失败';
        header.appendChild(failBadge);
    }
    if (partialCount > 0) {
        var partialBadge = document.createElement('span');
        partialBadge.className = 'ai-tool-summary-status is-warning';
        partialBadge.textContent = partialCount + ' 来源不可用';
        header.appendChild(partialBadge);
    }

    var arrowSpan = document.createElement('span');
    arrowSpan.className = 'ai-tool-summary-header-arrow';
    arrowSpan.innerHTML = CHEVRON_RIGHT_ICON;
    header.appendChild(arrowSpan);

    var body = document.createElement('div');
    body.className = 'ai-tool-summary-body';
    body.id = uid;
    body.appendChild(list);

    summary.appendChild(header);
    summary.appendChild(body);

    // 与来源/卡片面板一致：点击 header 切换面板 .open（aria-expanded 同步）
    header.addEventListener('click', function() {
        var isOpen = summary.classList.toggle('open');
        header.setAttribute('aria-expanded', isOpen);
    });

    el.appendChild(summary);
}

/**
 * 进入编辑模式 — 将消息文本替换为 textarea
 */
function enterEditMode(msgEl, originalContent) {
    const contentDiv = msgEl.querySelector('.msg-content');
    if (!contentDiv) return;

    msgEl.dataset.originalContent = originalContent;

    // 用户消息的内容直接挂载在 textContent 上, 清除以便插入 textarea
    contentDiv.textContent = '';

    const textarea = document.createElement('textarea');
    textarea.className = 'ai-msg-edit-textarea';
    textarea.value = originalContent;
    textarea.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            confirmEdit(msgEl);
        }
        if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            cancelEdit(msgEl);
        }
    });
    contentDiv.appendChild(textarea);

    setTimeout(() => {
        textarea.focus();
        textarea.select();
    }, 50);

    const actions = msgEl.querySelector('.ai-msg-actions');
    const moreBtn = actions?.querySelector('.more-btn');
    if (moreBtn) moreBtn.style.display = 'none';
    // 编辑模式下隐藏用户消息的 token 脚标
    const tokensEl = actions?.querySelector('.user-tokens');
    if (tokensEl) tokensEl.style.display = 'none';

    const editActions = document.createElement('div');
    editActions.className = 'ai-msg-edit-actions';

    const confirmBtn = document.createElement('button');
    confirmBtn.className = 'ai-msg-edit-btn ai-msg-edit-confirm';
    confirmBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
    confirmBtn.title = '确认 (Ctrl+Enter)';
    confirmBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        confirmEdit(msgEl);
    });

    const cancelBtn = document.createElement('button');
    cancelBtn.className = 'ai-msg-edit-btn ai-msg-edit-cancel';
    cancelBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
    cancelBtn.title = '取消 (Esc)';
    cancelBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        cancelEdit(msgEl);
    });

    editActions.appendChild(confirmBtn);
    editActions.appendChild(cancelBtn);

    if (actions) {
        actions.appendChild(editActions);
    }
}

/**
 * 取消编辑 — 恢复原文
 */
function cancelEdit(msgEl) {
    const contentDiv = msgEl.querySelector('.msg-content');
    if (!contentDiv) return;

    const textarea = contentDiv.querySelector('.ai-msg-edit-textarea');
    if (textarea) textarea.remove();

    // 恢复原始内容 (用户消息使用 textContent)
    contentDiv.textContent = msgEl.dataset.originalContent || '';

    const editActions = msgEl.querySelector('.ai-msg-edit-actions');
    if (editActions) editActions.remove();

    const moreBtn = msgEl.querySelector('.more-btn');
    if (moreBtn) moreBtn.style.display = '';

    const tokensEl = msgEl.querySelector('.user-tokens');
    if (tokensEl) tokensEl.style.display = '';
}

/**
 * 确认编辑
 */
function confirmEdit(msgEl) {
    const textarea = msgEl.querySelector('.ai-msg-edit-textarea');
    if (!textarea) return;

    const newContent = textarea.value;
    const originalContent = msgEl.dataset.originalContent || '';

    if (!newContent.trim()) {
        textarea.classList.add('ai-msg-edit-error');
        setTimeout(() => textarea.classList.remove('ai-msg-edit-error'), 500);
        return;
    }

    if (newContent === originalContent) {
        cancelEdit(msgEl);
        return;
    }

    applyEdit(msgEl, newContent);
}

/**
 * 应用编辑 — 更新内容、清除后续消息、调用后端、重新流式
 */
async function applyEdit(msgEl, newContent) {
    const contentDiv = msgEl.querySelector('.msg-content');
    if (contentDiv) {
        contentDiv.textContent = newContent;
    }

    _contextMsgContent = newContent;

    // 从元素获取 msgId
    const msgId = parseInt(msgEl.dataset.msgId);
    if (!msgId) return;

    // 删除后续所有消息 DOM
    let nextEl = msgEl.nextElementSibling;
    while (nextEl) {
        const toRemove = nextEl;
        nextEl = nextEl.nextElementSibling;
        if (toRemove.classList.contains('ai-msg')) {
            toRemove.remove();
        }
    }

    // 后端截断（保留本条及之后的消息，删除本条之前的旧消息已通过后端处理）
    if (activeSessionId !== null) {
        try {
            await window.go.main.App.TruncateAISessionAfterMessage(activeSessionId, msgId);
        } catch (_) { /* 静默失败 */ }
        try {
            await window.go.main.App.UpdateAIMessageContent(msgId, newContent);
        } catch (_) { /* 静默失败 */ }
    }

    // 截断 chatHistory 缓冲区
    const idx = chatHistory.findIndex(m => m.id === msgId);
    if (idx >= 0) {
        chatHistory = chatHistory.slice(0, idx + 1);
        chatHistory[chatHistory.length - 1].content = newContent;
    }

    // 更新 dataset 后再 cancelEdit, 避免 cancelEdit 从 dataset 恢复旧内容
    msgEl.dataset.originalContent = newContent;
    cancelEdit(msgEl);
    // 生成期间隐藏旧 token 数，待 stream-done 后更新时再显示
    const oldTokensEl = msgEl.querySelector('.user-tokens');
    if (oldTokensEl) oldTokensEl.style.display = 'none';

    if (!isStreaming) {
        if (!(await ensureAIReady('开始对话'))) return;
        await startStreaming(newContent, true, 0);
        scrollToBottom();
    }
}

/**
 * 处理删除消息 — 确认后移除 DOM、清理 chatHistory、同步后端
 */
async function handleDeleteMsg(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const confirmed = await window.showConfirmDialog('确定要删除这条消息吗？');
    if (!confirmed) return;

    const msgId = parseInt(msgEl.dataset.msgId);
    if (!msgId) return;

    // 仅移除本条消息（DOM）
    msgEl.remove();

    // 后端仅删除本条消息
    if (activeSessionId !== null) {
        try {
            await window.go.main.App.DeleteAIMessage(msgId);
        } catch (_) { /* 静默失败 */ }
    }

    // 从 chatHistory 中移除本条消息
    chatHistory = chatHistory.filter(m => m.id !== msgId);

    // 删除后如果 chatHistory 为空, 显示空对话欢迎语
    if (chatHistory.length === 0) {
        showWelcome();
    }

    // 刷新会话 token 总数显示
    updateContextSize();
    scrollToBottom();
}

async function handleRegenerate(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const msgId = parseInt(msgEl.dataset.msgId);
    if (!msgId) return;

    // 找到前一条消息（用户消息）
    const prevEl = msgEl.previousElementSibling;
    if (!prevEl || !prevEl.classList.contains('ai-msg')) return;
    const prevMsgId = parseInt(prevEl.dataset.msgId);
    if (!prevMsgId) return;

    // 移除该 AI 消息及之后的所有后续消息（DOM）
    let nextEl = msgEl;
    while (nextEl) {
        const toRemove = nextEl;
        nextEl = nextEl.nextElementSibling;
        if (toRemove.classList.contains('ai-msg')) {
            toRemove.remove();
        }
    }

    // 后端截断（保留前一条用户消息及之前的内容）
    if (activeSessionId !== null) {
        try {
            await window.go.main.App.TruncateAISessionAfterMessage(activeSessionId, prevMsgId);
        } catch (_) { /* 静默 */ }
    }

    // 截断 chatHistory 缓冲区
    const idx = chatHistory.findIndex(m => m.id === msgId);
    if (idx >= 0) {
        chatHistory = chatHistory.slice(0, idx);
    }

    // 再生（regenerate 不新建用户消息，userMsgID 传 0）
    if (!(await ensureAIReady('重新生成'))) return;
    await startStreaming('', true, 0);
    scrollToBottom();
}

/**
 * 重新发送用户消息
 */
async function handleResend(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const msgId = parseInt(msgEl.dataset.msgId);
    if (!msgId) return;

    const contentDiv = msgEl.querySelector('.msg-content');
    if (!contentDiv) return;
    const content = contentDiv.textContent || '';

    // 移除该消息及之后的所有后续消息（DOM）
    let nextEl = msgEl;
    while (nextEl) {
        const toRemove = nextEl;
        nextEl = nextEl.nextElementSibling;
        if (toRemove.classList.contains('ai-msg')) {
            toRemove.remove();
        }
    }

    // 后端截断（删除本条及之后的消息）
    if (activeSessionId !== null) {
        try {
            await window.go.main.App.TruncateAISessionAtMessage(activeSessionId, msgId);
        } catch (_) { /* 静默 */ }
        // 截断后后端已重算会话 token 缓存，立即刷新右上角总 token 显示
        updateContextSize();
    }

    // 截断 chatHistory 缓冲区
    const idx = chatHistory.findIndex(m => m.id === msgId);
    if (idx >= 0) {
        chatHistory = chatHistory.slice(0, idx);
    }

    // 先保存用户消息到数据库，拿到 msgId 和 token 数
    let newUserMsgId = 0;
    let resendTokens = 0;
    if (activeSessionId !== null) {
        try {
            const result = await window.go.main.App.SaveAIMessage(activeSessionId, content, 'user');
            newUserMsgId = result?.msgID || 0;
            resendTokens = result?.tokens || 0;
        } catch (_) { /* 静默 */ }
    }

    // 重新添加用户消息气泡（带上 msgId 和 token）
    addMessage(content, 'user', undefined, undefined, undefined, resendTokens, newUserMsgId || undefined);
    const newUserMsgEl = messagesInnerEl.lastElementChild;
    if (newUserMsgEl) {
        newUserMsgEl.appendChild(createMsgActions(content, 'user', undefined, resendTokens));
        bindMsgContextMenu(newUserMsgEl, content, 'user');
    }

    // 重新发送
    if (!(await ensureAIReady('重新发送'))) return;
    await startStreaming(content, false, newUserMsgId);
    scrollToBottom();
}

/* ── 笔记引用 ═══════════════════════════════════════════════════ */

const DOC_ICON = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>';
const CHECK_SVG = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';

/**
 * 打开笔记引用选择浮层
 */
async function openNoteRefModal() {
    if (!refModal) return;
    _refTempSelected = {};
    _refSelectAll = false;
    // 以已有引用笔记预填选中状态
    referencedNotes.forEach(n => { _refTempSelected[n.id] = true; });

    refModal.style.display = 'flex';

    // 重置搜索/筛选
    if (refSearch) refSearch.value = '';
    if (refSearchClear) refSearchClear.classList.remove('visible');
    _currentNotebookId = 0;
    if (refNotebookLabel) refNotebookLabel.textContent = '全部笔记本';
    if (refNotebookBtn) refNotebookBtn.classList.remove('active');
    if (refNotebookFilter) refNotebookFilter.classList.remove('open');

    // 读取分页设置
    await loadRefPageSize();

    _refListLoaded = false;
    _refCurrentPage = 1;
    _refTotalItems = 0;
    _refLoading = false;
    _refPendingRefresh = false;
    _refTagIds.clear();
    if (refTagLabel) refTagLabel.textContent = '标签';
    if (refTagBtn) refTagBtn.classList.remove('active');
    // 重置滚动位置
    if (refListWrap) refListWrap.scrollTop = 0;
    // 骨架屏默认可见
    if (refSkeleton) refSkeleton.classList.remove('hidden');
    if (refLoadingOverlay) refLoadingOverlay.classList.remove('active');

    // 并行加载笔记本选项、标签和笔记列表
    await Promise.all([
        loadAllNotebooks(),
        loadAllRefTags(),
        loadNoteList()
    ]);

    // 打开后确保全选按钮显示未选中状态
    updateSelectAllBtn();

    // 焦点到搜索框
    setTimeout(() => refSearch?.focus(), 150);
}

/**
 * 从设置中读取分页大小 (带缓存, 仅首次调用时向后端请求) 
 */
async function loadRefPageSize() {
    if (_pageSizeLoaded) return;
    _refPageSize = 20;
    try {
        if (window.go?.main?.App?.GetPageSize) {
            const saved = await window.go.main.App.GetPageSize();
            if (saved && saved >= 10 && saved <= 100) {
                _refPageSize = saved;
            }
        }
        _pageSizeLoaded = true;
    } catch (_) { /* 使用默认值 */ }
}

/**
 * 关闭笔记引用选择浮层
 */
function closeNoteRefModal() {
    if (!refModal) return;
    refModal.style.display = 'none';
    _refTempSelected = {};
    // 清理角色档案选择器可能劫持的确认按钮 handler，并恢复常规确认处理函数
    if (refConfirm) {
        refConfirm.onclick = null;
        refConfirm.addEventListener('click', confirmNoteSelection);
    }
}

/**
 * 加载所有笔记本到筛选下拉框 (带缓存, 仅首次调用时向后端请求) 
 */
async function loadAllNotebooks() {
    if (!refNotebookDropdown) return;
    try {
        if (_notebooksCache) {
            rebuildRefNotebookOptions();
            return;
        }
        const notebooks = await window.go.main.App.GetAllNotebooks() || [];
        _notebooksCache = notebooks;
        rebuildRefNotebookOptions();
    } catch (_) {
        /* 静默 */
    }
}

/**
 * 使用缓存的笔记本列表重建自定义下拉选项
 */
function rebuildRefNotebookOptions() {
    if (!refNotebookDropdown || !_notebooksCache) return;
    const currentId = _currentNotebookId || 0;
    let html = '';
    html += `<div class="ai-note-ref-filter-option${currentId === 0 ? ' selected' : ''}" data-notebook-id="0">全部笔记本</div>`;
    _notebooksCache.forEach(n => {
        const selected = currentId === n.id;
        html += `<div class="ai-note-ref-filter-option${selected ? ' selected' : ''}" data-notebook-id="${n.id}">${n.name}</div>`;
    });
    refNotebookDropdown.innerHTML = html;

    refNotebookDropdown.querySelectorAll('.ai-note-ref-filter-option').forEach(opt => {
        opt.addEventListener('click', (e) => {
            e.stopPropagation();
            const notebookId = parseInt(opt.dataset.notebookId);
            _currentNotebookId = notebookId;
            // 更新按钮文字
            if (refNotebookLabel) {
                if (notebookId === 0) refNotebookLabel.textContent = '全部笔记本';
                else {
                    const nb = (_notebooksCache || []).find(n => n.id === notebookId);
                    refNotebookLabel.textContent = nb ? nb.name : '全部笔记本';
                }
            }
            // 关闭下拉
            if (refNotebookFilter) refNotebookFilter.classList.remove('open');
            // 更新选中样式
            refNotebookFilter.classList.toggle('active', notebookId > 0);
            // 触发列表刷新
            loadNoteList(false);
        });
    });
}

/**
 * 加载所有标签到缓存 (带缓存, 仅首次调用时向后端请求) 
 */
async function loadAllRefTags() {
    try {
        if (_tagsCache) return;
        const tags = await window.go.main.App.GetAllTags() || [];
        _tagsCache = tags;
    } catch (_) { /* 静默 */ }
}

/**
 * 渲染标签筛选下拉菜单
 */
function renderRefTagDropdown() {
    if (!refTagDropdown) return;
    const tags = _tagsCache || [];
    let html = '';
    // "全部"选项
    const allSelected = _refTagIds.size === 0;
    html += `<div class="ai-note-ref-filter-option${allSelected ? ' selected' : ''}" data-tag-id="all">全部</div>`;
    // 各标签选项
    tags.forEach(tag => {
        const selected = _refTagIds.has(tag.id);
        html += `<div class="ai-note-ref-filter-option${selected ? ' selected' : ''}" data-tag-id="${tag.id}">#${tag.name || ''}</div>`;
    });
    refTagDropdown.innerHTML = html;

    // 绑定点击事件
    refTagDropdown.querySelectorAll('.ai-note-ref-filter-option').forEach(opt => {
        opt.addEventListener('click', (e) => {
            e.stopPropagation();
            const tagId = opt.dataset.tagId;
            if (tagId === 'all') {
                _refTagIds.clear();
            } else {
                const id = parseInt(tagId);
                if (_refTagIds.has(id)) {
                    _refTagIds.delete(id);
                } else {
                    _refTagIds.add(id);
                }
            }
            // 关闭下拉
            if (refTagFilter) refTagFilter.classList.remove('open');
            // 更新按钮 active 样式 + label
            updateRefTagFilterBtn();
            // 刷新列表
            loadNoteList();
        });
    });
}

/**
 * 更新标签筛选按钮状态
 */
function updateRefTagFilterBtn() {
    if (!refTagBtn || !refTagLabel) return;
    const count = _refTagIds.size;
    refTagBtn.classList.toggle('active', count > 0);
    if (count === 0) {
        refTagLabel.textContent = '标签';
    } else if (count === 1) {
        const id = Array.from(_refTagIds)[0];
        const tag = (_tagsCache || []).find(t => t.id === id);
        refTagLabel.textContent = '#' + (tag ? tag.name : '');
    } else {
        refTagLabel.textContent = count + ' 个标签';
    }
}

/**
 * 加载笔记列表 (根据当前搜索关键词和笔记本筛选) 
 * 加载策略 :
 *   - 首次加载 :显示骨架屏, 数据到达后替换
 *   - 二次加载 :保留旧列表, 显示半透明 overlay + 旋转环, 数据到达后替换
 *   - 点击加载更多: 追加到列表末尾
 */
async function loadNoteList(append = false) {
    if (!refList) {
        console.warn('[loadNoteList] refList is null, abort');
        return;
    }
    // 若正在加载中 :非追加调用 (笔记本/搜索切换) 设待刷新标志, 追加调用 (滚动) 直接略过
    if (_refLoading) {
        console.log('[loadNoteList] _refLoading is true, set pendingRefresh=', !append);
        if (!append) _refPendingRefresh = true;
        return;
    }
    // 非追加（筛选变更）且全选激活时，重置全选标志但不清除已选 ID
    if (!append && _refSelectAll) {
        _refSelectAll = false;
        // 不需要清除 _refTempSelected
    }
    _refLoading = true;

    const query = refSearch?.value.trim() || '';
    const notebookId = _currentNotebookId || 0;
    const page = append ? _refCurrentPage + 1 : 1;
    console.log('[loadNoteList] proceeding: append=', append, 'query=', query, 'notebookId=', notebookId, 'page=', page, '_refListLoaded=', _refListLoaded);

    // 首次 → 骨架屏; 二次刷新 → overlay; 追加显示加载指示器
    if (!append) {
        if (_refListLoaded && refLoadingOverlay) {
            refLoadingOverlay.classList.add('active');
        }
    } else {
        // 显示列表底部加载指示器
        const loader = document.getElementById('aiNoteRefListLoader');
        if (loader) loader.classList.add('visible');
    }

    try {
        let result;
        if (query || notebookId > 0 || _refTagIds.size > 0) {
            console.log('[loadNoteList] calling SearchNotes with notebookId=', notebookId);
            const tagIds = _refTagIds.size > 0 ? Array.from(_refTagIds) : [];
            result = await window.go.main.App.SearchNotes(query, page, _refPageSize, notebookId, 'updated_at', '', '', tagIds);
        } else {
            console.log('[loadNoteList] calling GetNotes (no filter)');
            result = await window.go.main.App.GetNotes(page, _refPageSize, 'updated_at', 0);
        }
        const notes = result?.items || [];
        _refTotalItems = result?.total || 0;
        console.log('[loadNoteList] API result: notes.length=', notes.length, 'total=', _refTotalItems);

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
        console.error('[loadNoteList] API error:', e);
        // 隐藏加载指示器
        const loader = document.getElementById('aiNoteRefListLoader');
        if (loader) loader.classList.remove('visible');

        if (!append) {
            if (refSkeleton) refSkeleton.classList.add('hidden');
            if (refLoadingOverlay) refLoadingOverlay.classList.remove('active');
            if (!_refListLoaded) {
                refList.innerHTML = '<div class="ai-note-ref-empty">加载失败</div>';
            }
        }
    } finally {
        console.log('[loadNoteList] finally: _refPendingRefresh=', _refPendingRefresh);
        _refLoading = false;
        // 若在加载期间有待刷新的筛选变更 (切换笔记本/搜索) , 自动重试
        if (_refPendingRefresh) {
            _refPendingRefresh = false;
            console.log('[loadNoteList] retrying due to pendingRefresh');
            loadNoteList();
        }
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

    // 全选模式下，确保所有笔记 ID 在 _refTempSelected 中
    if (_refSelectAll && notes.length > 0) {
        notes.forEach(n => { _refTempSelected[n.id] = true; });
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
        const tags = note.tags || note.Tags || [];
        const tagsHtml = tags.length > 0
            ? tags.slice(0, 3).map(t => {
                const tagId = t.id || t.ID || 0;
                const active = _refTagIds.has(tagId) ? ' filter-active' : '';
                return `<span class="ai-note-ref-item-tag${active}">#${t.name || t.Name || ''}</span>`;
              }).join('') : '';
        return `<div class="ai-note-ref-item${isSelected ? ' selected' : ''}" data-id="${note.id}" style="--i:${idx}">
            <div class="ai-note-ref-item-check">${CHECK_SVG}</div>
            <div class="ai-note-ref-item-info">
                <div class="ai-note-ref-item-title">${title}</div>
                <div class="ai-note-ref-item-meta">
                    ${date ? `<span>🕐 ${date}</span>` : ''}${tagsHtml}
                </div>
            </div>
        </div>`;
    }).join('');

    refList.innerHTML = html;
    updateRefCount();
}

/**
 * 追加笔记到列表末尾 (加载更多) 
 * @param {Array} notes - 新加载的笔记列表
 */
function appendToList(notes) {
    if (!refList || !notes || notes.length === 0) {
        return;
    }

    // 全选模式下，自动选中新追加的条目
    if (_refSelectAll && notes.length > 0) {
        notes.forEach(n => { _refTempSelected[n.id] = true; });
    }

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
        const tags = note.tags || note.Tags || [];
        const tagsHtml = tags.length > 0
            ? tags.slice(0, 3).map(t => {
                const tagId = t.id || t.ID || 0;
                const active = _refTagIds.has(tagId) ? ' filter-active' : '';
                return `<span class="ai-note-ref-item-tag${active}">#${t.name || t.Name || ''}</span>`;
              }).join('') : '';
        return `<div class="ai-note-ref-item${isSelected ? ' selected' : ''}" data-id="${note.id}" style="--i:${startIdx + idx};animation:none;opacity:1;transform:translateY(0)">
            <div class="ai-note-ref-item-check">${CHECK_SVG}</div>
            <div class="ai-note-ref-item-info">
                <div class="ai-note-ref-item-title">${title}</div>
                <div class="ai-note-ref-item-meta">
                    ${date ? `<span>🕐 ${date}</span>` : ''}${tagsHtml}
                </div>
            </div>
        </div>`;
    }).join('');

    // 追加新条目
    while (fragment.firstChild) {
        refList.appendChild(fragment.firstChild);
    }

    // 先追加条目再隐藏加载器, 避免 DOM 空隙导致白闪
    const loader = document.getElementById('aiNoteRefListLoader');
    if (loader) loader.classList.remove('visible');

    updateRefCount();
}

/**
 * 切换笔记选中状态
 * @param {string|number} id - 笔记 ID
 */
function toggleNoteSelection(id) {
    if (!id) return;
    if (_refTempSelected[id]) {
        delete _refTempSelected[id];
        // 手动取消选中时，退出全选模式
        if (_refSelectAll) {
            _refSelectAll = false;
            updateSelectAllBtn();
        }
    } else {
        _refTempSelected[id] = true;
    }
    // 仅更新选中态 (不重新加载列表, 保留滚动位置) 
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
        // 角色档案选择器模式下，允许空选中点击确认（用于清空档案）
        const isRoleplayMode = !!refConfirm.onclick;
        refConfirm.disabled = !isRoleplayMode && count === 0;
        refConfirm.style.opacity = count === 0 && !isRoleplayMode ? '0.5' : '1';
    }
    // 手动逐条选中所有条目时自动切换全选为勾选状态
    if (!_refSelectAll && count > 0 && _refTotalItems > 0 && count >= _refTotalItems) {
        _refSelectAll = true;
    } else if (_refSelectAll && count < _refTotalItems) {
        _refSelectAll = false;
    }
    // 同步全选按钮状态
    updateSelectAllBtn();
}

/**
 * 切换全选/取消全选
 */
async function toggleRefSelectAll() {
    if (_refSelectAll) {
        // 已全选 → 取消全选
        _refSelectAll = false;
        _refTempSelected = {};
        updateRefCount();
        // 更新所有列表条目的选中态
        if (refList) {
            refList.querySelectorAll('.ai-note-ref-item').forEach(item => {
                item.classList.remove('selected');
            });
        }
        // 更新全选按钮样式
        updateSelectAllBtn();
        return;
    }

    // 未全选 → 全选：根据当前筛选条件获取所有匹配 ID
    const query = refSearch?.value.trim() || '';
    const notebookId = _currentNotebookId || 0;
    const tagIds = _refTagIds.size > 0 ? Array.from(_refTagIds) : [];

    try {
        let ids = [];
        if (query || notebookId > 0 || tagIds.length > 0) {
            // 有筛选条件 → 调用 SearchNoteIDs
            ids = await window.go.main.App.SearchNoteIDs(query, notebookId, tagIds);
        } else {
            // 无筛选 → 获取所有笔记 ID
            ids = await window.go.main.App.GetAllNoteIDs();
        }

        if (!ids || ids.length === 0) return;

        _refSelectAll = true;
        ids.forEach(id => { _refTempSelected[id] = true; });

        // 更新所有列表条目的选中态
        if (refList) {
            refList.querySelectorAll('.ai-note-ref-item').forEach(item => {
                if (_refTempSelected[item.dataset.id]) {
                    item.classList.add('selected');
                } else {
                    item.classList.remove('selected');
                }
            });
        }

        updateRefCount();
        updateSelectAllBtn();
    } catch (e) {
        console.error('toggleRefSelectAll 失败:', e);
    }
}

/**
 * 更新全选按钮的选中状态样式
 */
function updateSelectAllBtn() {
    const btn = document.getElementById('aiNoteRefSelectAll');
    if (!btn) return;
    btn.classList.toggle('checked', _refSelectAll);
    // 全选时更换为对勾图标，取消时更换为方框图标
    const checkEl = btn.querySelector('.ai-note-ref-select-all-check');
    if (checkEl) {
        if (_refSelectAll) {
            checkEl.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
        } else {
            checkEl.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="3"/></svg>';
        }
    }
}

/**
 * 确认笔记选择, 更新 chips
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

        // 合并 :保留未取消的旧引用 + 新增的
        const newIds = new Set(ids);
        const keepNotes = referencedNotes.filter(n => !newIds.has(n.id));
        referencedNotes = [...keepNotes, ...refContext.notes];
    } catch (e) {
        console.error('confirmNoteSelection 失败:', e);
        return;
    }

    closeNoteRefModal();
    updateRefChips();
    await saveCurrentSessionConfig();
}

/**
 * 更新引用笔记 chips 显示
 */
function updateRefChips() {
    if (!refChips || !refBar) return;

    if (referencedNotes.length === 0) {
        refBar.style.display = 'none';
        if (addBtn) addBtn.classList.remove('has-ref');
        return;
    }

    refBar.style.display = '';
    if (addBtn) addBtn.classList.add('has-ref');

    // 渲染单个笔记 chips
    let html = referencedNotes.map(n => {
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

    // 3 篇以上时追加批量移除标签
    if (referencedNotes.length >= 3) {
        html += `<div class="ai-chat-ref-chip ai-chat-ref-chip-remove-all" title="一键移除全部引用">
            <svg class="ai-chat-ref-chip-remove-all-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            <span>移除全部 ${referencedNotes.length} 篇</span>
        </div>`;
    }

    refChips.innerHTML = html;

    // 绑定单个移除事件
    refChips.querySelectorAll('.ai-chat-ref-chip-remove').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            removeRefNote(btn.dataset.id);
        });
    });

    // 绑定批量移除事件
    const removeAllBtn = refChips.querySelector('.ai-chat-ref-chip-remove-all');
    if (removeAllBtn) {
        removeAllBtn.addEventListener('click', () => {
            referencedNotes = [];
            // 同步清理角色档案
            if (roleplayNotes.length > 0) {
                roleplayNotes = [];
                renderSkillChips();
            }
            updateRefChips();
            saveCurrentSessionConfig();
        });
    }
}

/**
 * 移除单条引用笔记
 * @param {string|number} id - 笔记 ID
 */
function removeRefNote(id) {
    referencedNotes = referencedNotes.filter(n => String(n.id) !== String(id));
    // 同步清理角色档案中相同 ID 的笔记
    const oldLen = roleplayNotes.length;
    roleplayNotes = roleplayNotes.filter(n => String(n.id) !== String(id));
    if (roleplayNotes.length !== oldLen) {
        renderSkillChips();
    }
    updateRefChips();
    saveCurrentSessionConfig();
}

/**
 * 渲染上传文件 chips
 */
function renderFileChips() {
    if (!fileChips || !fileBar) return;

    // 有上传文件时高亮按钮，与引用笔记按钮行为一致
    if (addBtn) addBtn.classList.toggle('has-ref', uploadedFiles.length > 0);

    if (uploadedFiles.length === 0) {
        fileBar.style.display = 'none';
        return;
    }

    fileBar.style.display = '';

    // 渲染单个文件 chips
    let html = uploadedFiles.map((f, idx) => {
        const truncTip = f.truncated ? '<span class="ai-chat-ref-chip-trunc">(内容已截断)</span>' : '';
        return `<div class="ai-chat-file-chip" data-index="${idx}">
            <span class="ai-chat-file-chip-icon">${DOC_ICON}</span>
            <span class="ai-chat-file-chip-name" title="${f.name.replace(/"/g, '&quot;')}">${f.name}</span>
            ${truncTip}
            <button class="ai-chat-file-chip-remove" data-index="${idx}" title="移除文件">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>`;
    }).join('');

    // 3 个文件以上时追加批量清除按钮
    if (uploadedFiles.length >= 3) {
        html += `<div class="ai-chat-ref-chip ai-chat-ref-chip-remove-all" title="一键移除全部文件">
            <svg class="ai-chat-ref-chip-remove-all-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            <span>移除全部 ${uploadedFiles.length} 个</span>
        </div>`;
    }

    fileChips.innerHTML = html;

    // 绑定单个移除事件
    fileChips.querySelectorAll('.ai-chat-file-chip-remove').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            const idx = parseInt(btn.dataset.index);
            if (!isNaN(idx)) {
                uploadedFiles.splice(idx, 1);
                renderFileChips();
            }
        });
    });

    // 绑定批量清除事件
    const removeAllBtn = fileChips.querySelector('.ai-chat-ref-chip-remove-all');
    if (removeAllBtn) {
        removeAllBtn.addEventListener('click', () => {
            uploadedFiles = [];
            renderFileChips();
        });
    }

}

/**
 * 初始化 AI 聊天文件拖拽上传
 */
function initAiChatFileDrop() {
    if (!aiChatContent || !aiChatDropOverlay) return;

    // ── 拖拽进入：显示遮罩 ──
    aiChatContent.addEventListener('dragenter', (e) => {
        e.preventDefault();
        // AI 正在回复时不显示拖拽遮罩
        if (isStreaming) return;
        if (!e.dataTransfer.types.includes('Files')) return;
        _aiDragCounter++;
        if (_aiDragCounter === 1) {
            aiChatDropOverlay.style.display = 'flex';
            // 用 requestAnimationFrame 确保 display 生效后再触发 transition
            requestAnimationFrame(() => {
                aiChatDropOverlay.classList.add('active');
            });
        }
    });

    // ── 拖拽悬停：必须 preventDefault 否则 drop 不触发 ──
    aiChatContent.addEventListener('dragover', (e) => {
        e.preventDefault();
    });

    // ── 拖拽离开：隐藏遮罩 ──
    aiChatContent.addEventListener('dragleave', (e) => {
        e.preventDefault();
        // AI 正在回复时跳过拖拽状态管理（无遮罩可关闭）
        if (isStreaming) return;
        _aiDragCounter--;
        if (_aiDragCounter <= 0) {
            _aiDragCounter = 0;
            aiChatDropOverlay.classList.remove('active');
            // transition 结束后隐藏 display
            setTimeout(() => {
                if (!aiChatDropOverlay.classList.contains('active')) {
                    aiChatDropOverlay.style.display = 'none';
                }
            }, 200);
        }
    });

    // ── 拖拽释放：隐藏遮罩（实际文件由 OnFileDrop 处理） ──
    aiChatContent.addEventListener('drop', (e) => {
        e.preventDefault();
        // AI 正在回复时跳过拖拽状态管理（无遮罩可关闭）
        if (isStreaming) return;
        _aiDragCounter = 0;
        aiChatDropOverlay.classList.remove('active');
        aiChatDropOverlay.style.display = 'none';
    });
}

// ── 拖拽文件处理（由 main.js OnFileDrop 回调调用） ──
window.handleAiChatFileDrop = async function(paths) {
    // 确保遮罩已隐藏
    _aiDragCounter = 0;
    if (aiChatDropOverlay) {
        aiChatDropOverlay.classList.remove('active');
        aiChatDropOverlay.style.display = 'none';
    }

    // AI 正在回复时禁止上传文件
    if (isStreaming) {
        window.showNotification?.('AI 正在回复中，请稍后再试', 'warning');
        return;
    }

    if (!paths || paths.length === 0) return;

    let progressCtrl = null;
    let done = false;
    const startTime = Date.now();

    // 注册上传进度事件监听
    const nm = window.nm;
    const unsub = window.runtime.EventsOn('import:ai-progress', (type, total) => {
        if (type === 'start') {
            progressCtrl = nm.showProgress('正在上传', total);
        } else if (type === 'complete') {
            done = true;
            if (!progressCtrl) return;

            // 最小展示 500ms，避免闪一下就消失
            const elapsed = Date.now() - startTime;
            setTimeout(() => {
                nm._dismiss(progressCtrl);
            }, Math.max(0, 500 - elapsed));
        }
    });

    try {
        const results = await window.go.main.App.ReadAIChatFiles(paths);
        if (!results || results.length === 0) return;

        if (done) {
            // 等保底延迟完成后再展示通知和渲染文件列表
            const elapsed = Date.now() - startTime;
            setTimeout(() => {
                showAIChatResults(results);
            }, Math.max(0, 500 - elapsed));
            return;
        }

        // 兜底：事件未收到时
        showAIChatResults(results);
    } catch (e) {
        window.showNotification?.('拖拽文件失败: ' + (e.message || e), 'error');
    } finally {
        unsub();
    }
};

// 展示 AI 文件上传结果通知（成功/失败详情），并渲染文件列表
function showAIChatResults(results) {
    let successCount = 0;
    let failCount = 0;

    for (const r of results) {
        if (r.error) {
            failCount++;
            window.showNotification?.(r.error, 'error');
        } else {
            successCount++;
            uploadedFiles.push({
                name: r.name,
                content: r.content,
                size: r.size,
                truncated: r.truncated,
            });
        }
    }

    if (successCount > 0) {
        const msg = failCount > 0
            ? `${successCount} 个文件上传成功，${failCount} 个失败`
            : `${successCount} 个文件上传成功`;
        window.nm?.show(msg, 'success');
    }

    renderFileChips();
};

/**
 * HTML table 元素转 Markdown 表格文本
 * @param {HTMLTableElement} tableEl
 * @returns {string}
 */
function tableToMarkdown(tableEl) {
    const rows = [];
    const trs = tableEl.querySelectorAll('tr');
    if (!trs.length) return '';
    trs.forEach((tr, index) => {
        const cells = tr.querySelectorAll('th, td');
        const row = '| ' + Array.from(cells).map(c => c.textContent.trim()).join(' | ') + ' |';
        rows.push(row);
        if (index === 0 && tr.querySelector('th')) {
            const sep = '| ' + Array.from(cells).map(() => '---').join(' | ') + ' |';
            rows.push(sep);
        }
    });
    return rows.join('\n');
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

/**
 * 保存当前会话的操作栏配置到后端
 */
async function saveCurrentSessionConfig() {
    if (!activeSessionId) return;
    try {
        await window.go.main.App.SaveSessionConfig(activeSessionId, {
            model_name: modelLabel?.textContent || '',
            agent_enabled: agentEnabled,
            enable_thinking: enableThinking,
            zhihu_search_enabled: searchSources.has('zhihu_search'),
            zhihu_global_search_enabled: searchSources.has('zhihu_global'),
            tavily_search_enabled: searchSources.has('tavily'),
            enable_card_recall: enableCardRecall,
            referenced_notes: JSON.stringify(referencedNotes),
            roleplay_notes: JSON.stringify(roleplayNotes),
            enabled_skills: JSON.stringify(activeSkills),
            recall_notebook_ids: JSON.stringify(Array.from(recallNotebookIds)),
        });
    } catch (_) {}
}

/**
 * 每次打开菜单时重新获取最新笔记本列表
 */
async function loadRecallNotebookMenu() {
    if (!cardRecallDropdown) return;
    try {
        const notebooks = await window.go.main.App.GetAllNotebooks();
        if (!notebooks || notebooks.length === 0) return;
        cardRecallDropdown.innerHTML = '';
        notebooks.forEach(nb => {
            const item = document.createElement('div');
            item.className = 'ai-chat-recall-item';
            item.dataset.id = nb.id;
            const cb = document.createElement('input');
            cb.type = 'checkbox';
            cb.id = 'aiRecallNb_' + nb.id;
            cb.checked = recallNotebookIds.has(nb.id);
            const label = document.createElement('span');
            label.textContent = nb.name;
            item.appendChild(cb);
            item.appendChild(label);
            // 点击整个 item 切换 checkbox
            item.addEventListener('click', (e) => {
                if (e.target.tagName === 'INPUT') return;
                cb.checked = !cb.checked;
                cb.dispatchEvent(new Event('change'));
            });
            // checkbox change 事件
            cb.addEventListener('change', async () => {
                const wasEnabled = enableCardRecall;         // 操作前召回开关状态
                const wasEmpty = recallNotebookIds.size === 0; // 操作前是否未选任何笔记本
                if (cb.checked) {
                    recallNotebookIds.add(nb.id);
                } else {
                    recallNotebookIds.delete(nb.id);
                }
                // 勾选导致召回从关闭变为开启：先校验量化配置与量化表
                if (cb.checked && wasEmpty && !wasEnabled) {
                    try {
                        const check = await window.go.main.App.ValidateCardRecall();
                        if (!check.ok) {
                            // 校验失败：回滚复选框与开关状态
                            cb.checked = false;
                            recallNotebookIds.delete(nb.id);
                            enableCardRecall = false;
                            if (cardRecallToggle) cardRecallToggle.classList.remove('active');
                            nm.show(check.message || '卡片召回校验未通过', 'warning');
                            return;
                        }
                    } catch (e) {
                        cb.checked = false;
                        recallNotebookIds.delete(nb.id);
                        enableCardRecall = false;
                        if (cardRecallToggle) cardRecallToggle.classList.remove('active');
                        nm.show('卡片召回校验失败: ' + e, 'error');
                        return;
                    }
                }
                // 如果全不选，卡片召回设为关闭；如果全选或有选，设为开启
                if (recallNotebookIds.size === 0) {
                    enableCardRecall = false;
                    if (cardRecallToggle) cardRecallToggle.classList.remove('active');
                } else {
                    enableCardRecall = true;
                    if (cardRecallToggle) cardRecallToggle.classList.add('active');
                }
                await saveCurrentSessionConfig();
            });
            cardRecallDropdown.appendChild(item);
        });
        // 每次打开菜单时，将滚动条重置到顶部
        cardRecallDropdown.scrollTop = 0;
    } catch (_) {}
}

/**
 * 暴露给 main.js 设置页用的全局同步方法：开→全选，关→全取消
 */
window.__syncRecallNotebooks = async function (isActive) {
    recallNotebookIds.clear();
    if (isActive) {
        try {
            const notebooks = await window.go.main.App.GetAllNotebooks();
            if (notebooks) {
                notebooks.forEach(nb => recallNotebookIds.add(nb.id));
            }
        } catch (_) {}
    }
    // 更新菜单复选框状态
    if (cardRecallDropdown) {
        cardRecallDropdown.querySelectorAll('.ai-chat-recall-item input[type="checkbox"]').forEach(cb => {
            cb.checked = isActive;
        });
    }
    // 同步工具栏 toggle
    if (cardRecallToggle) {
        cardRecallToggle.classList.toggle('active', isActive);
    }
    enableCardRecall = isActive;
    await saveCurrentSessionConfig();
};

/**
 * 语言选择浮层实例
 */
let langPickerEl = null;

/**
 * 打开语言选择浮层
 * @param {HTMLElement} anchorEl - 触发元素
 * @param {string} side - 'source' 或 'target'
 */
function openLangPicker(anchorEl, side) {
    // 移除已有浮层
    closeLangPicker();

    // 创建浮层
    const picker = document.createElement('div');
    picker.className = 'ai-chat-lang-picker';
    picker.id = 'aiChatLangPicker';

    const currentCode = activeSkills.translate[side];
    const otherSide = side === 'source' ? 'target' : 'source';
    const otherCode = activeSkills.translate[otherSide];

    LANG_CODES.forEach(code => {
        const item = document.createElement('div');
        item.className = 'ai-chat-lang-picker-item' + (code === currentCode ? ' selected' : '');
        item.dataset.langCode = code;

        const label = document.createElement('span');
        label.className = 'ai-chat-lang-label';
        label.textContent = getLanguageDisplayName(code);
        item.appendChild(label);

        item.addEventListener('click', (e) => {
            e.stopPropagation();
            if (code === otherCode) {
                // 选择了与另一侧相同的语言，自动交换
                activeSkills.translate[side] = code;
                activeSkills.translate[otherSide] = currentCode;
            } else {
                activeSkills.translate[side] = code;
            }
            renderSkillChips();
            saveCurrentSessionConfig();
            closeLangPicker();
        });

        picker.appendChild(item);
    });

    // 定位浮层
    document.body.appendChild(picker);

    // 获取位置
    const anchorRect = anchorEl.getBoundingClientRect();
    const pickerWidth = 140;
    let left = anchorRect.left + (anchorRect.width - pickerWidth) / 2;
    // 确保不溢出右侧
    if (left + pickerWidth > window.innerWidth - 8) {
        left = window.innerWidth - pickerWidth - 8;
    }
    if (left < 8) left = 8;

    picker.style.left = left + 'px';
    // 默认向上弹出；如果锚点太靠上（<260px），则向下弹出
    if (anchorRect.top < 260) {
        picker.style.top = (anchorRect.bottom + 4) + 'px';
    } else {
        picker.style.bottom = (window.innerHeight - anchorRect.top + 4) + 'px';
    }

    langPickerEl = picker;

    // 触发入场动画
    requestAnimationFrame(() => {
        picker.classList.add('open');
    });

    // 延迟绑定关闭事件，避免立即触发
    setTimeout(() => {
        document.addEventListener('click', closeLangPickerHandler, true);
    }, 0);
}

/**
 * 关闭语言选择浮层的 handler
 */
function closeLangPickerHandler(e) {
    if (langPickerEl && !langPickerEl.contains(e.target)) {
        closeLangPicker();
    }
}

/**
 * 关闭语言选择浮层
 */
function closeLangPicker() {
    if (langPickerEl) {
        langPickerEl.classList.remove('open');
        langPickerEl.remove();
        langPickerEl = null;
    }
    document.removeEventListener('click', closeLangPickerHandler, true);
}

