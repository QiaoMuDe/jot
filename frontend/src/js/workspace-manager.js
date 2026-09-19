/* ==========================================================================
   工作区管理器（AI Agent 沙箱文件空间）
   功能：浏览 / 上传文件 / 上传目录 / 下载到桌面 / 删除 工作区文件
   交互：目录行左 1/3 点击=选中（联动其下全部内容）、右 2/3 点击=展开/折叠；
         文件行整行点击=选中；checkbox 均原生可用，勾选目录联动子级
   竞态防护：代际序号 workspaceLoadSeq（范式参考 pmLoadSeq / streamGen）
   通知：window.showNotification；二次确认：window.showConfirmDialog（main.js 收口）
   ESC 关闭由 main.js 的 handleKeyboardNavigation 统一收口，本模块不注册 ESC 监听
   ========================================================================== */

/** 列表加载代际序号：请求前自增并捕获，响应返回时序号不匹配则丢弃（防慢响应乱序覆盖新结果） */
let workspaceLoadSeq = 0;
/** 当前选中的 relPath 集合（勾选目录时联动纳入其下全部内容） */
let workspaceSelected = new Set();
/** 展开状态的目录 relPath 集合（懒展开，单击目录行切换） */
let workspaceExpanded = new Set();
/** 最近一次加载的文件树数据：[{Name, RelPath, IsDir, Size, ModTime, Children:[...]}] */
let workspaceTreeData = null;
/** 是否有上传/下载/删除操作进行中（busy 期间禁用全部操作按钮） */
let workspaceBusy = false;
/** 删除确认框打开期间置位，防止重复触发确认 */
let workspaceDeleting = false;
/** 懒绑定标记：弹窗内部事件只绑定一次（与 vectorIndexModal 范式一致，常驻监听无残留） */
let workspaceBound = false;
/** 新建文件夹输入条开关状态：true=已展开，false=已关闭（toggle 用，避免依赖 isConnected 竞态） */
let workspaceNewDirOpen = false;
/** 失焦自动关闭定时器句柄：可被手动关闭/重开时取消，防止残留定时器误关重开后的输入条 */
let workspaceNewDirAutoCloseTimer = null;
/** 拖拽上传目标：null=根目录；非空=最后悬停的目录 RelPath（悬停亮起即目标） */
let workspaceDropTargetRel = null;
/** 面板内拖拽进出计数（防子元素 dragleave 误触发隐藏遮罩，参考 _aiDragCounter 范式） */
let workspaceDropCount = 0;

/** 上传类按钮 id（busy 时遍历禁用；当前仅余单个「上传」提示按钮） */
const WS_UPLOAD_LABELS = ['workspaceUploadBtn'];

/** 内联 SVG 图标（lucide 风格，禁止 emoji 当图标） */
const WS_ICONS = {
    folder: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>',
    file: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
};

/** 获取弹窗内部元素 */
function ws$(id) { return document.getElementById(id); }

/** HTML 转义（路径/名称可能含特殊字符，防注入） */
function wsEsc(text) {
    if (text === null || text === undefined) return '';
    return String(text)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

/** 字节数格式化：B / KB / MB / GB */
function formatWorkspaceSize(bytes) {
    if (bytes === null || bytes === undefined) return '';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

/** unix 秒 → 「YYYY-MM-DD HH:mm:ss」 */
function formatWorkspaceTime(unixSec) {
    if (!unixSec) return '';
    const d = new Date(unixSec * 1000);
    const pad = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

/* ============ 文件树渲染 ============ */

/**
 * 渲染文件树（一次性渲染整棵树；目录按 workspaceExpanded 懒展开子级）
 * @param {Array} tree - ListWorkspaceFiles 返回的节点数组
 */
function renderWorkspaceTree(tree) {
    const treeEl = ws$('workspaceTree');
    const emptyEl = ws$('workspaceEmpty');
    if (!treeEl || !emptyEl) return;
    const hasItems = Array.isArray(tree) && tree.length > 0;
    // 空状态与树容器互斥显示
    treeEl.style.display = hasItems ? '' : 'none';
    emptyEl.style.display = hasItems ? 'none' : '';
    treeEl.innerHTML = hasItems ? buildWorkspaceTreeHTML(tree, 0) : '';
}

/**
 * 递归构建树 HTML（depth 用于行缩进；仅展开的目录生成子列表）
 * @param {Array} nodes - 节点数组
 * @param {number} depth - 当前层级（0 起）
 * @returns {string} HTML 片段
 */
function buildWorkspaceTreeHTML(nodes, depth) {
    let html = '<ul class="workspace-tree-list">';
    for (const node of nodes) {
        const rel = node.RelPath || '';
        const isDir = !!node.IsDir;
        const expanded = workspaceExpanded.has(rel);
        html += `<li class="workspace-node${isDir ? ' is-dir' : ''}" data-rel="${wsEsc(rel)}" data-is-dir="${isDir ? '1' : ''}" style="--depth:${depth}">`;
        html += '<div class="workspace-row">';
        // 无箭头无占位：checkbox 即为行首，条目整体左移；展开/折叠通过单击目录行触发
        html += `<label class="workspace-check" title="${isDir ? '选中目录' : '选中文件'}"><input type="checkbox" data-rel="${wsEsc(rel)}"></label>`;
        html += `<span class="workspace-icon">${isDir ? WS_ICONS.folder : WS_ICONS.file}</span>`;
        html += `<span class="workspace-name" title="${wsEsc(node.Name || '')}">${wsEsc(node.Name || '')}</span>`;
        html += `<span class="workspace-size">${isDir ? '' : formatWorkspaceSize(node.Size)}</span>`;
        html += `<span class="workspace-mtime">${isDir ? '' : formatWorkspaceTime(node.ModTime)}</span>`;
        html += '</div>';
        // 目录已展开且有子级时递归渲染子列表（懒展开：折叠时不生成子级 DOM）
        if (isDir && expanded && Array.isArray(node.Children) && node.Children.length > 0) {
            html += buildWorkspaceTreeHTML(node.Children, depth + 1);
        }
        html += '</li>';
    }
    html += '</ul>';
    return html;
}

/**
 * 渲染后恢复勾选：将 workspaceSelected 中仍存在的项重新勾选，并清理失效项、更新计数与按钮态
 */
function applyWorkspaceSelection() {
    const treeEl = ws$('workspaceTree');
    if (treeEl) {
        treeEl.querySelectorAll('input[type="checkbox"]').forEach(input => {
            input.checked = workspaceSelected.has(input.dataset.rel);
        });
    }
    // 清理已不存在的失效选择（如删除后），基于完整树数据而非 DOM（折叠子级不影响）
    pruneWorkspaceSelection();
    updateWorkspaceToolbar();
}

/** 从完整树数据收集全部有效 relPath，清除选中集合中已失效的项 */
function pruneWorkspaceSelection() {
    const valid = new Set();
    collectWorkspaceRels(workspaceTreeData, valid);
    for (const rel of [...workspaceSelected]) {
        if (!valid.has(rel)) workspaceSelected.delete(rel);
    }
}

/** 递归收集树中全部 relPath 到 out 集合 */
function collectWorkspaceRels(nodes, out) {
    if (!Array.isArray(nodes)) return;
    for (const node of nodes) {
        if (node.RelPath) out.add(node.RelPath);
        if (Array.isArray(node.Children) && node.Children.length > 0) {
            collectWorkspaceRels(node.Children, out);
        }
    }
}

/** 收集工作区全部 relPath（含折叠目录内的文件），供全选使用 */
function collectAllRels() {
    const all = new Set();
    collectWorkspaceRels(workspaceTreeData, all);
    return [...all];
}

/**
 * 全选/全不选切换：点击时在全选（含折叠目录）与清空之间切换
 * 按钮形态 + click 事件，不依赖 checkbox change/label 转发，disabled 时直接忽略
 */
function onWorkspaceSelectAllClick() {
    const selectAll = ws$('workspaceSelectAll');
    if (!selectAll || selectAll.disabled) return;
    if (selectAll.classList.contains('is-checked')) workspaceSelected = new Set();
    else workspaceSelected = new Set(collectAllRels());
    applyWorkspaceSelection();
}

/** 判断选中集合中是否包含目录（删除时决定递归参数与文案） */
function selectedContainsDir() {
    const check = (nodes) => {
        if (!Array.isArray(nodes)) return false;
        for (const node of nodes) {
            if (workspaceSelected.has(node.RelPath) && node.IsDir) return true;
            if (check(node.Children)) return true;
        }
        return false;
    };
    return check(workspaceTreeData);
}

/* ============ 交互（事件委托） ============ */

/** 切换单行选中：翻转该行 checkbox 并应用（含目录联动） */
function toggleRowSelect(li) {
    const input = li.querySelector('input[type="checkbox"]');
    if (!input) return;
    input.checked = !input.checked;
    applyRowCheckState(input);
}

/**
 * 树容器 mousemove 委托：目录行按悬停位置切换光标（左 1/3=选中 pointer、右 2/3=展开 alias）
 * 文件行整行为 pointer（CSS 静态指定）；离开行时清理残留类
 */
function onWorkspaceTreeMove(e) {
    const row = e.target.closest('.workspace-row');
    if (!row) return;
    const li = row.closest('.workspace-node');
    if (!li || li.dataset.isDir !== '1') {
        row.classList.remove('ws-cursor-select', 'ws-cursor-expand');
        return;
    }
    const rect = row.getBoundingClientRect();
    const x = e.clientX - rect.left;
    row.classList.toggle('ws-cursor-select', x < rect.width * 0.33);
    row.classList.toggle('ws-cursor-expand', x >= rect.width * 0.33);
}

/**
 * 树容器点击委托（区域分流，无单击/双击判定）：
 * - 目录行：左 1/3（含 checkbox 区）点击=选中（联动子级）；右 2/3 点击=展开/折叠
 * - 文件行：整行点击=选中
 * checkbox 区域的点击由原生 change 处理，此处跳过
 */
function onWorkspaceTreeClick(e) {
    const row = e.target.closest('.workspace-row');
    if (!row) return;
    if (e.target.closest('.workspace-check')) return;
    const li = row.closest('.workspace-node');
    if (!li) return;
    if (li.dataset.isDir === '1') {
        const rect = row.getBoundingClientRect();
        const x = e.clientX - rect.left;
        if (x < rect.width * 0.33) toggleRowSelect(li);
        else toggleWorkspaceDir(li.dataset.rel);
    } else {
        toggleRowSelect(li);
    }
}

/** 切换目录展开/折叠并重渲染（保留勾选） */
function toggleWorkspaceDir(rel) {
    if (!rel) return;
    if (workspaceExpanded.has(rel)) workspaceExpanded.delete(rel);
    else workspaceExpanded.add(rel);
    if (!workspaceTreeData) return;
    renderWorkspaceTree(workspaceTreeData);
    applyWorkspaceSelection();
}

/** 在完整树数据中按 relPath 查找节点（含折叠子级） */
function getTreeNodeByRel(relPath) {
    let found = null;
    const walk = (nodes) => {
        if (!Array.isArray(nodes) || found) return;
        for (const node of nodes) {
            if (node.RelPath === relPath) { found = node; return; }
            walk(node.Children);
        }
    };
    walk(workspaceTreeData);
    return found;
}

/**
 * 按复选框状态更新选中集合：目录勾选/取消时递归联动其下全部内容（自身 + 全部后代）
 * 之后统一 applyWorkspaceSelection 同步 DOM 勾选、清理失效项并更新计数/按钮态
 */
function applyRowCheckState(input) {
    const rel = input.dataset.rel;
    if (!rel) return;
    const node = getTreeNodeByRel(rel);
    const rels = new Set([rel]);
    if (node && node.IsDir) collectWorkspaceRels(node.Children, rels);
    if (input.checked) rels.forEach(r => workspaceSelected.add(r));
    else rels.forEach(r => workspaceSelected.delete(r));
    applyWorkspaceSelection();
}

/**
 * 剪枝：去掉集合中已被选中目录覆盖的子孙条目（避免下载/删除时目录与其子级重复处理）
 */
function pruneCoveredRels(rels) {
    const set = new Set(rels);
    return rels.filter(rel => {
        const parts = rel.split('/');
        for (let i = 1; i < parts.length; i++) {
            if (set.has(parts.slice(0, i).join('/'))) return false;
        }
        return true;
    });
}

/**
 * 树容器勾选委托：目录联动子级选中（选中目录=选中目录及其下全部内容，取消同理）
 */
function onWorkspaceTreeChange(e) {
    const input = e.target;
    if (input.type !== 'checkbox' || !input.dataset.rel) return;
    applyRowCheckState(input);
}

/**
 * 更新「已选 N 项」计数、全选状态与下载/删除按钮可用态（未选中或操作进行中时禁用）
 */
function updateWorkspaceToolbar() {
    const count = workspaceSelected.size;
    const countEl = ws$('workspaceCount');
    if (countEl) countEl.textContent = `已选 ${count} 项`;
    // 全选按钮：全部条目均已选中时勾选态并显示「取消全选」；空树或操作进行中禁用
    const selectAll = ws$('workspaceSelectAll');
    if (selectAll) {
        const total = collectAllRels().length;
        const allChecked = total > 0 && count === total;
        selectAll.classList.toggle('is-checked', allChecked);
        selectAll.disabled = total === 0 || workspaceBusy;
        const selectAllText = ws$('workspaceSelectAllText');
        if (selectAllText) selectAllText.textContent = allChecked ? '取消全选' : '全选';
    }
    const dlBtn = ws$('workspaceDownloadBtn');
    const delBtn = ws$('workspaceDeleteBtn');
    const disabled = count === 0 || workspaceBusy;
    if (dlBtn) {
        dlBtn.disabled = disabled;
        dlBtn.title = count === 0 ? '请先选择内容' : '下载选中内容到桌面';
    }
    if (delBtn) {
        delBtn.disabled = disabled;
        delBtn.title = count === 0 ? '请先选择内容' : '删除选中内容（不可恢复）';
    }
}

/**
 * 设置操作进行中状态：busy 时全部操作按钮禁用，触发上传的按钮显示「上传中…」，刷新图标旋转
 * @param {boolean} busy - 是否进行中
 * @param {string|null} activeId - 触发上传操作的按钮 id（仅该按钮显示 loading 文案）
 */
function setWorkspaceBusy(busy, activeId) {
    workspaceBusy = busy;
    // 上传类按钮统一禁用；触发者用 title 提示「上传中…」+ icon 脉冲（不写回文本）
    WS_UPLOAD_LABELS.forEach(id => {
        const btn = ws$(id);
        if (!btn) return;
        btn.disabled = busy;
        if (busy && id === activeId) {
            if (btn.dataset.prevTitle === undefined) btn.dataset.prevTitle = btn.title || '';
            btn.title = '上传中…';
            btn.classList.add('is-loading');
        } else {
            // 恢复：仅当此前设置过才还原 + 清理缓存；未设置过（非上传触发的 busy）
            // 则保持按钮原有 title 不变，避免外部调用清空 HTML 预设的悬停提示
            btn.classList.remove('is-loading');
            if (btn.dataset.prevTitle !== undefined) {
                btn.title = btn.dataset.prevTitle;
                delete btn.dataset.prevTitle;
            }
        }
    });
    const refreshBtn = ws$('workspaceRefreshBtn');
    if (refreshBtn) {
        refreshBtn.disabled = busy;
        refreshBtn.classList.toggle('is-loading', busy);
    }
    updateWorkspaceToolbar();
}

/* ============ 数据加载 ============ */

/**
 * 加载工作区文件树（代际序号竞态防护）
 * 失败时通知；成功后渲染并恢复/清理选择
 */
async function loadWorkspaceTree() {
    const seq = ++workspaceLoadSeq;
    const modal = ws$('workspaceModal');
    // 弹窗已关闭则直接丢弃（避免为隐藏弹窗渲染）
    if (modal && modal.style.display === 'none') return;
    let tree = [];
    try {
        tree = await window.go.main.App.ListWorkspaceFiles();
    } catch (err) {
        if (seq !== workspaceLoadSeq) return;
        window.showNotification?.(`加载工作区文件失败：${err?.message || err}`, 'error');
        return;
    }
    // 过期响应（期间有新的加载请求）或弹窗已关闭 → 丢弃
    if (seq !== workspaceLoadSeq) return;
    if (modal && modal.style.display === 'none') return;
    workspaceTreeData = Array.isArray(tree) ? tree : [];
    renderWorkspaceTree(workspaceTreeData);
    applyWorkspaceSelection();
}

/**
 * 手动刷新：重新加载文件树（选择保留，渲染后恢复勾选）
 */
async function refreshWorkspace() {
    if (workspaceBusy) return;
    const refreshBtn = ws$('workspaceRefreshBtn');
    if (refreshBtn) {
        refreshBtn.classList.add('is-loading');
        refreshBtn.disabled = true;
    }
    try {
        await loadWorkspaceTree();
    } finally {
        if (refreshBtn) {
            refreshBtn.classList.remove('is-loading');
            refreshBtn.disabled = false;
        }
    }
}

/* ============ 上传 / 下载 / 删除 ============ */

/**
 * 上传文件到工作区：调 UploadFilesToWorkspace（多选对话框，取消返回空数组）
 * 完成后通知结果并刷新（刷新保留选择）
 */
async function uploadWorkspaceFiles() {
    if (workspaceBusy) return;
    setWorkspaceBusy(true, 'workspaceUploadBtn');
    try {
        let results = [];
        try {
            results = await window.go.main.App.UploadFilesToWorkspace();
        } catch (err) {
            window.showNotification?.(`上传文件失败：${err?.message || err}`, 'error');
            return;
        }
        if (!Array.isArray(results)) results = [];
        // 取消对话框 → 空数组，不通知不刷新
        if (results.length === 0) return;
        const okCount = results.filter(r => !r.Error).length;
        const errors = results.filter(r => r.Error);
        if (okCount > 0) window.showNotification?.(`成功上传 ${okCount} 个文件到工作区`, 'success');
        errors.forEach(r => window.showNotification?.(`${r.Name || '文件'} 上传失败：${r.Error}`, 'error'));
        await loadWorkspaceTree();
    } finally {
        setWorkspaceBusy(false, null);
    }
}

/**
 * 下载选中内容到桌面：调 DownloadWorkspaceFiles(relPaths)
 * 完成后通知结果并刷新（刷新保留选择）
 */
async function downloadWorkspaceFiles() {
    const relPaths = pruneCoveredRels([...workspaceSelected]);
    if (relPaths.length === 0 || workspaceBusy) return;
    setWorkspaceBusy(true, null);
    try {
        let results = [];
        try {
            results = await window.go.main.App.DownloadWorkspaceFiles(relPaths);
        } catch (err) {
            window.showNotification?.(`下载失败：${err?.message || err}`, 'error');
            return;
        }
        if (!Array.isArray(results)) results = [];
        const okCount = results.filter(r => !r.Error).length;
        const errors = results.filter(r => r.Error);
        if (okCount > 0) window.showNotification?.(`已下载 ${okCount} 项到桌面`, 'success');
        errors.forEach(r => window.showNotification?.(`${r.Name || '文件'} 下载失败：${r.Error}`, 'error'));
        await loadWorkspaceTree();
    } finally {
        setWorkspaceBusy(false, null);
    }
}

/**
 * 删除选中内容：二次确认（含目录时提示递归）→ DeleteWorkspaceFiles(relPaths, recursive)
 * 成功后从选中集合移除已删项并刷新
 */
async function deleteWorkspaceFiles() {
    const relPaths = pruneCoveredRels([...workspaceSelected]);
    if (relPaths.length === 0 || workspaceBusy || workspaceDeleting) return;
    workspaceDeleting = true;
    try {
        const hasDir = selectedContainsDir();
        let msg = `确定删除选中的 ${relPaths.length} 项吗？删除后不可恢复。`;
        if (hasDir) msg += '将递归删除目录及其内容。';
        const confirmed = await window.showConfirmDialog(msg, '删除');
        if (!confirmed) return;
        setWorkspaceBusy(true, null);
        try {
            let results = [];
            try {
                results = await window.go.main.App.DeleteWorkspaceFiles(relPaths, hasDir);
            } catch (err) {
                window.showNotification?.(`删除失败：${err?.message || err}`, 'error');
                return;
            }
            if (!Array.isArray(results)) results = [];
            const okCount = results.filter(r => !r.Error).length;
            const errors = results.filter(r => r.Error);
            if (okCount > 0) window.showNotification?.(`成功删除 ${okCount} 项`, 'success');
            errors.forEach(r => window.showNotification?.(`${r.Name || '文件'} 删除失败：${r.Error}`, 'error'));
            // 成功删除的项从选中集合移除（Target 或 Name 与 relPath 匹配）
            results.filter(r => !r.Error).forEach(r => {
                const t = r.Target || r.Name;
                if (t && workspaceSelected.has(t)) workspaceSelected.delete(t);
            });
            // 刷新后 applyWorkspaceSelection 会清理其余失效项
            await loadWorkspaceTree();
        } finally {
            setWorkspaceBusy(false, null);
        }
    } finally {
        workspaceDeleting = false;
    }
}

/* ============ 拖拽上传（方案 A：悬停目录行即目标，无遮罩不遮挡树） ============ */

/** 清理全部目录行拖拽高亮 */
function clearWorkspaceDropHover() {
    const treeEl = ws$('workspaceTree');
    if (treeEl) treeEl.querySelectorAll('.workspace-row.ws-drop-target').forEach(el => el.classList.remove('ws-drop-target'));
}

/** 解析拖拽当前悬停的目录 RelPath（命中目录行返回 rel，否则 null 表示根目录） */
function resolveWorkspaceDropTarget(clientX, clientY) {
    const el = document.elementFromPoint(clientX, clientY);
    const node = el?.closest('.workspace-node.is-dir');
    return node ? (node.dataset.rel || null) : null;
}

/** 更新拖拽目标：仅在 rel 变化时更新行级高亮（dragover 高频触发，防抖避免无谓 DOM 操作） */
function updateWorkspaceDropTarget(rel) {
    const treeEl = ws$('workspaceTree');
    // 拖拽中始终给树容器加高亮边框（根目录目标时同样可见，不遮挡内容）
    treeEl?.classList.add('ws-drag-active');
    if (rel === workspaceDropTargetRel) return;
    workspaceDropTargetRel = rel;
    clearWorkspaceDropHover();
    if (rel) {
        const row = treeEl?.querySelector(`.workspace-node.is-dir[data-rel="${CSS.escape(rel)}"] .workspace-row`);
        if (row) row.classList.add('ws-drop-target');
    }
}

/** 面板 dragenter：仅文件拖拽进入拖拽态，并记录悬停目标 */
function onWorkspaceDragEnter(e) {
    if (!e.dataTransfer?.types?.includes('Files')) return;
    workspaceDropCount++;
    updateWorkspaceDropTarget(resolveWorkspaceDropTarget(e.clientX, e.clientY));
}

/** 面板 dragover：须 preventDefault 否则 drop 不触发；更新悬停目标（防抖） */
function onWorkspaceDragOver(e) {
    e.preventDefault();
    if (!e.dataTransfer?.types?.includes('Files')) return;
    updateWorkspaceDropTarget(resolveWorkspaceDropTarget(e.clientX, e.clientY));
}

/** 面板 dragleave：计数递减，归零退出拖拽态并清理高亮 */
function onWorkspaceDragLeave(e) {
    if (!e.dataTransfer?.types?.includes('Files')) return;
    if (e.relatedTarget?.closest?.('#workspaceModal')) return; // 面板内元素间移动不退出
    workspaceDropCount--;
    if (workspaceDropCount <= 0) {
        workspaceDropCount = 0;
        resetWorkspaceDropState();
    }
}

/** 面板 drop（真实文件由 main.js OnFileDrop 处理）：仅复位视觉，保留悬停目标供 handleWorkspaceDrop 消费（规避 OnFileDrop 坐标在 DPI 缩放下的偏移） */
function onWorkspaceDrop(e) {
    e.preventDefault();
    workspaceDropCount = 0;
    clearWorkspaceDropHover();
    ws$('workspaceTree')?.classList.remove('ws-drag-active');
}

/** 将目标 rel 的祖先链全部加入展开集（a/b/c → 展开 a、a/b、a/b/c），上传后用户能直接看到 */
function expandWorkspaceTargetChain(rel) {
    if (!rel) return;
    workspaceExpanded.add(rel);
    const parts = rel.split('/');
    for (let i = 1; i < parts.length; i++) {
        workspaceExpanded.add(parts.slice(0, i).join('/'));
    }
}

/**
 * 面板拖拽上传入口（由 main.js OnFileDrop 回调调用）：
 * 优先用 dragover 最后一刻的悬停目标 workspaceDropTargetRel（所见即目标，DOM 本身
 * 用的是浏览器视口 CSS 像素，不受 DPI 缩放影响）；仅当其为空时才用 OnFileDrop 的释放
 * 坐标实时解析兜底（规避 Windows 高 DPI 下物理像素偏移）。消费后即清空，避免残留。
 * 完成后通知结果、展开目标目录祖先链并刷新（刷新保留选择）。
 */
window.handleWorkspaceDrop = async function (paths, clientX, clientY) {
    workspaceDropCount = 0;
    // 优先悬停目标；其为空时坐标兜底；均无则上传到根目录
    const targetRel = workspaceDropTargetRel ?? resolveWorkspaceDropTarget(clientX, clientY) ?? '';
    workspaceDropTargetRel = null;
    clearWorkspaceDropHover();
    ws$('workspaceTree')?.classList.remove('ws-drag-active');
    if (!Array.isArray(paths) || paths.length === 0) return;
    if (workspaceBusy) {
        window.showNotification?.('当前有操作进行中，请稍候再拖拽上传', 'info');
        return;
    }
    setWorkspaceBusy(true, null);
    try {
        let results = [];
        try {
            results = await window.go.main.App.UploadPathsToWorkspace(paths, targetRel);
        } catch (err) {
            window.showNotification?.(`拖拽上传失败：${err?.message || err}`, 'error');
            return;
        }
        if (!Array.isArray(results)) results = [];
        const okCount = results.filter(r => !r.Error).length;
        const errors = results.filter(r => r.Error);
        if (okCount > 0) {
            window.showNotification?.(`成功上传 ${okCount} 项${targetRel ? `到「${targetRel}」` : '到工作区'}`, 'success');
        }
        errors.forEach(r => window.showNotification?.(`${r.Name || '文件'} 上传失败：${r.Error}`, 'error'));
        // 目标目录祖先链全部展开，刷新后用户能直接看到新上传内容
        expandWorkspaceTargetChain(targetRel);
        await loadWorkspaceTree();
    } finally {
        setWorkspaceBusy(false, null);
    }
};

/* ============ 懒绑定事件 ============ */

/** 解析「新建文件夹」的父目录：单个选中目录→其内，其余情况→根目录 */
function resolveWorkspaceCreateParent() {
    if (workspaceSelected.size === 1) {
        const only = [...workspaceSelected][0];
        const node = ws$('workspaceTree')?.querySelector(`.workspace-node.is-dir[data-rel="${CSS.escape(only)}"]`);
        if (node) return only;
    }
    return '';
}

/**
 * 「新建文件夹」按钮点击切换：首次点击展开内联输入条，再次点击关闭。
 * 基于显式状态 workspaceNewDirOpen 判定，关闭/展开均幂等，避免 isConnected 竞态导致的闪烁。
 */
function toggleWorkspaceNewDirInput() {
    if (workspaceNewDirOpen) { closeWorkspaceNewDirInput(); return; }
    showWorkspaceNewDirInput();
}

/** 关闭新建文件夹输入条：移除 DOM 并复位状态（幂等，可重复调用） */
function closeWorkspaceNewDirInput() {
    workspaceNewDirOpen = false;
    // 取消挂起的失焦自动关闭定时器：防止在「先关后重开」（如创建失败重开输入条）场景下
    // 残留定时器把重开后的输入条再次关闭
    if (workspaceNewDirAutoCloseTimer) {
        clearTimeout(workspaceNewDirAutoCloseTimer);
        workspaceNewDirAutoCloseTimer = null;
    }
    ws$('workspaceNewDirBar')?.remove();
}

/**
 * 显示「新建文件夹」内联输入条：Enter 确认、Esc 取消；失焦取消但允许点击条内确定/取消
 * （focusout 判定 relatedTarget 是否仍在条内，避免 blur→click 竞态误触取消）。
 * 确认或回车创建后即关闭本输入条。
 */
function showWorkspaceNewDirInput() {
    const parentRel = resolveWorkspaceCreateParent();
    const modal = ws$('workspaceModal');
    let bar = ws$('workspaceNewDirBar');
    if (!bar) {
        bar = document.createElement('div');
        bar.className = 'workspace-newdir-bar';
        bar.id = 'workspaceNewDirBar';
        bar.innerHTML = '<span class="workspace-newdir-tag">新建文件夹</span>'
            + '<input class="workspace-newdir-input" type="text" maxlength="120" placeholder="输入文件夹名">'
            + '<button class="btn btn-save btn-sm workspace-newdir-ok">确定</button>'
            + '<button class="btn btn-sm workspace-newdir-cancel">取消</button>';
        modal?.querySelector('.workspace-toolbar')?.insertAdjacentElement('afterend', bar);
        const input = bar.querySelector('.workspace-newdir-input');
        const ok = bar.querySelector('.workspace-newdir-ok');
        const cancel = bar.querySelector('.workspace-newdir-cancel');
        const finish = (commit) => {
            if (commit) {
                const name = input.value.trim();
                if (name) {
                    closeWorkspaceNewDirInput(); // 确认/回车创建后立即关闭输入条（并取消挂起的自动关闭）
                    // 创建失败不重开输入条（错误通知已在 createWorkspaceDirectory 内部发出）
                    createWorkspaceDirectory(parentRel, name).catch(() => {});
                } else {
                    input.focus();
                }
                return;
            }
            closeWorkspaceNewDirInput();
        };
        input.addEventListener('keydown', (e) => { if (e.key === 'Enter') finish(true); else if (e.key === 'Escape') finish(false); });
        ok.addEventListener('click', () => finish(true));
        cancel.addEventListener('click', () => finish(false));
        bar.addEventListener('focusout', (e) => {
            const t = e.relatedTarget;
            // 焦点仍在输入条内，或移向「新建文件夹」按钮/其内部 → 不自动关闭。
            // 移向按钮本职是切换（交予 click/toggle 处理）；若在此延迟关闭，
            // 长按按钮松手时 click 又会走 toggle 重新打开，造成"关闭后又闪开"。
            if (bar.contains(t) || (t?.closest && t.closest('#workspaceNewDirBtn'))) return;
            // 先取消旧定时器再挂新句柄，确保任一时刻至多一个自动关闭在途
            if (workspaceNewDirAutoCloseTimer) clearTimeout(workspaceNewDirAutoCloseTimer);
            workspaceNewDirAutoCloseTimer = setTimeout(() => {
                workspaceNewDirAutoCloseTimer = null;
                closeWorkspaceNewDirInput();
            }, 120);
        });
    }
    workspaceNewDirOpen = true;
    const input = bar.querySelector('.workspace-newdir-input');
    input.value = '';
    input.focus();
}

/** 创建工作区目录：调用后端 CreateDirectory 后通知、展开父目录祖先链并刷新保留选择 */
async function createWorkspaceDirectory(parentRel, name) {
    if (workspaceBusy) { showWorkspaceNewDirInput(); return; }
    setWorkspaceBusy(true, 'workspaceNewDirBtn');
    const bar = ws$('workspaceNewDirBar');
    try {
        let rel;
        try {
            rel = await window.go.main.App.CreateDirectory(parentRel, name);
        } catch (err) {
            window.showNotification?.(`新建文件夹失败：${err?.message || err}`, 'error');
            throw err; // 失败向上抛，由调用方.finish 的 .catch 重开输入条并保留输入
        }
        window.showNotification?.(`已新建文件夹「${name}」`, 'success');
        // 展开父目录祖先链后刷新：后端 ListWorkspaceFiles 现会保留空目录，故新建即见。
        expandWorkspaceTargetChain(parentRel);
        await loadWorkspaceTree();
    } finally {
        setWorkspaceBusy(false, null);
    }
}

/** 懒绑定弹窗内部交互事件（打开时首次调用，只绑定一次）
 * 事件均挂在常驻 DOM 上（遮罩/按钮/树容器委托），关闭无需解绑，无残留监听器 */
function bindWorkspaceEvents() {
    if (workspaceBound) return;
    workspaceBound = true;
    ws$('workspaceClose')?.addEventListener('click', closeWorkspaceManager);
    ws$('workspaceOverlay')?.addEventListener('click', closeWorkspaceManager);
    ws$('workspaceUploadBtn')?.addEventListener('click', uploadWorkspaceFiles);
    ws$('workspaceDownloadBtn')?.addEventListener('click', downloadWorkspaceFiles);
    ws$('workspaceDeleteBtn')?.addEventListener('click', deleteWorkspaceFiles);
    ws$('workspaceRefreshBtn')?.addEventListener('click', refreshWorkspace);
    ws$('workspaceNewDirBtn')?.addEventListener('click', toggleWorkspaceNewDirInput);
    // 阻止新建按钮抢占焦点：输入框不失焦 → focusout 不触发自动关闭，toggle 完全由 click 控制，
    // 从根上消除「长按按钮→松手 click 重开」的闪烁竞态（原依赖 relatedTarget 判定不可靠）。
    ws$('workspaceNewDirBtn')?.addEventListener('mousedown', (e) => e.preventDefault());
    const treeEl = ws$('workspaceTree');
    treeEl?.addEventListener('click', onWorkspaceTreeClick);
    treeEl?.addEventListener('mousemove', onWorkspaceTreeMove);
    treeEl?.addEventListener('change', onWorkspaceTreeChange);
    ws$('workspaceSelectAll')?.addEventListener('click', onWorkspaceSelectAllClick);
    // 面板拖拽上传（不拦截冒泡，由 main.js/ai-chat.js 依赖面板打开守卫屏蔽其他拖拽）
    const wsModalEl = ws$('workspaceModal');
    wsModalEl?.addEventListener('dragenter', onWorkspaceDragEnter);
    wsModalEl?.addEventListener('dragover', onWorkspaceDragOver);
    wsModalEl?.addEventListener('dragleave', onWorkspaceDragLeave);
    wsModalEl?.addEventListener('drop', onWorkspaceDrop);
}

/** 复位拖拽上传相关状态（打开/关闭/刷新共用） */
function resetWorkspaceDropState() {
    workspaceDropTargetRel = null;
    workspaceDropCount = 0;
    clearWorkspaceDropHover();
    ws$('workspaceTree')?.classList.remove('ws-drag-active');
}

/* ============ 对外接口 ============ */

/**
 * 打开工作区管理器：懒绑定事件 + 复位状态 + 显示弹窗 + 加载文件树
 */
export async function openWorkspaceManager() {
    const modal = ws$('workspaceModal');
    if (!modal) return;
    // 后端绑定尚未生成时（wails build / wails generate module 前）给出明确提示
    if (!window.go?.main?.App || typeof window.go.main.App.ListWorkspaceFiles !== 'function') {
        window.showNotification?.('工作区后端绑定尚未生成，请重新构建应用后重试', 'warning');
        return;
    }
    bindWorkspaceEvents();
    // 复位状态：清空选择/展开/树数据与 busy 态
    workspaceSelected = new Set();
    workspaceExpanded = new Set();
    workspaceTreeData = null;
    workspaceDeleting = false;
    closeWorkspaceNewDirInput(); // 新建输入条复位：移除 DOM 并取消挂起的自动关闭定时器
    setWorkspaceBusy(false, null);
    updateWorkspaceToolbar();
    resetWorkspaceDropState();
    // 清空上次残留的树内容，避免显示闪现
    const treeEl = ws$('workspaceTree');
    if (treeEl) treeEl.innerHTML = '';
    modal.style.display = 'flex';
    await loadWorkspaceTree();
}

/**
 * 关闭工作区管理器：隐藏弹窗 + 复位状态与按钮（含 loading/确认残留）
 * 监听器为懒绑定常驻，无需解绑；ESC 由 main.js handleKeyboardNavigation 调用本函数
 */
export function closeWorkspaceManager() {
    const modal = ws$('workspaceModal');
    if (modal) modal.style.display = 'none';
    workspaceSelected = new Set();
    workspaceExpanded = new Set();
    workspaceTreeData = null;
    workspaceDeleting = false;
    setWorkspaceBusy(false, null);
    closeWorkspaceNewDirInput(); // 清理新建输入条残留（含状态复位），下次打开不复现
    resetWorkspaceDropState();
}
