/* ===== 数据管理函数 ===== */

/**
 * 数字递增动画（从 0 渐变到目标值）
 * @param {HTMLElement} element - 显示数字的元素
 * @param {number} targetValue - 目标数值
 * @param {number} duration - 动画时长（毫秒）
 */
export function animateCountUp(element, targetValue, duration = 300) {
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
 * 重新加载所有设置（恢复出厂/导入/还原后调用）
 */
async function reloadSettings() {
    window.loadThemeSetting?.();
    window.loadFontSettings?.();
    window.loadSortSettings?.();
    window.loadPageSizeSetting?.();
    window.loadQuickNoteSetting?.();
    window.loadSyntaxHighlightSetting?.();
    window.loadCodeHighlightThemeSetting?.();
    window.loadAISettings?.();
}

/**
 * 加载数据统计概览
 */
export async function loadDataStats() {
    const { els, state } = window;
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
export async function resetDatabase() {
    const { els, nm, state, showConfirmDialog } = window;

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
    window.loadNotebooks();
    // 重置后重新应用默认设置
    reloadSettings();
    // 重置后折叠侧栏，用户展开时自动触发刷新笔记本数据
    if (els.notebookSidebar) {
        els.notebookSidebar.classList.add('collapsed');
        localStorage.setItem('jot_sidebar_collapsed', 'true');
        window.updateSidebarMenuItem();
    }
    // 重置后 activeNotebookId 设为新默认笔记本
    state.activeNotebookId = 1;
    // 切回首页并刷新笔记列表，确保显示的笔记是最新状态
    window.switchView('grid');
    window.loadNotes();
}

/**
 * 数据库瘦身：执行 VACUUM 回收磁盘空间
 */
export async function vacuumDatabase() {
    const { nm } = window;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.VacuumDatabase) {
            const msg = await window.go.main.App.VacuumDatabase();
            nm.show(msg, 'success');
            await loadDataStats();
        } else {
            nm.show('数据库瘦身功能不可用', 'error');
        }
    } catch (err) {
        console.error('数据库瘦身失败:', err);
        nm.show('数据库瘦身失败：' + err.message, 'error');
    }
}

/**
 * 在文件管理器中打开数据库目录
 */
export async function openDataDir() {
    const { nm } = window;
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
export async function exportData() {
    const { nm } = window;
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
export async function importData() {
    const { nm } = window;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ImportDatabaseWithDialog) {
            const result = await window.go.main.App.ImportDatabaseWithDialog();
            if (result && result.message !== '已取消') {
                nm.show(result.message, 'success');
                if (result.success_count > 0) {
                    // 刷新所有数据
                    window.loadNotes();
                    loadDataStats();
                    window.loadTags();
                    reloadSettings();
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
export async function loadBackupInfo() {
    const { els } = window;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetBackupInfo) {
            const info = await window.go.main.App.GetBackupInfo();
            if (info && info.file_name) {
                els.backupInfo.classList.add('has-backup');
                els.backupStatusText.textContent = `${info.file_time}，${info.file_size}`;
            } else {
                els.backupInfo.classList.remove('has-backup');
                els.backupStatusText.textContent = '暂无备份';
            }
        }
    } catch (err) {
        console.error('加载备份信息失败:', err);
        els.backupInfo.classList.remove('has-backup');
        els.backupStatusText.textContent = '暂无备份';
    }
}

/**
 * 一键备份（带按钮加载状态）
 */
export async function backupToDir() {
    const { els, nm } = window;
    const btn = els.backupBtn;
    const labelEl = btn.querySelector('.dar-label');
    const origText = labelEl ? labelEl.textContent : '';
    btn.disabled = true;
    if (labelEl) labelEl.textContent = '⏳ 备份中…';
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
        if (labelEl) labelEl.textContent = origText;
    }
}

/**
 * 一键还原（带按钮加载状态 + 确认提示）
 */
export async function restoreFromDir() {
    const { els, nm, showConfirmDialog } = window;
    const btn = els.restoreBtn;
    const labelEl = btn.querySelector('.dar-label');
    const origText = labelEl ? labelEl.textContent : '';
    // 自定义确认弹窗
    const confirmed = await showConfirmDialog('确定要从最新备份恢复数据吗？当前所有笔记将被替换为备份内容，此操作不可撤销。');
    if (!confirmed) return;

    btn.disabled = true;
    if (labelEl) labelEl.textContent = '⏳ 还原中…';
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RestoreFromDir) {
            const result = await window.go.main.App.RestoreFromDir();
            if (result && result.message) {
                nm.show(result.message, 'success');
                if (result.success_count > 0) {
                    window.loadNotes();
                    loadDataStats();
                    window.loadTags();
                    reloadSettings();
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
        if (labelEl) labelEl.textContent = origText;
    }
}
