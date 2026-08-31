import hljs from 'highlight.js';
import { marked } from 'marked';
import alert from 'marked-alert';
import mermaid from 'mermaid';
import { EventsOn, Quit, WindowFullscreen, WindowIsFullscreen, WindowIsMaximised, WindowMinimise, WindowToggleMaximise, WindowUnfullscreen } from '../wailsjs/runtime/runtime.js';
import './css/index.css';
import { applyAIHighlightTheme } from './js/hljs-themes.js';
import { codeHighlightThemePairing, isDarkTheme, themeLabels } from './js/theme-config.js';

// CodeMirror 6 导入
import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { json } from '@codemirror/lang-json';
import { bracketMatching, foldGutter, foldKeymap, indentOnInput } from '@codemirror/language';
import { SearchQuery, closeSearchPanel, highlightSelectionMatches, openSearchPanel, searchKeymap, searchPanelOpen, setSearchQuery } from '@codemirror/search';
import { Compartment, EditorState } from '@codemirror/state';
import { EditorView, drawSelection, highlightActiveLine, highlightActiveLineGutter, highlightSpecialChars, keymap, lineNumbers, placeholder, scrollPastEnd } from '@codemirror/view';
import { codeHighlightThemeLabels, getHighlightExtension, jotTheme } from './js/cm6-syntax-highlight.js';

// 独立模块
import { SVGS, debounce, formatTime, getSummary } from './js/constants.js';
import { NotificationManager, getMockNotes, getMockTags } from './js/notification.js';
// 预设品牌徽章模块
import { createPresetBadge } from './js/preset-brand.js';

// 数据管理模块
import { backupToDir, cleanupOrphanImages, clearAISessions, clearCompletedTodos, deleteAllVectors, exportData, importData, loadDataStats, onVectorIndexCloseRequested, openDataDir, openLogDir, openVectorIndexModal, resetDatabase, restoreFromDir, vacuumDatabase } from './js/data-management.js';

// 回收站页面模块
import { loadTrashNotes } from './js/trash-page.js';
// restoreAllNotes, emptyTrash 等函数通过 window 全局暴露（供 HTML 模板 onclick 调用）

// AI 对话页面模块
import { initAIChat, onAIChatViewActivated, resetAIChatState } from './js/ai-chat.js';
import { initCalendarView } from './js/calendar.js';
// 启动器网格模块
import { initLauncher } from './js/launcher.js';
// 密码管理视图模块
import { initPasswordManager } from './js/password-manager.js';
// 编辑器操作菜单模块（通过 window 暴露 initEditorActionsMenu）
import './js/editor-actions.js';

// 暴露到 window，供 data-management.js 等模块在重置/还原后预加载 AI 聊天页面
window.onAIChatViewActivated = onAIChatViewActivated;
// 供还原备份/恢复出厂后清空前端 AI 会话缓存（旧数据已失效）
window.resetAIChatState = resetAIChatState;

// 配置 marked（breaks + gfm；代码高亮在 updatePreview 中通过 hljs 后处理实现）
marked.setOptions({
    breaks: true,
    gfm: true,
});
marked.use(alert());




/* ===== CodeMirror 6 集成 ===== */

/**
 * CodeMirror 6 编辑器实例（全局单例）
 */
let cmEditor = null;
let cmReadOnlyCompartment = null;
let _mcpImportEditor = null;

/**
 * 编辑器打开/关闭操作代际计数器：
 * 每次 openEditor / closeEditor 递增，openEditor 的异步续体据此判断自己是否已被
 * 更新的打开/关闭操作取代，避免旧笔记内容覆盖新笔记（切换笔记时的内容闪烁）。
 */
let editorOpSeq = 0;

/**
 * 预览渲染请求代际计数器：
 * 每次 updatePreview 递增并随 Worker 消息携带，主线程据此丢弃过期渲染结果，
 * 避免旧笔记的预览渲染结果晚到覆盖新笔记。
 */
let previewRenderSeq = 0;

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
function initCodeMirror(container, content = '', readOnly = false, useSyntaxHighlight = true, fileExt = '.md', themeName = 'monokai-dimmed', enableWordWrap = false) {
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
            // 搜索面板打开时 Esc 只关闭面板并阻止冒泡，避免误触发全局 ESC 关闭编辑器
            // （面板未打开时返回 false，交由后续 keymap / 全局处理，保持原有 Esc 行为）
            {
                key: 'Escape',
                scope: 'editor search-panel',
                run: (view) => {
                    if (searchPanelOpen(view.state)) {
                        closeSearchPanel(view);
                        return true;
                    }
                    return false;
                },
                stopPropagation: true
            },
            ...searchKeymap,
            ...closeBracketsKeymap,
            ...completionKeymap,
            ...foldKeymap,
            indentWithTab,

        ]),
        closeBrackets(),
        autocompletion(),
        ...(useSyntaxHighlight ? getHighlightExtension(fileExt, themeName) : []),
        ...(enableWordWrap ? [EditorView.lineWrapping] : []),
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
    window.cmEditor = null;

    cmEditor = new EditorView({
        state,
        parent: container,
    });
    window.cmEditor = cmEditor;

    // 粘贴图片上传（仅 .md 编辑模式）
    cmEditor.dom.addEventListener('paste', handlePaste);

    // 阻止拖拽文件时 CM6 默认的内容插入行为（由 OnFileDrop 统一处理）
    cmEditor.dom.addEventListener('drop', (e) => {
        if (e.dataTransfer?.files?.length > 0) {
            e.preventDefault();
        }
    });

    return cmEditor;
}

/**
 * 处理粘贴事件：检测剪贴板中的图片文件，上传并插入 Markdown 图片语法
 * @param {ClipboardEvent} e
 */
async function handlePaste(e) {
    // 仅在 .md 笔记且编辑模式下处理图片粘贴
    if (els.editorFileExt.textContent !== '.md') {
        // 非 .md 笔记粘贴图片时提示用户
        const hasImage = Array.from(e.clipboardData.files).some(f => f.type.startsWith('image/'));
        if (hasImage) {
            window.showNotification?.('图片粘贴仅支持 .md 格式笔记', 'info');
        }
        return;
    }
    if (els.editorNoteTitle.readOnly) return;

    const files = Array.from(e.clipboardData.files).filter(f => f.type.startsWith('image/'));
    if (files.length === 0) return;

    e.preventDefault();

    // 在异步操作前缓存光标位置，避免 async 期间光标变化导致插入位置错误
    let pos = cmEditor.state.selection.main.head;

    for (const file of files) {
        try {
            // 将图片文件转为 base64 字符串（Wails JSON IPC 不支持直接传 []byte）
            const base64 = await new Promise((resolve, reject) => {
                const reader = new FileReader();
                reader.onload = () => {
                    const result = reader.result; // "data:image/png;base64,iVBOR..."
                    resolve(result.split(',')[1]); // 去掉 data:...;base64, 前缀
                };
                reader.onerror = reject;
                reader.readAsDataURL(file);
            });
            const url = await window.go.main.App.SaveImage(file.name, base64);
            const markdown = `![${file.name}](${url})`;
            cmEditor.dispatch({
                changes: { from: pos, insert: markdown },
                // 光标停留在图片语法末尾，方便后续继续输入
                selection: { anchor: pos + markdown.length, head: pos + markdown.length }
            });
            pos += markdown.length; // 多图连续粘贴时，依次往后插
        } catch (err) {
            console.error('上传图片失败:', file.name, err);
        }
    }

    cmEditor.focus();
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
// 暴露给其他模块（editor-actions.js 的 AI 写作锁定输入使用）
window.setCMReadOnly = setCMReadOnly;

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
    if (els.editorActionsBtn) {
        els.editorActionsBtn.style.display = readOnly ? 'none' : '';
    }
    els.editorEditBtn.style.display = readOnly ? '' : 'none';
    els.editorViewBtn.style.display = (!readOnly && state.editingNoteId != null) ? '' : 'none';
    els.editorFileExt.classList.toggle('file-ext-readonly', readOnly);
    // 切换标签选择器只读
    renderTagSelector(readOnly);
    // 切换 CM6 只读状态
    setCMReadOnly(readOnly);
    // Markdown 笔记：自动切换预览/编辑模式
    const isMd = els.editorFileExt.textContent === '.md';
    if (isMd && readOnly) {
        // 查看模式 → Markdown 预览（走 Worker 离线程渲染）
        els.editorOverlay.dataset.mode = 'preview';
        els.editorModeBtns.forEach(btn => {
            btn.classList.toggle('active', btn.dataset.mode === 'preview');
        });
        // 从 CM6 获取最新内容走 Worker 渲染
        updatePreview();
    } else if (isMd) {
        // 编辑模式 → 切回纯文本编辑
        switchEditorMode('edit');
        _setPreviewLayout(false);
    } else {
        // 非 Markdown 笔记：隐藏 TOC
        _setPreviewLayout(false);
        _closeToc();
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
        window.cmEditor = null;
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
    currentView: 'grid',       // grid | search | settings | data | trash | todo
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
// 模拟笔记数据（后端未绑定时使用，见 loadNotes 降级分支）
let mockNotes = null;

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
    viewTodo: $('viewTodo'),
    viewCalendar: $('viewCalendar'),
    viewPasswordManager: $('viewPasswordManager'),
    todoBackBtn: $('todoBackBtn'),
    todoInput: $('todoInput'),
    todoFab: $('todoFab'),
    todoFabPanel: $('todoFabPanel'),
    todoList: $('todoList'),
    todoEmpty: $('todoEmpty'),
    todoClearCompletedBtn: $('todoClearCompletedBtn'),
    viewEditor: $('viewEditor'),
    editorPanel: $('editorPanel'),

    cardGrid: $('cardGrid'),
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
    editorActionsBtn: $('editorActionsBtn'),
    editorActionsMenu: $('editorActionsMenu'),
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
    mdHighlightToggle: $('mdHighlightToggle'),
    noteOpenFullscreenToggle: $('noteOpenFullscreenToggle'),
    editorWordWrapToggle: $('editorWordWrapToggle'),
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
    pageSizeSettingDesc: $('pageSizeSettingDesc'),
    // 排序分段控件
    sortControl: $('sortControl'),
    sortIndicator: $('sortIndicator'),
    fontSizeSlider: $('fontSizeSlider'),
    fontSizeValue: $('fontSizeValue'),
    fontPreviewText: $('fontPreviewText'),
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
    openLogDirBtn: $('openLogDirBtn'),
    logLevelControl: $('logLevelControl'),
    logLevelIndicator: $('logLevelIndicator'),
    // 设置页侧边栏
    settingsNav: document.querySelector('.settings-nav'),
    settingsPanels: document.querySelector('.settings-panels'),
    clearAISessionsBtn: $('clearAISessionsBtn'),
    clearCompletedTodosBtn: $('clearCompletedTodosBtn'),
    cleanupOrphanImagesBtn: $('cleanupOrphanImagesBtn'),
    vectorIndexBtn: $('vectorIndexBtn'),
    deleteVectorsBtn: $('deleteVectorsBtn'),
    dataContent: $('dataContent'),
    dataNav: document.querySelector('.data-nav'),
    dataPanels: document.querySelector('.data-panels'),
    dataLetter: $('dataLetter'),
    letterDate: $('letterDate'),
    letterBody: $('letterBody'),
    letterFooter: $('letterFooter'),

    // 备份还原
    backupBtn: $('backupBtn'),
    restoreBtn: $('restoreBtn'),
    backupInfo: $('backupInfo'),
    backupStatusText: $('backupStatusText'),

    // 字数统计
    editorWordCount: $('editorWordCount'),
    editorFileExt: $('editorFileExt'),
    editorEditTime: $('editorEditTime'),
    editorPanes: document.querySelector('.editor-panes'),
    tocSidebar: $('tocSidebar'),
    tocBody: $('tocBody'),
    tocToggleBtn: $('tocToggleBtn'),

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
    fabAI: $('fabAI'),
    fabBatch: $('fabBatch'),
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

    // AI 配置 - 对话连接
    aiBaseURL: $('aiBaseURL'),
    aiAPIKey: $('aiAPIKey'),
    aiModelTrigger: $('aiModelTrigger'),
    aiModelDropdown: $('aiModelDropdown'),
    aiModelLabel: $('aiModelLabel'),
    aiTestURLBtn: $('aiTestURLBtn'),
    aiAPIKeyToggle: $('aiAPIKeyToggle'),
    aiFetchModelsBtn: $('aiFetchModelsBtn'),
    aiSettingModelSearch: $('aiSettingModelSearch'),

    // AI 配置 - 向量嵌入连接
    aiEmbedBaseURL: $('aiEmbedBaseURL'),
    aiEmbedAPIKey: $('aiEmbedAPIKey'),
    aiEmbedModelTrigger: $('aiEmbedModelTrigger'),
    aiEmbedModelDropdown: $('aiEmbedModelDropdown'),
    aiEmbedModelLabel: $('aiEmbedModelLabel'),
    aiEmbedTestURLBtn: $('aiEmbedTestURLBtn'),
    aiEmbedAPIKeyToggle: $('aiEmbedAPIKeyToggle'),
    aiEmbedFetchModelsBtn: $('aiEmbedFetchModelsBtn'),
    aiEmbedModelSearch: $('aiEmbedModelSearch'),
    aiEmbedPresetTrigger: $('aiEmbedPresetTrigger'),
    aiEmbedPresetDropdown: $('aiEmbedPresetDropdown'),
    aiEmbedPresetLabel: $('aiEmbedPresetLabel'),
    presetModalTestBtn: $('presetModalTestBtn'),
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
        todo: els.viewTodo,
        'calendar': els.viewCalendar,
        'password-manager': els.viewPasswordManager,
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

    // 悬浮操作按钮仅在网格视图显示（带过渡动画）
    els.fabGroup.classList.toggle('fab-hidden', view !== 'grid');
    // 笔记本侧栏折叠按钮仅在网格视图显示
    els.notebookSidebarToggle.style.display = view === 'grid' ? '' : 'none';
    // 待办 FAB + 输入面板仅在待办视图显示（带过渡动画）
    if (els.todoFab) els.todoFab.classList.toggle('fab-hidden', view !== 'todo');
    if (els.todoFabPanel) {
        if (view !== 'todo') {
            // 离开待办视图时关闭面板，避免切回时面板意外打开
            els.todoFabPanel.classList.remove('open');
            els.todoFab?.classList.remove('open');
        }
        els.todoFabPanel.classList.toggle('fab-hidden', view !== 'todo');
    }

    _viewAnimating = true;

    /**
     * 执行目标视图的进入动画
     */
    const showTargetView = () => {
        // 清除可能残留的内联 display 样式
        targetView.style.display = '';
        // 添加 active 类，通过 CSS 规则 .view.active 显示
        targetView.classList.add('active');

        // 切换到首页时重置滚动位置到顶部
        if (view === 'grid' && els.mainContent) {
            els.mainContent.scrollTop = 0;
        }

        // 加载对应视图的数据（异步，在动画期间并行加载）
        switch (view) {
            case 'settings':
                loadSettings();
                buildCodeHighlightThemeDropdown();
                initCodeHighlightThemeSettings();
                initCodePreview();
                initSettingsSidebarNav();
                loadTags();
                // 每次进入设置页 Key 输入框默认隐藏（对话 + 嵌入）
                resetApiKeyVisibility(els.aiAPIKey, els.aiAPIKeyToggle);
                resetApiKeyVisibility(els.aiEmbedAPIKey, els.aiEmbedAPIKeyToggle);
                // 每次进入设置页默认回到"外观"面板
                switchSettingsTab('appearance');
                break;
            case 'data':
                initDataNav();
                loadDataStats();
                // 每次进入数据管理页默认回到"概览"面板
                switchDataTab('overview');
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
            case 'calendar':
                if (typeof window.refreshCalendarView === 'function') {
                    window.refreshCalendarView();
                }
                break;
            case 'password-manager':
                if (typeof window.refreshPasswordManagerView === 'function') {
                    window.refreshPasswordManagerView();
                }
                break;
            case 'todo':
                _todoFilter = 'active';
                window._todoFilter = _todoFilter;
                document.querySelectorAll('.todo-filter-btn').forEach(btn => btn.classList.remove('active'));
                const activeFilterBtn = document.querySelector('.todo-filter-btn[data-filter="active"]');
                if (activeFilterBtn) activeFilterBtn.classList.add('active');
                loadTodos();
                // 切换到待办页时自动弹出输入面板
                setTimeout(() => openTodoInputPanel(), 100);
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
            // 回收站视图跳过 view-enter 淡入，避免与异步数据加载产生闪烁
            if (view !== 'trash') {
                targetView.classList.add('view-enter');
                targetView.addEventListener('animationend', function onEnterEnd() {
                    targetView.removeEventListener('animationend', onEnterEnd);
                    targetView.classList.remove('view-enter');
                    _viewAnimating = false;
                }, { once: true });
            } else {
                _viewAnimating = false;
            }
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
    // 加载前是否已有卡片：首次加载（无卡片）保留全量入场动画；刷新（有卡片）原地替换避免闪烁
    const hadCards = state.notes.length > 0;
    // 隐藏空状态；不再清空/隐藏卡片网格：已有卡片保持可见，数据到达后整体替换，避免"消失→重来"闪烁
    if (els.emptyNotes) els.emptyNotes.style.display = 'none';

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
    // 首次加载（无旧卡片）保留全量入场动画；刷新时已有卡片保持可见，无动画原地替换避免"从 opacity:0 重放"闪感
    renderCardGrid(hadCards ? 'none' : undefined);

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
    // 保存期间用户可能已切换到其他笔记：仅当仍处于"新建"状态时才关闭编辑器
    if (state.editingNoteId === null) {
        closeEditor();
    }
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
    // 保存前捕获当前编辑的笔记：保存完成时若用户已切换到其他笔记则不关闭编辑器
    const editingIdAtStart = state.editingNoteId;

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
    // 保存期间用户可能已切换到其他笔记：仅当仍编辑本笔记时才关闭编辑器
    if (state.editingNoteId === editingIdAtStart) {
        closeEditor();
    }
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
 * 创建笔记副本：基于原笔记创建一篇新笔记（正文/后缀/所属笔记本/标签复制，
 * 标题自动生成"原标题 副本"，同名冲突递增序号；置顶状态不复制）。
 */
window.duplicateNote = async function (id) {
    const note = state.notes.find((n) => n.id === id);
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DuplicateNote) {
            const dup = await window.go.main.App.DuplicateNote(id);
            if (!dup || !dup.id) throw new Error('返回的副本笔记无效');
            nm.show(`已创建副本「${dup.title}」`, 'success');
        } else {
            console.warn('DuplicateNote 未绑定，模拟创建副本');
            if (!note) return;
            state.notes.unshift({
                id: Date.now(),
                title: (note.title || '未命名') + ' 副本',
                content: note.content || '',
                file_ext: note.file_ext || 'md',
                notebook_id: note.notebook_id,
                pinned: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                tags: note.tags ? note.tags.slice() : [],
            });
            nm.show('已创建副本', 'success');
        }
    } catch (err) {
        console.error('创建副本失败:', err);
        nm.show('创建副本失败', 'error');
        return;
    }
    await loadNotes();
};

/**
 * 复制笔记正文到剪贴板
 * 列表查询仅返回截断的前 200 字符，需按 id 获取完整正文再复制（仅复制正文，不含标题）
 */
async function copyNote(id) {
    const note = state.notes.find((n) => n.id === id);
    if (!note) return;
    let content = note.content || '';
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNoteContent) {
            const fullContent = await window.go.main.App.GetNoteContent(id);
            if (fullContent != null) content = fullContent;
        }
    } catch (err) {
        console.error('获取完整笔记内容失败，降级使用列表截断内容:', err);
    }
    try {
        await navigator.clipboard.writeText(content);
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
function showConfirmDialog(msg, okText = '确定', cancelText = '取消') {
    return new Promise((resolve) => {
        // 确保第三方按钮在普通确认框中隐藏
        if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
        
        els.confirmDialogMsg.textContent = msg;
        // 自定义按钮文本
        if (els.confirmOkBtn) els.confirmOkBtn.textContent = okText;
        if (els.confirmCancelBtn) els.confirmCancelBtn.textContent = cancelText;
        els.confirmDialog.classList.add('visible');
        // 让确认框获取焦点，确保 ESC 事件被确认框的 keydown 处理，而非冒泡到下层弹窗
        els.confirmDialog.setAttribute('tabindex', '-1');
        els.confirmDialog.focus();

        const cleanup = (result) => {
            els.confirmDialog.classList.remove('visible');
            // 保持第三方按钮隐藏（普通确认框用不到）
            if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
            // 等关闭动画结束（最长 200ms）后再恢复默认按钮文本，避免动画期间文字瞬变
            setTimeout(() => {
                // 若期间已有新弹窗打开，则不恢复，避免覆盖新弹窗的按钮文本
                if (els.confirmDialog.classList.contains('visible')) return;
                if (els.confirmOkBtn) els.confirmOkBtn.textContent = '确定';
                if (els.confirmCancelBtn) els.confirmCancelBtn.textContent = '取消';
            }, 260);
            resolve(result);
        };

        els.confirmOkBtn.onclick = () => cleanup(true);
        els.confirmCancelBtn.onclick = () => cleanup(false);
        els.confirmDialog.onclick = (e) => {
            if (e.target === els.confirmDialog) cleanup(false);
        };
    });
}

// 确认框 ESC 关闭（全局单次绑定，不重复）
if (els.confirmDialog && !els.confirmDialog._escBound) {
    els.confirmDialog._escBound = true;
    els.confirmDialog.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && els.confirmDialog.classList.contains('visible')) {
            e.stopPropagation();
            els.confirmCancelBtn.click();
        }
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

/* 当前选中的标签颜色（null = 未选中，创建时随机生成） */
let selectedTagColor = null;

/**
 * 初始化预设色块选择器的交互（可点击切换选中/取消）
 */
function initColorPresets() {
    const presets = document.querySelectorAll('.color-preset:not(.color-preset-custom)');
    const customBtn = document.querySelector('.color-preset-custom');
    const colorInput = els.newTagColor;
    if (!presets.length || !customBtn) return;

    // 预设色块点击 → 切换选中/取消
    presets.forEach(p => {
        p.addEventListener('click', () => {
            if (p.classList.contains('active')) {
                // 再次点击 → 取消选中
                p.classList.remove('active');
                selectedTagColor = null;
            } else {
                // 点击其他色块 → 选中
                presets.forEach(sp => sp.classList.remove('active'));
                p.classList.add('active');
                selectedTagColor = p.dataset.color;
                colorInput.value = selectedTagColor;
            }
        });
    });

    // 自定义按钮（内置 input[type=color]） → 点击即打开原生选色器
    // capture phase 拦截：已选中时阻止事件到达内部 input，实现再次点击取消
    customBtn.addEventListener('click', (e) => {
        if (customBtn.classList.contains('active')) {
            // 已选中 → 阻止事件传到内部 input（防止 picker 打开），取消选中
            e.stopPropagation();
            customBtn.classList.remove('active');
            selectedTagColor = null;
        } else {
            // 未选中 → 取消其它色块的选中，让事件自然到达 input 打开选色器
            presets.forEach(p => p.classList.remove('active'));
        }
    }, true); // capture phase

    // 原生选色器选择 → 更新选中色，给自定义按钮也加上选中态
    colorInput.addEventListener('input', () => {
        selectedTagColor = colorInput.value;
        presets.forEach(p => p.classList.remove('active'));
        customBtn.classList.add('active');
    });
}

/**
 * 生成一个随机标签颜色（HSL，保证鲜艳度适中）
 */
function getRandomTagColor() {
    const hue = Math.floor(Math.random() * 360);
    return `hsl(${hue}, 60%, 55%)`;
}

/**
 * 创建标签（颜色取自当前选中的色块，未选中则随机生成）
 * 增量追加到 DOM，不再全量重渲染
 */
async function createTag() {
    const name = els.newTagName.value.trim();
    const color = selectedTagColor || getRandomTagColor();
    if (!name) {
        nm.show('请输入标签名称', 'warning');
        return;
    }

    // 检查标签名是否已存在
    if (state.tags && state.tags.some(tag => tag.name === name)) {
        nm.show('该标签已存在', 'warning');
        els.newTagName.value = '';
        return;
    }

    let createdTag;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateTag) {
            createdTag = await window.go.main.App.CreateTag(name, color);
        } else {
            console.warn('CreateTag 未绑定');
        }
    } catch (err) {
        console.error('创建标签失败:', err);
        nm.show('创建标签失败', 'error');
        return;
    }

    if (!createdTag) {
        nm.show('创建标签失败', 'error');
        return;
    }

    els.newTagName.value = '';

    // 增量追加到 state.tags
    state.tags.push(createdTag);

    // 增量追加 DOM 元素（移除空状态 → 追加新标签）
    if (state.tags.length === 1) {
        // 首个标签：全量渲染（从空状态切到标签列表）
        renderTagList();
    } else {
        const wrapper = document.createElement('div');
        wrapper.innerHTML = createTagElementHtml(createdTag);
        const el = wrapper.firstElementChild;
        // 移除动画类（不重复播放已有标签的入场动画）
        el.style.animation = 'none';
        els.tagList.appendChild(el);
        // 下一帧添加动画类以触发入场
        requestAnimationFrame(() => {
            el.style.animation = '';
            el.classList.add('tag-just-added');
        });
    }

    renderTagSelector();
    nm.show('标签已创建', 'success');
}

/* ===== 字体设置函数 ===== */

/**
 * 加载已保存的字体设置并应用到页面
 */


/**
 * 更新字体设置的 UI 状态
 */
function updateFontSettingsUI(fontFamily, fontSize) {
    // 更新字体族显示
    els.fontFamilyDisplay.textContent = fontFamily;

    // 更新滑条和数值显示
    if (els.fontSizeSlider) {
        els.fontSizeSlider.value = fontSize;
    }
    if (els.fontSizeValue) {
        els.fontSizeValue.textContent = fontSize + 'px';
    }

    // 更新预览区
    updateFontPreview(fontFamily, fontSize);
}

/**
 * 更新字体预览区
 */
function updateFontPreview(fontFamily, fontSize) {
    if (!els.fontPreviewText) return;
    els.fontPreviewText.style.fontFamily = fontFamily + ', system-ui, -apple-system, sans-serif';
    els.fontPreviewText.style.fontSize = fontSize + 'px';
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
        document.documentElement.style.setProperty('--font-mono', `${fontFamily}, 'Consolas', 'Monaco', 'Courier New', monospace`);
        els.fontFamilyDisplay.textContent = fontFamily;
    } else {
        document.documentElement.style.removeProperty('--font-family');
        document.documentElement.style.removeProperty('--font-mono');
        els.fontFamilyDisplay.textContent = '系统默认';
    }
}

/**
 * 应用字体大小
 */
function applyFontSize(size) {
    document.documentElement.style.setProperty('--font-size-base', `${size}px`);
}



/* ===== 主题设置函数 ===== */

/**
 * 应用指定主题
 * @param {string} themeName - 'default' | 'light' | 'dark'
 */


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
 * 构建主题下拉菜单列表
 * 从 themeLabels 数据动态生成菜单项，新增主题时无需修改 HTML
 */
function buildThemeDropdown() {
    const dropdown = els.themeDropdown;
    if (!dropdown) return;
    dropdown.innerHTML = '';
    const currentTheme = getCurrentTheme();
    for (const [key, label] of Object.entries(themeLabels)) {
        const item = document.createElement('div');
        item.className = 'theme-select-item' + (key === currentTheme ? ' active' : '');
        item.dataset.themeValue = key;
        item.textContent = label;
        item.addEventListener('click', async () => {
            applyTheme(key);
            localStorage.setItem('jot_theme', key);
            await saveSettings();
            nm.show('主题设置已保存', 'success');
        });
        dropdown.appendChild(item);
    }
    // 使下拉菜单可聚焦以接收键盘方向键事件
    dropdown.setAttribute('tabindex', '-1');
    let _lastKeyTime = 0;
    const KEY_DELAY = 250;
    dropdown.addEventListener('keydown', (e) => {
        const now = Date.now();
        if (now - _lastKeyTime < KEY_DELAY) {
            e.preventDefault();
            return;
        }
        const items = dropdown.querySelectorAll('.theme-select-item');
        if (items.length === 0) return;
        const currentIndex = Array.from(items).findIndex(item =>
            item.classList.contains('active')
        );
        let targetIndex;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            targetIndex = currentIndex < items.length - 1 ? currentIndex + 1 : 0;
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            targetIndex = currentIndex > 0 ? currentIndex - 1 : items.length - 1;
        } else {
            return;
        }
        _lastKeyTime = now;
        items[targetIndex].click();
        items[targetIndex].scrollIntoView({ block: 'nearest' });
    });
}

/**
 * 构建代码高亮主题下拉菜单列表
 * 从 codeHighlightThemeLabels 数据动态生成菜单项
 */
function buildCodeHighlightThemeDropdown() {
    const dropdown = document.getElementById('codeHighlightThemeDropdown');
    if (!dropdown) return;
    dropdown.innerHTML = '';
    for (const [key, label] of Object.entries(codeHighlightThemeLabels)) {
        const item = document.createElement('div');
        item.className = 'theme-select-item' + (key === codeHighlightTheme ? ' active' : '');
        item.dataset.themeValue = key;
        item.textContent = label;
        item.addEventListener('click', async () => {
            applyCodeHighlightThemeUI(key);
            applyCodeHighlightTheme(key);
            codeHighlightTheme = key;
            await saveSettings();
            nm.show('代码高亮主题已保存', 'success');
        });
        dropdown.appendChild(item);
    }
    // 使下拉菜单可聚焦以接收键盘方向键事件
    dropdown.setAttribute('tabindex', '-1');
    let _lastCodeKeyTime = 0;
    const CODE_KEY_DELAY = 250;
    dropdown.addEventListener('keydown', (e) => {
        const now = Date.now();
        if (now - _lastCodeKeyTime < CODE_KEY_DELAY) {
            e.preventDefault();
            return;
        }
        const items = dropdown.querySelectorAll('.theme-select-item');
        if (items.length === 0) return;
        const currentIndex = Array.from(items).findIndex(item =>
            item.classList.contains('active')
        );
        let targetIndex;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            targetIndex = currentIndex < items.length - 1 ? currentIndex + 1 : 0;
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            targetIndex = currentIndex > 0 ? currentIndex - 1 : items.length - 1;
        } else {
            return;
        }
        _lastCodeKeyTime = now;
        items[targetIndex].click();
        items[targetIndex].scrollIntoView({ block: 'nearest' });
    });
}

let _themeInited = false;

/**
 * 初始化主题设置下拉菜单事件（只处理触发按钮和外部点击关闭）
 * 菜单项由 buildThemeDropdown() 动态生成并绑定事件
 */
function initThemeSettings() {
    if (_themeInited) return;
    _themeInited = true;

    const trigger = els.themeTrigger;
    const dropdown = els.themeDropdown;
    if (!trigger || !dropdown) return;

    // 点击触发按钮切换下拉菜单
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        if (dropdown.children.length === 0) return;
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
        // 打开时让下拉菜单聚焦，以接收键盘事件
        if (dropdown.classList.contains('open')) {
            dropdown.focus({preventScroll: true});
        }
    });

    // 点击外部关闭下拉菜单
    document.addEventListener('click', (e) => {
        if (dropdown.classList.contains('open') &&
            !trigger.contains(e.target) &&
            !dropdown.contains(e.target)) {
            dropdown.classList.remove('open');
            trigger.classList.remove('open');
        }
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
        saveSettings();
        nm.show('字体设置已保存', 'success');
        closeFontFamilyDropdown();
    });

    // 点击外部关闭
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.font-family-select')) {
            closeFontFamilyDropdown();
        }
    });

    // 字体大小滑条
    els.fontSizeSlider.addEventListener('input', () => {
        const size = parseInt(els.fontSizeSlider.value, 10);
        applyFontSize(size);
        updateFontSettingsUI(getCurrentFontFamily(), size);
    });

    els.fontSizeSlider.addEventListener('change', () => {
        const size = parseInt(els.fontSizeSlider.value, 10);
        applyFontSize(size);
        updateFontSettingsUI(getCurrentFontFamily(), size);
        saveSettings();
        nm.show('字体设置已保存', 'success');
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
    // 绑定排序分段控件事件
    if (els.sortControl) {
        const moveIndicator = (btn) => {
            const btns = Array.from(els.sortControl.querySelectorAll('.segmented-btn'));
            const index = btns.indexOf(btn);
            if (index >= 0) {
                const cw = els.sortControl.offsetWidth;
                const segW = (cw - 8) / btns.length;
                els.sortIndicator.style.transform = `translateX(${2 + index * segW}px)`;
                els.sortIndicator.style.width = `${segW}px`;
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
                await saveSettings();
                nm.show('排序方式已保存', 'success');
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
                const segW = (cw - 8) / btns.length;
                els.pageSizeIndicator.style.transform = `translateX(${2 + index * segW}px)`;
                els.pageSizeIndicator.style.width = `${segW}px`;
            }
        };
        els.pageSizeControl.querySelectorAll('.segmented-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const size = parseInt(btn.dataset.value, 10);
                // 更新 active 状态和指示器位置
                els.pageSizeControl.querySelectorAll('.segmented-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                moveIndicator(btn);
                els.pageSizeSettingDesc.textContent = `每页显示 ${size} 条`;
                await saveSettings();
                nm.show('分页大小已保存', 'success');
                resetPagination();
                await loadNotes();
            });
        });
    }
}



// ── 全局 AI 辅助函数 ──

/**
 * 重置 API Key 输入框为隐藏状态（进入设置页时调用，对话/嵌入共用）
 * @param {HTMLInputElement} input - Key 输入框
 * @param {HTMLElement} toggleBtn - 显示/隐藏切换按钮
 */
function resetApiKeyVisibility(input, toggleBtn) {
    if (!input) return;
    input.type = 'password';
    const eye = toggleBtn?.querySelector('.toggle-eye');
    const eyeOff = toggleBtn?.querySelector('.toggle-eye-off');
    if (eye) eye.style.display = '';
    if (eyeOff) eyeOff.style.display = 'none';
}

/**
 * 渲染预设下拉列表（对话/嵌入共用）
 * @param {HTMLElement} container - 下拉列表容器
 * @param {HTMLElement} labelEl - 触发按钮上的标签元素
 * @param {Array} profiles - 预设列表（来自 App.GetProfiles）
 * @param {Object} current - 当前表单值 {base_url, api_key}
 * @param {Function} onSelect - 点击预设项的回调，参数为预设对象
 * @returns {number|null} 匹配高亮的预设 id
 */
function renderPresetList(container, labelEl, profiles, current, onSelect) {
    if (!container) return null;
    if (!profiles || profiles.length === 0) {
        container.innerHTML = '';
        setPresetTriggerLabel(labelEl, null, true);
        return null;
    }

    container.innerHTML = '';
    let matchedId = null;
    for (const p of profiles) {
        const item = document.createElement('div');
        item.className = 'theme-select-item preset-option';
        item.dataset.profileId = p.id;
        // 品牌徽章（按 API 地址域名识别，未命中回退首字符色块）
        item.appendChild(createPresetBadge(p.base_url, p.name));
        // 展示名称
        const nameSpan = document.createElement('span');
        nameSpan.textContent = p.name;
        item.appendChild(nameSpan);
        // 按当前表单值匹配高亮（不再依赖 is_active 字段做唯一选中）
        const isMatch = current &&
            (p.base_url || '') === (current.base_url || '') &&
            (p.api_key || '') === (current.api_key || '');
        if (isMatch) {
            matchedId = p.id;
            item.classList.add('active');
        }
        // 点击切换预设
        item.addEventListener('click', () => onSelect && onSelect(p));
        container.appendChild(item);
    }
    const matched = profiles.find(p => p.id === matchedId);
    setPresetTriggerLabel(labelEl, matched || null, false);
    return matchedId;
}

// 设置预设触发按钮标签：有匹配预设时前置小号品牌徽章 + 名称；否则显示提示文本
// empty 为 true 表示预设列表为空（显示"无预设配置"），否则显示"选择预设"
function setPresetTriggerLabel(labelEl, profile, empty) {
    if (!labelEl) return;
    labelEl.innerHTML = '';
    if (profile) {
        labelEl.appendChild(createPresetBadge(profile.base_url, profile.name, true));
        const nameSpan = document.createElement('span');
        nameSpan.className = 'preset-option-name';
        nameSpan.textContent = profile.name;
        labelEl.appendChild(nameSpan);
    } else {
        labelEl.textContent = empty ? '无预设配置' : '选择预设';
    }
}

/**
 * 向指定模型下拉菜单添加一个选项（对话/嵌入共用）
 * @param {HTMLElement} dropdown - 模型下拉容器
 * @param {string} model - 模型名称
 * @param {boolean} active - 是否高亮选中
 */
function addModelDropdownItemTo(dropdown, model, active) {
    if (!dropdown) return;
    const item = document.createElement('div');
    item.className = 'theme-select-item' + (active ? ' active' : '');
    item.dataset.modelValue = model;
    item.textContent = model;
    // 插入到搜索框前面，确保搜索框在底部
    const searchWrap = dropdown.querySelector('.ai-model-search-wrap');
    if (searchWrap) {
        dropdown.insertBefore(item, searchWrap);
    } else {
        dropdown.appendChild(item);
    }
}

/**
 * 绑定模型下拉菜单的通用交互（打开/选择/外部关闭/搜索过滤/Esc、Enter 处理，对话/嵌入共用）
 * @param {Object} m - { trigger, dropdown, label, searchInput, onSelect }
 */
function bindModelDropdown(m) {
    const { trigger, dropdown, label, searchInput, onSelect } = m;
    if (!trigger || !dropdown || !label) return;

    // 清空本模块模型搜索框，并恢复所有模型项文本（清除可能的高亮 innerHTML）
    const clearSearch = () => {
        if (searchInput) {
            searchInput.value = '';
        }
        dropdown.querySelectorAll('.theme-select-item').forEach(item => {
            const model = item.dataset.modelValue || item.dataset.model;
            if (model) item.textContent = model;
            item.style.display = '';
        });
    };

    // 点击触发按钮切换下拉菜单
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        const hasItems = dropdown.querySelectorAll('.theme-select-item').length > 0;
        if (!hasItems) return;
        const wasOpen = dropdown.classList.contains('open');
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
        if (wasOpen) {
            clearSearch();
        } else {
            // 打开后聚焦搜索框
            setTimeout(() => {
                const search = dropdown.querySelector('.ai-model-search');
                if (search) search.focus();
            }, 50);
        }
    });

    // 点击模型项 → 选中并回调（回调内完成保存）
    dropdown.addEventListener('click', (e) => {
        const item = e.target.closest('.theme-select-item');
        if (!item) return;
        const model = item.dataset.modelValue;
        if (!model) return;
        dropdown.querySelectorAll('.theme-select-item').forEach(i => i.classList.remove('active'));
        item.classList.add('active');
        label.textContent = model;
        dropdown.classList.remove('open');
        trigger.classList.remove('open');
        clearSearch();
        if (onSelect) onSelect(model);
    });

    // 点击外部关闭
    document.addEventListener('click', (e) => {
        if (!trigger.contains(e.target) && !dropdown.contains(e.target)) {
            dropdown.classList.remove('open');
            trigger.classList.remove('open');
            clearSearch();
        }
    });

    // 搜索过滤 + 关键字高亮
    if (searchInput) {
        searchInput.addEventListener('input', () => {
            const query = searchInput.value.trim();
            dropdown.querySelectorAll('.theme-select-item').forEach(item => {
                const model = item.dataset.modelValue;
                if (!model) return;
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
        searchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                dropdown.classList.remove('open');
                trigger.classList.remove('open');
                clearSearch();
            }
            if (e.key === 'Enter') e.preventDefault();
        });
    }
}

/**
 * 获取模型列表并渲染到指定下拉（对话/嵌入共用）
 * @param {Object} m - { url, key, dropdown, label, savedModel, onSaved }
 */
async function fetchModelsAndRender(m) {
    const models = await window.go.main.App.FetchAIModels(m.url, m.key);
    if (models && models.length > 0) {
        // 仅清除模型列表项，保留搜索框
        m.dropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
        for (const model of models) {
            addModelDropdownItemTo(m.dropdown, model, model === m.savedModel);
        }
        // 将第一个模型设为标签并保存，避免"显示了但没保存"
        m.label.textContent = models[0];
        if (m.onSaved) await m.onSaved();
        // 根据模型数量控制搜索框可见性
        const wrap = m.dropdown.querySelector('.ai-model-search-wrap');
        if (wrap) wrap.style.display = models.length > 1 ? '' : 'none';
        nm.show(`已获取 ${models.length} 个模型`, 'success');
    } else {
        nm.show('未获取到可用模型', 'warning');
    }
}

/**
 * 按钮加载状态切换（设置加载中态：禁用 + spinner）
 * @param {HTMLButtonElement} btn - 按钮元素
 * @param {boolean} loading - 是否进入加载态
 */
function setBtnLoading(btn, loading) {
    if (!btn) return;
    if (loading) {
        btn.dataset.origText = btn.textContent;
        btn.textContent = '';
        btn.classList.add('btn-loading');
        btn.disabled = true;
        if (!btn.querySelector('.btn-spinner')) {
            const spinner = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            spinner.setAttribute('class', 'btn-spinner');
            spinner.setAttribute('viewBox', '0 0 24 24');
            spinner.setAttribute('fill', 'none');
            spinner.setAttribute('aria-hidden', 'true');
            spinner.innerHTML = '<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-dasharray="31.4 31.4" opacity="0.3"/>' +
                '<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-dasharray="31.4 31.4" stroke-dashoffset="-10" opacity="0.85"/>';
            btn.appendChild(spinner);
        }
    } else {
        btn.classList.remove('btn-loading');
        btn.disabled = false;
        const spinner = btn.querySelector('.btn-spinner');
        if (spinner) spinner.remove();
        if (btn.dataset.origText) {
            btn.textContent = btn.dataset.origText;
        }
        delete btn.dataset.origText;
    }
}

/**
 * 初始化 API 连接模块（对话/嵌入共用）
 * 负责：测试连通性、Key 显隐、获取模型、模型下拉交互、
 *       URL/Key 自动保存、预设下拉展开/关闭。
 * @param {Object} m - 模块配置对象
 *   { baseURL, apiKey, apiKeyToggle, testBtn, fetchBtn,
 *     modelTrigger, modelDropdown, modelLabel, modelSearch,
 *     presetTrigger, presetDropdown,
 *     getSavedModel, onModelChange,
 *     onSettingsSaved, onBeforeOpenPreset }
 */
function initApiConnectionModule(m) {
    // 保存当前模块配置（未填完不保存），并刷新对应预设列表
    const saveModuleConfig = async () => {
        const url = m.baseURL.value.trim();
        const model = m.modelLabel.textContent;
        // URL 必填
        if (!url) return;
        // 模型未选中（占位符）时不保存模型名
        if (model === '-- 请先获取模型列表 --' || !model) return;
        await saveSettings();
        nm.show('AI 配置已保存', 'success');
        if (m.onSettingsSaved) m.onSettingsSaved();
    };

    // 测试 URL 连通性
    if (m.testBtn) {
        m.testBtn.addEventListener('click', async () => {
            const url = m.baseURL.value.trim();
            const key = m.apiKey.value.trim();
            if (!url) {
                nm.show('请先填写 API 地址', 'warning');
                return;
            }
            setBtnLoading(m.testBtn, true);
            try {
                const ok = await window.go.main.App.TestAIBaseURL(url, key);
                if (ok) {
                    nm.show('服务连接成功', 'success');
                    // 测试成功即持久化当前表单值（覆盖"改值后未失焦直接点测试"场景；失败不保存避免误存错误值）
                    await saveSettings();
                    if (m.onSettingsSaved) m.onSettingsSaved();
                } else {
                    nm.show('服务连接失败，请检查地址和 Key 是否正确', 'warning');
                }
            } catch (e) {
                nm.show('连接失败: ' + e, 'error');
            } finally {
                setBtnLoading(m.testBtn, false);
            }
        });
    }

    // API Key 显示/隐藏切换
    if (m.apiKeyToggle) {
        m.apiKeyToggle.addEventListener('click', () => {
            const eye = m.apiKeyToggle.querySelector('.toggle-eye');
            const eyeOff = m.apiKeyToggle.querySelector('.toggle-eye-off');
            if (m.apiKey.type === 'password') {
                m.apiKey.type = 'text';
                if (eye) eye.style.display = 'none';
                if (eyeOff) eyeOff.style.display = '';
            } else {
                m.apiKey.type = 'password';
                if (eye) eye.style.display = '';
                if (eyeOff) eyeOff.style.display = 'none';
            }
        });
    }

    // 获取模型列表
    if (m.fetchBtn) {
        m.fetchBtn.addEventListener('click', async () => {
            const url = m.baseURL.value.trim();
            const key = m.apiKey.value.trim();
            if (!url) {
                nm.show('请先填写 API 地址', 'warning');
                return;
            }
            setBtnLoading(m.fetchBtn, true);
            try {
                await fetchModelsAndRender({
                    url,
                    key,
                    dropdown: m.modelDropdown,
                    label: m.modelLabel,
                    savedModel: m.getSavedModel ? await m.getSavedModel() : null,
                    onSaved: m.onModelChange || saveModuleConfig,
                });
                // 同步持久化 URL/Key（覆盖"改值后未失焦直接点获取"场景；模型已由 fetchModelsAndRender 内部保存）
                await saveSettings();
                if (m.onSettingsSaved) m.onSettingsSaved();
            } catch (e) {
                nm.show('获取模型列表失败: ' + e, 'error');
            } finally {
                setBtnLoading(m.fetchBtn, false);
            }
        });
    }

    // 模型下拉交互（选择模型时自动保存）
    bindModelDropdown({
        trigger: m.modelTrigger,
        dropdown: m.modelDropdown,
        label: m.modelLabel,
        searchInput: m.modelSearch,
        onSelect: m.onModelChange || saveModuleConfig,
    });

    // ── 自动保存 ▸ URL 输入完成 ──
    if (m.baseURL) {
        m.baseURL.addEventListener('change', async () => {
            const url = m.baseURL.value.trim();
            if (url.endsWith('/')) {
                m.baseURL.classList.add('input-error');
                nm.show('API 地址不能以斜杠结尾', 'error');
                return;
            }
            if (!url) {
                nm.show('请先填写 API 地址', 'warning');
                return;
            }
            await saveSettings();
            nm.show('AI 配置已保存', 'success');
            if (m.onSettingsSaved) m.onSettingsSaved();
        });
        // 用户修正后自动移除错误样式；从斜杠错误态恢复时立即自动保存一次（无需再次失焦）
        m.baseURL.addEventListener('input', async () => {
            if (!m.baseURL.value.trim().endsWith('/')) {
                const wasError = m.baseURL.classList.contains('input-error');
                m.baseURL.classList.remove('input-error');
                if (wasError) {
                    await saveSettings();
                    nm.show('AI 配置已保存', 'success');
                    if (m.onSettingsSaved) m.onSettingsSaved();
                }
            }
        });
    }

    // ── 自动保存 ▸ Key 输入完成 ──
    if (m.apiKey) {
        m.apiKey.addEventListener('change', async () => {
            await saveSettings();
            nm.show('AI 配置已保存', 'success');
            if (m.onSettingsSaved) m.onSettingsSaved();
        });
    }

    // ── 预设下拉展开/关闭 ──
    if (m.presetTrigger && m.presetDropdown) {
        m.presetTrigger.addEventListener('click', async (e) => {
            e.stopPropagation();
            const profiles = await window.go.main.App.GetProfiles();
            if (profiles.length === 0) return;
            // 对话模块的管理列表已展开时先收起
            if (m.onBeforeOpenPreset) await m.onBeforeOpenPreset();
            m.presetTrigger.classList.toggle('open');
            m.presetDropdown.classList.toggle('open');
        });
        document.addEventListener('click', () => {
            m.presetDropdown.classList.remove('open');
            m.presetTrigger.classList.remove('open');
        });
    }
}

/**
 * 重新定位日志级别分段控件指示器（面板从 hidden→visible 时调用）
 */
function repositionLogLevelIndicator() {
    const indicator = els.logLevelIndicator;
    const seg = els.logLevelControl;
    if (!indicator || !seg) return;
    const target = seg.querySelector('.segmented-btn.active');
    if (!target) return;
    const btns = Array.from(seg.querySelectorAll('.segmented-btn'));
    if (btns.length === 0) return;
    const cw = seg.offsetWidth;
    // 面板仍未显示时跳过（如 loadSettings 执行时日志设置面板处于 display:none 状态）
    if (cw === 0) return;
    const index = btns.indexOf(target);
    if (index < 0) return;
    const segW = (cw - 8) / btns.length;
    indicator.style.transform = `translateX(${2 + index * segW}px)`;
    indicator.style.width = `${segW}px`;
}

async function initAISettings() {

    // ── 对话连接模块初始化（预设下拉、URL/Key、模型获取/选择共用同一套逻辑）──
    initApiConnectionModule({
        baseURL: els.aiBaseURL,
        apiKey: els.aiAPIKey,
        apiKeyToggle: els.aiAPIKeyToggle,
        testBtn: els.aiTestURLBtn,
        fetchBtn: els.aiFetchModelsBtn,
        modelTrigger: els.aiModelTrigger,
        modelDropdown: els.aiModelDropdown,
        modelLabel: els.aiModelLabel,
        modelSearch: els.aiSettingModelSearch,
        presetTrigger: $('presetTrigger'),
        presetDropdown: $('presetDropdown'),
        // 获取模型时用于高亮当前已保存的模型
        getSavedModel: async () => (await window.go.main.App.GetAIConfig()).model,
        // URL/Key/模型 变更保存后刷新对话预设下拉
        onSettingsSaved: () => { loadProfiles(); },
        // 展开预设下拉前先收起管理列表（仅对话模块有管理列表）
        onBeforeOpenPreset: async () => { if (presetMgrExpanded) await closePresetMgrList(); },
    });

    // ── 向量嵌入连接模块初始化（结构对应对话连接，预设切换走 SwitchProfile("embed", id)）──
    initApiConnectionModule({
        baseURL: els.aiEmbedBaseURL,
        apiKey: els.aiEmbedAPIKey,
        apiKeyToggle: els.aiEmbedAPIKeyToggle,
        testBtn: els.aiEmbedTestURLBtn,
        fetchBtn: els.aiEmbedFetchModelsBtn,
        modelTrigger: els.aiEmbedModelTrigger,
        modelDropdown: els.aiEmbedModelDropdown,
        modelLabel: els.aiEmbedModelLabel,
        modelSearch: els.aiEmbedModelSearch,
        presetTrigger: els.aiEmbedPresetTrigger,
        presetDropdown: els.aiEmbedPresetDropdown,
        // 获取模型时用于高亮当前已保存的嵌入模型
        getSavedModel: async () => (await window.go.main.App.GetAllSettings()).ai_embed_model,
        // URL/Key/模型 变更保存后刷新嵌入预设下拉
        onSettingsSaved: () => { loadProfilesEmbed(); },
        // 展开预设下拉前先收起管理列表（与对话模块一致）
        onBeforeOpenPreset: async () => { if (presetMgrExpanded) await closePresetMgrList(); },
    });

    // 深度思考切换
    const settingSearchToggle = document.getElementById('aiSettingSearchToggle');
    if (settingSearchToggle) {
        settingSearchToggle.addEventListener('click', async () => {
            const isActive = settingSearchToggle.classList.toggle('active');
            await saveSettings();
            nm.show(isActive ? '深度思考已开启' : '深度思考已关闭', isActive ? 'success' : 'info');
            // 同步工具栏 toggle
            const toolbarToggle = document.getElementById('aiChatSearchToggle');
            if (toolbarToggle) {
                toolbarToggle.classList.toggle('active', isActive);
            }
        });
    }


    // ── Agent 工具管理面板（仿「配置预设管理」行内展开：点击按钮在设置项下方展开列表） ──
    const agentToolsBtn = document.getElementById('aiAgentToolsBtn');
    const agentToolsSettingItem = document.getElementById('aiAgentToolsSettingItem');
    if (agentToolsBtn) {
        agentToolsBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (agentToolsMgrExpanded) {
                closeAgentToolsMgrList();
            } else {
                renderAgentToolsMgrList();
            }
            // 同步按钮 open 态（展开立即更新；收起由 closeAgentToolsMgrList 动画回调更新）
            if (!agentToolsMgrExpanded) {
                agentToolsBtn.classList.add('open');
                agentToolsBtn.setAttribute('aria-expanded', 'true');
            } else {
                agentToolsBtn.classList.remove('open');
                agentToolsBtn.setAttribute('aria-expanded', 'false');
            }
        });
        // 点击页面其它区域（排除按钮、设置项容器与管理面板自身）关闭面板
        document.addEventListener('click', (e) => {
            if (!agentToolsMgrExpanded) return;
            if (agentToolsSettingItem && agentToolsSettingItem.contains(e.target)) return;
            if (agentToolsMgrContainer && agentToolsMgrContainer.contains(e.target)) return;
            closeAgentToolsMgrList();
        });
        // 按 ESC 关闭面板
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && agentToolsMgrExpanded) {
                closeAgentToolsMgrList();
            }
        });
    }

    // ── 卡片召回条数保存 ──
    const cardRecallLimit = document.getElementById('aiSettingCardRecallLimit');
    if (cardRecallLimit) {
        cardRecallLimit.addEventListener('change', async (e) => {
            let val = parseInt(e.target.value);
            if (isNaN(val) || val < 1) {
                val = 5;
                e.target.value = 5;
            }
            if (val > 30) {
                val = 30;
                e.target.value = 30;
            }
            await saveSettings();
            nm.show('召回条数已保存（' + val + ' 条/次）', 'success');
        });
    }

    // ── 锁屏密码 toggle ──
    const screenLockToggle = document.getElementById('screenLockToggle');
    if (screenLockToggle) {
        screenLockToggle.addEventListener('click', async (e) => {
            e.stopPropagation();
            const wasActive = screenLockToggle.classList.contains('active');
            const pwdRow = document.getElementById('screenLockPasswordRow');

            // 关闭锁屏时先确认
            if (wasActive) {
                const confirmed = await showConfirmDialog('确定关闭锁屏密码？关闭后已设置的密码将被清除。');
                if (!confirmed) return;
            }

            const isActive = screenLockToggle.classList.toggle('active');
            if (pwdRow) {
                pwdRow.classList.toggle('collapsed', !isActive);
            }
            await saveSettings();
            // 关闭后密码已被清空，按钮重置为"设置密码"；开启时密码也是空的，同样显示"设置密码"
            const changeBtn = document.getElementById('pwdChangeBtn');
            if (changeBtn) changeBtn.textContent = '设置密码';
            nm.show(isActive ? '锁屏密码已启用' : '锁屏密码已关闭', 'info');
        });
    }

    // ── 锁屏密码修改弹窗 ──
    const pwdChangeBtn = document.getElementById('pwdChangeBtn');
    const pwdModal = document.getElementById('pwdModal');
    if (pwdChangeBtn && pwdModal) {
        // 打开弹窗
        pwdChangeBtn.addEventListener('click', () => {
            const isPasswordSet = pwdChangeBtn.textContent.includes('修改密码');
            document.getElementById('pwdOldField').style.display = isPasswordSet ? '' : 'none';
            document.getElementById('pwdOldInput').value = '';
            document.getElementById('pwdNewInput').value = '';
            document.getElementById('pwdConfirmInput').value = '';
            document.getElementById('pwdModalError').style.display = 'none';
            document.getElementById('pwdModalSaveBtn').disabled = true;
            pwdModal.style.display = 'flex';
            requestAnimationFrame(() => pwdModal.classList.add('visible'));
            setTimeout(() => {
                (isPasswordSet ? document.getElementById('pwdOldInput') : document.getElementById('pwdNewInput')).focus();
            }, 200);
        });

        // 关闭弹窗（仅移除 visible 触发退出动画，transitionend 负责隐藏）
        const closeModal = () => {
            pwdModal.classList.remove('visible');
        };

        // 动画结束后自动隐藏 DOM
        pwdModal.addEventListener('transitionend', () => {
            if (!pwdModal.classList.contains('visible')) {
                pwdModal.style.display = 'none';
            }
        });

        document.getElementById('pwdModalCloseBtn').addEventListener('click', closeModal);
        document.getElementById('pwdModalCancelBtn').addEventListener('click', closeModal);
        pwdModal.addEventListener('click', (e) => { if (e.target === pwdModal) closeModal(); });

        // 表单输入验证
        const validatePwdForm = () => {
            const isPasswordSet = pwdChangeBtn.textContent.includes('修改密码');
            const oldVal = document.getElementById('pwdOldInput').value;
            const newVal = document.getElementById('pwdNewInput').value;
            const confirmVal = document.getElementById('pwdConfirmInput').value;
            const errorEl = document.getElementById('pwdModalError');
            const saveBtn = document.getElementById('pwdModalSaveBtn');
            errorEl.style.display = 'none';
            if (isPasswordSet && !oldVal) { saveBtn.disabled = true; return; }
            if (!newVal || !confirmVal) { saveBtn.disabled = true; return; }
            if (newVal !== confirmVal) {
                saveBtn.disabled = true;
                errorEl.textContent = '两次密码输入不一致';
                errorEl.style.display = '';
                return;
            }
            // 修改密码时新密码不能与旧密码相同
            if (isPasswordSet && newVal === oldVal) {
                saveBtn.disabled = true;
                errorEl.textContent = '新密码不能与旧密码相同';
                errorEl.style.display = '';
                return;
            }
            saveBtn.disabled = false;
        };
        document.getElementById('pwdOldInput').addEventListener('input', validatePwdForm);
        document.getElementById('pwdNewInput').addEventListener('input', validatePwdForm);
        document.getElementById('pwdConfirmInput').addEventListener('input', validatePwdForm);

        // 密码可见切换（点击切换）
        document.querySelectorAll('.pwd-modal-eye').forEach(btn => {
            const input = document.getElementById(btn.dataset.target);
            if (!input) return;
            btn.addEventListener('click', () => {
                const isPassword = input.type === 'password';
                input.type = isPassword ? 'text' : 'password';
                btn.querySelector('.eye-icon').style.display = isPassword ? 'none' : '';
                btn.querySelector('.eye-off-icon').style.display = isPassword ? '' : 'none';
            });
        });

        // Enter 键提交
        const pwdInputs = [document.getElementById('pwdOldInput'), document.getElementById('pwdNewInput'), document.getElementById('pwdConfirmInput')];
        pwdInputs.forEach(input => {
            if (!input) return;
            input.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    const saveBtn = document.getElementById('pwdModalSaveBtn');
                    if (!saveBtn.disabled) saveBtn.click();
                }
            });
        });

        // 保存密码
        document.getElementById('pwdModalSaveBtn').addEventListener('click', async () => {
            const saveBtn = document.getElementById('pwdModalSaveBtn');
            saveBtn.disabled = true;
            const isPasswordSet = pwdChangeBtn.textContent.includes('修改密码');
            const oldPwd = isPasswordSet ? document.getElementById('pwdOldInput').value : '';
            const newPwd = document.getElementById('pwdNewInput').value;
            const confirmPwd = document.getElementById('pwdConfirmInput').value;
            const errorEl = document.getElementById('pwdModalError');
            if (newPwd !== confirmPwd) {
                errorEl.textContent = '两次密码输入不一致';
                errorEl.style.display = '';
                saveBtn.disabled = false;
                return;
            }
            if (!newPwd) {
                errorEl.textContent = '新密码不能为空';
                errorEl.style.display = '';
                saveBtn.disabled = false;
                return;
            }
            try {
                await window.go.main.App.SetScreenLockPassword(oldPwd, newPwd);
                closeModal();
                document.getElementById('pwdChangeBtn').textContent = '修改密码';
                nm.show('锁屏密码已保存', 'success');
            } catch (err) {
                errorEl.textContent = err.message || '保存失败，请重试';
                errorEl.style.display = '';
                saveBtn.disabled = false;
            }
        });
    }

    // ── 最大上传文件大小自动保存 ──
    const maxFileSize = document.getElementById('maxFileSize');
    if (maxFileSize) {
        maxFileSize.addEventListener('change', async () => {
            const val = parseInt(maxFileSize.value);
            if (isNaN(val) || val < 1) {
                maxFileSize.value = 1;
                nm.show('最大上传文件大小必须大于 0，已重置为 1', 'warning');
                return;
            }
            if (val > 100) {
                maxFileSize.value = 100;
                nm.show('最大上传文件大小不能超过 100，已重置为 100', 'warning');
                return;
            }
            await saveSettings();
            nm.show('最大上传文件大小已保存（' + val + ' MB）', 'success');
        });
    }

    // ── 大文件预览阈值自动保存 ──
    const largeFileThreshold = document.getElementById('aiLargeFilePreviewThreshold');
    if (largeFileThreshold) {
        largeFileThreshold.addEventListener('change', async () => {
            const val = parseInt(largeFileThreshold.value);
            if (isNaN(val) || val < 1) {
                largeFileThreshold.value = 10000;
                nm.show('大文件预览阈值必须大于 0，已重置为 10000', 'warning');
                return;
            }
            if (val > 100000) {
                largeFileThreshold.value = 100000;
                nm.show('大文件预览阈值不能超过 100000，已重置为 100000', 'warning');
                return;
            }
            await saveSettings();
            nm.show('大文件预览阈值已保存', 'success');
        });
    }

    // ── Agent 最大运行次数保存 ──
    const agentMaxIterations = document.getElementById('aiAgentMaxIterations');
    if (agentMaxIterations) {
        agentMaxIterations.addEventListener('change', async () => {
            const val = parseInt(agentMaxIterations.value);
            if (isNaN(val) || val < 1) {
                agentMaxIterations.value = 20;
                nm.show('Agent 运行上限必须大于 0，已重置为 20', 'warning');
                return;
            }
            if (val > 100) {
                agentMaxIterations.value = 100;
                nm.show('Agent 运行上限不能超过 100，已重置为 100', 'warning');
                return;
            }
            await saveSettings();
            nm.show('Agent 运行上限已保存', 'success');
        });
    }

    // ── 回收站自动清理天数保存 ──
    const retentionDaysInput = document.getElementById('trashCleanupRetentionDays');
    if (retentionDaysInput) {
        retentionDaysInput.addEventListener('change', async () => {
            let val = parseInt(retentionDaysInput.value);
            if (isNaN(val) || val < 1) {
                val = 30;
                retentionDaysInput.value = 30;
                nm.show('自动清理天数必须大于 0，已重置为 30', 'warning');
                return;
            }
            if (val > 365) {
                val = 365;
                retentionDaysInput.value = 365;
                nm.show('自动清理天数不能超过 365，已重置为 365', 'warning');
                return;
            }
            await saveSettings();
            nm.show('回收站自动清理天数已保存', 'success');
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
    // 向量嵌入连接的新增/管理按钮（预设为共享表；新增弹窗统一纯净打开不预填当前连接，管理列表按所在行插入）
    document.getElementById('aiEmbedPresetAddBtn')?.addEventListener('click', openAddProfileModal);
    document.getElementById('aiEmbedPresetMgrBtn')?.addEventListener('click', (e) => {
        if (presetMgrExpanded) {
            closePresetMgrList();
        } else {
            renderPresetMgrList(e.currentTarget.closest('.preset-select-row'));
        }
    });

    // ── 预设弹窗事件 ──
    document.getElementById('presetModalClose')?.addEventListener('click', () => closePresetModal());
    document.getElementById('presetModalCancel')?.addEventListener('click', () => closePresetModal());
    document.getElementById('presetModalSave')?.addEventListener('click', savePresetModal);
    document.getElementById('presetModalTestBtn')?.addEventListener('click', testPresetConnection);
    // 点击遮罩关闭弹窗
    document.getElementById('presetModalOverlay')?.addEventListener('click', (e) => {
        if (e.target === e.currentTarget) closePresetModal();
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
    // 用户修正预设 URL 后自动移除错误样式
    document.getElementById('presetModalURL')?.addEventListener('input', function () {
        if (!this.value.trim().endsWith('/')) {
            this.classList.remove('input-error');
        }
    });

    // 测试预设连接
    async function testPresetConnection() {
        const baseURL = document.getElementById('presetModalURL').value.trim();
        const apiKey = document.getElementById('presetModalKey').value.trim();
        if (!baseURL) {
            nm.show('请先填写 API 地址', 'warning');
            return;
        }
        setBtnLoading(els.presetModalTestBtn, true, '测试中…');
        try {
            const ok = await window.go.main.App.TestAIConnection(baseURL, apiKey);
            if (ok) {
                const presetName = document.getElementById('presetModalName').value.trim();
                nm.show(presetName ? `「${presetName}」连接成功` : '连接成功', 'success');
            } else {
                nm.show('连接失败，请检查地址和 Key 是否正确', 'warning');
            }
        } catch (e) {
            nm.show('连接失败: ' + (e.message || e), 'error');
        } finally {
            setBtnLoading(els.presetModalTestBtn, false);
        }
    }
}

// ── API 配置预设管理 ──

// 当前编辑的预设 ID（编辑模式用）
let editingProfileId = null;

// 预设弹窗打开时的表单初始值快照（用于关闭时判断是否有未保存修改）
let presetModalInitial = { name: '', url: '', key: '' };

// 管理列表是否已展开
let presetMgrExpanded = false;
let presetMgrContainer = null;

// 加载预设列表到对话模块的下拉（按当前表单值匹配高亮，不再依赖 is_active）
async function loadProfiles() {
    try {
        const profiles = await window.go.main.App.GetProfiles();
        const dropdown = document.getElementById('presetDropdown');
        const label = document.getElementById('presetLabel');
        if (!dropdown) return;

        // 按当前对话表单值（地址/Key）与预设匹配高亮
        renderPresetList(dropdown, label, profiles, {
            base_url: (els.aiBaseURL.value || '').trim(),
            api_key: (els.aiAPIKey.value || '').trim(),
        }, (p) => switchProfile('chat', p.id));
    } catch (e) {
        console.warn('加载预设失败:', e);
    }
}

// 加载预设列表到嵌入模块的下拉（按当前嵌入表单值匹配高亮）
async function loadProfilesEmbed() {
    try {
        const profiles = await window.go.main.App.GetProfiles();
        if (!els.aiEmbedPresetDropdown) return;

        // 按当前嵌入表单值（地址/Key）与预设匹配高亮
        renderPresetList(els.aiEmbedPresetDropdown, els.aiEmbedPresetLabel, profiles, {
            base_url: (els.aiEmbedBaseURL.value || '').trim(),
            api_key: (els.aiEmbedAPIKey.value || '').trim(),
        }, (p) => switchProfileEmbed(p.id));
    } catch (e) {
        console.warn('加载嵌入预设失败:', e);
    }
}

// 切换对话连接预设（target 固定为 "chat"，后续后端将配套 target 参数）
async function switchProfile(target, id, silent) {
    try {
        await window.go.main.App.SwitchProfile(target, id);
        // 刷新当前配置的输入框
        const cfg = await window.go.main.App.GetAIConfig();
        els.aiBaseURL.value = cfg.base_url || '';
        els.aiAPIKey.value = cfg.api_key || '';
        // 重置模型下拉
        els.aiModelLabel.textContent = '-- 请先获取模型列表 --';
        els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
        const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
        if (wrap) wrap.style.display = 'none';
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

// 切换向量嵌入连接预设（调用 SwitchProfile("embed", id)，切换后从整体设置回显嵌入键）
async function switchProfileEmbed(id, silent) {
    try {
        await window.go.main.App.SwitchProfile('embed', id);
        // 从整体设置读取嵌入键回显（后端尚未支持 ai_embed_* 时为空值兜底）
        const cfg = await window.go.main.App.GetAllSettings();
        els.aiEmbedBaseURL.value = cfg.ai_embed_base_url || '';
        els.aiEmbedAPIKey.value = cfg.ai_embed_api_key || '';
        // 重置模型下拉
        els.aiEmbedModelLabel.textContent = '-- 请先获取模型列表 --';
        els.aiEmbedModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
        const wrap = els.aiEmbedModelDropdown.querySelector('.ai-model-search-wrap');
        if (wrap) wrap.style.display = 'none';
        // 刷新嵌入预设下拉的选中态
        await loadProfilesEmbed();
        if (!silent) {
            nm.show('已切换到嵌入预设', 'success');
        }
    } catch (e) {
        nm.show('切换嵌入预设失败: ' + e, 'error');
    }
}

// 打开新增预设弹窗：每次打开都是纯净的空表单，不预填当前连接表单值
function openAddProfileModal() {
    editingProfileId = null;
    document.getElementById('presetModalTitle').textContent = '新增配置';
    document.getElementById('presetModalName').value = '';
    document.getElementById('presetModalURL').value = '';
    document.getElementById('presetModalURL').classList.remove('input-error');
    document.getElementById('presetModalKey').value = '';
    // 重置 Key 为隐藏状态
    var keyInput = document.getElementById('presetModalKey');
    var eye = document.querySelector('#presetModalKeyToggle .toggle-eye');
    var eyeOff = document.querySelector('#presetModalKeyToggle .toggle-eye-off');
    if (keyInput && eye && eyeOff) {
        keyInput.type = 'password';
        eye.style.display = '';
        eyeOff.style.display = 'none';
    }
    // 记录表单初始快照，用于关闭时判断是否有未保存修改
    presetModalInitial = {
        name: document.getElementById('presetModalName').value,
        url: document.getElementById('presetModalURL').value,
        key: document.getElementById('presetModalKey').value,
    };
    const overlay = document.getElementById('presetModalOverlay');
    overlay.classList.add('visible');
    document.getElementById('presetModalName').focus();
}

// 打开编辑预设弹窗
function openEditProfileModal(id, name, baseURL, apiKey) {
    editingProfileId = id;
    document.getElementById('presetModalTitle').textContent = '编辑配置';
    document.getElementById('presetModalName').value = name || '';
    document.getElementById('presetModalURL').value = baseURL || '';
    document.getElementById('presetModalURL').classList.remove('input-error');
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
    // 记录表单初始快照，用于关闭时判断是否有未保存修改
    presetModalInitial = {
        name: document.getElementById('presetModalName').value,
        url: document.getElementById('presetModalURL').value,
        key: document.getElementById('presetModalKey').value,
    };
    const overlay = document.getElementById('presetModalOverlay');
    overlay.classList.add('visible');
    document.getElementById('presetModalName').focus();
}

// 关闭预设弹窗；force 为 true 时跳过未保存修改确认（保存成功后使用）
async function closePresetModal(force = false) {
    // force 必须是字面量 true 才跳过确认（防御：避免事件对象等 truthy 值误传入跳过确认）
    if (force !== true && hasPresetModalChanges()) {
        const ok = await showConfirmDialog('有未保存的修改，确定放弃并关闭吗？');
        if (!ok) return;
    }
    document.getElementById('presetModalOverlay').classList.remove('visible');
    editingProfileId = null;
}

// 判断预设弹窗表单是否相对初始快照有修改
function hasPresetModalChanges() {
    const g = (id) => (document.getElementById(id)?.value ?? '');
    return g('presetModalName') !== presetModalInitial.name
        || g('presetModalURL') !== presetModalInitial.url
        || g('presetModalKey') !== presetModalInitial.key;
}

// 保存预设（新增或编辑）
async function savePresetModal() {
    const name = document.getElementById('presetModalName').value.trim();
    const baseURL = document.getElementById('presetModalURL').value.trim();
    const apiKey = document.getElementById('presetModalKey').value.trim();
    if (!name) { nm.show('请输入名称', 'error'); return; }
    if (!baseURL) { nm.show('请输入 API 地址', 'error'); return; }
    // 检查名称是否已存在（编辑时排除自身）
    const existingProfiles = await window.go.main.App.GetProfiles();
    const nameExists = existingProfiles.some(p => p.name === name && p.id !== editingProfileId);
    if (nameExists) {
        nm.show(`名称「${name}」已被使用，请换一个`, 'error');
        return;
    }
    if (baseURL.endsWith('/')) {
        const urlInput = document.getElementById('presetModalURL');
        urlInput.classList.add('input-error');
        nm.show('API 地址不能以斜杠结尾', 'error');
        return;
    }
    try {
        let profile = null;
        if (editingProfileId) {
            await window.go.main.App.UpdateProfile(editingProfileId, name, baseURL, apiKey);
            nm.show('配置已更新', 'success');
        } else {
            profile = await window.go.main.App.CreateProfile(name, baseURL, apiKey);
            nm.show('配置已新增', 'success');
        }
        const wasEditingId = editingProfileId; // 保存编辑状态，closePresetModal 会清空
        closePresetModal(true); // 保存成功后跳过未保存修改确认
        await loadProfiles();
        await loadProfilesEmbed(); // 预设共享，两个模块的下拉同步刷新
        if (wasEditingId && presetMgrExpanded && presetMgrContainer && presetMgrContainer.parentNode) {
            // 编辑模式：精准替换对应行，避免全量重渲染，无需入场动画
            const updatedProfile = { id: wasEditingId, name, base_url: baseURL, api_key: apiKey };
            const oldRow = presetMgrContainer.querySelector(`[data-profile-id="${wasEditingId}"]`);
            if (oldRow) {
                const newRow = createPresetRowElement(updatedProfile);
                newRow.classList.remove('preset-row-enter'); // 编辑模式无需入场动画
                oldRow.replaceWith(newRow);
            }
        } else if (!wasEditingId && profile && presetMgrExpanded && presetMgrContainer && presetMgrContainer.parentNode) {
            const row = createPresetRowElement(profile);
            // 初始态：零高度、透明、左侧偏移
            row.style.maxHeight = '0';
            row.style.overflow = 'hidden';
            row.style.opacity = '0';
            row.style.transform = 'translateX(-30px)';
            row.style.paddingTop = '0';
            row.style.paddingBottom = '0';
            // 插入到标题栏之后（此时行不占可见空间）
            const header = presetMgrContainer.firstElementChild;
            if (header) {
                header.after(row);
            } else {
                presetMgrContainer.appendChild(row);
            }
            // 下一帧触发动画
            requestAnimationFrame(() => {
                row.classList.remove('preset-row-enter');
                row.classList.add('preset-row-insert');
            });
            // 动画结束后清理 inline 覆盖
            row.addEventListener('animationend', () => {
                row.style.maxHeight = '';
                row.style.overflow = '';
            }, { once: true });
        }
    } catch (e) {
        nm.show('保存失败: ' + e, 'error');
    }
}

// 删除预设
async function deleteProfile(id, name, rowEl) {
    const confirmed = await showConfirmDialog(`确定删除配置「${name}」吗？`);
    if (!confirmed) return;
    // 先播放删除动画
    if (rowEl) {
        rowEl.classList.remove('preset-row-enter', 'preset-row-insert');
        rowEl.classList.add('preset-delete-out');
        await new Promise(resolve => {
            rowEl.addEventListener('animationend', resolve, { once: true });
        });
    }
    try {
        await window.go.main.App.DeleteProfile(id);
        nm.show('配置已删除', 'success');
        await loadProfiles();
        await loadProfilesEmbed(); // 预设共享，两个模块的下拉同步刷新
        // 仅移除已播放删除动画的行，避免全量重渲染的闪烁
        if (rowEl && rowEl.parentNode) {
            rowEl.remove();
        }
    } catch (e) {
        nm.show('删除失败: ' + e, 'error');
    }
}

// 渲染管理列表（展开在设置页内；anchorRow 指定插入位置，默认对话连接行）
function renderPresetMgrList(anchorRow) {
    if (!presetMgrContainer) {
        presetMgrContainer = document.createElement('div');
        presetMgrContainer.className = 'preset-mgr-list';
        presetMgrContainer.style.cssText = 'margin-top:8px;padding:12px;border:1px solid var(--border);border-radius:var(--radius-md);background:var(--input-bg);';
        // 初始化滚动条自动显隐（仅第一次）
        let timer = null;
        presetMgrContainer.addEventListener('scroll', (e) => {
            if (e.target !== presetMgrContainer) return;
            presetMgrContainer.classList.add('scrolling');
            clearTimeout(timer);
            timer = setTimeout(() => {
                presetMgrContainer.classList.remove('scrolling');
            }, 1000);
        });
        // 插入到触发按钮所在行的下方
        const presetRow = anchorRow || document.querySelector('.preset-select-row');
        if (presetRow) {
            presetRow.after(presetMgrContainer);
        } else {
            const settingsSection = document.querySelector('.ai-setting-item.preset-select-row');
            if (settingsSection) settingsSection.after(presetMgrContainer);
        }
        // 下一帧触发入场动画
        requestAnimationFrame(() => {
            presetMgrContainer.classList.add('open');
        });
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
        profiles.forEach((p, index) => {
            const row = createPresetRowElement(p);
            row.style.animationDelay = `${index * 50}ms`;
            presetMgrContainer.appendChild(row);
        });
    }).catch(e => {
        nm.show('加载失败: ' + e, 'error');
    });
}

// 创建单行预设列表条目 DOM 元素（带编辑/删除事件绑定）
function createPresetRowElement(p) {
    const row = document.createElement('div');
    row.style.cssText = 'display:flex;align-items:center;justify-content:space-between;padding:6px 8px;border-radius:var(--radius-sm);gap:8px;';
    row.style.borderBottom = '1px solid var(--border)';
    row.classList.add('preset-row-enter');
    row.dataset.profileId = p.id;
    // 信息区
    const info = document.createElement('div');
    info.style.cssText = 'flex:1;min-width:0;display:flex;flex-direction:column;gap:2px;';
    const nameRow = document.createElement('div');
    nameRow.style.cssText = 'display:flex;align-items:center;gap:10px;';
    // 品牌徽章（与预设下拉列表一致）
    nameRow.appendChild(createPresetBadge(p.base_url, p.name));
    const nameSpan = document.createElement('strong');
    nameSpan.style.cssText = 'font-size:0.85rem;color:var(--text-primary);';
    nameSpan.textContent = p.name;
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
        openEditProfileModal(p.id, p.name, p.base_url, p.api_key);
    });
    const delBtn = document.createElement('button');
    delBtn.className = 'btn btn-sm btn-danger';
    delBtn.textContent = '删除';
    delBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        deleteProfile(p.id, p.name, row);
    });
    actions.appendChild(editBtn);
    actions.appendChild(delBtn);
    row.appendChild(info);
    row.appendChild(actions);
    return row;
}

// 关闭管理列表（返回 Promise，动画结束后 resolve）
function closePresetMgrList() {
    presetMgrExpanded = false;
    if (!presetMgrContainer) return Promise.resolve();
    const container = presetMgrContainer;
    container.classList.remove('open');
    container.classList.remove('closing');
    // 使用 Web Animations API 直接驱动关闭动画，彻底绕过 CSS animation 切换不重启的问题
    const anim = container.animate([
        { opacity: 1, transform: 'scaleY(1)', filter: 'blur(0)', maxHeight: '500px', paddingTop: '12px', paddingBottom: '12px' },
        { opacity: 0, transform: 'scaleY(0.95)', filter: 'blur(2px)', maxHeight: '0', paddingTop: '0', paddingBottom: '0' }
    ], { duration: 280, easing: 'ease-in-out', fill: 'both', transformOrigin: 'top center' });
    return new Promise(resolve => {
        anim.onfinish = () => {
            if (container.parentNode) {
                container.parentNode.removeChild(container);
                if (presetMgrContainer === container) {
                    presetMgrContainer = null;
                }
            }
            resolve();
        };
    });
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




/**
 * 删除标签（含删除动画）
 */
async function deleteTag(id) {
    // 播放淡出动画
    const el = document.querySelector(`.tag-item[data-tag-id="${id}"]`);
    if (el) {
        el.classList.add('tag-deleting');
        // 等待动画结束（设 300ms 超时，防止无动画时 promise 挂起）
        await new Promise((resolve) => {
            const timer = setTimeout(resolve, 300);
            el.addEventListener('animationend', () => {
                clearTimeout(timer);
                resolve();
            }, { once: true });
        });
        // 动画结束后从 DOM 移除
        el.remove();
    }

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteTag) {
            await window.go.main.App.DeleteTag(id);
        } else {
            console.warn('DeleteTag 未绑定');
        }
    } catch (err) {
        console.error('删除标签失败:', err);
    }

    // 从 state.tags 移除
    state.tags = state.tags.filter(t => t.id !== id);

    // 若列表为空则显示空状态
    if (state.tags.length === 0) {
        showTagEmptyState();
    }

    renderTagSelector();
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
                <button class="card-action-btn pin-btn" onclick="event.stopPropagation(); window.handlePinClick(event, ${note.id})" title="${note.pinned ? '已置顶' : '置顶'}">
                    ${note.pinned ? SVGS.pinFilled : SVGS.pinOutline}
                </button>
            </div>
        </div>
        `
        )
        .join('');

    // 卡片入场动画：交错淡入 + 微上移，使用 backwards 避免阻塞 :hover
    const cards = els.cardGrid.querySelectorAll('.note-card');
    if (animateMode === 'none') {
        // 批量操作，无动画
        cards.forEach(card => { card.style.opacity = '1'; });
    } else if (animateMode === 'append' && typeof prevCount === 'number') {
        // 追加模式：已有卡片可见，新卡片带交错动画
        cards.forEach((card, index) => {
            if (index < prevCount) {
                card.style.opacity = '1';
                card.style.animation = 'none';
            } else {
                const delay = Math.min((index - prevCount) * 30, 360);
                card.style.willChange = 'transform, opacity';
                card.style.animation = `cardEnter 0.35s cubic-bezier(0.16, 1, 0.3, 1) backwards`;
                card.style.animationDelay = `${delay}ms`;
            }
        });
    } else {
        // 全量刷新：所有卡片带交错动画（backwards 无需清理）
        cards.forEach((card, index) => {
            const delay = Math.min(index * 30, 360);
            card.style.willChange = 'transform, opacity';
            card.style.animation = `cardEnter 0.35s cubic-bezier(0.16, 1, 0.3, 1) backwards`;
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
/**
 * 生成单个标签卡片的 HTML 字符串
 * @param {object} tag - { id, name, color, notes? }
 * @returns {string}
 */
function createTagElementHtml(tag) {
    const color = tag.color || '#6366f1';
    const count = tag.notes ? tag.notes.length : '';
    return `<div class="tag-item" data-tag-id="${tag.id}" style="--tag-color: ${color}">
        <span class="tag-color-dot"></span>
        <span class="tag-name">${escapeHtml(tag.name)}</span>
        ${count ? `<span class="tag-count">${count}</span>` : ''}
        <button class="tag-delete-btn" onclick="window.deleteTag(${tag.id})">${SVGS.windowClose}</button>
    </div>`;
}

/** 显示标签列表空状态 */
function showTagEmptyState() {
    els.tagList.innerHTML = `
    <div class="tag-empty">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z"/>
            <line x1="7" y1="7" x2="7.01" y2="7"/>
        </svg>
        <span class="tag-empty-text">暂无标签</span>
        <span class="tag-empty-hint">在上方输入标签名称开始创建</span>
    </div>`;
}

/**
 * 全量渲染标签列表
 */
function renderTagList() {
    if (state.tags.length === 0) {
        showTagEmptyState();
        return;
    }
    els.tagList.innerHTML = state.tags.map(tag => createTagElementHtml(tag)).join('');
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
    // 同步大纲按钮可见性
    if (els.tocToggleBtn) {
        els.tocToggleBtn.classList.toggle('show-in-preview', isMd);
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
		window.cmEditor = null;
		const useSyntaxHighlight = els.mdHighlightToggle.checked;
		const enableWordWrap = els.editorWordWrapToggle?.checked || false;
		initCodeMirror(container, content, isReadOnly, useSyntaxHighlight, value, codeHighlightTheme, enableWordWrap);
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

    // 同步大纲按钮可见性
    if (els.tocToggleBtn) {
        els.tocToggleBtn.classList.toggle('show-in-preview', newExt === '.md');
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
		window.cmEditor = null;
		const useSyntaxHighlight = els.mdHighlightToggle.checked;
		const enableWordWrap = els.editorWordWrapToggle?.checked || false;
		initCodeMirror(container, content, isReadOnly, useSyntaxHighlight, newExt, codeHighlightTheme, enableWordWrap);
    }
}

/**
 * 打开编辑器（新建/编辑/查看）
 * @param {number|null} noteId - 笔记 ID，null 表示新建
 * @param {boolean} readOnly - 是否为只读查看模式
 * @param {boolean} [startFullscreen] - 是否以全屏尺寸打开
 * @param {boolean} [hideEditBtn] - 是否隐藏"编辑"按钮（从召回卡片打开时用）
 */
async function openEditor(noteId, readOnly, startFullscreen, hideEditBtn) {
    // 捕获本次打开的操作代际：若阶段二异步加载期间发生了新的打开/关闭，本次续体将被放弃
    const mySeq = ++editorOpSeq;
    state.editingNoteId = noteId || null;
    state.selectedTags = [];

    const isReadOnly = readOnly && noteId != null;
    let noteData = null;
    let editorContent = '';

    // ── 阶段一：面板立即展示（同步，无 await） ──

    // 笔记元数据从 state.notes 中同步获取（已有缓存）
    if (noteId) {
        noteData = state.notes.find((n) => n.id === noteId) || null;
        if (noteData) {
            els.editorNoteTitle.value = noteData.title || '';
            state.selectedTags = (noteData.tags || []).map((t) => t.id);
        } else {
            // noteId 存在但不在缓存（如从搜索/召回/其他笔记本打开）：
            // 清空上一笔记残留的标题与标签，等待阶段二从后端加载后再填充
            els.editorNoteTitle.value = '';
            state.selectedTags = [];
        }
    } else {
        const now = new Date();
        const pad = (n) => String(n).padStart(2, '0');
        els.editorNoteTitle.value = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())} ☺️`;
        state._defaultNewNoteTitle = els.editorNoteTitle.value;
    }

    // 只读/编辑模式 UI 切换（同步）
    els.editorNoteTitle.readOnly = isReadOnly;
    els.editorNoteTitle.classList.toggle('editor-input-readonly', isReadOnly);
    els.editorSaveBtn.style.display = isReadOnly ? 'none' : '';
    els.editorCancelBtn.style.display = isReadOnly ? 'none' : '';
    els.editorPanel.classList.toggle('editor-view-mode', isReadOnly);
    if (els.editorTypeToggle)
        els.editorTypeToggle.style.display = isReadOnly ? 'none' : '';
    if (els.editorActionsBtn)
        els.editorActionsBtn.style.display = isReadOnly ? 'none' : '';
    els.editorEditBtn.style.display = (isReadOnly && !hideEditBtn) ? '' : 'none';
    els.editorViewBtn.style.display = (!isReadOnly && state.editingNoteId != null) ? '' : 'none';
    els.editorFileExt.classList.toggle('file-ext-readonly', !!isReadOnly);

    // 文件后缀和模式切换
    let ext = (noteData && noteData.file_ext) || '.txt';
    let isMd = ext === '.md';
    els.editorModes.style.display = isMd ? '' : 'none';
    updateFileExtDisplay(ext);
    if (els.tocToggleBtn)
        els.tocToggleBtn.classList.toggle('show-in-preview', isMd);
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = isMd ? 'M' : 'T';
        els.editorTypeToggle.title = isMd ? '切换为纯文本格式' : '切换为 Markdown 格式';
    }

    els.editorOverlay.dataset.mode = 'edit';

    // ★ 清空上一笔记的预览残留：无论本次是哪种模式，先清掉 mdRendered 与预览哈希缓存，
    //   防止 closeEditor 清理被跳过（<200ms 内重新打开）时旧笔记的预览内容短暂显示
    els.mdRendered.innerHTML = '';
    _lastPreviewContent = '';

    // 查看模式：预览状态显示（数据在阶段二填充）
    if (isReadOnly && noteData) {
        els.editorEditTime.textContent = '最近编辑 ' + formatTime(noteData.updated_at || noteData.created_at);
        if (!isMd) {
            els.mdRendered.style.display = 'none';
            _setPreviewLayout(false);
            _closeToc();
        } else {
            els.editorOverlay.dataset.mode = 'preview';
            els.editorModeBtns.forEach(btn => {
                btn.classList.toggle('active', btn.dataset.mode === 'preview');
            });
            els.mdRendered.innerHTML = '<p class="md-empty">暂无内容</p>';
            _setPreviewLayout(false);
            _closeToc();
            // 内容在阶段二加载后触发 updatePreview
        }
    } else {
        els.editorEditTime.textContent = '';
        switchEditorMode('edit');
    }

    // 标题输入监听（先移除再按需添加：closeEditor 清理被跳过时防止监听器重复绑定导致 onEditorInput 双调）
    els.editorNoteTitle.removeEventListener('input', onEditorInput);
    if (!isReadOnly) {
        els.editorNoteTitle.addEventListener('input', onEditorInput);
        state._titleInputListenerAttached = true;
    } else {
        state._titleInputListenerAttached = false;
    }

    // 每次 openEditor 均为"新打开"：重置从查看模式进入编辑的标志（closeEditor 清理被跳过时避免残留旧值）
    state.enteredFromViewMode = false;

    // ── 立即显示面板 + 骨架屏（不等数据加载） ──
    els.mainContent.style.overflow = 'hidden';
    els.viewEditor.classList.add('active');

    // 在 CM6 挂载点显示骨架屏 shimmer
    const contentArea = document.getElementById('editorNoteContent');
    contentArea.innerHTML = ''
        + '<div class="editor-skeleton">'
        + '<div class="editor-skeleton-line"></div>'
        + '<div class="editor-skeleton-line"></div>'
        + '<div class="editor-skeleton-line"></div>'
        + '<div class="editor-skeleton-line"></div>'
        + '</div>';

    // 启动入场动画
    const overlay = els.editorOverlay;
    const panel = els.editorPanel;
    const body = panel.querySelector('.editor-body');
    document.getElementById('topbar').classList.add('editor-fullscreen');

    if (startFullscreen) {
        panel.style.transition = 'none';
        overlay.classList.add('fullscreening');
        panel.classList.add('fullscreen');
        void panel.offsetHeight;
        panel.style.transition = '';
        state._isFullscreen = true;
        if (els.editorFullscreenBtn) {
            els.editorFullscreenBtn.innerHTML = SVGS.editorExitFullscreen;
            els.editorFullscreenBtn.title = '退出全屏';
            els.editorFullscreenBtn.classList.add('fullscreen');
        }
        if (els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed'))
            els.notebookSidebar.classList.add('collapsed');
        overlay.style.opacity = '1';
        panel.style.opacity = '1';
        panel.style.transform = 'none';   // scale(1) 与 none 视觉一致，但 none 不破坏 position: sticky
    } else {
        overlay.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
        panel.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
        if (body)
            body.style.animation = 'viewEnter 0.25s ease-out forwards';
        // 动画结束后清除 transform（scale(1) 与 none 视觉一致，但 none 不破坏 position: sticky）
        panel.addEventListener('animationend', function editorPanelEnterEnd(e) {
            if (e.target !== panel || e.animationName !== 'modalEnter') return;
            panel.removeEventListener('animationend', editorPanelEnterEnd);
            panel.style.animation = 'none';
            panel.style.transform = 'none';
            panel.style.opacity = '1';
        }, { once: true });
    }

    // ── 阶段二：后台加载数据 + CM6（并行，不阻塞面板动画） ──
    try {
        const loadPromises = [];

        // 加载完整内容（包含 noteId 不在缓存中的回退场景）
        const contentPromise = (async () => {
            let fullContent = '';
            if (noteId) {
                if (noteData) {
                    try {
                        if (window.go.main.App.GetNoteContent)
                            fullContent = await window.go.main.App.GetNoteContent(noteId) || '';
                    } catch (err) {
                        console.error('获取完整笔记内容失败:', err);
                        fullContent = noteData.content || '';
                    }
                } else {
                    // noteId 存在但不在 state.notes 中（如从召回卡片打开），从后端加载完整笔记
                    try {
                        if (window.go.main.App.GetNote) {
                            const fetchedNote = await window.go.main.App.GetNote(noteId);
                            if (fetchedNote) {
                                noteData = fetchedNote;
                                // ★ 竞态保护：本次打开已被更新的 open/close 取代时，
                                //   不再污染标题/标签（否则旧笔记续体会覆盖新打开的笔记界面）
                                if (mySeq === editorOpSeq) {
                                    els.editorNoteTitle.value = fetchedNote.title || '';
                                    state.selectedTags = (fetchedNote.tags || []).map((t) => t.id);
                                }
                                try {
                                    if (window.go.main.App.GetNoteContent)
                                        fullContent = await window.go.main.App.GetNoteContent(noteId) || '';
                                } catch (err) {
                                    console.error('获取完整笔记内容失败:', err);
                                    fullContent = fetchedNote.content || '';
                                }
                            }
                        }
                    } catch (err) {
                        console.error('获取笔记失败:', err);
                    }
                }
            }
            editorContent = fullContent;
        })();
        loadPromises.push(contentPromise);

        // 加载标签（与内容加载并行）
        const tagsPromise = loadTagsForEditor(isReadOnly);
        loadPromises.push(tagsPromise);

        await Promise.all(loadPromises);
    } catch (err) {
        console.error('编辑器数据加载失败:', err);
    }

    // ★ 竞态保护：本次打开已被更新的 open/close 操作取代 → 放弃后续初始化，
    //   防止旧笔记内容（initCodeMirror / 预览渲染）覆盖新打开的笔记
    if (mySeq !== editorOpSeq) return;

    // 校正：从后端加载的笔记（不在缓存中），更新阶段一无法获取的信息
    if (noteData) {
        const loadedExt = (noteData.file_ext || '.txt');
        if (loadedExt !== els.editorFileExt.textContent) {
            ext = loadedExt;
            isMd = ext === '.md';
            updateFileExtDisplay(ext);
            els.editorModes.style.display = isMd ? '' : 'none';
            if (els.tocToggleBtn)
                els.tocToggleBtn.classList.toggle('show-in-preview', isMd);
            if (els.editorTypeToggle) {
                els.editorTypeToggle.textContent = isMd ? 'M' : 'T';
                els.editorTypeToggle.title = isMd ? '切换为纯文本格式' : '切换为 Markdown 格式';
            }
            if (isReadOnly && isMd) {
                els.editorOverlay.dataset.mode = 'preview';
                els.editorModeBtns.forEach(btn => {
                    btn.classList.toggle('active', btn.dataset.mode === 'preview');
                });
                _setPreviewLayout(false);
                _closeToc();
            }
        }
        // 查看模式更新编辑时间
        if (isReadOnly && !els.editorEditTime.textContent) {
            els.editorEditTime.textContent = '最近编辑 ' + formatTime(noteData.updated_at || noteData.created_at);
        }
        // 重绘标签选择器（校正并行加载时序：非缓存笔记的 state.selectedTags 可能在 loadTagsForEditor 之后才填充）
        renderTagSelector(isReadOnly);
    }

    // 移除骨架屏
    contentArea.innerHTML = '';

    // 初始化 CM6（已拿到内容）
    const useSyntaxHighlight = els.mdHighlightToggle.checked;
    const enableWordWrap = els.editorWordWrapToggle?.checked || false;
    initCodeMirror(contentArea, editorContent, isReadOnly, useSyntaxHighlight, ext, codeHighlightTheme, enableWordWrap);
    updateWordCount();

    // 编辑模式下记录快照
    if (!isReadOnly && state.editingNoteId) {
        state._editSnapshot = {
            title: els.editorNoteTitle.value.trim(),
            content: getEditorContent().trim(),
            tags: [...state.selectedTags].sort(),
            fileExt: els.editorFileExt.textContent
        };
    }

    // 查看模式：Markdown 预览（CM6 就绪后刷新）
    // 大文件自动切换纯文本模式：内容长度超过大文件预览阈值（ai_large_file_preview_threshold）时跳过预览
    if (isReadOnly && isMd && els.editorOverlay.dataset.mode === 'preview') {
        const largeFileThreshold = parseInt(document.getElementById('aiLargeFilePreviewThreshold')?.value) || 10000;
        if (editorContent.length > largeFileThreshold) {
            // 内容过长，自动切换为纯文本模式
            switchEditorMode('edit');
            _setPreviewLayout(false);
            _closeToc();
            window.showNotification?.('笔记内容超过纯文本预览阈值，已自动切换为纯文本模式', 'info');
        } else {
            updatePreview();
        }
    }

    // 新建笔记时自动聚焦
    if (!state.editingNoteId && els.editorOverlay.dataset.mode !== 'preview' && document.hasFocus()) {
        window.focus();
        cmEditor?.focus();
    }
}

/** 预览渲染处理中标志，防重复请求 */
let _previewWorkerLoading = false;

/** 预览模式查找条状态 */
let _previewFindBarVisible = false;   // 查找条是否打开
let _previewSearchQuery = '';         // 当前搜索关键词
let _previewMarkMatches = [];         // 当前高亮的 <mark> 元素数组
let _previewMarkCurrent = -1;         // 当前激活匹配索引
let _previewFindTimer = null;         // 输入防抖定时器

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
            const { html, error, headings, seq } = e.data;
            // ★ 竞态保护：丢弃过期渲染结果（属于更早一次 updatePreview 请求），仅释放 loading 标志，
            //   防止旧笔记的预览渲染结果晚到覆盖当前笔记（切换笔记时预览区闪烁）
            if (seq !== undefined && seq !== previewRenderSeq) {
                _previewWorkerLoading = false;
                return;
            }
            if (error) {
                console.error('Preview Worker:', error);
                els.mdRendered.innerHTML = '<p class="md-error">渲染失败</p>';
                _previewWorkerLoading = false;
                return;
            }
            // 在 innerHTML 替换前捕获 loading 元素引用，用于后续交叉淡出
            const oldLoading = els.mdRendered.querySelector('.md-rendered-loading');
            // 设置渲染结果
            els.mdRendered.innerHTML = html;
            // hljs 高亮（必须在主线程，需要 DOM 环境，跳过 Mermaid 代码块）
            els.mdRendered.querySelectorAll('pre code').forEach((block) => {
                if (block.classList.contains('language-mermaid')) return;
                if (typeof hljs !== 'undefined') {
                    hljs.highlightElement(block);
                }
            });
            // 复制按钮、语言标签、表格按钮等 DOM 后处理
            _applyPreviewDOMHelpers();
            // 为 Mermaid 代码块设置交互结构（默认显示源码，不自动渲染）
            renderMermaidBlocks(els.mdRendered);
            // 生成标题锚点 ID 并渲染 TOC
            _ensureHeadingIds();
            _renderToc(headings || []);
            _bindTocScrollListener();
            _setPreviewLayout(true);
            _ensureTocReady();  // 确保 TOC 状态正确
            // 内容淡入动画
            _applyPreviewFadeIn();
            // 交叉淡入：把之前捕获的 loading 元素转为 absolute 覆盖层置于内容上方，平滑淡出
            if (oldLoading) {
                oldLoading.style.position = 'absolute';
                oldLoading.style.inset = '0';
                oldLoading.style.zIndex = '5';
                oldLoading.style.minHeight = '0';
                oldLoading.style.background = 'var(--card-bg)';
                oldLoading.style.animation = 'none';  // 停止脉动
                oldLoading.style.pointerEvents = 'none';
                els.mdRendered.appendChild(oldLoading);
                requestAnimationFrame(() => {
                    oldLoading.style.opacity = '0';
                    oldLoading.style.transition = 'opacity 0.25s ease-out';
                });
                oldLoading.addEventListener('transitionend', () => oldLoading.remove(), { once: true });
            }
            // 若预览查找条仍打开，重新执行搜索（innerHTML 替换后 mark 已丢失）
            if (_previewFindBarVisible && _previewSearchQuery) {
                runPreviewSearch(_previewSearchQuery);
            }
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
    // 代码高亮（跳过 Mermaid 代码块，由其独立渲染）
    els.mdRendered.querySelectorAll('pre code').forEach((block) => {
        if (block.classList.contains('language-mermaid')) return;
        if (typeof hljs !== 'undefined') {
            hljs.highlightElement(block);
        }
    });
    // 先包裹 pre → .pre-wrapper，再添加语言标签
    // （需要在添加复制按钮前完成包裹，使复制按钮直接放在 .pre-wrapper 内，
    //  与 AI 消息的 DOM 结构一致，消除两个按钮间 1px 合成层偏移）
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
    // 为每个代码块添加复制按钮（放在 .pre-wrapper 内，与 AI 消息 DOM 结构一致）
    els.mdRendered.querySelectorAll('.pre-wrapper').forEach((wrapper) => {
        if (wrapper.querySelector('.copy-code-btn')) return;
        const pre = wrapper.querySelector('pre');
        const codeEl = pre && pre.querySelector('code');
        const btn = document.createElement('button');
        const isSingleLine = codeEl && !codeEl.textContent.trim().includes('\n');
        btn.className = 'copy-code-btn' + (isSingleLine ? ' copy-code-btn--single' : '');
        btn.innerHTML = SVGS.copy + ' 复制';
        btn.title = '复制代码';
        btn.addEventListener('click', async () => {
            const code = pre.querySelector('code').textContent;
            try {
                await navigator.clipboard.writeText(code);
                // 先触发渲染按钮滑出动画，再变"已复制"
                wrapper.classList.add('copying');
                await new Promise(r => setTimeout(r, 80));
                btn.classList.add('copied');
                btn.innerHTML = SVGS.checkmark + ' 已复制';
                wrapper.classList.remove('copying');
                setTimeout(() => {
                    btn.classList.remove('copied');
                    btn.innerHTML = SVGS.copy + ' 复制';
                }, 1500);
            } catch {
                btn.innerHTML = SVGS.xmark + ' 复制失败';
                setTimeout(() => { btn.innerHTML = SVGS.copy + ' 复制'; }, 1000);
            }
        });
        wrapper.appendChild(btn);
    });
    // 为每个表格添加复制按钮（按钮放在首行最后一个 <th> 内，锚定 HTML 列而非可视边缘）
    els.mdRendered.querySelectorAll('table').forEach((table) => {
        // 仅保证 table 被 .table-wrapper 包裹（用于间距），不重复创建
        if (!table.parentNode.classList.contains('table-wrapper')) {
            const w = document.createElement('div');
            w.className = 'table-wrapper';
            table.parentNode.insertBefore(w, table);
            w.appendChild(table);
        }
        const lastTh = table.querySelector('tr:first-child th:last-child');
        if (!lastTh) return;
        if (lastTh.querySelector('.copy-table-btn')) return;

        const btn = document.createElement('button');
        btn.className = 'copy-table-btn';
        btn.innerHTML = SVGS.copy + ' 复制';
        btn.title = '复制表格';
        btn.addEventListener('click', async () => {
            const md = tableToMarkdown(table);
            try {
                await navigator.clipboard.writeText(md);
                btn.classList.add('copied');
                btn.innerHTML = SVGS.checkmark + ' 已复制';
                setTimeout(() => {
                    btn.classList.remove('copied');
                    btn.innerHTML = SVGS.copy + ' 复制';
                }, 1500);
            } catch {
                btn.innerHTML = SVGS.xmark + ' 复制失败';
                setTimeout(() => { btn.innerHTML = SVGS.copy + ' 复制'; }, 1000);
            }
        });
        lastTh.appendChild(btn);
    });

    // 图片灯箱：点击图片展开全屏查看（丝滑开合动画）
    els.mdRendered.querySelectorAll('img').forEach((img) => {
        img.style.cursor = 'zoom-in';
        img.addEventListener('click', (e) => {
            e.stopPropagation();

            const overlay = document.createElement('div');
            overlay.className = 'image-lightbox';
            overlay.innerHTML = `
                <button class="lightbox-close" aria-label="关闭">✕</button>
                <img src="${img.src}" alt="${img.alt || ''}" draggable="false">
            `;

            // 打开动画：DOM 插入后下一帧添加 .active 触发 CSS transition
            window.__lightboxOpen = true;
            document.body.appendChild(overlay);
            requestAnimationFrame(() => {
                requestAnimationFrame(() => {
                    overlay.classList.add('active');
                });
            });

            // 关闭逻辑：加 .closing → 等 transition 结束 → 移除
            const close = () => {
                if (overlay.classList.contains('closing')) return;
                window.__lightboxOpen = false;
                document.removeEventListener('keydown', onKey);
                overlay.classList.remove('active');
                overlay.classList.add('closing');
                overlay.addEventListener('transitionend', () => {
                    overlay.remove();
                }, { once: true });
            };

            overlay.addEventListener('click', (e) => {
                if (e.target === overlay) close();
            });
            overlay.querySelector('.lightbox-close')?.addEventListener('click', close);

            // ESC 键关闭
            const onKey = (ke) => { if (ke.key === 'Escape') close(); };
            document.addEventListener('keydown', onKey);
        });
    });
}

/* ===== Mermaid 图表渲染 ===== */

/** Mermaid 引擎是否已初始化 */
let _mermaidInited = false;

/**
 * 一次性初始化 mermaid 引擎
 * 设置 startOnLoad: false 由用户手动触发渲染，securityLevel: 'loose' 允许渲染 HTML
 */
function initMermaid() {
    if (_mermaidInited) return;
    mermaid.initialize({
        startOnLoad: false,
        securityLevel: 'loose',
        theme: getMermaidTheme(),
    });
    _mermaidInited = true;
}

/**
 * 根据当前系统主题获取 Mermaid 主题名称
 * @returns {'dark'|'default'}
 */
function getMermaidTheme() {
    const themeName = document.documentElement.getAttribute('data-theme') || 'default';
    return isDarkTheme[themeName] ? 'dark' : 'default';
}

/**
 * 为单个 Mermaid 代码块设置交互结构
 * 创建 .mermaid-rendered 容器 + .mermaid-toggle 按钮，默认显示源码
 * @param {HTMLPreElement} pre - <pre> 元素
 */
function setupMermaidBlock(pre) {
    if (pre.dataset.mermaidProcessed) return;
    pre.dataset.mermaidProcessed = 'true';

    const code = pre.querySelector('code.language-mermaid');
    if (!code) return;

    // 确保 pre 已被 pre-wrapper 包裹
    let wrapper = pre.parentNode;
    if (!wrapper.classList.contains('pre-wrapper')) {
        wrapper = document.createElement('div');
        wrapper.className = 'pre-wrapper has-mermaid';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);
    } else {
        wrapper.classList.add('has-mermaid');
    }

    // 在 data-mermaid-code 中存储原始源码（去除首尾空白）
    const mermaidCode = code.textContent.trim();
    pre.dataset.mermaidCode = mermaidCode;

    // 创建渲染结果容器（初始隐藏）
    const rendered = document.createElement('div');
    rendered.className = 'mermaid-rendered';
    rendered.style.display = 'none';
    wrapper.appendChild(rendered);

    // 初始状态标记：源码视图
    wrapper.dataset.mermaidView = 'source';

    // 创建切换按钮
    const toggleBtn = document.createElement('button');
    toggleBtn.className = 'mermaid-toggle';
    toggleBtn.innerHTML = SVGS.diagram + ' 渲染';
    toggleBtn.title = '渲染为 Mermaid 图表';
    toggleBtn.addEventListener('click', () => toggleMermaidView(toggleBtn));
    wrapper.appendChild(toggleBtn);
}


/**
 * 在渲染视图和源码视图之间切换，用 dataset.mermaidView 追踪状态
 * 渲染流程：pre 淡出（200ms）→ display:none 彻底隐藏 → rendered 显示
 * 切回流程：rendered 隐藏 → pre 即刻恢复（无动画）
 * @param {HTMLButtonElement} btn - 切换按钮
 */
async function toggleMermaidView(btn) {
    const wrapper = btn.parentNode;
    const pre = wrapper.querySelector('pre');
    const rendered = wrapper.querySelector('.mermaid-rendered');
    if (!pre || !rendered) return;

    if (wrapper.dataset.mermaidView === 'rendered') {
        // 切回源码视图：取消未执行的切换定时器，恢复 pre，隐藏 rendered
        if (wrapper._mermaidTimer) {
            clearTimeout(wrapper._mermaidTimer);
            wrapper._mermaidTimer = null;
        }
        rendered.style.display = 'none';
        pre.style.display = '';
        pre.classList.remove('pre-hiding');
        wrapper.dataset.mermaidView = 'source';
        btn.innerHTML = SVGS.diagram + ' 渲染';
        btn.title = '渲染为 Mermaid 图表';
    } else {
        // 切到渲染视图
        wrapper.dataset.mermaidView = 'rendered';
        btn.innerHTML = SVGS.code + ' 源码';
        btn.title = '显示源码';

        // 异步渲染 SVG
        const mermaidCode = pre.dataset.mermaidCode;
        if (mermaidCode) {
            mermaid.initialize({ theme: getMermaidTheme() });
            const id = 'mermaid-' + Date.now() + '-' + Math.random().toString(36).slice(2, 8);
            try {
                const { svg } = await mermaid.render(id, mermaidCode);
                rendered.innerHTML = svg;
            } catch (err) {
                console.warn('Mermaid render error:', err);
                rendered.innerHTML = `<div class="mermaid-error">Mermaid 渲染失败：${err.message}</div>`;
            }
        }

        // 触发 pre 淡出
        pre.classList.add('pre-hiding');
        // 淡出完成后：隐藏 pre，显示 rendered
        wrapper._mermaidTimer = setTimeout(() => {
            pre.style.display = 'none';
            pre.classList.remove('pre-hiding');
            rendered.style.display = '';
            wrapper._mermaidTimer = null;
        }, 220);
    }
}

/**
 * 遍历容器中所有未处理的 Mermaid 代码块，为每个设置交互结构
 * @param {HTMLElement} container - 包含渲染后 HTML 的容器元素
 */
function renderMermaidBlocks(container) {
    if (!container) return;
    initMermaid();
    container.querySelectorAll('pre code.language-mermaid').forEach((code) => {
        const pre = code.parentNode;
        setupMermaidBlock(pre);
    });
}

// 暴露给 AI 模块使用
window.renderMermaidBlocks = renderMermaidBlocks;

/**
 * 为预览中的 h1~h6 元素生成唯一锚点 ID
 */
function _ensureHeadingIds() {
    if (!els.mdRendered) return;
    const usedIds = new Set();
    els.mdRendered.querySelectorAll('h1,h2,h3,h4,h5,h6').forEach((h) => {
        let id = h.textContent.trim()
            .toLowerCase()
            .replace(/[^\w\u4e00-\u9fff]+/g, '-')
            .replace(/^-+|-+$/g, '');
        if (!id) id = 'heading';
        // 处理重复 ID：追加 -1, -2 后缀
        let candidate = id;
        if (usedIds.has(id)) {
            let n = 1;
            while (usedIds.has(id + '-' + n)) n++;
            candidate = id + '-' + n;
        }
        usedIds.add(candidate);
        h.id = candidate;
    });
}

/**
 * 从当前 mdRendered DOM 中提取标题数组
 * @returns {Array<{depth:number, text:string}>}
 */
function _extractHeadingsFromDOM() {
    if (!els.mdRendered) return [];
    const headings = [];
    els.mdRendered.querySelectorAll('h1,h2,h3,h4,h5,h6').forEach((h) => {
        headings.push({ depth: parseInt(h.tagName.charAt(1), 10), text: h.textContent.trim() });
    });
    return headings;
}

/**
 * 渲染 TOC 侧栏
 * @param {Array<{depth:number, text:string}>} headings
 */
function _renderToc(headings) {
    if (!els.tocBody) return;
    els.tocBody.innerHTML = '';
    if (!headings || headings.length === 0) {
        const empty = document.createElement('div');
        empty.className = 'toc-empty';
        empty.textContent = '无标题';
        els.tocBody.appendChild(empty);
        return;
    }
    const list = document.createElement('div');
    list.className = 'toc-list';
    headings.forEach((h, index) => {
        const btn = document.createElement('button');
        btn.className = 'toc-item depth-' + Math.min(h.depth, 6);
        btn.textContent = h.text || '(空)';
        btn.dataset.tocIndex = index;
        btn.addEventListener('click', () => {
            const allHeadings = els.mdRendered.querySelectorAll('h1,h2,h3,h4,h5,h6');
            const tocItems = els.tocBody.querySelectorAll('.toc-item');
            const matched = allHeadings[index];
            if (matched && matched.id) {
                // 立即更新 TOC 高亮，不依赖 scroll 事件
                tocItems.forEach((item, i) => {
                    item.classList.toggle('active', i === index);
                });
                // 锁定滚动高亮 ~500ms，防止 smooth 滚动过程中中间位置覆盖点击高亮
                clearTimeout(_tocScrollTimer);
                _tocScrollTimer = setTimeout(() => {
                    _tocScrollTimer = null;
                }, 500);
                matched.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        });
        list.appendChild(btn);
    });
    els.tocBody.appendChild(list);
}

/** TOC 滚动高亮防抖定时器 */
let _tocScrollTimer = null;

/**
 * 更新 TOC 滚动高亮
 */
function _updateTocScrollHighlight() {
    if (_tocScrollTimer) return;
    _tocScrollTimer = setTimeout(() => {
        _tocScrollTimer = null;
        if (!els.mdRendered || !els.tocBody) return;
        const headings = els.mdRendered.querySelectorAll('h1,h2,h3,h4,h5,h6');
        const tocItems = els.tocBody.querySelectorAll('.toc-item');
        if (!headings.length || !tocItems.length) return;

        const containerTop = els.mdRendered.getBoundingClientRect().top;

        // 找到当前最接近顶部的标题
        let activeIndex = -1;
        let lastAboveIndex = -1;
        headings.forEach((h, i) => {
            const rect = h.getBoundingClientRect();
            if (rect.top >= containerTop - 20) {
                if (activeIndex === -1) activeIndex = i;
                return;
            }
            // 如果标题在视口上方，记录最后一个在上方的
            lastAboveIndex = i;
        });
        // 若未有标题在视口内，则使用最后一个在上方的标题
        if (activeIndex === -1 && lastAboveIndex >= 0) activeIndex = lastAboveIndex;
        // 若所有标题都在上方，选中最后一个
        if (activeIndex === -1 && headings.length > 0) activeIndex = headings.length - 1;
        // 若第一个标题还在下方，选中第一个
        if (activeIndex === -1) activeIndex = 0;

        // 高亮对应的 TOC 项
        tocItems.forEach((item, i) => {
            item.classList.toggle('active', i === activeIndex);
        });

        // 将高亮项滚动到 TOC 可视区域
        const activeItem = tocItems[activeIndex];
        if (activeItem) {
            const itemRect = activeItem.getBoundingClientRect();
            const tocRect = els.tocBody.getBoundingClientRect();
            if (itemRect.top < tocRect.top || itemRect.bottom > tocRect.bottom) {
                activeItem.scrollIntoView({ block: 'nearest' });
            }
        }
    }, 100);
}

/** 已绑定的 TOC 滚动监听标志 */
let _tocScrollBound = false;

/**
 * 绑定 TOC 滚动监听（防重复绑定）
 */
function _bindTocScrollListener() {
    if (_tocScrollBound || !els.mdRendered) return;
    els.mdRendered.addEventListener('scroll', _updateTocScrollHighlight, { passive: true });
    _tocScrollBound = true;
}

/**
 * 初始化 TOC 侧栏展开/折叠按钮
 * TOC 默认隐藏，用户点击展开按钮打开，点击关闭按钮收起
 * 展开/折叠状态仅在当前会话生效，不持久化到 localStorage
 */
function _initTocToggle() {
    if (!els.tocSidebar || !els.tocToggleBtn) return;
    // 点击顶部按钮切换展开/折叠
    els.tocToggleBtn.addEventListener('click', () => {
        // 无渲染内容时提示不展开
        const mdContent = els.mdRendered.textContent.trim();
        if (!mdContent) {
            nm.show('正文暂无内容，无法生成目录', 'info');
            return;
        }
        // 无标题时提示不展开
        const hasHeadings = els.mdRendered.querySelectorAll('h1,h2,h3,h4,h5,h6').length > 0;
        if (!hasHeadings) {
            nm.show('当前文档未提取到标题', 'info');
            return;
        }
        const isOpen = els.tocSidebar.classList.toggle('toc-visible');
        els.tocToggleBtn.classList.toggle('active', isOpen);
    });
}

/**
 * 关闭 TOC 侧栏，同步清除按钮活跃态
 */
function _closeToc() {
    if (els.tocSidebar) els.tocSidebar.classList.remove('toc-visible');
    if (els.tocToggleBtn) els.tocToggleBtn.classList.remove('active');
}

/**
 * 检查当前笔记是否为 Markdown 格式
 * @returns {boolean}
 */
function _isMarkdownNote() {
    return els.editorFileExt && els.editorFileExt.textContent === '.md';
}

/**
 * 更新 editor-panes 的 data-preview-layout 属性
 * @param {boolean} enable
 */
function _setPreviewLayout(enable) {
    if (!els.editorPanes) return;
    if (enable && _isMarkdownNote()) {
        els.editorPanes.setAttribute('data-preview-layout', '');
    } else {
        els.editorPanes.removeAttribute('data-preview-layout');
    }
}

/**
 * 集成 TOC：为渲染好的 mdRendered 生成标题 ID、更新 TOC 侧栏
 */
function _integrateToc() {
    // 仅 Markdown 笔记启用 TOC
    if (!_isMarkdownNote()) {
        _closeToc();
        return;
    }
    _ensureHeadingIds();
    const headings = _extractHeadingsFromDOM();
    _renderToc(headings);
    _bindTocScrollListener();
    _setPreviewLayout(true);
}

/**
 * 确保 TOC 侧栏内容已准备（不控制显隐，显隐由用户按钮 + localStorage 控制）
 */
function _ensureTocReady() {
    // 仅确保 TOC 相关状态，不修改 toc-visible
    if (!_isMarkdownNote()) {
        _closeToc();
    }
}

/**
 * 预览淡入动画
 */
function _applyPreviewFadeIn() {
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

/**
 * 更新 Markdown 预览区
 * 通过 Web Worker 离线程解析，不阻塞主线程 UI
 * @param {string} [content] - 可选，指定渲染内容；不传则从 CM6 编辑器获取
 */
function updatePreview(content) {
    if (content === undefined) content = getEditorContent();
    if (!content.trim()) {
        previewRenderSeq++;   // 使在途 Worker 渲染结果失效，防止其覆盖"暂无内容"
        els.mdRendered.innerHTML = '<p class="md-empty">暂无内容</p>';
        _lastPreviewContent = '';
        _setPreviewLayout(false);
        _closeToc();
        if (els.tocBody) els.tocBody.innerHTML = '';
        return;
    }
    // 内容未变化则跳过重复渲染（哈希缓存）
    if (content === _lastPreviewContent) {
        // 但仍需恢复 TOC 和预览布局状态
        // （场景：编辑模式内切"预览"、查看→编辑→查看等，内容未变但 TOC 被之前操作移除了）
        _setPreviewLayout(true);
        _ensureTocReady();
        return;
    }
    _lastPreviewContent = content;

    // 有 Worker 且不在处理中则走 Worker 渲染
    if (_previewWorker && !_previewWorkerLoading) {
        _previewWorkerLoading = true;
        // 显示加载状态
        els.mdRendered.innerHTML = '<div class="md-rendered-loading">加载中…</div>';
        previewRenderSeq++;
        _previewWorker.postMessage({ content, seq: previewRenderSeq });
        return;
    }

    // 无 Worker 或 Worker 正忙时回退到主线程同步渲染
    previewRenderSeq++;   // 使在途 Worker 渲染结果失效，防止其稍后到达覆盖本次同步渲染结果
    els.mdRendered.innerHTML = marked.parse(content);
    _applyPreviewDOMHelpers();
    renderMermaidBlocks(els.mdRendered);
    // 主线程回退路径，自己提取标题
    _integrateToc();
    // 回退路径的淡入动画
    _applyPreviewFadeIn();
    // 若预览查找条仍打开，重新执行搜索（innerHTML 替换后 mark 已丢失）
    if (_previewFindBarVisible && _previewSearchQuery) {
        runPreviewSearch(_previewSearchQuery);
    }
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
    // 预览模式下隐藏操作按钮，编辑模式下显示
    if (els.editorActionsBtn) {
        els.editorActionsBtn.style.display = mode === 'preview' ? 'none' : '';
    }
    // 预览模式下立即渲染（CM6 未就绪时跳过，等 initCodeMirror 完成后自动刷新）
    if (mode === 'preview' && cmEditor) {
        updatePreview();
    } else if (mode === 'edit') {
        _setPreviewLayout(false);
        _closeToc();
        closePreviewFindBar();
    }
    // 切换模式时关闭查找/替换条（CM6 search 自管理）
    if (cmEditor) {
        cmEditor.focus();
    }
}
// 暴露给其他模块（editor-actions.js 操作菜单的预览模式切换使用）
window.switchEditorMode = switchEditorMode;

/**
 * 打开预览查找条（预览模式下 Ctrl+F 触发，直接在预览渲染区搜索）
 */
function openPreviewFindBar() {
    const bar = document.getElementById('editorFindBar');
    const input = document.getElementById('findInput');
    if (!bar || !input) return;
    bar.style.display = '';
    _previewFindBarVisible = true;
    input.value = '';
    _previewSearchQuery = '';
    _clearPreviewMarks();
    updateFindCount(0);
    input.focus();
    input.select();
}

/**
 * 关闭预览查找条并清除高亮
 */
function closePreviewFindBar() {
    const bar = document.getElementById('editorFindBar');
    if (bar) bar.style.display = 'none';
    _previewFindBarVisible = false;
    _previewSearchQuery = '';
    _clearPreviewMarks();
}

/**
 * 清除预览区所有 mark 高亮（还原为文本节点）
 */
function _clearPreviewMarks() {
    _previewMarkMatches.forEach((m) => {
        if (m.parentNode) m.replaceWith(document.createTextNode(m.textContent));
    });
    _previewMarkMatches = [];
    _previewMarkCurrent = -1;
}

/**
 * 在 mdRendered DOM 中执行搜索并高亮
 * 使用 TreeWalker 遍历文本节点，跳过 svg(Mermaid)/script/style 内部文本
 * @param {string} query - 搜索关键词
 */
function runPreviewSearch(query) {
    _clearPreviewMarks();
    updateFindCount(0);
    const text = query.trim();
    if (!text) {
        _previewSearchQuery = '';
        return;
    }
    _previewSearchQuery = query;

    const lower = text.toLowerCase();
    const walker = document.createTreeWalker(els.mdRendered, NodeFilter.SHOW_TEXT, {
        acceptNode(node) {
            if (node.parentElement && node.parentElement.closest('svg, script, style')) {
                return NodeFilter.FILTER_REJECT;
            }
            return NodeFilter.FILTER_ACCEPT;
        }
    });
    // 收集 (node, ranges)——同一节点多个匹配合并处理
    const grouped = [];
    let n;
    while ((n = walker.nextNode())) {
        const nodeText = n.nodeValue;
        const lowerNode = nodeText.toLowerCase();
        const ranges = [];
        let idx = 0;
        while ((idx = lowerNode.indexOf(lower, idx)) !== -1) {
            ranges.push({ start: idx, end: idx + text.length });
            idx += text.length; // 不重叠匹配
            if (ranges.length >= 500) break; // 单节点匹配上限，防极端卡顿
        }
        if (ranges.length) grouped.push({ node: n, ranges });
        if (grouped.length >= 1000) break; // 总分组上限，防极端卡顿
    }

    // 拆分文本节点并包裹 mark
    for (const { node, ranges } of grouped) {
        const frag = document.createDocumentFragment();
        let last = 0;
        for (const r of ranges) {
            if (r.start > last) frag.appendChild(document.createTextNode(node.nodeValue.slice(last, r.start)));
            const mark = document.createElement('mark');
            mark.textContent = node.nodeValue.slice(r.start, r.end);
            frag.appendChild(mark);
            _previewMarkMatches.push(mark);
            last = r.end;
        }
        if (last < node.nodeValue.length) frag.appendChild(document.createTextNode(node.nodeValue.slice(last)));
        node.parentNode.replaceChild(frag, node);
    }

    updateFindCount(_previewMarkMatches.length);
}

/**
 * 导航到上一个/下一个匹配（dir = 1 下一个，-1 上一个）
 */
function navigatePreviewMatch(dir) {
    const total = _previewMarkMatches.length;
    if (!total) return;
    _previewMarkCurrent = (_previewMarkCurrent + dir + total) % total;
    _previewMarkMatches.forEach((m, i) => m.classList.toggle('active', i === _previewMarkCurrent));
    updateFindCount(total);
    const active = _previewMarkMatches[_previewMarkCurrent];
    if (active) active.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

/**
 * 更新查找条计数显示（0/0 或 (当前+1)/总数）
 * @param {number} total
 */
function updateFindCount(total) {
    const el = document.getElementById('findCount');
    if (!el) return;
    el.textContent = total ? `${_previewMarkCurrent + 1}/${total}` : '0/0';
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
    // 递增操作代际：使所有仍在异步加载中的 openEditor 续体失效（其初始化将被放弃）
    const mySeq = ++editorOpSeq;
    const overlay = els.editorOverlay;
    const panel = els.editorPanel;
    const body = panel.querySelector('.editor-body');

    // 退出动画
    panel.style.animation = 'modalExit 0.18s ease-in forwards';
    overlay.style.animation = 'overlayFadeOut 0.15s ease-in forwards';

    // 动画完成后执行清理
    setTimeout(() => {
        // ★ 竞态保护：延迟清理期间发生了新的 open/close → 跳过本次清理
        //   （新 openEditor 已接管面板/CM6/状态，清理会误关其面板、误毁其 CM6）
        if (mySeq !== editorOpSeq) return;
        // 重置动画
        overlay.style.animation = '';
        panel.style.animation = '';
        panel.style.transform = '';   // 清除 inline transform，让 CSS 默认值生效
        panel.style.opacity = '';     // 清除 inline opacity，让 CSS 默认值生效
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
        closePreviewFindBar();
        els.mdRendered.style.display = 'none';
        els.mdRendered.innerHTML = '';
        _lastPreviewContent = '';
        delete els.editorOverlay.dataset.mode;
        // 重置 TOC 状态
        _tocScrollBound = false;
        _setPreviewLayout(false);
        _closeToc();
        if (els.tocBody) els.tocBody.innerHTML = '';
        // 使在途预览 Worker 渲染结果失效（防止面板关闭后旧结果写入 mdRendered，残留上次笔记内容）
        previewRenderSeq++;
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
    // 清理按下态，避免残留
    menu.querySelectorAll('.pressed').forEach(el => el.classList.remove('pressed'));
    menu.style.animation = 'modalExit 0.1s ease-in forwards';
    const onEnd = (ev) => {
        // 先移除监听再校验动画名：若隐藏期间菜单被重新打开（menuEnter 取代 modalExit），
        // 旧监听不得在 menuEnter 结束时误关新菜单
        menu.removeEventListener('animationend', onEnd);
        if (ev.animationName !== 'modalExit') return;
        menu.classList.remove('active');
        menu.style.animation = '';
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

    // 更新标签菜单项可用性：已有 3 个标签时不可再添加；无标签时不可移除
    if (note) {
        const tagCount = (note.tags || []).length;
        const addTagItem = menu.querySelector('[data-action="add-tag"]');
        const removeTagItem = menu.querySelector('[data-action="remove-tag"]');
        if (addTagItem) {
            addTagItem.classList.toggle('disabled', tagCount >= 3);
            addTagItem.title = tagCount >= 3 ? '该笔记标签已达上限（3 个）' : '';
        }
        if (removeTagItem) {
            removeTagItem.classList.toggle('disabled', tagCount === 0);
            removeTagItem.title = tagCount === 0 ? '该笔记暂无标签' : '';
        }
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
    // 检查菜单是否超出视口边界，若超出则修正位置
    requestAnimationFrame(function () {
        const menuHeight = menu.offsetHeight;
        let top = parseFloat(menu.style.top);
        if (top + menuHeight > window.innerHeight - 8) {
            top = window.innerHeight - menuHeight - 8;
        }
        if (top < 8) {
            top = 8;
        }
        menu.style.top = top + 'px';
    });
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
        case 'duplicate':
            window.duplicateNote(id);
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
        case 'add-tag':
            openBatchTagPicker('add', [id]);
            break;
        case 'remove-tag':
            openBatchTagPicker('remove', [id]);
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
 * 切换编辑器中选择的标签（最多 3 个）
 */
window.toggleEditorTag = function (tagId, el) {
    const idx = state.selectedTags.indexOf(tagId);
    if (idx > -1) {
        // 取消选中：无限制
        state.selectedTags.splice(idx, 1);
        el.classList.remove('active');
    } else {
        // 新增选中：检查是否已达上限 3
        if (state.selectedTags.length >= 3) {
            window.showNotification('一篇笔记最多选择 3 个标签', 'warning');
            return;
        }
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
        bar.classList.remove('visible');
        // 标记已退出退出动画阶段，阻止上一次 transitionend 误触
        if (bar._batchExiting) {
            bar._batchExiting = false;
        }
        // 强制回流后触发 transition
        void bar.offsetHeight;
        bar.classList.add('visible');
        renderCardGrid('none');
        updateBatchBar();
    } else {
        // 退出批量模式
        clearSelection();
        bar.classList.remove('visible');
        bar._batchExiting = true;
        const onEnd = (e) => {
            if (e.propertyName === 'max-height' && bar._batchExiting) {
                bar.style.display = 'none';
                bar._batchExiting = false;
            }
            bar.removeEventListener('transitionend', onEnd);
        };
        bar.addEventListener('transitionend', onEnd);
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
    if (els.batchPinBtn) {
        if (count > 0) {
            const anyPinned = Array.from(state.selectedNoteIds).some(id => {
                const note = state.notes.find(n => n.id === id);
                return note && note.pinned;
            });
            els.batchPinBtn.textContent = anyPinned ? '取消置顶' : '置顶';
        } else {
            els.batchPinBtn.textContent = '置顶';
        }
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
    if (ids.length === 0) {
        nm.show('请先选择笔记', 'warning');
        return;
    }
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
 * 与按钮文字保持一致：显示"取消置顶"时执行取消置顶，显示"置顶"时执行置顶
 */
async function batchPinSelected() {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) {
        nm.show('请先选择笔记', 'warning');
        return;
    }

    const pin = els.batchPinBtn.textContent === '置顶';

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
let batchTagNoteIds = null; // 右键菜单单笔记模式：显式笔记 ID 数组；null = 批量模式使用 selectedNoteIds
let batchTagAddLimit = null; // 单笔记添加模式：可再添加的标签数上限（3 - 笔记已有标签数）

/**
 * 收集指定笔记中已包含的标签 ID 集合
 */
function getTagIdsInNotes(noteIds) {
    const ids = new Set();
    const idSet = new Set(noteIds);
    for (const note of state.notes) {
        if (idSet.has(note.id) && note.tags) {
            note.tags.forEach(t => ids.add(t.id));
        }
    }
    return ids;
}

/**
 * 打开批量标签选择弹窗
 * @param {string} action - 'add' | 'remove'
 * @param {number[]} [noteIds] - 可选。右键菜单单笔记模式传入的笔记 ID 数组；缺省时使用批量选中的笔记
 */
function openBatchTagPicker(action, noteIds = null) {
    batchTagAction = action;
    batchTagNoteIds = noteIds ? [...noteIds] : null;
    if (!batchTagNoteIds) {
        if (state.selectedNoteIds.size === 0) {
            nm.show('请先选择笔记', 'warning');
            batchTagAction = null;
            return;
        }
    }
    const isAdd = action === 'add';
    // 单笔记模式标题更明确
    els.batchTagTitle.textContent = batchTagNoteIds && batchTagNoteIds.length === 1
        ? (isAdd ? '添加标签' : '移除标签')
        : (isAdd ? '批量添加标签' : '批量移除标签');

    // 单笔记添加模式：计算可再添加的标签数上限（最多 3 个）
    batchTagAddLimit = null;
    if (isAdd && batchTagNoteIds && batchTagNoteIds.length === 1) {
        const note = state.notes.find(n => n.id === batchTagNoteIds[0]);
        const existing = (note && note.tags) ? note.tags.length : 0;
        if (existing >= 3) {
            nm.show('该笔记标签已达上限（3 个）', 'info');
            batchTagAction = null;
            batchTagNoteIds = null;
            return;
        }
        batchTagAddLimit = 3 - existing;
    } else if (isAdd && !batchTagNoteIds) {
        // 批量添加模式：按所有选中笔记的最小剩余额度收紧，避免部分笔记超过 3 个标签
        let minRemaining = 3;
        for (const id of state.selectedNoteIds) {
            const note = state.notes.find(n => n.id === id);
            const existing = (note && note.tags) ? note.tags.length : 0;
            minRemaining = Math.min(minRemaining, 3 - existing);
            if (minRemaining <= 0) break;
        }
        if (minRemaining <= 0) {
            nm.show('所选笔记中有笔记已达 3 个标签上限', 'info');
            batchTagAction = null;
            return;
        }
        batchTagAddLimit = minRemaining;
    }

    // 移除模式：先检查笔记是否包含标签
    if (!isAdd) {
        const ids = batchTagNoteIds || Array.from(state.selectedNoteIds);
        if (getTagIdsInNotes(ids).size === 0) {
            nm.show(batchTagNoteIds ? '该笔记暂无标签' : '当前选中的笔记中没有可移除的标签', 'info');
            batchTagAction = null;
            batchTagNoteIds = null;
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
    batchTagNoteIds = null;
    batchTagAddLimit = null;
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
    const isAdd = batchTagAction === 'add';
    const tagIdsInNotes = isRemove ? getTagIdsInNotes(batchTagNoteIds || Array.from(state.selectedNoteIds)) : new Set();
    // 单笔记添加模式：该笔记已有的标签不可重复添加（禁用 + 提示，避免重复添加/白占可选项额度）
    const existingTagIds = (isAdd && batchTagNoteIds && batchTagNoteIds.length === 1)
        ? getTagIdsInNotes(batchTagNoteIds)
        : new Set();

    // 单笔记添加模式防御：剩余额度为 0（正常情况下在打开弹窗时已被拦截）
    if (isAdd && batchTagAddLimit !== null && batchTagAddLimit <= 0) {
        list.innerHTML = '<div class="batch-tag-empty">该笔记标签已满（最多 3 个）</div>';
        return;
    }

    list.innerHTML = state.tags
        .map(tag => {
            // 移除模式：不在笔记中的标签不可选；单笔记添加模式：已有标签不可选
            const disabled = (isRemove && !tagIdsInNotes.has(tag.id)) || existingTagIds.has(tag.id);
            const title = existingTagIds.has(tag.id) ? ' title="该笔记已有此标签"' : '';
            const cls = `batch-tag-chip${disabled ? ' disabled' : ''}`;
            return `<div class="${cls}"${title} data-tag-id="${tag.id}" data-tag-color="${tag.color || '#6B7280'}" style="--tag-color:${tag.color || '#6B7280'}">${escapeHtml(tag.name)}</div>`;
        })
        .join('');

    // 绑定点击事件
    list.querySelectorAll('.batch-tag-chip:not(.disabled)').forEach(el => {
        el.addEventListener('click', () => onBatchTagClick(el));
    });
}

/**
 * 点击批量标签芯片：切换选中态，更新确认按钮计数
 * 追加模式下最多选择 3 个标签（单笔记模式按剩余额度收紧）
 */
function onBatchTagClick(el) {
    const isAdd = batchTagAction === 'add';
    const limit = batchTagAddLimit !== null ? batchTagAddLimit : 3;
    // 追加模式下，如果芯片当前未选中且已选 ≥ 上限，拒绝
    if (isAdd && !el.classList.contains('selected')) {
        const count = els.batchTagList.querySelectorAll('.batch-tag-chip.selected').length;
        if (count >= limit) {
            window.showNotification('一篇笔记最多选择 3 个标签', 'warning');
            return;
        }
    }
    el.classList.toggle('selected');
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
    const ids = batchTagNoteIds ? [...batchTagNoteIds] : Array.from(state.selectedNoteIds);
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
        clearSelection();
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
    // 清除可能残留的离场标记
    menu.classList.remove('exiting');
    // 使用 CSS class 触发入场动画（moreMenuIn 由 CSS 控制）
    menu.classList.add('active');
    els.moreMenuBtn.classList.add('active');
}

/**
 * 关闭更多菜单（含离场动画）
 */
function closeMoreMenu(menu) {
    if (!menu.classList.contains('active')) return;
    menu.classList.add('exiting');
    els.moreMenuBtn.classList.remove('active');
    const onEnd = () => {
        menu.classList.remove('active', 'exiting');
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

    // 浮动 AI 按钮
    els.fabAI.addEventListener('click', () => {
        switchView('ai-chat');
    });

    // 浮动批量管理按钮
    els.fabBatch.addEventListener('click', () => {
        switchView('grid');
        toggleBatchMode();
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
                if (state.currentView === 'grid' && !state.searchKeyword) {
                    // 已在首页且无搜索：只平滑滚动到顶部，避免重建 DOM 引起闪烁
                    els.mainContent.scrollTo({ top: 0, behavior: 'smooth' });
                } else {
                    state.searchKeyword = '';
                    switchView('grid');
                    resetPagination();
                    loadNotes();
                }
            } else if (item.dataset.action === 'sidebar-toggle') {
                if (state.currentView !== 'grid') {
                    // 先跳转到笔记首页，再展开侧栏
                    state.searchKeyword = '';
                    switchView('grid');
                    resetPagination();
                    loadNotes();
                    // 非网格视图已将侧栏折叠，返回首页后将其展开
                    if (els.notebookSidebar?.classList.contains('collapsed')) {
                        els.notebookSidebar.classList.remove('collapsed');
                        localStorage.setItem('jot_sidebar_collapsed', 'false');
                        updateSidebarMenuItem();
                        updateNotebookSidebarToggleBtn();
                        // 此路径绕过 toggleSidebar，需手动刷新笔记本计数（否则展开后计数为旧值）
                        loadNotebooks();
                    }
                } else {
                    toggleSidebar();
                }
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
            } else if (item.dataset.action === 'calendar') {
                switchView('calendar');
            } else if (item.dataset.action === 'todo') {
                switchView('todo');
            } else if (item.dataset.action === 'password-manager') {
                switchView('password-manager');
            } else if (item.dataset.action === 'help') {
                openShortcuts();
            } else if (item.dataset.action === 'about') {
                showAbout();
            }
        }
    });

    // 点击品牌名返回所有笔记（与其他页面返回逻辑一致）
    document.querySelector('.topbar-brand')?.addEventListener('click', () => {
        state.searchKeyword = '';
        switchView('grid');
        loadNotes();
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
    initColorPresets();

    // 回收站按钮
    els.trashBackBtn.addEventListener('click', () => {
        switchView('grid');
        loadNotes();
    });
    els.restoreAllBtn.addEventListener('click', window.restoreAllNotes);
    els.emptyTrashBtn.addEventListener('click', window.emptyTrash);

    // 数据管理按钮
    els.dataBackBtn.addEventListener('click', () => {
        switchView('grid');
        loadNotes();
    });
    els.exportDataBtn.addEventListener('click', exportData);
    els.importDataBtn.addEventListener('click', importData);
    // 备份还原按钮
    els.backupBtn?.addEventListener('click', backupToDir);
    els.restoreBtn?.addEventListener('click', restoreFromDir);
    els.resetAllBtn.addEventListener('click', resetDatabase);
    els.vacuumDbBtn.addEventListener('click', vacuumDatabase);
    els.openDataDirBtn.addEventListener('click', openDataDir);
    els.openLogDirBtn?.addEventListener('click', openLogDir);
    if (els.logLevelControl && els.logLevelIndicator) {
        const moveLogLevelIndicator = (btn) => {
            const btns = Array.from(els.logLevelControl.querySelectorAll('.segmented-btn'));
            const index = btns.indexOf(btn);
            if (index >= 0) {
                const cw = els.logLevelControl.offsetWidth;
                const segW = (cw - 8) / btns.length;
                els.logLevelIndicator.style.transform = `translateX(${2 + index * segW}px)`;
                els.logLevelIndicator.style.width = `${segW}px`;
            }
        };
        els.logLevelControl.querySelectorAll('.segmented-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                // 切换 active 状态
                els.logLevelControl.querySelectorAll('.segmented-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                moveLogLevelIndicator(btn);
                // 保存设置
                await saveSettings();
                window.nm?.show?.('日志级别已保存', 'success');
            });
        });
    }
    els.clearAISessionsBtn.addEventListener('click', clearAISessions);
    els.clearCompletedTodosBtn.addEventListener('click', clearCompletedTodos);
    els.cleanupOrphanImagesBtn.addEventListener('click', cleanupOrphanImages);
    // AI 向量索引
    els.vectorIndexBtn?.addEventListener('click', openVectorIndexModal);
    els.deleteVectorsBtn?.addEventListener('click', deleteAllVectors);

    els.mdRefBackBtn.addEventListener('click', () => {
        switchView('grid');
        loadNotes();
    });

    els.todoBackBtn.addEventListener('click', () => {
        switchView('grid');
        loadNotes();
    });

    els.settingsBackBtn.addEventListener('click', () => {
        switchView('grid');
        loadNotes();
    });

    // 语法高亮开关
    els.mdHighlightToggle.addEventListener('change', async () => {
        await saveSettings();
        nm.show('设置已保存', 'success');
    });

    // 全屏打开笔记开关
    els.noteOpenFullscreenToggle.addEventListener('change', async () => {
        await saveSettings();
        nm.show('设置已保存', 'success');
    });

    // 自动换行开关
    els.editorWordWrapToggle.addEventListener('change', async () => {
        await saveSettings();
        nm.show('设置已保存', 'success');
    });

    // 右键菜单：点击其他区域关闭
    document.addEventListener('click', hideContextMenu);
    document.addEventListener('click', () => closeMoreMenu(els.moreMenu));
    // 右键菜单项点击：动作立即执行，回弹效果由 mousedown/mouseleave 提供，互不阻塞
    els.contextMenu.addEventListener('click', (e) => {
        const item = e.target.closest('.context-menu-item');
        if (item && item.dataset.action) {
            e.stopPropagation();
            // 置灰项不执行动作，仅提示原因
            if (item.classList.contains('disabled')) {
                if (item.dataset.action === 'add-tag') nm.show('该笔记标签已达上限（3 个）', 'info');
                else if (item.dataset.action === 'remove-tag') nm.show('该笔记暂无标签', 'info');
                return;
            }
            window.handleContextAction(item.dataset.action);
        }
    });
    // 右键菜单项按下反馈：按下瞬间缩小（0.06s），松开时经 spring 缓动弹回（0.18s）
    els.contextMenu.addEventListener('mousedown', (e) => {
        if (e.button !== 0) return; // 仅左键触发按压反馈
        const item = e.target.closest('.context-menu-item');
        if (item && !item.classList.contains('disabled')) {
            item.classList.add('pressed');
        }
    });
    // 鼠标移出菜单时清理按下态（如按下后拖到菜单外松开）
    els.contextMenu.addEventListener('mouseleave', () => {
        els.contextMenu.querySelectorAll('.pressed').forEach(el => el.classList.remove('pressed'));
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
        if (state.selectedNoteIds.size === 0) {
            nm.show('请先选择笔记', 'warning');
            return;
        }
        openMoveDialog([...state.selectedNoteIds]);
    });
    els.moveNotebookClose.addEventListener('click', closeMoveDialog);
    els.moveNotebookCancel.addEventListener('click', closeMoveDialog);
    els.moveNotebookConfirm.addEventListener('click', confirmMoveNotes);
    els.moveNotebookDialog.addEventListener('click', (e) => {
        if (e.target === els.moveNotebookDialog) closeMoveDialog();
    });

    // 关于页面入口已移至菜单“帮助参考 > 关于”，品牌名点击由父级 .topbar-brand 统一处理
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

    // 预览模式查找条事件（复用废弃的 editorFindBar 骨架，仅在预览模式下启用）
    const findInput = document.getElementById('findInput');
    const findPrevBtn = document.getElementById('findPrevBtn');
    const findNextBtn = document.getElementById('findNextBtn');
    const findCloseBtn = document.getElementById('findCloseBtn');
    if (findInput && findPrevBtn && findNextBtn && findCloseBtn) {
        findInput.addEventListener('input', () => {
            clearTimeout(_previewFindTimer);
            _previewFindTimer = setTimeout(() => runPreviewSearch(findInput.value), 150);
        });
        findInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                navigatePreviewMatch(e.shiftKey ? -1 : 1);
            }
        });
        findPrevBtn.addEventListener('click', () => navigatePreviewMatch(-1));
        findNextBtn.addEventListener('click', () => navigatePreviewMatch(1));
        findCloseBtn.addEventListener('click', closePreviewFindBar);
    }

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

    // 待办清单事件：FAB + 内联输入面板
    if (els.todoFab && els.todoFabPanel) {
        // FAB 点击切换面板
        els.todoFab.addEventListener('click', () => {
            if (els.todoFabPanel.classList.contains('open')) {
                closeTodoInputPanel();
            } else {
                openTodoInputPanel();
            }
        });

        // 面板内输入框键盘事件
        const panelInput = els.todoInput;
        if (panelInput) {
            panelInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !e.ctrlKey && !e.shiftKey) {
                    e.preventDefault();
                    addTodo();
                } else if (e.key === 'Enter' && e.ctrlKey) {
                    e.preventDefault();
                    const start = panelInput.selectionStart;
                    const end = panelInput.selectionEnd;
                    const val = panelInput.value;
                    panelInput.value = val.substring(0, start) + '\n' + val.substring(end);
                    panelInput.selectionStart = panelInput.selectionEnd = start + 1;
                    autoResizeTodoInput();
                }
            });

            // 输入时自动扩展高度
            panelInput.addEventListener('input', autoResizeTodoInput);
        }

        // 点击面板外部关闭
        document.addEventListener('click', (e) => {
            if (!els.todoFabPanel.classList.contains('open')) return;
            const target = e.target;
            if (!els.todoFabPanel.contains(target) && !els.todoFab.contains(target)) {
                closeTodoInputPanel();
            }
        });
    }

    // 清空按钮：按当前筛选分类清空对应范围的待办
    els.todoClearCompletedBtn?.addEventListener('click', clearTodosByFilter);

    // 事件委托：筛选按钮 + 待办项操作（checkbox、删除、编辑）
    els.viewTodo?.addEventListener('click', (e) => {
        // 筛选按钮切换
        const filterBtn = e.target.closest('.todo-filter-btn');
        if (filterBtn) {
            const filter = filterBtn.dataset.filter;
            if (!filter) return;
            _todoFilter = filter;
            window._todoFilter = _todoFilter;
            document.querySelectorAll('.todo-filter-btn').forEach(btn => btn.classList.remove('active'));
            filterBtn.classList.add('active');
            loadTodos();
            return;
        }

        // 待办项操作：通过 data-action 委托
        const actionEl = e.target.closest('[data-action]');
        if (!actionEl) return;
        const item = actionEl.closest('.todo-item');
        if (!item) return;
        const id = parseInt(item.dataset.id);
        const action = actionEl.dataset.action;

        if (action === 'toggle') {
            toggleTodo(id);
        } else if (action === 'delete') {
            deleteTodo(id);
        } else if (action === 'edit') {
            // 双击编辑由 dblclick 事件处理，这里防止单击干扰
        }
    });

    // 待办项双击编辑（事件委托）
    els.viewTodo?.addEventListener('dblclick', (e) => {
        const textEl = e.target.closest('.todo-text');
        if (!textEl) return;
        const item = textEl.closest('.todo-item');
        if (!item) return;
        const id = parseInt(item.dataset.id);
        editTodo(id);
    });

    // 待办项 tooltip 委托
    els.todoList?.addEventListener('mouseenter', (e) => {
        const item = e.target.closest('.todo-item');
        if (!item || item.classList.contains('editing')) return;
        const textEl = item.querySelector('.todo-text');
        if (!textEl) return;
        const text = textEl.textContent || '';
        if (!text) return;
        clearTimeout(todoTooltipTimer);
        todoTooltipTimer = setTimeout(() => {
            if (item.classList.contains('editing')) return;
            showTodoTooltip(item, text, e.clientX, e.clientY);
        }, 600);
    }, true);

    els.todoList?.addEventListener('mouseleave', () => {
        clearTimeout(todoTooltipTimer);
        hideTodoTooltip();
    }, true);

    // 搜索弹窗事件绑定(替代原 topbar 搜索框)
    initSearchModalListeners();

    // 全局拦截所有链接→系统默认浏览器打开（Wails WebView2 内避免应用内导航）
    document.addEventListener('click', function (e) {
        const link = e.target.closest('a');
        if (link && link.href && !link.getAttribute('href').startsWith('#') && !link.href.startsWith('javascript:')) {
            e.preventDefault();
            window.runtime.BrowserOpenURL(link.href);
        }
    });
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
 * 处理键盘快捷键（Ctrl/Cmd+Home/End, PgUp/PgDn, Ctrl/Cmd+F, Ctrl/Cmd+H）
 */
async function handleKeyboardNavigation(e) {
    const container = getScrollContainer();

    // Enter: 预设弹窗保存（需在 Ctrl 快捷键之前拦截）
    if (e.key === 'Enter') {
        // 确认框打开时（如预设弹窗"未保存修改"确认）忽略 Enter，避免误触发预设保存
        if (els.confirmDialog && els.confirmDialog.classList.contains('visible')) {
            return;
        }
        const presetOverlay = document.getElementById('presetModalOverlay');
        if (presetOverlay && presetOverlay.classList.contains('visible')) {
            e.preventDefault();
            savePresetModal();
            return;
        }
    }

    // Ctrl/Cmd+S: 编辑器内保存（编辑/新建模式有效，查看模式忽略）
    if ((e.ctrlKey || e.metaKey) && (e.key === 's' || e.key === 'S')) {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active') && els.editorSaveBtn.style.display !== 'none') {
            (state.editingNoteId ? updateNote(state.editingNoteId) : createNote());
        }
        return;
    }

    // Ctrl/Cmd+F: 编辑器内搜索（预览模式在预览渲染区直接搜索；编辑模式用 CM6 搜索并填充选中文本）;编辑器外则打开搜索弹窗
    if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active') && cmEditor) {
            // 预览模式：直接在预览渲染区搜索（不切回纯文本）
            if (els.editorOverlay.dataset.mode === 'preview') {
                openPreviewFindBar();
                return;
            }
            // 编辑模式：将当前选中文本填充到搜索框
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

    // Ctrl/Cmd+H: 编辑器内查找替换（仅在编辑模式生效）
    if ((e.ctrlKey || e.metaKey) && e.key === 'h') {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active') && cmEditor && els.editorOverlay.dataset.mode !== 'preview') {
            openSearchPanel(cmEditor);
        }
        return;
    }

    // Ctrl/Cmd+N: 打开新建笔记（编辑器未打开时）
    if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        if (!els.viewEditor.classList.contains('active')) {
            openEditor(null, false, getNoteOpenFullscreen());
        }
        return;
    }

    // Ctrl/Cmd+L: 编辑器打开时切换编辑/预览模式（仅 Markdown 模式支持）
    if ((e.ctrlKey || e.metaKey) && (e.key === 'l' || e.key === 'L') && els.viewEditor.classList.contains('active') && els.editorFileExt.textContent === '.md') {
        e.preventDefault();
        const current = els.editorOverlay.dataset.mode;
        switchEditorMode(current === 'preview' ? 'edit' : 'preview');
        return;
    }

    // Ctrl/Cmd+J: AI 助手侧栏折叠/展开（仅 AI 助手视图生效）
    if ((e.ctrlKey || e.metaKey) && (e.key === 'j' || e.key === 'J') && state.currentView === 'ai-chat') {
        e.preventDefault();
        if (typeof window.toggleAISessionSidebar === 'function') {
            window.toggleAISessionSidebar();
        }
        return;
    }

    // Ctrl/Cmd+E: 切换编辑器全屏模式（仅编辑器打开时有效）
    if ((e.ctrlKey || e.metaKey) && (e.key === 'e' || e.key === 'E')) {
        e.preventDefault();
        if (els.viewEditor.classList.contains('active')) {
            toggleEditorFullscreen();
        }
        return;
    }

    // Ctrl/Cmd+P: 打开/关闭启动器网格
    if ((e.ctrlKey || e.metaKey) && (e.key === 'p' || e.key === 'P')) {
        e.preventDefault();
        // 编辑器/查看器/新建页面打开时屏蔽启动器
        if (els.viewEditor.classList.contains('active')) return;
        const launcher = document.getElementById('launcher');
        if (launcher && launcher.classList.contains('visible')) {
            if (typeof window.closeLauncher === 'function') window.closeLauncher();
        } else {
            if (typeof window.openLauncher === 'function') window.openLauncher();
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

    // Ctrl/Cmd+Q: 退出程序（全局生效，退出前提示保存）
    if ((e.ctrlKey || e.metaKey) && (e.key === 'q' || e.key === 'Q')) {
        e.preventDefault();
        await handleAppExit();
        return;
    }

    // Escape: 关闭查找条或退出当前子视图
    if (e.key === 'Escape') {
        e.preventDefault();
        // 向量索引弹窗打开时：Esc 走其关闭逻辑（嵌入中确认停止，否则直接关闭）
        const viModal = document.getElementById('vectorIndexModal');
        if (viModal && viModal.style.display !== 'none') {
            onVectorIndexCloseRequested();
            return;
        }
        // 如果引用笔记选择器浮层打开，跳过全局 ESC 导航（由 ai-chat.js 处理关闭）
        const refModal = document.getElementById('aiNoteRefModal');
        if (refModal && refModal.style.display !== 'none') {
            return;
        }
        // 启动器打开时关闭它
        const launcherEl = document.getElementById('launcher');
        if (launcherEl && launcherEl.classList.contains('visible')) {
            if (typeof window.closeLauncher === 'function') window.closeLauncher();
            return;
        }
        // 搜索弹窗打开时关闭它（不继续执行导航逻辑）
        if (els.searchModal && els.searchModal.classList.contains('visible')) {
            closeSearchModal();
            return;
        }
        // 待办输入面板打开时关闭它
        if (els.todoFabPanel && els.todoFabPanel.classList.contains('open')) {
            closeTodoInputPanel();
            return;
        }
        // 确认框打开时忽略 ESC，避免破坏 Promise 链
        if (els.confirmDialog && els.confirmDialog.classList.contains('visible')) {
            return;
        }
        // 导入冲突弹窗打开时：ESC 终止导入并关闭弹窗
        const conflictOverlay = document.querySelector('.import-conflict-overlay');
        if (conflictOverlay && conflictOverlay.classList.contains('visible')) {
            conflictOverlay._onCancel?.();
            return;
        }
        // 密码管理：右键菜单/编辑/详情弹层打开时只关闭最上层弹层（不继续执行导航逻辑）
        if (typeof window.pmHandleEscape === 'function' && window.pmHandleEscape()) {
            return;
        }
        // MCP 服务器新增/编辑表单打开时关闭它
        const mcpFormDlg = document.getElementById('mcpServerFormDialog');
        if (mcpFormDlg && mcpFormDlg.classList.contains('visible')) {
            closeMCPServerForm();
            return;
        }
        // MCP 服务器导入对话框打开时关闭它
        const mcpImportDlg = document.getElementById('mcpServerImportDialog');
        if (mcpImportDlg && mcpImportDlg.classList.contains('visible')) {
            closeMCPImportDialog();
            return;
        }
        // 预设弹窗打开时关闭它（不继续执行导航逻辑）
        const presetOverlay = document.getElementById('presetModalOverlay');
        if (presetOverlay && presetOverlay.classList.contains('visible')) {
            closePresetModal();
            return;
        }
        // 密码弹窗打开时关闭它
        const pwdModal = document.getElementById('pwdModal');
        if (pwdModal && pwdModal.style.display !== 'none' && pwdModal.classList.contains('visible')) {
            pwdModal.classList.remove('visible');
            return;
        }
        // 关于页面打开时关闭它
        if (els.viewAbout.style.display === 'flex') {
            closeAbout();
            return;
        }
        // 灯箱打开时，ESC 不关闭笔记
        if (window.__lightboxOpen) {
            return;
        }
        // 如果编辑器处于全屏模式，先退出全屏
        if (els.editorPanel.classList.contains('fullscreen')) {
            toggleEditorFullscreen();
            return;
        }
        // CM6 搜索面板打开时，Esc 先关闭它（避免误关编辑器）
        if (cmEditor && searchPanelOpen(cmEditor.state)) {
            closeSearchPanel(cmEditor);
            return;
        }
        // 预览查找条打开时，先关闭它（避免 ESC 直接关掉编辑器）
        if (_previewFindBarVisible) {
            closePreviewFindBar();
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
        // 锁屏界面打开时屏蔽 Ctrl+数字快捷键
        const lockScreen = document.getElementById('lockScreen');
        if (lockScreen && lockScreen.style.display !== 'none') {
            return;
        }
        // 笔记编辑器/查看器/新建页面打开时屏蔽 Ctrl+数字快捷键
        const viewEditor = document.getElementById('viewEditor');
        const viewPreview = document.getElementById('viewPreview');
        if ((viewEditor && viewEditor.classList.contains('active')) || 
            (viewPreview && viewPreview.classList.contains('active'))) {
            return;
        }
        // Ctrl+0: 锁屏（需先在设置中启用）
        if (e.key === '0') {
            e.preventDefault();

            // 获取设置，检查锁屏状态
            const lockCfg = await window.go.main.App.GetAllSettings();
            const lockEnabled = lockCfg.screen_lock_enabled === true || lockCfg.screen_lock_enabled === 'true';
            const hasPassword = lockCfg.screen_lock_password && lockCfg.screen_lock_password !== '';

            // 未启用锁屏 → 提示
            if (!lockEnabled) {
                nm.show('请先在「设置 → 锁屏密码」中启用锁屏功能', 'warning');
                return;
            }

            // 已启用但无密码 → 提示
            if (!hasPassword) {
                nm.show('请先设置锁屏密码后再使用锁屏功能', 'warning');
                return;
            }

            // 已启用且有密码 → 切首页 → 锁屏
            switchView('grid');
            await loadNotes();

            const lockScreen = document.getElementById('lockScreen');
            if (!lockScreen) return;
            lockScreen.style.display = 'flex';
            // 清空输入框，防止异步期间残余键盘事件填充字符
            const lockPwdInput = document.getElementById('lockPasswordInput');
            if (lockPwdInput) lockPwdInput.value = '';
            requestAnimationFrame(() => lockScreen.classList.add('entering'));
            setTimeout(() => lockScreen.classList.remove('entering'), 700);
            setTimeout(() => lockPwdInput?.focus(), 100);
            return;
        }
    }

    // 编辑器打开时，Ctrl/Cmd+Home/End 和 PgUp/PgDn 交由编辑器/textarea 原生处理
    if (els.viewEditor.classList.contains('active') &&
        (((e.ctrlKey || e.metaKey) && (e.key === 'Home' || e.key === 'End')) || e.key === 'PageUp' || e.key === 'PageDown')) {
        return;
    }

    if (!container) return;

    // Ctrl/Cmd+Home: 滚动到顶部
    if ((e.ctrlKey || e.metaKey) && e.key === 'Home') {
        e.preventDefault();
        container.scrollTop = 0;
        return;
    }
    // Ctrl/Cmd+End: 加载所有剩余页后跳到底部
    if ((e.ctrlKey || e.metaKey) && e.key === 'End') {
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
    const containers = [els.mainContent, document.querySelector('.ai-chat-messages'), document.querySelector('.settings-panels'), document.querySelector('.data-panels')].filter(Boolean);
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
    // 主内容区滚动时控制 FAB 组位置和 "↑" 按钮显隐（阈值 300px）
    if (els.mainContent) {
        els.mainContent.addEventListener('scroll', () => {
            const scrollY = els.mainContent.scrollTop;
            const isScrolled = scrollY > 300;
            els.fabGroup.classList.toggle('scrolled', isScrolled);
            els.backToTopBtn.classList.toggle('visible', isScrolled);
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
    // 重置滚动位置到顶部
    els.shortcutsBody.scrollTop = 0;
    // 渲染快捷键列表
    renderShortcutsPage();
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
        // 重置遮罩和卡片的动画
        els.shortcutsView.style.animation = '';
        card.style.animation = '';
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
        { key: 'Ctrl + P', desc: '打开启动器菜单' },
        { key: 'Ctrl + E', desc: '编辑器内切换全屏' },
        { key: 'F11', desc: '切换窗口全屏' },
        { key: 'Ctrl + Q', desc: '退出程序' },
        { key: 'PgUp', desc: '上翻一页' },
        { key: 'PgDn', desc: '下翻一页 / 触底加载更多' },
        { key: 'Ctrl + Home', desc: '回到顶部' },
        { key: 'Ctrl + End', desc: '加载全部并滚到底部' },
        { key: 'Escape', desc: '关闭弹窗 / 返回上一页' },

        { key: 'Ctrl + 0', desc: '锁屏（需先在设置中启用）' },
        { key: 'Ctrl + J', desc: 'AI 侧栏折叠/展开' },
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
        // 双击重命名
        const nameEl = item.querySelector('.notebook-name');
        if (nameEl) {
            nameEl.addEventListener('dblclick', (e) => {
                e.stopPropagation();
                const id = parseInt(item.dataset.notebookId);
                showRenameNotebookDialog(id, item.dataset.notebookName);
            });
        }
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
    overlay.tabIndex = -1;
    overlay.innerHTML = `
        <div class="new-notebook-dialog">
            <div class="new-notebook-title">新建笔记本</div>
            <input type="text" class="new-notebook-input" id="newNotebookInput" placeholder="输入笔记本名称..." autofocus>
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

    /** 关闭弹窗并清理（async，有未保存内容时弹确认） */
    const close = async (force) => {
        if (!force && input.value.trim()) {
            const confirmed = await showConfirmDialog('内容尚未保存，确定关闭？', '关闭', '继续编辑');
            if (!confirmed) { overlay.focus(); return; }
        }
        overlay.classList.remove('visible');
        setTimeout(() => overlay.remove(), 200);
    };

    /** 执行创建 */
    const doCreate = async () => {
        const name = input.value.trim();
        if (!name) {
            input.classList.add('shake');
            setTimeout(() => input.classList.remove('shake'), 400);
            nm.show('请输入笔记本名称', 'warning');
            return;
        }
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateNotebook) {
                const notebook = await window.go.main.App.CreateNotebook(name);
                if (notebook) {
                    close(true);
                    await loadNotebooks();
                    state.activeNotebookId = notebook.id;
                    resetPagination();
                    await loadNotes();
                    renderNotebookList();
                    nm.show('笔记本已创建', 'success');
                }
            } else {
                console.warn('CreateNotebook 未绑定');
                close(true);
            }
        } catch (err) {
            const msg = (typeof err === 'string' ? err : err?.message || '创建笔记本失败');
            console.error('创建笔记本失败:', msg);
            nm.show(msg, 'error');
        }
    };

    input.addEventListener('input', () => {
        const runes = [...input.value];
        if (runes.length > 50) {
            input.value = runes.slice(0, 50).join('');
            input.classList.add('shake');
            setTimeout(() => input.classList.remove('shake'), 400);
            nm.show('名称不能超过50个字符', 'warning');
        }
    });
    confirmBtn.addEventListener('click', doCreate);
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') doCreate();
    });
    overlay.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') { e.stopPropagation(); close(); }
    });
    cancelBtn.addEventListener('click', () => close());
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

    // 按下回弹反馈：mousedown 缩小，鼠标移出清理（动作零延迟）
    menu.addEventListener('mousedown', (e) => {
        if (e.button !== 0) return; // 仅左键触发按压反馈
        const item = e.target.closest('.notebook-context-item');
        if (item && !item.classList.contains('disabled')) {
            item.classList.add('pressed');
        }
    });
    menu.addEventListener('mouseleave', () => {
        menu.querySelectorAll('.pressed').forEach(el => el.classList.remove('pressed'));
    });

    // 点击其他地方关闭
    const closeMenu = (e2) => {
        if (!menu.contains(e2.target)) {
            closeNotebookMenu();
        }
    };

    // 关闭笔记本右键菜单：先淡出再移除，给回弹留出可见时间（防重复关闭）
    const closeNotebookMenu = () => {
        if (menu._closing) return;
        menu._closing = true;
        menu.querySelectorAll('.pressed').forEach(el => el.classList.remove('pressed'));
        menu.style.pointerEvents = 'none'; // 淡出期间禁止再点击，防重复触发动作
        menu.style.opacity = '0';
        setTimeout(() => {
            menu.remove();
            document.removeEventListener('click', closeMenu);
        }, 130);
    };

    // 点击菜单项
    menu.addEventListener('click', async (e) => {
        const item = e.target.closest('.notebook-context-item');
        if (!item || item.classList.contains('disabled')) return;
        const action = item.dataset.action;
        closeNotebookMenu();

        if (action === 'rename') {
            showRenameNotebookDialog(notebookId, notebookName);
        } else if (action === 'delete') {
            showDeleteNotebookDialog(notebookId, notebookName);
        }
    });

    setTimeout(() => document.addEventListener('click', closeMenu), 0);
}

/**
 * 弹出对话框重命名笔记本
 */
let _renameNotebookDialog = null;

function closeRenameDialog() {
    if (_renameNotebookDialog) {
        _renameNotebookDialog.classList.remove('visible');
        const el = _renameNotebookDialog;
        setTimeout(() => el.remove(), 200);
        _renameNotebookDialog = null;
    }
}

function showRenameNotebookDialog(notebookId, currentName) {
    closeRenameDialog();

    const overlay = document.createElement('div');
    overlay.className = 'new-notebook-overlay';
    overlay.tabIndex = -1;
    overlay.innerHTML = `
        <div class="new-notebook-dialog">
            <div class="new-notebook-title">重命名笔记本</div>
            <input type="text" class="new-notebook-input" id="renameNotebookInput" value="${escapeHtml(currentName)}" autofocus>
            <div class="new-notebook-actions">
                <button class="btn btn-cancel" id="renameNotebookCancelBtn">取消</button>
                <button class="btn btn-save" id="renameNotebookConfirmBtn">确认</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
    _renameNotebookDialog = overlay;

    const input = overlay.querySelector('#renameNotebookInput');
    const confirmBtn = overlay.querySelector('#renameNotebookConfirmBtn');
    const cancelBtn = overlay.querySelector('#renameNotebookCancelBtn');

    requestAnimationFrame(() => {
        overlay.classList.add('visible');
        input.focus();
        input.select();
    });

    const close = async (force) => {
        if (!force && input.value.trim() !== currentName.trim()) {
            const confirmed = await showConfirmDialog('内容尚未保存，确定关闭？', '关闭', '继续编辑');
            if (!confirmed) { overlay.focus(); return; }
        }
        _renameNotebookDialog = null;
        overlay.classList.remove('visible');
        setTimeout(() => overlay.remove(), 200);
    };

    const doRename = async () => {
        const newName = input.value.trim();
        if (!newName) {
            input.classList.add('shake');
            setTimeout(() => input.classList.remove('shake'), 400);
            nm.show('请输入笔记本名称', 'warning');
            return;
        }
        if (newName === currentName) {
            close(true);
            return;
        }
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.RenameNotebook) {
                await window.go.main.App.RenameNotebook(notebookId, newName);
                close(true);
                await loadNotebooks();
                nm.show('笔记本已重命名', 'success');
            }
        } catch (err) {
            const msg = (typeof err === 'string' ? err : err?.message || '重命名失败');
            console.error('重命名失败:', msg);
            nm.show(msg, 'error');
        }
    };

    input.addEventListener('input', () => {
        const runes = [...input.value];
        if (runes.length > 50) {
            input.value = runes.slice(0, 50).join('');
            input.classList.add('shake');
            setTimeout(() => input.classList.remove('shake'), 400);
            nm.show('名称不能超过50个字符', 'warning');
        }
    });
    confirmBtn.addEventListener('click', doRename);
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') doRename();
    });
    overlay.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') { e.stopPropagation(); close(); }
    });
    cancelBtn.addEventListener('click', () => close());
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close();
    });
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
            // 保持"不保存"按钮隐藏（仅三选一对话框使用）
            if (els.confirmThirdBtn) els.confirmThirdBtn.style.display = 'none';
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
    if (els.searchModalEmptyDesc) els.searchModalEmptyDesc.textContent = '输入关键字搜索标题或内容';
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
        // 摘要(后端已围绕关键词智能截取约200字符，仅做空白归一化)
        const content = note.content !== undefined ? note.content : (note.Content || '');
        const snippet = String(content || '').replace(/\s+/g, ' ').trim();
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
        els.searchModalTagLabel.textContent = (t && t.name) || '';
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
            text: tag.name || '',
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
    // 待办 FAB 默认隐藏（初始视图为笔记网格）
    if (els.todoFab) els.todoFab.classList.add('fab-hidden');
    if (els.todoFabPanel) els.todoFabPanel.classList.add('fab-hidden');
    initEventListeners();
    initMCPServerSettings();
    initLockScreenEvents();
    initEditorActionsMenu();
    initFontSettings();
    buildThemeDropdown();
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
    // 并行加载设置与笔记本列表（互不依赖），缩短首屏空白窗口；各自内部已有 try/catch，外层再兜底防止 reject 中断 init
    await Promise.all([loadSettings().catch(() => {}), loadNotebooks().catch(() => {})]);
    await initSortSettings();
    initAISettings();
    // 先恢复侧栏折叠状态
    restoreSidebarState();
    // 确保 activeNotebookId 有值（默认为 1）
    if (!state.activeNotebookId && state.notebooks.length > 0) {
        state.activeNotebookId = state.notebooks[0].id;
    }
    // 笔记与标签并行加载（互不依赖）
    await Promise.all([loadNotes(), loadTags()]);
    // 初始化预览渲染 Worker
    initPreviewWorker();
    // 初始化 TOC 侧栏展开/折叠按钮
    _initTocToggle();
    // 初始化无边框窗口控制
    initWindowControls();
    // 注册文件拖拽导入
    initFileDrop();
    // 初始化 AI 对话页面
    initAIChat();
    // 初始化日历视图
    initCalendarView();
    // 初始化启动器网格
    initLauncher();
    // 初始化密码管理视图
    initPasswordManager();
    // --- 锁屏密码检查 ---
    await checkScreenLock();
    // --- 未完成待办启动提示 ---
    checkUnfinishedTodosReminder();
}

/**
 * 启动时提示未完成待办：锁屏启用时等解锁后延迟弹出，否则直接延迟弹出；每次启动仅一次
 */
async function checkUnfinishedTodosReminder() {
    try {
        // 绑定未就绪时静默跳过（与 checkScreenLock 同款 guard）
        if (!window.go?.main?.App?.GetAllSettings) return;
        // 读锁屏配置，判断是否启用（与 checkScreenLock 同源）
        const cfg = await window.go.main.App.GetAllSettings();
        const lockEnabled = cfg.screen_lock_enabled === true || cfg.screen_lock_enabled === 'true';

        const show = async () => {
            // 解锁后/无锁屏时延迟，等界面渲染稳定
            setTimeout(async () => {
                try {
                    // 查询未完成待办数量
                    const count = await window.go.main.App.CountUnfinishedTodos();
                    if (!count || count <= 0) return;
                    // 弹窗：去查看 → 跳转待办视图
                    const go = await showConfirmDialog(`你有 ${count} 个未完成的待办事项，是否现在去查看？`, '去查看', '取消');
                    if (go) switchView('todo');
                } catch (e) {
                    console.warn('未完成待办提示失败:', e);
                }
            }, lockEnabled ? 1000 : 600);
        };

        if (lockEnabled) {
            // 等解锁成功后延迟弹出；用一次性监听，避免重复注册
            const onUnlocked = () => {
                document.removeEventListener('app-unlocked', onUnlocked);
                show();
            };
            document.addEventListener('app-unlocked', onUnlocked);
        } else {
            show();
        }
    } catch (e) {
        console.warn('未完成待办提示检查失败:', e);
    }
}

/**
 * 检查是否需要显示锁屏
 */
async function checkScreenLock() {
    try {
        if (!window.go?.main?.App?.GetAllSettings) return;
        const cfg = await window.go.main.App.GetAllSettings();
        const enabled = cfg.screen_lock_enabled === true || cfg.screen_lock_enabled === 'true';
        if (!enabled) return;

        const lockScreen = document.getElementById('lockScreen');
        if (!lockScreen) return;

        // 延迟短暂时间后显示锁屏（等待UI完全渲染）
        setTimeout(() => {
            lockScreen.style.display = 'flex';
            requestAnimationFrame(() => {
                lockScreen.classList.add('entering');
            });
            // 入场动画约 650ms 后清除 entering class（最长 stagger 0.25s+0.4s）
            setTimeout(() => {
                lockScreen.classList.remove('entering');
            }, 700);
            const input = document.getElementById('lockPasswordInput');
            if (input) setTimeout(() => input.focus(), 100);
        }, 100);
    } catch (e) {
        console.warn('锁屏检查失败:', e);
    }
}

/**
 * 执行解锁操作
 */
async function unlockApp() {
    const input = document.getElementById('lockPasswordInput');
    const lockScreen = document.getElementById('lockScreen');
    const errorMsg = document.getElementById('lockErrorMsg');
    const unlockBtn = document.getElementById('lockUnlockBtn');
    const lockIcon = document.querySelector('.lock-screen-icon');

    if (!input || !lockScreen) return;

    const password = input.value.trim();
    if (!password) {
        input.classList.add('shake');
        if (lockIcon) lockIcon.classList.add('error-shake');
        if (errorMsg) {
            errorMsg.textContent = '请输入密码';
            errorMsg.style.display = '';
        }
        setTimeout(() => {
            input.classList.remove('shake');
            if (lockIcon) lockIcon.classList.remove('error-shake');
            input.focus();
        }, 750);
        setTimeout(() => {
            if (errorMsg) errorMsg.style.display = 'none';
        }, 800);
        return;
    }

    // 禁用按钮，防止重复点击
    if (unlockBtn) unlockBtn.disabled = true;
    if (errorMsg) errorMsg.style.display = 'none';

    try {
        const ok = await window.go.main.App.VerifyScreenLockPassword(password);
        if (ok) {
            // 解锁成功 - 内容向上飘散 + 模糊渐隐
            lockScreen.classList.add('exit');
            setTimeout(() => {
                lockScreen.style.display = 'none';
                lockScreen.classList.remove('exit');
                if (unlockBtn) unlockBtn.disabled = false;
                // 通知启动期延迟任务（如未完成待办提示）可以弹出了
                document.dispatchEvent(new CustomEvent('app-unlocked'));
            }, 500);
        } else {
            // 解锁失败 - 抖动动画 + 提示
            input.classList.add('shake');
            if (lockIcon) lockIcon.classList.add('error-shake');
            if (errorMsg) {
                errorMsg.textContent = '密码错误，请重试';
                errorMsg.style.display = '';
            }
            input.value = '';
            if (unlockBtn) unlockBtn.disabled = false;

            // 动画结束后移除 error-shake，0.9 秒后隐藏提示
            setTimeout(() => {
                input.classList.remove('shake');
                if (lockIcon) lockIcon.classList.remove('error-shake');
                input.focus();
            }, 750);
            setTimeout(() => {
                if (errorMsg) errorMsg.style.display = 'none';
            }, 800);
        }
    } catch (e) {
        console.error('验证密码失败:', e);
        if (unlockBtn) unlockBtn.disabled = false;
    }
}

/**
 * 初始化锁屏交互事件
 */
function initLockScreenEvents() {
    const input = document.getElementById('lockPasswordInput');
    const unlockBtn = document.getElementById('lockUnlockBtn');
    const toggleBtn = document.getElementById('lockPasswordToggle');
    const lockIcon = document.querySelector('.lock-screen-icon');

    // 输入聚焦时锁子注视效果
    if (input && lockIcon) {
        input.addEventListener('focus', () => lockIcon.classList.add('focused'));
        input.addEventListener('blur', () => lockIcon.classList.remove('focused'));
    }

    // 回车触发解锁
    if (input) {
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') unlockApp();
        });
        // 输入时清除错误提示
        input.addEventListener('input', () => {
            const errorMsg = document.getElementById('lockErrorMsg');
            if (errorMsg) errorMsg.style.display = 'none';
        });
    }

    // 点击解锁按钮
    if (unlockBtn) {
        unlockBtn.addEventListener('click', unlockApp);
    }

    // 退出应用按钮
    const quitBtn = document.getElementById('lockQuitBtn');
    if (quitBtn) {
        quitBtn.addEventListener('click', () => {
            Quit();
        });
    }

    // 显示/隐藏密码
    if (toggleBtn && input) {
        toggleBtn.addEventListener('click', () => {
            const isPassword = input.type === 'password';
            input.type = isPassword ? 'text' : 'password';
            toggleBtn.querySelector('.toggle-eye').style.display = isPassword ? 'none' : '';
            toggleBtn.querySelector('.toggle-eye-off').style.display = isPassword ? '' : 'none';
        });
    }
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
let _importing = false;
function initFileDrop() {
    const dropOverlay = document.getElementById('dropOverlay');
    let registered = false;
    if (registered) return;
    registered = true;

    document.addEventListener('dragenter', (e) => {
        e.preventDefault();
        if (!e.dataTransfer.types.includes('Files')) return;

        // AI 流式回复期间不显示全局拖拽遮罩（文件上传被禁用）
        if (window.__aiStreaming) return;

        // 在 AI 聊天区域内拖拽时不操作全局遮罩（由 ai-chat.js 自行处理）
        if (e.target.closest('.ai-chat-content')) return;

        _dragCounter++;

        // 编辑器拖拽悬停视觉反馈
        if (_dragCounter === 1) {
            // 检查是否悬停到 CM6 编辑器上
            const cmEl = document.querySelector('.cm-editor');
            if (cmEl && cmEl.matches(':hover')) {
                cmEl.classList.add('dragover');
            }
        }
        if (_dragCounter === 1 && dropOverlay) {
            dropOverlay.classList.add('active');
            if (_importing) {
                dropOverlay.classList.add('disabled');
                const p = dropOverlay.querySelector('p');
                if (p) p.textContent = '导入进行中，请稍候';
            }
        }
    });

    document.addEventListener('dragover', (e) => {
        e.preventDefault();
        // 拖拽过程中动态检测是否在编辑器上方，切换 dragover 类
        const cmEl = document.querySelector('.cm-editor');
        if (cmEl) {
            const overEditor = cmEl.matches(':hover');
            cmEl.classList.toggle('dragover', overEditor);
        }
    });

    document.addEventListener('dragleave', (e) => {
        e.preventDefault();

        // AI 流式回复期间不操作全局拖拽遮罩
        if (window.__aiStreaming) return;

        // 在 AI 聊天区域内拖拽时不操作全局遮罩（由 ai-chat.js 自行处理）
        if (e.target.closest('.ai-chat-content')) return;

        _dragCounter--;
        if (_dragCounter <= 0) {
            _dragCounter = 0;
            if (dropOverlay) {
                dropOverlay.classList.remove('active');
                dropOverlay.classList.remove('disabled');
                const p = dropOverlay.querySelector('p');
                if (p) p.textContent = '释放以导入文件';
            }
            // 移除编辑器悬停样式
            const cmEl = document.querySelector('.cm-editor');
            if (cmEl) cmEl.classList.remove('dragover');
        }
    });

    // HTML5 drop 仅重置遮罩，不处理文件（由 OnFileDrop 接手）
    document.addEventListener('drop', (e) => {
        e.preventDefault();

        // AI 流式回复期间不操作全局拖拽遮罩
        if (window.__aiStreaming) return;

        // 在 AI 聊天区域内拖拽时不操作全局遮罩（由 ai-chat.js 自行处理）
        if (e.target.closest('.ai-chat-content')) return;

        _dragCounter = 0;
        if (dropOverlay) {
            dropOverlay.classList.remove('active');
            dropOverlay.classList.remove('disabled');
            const p = dropOverlay.querySelector('p');
            if (p) p.textContent = '释放以导入文件';
        }
        const cmEl = document.querySelector('.cm-editor');
        if (cmEl) cmEl.classList.remove('dragover');
    });

    // Wails OnFileDrop：OS 级拦截，直接返回文件路径
    // 回调签名：(x, y, paths) — x/y 为释放坐标，paths 为文件路径数组
    if (window.runtime && window.runtime.OnFileDrop) {
        console.log('[拖拽] 注册 OnFileDrop 回调 (useDropTarget=false)');
        window.runtime.OnFileDrop(async (x, y, paths) => {
            console.log('[拖拽] OnFileDrop 触发, paths:', paths);
            // 确保遮罩已隐藏
            _dragCounter = 0;
            if (dropOverlay) {
                dropOverlay.classList.remove('active');
                dropOverlay.classList.remove('disabled');
                const p = dropOverlay.querySelector('p');
                if (p) p.textContent = '释放以导入文件';
            }
            const cmEl = document.querySelector('.cm-editor');
            if (cmEl) cmEl.classList.remove('dragover');
            if (!paths || paths.length === 0) return;

            // 判断释放位置是否在 AI 聊天内容区
            const target = document.elementFromPoint(x, y);
            const aiChatContent = target?.closest('.ai-chat-content');
            if (aiChatContent) {
                if (typeof window.handleAiChatFileDrop === 'function') {
                    await window.handleAiChatFileDrop(paths);
                }
                return;
            }

            // 编辑器打开时（任何模式）全局禁止通过拖拽创建笔记
            if (cmEditor !== null) {
                // 判断释放位置是否在 CM6 编辑器上
                const target = document.elementFromPoint(x, y);
                const cmEditorEl = target?.closest('.cm-editor');
                if (!cmEditorEl) return; // 拖到编辑器外 → 忽略

                // 查看模式（只读）→ 忽略
                if (els.editorNoteTitle.readOnly) return;

                // 检查是否为 .md 笔记（仅 .md 支持图片拖拽）
                const isMd = els.editorFileExt.textContent === '.md';

                // 编辑/新建模式 → 区分文件类型处理
                const imgPaths = isMd
                    ? paths.filter(p => /\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(p))
                    : [];
                const textPaths = paths.filter(p => !/\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(p));

                // 非 .md 笔记拖拽图片时提示
                if (!isMd && paths.some(p => /\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(p))) {
                    window.showNotification?.('图片拖拽仅支持 .md 格式笔记', 'info');
                }

                // 处理图片：上传 + 插入 Markdown（缓存光标位置，依次往后插）
                let pos = cmEditor.state.selection.main.head;
                let hasInsert = false;
                for (const p of imgPaths) {
                    try {
                        const url = await window.go.main.App.SaveImageFromPath(p);
                        const filename = p.split(/[/\\]/).pop();
                        const markdown = `![${filename}](${url})`;
                        cmEditor.dispatch({
                            changes: { from: pos, insert: markdown },
                            selection: { anchor: pos + markdown.length, head: pos + markdown.length }
                        });
                        pos += markdown.length;
                        hasInsert = true;
                    } catch (err) {
                        console.error('拖拽上传图片失败:', p, err);
                    }
                }

                // 处理文本文件：读取内容并插入光标处
                for (const p of textPaths) {
                    try {
                        const content = await window.go.main.App.ReadTextFile(p);
                        cmEditor.dispatch({
                            changes: { from: pos, insert: content },
                            selection: { anchor: pos + content.length, head: pos + content.length }
                        });
                        pos += content.length;
                        hasInsert = true;
                    } catch (err) {
                        // 二进制文件或不支持的文件 → 忽略
                        console.log('拖拽文件已忽略（非文本或二进制）:', p);
                    }
                }

                if (hasInsert) cmEditor.focus();
            } else {
                // 编辑器未打开 → 走原有笔记导入
                handleFileDropPaths(paths, state.activeNotebookId);
            }
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

// 展示导入结果通知（成功/失败详情），并刷新 UI
function showImportResults(results, onComplete, onDone) {
    let successCount = 0;
    let updatedCount = 0;
    let skippedCount = 0;
    const failItems = [];
    const importedNoteIds = [];
    const conflicts = [];

    for (const result of results) {
        const label = result.path ? result.path.split(/[/\\]/).pop() || '文件' : '文件';
        if (result.status === 'conflict') {
            conflicts.push(result);
        } else if (result.success) {
            if (result.status === 'updated') {
                updatedCount++;
                importedNoteIds.push(result.note_id);
            } else if (result.status === 'skipped') {
                skippedCount++;
            } else {
                successCount++;
                importedNoteIds.push(result.note_id);
            }
        } else {
            failItems.push({ label, error: result.error || '导入失败' });
        }
    }

    const failCount = failItems.length;
    const conflictCount = conflicts.length;
    const parts = [];
    if (successCount > 0) parts.push(`新建 ${successCount} 个`);
    if (updatedCount > 0) parts.push(`覆盖 ${updatedCount} 个`);
    if (skippedCount > 0) parts.push(`跳过 ${skippedCount} 个`);
    if (conflictCount > 0) parts.push(`冲突 ${conflictCount} 个`);
    if (failCount > 0) parts.push(`失败 ${failCount} 个`);

    if (conflicts.length > 0) {
        // 有冲突时先弹窗处理冲突，完成后再显示结果通知
        showImportConflictDialog(conflicts, (result) => {
            if (result === false) {
                // 用户取消了冲突弹窗（ESC 或点击遮罩）
                const cancelParts = [];
                if (successCount > 0) cancelParts.push(`新建 ${successCount} 个`);
                if (updatedCount > 0) cancelParts.push(`覆盖 ${updatedCount} 个`);
                if (conflictCount > 0) cancelParts.push(`冲突 ${conflictCount} 个`);
                const msg = `导入已取消${cancelParts.length > 0 ? ': ' + cancelParts.join(', ') : ''}`;
                nm.show(msg, 'warning', 3000);
                if (successCount > 0 || updatedCount > 0) {
                    loadNotes().then(() => {
                        loadNotebooks();
                        flashNoteCards(importedNoteIds);
                    });
                }
                if (onComplete) onComplete();
                if (onDone) onDone();
                return;
            }
            // result === null，正常完成 — 统计冲突处理结果
            for (const r of (result || [])) {
                if (r.status === 'updated') {
                    updatedCount++;
                    importedNoteIds.push(r.note_id);
                } else if (r.status === 'skipped') {
                    skippedCount++;
                }
            }
            // 重新组装通知
            const finalParts = [];
            if (successCount > 0) finalParts.push(`新建 ${successCount} 个`);
            if (updatedCount > 0) finalParts.push(`覆盖 ${updatedCount} 个`);
            if (skippedCount > 0) finalParts.push(`跳过 ${skippedCount} 个`);
            if (failCount > 0) finalParts.push(`失败 ${failCount} 个`);
            const msg = `导入完成: ${finalParts.join(', ')}`;
            nm.show(msg, failCount > 0 ? 'warning' : 'success', failCount > 0 ? 8000 : 3000);
            if (importedNoteIds.length > 0) {
                loadNotes().then(() => {
                    loadNotebooks();
                    flashNoteCards(importedNoteIds);
                });
            }
            if (onComplete) onComplete();
            if (onDone) onDone();
        });
        return;
    }

    // 无冲突，直接显示结果
    let msg = `导入完成: ${parts.join(', ')}`;
    nm.show(msg, failCount > 0 ? 'warning' : 'success', failCount > 0 ? 8000 : 3000);

    if (successCount > 0 || updatedCount > 0) {
        loadNotes().then(() => {
            loadNotebooks();
            flashNoteCards(importedNoteIds);
        });
    }
    if (onComplete) onComplete();
    if (onDone) onDone();
}

/**
 * 显示导入冲突解决弹窗
 * @param {Array} conflicts - 冲突结果列表（含 note_id, title, content, file_ext, file_time, note_time, path）
 * @param {Function} onComplete - 全部处理完成后的回调，参数为处理结果数组
 */
function showImportConflictDialog(conflicts, onComplete) {
    const resolved = [];
    let items = [...conflicts];

    // 创建弹窗 DOM
    const overlay = document.createElement('div');
    overlay.className = 'import-conflict-overlay';
    overlay.innerHTML = `
        <div class="import-conflict-dialog">
            <div class="import-conflict-header">
                <h3>发现 ${items.length} 个冲突文件</h3>
                <div class="import-conflict-actions">
                    <button class="import-conflict-batch-btn" data-action="overwrite-all" title="用所有导入文件的内容替换对应笔记">全部覆盖</button>
                    <button class="import-conflict-batch-btn import-conflict-batch-skip" data-action="skip-all" title="跳过所有冲突文件，笔记内容保持不变">全部跳过</button>
                </div>
            </div>
            <div class="import-conflict-list"></div>
        </div>
    `;

    const listEl = overlay.querySelector('.import-conflict-list');

    function renderItems() {
        listEl.innerHTML = '';
        if (items.length === 0) {
            // 全部处理完，关闭弹窗并传递处理结果
            close(false, resolved);
            return;
        }
        for (const item of items) {
            const fileName = item.path ? item.path.split(/[/\\]/).pop() || '文件' : '文件';
            const fileDate = new Date(item.file_time * 1000).toLocaleString('zh-CN');
            const noteDate = new Date(item.note_time * 1000).toLocaleString('zh-CN');
            const el = document.createElement('div');
            el.className = 'import-conflict-item';
            el.innerHTML = `
                <div class="import-conflict-item-info">
                    <div class="import-conflict-item-title">
                        ${(!item.content || item.content.trim() === '') ? '<span class="import-conflict-empty-badge" title="导入文件内容为空，覆盖将清空笔记">空文件</span>' : ''}
                        ${escapeHtml(item.title)}
                    </div>
                    <div class="import-conflict-item-detail">
                        <span>笔记: ${noteDate}</span>
                        <span>文件: ${fileDate}</span>
                    </div>
                </div>
                <div class="import-conflict-item-actions">
                    <button class="import-conflict-btn import-conflict-overwrite" data-action="overwrite" title="用导入文件的内容替换笔记内容">覆盖</button>
                    <button class="import-conflict-btn import-conflict-skip" data-action="skip" title="跳过此文件，笔记内容保持不变">跳过</button>
                </div>
            `;
            el.querySelector('[data-action="overwrite"]').addEventListener('click', () => handleItem('overwrite', item, el));
            el.querySelector('[data-action="skip"]').addEventListener('click', () => handleItem('skip', item, el));
            listEl.appendChild(el);
        }
    }

    async function handleItem(action, item, el) {
        const overwrite = action === 'overwrite';
        const msg = overwrite
            ? `确认覆盖笔记「${item.title}」？`
            : `确认跳过笔记「${item.title}」？`;
        const ok = await showConfirmDialog(msg, '确定', '取消');
        if (!ok) return;
        try {
            const result = await window.go.main.App.ResolveImportConflict(
                item.note_id, overwrite, item.title, item.content, item.file_ext
            );
            resolved.push(result);
        } catch (err) {
            resolved.push({ note_id: item.note_id, success: false, status: 'error', error: err.message });
        }
        // 记录剩余条目的当前位置（First）
        const remaining = [...listEl.querySelectorAll('.import-conflict-item')].filter(e => e !== el);
        const oldPositions = new Map();
        remaining.forEach(e => oldPositions.set(e, e.getBoundingClientRect()));

        // 折叠移除条目
        items = items.filter(i => i !== item);
        el.style.maxHeight = el.offsetHeight + 'px';
        el.classList.add('collapsing');

        await new Promise(resolve => {
            el.addEventListener('transitionend', resolve, { once: true });
            // fallback 防止 transitionend 不触发
            setTimeout(resolve, 300);
        });
        el.remove();

        if (items.length === 0) {
            close(false, resolved);
            return;
        }

        // FLIP 动画：重建 DOM 后平滑移动剩余条目
        renderItems();
        const newItems = [...listEl.querySelectorAll('.import-conflict-item')];
        newItems.forEach(el => {
            const oldPos = oldPositions.get(el);
            if (!oldPos) return;
            const newPos = el.getBoundingClientRect();
            const dy = oldPos.top - newPos.top;
            if (Math.abs(dy) < 1) return;
            el.style.transform = `translateY(${dy}px)`;
            el.style.transition = 'none';
            requestAnimationFrame(() => {
                requestAnimationFrame(() => {
                    el.style.transition = 'transform 0.25s ease';
                    el.style.transform = '';
                });
            });
        });
    }

    async function handleBatch(action) {
        const overwrite = action === 'overwrite';
        const msg = overwrite
            ? `确认覆盖全部 ${items.length} 个笔记？`
            : `确认跳过全部 ${items.length} 个笔记？`;
        const ok = await showConfirmDialog(msg, '确定', '取消');
        if (!ok) return;
        const batchItems = [...items];
        for (const item of batchItems) {
            try {
                const result = await window.go.main.App.ResolveImportConflict(
                    item.note_id, overwrite, item.title, item.content, item.file_ext
                );
                resolved.push(result);
            } catch (err) {
                resolved.push({ note_id: item.note_id, success: false, status: 'error', error: err.message });
            }
        }
        items = [];
        renderItems();
    }

    function close(cancelled = false, result = null) {
        overlay.classList.remove('visible');
        setTimeout(() => {
            overlay.remove();
            if (onComplete) onComplete(cancelled ? false : result);
        }, 200);
    }

    // 挂载取消回调供全局 ESC 调用
    overlay._onCancel = () => close(true);

    // 绑定批量按钮
    overlay.querySelector('[data-action="overwrite-all"]').addEventListener('click', () => handleBatch('overwrite'));
    overlay.querySelector('[data-action="skip-all"]').addEventListener('click', () => handleBatch('skip'));

    // 点击遮罩关闭（视为取消）
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close(true);
    });

    // ESC 关闭（视为取消）
    overlay.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') close(true);
    });

    // 挂载并显示
    document.body.appendChild(overlay);
    requestAnimationFrame(() => {
        overlay.classList.add('visible');
    });

    renderItems();
}

async function handleFileDropPaths(paths, notebookId) {
    if (!paths || paths.length === 0) return;
    if (_importing) {
        nm.show('导入进行中，请稍候', 'warning');
        return;
    }
    _importing = true;

    if (!window.go || !window.go.main || !window.go.main.App || !window.go.main.App.ImportFiles) {
        nm.show('文件导入功能暂不可用', 'error');
        _importing = false;
        return;
    }

    let progressCtrl = null;
    let done = false;
    const startTime = Date.now();

    // 注册进度事件监听（在调用 RPC 前注册）
    const unsub = window.runtime.EventsOn('import:progress', (type, total) => {
        if (type === 'start') {
            progressCtrl = nm.showProgress('正在导入', total);
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
        const results = await window.go.main.App.ImportFiles(paths, notebookId);
        if (!results || results.length === 0) {
            _importing = false;
            return;
        }

        const doneFn = () => { _importing = false; };
        if (done) {
            // 等保底延迟完成后再展示通知和刷新 UI
            const elapsed = Date.now() - startTime;
            setTimeout(() => {
                showImportResults(results, null, doneFn);
            }, Math.max(0, 500 - elapsed));
            return;
        }

        // 兜底：事件未收到时，直接处理结果
        showImportResults(results, null, doneFn);
    } catch (err) {
        console.error('批量导入失败:', err);
        nm.show('文件导入失败：' + (err.message || '未知错误'), 'error');
        _importing = false;
    } finally {
        unsub();
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
 * 检查是否应以全屏模式打开笔记
 */
function getNoteOpenFullscreen() {
    return els.noteOpenFullscreenToggle?.checked || false;
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
    applyAIHighlightTheme(themeName);
    // 若编辑器已打开，销毁重创建
    if (cmEditor) {
        const container = els.editorNoteContent;
        const content = cmEditor.state.doc.toString();
        const isReadOnly = cmEditor.state.readOnly;
        cmEditor.destroy();
		cmEditor = null;
		window.cmEditor = null;
		// 从设置中获取当前的 useSyntaxHighlight
		const useSyntaxHighlight = els.mdHighlightToggle.checked;
        const enableWordWrap = els.editorWordWrapToggle?.checked || false;
        initCodeMirror(container, content, isReadOnly, useSyntaxHighlight, els.editorFileExt.textContent, themeName, enableWordWrap);
    }
    // 同步更新设置页代码预览
    if (_codePreviewInited) {
        const container = document.getElementById('codePreview');
        if (container) {
            buildCodePreview(container, themeName);
        }
    }
}

let _codeHighlightThemeInited = false;
let _settingsSidebarInited = false;
let _dataNavInited = false;

/**
 * 初始化设置页侧边栏导航事件绑定
 */
function initSettingsSidebarNav() {
    if (_settingsSidebarInited) return;
    _settingsSidebarInited = true;

    const nav = els.settingsNav;
    if (!nav) return;

    nav.querySelectorAll('.settings-nav-item').forEach(item => {
        item.addEventListener('click', () => {
            const panelName = item.dataset.panel;
            if (panelName) switchSettingsTab(panelName);
        });
    });
}

/**
 * 初始化数据管理页侧边栏导航事件绑定
 */
function initDataNav() {
    if (_dataNavInited) return;
    _dataNavInited = true;

    const nav = els.dataNav;
    if (!nav) return;

    nav.querySelectorAll('.data-nav-item').forEach(item => {
        item.addEventListener('click', () => {
            const panelName = item.dataset.panel;
            if (panelName) switchDataTab(panelName);
        });
    });
}

/**
 * 初始化代码高亮主题设置（只处理触发按钮和外部点击关闭）
 * 菜单项由 buildCodeHighlightThemeDropdown() 动态生成并绑定事件
 */
function initCodeHighlightThemeSettings() {
    if (_codeHighlightThemeInited) return;
    _codeHighlightThemeInited = true;

    const trigger = document.getElementById('codeHighlightThemeTrigger');
    const dropdown = document.getElementById('codeHighlightThemeDropdown');
    if (!trigger || !dropdown) return;

    // 点击触发按钮切换下拉菜单
    trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        trigger.classList.toggle('open');
        dropdown.classList.toggle('open');
        // 打开时让下拉菜单聚焦，以接收键盘事件
        if (dropdown.classList.contains('open')) {
            dropdown.focus({preventScroll: true});
        }
    });

    // 点击外部关闭下拉菜单
    document.addEventListener('click', (e) => {
        if (dropdown.classList.contains('open') &&
            !trigger.contains(e.target) &&
            !dropdown.contains(e.target)) {
            dropdown.classList.remove('open');
            trigger.classList.remove('open');
        }
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
        'package main',
        '',
        'import "fmt"',
        '',
        'type Counter struct {',
        '    count int',
        '}',
        '',
        'func (c *Counter) Increment() {',
        '    c.count++',
        '}',
        '',
        'func (c *Counter) Exceed(limit int) bool {',
        '    if c.count > limit {',
        '        fmt.Printf("Count %d exceeded limit!\\n", c.count)',
        '        return true',
        '    }',
        '    return false',
        '}',
        '',
        'func main() {',
        '    counter := &Counter{}',
        '    for i := 0; i < 15; i++ {',
        '        counter.Increment()',
        '    }',
        '    if counter.Exceed(10) {',
        '        fmt.Println("Limit reached")',
        '    }',
        '}',
    ].join('\n');

    const extensions = [
        EditorView.editable.of(false),
        EditorView.theme({
            '&': { backgroundColor: 'transparent' },
            '.cm-scroller': { overflow: 'auto', maxHeight: '200px', fontFamily: 'var(--font-mono)', fontSize: '12px' },
            '.cm-gutters': { display: 'none' },
            '.cm-line': { padding: '0 2px' },
            '.cm-editor': { outline: 'none' },
            '&.cm-focused': { outline: 'none' },
        }),
    ];

    const highlightExt = getHighlightExtension('.go', themeName);
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

/* ===== 统一设置加载/保存 ===== */

/* ===== Agent 工具设置（设置页「对话与搜索」面板） ===== */

/** 当前禁用的 Agent 工具名数组（对应设置键 ai_agent_tools_disabled，值为 JSON 数组字符串） */
let agentToolsDisabled = [];

/** 后端返回的完整 Agent 工具清单（[{Name, Label, Enabled}]），用于渲染管理面板 */
let agentToolsMeta = [];
// 本次会话内工具启/停变更记录：关闭面板后汇总提示一次（去重）
let agentToolsChanges = { enabled: [], disabled: [] };
// Agent 工具管理面板展开状态与容器（仿「配置预设管理」行内展开）
let agentToolsMgrExpanded = false;
let agentToolsMgrContainer = null;
let agentToolsSelectAllCheckbox = null;

/**
 * 关闭设置页「Agent 工具」管理面板（收起动画结束后移除容器，返回 Promise）
 */
function closeAgentToolsMgrList() {
    agentToolsMgrExpanded = false;
    // 关闭面板时汇总本次会话的工具启停变更，提示一次
    const enCount = agentToolsChanges.enabled.length;
    const deCount = agentToolsChanges.disabled.length;
    if (enCount > 0 || deCount > 0) {
        const parts = [];
        if (deCount > 0) parts.push(`禁用 ${deCount} 个工具`);
        if (enCount > 0) parts.push(`启用 ${enCount} 个工具`);
        nm.show(`工具配置已保存：${parts.join('，')}`, 'success');
        agentToolsChanges = { enabled: [], disabled: [] };
    }
    if (!agentToolsMgrContainer) {
        // 无容器时仍需复位按钮态
        const btn = document.getElementById('aiAgentToolsBtn');
        if (btn) {
            btn.classList.remove('open');
            btn.setAttribute('aria-expanded', 'false');
        }
        return Promise.resolve();
    }
    const container = agentToolsMgrContainer;
    container.classList.remove('open');
    container.classList.remove('closing');
    // 收起动画：只用合成属性（opacity/transform）+ 精确高度的 max-height 折叠，
    // 避免 filter: blur（CPU 高斯模糊，14 行文本每帧模糊是卡顿主因）与 500px 假想高度造成的首帧跳变；
    // 时长 200ms + 标准缓动，收尾利落不拖沓。
    const startH = container.offsetHeight; // 实际显示高度（受 CSS max-height: 360px 限制）
    const anim = container.animate([
        { opacity: 1, transform: 'scaleY(1)', maxHeight: startH + 'px', paddingTop: '12px', paddingBottom: '12px' },
        { opacity: 0, transform: 'scaleY(0.98)', maxHeight: '0px', paddingTop: '0px', paddingBottom: '0px' }
    ], { duration: 200, easing: 'cubic-bezier(0.4, 0, 0.2, 1)', fill: 'both', transformOrigin: 'top center' });
    return new Promise(resolve => {
        anim.onfinish = () => {
            if (container.parentNode) {
                container.parentNode.removeChild(container);
                if (agentToolsMgrContainer === container) {
                    agentToolsMgrContainer = null;
                    agentToolsSelectAllCheckbox = null; // 重置全选 checkbox 引用
                }
            }
            const btn = document.getElementById('aiAgentToolsBtn');
            if (btn) {
                btn.classList.remove('open');
                btn.setAttribute('aria-expanded', 'false');
            }
            resolve();
        };
    });
}

/**
 * 更新「Agent 工具」按钮文案（已启用 N/M）
 */
function updateAgentToolsButtonText() {
    const btnText = document.getElementById('aiAgentToolsBtnText');
    if (!btnText) return;
    let enabledCount = 0;
    let totalCount = 0;
    agentToolsMeta.forEach((tool) => {
        if (tool.PlanOnly) return; // 排除 Plan 模式专属工具
        totalCount++;
        if (agentToolsDisabled.indexOf(tool.Name) === -1) enabledCount++;
    });
    btnText.textContent = `已启用 ${enabledCount}/${totalCount}`;
}

/**
 * 同步全选 checkbox 的状态（checked / unchecked / indeterminate）
 */
function updateSelectAllCheckboxState() {
    if (!agentToolsSelectAllCheckbox) return;
    // 排除 Plan 模式专属工具后统计
    const controllableTools = agentToolsMeta.filter(tool => !tool.PlanOnly);
    if (controllableTools.length === 0) {
        agentToolsSelectAllCheckbox.checked = false;
        agentToolsSelectAllCheckbox.indeterminate = false;
        return;
    }
    const enabledCount = controllableTools.filter(tool =>
        agentToolsDisabled.indexOf(tool.Name) === -1
    ).length;
    if (enabledCount === controllableTools.length) {
        agentToolsSelectAllCheckbox.checked = true;
        agentToolsSelectAllCheckbox.indeterminate = false;
    } else if (enabledCount === 0) {
        agentToolsSelectAllCheckbox.checked = false;
        agentToolsSelectAllCheckbox.indeterminate = false;
    } else {
        agentToolsSelectAllCheckbox.indeterminate = true;
        agentToolsSelectAllCheckbox.checked = false;
    }
}

/**
 * 批量启用/禁用所有 Agent 工具
 */
function toggleSelectAllTools() {
    // 注意：change 事件在状态切换后触发，所以 checked 已是新状态
    // checked（且非 indeterminate）→ 启用全部；unchecked 或 indeterminate → 禁用全部
    const shouldEnable = agentToolsSelectAllCheckbox.checked && !agentToolsSelectAllCheckbox.indeterminate;

    agentToolsMeta.forEach(tool => {
        if (tool.PlanOnly) return; // Plan 模式专属工具不参与全选/全不选
        const isEnabled = agentToolsDisabled.indexOf(tool.Name) === -1;
        if (isEnabled === shouldEnable) return; // 状态未变，跳过

        if (shouldEnable) {
            // 启用：从禁用列表移除
            const idx = agentToolsDisabled.indexOf(tool.Name);
            if (idx !== -1) agentToolsDisabled.splice(idx, 1);
            // 记录变更
            if (agentToolsChanges.enabled.indexOf(tool.Name) === -1) {
                agentToolsChanges.enabled.push(tool.Name);
            }
            // 清除相反方向的变更记录
            const deIdx = agentToolsChanges.disabled.indexOf(tool.Name);
            if (deIdx !== -1) agentToolsChanges.disabled.splice(deIdx, 1);
        } else {
            // 禁用：加入禁用列表
            if (agentToolsDisabled.indexOf(tool.Name) === -1) {
                agentToolsDisabled.push(tool.Name);
            }
            // 记录变更
            if (agentToolsChanges.disabled.indexOf(tool.Name) === -1) {
                agentToolsChanges.disabled.push(tool.Name);
            }
            // 清除相反方向的变更记录
            const enIdx = agentToolsChanges.enabled.indexOf(tool.Name);
            if (enIdx !== -1) agentToolsChanges.enabled.splice(enIdx, 1);
        }
    });

    // 更新所有子 checkbox 的 UI 状态（排除 Plan 模式专属工具）
    document.querySelectorAll('.ai-agent-tools-item:not(.is-plan-only) input[type="checkbox"]').forEach(cb => {
        cb.checked = shouldEnable;
    });

    updateAgentToolsButtonText();
    updateSelectAllCheckboxState();
    saveSettings();
}

/**
 * 渲染设置页「Agent 工具」管理面板（插入到设置项下方，仿「配置预设管理」行内展开）
 * 勾选状态 = 不在 agentToolsDisabled 中；变化时立即保存
 */
function renderAgentToolsMgrList() {
    const anchor = document.getElementById('aiAgentToolsSettingItem');
    if (!anchor) return;
    if (!agentToolsMgrContainer) {
        agentToolsMgrContainer = document.createElement('div');
        agentToolsMgrContainer.className = 'agent-tools-mgr-list';
        // 滚动条自动显隐（仅第一次创建时绑定）
        let timer = null;
        agentToolsMgrContainer.addEventListener('scroll', (e) => {
            if (e.target !== agentToolsMgrContainer) return;
            agentToolsMgrContainer.classList.add('scrolling');
            clearTimeout(timer);
            timer = setTimeout(() => {
                agentToolsMgrContainer.classList.remove('scrolling');
            }, 1000);
        });
        // 插入到设置项下方
        anchor.after(agentToolsMgrContainer);
        // 下一帧触发入场动画
        requestAnimationFrame(() => {
            agentToolsMgrContainer.classList.add('open');
        });
    }
    agentToolsMgrContainer.innerHTML = '';
    agentToolsMgrExpanded = true;

    // header：全选 checkbox + 标题 + 关闭按钮
    const header = document.createElement('div');
    header.className = 'agent-tools-mgr-header';

    // 左侧容器：全选 checkbox + 标题
    const headerLeft = document.createElement('div');
    headerLeft.className = 'agent-tools-mgr-header-left';

    // 全选 checkbox
    agentToolsSelectAllCheckbox = document.createElement('input');
    agentToolsSelectAllCheckbox.type = 'checkbox';
    agentToolsSelectAllCheckbox.className = 'agent-tools-mgr-select-all';
    agentToolsSelectAllCheckbox.addEventListener('change', toggleSelectAllTools);
    updateSelectAllCheckboxState(); // 初始化状态

    const title = document.createElement('span');
    title.className = 'agent-tools-mgr-title';
    title.textContent = 'Agent 工具';

    headerLeft.appendChild(agentToolsSelectAllCheckbox);
    headerLeft.appendChild(title);

    const closeBtn = document.createElement('button');
    closeBtn.className = 'btn btn-sm btn-secondary';
    closeBtn.textContent = '关闭';
    closeBtn.addEventListener('click', closeAgentToolsMgrList);
    header.appendChild(headerLeft);
    header.appendChild(closeBtn);
    agentToolsMgrContainer.appendChild(header);

    // 工具行列表
    agentToolsMeta.forEach((tool, index) => {
        const row = createAgentToolRow(tool);
        row.style.animationDelay = `${index * 30}ms`;
        agentToolsMgrContainer.appendChild(row);
    });

    updateAgentToolsButtonText();
}

/**
 * 创建单个 Agent 工具行（checkbox + 名称 + 说明）
 */
function createAgentToolRow(tool) {
    const itemLabel = document.createElement('label');
    itemLabel.className = 'ai-agent-tools-item';

    const isEnabled = agentToolsDisabled.indexOf(tool.Name) === -1;

    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.checked = isEnabled;
    checkbox.addEventListener('change', () => {
        if (checkbox.checked) {
            // 启用 → 从禁用列表移除
            const idx = agentToolsDisabled.indexOf(tool.Name);
            if (idx !== -1) agentToolsDisabled.splice(idx, 1);
            // 记录变更（去重，与禁用记录互斥）
            const deIdx = agentToolsChanges.disabled.indexOf(tool.Name);
            if (deIdx !== -1) agentToolsChanges.disabled.splice(deIdx, 1);
            if (agentToolsChanges.enabled.indexOf(tool.Name) === -1) {
                agentToolsChanges.enabled.push(tool.Name);
            }
        } else {
            // 禁用 → 加入禁用列表
            if (agentToolsDisabled.indexOf(tool.Name) === -1) {
                agentToolsDisabled.push(tool.Name);
            }
            // 记录变更（去重，与启用记录互斥）
            const enIdx = agentToolsChanges.enabled.indexOf(tool.Name);
            if (enIdx !== -1) agentToolsChanges.enabled.splice(enIdx, 1);
            if (agentToolsChanges.disabled.indexOf(tool.Name) === -1) {
                agentToolsChanges.disabled.push(tool.Name);
            }
        }
        updateAgentToolsButtonText();
        updateSelectAllCheckboxState();
        saveSettings();
    });

    const nameSpan = document.createElement('span');
    nameSpan.className = 'ai-agent-tools-name';
    nameSpan.textContent = tool.Name;

    const descSpan = document.createElement('span');
    descSpan.className = 'ai-agent-tools-desc';
    descSpan.textContent = tool.Label || '';

    itemLabel.appendChild(checkbox);
    itemLabel.appendChild(nameSpan);
    itemLabel.appendChild(descSpan);

    // Plan 模式专属工具：禁用展示 + 点击抖动提示
    if (tool.PlanOnly) {
        checkbox.disabled = true;
        itemLabel.classList.add('is-plan-only');
        const hint = document.createElement('span');
        hint.className = 'plan-only-hint';
        hint.textContent = '仅 Plan 模式可用';
        itemLabel.appendChild(hint);
        itemLabel.addEventListener('click', (e) => {
            e.preventDefault();
            if (!itemLabel.classList.contains('shake')) {
                itemLabel.classList.add('shake');
                setTimeout(() => itemLabel.classList.remove('shake'), 400);
            }
            window.showNotification?.('此工具仅在 Plan 模式下可用，请切换到 Plan 模式', 'info');
        });
    }

    return itemLabel;
}

/* ===== MCP 服务器设置（设置页「MCP 服务器」面板） ===== */

/** 当前 MCP 服务器列表缓存（来自后端 GetMCPServers） */
let mcpServers = [];

/** MCP 服务器表单模式：create 新增 / edit 编辑 */
let mcpFormMode = 'create';
/** 编辑模式下的服务器 ID（新增时为 0） */
let mcpFormEditId = 0;
/** 保存请求进行中标记，防止重复提交 */
let mcpFormSaving = false;
/** 表单中服务器的启用状态（新增默认启用；编辑沿用原值） */
let mcpFormEnabled = true;
/** 表单中服务器的传输方式（新增默认 stdio；编辑沿用原值） */
let mcpFormTransport = 'stdio';
/** MCP 服务器表单打开时的初始值快照（用于关闭时判断是否有未保存修改） */
let mcpFormInitial = {
    name: '', transport: 'stdio', command: '', args: '', env: '', url: '', headers: '',
};
/** 传输方式选项文案（与表单下拉项一致） */
const MCP_TRANSPORT_OPTIONS = {
    stdio: 'stdio（本地进程）',
    sse: 'sse（远程流式）',
    http: 'http（远程）',
};

/**
 * 格式化后端错误信息：去掉 Wails 包装的 "Error: " 前缀
 * @param {*} err - 后端返回的错误（字符串或 Error 对象）
 * @returns {string}
 */
function mcpErrMsg(err) {
    if (!err) return '未知错误';
    const s = String(err);
    return s.replace(/^Error:\s*/, '');
}

/**
 * 将 MCP 服务器记录数组序列化为本项目分享格式 JSON
 * 输出只含 name / transport / 按 transport 条件附带 command / args / env 或 url / headers / enabled；
 * 排除 id / sort_order / created_at / updated_at 等运行时/数据库字段。
 * @param {Array} servers - MCP 服务器记录数组
 * @returns {Array} 序列化后的纯数据对象数组
 */
function buildMCPServersShareJSON(servers) {
    if (!Array.isArray(servers)) return [];
    const out = [];
    for (const srv of servers) {
        if (!srv || typeof srv !== 'object') continue;
        const transport = srv.transport || 'stdio';
        const item = {
            name: srv.name || '',
            transport,
            enabled: !!srv.enabled,
        };
        if (transport === 'stdio') {
            if (srv.command) item.command = srv.command;
            if (Array.isArray(srv.args) && srv.args.length > 0) item.args = srv.args.slice();
            if (srv.env && typeof srv.env === 'object' && Object.keys(srv.env).length > 0) item.env = { ...srv.env };
        } else {
            if (srv.url) item.url = srv.url;
            if (srv.headers && typeof srv.headers === 'object' && Object.keys(srv.headers).length > 0) {
                item.headers = { ...srv.headers };
            }
        }
        out.push(item);
    }
    return out;
}

/**
 * 复制文本到剪贴板：clipboard API 主路径，execCommand 降级。
 * @param {string} text - 要复制的文本
 * @param {string} successMsg - 成功提示
 * @param {string} emptyMsg - 空数据提示（传 '' 时不提示空态）
 * @returns {Promise<boolean>} 是否复制成功
 */
async function copyMCPServersShare(text, successMsg, emptyMsg) {
    // 空数据短路：直接走空态提示，不复制
    if (typeof text !== 'string' || text.length === 0) {
        if (emptyMsg) nm.show(emptyMsg, 'info');
        return false;
    }
    // 主路径：navigator.clipboard
    try {
        if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            await navigator.clipboard.writeText(text);
            if (successMsg) nm.show(successMsg, 'success');
            return true;
        }
    } catch (e) {
        // 拒签或不支持时降级
    }
    // 降级方案：隐藏 textarea + document.execCommand('copy')
    try {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.setAttribute('readonly', '');
        ta.style.position = 'fixed';
        ta.style.top = '0';
        ta.style.left = '0';
        ta.style.opacity = '0';
        ta.style.pointerEvents = 'none';
        document.body.appendChild(ta);
        ta.focus();
        ta.select();
        const ok = document.execCommand('copy');
        document.body.removeChild(ta);
        if (ok) {
            if (successMsg) nm.show(successMsg, 'success');
            return true;
        }
        nm.show('复制失败，请手动复制', 'error');
        return false;
    } catch (e) {
        nm.show('复制失败，请手动复制', 'error');
        return false;
    }
}

/**
 * 创建 MCP 导入对话框的 CM6 JSON 编辑器（每次打开时重建，确保主题同步）
 */
function createMCPImportEditor() {
    const container = document.getElementById('mcpServerImportInput');
    if (!container) return;
    // 销毁旧实例
    if (_mcpImportEditor) {
        _mcpImportEditor.destroy();
        _mcpImportEditor = null;
    }

    const extensions = [
        json(),
        getHighlightExtension('.json', codeHighlightTheme),
        EditorView.theme({
            '&': {
                height: '100%',
                backgroundColor: 'var(--input-bg)',
            },
            '&.cm-focused': {
                backgroundColor: 'var(--card-bg)',
            },
            '.cm-scroller': {
                fontFamily: 'var(--font-mono)',
                fontSize: '0.8rem',
                lineHeight: '1.55',
                overflow: 'auto',
            },
            '.cm-content': {
                fontFamily: 'var(--font-mono)',
                padding: '8px 12px',
            },
            '.cm-gutters': { display: 'none' },
            '.cm-activeLine': { backgroundColor: 'transparent' },
        }),
        EditorView.lineWrapping,
        placeholder('在此粘贴 JSON 配置…'),
    ];

    const state = EditorState.create({ doc: '', extensions });
    _mcpImportEditor = new EditorView({ state, parent: container });
}

/**
 * 打开 MCP 导入对话框
 */
function openMCPImportDialog() {
    const dialog = document.getElementById('mcpServerImportDialog');
    if (!dialog) return;
    createMCPImportEditor();
    dialog.style.display = 'flex';
    requestAnimationFrame(() => dialog.classList.add('visible'));
    setTimeout(() => {
        if (_mcpImportEditor) _mcpImportEditor.focus();
    }, 200);
}

/**
 * 关闭 MCP 导入对话框并清空编辑器
 */
function closeMCPImportDialog() {
    const dialog = document.getElementById('mcpServerImportDialog');
    if (!dialog || dialog.style.display === 'none') return;
    dialog.classList.remove('visible');
    setTimeout(() => {
        if (!dialog.classList.contains('visible')) {
            dialog.style.display = 'none';
            if (_mcpImportEditor) {
                _mcpImportEditor.destroy();
                _mcpImportEditor = null;
            }
        }
    }, 220);
}

/**
 * 处理导入按钮点击：两阶段流程
 *   阶段 1：仅解析+校验（App.ParseMCPServersImport），校验失败 → 抖动+通知，对话框不关
 *   阶段 2：校验通过 → 关闭对话框 → 调 App.ImportMCPServers 实际入库
 */
async function handleMCPImport() {
    const dialog = document.getElementById('mcpServerImportDialog');
    const container = document.getElementById('mcpServerImportInput');
    if (!container) return;
    const text = _mcpImportEditor ? _mcpImportEditor.state.doc.toString().trim() : '';
    if (!text) {
        nm.show('请粘贴 JSON', 'error');
        shakeMCPFormInput(container);
        if (_mcpImportEditor) _mcpImportEditor.focus();
        return;
    }

    // 防重复提交：禁用按钮 + 切换文案
    const confirmBtn = document.getElementById('mcpServerImportConfirmBtn');
    if (confirmBtn) {
        confirmBtn.disabled = true;
        confirmBtn.textContent = '校验中...';
    }

    try {
        // ===== 阶段 1：仅解析+校验，不入库 =====
        const parseResult = await window.go.main.App.ParseMCPServersImport(text);
        if (!parseResult || !parseResult.ok) {
            // 校验失败：抖动 + 通知，对话框不关，按钮恢复
            const reason = (parseResult && parseResult.error)
                || 'JSON 校验失败';
            nm.show(reason, 'error', 5000);
            shakeMCPFormInput(container);
            if (confirmBtn) {
                confirmBtn.disabled = false;
                confirmBtn.textContent = '导入';
            }
            return;
        }

        // ===== 阶段 2：校验通过,先实际入库再决定是否关对话框 =====
        // B5: 不在阶段 2 入口立即关对话框,失败时保留对话框与编辑器,让用户可改后再导
        let results = [];
        try {
            results = await window.go.main.App.ImportMCPServers(text);
        } catch (e) {
            // binding 异常（不在预期内）: 通知 + 抖动 + 对话框不关
            const msg = mcpErrMsg(e);
            nm.show('导入失败: ' + msg, 'error', 5000);
            shakeMCPFormInput(container);
            if (confirmBtn) {
                confirmBtn.disabled = false;
                confirmBtn.textContent = '导入';
            }
            return;
        }
        const safeResults = Array.isArray(results) ? results : [];
        const success = safeResults.filter(r => r && r.ok).length;
        const failed = safeResults.filter(r => r && !r.ok);
        const totalFailed = failed.length;

        // 通知文案:仅列失败条目名称,不列具体原因
        let summary;
        if (totalFailed === 0) {
            summary = `已导入 ${success} 条 MCP 服务器`;
        } else {
            const failedNames = failed
                .map(r => (r.name && r.name.trim()) ? r.name : `第${r.index || '?'}条`)
                .join('、');
            summary = `已导入 ${success} 条,失败 ${totalFailed} 条: ${failedNames},详见日志`;
        }
        const level = totalFailed === 0 ? 'success' : (success === 0 ? 'error' : 'warn');
        const duration = totalFailed === 0 ? 3000 : 5000;
        nm.show(summary, level, duration);

        if (totalFailed > 0) {
            // B5: 失败时保留对话框与编辑器,让用户可改后再导
            shakeMCPFormInput(container);
            if (confirmBtn) {
                confirmBtn.disabled = false;
                confirmBtn.textContent = '导入';
            }
            return;
        }

        // 全部成功:关对话框 + 刷新列表与全局池（silent: 静默预热,避免与"已导入 N 条"通知冗余）
        closeMCPImportDialog();
        try {
            await loadMCPServers();
            await warmupMCPServers({ silent: true });
        } catch (e) { /* 刷新失败不影响主通知 */ }
    } catch (e) {
        // 阶段 1 的 binding 异常(binding 未就绪 / panic)
        const msg = mcpErrMsg(e);
        nm.show('导入失败: ' + msg, 'error', 5000);
        shakeMCPFormInput(container);
    } finally {
        // 兜底恢复按钮(阶段 1 失败时已在 try 内恢复;阶段 2 成功/失败也在分支内恢复)
        if (confirmBtn && confirmBtn.disabled) {
            confirmBtn.disabled = false;
            confirmBtn.textContent = '导入';
        }
    }
}

/**
 * 从后端加载 MCP 服务器列表并渲染
 */
async function loadMCPServers() {
    try {
        mcpServers = (await window.go.main.App.GetMCPServers()) || [];
        renderMCPServerList();
    } catch (e) {
        nm.show('获取 MCP 服务器列表失败', 'error');
    }
}

/**
 * 渲染服务器列表；空列表显示空态
 */
function renderMCPServerList() {
    const listEl = document.getElementById('mcpServerList');
    const emptyEl = document.getElementById('mcpServerEmpty');
    if (!listEl || !emptyEl) return;
    listEl.innerHTML = '';
    if (!mcpServers.length) {
        emptyEl.hidden = false;
        return;
    }
    emptyEl.hidden = true;
    mcpServers.forEach((srv) => listEl.appendChild(buildMCPServerItem(srv)));
}

/**
 * 构建单个服务器列表条目（信息区 + 操作区）
 * @param {object} srv - MCP 服务器记录
 * @returns {HTMLElement}
 */
function buildMCPServerItem(srv) {
    const item = document.createElement('div');
    item.className = 'mcp-server-item';

    // ── 信息区：名称（+ 传输徽标）横排，描述另起一行 ──
    const info = document.createElement('div');
    info.className = 'mcp-server-item-info';

    const nameRow = document.createElement('div');
    nameRow.className = 'mcp-server-item-name-row';

    const nameEl = document.createElement('span');
    nameEl.className = 'mcp-server-item-name';
    nameEl.textContent = srv.name || '';
    nameRow.appendChild(nameEl);

    const badge = document.createElement('span');
    badge.className = `mcp-server-item-badge mcp-badge-${srv.transport || 'stdio'}`;
    badge.textContent = (srv.transport || 'stdio').toUpperCase();
    nameRow.appendChild(badge);

    const desc = document.createElement('span');
    desc.className = 'mcp-server-item-desc';
    if (srv.transport === 'stdio') {
        // stdio：显示命令及参数摘要
        let descText = srv.command || '';
        if (Array.isArray(srv.args) && srv.args.length > 0) {
            const argsText = srv.args.join(' ');
            descText = descText ? `${descText} ${argsText}` : argsText;
        }
        desc.textContent = descText;
    } else {
        // sse / http：显示 url
        desc.textContent = srv.url || '';
    }

    info.appendChild(nameRow);
    info.appendChild(desc);

    // ── 操作区：启用开关 + 分享 + 测试 + 编辑 + 删除 ──
    const actions = document.createElement('div');
    actions.className = 'mcp-server-item-actions';

    const toggle = document.createElement('div');
    toggle.className = 'ai-chat-toggle-switch';
    if (srv.enabled) toggle.classList.add('active');
    toggle.title = srv.enabled ? '点击停用' : '点击启用';
    const knob = document.createElement('div');
    knob.className = 'ai-chat-toggle-knob';
    toggle.appendChild(knob);
    toggle.addEventListener('click', () => toggleMCPServer(srv, toggle));

    const shareBtn = document.createElement('button');
    shareBtn.className = 'btn btn-sm mcp-server-accent-btn';
    shareBtn.title = '复制该服务器配置为 JSON';
    shareBtn.textContent = '分享';
    shareBtn.addEventListener('click', () => {
        const text = JSON.stringify(buildMCPServersShareJSON([srv]), null, 2);
        copyMCPServersShare(text, `已复制「${srv.name}」配置`, '');
    });

    const editBtn = document.createElement('button');
    editBtn.className = 'btn btn-sm mcp-server-accent-btn';
    editBtn.textContent = '编辑';
    editBtn.addEventListener('click', () => openMCPServerForm(srv));

    const testBtn = document.createElement('button');
    testBtn.className = 'btn btn-sm mcp-server-accent-btn';
    testBtn.textContent = '测试';
    testBtn.title = '测试该传输方式的连接是否可用';
    testBtn.addEventListener('click', () => testMCPServer(srv, testBtn));

    const delBtn = document.createElement('button');
    delBtn.className = 'btn btn-sm mcp-server-del-btn';
    delBtn.textContent = '删除';
    delBtn.addEventListener('click', () => deleteMCPServer(srv));

    actions.appendChild(toggle);
    actions.appendChild(shareBtn);
    actions.appendChild(testBtn);
    actions.appendChild(editBtn);
    actions.appendChild(delBtn);

    item.appendChild(info);
    item.appendChild(actions);
    return item;
}

/**
 * 行内启用开关切换：翻转缓存中的 enabled 并保存；失败回滚 UI
 * @param {object} srv - 列表条目对应的服务器记录（缓存引用）
 * @param {HTMLElement} toggleEl - 开关元素
 */
async function toggleMCPServer(srv, toggleEl) {
    const target = mcpServers.find((s) => s.id === srv.id) || srv;
    const newEnabled = !target.enabled;
    // 先乐观更新 UI
    target.enabled = newEnabled;
    toggleEl.classList.toggle('active', newEnabled);
    toggleEl.title = newEnabled ? '点击停用' : '点击启用';
    try {
        await window.go.main.App.SaveMCPServer({ ...target, enabled: newEnabled });
        nm.show(newEnabled ? `已启用「${target.name}」` : `已停用「${target.name}」`, 'success');
        // 同步预热池：启用→预热该服务器，停用→关闭该服务器连接
        await warmupMCPServers();
    } catch (e) {
        // 失败回滚 UI 状态
        target.enabled = !newEnabled;
        toggleEl.classList.toggle('active', target.enabled);
        toggleEl.title = target.enabled ? '点击停用' : '点击启用';
        nm.show(`操作失败：${mcpErrMsg(e)}`, 'error');
    }
}

/**
 * 测试 MCP 服务器连接是否可用（连接 + 握手 + 工具发现）
 * @param {object} srv - 服务器记录
 * @param {HTMLElement} btn - 触发测试的按钮（用于加载态）
 */
async function testMCPServer(srv, btn) {
    const startAt = Date.now();
    setBtnLoading(btn, true);
    let message = '';
    let type = 'error';
    try {
        const res = await window.go.main.App.TestMCPServer(srv.id);
        if (res && res.ok) {
            const toolText = res.tool_num > 0 ? `，发现 ${res.tool_num} 个工具` : '';
            message = `「${srv.name}」连接成功${toolText}`;
            type = 'success';
        } else {
            // 后端错误文案已含服务器名（如「MCP 服务器 xxx 连接失败: …」），直接展示避免重复前缀
            message = mcpErrMsg((res && res.message) || '未知错误');
        }
    } catch (e) {
        message = `「${srv.name}」测试出错：${mcpErrMsg(e)}`;
    } finally {
        // 保证加载动画至少可见 600ms（本地连接可能瞬时完成，避免动画一闪而过）
        const rest = 600 - (Date.now() - startAt);
        if (rest > 0) await new Promise((r) => setTimeout(r, rest));
        setBtnLoading(btn, false);
    }
    if (message) nm.show(message, type);
}

/**
 * 删除服务器（带确认对话框）
 * @param {object} srv - 服务器记录
 */
async function deleteMCPServer(srv) {
    const ok = await showConfirmDialog(`确定删除 MCP 服务器「${srv.name}」？`, '删除');
    if (!ok) return;
    try {
        await window.go.main.App.DeleteMCPServer(srv.id);
        await loadMCPServers();
        nm.show(`已删除「${srv.name}」`, 'success');
        // 同步预热池：删除后关闭该服务器连接
        await warmupMCPServers();
    } catch (e) {
        nm.show(`删除失败：${mcpErrMsg(e)}`, 'error');
    }
}

/**
 * 打开 MCP 服务器表单对话框
 * @param {object|null} srv - 编辑的服务器记录；null 表示新增
 */
function openMCPServerForm(srv) {
    const dialog = document.getElementById('mcpServerFormDialog');
    if (!dialog) return;
    const title = document.getElementById('mcpServerFormTitle');
    const nameInput = document.getElementById('mcpServerNameInput');
    const commandInput = document.getElementById('mcpServerCommandInput');
    const argsInput = document.getElementById('mcpServerArgsInput');
    const envInput = document.getElementById('mcpServerEnvInput');
    const urlInput = document.getElementById('mcpServerUrlInput');
    const headersInput = document.getElementById('mcpServerHeadersInput');

    mcpFormSaving = false;

    if (srv) {
        // 编辑模式：预填充表单
        mcpFormMode = 'edit';
        mcpFormEditId = srv.id;
        title.textContent = '编辑 MCP 服务器';
        nameInput.value = srv.name || '';
        setMCPFormTransport(srv.transport || 'stdio');
        commandInput.value = srv.command || '';
        // 参数数组每行一个
        argsInput.value = Array.isArray(srv.args) ? srv.args.join('\n') : '';
        // 环境变量对象转 KEY=VALUE 每行一个
        envInput.value = srv.env ? Object.keys(srv.env).map((k) => `${k}=${srv.env[k]}`).join('\n') : '';
        urlInput.value = srv.url || '';
        // 请求头对象转 KEY=VALUE 每行一个
        headersInput.value = srv.headers ? Object.keys(srv.headers).map((k) => `${k}=${srv.headers[k]}`).join('\n') : '';
        mcpFormEnabled = !!srv.enabled;
    } else {
        // 新增模式：清空字段并使用默认值
        mcpFormMode = 'create';
        mcpFormEditId = 0;
        title.textContent = '添加 MCP 服务器';
        nameInput.value = '';
        setMCPFormTransport('stdio');
        commandInput.value = '';
        argsInput.value = '';
        envInput.value = '';
        urlInput.value = '';
        headersInput.value = '';
        mcpFormEnabled = true;
    }

    // 记录表单初始快照，用于关闭时判断是否有未保存修改
    mcpFormInitial = {
        name: nameInput.value,
        transport: mcpFormTransport,
        command: commandInput.value,
        args: argsInput.value,
        env: envInput.value,
        url: urlInput.value,
        headers: headersInput.value,
    };

    // 显示对话框（visible 类触发淡入动画，与 pwdModal 一致）
    dialog.style.display = 'flex';
    requestAnimationFrame(() => dialog.classList.add('visible'));
    setTimeout(() => nameInput.focus(), 200);
}

// 判断 MCP 服务器表单是否相对初始快照有修改
function hasMCPServerFormChanges() {
    const g = (id) => (document.getElementById(id)?.value ?? '');
    return g('mcpServerNameInput') !== mcpFormInitial.name
        || mcpFormTransport !== mcpFormInitial.transport
        || g('mcpServerCommandInput') !== mcpFormInitial.command
        || g('mcpServerArgsInput') !== mcpFormInitial.args
        || g('mcpServerEnvInput') !== mcpFormInitial.env
        || g('mcpServerUrlInput') !== mcpFormInitial.url
        || g('mcpServerHeadersInput') !== mcpFormInitial.headers;
}

/**
 * 关闭 MCP 服务器表单对话框
 * @param {boolean} force - true 时跳过未保存修改确认（保存成功后使用）
 */
async function closeMCPServerForm(force = false) {
    const dialog = document.getElementById('mcpServerFormDialog');
    if (!dialog || dialog.style.display === 'none') return;
    // force 必须是字面量 true 才跳过确认（防御：避免事件对象等 truthy 值误传入跳过确认）
    if (force !== true && hasMCPServerFormChanges()) {
        const ok = await showConfirmDialog('有未保存的修改，确定放弃并关闭吗？');
        if (!ok) return;
    }
    dialog.classList.remove('visible');
    // 等关闭过渡结束后隐藏 DOM；期间若重新打开（重新加回 visible）则不隐藏
    setTimeout(() => {
        if (!dialog.classList.contains('visible')) {
            dialog.style.display = 'none';
        }
    }, 220);
}

/**
 * 设置表单传输方式：更新缓存值、触发器文案与选中态，并联动显隐输入组
 * @param {string} transport - stdio / sse / http
 */
function setMCPFormTransport(transport) {
    mcpFormTransport = transport;
    const label = document.getElementById('mcpServerTransportLabel');
    if (label) label.textContent = MCP_TRANSPORT_OPTIONS[transport] || transport;
    const dropdown = document.getElementById('mcpServerTransportDropdown');
    if (dropdown) {
        dropdown.querySelectorAll('.theme-select-item').forEach((item) => {
            item.classList.toggle('active', item.dataset.transport === transport);
        });
    }
    updateMCPServerTransportGroups(transport);
}

/**
 * 按当前传输方式显隐 stdio / url 输入组
 * @param {string} transport - stdio / sse / http
 */
function updateMCPServerTransportGroups(transport) {
    const stdioGroup = document.getElementById('mcpServerStdioGroup');
    const urlGroup = document.getElementById('mcpServerUrlGroup');
    if (!stdioGroup || !urlGroup) return;
    const isStdio = transport === 'stdio';
    // 用 collapsed 类而非 hidden：配合 CSS 0fr/1fr 过渡实现平滑展开/收起
    stdioGroup.classList.toggle('collapsed', !isStdio);
    urlGroup.classList.toggle('collapsed', isStdio);
}

/**
 * 校验失败反馈：输入框抖动 + 红色边框闪烁，动画结束后自动恢复原样
 * @param {HTMLElement} el - 目标输入框
 */
function shakeMCPFormInput(el) {
    if (!el) return;
    el.classList.remove('mcp-input-invalid');
    // 强制 reflow，保证连续触发时动画能重新播放
    void el.offsetWidth;
    el.classList.add('mcp-input-invalid');
    const clear = () => el.classList.remove('mcp-input-invalid');
    el.addEventListener('animationend', function handler(e) {
        if (e.animationName === 'mcpFormInputError') {
            clear();
            el.removeEventListener('animationend', handler);
        }
    });
    // reduced-motion（无动画）时 animationend 不触发，定时兜底恢复
    setTimeout(clear, 800);
}

/**
 * 保存 MCP 服务器表单（新增 / 编辑）
 */
async function saveMCPServerForm() {
    if (mcpFormSaving) return;
    const nameInput = document.getElementById('mcpServerNameInput');
    const commandInput = document.getElementById('mcpServerCommandInput');
    const argsInput = document.getElementById('mcpServerArgsInput');
    const envInput = document.getElementById('mcpServerEnvInput');
    const urlInput = document.getElementById('mcpServerUrlInput');
    const headersInput = document.getElementById('mcpServerHeadersInput');
    if (!nameInput) return;

    const name = nameInput.value.trim();
    const transport = mcpFormTransport;
    const command = commandInput ? commandInput.value.trim() : '';
    const url = urlInput ? urlInput.value.trim() : '';

    // 校验：名称必填
    if (!name) {
        nm.show('请输入服务器名称', 'error');
        shakeMCPFormInput(nameInput);
        nameInput.focus();
        return;
    }
    // 校验：名称不能含空白（名称拼入工具名前缀 mcp_{name}_{tool}）
    if (/\s/.test(name)) {
        nm.show('服务器名称不能包含空格等空白字符', 'error');
        shakeMCPFormInput(nameInput);
        nameInput.focus();
        return;
    }
    // 校验：stdio 需命令，sse/http 需 URL
    if (transport === 'stdio' && !command) {
        nm.show('请输入启动命令', 'error');
        shakeMCPFormInput(commandInput);
        commandInput.focus();
        return;
    }
    if (transport !== 'stdio' && !url) {
        nm.show('请输入服务器 URL', 'error');
        shakeMCPFormInput(urlInput);
        urlInput.focus();
        return;
    }

    // 解析参数：逐行过滤空行
    const args = (argsInput.value || '').split('\n').map((l) => l.trim()).filter((l) => l !== '');

    // 解析环境变量：每行必须为 KEY=VALUE，重复 KEY 后者覆盖
    const env = {};
    const envLines = (envInput.value || '').split('\n');
    for (const rawLine of envLines) {
        const line = rawLine.trim();
        if (!line) continue;
        const eqIdx = line.indexOf('=');
        if (eqIdx === -1) {
            nm.show('环境变量需为 KEY=VALUE 格式', 'error');
            shakeMCPFormInput(envInput);
            envInput.focus();
            return;
        }
        const key = line.slice(0, eqIdx).trim();
        if (!key) {
            nm.show('环境变量需为 KEY=VALUE 格式', 'error');
            shakeMCPFormInput(envInput);
            envInput.focus();
            return;
        }
        if (/\s/.test(key)) {
            nm.show('环境变量 KEY 不能包含空白字符', 'error');
            shakeMCPFormInput(envInput);
            envInput.focus();
            return;
        }
        env[key] = line.slice(eqIdx + 1).trim();
    }

    // 解析请求头：每行必须为 KEY=VALUE，重复 KEY 后者覆盖
    const headers = {};
    const headerLines = (headersInput.value || '').split('\n');
    for (const rawLine of headerLines) {
        const line = rawLine.trim();
        if (!line) continue;
        const eqIdx = line.indexOf('=');
        if (eqIdx === -1) {
            nm.show('请求头需为 KEY=VALUE 格式', 'error');
            shakeMCPFormInput(headersInput);
            headersInput.focus();
            return;
        }
        const key = line.slice(0, eqIdx).trim();
        if (!key) {
            nm.show('请求头需为 KEY=VALUE 格式', 'error');
            shakeMCPFormInput(headersInput);
            headersInput.focus();
            return;
        }
        if (/\s/.test(key)) {
            nm.show('请求头 KEY 不能包含空白字符', 'error');
            shakeMCPFormInput(headersInput);
            headersInput.focus();
            return;
        }
        headers[key] = line.slice(eqIdx + 1).trim();
    }

    // 防重复提交
    mcpFormSaving = true;
    const payload = {
        id: mcpFormEditId,
        name,
        transport,
        command,
        args,
        env,
        url,
        headers,
        enabled: mcpFormEnabled,
    };
    try {
        await window.go.main.App.SaveMCPServer(payload);
        closeMCPServerForm(true); // 保存成功后跳过未保存修改确认
        await loadMCPServers();
        nm.show(mcpFormMode === 'create' ? 'MCP 服务器已添加' : 'MCP 服务器已更新', 'success');
        // 同步预热池：新增/编辑后预热（配置变更自动重连）
        await warmupMCPServers();
    } catch (e) {
        // 后端校验/存储错误（Wails 以异常形式返回）
        nm.show(mcpErrMsg(e), 'error');
    } finally {
        mcpFormSaving = false;
    }
}

/**
 * 预热/同步全局 MCP 连接池（后端 Reconcile：关闭已停用/删除的，预热启用的服务器）。
 * 首次进入 AI 助手、设置页启用/停用/增删改 MCP 服务器后调用；后端幂等。
 * 结果汇总为一条通知展示；无启用服务器时不打扰。
 * 暴露到 window 供 ai-chat.js 首次进入 AI 助手时调用。
 * @param {Object} [options] 配置项
 * @param {boolean} [options.silent=false] true=不展示"X 台已就绪"通知(用于导入等内部动作,避免通知冗余)
 */
async function warmupMCPServers(options = {}) {
    const silent = !!options.silent;
    try {
        const res = await window.go.main.App.WarmupMCPServers();
        if (!silent && res && res.total) {
            const usable = (res.warmed || 0) + (res.reused || 0);
            if (!res.failed) {
                const reuseText = res.reused > 0 ? `（复用 ${res.reused} 台）` : '';
                nm.show(`MCP 服务器已就绪：${usable} 台连接${reuseText}，共 ${res.tool_total} 个工具`, 'success');
            } else {
                const detail = (res.failed_msgs || []).join('；');
                const type = usable === 0 ? 'error' : 'warning';
                nm.show(`MCP 服务器预热：${usable} 台可用，${res.failed} 台失败${detail ? `（${detail}）` : ''}`, type, 6000);
            }
        }
    } catch (e) {
        if (!silent) {
            nm.show(`MCP 服务器预热失败：${mcpErrMsg(e)}`, 'error');
        }
    }
    // 无论预热结果如何，都刷新 Agent 工具列表（禁用/启用服务器后工具列表需同步）
    await refreshAgentToolsMeta();
}
window.warmupMCPServers = warmupMCPServers;

/**
 * 重新加载 Agent 工具元信息（内置 + MCP 工具），刷新开关列表显示。
 * 通常在 MCP 服务器预热/变更后调用，使工具开关列表与池状态同步。
 */
async function refreshAgentToolsMeta() {
    try {
        agentToolsMeta = (await window.go.main.App.GetAgentTools()) || [];
    } catch (e) {
        agentToolsMeta = [];
    }
    updateAgentToolsButtonText();
    // 如果工具管理面板已展开，重新渲染
    if (agentToolsMgrExpanded) {
        renderAgentToolsMgrList();
    }
}

/**
 * 初始化 MCP 服务器设置面板交互（事件绑定）
 */
function initMCPServerSettings() {
    // 添加按钮 → 打开新增表单
    const addBtn = document.getElementById('mcpServerAddBtn');
    if (addBtn) addBtn.addEventListener('click', () => openMCPServerForm(null));

    // 取消按钮 → 关闭表单
    const cancelBtn = document.getElementById('mcpServerFormCancelBtn');
    if (cancelBtn) cancelBtn.addEventListener('click', closeMCPServerForm);

    // 对话框：点击 backdrop 关闭（Esc 关闭由全局 handleKeyboardNavigation 统一处理）
    const dialog = document.getElementById('mcpServerFormDialog');
    if (dialog) {
        dialog.addEventListener('click', (e) => {
            if (e.target === dialog || (e.target.classList && e.target.classList.contains('mcp-server-form-overlay'))) {
                closeMCPServerForm();
            }
        });
    }

    // 传输方式自定义下拉（theme-select 样式）：开关 / 选择 / 外部点击关闭
    const transportTrigger = document.getElementById('mcpServerTransportTrigger');
    const transportDropdown = document.getElementById('mcpServerTransportDropdown');
    if (transportTrigger && transportDropdown) {
        transportTrigger.addEventListener('click', (e) => {
            e.stopPropagation();
            transportTrigger.classList.toggle('open');
            transportDropdown.classList.toggle('open');
        });
        transportDropdown.addEventListener('click', (e) => {
            const item = e.target.closest('.theme-select-item');
            if (!item) return;
            setMCPFormTransport(item.dataset.transport);
            transportDropdown.classList.remove('open');
            transportTrigger.classList.remove('open');
        });
        document.addEventListener('click', (e) => {
            if (!transportTrigger.contains(e.target) && !transportDropdown.contains(e.target)) {
                transportDropdown.classList.remove('open');
                transportTrigger.classList.remove('open');
            }
        });
    }

    // 保存按钮 → 保存表单
    const saveBtn = document.getElementById('mcpServerFormSaveBtn');
    if (saveBtn) saveBtn.addEventListener('click', saveMCPServerForm);

    // 分享全部按钮 → 复制全部服务器配置为 JSON
    // B6: 点击时现取(缓存为空时调 GetMCPServers),避免面板首开未加载时返回空
    // B7: 幂等防护(_shareAllBound 标志),防止 initMCPServerSettings 多次调用时重复绑定
    const shareAllBtn = document.getElementById('mcpServerShareAllBtn');
    if (shareAllBtn && !shareAllBtn._shareAllBound) {
        shareAllBtn._shareAllBound = true;
        shareAllBtn.addEventListener('click', async () => {
            let list = mcpServers;
            if (!Array.isArray(list) || list.length === 0) {
                // 缓存为空时现取一次(应对面板首开未加载或列表尚未渲染)
                try {
                    list = (await window.go.main.App.GetMCPServers()) || [];
                    mcpServers = list; // 同步全局缓存,与 loadMCPServers 行为一致
                } catch (e) {
                    nm.show('获取服务器列表失败: ' + mcpErrMsg(e), 'error');
                    return;
                }
            }
            const text = JSON.stringify(buildMCPServersShareJSON(list), null, 2);
            const n = list.length;
            copyMCPServersShare(text, `已复制 ${n} 条服务器配置`, '当前没有可分享的服务器');
        });
    }

    // 导入按钮 → 打开导入对话框
    const importBtn = document.getElementById('mcpServerImportBtn');
    if (importBtn) importBtn.addEventListener('click', openMCPImportDialog);

    // 导入对话框：取消 / 确认 / backdrop 关闭
    const importCancelBtn = document.getElementById('mcpServerImportCancelBtn');
    if (importCancelBtn) importCancelBtn.addEventListener('click', closeMCPImportDialog);
    const importConfirmBtn = document.getElementById('mcpServerImportConfirmBtn');
    if (importConfirmBtn) importConfirmBtn.addEventListener('click', handleMCPImport);
    const importDialog = document.getElementById('mcpServerImportDialog');
    if (importDialog) {
        importDialog.addEventListener('click', (e) => {
            if (e.target === importDialog || (e.target.classList && e.target.classList.contains('mcp-server-form-overlay'))) {
                closeMCPImportDialog();
            }
        });
    }
}

/**
 * 一次性从后端加载所有设置并应用到前端
 */
async function loadSettings() {
    try {
        const cfg = await window.go.main.App.GetAllSettings();

        // --- 主题 ---
        localStorage.setItem('jot_theme', cfg.theme);
        applyTheme(cfg.theme);

        // --- 字体 ---
        applyFontFamily(cfg.font_family);
        applyFontSize(cfg.font_size);
        updateFontSettingsUI(cfg.font_family, cfg.font_size);

        // --- 排序 ---
        if (els.sortControl) {
            const btns = els.sortControl.querySelectorAll('.segmented-btn');
            const cw = els.sortControl.offsetWidth;
            const segW = (cw - 8) / btns.length;
            btns.forEach((b, i) => {
                const isActive = b.dataset.sortValue === cfg.sort_order;
                b.classList.toggle('active', isActive);
                if (isActive) {
                    els.sortIndicator.style.transform = `translateX(${2 + i * segW}px)`;
                }
            });
        }

        // --- 分页大小 ---
        if (els.pageSizeControl) {
            const btns = els.pageSizeControl.querySelectorAll('.segmented-btn');
            const cw = els.pageSizeControl.offsetWidth;
            const segW = (cw - 8) / btns.length;
            btns.forEach((b, i) => {
                const isActive = parseInt(b.dataset.value, 10) === cfg.page_size;
                b.classList.toggle('active', isActive);
                if (isActive) {
                    els.pageSizeIndicator.style.transform = `translateX(${2 + i * segW}px)`;
                }
            });
        }
        els.pageSizeSettingDesc.textContent = `每页显示 ${cfg.page_size} 条`;

        // --- 语法高亮 checkbox ---
        if (els.mdHighlightToggle) els.mdHighlightToggle.checked = cfg.cm_syntax_highlight;

        // --- 全屏打开 checkbox ---
        if (els.noteOpenFullscreenToggle) els.noteOpenFullscreenToggle.checked = cfg.note_open_fullscreen;

        // --- 自动换行 checkbox ---
        if (els.editorWordWrapToggle) els.editorWordWrapToggle.checked = cfg.editor_word_wrap;

        // --- 代码高亮主题 ---
        codeHighlightTheme = cfg.code_highlight_theme || 'monokai-dimmed';
        applyAIHighlightTheme(codeHighlightTheme);
        applyCodeHighlightThemeUI(codeHighlightTheme);
        // 同步重建预览代码块（覆盖再次进入设置页时 _codePreviewInited 守卫跳过的问题）
        if (_codePreviewInited) {
            const container = document.getElementById('codePreview');
            if (container) {
                buildCodePreview(container, codeHighlightTheme);
            }
        }

        // --- AI: base_url & api_key ---
        if (els.aiBaseURL) els.aiBaseURL.value = cfg.ai_base_url || '';
        if (els.aiAPIKey) els.aiAPIKey.value = cfg.ai_api_key || '';

        // --- AI: 模型下拉 ---
        if (els.aiModelDropdown) {
            els.aiModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
            if (cfg.ai_model) {
                els.aiModelLabel.textContent = cfg.ai_model;
                addModelDropdownItem(cfg.ai_model, true);
            } else {
                els.aiModelLabel.textContent = '-- 请先获取模型列表 --';
                const wrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
                if (wrap) wrap.style.display = 'none';
            }
            const loadWrap = els.aiModelDropdown.querySelector('.ai-model-search-wrap');
            if (loadWrap) {
                loadWrap.style.display = els.aiModelDropdown.querySelectorAll('.theme-select-item').length > 1 ? '' : 'none';
            }
        }

        // --- AI 向量嵌入: base_url & api_key ---
        if (els.aiEmbedBaseURL) els.aiEmbedBaseURL.value = cfg.ai_embed_base_url || '';
        if (els.aiEmbedAPIKey) els.aiEmbedAPIKey.value = cfg.ai_embed_api_key || '';

        // --- AI 向量嵌入: 模型下拉 ---
        if (els.aiEmbedModelDropdown) {
            els.aiEmbedModelDropdown.querySelectorAll('.theme-select-item').forEach(el => el.remove());
            if (cfg.ai_embed_model) {
                els.aiEmbedModelLabel.textContent = cfg.ai_embed_model;
                addModelDropdownItemTo(els.aiEmbedModelDropdown, cfg.ai_embed_model, true);
            } else {
                els.aiEmbedModelLabel.textContent = '-- 请先获取模型列表 --';
                const embedWrap = els.aiEmbedModelDropdown.querySelector('.ai-model-search-wrap');
                if (embedWrap) embedWrap.style.display = 'none';
            }
            const embedLoadWrap = els.aiEmbedModelDropdown.querySelector('.ai-model-search-wrap');
            if (embedLoadWrap) {
                embedLoadWrap.style.display = els.aiEmbedModelDropdown.querySelectorAll('.theme-select-item').length > 1 ? '' : 'none';
            }
        }

        // --- AI: 搜索源开关 ---
        const searchToggle = document.getElementById('aiSettingSearchToggle');
        if (searchToggle) searchToggle.classList.toggle('active', cfg.ai_thinking_enabled);

        // --- Agent 工具开关（禁用清单 + 浮层渲染） ---
        try {
            agentToolsDisabled = JSON.parse(cfg.ai_agent_tools_disabled || '[]') || [];
        } catch (e) {
            agentToolsDisabled = [];
        }
        if (!Array.isArray(agentToolsDisabled)) agentToolsDisabled = [];
        try {
            agentToolsMeta = (await window.go.main.App.GetAgentTools()) || [];
        } catch (e) {
            agentToolsMeta = [];
        }
        updateAgentToolsButtonText();
        closeAgentToolsMgrList();

        // --- AI: 限制输入 ---
        const cardRecallLimit = document.getElementById('aiSettingCardRecallLimit');
        if (cardRecallLimit) cardRecallLimit.value = cfg.ai_card_recall_limit;

        const largeFilePreviewThreshold = document.getElementById('aiLargeFilePreviewThreshold');
        if (largeFilePreviewThreshold) largeFilePreviewThreshold.value = cfg.ai_large_file_preview_threshold;

        const maxFileSize = document.getElementById('maxFileSize');
        if (maxFileSize) maxFileSize.value = cfg.max_file_size;

        const agentMaxIterations = document.getElementById('aiAgentMaxIterations');
        if (agentMaxIterations) agentMaxIterations.value = cfg.ai_agent_max_iterations || 20;

        const retentionDays = document.getElementById('trashCleanupRetentionDays');
        if (retentionDays) retentionDays.value = cfg.trash_cleanup_retention_days || 30;

        // --- 日志级别 ---
        if (els.logLevelControl) {
            const segBtns = els.logLevelControl.querySelectorAll('.segmented-btn');
            const logLevel = cfg.log_level !== undefined ? cfg.log_level : 1;
            segBtns.forEach(btn => {
                btn.classList.toggle('active', parseInt(btn.dataset.value) === logLevel);
            });
            // 移动指示器（面板隐藏时 offsetWidth=0，定位会在面板显示后补充执行）
            if (els.logLevelIndicator) {
                repositionLogLevelIndicator();
            }
        }

        // --- AI: 预设配置 ---
        await loadProfiles();

        // --- AI 向量嵌入: 预设配置 ---
        await loadProfilesEmbed();

        // --- 锁屏密码 ---
        if (document.getElementById('screenLockToggle')) {
            const isEnabled = cfg.screen_lock_enabled === true || cfg.screen_lock_enabled === 'true';
            const toggle = document.getElementById('screenLockToggle');
            const pwdRow = document.getElementById('screenLockPasswordRow');
            const changeBtn = document.getElementById('pwdChangeBtn');
            toggle.classList.toggle('active', isEnabled);
            if (pwdRow) {
                pwdRow.classList.toggle('collapsed', !isEnabled);
            }
            if (changeBtn) {
                const hasPassword = cfg.screen_lock_password && cfg.screen_lock_password !== '';
                changeBtn.textContent = hasPassword ? '修改密码' : '设置密码';
            }
        }
    } catch (e) {
        console.warn('loadSettings: 加载设置失败', e);
    }

    // --- MCP 服务器列表（非阻塞加载，不影响既有流程） ---
    loadMCPServers();
}

/**
 * 从前端 DOM 收集所有设置，一次性保存到后端
 */
async function saveSettings() {
    try {
        const cfg = {
            theme: localStorage.getItem('jot_theme') || 'default',
            font_family: els.fontFamilyDisplay?.textContent || '',
            font_size: parseInt(els.fontSizeSlider?.value) || 16,
            code_highlight_theme: codeHighlightTheme || 'monokai-dimmed',
            note_open_fullscreen: els.noteOpenFullscreenToggle?.checked || false,
            sort_order: (() => {
                const active = els.sortControl?.querySelector('.segmented-btn.active');
                return active?.dataset.sortValue || 'updated_at';
            })(),
            page_size: (() => {
                const active = els.pageSizeControl?.querySelector('.segmented-btn.active');
                return parseInt(active?.dataset.value) || 20;
            })(),
            cm_syntax_highlight: els.mdHighlightToggle?.checked || false,
            ai_base_url: els.aiBaseURL?.value || '',
            ai_api_key: els.aiAPIKey?.value || '',
            ai_model: (() => {
                const m = els.aiModelLabel?.textContent || '';
                return m === '-- 请先获取模型列表 --' ? '' : m;
            })(),
            ai_embed_base_url: els.aiEmbedBaseURL?.value || '',
            ai_embed_api_key: els.aiEmbedAPIKey?.value || '',
            ai_embed_model: (() => {
                const m = els.aiEmbedModelLabel?.textContent || '';
                return m === '-- 请先获取模型列表 --' ? '' : m;
            })(),
            ai_thinking_enabled: document.getElementById('aiSettingSearchToggle')?.classList.contains('active') || false,
            ai_card_recall_limit: parseInt(document.getElementById('aiSettingCardRecallLimit')?.value) || 5,
            ai_large_file_preview_threshold: parseInt(document.getElementById('aiLargeFilePreviewThreshold')?.value) || 10000,
            max_file_size: parseInt(document.getElementById('maxFileSize')?.value) || 1,
            ai_agent_max_iterations: parseInt(document.getElementById('aiAgentMaxIterations')?.value) || 20,
            ai_agent_tools_disabled: JSON.stringify(agentToolsDisabled || []),
            trash_cleanup_retention_days: parseInt(document.getElementById('trashCleanupRetentionDays')?.value) || 30,
            log_level: els.logLevelControl ? parseInt(els.logLevelControl.querySelector('.segmented-btn.active')?.dataset?.value || '1') : 1,
            screen_lock_enabled: document.getElementById('screenLockToggle')?.classList.contains('active') || false,
            editor_word_wrap: els.editorWordWrapToggle?.checked || false,
        };
        await window.go.main.App.SaveAllSettings(cfg);
    } catch (e) {
        console.error('保存设置失败:', e);
    }
}

/** 切换动画进行中标记，防止快速连续点击导致面板重叠 */
let _settingsAnimating = false;

/**
 * 切换设置页侧边栏导航面板
 * @param {string} panelName - data-panel 属性值，如 'appearance', 'editor' 等
 */
function switchSettingsTab(panelName) {
    // 切换面板时关闭「Agent 工具」管理面板
    closeAgentToolsMgrList();
    // 切换面板时关闭预设管理列表（对话链接 / 嵌入链接两处共用同一容器）
    closePresetMgrList();
    // 切换面板时关闭 MCP 服务器表单对话框
    closeMCPServerForm();

    // 动画进行中 → 忽略本次切换，避免面板重叠
    if (_settingsAnimating) return;

    const nav = els.settingsNav;
    const panelsContainer = els.settingsPanels;
    if (!nav || !panelsContainer) return;

    // 查找目标导航项和目标面板
    const targetItem = nav.querySelector(`.settings-nav-item[data-panel="${panelName}"]`);
    const targetPanel = panelsContainer.querySelector(`.settings-panel[data-panel="${panelName}"]`);
    if (!targetItem || !targetPanel) return;

    // 如果已经是激活状态，不做任何事
    if (targetPanel.classList.contains('active') && targetItem.classList.contains('active')) return;

    // 更新侧边栏导航激活态
    nav.querySelectorAll('.settings-nav-item').forEach(item => item.classList.remove('active'));
    targetItem.classList.add('active');

    // 获取当前显示的面板
    const currentPanel = panelsContainer.querySelector('.settings-panel.active');

    // 检测是否应跳过动画（prefers-reduced-motion）
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    if (!currentPanel || prefersReducedMotion) {
        // 无当前面板或用户偏好减少动效 → 直接切换
        if (currentPanel) currentPanel.classList.remove('active');
        targetPanel.classList.add('active');
        // 面板从 hidden→visible，重算可能受 display:none 影响的分段控件指示器
        repositionLogLevelIndicator();
        return;
    }

    // --- 播放切换动画 ---
    _settingsAnimating = true;

    // 阶段1: 旧面板退出动画
    currentPanel.classList.remove('active');
    currentPanel.classList.add('panel-exit');

    currentPanel.addEventListener('animationend', function onExitEnd() {
        currentPanel.removeEventListener('animationend', onExitEnd);
        currentPanel.classList.remove('panel-exit');

        // 阶段2: 新面板进入动画
        targetPanel.classList.add('panel-enter');
        // 面板已具备 display:block，立即重算分段控件指示器（不需要等动画播完）
        repositionLogLevelIndicator();

        targetPanel.addEventListener('animationend', function onEnterEnd() {
            targetPanel.removeEventListener('animationend', onEnterEnd);
            targetPanel.classList.remove('panel-enter');
            targetPanel.classList.add('active');
            _settingsAnimating = false;
        });
    });
}

/** 数据管理页切换动画进行中标记，防止快速连续点击导致面板重叠 */
let _dataAnimating = false;

/**
 * 切换数据管理页侧边栏导航面板
 * @param {string} panelName - data-panel 属性值，如 'overview', 'transfer' 等
 */
function switchDataTab(panelName) {
    // 动画进行中 → 忽略本次切换，避免面板重叠
    if (_dataAnimating) return;

    const nav = els.dataNav;
    const panelsContainer = els.dataPanels;
    if (!nav || !panelsContainer) return;

    // 查找目标导航项和目标面板
    const targetItem = nav.querySelector(`.data-nav-item[data-panel="${panelName}"]`);
    const targetPanel = panelsContainer.querySelector(`.data-panel[data-panel="${panelName}"]`);
    if (!targetItem || !targetPanel) return;

    // 如果已经是激活状态，不做任何事
    if (targetPanel.classList.contains('active') && targetItem.classList.contains('active')) return;

    // 更新侧边栏导航激活态
    nav.querySelectorAll('.data-nav-item').forEach(item => item.classList.remove('active'));
    targetItem.classList.add('active');

    // 获取当前显示的面板
    const currentPanel = panelsContainer.querySelector('.data-panel.active');

    // 检测是否应跳过动画（prefers-reduced-motion）
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    if (!currentPanel || prefersReducedMotion) {
        // 无当前面板或用户偏好减少动效 → 直接切换
        if (currentPanel) currentPanel.classList.remove('active');
        targetPanel.classList.add('active');
        return;
    }

    // --- 播放切换动画 ---
    _dataAnimating = true;

    // 阶段1: 旧面板退出动画
    currentPanel.classList.remove('active');
    currentPanel.classList.add('panel-exit');

    currentPanel.addEventListener('animationend', function onExitEnd() {
        currentPanel.removeEventListener('animationend', onExitEnd);
        currentPanel.classList.remove('panel-exit');

        // 阶段2: 新面板进入动画
        targetPanel.classList.add('panel-enter');

        targetPanel.addEventListener('animationend', function onEnterEnd() {
            targetPanel.removeEventListener('animationend', onEnterEnd);
            targetPanel.classList.remove('panel-enter');
            targetPanel.classList.add('active');
            _dataAnimating = false;
        });
    });
}

/* ===== 待办清单模块 ===== */

/** 当前待办筛选状态：active | done */
let _todoFilter = 'active';
window._todoFilter = _todoFilter;

/**
 * 从后端加载所有待办项并渲染
 */
async function loadTodos() {
    try {
        if (!window.go?.main?.App?.ListTodos) return;
        const todos = await window.go.main.App.ListTodos();
        renderTodos(todos, _todoFilter);
        updateTodoStats(todos);
    } catch (err) {
        console.error('加载待办失败:', err);
    }
}

/**
 * 渲染待办列表
 * @param {Array} todos - 待办项数组
 * @param {string} filter - all | active | done
 */
/**
 * 渲染待办列表
 * @param {Array} todos - 待办项数组
 * @param {string} filter - all | active | done
 */
function renderTodos(todos, filter) {
    const listEl = els.todoList;
    const emptyEl = els.todoEmpty;
    if (!listEl) return;

    // 筛选
    let filtered = todos;
    if (filter === 'active') {
        filtered = todos.filter(t => !t.done);
    } else if (filter === 'done') {
        filtered = todos.filter(t => t.done);
    }

    if (filtered.length === 0) {
        listEl.innerHTML = '';
        if (emptyEl) emptyEl.style.display = 'flex';
        updateTodoStats(todos);
        return;
    }

    if (emptyEl) emptyEl.style.display = 'none';

    listEl.innerHTML = filtered.map((todo, idx) => `
        <div class="todo-item${todo.done ? ' completed' : ''}" data-id="${todo.id}" data-completed="${todo.done}" style="animation-delay:${idx * 40}ms;">
            <button class="todo-checkbox ${todo.done ? 'checked' : ''}" data-action="toggle">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="20 6 9 17 4 12"/>
                </svg>
            </button>
            <span class="todo-text ${todo.done ? 'done' : ''}" data-action="edit">${escapeHtml(todo.text)}</span>
            <button class="todo-delete-btn" data-action="delete" title="删除">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
            </button>
        </div>
    `).join('');

    // 入场动画：每个条目交错弹入 —— 动画由 CSS 的 animation-delay 控制
    requestAnimationFrame(() => {
        listEl.querySelectorAll('.todo-item').forEach(el => {
            el.classList.add('todo-enter');
            el.addEventListener('animationend', () => {
                el.classList.remove('todo-enter');
            }, { once: true });
        });
    });

    updateTodoStats(todos);
}

/**
 * 自动扩展待办输入框高度
 */
function autoResizeTodoInput() {
    const textarea = els.todoInput;
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
}

/**
 * 更新筛选按钮上的数量显示
 */
function updateTodoStats(todos) {
    const total = todos.length;
    const pending = todos.filter(t => !t.done).length;
    const done = total - pending;
    const activeBtn = document.querySelector('.todo-filter-btn[data-filter="active"]');
    const doneBtn = document.querySelector('.todo-filter-btn[data-filter="done"]');
    const allBtn = document.querySelector('.todo-filter-btn[data-filter="all"]');
    if (activeBtn) activeBtn.textContent = pending > 0 ? `待办 ${pending}` : '待办';
    if (doneBtn) doneBtn.textContent = done > 0 ? `已完成 ${done}` : '已完成';
    if (allBtn) allBtn.textContent = total > 0 ? `全部 ${total}` : '全部';
}

/**
 * 添加待办项
 */
async function addTodo() {
    const input = els.todoInput;
    if (!input) return;
    const text = input.value.trim();
    if (!text) {
        nm.show('请输入待办内容', 'warning');
        return;
    }

    try {
        if (!window.go?.main?.App?.CreateTodo) return;
        const newTodo = await window.go.main.App.CreateTodo(text);
        input.value = '';
        // 重置输入框高度
        input.style.height = 'auto';

        // 如果不在"待办"分类，自动切换到待办并刷新
        if (_todoFilter !== 'active') {
            _todoFilter = 'active';
            window._todoFilter = _todoFilter;
            document.querySelectorAll('.todo-filter-btn').forEach(btn => btn.classList.remove('active'));
            const activeBtn = document.querySelector('.todo-filter-btn[data-filter="active"]');
            if (activeBtn) activeBtn.classList.add('active');
            loadTodos();
            return;
        }

        // 隐藏空状态
        if (els.todoEmpty) els.todoEmpty.style.display = 'none';

        const listEl = els.todoList;
        const existingItems = [...listEl.querySelectorAll('.todo-item')];

        // 无已有条目或动画进行中 → fallback：直接插入
        if (existingItems.length === 0 || listEl.dataset.todoAnimating === 'true') {
            insertNewTodoItem(newTodo);
            updateTodoStatsAfterAdd();
            return;
        }

        // === 两段式动画：已有条目先平滑下移 → 再插入新条目 ===
        listEl.dataset.todoAnimating = 'true';

        // 计算下移距离 = 第一个条目的高度 + 列表 gap(6px)
        const shiftY = existingItems[0].offsetHeight + 6;

        // 为防止下移时条目被容器裁剪，临时给列表底部加 padding
        listEl.style.paddingBottom = shiftY + 'px';

        // Phase 1: 直接用 inline style 触发下移（不用 CSS class，避免 .todo-item 的 transition 冲突）
        existingItems.forEach(el => {
            el.style.transition = 'transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)';
            el.style.transform = `translateY(${shiftY}px)`;
        });

        // Phase 2: 下移完成后插入新条目
        setTimeout(() => {
            // 使用 rAF 批量处理所有变更，保证浏览器在一帧内完成渲染
            requestAnimationFrame(() => {
                // ① 先插入新条目（改变布局，把已有条目往下推）
                insertNewTodoItem(newTodo);
                listEl.style.paddingBottom = '';

                // ② 强制 reflow，确保新条目的布局尺寸已生效到渲染树
                void listEl.offsetHeight;

                // ③ 清除 transform——此时已有条目已被新条目推到正确位置，不会跳动
                existingItems.forEach(el => {
                    el.style.transition = 'none';
                    el.style.transform = '';
                });

                delete listEl.dataset.todoAnimating;
                updateTodoStatsAfterAdd();

                // ④ 下一帧再恢复 transition
                requestAnimationFrame(() => {
                    existingItems.forEach(el => {
                        el.style.transition = '';
                    });
                });
            });
        }, 350);
    } catch (err) {
        console.error('添加待办失败:', err);
    }
}

/**
 * 打开待办输入面板
 */
function openTodoInputPanel() {
    els.todoFabPanel?.classList.add('open');
    els.todoFab?.classList.add('open');
    setTimeout(() => els.todoInput?.focus(), 100);
}

/**
 * 关闭待办输入面板
 */
function closeTodoInputPanel() {
    els.todoFabPanel?.classList.remove('open');
    els.todoFab?.classList.remove('open');
}

/**
 * 构建待办条目的 HTML 字符串（用于新增条目）
 */
function buildTodoItemHTML(todo) {
    return `
        <div class="todo-item" data-id="${todo.id}" data-completed="${todo.done}">
            <button class="todo-checkbox" data-action="toggle">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="20 6 9 17 4 12"/>
                </svg>
            </button>
            <span class="todo-text" data-action="edit">${escapeHtml(todo.text)}</span>
            <button class="todo-delete-btn" data-action="delete" title="删除">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
            </button>
        </div>
    `;
}

/**
 * 直接插入新待办条目到列表顶部（无下移动画，用于 fallback 场景）
 */
function insertNewTodoItem(newTodo) {
    const wrapper = document.createElement('div');
    wrapper.innerHTML = buildTodoItemHTML(newTodo);
    const itemEl = wrapper.firstElementChild;
    itemEl.classList.add('todo-item-enter');
    els.todoList.prepend(itemEl);

    // 入场动画结束后移除类
    itemEl.addEventListener('animationend', () => {
        itemEl.classList.remove('todo-item-enter');
    }, { once: true });
    // tooltip 由事件委托处理
}

/**
 * 新增后异步更新统计
 */
async function updateTodoStatsAfterAdd() {
    try {
        const allTodos = await window.go.main.App.ListTodos();
        updateTodoStats(allTodos);
    } catch (_) {}
}

/**
 * 切换待办完成状态
 * @param {number} id
 */
async function toggleTodo(id) {
    const item = els.todoList?.querySelector(`.todo-item[data-id="${id}"]`);
    if (!item) return;

    const isDone = item.querySelector('.todo-checkbox')?.classList.contains('checked');
    const newDone = !isDone;

    if (_todoFilter === 'all') {
        // "全部"筛选下：原地切换样式，不播 exit anim，移动位置
        setTodoItemDone(item, newDone);

        // 标记完成 → 移到底部；取消完成 → 移到顶部
        if (newDone) {
            els.todoList?.appendChild(item);
        } else {
            els.todoList?.prepend(item);
        }

        // 调 API + 更新统计
        try {
            if (!window.go?.main?.App?.ToggleTodo) return;
            await window.go.main.App.ToggleTodo(id);
            await refreshTodoStats();
        } catch (err) {
            console.error('切换待办状态失败:', err);
        }
        return;
    }

    // 筛选模式下：播 exit anim → 移除
    item.classList.add(isDone ? 'todo-activating' : 'todo-completing');
    await new Promise(r => setTimeout(r, 300));

    // 直接从 DOM 移除
    item.remove();

    try {
        if (!window.go?.main?.App?.ToggleTodo) return;
        await window.go.main.App.ToggleTodo(id);
        await refreshTodoStats();
    } catch (err) {
        console.error('切换待办状态失败:', err);
    }
}

/**
 * 删除待办项
 * @param {number} id
 */
async function deleteTodo(id) {
    const item = els.todoList?.querySelector(`.todo-item[data-id="${id}"]`);
    if (item) {
        item.classList.add('todo-deleting');
        await new Promise(r => setTimeout(r, 300));
        item.remove();
    }

    try {
        if (!window.go?.main?.App?.DeleteTodo) return;
        await window.go.main.App.DeleteTodo(id);
        await refreshTodoStats();
    } catch (err) {
        console.error('删除待办失败:', err);
    }
}

/**
 * 按当前筛选分类清空待办（待办页「清空」按钮）
 * 分类映射：active=清空所有未完成，done=清空所有已完成，all=清空全部
 */
async function clearTodosByFilter() {
    if (!window.go?.main?.App?.ClearTodosByFilter) return;
    const filter = _todoFilter || 'active';
    // 确认文案按分类给出明确范围提示
    const confirmMsg = {
        active: '确定清空所有未完成的待办事项吗？此操作不可恢复。',
        done: '确定清空所有已完成的待办事项吗？此操作不可恢复。',
        all: '确定清空全部待办事项（含已完成和未完成）吗？此操作不可恢复。'
    }[filter] || '确定清空所有已完成的待办事项吗？此操作不可恢复。';

    const confirmed = await showConfirmDialog(confirmMsg, '清空');
    if (!confirmed) return;

    try {
        const msg = await window.go.main.App.ClearTodosByFilter(filter);
        nm.show(msg, 'success');
        await loadDataStats();
        await loadTodos();
    } catch (err) {
        console.error('按分类清空待办失败:', err);
        nm.show('清空失败：' + (err.message || err), 'error');
    }
}

/**
 * 刷新统计 + 空状态（不重渲染列表）
 */
async function refreshTodoStats() {
    try {
        if (!window.go?.main?.App?.ListTodos) return;
        const todos = await window.go.main.App.ListTodos();
        updateTodoStats(todos);

        // 空状态检查
        let filtered = todos;
        if (_todoFilter === 'active') {
            filtered = todos.filter(t => !t.done);
        } else if (_todoFilter === 'done') {
            filtered = todos.filter(t => t.done);
        }
        if (els.todoEmpty) {
            els.todoEmpty.style.display = filtered.length === 0 ? 'flex' : 'none';
        }
    } catch (err) {
        console.error('刷新统计失败:', err);
    }
}

/**
 * 设置待办项完成状态并切换 DOM 样式
 */
function setTodoItemDone(item, done) {
    item.classList.toggle('completed', done);
    item.dataset.completed = done ? 'true' : 'false';
    const checkbox = item.querySelector('.todo-checkbox');
    checkbox?.classList.toggle('checked', done);
    const text = item.querySelector('.todo-text');
    text?.classList.toggle('done', done);
}

/**
 * 编辑待办文本（双击触发）
 * @param {number} id
 */
async function editTodo(id) {
    const item = els.todoList?.querySelector(`.todo-item[data-id="${id}"]`);
    if (!item) return;
    const textEl = item.querySelector('.todo-text');
    if (!textEl) return;

    item.classList.add('editing');

    // 编辑时关闭悬浮预览
    clearTimeout(todoTooltipTimer);
    hideTodoTooltip();

    const oldText = textEl.textContent;
    const textarea = document.createElement('textarea');
    textarea.className = 'todo-text';
    textarea.value = oldText;
    textarea.rows = 4;
    textarea.style.width = '100%';
    textarea.style.border = '1px solid var(--accent)';
    textarea.style.outline = 'none';
    textarea.style.fontSize = '0.875rem';
    textarea.style.fontFamily = 'inherit';
    textarea.style.padding = '10px 12px';
    textarea.style.lineHeight = '1.6';
    textarea.style.borderRadius = 'var(--radius-sm)';
    textarea.style.color = 'var(--text-primary)';
    textarea.style.background = 'var(--card-bg)';
    textarea.style.resize = 'none';
    textarea.style.whiteSpace = 'pre-wrap';
    textarea.style.overflow = 'auto';

    textEl.replaceWith(textarea);
    textarea.focus();
    textarea.select();

    const finishEdit = async (save) => {
        const newText = save ? textarea.value.trim() : oldText;
        const changed = save && newText && newText !== oldText;

        if (changed) {
            try {
                if (window.go?.main?.App?.UpdateTodo) {
                    await window.go.main.App.UpdateTodo(id, newText);
                }
            } catch (err) {
                console.error('编辑待办失败:', err);
            }
        }

        item.classList.remove('editing');
        // 恢复文本显示
        const span = document.createElement('span');
        span.className = 'todo-text';
        if (item.classList.contains('completed')) {
            span.classList.add('done');
        }
        span.textContent = save ? (newText || oldText) : oldText;
        span.ondblclick = () => editTodo(id);
        textarea.replaceWith(span);

        // 内容有变更 → 播放保存动画，更新统计
        if (changed) {
            item.classList.add('todo-saved');
            item.addEventListener('animationend', () => {
                item.classList.remove('todo-saved');
            }, { once: true });

            await refreshTodoStats();
            document.dispatchEvent(new CustomEvent('todos-updated'));
        }
    };

    textarea.addEventListener('keydown', async (e) => {
        if (e.key === 'Enter' && !e.ctrlKey && !e.shiftKey) {
            e.preventDefault();
            await finishEdit(true);
        } else if (e.key === 'Enter' && e.ctrlKey) {
            e.preventDefault();
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            const val = textarea.value;
            textarea.value = val.substring(0, start) + '\n' + val.substring(end);
            textarea.selectionStart = textarea.selectionEnd = start + 1;
        } else if (e.key === 'Escape') {
            await finishEdit(false);
        }
    });

    textarea.addEventListener('blur', () => finishEdit(true));
}

/* ==================== 悬浮预览 Tooltip ==================== */
let todoTooltipEl = null;
let todoTooltipTimer = null;

function showTodoTooltip(item, text, mouseX, mouseY) {
    hideTodoTooltip();

    const el = document.createElement('div');
    el.className = 'todo-tooltip';
    el.textContent = text;
    document.body.appendChild(el);

    // 基于鼠标位置定位
    positionTodoTooltip(mouseX, mouseY, el);

    // 触发回流后添加 visible 类启动动画
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            el.classList.add('visible');
        });
    });

    todoTooltipEl = el;
}

function positionTodoTooltip(mouseX, mouseY, el) {
    const tooltipW = el.offsetWidth;
    const tooltipH = el.offsetHeight;
    const gap = 12;
    const margin = 8;

    // 默认：光标右下方
    let left = mouseX + gap;
    let top = mouseY + gap;

    // 超出右边界 → 显示在光标左侧
    if (left + tooltipW > window.innerWidth - margin) {
        left = mouseX - tooltipW - gap;
    }

    // 超出下边界 → 显示在光标上方
    if (top + tooltipH > window.innerHeight - margin) {
        top = mouseY - tooltipH - gap;
    }

    // 防止左/上溢出
    if (left < margin) left = margin;
    if (top < margin) top = margin;

    el.style.left = left + 'px';
    el.style.top = top + 'px';

    // transform-origin 从光标位置展开
    const originX = left > mouseX ? '0%' : '100%';
    const originY = top > mouseY ? '0%' : '100%';
    el.style.transformOrigin = `${originX} ${originY}`;
}

function hideTodoTooltip() {
    if (todoTooltipEl) {
        todoTooltipEl.classList.remove('visible');
        const el = todoTooltipEl;
        el.addEventListener('transitionend', () => el.remove(), { once: true });
        setTimeout(() => { if (el.parentNode) el.remove(); }, 200);
        todoTooltipEl = null;
    }
}

// 滚动/窗口变化时隐藏 tooltip
window.addEventListener('scroll', hideTodoTooltip, true);
window.addEventListener('resize', hideTodoTooltip);

// 将待办函数暴露到 window（供 data-management.js 等模块调用）
window.addTodo = addTodo;
window.toggleTodo = toggleTodo;
window.deleteTodo = deleteTodo;
window.editTodo = editTodo;
window.loadTodos = loadTodos;

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
window.loadSettings = loadSettings;
window.saveSettings = saveSettings;
window.switchSettingsTab = switchSettingsTab;
window.closeEditorSafe = closeEditorSafe;
window.openShortcuts = openShortcuts;
window.showAbout = showAbout;
window.toggleSidebar = toggleSidebar;
window.toggleBatchMode = toggleBatchMode;
window.resetPagination = resetPagination;
window.loadTrashNotes = loadTrashNotes;
window.updateNotebookSidebarToggleBtn = updateNotebookSidebarToggleBtn;

// 应用启动
document.addEventListener('DOMContentLoaded', init);

