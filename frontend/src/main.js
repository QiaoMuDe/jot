import './style.css';
import './app.css';
import { marked } from 'marked';
import hljs from 'highlight.js/lib/core';
import javascript from 'highlight.js/lib/languages/javascript';
import python from 'highlight.js/lib/languages/python';
import css from 'highlight.js/lib/languages/css';
import html from 'highlight.js/lib/languages/xml';
import bash from 'highlight.js/lib/languages/bash';
import json from 'highlight.js/lib/languages/json';
import 'highlight.js/styles/github.css';

// 注册常用语言
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('js', javascript);
hljs.registerLanguage('python', python);
hljs.registerLanguage('py', python);
hljs.registerLanguage('css', css);
hljs.registerLanguage('html', html);
hljs.registerLanguage('xml', html);
hljs.registerLanguage('bash', bash);
hljs.registerLanguage('sh', bash);
hljs.registerLanguage('json', json);

// 配置 marked 使用 hljs 高亮代码块
marked.setOptions({
    highlight: function (code, lang) {
        if (lang && hljs.getLanguage(lang)) {
            try {
                return hljs.highlight(code, { language: lang }).value;
            } catch (e) { }
        }
        return hljs.highlightAuto(code).value;
    },
    breaks: true,
    gfm: true,
});

/* ===== 应用状态 ===== */
const state = {
    notes: [],
    tags: [],
    trashNotes: [],
    currentView: 'grid',       // grid | search | settings | data | trash
    editingNoteId: null,        // null = 新建, number = 编辑
    selectedTags: [],
    searchKeyword: '',
    searchSource: 'input',      // 'input' | 'tag' — 搜索触发来源
    batchMode: false,           // 是否处于批量管理模式
    selectedNoteIds: new Set(), // 选中的笔记 ID 集合
    totalAllNotes: 0,           // 所有未删除笔记的总数（用于全选判断）
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
    newNoteBtn: $('newNoteBtn'),
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
    editorFullscreenBtn: $('editorFullscreenBtn'),
    editorCancelBtn: $('editorCancelBtn'),
    editorSaveBtn: $('editorSaveBtn'),
    mdRendered: $('mdRendered'),

    // 设置
    tagList: $('tagList'),
    quickNoteToggle: $('quickNoteToggle'),
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
    dataContent: $('dataContent'),
    statTotalNotes: $('statTotalNotes'),
    statTotalTags: $('statTotalTags'),
    statTrashedNotes: $('statTrashedNotes'),

    // 字数统计
    editorWordCount: $('editorWordCount'),
    autoSaveIndicator: $('autoSaveIndicator'),
    editorEditTime: $('editorEditTime'),

    // 撤销 Toast
    undoToast: $('undoToast'),
    undoToastMsg: $('undoToastMsg'),
    undoToastBtn: $('undoToastBtn'),

    // 右键菜单
    contextMenu: $('contextMenu'),

    // 确认对话框
    confirmDialog: $('confirmDialog'),
    confirmDialogMsg: $('confirmDialogMsg'),
    confirmOkBtn: $('confirmOkBtn'),
    confirmCancelBtn: $('confirmCancelBtn'),

    // 主内容区（用于网格视图滚动）
    mainContent: $('mainContent'),

    // 批量操作
    batchModeBtn: $('batchModeBtn'),
    batchBar: $('batchBar'),
    batchCount: $('batchCount'),
    batchDeleteBtn: $('batchDeleteBtn'),
    batchCancelBtn: $('batchCancelBtn'),
    batchSelectAllBtn: $('batchSelectAllBtn'),

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

/**
 * 切换右侧主内容区视图
 */
function switchView(view) {
    // 切换视图时强制退出批量模式
    if (state.batchMode) {
        state.batchMode = false;
        els.batchModeBtn.classList.remove('active');
        state.selectedNoteIds.clear();
        els.batchBar.style.display = 'none';
    }

    // 隐藏所有视图
    document.querySelectorAll('.view').forEach((v) => {
        v.classList.remove('active');
    });

    state.currentView = view;

    // 新建按钮和批量管理按钮仅在笔记网格视图显示
    els.newNoteBtn.style.display = view === 'grid' ? 'flex' : 'none';
    els.batchModeBtn.style.display = view === 'grid' ? '' : 'none';
    // 悬浮操作按钮仅在网格视图显示
    els.fabGroup.style.display = view === 'grid' ? '' : 'none';

    switch (view) {
        case 'grid':
            els.viewGrid.classList.add('active');
            break;
        case 'search':
            els.viewSearch.classList.add('active');
            break;
        case 'settings':
            els.viewSettings.classList.add('active');
            loadFontSettings();
            loadThemeSetting();
            loadSortSettings();
            loadPageSizeSetting();
            loadTags();
            break;
        case 'data':
            els.viewData.classList.add('active');
            loadDataStats();
            break;
        case 'trash':
            els.viewTrash.classList.add('active');
            loadTrashNotes();
            break;
    }
}

/* ===== Wails 后端调用封装 ===== */

/**
 * 加载笔记列表（第 1 页，重置分页）
 */
async function loadNotes() {
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
            result = await window.go.main.App.GetNotes(1, pageSize, sortBy);
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
            result = await window.go.main.App.GetNotes(nextPage, pageSize, sortBy);
        }

        if (result && result.items && result.items.length > 0) {
            state.notes = state.notes.concat(result.items);
            currentPage = nextPage;
            hasMoreNotes = state.notes.length < totalNotes;
        } else {
            hasMoreNotes = false;
        }

        renderCardGrid();
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
                const result = await window.go.main.App.GetNotes(page, pageSize, sortBy);
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
        renderCardGrid();
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
    const content = els.editorNoteContent.value.trim();
    if (!title) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateNote) {
            const note = await window.go.main.App.CreateNote(title, content);
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
                pinned: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                tags: state.tags.filter((t) => state.selectedTags.includes(t.id)),
            });
        }
    } catch (err) {
        console.error('创建笔记失败:', err);
    }
    closeEditor();
    await loadNotes();
}

/**
 * 更新笔记
 */
async function updateNote(id) {
    const title = els.editorNoteTitle.value.trim();
    const content = els.editorNoteContent.value.trim();
    if (!title) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.UpdateNote) {
            await window.go.main.App.UpdateNote(id, title, content);
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
    closeEditor();
    await loadNotes();
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
    showUndoToast('笔记已删除', id);
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
        showUndoToast('已复制到剪贴板');
    } catch (err) {
        console.error('复制失败:', err);
        showUndoToast('复制失败');
    }
}

/**
 * 显示撤销 Toast
 * @param {string} msg - 提示信息
 * @param {number|number[]} [noteIds] - 可撤销的笔记 ID（不传则为纯提示）
 */
let undoToastTimer = null;
let undoNoteId = null;
let toastTimer = null;

function showUndoToast(msg, noteIds) {
    // 清除之前的定时器
    if (undoToastTimer) {
        clearTimeout(undoToastTimer);
        undoToastTimer = null;
    }

    els.undoToastMsg.textContent = msg;
    undoNoteId = noteIds || null;

    // 有可撤销的笔记 ID 时显示按钮，否则隐藏
    els.undoToastBtn.style.display = noteIds ? '' : 'none';
    els.undoToast.classList.add('active');

    // 5 秒后自动消失
    undoToastTimer = setTimeout(() => {
        els.undoToast.classList.remove('active');
        undoNoteId = null;
        undoToastTimer = null;
    }, 5000);
}

/**
 * 通用 Toast 提示（自动 3 秒消失）
 * @param {string} msg - 提示内容
 */
function showToast(msg) {
    // 复用 undo-toast 的 UI，隐藏撤销按钮
    if (toastTimer) clearTimeout(toastTimer);
    els.undoToastMsg.textContent = msg;
    els.undoToastBtn.style.display = 'none';
    els.undoToast.classList.add('active');
    toastTimer = setTimeout(() => {
        els.undoToast.classList.remove('active');
        toastTimer = null;
    }, 3000);
}

/**
 * 撤销删除（支持单条和批量）
 */
async function undoDelete() {
    if (undoNoteId == null) return;
    try {
        if (Array.isArray(undoNoteId)) {
            // 批量撤销
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchRestoreNotes) {
                await window.go.main.App.BatchRestoreNotes(undoNoteId);
            } else {
                console.warn('BatchRestoreNotes 未绑定');
            }
        } else {
            // 单条撤销
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreNote) {
                await window.go.main.App.RestoreNote(undoNoteId);
            } else {
                console.warn('RestoreNote 未绑定');
            }
        }
    } catch (err) {
        console.error('撤销删除失败:', err);
    }
    // 关闭 Toast
    els.undoToast.classList.remove('active');
    if (undoToastTimer) {
        clearTimeout(undoToastTimer);
        undoToastTimer = null;
    }
    undoNoteId = null;
    await loadNotes();
}

/**
 * 显示自定义确认对话框
 * @param {string} msg - 提示信息
 * @returns {Promise<boolean>}
 */
function showConfirmDialog(msg) {
    return new Promise((resolve) => {
        els.confirmDialogMsg.textContent = msg;
        els.confirmDialog.style.display = 'flex';

        const cleanup = (result) => {
            els.confirmDialog.style.display = 'none';
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
            const result = await window.go.main.App.SearchNotes(kw, 1, 100);
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
 */
async function togglePin(id) {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.TogglePinNote) {
            await window.go.main.App.TogglePinNote(id);
        } else {
            console.warn('TogglePinNote 未绑定，模拟置顶切换');
            const note = state.notes.find((n) => n.id === id);
            if (note) note.pinned = !note.pinned;
        }
    } catch (err) {
        console.error('置顶切换失败:', err);
    }
    await loadNotes();
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
        showToast('该标签已存在');
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
        showToast('创建标签失败');
    }
    els.newTagName.value = '';
    await loadTags();
}

/* ===== 字体设置函数 ===== */

/**
 * 加载已保存的字体设置并应用到页面
 */
async function loadFontSettings() {
    let fontFamily = 'DM Sans';
    let fontSize = 16;

    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetSetting) {
        const savedFamily = await window.go.main.App.GetSetting('font_family');
        if (savedFamily) fontFamily = savedFamily;
        const savedSize = await window.go.main.App.GetSetting('font_size');
        if (savedSize) fontSize = parseInt(savedSize, 10);
    } else {
        // 从 localStorage 回退
        fontFamily = localStorage.getItem('jot_font_family') || 'DM Sans';
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

    // 渲染字体族下拉选项
    renderFontFamilyOptions(fontFamily);
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
                'DM Sans', 'Arial', 'Helvetica', 'Verdana', 'Georgia',
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
    document.documentElement.style.setProperty('--font-family', `${fontFamily}, system-ui, -apple-system, sans-serif`);
    els.fontFamilyDisplay.textContent = fontFamily;
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
 * 恢复笔记
 */
async function restoreNote(id) {
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
    showToast('笔记已恢复');
}

/**
 * 全部恢复回收站笔记
 */
async function restoreAllNotes() {
    if (!state.trashNotes || state.trashNotes.length === 0) {
        showUndoToast('回收站为空');
        return;
    }
    const confirmed = await showConfirmDialog('确定要恢复回收站中的所有笔记吗？');
    if (!confirmed) return;
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
            showUndoToast(`已恢复 ${trashedIds.length} 条笔记`, trashedIds);
        } else {
            showUndoToast('已恢复所有笔记');
        }
    } catch (err) {
        console.error('全部恢复失败:', err);
        showUndoToast('恢复失败');
    }
    await loadTrashNotes();
    await loadNotes();
}

/**
 * 清空回收站（永久删除所有）
 */
async function emptyTrash() {
    if (!state.trashNotes || state.trashNotes.length === 0) {
        showUndoToast('回收站为空');
        return;
    }
    const confirmed = await showConfirmDialog('确定要永久清空回收站中的所有笔记吗？此操作不可撤销。');
    if (!confirmed) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.EmptyTrash) {
            await window.go.main.App.EmptyTrash();
        } else {
            console.warn('EmptyTrash 未绑定');
        }
        showUndoToast('回收站已清空');
    } catch (err) {
        console.error('清空回收站失败:', err);
        showUndoToast('清空失败');
    }
    await loadTrashNotes();
    await loadNotes();
}

/**
 * 永久删除笔记
 */
async function permanentDeleteNote(id) {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.PermanentDeleteNote) {
            await window.go.main.App.PermanentDeleteNote(id);
        } else {
            console.warn('PermanentDeleteNote 未绑定');
        }
        showUndoToast('笔记已永久删除');
    } catch (err) {
        console.error('永久删除失败:', err);
        showUndoToast('删除失败');
    }
    await loadTrashNotes();
}

/* ===== 数据管理函数 ===== */

/**
 * 加载数据统计概览
 */
async function loadDataStats() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetDataStats) {
            const stats = await window.go.main.App.GetDataStats();
            if (stats) {
                els.statTotalNotes.textContent = stats.total_notes;
                els.statTotalTags.textContent = stats.total_tags;
                els.statTrashedNotes.textContent = stats.trashed_notes;
            }
        } else {
            console.warn('GetDataStats 未绑定');
            // 显示模拟数据
            els.statTotalNotes.textContent = state.notes.length;
            els.statTotalTags.textContent = state.tags.length;
        }
    } catch (err) {
        console.error('加载统计数据失败:', err);
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
                showImportResult({ success_count: 0, fail_count: 0, skipped_count: 0, message: msg });
            }
        } else {
            console.warn('ExportDataWithDialog 未绑定');
            // 降级：前端导出并下载
            const mockExport = state.notes.map(n => ({
                title: n.title,
                content: n.content,
                pinned: n.pinned,
                tags: (n.tags || []).map(t => ({ name: t.name, color: t.color })),
                created_at: n.created_at,
                updated_at: n.updated_at,
            }));
            const jsonStr = JSON.stringify(mockExport, null, 2);
            const blob = new Blob([jsonStr], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `jot-notes-${new Date().toISOString().slice(0, 10)}.json`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }
    } catch (err) {
        console.error('导出数据失败:', err);
        showImportResult({ success_count: 0, fail_count: 0, skipped_count: 0, message: '导出失败：' + err.message });
    }
}

/**
 * 导入笔记数据
 */
async function importData() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.addEventListener('change', async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        try {
            const text = await file.text();

            if (window.go && window.go.main && window.go.main.App && window.go.main.App.ImportData) {
                const result = await window.go.main.App.ImportData(text);
                if (result) {
                    showImportResult(result);
                }
            } else {
                console.warn('ImportData 未绑定');
                alert('导入功能需要在后端绑定时可用');
            }
        } catch (err) {
            console.error('导入数据失败:', err);
            showImportResult({ success_count: 0, fail_count: 0, message: '导入失败：' + err.message });
        }
    });
    input.click();
}

/**
 * 显示导入结果
 */
function showImportResult(result) {
    const el = els.importResult;
    el.style.display = 'block';

    // 导出成功消息
    if (result.message && result.success_count === 0 && result.fail_count === 0 && !result.skipped_count) {
        el.className = 'import-result success';
        el.textContent = result.message;
        setTimeout(() => { el.style.display = 'none'; }, 5000);
        return;
    }

    if (result.fail_count > 0 && result.success_count === 0) {
        el.className = 'import-result error';
        el.textContent = result.message || '导入失败';
    } else {
        el.className = 'import-result success';
        let text = `导入完成：成功 ${result.success_count} 条`;
        if (result.skipped_count > 0) {
            text += `，跳过 ${result.skipped_count} 条（已存在）`;
        }
        if (result.fail_count > 0) {
            text += `，失败 ${result.fail_count} 条`;
        }
        el.textContent = text;
    }
    setTimeout(() => { el.style.display = 'none'; }, 5000);
}

/* ===== 渲染函数 ===== */

/**
 * 渲染卡片网格/列表
 */
function renderCardGrid() {
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
        els.cardGrid.innerHTML = `
            <div class="empty-state" style="grid-column: 1 / -1;">
                <div class="empty-icon">📝</div>
                <div class="empty-title">还没有笔记</div>
                <div class="empty-desc">点击左侧 "+" 按钮创建你的第一条笔记</div>
            </div>
        `;
        return;
    }

    els.cardGrid.innerHTML = sorted
        .map(
            (note) => `
        <div class="note-card${state.batchMode ? ' batch-mode' : ''}${state.selectedNoteIds.has(note.id) ? ' selected' : ''}" onclick="${state.batchMode ? `window.toggleNoteSelection(${note.id})` : `window.viewNote(${note.id})`}" oncontextmenu="${state.batchMode ? 'event.preventDefault()' : `event.preventDefault(); window.showContextMenu(event, ${note.id})`}">
            ${note.pinned ? '<div class="card-pin-badge">★</div>' : ''}
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
                    : `<button class="card-action-btn" onclick="event.stopPropagation(); window.togglePin(${note.id})" title="${note.pinned ? '取消置顶' : '置顶'}">
                           ${note.pinned ? '★' : '☆'}
                       </button>`
                }
            </div>
        </div>
        `
        )
        .join('');

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
        els.searchResults.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🔍</div>
                <div class="empty-title">未找到相关笔记</div>
                <div class="empty-desc">尝试其他关键词搜索</div>
            </div>
        `;
        return;
    }

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
            <button class="tag-delete-btn" onclick="window.deleteTag(${tag.id})">✕</button>
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
        // 只读模式：仅展示标签，不可切换
        els.tagSelector.innerHTML = state.tags
            .map(
                (tag) => `
            <span class="card-tag" style="background-color: ${tag.color || '#6366f1'}; cursor: default;"
                  data-tag-id="${tag.id}">
                ${escapeHtml(tag.name)}
            </span>
            `
            )
            .join('');
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
 * 渲染回收站列表
 */
function renderTrashList() {
    if (state.trashNotes.length === 0) {
        els.trashListInner.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🗑️</div>
                <div class="empty-title">回收站为空</div>
                <div class="empty-desc">删除的笔记会出现在这里</div>
            </div>
        `;
        return;
    }

    els.trashListInner.innerHTML = state.trashNotes
        .map(
            (note) => `
        <div class="trash-item">
            <div class="trash-item-info">
                <div class="trash-item-title">${escapeHtml(note.title || '无标题')}</div>
                <div class="trash-item-time">删除于 ${formatTime(note.deleted_at || note.updated_at)}</div>
            </div>
            <div class="trash-item-actions">
                <button class="btn btn-restore btn-sm" onclick="window.restoreNote(${note.id})">恢复</button>
                <button class="btn btn-perm-delete btn-sm" onclick="window.permanentDeleteNote(${note.id})">永久删除</button>
            </div>
        </div>
        `
        )
        .join('');
}

/* ===== 编辑器函数 ===== */

let autoSaveTimer = null;

/**
 * 更新字数统计
 */
function updateWordCount() {
    const title = els.editorNoteTitle.value || '';
    const content = els.editorNoteContent.value || '';
    const text = title + content;
    const charCount = text.length;
    const wordCount = text.replace(/[\s]/g, '').length;
    els.editorWordCount.textContent = `字数：${wordCount} ｜ 字符：${charCount}`;
}

/**
 * 启动自动保存（3 秒防抖）
 */
function startAutoSave() {
    if (autoSaveTimer) {
        clearTimeout(autoSaveTimer);
        autoSaveTimer = null;
    }
    // 新创建的笔记（无 ID）不自动保存
    if (!state.editingNoteId) return;
    autoSaveTimer = setTimeout(async () => {
        const title = els.editorNoteTitle.value.trim();
        if (!title) return;
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.UpdateNote) {
                await window.go.main.App.UpdateNote(
                    state.editingNoteId,
                    title,
                    els.editorNoteContent.value || ''
                );
                // 显示自动保存成功指示
                els.autoSaveIndicator.textContent = '已保存 ✓';
                els.autoSaveIndicator.className = 'auto-save-indicator active saved';
                setTimeout(() => {
                    els.autoSaveIndicator.className = 'auto-save-indicator';
                }, 2000);
            }
        } catch (err) {
            console.error('自动保存失败:', err);
            els.autoSaveIndicator.textContent = '保存失败';
            els.autoSaveIndicator.className = 'auto-save-indicator active';
            setTimeout(() => {
                els.autoSaveIndicator.className = 'auto-save-indicator';
            }, 2000);
        }
        autoSaveTimer = null;
    }, 3000);
}

/**
 * 打开编辑器（新建/编辑/查看）
 * @param {number|null} noteId - 笔记 ID，null 表示新建
 * @param {boolean} readOnly - 是否为只读查看模式
 */
async function openEditor(noteId, readOnly) {
    state.editingNoteId = noteId || null;
    state.selectedTags = [];

    const isReadOnly = readOnly && noteId != null;
    let noteData = null;

    if (noteId) {
        const note = state.notes.find((n) => n.id === noteId);
        noteData = note;
        if (note) {
            // 查看/编辑模式
            els.editorNoteTitle.value = note.title || '';
            els.editorNoteContent.value = note.content || '';
            state.selectedTags = (note.tags || []).map((t) => t.id);
        } else {
            document.getElementById('colorPicker').value = '#6366f1';
        }
    } else {
        // 新建模式
        els.editorNoteTitle.value = '';
        els.editorNoteContent.value = '';
    }

    // 只读模式：禁用输入，隐藏保存/取消按钮
    els.editorNoteTitle.readOnly = isReadOnly;
    els.editorNoteContent.readOnly = isReadOnly;
    els.editorNoteTitle.classList.toggle('editor-input-readonly', isReadOnly);
    els.editorNoteContent.classList.toggle('editor-textarea-readonly', isReadOnly);
    els.editorSaveBtn.style.display = isReadOnly ? 'none' : '';
    els.editorCancelBtn.style.display = isReadOnly ? 'none' : '';
    els.editorPanel.classList.toggle('editor-view-mode', isReadOnly);

    // 查看模式：显示最近编辑时间 + Markdown 渲染
    if (isReadOnly && noteData) {
        els.editorEditTime.textContent = '最近编辑 ' + formatTime(noteData.updated_at || noteData.created_at);
        // 渲染 Markdown
        const content = noteData.content || '';
        if (content.trim()) {
            els.mdRendered.innerHTML = marked.parse(content);
        } else {
            els.mdRendered.innerHTML = '<p class="md-empty">暂无内容</p>';
        }
        // 隐藏 textarea，显示渲染视图
        els.editorNoteContent.style.display = 'none';
        els.mdRendered.style.display = 'block';
    } else {
        els.editorEditTime.textContent = '';
        // 确保 textarea 可见，渲染视图隐藏
        els.editorNoteContent.style.display = '';
        els.mdRendered.style.display = 'none';
    }

    // 字数统计
    updateWordCount();
    // 重置自动保存指示器
    els.autoSaveIndicator.className = 'auto-save-indicator';
    els.autoSaveIndicator.textContent = '';
    // 非只读模式下监听输入变化
    if (!isReadOnly) {
        els.editorNoteTitle.addEventListener('input', onEditorInput);
        els.editorNoteContent.addEventListener('input', onEditorInput);
    } else {
        els.editorNoteTitle.removeEventListener('input', onEditorInput);
        els.editorNoteContent.removeEventListener('input', onEditorInput);
    }

    // 加载标签并渲染标签选择器（等待标签加载完毕再显示编辑器，避免闪烁）
    await loadTagsForEditor(isReadOnly);
    // 锁定主内容区滚动，隐藏滚动条槽位
    els.mainContent.style.overflow = 'hidden';
    // 显示编辑器
    els.viewEditor.classList.add('active');
}

/**
 * 编辑器输入事件处理：更新字数 + 触发自动保存
 */
function onEditorInput() {
    updateWordCount();
    startAutoSave();
}

/**
 * 关闭编辑器
 */
/**
 * 切换编辑器全屏模式
 */
function toggleEditorFullscreen() {
    const panel = els.editorPanel;
    const btn = els.editorFullscreenBtn;
    panel.classList.toggle('fullscreen');
    const isFullscreen = panel.classList.contains('fullscreen');
    btn.textContent = isFullscreen ? '⤡' : '⛶';
    btn.title = isFullscreen ? '退出全屏' : '全屏编辑';
    btn.classList.toggle('fullscreen', isFullscreen);
}

function closeEditor() {
    els.viewEditor.classList.remove('active');
    // 恢复主内容区滚动
    els.mainContent.style.overflow = '';
    // 退出全屏模式
    els.editorPanel.classList.remove('fullscreen');
    els.editorFullscreenBtn.textContent = '⛶';
    els.editorFullscreenBtn.title = '全屏编辑';
    els.editorFullscreenBtn.classList.remove('fullscreen');
    // 清理事件监听
    els.editorNoteTitle.removeEventListener('input', onEditorInput);
    els.editorNoteContent.removeEventListener('input', onEditorInput);
    // 清除自动保存定时器
    if (autoSaveTimer) {
        clearTimeout(autoSaveTimer);
        autoSaveTimer = null;
    }
    state.editingNoteId = null;
    state.selectedTags = [];
    // 字数归零
    els.editorWordCount.textContent = '';
    // 重置 Markdown 渲染/编辑显示状态
    els.editorNoteContent.style.display = '';
    els.mdRendered.style.display = 'none';
    els.mdRendered.innerHTML = '';
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
    els.contextMenu.classList.remove('active');
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
    menu.style.left = event.clientX + 'px';
    menu.style.top = event.clientY + 'px';
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
    }
};

/**
 * 置顶切换（全局）
 */
window.togglePin = async function (id) {
    await togglePin(id);
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
    if (!state.batchMode) {
        // 退出批量模式：清空选中
        clearSelection();
    }
    els.batchModeBtn.classList.toggle('active', state.batchMode);
    renderCardGrid();
    // 进入批量模式时显示 bar，退出时隐藏
    if (state.batchMode) {
        els.batchBar.style.display = 'flex';
    } else {
        els.batchBar.style.display = 'none';
    }
    updateBatchBar();
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
    renderCardGrid();
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
        renderCardGrid();
    } else {
        // 全选：先从后端拉取所有 ID，再塞入选中的 Set
        selectAllIds();
    }
}

/**
 * 全选：获取所有未删除笔记的 ID
 */
async function selectAllIds() {
    let ids = [];
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllNoteIDs) {
        try {
            ids = await window.go.main.App.GetAllNoteIDs();
        } catch (err) {
            console.error('获取全部 ID 失败，降级为当前页:', err);
            ids = state.notes.map(n => n.id);
        }
    } else {
        ids = state.notes.map(n => n.id);
    }
    ids.forEach(id => state.selectedNoteIds.add(id));
    updateBatchBar();
    renderCardGrid();
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
}

/**
 * 取消选中
 */
function clearSelection() {
    state.selectedNoteIds.clear();
    updateBatchBar();
    if (state.batchMode) {
        renderCardGrid();
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
    showUndoToast(`已删除 ${ids.length} 条笔记`, ids);
}

/* ===== HTML 转义 ===== */

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/* ===== 事件绑定 ===== */

function initEventListeners() {
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

    // 新建笔记
    els.newNoteBtn.addEventListener('click', () => {
        openEditor(null);
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
        els.moreMenu.classList.toggle('active');
    });
    // 更多菜单项点击
    els.moreMenu.addEventListener('click', (e) => {
        const item = e.target.closest('.dropdown-item');
        if (item && item.dataset.action) {
            els.moreMenu.classList.remove('active');
            if (item.dataset.action === 'home') {
                state.searchKeyword = '';
                els.searchInput.value = '';
                switchView('grid');
                resetPagination();
                loadNotes();
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
    els.editorCloseBtn.addEventListener('click', closeEditor);
    els.editorFullscreenBtn.addEventListener('click', toggleEditorFullscreen);
    els.editorCancelBtn.addEventListener('click', closeEditor);
    els.editorSaveBtn.addEventListener('click', async () => {
        if (state.editingNoteId) {
            await updateNote(state.editingNoteId);
        } else {
            await createNote();
        }
    });
    // 点击蒙层关闭编辑器
    els.editorOverlay.addEventListener('click', (e) => {
        if (e.target === els.editorOverlay) closeEditor();
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

    // 撤销 Toast
    els.undoToastBtn.addEventListener('click', undoDelete);


    els.settingsBackBtn.addEventListener('click', () => {
        switchView('grid');
    });

    // 快速笔记开关
    els.quickNoteToggle.addEventListener('change', async (e) => {
        try {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SetSetting) {
                await window.go.main.App.SetSetting('quick_note_enabled', String(e.target.checked));
            } else {
                localStorage.setItem('quick_note_enabled', String(e.target.checked));
            }
        } catch (err) {
            console.error('保存快速笔记设置失败:', err);
        }
    });

    // 右键菜单：点击其他区域关闭
    document.addEventListener('click', hideContextMenu);
    document.addEventListener('click', () => els.moreMenu.classList.remove('active'));
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
    els.batchModeBtn.addEventListener('click', toggleBatchMode);
    els.batchDeleteBtn.addEventListener('click', batchDeleteSelected);
    els.batchCancelBtn.addEventListener('click', () => {
        if (state.batchMode) toggleBatchMode();
    });
    els.batchSelectAllBtn.addEventListener('click', toggleSelectAll);

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

/**
 * 处理键盘快捷键（Ctrl+Home/End, PgUp/PgDn）
 */
function handleKeyboardNavigation(e) {
    const container = getScrollContainer();

    // Ctrl+F: 聚焦搜索输入框
    if (e.ctrlKey && e.key === 'f') {
        e.preventDefault();
        els.searchInput.focus();
        els.searchInput.select();
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

    // Escape: 退出当前子视图，回到首页
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

    // 数字键快捷导航（仅在非输入框内生效）
    if (!e.ctrlKey && !e.metaKey && !e.altKey && !e.target.closest('input, textarea')) {
        switch (e.key) {
            case '1':
                e.preventDefault();
                state.searchKeyword = '';
                els.searchInput.value = '';
                switchView('grid');
                return;
            case '2':
                e.preventDefault();
                switchView('data');
                return;
            case '3':
                e.preventDefault();
                switchView('trash');
                return;
            case '4':
                e.preventDefault();
                switchView('settings');
                return;
            case '5':
                e.preventDefault();
                openShortcuts();
                return;
        }
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
    const containers = [els.mainContent, els.searchResults].filter(Boolean);
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
 * 打开关于页面，获取版本信息
 */
async function showAbout() {
    els.viewAbout.style.display = 'flex';
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
 * 关闭关于页面
 */
function closeAbout() {
    els.viewAbout.style.display = 'none';
}

/**
 * 打开快捷键说明模态框
 */
function openShortcuts() {
    els.shortcutsView.style.display = 'flex';
    renderShortcutsPage();
}

/**
 * 关闭快捷键说明模态框
 */
function closeShortcuts() {
    els.shortcutsView.style.display = 'none';
}

/* ===== 快捷键说明页面 ===== */

/**
 * 渲染快捷键说明页面
 */
function renderShortcutsPage() {
    const shortcuts = [
        { key: 'Ctrl + N', desc: '新建笔记' },
        { key: 'Ctrl + F', desc: '聚焦搜索框' },
        { key: 'PgUp', desc: '上翻一页' },
        { key: 'PgDn', desc: '下翻一页 / 触底加载更多' },
        { key: 'Ctrl + Home', desc: '回到顶部' },
        { key: 'Ctrl + End', desc: '加载全部并滚到底部' },
        { key: 'Escape', desc: '关闭弹窗 / 返回上一页' },
        { key: '1 2 3 4 5', desc: '快速切换视图' },
    ];
    els.shortcutsBody.innerHTML = shortcuts.map(s => `
        <div class="shortcut-row">
            <div class="shortcut-key">${s.key.replace(/(\w+)/g, '<kbd>$1</kbd>')}</div>
            <div class="shortcut-desc">${s.desc}</div>
        </div>
    `).join('');
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
    await loadNotes();
    await loadTags();
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
            openEditor(null);
            toggleEditorFullscreen();
        }
    } catch (err) {
        console.error('加载快速笔记设置失败:', err);
    }
}

// 应用启动
document.addEventListener('DOMContentLoaded', init);
