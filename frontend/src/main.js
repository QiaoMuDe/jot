import './style.css';
import './app.css';

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
};

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
    editorCancelBtn: $('editorCancelBtn'),
    editorSaveBtn: $('editorSaveBtn'),

    // 设置
    tagList: $('tagList'),
    newTagName: $('newTagName'),
    newTagColor: $('newTagColor'),
    addTagBtn: $('addTagBtn'),
    // 回收站
    trashList: $('trashList'),
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
    editorCopyBtn: $('editorCopyBtn'),

    // 撤销 Toast
    undoToast: $('undoToast'),
    undoToastMsg: $('undoToastMsg'),
    undoToastBtn: $('undoToastBtn'),

    // 右键菜单
    contextMenu: $('contextMenu'),

    // 主内容区（用于网格视图滚动）
    mainContent: $('mainContent'),

    // 批量操作
    batchModeBtn: $('batchModeBtn'),
    batchBar: $('batchBar'),
    batchCount: $('batchCount'),
    batchDeleteBtn: $('batchDeleteBtn'),
    batchCancelBtn: $('batchCancelBtn'),
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

    // 新建按钮仅在笔记网格视图显示
    els.newNoteBtn.style.display = view === 'grid' ? 'flex' : 'none';

    switch (view) {
        case 'grid':
            els.viewGrid.classList.add('active');
            break;
        case 'search':
            els.viewSearch.classList.add('active');
            break;
        case 'settings':
            els.viewSettings.classList.add('active');
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
 * 加载笔记列表
 */
async function loadNotes() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNotes) {
            const result = await window.go.main.App.GetNotes(1, 100);
            state.notes = (result && result.items) || [];
        } else {
            console.warn('GetNotes 未绑定，使用模拟数据');
            if (!mockNotes) {
                mockNotes = getMockNotes();
            }
            state.notes = mockNotes;
        }
    } catch (err) {
        console.error('加载笔记失败:', err);
        if (!mockNotes) {
            mockNotes = getMockNotes();
        }
        state.notes = mockNotes;
    }
    renderCardGrid();
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
 * @param {number} [noteId] - 可撤销的笔记 ID（不传则为纯提示）
 */
let undoToastTimer = null;
let undoNoteId = null;

function showUndoToast(msg, noteId) {
    // 清除之前的定时器
    if (undoToastTimer) {
        clearTimeout(undoToastTimer);
        undoToastTimer = null;
    }

    els.undoToastMsg.textContent = msg;
    undoNoteId = noteId || null;

    // 有可撤销的笔记 ID 时显示按钮，否则隐藏
    els.undoToastBtn.style.display = noteId ? '' : 'none';
    els.undoToast.classList.add('active');

    // 5 秒后自动消失
    undoToastTimer = setTimeout(() => {
        els.undoToast.classList.remove('active');
        undoNoteId = null;
        undoToastTimer = null;
    }, 5000);
}

/**
 * 撤销删除
 */
async function undoDelete() {
    if (undoNoteId == null) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreNote) {
            await window.go.main.App.RestoreNote(undoNoteId);
        } else {
            console.warn('RestoreNote 未绑定');
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

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CreateTag) {
            await window.go.main.App.CreateTag(name, color);
        } else {
            console.warn('CreateTag 未绑定');
        }
    } catch (err) {
        console.error('创建标签失败:', err);
    }
    els.newTagName.value = '';
    await loadTags();
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
}

/**
 * 全部恢复回收站笔记
 */
async function restoreAllNotes() {
    if (!confirm('确定要恢复回收站中的所有笔记吗？')) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreAllNotes) {
            await window.go.main.App.RestoreAllNotes();
        } else {
            console.warn('RestoreAllNotes 未绑定');
        }
    } catch (err) {
        console.error('全部恢复失败:', err);
    }
    await loadTrashNotes();
    await loadNotes();
}

/**
 * 清空回收站（永久删除所有）
 */
async function emptyTrash() {
    if (!confirm('确定要永久清空回收站中的所有笔记吗？此操作不可撤销。')) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.EmptyTrash) {
            await window.go.main.App.EmptyTrash();
        } else {
            console.warn('EmptyTrash 未绑定');
        }
    } catch (err) {
        console.error('清空回收站失败:', err);
    }
    await loadTrashNotes();
    await loadNotes();
}

/**
 * 永久删除笔记
 */
async function permanentDeleteNote(id) {
    if (!confirm('确定要永久删除这条笔记吗？此操作不可撤销。')) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.PermanentDeleteNote) {
            await window.go.main.App.PermanentDeleteNote(id);
        } else {
            console.warn('PermanentDeleteNote 未绑定');
        }
    } catch (err) {
        console.error('永久删除失败:', err);
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
    const sorted = [...state.notes].sort((a, b) => {
        if (a.pinned && !b.pinned) return -1;
        if (!a.pinned && b.pinned) return 1;
        return new Date(b.updated_at || b.created_at) - new Date(a.updated_at || a.created_at);
    });

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
            ${note.pinned ? '<div class="card-pin-badge">📌</div>' : ''}
            <div class="card-body">
                <div class="card-title">${escapeHtml(note.title || '无标题')}</div>
                <div class="card-content">${escapeHtml(getSummary(note.content, 200))}</div>
            </div>
            <div class="card-footer">
                <div class="card-tags">
                    ${(note.tags || [])
                        .map(
                            (tag) =>
                                `<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}" onclick="event.stopPropagation(); window.searchByTag('${escapeHtml(tag.name)}')">${escapeHtml(tag.name)}</span>`
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
             onclick="window.toggleEditorTag(${tag.id})">
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
        els.trashList.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🗑️</div>
                <div class="empty-title">回收站为空</div>
                <div class="empty-desc">删除的笔记会出现在这里</div>
            </div>
        `;
        return;
    }

    els.trashList.innerHTML = state.trashNotes
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
function openEditor(noteId, readOnly) {
    state.editingNoteId = noteId || null;
    state.selectedTags = [];

    const isReadOnly = readOnly && noteId != null;

    if (noteId) {
        const note = state.notes.find((n) => n.id === noteId);
        if (note) {
            els.editorTitle.textContent = isReadOnly ? '查看笔记' : '编辑笔记';
            els.editorNoteTitle.value = note.title || '';
            els.editorNoteContent.value = note.content || '';
            state.selectedTags = (note.tags || []).map((t) => t.id);
        }
    } else {
        els.editorTitle.textContent = '新建笔记';
        els.editorNoteTitle.value = '';
        els.editorNoteContent.value = '';
    }

    // 只读模式：禁用输入，隐藏保存/取消按钮，显示复制按钮
    els.editorNoteTitle.readOnly = isReadOnly;
    els.editorNoteContent.readOnly = isReadOnly;
    els.editorNoteTitle.classList.toggle('editor-input-readonly', isReadOnly);
    els.editorNoteContent.classList.toggle('editor-textarea-readonly', isReadOnly);
    els.editorSaveBtn.style.display = isReadOnly ? 'none' : '';
    els.editorCancelBtn.style.display = isReadOnly ? 'none' : '';
    els.editorPanel.classList.toggle('editor-view-mode', isReadOnly);

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

    // 加载标签并渲染标签选择器
    loadTagsForEditor(isReadOnly);
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
function closeEditor() {
    els.viewEditor.classList.remove('active');
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
window.toggleEditorTag = function (tagId) {
    const idx = state.selectedTags.indexOf(tagId);
    if (idx > -1) {
        state.selectedTags.splice(idx, 1);
    } else {
        state.selectedTags.push(tagId);
    }
    renderTagSelector();
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
        els.batchCount.textContent = '0';
    } else {
        els.batchBar.style.display = 'none';
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
    renderCardGrid();
};

/**
 * 更新批量操作栏
 */
function updateBatchBar() {
    const count = state.selectedNoteIds.size;
    els.batchCount.textContent = count;
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
    if (!confirm(`确定要删除选中的 ${ids.length} 条笔记吗？`)) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchDeleteNotes) {
            await window.go.main.App.BatchDeleteNotes(ids);
        } else {
            console.warn('BatchDeleteNotes 未绑定，模拟批量删除');
            mockNotes = mockNotes.filter(n => !ids.includes(n.id));
        }
    } catch (err) {
        console.error('批量删除失败:', err);
    }
    clearSelection();
    await loadNotes();
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
    // 输入框自动搜索（防抖 250ms）
    els.searchInput.addEventListener('input', debounce(function () {
        const kw = this.value.trim();
        if (kw) {
            searchNotes(kw, 'input');
        } else {
            state.searchKeyword = '';
            switchView('grid');
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
            if (item.dataset.action === 'settings') {
                switchView('settings');
            } else if (item.dataset.action === 'data') {
                switchView('data');
            } else if (item.dataset.action === 'trash') {
                switchView('trash');
                loadTrashNotes();
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

    // 编辑器复制按钮
    els.editorCopyBtn.addEventListener('click', () => {
        const title = els.editorNoteTitle.value || '';
        const content = els.editorNoteContent.value || '';
        const text = (title ? title + '\n\n' : '') + content;
        navigator.clipboard.writeText(text).then(() => {
            showUndoToast('已复制到剪贴板');
        }).catch(() => {
            showUndoToast('复制失败');
        });
    });
    els.settingsBackBtn.addEventListener('click', () => {
        switchView('grid');
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
            return els.trashList;
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

    // Escape: 退出批量模式
    if (e.key === 'Escape' && state.batchMode) {
        e.preventDefault();
        toggleBatchMode();
        return;
    }

    if (!container) return;

    // Ctrl+Home: 滚动到顶部
    if (e.ctrlKey && e.key === 'Home') {
        e.preventDefault();
        container.scrollTop = 0;
        return;
    }
    // Ctrl+End: 滚动到底部
    if (e.ctrlKey && e.key === 'End') {
        e.preventDefault();
        container.scrollTop = container.scrollHeight;
        return;
    }
    // PgUp: 向上翻一页
    if (e.key === 'PageUp') {
        e.preventDefault();
        container.scrollTop -= container.clientHeight;
        return;
    }
    // PgDn: 向下翻一页
    if (e.key === 'PageDown') {
        e.preventDefault();
        container.scrollTop += container.clientHeight;
        return;
    }
}

/* ===== 初始化 ===== */

async function init() {
    initEventListeners();
    state.selectedTags = [];
    await loadNotes();
    await loadTags();
}

// 应用启动
document.addEventListener('DOMContentLoaded', init);
