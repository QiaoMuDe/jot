import './css/index.css';
import { WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit, EventsOn, WindowFullscreen, WindowUnfullscreen, WindowIsFullscreen } from '../wailsjs/runtime/runtime.js';
import { marked } from 'marked';
import hljs from 'highlight.js';
import 'highlight.js/styles/github.css';

// CodeMirror 6 导入
import { EditorState, Compartment } from '@codemirror/state';
import { EditorView, lineNumbers, highlightActiveLineGutter, keymap, highlightSpecialChars, drawSelection, highlightActiveLine, placeholder, scrollPastEnd } from '@codemirror/view';
import { javascript } from '@codemirror/lang-javascript';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { searchKeymap, highlightSelectionMatches, openSearchPanel, setSearchQuery, SearchQuery } from '@codemirror/search';
import { closeBrackets, closeBracketsKeymap, completionKeymap, autocompletion } from '@codemirror/autocomplete';
import { bracketMatching, indentOnInput, foldGutter, foldKeymap } from '@codemirror/language';
import { jotTheme, getHighlightExtension, codeHighlightThemeNames, codeHighlightThemeLabels } from './js/cm6-syntax-highlight.js';

// 独立模块
import { SVGS, formatTime, highlightText, getSummary, debounce } from './js/constants.js';
import { NotificationManager, getMockNotes, getMockTags } from './js/notification.js';

// 数据管理模块
import { animateCountUp, loadDataStats, resetDatabase, vacuumDatabase, openDataDir, exportData, importData, loadBackupInfo, backupToDir, restoreFromDir } from './js/data-management.js';

// 回收站页面模块
import { loadTrashNotes } from './js/trash-page.js';
// restoreAllNotes, emptyTrash 等函数通过 window 全局暴露（供 HTML 模板 onclick 调用）

// AI 对话页面模块
import { initAIChat, onAIChatViewActivated } from './js/ai-chat.js';

// 配置 marked（breaks + gfm；代码高亮在 updatePreview 中通过 hljs 后处理实现）
marked.setOptions({
    breaks: true,
    gfm: true,
});




/* ===== CodeMirror 6 集成 ===== */

/**
 * CodeMirror 6 编辑器实例（全局单例）
 */
let cmEditor = null;
let cmReadOnlyCompartment = null;

/** 当前代码高亮主题名称 */
let codeHighlightTheme = 'monokai-dimmed';

/**
 * 初始化 CodeMirror 6 编辑器
 * @param {HTMLElement} container - 挂载容器
 * @param {string} content - 初始内容
 * @param {boolean} readOnly - 是否只读
 * @param {boolean} useSyntaxHighlight - 是否启用语法高亮
 * @param {string} [fileExt='.md'] - 文件扩展名（含前导点号），用于选择语言解析器
 * @returns {EditorView}
 */
function initCodeMirror(container, content = '', readOnly = false, useSyntaxHighlight = true, fileExt = '.md', themeName = 'monokai-dimmed') {
    // 每次初始化创建新的 Compartment（旧实例销毁后旧 compartment 随之失效）
    cmReadOnlyCompartment = new Compartment();
    const extensions = [
        lineNumbers(),
        highlightActiveLineGutter(),
        highlightActiveLine(),
        drawSelection(),
        highlightSpecialChars(),
        history(),
        bracketMatching(),
        indentOnInput(),
        foldGutter(),
        placeholder('在此输入笔记内容...'),
        scrollPastEnd(),
        keymap.of([
            ...defaultKeymap,
            ...historyKeymap,
            ...searchKeymap,
            ...closeBracketsKeymap,
            ...completionKeymap,
            ...foldKeymap,
            indentWithTab,

        ]),
        closeBrackets(),
        autocompletion(),
        ...(useSyntaxHighlight ? getHighlightExtension(fileExt, themeName) : []),
        highlightSelectionMatches(),
        EditorView.contentAttributes.of({ spellcheck: 'true' }),
        jotTheme,
        cmReadOnlyCompartment.of(EditorState.readOnly.of(readOnly)),
        // 监听内容变化以触发自动保存和字数更新
        EditorView.updateListener.of((update) => {
            if (update.docChanged) {
                onEditorInput();
            }
        }),
    ];

    const state = EditorState.create({
        doc: content || '',
        extensions,
    });

    // 销毁旧实例（防止重复初始化）
    if (cmEditor) {
        cmEditor.destroy();
    }

    cmEditor = new EditorView({
        state,
        parent: container,
    });

    return cmEditor;
}

/**
 * 切换 CM6 编辑器只读状态（不重建实例，避免闪烁）
 * @param {boolean} readOnly
 */
function setCMReadOnly(readOnly) {
    if (cmEditor && cmReadOnlyCompartment) {
        cmEditor.dispatch({
            effects: cmReadOnlyCompartment.reconfigure(EditorState.readOnly.of(readOnly))
        });
    }
}

/**
 * 内联切换编辑器查看/编辑模式，不重建 CM6 实例，避免闪烁
 * @param {boolean} readOnly - true=查看模式, false=编辑模式
 */
function switchEditorReadOnly(readOnly) {
    // 切换标题只读状态
    els.editorNoteTitle.readOnly = readOnly;
    els.editorNoteTitle.classList.toggle('editor-input-readonly', readOnly);
    // 切换按钮显隐
    els.editorSaveBtn.style.display = readOnly ? 'none' : '';
    els.editorCancelBtn.style.display = readOnly ? 'none' : '';
    els.editorPanel.classList.toggle('editor-view-mode', readOnly);
    if (els.editorTypeToggle) {
        els.editorTypeToggle.style.display = readOnly ? 'none' : '';
    }
    els.editorEditBtn.style.display = readOnly ? '' : 'none';
    els.editorViewBtn.style.display = (!readOnly && state.enteredFromViewMode) ? '' : 'none';
    els.editorFileExt.classList.toggle('file-ext-readonly', readOnly);
    // 切换标签选择器只读
    renderTagSelector(readOnly);
    // 切换 CM6 只读状态
    setCMReadOnly(readOnly);
    // Markdown 笔记：自动切换预览/编辑模式
    const isMd = els.editorFileExt.textContent === '.md';
    if (isMd && readOnly) {
        // 查看模式 → Markdown 预览
        els.editorOverlay.dataset.mode = 'preview';
        els.editorModeBtns.forEach(btn => {
            btn.classList.toggle('active', btn.dataset.mode === 'preview');
        });
        // 从 CM6 获取最新内容渲染预览
        if (getEditorContent().trim()) {
            els.mdRendered.innerHTML = marked.parse(getEditorContent());
            _applyPreviewDOMHelpers();
        } else {
            els.mdRendered.innerHTML = '<p class="md-empty">暂无内容</p>';
        }
    } else if (isMd) {
        // 编辑模式 → 切回纯文本编辑
        switchEditorMode('edit');
    }
    // 编辑模式记录快照，查看模式清除快照
    if (!readOnly) {
        state._editSnapshot = {
            title: els.editorNoteTitle.value.trim(),
            content: getEditorContent().trim(),
            tags: [...state.selectedTags].sort(),
            fileExt: els.editorFileExt.textContent
        };
    } else {
        state._editSnapshot = null;
    }
}

/**
 * 销毁 CodeMirror 6 实例
 */
function destroyCodeMirror() {
    if (cmEditor) {
        cmEditor.destroy();
        cmEditor = null;
    }
}

/**
 * 设置 CodeMirror 6 编辑器内容（替换全部文档）
 * @param {string} content - 新内容
 */
function setEditorContent(content) {
    if (cmEditor) {
        cmEditor.dispatch({
            changes: { from: 0, to: cmEditor.state.doc.length, insert: content || '' }
        });
    }
}

/** 全局通知管理器实例 */
const nm = new NotificationManager();

/* ===== 应用状态 ===== */
const state = {
    notes: [],
    tags: [],
    currentView: 'grid',       // grid | search | settings | data | trash
    _isFullscreen: false,
    editingNoteId: null,        // null = 新建, number = 编辑
    selectedTags: [],
    searchKeyword: '',
    searchSource: 'input',      // 'input' | 'tag' — 搜索触发来源
    batchMode: false,           // 是否处于批量管理模式
    selectedNoteIds: new Set(), // 选中的笔记 ID 集合
    totalAllNotes: 0,           // 所有未删除笔记的总数（用于全选判断）
    activeNotebookId: 1,        // 当前激活的笔记本 ID，默认为 1（默认笔记本）
    notebooks: [],              // 笔记本列表
    enteredFromViewMode: false, // 是否从查看模式点击编辑按钮进入编辑模式（控制返回按钮显示）
    _titleInputListenerAttached: false, // 编辑器标题 input 监听是否已绑定（用于清理）
    // 搜索弹窗状态(替代原 topbar 搜索)
    _searchModalPrevFocus: null,        // 弹窗打开前 document.activeElement
    searchModalKeyword: '',
    searchModalPage: 1,
    searchModalTotal: 0,
    searchModalHasMore: false,
    searchModalLoading: false,
    searchModalNotebookId: 0,
    searchModalTagIds: new Set(),
    searchModalDateStart: '',
    searchModalDateEnd: '',
    searchModalSelectedIndex: -1,
    searchModalSortBy: 'updated_at',
};


// 分页状态
let currentPage = 1;
let totalNotes = 0;
let isLoadingMore = false;
let hasMoreNotes = true;
let pageSize = 18;

/* ===== DOM 引用 ===== */
const $ = (id) => document.getElementById(id);

const els = {
    // 侧边栏
    searchInput: $('searchInput'), // [已废弃] topbar 搜索框已迁移到弹窗,所有引用应改为 openSearchModal()
    // 搜索弹窗(替代原 topbar 搜索框)
    searchModal: $('searchModal'),
    searchModalInput: $('searchModalInput'),
    searchModalResults: $('searchModalResults'),
    searchModalEmpty: $('searchModalEmpty'),
    searchModalEmptyTitle: $('searchModalEmptyTitle'),
    searchModalEmptyDesc: $('searchModalEmptyDesc'),
    searchModalFooter: $('searchModalFooter'),
    searchModalCount: $('searchModalCount'),
    searchModalNotebookBtn: $('searchModalNotebookBtn'),
    searchModalNotebookLabel: $('searchModalNotebookLabel'),
    searchModalNotebookDropdown: $('searchModalNotebookDropdown'),
    searchModalTagBtn: $('searchModalTagBtn'),
    searchModalTagLabel: $('searchModalTagLabel'),
    searchModalTagDropdown: $('searchModalTagDropdown'),
    searchModalDateBtn: $('searchModalDateBtn'),
    searchModalDateLabel: $('searchModalDateLabel'),
    searchModalDateDropdown: $('searchModalDateDropdown'),
    searchModalSortBtn: $('searchModalSortBtn'),
    searchModalSortLabel: $('searchModalSortLabel'),
    searchModalSortDropdown: $('searchModalSortDropdown'),
    // 更多菜单
    moreMenuBtn: $('moreMenuBtn'),
    moreMenu: $('moreMenu'),

    // 视图
    viewGrid: $('viewGrid'),
    viewSettings: $('viewSettings'),
    viewData: $('viewData'),
    viewTrash: $('viewTrash'),
    viewMdRef: $('viewMdRef'),
    viewAiChat: $('viewAiChat'),
    viewEditor: $('viewEditor'),
    editorPanel: $('editorPanel'),

    cardGrid: $('cardGrid'),
    skeletonGrid: $('skeletonGrid'),
    emptyNotes: $('emptyNotes'),

    // 编辑器
    editorOverlay: $('editorOverlay'),
    editorTitle: $('editorTitle'),
    editorNoteTitle: $('editorNoteTitle'),
    editorNoteContent: $('editorNoteContent'),
    tagSelector: $('tagSelector'),
    editorCloseBtn: $('editorCloseBtn'),
    editorEditBtn: $('editorEditBtn'),
    editorViewBtn: $('editorViewBtn'),
    editorFullscreenBtn: $('editorFullscreenBtn'),
    editorCancelBtn: $('editorCancelBtn'),
    editorSaveBtn: $('editorSaveBtn'),
    mdRendered: $('mdRendered'),
    editorModeBtns: document.querySelectorAll('.mode-btn'),
    editorModes: $('editorModes'),
    editorTypeToggle: $('editorTypeToggle'),

    // 格式化工具栏
    // 设置
    tagList: $('tagList'),
    quickNoteToggle: $('quickNoteToggle'),
    mdHighlightToggle: $('mdHighlightToggle'),
    noteOpenFullscreenToggle: $('noteOpenFullscreenToggle'),
    newTagName: $('newTagName'),
    newTagColor: $('newTagColor'),
    addTagBtn: $('addTagBtn'),
    // 字体设置
    fontFamilyTrigger: $('fontFamilyTrigger'),
    fontFamilyDisplay: $('fontFamilyDisplay'),
    fontFamilyDropdown: $('fontFamilyDropdown'),
    fontFamilySearch: $('fontFamilySearch'),
    // 主题设置（下拉菜单）
    themeControl: $('themeControl'),
    themeTrigger: $('themeTrigger'),
    themeDropdown: $('themeDropdown'),
    themeLabel: $('themeLabel'),
    // 分页设置
    pageSizeControl: $('pageSizeControl'),
    pageSizeIndicator: $('pageSizeIndicator'),
    pageSizeLabel: $('pageSizeLabel'),
    // 排序分段控件
    sortControl: $('sortControl'),
    sortIndicator: $('sortIndicator'),
    fontSizePresets: $('fontSizePresets'),
    fontSizeInput: $('fontSizeInput'),
    // 回收站
    trashList: $('trashList'),
    trashListInner: $('trashListInner'),
    trashBackBtn: $('trashBackBtn'),
    mdRefBackBtn: $('mdRefBackBtn'),
    restoreAllBtn: $('restoreAllBtn'),
    emptyTrashBtn: $('emptyTrashBtn'),

    // 数据管理
    dataBackBtn: $('dataBackBtn'),
    settingsBackBtn: $('settingsBackBtn'),
    exportDataBtn: $('exportDataBtn'),
    importDataBtn: $('importDataBtn'),
    importResult: $('importResult'),
    resetAllBtn: $('resetAllBtn'),
    vacuumDbBtn: $('vacuumDbBtn'),
    openDataDirBtn: $('openDataDirBtn'),
    dataContent: $('dataContent'),
    statTotalNotes: $('statTotalNotes'),
    statTotalTags: $('statTotalTags'),
    statTrashedNotes: $('statTrashedNotes'),
    statTotalNotebooks: $('statTotalNotebooks'),
    statDBSize: $('statDBSize'),
    statAISessions: $('statAISessions'),
    statAIMessages: $('statAIMessages'),

    // 备份还原
    backupBtn: $('backupBtn'),
    restoreBtn: $('restoreBtn'),
    backupInfo: $('backupInfo'),
    backupStatusText: $('backupStatusText'),

    // 字数统计
    editorWordCount: $('editorWordCount'),
    editorFileExt: $('editorFileExt'),
    editorEditTime: $('editorEditTime'),

    // 通知容器
    notificationContainer: $('notificationContainer'),

    // 右键菜单
    contextMenu: $('contextMenu'),

    // 确认对话框
    confirmDialog: $('confirmDialog'),
    confirmDialogMsg: $('confirmDialogMsg'),
    confirmOkBtn: $('confirmOkBtn'),
    confirmCancelBtn: $('confirmCancelBtn'),
    confirmThirdBtn: $('confirmThirdBtn'),

    // 主内容区（用于网格视图滚动）
    mainContent: $('mainContent'),

    // 批量操作
    batchBar: $('batchBar'),
    batchCount: $('batchCount'),
    batchDeleteBtn: $('batchDeleteBtn'),
    batchCancelBtn: $('batchCancelBtn'),
    batchSelectAllBtn: $('batchSelectAllBtn'),
    batchAddTagBtn: $('batchAddTagBtn'),
    batchRemoveTagBtn: $('batchRemoveTagBtn'),
    batchPinBtn: $('batchPinBtn'),
    batchTagOverlay: $('batchTagOverlay'),
    batchTagTitle: $('batchTagTitle'),
    batchTagList: $('batchTagList'),
    batchTagCloseBtn: $('batchTagCloseBtn'),
    batchTagFooter: $('batchTagFooter'),
    batchTagConfirmBtn: $('batchTagConfirmBtn'),

    // 浮动操作按钮
    fabGroup: $('fabGroup'),
    fabNewNote: $('fabNewNote'),
    backToTopBtn: $('backToTopBtn'),

    // 关于页面
    viewAbout: $('viewAbout'),
    aboutCloseBtn: $('aboutCloseBtn'),
    aboutVersion: $('aboutVersion'),
    aboutProjectLink: $('aboutProjectLink'),

    // 快捷键页面
    shortcutsView: $('shortcutsView'),
    shortcutsCloseBtn: $('shortcutsCloseBtn'),
    shortcutsBody: $('shortcutsBody'),

    // 笔记本侧栏
    notebookList: $('notebookList'),
    newNotebookBtn: $('newNotebookBtn'),
    notebookSidebar: $('notebookSidebar'),
    notebookSidebarToggle: $('notebookSidebarToggle'),

    // 移动到弹窗
    moveNotebookDialog: $('moveNotebookDialog'),
    moveNotebookList: $('moveNotebookList'),
    moveNotebookConfirm: $('moveNotebookConfirm'),
    moveNotebookCancel: $('moveNotebookCancel'),
    moveNotebookClose: $('moveNotebookClose'),
    moveNotebookEmpty: $('moveNotebookEmpty'),
    batchMoveBtn: $('batchMoveBtn'),

    // AI 配置
    aiBaseURL: $('aiBaseURL'),
    aiAPIKey: $('aiAPIKey'),
    aiProviderSelect: $('aiProviderSelect'),
    aiProviderTrigger: $('aiProviderTrigger'),
    aiProviderDropdown: $('aiProviderDropdown'),
    aiProviderLabel: $('aiProviderLabel'),
    aiModelTrigger: $('aiModelTrigger'),
    aiModelDropdown: $('aiModelDropdown'),
    aiModelLabel: $('aiModelLabel'),
    aiTestURLBtn: $('aiTestURLBtn'),
    aiAPIKeyToggle: $('aiAPIKeyToggle'),
    aiFetchModelsBtn: $('aiFetchModelsBtn'),
};

/**
 * 重置分页状态
 */
function resetPagination() {
    currentPage = 1;
    totalNotes = 0;
    isLoadingMore = false;
    hasMoreNotes = true;
    state.totalAllNotes = 0;
}

/* ===== 视图切换 ===== */

// 视图动画状态锁，防止快速切换导致动画冲突
let _viewAnimating = false;

/**
 * 切换右侧主内容区视图（带动画过渡）
 * 1. 当前视图添加 .view-exit，animationend 后隐藏
 * 2. 目标视图添加 .view-enter（通过 requestAnimationFrame 确保生效）
 */
function switchView(view) {
    // 视图映射
    const viewMap = {
        grid: els.viewGrid,
        settings: els.viewSettings,
        data: els.viewData,
        trash: els.viewTrash,
        'md-ref': els.viewMdRef,
        'ai-chat': els.viewAiChat,
    };
    const targetView = viewMap[view];
    if (!targetView || _viewAnimating) return;

    const currentViewEl = document.querySelector('.view.active');
    if (targetView === currentViewEl) return;

    // 切换视图时强制退出批量模式
    if (state.batchMode) {
        toggleBatchMode();
    }

    state.currentView = view;

    // 悬浮操作按钮仅在网格视图显示
    els.fabGroup.style.display = view === 'grid' ? '' : 'none';
    // 笔记本侧栏折叠按钮仅在网格视图显示
    els.notebookSidebarToggle.style.display = view === 'grid' ? '' : 'none';

    _viewAnimating = true;

    /**
     * 执行目标视图的进入动画
     */
    const showTargetView = () => {
        // 清除可能残留的内联 display 样式
        targetView.style.display = '';
        // 添加 active 类，通过 CSS 规则 .view.active 显示
        targetView.classList.add('active');

        // 加载对应视图的数据（异步，在动画期间并行加载）
        switch (view) {
            case 'settings':
                loadFontSettings();
                loadThemeSetting();
                loadSortSettings();
                loadPageSizeSetting();
                loadSyntaxHighlightSetting();
                initCodeHighlightThemeSettings();
                initCodePreview();
                loadTags();
                loadAISettings();
                // 每次进入设置页 Key 输入框默认隐藏
                els.aiAPIKey.type = 'password';
                const initEye = els.aiAPIKeyToggle.querySelector('.toggle-eye');
                const initEyeOff = els.aiAPIKeyToggle.querySelector('.toggle-eye-off');
                if (initEye) initEye.style.display = '';
                if (initEyeOff) initEyeOff.style.display = 'none';
                break;
            case 'data':
                loadDataStats();
                break;
            case 'trash':
                loadTrashNotes();
                break;
            case 'md-ref':
                try { renderMdRefCards(); } catch (e) { console.warn('MD 语法手册渲染失败:', e); }
                break;
            case 'ai-chat':
                // 使用 setTimeout 确保 DOM 已更新
                setTimeout(() => onAIChatViewActivated(), 50);
                break;
        }

        // 非网格视图自动折叠侧栏（设置/数据管理/回收站与笔记本切换无关）
        if (view !== 'grid' && !els.notebookSidebar?.classList.contains('collapsed')) {
            els.notebookSidebar.classList.add('collapsed');
            localStorage.setItem('jot_sidebar_collapsed', 'true');
            updateSidebarMenuItem();
        }

        // 使用 requestAnimationFrame 确保 class 切换在下一渲染帧生效
        requestAnimationFrame(() => {
            targetView.classList.add('view-enter');
            targetView.addEventListener('animationend', function onEnterEnd() {
                targetView.removeEventListener('animationend', onEnterEnd);
                targetView.classList.remove('view-enter');
                _viewAnimating = false;
            }, { once: true });
        });
    };

    if (currentViewEl) {
        // 当前视图执行退出动画
        currentViewEl.classList.add('view-exit');
        currentViewEl.addEventListener('animationend', function onExitEnd() {
            currentViewEl.removeEventListener('animationend', onExitEnd);
            currentViewEl.classList.remove('active', 'view-exit');
            currentViewEl.style.display = 'none';
            showTargetView();
        }, { once: true });
    } else {
        // 没有当前活跃视图，直接显示目标视图
        showTargetView();
    }
}

/* ===== Wails 后端调用封装 ===== */

/**
 * 加载笔记列表（第 1 页，重置分页）
 */
async function loadNotes() {
    // 显示骨架屏加载状态
    if (els.skeletonGrid) els.skeletonGrid.style.display = '';
    if (els.emptyNotes) els.emptyNotes.style.display = 'none';
    if (els.cardGrid) { els.cardGrid.style.display = 'none'; els.cardGrid.innerHTML = ''; }

    try {
        // 获取当前排序方式
        let sortBy = 'updated_at';
        const checkedRadio = document.querySelector('input[name="sortOrder"]:checked');
        if (checkedRadio) sortBy = checkedRadio.value;

        // 获取分页大小（当前选中按钮的值）
        const activeBtn = els.pageSizeControl?.querySelector('.segmented-btn.active');
        const savedPageSize = activeBtn ? parseInt(activeBtn.dataset.value, 10) : 20;
        pageSize = savedPageSize;

        // 重置分页状态
        resetPagination();

        // 加载第 1 页
        let result = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNotes) {
            result = await window.go.main.App.GetNotes(1, pageSize, sortBy, state.activeNotebookId);
        }

        if (result) {
            state.notes = result.items || [];
            totalNotes = result.total || 0;
            state.totalAllNotes = totalNotes;
            hasMoreNotes = state.notes.length < totalNotes;
            currentPage = 1;
        } else {
            // 降级：后端未绑定时使用模拟数据
            console.warn('GetNotes 未绑定，使用模拟数据');
            if (!mockNotes) {
                mockNotes = getMockNotes();
            }
            state.notes = mockNotes;
            totalNotes = state.notes.length;
            state.totalAllNotes = totalNotes;
            hasMoreNotes = false;
        }
    } catch (err) {
        console.error('加载笔记失败:', err);
        if (!mockNotes) {
            mockNotes = getMockNotes();
        }
        state.notes = mockNotes;
        totalNotes = state.notes.length;
        state.totalAllNotes = totalNotes;
        hasMoreNotes = false;
    }
    renderCardGrid();

}

/**
 * 加载更多笔记（追加到列表末尾，滚动懒加载）
 */
async function loadMoreNotes() {
    if (isLoadingMore || !hasMoreNotes) return;

    isLoadingMore = true;
    showLoadingIndicator(true);
    const loadStart = Date.now(); // 记录加载开始时间
    const prevCount = state.notes.length; // 记录追加前卡片数

    try {
        let sortBy = 'updated_at';
        const checkedRadio = document.querySelector('input[name="sortOrder"]:checked');
        if (checkedRadio) sortBy = checkedRadio.value;
        // 获取分页大小（当前选中按钮的值）
        const activeBtn = els.pageSizeControl?.querySelector('.segmented-btn.active');
        const savedPageSize = activeBtn ? parseInt(activeBtn.dataset.value, 10) : 20;
        pageSize = savedPageSize;
        const nextPage = currentPage + 1;

        let result = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNotes) {
            result = await window.go.main.App.GetNotes(nextPage, pageSize, sortBy, state.activeNotebookId);
        }

        if (result && result.items && result.items.length > 0) {
            state.notes = state.notes.concat(result.items);
            currentPage = nextPage;
            hasMoreNotes = state.notes.length < totalNotes;
        } else {
            hasMoreNotes = false;
        }

        renderCardGrid('append', prevCount);
    } catch (err) {
        console.error('加载更多笔记失败:', err);
    } finally {
        // 确保加载动画至少显示 1 秒，避免闪一下就消失
        const elapsed = Date.now() - loadStart;
        const minDisplay = 1000;
        if (elapsed < minDisplay) {
            await new Promise(r => setTimeout(r, minDisplay - elapsed));
        }
        isLoadingMore = false;
        showLoadingIndicator(false);
    }
}

/**
 * 加载所有剩余页面（Ctrl+End 使用）
 * 一次性请求所有未加载的页，合并后跳到底部
 */
async function loadAllRemainingNotes() {
    if (!hasMoreNotes || isLoadingMore) return;

    isLoadingMore = true;
    showLoadingIndicator(true);
    const prevCount = state.notes.length; // 记录追加前卡片数

    try {
        // 获取排序和分页参数
        let sortBy = 'updated_at';
        const checkedRadio = document.querySelector('input[name="sortOrder"]:checked');
        if (checkedRadio) sortBy = checkedRadio.value;
        const activeBtn = els.pageSizeControl?.querySelector('.segmented-btn.active');
        const savedPageSize = activeBtn ? parseInt(activeBtn.dataset.value, 10) : 20;
        pageSize = savedPageSize;

        // 计算剩余页数，逐一加载
        const loadedCount = state.notes.length;
        let totalPages = Math.ceil(totalNotes / pageSize);
        // 如果 total 未知（降级场景），直接取当前数据判断
        if (totalNotes === 0 && state.notes.length === 0) return;

        const remainingPages = [];
        for (let p = currentPage + 1; p <= totalPages; p++) {
            remainingPages.push(p);
        }

        // 逐一请求未加载的页
        let allNewItems = [];
        for (const page of remainingPages) {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNotes) {
                const result = await window.go.main.App.GetNotes(page, pageSize, sortBy, state.activeNotebookId);
                if (result && result.items && result.items.length > 0) {
                    allNewItems = allNewItems.concat(result.items);
                    currentPage = page;
                }
            }
        }

        // 合并数据
        if (allNewItems.length > 0) {
            state.notes = state.notes.concat(allNewItems);
        }
        hasMoreNotes = false;

        // 重新渲染并跳到底部
        renderCardGrid('append', prevCount);
        const container = getScrollContainer();
        if (container) {
            // 等待 DOM 更新后滚动到底部
            requestAnimationFrame(() => {
                container.scrollTop = container.scrollHeight;
            });
        }
    } catch (err) {
        console.error('加载全部失败:', err);
    } finally {
        isLoadingMore = false;
        showLoadingIndicator(false);
    }
}

/**
 * 显示/隐藏加载指示器
 */
function showLoadingIndicator(show) {
    let indicator = document.getElementById('loadingIndicator');
    if (show) {
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'loadingIndicator';
            indicator.className = 'loading-indicator';
            const spinner = document.createElement('span');
            spinner.className = 'loading-spinner';
            indicator.appendChild(spinner);
            indicator.appendChild(document.createTextNode('加载中...'));
            document.getElementById('viewGrid').querySelector('.card-grid').after(indicator);
        }
        indicator.style.display = 'flex';
    } else if (indicator) {
        indicator.style.display = 'none';
    }
}

/**
 * 创建笔记
 */
async function createNote() {
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    if (!title) {
        nm.show('标题不能为空，请输入标题后再保存', 'warning');
        return;
    }

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateNote) {
            const note = await window.go.main.App.CreateNote(title, content, els.editorFileExt.textContent, state.activeNotebookId);
            // 为笔记添加标签
            if (note && note.id && state.selectedTags.length > 0) {
                for (const tagId of state.selectedTags) {
                    try {
                        await window.go.main.App.AddTagToNote(note.id, tagId);
                    } catch (e) {
                        console.error('添加标签失败:', e);
                    }
                }
            }
        } else {
            console.warn('CreateNote 未绑定，模拟创建');
            state.notes.unshift({
                id: Date.now(),
                title: title,
                content: content,
                notebook_id: state.activeNotebookId,
                pinned: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                tags: state.tags.filter((t) => state.selectedTags.includes(t.id)),
            });
        }
    } catch (err) {
        console.error('创建笔记失败:', err);
    }
    nm.show('笔记已创建', 'success');
    closeEditor();
    await loadNotes();
    await loadNotebooks();
}

/**
 * 更新笔记
 */
async function updateNote(id) {
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    if (!title) {
        nm.show('标题不能为空，请输入标题后再保存', 'warning');
        return;
    }

    // 脏检测：有快照且内容无变更 → 跳过保存直接关闭
    const snapshot = state._editSnapshot;
    if (snapshot) {
        const currentTags = [...state.selectedTags].sort();
        const tagsChanged = JSON.stringify(currentTags) !== JSON.stringify(snapshot.tags);
        const extChanged = els.editorFileExt.textContent !== snapshot.fileExt;
        if (title === snapshot.title && content === snapshot.content && !tagsChanged && !extChanged) {
            closeEditor();
            return;
        }
    }

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.UpdateNote) {
            await window.go.main.App.UpdateNote(id, title, content, els.editorFileExt.textContent);
            // 更新笔记标签：先移除所有，再添加选中的
            const note = await window.go.main.App.GetNote(id);
            if (note && note.tags) {
                for (const t of note.tags) {
                    try { await window.go.main.App.RemoveTagFromNote(id, t.id); } catch (e) {}
                }
            }
            for (const tagId of state.selectedTags) {
                try { await window.go.main.App.AddTagToNote(id, tagId); } catch (e) {}
            }
        } else {
            console.warn('UpdateNote 未绑定，模拟更新');
            const note = mockNotes.find((n) => n.id === id);
            if (note) {
                note.title = title;
                note.content = content;
                note.updated_at = new Date().toISOString();
                note.tags = state.tags.filter((t) => state.selectedTags.includes(t.id));
            }
        }
    } catch (err) {
        console.error('更新笔记失败:', err);
    }
    nm.show('笔记已更新', 'success');
    closeEditor();
    await loadNotes();
    await loadNotebooks();
}

/**
 * 删除笔记（软删除，显示撤销 Toast）
 */
async function deleteNote(id) {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteNote) {
            await window.go.main.App.DeleteNote(id);
        } else {
            console.warn('DeleteNote 未绑定，模拟删除');
            mockNotes = mockNotes.filter((n) => n.id !== id);
        }
    } catch (err) {
        console.error('删除笔记失败:', err);
        return;
    }
    await loadNotes();
    await loadNotebooks();
    nm.showUndo('笔记已删除', () => undoDelete(id));
}

/**
 * 复制笔记内容到剪贴板
 */
async function copyNote(id) {
    const note = state.notes.find((n) => n.id === id);
    if (!note) return;
    const text = (note.title ? note.title + '\n\n' : '') + (note.content || '');
    try {
        await navigator.clipboard.writeText(text);
        nm.show('已复制到剪贴板', 'success');
    } catch (err) {
        console.error('复制失败:', err);
        nm.show('复制失败', 'error');
    }
}

/**
 * 导出笔记为 Markdown 文件
 */
window.exportNote = async function (id) {
    try {
        const result = await window.go.main.App.ExportNoteAsMarkdown(id);
        if (result && result !== '已取消') {
            nm.show(result, 'success');
        }
    } catch (err) {
        nm.show('导出失败', 'error');
        console.error('导出失败:', err);
    }
};



/**
 * 撤销删除（支持单条和批量）
 * @param {number|number[]} noteIds - 要恢复的笔记 ID
 */
async function undoDelete(noteIds) {
    if (noteIds == null) return;
    try {
        if (Array.isArray(noteIds)) {
            // 批量撤销
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchRestoreNotes) {
                await window.go.main.App.BatchRestoreNotes(noteIds);
            } else {
                console.warn('BatchRestoreNotes 未绑定');
            }
        } else {
            // 单条撤销
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreNote) {
                await window.go.main.App.RestoreNote(noteIds);
            }
        }
    } catch (err) {
        console.error('撤销删除失败:', err);
    }
    await loadNotes();
    await loadNotebooks();
}

/**
 * 显示自定义确认对话框
 * @param {string} msg - 提示信息
 * @returns {Promise<boolean>}
 */
function showConfirmDialog(msg) {
    return new Promise((resolve) => {
        // 确保第三方按钮在普通确认框中隐藏
        if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
        
        els.confirmDialogMsg.textContent = msg;
        els.confirmDialog.classList.add('visible');

        const cleanup = (result) => {
            els.confirmDialog.classList.remove('visible');
            // 恢复第三方按钮显示
            if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = '';
            resolve(result);
        };

        els.confirmOkBtn.onclick = () => cleanup(true);
        els.confirmCancelBtn.onclick = () => cleanup(false);
        els.confirmDialog.onclick = (e) => {
            if (e.target === els.confirmDialog) cleanup(false);
        };
    });
}

/** 笔记本侧栏键盘导航当前聚焦的索引（-1 表示无聚焦） */
let notebookFocusIndex = -1;

/**
 * 处理笔记本侧栏键盘导航（上下键移动选择，回车切换）
 */
function handleNotebookListKeydown(e) {
    const items = els.notebookList.querySelectorAll('.notebook-item');
    if (!items || items.length === 0) return;

    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        // 计算下一个索引
        const direction = e.key === 'ArrowDown' ? 1 : -1;
        if (notebookFocusIndex === -1) {
            // 首次聚焦，从当前 active 项开始，没有则从第一项开始
            const activeIdx = Array.from(items).findIndex(item => item.classList.contains('active'));
            notebookFocusIndex = activeIdx >= 0 ? activeIdx : 0;
        } else {
            notebookFocusIndex = Math.max(0, Math.min(items.length - 1, notebookFocusIndex + direction));
        }
        updateNotebookKeyboardFocus(items);
    } else if (e.key === 'Enter') {
        e.preventDefault();
        if (notebookFocusIndex < 0 || notebookFocusIndex >= items.length) return;
        const focusedItem = items[notebookFocusIndex];
        const id = parseInt(focusedItem.dataset.notebookId);
        if (id && id !== state.activeNotebookId) {
            switchNotebook(id);
        }
    }
}

/**
 * 更新笔记本列表的键盘聚焦高亮
 */
function updateNotebookKeyboardFocus(items) {
    if (!items) items = els.notebookList.querySelectorAll('.notebook-item');
    items.forEach((item, index) => {
        item.classList.toggle('keyboard-focus', index === notebookFocusIndex);
    });
    // 滚动聚焦项到视口内
    if (notebookFocusIndex >= 0 && items[notebookFocusIndex]) {
        items[notebookFocusIndex].scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
}

/**
 * 清除笔记本键盘聚焦（列表重新渲染后调用）
 */
function clearNotebookKeyboardFocus() {
    if (els.notebookList) {
        els.notebookList.querySelectorAll('.notebook-item.keyboard-focus').forEach(item => {
            item.classList.remove('keyboard-focus');
        });
    }
    notebookFocusIndex = -1;
}

/**
 * 显示保存确认对话框（退出程序前使用）
 * @param {string} msg - 提示信息
 * @returns {Promise<'save'|'discard'|'cancel'>}
 */
function showSaveConfirmDialog(msg) {
    return new Promise((resolve) => {
        if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = '';

        els.confirmDialogMsg.textContent = msg;
        els.confirmDialog.classList.add('visible');

        const cleanup = (result) => {
            els.confirmDialog.classList.remove('visible');
            // 恢复第三方按钮为隐藏状态,防止 showSaveConfirmDialog 残留的显式空值污染后续弹窗
            if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
            resolve(result);
        };

        els.confirmOkBtn.onclick = () => cleanup('save');
        els.confirmThirdBtn.onclick = () => cleanup('discard');
        els.confirmCancelBtn.onclick = () => cleanup('cancel');
        els.confirmDialog.onclick = (e) => {
            if (e.target === els.confirmDialog) cleanup('cancel');
        };
    });
}

/**
 * 置顶切换
 * 本地更新置顶状态后局部渲染网格，避免全量重新加载
 */
async function togglePin(id) {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.TogglePinNote) {
            await window.go.main.App.TogglePinNote(id);
        } else {
            console.warn('TogglePinNote 未绑定，模拟置顶切换');
        }
        // 本地切换置顶状态，避免重新加载全部笔记
        const note = state.notes.find((n) => n.id === id);
        if (note) {
            note.pinned = !note.pinned;
        }
    } catch (err) {
        console.error('置顶切换失败:', err);
    }
    // 仅重新渲染卡片网格（无动画），不重新请求后端
    await renderCardGrid('none');
}

/**
 * 加载标签
 */
async function loadTags() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllTags) {
            const tags = await window.go.main.App.GetAllTags();
            state.tags = tags || [];
        } else {
            console.warn('GetAllTags 未绑定');
            state.tags = getMockTags();
        }
    } catch (err) {
        console.error('加载标签失败:', err);
        state.tags = [];
    }
    renderTagList();
    renderTagSelector();
}

/**
 * 创建标签
 */
async function createTag() {
    const name = els.newTagName.value.trim();
    const color = els.newTagColor.value;
    if (!name) return;

    // 检查标签名是否已存在
    if (state.tags && state.tags.some(tag => tag.name === name)) {
        nm.show('该标签已存在', 'warning');
        els.newTagName.value = '';
        return;
    }

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateTag) {
            await window.go.main.App.CreateTag(name, color);
        } else {
            console.warn('CreateTag 未绑定');
        }
    } catch (err) {
        console.error('创建标签失败:', err);
        nm.show('创建标签失败', 'error');
    }
    els.newTagName.value = '';
    await loadTags();
    nm.show('标签已创建', 'success');
}

/* ===== 字体设置函数 ===== */

/**
 * 加载已保存的字体设置并应用到页面
 */
async function loadFontSettings() {
    let fontFamily = '';
    let fontSize = 16;

    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
        const savedFamily = await window.go.main.App.GetSetting('font_family');
        if (savedFamily) fontFamily = savedFamily;
        const savedSize = await window.go.main.App.GetSetting('font_size');
        if (savedSize) fontSize = parseInt(savedSize, 10);
    } else {
        // 从 localStorage 回退
        fontFamily = localStorage.getItem('jot_font_family') || '';
        fontSize = parseInt(localStorage.getItem('jot_font_size') || '16', 10);
    }

    applyFontFamily(fontFamily);
    applyFontSize(fontSize);
    updateFontSettingsUI(fontFamily, fontSize);
}

/**
 * 更新字体设置的 UI 状态
 */
function updateFontSettingsUI(fontFamily, fontSize) {
    // 更新字体族显示
    els.fontFamilyDisplay.textContent = fontFamily;

    // 更新大小预设高亮
    els.fontSizePresets.querySelectorAll('.font-size-btn').forEach(btn => {
        btn.classList.toggle('active', parseInt(btn.dataset.size, 10) === fontSize);
    });

    // 更新自定义输入框
    els.fontSizeInput.value = fontSize;
}

// 缓存全量字体列表，供搜索过滤使用
let fontFamilyList = [];

/**
 * 渲染字体族下拉选项（支持过滤）
 */
async function renderFontFamilyOptions(selectedFont, filterText) {
    // 首次调用时获取字体列表并缓存
    if (fontFamilyList.length === 0) {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSystemFonts) {
            fontFamilyList = await window.go.main.App.GetSystemFonts();
        } else {
            fontFamilyList = [
                'Arial', 'Helvetica', 'Verdana', 'Georgia',
                'Times New Roman', 'Courier New', 'Segoe UI', 'Microsoft YaHei',
                'PingFang SC', 'Noto Sans SC',
            ];
        }
        fontFamilyList = [...new Set(fontFamilyList)];
    }

    // 按过滤条件筛选
    const kw = (filterText || '').toLowerCase();
    const filtered = kw ? fontFamilyList.filter(f => f.toLowerCase().includes(kw)) : fontFamilyList;

    // 获取或创建选项容器
    let container = els.fontFamilyDropdown.querySelector('.font-family-options');
    if (!container) {
        container = document.createElement('div');
        container.className = 'font-family-options';
        els.fontFamilyDropdown.appendChild(container);
    }

    container.innerHTML = filtered.map(font => `
        <div class="font-family-option${font === selectedFont ? ' selected' : ''}"
             data-font="${font}"
             style="font-family: ${font}">
            ${highlightMatch(font, kw)}
        </div>
    `).join('') || '<div class="font-family-option disabled" style="font-style:italic">无匹配字体</div>';
}

/**
 * 高亮匹配文本
 */
function highlightMatch(text, keyword) {
    if (!keyword) return text;
    const idx = text.toLowerCase().indexOf(keyword);
    if (idx < 0) return text;
    return text.slice(0, idx) + '<strong style="font-weight:700">' + text.slice(idx, idx + keyword.length) + '</strong>' + text.slice(idx + keyword.length);
}

/**
 * 应用字体族
 */
function applyFontFamily(fontFamily) {
    if (fontFamily) {
        document.documentElement.style.setProperty('--font-family', `${fontFamily}, system-ui, -apple-system, sans-serif`);
        els.fontFamilyDisplay.textContent = fontFamily;
    } else {
        document.documentElement.style.removeProperty('--font-family');
        els.fontFamilyDisplay.textContent = '系统默认';
    }
}

/**
 * 应用字体大小
 */
function applyFontSize(size) {
    document.documentElement.style.setProperty('--font-size-base', `${size}px`);
}

/**
 * 保存字体设置到后端
 */
async function saveFontSetting(key, value) {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
        try {
            await window.go.main.App.SetSetting(key, value);
        } catch (err) {
            console.error('保存字体设置失败:', err);
            localStorage.setItem('jot_' + key, value);
        }
    } else {
        localStorage.setItem('jot_' + key, value);
    }
    nm.show('字体设置已保存', 'success');
}

/* ===== 主题设置函数 ===== */

/**
 * 应用指定主题
 * @param {string} themeName - 'default' | 'light' | 'dark'
 */
/** 代码高亮主题配对映射：系统主题 → 推荐代码高亮主题 */
const codeHighlightThemePairing = {
    'catppuccin-latte': 'catppuccin-mocha',
    'catppuccin-mocha': 'catppuccin-mocha',
    'gruvbox-dark': 'gruvbox-dark',
    'gruvbox-light': 'gruvbox-dark',
    'ayu-mirage': 'ayu-mirage',
    'dracula': 'dracula',
    'default': 'monokai-dimmed',
    'nord': 'monokai-dimmed',
    'monokai-pro': 'monokai-dimmed',
    'light': 'github-light',
    'tokyo-night': 'github-dark',
    'dark': 'github-dark',
};

const themeLabels = {
    'default': '默认',
    'catppuccin-latte': '暖咖',
    'nord': '北极',
    'gruvbox-light': '旧纸',
    'light': '浅色',
    'gruvbox-dark': '陈酿',
    'monokai-pro': '粉墨',
    'ayu-mirage': '暮光',
    'tokyo-night': '夜幕',
    'catppuccin-mocha': '暖夜',
    'dark': '深色',
    'dracula': '德古拉',
};

function applyTheme(themeName) {
    document.documentElement.setAttribute('data-theme', themeName);
    // 同步下拉菜单标签和选中态
    if (els.themeLabel) {
        els.themeLabel.textContent = themeLabels[themeName] || themeName;
    }
    if (els.themeDropdown) {
        els.themeDropdown.querySelectorAll('.theme-select-item').forEach(item => {
            item.classList.toggle('active', item.dataset.themeValue === themeName);
        });
    }
    // 更新代码高亮主题下拉菜单配对标记
    updateCodeHighlightThemePairing(themeName);
}

/**
 * 更新代码高亮主题下拉菜单中的配对标记
 * 在推荐配对项旁添加星标提示
 * @param {string} themeName - 当前系统主题名称
 */
function updateCodeHighlightThemePairing(themeName) {
    const dropdown = document.getElementById('codeHighlightThemeDropdown');
    if (!dropdown) return;
    const paired = codeHighlightThemePairing[themeName];
    dropdown.querySelectorAll('.theme-select-item').forEach(item => {
        const val = item.dataset.themeValue;
        // 移除旧标记
        item.innerHTML = item.innerHTML.replace(/^✦\s*/, '');
        // 添加配对标记
        if (val === paired && paired) {
            item.innerHTML = '✦ ' + item.innerHTML;
        }
    });
}

/**
 * 获取当前主题名称
 */
function getCurrentTheme() {
    return document.documentElement.getAttribute('data-theme') || 'default';
}

/**
 * 从后端加载已保存的主题设置并应用
 */
async function loadThemeSetting() {
    let theme = 'default';
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
        const saved = await window.go.main.App.GetSetting('theme');
        if (saved) theme = saved;
    } else {
        theme = localStorage.getItem('jot_theme') || 'default';
    }
    applyTheme(theme);
}

/**
 * 保存主题设置到后端
 */
async function saveThemeSetting(themeName) {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
        try {
            await window.go.main.App.SetSetting('theme', themeName);
        } catch (err) {
            console.error('保存主题设置失败:', err);
            localStorage.setItem('jot_theme', themeName);
        }
    } else {
        localStorage.setItem('jot_theme', themeName);
    }
    nm.show('主题设置已保存', 'success');
}

let _themeInited = false;

/**
 * 初始化主题设置下拉菜单事件
 */
function initThemeSettings() {
    if (_themeInited) return;
    _themeInited = true;

    const trigger = els.themeTrigger;
    const dropdown = els.themeDropdown;
    const items = dropdown?.querySelectorAll('.theme-select-item');
    if (!trigger || !dropdown || !items) return;

    // 点击触发按钮切换下拉菜单
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        if (dropdown.children.length === 0) return;
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
    });

    // 点击选项切换主题
    items.forEach(item => {
        item.addEventListener('click', async () => {
            const theme = item.dataset.themeValue;
            if (!theme) return;
            dropdown.classList.remove('open');
            trigger.classList.remove('open');
            applyTheme(theme);
            await saveThemeSetting(theme);
        });
    });

    // 点击外部关闭下拉菜单
    document.addEventListener('click', () => {
        dropdown.classList.remove('open');
        trigger.classList.remove('open');
    });
}

/**
 * 初始化字体设置下拉菜单事件
 */
function initFontSettings() {
    // 打开/关闭字体下拉菜单
    els.fontFamilyTrigger.addEventListener('click', (e) => {
        e.stopPropagation();
        const isOpen = els.fontFamilyDropdown.classList.contains('open');
        closeFontFamilyDropdown();
        if (!isOpen) {
            els.fontFamilyDropdown.classList.add('open');
            els.fontFamilyTrigger.classList.add('open');
            // 聚焦搜索框并清空搜索
            els.fontFamilySearch.value = '';
            els.fontFamilySearch.focus();
            renderFontFamilyOptions(getCurrentFontFamily(), '');
            // 滚动选项容器到顶部
            const container = els.fontFamilyDropdown.querySelector('.font-family-options');
            if (container) container.scrollTop = 0;
        }
    });

    // 搜索输入实时过滤
    let searchTimer = null;
    els.fontFamilySearch.addEventListener('input', () => {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => {
            renderFontFamilyOptions(getCurrentFontFamily(), els.fontFamilySearch.value);
        }, 100);
    });

    // 搜索框键盘导航
    els.fontFamilySearch.addEventListener('keydown', (e) => {
        const container = els.fontFamilyDropdown.querySelector('.font-family-options');
        if (!container) return;
        const items = container.querySelectorAll('.font-family-option:not(.disabled)');
        const currentIndex = Array.from(items).findIndex(item => item.classList.contains('highlighted'));

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                items.forEach(item => item.classList.remove('highlighted'));
                const nextIndex = Math.min(currentIndex + 1, items.length - 1);
                if (items[nextIndex]) {
                    items[nextIndex].classList.add('highlighted');
                    items[nextIndex].scrollIntoView({ block: 'nearest' });
                }
                break;
            case 'ArrowUp':
                e.preventDefault();
                items.forEach(item => item.classList.remove('highlighted'));
                const prevIndex = Math.max(currentIndex - 1, 0);
                if (items[prevIndex]) {
                    items[prevIndex].classList.add('highlighted');
                    items[prevIndex].scrollIntoView({ block: 'nearest' });
                }
                break;
            case 'Enter':
                e.preventDefault();
                if (currentIndex >= 0 && items[currentIndex]) {
                    items[currentIndex].click();
                }
                break;
            case 'Escape':
                e.preventDefault();
                closeFontFamilyDropdown();
                break;
        }
    });

    // 点击选项
    els.fontFamilyDropdown.addEventListener('click', (e) => {
        const option = e.target.closest('.font-family-option');
        if (!option || option.classList.contains('disabled')) return;
        const font = option.dataset.font;
        applyFontFamily(font);
        updateFontSettingsUI(font, getCurrentFontSize());
        saveFontSetting('font_family', font);
        closeFontFamilyDropdown();
    });

    // 点击外部关闭
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.font-family-select')) {
            closeFontFamilyDropdown();
        }
    });

    // 字体大小预设
    els.fontSizePresets.addEventListener('click', (e) => {
        const btn = e.target.closest('.font-size-btn');
        if (!btn) return;
        const size = parseInt(btn.dataset.size, 10);
        applyFontSize(size);
        updateFontSettingsUI(getCurrentFontFamily(), size);
        saveFontSetting('font_size', String(size));
    });

    // 自定义字体大小输入
    els.fontSizeInput.addEventListener('change', () => {
        const size = parseInt(els.fontSizeInput.value, 10);
        if (isNaN(size) || size < 10 || size > 32) {
            els.fontSizeInput.value = getCurrentFontSize();
            return;
        }
        applyFontSize(size);
        updateFontSettingsUI(getCurrentFontFamily(), size);
        saveFontSetting('font_size', String(size));
    });
}

/**
 * 关闭字体族下拉菜单
 */
function closeFontFamilyDropdown() {
    els.fontFamilyDropdown.classList.remove('open');
    els.fontFamilyTrigger.classList.remove('open');
}

/**
 * 获取当前字体族
 */
function getCurrentFontFamily() {
    return els.fontFamilyDisplay.textContent;
}

/**
 * 获取当前字体大小
 */
function getCurrentFontSize() {
    const size = document.documentElement.style.getPropertyValue('--font-size-base');
    return parseInt(size, 10) || 16;
}

/* ===== 排序和分页设置函数 ===== */

/**
 * 初始化排序和分页设置
 */
async function initSortSettings() {
    // 加载排序和分页设置
    await loadSortSettings();
    await loadPageSizeSetting();
    // 绑定排序分段控件事件
    if (els.sortControl) {
        const moveIndicator = (btn) => {
            const btns = Array.from(els.sortControl.querySelectorAll('.segmented-btn'));
            const index = btns.indexOf(btn);
            if (index >= 0) {
                const cw = els.sortControl.offsetWidth;
                const segW = (cw - 4) / btns.length;
                els.sortIndicator.style.transform = `translateX(${2 + index * segW}px)`;
            }
        };
        els.sortControl.querySelectorAll('.segmented-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const order = btn.dataset.sortValue;
                if (!order) return;
                // 更新 active 状态和指示器位置
                els.sortControl.querySelectorAll('.segmented-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                moveIndicator(btn);
                if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSortOrder) {
                    await window.go.main.App.SetSortOrder(order);
                    nm.show('排序方式已保存', 'success');
                }
                resetPagination();
                await loadNotes();
            });
        });
    }
    // 绑定分页大小分段控件事件
    if (els.pageSizeControl) {
        const moveIndicator = (btn) => {
            const btns = Array.from(els.pageSizeControl.querySelectorAll('.segmented-btn'));
            const index = btns.indexOf(btn);
            if (index >= 0) {
                const cw = els.pageSizeControl.offsetWidth;
                const segW = (cw - 4) / btns.length;
                els.pageSizeIndicator.style.transform = `translateX(${2 + index * segW}px)`;
            }
        };
        els.pageSizeControl.querySelectorAll('.segmented-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const size = parseInt(btn.dataset.value, 10);
                // 更新 active 状态和指示器位置
                els.pageSizeControl.querySelectorAll('.segmented-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                moveIndicator(btn);
                els.pageSizeLabel.textContent = `${size} 条 / 页`;
                if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetPageSize) {
                    await window.go.main.App.SetPageSize(size);
                    nm.show('分页大小已保存', 'success');
                }
                resetPagination();
                await loadNotes();
            });
        });
    }
}

// ── AI 配置初始化 ──
async function loadAISettings() {
    try {
        const cfg = await window.go.main.App.GetAIConfig();
        els.aiBaseURL.value = cfg.base_url || '';
        els.aiAPIKey.value = cfg.api_key || '';
        const provider = cfg.provider || 'openai';
        // 设置服务商下拉选中态
        if (els.aiProviderDropdown) {
            const items = els.aiProviderDropdown.querySelectorAll('.theme-select-item');
            items.forEach(item => item.classList.toggle('active', item.dataset.providerValue === provider));
        }
        if (els.aiProviderLabel) {
            const labels = { openai: 'OpenAI 兼容', ollama: 'Ollama' };
            els.aiProviderLabel.textContent = labels[provider] || 'OpenAI 兼容';
        }
        // 仅更新"获取列表"按钮状态，不覆盖已保存的 URL
        const canFetch = provider === 'openai' || provider === 'ollama';
        if (els.aiFetchModelsBtn) {
            els.aiFetchModelsBtn.disabled = !canFetch;
            els.aiFetchModelsBtn.style.opacity = canFetch ? '' : '0.5';
            els.aiFetchModelsBtn.title = canFetch ? '' : '该服务商不支持获取模型列表';
        }
    } catch (e) {
        console.warn('loadAISettings: provider/URL loading error', e);
    }

    // 独立加载模型：避免前面 UI 异常导致模型设置丢失
    let cfg;
    try {
        cfg = await window.go.main.App.GetAIConfig();
        // 仅清除模型列表项，保留搜索框
        els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
        if (cfg.model) {
            els.aiModelLabel.textContent = cfg.model;
            addModelDropdownItem(cfg.model, true);
        }
    } catch (e) {
        console.warn('loadAISettings: model loading error', e);
    }
    // 根据模型数量控制搜索框可见性
    const loadWrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
    if (loadWrap) {
        loadWrap.style.display = els.aiModelDropdown.querySelectorAll('.theme-select-item').length > 1 ? '' : 'none';
    }

    // 深度思考状态
    const searchToggle = document.getElementById('aiSettingSearchToggle');
    if (searchToggle) {
        let enabled = false;
        try {
            const val = await window.go.main.App.GetSetting('ai_thinking_enabled');
            if (val !== '') enabled = val === 'true';
            else enabled = localStorage.getItem('ai_thinking_enabled') === 'true';
        } catch (_) {
            enabled = localStorage.getItem('ai_thinking_enabled') === 'true';
        }
        if (enabled) searchToggle.classList.add('active');
    }

    // 联网搜索配置
    const tavilyKey = document.getElementById('aiTavilyApiKey');
    if (tavilyKey && cfg.tavily_api_key) {
        tavilyKey.value = cfg.tavily_api_key;
    }
    const webSearchToggle = document.getElementById('aiSettingWebSearchToggle');
    if (webSearchToggle) {
        let webSearchEnabled = false;
        try {
            const val = await window.go.main.App.GetSetting('ai_web_search_enabled');
            if (val !== '') webSearchEnabled = val === 'true';
            else webSearchEnabled = localStorage.getItem('ai_web_search_enabled') === 'true';
        } catch (_) {
            webSearchEnabled = localStorage.getItem('ai_web_search_enabled') === 'true';
        }
        if (webSearchEnabled) webSearchToggle.classList.add('active');
    }

    // 卡片召回配置
    const cardRecallToggle = document.getElementById('aiSettingCardRecallToggle');
    if (cardRecallToggle) {
        let cardRecallEnabled = false;
        try {
            const val = await window.go.main.App.GetSetting('ai_card_recall_enabled');
            if (val !== '') cardRecallEnabled = val === 'true';
            else cardRecallEnabled = localStorage.getItem('ai_card_recall_enabled') === 'true';
        } catch (_) {
            cardRecallEnabled = localStorage.getItem('ai_card_recall_enabled') === 'true';
        }
        if (cardRecallEnabled) cardRecallToggle.classList.add('active');
    }
    const cardRecallLimit = document.getElementById('aiSettingCardRecallLimit');
    if (cardRecallLimit) {
        try {
            const val = await window.go.main.App.GetSetting('ai_card_recall_limit');
            if (val) {
                cardRecallLimit.value = val;
            } else {
                const saved = localStorage.getItem('ai_card_recall_limit');
                if (saved) cardRecallLimit.value = saved;
            }
        } catch (_) {
            const saved = localStorage.getItem('ai_card_recall_limit');
            if (saved) cardRecallLimit.value = saved;
        }
    }

    // 引用截断字数
    const refMaxChars = document.getElementById('aiRefMaxChars');
    if (refMaxChars) {
        try {
            const val = await window.go.main.App.GetAIRefMaxChars();
            refMaxChars.value = val;
        } catch (_) { /* 使用 HTML 默认值 1000 */ }
    }

    // 联网搜索结果数
    const searchResultLimit = document.getElementById('aiSearchResultLimit');
    if (searchResultLimit) {
        try {
            const val = await window.go.main.App.GetAISearchResultLimit();
            searchResultLimit.value = val;
        } catch (_) { /* 使用 HTML 默认值 5 */ }
    }

    // 加载预设列表
    await loadProfiles();
}

// ── 全局 AI 辅助函数 ──
function getActiveProvider() {
    const dropdown = els.aiProviderDropdown;
    const active = dropdown?.querySelector('.theme-select-item.active');
    return active?.dataset.providerValue || 'openai';
}

function setActiveProvider(value) {
    const labels = { openai: 'OpenAI 兼容', ollama: 'Ollama' };
    const dropdown = els.aiProviderDropdown;
    dropdown.querySelectorAll('.theme-select-item').forEach(i => i.classList.remove('active'));
    const target = dropdown?.querySelector(`[data-provider-value="${value}"]`);
    if (target) target.classList.add('active');
    if (els.aiProviderLabel) els.aiProviderLabel.textContent = labels[value] || value;
}

function updateProviderUI() {
    const provider = getActiveProvider();
    const defaultURLs = {
        openai: 'https://api.openai.com/v1',
        ollama: 'http://localhost:11434',
    };
    if (els.aiBaseURL && defaultURLs[provider]) {
        els.aiBaseURL.value = defaultURLs[provider];
    }
    const canFetch = provider === 'openai' || provider === 'ollama';
    if (els.aiFetchModelsBtn) {
        els.aiFetchModelsBtn.disabled = !canFetch;
        els.aiFetchModelsBtn.style.opacity = canFetch ? '' : '0.5';
        els.aiFetchModelsBtn.title = canFetch ? '' : '该服务商不支持获取模型列表';
    }
}

async function initAISettings() {
    await loadAISettings();

    // ---- 模型下拉菜单事件 ----
    const trigger = els.aiModelTrigger;
    const dropdown = els.aiModelDropdown;

    // 点击触发按钮切换下拉菜单
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        const hasItems = dropdown.querySelectorAll('.theme-select-item').length > 0;
        if (!hasItems) return;
        const wasOpen = dropdown.classList.contains('open');
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
        if (wasOpen) {
            clearSettingModelSearch();
        } else {
            // 打开后聚焦搜索框
            setTimeout(() => {
                const search = dropdown.querySelector('.ai-model-search');
                if (search) search.focus();
            }, 50);
        }
    });

    // 点击模型项
    // ── 自动保存 ▸ 选择模型 ──
    dropdown.addEventListener('click', (e) => {
        const item = e.target.closest('.theme-select-item');
        if (!item) return;
        const model = item.dataset.modelValue;
        if (!model) return;
        dropdown.querySelectorAll('.theme-select-item').forEach(i => i.classList.remove('active'));
        item.classList.add('active');
        els.aiModelLabel.textContent = model;
        dropdown.classList.remove('open');
        trigger.classList.remove('open');
        clearSettingModelSearch();
        saveAIConfig();
    });

    // 点击外部关闭
    document.addEventListener('click', (e) => {
        if (!trigger.contains(e.target) && !dropdown.contains(e.target)) {
            dropdown.classList.remove('open');
            trigger.classList.remove('open');
            clearSettingModelSearch();
        }
    });

    // 设置页模型搜索过滤 + 关键字高亮
    const settingSearch = document.getElementById('aiSettingModelSearch');
    if (settingSearch) {
        settingSearch.addEventListener('input', () => {
            const query = settingSearch.value.trim();
            dropdown.querySelectorAll('.theme-select-item').forEach(item => {
                const model = item.dataset.modelValue;
                if (!query) {
                    item.textContent = model;
                    item.style.display = '';
                    return;
                }
                const lowerModel = model.toLowerCase();
                const lowerQuery = query.toLowerCase();
                const idx = lowerModel.indexOf(lowerQuery);
                if (idx !== -1) {
                    const before = model.substring(0, idx);
                    const match = model.substring(idx, idx + query.length);
                    const after = model.substring(idx + query.length);
                    item.innerHTML = before + '<mark class="ai-search-highlight">' + match + '</mark>' + after;
                    item.style.display = '';
                } else {
                    item.textContent = model;
                    item.style.display = 'none';
                }
            });
        });
        settingSearch.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                dropdown.classList.remove('open');
                trigger.classList.remove('open');
                clearSettingModelSearch();
            }
            if (e.key === 'Enter') e.preventDefault();
        });
    }

    function clearSettingModelSearch() {
        const search = document.getElementById('aiSettingModelSearch');
        if (search) {
            search.value = '';
            // 恢复所有 item 的 textContent（清除可能的 innerHTML 高亮）
            document.querySelectorAll('.theme-select-item').forEach(item => {
                const model = item.dataset.modelValue || item.dataset.model;
                if (model) item.textContent = model;
                item.style.display = '';
            });
        }
    }

    // ── 服务商下拉菜单事件 ──
    const provTrigger = els.aiProviderTrigger;
    const provDropdown = els.aiProviderDropdown;

    if (provTrigger && provDropdown) {
        // 点击触发按钮切换下拉菜单
        provTrigger.addEventListener('click', (e) => {
            e.stopPropagation();
            provTrigger.classList.toggle('open');
            provDropdown.classList.toggle('open');
        });

        // 点击选项
        provDropdown.addEventListener('click', (e) => {
            const item = e.target.closest('.theme-select-item');
            if (!item) return;
            const value = item.dataset.providerValue;
            if (!value) return;
            setActiveProvider(value);
            provDropdown.classList.remove('open');
            provTrigger.classList.remove('open');
            // 更新 UI 并保存
            updateProviderUI();
            // 切换服务商时：如果 URL 为空或匹配旧默认值，则填入新默认 URL
            const defaultURLs = {
                openai: 'https://api.openai.com/v1',
                ollama: 'http://localhost:11434',
            };
            const oldDefaults = { openai: 'https://api.openai.com/v1', ollama: 'http://localhost:11434' };
            const currentURL = els.aiBaseURL.value.trim();
            // 反向映射：判断当前 URL 是否属于某个服务商的默认值
            const isOldDefault = Object.values(oldDefaults).includes(currentURL);
            if (!currentURL || isOldDefault) {
                els.aiBaseURL.value = defaultURLs[value] || currentURL;
            }
            els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
            els.aiModelLabel.textContent = '-- 请先获取模型列表 --';
            const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
            if (wrap) wrap.style.display = 'none';
            const cfg = { base_url: els.aiBaseURL.value.trim(), api_key: els.aiAPIKey.value.trim(), model: '', provider: value };
            window.go.main.App.SaveAIConfig(cfg)
                .then(() => nm.show('AI 配置已保存', 'success'))
                .catch(() => {});
        });

        // 点击外部关闭
        document.addEventListener('click', () => {
            provDropdown.classList.remove('open');
            provTrigger.classList.remove('open');
        });
    }

    // ── 按钮加载状态 ──
    function setBtnLoading(btn, loading, label) {
        if (loading) {
            btn.dataset.origText = btn.textContent;
            btn.classList.add('btn-loading');
            btn.disabled = true;
            // 注入 spinner SVG
            const spinner = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            spinner.setAttribute('class', 'btn-spinner');
            spinner.setAttribute('viewBox', '0 0 24 24');
            spinner.setAttribute('fill', 'none');
            spinner.innerHTML = '<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-dasharray="31.4 31.4" opacity="0.3"/>' +
                '<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-dasharray="31.4 31.4" stroke-dashoffset="-10" opacity="0.85"/>';
            btn.prepend(spinner);
            btn.childNodes[1].textContent = label || '处理中…';
        } else {
            btn.classList.remove('btn-loading');
            btn.disabled = false;
            const spinner = btn.querySelector('.btn-spinner');
            if (spinner) spinner.remove();
            if (btn.dataset.origText) btn.textContent = btn.dataset.origText;
            delete btn.dataset.origText;
        }
    }

    // 测试 URL 连通性
    els.aiTestURLBtn.addEventListener('click', async () => {
        const url = els.aiBaseURL.value.trim();
        const key = els.aiAPIKey.value.trim();
        const provider = getActiveProvider();
        if (!url) {
            nm.show('请先填写 API 地址', 'warning');
            return;
        }
        if (provider === 'openai' && !key) {
            nm.show('请先填写 API Key', 'warning');
            return;
        }
        setBtnLoading(els.aiTestURLBtn, true, '测试中…');
        try {
            const ok = await window.go.main.App.TestAIBaseURL(url, key);
            if (ok) {
                nm.show('API 地址连接成功', 'success');
            } else {
                nm.show('API 地址连接失败，请检查地址', 'warning');
            }
        } catch (e) {
            nm.show('连接失败: ' + e, 'error');
        } finally {
            setBtnLoading(els.aiTestURLBtn, false);
        }
    });

    // API Key 显示/隐藏切换
    els.aiAPIKeyToggle.addEventListener('click', () => {
        const input = els.aiAPIKey;
        const eye = els.aiAPIKeyToggle.querySelector('.toggle-eye');
        const eyeOff = els.aiAPIKeyToggle.querySelector('.toggle-eye-off');
        if (input.type === 'password') {
            input.type = 'text';
            eye.style.display = 'none';
            eyeOff.style.display = '';
        } else {
            input.type = 'password';
            eye.style.display = '';
            eyeOff.style.display = 'none';
        }
    });

    // 获取模型列表
    els.aiFetchModelsBtn.addEventListener('click', async () => {
        const url = els.aiBaseURL.value.trim();
        const key = els.aiAPIKey.value.trim();
        const provider = getActiveProvider();
        if (!url) {
            nm.show('请先填写 API 地址', 'warning');
            return;
        }
        if (provider === 'openai' && !key) {
            nm.show('请先填写 API Key', 'warning');
            return;
        }
        setBtnLoading(els.aiFetchModelsBtn, true, '获取中…');
        try {
            const models = await window.go.main.App.FetchAIModels(url, key);
            if (models && models.length > 0) {
                // 仅清除模型列表项，保留搜索框
                els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
                const savedModel = (await window.go.main.App.GetAIConfig()).model;
                for (const m of models) {
                    addModelDropdownItem(m, m === savedModel);
                }
                // 将第一个模型设为标签并保存，避免"显示了但没保存"
                els.aiModelLabel.textContent = models[0];
                saveAIConfig();
                // 根据模型数量控制搜索框可见性
                const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
                if (wrap) wrap.style.display = models.length > 1 ? '' : 'none';
                nm.show(`已获取 ${models.length} 个模型`, 'success');
            } else {
                nm.show('未获取到可用模型', 'warning');
            }
        } catch (e) {
            nm.show('获取模型列表失败: ' + e, 'error');
        } finally {
            setBtnLoading(els.aiFetchModelsBtn, false);
        }
    });

    // ── 保存 AI 配置 ──
    async function saveAIConfig() {
        const url = els.aiBaseURL.value.trim();
        const key = els.aiAPIKey.value.trim();
        const model = els.aiModelLabel.textContent;
        const provider = getActiveProvider();
        const hasItems = els.aiModelDropdown.children.length > 0;

        if (!url || !key) return; // 未填完不保存
        if (!hasItems || model === '-- 请先获取模型列表 --' || !model) return;

        try {
            await window.go.main.App.SaveAIConfig({ base_url: url, api_key: key, model, provider, tavily_api_key: document.getElementById('aiTavilyApiKey')?.value?.trim() || '' });
            nm.show('AI 配置已保存', 'success');
            // 刷新预设下拉（可能自动创建了默认配置）
            loadProfiles();
        } catch (e) {
            nm.show('保存配置失败: ' + e, 'error');
        }
    }

    // ── 自动保存 ▸ URL 输入完成（独立保存，不依赖其他字段） ──
    els.aiBaseURL.addEventListener('change', async () => {
        const url = els.aiBaseURL.value.trim();
        try {
            const cfg = await window.go.main.App.GetAIConfig();
            cfg.base_url = url;
            cfg.provider = getActiveProvider();
            await window.go.main.App.SaveAIConfig(cfg);
            nm.show('AI 配置已保存', 'success');
        } catch (e) {
            nm.show('保存配置失败: ' + e, 'error');
        }
    });
    // ── 自动保存 ▸ Key 输入完成（独立保存，不依赖其他字段） ──
    els.aiAPIKey.addEventListener('change', async () => {
        const key = els.aiAPIKey.value.trim();
        try {
            const cfg = await window.go.main.App.GetAIConfig();
            cfg.api_key = key;
            cfg.provider = getActiveProvider();
            await window.go.main.App.SaveAIConfig(cfg);
            nm.show('AI 配置已保存', 'success');
        } catch (e) {
            nm.show('保存配置失败: ' + e, 'error');
        }
    });

    // 深度思考切换
    const settingSearchLine = document.getElementById('aiSettingSearchLine');
    if (settingSearchLine) {
        settingSearchLine.addEventListener('click', async () => {
            const toggleSwitch = document.getElementById('aiSettingSearchToggle');
            if (!toggleSwitch) return;
            const isActive = toggleSwitch.classList.toggle('active');
            localStorage.setItem('ai_thinking_enabled', String(isActive));
            try { await window.go.main.App.SetSetting('ai_thinking_enabled', String(isActive)); } catch (_) {}
            nm.show(isActive ? '深度思考已开启' : '深度思考已关闭', isActive ? 'success' : 'info');
            // 同步工具栏 toggle
            const toolbarToggle = document.getElementById('aiChatSearchToggle');
            if (toolbarToggle) {
                toolbarToggle.classList.toggle('active', isActive);
            }
        });
    }

    // ── Tavily 注册链接（通过系统浏览器打开） ──
    const tavilyLink = document.querySelector('.tavily-link');
    if (tavilyLink) {
        tavilyLink.addEventListener('click', (e) => {
            e.preventDefault();
            window.runtime.BrowserOpenURL('https://tavily.com');
        });
    }

    // ── Tavily API Key 显示/隐藏切换 ──
    const tavilyToggleBtn = document.getElementById('aiTavilyToggleBtn');
    const tavilyKeyInput = document.getElementById('aiTavilyApiKey');
    if (tavilyToggleBtn && tavilyKeyInput) {
        tavilyToggleBtn.addEventListener('click', () => {
            const eye = tavilyToggleBtn.querySelector('.toggle-eye');
            const eyeOff = tavilyToggleBtn.querySelector('.toggle-eye-off');
            if (tavilyKeyInput.type === 'password') {
                tavilyKeyInput.type = 'text';
                eye.style.display = 'none';
                eyeOff.style.display = '';
            } else {
                tavilyKeyInput.type = 'password';
                eye.style.display = '';
                eyeOff.style.display = 'none';
            }
        });
    }

    // ── Tavily API Key 自动保存 ──
    const tavilyKey = document.getElementById('aiTavilyApiKey');
    if (tavilyKey) {
        tavilyKey.addEventListener('change', async () => {
            const key = tavilyKey.value.trim();
            try {
                const cfg = await window.go.main.App.GetAIConfig();
                cfg.tavily_api_key = key;
                await window.go.main.App.SaveAIConfig(cfg);
                nm.show(key ? 'Tavily API Key 已保存' : 'Tavily API Key 已清除', 'success');
            } catch (e) {
                nm.show('保存失败: ' + e, 'error');
            }
        });
    }

    // ── Tavily 测试连接 ──
    const testBtn = document.getElementById('aiTestTavilyBtn');
    if (testBtn) {
        testBtn.addEventListener('click', async () => {
            const key = document.getElementById('aiTavilyApiKey').value.trim();
            if (!key) {
                nm.show('请先输入 Tavily API Key', 'warning');
                return;
            }
            setBtnLoading(testBtn, true, '测试中…');
            try {
                await window.go.main.App.TestTavilyConnection(key);
                nm.show('连接成功！', 'success');
            } catch (e) {
                nm.show('连接失败: ' + (e.message || e), 'error');
            } finally {
                setBtnLoading(testBtn, false);
            }
        });
    }

    // ── 联网搜索默认开启切换 ──
    const settingWebSearchLine = document.getElementById('aiSettingWebSearchLine');
    if (settingWebSearchLine) {
        settingWebSearchLine.addEventListener('click', async () => {
            const toggleSwitch = document.getElementById('aiSettingWebSearchToggle');
            if (!toggleSwitch) return;
            const isActive = toggleSwitch.classList.toggle('active');
            localStorage.setItem('ai_web_search_enabled', String(isActive));
            try { await window.go.main.App.SetSetting('ai_web_search_enabled', String(isActive)); } catch (_) {}
            nm.show(isActive ? '联网搜索已默认开启' : '联网搜索已默认关闭', isActive ? 'success' : 'info');
            // 同步工具栏 toggle
            const toolbarToggle = document.getElementById('aiChatWebSearchToggle');
            if (toolbarToggle) {
                toolbarToggle.classList.toggle('active', isActive);
            }
        });
    }

    // ── 卡片召回切换 ──
    const settingCardRecallLine = document.getElementById('aiSettingCardRecallLine');
    if (settingCardRecallLine) {
        settingCardRecallLine.addEventListener('click', async () => {
            const toggleSwitch = document.getElementById('aiSettingCardRecallToggle');
            if (!toggleSwitch) return;
            const isActive = toggleSwitch.classList.toggle('active');
            localStorage.setItem('ai_card_recall_enabled', String(isActive));
            try { await window.go.main.App.SetSetting('ai_card_recall_enabled', String(isActive)); } catch (_) {}
            nm.show(isActive ? '卡片召回已开启' : '卡片召回已关闭', isActive ? 'success' : 'info');
            // 同步工具栏 toggle
            const toolbarToggle = document.getElementById('aiChatCardRecallToggle');
            if (toolbarToggle) {
                toolbarToggle.classList.toggle('active', isActive);
            }
        });
    }

    // ── 卡片召回条数保存 ──
    const cardRecallLimit = document.getElementById('aiSettingCardRecallLimit');
    if (cardRecallLimit) {
        cardRecallLimit.addEventListener('change', async (e) => {
            let val = parseInt(e.target.value);
            if (isNaN(val) || val < 1) {
                val = 3;
                e.target.value = 3;
            }
            if (val > 10) {
                val = 10;
                e.target.value = 10;
            }
            localStorage.setItem('ai_card_recall_limit', String(val));
            try {
                await window.go.main.App.SetSetting('ai_card_recall_limit', String(val));
                nm.show('召回条数已保存（' + val + ' 条/次）', 'success');
            } catch (err) {
                nm.show('保存失败: ' + err, 'error');
            }
        });
    }

    // ── 引用截断字数自动保存 ──
    const refMaxChars = document.getElementById('aiRefMaxChars');
    if (refMaxChars) {
        refMaxChars.addEventListener('change', async () => {
            const val = parseInt(refMaxChars.value);
            if (isNaN(val) || val < 1) {
                refMaxChars.value = 1000;
                nm.show('截断字数必须大于 0，已重置为 1000', 'warning');
                return;
            }
            try {
                await window.go.main.App.SetAIRefMaxChars(val);
                nm.show('引用截断字数已保存', 'success');
            } catch (e) {
                nm.show('保存失败: ' + e, 'error');
            }
        });
    }

    // ── 联网搜索结果数自动保存 ──
    const searchResultLimit = document.getElementById('aiSearchResultLimit');
    if (searchResultLimit) {
        searchResultLimit.addEventListener('change', async () => {
            const val = parseInt(searchResultLimit.value);
            if (isNaN(val) || val < 1) {
                searchResultLimit.value = 5;
                nm.show('搜索结果数必须大于 0，已重置为 5', 'warning');
                return;
            }
            if (val > 20) {
                searchResultLimit.value = 20;
                nm.show('搜索结果数不能超过 20，已重置为 20', 'warning');
                return;
            }
            try {
                await window.go.main.App.SetAISearchResultLimit(val);
                nm.show('搜索结果数已保存', 'success');
            } catch (e) {
                nm.show('保存失败: ' + e, 'error');
            }
        });
    }

    // ── 预设下拉菜单事件 ──
    const presetTrigger = document.getElementById('presetTrigger');
    const presetDropdown = document.getElementById('presetDropdown');
    if (presetTrigger && presetDropdown) {
        presetTrigger.addEventListener('click', (e) => {
            e.stopPropagation();
            presetTrigger.classList.toggle('open');
            presetDropdown.classList.toggle('open');
        });
        document.addEventListener('click', () => {
            presetDropdown.classList.remove('open');
            presetTrigger.classList.remove('open');
        });
    }

    // ── 新增/管理按钮事件 ──
    document.getElementById('presetAddBtn')?.addEventListener('click', openAddProfileModal);
    document.getElementById('presetMgrBtn')?.addEventListener('click', () => {
        if (presetMgrExpanded) {
            closePresetMgrList();
        } else {
            renderPresetMgrList();
        }
    });

    // ── 预设弹窗事件 ──
    document.getElementById('presetModalClose')?.addEventListener('click', closePresetModal);
    document.getElementById('presetModalCancel')?.addEventListener('click', closePresetModal);
    document.getElementById('presetModalSave')?.addEventListener('click', savePresetModal);
    // 点击遮罩关闭弹窗
    document.getElementById('presetModalOverlay')?.addEventListener('click', (e) => {
        if (e.target === e.currentTarget) closePresetModal();
    });
    // 弹窗内服务商下拉切换
    document.getElementById('presetModalProviderTrigger')?.addEventListener('click', (e) => {
        e.stopPropagation();
        const dd = document.getElementById('presetModalProviderDropdown');
        dd?.classList.toggle('open');
    });
    document.querySelectorAll('#presetModalProviderDropdown .theme-select-item').forEach(item => {
        item.addEventListener('click', () => {
            document.querySelectorAll('#presetModalProviderDropdown .theme-select-item').forEach(i => i.classList.remove('active'));
            item.classList.add('active');
            document.getElementById('presetModalProviderLabel').textContent = item.textContent.trim();
            document.getElementById('presetModalProviderDropdown')?.classList.remove('open');
        });
    });
    // 点击弹窗遮罩关闭下拉
    document.getElementById('presetModalOverlay')?.addEventListener('click', (e) => {
        if (e.target === e.currentTarget) {
            document.getElementById('presetModalProviderDropdown')?.classList.remove('open');
        }
    });
    // 弹窗内 API Key 显示/隐藏切换
    document.getElementById('presetModalKeyToggle')?.addEventListener('click', () => {
        const input = document.getElementById('presetModalKey');
        const eye = document.querySelector('#presetModalKeyToggle .toggle-eye');
        const eyeOff = document.querySelector('#presetModalKeyToggle .toggle-eye-off');
        if (input && eye && eyeOff) {
            if (input.type === 'password') {
                input.type = 'text';
                eye.style.display = 'none';
                eyeOff.style.display = '';
            } else {
                input.type = 'password';
                eye.style.display = '';
                eyeOff.style.display = 'none';
            }
        }
    });
}

// ── API 配置预设管理 ──

// 当前编辑的预设 ID（编辑模式用）
let editingProfileId = null;

// 管理列表是否已展开
let presetMgrExpanded = false;
let presetMgrContainer = null;

// 加载预设列表到下拉
async function loadProfiles() {
    try {
        const profiles = await window.go.main.App.GetProfiles();
        const dropdown = document.getElementById('presetDropdown');
        const label = document.getElementById('presetLabel');
        if (!dropdown) return;
        dropdown.innerHTML = '';
        if (profiles.length === 0) {
            label.textContent = '无预设配置';
            return;
        }
        // 判断当前使用的配置匹配哪个预设
        let activeId = null;
        for (let p of profiles) {
            const item = document.createElement('div');
            item.className = 'theme-select-item';
            item.dataset.profileId = p.id;
            // 展示名称 + 服务商标识
            const badge = document.createElement('span');
            badge.className = 'preset-provider-badge';
            badge.style.marginRight = '6px';
            badge.textContent = p.provider === 'ollama' ? 'Ollama' : 'OpenAI';
            const nameSpan = document.createElement('span');
            nameSpan.textContent = p.name;
            item.appendChild(badge);
            item.appendChild(nameSpan);
            if (p.is_active) {
                activeId = p.id;
                item.classList.add('active');
            }
            // 点击切换
            item.addEventListener('click', () => switchProfile(p.id));
            dropdown.appendChild(item);
        }
        label.textContent = activeId
            ? (profiles.find(p => p.id === activeId)?.name || '选择预设')
            : '选择预设';
    } catch (e) {
        console.warn('加载预设失败:', e);
    }
}

// 切换预设
async function switchProfile(id, silent) {
    try {
        await window.go.main.App.SwitchProfile(id);
        // 刷新当前配置的输入框
        const cfg = await window.go.main.App.GetAIConfig();
        els.aiBaseURL.value = cfg.base_url || '';
        els.aiAPIKey.value = cfg.api_key || '';
        // 更新服务商下拉
        if (els.aiProviderDropdown) {
            const items = els.aiProviderDropdown.querySelectorAll('.theme-select-item');
            items.forEach(item => item.classList.toggle('active', item.dataset.providerValue === cfg.provider));
        }
        if (els.aiProviderLabel) {
            const labels = { openai: 'OpenAI 兼容', ollama: 'Ollama' };
            els.aiProviderLabel.textContent = labels[cfg.provider] || 'OpenAI 兼容';
        }
        // 重置模型下拉
        els.aiModelLabel.textContent = '-- 请先获取模型列表 --';
        els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
        // 同步清除 AI 聊天工具栏的模型
        const chatLabel = document.getElementById('aiChatModelLabel');
        if (chatLabel) chatLabel.textContent = '--';
        // 通知 ai-chat 模块重置模型缓存
        document.dispatchEvent(new CustomEvent('profile-switched'));
        // 刷新预设下拉的选中态
        await loadProfiles();
        if (!silent) {
            nm.show('已切换到配置预设', 'success');
        }
    } catch (e) {
        nm.show('切换预设失败: ' + e, 'error');
    }
}

// 打开新增预设弹窗
function openAddProfileModal() {
    editingProfileId = null;
    document.getElementById('presetModalTitle').textContent = '新增配置';
    document.getElementById('presetModalName').value = '';
    document.getElementById('presetModalURL').value = els.aiBaseURL.value || '';
    document.getElementById('presetModalKey').value = els.aiAPIKey.value || '';
    // 重置 Key 为隐藏状态
    var keyInput = document.getElementById('presetModalKey');
    var eye = document.querySelector('#presetModalKeyToggle .toggle-eye');
    var eyeOff = document.querySelector('#presetModalKeyToggle .toggle-eye-off');
    if (keyInput && eye && eyeOff) {
        keyInput.type = 'password';
        eye.style.display = '';
        eyeOff.style.display = 'none';
    }
    // 重置服务商为当前选中的
    const currentProvider = getActiveProvider();
    const providerItems = document.querySelectorAll('#presetModalProviderDropdown .theme-select-item');
    providerItems.forEach(item => item.classList.toggle('active', item.dataset.presetProvider === currentProvider));
    document.getElementById('presetModalProviderLabel').textContent =
        currentProvider === 'ollama' ? 'Ollama' : 'OpenAI 兼容';
    document.getElementById('presetModalOverlay').classList.add('visible');
    document.getElementById('presetModalName').focus();
}

// 打开编辑预设弹窗
function openEditProfileModal(id, name, provider, baseURL, apiKey) {
    editingProfileId = id;
    document.getElementById('presetModalTitle').textContent = '编辑配置';
    document.getElementById('presetModalName').value = name || '';
    document.getElementById('presetModalURL').value = baseURL || '';
    document.getElementById('presetModalKey').value = apiKey || '';
    // 重置 Key 为隐藏状态
    var keyInput = document.getElementById('presetModalKey');
    var eye = document.querySelector('#presetModalKeyToggle .toggle-eye');
    var eyeOff = document.querySelector('#presetModalKeyToggle .toggle-eye-off');
    if (keyInput && eye && eyeOff) {
        keyInput.type = 'password';
        eye.style.display = '';
        eyeOff.style.display = 'none';
    }
    // 设置服务商
    const providerItems = document.querySelectorAll('#presetModalProviderDropdown .theme-select-item');
    providerItems.forEach(item => item.classList.toggle('active', item.dataset.presetProvider === provider));
    document.getElementById('presetModalProviderLabel').textContent =
        provider === 'ollama' ? 'Ollama' : 'OpenAI 兼容';
    document.getElementById('presetModalOverlay').classList.add('visible');
    document.getElementById('presetModalName').focus();
}

// 关闭预设弹窗
function closePresetModal() {
    document.getElementById('presetModalOverlay').classList.remove('visible');
    editingProfileId = null;
}

// 保存预设（新增或编辑）
async function savePresetModal() {
    const name = document.getElementById('presetModalName').value.trim();
    const providerEl = document.querySelector('#presetModalProviderDropdown .theme-select-item.active');
    const provider = providerEl ? providerEl.dataset.presetProvider : 'openai';
    const baseURL = document.getElementById('presetModalURL').value.trim();
    const apiKey = document.getElementById('presetModalKey').value.trim();
    if (!name) { nm.show('请输入名称', 'error'); return; }
    if (!baseURL) { nm.show('请输入 API 地址', 'error'); return; }
    try {
        if (editingProfileId) {
            await window.go.main.App.UpdateProfile(editingProfileId, name, provider, baseURL, apiKey);
            nm.show('配置已更新', 'success');
        } else {
            const profile = await window.go.main.App.CreateProfile(name, provider, baseURL, apiKey);
            nm.show('配置已新增', 'success');
        }
        closePresetModal();
        await loadProfiles();
        // 如果管理列表展开，同步刷新
        if (presetMgrExpanded && presetMgrContainer && presetMgrContainer.parentNode) {
            renderPresetMgrList();
        }
    } catch (e) {
        nm.show('保存失败: ' + e, 'error');
    }
}

// 删除预设
async function deleteProfile(id, name) {
    const confirmed = await showConfirmDialog(`确定删除配置「${name}」吗？`);
    if (!confirmed) return;
    try {
        await window.go.main.App.DeleteProfile(id);
        nm.show('配置已删除', 'success');
        await loadProfiles();
        // 如果管理列表已展开，刷新它
        if (presetMgrExpanded && presetMgrContainer && presetMgrContainer.parentNode) {
            renderPresetMgrList();
        }
    } catch (e) {
        nm.show('删除失败: ' + e, 'error');
    }
}

// 渲染管理列表（展开在设置页内）
function renderPresetMgrList() {
    if (!presetMgrContainer) {
        presetMgrContainer = document.createElement('div');
        presetMgrContainer.className = 'preset-mgr-list';
        presetMgrContainer.style.cssText = 'margin-top:8px;padding:12px;border:1px solid var(--border);border-radius:var(--radius-md);background:var(--input-bg);';
        // 插入到管理按钮所在行的下方
        const presetRow = document.querySelector('.preset-select-row');
        if (presetRow) {
            presetRow.after(presetMgrContainer);
        } else {
            const settingsSection = document.querySelector('.ai-setting-item.preset-select-row');
            if (settingsSection) settingsSection.after(presetMgrContainer);
        }
    }
    presetMgrContainer.innerHTML = '';
    presetMgrExpanded = true;

    const header = document.createElement('div');
    header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;margin-bottom:8px;';
    const title = document.createElement('span');
    title.style.cssText = 'font-size:0.85rem;font-weight:600;color:var(--text-primary);';
    title.textContent = '配置预设管理';
    const closeBtn = document.createElement('button');
    closeBtn.className = 'btn btn-sm btn-secondary';
    closeBtn.textContent = '关闭';
    closeBtn.addEventListener('click', closePresetMgrList);
    header.appendChild(title);
    header.appendChild(closeBtn);
    presetMgrContainer.appendChild(header);

    // 加载并显示预设列表
    window.go.main.App.GetProfiles().then(profiles => {
        if (profiles.length === 0) {
            const empty = document.createElement('div');
            empty.style.cssText = 'text-align:center;padding:12px;color:var(--text-muted);font-size:0.8rem;';
            empty.textContent = '暂无预设配置';
            presetMgrContainer.appendChild(empty);
            return;
        }
        profiles.forEach(p => {
            const row = document.createElement('div');
            row.style.cssText = 'display:flex;align-items:center;justify-content:space-between;padding:6px 8px;border-radius:var(--radius-sm);gap:8px;';
            row.style.borderBottom = '1px solid var(--border)';
            // 信息区
            const info = document.createElement('div');
            info.style.cssText = 'flex:1;min-width:0;display:flex;flex-direction:column;gap:2px;';
            const nameRow = document.createElement('div');
            nameRow.style.cssText = 'display:flex;align-items:center;gap:10px;';
            const badge = document.createElement('span');
            badge.className = 'preset-provider-badge';
            badge.textContent = p.provider === 'ollama' ? 'Ollama' : 'OpenAI';
            const nameSpan = document.createElement('strong');
            nameSpan.style.cssText = 'font-size:0.85rem;color:var(--text-primary);';
            nameSpan.textContent = p.name;
            nameRow.appendChild(badge);
            nameRow.appendChild(nameSpan);
            const detail = document.createElement('div');
            detail.style.cssText = 'font-size:0.75rem;color:var(--text-muted);';
            detail.textContent = p.base_url;
            info.appendChild(nameRow);
            info.appendChild(detail);
            // 操作区
            const actions = document.createElement('div');
            actions.style.cssText = 'display:flex;gap:4px;flex-shrink:0;';
            const editBtn = document.createElement('button');
            editBtn.className = 'btn btn-sm btn-save';
            editBtn.textContent = '编辑';
            editBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                openEditProfileModal(p.id, p.name, p.provider, p.base_url, p.api_key);
            });
            const delBtn = document.createElement('button');
            delBtn.className = 'btn btn-sm btn-danger';
            delBtn.textContent = '删除';
            if (p.is_default) {
                delBtn.style.display = 'none';
            } else {
                delBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    deleteProfile(p.id, p.name);
                });
            }
            actions.appendChild(editBtn);
            actions.appendChild(delBtn);
            row.appendChild(info);
            row.appendChild(actions);
            presetMgrContainer.appendChild(row);
        });
    }).catch(e => {
        nm.show('加载失败: ' + e, 'error');
    });
}

// 关闭管理列表
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

/**
 * 向模型下拉菜单添加一个选项
 */
function addModelDropdownItem(model, active) {
    const item = document.createElement('div');
    item.className = 'theme-select-item' + (active ? ' active' : '');
    item.dataset.modelValue = model;
    item.textContent = model;
    // 插入到搜索框前面，确保搜索框在底部
    const searchWrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
    if (searchWrap) {
        els.aiModelDropdown.insertBefore(item, searchWrap);
    } else {
        els.aiModelDropdown.appendChild(item);
    }
}

// 辅助函数: 设置状态提示
function setAIStatus(elId, msg, type) {
    const el = document.getElementById(elId);
    if (!el) return;
    el.textContent = msg;
    el.className = 'ai-status ' + type;
    el.style.display = '';
}

/**
 * 加载已保存的排序方式
 */
async function loadSortSettings() {
    let sortOrder = 'updated_at';
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSortOrder) {
        const saved = await window.go.main.App.GetSortOrder();
        if (saved) sortOrder = saved;
    }
    // 高亮对应按钮并移动指示器
    if (els.sortControl) {
        const btns = els.sortControl.querySelectorAll('.segmented-btn');
        const cw = els.sortControl.offsetWidth;
        const segW = (cw - 4) / btns.length;
        btns.forEach((b, i) => {
            const isActive = b.dataset.sortValue === sortOrder;
            b.classList.toggle('active', isActive);
            if (isActive) {
                els.sortIndicator.style.transform = `translateX(${2 + i * segW}px)`;
            }
        });
    }
}

/**
 * 加载已保存的分页大小
 */
async function loadPageSizeSetting() {
    let size = 20;
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetPageSize) {
        const saved = await window.go.main.App.GetPageSize();
        if (saved && saved >= 20 && saved <= 100) size = saved;
    }
    // 高亮对应按钮并移动指示器
    if (els.pageSizeControl) {
        const btns = els.pageSizeControl.querySelectorAll('.segmented-btn');
        const cw = els.pageSizeControl.offsetWidth;
        const segW = (cw - 4) / btns.length;
        btns.forEach((b, i) => {
            const isActive = parseInt(b.dataset.value, 10) === size;
            b.classList.toggle('active', isActive);
            if (isActive) {
                els.pageSizeIndicator.style.transform = `translateX(${2 + i * segW}px)`;
            }
        });
    }
    els.pageSizeLabel.textContent = `${size} 条 / 页`;
}

/**
 * 删除标签
 */
async function deleteTag(id) {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteTag) {
            await window.go.main.App.DeleteTag(id);
        } else {
            console.warn('DeleteTag 未绑定');
        }
    } catch (err) {
        console.error('删除标签失败:', err);
    }
    await loadTags();
    await loadNotes();
    nm.show('标签已删除', 'success');
}

/* ===== 渲染函数 ===== */

/**
 * 渲染卡片网格/列表
 * @param {string} [animateMode] - 'append' 追加模式（已有卡片不重播动画），'none' 无动画，省略则全部卡片播放入场动画
 * @param {number} [prevCount] - 追加前已有的卡片数量（animateMode='append' 时使用）
 */
function renderCardGrid(animateMode, prevCount) {
    // 获取当前排序方式，用于本地回落排序
    const checkedRadio = document.querySelector('input[name="sortOrder"]:checked');
    const sortBy = checkedRadio ? checkedRadio.value : 'updated_at';

    const sorted = [...state.notes].sort((a, b) => {
        if (a.pinned && !b.pinned) return -1;
        if (!a.pinned && b.pinned) return 1;
        if (sortBy === 'title') {
            const titleA = (a.title || '').toLowerCase();
            const titleB = (b.title || '').toLowerCase();
            if (titleA < titleB) return -1;
            if (titleA > titleB) return 1;
            return 0;
        }
        const dateA = new Date(sortBy === 'created_at' ? a.created_at : (a.updated_at || a.created_at));
        const dateB = new Date(sortBy === 'created_at' ? b.created_at : (b.updated_at || b.created_at));
        return dateB - dateA;
    });

    // 先清理旧 footer，确保空数据时也能正确移除
    const oldFooter = els.viewGrid.querySelector('.notes-footer');
    if (oldFooter) oldFooter.remove();

    // 隐藏骨架屏
    if (els.skeletonGrid) els.skeletonGrid.style.display = 'none';

    if (sorted.length === 0) {
        // 隐藏卡片网格，显示空状态
        els.cardGrid.style.display = 'none';
        if (els.emptyNotes) els.emptyNotes.style.display = 'flex';
        return;
    }

    // 有笔记时：隐藏空状态，显示卡片网格
    if (els.emptyNotes) els.emptyNotes.style.display = 'none';
    els.cardGrid.style.display = '';

    els.cardGrid.innerHTML = sorted
        .map(
            (note) => `
        <div class="note-card${note.pinned ? ' pinned' : ''}${state.batchMode ? ' batch-mode' : ''}${state.selectedNoteIds.has(note.id) ? ' selected' : ''}" data-id="${note.id}" onclick="${state.batchMode ? `window.toggleNoteSelection(${note.id})` : `window.viewNote(${note.id})`}" oncontextmenu="${state.batchMode ? 'event.preventDefault()' : `event.preventDefault(); window.showContextMenu(event, ${note.id})`}">
            <div class="card-body">
                <div class="card-title">${escapeHtml(note.title || '无标题')}</div>
                <div class="card-content">${escapeHtml(getSummary(note.content, 200))}</div>
            </div>
            <div class="card-footer">
                <div class="card-tags">
                    ${(note.tags || [])
                        .map(
                            (tag) =>
                                `<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}" onclick="${state.batchMode ? `event.stopPropagation()` : `event.stopPropagation(); window.searchByTag(${tag.id}, '${escapeHtml(tag.name)}')`}">${escapeHtml(tag.name)}</span>`
                        )
                        .join('')}
                </div>
                <span class="card-time">${formatTime(note.updated_at || note.created_at)}</span>
            </div>
            <div class="card-actions" onclick="event.stopPropagation()">
                ${state.batchMode
                    ? `<input type="checkbox" class="batch-checkbox" ${state.selectedNoteIds.has(note.id) ? 'checked' : ''} onclick="event.stopPropagation(); window.toggleNoteSelection(${note.id})">`
                    : ''
                }
                <button class="card-action-btn pin-btn${state.batchMode ? ' disabled' : ''}" onclick="event.stopPropagation()${state.batchMode ? '' : `; window.handlePinClick(event, ${note.id})`}" title="${note.pinned ? '已置顶' : (state.batchMode ? '批量模式下不可操作' : '置顶')}">
                    ${note.pinned ? SVGS.pinFilled : SVGS.pinOutline}
                </button>
            </div>
        </div>
        `
        )
        .join('');

    // 卡片入场动画
    const cards = els.cardGrid.querySelectorAll('.note-card');
    if (animateMode === 'none') {
        // 无动画模式（批量操作），直接设为可见
        cards.forEach(card => { card.style.opacity = '1'; });
    } else if (animateMode === 'append' && typeof prevCount === 'number') {
        // 追加模式：已有卡片不重播动画，新卡片带交错入场
        cards.forEach((card, index) => {
            if (index < prevCount) {
                card.style.opacity = '1';
                card.style.animation = 'none';
            } else {
                const delay = Math.min((index - prevCount) * 40, 480);
                card.style.animation = `cardEnter 0.25s ease-out forwards`;
                card.style.animationDelay = `${delay}ms`;
            }
        });
    } else {
        // 全量刷新模式：所有卡片带交错入场动画
        cards.forEach((card, index) => {
            const delay = Math.min(index * 40, 480);
            card.style.animation = `cardEnter 0.25s ease-out forwards`;
            card.style.animationDelay = `${delay}ms`;
        });
    }

    // 添加加载完成提示（底部已全部加载）
    const gridContainer = els.viewGrid.querySelector('.card-grid') || els.viewGrid;

    if (!hasMoreNotes && totalNotes > 0) {
        const footer = document.createElement('div');
        footer.className = 'notes-footer';
        footer.textContent = `共 ${totalNotes} 条笔记`;
        gridContainer.after(footer);
    }
}

/**
 * 渲染标签管理列表
 */
function renderTagList() {
    if (state.tags.length === 0) {
        els.tagList.innerHTML = '<div style="color: #94a3b8; font-size: 13px;">暂无标签</div>';
        return;
    }

    els.tagList.innerHTML = state.tags
        .map(
            (tag) => `
        <div class="tag-item" style="background-color: ${tag.color || '#6366f1'}">
            ${escapeHtml(tag.name)}
            <button class="tag-delete-btn" onclick="window.deleteTag(${tag.id})">${SVGS.windowClose}</button>
        </div>
        `
        )
        .join('');
}

/**
 * 渲染编辑器中的标签选择器
 * @param {boolean} readOnly - 是否为只读模式（仅展示，不可切换）
 */
function renderTagSelector(readOnly) {
    if (state.tags.length === 0) {
        const msg = readOnly
            ? '<div style="color: #94a3b8; font-size: 12px;">暂无标签</div>'
            : '<div style="color: #94a3b8; font-size: 12px;">暂无可用标签，请先在设置页添加</div>';
        els.tagSelector.innerHTML = msg;
        return;
    }

    if (readOnly) {
        // 只读模式：仅展示笔记已添加的标签，不可切换
        const noteTags = state.tags.filter(tag => state.selectedTags.includes(tag.id));
        if (noteTags.length === 0) {
            els.tagSelector.innerHTML = '<div style="color: #94a3b8; font-size: 12px;">暂无标签</div>';
        } else {
            els.tagSelector.innerHTML = noteTags
                .map(
                    (tag) => `
            <span class="card-tag" style="background-color: ${tag.color || '#6366f1'}; cursor: default;"
                  data-tag-id="${tag.id}">
                ${escapeHtml(tag.name)}
            </span>
            `
                )
                .join('');
        }
        return;
    }

    els.tagSelector.innerHTML = state.tags
        .map(
            (tag) => `
        <div class="tag-chip ${state.selectedTags.includes(tag.id) ? 'active' : ''}"
             style="background-color: ${tag.color || '#6366f1'}"
             data-tag-id="${tag.id}"
             onclick="window.toggleEditorTag(${tag.id}, this)">
            ${escapeHtml(tag.name)}
        </div>
        `
        )
        .join('');
}

/* ===== 编辑器函数 ===== */

/**
 * 获取编辑器内容
 * @returns {string}
 */
function getEditorContent() {
    return cmEditor ? cmEditor.state.doc.toString() : '';
}

// 暴露给外部模块（如 ai-chat.js）
window.getEditorContent = getEditorContent;

/**
 * 在 CM6 编辑器光标位置插入文本
 */
window.insertTextToEditor = function(text) {
    if (!cmEditor) return;
    cmEditor.dispatch(cmEditor.state.replaceSelection(text));
    cmEditor.focus();
};

/**
 * 更新字数统计
 */
function updateWordCount() {
    const content = getEditorContent();
    const charCount = content.length;
    const wordCount = content.replace(/[\s]/g, '').length;
    els.editorWordCount.textContent = `${wordCount} 个字数 | ${charCount} 个字符`;
}

/** 更新状态栏文件后缀显示 */
function updateFileExtDisplay(ext) {
    els.editorFileExt.textContent = ext || '';
}



/** 打开后缀编辑对话框（仅编辑/新建模式可用） */
function openFileExtDialog() {
    const viewEditor = document.getElementById('viewEditor');
    // 查看模式（只读）下忽略点击
    if (viewEditor && viewEditor.classList.contains('active') && els.editorSaveBtn.style.display === 'none') {
        return;
    }
    const currentExt = els.editorFileExt.textContent;
    document.getElementById('fileExtInput').value = currentExt;
    document.getElementById('fileExtError').style.display = 'none';
    document.getElementById('fileExtError').textContent = '';
    document.getElementById('fileExtDialog').style.display = 'flex';
    // 自动聚焦并选中后缀名部分（不含点）
    const input = document.getElementById('fileExtInput');
    input.focus();
    const dotIdx = currentExt.indexOf('.');
    if (dotIdx >= 0) {
        input.setSelectionRange(dotIdx + 1, currentExt.length);
    } else {
        input.select();
    }
}

/** 关闭后缀编辑对话框 */
function closeFileExtDialog() {
    document.getElementById('fileExtDialog').style.display = 'none';
}

/** 保存后缀编辑 */
async function saveFileExt() {
    const input = document.getElementById('fileExtInput');
    const errorEl = document.getElementById('fileExtError');
    let value = input.value.trim();

    // 校验
    if (!value) {
        errorEl.textContent = '后缀不能为空';
        errorEl.style.display = '';
        input.focus();
        return;
    }
    if (!value.startsWith('.')) {
        value = '.' + value;
    }
    if (!/^\.[a-zA-Z0-9_]{1,9}$/.test(value)) {
        errorEl.textContent = '后缀以 . 开头，只能包含字母、数字、下划线（1-9 位）';
        errorEl.style.display = '';
        input.focus();
        return;
    }

    // 更新显示（不立即保存到后端，随主保存一起提交）
    els.editorFileExt.textContent = value;
    closeFileExtDialog();

    // 根据新后缀同步编辑器 UI
    const isMd = value.toLowerCase() === '.md';
    els.editorModes.style.display = isMd ? '' : 'none';
    if (!isMd) {
        switchEditorMode('edit');
    }
    // 同步顶部 T/M 切换按钮显示
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = isMd ? 'M' : 'T';
        els.editorTypeToggle.title = isMd ? '切换为纯文本格式' : '切换为 Markdown 格式';
    }
    // 刷新字数统计显示
    updateWordCount();

    // 重新初始化 CM6 刷新语法高亮
    if (cmEditor) {
        const container = els.editorNoteContent;
        const content = cmEditor.state.doc.toString();
        const isReadOnly = els.editorSaveBtn.style.display === 'none';
        cmEditor.destroy();
        cmEditor = null;
        const useSyntaxHighlight = els.mdHighlightToggle.checked;
        initCodeMirror(container, content, isReadOnly, useSyntaxHighlight, value, codeHighlightTheme);
    }
}

/** 快速切换笔记类型（.md ↔ .txt），更新按钮显示并保存到后端 */
async function toggleFileExt() {
    const currentExt = els.editorFileExt.textContent || '.txt';
    const newExt = currentExt === '.md' ? '.txt' : '.md';
    els.editorFileExt.textContent = newExt;

    // 更新按钮显示
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = newExt === '.md' ? 'M' : 'T';
        els.editorTypeToggle.title = newExt === '.md' ? '切换为纯文本格式' : '切换为 Markdown 格式';
    }

    // 同步编辑器 UI（编辑/预览模式按钮可见性）
    els.editorModes.style.display = newExt === '.md' ? '' : 'none';
    if (newExt !== '.md') {
        switchEditorMode('edit');
    }

    // 刷新字数统计显示
    updateWordCount();

    // 重新初始化 CM6 刷新语法高亮
    if (cmEditor) {
        const container = els.editorNoteContent;
        const content = cmEditor.state.doc.toString();
        const isReadOnly = els.editorSaveBtn.style.display === 'none';
        cmEditor.destroy();
        cmEditor = null;
        const useSyntaxHighlight = els.mdHighlightToggle.checked;
        initCodeMirror(container, content, isReadOnly, useSyntaxHighlight, newExt, codeHighlightTheme);
    }
}

/**
 * 打开编辑器（新建/编辑/查看）
 * @param {number|null} noteId - 笔记 ID，null 表示新建
 * @param {boolean} readOnly - 是否为只读查看模式
 */
async function openEditor(noteId, readOnly, startFullscreen) {
    state.editingNoteId = noteId || null;
    state.selectedTags = [];

    const isReadOnly = readOnly && noteId != null;
    let noteData = null;
    // 暂存初始内容，用于 initCodeMirror
    let editorContent = '';

    if (noteId) {
        const note = state.notes.find((n) => n.id === noteId);
        noteData = note;
        if (note) {
            // 查看/编辑模式
            els.editorNoteTitle.value = note.title || '';
            // 列表中的 content 是截断版本（前 200 字符），从后端加载完整内容
            try {
                if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNoteContent) {
                    editorContent = await window.go.main.App.GetNoteContent(noteId) || '';
                } else {
                    editorContent = note.content || '';
                }
            } catch (err) {
                console.error('获取完整笔记内容失败:', err);
                editorContent = note.content || '';
            }
            state.selectedTags = (note.tags || []).map((t) => t.id);
        } else {
            document.getElementById('colorPicker').value = '#6366f1';
        }
    } else {
        // 新建模式：默认标题为当前日期时间 + 表情
        const now = new Date();
        const pad = (n) => String(n).padStart(2, '0');
        els.editorNoteTitle.value = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())} ☺️`;
        editorContent = '';
        // 缓存默认标题，用于判断用户是否改过标题
        state._defaultNewNoteTitle = els.editorNoteTitle.value;
    }

    // 只读模式：禁用输入，隐藏保存/取消按钮
    els.editorNoteTitle.readOnly = isReadOnly;
    els.editorNoteTitle.classList.toggle('editor-input-readonly', isReadOnly);
    els.editorSaveBtn.style.display = isReadOnly ? 'none' : '';
    els.editorCancelBtn.style.display = isReadOnly ? 'none' : '';
    els.editorPanel.classList.toggle('editor-view-mode', isReadOnly);
    // 只读模式隐藏笔记类型切换按钮，编辑/新建模式显示
    if (els.editorTypeToggle) {
        els.editorTypeToggle.style.display = isReadOnly ? 'none' : '';
    }
    // 只读模式显示编辑按钮，编辑/新建模式隐藏
    els.editorEditBtn.style.display = isReadOnly ? '' : 'none';
    // 从查看模式进入编辑时显示"返回查看模式"按钮
    els.editorViewBtn.style.display = (!isReadOnly && state.enteredFromViewMode) ? '' : 'none';
    // 只读模式禁用后缀点击（查看模式不可点击，编辑/新建模式可点击）
    els.editorFileExt.classList.toggle('file-ext-readonly', !!isReadOnly);

    // Markdown 模式显示底部「编辑/预览」切换按钮，纯文本模式隐藏
    const ext = (noteData && noteData.file_ext) || '.txt';
    const isMd = ext === '.md';
    els.editorModes.style.display = isMd ? '' : 'none';

    // 设置文件后缀显示
    updateFileExtDisplay(ext);

    // 初始化类型切换按钮显示
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = isMd ? 'M' : 'T';
        els.editorTypeToggle.title = isMd ? '切换为纯文本格式' : '切换为 Markdown 格式';
    }

    // 统一初始化编辑器模式为纯文本编辑（data-mode 值影响 flex 布局 CSS 选择器）
    // 后续各分支根据情况可 override：查看+Markdown → 'preview'
    els.editorOverlay.dataset.mode = 'edit';

    // 查看模式：显示最近编辑时间 + 按类型选择渲染方式
    if (isReadOnly && noteData) {
        els.editorEditTime.textContent = '最近编辑 ' + formatTime(noteData.updated_at || noteData.created_at);
        if (els.editorFileExt.textContent !== '.md') {
            // 纯文本：CM6 自动以只读模式展示（data-mode 已由默认值 'edit' 覆盖）
            els.mdRendered.style.display = 'none';
        } else {
            // Markdown：直接从暂存内容渲染预览，无需等 CM6 初始化
            els.editorOverlay.dataset.mode = 'preview';
            els.editorModeBtns.forEach(btn => {
                btn.classList.toggle('active', btn.dataset.mode === 'preview');
            });
            if (editorContent.trim()) {
                els.mdRendered.innerHTML = marked.parse(editorContent);
                // 同步 DOM 后处理：hljs 高亮 + 代码块/表格复制按钮 + 语言标签
                // 内部已包含 hljs.highlightElement,无需重复调用
                _applyPreviewDOMHelpers();
            } else {
                els.mdRendered.innerHTML = '<p class="md-empty">暂无内容</p>';
            }
            // 动画延迟一个 tick，确保 DOM 已更新
            requestAnimationFrame(() => {
                els.mdRendered.style.animation = 'animFadeIn 0.2s ease-out forwards';
                requestAnimationFrame(() => {
                    const codeBlocks = els.mdRendered.querySelectorAll('pre');
                    codeBlocks.forEach((block, index) => {
                        block.style.animation = `animFadeIn 0.2s ease-out forwards`;
                        block.style.animationDelay = `${index * 50}ms`;
                    });
                });
            });
        }
    } else {
        els.editorEditTime.textContent = '';
        // 编辑模式切换到纯文本
        switchEditorMode('edit');
    }

    // 标题输入监听（CM6 内容变化由 ViewUpdate listener 自动处理）
    if (!isReadOnly) {
        els.editorNoteTitle.addEventListener('input', onEditorInput);
        state._titleInputListenerAttached = true;
    } else {
        els.editorNoteTitle.removeEventListener('input', onEditorInput);
        state._titleInputListenerAttached = false;
    }

    // 加载标签并渲染标签选择器（等待标签加载完毕再显示编辑器，避免闪烁）
    await loadTagsForEditor(isReadOnly);
    // 锁定主内容区滚动，隐藏滚动条槽位
    els.mainContent.style.overflow = 'hidden';
    // 强制确保 overlay 和 panel 在视图可见前 opacity:0（防闪烁）
    els.editorOverlay.style.opacity = '0';
    els.editorPanel.style.opacity = '0';
    void els.editorOverlay.offsetHeight;
    // 显示编辑器
    els.viewEditor.classList.add('active');

    // 先初始化 CM6（此时编辑器 opacity: 0，用户还看不到）
    const useSyntaxHighlight = els.mdHighlightToggle.checked;
    initCodeMirror(els.editorNoteContent, editorContent, isReadOnly, useSyntaxHighlight, els.editorFileExt.textContent, codeHighlightTheme);
    // 字数统计（需在 CM6 初始化之后，否则 getEditorContent 返回空）
    updateWordCount();
    // 编辑模式下记录快照，用于蒙层点击判断内容是否有改动
    if (!isReadOnly && state.editingNoteId) {
        state._editSnapshot = {
            title: els.editorNoteTitle.value.trim(),
            content: getEditorContent().trim(),
            tags: [...state.selectedTags].sort(),
            fileExt: els.editorFileExt.textContent
        };
    }
    // CM6 就绪后刷新预览（解决查看模式下 CM6 初始化前预览无法渲染的问题）
    if (els.editorOverlay.dataset.mode === 'preview') {
        updatePreview();
    }

    // 再启动编辑器打开动画（CM6 已渲染完毕，随面板一起淡入）
    const overlay = els.editorOverlay;
    const panel = els.editorPanel;
    const body = panel.querySelector('.editor-body');
    // 编辑器打开时自动隐藏 #topbar 搜索框和更多菜单（不受全屏限制）
    document.getElementById('topbar').classList.add('editor-fullscreen');

    if (startFullscreen) {
        // 快速笔记启动：直接以全屏尺寸打开，不经过悬浮卡片，跳过所有动画
        // 先禁用 CSS transition，避免添加 fullscreen class 时触发尺寸过渡动画
        panel.style.transition = 'none';
        overlay.classList.add('fullscreening');
        panel.classList.add('fullscreen');
        // 强制回流确保 transition:none 生效
        void panel.offsetHeight;
        // 恢复 transition（后续用户操作的过渡仍需保留）
        panel.style.transition = '';

        state._isFullscreen = true;
        if (els.editorFullscreenBtn) {
            els.editorFullscreenBtn.innerHTML = SVGS.editorExitFullscreen;
            els.editorFullscreenBtn.title = '退出全屏';
            els.editorFullscreenBtn.classList.add('fullscreen');
        }
        // 收起侧栏
        if (els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed')) {
            els.notebookSidebar.classList.add('collapsed');
        }

        // 覆盖 CSS 初始 opacity:0/scale(0.85)，立即显示
        overlay.style.opacity = '1';
        panel.style.opacity = '1';
        panel.style.transform = 'scale(1)';
    } else {
        // 清除内联 opacity，让 animation 控制淡入
        overlay.style.opacity = '';
        panel.style.opacity = '';
        overlay.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
        panel.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
        // 内容区域与面板同步入场
        if (body) {
            body.style.animation = 'viewEnter 0.25s ease-out forwards';
        }
    }

    // 新建笔记时，光标自动聚焦到内容输入框（仅在窗口已激活时生效，启动时自动打开的快速笔记跳过）
    if (!state.editingNoteId && els.editorOverlay.dataset.mode !== 'preview' && document.hasFocus()) {
        window.focus();
        cmEditor?.focus();
    }
}

/** 预览渲染处理中标志，防重复请求 */
let _previewWorkerLoading = false;

/**
 * 初始化预览渲染 Worker
 */
function initPreviewWorker() {
    try {
        _previewWorker = new Worker(
            new URL('./js/preview-worker.js', import.meta.url),
            { type: 'module' }
        );
        _previewWorker.onmessage = function (e) {
            const { html, error } = e.data;
            if (error) {
                console.error('Preview Worker:', error);
                els.mdRendered.innerHTML = '<p class="md-error">渲染失败</p>';
                _previewWorkerLoading = false;
                return;
            }
            // 设置渲染结果
            els.mdRendered.innerHTML = html;
            // hljs 高亮（必须在主线程，需要 DOM 环境）
            els.mdRendered.querySelectorAll('pre code').forEach((block) => {
                if (typeof hljs !== 'undefined') {
                    hljs.highlightElement(block);
                }
            });
            // 复制按钮、语言标签、表格按钮等 DOM 后处理
            _applyPreviewDOMHelpers();
            // 隐藏加载状态
            const loadingEl = els.mdRendered.querySelector('.md-rendered-loading');
            if (loadingEl) loadingEl.remove();
            _previewWorkerLoading = false;
        };
    } catch (err) {
        console.warn('Preview Worker init fallback:', err);
        _previewWorker = null;
    }
}

/**
 * 预览渲染后的 DOM 辅助处理（复制按钮、语言标签、表格按钮）
 */
function _applyPreviewDOMHelpers() {
    // 代码高亮
    els.mdRendered.querySelectorAll('pre code').forEach((block) => {
        if (typeof hljs !== 'undefined') {
            hljs.highlightElement(block);
        }
    });
    // 为每个代码块添加复制按钮
    els.mdRendered.querySelectorAll('pre').forEach((pre) => {
        if (pre.querySelector('.copy-code-btn')) return;
        const btn = document.createElement('button');
        const codeEl = pre.querySelector('code');
        const isSingleLine = codeEl && !codeEl.textContent.trim().includes('\n');
        btn.className = 'copy-code-btn' + (isSingleLine ? ' copy-code-btn--single' : '');
        btn.textContent = '复制';
        btn.title = '复制代码';
        btn.addEventListener('click', async () => {
            const code = pre.querySelector('code').textContent;
            try {
                await navigator.clipboard.writeText(code);
                btn.classList.add('copied');
                btn.innerHTML = SVGS.checkmark + ' 已复制';
                setTimeout(() => {
                    btn.classList.remove('copied');
                    btn.textContent = '复制';
                }, 1500);
            } catch {
                btn.innerHTML = SVGS.xmark + ' 复制失败';
                setTimeout(() => { btn.textContent = '复制'; }, 1000);
            }
        });
        pre.appendChild(btn);
    });
    // 为每个代码块添加语言标签
    els.mdRendered.querySelectorAll('pre').forEach((pre) => {
        if (pre.parentNode.classList.contains('pre-wrapper')) return;
        const code = pre.querySelector('code');
        if (!code) return;
        const langClass = Array.from(code.classList).find(cls => cls.startsWith('language-'));
        const lang = langClass ? langClass.replace('language-', '') : '';
        if (!lang) return;
        const wrapper = document.createElement('div');
        wrapper.className = 'pre-wrapper';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);
        const badge = document.createElement('span');
        badge.className = 'code-lang-badge';
        badge.textContent = lang.charAt(0).toUpperCase() + lang.slice(1);
        wrapper.appendChild(badge);
    });
    // 为每个表格添加复制按钮
    els.mdRendered.querySelectorAll('table').forEach((table) => {
        let wrapper = table.parentNode;
        if (wrapper.classList.contains('table-wrapper')) return;
        wrapper = document.createElement('div');
        wrapper.className = 'table-wrapper';
        table.parentNode.insertBefore(wrapper, table);
        wrapper.appendChild(table);
        const btn = document.createElement('button');
        btn.className = 'copy-table-btn';
        btn.textContent = '复制';
        btn.title = '复制表格';
        btn.addEventListener('click', async () => {
            const md = tableToMarkdown(table);
            try {
                await navigator.clipboard.writeText(md);
                btn.classList.add('copied');
                btn.innerHTML = SVGS.checkmark + ' 已复制';
                setTimeout(() => {
                    btn.classList.remove('copied');
                    btn.textContent = '复制';
                }, 1500);
            } catch {
                btn.innerHTML = SVGS.xmark + ' 复制失败';
                setTimeout(() => { btn.textContent = '复制'; }, 1000);
            }
        });
        wrapper.appendChild(btn);
        // 将按钮 top 定位到第一行(表头)的垂直中线
        // 单纯 CSS 无法跨越 thead 边界(HTML 不允许 button 放在 thead 内),
        // 通过 JS 测量第一行相对 wrapper 的偏移来精确对齐
        const updateBtnPosition = () => {
            // 用 tr:first-child 作为参考(兼容没有 thead 标签的情况)
            const firstRow = table.querySelector('tr');
            if (!firstRow) return;
            const rowRect = firstRow.getBoundingClientRect();
            const wrapperRect = wrapper.getBoundingClientRect();
            const centerY = rowRect.top - wrapperRect.top + rowRect.height / 2;
            const btnHeight = btn.offsetHeight || 24;
            btn.style.top = (centerY - btnHeight / 2) + 'px';
        };
        // 双重 rAF 兜底：第一次 layout 刚完成, 第二次确保样式完全稳定
        requestAnimationFrame(() => {
            updateBtnPosition();
            requestAnimationFrame(updateBtnPosition);
        });
        // 监听 table 尺寸变化, 响应式 / 字体加载等场景下自动重新对齐
        const ro = new ResizeObserver(updateBtnPosition);
        ro.observe(table);
    });
}

/**
 * 更新 Markdown 预览区
 * 通过 Web Worker 离线程解析，不阻塞主线程 UI
 */
function updatePreview() {
    const content = getEditorContent();
    if (!content.trim()) {
        els.mdRendered.innerHTML = '<p class="md-empty">暂无内容</p>';
        _lastPreviewContent = '';
        return;
    }
    // 内容未变化则跳过重复渲染（哈希缓存）
    if (content === _lastPreviewContent) return;
    _lastPreviewContent = content;

    // 有 Worker 且不在处理中则走 Worker 渲染
    if (_previewWorker && !_previewWorkerLoading) {
        _previewWorkerLoading = true;
        // 显示加载状态
        els.mdRendered.innerHTML = '<div class="md-rendered-loading">加载中…</div>';
        _previewWorker.postMessage(content);
        return;
    }

    // 无 Worker 或 Worker 正忙时回退到主线程同步渲染
    els.mdRendered.innerHTML = marked.parse(content);
    _applyPreviewDOMHelpers();
}

/**
 * 将 HTML table 元素转换为 Markdown 表格语法文本
 * @param {HTMLTableElement} tableEl
 * @returns {string}
 */
function tableToMarkdown(tableEl) {
    const rows = [];
    // 获取所有行（含 thead + tbody 的行）
    const trs = tableEl.querySelectorAll('tr');
    if (!trs.length) return '';
    trs.forEach((tr, index) => {
        const cells = tr.querySelectorAll('th, td');
        const row = '| ' + Array.from(cells).map(c => c.textContent.trim()).join(' | ') + ' |';
        rows.push(row);
        // 表头后添加分隔行
        if (index === 0 && tr.querySelector('th')) {
            const sep = '| ' + Array.from(cells).map(() => '---').join(' | ') + ' |';
            rows.push(sep);
        }
    });
    return rows.join('\n');
}

/**
 * 切换编辑器模式（纯文本/预览）
 */
function switchEditorMode(mode) {
    // 更新按钮活跃状态
    els.editorModeBtns.forEach(btn => {
        btn.classList.toggle('active', btn.dataset.mode === mode);
    });
    // 更新 overlay 的 data-mode
    els.editorOverlay.dataset.mode = mode;
    // 预览模式下立即渲染（CM6 未就绪时跳过，等 initCodeMirror 完成后自动刷新）
    if (mode === 'preview' && cmEditor) {
        updatePreview();
    }
    // 切换模式时关闭查找/替换条（CM6 search 自管理）
    if (cmEditor) {
        cmEditor.focus();
    }
}

/**
 * 编辑器输入事件处理：更新字数 + 预览渲染
 */
function onEditorInput() {
    updateWordCount();
    // 预览模式下自动更新
    if (els.editorOverlay.dataset.mode === 'preview') {
        debouncedUpdatePreview();
    }
}

// 防抖预览更新
const debouncedUpdatePreview = debounce(updatePreview, 300);

/**
 * 关闭编辑器
 */
/**
 * 切换编辑器全屏模式
 */
function toggleEditorFullscreen() {
    const panel = els.editorPanel;
    const btn = els.editorFullscreenBtn;
    const overlay = els.editorOverlay;
    const goFullscreen = !state._isFullscreen;
    const mdRendered = els.mdRendered;
    const body = panel.querySelector('.editor-body');

    // 清除上一次未完成的定时器
    if (panel._fsTimer) { clearTimeout(panel._fsTimer); panel._fsTimer = null; }

    /* 阶段1（0→50ms）：内容快速淡出 */
    if (body) body.style.transition = 'opacity 0.05s ease-out';
    if (body) body.style.opacity = '0';

    panel._fsTimer = setTimeout(() => {
        /* 阶段2（50ms）：隐藏内容 DOM，切换 class（CSS transition 处理 350ms 过渡） */
        if (mdRendered) mdRendered.style.display = 'none';
        if (body) body.style.transition = '';

        state._isFullscreen = goFullscreen;
        panel.classList.toggle('fullscreen', goFullscreen);
        overlay.classList.toggle('fullscreening', goFullscreen);
        btn.innerHTML = goFullscreen ? SVGS.editorExitFullscreen : SVGS.editorFullscreen;
        btn.title = goFullscreen ? '退出全屏' : '全屏编辑';
        btn.classList.toggle('fullscreen', goFullscreen);

        // 全屏时自动收起侧栏
        if (goFullscreen && els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed')) {
            els.notebookSidebar.classList.add('collapsed');
        }

        /* 阶段3（50ms + 350ms）：等待 CSS transition 完成 */
        panel._fsTimer = setTimeout(() => {
            /* 恢复内容，淡入 */
            if (mdRendered) mdRendered.style.display = '';
            if (body) {
                body.style.transition = 'opacity 0.12s ease-out';
                body.style.opacity = '1';
                setTimeout(() => {
                    body.style.transition = '';
                }, 130);
            }
            panel._fsTimer = null;
        }, 350);
    }, 50);
}

/**
 * 保存编辑器内容（退出程序前调用）
 */
async function saveEditorContent() {
    if (!els.viewEditor.classList.contains('active')) return;
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    if (!title || !content) return;
    if (!window.go || !window.go.main || !window.go.main.App) return;
    try {
        if (state.editingNoteId) {
            await window.go.main.App.UpdateNote(state.editingNoteId, title, content, els.editorFileExt.textContent);
        } else if (window.go.main.App.CreateNote) {
            await window.go.main.App.CreateNote(title, content, els.editorFileExt.textContent, state.activeNotebookId);
        }
    } catch (err) {
        console.error('退出前保存失败:', err);
    }
}

/**
 * 退出程序前的统一处理：提示保存 → 执行退出
 */
async function handleAppExit() {
    // 编辑器打开且有内容 + 非只读模式 → 询问是否保存
    if (els.viewEditor.classList.contains('active') && els.editorSaveBtn.style.display !== 'none') {
        const title = els.editorNoteTitle.value.trim();
        const content = getEditorContent().trim();
        const snapshot = state._editSnapshot;
        const extChanged = snapshot ? els.editorFileExt.textContent !== snapshot.fileExt : false;
        // 有内容变更、后缀变更 或 有内容 → 询问是否保存
        if ((title && content) || extChanged) {
            const action = await showSaveConfirmDialog('笔记内容尚未保存，退出前是否保存？');
            if (action === 'cancel') return;               // 取消：不退出
            if (action === 'save') {
                await saveEditorContent();                  // 保存后退出
            }
            // discard: 直接继续退出
        }
    }
    Quit();
}

function closeEditor() {
    const overlay = els.editorOverlay;
    const panel = els.editorPanel;
    const body = panel.querySelector('.editor-body');

    // 退出动画
    panel.style.animation = 'modalExit 0.18s ease-in forwards';
    overlay.style.animation = 'overlayFadeOut 0.15s ease-in forwards';

    // 动画完成后执行清理
    setTimeout(() => {
        // 重置动画
        overlay.style.animation = '';
        panel.style.animation = '';
        if (body) body.style.animation = '';
        els.mdRendered.style.animation = '';

        els.viewEditor.classList.remove('active');
        // 恢复主内容区滚动
        els.mainContent.style.overflow = '';
        // 退出全屏模式（仅当实际处于全屏状态时重置）
        if (state._isFullscreen) {
            els.editorPanel.classList.remove('fullscreen');
            els.editorFullscreenBtn.innerHTML = SVGS.editorFullscreen;
            els.editorFullscreenBtn.title = '全屏编辑';
            els.editorFullscreenBtn.classList.remove('fullscreen');
            state._isFullscreen = false;
        }
        document.getElementById('topbar').classList.remove('editor-fullscreen');
        // 重置查看模式标志
        state.enteredFromViewMode = false;
        // 清理事件监听
        if (state._titleInputListenerAttached) {
            els.editorNoteTitle.removeEventListener('input', onEditorInput);
            state._titleInputListenerAttached = false;
        }
        // 销毁 CM6 编辑器
        destroyCodeMirror();
        state.editingNoteId = null;
        state.selectedTags = [];
        state._editSnapshot = null;
        state._defaultNewNoteTitle = null;
        // 字数归零
        els.editorWordCount.textContent = '';
        // 清除文件后缀显示
        els.editorFileExt.textContent = '';
        // 重置 Markdown 渲染/编辑显示状态
        els.mdRendered.style.display = 'none';
        els.mdRendered.innerHTML = '';
        delete els.editorOverlay.dataset.mode;
    }, 200);
}

/**
 * 安全关闭编辑器：检查未保存内容，有改动时弹出保存确认
 */
async function closeEditorSafe() {
    // 查看模式或保存按钮不可见 → 直接关闭
    if (els.editorSaveBtn.style.display === 'none') {
        closeEditor();
        return;
    }

    // 新建模式：内容为空 → 直接关闭
    if (!state.editingNoteId) {
        const title = els.editorNoteTitle.value.trim();
        const content = getEditorContent().trim();
        // 标题是默认自动生成的且内容为空 → 未编辑过，直接关闭
        if (state._defaultNewNoteTitle && title === state._defaultNewNoteTitle && !content) {
            closeEditor();
            return;
        }
        if (!title && !content) {
            closeEditor();
            return;
        }
    } else {
        // 编辑模式：有快照且无改动 → 直接关闭
        const snapshot = state._editSnapshot;
        if (snapshot) {
            const currentTitle = els.editorNoteTitle.value.trim();
            const currentContent = getEditorContent().trim();
            const currentTags = [...state.selectedTags].sort();
            const tagsChanged = JSON.stringify(currentTags) !== JSON.stringify(snapshot.tags);
            const extChanged = els.editorFileExt.textContent !== snapshot.fileExt;
            if (currentTitle === snapshot.title && currentContent === snapshot.content && !tagsChanged && !extChanged) {
                closeEditor();
                return;
            }
        } else {
            closeEditor();
            return;
        }
    }

    // 有未保存的内容 → 弹出确认
    const action = await showSaveConfirmDialog('笔记内容尚未保存，是否保存？');
    if (action === 'cancel') return;

    if (action === 'save') {
        if (state.editingNoteId) {
            await updateNote(state.editingNoteId);
        } else {
            await createNote();
        }
        // createNote/updateNote 内已调用了 closeEditor，不再重复执行
        return;
    }
    // discard: 放弃修改，关闭编辑器
    closeEditor();
}

/**
 * 加载标签并渲染编辑器标签选择器
 * @param {boolean} readOnly - 是否为只读模式（标签不可切换）
 */
async function loadTagsForEditor(readOnly) {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllTags) {
            const tags = await window.go.main.App.GetAllTags();
            state.tags = tags || [];
        } else {
            state.tags = getMockTags();
        }
    } catch (err) {
        console.error('加载标签失败:', err);
        state.tags = [];
    }
    renderTagSelector(readOnly);
}

/* ===== 右键菜单函数 ===== */

let contextMenuNoteId = null;

/**
 * 隐藏右键菜单
 */
function hideContextMenu() {
    const menu = els.contextMenu;
    if (!menu.classList.contains('active')) return;
    menu.style.animation = 'modalExit 0.1s ease-in forwards';
    const onEnd = () => {
        menu.classList.remove('active');
        menu.style.animation = '';
        menu.removeEventListener('animationend', onEnd);
    };
    menu.addEventListener('animationend', onEnd);
    contextMenuNoteId = null;
    // 恢复主内容区滚动
    els.mainContent.style.overflow = '';
}

/* ===== 全局暴露给 onclick 的函数 ===== */

/**
 * 打开笔记（编辑模式）
 */
window.openNote = function (id) {
    openEditor(id, false, getNoteOpenFullscreen());
};

/**
 * 查看笔记（只读模式）
 */
window.viewNote = function (id) {
    openEditor(id, true, getNoteOpenFullscreen());
};

/**
 * 显示右键菜单
 */
window.showContextMenu = function (event, noteId) {
    contextMenuNoteId = noteId;
    const menu = els.contextMenu;
    // 更新置顶选项文本
    const note = state.notes.find((n) => n.id === noteId);
    const pinItem = menu.querySelector('[data-action="pin"]');
    if (pinItem && note) {
        pinItem.textContent = note.pinned ? '取消置顶' : '置顶';
    }

    // 计算 transform-origin：靠近左上角还是右下角
    const isRight = event.clientX > window.innerWidth / 2;
    const isBottom = event.clientY > window.innerHeight / 2;
    const originX = isRight ? 'right' : 'left';
    const originY = isBottom ? 'bottom' : 'top';
    menu.style.transformOrigin = `${originX} ${originY}`;

    menu.style.left = event.clientX + 'px';
    menu.style.top = event.clientY + 'px';
    menu.style.animation = 'menuEnter 0.15s ease-out forwards';
    menu.classList.add('active');
    // 锁定主内容区滚动，防止菜单打开时误滚动
    els.mainContent.style.overflow = 'hidden';
};

/**
 * 处理右键菜单点击
 */
window.handleContextAction = function (action) {
    const id = contextMenuNoteId;
    hideContextMenu();
    if (id == null) return;
    switch (action) {
        case 'view':
            window.viewNote(id);
            break;
        case 'edit':
            window.openNote(id);
            break;
        case 'pin':
            window.togglePin(id);
            break;
        case 'delete':
            window.deleteNote(id);
            break;
        case 'copy':
            copyNote(id);
            break;
        case 'export':
            exportNote(id);
            break;
        case 'move':
            openMoveDialog([id]);
            break;
    }
};

/**
 * 置顶切换（全局）
 */
window.togglePin = async function (id) {
    await togglePin(id);
};

/**
 * 处理置顶按钮点击（带动画）
 * 先播放旋转动画，动画结束后执行置顶逻辑
 */
window.handlePinClick = function (event, id) {
    event.stopPropagation();
    const btn = event.currentTarget;
    // 防止重复点击
    if (btn.classList.contains('animating')) return;
    btn.classList.add('animating');
    btn.addEventListener('animationend', async function onPinAnimEnd() {
        btn.removeEventListener('animationend', onPinAnimEnd);
        btn.classList.remove('animating');
        await window.togglePin(id);
    }, { once: true });
};

/**
 * 删除笔记（全局）
 */
window.deleteNote = async function (id) {
    await deleteNote(id);
};

/**
 * 切换编辑器中选择的标签
 */
window.toggleEditorTag = function (tagId, el) {
    const idx = state.selectedTags.indexOf(tagId);
    if (idx > -1) {
        state.selectedTags.splice(idx, 1);
        el.classList.remove('active');
    } else {
        state.selectedTags.push(tagId);
        el.classList.add('active');
    }
    // 点击脉冲动画
    el.classList.add('clicked');
    setTimeout(() => el.classList.remove('clicked'), 250);
};

/**
 * 删除标签（全局）
 */
window.deleteTag = async function (id) {
    await deleteTag(id);
};

/**
 * 按标签搜索（全局）- 打开搜索弹窗并预选该标签作为过滤器
 */
window.searchByTag = function (tagId, tagName) {
    openSearchModal();
    state.searchModalTagIds = new Set([tagId]);
    updateTagFilterLabel();
    updateSearchModalFilterBtnActive();
    _triggerFilterSearch();
};

/* ===== 批量管理函数 ===== */

/**
 * 切换批量管理模式
 */
function toggleBatchMode() {
    state.batchMode = !state.batchMode;
    const bar = els.batchBar;

    if (state.batchMode) {
        // 进入批量模式
        clearSelection();
        bar.style.display = 'flex';
        bar.style.animation = 'slideUp 0.25s ease-out forwards';
        renderCardGrid('none');
        updateBatchBar();

        // 复选框交错淡入
        const checkboxes = document.querySelectorAll('.batch-checkbox');
        checkboxes.forEach((cb, i) => {
            cb.style.animation = `scaleBounce 0.3s ease-out ${i * 20}ms forwards`;
        });
    } else {
        // 退出批量模式：清空选中 + bar 滑出
        clearSelection();
        bar.style.animation = 'slideDown 0.15s ease-in forwards';
        bar.addEventListener('animationend', function onEnd() {
            bar.style.display = 'none';
            bar.style.animation = '';
            bar.removeEventListener('animationend', onEnd);
        });
        renderCardGrid('none');
        updateBatchBar();
    }
}

/**
 * 切换笔记选中状态
 */
window.toggleNoteSelection = function (id) {
    if (state.selectedNoteIds.has(id)) {
        state.selectedNoteIds.delete(id);
    } else {
        state.selectedNoteIds.add(id);
    }
    updateBatchBar();
    renderCardGrid('none');
};

/**
 * 全选/取消全选（全选时从后端获取所有笔记 ID）
 */
function toggleSelectAll() {
    const allSelected = state.selectedNoteIds.size === state.totalAllNotes;

    if (allSelected) {
        // 取消全选
        state.selectedNoteIds.clear();
        updateBatchBar();
        renderCardGrid('none');
    } else {
        // 全选：先从后端拉取所有 ID，再塞入选中的 Set
        selectAllIds();
    }
}

/**
 * 全选：获取当前笔记本的所有笔记 ID
 */
async function selectAllIds() {
    let ids = [];
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNoteIDsByNotebook) {
        try {
            ids = await window.go.main.App.GetNoteIDsByNotebook(state.activeNotebookId);
        } catch (err) {
            console.error('获取笔记本笔记 ID 失败，降级为当前页:', err);
            ids = state.notes.map(n => n.id);
        }
    } else {
        ids = state.notes.map(n => n.id);
    }
    ids.forEach(id => state.selectedNoteIds.add(id));
    updateBatchBar();
    renderCardGrid('none');
}

/**
 * 更新批量操作栏
 */
function updateBatchBar() {
    const count = state.selectedNoteIds.size;
    els.batchCount.textContent = count;
    // 同步全选按钮文字
    const total = state.totalAllNotes || state.notes.length;
    if (els.batchSelectAllBtn) {
        if (total > 0 && count === total) {
            els.batchSelectAllBtn.textContent = '取消全选';
        } else {
            els.batchSelectAllBtn.textContent = '全选';
        }
    }
    // 更新批量置顶按钮文字
    if (els.batchPinBtn && count > 0) {
        const allPinned = Array.from(state.selectedNoteIds).every(id => {
            const note = state.notes.find(n => n.id === id);
            return note && note.pinned;
        });
        els.batchPinBtn.textContent = allPinned ? '取消置顶' : '置顶';
    }
}

/**
 * 取消选中
 */
function clearSelection() {
    state.selectedNoteIds.clear();
    updateBatchBar();
    if (state.batchMode) {
        renderCardGrid('none');
    }
}

/**
 * 批量删除选中的笔记
 */
async function batchDeleteSelected() {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchDeleteNotes) {
            await window.go.main.App.BatchDeleteNotes(ids);
        } else {
            console.warn('BatchDeleteNotes 未绑定，模拟批量删除');
            mockNotes = mockNotes.filter(n => !ids.includes(n.id));
        }
    } catch (err) {
        console.error('批量删除失败:', err);
        return;
    }
    clearSelection();
    await loadNotes();
    await loadNotebooks();
    nm.showUndo(`已删除 ${ids.length} 条笔记`, () => undoDelete(ids));
}

/**
 * 批量置顶/取消置顶选中的笔记
 * 判断策略：如果至少有一条未置顶则全部置顶，否则全部取消置顶
 */
async function batchPinSelected() {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) return;

    // 判断当前选中笔记中是否有未置顶的
    const anyUnpinned = state.notes.some(n => ids.includes(n.id) && !n.pinned);
    const pin = anyUnpinned; // 有未置顶 → 全部置顶；全部已置顶 → 取消置顶

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchPinNotes) {
            await window.go.main.App.BatchPinNotes(ids, pin);
        } else {
            console.warn('BatchPinNotes 未绑定，模拟批量置顶切换');
            state.notes.forEach(n => {
                if (ids.includes(n.id)) n.pinned = pin;
            });
        }
        // 本地同步更新 state.notes 中的 pinned 状态
        state.notes.forEach(n => {
            if (ids.includes(n.id)) n.pinned = pin;
        });
    } catch (err) {
        console.error('批量置顶/取消置顶失败:', err);
        return;
    }
    clearSelection();
    renderCardGrid('none');
    nm.showUndo(`已${pin ? '置顶' : '取消置顶'} ${ids.length} 条笔记`);
}

/* ===== 批量标签操作 ===== */

let batchTagAction = null; // 'add' | 'remove'

/**
 * 收集选中笔记中已包含的标签 ID 集合
 */
function getTagIdsInSelectedNotes() {
    const ids = new Set();
    for (const note of state.notes) {
        if (state.selectedNoteIds.has(note.id) && note.tags) {
            note.tags.forEach(t => ids.add(t.id));
        }
    }
    return ids;
}

/**
 * 打开批量标签选择弹窗
 */
function openBatchTagPicker(action) {
    if (state.selectedNoteIds.size === 0) {
        nm.show('请先选择笔记', 'warning');
        return;
    }
    batchTagAction = action;
    const isAdd = action === 'add';
    els.batchTagTitle.textContent = isAdd ? '批量添加标签' : '批量移除标签';

    // 移除模式：先检查选中笔记是否包含标签
    if (!isAdd) {
        const tagIdsInNotes = getTagIdsInSelectedNotes();
        if (tagIdsInNotes.size === 0) {
            nm.show('当前选中的笔记中没有可移除的标签', 'info');
            batchTagAction = null;
            return;
        }
    }

    // 显示底部确认按钮
    els.batchTagFooter.style.display = 'flex';
    els.batchTagConfirmBtn.textContent = isAdd ? '确定添加' : '确定移除';

    renderBatchTagList();
    els.batchTagOverlay.style.display = 'flex';
    requestAnimationFrame(() => {
        els.batchTagOverlay.style.opacity = '1';
    });
}

/**
 * 关闭弹窗
 */
function closeBatchTagPicker() {
    els.batchTagOverlay.style.display = 'none';
    els.batchTagOverlay.style.opacity = '';
    els.batchTagFooter.style.display = 'none';
    batchTagAction = null;
}

/**
 * 渲染批量标签列表
 */
function renderBatchTagList() {
    const list = els.batchTagList;
    if (!state.tags || state.tags.length === 0) {
        list.innerHTML = '<div class="batch-tag-empty">暂无标签，请先在设置中创建标签</div>';
        return;
    }

    const isRemove = batchTagAction === 'remove';
    const tagIdsInNotes = isRemove ? getTagIdsInSelectedNotes() : new Set();

    list.innerHTML = state.tags
        .map(tag => {
            // 移除模式：不在选中笔记中的标签不可选
            const disabled = isRemove && !tagIdsInNotes.has(tag.id);
            const cls = `batch-tag-chip${disabled ? ' disabled' : ''}`;
            return `<div class="${cls}" data-tag-id="${tag.id}" data-tag-color="${tag.color || '#6B7280'}" style="--tag-color:${tag.color || '#6B7280'}">${escapeHtml(tag.name)}</div>`;
        })
        .join('');

    // 绑定点击事件
    list.querySelectorAll('.batch-tag-chip:not(.disabled)').forEach(el => {
        el.addEventListener('click', () => onBatchTagClick(el));
    });
}

/**
 * 点击标签：切换选中态，更新确认按钮计数
 */
function onBatchTagClick(el) {
    el.classList.toggle('selected');
    const isAdd = batchTagAction === 'add';
    const count = els.batchTagList.querySelectorAll('.batch-tag-chip.selected').length;
    const label = isAdd ? '确定添加' : '确定移除';
    els.batchTagConfirmBtn.textContent = count > 0 ? `${label}（${count}）` : label;
}

/**
 * 确认批量标签操作（由确定按钮触发）
 */
async function confirmBatchTagAction() {
    const selectedChips = els.batchTagList.querySelectorAll('.batch-tag-chip.selected');
    if (selectedChips.length === 0) {
        nm.show('请先选择标签', 'warning');
        return;
    }
    const isAdd = batchTagAction === 'add';
    const ids = Array.from(state.selectedNoteIds);
    const tagNames = [];
    try {
        for (const chip of selectedChips) {
            const tagId = parseInt(chip.dataset.tagId);
            tagNames.push(chip.textContent);
            if (isAdd) {
                await window.go.main.App.BatchAddTagToNotes(ids, tagId);
            } else {
                await window.go.main.App.BatchRemoveTagFromNotes(ids, tagId);
            }
        }
    } catch (err) {
        console.error(`批量${isAdd ? '添加' : '移除'}标签失败:`, err);
        closeBatchTagPicker();
        nm.show('操作失败', 'error');
        return;
    }
    closeBatchTagPicker();
    // 不退出批量模式，保持选中状态
    await loadNotes();
    nm.show(`已${isAdd ? '添加' : '移除'} ${tagNames.length} 个标签`, 'success');
}

/* ===== 移动到目标笔记本 ===== */

/** 当前待迁移的笔记 ID 列表 */
let moveNoteIds = [];

/**
 * 打开目标笔记本选择器弹窗
 * @param {number[]} noteIds - 要迁移的笔记 ID 数组
 */
async function openMoveDialog(noteIds) {
    moveNoteIds = noteIds;
    const dialog = els.moveNotebookDialog;
    const list = els.moveNotebookList;
    const empty = els.moveNotebookEmpty;
    const confirmBtn = els.moveNotebookConfirm;
    const allNotebooks = state.notebooks || [];

    // 过滤：排除当前笔记本
    const targets = allNotebooks.filter(nb => nb.id !== state.activeNotebookId);

    // 重置弹窗状态
    list.innerHTML = '';
    confirmBtn.disabled = true;
    empty.style.display = 'none';

    if (targets.length === 0) {
        // 空状态：没有其他笔记本
        empty.style.display = 'block';
    } else {
        // 渲染笔记本列表
        targets.forEach(nb => {
            const item = document.createElement('div');
            item.className = 'move-notebook-item';
            item.dataset.id = nb.id;
            item.innerHTML = `
                <div class="move-notebook-radio"></div>
                <span class="move-notebook-name">${escapeHtml(nb.name)}</span>
                <span class="move-notebook-badge">${nb.noteCount ?? 0}</span>
            `;
            item.addEventListener('click', () => {
                // 取消其他项的选中
                list.querySelectorAll('.move-notebook-item.selected').forEach(el => el.classList.remove('selected'));
                // 选中当前项
                item.classList.add('selected');
                confirmBtn.disabled = false;
            });
            list.appendChild(item);
        });
    }

    // 显示弹窗（弹簧动画由 CSS 驱动）
    dialog.style.display = 'flex';
    requestAnimationFrame(() => {
        dialog.classList.add('visible');
    });
}

/** 关闭目标笔记本选择器弹窗 */
function closeMoveDialog() {
    const dialog = els.moveNotebookDialog;
    dialog.classList.remove('visible');
    // 等出场动画完成后隐藏
    setTimeout(() => {
        dialog.style.display = 'none';
    }, 200);
}

/** 确认迁移 — 将选中的笔记移动到目标笔记本 */
async function confirmMoveNotes() {
    const selectedItem = els.moveNotebookList.querySelector('.move-notebook-item.selected');
    if (!selectedItem || moveNoteIds.length === 0) return;

    const targetId = parseInt(selectedItem.dataset.id);
    const targetName = selectedItem.querySelector('.move-notebook-name').textContent;
    const confirmBtn = els.moveNotebookConfirm;

    // 防止重复点击
    confirmBtn.disabled = true;

    try {
        if (moveNoteIds.length === 1) {
            // 单条迁移
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.MoveNoteToNotebook) {
                await window.go.main.App.MoveNoteToNotebook(moveNoteIds[0], targetId);
            } else {
                console.warn('MoveNoteToNotebook 未绑定，模拟迁移');
            }
        } else {
            // 批量迁移
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchMoveNotesToNotebook) {
                await window.go.main.App.BatchMoveNotesToNotebook(moveNoteIds, targetId);
            } else {
                console.warn('BatchMoveNotesToNotebook 未绑定，模拟批量迁移');
            }
        }

        closeMoveDialog();
        // 刷新笔记列表和笔记本列表（badge 计数同步更新）
        await loadNotes();
        await loadNotebooks();
        renderNotebookList();
        nm.show(`已将 ${moveNoteIds.length} 条笔记移动到「${targetName}」`, 'success');
    } catch (err) {
        console.error('迁移笔记失败:', err);
        nm.show('迁移笔记失败: ' + (err.message || err), 'error');
        confirmBtn.disabled = false;
    }

    moveNoteIds = [];
}

/* ===== HTML 转义 ===== */

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/* ===== 更多菜单动画 ===== */

/**
 * 打开更多菜单
 */
function openMoreMenu(menu) {
    menu.style.animation = 'menuEnter 0.15s ease-out forwards';
    menu.classList.add('active');
}

/**
 * 关闭更多菜单（含离场动画）
 */
function closeMoreMenu(menu) {
    if (!menu.classList.contains('active')) return;
    menu.style.animation = 'modalExit 0.1s ease-in forwards';
    const onEnd = () => {
        menu.classList.remove('active');
        menu.style.animation = '';
        menu.removeEventListener('animationend', onEnd);
    };
    menu.addEventListener('animationend', onEnd);
}

/* ===== 事件绑定 ===== */

function initEventListeners() {
    // 全局禁用浏览器默认右键菜单（应用已有自定义右键菜单）
    document.addEventListener('contextmenu', (e) => e.preventDefault());

    // 搜索(已迁移到弹窗,事件在 initSearchModalListeners 中绑定)

    // 浮动新建按钮
    els.fabNewNote.addEventListener('click', () => {
        openEditor(null, false, getNoteOpenFullscreen());
    });

    // 回到顶部
    els.backToTopBtn.addEventListener('click', () => {
        els.mainContent.scrollTo({ top: 0, behavior: 'smooth' });
    });

    // 笔记本侧栏折叠/展开按钮
    els.notebookSidebarToggle?.addEventListener('click', () => {
        toggleSidebar();
        updateNotebookSidebarToggleBtn();
    });

    // 更多菜单按钮
    els.moreMenuBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const menu = els.moreMenu;
        const isOpen = menu.classList.contains('active');
        if (isOpen) {
            closeMoreMenu(menu);
        } else {
            openMoreMenu(menu);
        }
    });
    // 更多菜单项点击
    els.moreMenu.addEventListener('click', (e) => {
        const item = e.target.closest('.dropdown-item');
        if (item && item.dataset.action) {
            closeMoreMenu(els.moreMenu);
            if (item.dataset.action === 'home') {
                state.searchKeyword = '';
                switchView('grid');
                resetPagination();
                loadNotes();
            } else if (item.dataset.action === 'sidebar-toggle') {
                toggleSidebar();
            } else if (item.dataset.action === 'batch-mode') {
                switchView('grid');
                toggleBatchMode();
            } else if (item.dataset.action === 'settings') {
                switchView('settings');
            } else if (item.dataset.action === 'data') {
                switchView('data');
            } else if (item.dataset.action === 'trash') {
                switchView('trash');
                loadTrashNotes();
            } else if (item.dataset.action === 'md-ref') {
                switchView('md-ref');
            } else if (item.dataset.action === 'ai-chat') {
                switchView('ai-chat');
            } else if (item.dataset.action === 'help') {
                openShortcuts();
            }
        }
    });

    // 点击品牌名返回所有笔记
    document.querySelector('.topbar-brand')?.addEventListener('click', () => {
        state.searchKeyword = '';
        switchView('grid');
    });

    // 编辑器
    els.editorCloseBtn.addEventListener('click', closeEditorSafe);
    els.editorTypeToggle?.addEventListener('click', toggleFileExt);
    els.editorEditBtn.addEventListener('click', () => {
        const noteId = state.editingNoteId;
        if (noteId) {
            state.enteredFromViewMode = true;
            // 内联切换为编辑模式，不重建 CM6 实例，避免闪烁
            switchEditorReadOnly(false);
        }
    });
    els.editorViewBtn.addEventListener('click', async () => {
        const noteId = state.editingNoteId;
        if (!noteId) return;

        const snapshot = state._editSnapshot;
        const title = els.editorNoteTitle.value.trim();
        const content = getEditorContent().trim();
        const currentTags = [...state.selectedTags].sort();

        // 变更检测：无修改则静默切回查看模式
        const tagsChanged = snapshot ? JSON.stringify(currentTags) !== JSON.stringify(snapshot.tags) : true;
        const extChanged = snapshot ? els.editorFileExt.textContent !== snapshot.fileExt : true;
        const hasChanged = !snapshot || title !== snapshot.title || content !== snapshot.content || tagsChanged || extChanged;

        state.enteredFromViewMode = false;

        if (!hasChanged) {
            // 无变更：直接切回查看模式，不弹通知（内联切换，避免闪烁）
            switchEditorReadOnly(true);
            return;
        }

        // 有变更：保存 + 通知 + 切回查看模式
        if (title && window.go?.main?.App?.UpdateNote) {
            try {
                await window.go.main.App.UpdateNote(noteId, title, content, els.editorFileExt.textContent);
                // 更新标签：先移除所有标签再重新添加选中的
                const note = await window.go.main.App.GetNote(noteId);
                if (note?.tags) {
                    for (const t of note.tags) {
                        try { await window.go.main.App.RemoveTagFromNote(noteId, t.id); } catch (e) {}
                    }
                }
                for (const tagId of state.selectedTags) {
                    try { await window.go.main.App.AddTagToNote(noteId, tagId); } catch (e) {}
                }
            } catch (err) {
                console.error('保存失败:', err);
            }
        }
        nm.show('笔记已更新', 'success');
        // 同步更新 state.notes 中的本地缓存，避免 loadNotes() 全量刷新
        const cached = state.notes.find(n => n.id === noteId);
        if (cached) {
            cached.title = title;
            cached.content = content;
            cached.file_ext = els.editorFileExt.textContent;
            cached.updated_at = new Date().toISOString();
        }
        state._editSnapshot = null;
        // 内联切回查看模式，不重建 CM6 实例，避免闪烁
        switchEditorReadOnly(true);
        await loadNotes();
    });
    els.editorFullscreenBtn.addEventListener('click', toggleEditorFullscreen);
    els.editorCancelBtn.addEventListener('click', closeEditorSafe);
    els.editorSaveBtn.addEventListener('click', async () => {
        if (state.editingNoteId) {
            await updateNote(state.editingNoteId);
        } else {
            await createNote();
        }
    });
    // 点击蒙层关闭编辑器（编辑/新建模式有未保存内容时弹出保存确认）
    els.editorOverlay.addEventListener('click', async (e) => {
        if (e.target !== els.editorOverlay) return;
        await closeEditorSafe();
    });

    // 编辑器模式切换（纯文本/分栏/预览）
    els.editorModeBtns.forEach(btn => {
        btn.addEventListener('click', () => switchEditorMode(btn.dataset.mode));
    });

    // 标签管理
    els.addTagBtn.addEventListener('click', createTag);
    els.newTagName.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') createTag();
    });

    // 回收站按钮
    els.trashBackBtn.addEventListener('click', () => {
        switchView('grid');
    });
    els.restoreAllBtn.addEventListener('click', window.restoreAllNotes);
    els.emptyTrashBtn.addEventListener('click', window.emptyTrash);

    // 数据管理按钮
    els.dataBackBtn.addEventListener('click', () => {
        switchView('grid');
    });
    els.exportDataBtn.addEventListener('click', exportData);
    els.importDataBtn.addEventListener('click', importData);
    // 备份还原按钮
    els.backupBtn?.addEventListener('click', backupToDir);
    els.restoreBtn?.addEventListener('click', restoreFromDir);
    els.resetAllBtn.addEventListener('click', resetDatabase);
    els.vacuumDbBtn.addEventListener('click', vacuumDatabase);
    els.openDataDirBtn.addEventListener('click', openDataDir);

    els.mdRefBackBtn.addEventListener('click', () => {
        switchView('grid');
    });

    els.settingsBackBtn.addEventListener('click', () => {
        switchView('grid');
    });

    // 快速笔记开关
    els.quickNoteToggle.addEventListener('change', async (e) => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
                await window.go.main.App.SetSetting('quick_note_enabled', String(e.target.checked));
                nm.show('设置已保存', 'success');
            } else {
                localStorage.setItem('quick_note_enabled', String(e.target.checked));
                nm.show('设置已保存', 'success');
            }
        } catch (err) {
            console.error('保存快速笔记设置失败:', err);
        }
    });

    // 语法高亮开关
    els.mdHighlightToggle.addEventListener('change', async (e) => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
                await window.go.main.App.SetSetting('cm_syntax_highlight', String(e.target.checked));
                nm.show('设置已保存', 'success');
            } else {
                localStorage.setItem('cm_syntax_highlight', String(e.target.checked));
                nm.show('设置已保存', 'success');
            }
        } catch (err) {
            console.error('保存语法高亮设置失败:', err);
        }
    });

    // 全屏打开笔记开关
    els.noteOpenFullscreenToggle.addEventListener('change', async (e) => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
                await window.go.main.App.SetSetting('note_open_fullscreen', String(e.target.checked));
                nm.show('设置已保存', 'success');
            } else {
                localStorage.setItem('note_open_fullscreen', String(e.target.checked));
                nm.show('设置已保存', 'success');
            }
        } catch (err) {
            console.error('保存全屏打开设置失败:', err);
        }
    });

    // 右键菜单：点击其他区域关闭
    document.addEventListener('click', hideContextMenu);
    document.addEventListener('click', () => closeMoreMenu(els.moreMenu));
    // 右键菜单项点击
    els.contextMenu.addEventListener('click', (e) => {
        const item = e.target.closest('.context-menu-item');
        if (item && item.dataset.action) {
            e.stopPropagation();
            window.handleContextAction(item.dataset.action);
        }
    });
    // 右键菜单内阻止冒泡，避免触发 document.click 关闭
    els.contextMenu.addEventListener('contextmenu', (e) => e.preventDefault());

    // 批量管理模式
    els.batchDeleteBtn.addEventListener('click', batchDeleteSelected);
    els.batchCancelBtn.addEventListener('click', () => {
        if (state.batchMode) toggleBatchMode();
    });
    els.batchSelectAllBtn.addEventListener('click', toggleSelectAll);

    // 批量置顶
    els.batchPinBtn.addEventListener('click', batchPinSelected);

    // 批量标签操作
    els.batchAddTagBtn.addEventListener('click', () => openBatchTagPicker('add'));
    els.batchRemoveTagBtn.addEventListener('click', () => openBatchTagPicker('remove'));
    els.batchTagCloseBtn.addEventListener('click', closeBatchTagPicker);
    els.batchTagConfirmBtn.addEventListener('click', confirmBatchTagAction);
    els.batchTagOverlay.addEventListener('click', (e) => {
        if (e.target === els.batchTagOverlay) closeBatchTagPicker();
    });

    // 移动到目标笔记本
    els.batchMoveBtn.addEventListener('click', () => {
        if (state.selectedNoteIds.size === 0) return;
        openMoveDialog([...state.selectedNoteIds]);
    });
    els.moveNotebookClose.addEventListener('click', closeMoveDialog);
    els.moveNotebookCancel.addEventListener('click', closeMoveDialog);
    els.moveNotebookConfirm.addEventListener('click', confirmMoveNotes);
    els.moveNotebookDialog.addEventListener('click', (e) => {
        if (e.target === els.moveNotebookDialog) closeMoveDialog();
    });

    // 关于页面
    document.querySelector('.brand-name').addEventListener('click', (e) => {
        e.stopPropagation();
        showAbout();
    });
    els.aboutCloseBtn.addEventListener('click', closeAbout);
    els.viewAbout.addEventListener('click', (e) => {
        if (e.target === els.viewAbout) closeAbout();
    });
    els.aboutProjectLink.addEventListener('click', async () => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenProjectURL) {
                await window.go.main.App.OpenProjectURL('https://gitee.com/MM-Q/jot.git');
            } else {
                // 后端未绑定时使用替代方案
                window.open('https://gitee.com/MM-Q/jot.git', '_blank');
            }
        } catch (err) {
            console.error('打开项目地址失败:', err);
        }
    });

    // 快捷键关闭按钮
    els.shortcutsCloseBtn.addEventListener('click', closeShortcuts);
    els.shortcutsView.addEventListener('click', (e) => {
        if (e.target === els.shortcutsView) closeShortcuts();
    });

    // 键盘快捷键导航
    document.addEventListener('keydown', handleKeyboardNavigation);

    // 笔记本侧栏事件
    els.newNotebookBtn?.addEventListener('click', showNewNotebookDialog);

    // 笔记本侧栏键盘导航
    if (els.notebookList) {
        els.notebookList.addEventListener('keydown', handleNotebookListKeydown);
        els.notebookList.addEventListener('blur', () => {
            clearNotebookKeyboardFocus();
        });
    }

    // 文件后缀编辑对话框事件
    els.editorFileExt.addEventListener('click', openFileExtDialog);
    document.getElementById('fileExtSaveBtn').addEventListener('click', saveFileExt);
    document.getElementById('fileExtCancelBtn').addEventListener('click', closeFileExtDialog);
    document.getElementById('fileExtInput').addEventListener('keydown', (e) => {
        if (e.key === 'Enter') saveFileExt();
        if (e.key === 'Escape') {
            e.stopPropagation();
            closeFileExtDialog();
        }
    });
    document.querySelector('.file-ext-dialog-overlay').addEventListener('click', closeFileExtDialog);

    // 搜索弹窗事件绑定(替代原 topbar 搜索框)
    initSearchModalListeners();
}

/* ===== 键盘快捷键导航 ===== */

/**
 * 获取当前视图的可滚动容器
 */
function getScrollContainer() {
    switch (state.currentView) {
        case 'grid':
        case 'data':
        case 'trash':
            return els.mainContent;
        default:
            return null;
    }
}

// FindReplaceManager 已删除（CM6 search 替代）

/**
 * 处理键盘快捷键（Ctrl+Home/End, PgUp/PgDn, Ctrl+F, Ctrl+H）
 */
async function handleKeyboardNavigation(e) {
    const container = getScrollContainer();

    // Ctrl+S: 编辑器内保存（编辑/新建模式有效，查看模式忽略）
    if (e.ctrlKey && (e.key === 's' || e.key === 'S')) {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active') && els.editorSaveBtn.style.display !== 'none') {
            (state.editingNoteId ? updateNote(state.editingNoteId) : createNote());
        }
        return;
    }

    // Ctrl+F: 编辑器内搜索（自动填充选中文本，预览模式自动切到编辑模式）;编辑器外则打开搜索弹窗
    if (e.ctrlKey && e.key === 'f') {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active') && cmEditor) {
            // 预览模式自动切回编辑模式
            if (els.editorOverlay.dataset.mode === 'preview') {
                switchEditorMode('edit');
            }
            // 将当前选中文本填充到搜索框
            const sel = cmEditor.state.selection.main;
            if (!sel.empty) {
                const selectedText = cmEditor.state.sliceDoc(sel.from, sel.to);
                cmEditor.dispatch({
                    effects: setSearchQuery.of(new SearchQuery({ search: selectedText }))
                });
            }
            openSearchPanel(cmEditor);
            return;
        }
        // 编辑器外:打开搜索弹窗(替代原 topbar 搜索框聚焦)
        openSearchModal();
        return;
    }

    // Ctrl+H: 编辑器内查找替换（仅在编辑模式生效）
    if (e.ctrlKey && e.key === 'h') {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active') && cmEditor && els.editorOverlay.dataset.mode !== 'preview') {
            openSearchPanel(cmEditor);
        }
        return;
    }

    // Ctrl+N: 打开新建笔记（编辑器未打开时）
    if (e.ctrlKey && e.key === 'n') {
        e.preventDefault();
        if (!els.viewEditor.classList.contains('active')) {
            openEditor(null, false, getNoteOpenFullscreen());
        }
        return;
    }

    // Ctrl+L: 编辑器打开时切换编辑/预览模式（仅 Markdown 模式支持）
    if (e.ctrlKey && (e.key === 'l' || e.key === 'L') && els.viewEditor.classList.contains('active') && els.editorFileExt.textContent === '.md') {
        e.preventDefault();
        const current = els.editorOverlay.dataset.mode;
        switchEditorMode(current === 'preview' ? 'edit' : 'preview');
        return;
    }

    // Ctrl+E: 切换编辑器全屏模式（仅编辑器打开时有效）
    if (e.ctrlKey && (e.key === 'e' || e.key === 'E')) {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active')) {
            toggleEditorFullscreen();
        }
        return;
    }

    // F11: 切换窗口 OS 全屏（全局可用，与编辑器全屏独立）
    if (e.key === 'F11') {
        e.preventDefault();
        WindowIsFullscreen().then(isWinFs => {
            if (isWinFs) {
                WindowUnfullscreen();
            } else {
                WindowFullscreen();
            }
        });
        return;
    }

    // Ctrl+Q: 退出程序（全局生效，退出前提示保存）
    if (e.ctrlKey && (e.key === 'q' || e.key === 'Q')) {
        e.preventDefault();
        await handleAppExit();
        return;
    }

    // Escape: 关闭查找条或退出当前子视图
    if (e.key === 'Escape') {
        e.preventDefault();
        // 关于页面打开时关闭它
        if (els.viewAbout.style.display === 'flex') {
            closeAbout();
            return;
        }
        // 如果编辑器处于全屏模式，先退出全屏
        if (els.editorPanel.classList.contains('fullscreen')) {
            toggleEditorFullscreen();
            return;
        }
        // 编辑器打开时关闭它（检查未保存内容）
        if (els.viewEditor.classList.contains('active')) {
            closeEditorSafe();
            return;
        }
        if (els.shortcutsView.style.display !== 'none') {
            closeShortcuts();
            return;
        }
        if (state.batchMode) {
            toggleBatchMode();
        } else if (state.currentView === 'search') {
            // 搜索页：清空搜索后回到首页
            state.searchKeyword = '';
            state.searchSource = 'input';
            switchView('grid');
            loadNotes();
        } else if (state.currentView !== 'grid') {
            // 设置、数据管理、回收站等子视图 → 回到首页
            switchView('grid');
            loadNotes();
        }
        return;
    }

    // Ctrl+A/Ctrl+D 快捷键处理
    if ((e.ctrlKey || e.metaKey) && !e.target.closest('input, textarea, [contenteditable]')) {
        if (e.key === 'a') {
            e.preventDefault(); // 阻止浏览器默认全选
            if (state.batchMode && state.currentView === 'grid') {
                selectAllIds();
            }
            return;
        }
        if (e.key === 'd') {
            if (state.batchMode && state.currentView === 'grid') {
                e.preventDefault();
                state.selectedNoteIds.clear();
                updateBatchBar();
                renderCardGrid('none');
                return;
            }
        }
    }

    // Ctrl+数字键快捷导航（仅在非输入框内生效）
    if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.target.closest('input, textarea, [contenteditable]')) {
        switch (e.key) {
            case '1':
                e.preventDefault();
                state.searchKeyword = '';
                switchView('grid');
                return;
            case '2':
                e.preventDefault();
                toggleSidebar();
                updateSidebarMenuItem();
                return;
            case '3':
                e.preventDefault();
                switchView('grid');
                toggleBatchMode();
                return;
            case '4':
                e.preventDefault();
                switchView('data');
                return;
            case '5':
                e.preventDefault();
                switchView('trash');
                return;
            case '6':
                e.preventDefault();
                switchView('settings');
                return;
            case '7':
                e.preventDefault();
                if (els.shortcutsView.style.display !== 'none') {
                    closeShortcuts();
                } else {
                    openShortcuts();
                }
                return;
            case '8':
                e.preventDefault();
                switchView('md-ref');
                return;
            case '9':
                e.preventDefault();
                switchView('ai-chat');
                return;
        }
    }

    // 编辑器打开时，Ctrl+Home/End 和 PgUp/PgDn 交由编辑器/textarea 原生处理
    if (els.viewEditor.classList.contains('active') &&
        ((e.ctrlKey && (e.key === 'Home' || e.key === 'End')) || e.key === 'PageUp' || e.key === 'PageDown')) {
        return;
    }

    if (!container) return;

    // Ctrl+Home: 滚动到顶部
    if (e.ctrlKey && e.key === 'Home') {
        e.preventDefault();
        container.scrollTop = 0;
        return;
    }
    // Ctrl+End: 加载所有剩余页后跳到底部
    if (e.ctrlKey && e.key === 'End') {
        e.preventDefault();
        if (hasMoreNotes && !isLoadingMore) {
            loadAllRemainingNotes();
        } else {
            // 无需加载，直接跳到底部
            container.scrollTop = container.scrollHeight;
        }
        return;
    }
    // PgUp: 向上翻一页
    if (e.key === 'PageUp') {
        e.preventDefault();
        container.scrollTop -= container.clientHeight;
        return;
    }
    // PgDn: 向下翻一页；已到底时加载下一页
    if (e.key === 'PageDown') {
        e.preventDefault();
        const { scrollTop, scrollHeight, clientHeight } = container;
        if (scrollTop + clientHeight >= scrollHeight - 1) {
            // 已到底，主动加载下一页（不走 scroll 事件）
            if (hasMoreNotes && !isLoadingMore) {
                loadMoreNotes();
            }
            return;
        }
        _keyboardScroll = true;
        container.scrollTop = scrollTop + clientHeight;
        requestAnimationFrame(() => { _keyboardScroll = false; });
        return;
    }
}

/* ===== 滚动懒加载 ===== */

// 键盘滚动标志：键盘触发的滚动不触发懒加载
let _keyboardScroll = false;

/**
 * 绑定懒加载滚动事件（监听主内容区滚动到底部附近）
 */
function initScrollLoading() {
    els.mainContent.addEventListener('scroll', () => {
        if (state.currentView !== 'grid') return;
        // 键盘触发的滚动不触发懒加载
        if (_keyboardScroll) return;

        const { scrollTop, scrollHeight, clientHeight } = els.mainContent;
        if (scrollHeight - scrollTop - clientHeight < 200) {
            loadMoreNotes();
        }
    });
}

/* ===== 滚动条自动显隐 ===== */

/**
 * 给滚动容器绑定 scroll 事件：滚动时显示滑块，停止 1 秒后淡出；
 * 同时控制"回到顶部"按钮的显隐
 */
function initScrollbarAutoHide() {
    const containers = [els.mainContent, document.querySelector('.ai-chat-messages')].filter(Boolean);
    containers.forEach((container) => {
        let timer = null;
        container.addEventListener('scroll', (e) => {
            // 忽略子元素冒泡上来的 scroll 事件，只处理容器自身的滚动
            if (e.target !== container) return;
            container.classList.add('scrolling');
            clearTimeout(timer);
            timer = setTimeout(() => {
                container.classList.remove('scrolling');
            }, 1000);
        });
    });
    // 主内容区滚动时控制 "↑" 按钮显隐（阈值 300px）
    if (els.mainContent) {
        els.mainContent.addEventListener('scroll', () => {
            const scrollY = els.mainContent.scrollTop;
            els.backToTopBtn.classList.toggle('visible', scrollY > 300);
        });
    }
}

/* ===== 关于页面 ===== */

/**
 * 打开关于页面（带动画），获取版本信息
 */
async function showAbout() {
    els.viewAbout.style.display = 'flex';
    // 遮罩淡入
    els.viewAbout.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
    // 内容卡片缩放淡入
    const card = els.viewAbout.querySelector('.about-card');
    card.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
    // 品牌 Logo 弹性缩放
    const logo = els.viewAbout.querySelector('.about-logo');
    logo.style.animation = 'scaleBounce 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
    // 版本号延迟淡入
    els.aboutVersion.style.animation = 'animFadeIn 0.2s ease-out forwards';
    els.aboutVersion.style.animationDelay = '100ms';

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetVersion) {
            const version = await window.go.main.App.GetVersion();
            els.aboutVersion.textContent = version || '-';
        } else {
            // 后端未绑定时使用 Mock
            els.aboutVersion.textContent = 'v0.0.0 (Mock)';
        }
    } catch (err) {
        console.error('获取版本信息失败:', err);
        els.aboutVersion.textContent = '-';
    }
}

/**
 * 关闭关于页面（带动画）
 */
function closeAbout() {
    const card = els.viewAbout.querySelector('.about-card');
    card.style.animation = 'modalExit 0.15s ease-in forwards';
    els.viewAbout.style.animation = 'overlayFadeOut 0.15s ease-in forwards';
    // 重置子元素动画
    const logo = els.viewAbout.querySelector('.about-logo');
    logo.style.animation = '';
    els.aboutVersion.style.animation = '';
    els.aboutVersion.style.animationDelay = '';
    // 动画完成后隐藏
    setTimeout(() => {
        els.viewAbout.style.display = 'none';
        els.viewAbout.style.animation = '';
        card.style.animation = '';
    }, 200);
}

/**
 * 打开快捷键说明模态框（带动画）
 */
function openShortcuts() {
    els.shortcutsView.style.display = 'flex';
    // 遮罩淡入
    els.shortcutsView.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
    // 内容卡片缩放淡入
    const card = els.shortcutsView.querySelector('.shortcuts-card');
    card.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
    // 渲染快捷键列表
    renderShortcutsPage();
    // 快捷键列表项交错入场（确保 DOM 已渲染）
    requestAnimationFrame(() => {
        const rows = els.shortcutsBody.querySelectorAll('.shortcut-row');
        rows.forEach((row, index) => {
            row.style.animation = `viewEnter 0.25s cubic-bezier(0.16, 1, 0.3, 1) forwards`;
            row.style.animationDelay = `${index * 30}ms`;
        });
    });
}

/**
 * 关闭快捷键说明模态框（带动画）
 */
function closeShortcuts() {
    const card = els.shortcutsView.querySelector('.shortcuts-card');
    card.style.animation = 'modalExit 0.15s ease-in forwards';
    els.shortcutsView.style.animation = 'overlayFadeOut 0.15s ease-in forwards';
    // 动画完成后隐藏
    setTimeout(() => {
        els.shortcutsView.style.display = 'none';
        // 重置动画
        els.shortcutsView.style.animation = '';
        card.style.animation = '';
        const rows = els.shortcutsBody.querySelectorAll('.shortcut-row');
        rows.forEach(row => {
            row.style.animation = '';
            row.style.animationDelay = '';
        });
    }, 200);
}

/* ===== 快捷键说明页面 ===== */

/**
 * 渲染快捷键说明页面
 */
function renderShortcutsPage() {
    const shortcuts = [
        { key: 'Ctrl + N', desc: '新建笔记' },
        { key: 'Ctrl + S', desc: '编辑器内保存笔记' },
        { key: 'Ctrl + F', desc: '编辑器内查找 / 打开搜索弹窗' },
        { key: 'Ctrl + H', desc: '编辑器内查找替换' },
        { key: 'Ctrl + L', desc: '编辑器切换纯文本/预览' },
        { key: 'Ctrl + E', desc: '编辑器内切换全屏' },
        { key: 'F11', desc: '切换窗口全屏' },
        { key: 'Ctrl + Q', desc: '退出程序' },
        { key: 'PgUp', desc: '上翻一页' },
        { key: 'PgDn', desc: '下翻一页 / 触底加载更多' },
        { key: 'Ctrl + Home', desc: '回到顶部' },
        { key: 'Ctrl + End', desc: '加载全部并滚到底部' },
        { key: 'Escape', desc: '关闭弹窗 / 返回上一页' },
        { key: 'Ctrl + 1', desc: '笔记首页' },
        { key: 'Ctrl + 2', desc: '展开侧栏' },
        { key: 'Ctrl + 3', desc: '批量管理' },
        { key: 'Ctrl + 4', desc: '数据管理' },
        { key: 'Ctrl + 5', desc: '回收站' },
        { key: 'Ctrl + 6', desc: '设置' },
        { key: 'Ctrl + 7', desc: '快捷键说明' },
        { key: 'Ctrl + 8', desc: 'MD 语法' },
        { key: 'Ctrl + 9', desc: 'AI 助手' },
    ];
    els.shortcutsBody.innerHTML = shortcuts.map(s => `
        <div class="shortcut-row">
            <div class="shortcut-key">${s.key.replace(/(\w+)/g, '<kbd>$1</kbd>')}</div>
            <div class="shortcut-desc">${s.desc}</div>
        </div>
    `).join('');
}

/* ===== MD 语法手册渲染 ===== */

/**
 * 渲染 MD 语法手册中的预览卡片（每次进入视图都重新渲染）
 */
function renderMdRefCards() {
    document.querySelectorAll('.md-ref-card').forEach((card) => {
        const script = card.querySelector('.md-ref-source');
        const preview = card.querySelector('.md-ref-preview');
        if (!script || !preview) return;

        const source = script.textContent.trim();
        if (!source) return;

        // 使用 marked 解析 Markdown
        preview.innerHTML = marked.parse(source);

        // 为预览中的代码块启用语法高亮
        preview.querySelectorAll('pre code').forEach((block) => {
            try { hljs.highlightElement(block); } catch (e) { /* ignore */ }
        });
    });

    // 绑定复制按钮
    setupRefCopyButtons();
    // 绑定「打开编辑器试试」按钮
    setupMdRefTryButtons();
    // 绑定 TOC 平滑滚动
    document.querySelectorAll('.md-ref-toc-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = item.getAttribute('href').slice(1);
            const target = document.getElementById(targetId);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                document.querySelectorAll('.md-ref-toc-item').forEach(el => el.classList.remove('active'));
                item.classList.add('active');
                // 标记正在 TOC 滚动中，期间不因 scroll 事件清除选中态
                window._tocScrolling = true;
                clearTimeout(window._tocScrollTimer);
                window._tocScrollTimer = setTimeout(() => { window._tocScrolling = false; }, 800);
            }
        });
    });

    // 绑定 MD 参考页面回到顶部按钮
    const mdRefTopBtn = document.getElementById('mdRefTopBtn');
    if (mdRefTopBtn && !mdRefTopBtn._mdRefTopBound) {
        mdRefTopBtn._mdRefTopBound = true;
        mdRefTopBtn.addEventListener('click', () => {
            els.mainContent.scrollTo({ top: 0, behavior: 'smooth' });
        });
    }

    // 滚动显示/隐藏回到顶部按钮 + 清除 TOC 选中态（只处理一次全局绑定）
    if (!window._mdRefScrollBound) {
        window._mdRefScrollBound = true;
        els.mainContent.addEventListener('scroll', () => {
            const view = document.getElementById('viewMdRef');
            const btn = document.getElementById('mdRefTopBtn');
            if (!view || !btn) return;
            // 仅在 MD 语法视图可见时生效
            if (view.offsetParent !== null) {
                btn.classList.toggle('visible', els.mainContent.scrollTop > 300);
                // 用户手动滚动时清除 TOC 选中态（避开 TOC 平滑滚动期间）
                if (!window._tocScrolling) {
                    document.querySelectorAll('.md-ref-toc-item').forEach(el => el.classList.remove('active'));
                }
            }
        });
    }
}

/**
 * 为语法手册卡片标题栏的复制按钮绑定事件
 */
function setupRefCopyButtons() {
    document.querySelectorAll('.md-ref-editor-copy-btn').forEach(btn => {
        // 避免重复绑定
        if (btn._copyBound) return;
        btn._copyBound = true;

        btn.addEventListener('click', () => {
            const panel = btn.closest('.md-ref-source-panel');
            if (!panel) return;
            const code = panel.querySelector('pre code');
            if (!code) return;

            const text = code.textContent;
            navigator.clipboard.writeText(text).then(() => {
                const origText = btn.textContent;
                btn.textContent = '已复制 ✓';
                btn.classList.add('copied');
                setTimeout(() => {
                    btn.textContent = origText;
                    btn.classList.remove('copied');
                }, 500);
            }).catch(() => {
                btn.textContent = '复制失败';
                setTimeout(() => { btn.textContent = '复制'; }, 1000);
            });
        });
    });
}

/**
 * 为语法手册卡片绑定「打开编辑器试试」按钮
 */
function setupMdRefTryButtons() {
    document.querySelectorAll('.md-ref-try-btn').forEach(btn => {
        // 移除旧的监听器（如果有）
        if (btn._mdRefTryBound) {
            btn.removeEventListener('click', btn._mdRefTryHandler);
        }

        const handler = () => {
            const card = btn.closest('.md-ref-card');
            if (!card) return;

            const source = card.querySelector('.md-ref-source');
            const badge = card.querySelector('.md-ref-badge');
            if (!source) return;

            const rawSource = source.textContent.trim();
            const badgeText = badge ? badge.textContent.trim() : '示例';

            openMdRefTryEditor(rawSource, badgeText);
        };

        btn._mdRefTryHandler = handler;
        btn._mdRefTryBound = true;
        btn.addEventListener('click', handler);
    });
}

/**
 * 打开编辑器并预填 MD 语法示例内容
 * @param {string} source - 源码文本（可能含 HTML 实体）
 * @param {string} badgeText - 分类标签文字
 */
async function openMdRefTryEditor(source, badgeText) {
    // 解码 HTML 实体（&gt; → >, &lt; → <, &amp; → &）
    const decoded = source
        .replace(/&gt;/g, '>')
        .replace(/&lt;/g, '<')
        .replace(/&amp;/g, '&')
        .replace(/&quot;/g, '"')
        .replace(/&#39;/g, "'");

    // 切换到首页
    switchView('grid');
    // 等待编辑器完全初始化（包括 cmEditor 创建）
    await openEditor(null);
    // 设置标题和内容（此时 cmEditor 已就绪）
    els.editorNoteTitle.value = `[MD 语法] ${badgeText}`;
    setEditorContent(decoded);
    // 设为 Markdown 类型（覆盖 openEditor 内部默认的 'text'）
    els.editorFileExt.textContent = '.md';
    // 更新类型切换按钮文字和 tooltip
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = 'M';
        els.editorTypeToggle.title = '切换为纯文本格式';
    }
    // 显示「编辑/预览」切换按钮（仅 Markdown 模式显示）
    els.editorModes.style.display = '';
    // 编辑器聚焦
    if (cmEditor) cmEditor.focus();
}

/* ===== 笔记本相关函数 ===== */

/**
 * 加载笔记本列表
 */
async function loadNotebooks() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllNotebooks) {
            const notebooks = await window.go.main.App.GetAllNotebooks();
            state.notebooks = notebooks || [];
            // 一并获取笔记数，写入 state.notebooks 供选择器弹窗使用
            try {
                if (window.go.main.App.GetNotebookNoteCounts) {
                    const counts = await window.go.main.App.GetNotebookNoteCounts() || {};
                    state.notebooks.forEach(nb => {
                        nb.noteCount = counts[nb.id] || 0;
                    });
                }
            } catch (_) {}
        } else {
            console.warn('GetAllNotebooks 未绑定，使用模拟数据');
            state.notebooks = [
                { id: 1, name: '默认笔记本', sort_order: 0, noteCount: 0 },
            ];
        }
    } catch (err) {
        console.error('加载笔记本失败:', err);
        state.notebooks = [];
    }
    renderNotebookList();
}

/**
 * 渲染笔记本侧栏列表
 */
function renderNotebookList() {
    const list = els.notebookList;
    if (!list) return;

    if (state.notebooks.length === 0) {
        list.innerHTML = '<div style="padding: 12px 10px; color: var(--text-muted); font-size: 0.813rem;">暂无笔记本</div>';
        return;
    }

    // 获取笔记数量
    let noteCounts = {};
    // 从后端获取 counts（如果有）
    (async () => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNotebookNoteCounts) {
                noteCounts = await window.go.main.App.GetNotebookNoteCounts() || {};
            }
        } catch (e) {}
        renderListContent(noteCounts);
    })();
}

/**
 * 渲染笔记本列表内容（带笔记数 badge）
 */
function renderListContent(noteCounts) {
    const list = els.notebookList;
    if (!list) return;
    // 列表重新渲染，清除键盘聚焦
    clearNotebookKeyboardFocus();

    // 默认笔记本始终排在最前面
    const sorted = [...state.notebooks].sort((a, b) => {
        if (a.id === 1) return -1;
        if (b.id === 1) return 1;
        return (a.sort_order || 0) - (b.sort_order || 0);
    });

    list.innerHTML = sorted.map(nb => {
        const count = noteCounts[nb.id] || 0;
        const isActive = nb.id === state.activeNotebookId;
        return `
            <div class="notebook-item${isActive ? ' active' : ''}" data-notebook-id="${nb.id}" data-notebook-name="${escapeHtml(nb.name)}">
                <span class="notebook-name">${escapeHtml(nb.name)}</span>
                <span class="notebook-badge">${count}</span>
            </div>
        `;
    }).join('');

    // 绑定点击事件（事件委托）
    list.querySelectorAll('.notebook-item').forEach(item => {
        item.addEventListener('click', () => {
            const id = parseInt(item.dataset.notebookId);
            if (id === state.activeNotebookId) return;
            switchNotebook(id);
        });
        // 右键菜单
        item.addEventListener('contextmenu', (e) => {
            e.preventDefault();
            e.stopPropagation();
            const id = parseInt(item.dataset.notebookId);
            showNotebookContextMenu(e, id, item.dataset.notebookName);
        });
    });
}

/**
 * 切换笔记本
 */
async function switchNotebook(notebookId) {
    if (notebookId === state.activeNotebookId) return;

    // 切换笔记本时自动退出批量模式，避免选中残留
    if (state.batchMode) {
        toggleBatchMode();
    }

    state.activeNotebookId = notebookId;

    // 清除搜索内容和页码，回到笔记首页
    state.searchKeyword = '';
    state.searchSource = 'input';
    switchView('grid');
    resetPagination();

    // 重新加载笔记
    await loadNotes();
    // 刷新侧栏高亮
    renderNotebookList();
}

/**
 * 创建笔记本（显示输入弹窗）
 */
function showNewNotebookDialog() {
    // 创建模态框 DOM
    const overlay = document.createElement('div');
    overlay.className = 'new-notebook-overlay';
    overlay.innerHTML = `
        <div class="new-notebook-dialog">
            <div class="new-notebook-title">新建笔记本</div>
            <input type="text" class="new-notebook-input" id="newNotebookInput" placeholder="输入笔记本名称..." maxlength="50" autofocus>
            <div class="new-notebook-actions">
                <button class="btn btn-cancel" id="newNotebookCancelBtn">取消</button>
                <button class="btn btn-save" id="newNotebookConfirmBtn">创建</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);

    const input = overlay.querySelector('#newNotebookInput');
    const confirmBtn = overlay.querySelector('#newNotebookConfirmBtn');
    const cancelBtn = overlay.querySelector('#newNotebookCancelBtn');

    // 动画显示
    requestAnimationFrame(() => {
        overlay.classList.add('visible');
        input.focus();
    });

    /** 关闭弹窗并清理 */
    const close = () => {
        overlay.classList.remove('visible');
        setTimeout(() => overlay.remove(), 200);
    };

    /** 执行创建 */
    const doCreate = async () => {
        const name = input.value.trim();
        if (!name) {
            nm.show('请输入笔记本名称', 'warning');
            return;
        }
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateNotebook) {
                const notebook = await window.go.main.App.CreateNotebook(name);
                if (notebook) {
                    close();
                    await loadNotebooks();
                    // 自动切换到新笔记本
                    state.activeNotebookId = notebook.id;
                    resetPagination();
                    await loadNotes();
                    renderNotebookList();
                    nm.show('笔记本已创建', 'success');
                }
            } else {
                console.warn('CreateNotebook 未绑定');
                close();
            }
        } catch (err) {
            const msg = (typeof err === 'string' ? err : err?.message || '创建笔记本失败');
            console.error('创建笔记本失败:', msg);
            nm.show(msg, 'error');
        }
    };

    confirmBtn.addEventListener('click', doCreate);
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') doCreate();
        if (e.key === 'Escape') close();
    });
    cancelBtn.addEventListener('click', close);
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close();
    });
}

/**
 * 显示笔记本右键菜单
 */
function showNotebookContextMenu(event, notebookId, notebookName) {
    // 移除已有的右键菜单
    document.querySelectorAll('.notebook-context-menu').forEach(m => m.remove());

    const isDefault = notebookId === 1;
    const menu = document.createElement('div');
    menu.className = 'notebook-context-menu active';
    menu.style.left = event.clientX + 'px';
    menu.style.top = event.clientY + 'px';

    menu.innerHTML = `
        <div class="notebook-context-item${isDefault ? ' disabled' : ''}" data-action="rename">${isDefault ? '默认笔记本' : '重命名'}</div>
        <div class="notebook-context-item danger${isDefault ? ' disabled' : ''}" data-action="delete">${isDefault ? '不可删除' : '删除'}</div>
    `;
    document.body.appendChild(menu);

    // 点击其他地方关闭
    const closeMenu = (e2) => {
        if (!menu.contains(e2.target)) {
            menu.remove();
            document.removeEventListener('click', closeMenu);
        }
    };

    // 点击菜单项
    menu.addEventListener('click', async (e) => {
        const item = e.target.closest('.notebook-context-item');
        if (!item || item.classList.contains('disabled')) return;
        const action = item.dataset.action;
        document.removeEventListener('click', closeMenu);
        menu.remove();

        if (action === 'rename') {
            startInlineRename(notebookId, notebookName);
        } else if (action === 'delete') {
            showDeleteNotebookDialog(notebookId, notebookName);
        }
    });

    setTimeout(() => document.addEventListener('click', closeMenu), 0);
}

/**
 * 内联重命名笔记本
 */
function startInlineRename(notebookId, currentName) {
    const items = els.notebookList.querySelectorAll('.notebook-item');
    let targetItem = null;
    items.forEach(item => {
        if (parseInt(item.dataset.notebookId) === notebookId) {
            targetItem = item;
        }
    });
    if (!targetItem) return;

    const nameEl = targetItem.querySelector('.notebook-name');
    const originalName = currentName || nameEl.textContent;

    // 替换为输入框
    const input = document.createElement('input');
    input.type = 'text';
    input.className = 'notebook-rename-input';
    input.value = originalName;
    input.maxLength = 50;
    nameEl.replaceWith(input);
    input.focus();
    input.select();

    let renaming = false; // 防止 Enter + blur 重复执行

    /** 完成重命名 */
    const finishRename = async (save) => {
        if (renaming) return; // 已处理过，忽略重复触发
        renaming = true;

        if (!save) {
            // 取消：恢复原样
            const span = document.createElement('span');
            span.className = 'notebook-name';
            span.textContent = originalName;
            input.replaceWith(span);
            return;
        }

        const newName = input.value.trim();
        if (!newName || newName === originalName) {
            const span = document.createElement('span');
            span.className = 'notebook-name';
            span.textContent = originalName;
            input.replaceWith(span);
            return;
        }

        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.RenameNotebook) {
                await window.go.main.App.RenameNotebook(notebookId, newName);
                await loadNotebooks();
                nm.show('笔记本已重命名', 'success');
            }
        } catch (err) {
            const msg = (typeof err === 'string' ? err : err?.message || '重命名失败');
            console.error('重命名失败:', msg);
            nm.show(msg, 'error');
            const span = document.createElement('span');
            span.className = 'notebook-name';
            span.textContent = originalName;
            input.replaceWith(span);
        }
    };

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') finishRename(true);
        if (e.key === 'Escape') finishRename(false);
    });
    input.addEventListener('blur', () => finishRename(true));
}

/**
 * 显示删除笔记本确认弹窗
 */
function showDeleteNotebookDialog(notebookId, notebookName) {
    const msg = `确定要删除笔记本「${notebookName}」吗？`;
    // 从侧栏 badge 获取该笔记本下的真实笔记数（勿用 state.notes.length，它只是当前激活笔记本）
    const badgeEl = els.notebookList.querySelector(`[data-notebook-id="${notebookId}"] .notebook-badge`);
    const noteCount = parseInt(badgeEl?.textContent) || 0;
    const checkboxText = `同时将该笔记本中的 ${noteCount} 条笔记移入回收站`;

    // 隐藏"不保存"按钮(仅三选一对话框使用)
    if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';

    // 显示自定义确认对话框（带 checkbox 选项）
    els.confirmDialogMsg.textContent = msg;
    const optionArea = document.getElementById('confirmOptionArea');
    const checkbox = document.getElementById('confirmCheckbox');
    const checkboxTextEl = document.getElementById('confirmCheckboxText');
    if (optionArea && checkbox && checkboxTextEl) {
        checkbox.checked = false;
        checkboxTextEl.textContent = checkboxText;
        optionArea.style.display = 'block';
    }
    els.confirmDialog.classList.add('visible');

    const cleanup = (confirmed) => {
        els.confirmDialog.classList.remove('visible');
        if (optionArea) optionArea.style.display = 'none';
        // 恢复"不保存"按钮为 CSS 默认隐藏状态
        if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = '';
        if (!confirmed) return;
        doDeleteNotebook(notebookId, checkbox ? checkbox.checked : false);
    };

    els.confirmOkBtn.onclick = () => cleanup(true);
    els.confirmCancelBtn.onclick = () => cleanup(false);
    els.confirmDialog.onclick = (e) => {
        if (e.target === els.confirmDialog) cleanup(false);
    };
}

async function doDeleteNotebook(notebookId, deleteNotes) {
    // 在 DOM 被刷新前先捕获笔记本名称
    const notebookEl = els.notebookList.querySelector(`[data-notebook-id="${notebookId}"]`);
    const notebookName = notebookEl?.querySelector('.notebook-name')?.textContent || '';

    try {
        if (deleteNotes) {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteNotebookWithNotes) {
                await window.go.main.App.DeleteNotebookWithNotes(notebookId);
            }
        } else {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteNotebook) {
                await window.go.main.App.DeleteNotebook(notebookId);
            }
        }
        // 如果删除的是当前激活的笔记本，自动切到默认笔记本并回到笔记首页
        if (state.activeNotebookId === notebookId) {
            state.activeNotebookId = 1;
            state.searchKeyword = '';
            switchView('grid');
            resetPagination();
        }
        await loadNotebooks();
        await loadNotes();
        const suffix = deleteNotes ? '及其笔记已移入回收站' : '已删除';
        nm.show(`笔记本「${notebookName}」${suffix}`, 'success');
    } catch (err) {
        const msg = (typeof err === 'string' ? err : err?.message || '删除笔记本失败');
        console.error('删除笔记本失败:', msg);
        nm.show(msg, 'error');
    }
}

/**
 * 切换侧栏折叠/展开
 */
async function toggleSidebar() {
    const sidebar = els.notebookSidebar;
    if (!sidebar) return;
    const wasCollapsed = sidebar.classList.contains('collapsed');
    const isCollapsed = sidebar.classList.toggle('collapsed');
    // localStorage 持久化
    try {
        localStorage.setItem('jot_sidebar_collapsed', String(isCollapsed));
    } catch (e) {}
    // 从折叠→展开时刷新笔记本数据
    if (wasCollapsed && !isCollapsed) {
        await loadNotebooks();
    }
    updateSidebarMenuItem();
    updateNotebookSidebarToggleBtn();
}

/**
 * 更新菜单项文字：展开侧栏 ↔ 折叠侧栏
 */
function updateSidebarMenuItem() {
    const menuItem = els.moreMenu?.querySelector('[data-action="sidebar-toggle"]');
    if (!menuItem) return;
    const isCollapsed = els.notebookSidebar?.classList.contains('collapsed');
    const label = isCollapsed ? '展开侧栏' : '折叠侧栏';
    const showSvg = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="3" x2="9" y2="21"/></svg>';
    const hideSvg = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="15" y1="3" x2="15" y2="21"/></svg>';
    menuItem.innerHTML = (isCollapsed ? showSvg : hideSvg) + label;
    menuItem.title = `Ctrl+2 ${label}`;
}

/**
 * 更新笔记本侧栏折叠按钮图标
 */
function updateNotebookSidebarToggleBtn() {
    const btn = els.notebookSidebarToggle;
    if (!btn) return;
    const isCollapsed = els.notebookSidebar?.classList.contains('collapsed');
    const chevronLeft = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>';
    const chevronRight = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';
    btn.innerHTML = isCollapsed ? chevronLeft : chevronRight;
    btn.title = isCollapsed ? '展开侧栏' : '折叠侧栏';
}

/**
 * 恢复侧栏折叠状态（默认收起）
 */
function restoreSidebarState() {
    try {
        // 默认收起，仅当明确存储了 'false' 时才展开
        const collapsed = localStorage.getItem('jot_sidebar_collapsed') !== 'false';
        if (collapsed) {
            const sidebar = els.notebookSidebar;
            if (sidebar) sidebar.classList.add('collapsed');
        }
        updateSidebarMenuItem();
        updateNotebookSidebarToggleBtn();
    } catch (e) {}
}

/* ===== 搜索弹窗(替代原 topbar 搜索) ===== */

/**
 * 高亮搜索关键字(用于弹窗结果)
 * 与 font-search 的 highlightMatch 重名,使用单独命名避免冲突
 */
function highlightModalMatch(text, kw) {
    if (!text) return '';
    if (!kw) return escapeHtml(text);
    const escaped = escapeHtml(text);
    const re = new RegExp(kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
    return escaped.replace(re, m => `<mark>${m}</mark>`);
}

/**
 * 打开搜索弹窗
 */
function openSearchModal() {
    if (!els.searchModal) return;
    // 记录触发前的焦点元素,关闭时恢复
    state._searchModalPrevFocus = document.activeElement;
    // 锁定 body 滚动
    document.body.style.overflow = 'hidden';
    // 显示弹窗
    els.searchModal.classList.add('visible');
    // 重置状态
    state.searchModalKeyword = '';
    state.searchModalPage = 1;
    state.searchModalTotal = 0;
    state.searchModalHasMore = false;
    state.searchModalLoading = false;
    state.searchModalNotebookId = 0;
    state.searchModalTagIds = new Set();
    state.searchModalDateStart = '';
    state.searchModalDateEnd = '';
    state.searchModalSelectedIndex = -1;
    state.searchModalSortBy = 'updated_at';
    // 重置 UI
    els.searchModalInput.value = '';
    els.searchModalResults.innerHTML = '';
    els.searchModalEmpty.style.display = 'none';
    els.searchModalFooter.style.display = 'none';
    els.searchModalNotebookLabel.textContent = '全部';
    els.searchModalTagLabel.textContent = '全部';
    if (els.searchModalDateLabel) els.searchModalDateLabel.textContent = '不限';
    if (els.searchModalSortLabel) els.searchModalSortLabel.textContent = '更新时间';
    // 重置键盘提示 chip 可见度
    if (els.searchModalHints) els.searchModalHints.classList.remove('dim');
    // 重置过滤器按钮 active 状态
    updateSearchModalFilterBtnActive();
    // 重置空状态文案
    if (els.searchModalEmptyTitle) els.searchModalEmptyTitle.textContent = '开始搜索你的笔记';
    if (els.searchModalEmptyDesc) els.searchModalEmptyDesc.textContent = '输入关键字搜索标题、内容或标签';
    // 渲染过滤器下拉内容
    renderNotebookFilterDropdown();
    renderTagFilterDropdown();
    renderDateFilterDropdownSelection();
    // 弹窗动画完成后聚焦输入框
    const focusInput = () => {
        if (els.searchModalInput) {
            els.searchModalInput.focus();
        }
    };
    const contentEl = els.searchModalContent || els.searchModal?.querySelector('.search-modal-content');
    if (contentEl) {
        contentEl.addEventListener('transitionend', focusInput, { once: true });
        // 兜底：如果 transitionend 未触发（如 reduced-motion），500ms 后备
        setTimeout(focusInput, 500);
    } else {
        focusInput();
    }
}

/**
 * 关闭搜索弹窗
 */
function closeSearchModal() {
    if (!els.searchModal) return;
    // 先立即解锁 body 滚动和关闭下拉（避免卡住）
    document.body.style.overflow = '';
    closeAllFilterDropdowns();
    // 添加 closing class 触发退出动画
    els.searchModal.classList.add('closing');
    // 退出动画完成后移除 visible 和 closing
    setTimeout(() => {
        els.searchModal.classList.remove('visible');
        els.searchModal.classList.remove('closing');
    }, 150);
}

/**
 * 弹窗输入防抖处理
 */
let _searchModalInputTimer = null;
function handleSearchModalInput() {
    if (_searchModalInputTimer) clearTimeout(_searchModalInputTimer);
    _searchModalInputTimer = setTimeout(() => {
        const kw = els.searchModalInput ? els.searchModalInput.value.trim() : '';
        state.searchModalKeyword = kw;
        state.searchModalPage = 1;
        state.searchModalSelectedIndex = -1;
        searchModalLoadPage(1, false);
    }, 200);
}

/**
 * 加载弹窗搜索分页
 */
async function searchModalLoadPage(page, append) {
    if (state.searchModalLoading) return;
    state.searchModalLoading = true;
    try {
        const kw = state.searchModalKeyword;
        // 笔记本过滤(后端 SearchNotes 第 4 个参数 notebookId 支持)
        const notebookId = state.searchModalNotebookId || 0;
        let notes = [];
        let total = 0;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.SearchNotes) {
            // 后端实际返回: {Items, Total, Page, PageSize}
            const startDate = state.searchModalDateStart || '';
            const endDate = state.searchModalDateEnd || '';
            const tagIds = state.searchModalTagIds && state.searchModalTagIds.size > 0
                ? Array.from(state.searchModalTagIds)
                : [];
            const result = await window.go.main.App.SearchNotes(kw, page, pageSize, notebookId, state.searchModalSortBy, startDate, endDate, tagIds);
            notes = result?.items || [];
            total = result?.total || 0;
        } else {
            console.warn('SearchNotes 未绑定');
            notes = [];
            total = 0;
        }
        // 后端已支持标签 AND 过滤,此处不再需要客户端过滤
        state.searchModalTotal = total;
        const loaded = (page - 1) * pageSize + notes.length;
        state.searchModalHasMore = loaded < total;
        renderSearchModalResults(notes, append);
        // 底部状态
        if (page > 1 && !state.searchModalHasMore) {
            if (els.searchModalCount) els.searchModalCount.textContent = `共 ${state.searchModalTotal} 条结果`;
            if (els.searchModalFooter) els.searchModalFooter.style.display = 'block';
        } else if (notes.length > 0) {
            if (els.searchModalFooter) els.searchModalFooter.style.display = 'none';
        }
        // 空状态
        if (notes.length === 0 && page === 1) {
            if (els.searchModalEmpty) {
                els.searchModalEmpty.style.display = kw ? 'flex' : 'none';
                if (kw) {
                    if (els.searchModalEmptyTitle) els.searchModalEmptyTitle.textContent = '没有找到匹配的笔记';
                    if (els.searchModalEmptyDesc) els.searchModalEmptyDesc.textContent = `试试调整过滤器或换个关键词:「${kw}」`;
                }
            }
        } else {
            if (els.searchModalEmpty) els.searchModalEmpty.style.display = 'none';
        }
    } catch (err) {
        console.error('[searchModal] load page error:', err);
    } finally {
        state.searchModalLoading = false;
    }
}

/**
 * 渲染搜索弹窗结果列表
 */
function renderSearchModalResults(notes, append) {
    // 清理历史选中类(避免跨次渲染残留)
    if (els.searchModalResults) {
        els.searchModalResults.querySelectorAll('.search-modal-item.selected').forEach(el => el.classList.remove('selected'));
    }
    if (!append && els.searchModalResults) els.searchModalResults.innerHTML = '';
    if (!notes || notes.length === 0) return;
    const kw = state.searchModalKeyword;
    const fragment = document.createDocumentFragment();
    // 笔记本 SVG 图标模板
    const NB_SVG = '<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h12a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z"/><polyline points="14 2 14 8 20 8"/></svg>';
    notes.forEach((note, idx) => {
        const item = document.createElement('div');
        item.className = 'search-modal-item';
        // 兼容后端大小写(实际为小写,但兼容一下)
        const noteId = note.id !== undefined ? note.id : note.ID;
        item.dataset.noteId = String(noteId);
        // 存储笔记本 ID,便于打开时切换侧栏
        const notebookId = note.notebook_id !== undefined ? note.notebook_id : note.NotebookID;
        if (notebookId) item.dataset.notebookId = String(notebookId);
        // 索引基于已渲染数量(append 模式累加)
        const itemIdx = els.searchModalResults.children.length + idx;
        item.dataset.idx = String(itemIdx);
        // 所有结果无延迟入场(避免错峰动画造成视觉上的"防抖"感)
        item.style.animationDelay = '0ms';
        // 标题(高亮)
        const titleEl = document.createElement('div');
        titleEl.className = 'search-modal-item-title';
        const titleText = note.title !== undefined ? note.title : (note.Title || '');
        titleEl.innerHTML = highlightModalMatch(titleText || '(无标题)', kw);
        item.appendChild(titleEl);
        // 摘要(从内容截前 100 字)
        const content = note.content !== undefined ? note.content : (note.Content || '');
        const snippet = String(content || '').slice(0, 100).replace(/\s+/g, ' ').trim();
        const snippetEl = document.createElement('div');
        snippetEl.className = 'search-modal-item-snippet';
        snippetEl.innerHTML = highlightModalMatch(snippet, kw);
        item.appendChild(snippetEl);
        // meta(笔记本名 + 标签)
        const tags = note.tags || note.Tags || [];
        if (notebookId || (tags && tags.length)) {
            const meta = document.createElement('div');
            meta.className = 'search-modal-item-meta';
            // 笔记本名(SVG 图标 + 文字)
            if (state.notebooks && state.notebooks.length && notebookId) {
                const nb = state.notebooks.find(n => n.id === notebookId);
                if (nb) {
                    const nbSpan = document.createElement('span');
                    nbSpan.className = 'search-modal-item-notebook';
                    nbSpan.innerHTML = NB_SVG + '<span>「' + escapeHtml(nb.name || '') + '」</span>';
                    meta.appendChild(nbSpan);
                }
            }
            // 标签(最多 3 个)
            if (tags && tags.length) {
                tags.slice(0, 3).forEach(t => {
                    const tag = document.createElement('span');
                    const tagId = t.id || t.ID || 0;
                    tag.className = 'search-modal-item-tag' + (state.searchModalTagIds && state.searchModalTagIds.has(tagId) ? ' filter-active' : '');
                    tag.textContent = '#' + (t.name || t.Name || '');
                    meta.appendChild(tag);
                });
            }
            item.appendChild(meta);
        }
        // 点击打开
        item.addEventListener('click', () => {
            const id = parseInt(item.dataset.noteId, 10);
            const nbId = item.dataset.notebookId ? parseInt(item.dataset.notebookId, 10) : 0;
            _openNoteFromSearch(id, nbId);
        });
        // hover 选中
        item.addEventListener('mouseenter', () => {
            updateSelectedIndex(itemIdx);
        });
        fragment.appendChild(item);
    });
    els.searchModalResults.appendChild(fragment);
}

/**
 * 弹窗内键盘导航与关闭
 */
function handleSearchModalKeydown(e) {
    if (!els.searchModal || !els.searchModal.classList.contains('visible')) return;
    // 输入框内的输入事件由 input 监听处理,这里只处理功能键
    const items = els.searchModalResults ? els.searchModalResults.querySelectorAll('.search-modal-item') : [];
    if (e.key === 'Escape') {
        e.preventDefault();
        closeSearchModal();
        return;
    }
    if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (items.length === 0) return;
        const next = state.searchModalSelectedIndex < 0 ? 0 : (state.searchModalSelectedIndex + 1) % items.length;
        updateSelectedIndex(next);
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (items.length === 0) return;
        const prev = state.searchModalSelectedIndex <= 0 ? items.length - 1 : state.searchModalSelectedIndex - 1;
        updateSelectedIndex(prev);
    } else if (e.key === 'Enter') {
        e.preventDefault();
        const idx = state.searchModalSelectedIndex >= 0 ? state.searchModalSelectedIndex : 0;
        if (items[idx]) {
            const noteId = parseInt(items[idx].dataset.noteId, 10);
            const nbId = items[idx].dataset.notebookId ? parseInt(items[idx].dataset.notebookId, 10) : 0;
            _openNoteFromSearch(noteId, nbId);
        }
    }
}

/**
 * 更新弹窗内结果项的选中索引
 */
function updateSelectedIndex(idx) {
    if (!els.searchModalResults) return;
    const items = els.searchModalResults.querySelectorAll('.search-modal-item');
    items.forEach((el, i) => {
        el.classList.toggle('selected', i === idx);
        if (i === idx) {
            el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
    });
    state.searchModalSelectedIndex = idx;
}

/**
 * 创建一个带 ✓ 图标的过滤器下拉选项
 */
function _createFilterOption({ text, selected, onClick, dataValue }) {
    const opt = document.createElement('div');
    opt.className = 'search-modal-filter-option' + (selected ? ' selected' : '');
    if (dataValue) opt.dataset.value = dataValue;
    opt.innerHTML = '<svg class="check-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="5 12 10 17 19 8"/></svg><span>' + escapeHtml(text) + '</span>';
    if (onClick) opt.addEventListener('click', onClick);
    return opt;
}

/**
 * 从搜索弹窗打开笔记,并自动切换侧栏到该笔记所属的笔记本
 */
async function _openNoteFromSearch(noteId, notebookId) {
    closeSearchModal();
    // 切换侧栏到笔记所属的笔记本,同时刷新笔记列表
    if (notebookId && notebookId !== state.activeNotebookId) {
        state.activeNotebookId = notebookId;
        resetPagination();
        await loadNotes();
        renderNotebookList();
    }
    if (typeof openEditor === 'function') {
        openEditor(noteId, true, getNoteOpenFullscreen());
    } else {
        window.viewNote(noteId);
    }
}

/**
 * 更新过滤器按钮的 active 状态(根据 state 中的过滤值)
 */
function updateSearchModalFilterBtnActive() {
    if (els.searchModalNotebookBtn) {
        els.searchModalNotebookBtn.classList.toggle('active', state.searchModalNotebookId !== 0);
    }
    if (els.searchModalTagBtn) {
        els.searchModalTagBtn.classList.toggle('active', state.searchModalTagIds.size > 0);
    }
    if (els.searchModalDateBtn) {
        els.searchModalDateBtn.classList.toggle('active', state.searchModalDateStart !== '' && state.searchModalDateEnd !== '');
    }
    if (els.searchModalSortBtn) {
        els.searchModalSortBtn.classList.toggle('active', state.searchModalSortBy !== 'updated_at');
    }
}

/**
 * 更新标签过滤器的 label 文本(根据已选数量)
 */
function updateTagFilterLabel() {
    if (!els.searchModalTagLabel) return;
    if (state.searchModalTagIds.size === 0) {
        els.searchModalTagLabel.textContent = '全部';
    } else if (state.searchModalTagIds.size === 1) {
        const id = Array.from(state.searchModalTagIds)[0];
        const t = (state.tags || []).find(x => x.id === id);
        els.searchModalTagLabel.textContent = '#' + (t ? t.name : '');
    } else {
        els.searchModalTagLabel.textContent = `${state.searchModalTagIds.size} 个标签`;
    }
}

/**
 * 渲染日期过滤器下拉选项
 */
function renderDateFilterDropdownSelection() {
    const dd = els.searchModalDateDropdown;
    if (!dd) return;
    dd.innerHTML = '';
    const quickOptions = [
        { text: '今天', quick: 'today' },
        { text: '最近7天', quick: '7d' },
        { text: '最近30天', quick: '30d' },
        { text: '不限', quick: 'all' },
    ];
    const now = new Date();
    const y = now.getFullYear();
    const m = String(now.getMonth() + 1).padStart(2, '0');
    const d = String(now.getDate()).padStart(2, '0');
    const todayStr = y + '-' + m + '-' + d;

    quickOptions.forEach(opt => {
        // 判断当前选中的是否是此选项
        let isSelected = false;
        if (opt.quick === 'all') {
            isSelected = state.searchModalDateStart === '' && state.searchModalDateEnd === '';
        } else if (opt.quick === 'today') {
            isSelected = state.searchModalDateStart === todayStr && state.searchModalDateEnd === todayStr;
        } else if (opt.quick === '7d') {
            const past = new Date(now);
            past.setDate(past.getDate() - 6);
            const py = past.getFullYear();
            const pm = String(past.getMonth() + 1).padStart(2, '0');
            const pd = String(past.getDate()).padStart(2, '0');
            isSelected = state.searchModalDateStart === py + '-' + pm + '-' + pd && state.searchModalDateEnd === todayStr;
        } else if (opt.quick === '30d') {
            const past = new Date(now);
            past.setDate(past.getDate() - 29);
            const py = past.getFullYear();
            const pm = String(past.getMonth() + 1).padStart(2, '0');
            const pd = String(past.getDate()).padStart(2, '0');
            isSelected = state.searchModalDateStart === py + '-' + pm + '-' + pd && state.searchModalDateEnd === todayStr;
        }

        const optionEl = _createFilterOption({
            text: opt.text,
            selected: isSelected,
            dataValue: opt.quick,
            onClick: function() {
                // 计算日期范围
                if (opt.quick === 'all') {
                    state.searchModalDateStart = '';
                    state.searchModalDateEnd = '';
                } else if (opt.quick === 'today') {
                    state.searchModalDateStart = todayStr;
                    state.searchModalDateEnd = todayStr;
                } else if (opt.quick === '7d') {
                    const p = new Date(now);
                    p.setDate(p.getDate() - 6);
                    const py = p.getFullYear();
                    const pm = String(p.getMonth() + 1).padStart(2, '0');
                    const pd = String(p.getDate()).padStart(2, '0');
                    state.searchModalDateStart = py + '-' + pm + '-' + pd;
                    state.searchModalDateEnd = todayStr;
                } else if (opt.quick === '30d') {
                    const p = new Date(now);
                    p.setDate(p.getDate() - 29);
                    const py = p.getFullYear();
                    const pm = String(p.getMonth() + 1).padStart(2, '0');
                    const pd = String(p.getDate()).padStart(2, '0');
                    state.searchModalDateStart = py + '-' + pm + '-' + pd;
                    state.searchModalDateEnd = todayStr;
                }
                // 更新 label
                if (els.searchModalDateLabel) {
                    els.searchModalDateLabel.textContent = opt.text;
                }
                updateSearchModalFilterBtnActive();
                // 关闭下拉
                closeAllFilterDropdowns();
                _triggerFilterSearch();
            }
        });
        dd.appendChild(optionEl);
    });
}

function closeSearchModalDatePicker() {
    // 日历已移除，无需操作
}

/**
 * 渲染排序过滤器下拉选项
 */
function renderSortFilterDropdown() {
    const dd = els.searchModalSortDropdown;
    if (!dd) return;
    dd.innerHTML = '';
    const options = [
        { value: 'updated_at', text: '更新时间' },
        { value: 'created_at', text: '创建时间' },
        { value: 'title', text: '名称' },
    ];
    options.forEach(opt => {
        dd.appendChild(_createFilterOption({
            text: opt.text,
            selected: state.searchModalSortBy === opt.value,
            dataValue: opt.value,
            onClick: (e) => {
                e.stopPropagation();
                const sortBy = opt.value;
                if (sortBy === state.searchModalSortBy) {
                    closeAllFilterDropdowns();
                    return;
                }
                state.searchModalSortBy = sortBy;
                if (els.searchModalSortLabel) els.searchModalSortLabel.textContent = opt.text;
                updateSearchModalFilterBtnActive();
                closeAllFilterDropdowns();
                _triggerFilterSearch();
            }
        }));
    });
}

/**
 * 渲染笔记本过滤器下拉选项
 */
function renderNotebookFilterDropdown() {
    const dd = els.searchModalNotebookDropdown;
    if (!dd) return;
    dd.innerHTML = '';
    // 全部选项
    dd.appendChild(_createFilterOption({
        text: '全部',
        selected: state.searchModalNotebookId === 0,
        dataValue: 'all',
        onClick: (e) => {
            e.stopPropagation();
            state.searchModalNotebookId = 0;
            if (els.searchModalNotebookLabel) els.searchModalNotebookLabel.textContent = '全部';
            updateSearchModalFilterBtnActive();
            closeAllFilterDropdowns();
            _triggerFilterSearch();
        }
    }));
    if (!state.notebooks || state.notebooks.length === 0) return;
    state.notebooks.forEach(nb => {
        dd.appendChild(_createFilterOption({
            text: nb.name || '',
            selected: state.searchModalNotebookId === nb.id,
            onClick: (e) => {
                e.stopPropagation();
                state.searchModalNotebookId = nb.id;
                if (els.searchModalNotebookLabel) els.searchModalNotebookLabel.textContent = nb.name || '';
                updateSearchModalFilterBtnActive();
                closeAllFilterDropdowns();
                _triggerFilterSearch();
            }
        }));
    });
}

/**
 * 渲染标签过滤器下拉选项
 * 多选模式:点击不关闭下拉,通过 class 切换实现选中态(保留滚动位置,支持连续多选)
 */
function renderTagFilterDropdown() {
    const dd = els.searchModalTagDropdown;
    if (!dd) return;
    dd.innerHTML = '';
    if (!state.tags || state.tags.length === 0) {
        const empty = document.createElement('div');
        empty.className = 'search-modal-filter-option';
        empty.textContent = '(无标签)';
        empty.style.color = 'var(--text-muted)';
        empty.style.cursor = 'default';
        dd.appendChild(empty);
        return;
    }
    // "全部"选项 - 点击清除所有标签选择
    const allOpt = _createFilterOption({
        text: '全部',
        selected: state.searchModalTagIds.size === 0,
        dataValue: 'all',
        onClick: (e) => {
            e.stopPropagation();
            state.searchModalTagIds = new Set();
            dd.querySelectorAll('.search-modal-filter-option').forEach(o => o.classList.remove('selected'));
            allOpt.classList.add('selected');
            updateTagFilterLabel();
            updateSearchModalFilterBtnActive();
            closeAllFilterDropdowns();
            _triggerFilterSearch();
        }
    });
    dd.appendChild(allOpt);
    state.tags.forEach(tag => {
        const tagOpt = _createFilterOption({
            text: '#' + (tag.name || ''),
            selected: state.searchModalTagIds.has(tag.id),
            onClick: (e) => {
                e.stopPropagation();
                if (state.searchModalTagIds.has(tag.id)) {
                    state.searchModalTagIds.delete(tag.id);
                    tagOpt.classList.remove('selected');
                } else {
                    state.searchModalTagIds.add(tag.id);
                    tagOpt.classList.add('selected');
                }
                allOpt.classList.toggle('selected', state.searchModalTagIds.size === 0);
                updateTagFilterLabel();
                updateSearchModalFilterBtnActive();
                closeAllFilterDropdowns();
                _triggerFilterSearch();
            }
        });
        dd.appendChild(tagOpt);
    });
}

/**
 * 筛选变动后立即触发搜索(不走 input 防抖,避免 200ms 延迟导致键盘导航被覆盖)
 */
function _triggerFilterSearch() {
    if (_searchModalInputTimer) clearTimeout(_searchModalInputTimer);
    state.searchModalKeyword = els.searchModalInput ? els.searchModalInput.value.trim() : '';
    state.searchModalPage = 1;
    state.searchModalSelectedIndex = -1;
    searchModalLoadPage(1, false);
}
/**
 * 关闭所有弹窗内的过滤器下拉
 */
function closeAllFilterDropdowns() {
    document.querySelectorAll('.search-modal-filter.open').forEach(el => el.classList.remove('open'));
    // 关闭下拉后归还焦点到输入框,确保键盘导航(↑↓/⏎)持续可用
    if (els.searchModal && els.searchModal.classList.contains('visible') && els.searchModalInput) {
        els.searchModalInput.focus();
    }
}

/**
 * 初始化搜索弹窗相关事件绑定(在 initEventListeners 末尾调用)
 */
function initSearchModalListeners() {
    if (!els.searchModal) return;
    // 输入事件(防抖)
    if (els.searchModalInput) {
        els.searchModalInput.addEventListener('input', handleSearchModalInput);
    }
    // 弹窗内 keydown(在 capture 阶段拦截,避免与全局 handleKeyboardNavigation 冲突)
    els.searchModal.addEventListener('keydown', handleSearchModalKeydown, true);
    // 弹窗滚动加载更多
    if (els.searchModalResults) {
        els.searchModalResults.addEventListener('scroll', () => {
            if (!state.searchModalHasMore || state.searchModalLoading) return;
            const el = els.searchModalResults;
            if (el.scrollTop + el.clientHeight >= el.scrollHeight - 200) {
                state.searchModalPage += 1;
                searchModalLoadPage(state.searchModalPage, true);
            }
        });
    }
    // 点击遮罩关闭
    els.searchModal.addEventListener('click', (e) => {
        if (e.target.classList.contains('search-modal-mask') || e.target === els.searchModal) {
            closeSearchModal();
        }
    });
    // 三个过滤器按钮切换下拉
    [els.searchModalNotebookBtn, els.searchModalTagBtn, els.searchModalDateBtn, els.searchModalSortBtn].forEach((btn) => {
        if (!btn) return;
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            const filter = btn.parentElement;
            if (!filter) return;
            const wasOpen = filter.classList.contains('open');
            closeAllFilterDropdowns();
            if (!wasOpen) {
                // 每次展开前根据当前 state 重新渲染,确保"选中"状态同步
                const filterType = filter.dataset.filter;
                if (filterType === 'notebook') renderNotebookFilterDropdown();
                else if (filterType === 'tag') renderTagFilterDropdown();
                else if (filterType === 'date') renderDateFilterDropdownSelection();
                else if (filterType === 'sort') renderSortFilterDropdown();
                filter.classList.add('open');
            }
        });
    });

    // 点击弹窗其它区域关闭下拉(注:点击 mask/content 都会冒泡到这里)
    document.addEventListener('click', (e) => {
        if (!els.searchModal || !els.searchModal.classList.contains('visible')) return;
        // 点击在 .search-modal-filter 容器内不关闭
        if (e.target.closest && e.target.closest('.search-modal-filter')) return;
        closeAllFilterDropdowns();
    });
}

/* ===== 初始化 ===== */

async function init() {
    initEventListeners();
    initFontSettings();
    initThemeSettings();
    initScrollLoading();
    initScrollbarAutoHide();
    setupRefCopyButtons();
    // 窗口可见性变化时自动刷新（如从外部进程注入种子数据后切回应用）
    document.addEventListener('visibilitychange', () => {
        if (document.hidden) return;
        // 正在编辑或批量操作时不刷新，避免打断用户
        if (state.batchMode) return;
        if (els.editorPanel.style.display !== 'none') return;

        const view = state.currentView;
        if (view === 'grid') {
            resetPagination();
            loadNotes();
        } else if (view === 'trash') {
            loadTrashNotes();
        } else if (view === 'data') {
            loadDataStats();
        }
    });

    state.selectedTags = [];
    await loadThemeSetting();
    await loadFontSettings();
    await initSortSettings();
    initAISettings();
    // 快速笔记设置需在 loadNotes 之前加载，启用时先显示全屏编辑器再后台加载笔记
    await loadQuickNoteSetting();
    await loadSyntaxHighlightSetting();
    await loadNoteOpenFullscreenSetting();
    await loadCodeHighlightThemeSetting();
    // 先恢复侧栏折叠状态
    restoreSidebarState();
    // 加载笔记本列表（会设置默认选中）
    await loadNotebooks();
    // 确保 activeNotebookId 有值（默认为 1）
    if (!state.activeNotebookId && state.notebooks.length > 0) {
        state.activeNotebookId = state.notebooks[0].id;
    }
    await loadNotes();
    await loadTags();
    // 初始化预览渲染 Worker
    initPreviewWorker();
    // 初始化无边框窗口控制
    initWindowControls();
    // 注册文件拖拽导入
    initFileDrop();
    // 初始化 AI 对话页面
    initAIChat();
}

/**
 * 初始化文件拖拽导入
 *
 * _dragCounter 控制拖入/离开遮罩状态（避免子元素 dragleave 误触发），
 * Wails OnFileDrop 获取文件路径（需 main.go 中 EnableFileDrop: true），
 * 传后端 ImportFiles 统一完成 stat 检测目录、二进制检测和笔记创建。
 */
let _dragCounter = 0;
/** 预览渲染内容缓存，内容未变化时跳过重复渲染 */
let _lastPreviewContent = '';
/** 预览渲染 Worker 实例 */
let _previewWorker = null;
function initFileDrop() {
    const dropOverlay = document.getElementById('dropOverlay');
    let registered = false;
    if (registered) return;
    registered = true;

    document.addEventListener('dragenter', (e) => {
        e.preventDefault();
        if (!e.dataTransfer.types.includes('Files')) return;
        _dragCounter++;
        if (_dragCounter === 1 && dropOverlay) {
            dropOverlay.classList.add('active');
        }
    });

    document.addEventListener('dragover', (e) => {
        e.preventDefault();
    });

    document.addEventListener('dragleave', (e) => {
        e.preventDefault();
        _dragCounter--;
        if (_dragCounter <= 0) {
            _dragCounter = 0;
            if (dropOverlay) dropOverlay.classList.remove('active');
        }
    });

    // HTML5 drop 仅重置遮罩，不处理文件（由 OnFileDrop 接手）
    document.addEventListener('drop', (e) => {
        e.preventDefault();
        _dragCounter = 0;
        if (dropOverlay) dropOverlay.classList.remove('active');
    });

    // Wails OnFileDrop：OS 级拦截，直接返回文件路径
    // 回调签名：(x, y, paths) — x/y 为释放坐标，paths 为文件路径数组
    if (window.runtime && window.runtime.OnFileDrop) {
        console.log('[拖拽] 注册 OnFileDrop 回调 (useDropTarget=false)');
        window.runtime.OnFileDrop(async (x, y, paths) => {
            console.log('[拖拽] OnFileDrop 触发, paths:', paths);
            // 确保遮罩已隐藏
            _dragCounter = 0;
            if (dropOverlay) dropOverlay.classList.remove('active');
            if (!paths || paths.length === 0) return;

            // 调用后端 ImportFiles 统一处理（stat 检测目录 + 二进制检测 + 创建笔记到当前笔记本）
            handleFileDropPaths(paths, state.activeNotebookId);
        }, false);
    }
}

/**
 * 处理拖拽文件导入（Wails OnFileDrop → 后端 ImportFiles）
 */
/**
 * 在卡片网格中闪烁指定笔记卡片（红色边框动画）
 * @param {number[]} noteIds - 要闪烁的笔记 ID 数组
 */
function flashNoteCards(noteIds) {
    if (!noteIds || noteIds.length === 0) return;
    // 两次 requestAnimationFrame 确保 DOM 已渲染完毕
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            noteIds.forEach((id, index) => {
                const card = els.cardGrid.querySelector(`.note-card[data-id="${id}"]`);
                if (card) {
                    card.style.animation = `cardFlash 3s ease-out forwards`;
                    card.style.animationDelay = `${index * 150}ms`;
                    card.addEventListener('animationend', function handler() {
                        card.removeEventListener('animationend', handler);
                        card.classList.add('flash-done');
                        card.style.animation = '';
                    });
                }
            });
        });
    });
}

async function handleFileDropPaths(paths, notebookId) {
    if (!paths || paths.length === 0) return;

    if (!window.go || !window.go.main || !window.go.main.App || !window.go.main.App.ImportFiles) {
        nm.show('文件导入功能暂不可用', 'error');
        return;
    }

    try {
        const results = await window.go.main.App.ImportFiles(paths, notebookId);
        if (!results || results.length === 0) return;

        let successCount = 0;
        let failCount = 0;
        const importedNoteIds = [];

        for (const result of results) {
            const label = result.path ? result.path.split(/[/\\]/).pop() || '文件' : '文件';
            if (result.error && result.error.includes('文件夹')) {
                // 后端 stat 检测到目录
                failCount++;
                nm.show(`${label} ${result.error}`, 'warning');
            } else if (result.success) {
                successCount++;
                importedNoteIds.push(result.note_id);
            } else {
                failCount++;
                nm.show(`"${label}" ${result.error || '导入失败'}`, 'warning');
            }
        }

        if (successCount > 0) {
            const msg = failCount > 0
                ? `${successCount} 个文件导入成功，${failCount} 个失败`
                : `${successCount} 个文件导入成功`;
            nm.show(msg, 'success');
        }

        await loadNotes();
        await loadNotebooks();
        // 不再打开编辑器，改为红色边框闪烁标记导入的笔记
        flashNoteCards(importedNoteIds);
    } catch (err) {
        console.error('批量导入失败:', err);
        nm.show('文件导入失败：' + (err.message || '未知错误'), 'error');
    }
}

/**
 * 初始化无边框窗口控制按钮事件
 */
function initWindowControls() {
    const minimizeBtn = document.getElementById('windowMinimizeBtn');
    const maximizeBtn = document.getElementById('windowMaximizeBtn');
    const closeBtn = document.getElementById('windowCloseBtn');

    if (minimizeBtn) {
        minimizeBtn.addEventListener('click', () => WindowMinimise());
    }

    if (maximizeBtn) {
        // 初始化窗口最大化状态标记
        maximizeBtn.dataset.maximized = 'false';

        maximizeBtn.addEventListener('click', async () => {
            await WindowToggleMaximise();
            // 使用 data-* 属性追踪状态，完全避免 async 竞态
            const isMax = maximizeBtn.dataset.maximized === 'true';
            const nextMax = !isMax;
            maximizeBtn.dataset.maximized = nextMax ? 'true' : 'false';
            maximizeBtn.innerHTML = nextMax ? SVGS.windowRestore : SVGS.windowMaximize;
            maximizeBtn.title = nextMax ? '还原' : '最大化';
        });
    }

    if (closeBtn) {
        closeBtn.addEventListener('click', async () => {
            await handleAppExit();
        });
    }

    // 双击 topbar 空白区域最大化/还原
    const topbar = document.getElementById('topbar');
    if (topbar && maximizeBtn) {
        topbar.addEventListener('dblclick', async (e) => {
            // 如果双击的是按钮，不触发
            if (e.target.closest('.topbar-btn')) return;
            await WindowToggleMaximise();
            // 使用 data-* 属性追踪状态，完全避免 async 竞态
            const isMax = maximizeBtn.dataset.maximized === 'true';
            const nextMax = !isMax;
            maximizeBtn.dataset.maximized = nextMax ? 'true' : 'false';
            maximizeBtn.innerHTML = nextMax ? SVGS.windowRestore : SVGS.windowMaximize;
            maximizeBtn.title = nextMax ? '还原' : '最大化';
        });
    }

    // 监听窗口最大化状态变化事件
    EventsOn('wails:window:maximise', () => {
        if (maximizeBtn) updateMaximizeButtonIcon(maximizeBtn, true);
    });
    EventsOn('wails:window:unmaximise', () => {
        if (maximizeBtn) updateMaximizeButtonIcon(maximizeBtn, false);
    });
}

/**
 * 更新最大化按钮图标
 */
function updateMaximizeButtonIcon(btn, isMaximized) {
    if (typeof isMaximized !== 'boolean') {
        // 异步获取当前状态（兜底，不应走此路径）
        WindowIsMaximised().then(maximised => {
            btn.dataset.maximized = maximised ? 'true' : 'false';
            btn.innerHTML = maximised ? SVGS.windowRestore : SVGS.windowMaximize;
            btn.title = maximised ? '还原' : '最大化';
        }).catch(() => {
            // 如果获取失败，切换图标
            const nextMax = btn.dataset.maximized !== 'true';
            btn.dataset.maximized = nextMax ? 'true' : 'false';
            btn.innerHTML = nextMax ? SVGS.windowRestore : SVGS.windowMaximize;
            btn.title = nextMax ? '还原' : '最大化';
        });
    } else {
        btn.dataset.maximized = isMaximized ? 'true' : 'false';
        btn.innerHTML = isMaximized ? SVGS.windowRestore : SVGS.windowMaximize;
        btn.title = isMaximized ? '还原' : '最大化';
    }
}

/**
 * 加载快速笔记设置，启用时自动打开全屏编辑器
 */
async function loadQuickNoteSetting() {
    try {
        let enabled = false;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
            const val = await window.go.main.App.GetSetting('quick_note_enabled');
            enabled = val === 'true';
        } else {
            enabled = localStorage.getItem('quick_note_enabled') === 'true';
        }
        els.quickNoteToggle.checked = enabled;
        if (enabled) {
            openEditor(null, false, true);
        }
    } catch (err) {
        console.error('加载快速笔记设置失败:', err);
    }
}

/**
 * 加载 CM6 语法高亮设置
 */
async function loadSyntaxHighlightSetting() {
    try {
        let enabled = true;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
            const val = await window.go.main.App.GetSetting('cm_syntax_highlight');
            enabled = val !== 'false'; // 默认启用
        } else {
            const local = localStorage.getItem('cm_syntax_highlight');
            enabled = local !== 'false';
        }
        els.mdHighlightToggle.checked = enabled;
    } catch (err) {
        console.error('加载语法高亮设置失败:', err);
    }
}

/**
 * 加载笔记全屏打开设置
 */
window.loadNoteOpenFullscreenSetting = async function () {
    try {
        let enabled = false;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
            const val = await window.go.main.App.GetSetting('note_open_fullscreen');
            enabled = val === 'true';
        } else {
            enabled = localStorage.getItem('note_open_fullscreen') === 'true';
        }
        els.noteOpenFullscreenToggle.checked = enabled;
    } catch (_) {}
};

/**
 * 检查是否应以全屏模式打开笔记
 */
function getNoteOpenFullscreen() {
    return els.noteOpenFullscreenToggle?.checked || false;
}

/**
 * 加载代码高亮主题设置
 */
async function loadCodeHighlightThemeSetting() {
    try {
        let theme = 'monokai-dimmed';
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
            const val = await window.go.main.App.GetSetting('code_highlight_theme');
            if (val) theme = val;
        } else {
            theme = localStorage.getItem('code_highlight_theme') || 'monokai-dimmed';
        }
        codeHighlightTheme = theme;
        // 同步 UI 分段控件状态
        applyCodeHighlightThemeUI(theme);
    } catch (err) {
        console.error('加载代码高亮主题设置失败:', err);
        codeHighlightTheme = 'monokai-dimmed';
    }
}

/**
 * 同步代码高亮主题分段控件 UI 状态
 * @param {string} themeName
 */
function applyCodeHighlightThemeUI(themeName) {
    const label = document.getElementById('codeHighlightThemeLabel');
    if (!label) return;
    const displayLabel = codeHighlightThemeLabels[themeName] || themeName;
    label.textContent = displayLabel;
    // 同步下拉菜单选中态
    const dropdown = document.getElementById('codeHighlightThemeDropdown');
    if (dropdown) {
        dropdown.querySelectorAll('.theme-select-item').forEach(item => {
            item.classList.toggle('active', item.dataset.themeValue === themeName);
        });
    }
}

/**
 * 应用代码高亮主题（若编辑器已打开则销毁重创建）
 * @param {string} themeName
 */
function applyCodeHighlightTheme(themeName) {
    codeHighlightTheme = themeName;
    // 若编辑器已打开，销毁重创建
    if (cmEditor) {
        const container = els.editorNoteContent;
        const content = cmEditor.state.doc.toString();
        const isReadOnly = !(els.notePlaceholder && els.notePlaceholder.textContent === '');
        cmEditor.destroy();
        cmEditor = null;
        // 从设置中获取当前的 useSyntaxHighlight
        const useSyntaxHighlight = els.mdHighlightToggle.checked;
        initCodeMirror(container, content, isReadOnly, useSyntaxHighlight, els.editorFileExt.textContent, themeName);
    }
    // 同步更新设置页代码预览
    if (_codePreviewInited) {
        const container = document.getElementById('codePreview');
        if (container) {
            buildCodePreview(container, themeName);
        }
    }
}

/**
 * 保存代码高亮主题设置
 * @param {string} themeName
 */
async function saveCodeHighlightThemeSetting(themeName) {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
        try {
            await window.go.main.App.SetSetting('code_highlight_theme', themeName);
        } catch (err) {
            console.error('保存代码高亮主题设置失败:', err);
            localStorage.setItem('code_highlight_theme', themeName);
        }
    } else {
        localStorage.setItem('code_highlight_theme', themeName);
    }
    nm.show('代码高亮主题已保存', 'success');
}

let _codeHighlightThemeInited = false;

/**
 * 初始化代码高亮主题设置（绑定下拉菜单事件）
 * 只执行一次，避免事件监听器累加。
 */
function initCodeHighlightThemeSettings() {
    if (_codeHighlightThemeInited) return;
    _codeHighlightThemeInited = true;

    const trigger = document.getElementById('codeHighlightThemeTrigger');
    const dropdown = document.getElementById('codeHighlightThemeDropdown');
    const items = dropdown?.querySelectorAll('.theme-select-item');
    if (!trigger || !dropdown || !items) return;

    // 点击触发按钮切换下拉菜单
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
    });

    // 点击选项切换主题
    items.forEach(item => {
        item.addEventListener('click', async () => {
            const theme = item.dataset.themeValue;
            if (!theme) return;
            dropdown.classList.remove('open');
            trigger.classList.remove('open');
            applyCodeHighlightThemeUI(theme);
            applyCodeHighlightTheme(theme);
            await saveCodeHighlightThemeSetting(theme);
        });
    });

    // 点击外部关闭下拉菜单
    document.addEventListener('click', () => {
        dropdown.classList.remove('open');
        trigger.classList.remove('open');
    });
}

let _codePreviewInited = false;
let _codePreviewEditor = null;

function initCodePreview() {
    if (_codePreviewInited) return;
    _codePreviewInited = true;

    const container = document.getElementById('codePreview');
    if (!container) return;

    buildCodePreview(container, codeHighlightTheme);
}

function buildCodePreview(container, themeName) {
    // 销毁旧实例
    if (_codePreviewEditor) {
        _codePreviewEditor.destroy();
        _codePreviewEditor = null;
    }

    const previewCode = [
        'import { useState, useEffect } from "react";',
        '',
        'function Counter() {',
        '  const [count, setCount] = useState(0);',
        '',
        '  useEffect(() => {',
        '    // Auto-increment every second',
        '    const timer = setInterval(() => {',
        '      setCount(prev => prev + 1);',
        '    }, 1000);',
        '    return () => clearInterval(timer);',
        '  }, []);',
        '',
        '  if (count > 10) {',
        '    console.warn("Count exceeded limit!");',
        '  }',
        '',
        '  return <div>{count}</div>;',
        '}',
    ].join('\n');

    const extensions = [
        javascript(),
        EditorView.editable.of(false),
        EditorView.theme({
            '&': { height: 'auto', maxHeight: '200px', backgroundColor: 'transparent' },
            '.cm-scroller': { overflow: 'auto', fontFamily: 'Consolas, Monaco, monospace', fontSize: '12px' },
            '.cm-gutters': { display: 'none' },
            '.cm-line': { padding: '0 2px' },
            '.cm-editor': { outline: 'none' },
            '&.cm-focused': { outline: 'none' },
        }),
    ];

    const highlightExt = getHighlightExtension('.js', themeName);
    if (highlightExt.length > 0) extensions.push(...highlightExt);

    const state = EditorState.create({
        doc: previewCode,
        extensions,
    });

    _codePreviewEditor = new EditorView({
        state,
        parent: container,
    });
}

// 将内部引用暴露到 window，供 data-management.js / trash-page.js 模块使用
window.els = els;
window.nm = nm;
window.SVGS = SVGS;
window.state = state;
window.showConfirmDialog = showConfirmDialog;
window.loadNotes = loadNotes;
window.loadTags = loadTags;
window.loadNotebooks = loadNotebooks;
window.switchView = switchView;
window.openEditor = openEditor;
window.updateSidebarMenuItem = updateSidebarMenuItem;
window.undoDelete = undoDelete;
window.loadThemeSetting = loadThemeSetting;
window.loadFontSettings = loadFontSettings;
window.loadSortSettings = loadSortSettings;
window.loadPageSizeSetting = loadPageSizeSetting;
window.loadQuickNoteSetting = loadQuickNoteSetting;
window.loadSyntaxHighlightSetting = loadSyntaxHighlightSetting;
window.loadNoteOpenFullscreenSetting = loadNoteOpenFullscreenSetting;
window.loadCodeHighlightThemeSetting = loadCodeHighlightThemeSetting;
window.loadAISettings = loadAISettings;

// 应用启动
document.addEventListener('DOMContentLoaded', init);

