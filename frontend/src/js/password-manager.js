/**
 * 密码管理模块 - 列表展示 + 右键菜单 + 添加/编辑对话框 + 查看详情 + 批量操作
 * 数据来源均为 Wails 绑定方法（window.go.main.App.*）
 */

// 各字段最大长度（与后端 PasswordRecord 列定义一致；备注为 text 类型不限长）
const PM_MAX_LENGTH = { name: 200, username: 200, password: 500, url: 500 };

/** 密码掩码占位 */
const PM_PWD_MASK = '••••••';

// ---------- DOM 引用 ----------
let pmView, pmListEl, pmEmptyEl, pmSearchInput, pmSearchClearBtn, pmAddBtn, pmBatchToggleBtn;
let pmBatchBar, pmSelectAllCb, pmSelectedCountEl, pmBatchDeleteBtn;
let pmEditOverlay, pmEditTitle, pmEditCloseBtn, pmCancelBtn, pmSaveBtn;
let pmFieldName, pmFieldUsername, pmFieldPassword, pmPwdToggleBtn, pmFieldUrl, pmFieldNote;
let pmDetailOverlay, pmDetailTitle, pmDetailCloseBtn;
let pmDetailName, pmDetailUsername, pmCopyUsernameBtn;
let pmDetailPassword, pmDetailPwdToggleBtn, pmCopyPasswordBtn;
let pmDetailUrlRow, pmDetailUrl, pmCopyUrlBtn, pmOpenUrlBtn;
let pmDetailNote, pmDetailCreatedAt, pmDetailUpdatedAt;
let pmDetailDeleteBtn, pmDetailEditBtn;
let pmContextMenu, pmTooltip;

// ---------- 运行时状态 ----------
let pmRecords = [];            // 当前列表数据（PasswordListItem[]，不含密码）
let pmKeyword = '';            // 当前搜索关键词
let pmSearchTimer = null;      // 搜索防抖定时器
let pmBatchMode = false;       // 是否处于批量模式
let pmSelectedIds = new Set(); // 批量模式选中的记录 ID
let editingId = null;          // 编辑对话框的记录 ID（null = 新增）
let pmEditSnapshot = { name: '', username: '', password: '', url: '', note: '' }; // 编辑对话框打开时的表单快照（用于关闭时判断未保存修改）
let detailRecord = null;       // 详情对话框当前记录（含解码后的明文密码）
let detailPwdVisible = false;  // 详情对话框中密码是否可见
let _lastLengthNotifyAt = 0;   // 长度超限通知节流时间戳
let pmLoadSeq = 0;             // 列表加载序号（防慢响应乱序覆盖新结果）

/* ==================== 工具函数 ==================== */

/**
 * 安全复制文本到剪贴板（主路径 navigator.clipboard，失败降级 execCommand）
 * @param {string} text - 要复制的文本
 * @param {string} successMsg - 成功提示文案
 * @returns {Promise<boolean>} 是否复制成功
 */
async function pmCopyText(text, successMsg) {
    if (typeof text !== 'string' || text.length === 0) {
        window.nm?.show('内容为空，无需复制', 'info');
        return false;
    }
    try {
        if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            await navigator.clipboard.writeText(text);
            window.nm?.show(successMsg, 'success');
            return true;
        }
    } catch (e) {
        // 拒签或不支持时走降级
    }
    // 降级：隐藏 textarea + execCommand
    try {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.setAttribute('readonly', '');
        ta.style.position = 'fixed';
        ta.style.top = '0';
        ta.style.left = '0';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.focus();
        ta.select();
        const ok = document.execCommand('copy');
        document.body.removeChild(ta);
        if (ok) {
            window.nm?.show(successMsg, 'success');
            return true;
        }
        window.nm?.show('复制失败，请手动复制', 'error');
        return false;
    } catch (e) {
        window.nm?.show('复制失败，请手动复制', 'error');
        return false;
    }
}

/**
 * 格式化时间为 YYYY-MM-DD HH:mm:ss
 */
function formatDateTime(str) {
    if (!str) return '-';
    const d = new Date(str);
    if (isNaN(d.getTime())) return str;
    const p = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

/**
 * 将文本写入元素，搜索关键词命中部分用 <mark class="pm-mark"> 高亮
 * @param {HTMLElement} el - 目标元素（内容会被清空重填）
 * @param {string} text - 原始文本
 */
function pmFillWithHighlight(el, text) {
    el.textContent = '';
    if (!text) return;
    const kw = pmKeyword.trim();
    // 无关键词或含正则元字符时安全转义后再匹配（忽略大小写，与后端搜索行为一致）
    if (!kw) {
        el.textContent = text;
        return;
    }
    const escaped = kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const re = new RegExp(escaped, 'gi');
    let last = 0;
    let m;
    while ((m = re.exec(text)) !== null) {
        if (m[0].length === 0) { re.lastIndex++; continue; }
        if (m.index > last) {
            el.appendChild(document.createTextNode(text.slice(last, m.index)));
        }
        const mark = document.createElement('mark');
        mark.className = 'pm-mark';
        mark.textContent = m[0];
        el.appendChild(mark);
        last = m.index + m[0].length;
    }
    if (last < text.length) {
        el.appendChild(document.createTextNode(text.slice(last)));
    }
}

/**
 * 触发元素抖动动画（重复触发时先移除再重放）
 */
function triggerShake(el) {
    if (!el) return;
    el.classList.remove('pm-shake');
    void el.offsetWidth; // 强制 reflow 以重启动画
    el.classList.add('pm-shake');
}

/**
 * 长度超限通知（节流：1.5s 内最多提示一次）
 */
function notifyLengthLimit(max) {
    const now = Date.now();
    if (now - _lastLengthNotifyAt < 1500) return;
    _lastLengthNotifyAt = now;
    window.nm?.show(`最多只能输入 ${max} 个字符`, 'warning');
}

/**
 * 为输入框绑定实时长度校验：超出自动截断 + 抖动 + 通知
 */
function bindLengthGuard(input, max) {
    input.addEventListener('input', () => {
        if (input.value.length > max) {
            input.value = input.value.slice(0, max);
            triggerShake(input.closest('.pm-field') || input);
            notifyLengthLimit(max);
        }
    });
}

/**
 * 给定记录 ID 从当前列表数据取记录
 */
function findInList(id) {
    return pmRecords.find((r) => Number(r.id) === Number(id));
}

/* ==================== 列表加载与渲染 ==================== */

/**
 * 根据关键词加载记录列表（有词走 Search，否则 List），然后渲染
 * @param {object} [renderOpts] - 透传给 renderPmList（playEnter/flashId）
 */
async function loadPmRecords(renderOpts = {}) {
    // 每次加载分配新序号，返回时若已有更新的请求则丢弃本次结果（快速搜索时防旧响应覆盖新结果）
    const seq = ++pmLoadSeq;
    try {
        const kw = pmKeyword.trim();
        const app = window.go?.main?.App;
        if (!app) return;
        let data;
        if (kw) {
            data = await app.SearchPasswordRecords(kw);
        } else {
            data = await app.ListPasswordRecords();
        }
        if (seq !== pmLoadSeq) return;
        pmRecords = Array.isArray(data) ? data : [];
        // 过滤掉批量模式中已被删除的选择项
        const validIds = new Set(pmRecords.map((r) => Number(r.id)));
        pmSelectedIds.forEach((id) => {
            if (!validIds.has(Number(id))) pmSelectedIds.delete(Number(id));
        });
        renderPmList(renderOpts);
    } catch (e) {
        console.warn('加载密码记录失败:', e);
        window.nm?.show('加载密码记录失败', 'error');
    }
}

/* 行内小图标：用户名 / 链接 */
const PM_ICON_USER = '<svg class="pm-field-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>';
const PM_ICON_LINK = '<svg class="pm-field-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>';

/**
 * 渲染列表（三栏等宽：名称 | 用户名 | URL；批量模式追加左侧复选框）
 * @param {object} [opts]
 * @param {boolean} [opts.playEnter=false] - 是否播放逐条入场动画（仅进入视图时为 true）
 * @param {number|null} [opts.flashId=null] - 操作后需要高亮呼吸的记录 ID
 */
function renderPmList({ playEnter = false, flashId = null } = {}) {
    hideContextMenu();
    hideTooltip();
    // 清空前记录滚动位置，重建后还原，避免长列表操作后视角跳动
    const prevScrollTop = pmListEl.scrollTop;
    pmListEl.innerHTML = '';

    if (!pmRecords || pmRecords.length === 0) {
        pmEmptyEl.style.display = 'flex';
        return;
    }
    pmEmptyEl.style.display = 'none';

    const fragment = document.createDocumentFragment();

    pmRecords.forEach((rec, index) => {
        const item = document.createElement('div');
        item.className = 'pm-item';
        item.dataset.id = rec.id;
        if (playEnter) {
            item.style.animationDelay = `${Math.min(index * 0.03, 0.3)}s`;
        } else {
            item.classList.add('pm-no-enter');
        }
        if (flashId != null && Number(rec.id) === Number(flashId)) {
            item.classList.add('pm-flash');
        }

        // 左侧复选框（仅批量模式可见）
        const check = document.createElement('button');
        check.type = 'button';
        check.className = 'pm-check';
        check.title = '选中该记录';
        check.innerHTML = '<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
        if (pmSelectedIds.has(Number(rec.id))) {
            check.classList.add('checked');
            item.classList.add('selected');
        }
        check.addEventListener('click', (e) => {
            e.stopPropagation();
            toggleItemChecked(item, check);
        });

        // 三栏主体（信息层级：图标 + 文本）
        const main = document.createElement('div');
        main.className = 'pm-item-main';

        const nameEl = document.createElement('span');
        nameEl.className = 'pm-name';
        pmFillWithHighlight(nameEl, rec.name || '-');

        const userEl = document.createElement('span');
        userEl.className = 'pm-meta';
        userEl.innerHTML = PM_ICON_USER;
        const userText = document.createElement('span');
        userText.className = 'pm-meta-text';
        pmFillWithHighlight(userText, rec.username || '-');
        userEl.appendChild(userText);

        const urlCell = document.createElement('span');
        urlCell.className = 'pm-meta pm-url-cell';
        urlCell.innerHTML = PM_ICON_LINK;
        const urlText = document.createElement('span');
        urlText.className = 'pm-url-text';
        if (rec.url) {
            // 展示时剥离协议前缀降噪，hover tooltip 仍提示完整 URL
            pmFillWithHighlight(urlText, rec.url.replace(/^https?:\/\//i, ''));
            // URL 超长时通过 hover 自定义 tooltip 显示完整内容
            urlText.addEventListener('mouseenter', () => showTooltip(rec.url));
            urlText.addEventListener('mousemove', (ev) => moveTooltip(ev.clientX, ev.clientY));
            urlText.addEventListener('mouseleave', hideTooltip);
        } else {
            urlText.textContent = '-';
            urlText.classList.add('empty');
        }
        urlCell.appendChild(urlText);

        main.appendChild(nameEl);
        main.appendChild(userEl);
        main.appendChild(urlCell);

        item.appendChild(check);
        item.appendChild(main);

        // 右键菜单
        item.addEventListener('contextmenu', (e) => {
            e.preventDefault();
            openPmContextMenu(e, Number(rec.id));
        });

        // 左键：批量模式切换勾选，普通模式打开详情
        item.addEventListener('click', () => {
            if (pmBatchMode) {
                toggleItemChecked(item, check);
            } else {
                openPmDetail(Number(rec.id));
            }
        });

        fragment.appendChild(item);
    });

    pmListEl.appendChild(fragment);
    pmListEl.scrollTop = prevScrollTop;
    updatePmSelectionUI();
}

/* ==================== Tooltip ==================== */

function showTooltip(text) {
    if (!text) return;
    pmTooltip.textContent = text;
    pmTooltip.classList.add('visible');
}

function moveTooltip(x, y) {
    // 显示后才知道尺寸，先粗略定位，下一帧微调防止溢出视口
    pmTooltip.style.left = `${x + 14}px`;
    pmTooltip.style.top = `${y + 16}px`;
    requestAnimationFrame(() => {
        const rect = pmTooltip.getBoundingClientRect();
        let left = x + 14;
        let top = y + 16;
        if (left + rect.width > window.innerWidth - 8) left = x - rect.width - 10;
        if (top + rect.height > window.innerHeight - 8) top = y - rect.height - 10;
        pmTooltip.style.left = `${Math.max(8, left)}px`;
        pmTooltip.style.top = `${Math.max(8, top)}px`;
    });
}

function hideTooltip() {
    pmTooltip.classList.remove('visible');
}

/* ==================== 右键菜单 ==================== */

/**
 * 构建并弹出右键菜单
 * @param {MouseEvent} e
 * @param {number} id - 记录 ID
 */
function openPmContextMenu(e, id) {
    const rec = findInList(id);
    if (!rec) return;
    hideTooltip();

    pmContextMenu.innerHTML = '';

    const mkItem = (label, onClick, danger = false) => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'pm-context-menu-item' + (danger ? ' danger' : '');
        btn.textContent = label;
        btn.addEventListener('click', () => {
            hideContextMenu();
            onClick();
        });
        pmContextMenu.appendChild(btn);
    };
    const mkSep = () => {
        const sep = document.createElement('div');
        sep.className = 'pm-context-menu-sep';
        pmContextMenu.appendChild(sep);
    };

    mkItem('复制用户名', () => pmCopyText(rec.username, '用户名已复制'));
    mkItem('复制密码', async () => {
        try {
            const full = await window.go.main.App.GetPasswordRecord(id);
            await pmCopyText(full?.password || '', '密码已复制');
        } catch (err) {
            console.warn('获取密码失败:', err);
            window.nm?.show('获取密码失败', 'error');
        }
    });
    if (rec.url) {
        mkItem('复制链接', () => pmCopyText(rec.url, '链接已复制'));
    }
    mkSep();
    mkItem('查看详情', () => openPmDetail(id));
    mkItem('编辑', () => openPmEditDialog(id));
    mkSep();
    mkItem('删除', () => deletePmRecord(id), true);

    // 定位并夹紧到视口内
    pmContextMenu.style.display = 'block';
    const rect = pmContextMenu.getBoundingClientRect();
    let x = e.clientX;
    let y = e.clientY;
    if (x + rect.width > window.innerWidth - 8) x = window.innerWidth - rect.width - 8;
    if (y + rect.height > window.innerHeight - 8) y = window.innerHeight - rect.height - 8;
    pmContextMenu.style.left = `${Math.max(8, x)}px`;
    pmContextMenu.style.top = `${Math.max(8, y)}px`;
}

function hideContextMenu() {
    if (pmContextMenu) {
        pmContextMenu.querySelectorAll('.pressed').forEach(el => el.classList.remove('pressed'));
        pmContextMenu.style.display = 'none';
    }
}

function isContextMenuVisible() {
    return pmContextMenu && pmContextMenu.style.display !== 'none';
}

/* ==================== 添加 / 编辑对话框 ==================== */

/**
 * 将记录填充到编辑表单
 */
function fillPmEditForm(rec) {
    pmFieldName.value = rec.name || '';
    pmFieldUsername.value = rec.username || '';
    pmFieldPassword.value = rec.password || '';
    pmFieldUrl.value = rec.url || '';
    pmFieldNote.value = rec.note || '';
}

/**
 * 打开添加/编辑对话框
 * @param {number|null} id - null 表示新增，否则为待编辑记录 ID
 * @param {object} [presetRecord] - 已在内存中的完整记录（如从详情对话框进入），直接填充并跳过网络拉取
 */
async function openPmEditDialog(id, presetRecord) {
    editingId = id;
    pmEditTitle.textContent = id != null ? '编辑密码记录' : '添加密码记录';
    // 重置表单与错误态
    [pmFieldName, pmFieldUsername, pmFieldUrl].forEach((el) => { el.value = ''; el.classList.remove('invalid'); });
    pmFieldNote.value = '';
    pmFieldPassword.value = '';
    pmFieldPassword.classList.remove('invalid');
    resetEditPwdVisibility();

    if (presetRecord) {
        // 零延迟路径：详情等来源已持有完整数据，无需二次请求
        fillPmEditForm(presetRecord);
    } else if (id != null) {
        // 编辑前先拉取完整记录（含解码后的明文密码）
        try {
            const rec = await window.go.main.App.GetPasswordRecord(id);
            if (!rec) throw new Error('记录不存在');
            fillPmEditForm(rec);
        } catch (e) {
            console.warn('获取密码记录失败:', e);
            window.nm?.show('获取记录信息失败', 'error');
            return;
        }
    }

    // 记录表单快照（用于关闭时判断是否有未保存修改）
    pmEditSnapshot = {
        name: pmFieldName.value.trim(),
        username: pmFieldUsername.value.trim(),
        password: pmFieldPassword.value,
        url: pmFieldUrl.value.trim(),
        note: pmFieldNote.value,
    };

    pmEditOverlay.style.display = 'flex';
    setTimeout(() => pmFieldName.focus(), 50);
}

function closePmEditDialog(force) {
    if (force !== true && hasPmEditChanges()) {
        showPmConfirm('有未保存的修改，确定放弃并关闭吗？').then(ok => {
            if (ok) { pmEditOverlay.style.display = 'none'; editingId = null; }
        });
        return;
    }
    pmEditOverlay.style.display = 'none';
    editingId = null;
}

/**
 * 检测编辑对话框表单是否有未保存修改（对比当前值与打开时的快照）
 */
function hasPmEditChanges() {
    return pmFieldName.value.trim() !== pmEditSnapshot.name
        || pmFieldUsername.value.trim() !== pmEditSnapshot.username
        || pmFieldPassword.value !== pmEditSnapshot.password
        || pmFieldUrl.value.trim() !== pmEditSnapshot.url
        || pmFieldNote.value !== pmEditSnapshot.note;
}

/**
 * 密码管理页专用确认弹窗（复用全局 confirmDialog DOM，Promise<boolean>）
 */
function showPmConfirm(msg) {
    return new Promise(resolve => {
        const overlay = document.getElementById('confirmDialog');
        const msgEl = document.getElementById('confirmDialogMsg');
        const okBtn = document.getElementById('confirmOkBtn');
        const cancelBtn = document.getElementById('confirmCancelBtn');
        const thirdBtn = document.getElementById('confirmThirdBtn');
        if (!overlay || !okBtn || !cancelBtn) { resolve(true); return; }
        msgEl.textContent = msg;
        okBtn.textContent = '确定';
        cancelBtn.textContent = '取消';
        if (thirdBtn) thirdBtn.style.display = 'none';
        overlay.classList.add('visible');
        const cleanup = result => {
            overlay.classList.remove('visible');
            resolve(result);
        };
        okBtn.onclick = () => cleanup(true);
        cancelBtn.onclick = () => cleanup(false);
        overlay.onclick = e => { if (e.target === overlay) cleanup(false); };
    });
}

/**
 * 重置编辑对话框密码框为掩码态
 */
function resetEditPwdVisibility() {
    pmFieldPassword.type = 'password';
    pmPwdToggleBtn.textContent = '显示';
}

/**
 * 校验必填项，返回是否全部合法
 */
function validateRequiredFields() {
    let ok = true;
    let firstInvalid = null;
    const mark = (el) => {
        el.classList.add('invalid');
        triggerShake(el.closest('.pm-field') || el);
        if (!firstInvalid) firstInvalid = el;
    };
    [pmFieldName, pmFieldUsername, pmFieldPassword].forEach((el) => {
        el.classList.remove('invalid');
        if (!el.value.trim()) {
            mark(el);
            ok = false;
        }
    });
    if (!ok) window.nm?.show('请填写必填项', 'warning');
    if (firstInvalid) firstInvalid.focus();
    return ok;
}

/**
 * 保存（新增或更新）
 */
async function savePmRecord() {
    // 保存中直接忽略再次触发（防连按 Enter / 双击保存造成重复创建）
    if (pmSaveBtn.disabled) return;
    if (!validateRequiredFields()) return;
    const name = pmFieldName.value.trim();
    const username = pmFieldUsername.value.trim();
    const password = pmFieldPassword.value;
    const url = pmFieldUrl.value.trim();
    const note = pmFieldNote.value;

    const saveBtn = pmSaveBtn;
    saveBtn.disabled = true;
    // 关闭对话框会重置 editingId，先捕获用于刷新后的高亮定位
    let flashId = editingId != null ? Number(editingId) : null;
    try {
        const app = window.go.main.App;
        if (editingId != null) {
            await app.UpdatePasswordRecord(editingId, name, username, password, url, note);
            window.nm?.show('密码记录已更新', 'success');
        } else {
            // CreatePasswordRecord 返回含 ID 的完整记录，用于让列表顶部新行高亮
            const created = await app.CreatePasswordRecord(name, username, password, url, note);
            window.nm?.show('密码记录已添加', 'success');
            if (created && created.id != null) flashId = Number(created.id);
        }
        closePmEditDialog(true);
        loadPmRecords({ flashId });
    } catch (e) {
        console.warn('保存密码记录失败:', e);
        window.nm?.show('保存失败', 'error');
    } finally {
        saveBtn.disabled = false;
    }
}

/* ==================== 查看详情对话框 ==================== */

/**
 * 打开查看详情对话框（通过 GetPasswordRecord 获取完整数据，含明文密码）
 */
async function openPmDetail(id) {
    try {
        const rec = await window.go.main.App.GetPasswordRecord(id);
        if (!rec) throw new Error('记录不存在');
        detailRecord = rec;
        detailPwdVisible = false;

        pmDetailTitle.textContent = rec.name || '详情';
        pmDetailName.textContent = rec.name || '-';
        pmDetailUsername.textContent = rec.username || '-';

        setDetailPwdVisibility(false);

        // URL 行：为空显示 "-" 且隐藏操作按钮
        const hasUrl = !!(rec.url && rec.url.trim());
        pmDetailUrl.textContent = hasUrl ? rec.url : '-';
        pmCopyUrlBtn.style.display = hasUrl ? '' : 'none';
        pmOpenUrlBtn.style.display = hasUrl ? '' : 'none';

        pmDetailNote.textContent = rec.note || '-';
        pmDetailCreatedAt.textContent = formatDateTime(rec.created_at);
        pmDetailUpdatedAt.textContent = formatDateTime(rec.updated_at);

        pmDetailOverlay.style.display = 'flex';
    } catch (e) {
        console.warn('获取密码记录失败:', e);
        window.nm?.show('获取记录失败', 'error');
    }
}

function closePmDetailDialog() {
    pmDetailOverlay.style.display = 'none';
    detailRecord = null;
    detailPwdVisible = false;
}

/**
 * 切换详情对话框密码显隐
 */
function setDetailPwdVisibility(visible) {
    detailPwdVisible = visible;
    if (!detailRecord) return;
    if (visible) {
        pmDetailPassword.textContent = detailRecord.password || '-';
        pmDetailPassword.classList.remove('masked');
    } else {
        pmDetailPassword.textContent = PM_PWD_MASK;
        pmDetailPassword.classList.add('masked');
    }
    pmDetailPwdToggleBtn.textContent = visible ? '隐藏' : '显示';
}

/**
 * 从详情对话框跳转编辑：关闭详情后打开编辑对话框
 */
function editFromDetail() {
    const rec = detailRecord;
    if (!rec || rec.id == null) return;
    // 复用详情已在内存的数据，同一渲染帧内完成“关详情→开编辑”，消除异步拉取造成的空窗闪烁
    closePmDetailDialog();
    openPmEditDialog(Number(rec.id), rec);
}

/* ==================== 删除 ==================== */

/**
 * 删除单条记录（确认后软删除）
 */
async function deletePmRecord(id) {
    const rec = findInList(id);
    const name = rec?.name || '该记录';
    const ok = await window.showConfirmDialog(`确定删除“${name}”？`, '删除', '取消');
    if (!ok) return;
    try {
        await window.go.main.App.DeletePasswordRecord(id);
        window.nm?.show('已删除', 'success');
        if (pmDetailOverlay.style.display !== 'none') closePmDetailDialog();
        loadPmRecords();
    } catch (e) {
        console.warn('删除密码记录失败:', e);
        window.nm?.show('删除失败', 'error');
    }
}

/* ==================== 批量模式 ==================== */

/**
 * 进入/退出批量模式
 */
function togglePmBatchMode() {
    if (pmBatchMode) {
        exitPmBatchMode();
    } else {
        pmBatchMode = true;
        pmSelectedIds.clear();
        pmView.classList.add('batch-active');
        pmBatchBar.style.display = 'flex';
        pmBatchToggleBtn.classList.add('active');
        pmBatchToggleBtn.lastChild.textContent = ' 退出批量';
        syncBatchRowsStyle();
        updatePmSelectionUI();
    }
}

function exitPmBatchMode() {
    pmBatchMode = false;
    pmSelectedIds.clear();
    pmView.classList.remove('batch-active');
    pmBatchBar.style.display = 'none';
    pmBatchToggleBtn.classList.remove('active');
    pmBatchToggleBtn.lastChild.textContent = ' 批量操作';
    syncBatchRowsStyle();
    updatePmSelectionUI();
}

/**
 * 将内存中的选中集合同步到行样式（勾选图标 + 行高亮）
 */
function syncBatchRowsStyle() {
    pmListEl.querySelectorAll('.pm-item').forEach((item) => {
        const id = Number(item.dataset.id);
        const check = item.querySelector('.pm-check');
        const selected = pmSelectedIds.has(id);
        item.classList.toggle('selected', selected);
        if (check) check.classList.toggle('checked', selected);
    });
}

/**
 * 切换某行的选中状态
 */
function toggleItemChecked(item, check) {
    const id = Number(item.dataset.id);
    if (pmSelectedIds.has(id)) {
        pmSelectedIds.delete(id);
    } else {
        pmSelectedIds.add(id);
    }
    item.classList.toggle('selected', pmSelectedIds.has(id));
    check.classList.toggle('checked', pmSelectedIds.has(id));
    updatePmSelectionUI();
}

/**
 * 更新批量操作栏：计数、删除按钮可用性、全选复选框状态
 */
function updatePmSelectionUI() {
    const n = pmSelectedIds.size;
    pmSelectedCountEl.textContent = `已选 ${n} 项`;
    pmBatchDeleteBtn.disabled = n === 0;
    const total = pmRecords.length;
    pmSelectAllCb.checked = total > 0 && n >= total;
    pmSelectAllCb.indeterminate = n > 0 && n < total;
}

/**
 * 全选/取消全选（针对当前搜索结果）
 */
function handleSelectAllChange() {
    if (pmSelectAllCb.checked) {
        pmRecords.forEach((r) => pmSelectedIds.add(Number(r.id)));
    } else {
        pmSelectedIds.clear();
    }
    syncBatchRowsStyle();
    updatePmSelectionUI();
}

/**
 * 批量删除（确认后软删除所有选中项）
 */
async function batchDeleteSelected() {
    const ids = Array.from(pmSelectedIds);
    if (ids.length === 0) return;
    const ok = await window.showConfirmDialog(`确定删除 ${ids.length} 条记录？`, '删除', '取消');
    if (!ok) return;
    try {
        await window.go.main.App.BatchDeletePasswordRecords(ids);
        window.nm?.show(`已删除 ${ids.length} 条记录`, 'success');
        pmSelectedIds.clear();
        loadPmRecords();
    } catch (e) {
        console.warn('批量删除失败:', e);
        window.nm?.show('批量删除失败', 'error');
    }
}

/* ==================== 初始化 ==================== */

/**
 * 初始化密码管理视图 - 获取 DOM 引用并绑定事件
 */
export function initPasswordManager() {
    pmView = document.getElementById('viewPasswordManager');
    pmListEl = document.getElementById('pmList');
    pmEmptyEl = document.getElementById('pmEmpty');
    pmSearchInput = document.getElementById('pmSearchInput');
    pmSearchClearBtn = document.getElementById('pmSearchClearBtn');
    pmAddBtn = document.getElementById('pmAddBtn');
    pmBatchToggleBtn = document.getElementById('pmBatchToggleBtn');
    pmBatchBar = document.getElementById('pmBatchBar');
    pmSelectAllCb = document.getElementById('pmSelectAll');
    pmSelectedCountEl = document.getElementById('pmSelectedCount');
    pmBatchDeleteBtn = document.getElementById('pmBatchDeleteBtn');

    pmEditOverlay = document.getElementById('pmEditOverlay');
    pmEditTitle = document.getElementById('pmEditTitle');
    pmEditCloseBtn = document.getElementById('pmEditCloseBtn');
    pmCancelBtn = document.getElementById('pmCancelBtn');
    pmSaveBtn = document.getElementById('pmSaveBtn');
    pmFieldName = document.getElementById('pmFieldName');
    pmFieldUsername = document.getElementById('pmFieldUsername');
    pmFieldPassword = document.getElementById('pmFieldPassword');
    pmPwdToggleBtn = document.getElementById('pmPwdToggleBtn');
    pmFieldUrl = document.getElementById('pmFieldUrl');
    pmFieldNote = document.getElementById('pmFieldNote');

    pmDetailOverlay = document.getElementById('pmDetailOverlay');
    pmDetailTitle = document.getElementById('pmDetailTitle');
    pmDetailCloseBtn = document.getElementById('pmDetailCloseBtn');
    pmDetailName = document.getElementById('pmDetailName');
    pmDetailUsername = document.getElementById('pmDetailUsername');
    pmCopyUsernameBtn = document.getElementById('pmCopyUsernameBtn');
    pmDetailPassword = document.getElementById('pmDetailPassword');
    pmDetailPwdToggleBtn = document.getElementById('pmDetailPwdToggleBtn');
    pmCopyPasswordBtn = document.getElementById('pmCopyPasswordBtn');
    pmDetailUrlRow = document.getElementById('pmDetailUrlRow');
    pmDetailUrl = document.getElementById('pmDetailUrl');
    pmCopyUrlBtn = document.getElementById('pmCopyUrlBtn');
    pmOpenUrlBtn = document.getElementById('pmOpenUrlBtn');
    pmDetailNote = document.getElementById('pmDetailNote');
    pmDetailCreatedAt = document.getElementById('pmDetailCreatedAt');
    pmDetailUpdatedAt = document.getElementById('pmDetailUpdatedAt');
    pmDetailDeleteBtn = document.getElementById('pmDetailDeleteBtn');
    pmDetailEditBtn = document.getElementById('pmDetailEditBtn');

    pmContextMenu = document.getElementById('pmContextMenu');
    pmTooltip = document.getElementById('pmTooltip');

    if (!pmView || !pmListEl) return;

    // 返回首页
    document.getElementById('pmBackBtn')?.addEventListener('click', () => {
        if (window.switchView) window.switchView('grid');
    });

    // ---- 操作栏 ----
    pmAddBtn.addEventListener('click', () => openPmEditDialog(null));
    pmBatchToggleBtn.addEventListener('click', togglePmBatchMode);
    pmBatchDeleteBtn.addEventListener('click', batchDeleteSelected);
    pmSelectAllCb.addEventListener('change', handleSelectAllChange);

    // 搜索（防抖 250ms）
    pmSearchInput.addEventListener('input', () => {
        clearTimeout(pmSearchTimer);
        pmSearchTimer = setTimeout(() => {
            pmKeyword = pmSearchInput.value;
            pmSearchClearBtn.style.display = pmKeyword ? 'flex' : 'none';
            loadPmRecords();
        }, 250);
    });
    // Enter 立即搜索
    pmSearchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            clearTimeout(pmSearchTimer);
            pmKeyword = pmSearchInput.value;
            pmSearchClearBtn.style.display = pmKeyword ? 'flex' : 'none';
            loadPmRecords();
        }
    });
    pmSearchClearBtn.addEventListener('click', () => {
        pmSearchInput.value = '';
        pmKeyword = '';
        pmSearchClearBtn.style.display = 'none';
        pmSearchInput.focus();
        loadPmRecords();
    });

    // ---- 字段实时长度校验（备注 text 类型不限长，不绑定）----
    bindLengthGuard(pmFieldName, PM_MAX_LENGTH.name);
    bindLengthGuard(pmFieldUsername, PM_MAX_LENGTH.username);
    bindLengthGuard(pmFieldPassword, PM_MAX_LENGTH.password);
    bindLengthGuard(pmFieldUrl, PM_MAX_LENGTH.url);

    // 输入即清除必填错误标记
    [pmFieldName, pmFieldUsername, pmFieldPassword].forEach((el) => {
        el.addEventListener('input', () => el.classList.remove('invalid'));
    });

    // ---- 编辑对话框 ----
    pmPwdToggleBtn.addEventListener('click', () => {
        const showNow = pmFieldPassword.type === 'password';
        pmFieldPassword.type = showNow ? 'text' : 'password';
        pmPwdToggleBtn.textContent = showNow ? '隐藏' : '显示';
    });
    pmEditCloseBtn.addEventListener('click', closePmEditDialog);
    pmCancelBtn.addEventListener('click', closePmEditDialog);
    pmSaveBtn.addEventListener('click', savePmRecord);
    pmEditOverlay.addEventListener('mousedown', (e) => {
        if (e.target === pmEditOverlay) closePmEditDialog();
    });
    // 表单内 Enter 直接保存（Shift+Enter 换行仅在 textarea 生效）
    pmEditOverlay.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && e.target.tagName !== 'TEXTAREA') {
            e.preventDefault();
            savePmRecord();
        }
    });

    // ---- 查看详情对话框 ----
    pmDetailCloseBtn.addEventListener('click', closePmDetailDialog);
    pmDetailPwdToggleBtn.addEventListener('click', () => setDetailPwdVisibility(!detailPwdVisible));
    pmCopyUsernameBtn.addEventListener('click', () => pmCopyText(detailRecord?.username, '用户名已复制'));
    pmCopyPasswordBtn.addEventListener('click', () => pmCopyText(detailRecord?.password, '密码已复制'));
    pmCopyUrlBtn.addEventListener('click', () => pmCopyText(detailRecord?.url, '链接已复制'));
    pmOpenUrlBtn.addEventListener('click', () => {
        const url = (detailRecord?.url || '').trim();
        if (!url) return;
        try {
            window.runtime.BrowserOpenURL(url);
        } catch (e) {
            console.warn('打开链接失败:', e);
            window.nm?.show('打开链接失败', 'error');
        }
    });
    pmDetailDeleteBtn.addEventListener('click', () => {
        if (detailRecord) deletePmRecord(Number(detailRecord.id));
    });
    pmDetailEditBtn.addEventListener('click', editFromDetail);
    pmDetailOverlay.addEventListener('mousedown', (e) => {
        if (e.target === pmDetailOverlay) closePmDetailDialog();
    });

    // ---- 右键菜单关闭：点击外部（Escape 由 main.js 全局链经 window.pmHandleEscape 统一处理）----
    document.addEventListener('mousedown', (e) => {
        if (isContextMenuVisible() && !pmContextMenu.contains(e.target)) hideContextMenu();
    });
    // 右键菜单项按下回弹反馈：mousedown 缩小，鼠标移出清理（与笔记首页/笔记本侧栏右键菜单一致）
    pmContextMenu.addEventListener('mousedown', (e) => {
        if (e.button !== 0) return;
        const item = e.target.closest('.pm-context-menu-item');
        if (item) item.classList.add('pressed');
    });
    pmContextMenu.addEventListener('mouseleave', () => {
        pmContextMenu.querySelectorAll('.pressed').forEach(el => el.classList.remove('pressed'));
    });

    // 初始加载（播放入场动画，之后一切增删改/搜索刷新均静默）
    loadPmRecords({ playEnter: true });
}

/**
 * 刷新视图数据（main.js switchView 进入密码管理页时调用）
 */
window.refreshPasswordManagerView = function () {
    // 进入页面时若仍处于批量模式（异常残留），自动退出
    if (pmBatchMode) exitPmBatchMode();
    loadPmRecords({ playEnter: true });
};

/**
 * 退出密码管理页（main.js 切换视图离开本页时调用）
 */
window.exitPasswordManagerView = function () {
    if (pmBatchMode) exitPmBatchMode();
    hideContextMenu();
};

/**
 * 统一 ESC 出口：由 main.js 全局 Escape 分支调用
 * 按 右键菜单 → 编辑对话框 → 详情对话框 的层级只关闭最上层一个
 * @returns {boolean} 是否关闭了某个弹层（true = 本次 ESC 已被消费）
 */
window.pmHandleEscape = function () {
    // 应用级确认框打开时交给其自身处理，不消费本次 ESC
    if (document.getElementById('confirmDialog')?.classList.contains('visible')) return false;
    if (isContextMenuVisible()) { hideContextMenu(); return true; }
    if (pmEditOverlay && pmEditOverlay.style.display !== 'none') { closePmEditDialog(); return true; }
    if (pmDetailOverlay && pmDetailOverlay.style.display !== 'none') { closePmDetailDialog(); return true; }
    return false;
};
