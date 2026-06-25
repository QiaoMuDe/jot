import './style.css';
import './app.css';
import { WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit, EventsOn, WindowFullscreen, WindowUnfullscreen, WindowIsFullscreen } from '../wailsjs/runtime/runtime.js';
import { marked } from 'marked';
import hljs from 'highlight.js';
import 'highlight.js/styles/github.css';

// CodeMirror 6 导入
import { EditorState } from '@codemirror/state';
import { EditorView, lineNumbers, highlightActiveLineGutter, keymap, highlightSpecialChars, drawSelection, highlightActiveLine, placeholder, scrollPastEnd } from '@codemirror/view';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { searchKeymap, highlightSelectionMatches, openSearchPanel, setSearchQuery, SearchQuery } from '@codemirror/search';
import { closeBrackets, closeBracketsKeymap, completionKeymap, autocompletion } from '@codemirror/autocomplete';
import { bracketMatching, indentOnInput, foldGutter, foldKeymap, syntaxHighlighting, HighlightStyle } from '@codemirror/language';
import { markdown } from '@codemirror/lang-markdown';
import { tags } from '@lezer/highlight';

// 配置 marked（breaks + gfm；代码高亮在 updatePreview 中通过 hljs 后处理实现）
marked.setOptions({
    breaks: true,
    gfm: true,
});

/* ===== SVG 图标常量集 ===== */
/**
 * Lucide 风格 SVG 图标常量（24x24 viewBox, 1.5px stroke, currentColor）
 * 所有图标继承父元素 color, 适配明暗主题
 */
const SVGS = {
    // 窗口控制
    windowMinimize: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/></svg>`,
    windowMaximize: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/></svg>`,
    windowRestore: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="7" y="7" width="14" height="14" rx="2" ry="2"/><path d="M5 17V5a2 2 0 0 1 2-2h12"/></svg>`,
    windowClose: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,

    // 编辑器
    editorFullscreen: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg>`,
    editorExitFullscreen: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 14 10 14 10 20"/><polyline points="20 10 14 10 14 4"/><line x1="14" y1="10" x2="21" y2="3"/><line x1="3" y1="21" x2="10" y2="14"/></svg>`,
    edit: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/><path d="m15 5 4 4"/></svg>`,

    // 置顶（星标）
    pinFilled: `<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`,
    pinOutline: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`,

    // 操作按钮
    plus: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`,
    backToTop: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>`,
    menu: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="18" x2="21" y2="18"/></svg>`,
    search: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`,

    // 导航箭头
    chevronDown: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>`,
    chevronUp: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"/></svg>`,
    arrowLeft: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>`,

    // 状态符号
    checkmark: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`,
    xmark: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
    warning: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`,
    info: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`,
    undo: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>`,

    // 数据管理
    download: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    upload: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>`,
    save: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>`,
    folder: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,

    // 杂项
    trash: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>`,
    tag: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>`,
    copy: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`,
};

/* ===== 统一通知系统 ===== */

/**
 * 右上角浮动通知管理器
 * 单例模式，全局可引用
 */
class NotificationManager {
    constructor() {
        this.container = document.getElementById('notificationContainer');
        if (!this.container) {
            this.container = document.createElement('div');
            this.container.id = 'notificationContainer';
            this.container.className = 'notification-container';
            document.body.appendChild(this.container);
        }
    }

    /**
     * 显示通知
     * @param {string} message - 通知内容
     * @param {string} type - 类型：'success' | 'error' | 'warning' | 'info'
     * @param {number} duration - 自动消失毫秒数（默认 3000）
     */
    show(message, type = 'info', duration = 3000) {
        const el = document.createElement('div');
        el.className = `notification ${type}`;

        const iconSvg = { success: SVGS.checkmark, error: SVGS.windowClose, warning: SVGS.warning, info: SVGS.info };
        el.innerHTML = `
            <span class="notification-icon">${iconSvg[type] || SVGS.info}</span>
            <span class="notification-msg">${this._esc(message)}</span>
            <button class="notification-close" aria-label="关闭">${SVGS.windowClose}</button>
        `;

        this.container.appendChild(el);

        // 关闭按钮
        el.querySelector('.notification-close').addEventListener('click', () => this._dismiss(el));

        // 自动消失
        const timer = setTimeout(() => this._dismiss(el), duration);
        el._timer = timer;

        return el;
    }

    /**
     * 显示可撤销通知
     * @param {string} message - 通知内容
     * @param {Function} onUndo - 点击撤销的回调函数
     * @param {number} duration - 自动消失毫秒数（默认 5000）
     */
    showUndo(message, onUndo, duration = 5000) {
        const el = document.createElement('div');
        el.className = 'notification undo';
        el.innerHTML = `
            <span class="notification-icon" style="color:var(--accent,#6366f1)">${SVGS.undo}</span>
            <span class="notification-msg">${this._esc(message)}</span>
            <button class="notification-undo-btn">撤销</button>
        `;

        this.container.appendChild(el);

        // 撤销按钮
        el.querySelector('.notification-undo-btn').addEventListener('click', () => {
            if (typeof onUndo === 'function') onUndo();
            this._dismiss(el);
        });

        // 自动消失
        const timer = setTimeout(() => this._dismiss(el), duration);
        el._timer = timer;

        return el;
    }

    /**
     * 手动销毁通知
     */
    _dismiss(el) {
        if (el._dismissing) return;
        el._dismissing = true;
        clearTimeout(el._timer);
        el.classList.add('exit');
        el.addEventListener('animationend', () => {
            if (el.parentNode) el.parentNode.removeChild(el);
        }, { once: true });
    }

    /**
     * HTML 转义
     */
    _esc(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

/* ===== CodeMirror 6 集成 ===== */

/**
 * CodeMirror 6 编辑器实例（全局单例）
 */
let cmEditor = null;

/**
 * 初始化 CodeMirror 6 编辑器
 * @param {HTMLElement} container - 挂载容器
 * @param {string} content - 初始内容
 * @param {boolean} readOnly - 是否只读
 * @param {boolean} useMdHighlight - 是否启用 MD 语法高亮
 * @returns {EditorView}
 */
function initCodeMirror(container, content = '', readOnly = false, useMdHighlight = true) {
    // 自定义主题：匹配应用 UI 风格
    const jotTheme = EditorView.theme({
        '&': {
            backgroundColor: 'var(--card-bg)',
            color: 'var(--text-primary)',
            fontFamily: 'var(--font-family)',
            fontSize: 'var(--font-size-base)',
            flex: '1 1 0',
            minHeight: 0,
        },
        '.cm-scroller': {
            fontFamily: 'var(--font-family)',
            lineHeight: '1.7',
            overflow: 'auto',
        },
        '.cm-content': {
            caretColor: 'var(--accent)',
            padding: '0',
            fontFamily: 'var(--font-family)',
            fontSize: '0.938rem',
        },
        '.cm-cursor': {
            borderLeftColor: 'var(--accent)',
            borderLeftWidth: '2px',
        },
        '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
            backgroundColor: 'var(--accent-light) !important',
        },
        '.cm-activeLine': {
            backgroundColor: 'rgba(var(--accent-rgb), 0.05)',
        },
        '.cm-gutters': {
            backgroundColor: 'var(--card-bg)',
            border: 'none',
        },
        '.cm-lineNumbers .cm-gutterElement': {
            color: 'var(--text-muted)',
            fontSize: '0.75rem',
            lineHeight: '2.13',
            padding: '0 4px 0 4px',
        },
        '.cm-foldGutter .cm-gutterElement': {
            color: 'var(--text-muted)',
            fontSize: '0.75rem',
            padding: '0 2px 0 8px',
            cursor: 'default',
        },
        '.cm-matchingBracket': {
            backgroundColor: 'var(--accent-light)',
            outline: 'none',
        },
        '&.cm-focused .cm-matchingBracket': {
            backgroundColor: 'var(--accent-light)',
        },
        '.cm-searchMatch': {
            backgroundColor: 'var(--accent-light)',
        },
        '.cm-searchMatch.selected': {
            backgroundColor: 'var(--accent)',
        },
    });

    // Markdown 语法高亮样式（引用 CSS 变量，跟随主题变化）
    const jotHighlightStyle = HighlightStyle.define([
        { tag: tags.heading1, fontSize: '1.5rem', fontWeight: '700', color: 'var(--accent)' },
        { tag: tags.heading2, fontSize: '1.25rem', fontWeight: '700', color: 'var(--accent)' },
        { tag: tags.heading3, fontSize: '1.1rem', fontWeight: '600', color: 'var(--accent)' },
        { tag: tags.heading4, fontSize: '1rem', fontWeight: '600', color: 'var(--text-primary)' },
        { tag: tags.heading5, fontSize: '0.938rem', fontWeight: '600', color: 'var(--text-primary)' },
        { tag: tags.heading6, fontSize: '0.875rem', fontWeight: '600', color: 'var(--text-secondary)' },
        { tag: tags.strong, fontWeight: '700' },
        { tag: tags.emphasis, fontStyle: 'italic' },
        { tag: tags.strikethrough, textDecoration: 'line-through' },
        { tag: tags.link, color: 'var(--accent)', textDecoration: 'underline', cursor: 'pointer' },
        { tag: tags.url, color: 'var(--text-muted)', fontStyle: 'italic' },
        { tag: tags.quote, color: 'var(--text-secondary)', fontStyle: 'italic' },
        { tag: tags.monospace, background: 'var(--hover-bg)', borderRadius: '3px', padding: '1px 4px', fontFamily: 'Consolas, Monaco, monospace', fontSize: '0.85em' },
        { tag: tags.comment, color: 'var(--text-muted)', fontStyle: 'italic' },
        { tag: tags.list, color: 'var(--accent)', fontWeight: '500' },
        { tag: tags.contentSeparator, borderTop: '1px solid var(--border)', display: 'block', margin: '0.5em 0' },
        { tag: tags.escape, color: 'var(--text-muted)', fontWeight: '600' },
        { tag: tags.character, color: 'var(--text-muted)' },
        { tag: tags.labelName, color: 'var(--text-secondary)', fontStyle: 'italic' },
        { tag: tags.string, color: 'var(--text-secondary)' },
        { tag: tags.processingInstruction, color: 'var(--text-muted)', opacity: '0.6' },
    ]);

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
        ...(useMdHighlight ? [markdown(), syntaxHighlighting(jotHighlightStyle)] : []),
        highlightSelectionMatches(),
        EditorView.contentAttributes.of({ spellcheck: 'true' }),
        jotTheme,
        EditorState.readOnly.of(readOnly),
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
    trashNotes: [],
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
    searchInput: $('searchInput'),
    // 更多菜单
    moreMenuBtn: $('moreMenuBtn'),
    moreMenu: $('moreMenu'),

    // 视图
    viewGrid: $('viewGrid'),
    viewSearch: $('viewSearch'),
    viewSettings: $('viewSettings'),
    viewData: $('viewData'),
    viewTrash: $('viewTrash'),
    viewEditor: $('viewEditor'),
    editorPanel: $('editorPanel'),

    cardGrid: $('cardGrid'),
    skeletonGrid: $('skeletonGrid'),
    emptyNotes: $('emptyNotes'),
    emptySearch: $('emptySearch'),

    // 搜索
    searchResults: $('searchResults'),
    resultCount: $('resultCount'),
    searchBackBtn: $('searchBackBtn'),

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

    // 设置
    tagList: $('tagList'),
    quickNoteToggle: $('quickNoteToggle'),
    mdHighlightToggle: $('mdHighlightToggle'),
    newTagName: $('newTagName'),
    newTagColor: $('newTagColor'),
    addTagBtn: $('addTagBtn'),
    // 字体设置
    fontFamilyTrigger: $('fontFamilyTrigger'),
    fontFamilyDisplay: $('fontFamilyDisplay'),
    fontFamilyDropdown: $('fontFamilyDropdown'),
    fontFamilySearch: $('fontFamilySearch'),
    // 主题设置（分段控件）
    themeControl: $('themeControl'),
    themeIndicator: $('themeIndicator'),
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
    restoreAllBtn: $('restoreAllBtn'),
    emptyTrashBtn: $('emptyTrashBtn'),

    // 数据管理
    dataBackBtn: $('dataBackBtn'),
    settingsBackBtn: $('settingsBackBtn'),
    exportDataBtn: $('exportDataBtn'),
    importDataBtn: $('importDataBtn'),
    importResult: $('importResult'),
    resetAllBtn: $('resetAllBtn'),
    openDataDirBtn: $('openDataDirBtn'),
    dataContent: $('dataContent'),
    statTotalNotes: $('statTotalNotes'),
    statTotalTags: $('statTotalTags'),
    statTrashedNotes: $('statTrashedNotes'),
    statTotalNotebooks: $('statTotalNotebooks'),
    statDBSize: $('statDBSize'),

    // 备份还原
    backupBtn: $('backupBtn'),
    restoreBtn: $('restoreBtn'),
    backupInfo: $('backupInfo'),

    // 字数统计
    editorWordCount: $('editorWordCount'),
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

    // 移动到弹窗
    moveNotebookDialog: $('moveNotebookDialog'),
    moveNotebookList: $('moveNotebookList'),
    moveNotebookConfirm: $('moveNotebookConfirm'),
    moveNotebookCancel: $('moveNotebookCancel'),
    moveNotebookClose: $('moveNotebookClose'),
    moveNotebookEmpty: $('moveNotebookEmpty'),
    batchMoveBtn: $('batchMoveBtn'),
};

/* ===== 工具函数 ===== */

/**
 * 格式化时间戳为可读字符串
 */
function formatTime(isoString) {
    if (!isoString) return '';
    const date = new Date(isoString);
    const now = new Date();
    const diff = now - date;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes} 分钟前`;
    if (hours < 24) return `${hours} 小时前`;
    if (days < 7) return `${days} 天前`;

    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
}

/**
 * 高亮搜索关键词
 */
function highlightText(text, keyword) {
    if (!keyword || !text) return text || '';
    const escaped = keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(${escaped})`, 'gi');
    return text.replace(regex, '<span class="highlight">$1</span>');
}

/**
 * 获取纯文本摘要
 */
function getSummary(text, maxLen = 100) {
    if (!text) return '';
    const cleaned = text.replace(/<[^>]*>/g, '').trim();
    return cleaned.length > maxLen ? cleaned.slice(0, maxLen) + '...' : cleaned;
}

/**
 * 防抖工具函数
 */
function debounce(fn, delay) {
    let timer;
    return function (...args) {
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), delay);
    };
}

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
        search: els.viewSearch,
        settings: els.viewSettings,
        data: els.viewData,
        trash: els.viewTrash,
    };
    const targetView = viewMap[view];
    if (!targetView || _viewAnimating) return;

    const currentViewEl = document.querySelector('.view.active');
    if (targetView === currentViewEl) return;

    // 切换视图时强制退出批量模式
    if (state.batchMode) {
        state.batchMode = false;
        els.batchModeBtn.classList.remove('active');
        state.selectedNoteIds.clear();
        els.batchBar.style.display = 'none';
    }

    state.currentView = view;

    // 悬浮操作按钮仅在网格视图显示
    els.fabGroup.style.display = view === 'grid' ? '' : 'none';

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
                loadMdHighlightSetting();
                loadTags();
                break;
            case 'data':
                loadDataStats();
                break;
            case 'trash':
                loadTrashNotes();
                break;
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
            const note = await window.go.main.App.CreateNote(title, content, state.noteType, state.activeNotebookId);
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
                note_type: state.noteType,
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

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.UpdateNote) {
            await window.go.main.App.UpdateNote(id, title, content, state.noteType);
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
                note.note_type = state.noteType;
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
 * 搜索笔记
 */
async function searchNotes(keyword, source) {
    const kw = keyword.trim();
    if (!kw) {
        switchView('grid');
        return;
    }

    state.searchKeyword = kw;
    state.searchSource = source || 'input';

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.SearchNotes) {
            const result = await window.go.main.App.SearchNotes(kw, 1, 100, state.activeNotebookId);
            renderSearchResults((result && result.items) || [], kw);
        } else {
            console.warn('SearchNotes 未绑定，模拟搜索');
            const results = state.notes.filter(
                (n) => n.title.toLowerCase().includes(kw.toLowerCase()) ||
                    (n.content && n.content.toLowerCase().includes(kw.toLowerCase()))
            );
            renderSearchResults(results, kw);
        }
    } catch (err) {
        console.error('搜索失败:', err);
        renderSearchResults([], kw);
    }
    switchView('search');
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
function applyTheme(themeName) {
    document.documentElement.setAttribute('data-theme', themeName);
    // 同步分段控件选中状态和指示器位置
    if (els.themeControl) {
        const btns = els.themeControl.querySelectorAll('.segmented-btn');
        btns.forEach(btn => {
            const isActive = btn.dataset.themeValue === themeName;
            btn.classList.toggle('active', isActive);
            if (isActive && els.themeIndicator) {
                const index = Array.from(btns).indexOf(btn);
                const cw = els.themeControl.offsetWidth;
                const segW = (cw - btns.length * 4) / btns.length;
                els.themeIndicator.style.transform = `translateX(${2 + index * (segW + 4)}px)`;
            }
        });
    }
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

/**
 * 初始化主题设置分段控件事件
 */
function initThemeSettings() {
    if (!els.themeControl) return;

    // 点击分段按钮切换主题
    els.themeControl.querySelectorAll('.segmented-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
            const theme = btn.dataset.themeValue;
            if (!theme) return;
            // 更新 active 状态
            els.themeControl.querySelectorAll('.segmented-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            // 移动指示器
            const btns = Array.from(els.themeControl.querySelectorAll('.segmented-btn'));
            const index = btns.indexOf(btn);
            const cw = els.themeControl.offsetWidth;
            const segW = (cw - btns.length * 4) / btns.length;
            if (els.themeIndicator) {
                els.themeIndicator.style.transform = `translateX(${2 + index * (segW + 4)}px)`;
            }
            // 应用并保存
            applyTheme(theme);
            await saveThemeSetting(theme);

        });
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

/**
 * 加载回收站笔记
 */
async function loadTrashNotes() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetTrashNotes) {
            const result = await window.go.main.App.GetTrashNotes(1, 100);
            state.trashNotes = (result && result.items) || [];
        } else {
            console.warn('GetTrashNotes 未绑定');
            state.trashNotes = [];
        }
    } catch (err) {
        console.error('加载回收站失败:', err);
        state.trashNotes = [];
    }
    renderTrashList();
}

/**
 * 恢复笔记（带动画）
 */
async function restoreNote(id) {
    // 先对 DOM 元素应用 shrinkOut 动画
    const item = els.trashListInner.querySelector(`.trash-item[data-id="${id}"]`);
    if (item) {
        item.style.animation = 'shrinkOut 0.2s ease-out forwards';
        await new Promise(resolve => {
            item.addEventListener('animationend', resolve, { once: true });
        });
    }
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreNote) {
            await window.go.main.App.RestoreNote(id);
        } else {
            console.warn('RestoreNote 未绑定');
        }
    } catch (err) {
        console.error('恢复笔记失败:', err);
    }
    await loadTrashNotes();
    await loadNotes();
    await loadNotebooks();
    nm.show('笔记已恢复', 'success');
}

/**
 * 全部恢复回收站笔记（带动画）
 */
async function restoreAllNotes() {
    if (!state.trashNotes || state.trashNotes.length === 0) {
        nm.show('回收站为空', 'info');
        return;
    }
    const confirmed = await showConfirmDialog('确定要恢复回收站中的所有笔记吗？');
    if (!confirmed) return;

    // 所有项依次收缩淡出（30ms 延迟）
    const items = els.trashListInner.querySelectorAll('.trash-item');
    const restorePromises = Array.from(items).map((item, index) => {
        return new Promise(resolve => {
            item.style.animation = `shrinkOut 0.2s ease-out forwards`;
            item.style.animationDelay = `${index * 30}ms`;
            item.addEventListener('animationend', resolve, { once: true });
        });
    });
    await Promise.all(restorePromises);

    try {
        // 先获取回收站中的笔记 ID，用于撤销操作
        let trashedIds = [];
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetTrashNotes) {
            const result = await window.go.main.App.GetTrashNotes(1, 99999);
            if (result && result.Items) {
                trashedIds = result.Items.map(n => n.ID || n.id);
            }
        }

        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreAllNotes) {
            await window.go.main.App.RestoreAllNotes();
        } else {
            console.warn('RestoreAllNotes 未绑定');
        }

        if (trashedIds.length > 0) {
            nm.showUndo(`已恢复 ${trashedIds.length} 条笔记`, () => undoDelete(trashedIds));
        } else {
            nm.show('已恢复所有笔记', 'success');
        }
    } catch (err) {
        console.error('全部恢复失败:', err);
        nm.show('恢复失败', 'error');
    }
    await loadTrashNotes();
    await loadNotes();
    await loadNotebooks();
}

/**
 * 清空回收站（永久删除所有，带动画）
 */
async function emptyTrash() {
    if (!state.trashNotes || state.trashNotes.length === 0) {
        nm.show('回收站为空', 'info');
        return;
    }
    const confirmed = await showConfirmDialog('确定要永久清空回收站中的所有笔记吗？此操作不可撤销。');
    if (!confirmed) return;

    // 所有项依次收缩淡出 + 红色闪烁（30ms 延迟）
    const items = els.trashListInner.querySelectorAll('.trash-item');
    const emptyPromises = Array.from(items).map((item, index) => {
        return new Promise(resolve => {
            item.style.animation = `shrinkOut 0.25s ease-out forwards, dangerFlash 0.25s ease-out forwards`;
            item.style.animationDelay = `${index * 30}ms`;
            item.addEventListener('animationend', resolve, { once: true });
        });
    });
    await Promise.all(emptyPromises);

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.EmptyTrash) {
            await window.go.main.App.EmptyTrash();
        } else {
            console.warn('EmptyTrash 未绑定');
        }
        nm.show('回收站已清空', 'info');
    } catch (err) {
        console.error('清空回收站失败:', err);
        nm.show('清空失败', 'error');
    }
    await loadTrashNotes();
    await loadNotes();
}

/**
 * 永久删除笔记（带动画）
 */
async function permanentDeleteNote(id) {
    // 先对 DOM 元素应用 shrinkOut 动画 + 红色闪烁
    const item = els.trashListInner.querySelector(`.trash-item[data-id="${id}"]`);
    if (item) {
        item.style.animation = 'shrinkOut 0.25s ease-out forwards, dangerFlash 0.25s ease-out forwards';
        await new Promise(resolve => {
            item.addEventListener('animationend', resolve, { once: true });
        });
    }
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.PermanentDeleteNote) {
            await window.go.main.App.PermanentDeleteNote(id);
        } else {
            console.warn('PermanentDeleteNote 未绑定');
        }
        nm.show('笔记已永久删除', 'info');
    } catch (err) {
        console.error('永久删除失败:', err);
        nm.show('删除失败', 'error');
    }
    await loadTrashNotes();
}

/* ===== 数据管理函数 ===== */

/**
 * 数字递增动画（从 0 渐变到目标值）
 * @param {HTMLElement} element - 显示数字的元素
 * @param {number} targetValue - 目标数值
 * @param {number} duration - 动画时长（毫秒）
 */
function animateCountUp(element, targetValue, duration = 300) {
    const startTime = performance.now();
    const startValue = 0;
    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        // easeOutQuad 缓动
        const eased = 1 - Math.pow(1 - progress, 2);
        const currentValue = Math.floor(startValue + (targetValue - startValue) * eased);
        element.textContent = currentValue;
        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }
    requestAnimationFrame(update);
}

/**
 * 加载数据统计概览
 */
async function loadDataStats() {
    let totalNotes = 0, totalTags = 0, trashedNotes = 0, totalNotebooks = 0, dbSizeStr = '';
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetDataStats) {
            const stats = await window.go.main.App.GetDataStats();
            if (stats) {
                totalNotes = stats.total_notes || 0;
                totalTags = stats.total_tags || 0;
                trashedNotes = stats.trashed_notes || 0;
                totalNotebooks = stats.total_notebooks || 0;
                dbSizeStr = stats.db_size_str || '';
            }
        } else {
            console.warn('GetDataStats 未绑定');
            totalNotes = state.notes.length;
            totalTags = state.tags.length;
        }
    } catch (err) {
        console.error('加载统计数据失败:', err);
    }

    // 统计卡片交错入场动画
    const statCards = els.dataContent.querySelectorAll('.stat-card');
    const totalCards = statCards.length;
    if (totalCards > 0) {
        statCards.forEach((card, index) => {
            card.style.animation = `cardEnter 0.25s ease-out forwards`;
            card.style.animationDelay = `${index * 80}ms`;
        });
    }

    // 先全部设为 0/占位，然后数字递增动画
    els.statTotalNotes.textContent = '0';
    els.statTotalTags.textContent = '0';
    els.statTrashedNotes.textContent = '0';
    els.statTotalNotebooks.textContent = '0';
    els.statDBSize.textContent = dbSizeStr || '0';

    // 入场动画完成后启动 count-up（取最后一张卡片的动画结束时间）
    const lastDelay = (totalCards > 0 ? (totalCards - 1) * 80 : 0) + 250;
    setTimeout(() => {
        animateCountUp(els.statTotalNotes, totalNotes);
        animateCountUp(els.statTotalTags, totalTags);
        animateCountUp(els.statTrashedNotes, trashedNotes);
        animateCountUp(els.statTotalNotebooks, totalNotebooks);
    }, lastDelay + 50);

    // 加载备份信息
    loadBackupInfo();
}

/**
 * 恢复出厂设置：清空所有数据（笔记/标签/设置），重新初始化默认标签
 */
async function resetDatabase() {
    const confirmed = await showConfirmDialog(
        '确定要恢复出厂设置吗？这将永久删除所有笔记、标签和设置，此操作不可撤销。'
    );
    if (!confirmed) return;

    // 二次确认
    const confirmed2 = await showConfirmDialog(
        '再次确认：所有数据将被清空，且无法恢复。确定要继续吗？'
    );
    if (!confirmed2) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ResetDatabase) {
            await window.go.main.App.ResetDatabase();
            nm.show('已恢复出厂设置，所有数据已清空', 'success');
        } else {
            console.warn('ResetDatabase 未绑定');
            nm.show('功能不可用：后端未绑定', 'error');
        }
    } catch (err) {
        console.error('重置数据库失败:', err);
        nm.show('重置失败：' + err.message, 'error');
    }
    await loadDataStats();
    // 重新加载笔记本列表（数据已重置，旧 counts 不再有效）
    await loadNotebooks();
    // 切回首页并刷新笔记列表，确保显示的笔记是最新状态
    switchView('grid');
    loadNotes();
}

/**
 * 在文件管理器中打开数据库目录
 */
async function openDataDir() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenDataDir) {
            await window.go.main.App.OpenDataDir();
        } else {
            nm.show('打开文件管理器功能不可用', 'error');
        }
    } catch (err) {
        console.error('打开数据目录失败:', err);
        nm.show('打开数据目录失败：' + err.message, 'error');
    }
}

/**
 * 导出笔记数据
 */
async function exportData() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportDataWithDialog) {
            const msg = await window.go.main.App.ExportDataWithDialog();
            if (msg && msg !== '已取消') {
                nm.show(msg, 'success');
            }
        } else {
            nm.show('导出功能不可用', 'error');
        }
    } catch (err) {
        console.error('导出数据失败:', err);
        nm.show('导出失败：' + err.message, 'error');
    }
}

/**
 * 导入笔记数据
 */
async function importData() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ImportDatabaseWithDialog) {
            const result = await window.go.main.App.ImportDatabaseWithDialog();
            if (result && result.message !== '已取消') {
                nm.show(result.message, 'success');
                if (result.success_count > 0) {
                    // 刷新所有数据
                    loadNotes();
                    loadDataStats();
                    loadTags();
                }
            }
        } else {
            nm.show('导入功能不可用', 'error');
        }
    } catch (err) {
        console.error('导入数据失败:', err);
        nm.show('导入失败：' + err.message, 'error');
    }
}

/**
 * 加载最新备份信息并更新标签
 */
async function loadBackupInfo() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetBackupInfo) {
            const info = await window.go.main.App.GetBackupInfo();
            if (info && info.file_name) {
                els.backupInfo.innerHTML = `<span style="color:var(--success,#2ea043)">&#10003; 已有备份</span> \u2014 ${info.file_time}，${info.file_size}`;
                els.backupInfo.classList.add('has-backup');
            } else {
                els.backupInfo.textContent = '暂无备份';
                els.backupInfo.classList.remove('has-backup');
            }
        }
    } catch (err) {
        console.error('加载备份信息失败:', err);
        els.backupInfo.textContent = '暂无备份';
        els.backupInfo.classList.remove('has-backup');
    }
}

/**
 * 一键备份（带按钮加载状态）
 */
async function backupToDir() {
    const btn = els.backupBtn;
    const origText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '<span class="dab-text" style="opacity:0.7">⏳ 备份中…</span>';
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.BackupToDir) {
            const msg = await window.go.main.App.BackupToDir();
            if (msg) {
                nm.show(msg, 'success');
                loadBackupInfo();
            }
        } else {
            nm.show('备份功能不可用', 'error');
        }
    } catch (err) {
        console.error('备份失败:', err);
        nm.show('备份失败：' + (err.message || String(err)), 'error');
    } finally {
        btn.disabled = false;
        btn.innerHTML = origText;
    }
}

/**
 * 一键还原（带按钮加载状态 + 确认提示）
 */
async function restoreFromDir() {
    const btn = els.restoreBtn;
    // 自定义确认弹窗
    const confirmed = await showConfirmDialog('确定要从最新备份恢复数据吗？当前所有笔记将被替换为备份内容，此操作不可撤销。');
    if (!confirmed) return;

    const origText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '<span class="dab-text" style="opacity:0.7">⏳ 还原中…</span>';
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreFromDir) {
            const result = await window.go.main.App.RestoreFromDir();
            if (result && result.message) {
                nm.show(result.message, 'success');
                if (result.success_count > 0) {
                    loadNotes();
                    loadDataStats();
                    loadTags();
                }
            }
        } else {
            nm.show('还原功能不可用', 'error');
        }
    } catch (err) {
        console.error('还原失败:', err);
        nm.show('还原失败：' + (err.message || String(err)), 'error');
    } finally {
        btn.disabled = false;
        btn.innerHTML = origText;
    }
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
        <div class="note-card${note.pinned ? ' pinned' : ''}${state.batchMode ? ' batch-mode' : ''}${state.selectedNoteIds.has(note.id) ? ' selected' : ''}" onclick="${state.batchMode ? `window.toggleNoteSelection(${note.id})` : `window.viewNote(${note.id})`}" oncontextmenu="${state.batchMode ? 'event.preventDefault()' : `event.preventDefault(); window.showContextMenu(event, ${note.id})`}">
            <div class="card-body">
                <div class="card-title">${escapeHtml(note.title || '无标题')}</div>
                <div class="card-content">${escapeHtml(getSummary(note.content, 200))}</div>
            </div>
            <div class="card-footer">
                <div class="card-tags">
                    ${(note.tags || [])
                        .map(
                            (tag) =>
                                `<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}" onclick="${state.batchMode ? `event.stopPropagation()` : `event.stopPropagation(); window.searchByTag('${escapeHtml(tag.name)}')`}">${escapeHtml(tag.name)}</span>`
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
 * 渲染搜索结果
 */
function renderSearchResults(results, keyword) {
    els.resultCount.textContent = `找到 ${results.length} 条结果`;

    if (results.length === 0) {
        // 隐藏搜索结果列表，显示空搜索状态
        els.searchResults.style.display = 'none';
        if (els.emptySearch) els.emptySearch.style.display = 'flex';
        return;
    }

    // 有结果时：隐藏空搜索状态，显示搜索结果列表
    if (els.emptySearch) els.emptySearch.style.display = 'none';
    els.searchResults.style.display = '';

    els.searchResults.innerHTML = results
        .map(
            (note) => `
        <div class="search-result-item" onclick="window.viewNote(${note.id})" oncontextmenu="event.preventDefault(); window.showContextMenu(event, ${note.id})">
            <div class="result-title">${highlightText(escapeHtml(note.title || '无标题'), keyword)}</div>
            <div class="result-snippet">${highlightText(escapeHtml(getSummary(note.content, 150)), keyword)}</div>
            <div class="result-time">${formatTime(note.updated_at || note.created_at)}</div>
            ${(note.tags || []).length > 0 ? `
            <div class="result-tags">
                ${(note.tags || []).map(tag =>
                    `<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}" onclick="event.stopPropagation(); window.searchByTag('${escapeHtml(tag.name)}')">${escapeHtml(tag.name)}</span>`
                ).join('')}
            </div>` : ''}
        </div>
        `
        )
        .join('');
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

/**
 * 渲染回收站列表（带交错入场动画）
 */
function renderTrashList() {
    if (state.trashNotes.length === 0) {
        els.trashListInner.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">${SVGS.trash}</div>
                <p class="empty-state-title">回收站为空</p>
                <p class="empty-state-desc">删除的笔记会出现在这里</p>
            </div>
        `;
        return;
    }

    els.trashListInner.innerHTML = state.trashNotes
        .map(
            (note) => `
        <div class="trash-item" data-id="${note.id}">
            <div class="trash-item-info">
                <div class="trash-item-title">${escapeHtml(note.title || '无标题')}</div>
                <div class="trash-item-time">删除于 ${formatTime(note.deleted_at || note.updated_at)}</div>
            </div>
            <div class="trash-item-actions">
                <button class="btn btn-restore btn-sm" onclick="event.stopPropagation(); window.restoreNote(${note.id})">恢复</button>
                <button class="btn btn-perm-delete btn-sm" onclick="event.stopPropagation(); window.permanentDeleteNote(${note.id})">永久删除</button>
            </div>
        </div>
        `
        )
        .join('');

    // 交错入场动画
    const items = els.trashListInner.querySelectorAll('.trash-item');
    items.forEach((item, index) => {
        item.style.animation = `viewEnter 0.25s ease-out forwards`;
        item.style.animationDelay = `${index * 30}ms`;
    });
}

/* ===== 编辑器函数 ===== */

/**
 * 获取编辑器内容
 * @returns {string}
 */
function getEditorContent() {
    return cmEditor ? cmEditor.state.doc.toString() : '';
}

/**
 * 更新字数统计
 */
function updateWordCount() {
    const title = els.editorNoteTitle.value || '';
    const content = getEditorContent();
    const text = title + content;
    const charCount = text.length;
    const wordCount = text.replace(/[\s]/g, '').length;
    els.editorWordCount.textContent = `字数：${wordCount} ｜ 字符：${charCount}`;
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

    // 设置笔记类型
    state.noteType = (noteData && noteData.note_type) || 'text';
    // 更新类型切换按钮文字和 tooltip
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = state.noteType === 'text' ? 'T' : 'M';
        els.editorTypeToggle.title = state.noteType === 'text' ? '切换为 Markdown 格式' : '切换为纯文本格式';
    }
    // 纯文本模式隐藏底部「编辑/预览」切换按钮，Markdown 模式显示
    els.editorModes.style.display = state.noteType === 'markdown' ? '' : 'none';

    // 统一初始化编辑器模式为纯文本编辑（data-mode 值影响 flex 布局 CSS 选择器）
    // 后续各分支根据情况可 override：查看+Markdown → 'preview'
    els.editorOverlay.dataset.mode = 'edit';

    // 查看模式：显示最近编辑时间 + 按类型选择渲染方式
    if (isReadOnly && noteData) {
        els.editorEditTime.textContent = '最近编辑 ' + formatTime(noteData.updated_at || noteData.created_at);
        if (state.noteType === 'text') {
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

    // 字数统计
    updateWordCount();
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
    // 显示编辑器
    els.viewEditor.classList.add('active');

    // 先初始化 CM6（此时编辑器 opacity: 0，用户还看不到）
    // Markdown 笔记始终启用 MD 语法高亮，纯文本笔记根据设置决定
    const useMdHighlight = state.noteType === 'markdown' || els.mdHighlightToggle.checked;
    initCodeMirror(els.editorNoteContent, editorContent, isReadOnly, useMdHighlight);
    // 编辑模式下记录快照，用于蒙层点击判断内容是否有改动
    if (!isReadOnly && state.editingNoteId) {
        state._editSnapshot = {
            title: els.editorNoteTitle.value.trim(),
            content: getEditorContent().trim(),
            tags: [...state.selectedTags].sort()
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
        overlay.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
        panel.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
        // 内容区域延迟入场
        if (body) {
            requestAnimationFrame(() => {
                body.style.animation = 'viewEnter 0.25s ease-out forwards';
                body.style.animationDelay = '50ms';
            });
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
            new URL('./preview-worker.js', import.meta.url),
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
            await window.go.main.App.UpdateNote(state.editingNoteId, title, content, state.noteType);
        } else if (window.go.main.App.CreateNote) {
            await window.go.main.App.CreateNote(title, content, state.noteType, state.activeNotebookId);
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
        if (title && content) {
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
            if (currentTitle === snapshot.title && currentContent === snapshot.content && !tagsChanged) {
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
    openEditor(id, false);
};

/**
 * 查看笔记（只读模式）
 */
window.viewNote = function (id) {
    openEditor(id, true);
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
 * 恢复笔记（全局）
 */
window.restoreNote = async function (id) {
    await restoreNote(id);
};

/**
 * 永久删除笔记（全局）
 */
window.permanentDeleteNote = async function (id) {
    await permanentDeleteNote(id);
};

/**
 * 删除标签（全局）
 */
window.deleteTag = async function (id) {
    await deleteTag(id);
};

/**
 * 按标签搜索（全局）
 */
window.searchByTag = function (tagName) {
    els.searchInput.value = tagName;
    searchNotes(tagName, 'tag');
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

    // 搜索
    // 搜索框回车：非空立即搜索，空内容时刷新笔记
    els.searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            const kw = els.searchInput.value.trim();
            if (kw) {
                // 非空：立即搜索，跳过防抖
                searchNotes(kw, 'input');
            } else {
                // 空：切回网格并刷新
                state.searchKeyword = '';
                switchView('grid');
                resetPagination();
                loadNotes();
            }
        }
    });
    // 搜索框输入自动搜索（防抖 250ms）
    els.searchInput.addEventListener('input', debounce(function () {
        const kw = this.value.trim();
        if (kw) {
            searchNotes(kw, 'input');
        } else {
            state.searchKeyword = '';
            switchView('grid');
            resetPagination();
            loadNotes();
        }
    }, 250));
    els.searchBackBtn.addEventListener('click', () => {
        els.searchInput.value = '';
        state.searchKeyword = '';
        switchView('grid');
    });

    // 浮动新建按钮
    els.fabNewNote.addEventListener('click', () => {
        openEditor(null);
    });

    // 回到顶部
    els.backToTopBtn.addEventListener('click', () => {
        els.mainContent.scrollTo({ top: 0, behavior: 'smooth' });
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
                els.searchInput.value = '';
                switchView('grid');
                resetPagination();
                loadNotes();
            } else if (item.dataset.action === 'sidebar-toggle') {
                toggleSidebar();
                updateSidebarMenuItem();
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
            } else if (item.dataset.action === 'help') {
                openShortcuts();
            }
        }
    });

    // 点击品牌名返回所有笔记
    document.querySelector('.topbar-brand')?.addEventListener('click', () => {
        state.searchKeyword = '';
        els.searchInput.value = '';
        switchView('grid');
    });

    // 编辑器
    els.editorCloseBtn.addEventListener('click', closeEditorSafe);
    els.editorEditBtn.addEventListener('click', () => {
        const noteId = state.editingNoteId;
        if (noteId) {
            state.enteredFromViewMode = true;
            // 原地切换为编辑模式，不走 closeEditor（避免动画 setTimeout 冲突）
            openEditor(noteId, false);
        }
    });
    els.editorViewBtn.addEventListener('click', async () => {
        const noteId = state.editingNoteId;
        if (noteId) {
            // 先保存当前内容再切回查看模式
            const title = els.editorNoteTitle.value.trim();
            const content = getEditorContent().trim();
            if (title && window.go?.main?.App?.UpdateNote) {
                try {
                    await window.go.main.App.UpdateNote(noteId, title, content, state.noteType);
                    // 更新标签
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
            state.enteredFromViewMode = false;
            nm.show('笔记已更新', 'success');
            // 同步更新 state.notes 中的本地缓存，避免 loadNotes() 全量刷新
            const cached = state.notes.find(n => n.id === noteId);
            if (cached) {
                cached.title = title;
                cached.content = content;
                cached.note_type = state.noteType;
                cached.updated_at = new Date().toISOString();
            }
            state._editSnapshot = null;
            openEditor(noteId, true);
            await loadNotes();
        }
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

    // 笔记类型切换（纯文本/Markdown）— 单按钮 T/M 切换
    els.editorTypeToggle.addEventListener('click', () => {
        const newType = state.noteType === 'text' ? 'markdown' : 'text';
        state.noteType = newType;
        // 更新按钮文字和 tooltip
        els.editorTypeToggle.textContent = newType === 'text' ? 'T' : 'M';
        els.editorTypeToggle.title = newType === 'text' ? '切换为 Markdown 格式' : '切换为纯文本格式';
        // 纯文本模式隐藏底部「编辑/预览」切换按钮，Markdown 模式显示
        els.editorModes.style.display = newType === 'markdown' ? '' : 'none';
        // 切到纯文本时自动切回编辑模式
        if (newType === 'text') {
            switchEditorMode('edit');
        }
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
    els.restoreAllBtn.addEventListener('click', restoreAllNotes);
    els.emptyTrashBtn.addEventListener('click', emptyTrash);

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
    els.openDataDirBtn.addEventListener('click', openDataDir);

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

    // 纯文本 MD 高亮开关
    els.mdHighlightToggle.addEventListener('change', async (e) => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
                await window.go.main.App.SetSetting('md_highlight_plain', String(e.target.checked));
                nm.show('设置已保存', 'success');
            } else {
                localStorage.setItem('md_highlight_plain', String(e.target.checked));
                nm.show('设置已保存', 'success');
            }
        } catch (err) {
            console.error('保存 MD 高亮设置失败:', err);
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
}

/* ===== 模拟数据（后端未绑定时使用） ===== */

// Mock 数据的可变副本，确保修改可持久化
let mockNotes = null;

function getMockNotes() {
    return [
        {
            id: 1,
            title: '欢迎使用 jot',
            content: '这是一条示例笔记。你可以在这里记录你的想法、灵感、待办事项等。\n\n点击左侧 "+" 按钮创建新笔记，或点击笔记进行编辑。',
            pinned: true,
            created_at: new Date(Date.now() - 3600000).toISOString(),
            updated_at: new Date(Date.now() - 1800000).toISOString(),
            tags: [
                { id: 1, name: '入门', color: '#6366f1' },
            ],
        },
        {
            id: 2,
            title: '设计思路',
            content: 'jot 是一款简洁的笔记应用，采用双栏布局设计。\n左侧导航栏包含搜索、笔记列表和设置。\n右侧主内容区展示笔记卡片网格。',
            pinned: false,
            created_at: new Date(Date.now() - 86400000).toISOString(),
            updated_at: new Date(Date.now() - 43200000).toISOString(),
            tags: [
                { id: 2, name: '设计', color: '#8b5cf6' },
            ],
        },
        {
            id: 3,
            title: '待办事项',
            content: '- 实现后端 CRUD\n- 实现前端布局\n- 联调测试',
            pinned: false,
            created_at: new Date(Date.now() - 172800000).toISOString(),
            updated_at: new Date(Date.now() - 86400000).toISOString(),
            tags: [
                { id: 3, name: '待办', color: '#f59e0b' },
            ],
        },
    ];
}

function getMockTags() {
    return [
        { id: 1, name: '入门', color: '#6366f1' },
        { id: 2, name: '设计', color: '#8b5cf6' },
        { id: 3, name: '待办', color: '#f59e0b' },
    ];
}

/* ===== 键盘快捷键导航 ===== */

/**
 * 获取当前视图的可滚动容器
 */
function getScrollContainer() {
    switch (state.currentView) {
        case 'grid':
            return els.mainContent;
        case 'search':
            return els.searchResults;
        case 'data':
            return els.dataContent;
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

    // Ctrl+F: 编辑器内搜索（自动填充选中文本，预览模式自动切到编辑模式）
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
        els.searchInput.focus();
        els.searchInput.select();
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
            openEditor(null);
        }
        return;
    }

    // Ctrl+L: 编辑器打开时切换编辑/预览模式（仅 Markdown 模式支持）
    if (e.ctrlKey && (e.key === 'l' || e.key === 'L') && els.viewEditor.classList.contains('active') && state.noteType === 'markdown') {
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
            // 搜索页：清空搜索框后回到首页
            els.searchInput.value = '';
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
                els.searchInput.value = '';
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
                openShortcuts();
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
    const containers = [els.mainContent, els.searchResults, els.dataContent].filter(Boolean);
    containers.forEach((container) => {
        let timer = null;
        container.addEventListener('scroll', () => {
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
        { key: 'Ctrl + F', desc: '编辑器内查找 / 聚焦搜索框' },
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
    ];
    els.shortcutsBody.innerHTML = shortcuts.map(s => `
        <div class="shortcut-row">
            <div class="shortcut-key">${s.key.replace(/(\w+)/g, '<kbd>$1</kbd>')}</div>
            <div class="shortcut-desc">${s.desc}</div>
        </div>
    `).join('');
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
    els.searchInput.value = '';
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
    const checkboxText = `同时永久删除该笔记本中的 ${state.notes.length} 条笔记（不进回收站）`;

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
            els.searchInput.value = '';
            switchView('grid');
            resetPagination();
        }
        await loadNotebooks();
        await loadNotes();
        const suffix = deleteNotes ? '及其笔记已永久删除' : '已删除';
        nm.show(`笔记本「${els.notebookList.querySelector(`[data-notebook-id="${notebookId}"]`)?.querySelector('.notebook-name')?.textContent || ''}」${suffix}`, 'success');
    } catch (err) {
        const msg = (typeof err === 'string' ? err : err?.message || '删除笔记本失败');
        console.error('删除笔记本失败:', msg);
        nm.show(msg, 'error');
    }
}

/**
 * 切换侧栏折叠/展开
 */
function toggleSidebar() {
    const sidebar = els.notebookSidebar;
    if (!sidebar) return;
    const isCollapsed = sidebar.classList.toggle('collapsed');
    // localStorage 持久化
    try {
        localStorage.setItem('jot_sidebar_collapsed', String(isCollapsed));
    } catch (e) {}
}

/**
 * 更新菜单项文字：展开侧栏 ↔ 折叠侧栏
 */
function updateSidebarMenuItem() {
    const menuItem = els.moreMenu?.querySelector('[data-action="sidebar-toggle"]');
    if (!menuItem) return;
    const isCollapsed = els.notebookSidebar?.classList.contains('collapsed');
    const label = isCollapsed ? '展开侧栏' : '折叠侧栏';
    menuItem.textContent = label;
    menuItem.title = `Ctrl+2 ${label}`;
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
    } catch (e) {}
}

/* ===== 初始化 ===== */

async function init() {
    initEventListeners();
    initFontSettings();
    initThemeSettings();
    initScrollLoading();
    initScrollbarAutoHide();
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
    // 快速笔记设置需在 loadNotes 之前加载，启用时先显示全屏编辑器再后台加载笔记
    await loadQuickNoteSetting();
    await loadMdHighlightSetting();
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
        let lastNoteId = null;

        for (const result of results) {
            const label = result.path ? result.path.split(/[/\\]/).pop() || '文件' : '文件';
            if (result.error && result.error.includes('文件夹')) {
                // 后端 stat 检测到目录
                failCount++;
                nm.show(`${label} ${result.error}`, 'warning');
            } else if (result.success) {
                successCount++;
                lastNoteId = result.note_id;
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
        if (lastNoteId) openEditor(lastNoteId, true);
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
 * 加载纯文本 MD 语法高亮设置
 */
async function loadMdHighlightSetting() {
    try {
        let enabled = true;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
            const val = await window.go.main.App.GetSetting('md_highlight_plain');
            enabled = val !== 'false'; // 默认启用
        } else {
            const local = localStorage.getItem('md_highlight_plain');
            enabled = local !== 'false';
        }
        els.mdHighlightToggle.checked = enabled;
    } catch (err) {
        console.error('加载 MD 高亮设置失败:', err);
    }
}

// 应用启动
document.addEventListener('DOMContentLoaded', init);

