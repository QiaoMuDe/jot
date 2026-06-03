import './style.css';
import './app.css';

/* ===== 预设颜色方案 ===== */
const COLORS = [
    { name: '白色', value: '#ffffff' },
    { name: '黄色', value: '#fef3c7' },
    { name: '绿色', value: '#d1fae5' },
    { name: '蓝色', value: '#dbeafe' },
    { name: '紫色', value: '#ede9fe' },
    { name: '粉色', value: '#fce7f3' },
    { name: '橙色', value: '#ffedd5' },
];

/* ===== 应用状态 ===== */
const state = {
    notes: [],
    tags: [],
    trashNotes: [],
    currentView: 'grid',       // grid | search | settings | trash
    editingNoteId: null,        // null = 新建, number = 编辑
    selectedColor: '#ffffff',
    selectedTags: [],
    isGridView: true,
    searchKeyword: '',
};

/* ===== DOM 引用 ===== */
const $ = (id) => document.getElementById(id);

const els = {
    // 侧边栏
    notesList: $('notesList'),
    searchInput: $('searchInput'),
    searchBtn: $('searchBtn'),
    newNoteBtn: $('newNoteBtn'),
    settingsBtn: $('settingsBtn'),

    // 视图
    viewGrid: $('viewGrid'),
    viewSearch: $('viewSearch'),
    viewSettings: $('viewSettings'),
    viewTrash: $('viewTrash'),
    viewEditor: $('viewEditor'),

    cardGrid: $('cardGrid'),
    viewTitle: $('viewTitle'),
    gridViewBtn: $('gridViewBtn'),
    listViewBtn: $('listViewBtn'),

    // 搜索
    searchResults: $('searchResults'),
    resultCount: $('resultCount'),
    searchBackBtn: $('searchBackBtn'),

    // 编辑器
    editorOverlay: $('editorOverlay'),
    editorTitle: $('editorTitle'),
    editorNoteTitle: $('editorNoteTitle'),
    editorNoteContent: $('editorNoteContent'),
    colorPicker: $('colorPicker'),
    tagSelector: $('tagSelector'),
    editorCloseBtn: $('editorCloseBtn'),
    editorCancelBtn: $('editorCancelBtn'),
    editorSaveBtn: $('editorSaveBtn'),

    // 设置
    tagList: $('tagList'),
    newTagName: $('newTagName'),
    newTagColor: $('newTagColor'),
    addTagBtn: $('addTagBtn'),
    trashEntryBtn: $('trashEntryBtn'),

    // 回收站
    trashList: $('trashList'),
    trashBackBtn: $('trashBackBtn'),
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

/* ===== 视图切换 ===== */

/**
 * 切换右侧主内容区视图
 */
function switchView(view) {
    // 隐藏所有视图
    document.querySelectorAll('.view').forEach((v) => {
        v.classList.remove('active');
    });

    state.currentView = view;

    switch (view) {
        case 'grid':
            els.viewGrid.classList.add('active');
            els.viewTitle.textContent = state.searchKeyword
                ? `搜索: "${state.searchKeyword}"`
                : '所有笔记';
            break;
        case 'search':
            els.viewSearch.classList.add('active');
            break;
        case 'settings':
            els.viewSettings.classList.add('active');
            loadTags();
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
            state.notes = getMockNotes();
        }
    } catch (err) {
        console.error('加载笔记失败:', err);
        state.notes = getMockNotes();
    }
    renderNotesList();
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
            const note = await window.go.main.App.CreateNote(title, content, state.selectedColor);
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
                color: state.selectedColor,
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
            await window.go.main.App.UpdateNote(id, title, content, state.selectedColor);
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
            const note = state.notes.find((n) => n.id === id);
            if (note) {
                note.title = title;
                note.content = content;
                note.color = state.selectedColor;
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
 * 删除笔记
 */
async function deleteNote(id) {
    if (!confirm('确定要删除这条笔记吗？')) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteNote) {
            await window.go.main.App.DeleteNote(id);
        } else {
            console.warn('DeleteNote 未绑定，模拟删除');
            state.notes = state.notes.filter((n) => n.id !== id);
        }
    } catch (err) {
        console.error('删除笔记失败:', err);
    }
    await loadNotes();
}

/**
 * 搜索笔记
 */
async function searchNotes(keyword) {
    const kw = keyword.trim();
    if (!kw) {
        switchView('grid');
        return;
    }

    state.searchKeyword = kw;

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
            state.tags = [];
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

/* ===== 渲染函数 ===== */

/**
 * 渲染左侧笔记缩略列表
 */
function renderNotesList() {
    const sorted = [...state.notes].sort((a, b) => {
        if (a.pinned && !b.pinned) return -1;
        if (!a.pinned && b.pinned) return 1;
        return new Date(b.updated_at || b.created_at) - new Date(a.updated_at || a.created_at);
    });

    if (sorted.length === 0) {
        els.notesList.innerHTML = `
            <div style="padding: 20px 12px; text-align: center; color: rgba(255,255,255,0.3); font-size: 12px;">
                暂无笔记，点击 "+" 新建
            </div>
        `;
        return;
    }

    els.notesList.innerHTML = sorted
        .map(
            (note) => `
        <div class="note-thumb" data-id="${note.id}" onclick="window.openNote(${note.id})">
            <div class="thumb-color-indicator" style="background-color: ${note.color || '#ffffff'}"></div>
            <div class="thumb-content">
                <div class="thumb-title">${note.pinned ? '📌 ' : ''}${escapeHtml(note.title || '无标题')}</div>
                <div class="thumb-preview">${escapeHtml(getSummary(note.content, 80))}</div>
            </div>
        </div>
        `
        )
        .join('');
}

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
        <div class="note-card" onclick="window.openNote(${note.id})">
            ${note.pinned ? '<div class="card-pin-badge">📌</div>' : ''}
            <div class="card-color-bar" style="background-color: ${note.color || '#ffffff'}"></div>
            <div class="card-body">
                <div class="card-title">${escapeHtml(note.title || '无标题')}</div>
                <div class="card-content">${escapeHtml(getSummary(note.content, 200))}</div>
            </div>
            <div class="card-footer">
                <div class="card-tags">
                    ${(note.tags || [])
                        .map(
                            (tag) =>
                                `<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}">${escapeHtml(tag.name)}</span>`
                        )
                        .join('')}
                </div>
                <span class="card-time">${formatTime(note.updated_at || note.created_at)}</span>
            </div>
            <div class="card-actions" onclick="event.stopPropagation()">
                <button class="card-action-btn" onclick="window.openNote(${note.id})" title="编辑">✎</button>
                <button class="card-action-btn" onclick="window.togglePin(${note.id})" title="${note.pinned ? '取消置顶' : '置顶'}">
                    ${note.pinned ? '★' : '☆'}
                </button>
                <button class="card-action-btn delete" onclick="window.deleteNote(${note.id})" title="删除">✕</button>
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
        <div class="search-result-item" onclick="window.openNote(${note.id})">
            <div class="result-title">${highlightText(escapeHtml(note.title || '无标题'), keyword)}</div>
            <div class="result-snippet">${highlightText(escapeHtml(getSummary(note.content, 150)), keyword)}</div>
            <div class="result-time">${formatTime(note.updated_at || note.created_at)}</div>
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
 */
function renderTagSelector() {
    if (state.tags.length === 0) {
        els.tagSelector.innerHTML = '<div style="color: #94a3b8; font-size: 12px;">暂无可用标签，请先在设置页添加</div>';
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

/**
 * 打开编辑器（新建或编辑）
 */
function openEditor(noteId) {
    state.editingNoteId = noteId || null;
    state.selectedColor = '#ffffff';
    state.selectedTags = [];

    if (noteId) {
        const note = state.notes.find((n) => n.id === noteId);
        if (note) {
            els.editorTitle.textContent = '编辑笔记';
            els.editorNoteTitle.value = note.title || '';
            els.editorNoteContent.value = note.content || '';
            state.selectedColor = note.color || '#ffffff';
            state.selectedTags = (note.tags || []).map((t) => t.id);
        }
    } else {
        els.editorTitle.textContent = '新建笔记';
        els.editorNoteTitle.value = '';
        els.editorNoteContent.value = '';
    }

    // 渲染颜色选择器
    renderColorPicker();
    // 加载标签并渲染标签选择器
    loadTagsForEditor();
    // 显示编辑器
    els.viewEditor.classList.add('active');
}

/**
 * 加载标签并渲染编辑器标签选择器
 */
async function loadTagsForEditor() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllTags) {
            const tags = await window.go.main.App.GetAllTags();
            state.tags = tags || [];
        } else {
            state.tags = [];
        }
    } catch (err) {
        console.error('加载标签失败:', err);
        state.tags = [];
    }
    renderTagSelector();
}

/**
 * 渲染颜色选择器
 */
function renderColorPicker() {
    els.colorPicker.innerHTML = COLORS
        .map(
            (c) => `
        <div class="color-option ${c.value === state.selectedColor ? 'active' : ''} ${c.value === '#ffffff' ? 'active-white' : ''}"
             style="background-color: ${c.value}"
             data-color="${c.value}"
             title="${c.name}"
             onclick="window.selectEditorColor('${c.value}')">
        </div>
        `
        )
        .join('');
}

/**
 * 关闭编辑器
 */
function closeEditor() {
    els.viewEditor.classList.remove('active');
    state.editingNoteId = null;
    state.selectedColor = '#ffffff';
    state.selectedTags = [];
}

/* ===== 全局暴露给 onclick 的函数 ===== */

/**
 * 打开/查看笔记
 */
window.openNote = function (id) {
    openEditor(id);
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
 * 选择编辑器中颜色
 */
window.selectEditorColor = function (color) {
    state.selectedColor = color;
    renderColorPicker();
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
    els.searchBtn.addEventListener('click', () => {
        searchNotes(els.searchInput.value);
    });
    els.searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            searchNotes(els.searchInput.value);
        }
    });
    els.searchBackBtn.addEventListener('click', () => {
        els.searchInput.value = '';
        state.searchKeyword = '';
        switchView('grid');
    });

    // 新建笔记
    els.newNoteBtn.addEventListener('click', () => {
        openEditor(null);
    });

    // 设置按钮
    els.settingsBtn.addEventListener('click', () => {
        switchView('settings');
    });

    // 视图切换
    els.gridViewBtn.addEventListener('click', () => {
        state.isGridView = true;
        els.gridViewBtn.classList.add('active');
        els.listViewBtn.classList.remove('active');
        els.cardGrid.classList.remove('list-view');
    });
    els.listViewBtn.addEventListener('click', () => {
        state.isGridView = false;
        els.listViewBtn.classList.add('active');
        els.gridViewBtn.classList.remove('active');
        els.cardGrid.classList.add('list-view');
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

    // 回收站入口
    els.trashEntryBtn.addEventListener('click', () => {
        switchView('trash');
    });
    els.trashBackBtn.addEventListener('click', () => {
        switchView('settings');
    });
}

/* ===== 模拟数据（后端未绑定时使用） ===== */

function getMockNotes() {
    return [
        {
            id: 1,
            title: '欢迎使用 jot',
            content: '这是一条示例笔记。你可以在这里记录你的想法、灵感、待办事项等。\n\n点击左侧 "+" 按钮创建新笔记，或点击笔记进行编辑。',
            color: '#fef3c7',
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
            color: '#dbeafe',
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
            color: '#d1fae5',
            pinned: false,
            created_at: new Date(Date.now() - 172800000).toISOString(),
            updated_at: new Date(Date.now() - 86400000).toISOString(),
            tags: [
                { id: 3, name: '待办', color: '#f59e0b' },
            ],
        },
    ];
}

/* ===== 初始化 ===== */

async function init() {
    initEventListeners();
    state.selectedColor = '#ffffff';
    state.selectedTags = [];
    await loadNotes();
    await loadTags();
}

// 应用启动
document.addEventListener('DOMContentLoaded', init);
