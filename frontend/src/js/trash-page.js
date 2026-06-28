/* ===== 回收站页面模块 ===== */

/** 回收站笔记列表（模块级状态） */
let trashNotes = [];

/**
 * 加载回收站笔记
 */
export async function loadTrashNotes() {
    const { els, nm } = window;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetTrashNotes) {
            const result = await window.go.main.App.GetTrashNotes(1, 100);
            trashNotes = (result && result.items) || [];
        } else {
            console.warn('GetTrashNotes 未绑定');
            trashNotes = [];
        }
    } catch (err) {
        console.error('加载回收站失败:', err);
        trashNotes = [];
    }
    renderTrashList();
}

/**
 * 恢复笔记（带动画）
 */
async function restoreNote(id) {
    const { els, nm } = window;
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
    await window.loadNotes();
    await window.loadNotebooks();
    nm.show('笔记已恢复', 'success');
}

/**
 * 全部恢复回收站笔记（带动画）
 */
async function restoreAllNotes() {
    const { els, nm } = window;
    if (!trashNotes || trashNotes.length === 0) {
        nm.show('回收站为空', 'info');
        return;
    }
    const confirmed = await window.showConfirmDialog('确定要恢复回收站中的所有笔记吗？');
    if (!confirmed) return;

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
            nm.showUndo(`已恢复 ${trashedIds.length} 条笔记`, () => window.undoDelete(trashedIds));
        } else {
            nm.show('已恢复所有笔记', 'success');
        }
    } catch (err) {
        console.error('全部恢复失败:', err);
        nm.show('恢复失败', 'error');
    }
    await loadTrashNotes();
    await window.loadNotes();
    await window.loadNotebooks();
}

/**
 * 清空回收站（永久删除所有，带动画）
 */
async function emptyTrash() {
    const { els, nm } = window;
    if (!trashNotes || trashNotes.length === 0) {
        nm.show('回收站为空', 'info');
        return;
    }
    const confirmed = await window.showConfirmDialog('确定要永久清空回收站中的所有笔记吗？此操作不可撤销。');
    if (!confirmed) return;

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
    await window.loadNotes();
}

/**
 * 永久删除笔记（带动画）
 */
async function permanentDeleteNote(id) {
    const { els, nm } = window;
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

/**
 * 渲染回收站列表（带交错入场动画）
 */
function renderTrashList() {
    const { els, SVGS } = window;
    if (trashNotes.length === 0) {
        els.trashListInner.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">${SVGS.trash}</div>
                <p class="empty-state-title">回收站为空</p>
                <p class="empty-state-desc">删除的笔记会出现在这里</p>
            </div>
        `;
        return;
    }

    els.trashListInner.innerHTML = trashNotes
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

/**
 * HTML 转义（内部工具函数，与 main.js 中的 escapeHtml 相同）
 */
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * 格式化时间戳为可读字符串（内部工具函数，与 main.js 中的 formatTime 相同）
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
window.restoreAllNotes = restoreAllNotes;
window.emptyTrash = emptyTrash;
