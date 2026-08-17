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
    // AI 量化索引统计（笔记数 / 片段数 / 占用字节）
    let vecNoteCount = 0, vecChunkCount = 0, vecSizeBytes = 0;

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

    // 追加获取向量索引统计
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetVectorIndexStatus) {
            const v = await window.go.main.App.GetVectorIndexStatus();
            if (v) {
                vecNoteCount = v.noteCount || 0;
                vecChunkCount = v.chunkCount || 0;
                vecSizeBytes = v.sizeBytes || 0;
            }
        }
    } catch (err) {
        console.error('加载向量索引统计失败:', err);
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

        // 向量索引统计文案（未量化时给出占位）
        const vecIndexText = vecNoteCount > 0
            ? `<strong>${vecNoteCount}</strong> 篇笔记 / <strong>${vecChunkCount}</strong> 个片段（<strong>${(vecSizeBytes / 1048576).toFixed(2)}</strong> MB）`
            : '<span style="opacity:0.55">未量化</span>';

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
            <hr class="letter-divider">
            <p class="letter-section-title">🧠 AI 量化索引</p>
            <p>向量索引：${vecIndexText}</p>
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
    // 清除 AI 聊天前端状态（消息/会话 DOM + 内存缓存），避免切换视图时残留旧内容；
    // 内部会重建 .ai-chat-messages-inner 引用，防止消息渲染进已脱离 DOM 的孤儿节点
    window.resetAIChatState?.();
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
                    // 还原后数据库已替换，前端 AI 会话缓存（activeSessionId/sessions/chatHistory）已失效：
                    // 清空状态 + 预加载 AI 聊天页面，避免切会话时空白（与 resetDatabase 行为一致）
                    window.resetAIChatState?.();
                    window.onAIChatViewActivated?.();
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

/* ===== AI 量化索引 ===== */

// 量化弹窗状态（模块级）
let vectorIndexScope = 'all';        // 当前量化范围：all / notebooks / notes
let vectorIndexAllMode = 'all';      // 「全部笔记」范围量化模式：all（全部）/ unindexed（仅未量化）/ stale（仅需重新量化）
let vectorIndexSelected = new Set(); // 当前选中的 ID 集合（笔记本 ID 或笔记 ID）
let vectorIndexNotebooks = [];       // 笔记本列表缓存（含 noteCount）
let vectorIndexNotes = [];           // 笔记列表缓存（{ id, title }）
let vectorIndexBound = false;        // 弹窗内部事件是否已绑定（懒绑定，防止重复注册）
let vectorIndexRunning = false;      // 量化是否进行中（进行中禁止关闭弹窗）
let vectorIndexChunkResetTimer = null; // 块级进度延迟清零定时器（让上一篇 100% 完整显示）
let vectorIndexPickerTimer = null; // 选择区切换动画定时器（防止动画中断残留状态）
let vectorIndexStatus = null; // 已量化索引统计缓存（noteCount/chunkCount/sizeBytes），供「全部笔记」信息卡片使用
let vectorIndexLastErrorMsg = ''; // 单篇失败即时提示去重：记录上一条错误信息
let vectorIndexLastErrorAt = 0;   // 单篇失败即时提示去重：记录上一条错误时间

/**
 * HTML 转义（用于列表标题渲染）
 * @param {*} text - 原始文本
 * @returns {string} 转义后的 HTML 字符串
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text == null ? '' : String(text);
    return div.innerHTML;
}

/**
 * 高亮匹配关键字：返回转义后的 HTML，命中片段用 <mark> 包裹（保留原文大小写）
 * @param {string} text - 原始文本
 * @param {string} keyword - 搜索关键词（原始、未转义）
 * @returns {string} 安全 HTML
 */
function highlightKeyword(text, keyword) {
    const escapedText = escapeHtml(text);
    const escapedKeyword = escapeHtml(keyword || '').toLowerCase();
    if (!escapedKeyword) return escapedText;
    const lower = escapedText.toLowerCase();
    let result = '';
    let pos = 0;
    let idx = lower.indexOf(escapedKeyword, pos);
    while (idx !== -1) {
        result += escapedText.slice(pos, idx);
        result += '<mark class="vector-index-match">' + escapedText.slice(idx, idx + escapedKeyword.length) + '</mark>';
        pos = idx + escapedKeyword.length;
        idx = lower.indexOf(escapedKeyword, pos);
    }
    result += escapedText.slice(pos);
    return result;
}

/**
 * 清理向量索引进度事件监听（弹窗关闭时调用，防止监听泄漏）
 */
function cleanupVectorIndexEvents() {
    try {
        if (window.runtime && typeof window.runtime.EventsOff === 'function') {
            window.runtime.EventsOff('vector:index-progress', 'vector:index-done', 'vector:index-error');
        }
    } catch (err) {
        console.warn('清理向量索引事件监听失败:', err);
    }
}

/**
 * 异步测试量化服务连通性（弹窗打开后后台执行，不阻塞弹窗）
 * 失败时 toast 提示，成功静默；接口异常时忽略，由开始量化时的后端校验兜底
 */
async function checkVectorIndexConnection() {
    const app = window.go?.main?.App;
    if (!app?.TestVectorIndexConnection) return;
    try {
        const res = await app.TestVectorIndexConnection();
        if (res && !res.ok && res.message) {
            window.nm?.show?.(res.message, 'error');
        }
    } catch (_) { /* 忽略异常，后端 startVectorIndex 兜底 */ }
}

/**
 * 打开 AI 量化索引弹窗（懒绑定内部事件 + 注册进度事件 + 加载列表）
 */
export async function openVectorIndexModal() {
    const modal = document.getElementById('vectorIndexModal');
    if (!modal) return;

    // 打开弹窗前先校验量化连接配置（BaseURL/APIKey/Model 必填），
    // 未配置时提示引导去设置，不打开弹窗；校验接口异常时放行，由开始量化时的后端校验兜底
    if (window.go?.main?.App?.ValidateVectorIndexConfig) {
        try {
            const check = await window.go.main.App.ValidateVectorIndexConfig();
            if (check && !check.ok) {
                window.nm?.show?.(check.message || '量化连接未配置，请先在设置中完成配置', 'warning');
                return;
            }
        } catch (_) { /* 忽略校验异常，放行 */ }
    }

    // 配置校验通过后，异步测试量化服务连通性（不阻塞弹窗打开；失败时 toast 提示，成功静默）
    checkVectorIndexConnection();

    // 懒绑定弹窗内部交互事件（只执行一次）
    bindVectorIndexModalEvents();

    // 复位弹窗状态
    vectorIndexScope = 'all';
    vectorIndexAllMode = 'all';
    vectorIndexSelected = new Set();
    vectorIndexStatus = null; // 清空上次会话残留的状态，避免首次渲染显示过期计数
    vectorIndexRunning = false;
    setVectorIndexView('select');
    document.querySelectorAll('#vectorIndexScopeSeg .vector-index-scope-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.scope === 'all');
    });
    const nbSearch = document.getElementById('vectorIndexNotebookSearch');
    const ntSearch = document.getElementById('vectorIndexNoteSearch');
    if (nbSearch) nbSearch.value = '';
    if (ntSearch) ntSearch.value = '';
    const nbPicker = document.getElementById('vectorIndexNotebookPicker');
    const ntPicker = document.getElementById('vectorIndexNotePicker');
    if (nbPicker) nbPicker.style.display = 'none';
    if (ntPicker) ntPicker.style.display = 'none';
    // 复位「全部笔记」信息卡片（初始范围即全部笔记，加载完成后显示）
    const allInfo = document.getElementById('vectorIndexAllInfo');
    if (allInfo) {
        allInfo.style.display = 'none';
        allInfo.classList.remove('picker-enter', 'picker-leave');
    }
    const summaryEl = document.getElementById('vectorIndexSummary');
    const errorEl = document.getElementById('vectorIndexError');
    if (summaryEl) summaryEl.style.display = 'none';
    if (errorEl) errorEl.style.display = 'none';
    // 复位「开始量化」按钮可用态（上次量化成功后可能残留 disabled）
    const startBtn = document.getElementById('vectorIndexStartBtn');
    if (startBtn) startBtn.disabled = false;
    updateVectorIndexCount();

    // 先清理旧监听再注册，避免重复注册
    cleanupVectorIndexEvents();
    if (window.runtime && typeof window.runtime.EventsOn === 'function') {
        window.runtime.EventsOn('vector:index-progress', (payload) => updateVectorIndexProgress(payload));
        window.runtime.EventsOn('vector:index-done', (payload) => showVectorIndexSummary(payload));
        window.runtime.EventsOn('vector:index-error', (payload) => showVectorIndexError(payload));
    }

    // 显示弹窗
    modal.style.display = 'flex';
    // 弹窗布局稳定后定位分段指示条
    repositionVectorIndexScopeIndicator();

    // 并行加载笔记本、笔记列表（弹窗交互所需）；量化状态（含逐笔记内容比对）改为异步填充，
    // 不阻塞弹窗打开——先渲染信息卡片（状态未就绪时计数为 0），状态返回后自动刷新
    await Promise.all([
        loadVectorIndexNotebooks(),
        loadVectorIndexNotes(),
    ]);
    // 初始范围为「全部笔记」：渲染并显示信息卡片
    renderVectorIndexAllInfo();
    const allInfoEl = document.getElementById('vectorIndexAllInfo');
    if (allInfoEl) allInfoEl.style.display = '';
    // 异步加载量化状态并在就绪后刷新卡片（失败静默，卡片维持 0 计数）
    loadVectorIndexStatus().then(renderVectorIndexAllInfo);
}

/**
 * 关闭 AI 量化索引弹窗（量化进行中默认禁止关闭，关闭时清理进度事件监听）
 * @param {boolean} force - true 表示用户已确认停止量化，跳过拦截直接关闭
 */
export function closeVectorIndexModal(force = false) {
    // 量化进行中不允许关闭（除非已确认停止），避免事件清理导致进度 UI 中断
    if (vectorIndexRunning && !force) {
        window.nm?.show?.('量化进行中，请等待完成', 'warning');
        return;
    }
    const modal = document.getElementById('vectorIndexModal');
    if (modal) modal.style.display = 'none';
    // 取消块级进度延迟清零定时器，避免残留回调
    if (vectorIndexChunkResetTimer) {
        clearTimeout(vectorIndexChunkResetTimer);
        vectorIndexChunkResetTimer = null;
    }
    // 取消选择区切换动画定时器，避免残留回调
    if (vectorIndexPickerTimer) {
        clearTimeout(vectorIndexPickerTimer);
        vectorIndexPickerTimer = null;
    }
    // 关闭时清理事件监听，防止泄漏
    cleanupVectorIndexEvents();
    vectorIndexSelected = new Set();
    vectorIndexScope = 'all';
}

/**
 * 关闭请求处理（右上角 X / Esc）：量化进行中先弹出「是否停止」确认框，
 * 确认后调用后端 CancelVectorIndex 停止任务并强制关闭弹窗
 */
export async function onVectorIndexCloseRequested() {
    if (!vectorIndexRunning) {
        closeVectorIndexModal();
        return;
    }
    const confirmed = await window.showConfirmDialog('量化进行中，确定要停止并关闭吗？', '停止', '继续');
    if (!confirmed) return;
    // 异步停止后端任务，随后立即关闭弹窗并清理事件（后端 goroutine 收尾期间事件已卸载）
    try { await window.go?.main?.App?.CancelVectorIndex?.(); } catch (_) { /* 忽略停止失败 */ }
    closeVectorIndexModal(true);
}

/**
 * 定位分段控件滑动指示条（按当前 active 按钮计算 left/width）
 */
function repositionVectorIndexScopeIndicator() {
    const seg = document.getElementById('vectorIndexScopeSeg');
    const indicator = document.getElementById('vectorIndexScopeIndicator');
    const active = seg?.querySelector('.vector-index-scope-btn.active');
    if (!seg || !indicator || !active) return;
    requestAnimationFrame(() => {
        indicator.style.left = active.offsetLeft + 'px';
        indicator.style.width = active.offsetWidth + 'px';
    });
}

/**
 * 切换弹窗内部视图（选择视图 / 进度视图）
 * @param {string} view - 'select' 或 'progress'
 */
function setVectorIndexView(view) {
    const selectView = document.getElementById('vectorIndexSelectView');
    const progressView = document.getElementById('vectorIndexProgressView');
    if (!selectView || !progressView) return;
    // 取消未完成的选择区切换动画，避免与视图切换冲突残留 display 状态
    if (vectorIndexPickerTimer) { clearTimeout(vectorIndexPickerTimer); vectorIndexPickerTimer = null; }
    const showEl = view === 'select' ? selectView : progressView;
    const hideEl = view === 'select' ? progressView : selectView;
    // 快速路径：目标视图已显示则不做动画（打开弹窗初始状态 / 重复调用）
    if (hideEl.style.display === 'none') return;

    // 旧视图退场（170ms）后切换 display 并给新视图入场动画
    hideEl.classList.remove('view-enter');
    hideEl.classList.add('view-leave');
    setTimeout(() => {
        hideEl.style.display = 'none';
        hideEl.classList.remove('view-leave');
        showEl.style.display = '';
        showEl.classList.remove('view-leave');
        showEl.classList.add('view-enter');
        // 入场动画结束后清理 class，避免残留
        setTimeout(() => showEl.classList.remove('view-enter'), 260);
    }, 170);
}

/**
 * 选择区进入/退场动画：先让旧选择区退场（150ms），再展示新选择区并入场（200ms）
 * @param {HTMLElement|null} showEl - 要展示的选择区；null 表示无（切到「全部笔记」）
 * @param {HTMLElement|null} hideEl - 当前可见的选择区；null 表示当前无可见选择区
 */
function animateVectorIndexPicker(showEl, hideEl) {
    if (vectorIndexPickerTimer) { clearTimeout(vectorIndexPickerTimer); vectorIndexPickerTimer = null; }
    // 快速路径：目标与当前一致（或均为空），无需动画
    if (showEl === hideEl) return;
    const done = () => {
        if (hideEl) { hideEl.style.display = 'none'; hideEl.classList.remove('picker-leave'); }
        if (showEl) {
            showEl.style.display = '';
            showEl.classList.remove('picker-leave');
            showEl.classList.add('picker-enter');
            vectorIndexPickerTimer = setTimeout(() => {
                showEl.classList.remove('picker-enter');
                vectorIndexPickerTimer = null;
            }, 220);
        }
    };
    if (!hideEl || hideEl.style.display === 'none') {
        // 旧选择区不可见：直接展示新的并入场
        done();
        return;
    }
    // 旧选择区可见：先退场再展示新的
    hideEl.classList.add('picker-leave');
    vectorIndexPickerTimer = setTimeout(() => {
        vectorIndexPickerTimer = null;
        done();
    }, 150);
}

/**
 * 切换量化范围（全部笔记 / 指定笔记本 / 指定笔记）
 * @param {string} scope - 'all' / 'notebooks' / 'notes'
 */
function switchVectorIndexScope(scope) {
    vectorIndexScope = scope;
    // 切换范围时复位「全部笔记」量化模式（默认量化全部）
    vectorIndexAllMode = 'all';
    // 更新范围按钮高亮
    document.querySelectorAll('#vectorIndexScopeSeg .vector-index-scope-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.scope === scope);
    });
    // 移动滑动指示条到新选中的按钮
    repositionVectorIndexScopeIndicator();
    // 切换对应选择区显示（带进入/退场动画）
    const nbPicker = document.getElementById('vectorIndexNotebookPicker');
    const ntPicker = document.getElementById('vectorIndexNotePicker');
    const allInfo = document.getElementById('vectorIndexAllInfo');
    if (!nbPicker || !ntPicker) return;
    const target = scope === 'notebooks' ? nbPicker : scope === 'notes' ? ntPicker : (scope === 'all' ? allInfo : null);
    // 三个区域互斥可见，最多一个处于显示态
    const visible = [allInfo, ntPicker, nbPicker].find(el => el && el.style.display !== 'none') || null;
    animateVectorIndexPicker(target, visible);
    // 切到「全部笔记」时刷新已量化统计并渲染信息卡片
    if (scope === 'all') {
        loadVectorIndexStatus().then(renderVectorIndexAllInfo);
    }
    // 切换范围时清空已选
    vectorIndexSelected = new Set();
    updateVectorIndexCount();
}

/**
 * 懒绑定弹窗内部交互事件（打开弹窗时首次调用，只绑定一次）
 */
function bindVectorIndexModalEvents() {
    if (vectorIndexBound) return;
    vectorIndexBound = true;

    // 关闭：右上角按钮走「确认停止」流程；遮罩点击保持拦截提示
    document.getElementById('vectorIndexClose')?.addEventListener('click', onVectorIndexCloseRequested);
    document.getElementById('vectorIndexOverlay')?.addEventListener('click', () => closeVectorIndexModal());

    // 范围切换
    document.querySelectorAll('#vectorIndexScopeSeg .vector-index-scope-btn').forEach(btn => {
        btn.addEventListener('click', () => switchVectorIndexScope(btn.dataset.scope));
    });

    // 搜索过滤
    document.getElementById('vectorIndexNotebookSearch')?.addEventListener('input', renderVectorIndexNotebookList);
    document.getElementById('vectorIndexNoteSearch')?.addEventListener('input', renderVectorIndexNoteList);

    // 全选 / 取消全选（作用于当前搜索过滤后的列表）
    document.getElementById('vectorIndexNotebookSelectAll')?.addEventListener('change', (e) => {
        toggleVectorIndexSelectAll('notebooks', e.target.checked);
    });
    document.getElementById('vectorIndexNoteSelectAll')?.addEventListener('change', (e) => {
        toggleVectorIndexSelectAll('notes', e.target.checked);
    });

    // 列表项勾选（事件委托，动态渲染的 checkbox）
    document.getElementById('vectorIndexNotebookList')?.addEventListener('change', (e) => {
        const item = e.target.closest('.vector-index-item');
        if (!item) return;
        const id = Number(item.dataset.id);
        if (e.target.checked) vectorIndexSelected.add(id); else vectorIndexSelected.delete(id);
        updateVectorIndexCount();
        syncVectorIndexSelectAllState('notebooks');
    });
    document.getElementById('vectorIndexNoteList')?.addEventListener('change', (e) => {
        const item = e.target.closest('.vector-index-item');
        if (!item) return;
        const id = Number(item.dataset.id);
        if (e.target.checked) vectorIndexSelected.add(id); else vectorIndexSelected.delete(id);
        updateVectorIndexCount();
        syncVectorIndexSelectAllState('notes');
    });

    // 开始量化
    document.getElementById('vectorIndexStartBtn')?.addEventListener('click', startVectorIndex);
}

/**
 * 加载笔记本列表并渲染（调 App.GetAllNotebooks，附带笔记数）
 */
async function loadVectorIndexNotebooks() {
    const listEl = document.getElementById('vectorIndexNotebookList');
    if (!listEl) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllNotebooks) {
            const notebooks = await window.go.main.App.GetAllNotebooks();
            vectorIndexNotebooks = notebooks || [];
            // 一并获取笔记数，供列表右侧展示
            try {
                if (window.go.main.App.GetNotebookNoteCounts) {
                    const counts = await window.go.main.App.GetNotebookNoteCounts() || {};
                    vectorIndexNotebooks.forEach(nb => { nb.noteCount = counts[nb.id] || 0; });
                }
            } catch (_) { /* 忽略计数失败 */ }
        } else {
            vectorIndexNotebooks = [];
        }
    } catch (err) {
        console.error('加载笔记本列表失败:', err);
        vectorIndexNotebooks = [];
    }
    renderVectorIndexNotebookList();
}

/**
 * 加载全部笔记（ID + 标题）并渲染（通过 App.GetNotes 分页循环拉取）
 */
async function loadVectorIndexNotes() {
    const listEl = document.getElementById('vectorIndexNoteList');
    if (!listEl) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNotes) {
            // 先取总数，再按较大分页循环拉取全部笔记（仅保留 id + title）
            const first = await window.go.main.App.GetNotes(1, 1, 'updated_at', 0);
            const total = first?.total || 0;
            const pageSize = 200;
            const pageCount = Math.max(1, Math.ceil(total / pageSize));
            vectorIndexNotes = [];
            for (let page = 1; page <= pageCount; page++) {
                const res = await window.go.main.App.GetNotes(page, pageSize, 'updated_at', 0);
                const items = res?.items || [];
                items.forEach(n => vectorIndexNotes.push({ id: n.id, title: n.title || '未命名笔记' }));
            }
        } else {
            vectorIndexNotes = [];
        }
    } catch (err) {
        console.error('加载笔记列表失败:', err);
        vectorIndexNotes = [];
    }
    renderVectorIndexNoteList();
}

/**
 * 获取量化弹窗完整状态（GetVectorIndexOverview：全局统计 + 未量化/需重新量化/已最新分类），
 * 供「全部笔记」信息卡片使用；注意该接口含逐笔记内容比对，仅弹窗调用
 */
async function loadVectorIndexStatus() {
    vectorIndexStatus = null;
    try {
        if (window.go?.main?.App?.GetVectorIndexOverview) {
            const v = await window.go.main.App.GetVectorIndexOverview();
            if (v) {
                vectorIndexStatus = {
                    noteCount: v.noteCount || 0,
                    chunkCount: v.chunkCount || 0,
                    sizeBytes: v.sizeBytes || 0,
                    totalNotes: v.totalNotes || 0,
                    unindexedNotes: v.unindexedNotes || 0,
                    staleNotes: v.staleNotes || 0,
                    upToDateNotes: v.upToDateNotes || 0,
                };
            }
        }
    } catch (_) { /* 忽略统计失败，卡片按 0 显示 */ }
}

/**
 * 定位「全部笔记」模式分段滑块的滑动指示条（复刻设置页排序方式控件的算法）
 * segW = (容器宽 - 8) / 按钮数，指示条 translateX(2 + index * segW)，宽度 = segW
 */
function repositionVectorIndexAllModeIndicator() {
    const seg = document.getElementById('vectorIndexAllModeSeg');
    const indicator = document.getElementById('vectorIndexAllModeIndicator');
    if (!seg || !indicator) return;
    const btns = Array.from(seg.querySelectorAll('.segmented-btn'));
    const active = btns.find(b => b.classList.contains('active'));
    if (!active) return;
    const cw = seg.offsetWidth;
    if (cw === 0) return; // 容器尚未布局（如初次打开时 display:none 刚切换），跳过由 rAF 重试
    const segW = (cw - 8) / btns.length;
    indicator.style.transform = `translateX(${2 + btns.indexOf(active) * segW}px)`;
    indicator.style.width = `${segW}px`;
}

/**
 * 渲染「全部笔记」信息卡片（未量化 / 需重新量化 / 已量化最新 / 总笔记 / 片段 / 占用 + 量化模式分段滑块）
 * 「需重新量化」= 已量化但内容（标题/正文/标签/创建时间参与切块的全部输入）与量化时不一致
 */
function renderVectorIndexAllInfo() {
    const el = document.getElementById('vectorIndexAllInfo');
    if (!el) return;
    const total = vectorIndexStatus?.totalNotes ?? vectorIndexNotes.length;
    const unindexed = vectorIndexStatus?.unindexedNotes || 0;
    const stale = vectorIndexStatus?.staleNotes || 0;
    const upToDate = vectorIndexStatus?.upToDateNotes || 0;
    const chunks = vectorIndexStatus?.chunkCount || 0;
    const sizeMB = ((vectorIndexStatus?.sizeBytes || 0) / 1048576).toFixed(2);
    if (total === 0) {
        el.innerHTML = '<p class="vector-index-all-note">当前没有可量化的笔记</p>';
        return;
    }
    el.innerHTML = `
        <div class="vector-index-all-cards">
            <div class="vector-index-all-card">
                <div class="vector-index-all-card-num">${unindexed}</div>
                <div class="vector-index-all-card-label">未量化</div>
            </div>
            <div class="vector-index-all-card">
                <div class="vector-index-all-card-num">${stale}</div>
                <div class="vector-index-all-card-label" title="已量化但内容（标题/正文/标签等）已编辑变化的笔记">需重新量化</div>
            </div>
            <div class="vector-index-all-card">
                <div class="vector-index-all-card-num">${upToDate}</div>
                <div class="vector-index-all-card-label">已量化（最新）</div>
            </div>
            <div class="vector-index-all-card">
                <div class="vector-index-all-card-num">${total}</div>
                <div class="vector-index-all-card-label">总笔记数</div>
            </div>
            <div class="vector-index-all-card">
                <div class="vector-index-all-card-num">${chunks}</div>
                <div class="vector-index-all-card-label">片段数</div>
            </div>
            <div class="vector-index-all-card">
                <div class="vector-index-all-card-num vector-index-all-card-num-sm">${sizeMB} MB</div>
                <div class="vector-index-all-card-label">占用空间</div>
            </div>
        </div>
        <div class="segmented-control vector-index-all-mode" id="vectorIndexAllModeSeg">
            <div class="segmented-indicator" id="vectorIndexAllModeIndicator"></div>
            <button type="button" class="segmented-btn${vectorIndexAllMode === 'all' ? ' active' : ''}" data-mode="all">量化全部</button>
            <button type="button" class="segmented-btn${vectorIndexAllMode === 'unindexed' ? ' active' : ''}" data-mode="unindexed"${unindexed === 0 ? ' disabled title="没有未量化的笔记"' : ''}>仅未量化</button>
            <button type="button" class="segmented-btn${vectorIndexAllMode === 'stale' ? ' active' : ''}" data-mode="stale"${stale === 0 ? ' disabled title="没有需要重新量化的笔记"' : ''}>仅需重新量化</button>
        </div>`;
    // 模式切换：更新 active 态并滑动指示条（数量为 0 的选项已 disabled，无需额外校验）
    el.querySelectorAll('#vectorIndexAllModeSeg .segmented-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            if (btn.disabled) return;
            vectorIndexAllMode = btn.dataset.mode;
            el.querySelectorAll('#vectorIndexAllModeSeg .segmented-btn').forEach(b => b.classList.toggle('active', b === btn));
            repositionVectorIndexAllModeIndicator();
        });
    });
    // 初次渲染时容器可能刚从 display:none 变为可见，rAF 内布局完成后再定位指示条
    requestAnimationFrame(repositionVectorIndexAllModeIndicator);
}

/**
 * 渲染笔记本多选列表（按搜索词过滤，保留勾选状态）
 */
function renderVectorIndexNotebookList() {
    const listEl = document.getElementById('vectorIndexNotebookList');
    if (!listEl) return;
    const keyword = (document.getElementById('vectorIndexNotebookSearch')?.value || '').trim().toLowerCase();
    const filtered = vectorIndexNotebooks.filter(nb => !keyword || String(nb.name || '').toLowerCase().includes(keyword));

    if (filtered.length === 0) {
        listEl.innerHTML = '<div class="vector-index-empty">没有可用的笔记本</div>';
        syncVectorIndexSelectAllState('notebooks');
        return;
    }
    listEl.innerHTML = filtered.map(nb => `
        <label class="vector-index-item" data-id="${nb.id}">
            <input type="checkbox" ${vectorIndexSelected.has(nb.id) ? 'checked' : ''}>
            <span class="vector-index-item-title">${highlightKeyword(nb.name || '未命名笔记本', keyword)}</span>
            <span class="vector-index-item-sub">${nb.noteCount || 0} 篇</span>
        </label>
    `).join('');
    syncVectorIndexSelectAllState('notebooks');
}

/**
 * 渲染笔记多选列表（按搜索词过滤，保留勾选状态）
 */
function renderVectorIndexNoteList() {
    const listEl = document.getElementById('vectorIndexNoteList');
    if (!listEl) return;
    const keyword = (document.getElementById('vectorIndexNoteSearch')?.value || '').trim().toLowerCase();
    const filtered = vectorIndexNotes.filter(n => !keyword || String(n.title || '').toLowerCase().includes(keyword));

    if (filtered.length === 0) {
        listEl.innerHTML = '<div class="vector-index-empty">没有匹配的笔记</div>';
        syncVectorIndexSelectAllState('notes');
        return;
    }
    listEl.innerHTML = filtered.map(n => `
        <label class="vector-index-item" data-id="${n.id}">
            <input type="checkbox" ${vectorIndexSelected.has(n.id) ? 'checked' : ''}>
            <span class="vector-index-item-title">${highlightKeyword(n.title, keyword)}</span>
        </label>
    `).join('');
    syncVectorIndexSelectAllState('notes');
}

/**
 * 更新底部已选计数文案（「全部笔记」范围不显示计数）
 */
function updateVectorIndexCount() {
    const countEl = document.getElementById('vectorIndexCount');
    if (!countEl) return;
    // 「全部笔记」范围：隐藏已选计数，底部按钮居中
    if (vectorIndexScope === 'all') {
        countEl.style.display = 'none';
        const footer = countEl.parentElement;
        if (footer) footer.classList.add('centered');
        return;
    }
    countEl.style.display = '';
    const footer = countEl.parentElement;
    if (footer) footer.classList.remove('centered');
    const n = vectorIndexSelected.size;
    countEl.textContent = vectorIndexScope === 'notebooks' ? `已选 ${n} 个笔记本` : `已选 ${n} 篇`;
}

/**
 * 全选 / 取消全选（仅作用于当前搜索过滤后的列表）
 * @param {string} scope - 'notebooks' / 'notes'
 * @param {boolean} checked - 是否勾选
 */
function toggleVectorIndexSelectAll(scope, checked) {
    const source = scope === 'notebooks' ? vectorIndexNotebooks : vectorIndexNotes;
    const searchEl = document.getElementById(scope === 'notebooks' ? 'vectorIndexNotebookSearch' : 'vectorIndexNoteSearch');
    const keyword = (searchEl?.value || '').trim().toLowerCase();
    source.forEach(item => {
        const title = String(scope === 'notebooks' ? item.name : item.title || '');
        if (keyword && !title.toLowerCase().includes(keyword)) return;
        if (checked) vectorIndexSelected.add(item.id); else vectorIndexSelected.delete(item.id);
    });
    if (scope === 'notebooks') renderVectorIndexNotebookList(); else renderVectorIndexNoteList();
    updateVectorIndexCount();
}

/**
 * 同步「全选」复选框状态（过滤后全部勾选 → 勾选；部分 → 半选；无 → 取消）
 * @param {string} scope - 'notebooks' / 'notes'
 */
function syncVectorIndexSelectAllState(scope) {
    const checkbox = document.getElementById(scope === 'notebooks' ? 'vectorIndexNotebookSelectAll' : 'vectorIndexNoteSelectAll');
    if (!checkbox) return;
    const source = scope === 'notebooks' ? vectorIndexNotebooks : vectorIndexNotes;
    const searchEl = document.getElementById(scope === 'notebooks' ? 'vectorIndexNotebookSearch' : 'vectorIndexNoteSearch');
    const keyword = (searchEl?.value || '').trim().toLowerCase();
    const visible = source.filter(item => !keyword || String(scope === 'notebooks' ? item.name : item.title || '').toLowerCase().includes(keyword));
    const allChecked = visible.length > 0 && visible.every(item => vectorIndexSelected.has(item.id));
    checkbox.checked = allChecked;
    checkbox.indeterminate = !allChecked && visible.some(item => vectorIndexSelected.has(item.id));
}

/**
 * 开始量化：按当前范围调用对应后端方法，成功后切换到进度视图
 */
async function startVectorIndex() {
    const { nm } = window;
    const app = window.go?.main?.App;
    if (!app) {
        nm.show('量化功能不可用：后端未绑定', 'error');
        return;
    }

    // 根据范围选择后端方法
    let fn = null;
    let args = [];
    if (vectorIndexScope === 'all') {
        // 「全部笔记」范围：按所选量化模式分发（全部 / 仅未量化 / 仅需重新量化）
        if (vectorIndexAllMode === 'unindexed') {
            if ((vectorIndexStatus?.unindexedNotes || 0) === 0) {
                nm.show('所有笔记都已量化，无需处理', 'info');
                return;
            }
            fn = app.IndexNotesUnindexed;
            args = [];
        } else if (vectorIndexAllMode === 'stale') {
            if ((vectorIndexStatus?.staleNotes || 0) === 0) {
                nm.show('没有需要重新量化的笔记', 'info');
                return;
            }
            fn = app.IndexNotesStale;
            args = [];
        } else {
            fn = app.IndexNotesByAll;
            args = [];
        }
    } else if (vectorIndexScope === 'notebooks') {
        if (vectorIndexSelected.size === 0) {
            nm.show('请先选择至少一个笔记本', 'warning');
            return;
        }
        fn = app.IndexNotesByNotebooks;
        args = [Array.from(vectorIndexSelected)];
    } else {
        if (vectorIndexSelected.size === 0) {
            nm.show('请先选择至少一篇笔记', 'warning');
            return;
        }
        fn = app.IndexNotesByIDs;
        args = [Array.from(vectorIndexSelected)];
    }

    if (typeof fn !== 'function') {
        nm.show('量化功能不可用：后端未绑定', 'error');
        return;
    }

    // 切换到进度视图并复位进度显示
    vectorIndexRunning = true;
    setVectorIndexView('progress');
    resetVectorIndexProgressUI();
    const startBtn = document.getElementById('vectorIndexStartBtn');
    if (startBtn) startBtn.disabled = true;

    try {
        // 调用后端开始量化（进度由 vector:index-progress 等事件驱动）
        await fn.apply(app, args);
    } catch (err) {
        console.error('开始量化失败:', err);
        vectorIndexRunning = false;
        if (startBtn) startBtn.disabled = false;
        showVectorIndexError({ error: err?.message || String(err) });
    }
}

/**
 * 复位进度视图显示（进度条 / 百分比 / 阶段 / 当前标题 / 摘要 / 错误）
 */
function resetVectorIndexProgressUI() {
    // 取消块级进度延迟清零定时器，避免残留回调污染下一次进度显示
    if (vectorIndexChunkResetTimer) {
        clearTimeout(vectorIndexChunkResetTimer);
        vectorIndexChunkResetTimer = null;
    }
    const fill = document.getElementById('vectorIndexProgressFill');
    const percent = document.getElementById('vectorIndexProgressPercent');
    const stage = document.getElementById('vectorIndexProgressStage');
    const current = document.getElementById('vectorIndexCurrentTitle');
    const summary = document.getElementById('vectorIndexSummary');
    const error = document.getElementById('vectorIndexError');
    const chunkFill = document.getElementById('vectorIndexChunkFill');
    const chunkPercent = document.getElementById('vectorIndexChunkPercent');
    const chunkStage = document.getElementById('vectorIndexChunkStage');
    // 新任务开始：恢复分块进度区块与「当前处理笔记」标题显示
    const chunkBlock = document.getElementById('vectorIndexChunkBlock');
    if (chunkBlock) chunkBlock.style.display = '';
    if (current) current.style.display = '';
    if (fill) { fill.style.width = '0%'; fill.classList.remove('is-done'); }
    if (percent) percent.textContent = '0%';
    if (stage) stage.textContent = '准备中…';
    if (current) current.textContent = '';
    if (summary) summary.style.display = 'none';
    if (error) error.style.display = 'none';
    if (chunkFill) chunkFill.style.width = '0%';
    if (chunkPercent) chunkPercent.textContent = '0%';
    if (chunkStage) chunkStage.textContent = '';
}

/**
 * 更新量化进度（vector:index-progress 事件回调）
 * @param {{done:number, total:number, title:string, stage:string, chunk_done:number, chunk_total:number}} payload - 进度负载
 */
function updateVectorIndexProgress(payload) {
    const p = payload || {};
    const done = Number(p.done) || 0;
    const total = Number(p.total) || 0;
    const stage = p.stage;
    // embedding 阶段当前篇按「处理到一半」计：单篇量化时进度从 50% 起步，
    // 避免 embedding 期间长时间停在 0%；done/error 阶段按实际完成篇数计
    const isEmbedding = stage === 'embedding';
    const percent = total > 0
        ? Math.min(100, Math.round((isEmbedding ? done + 0.5 : done) / total * 100))
        : 0;

    const fill = document.getElementById('vectorIndexProgressFill');
    if (fill) fill.style.width = percent + '%';
    const percentEl = document.getElementById('vectorIndexProgressPercent');
    if (percentEl) percentEl.textContent = percent + '%';

    // 阶段文案映射
    const stageMap = { embedding: '正在生成向量…', done: '量化完成', error: '处理失败，跳过' };
    const stageEl = document.getElementById('vectorIndexProgressStage');
    if (stageEl && p.stage) stageEl.textContent = stageMap[p.stage] || p.stage;
    const currentEl = document.getElementById('vectorIndexCurrentTitle');
    if (currentEl && p.title) currentEl.textContent = p.title;

    // 单篇处理失败：即时弹通知提示原因（同一错误 3 秒内去重，避免批量失败刷屏）
    if (stage === 'error' && p.error) {
        const now = Date.now();
        if (p.error !== vectorIndexLastErrorMsg || now - vectorIndexLastErrorAt > 3000) {
            window.nm?.show?.(`笔记「${p.title || '未知'}」量化失败：${p.error}`, 'error');
        }
        vectorIndexLastErrorMsg = p.error;
        vectorIndexLastErrorAt = now;
    }

    // 块级进度：当前笔记分块的向量化进度
    const chunkDone = Number(p.chunk_done) || 0;
    const chunkTotal = Number(p.chunk_total) || 0;

    // 块级进度条 DOM 更新函数
    const setChunkUI = (percent, text) => {
        const chunkFill = document.getElementById('vectorIndexChunkFill');
        if (chunkFill) chunkFill.style.width = percent + '%';
        const chunkPercentEl = document.getElementById('vectorIndexChunkPercent');
        if (chunkPercentEl) chunkPercentEl.textContent = percent + '%';
        const chunkStageEl = document.getElementById('vectorIndexChunkStage');
        if (chunkStageEl) chunkStageEl.textContent = text || '';
    };

    // 新篇开始（块级进度归 0 的事件）：延迟 250ms 再清零，
    // 让上一篇进度条完整走到 100% 后再切换到新笔记（避免 100% 被瞬间覆盖）
    if (chunkTotal > 0 && chunkDone === 0) {
        if (vectorIndexChunkResetTimer) clearTimeout(vectorIndexChunkResetTimer);
        vectorIndexChunkResetTimer = setTimeout(() => {
            vectorIndexChunkResetTimer = null;
            setChunkUI(0, `本笔记 0/${chunkTotal} 块`);
        }, 250);
        return;
    }

    // 有真实块级进度：立即更新并取消延迟清零
    if (vectorIndexChunkResetTimer) {
        clearTimeout(vectorIndexChunkResetTimer);
        vectorIndexChunkResetTimer = null;
    }
    const chunkPercent = chunkTotal > 0 ? Math.min(100, Math.round(chunkDone / chunkTotal * 100)) : 0;
    setChunkUI(chunkPercent, chunkTotal > 0 ? `本笔记 ${chunkDone}/${chunkTotal} 块` : '');
}

/**
 * 量化完成回调（vector:index-done 事件）：显示摘要并刷新信笺统计
 * @param {{success:number, failed:number}} payload - 完成负载
 */
async function showVectorIndexSummary(payload) {
    const p = payload || {};
    const success = Number(p.success) || 0;
    const failed = Number(p.failed) || 0;
    vectorIndexRunning = false;

    const summary = document.getElementById('vectorIndexSummary');
    if (summary) {
        summary.style.display = '';
        summary.innerHTML = `量化完成：成功 <strong>${success}</strong> 篇 / 失败 <strong>${failed}</strong> 篇`;
    }
    const stage = document.getElementById('vectorIndexProgressStage');
    if (stage) stage.textContent = '已完成';
    // 进度条切换为完成态（成功色 + 单次脉冲）
    const fill = document.getElementById('vectorIndexProgressFill');
    if (fill) fill.classList.add('is-done');
    // 全部处理完成后：隐藏分块进度条与「当前处理笔记」标题，仅保留笔记总进度条与摘要
    const chunkBlock = document.getElementById('vectorIndexChunkBlock');
    if (chunkBlock) chunkBlock.style.display = 'none';
    const currentTitle = document.getElementById('vectorIndexCurrentTitle');
    if (currentTitle) currentTitle.style.display = 'none';
    // 完成后刷新信笺统计，保持最新
    loadDataStats();
}

/**
 * 量化错误回调（vector:index-error 事件）：展示错误信息
 * @param {{error:string}} payload - 错误负载
 */
function showVectorIndexError(payload) {
    const p = payload || {};
    vectorIndexRunning = false;
    const errorEl = document.getElementById('vectorIndexError');
    if (errorEl) {
        errorEl.style.display = '';
        errorEl.textContent = `量化失败：${p.error || '未知错误'}`;
    }
    const stage = document.getElementById('vectorIndexProgressStage');
    if (stage) stage.textContent = '已中断';
}

/**
 * 删除所有量化内容（二次确认后调 App.DeleteAllVectors，完成后刷新统计）
 */
export async function deleteAllVectors() {
    const { nm, showConfirmDialog } = window;

    const confirmed = await showConfirmDialog('确定要删除所有量化内容吗？笔记的向量索引数据将被清空，此操作不可撤销。');
    if (!confirmed) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteAllVectors) {
            await window.go.main.App.DeleteAllVectors();
            nm.show('量化内容已删除', 'success');
        } else {
            nm.show('功能不可用：后端未绑定', 'error');
        }
    } catch (err) {
        console.error('删除量化内容失败:', err);
        nm.show('删除失败：' + err.message, 'error');
    }
    await loadDataStats();
}
