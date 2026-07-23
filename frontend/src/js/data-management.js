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
    window.loadSettings?.();
}

/**
 * 加载数据统计概览 — 信笺风格
 */
export async function loadDataStats() {
    const { els, state } = window;
    let totalNotes = 0, totalTags = 0, trashedNotes = 0, totalNotebooks = 0, dbSizeStr = '';
    let aiSessions = 0, aiMessages = 0, totalTokens = 0;
    let avgResponseTime = 0, avgThinkingTime = 0, maxResponseTime = 0;
    let totalTodos = 0, completedTodos = 0;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetDataStats) {
            const stats = await window.go.main.App.GetDataStats();
            if (stats) {
                totalNotes = stats.total_notes || 0;
                totalTags = stats.total_tags || 0;
                trashedNotes = stats.trashed_notes || 0;
                totalNotebooks = stats.total_notebooks || 0;
                dbSizeStr = stats.db_size_str || '';
                aiSessions = stats.ai_sessions || 0;
                aiMessages = stats.ai_messages || 0;
                totalTokens = stats.total_tokens || 0;
                avgResponseTime = stats.avg_response_time || 0;
                avgThinkingTime = stats.avg_thinking_time || 0;
                maxResponseTime = stats.max_response_time || 0;
                totalTodos = stats.total_todos || 0;
                completedTodos = stats.completed_todos || 0;
            }
        } else {
            console.warn('GetDataStats 未绑定');
            totalNotes = state.notes.length;
            totalTags = state.tags.length;
        }
    } catch (err) {
        console.error('加载统计数据失败:', err);
    }

    // 获取信件元素
    const letterEl = els.dataLetter;
    const dateEl = els.letterDate;
    const bodyEl = els.letterBody;

    if (!letterEl || !bodyEl) return;

    // 设置日期
    if (dateEl) {
        const now = new Date();
        dateEl.textContent = `${now.getFullYear()} 年 ${now.getMonth() + 1} 月 ${now.getDate()} 日`;
    }

    const hasData = totalNotes > 0 || aiMessages > 0 || totalTodos > 0;

    if (!hasData) {
        // 空数据占位
        bodyEl.innerHTML = '<p class="data-letter-empty">你还没有开始记录呢，快去写第一篇笔记吧！</p>';
        // 隐藏落款
        const footerEl = els.letterFooter;
        if (footerEl) footerEl.style.display = 'none';
    } else {
        // 显示落款
        const footerEl = els.letterFooter;
        if (footerEl) footerEl.style.display = '';

        // 星级辅助函数：根据阈值数组 [5星上限, 4星上限, 3星上限, 2星上限] 计算星级
        const getStars = (value, thresholds) => {
            const count = value <= thresholds[0] ? 5 : value <= thresholds[1] ? 4 : value <= thresholds[2] ? 3 : value <= thresholds[3] ? 2 : 1;
            return `<span class="star-icon">${'★'.repeat(count)}${'☆'.repeat(5 - count)}</span>`;
        };

        // 每行用各自的值和阈值计算星星
        const responseStars = getStars(avgResponseTime, [3, 6, 10, 20]);
        const thinkingStars = getStars(avgThinkingTime, [1, 3, 6, 10]);
        const maxStars = getStars(maxResponseTime, [10, 20, 30, 60]);

        // 拼接信纸正文 HTML
        bodyEl.innerHTML = `
            <p class="letter-section-title">📝 笔记与存储</p>
            <p>
                截至目前，你的笔记本里共收录了 <strong>${totalNotes}</strong> 篇笔记，
                分散在 <strong>${totalNotebooks}</strong> 个笔记本中，标记了 <strong>${totalTags}</strong> 个标签。
                回收站中暂有 <strong>${trashedNotes}</strong> 篇待处理的笔记。
                数据库当前占用 <strong>${dbSizeStr || '0 B'}</strong>。
            </p>
            <hr class="letter-divider">
            <p class="letter-section-title">✓ 待办事项</p>
            <p>
                你共创建了 <strong>${totalTodos}</strong> 个待办事项，
                已完成 <strong>${completedTodos}</strong> 项，
                完成率 <strong>${totalTodos > 0 ? Math.round(completedTodos / totalTodos * 100) : 0}%</strong>。
            </p>
            <hr class="letter-divider">
            <p class="letter-section-title">🤖 AI 统计数据</p>
            <p>
                在 AI 方面，你进行了 <strong>${aiSessions}</strong> 次会话，
                累计发送 <strong>${aiMessages.toLocaleString()}</strong> 条消息，
                消耗 <strong>${totalTokens.toLocaleString()}</strong> Token。
            </p>
            <div class="letter-stars">
                <div class="star-row">平均等待 ${avgResponseTime.toFixed(1)}s &nbsp; ${responseStars}</div>
                <div class="star-row">思考耗时 ${avgThinkingTime.toFixed(1)}s &nbsp; ${thinkingStars}</div>
                <div class="star-row">最长等待 ${maxResponseTime.toFixed(1)}s &nbsp; ${maxStars}</div>
            </div>
        `;
    }

    // 播放入场动画
    letterEl.classList.remove('reveal');
    // 强制 reflow 确保动画重新触发
    void letterEl.offsetWidth;
    letterEl.classList.add('reveal');

    // 加载备份信息
    loadBackupInfo();
}

/**
 * 清空所有 AI 会话和消息
 */
export async function clearAISessions() {
    const { nm, showConfirmDialog } = window;

    const confirmed = await showConfirmDialog('确定要清空所有 AI 会话吗？所有对话记录和消息将被永久删除，此操作不可撤销。');
    if (!confirmed) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ClearAllAISessions) {
            await window.go.main.App.ClearAllAISessions();
            nm.show('AI 会话已清空', 'success');
        } else {
            nm.show('功能不可用：后端未绑定', 'error');
        }
    } catch (err) {
        console.error('清空 AI 会话失败:', err);
        nm.show('清空失败：' + err.message, 'error');
    }
    await loadDataStats();
}

/**
 * 清空所有已完成的待办事项
 */
export async function clearCompletedTodos() {
    const { nm, showConfirmDialog } = window;

    const confirmed = await showConfirmDialog('确定要清空所有已完成的待办事项吗？此操作不可撤销。');
    if (!confirmed) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ClearCompletedTodos) {
            const msg = await window.go.main.App.ClearCompletedTodos();
            nm.show(msg, 'success');
        } else {
            nm.show('功能不可用：后端未绑定', 'error');
        }
    } catch (err) {
        console.error('清空已完成待办失败:', err);
        nm.show('清空失败：' + err.message, 'error');
    }
    await loadDataStats();
    // 如果当前在待办清单页面，刷新待办列表
    if (window._todoFilter !== undefined) {
        window.loadTodos?.();
    }
}

/**
 * 清理未引用的孤儿图片
 */
export async function cleanupOrphanImages() {
    const { nm, showConfirmDialog } = window;

    const confirmed = await showConfirmDialog('确定要清理未引用的图片吗？这将删除笔记中不再使用的图片文件。');
    if (!confirmed) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.CleanupOrphanImages) {
            const count = await window.go.main.App.CleanupOrphanImages();
            if (count > 0) {
                nm.show(`已清理 ${count} 张未引用图片`, 'success');
            } else {
                nm.show('没有需要清理的未引用图片', 'success');
            }
        } else {
            nm.show('功能不可用：后端未绑定', 'error');
        }
    } catch (err) {
        console.error('清理未引用图片失败:', err);
        nm.show('清理失败：' + err.message, 'error');
    }
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
    // 清除 AI 聊天页面的旧消息 HTML，防止后续切换视图时闪烁旧内容
    const aiMessagesEl = document.getElementById('aiChatMessages');
    if (aiMessagesEl) aiMessagesEl.innerHTML = '';
    // 清除 AI 会话侧边栏中的旧会话列表
    const aiSessionListEl = document.getElementById('aiSessionList');
    if (aiSessionListEl) aiSessionListEl.innerHTML = '';
    // 提前预加载 AI 聊天页面状态，使 AI 助手选项卡切换时不再闪烁
    window.onAIChatViewActivated?.();
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
            // VACUUM 成功后自动执行孤儿图片清理
            let imageMsg = '';
            try {
                if (window.go.main.App.CleanupOrphanImages) {
                    const count = await window.go.main.App.CleanupOrphanImages();
                    if (count > 0) {
                        imageMsg = `，已清理 ${count} 张未引用图片`;
                    } else {
                        imageMsg = '，无未引用图片';
                    }
                }
            } catch (imgErr) {
                console.error('清理未引用图片失败:', imgErr);
            }
            nm.show(msg + imageMsg, 'success');
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
 * 在文件管理器中打开日志目录
 */
export async function openLogDir() {
    const { nm } = window;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenLogDir) {
            await window.go.main.App.OpenLogDir();
        } else {
            nm.show('打开日志目录功能不可用', 'error');
        }
    } catch (e) {
        console.error('打开日志目录失败:', e);
        nm.show('打开日志目录失败', 'error');
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
                    // 提前预加载 AI 聊天页面状态
                    window.onAIChatViewActivated?.();
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
    const { els, nm, showConfirmDialog } = window;
    const btn = els.backupBtn;
    const labelEl = btn.querySelector('.dar-label');
    const origText = labelEl ? labelEl.textContent : '';

    // 确认弹窗，防止误触覆盖上次备份
    const confirmed = await showConfirmDialog('一键备份将覆盖上次备份内容，确定继续吗？');
    if (!confirmed) return;

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
