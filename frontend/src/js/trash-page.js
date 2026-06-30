/* ===== 回收站页面模块（支持笔记和笔记本两种条目） ===== */

/** 回收站笔记列表 */
let trashNotes = [];
/** 回收站笔记本列表 */
let trashNotebooks = [];

/**
 * 加载回收站中的笔记和笔记本
 */
export async function loadTrashNotes() {
    const { els, nm } = window;
    try {
        // 并行加载笔记和笔记本
        const [noteResult, notebookResult] = await Promise.all([
            (window.go && window.go.main && window.go.main.App && window.go.main.App.GetTrashNotes)
                ? window.go.main.App.GetTrashNotes(1, 100).catch(() => ({ items: [] }))
                : Promise.resolve({ items: [] }),
            (window.go && window.go.main && window.go.main.App && window.go.main.App.GetTrashNotebooks)
                ? window.go.main.App.GetTrashNotebooks(1, 100).catch(() => ({ items: [] }))
                : Promise.resolve({ items: [] }),
        ]);

        trashNotes = (noteResult && noteResult.items) || [];
        trashNotebooks = (notebookResult && notebookResult.items) || [];
    } catch (err) {
        console.error('加载回收站失败:', err);
        trashNotes = [];
        trashNotebooks = [];
    }
    renderTrashList();
}

// ===== SVG 图标 =====

const NOTE_ICON_SVG = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>';
const NOTEBOOK_ICON_SVG = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>';

// ===== 一次性初始化 =====
// 顶部操作按钮点击脉冲反馈（仅初始化一次）
if (!window._trashTopBtnPulseInited) {
    window._trashTopBtnPulseInited = true;
    const header = document.querySelector('#viewTrash .view-controls');
    if (header) {
        header.addEventListener('click', (e) => {
            const btn = e.target.closest('.btn-restore, .btn-perm-delete');
            if (btn) {
                btn.style.animation = 'none';
                void btn.offsetWidth;
                btn.style.animation = 'pulseClick 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) forwards';
            }
        });
    }
}

// ===== 笔记操作 =====

/**
 * 恢复笔记（带动画）
 */
async function restoreNote(id) {
    const { els, nm } = window;
    const item = els.trashListInner.querySelector(`.trash-item[data-id="${id}"][data-type="note"]`);
    if (item) {
        item.style.animation = 'restoreOut 0.3s ease-out forwards';
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
    await window.loadNotes();
    await window.loadNotebooks();
    nm.show('笔记已恢复', 'success');
}

/**
 * 永久删除笔记（带动画）
 */
async function permanentDeleteNote(id) {
    const { els, nm } = window;
    const item = els.trashListInner.querySelector(`.trash-item[data-id="${id}"][data-type="note"]`);
    if (item) {
        item.style.animation = 'deleteOut 0.45s ease-out forwards';
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

// ===== 笔记本操作 =====

/**
 * 恢复笔记本（带动画）
 */
async function restoreNotebook(id) {
    const { els, nm } = window;
    const item = els.trashListInner.querySelector(`.trash-item[data-id="${id}"][data-type="notebook"]`);
    if (item) {
        item.style.animation = 'restoreOut 0.3s ease-out forwards';
        await new Promise(resolve => {
            item.addEventListener('animationend', resolve, { once: true });
        });
    }
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreTrashNotebook) {
            await window.go.main.App.RestoreTrashNotebook(id);
        } else {
            console.warn('RestoreTrashNotebook 未绑定');
        }
    } catch (err) {
        console.error('恢复笔记本失败:', err);
    }
    await loadTrashNotes();
    await window.loadNotebooks();
    nm.show('笔记本已恢复', 'success');
}

/**
 * 永久删除笔记本（带动画）
 */
async function permanentDeleteNotebook(id) {
    const { els, nm } = window;
    const item = els.trashListInner.querySelector(`.trash-item[data-id="${id}"][data-type="notebook"]`);
    if (item) {
        item.style.animation = 'deleteOut 0.45s ease-out forwards';
        await new Promise(resolve => {
            item.addEventListener('animationend', resolve, { once: true });
        });
    }
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.PermanentDeleteTrashNotebook) {
            await window.go.main.App.PermanentDeleteTrashNotebook(id);
        } else {
            console.warn('PermanentDeleteTrashNotebook 未绑定');
        }
        nm.show('笔记本已永久删除', 'info');
    } catch (err) {
        console.error('永久删除笔记本失败:', err);
        nm.show('删除失败', 'error');
    }
    await loadTrashNotes();
    await window.loadNotebooks();
}

// ===== 批量操作 =====

/**
 * 全部恢复（笔记 + 笔记本，带动画）
 */
async function restoreAllNotes() {
    const { els, nm } = window;
    const totalCount = trashNotes.length + trashNotebooks.length;
    if (totalCount === 0) {
        nm.show('回收站为空', 'info');
        return;
    }
    const confirmed = await window.showConfirmDialog('确定要恢复回收站中的所有笔记和笔记本吗？');
    if (!confirmed) return;

    const items = els.trashListInner.querySelectorAll('.trash-item');
    const restorePromises = Array.from(items).map((item, index) => {
        return new Promise(resolve => {
            item.style.animation = `restoreOut 0.3s ease-out forwards`;
            item.style.animationDelay = `${index * 40}ms`;
            item.addEventListener('animationend', resolve, { once: true });
        });
    });
    await Promise.all(restorePromises);

    try {
        // 恢复所有笔记
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreAllNotes) {
            await window.go.main.App.RestoreAllNotes();
        }

        // 恢复所有笔记本
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreAllTrashNotebooks) {
            await window.go.main.App.RestoreAllTrashNotebooks();
        }

        nm.show(`已恢复 ${totalCount} 个条目`, 'success');
    } catch (err) {
        console.error('全部恢复失败:', err);
        nm.show('恢复失败', 'error');
    }
    await loadTrashNotes();
    await window.loadNotes();
    await window.loadNotebooks();
}

/**
 * 清空回收站（笔记 + 笔记本，带动画）
 */
async function emptyTrash() {
    const { els, nm } = window;
    const totalCount = trashNotes.length + trashNotebooks.length;
    if (totalCount === 0) {
        nm.show('回收站为空', 'info');
        return;
    }
    const confirmed = await window.showConfirmDialog('确定要永久清空回收站中的所有笔记和笔记本吗？此操作不可撤销。');
    if (!confirmed) return;

    const items = els.trashListInner.querySelectorAll('.trash-item');
    const emptyPromises = Array.from(items).map((item, index) => {
        return new Promise(resolve => {
            item.style.animation = `deleteOut 0.45s ease-out forwards`;
            item.style.animationDelay = `${index * 50}ms`;
            item.addEventListener('animationend', resolve, { once: true });
        });
    });
    await Promise.all(emptyPromises);

    try {
        // 清空笔记
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.EmptyTrash) {
            await window.go.main.App.EmptyTrash();
        }

        // 清空笔记本
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.EmptyTrashNotebooks) {
            await window.go.main.App.EmptyTrashNotebooks();
        }

        nm.show('回收站已清空', 'info');
    } catch (err) {
        console.error('清空回收站失败:', err);
        nm.show('清空失败', 'error');
    }
    await loadTrashNotes();
    await window.loadNotes();
    await window.loadNotebooks();
}

// ===== 渲染 =====

/**
 * 渲染回收站列表（笔记和笔记本混合展示，按删除时间倒序，带交错入场动画）
 */
function renderTrashList() {
    const { els, SVGS } = window;

    // 构建混合条目列表
    const noteItems = trashNotes.map(n => ({
        type: 'note',
        id: n.id,
        title: n.title || '无标题',
        deletedAt: n.deleted_at || n.updated_at,
    }));

    const notebookItems = trashNotebooks.map(nb => ({
        type: 'notebook',
        id: nb.id,
        title: nb.name || '未命名笔记本',
        deletedAt: nb.deleted_at,
    }));

    const allItems = [...noteItems, ...notebookItems].sort((a, b) => {
        const dateA = new Date(a.deletedAt || 0).getTime();
        const dateB = new Date(b.deletedAt || 0).getTime();
        return dateB - dateA; // 最新的在前
    });

    if (allItems.length === 0) {
        els.trashListInner.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">${SVGS.trash}</div>
                <p class="empty-state-title">回收站为空</p>
                <p class="empty-state-desc">删除的笔记和笔记本会出现在这里</p>
            </div>
        `;
        return;
    }

    els.trashListInner.innerHTML = allItems
        .map(
            (item) => {
                const isNote = item.type === 'note';
                const iconHtml = isNote ? NOTE_ICON_SVG : NOTEBOOK_ICON_SVG;
                const iconClass = isNote ? 'icon-note' : 'icon-notebook';
                const restoreFn = isNote ? 'restoreNote' : 'restoreNotebook';
                const deleteFn = isNote ? 'permanentDeleteNote' : 'permanentDeleteNotebook';

                return `
            <div class="trash-item" data-id="${item.id}" data-type="${item.type}">
                <div class="trash-item-icon ${iconClass}">${iconHtml}</div>
                <div class="trash-item-info">
                    <div class="trash-item-title">${escapeHtml(item.title)}</div>
                    <div class="trash-item-time">${isNote ? '笔记' : '笔记本'} · 删除于 ${formatTime(item.deletedAt)}</div>
                </div>
                <div class="trash-item-actions">
                    <button class="btn btn-restore btn-sm" onclick="event.stopPropagation(); window.${restoreFn}(${item.id})">恢复</button>
                    <button class="btn btn-perm-delete btn-sm" onclick="event.stopPropagation(); window.${deleteFn}(${item.id})">永久删除</button>
                </div>
            </div>
            `;
            }
        )
        .join('');

    // 交错入场动画（弹性弹入）
    const items = els.trashListInner.querySelectorAll('.trash-item');
    items.forEach((item, index) => {
        item.style.animation = `trashEnter 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) forwards`;
        item.style.animationDelay = `${index * 40}ms`;
    });

    // 按钮点击脉冲反馈（事件委托，仅绑定一次）
    if (!els.trashListInner._pulseInited) {
        els.trashListInner._pulseInited = true;
        els.trashListInner.addEventListener('click', (e) => {
            const btn = e.target.closest('.btn-restore, .btn-perm-delete');
            if (btn) {
                btn.style.animation = 'none';
                void btn.offsetWidth;
                btn.style.animation = 'pulseClick 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) forwards';
            }
        });
    }
}

/**
 * HTML 转义
 */
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * 格式化时间戳
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

// 将函数暴露到 window，供 HTML 模板 onclick 调用
window.restoreNote = restoreNote;
window.permanentDeleteNote = permanentDeleteNote;
window.restoreNotebook = restoreNotebook;
window.permanentDeleteNotebook = permanentDeleteNotebook;
window.restoreAllNotes = restoreAllNotes;
window.emptyTrash = emptyTrash;