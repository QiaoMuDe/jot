/* ===== 启动器网格模块（Ctrl+P 触发） ===== */
import { pinyin } from 'pinyin-pro';

/** 13 个功能项定义 */
const launcherItems = [
    {
        action: 'home',
        label: '笔记首页',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>'
    },
    {
        action: 'sidebar-toggle',
        label: '展开侧栏',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="3" x2="9" y2="21"/></svg>'
    },
    {
        action: 'batch-mode',
        label: '批量管理',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>'
    },
    {
        action: 'data',
        label: '数据管理',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>'
    },
    {
        action: 'trash',
        label: '回收站',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>'
    },
    {
        action: 'settings',
        label: '设置',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>'
    },
    {
        action: 'calendar',
        label: '笔记日历',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="3" x2="9" y2="21"/><line x1="15" y1="3" x2="15" y2="21"/></svg>'
    },
    {
        action: 'todo',
        label: '待办清单',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>'
    },
    {
        action: 'ai-chat',
        label: 'AI 助手',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/></svg>'
    },
    {
        action: 'password-manager',
        label: '密码管理',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>'
    },
    {
        action: 'help',
        label: '快捷键说明',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01M8 12h.01M12 12h.01M16 12h.01M6 16h.01M10 16h.01M14 16h.01"/></svg>'
    },
    {
        action: 'md-ref',
        label: 'MD 语法',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>'
    },
    {
        action: 'about',
        label: '关于',
        svg: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
    }
];

/** 当前选中项索引 */
let _selectedIndex = -1;

/** 执行中标志（防止双击/交叉触发 */ 
let _executing = false;

/** DOM 引用 */
let _launcherEl = null;
let _launcherInput = null;
let _launcherGrid = null;
let _launcherEmpty = null;
/** 缓存所有 launcher-item DOM 元素，避免每次过滤都 querySelectorAll */
let _launcherItemEls = null;
/** 过滤防抖定时器 */
let _filterDebounceTimer = null;

/**
 * 获取所有可见（未隐藏）的 item DOM 元素
 */
function getVisibleItems() {
    if (!_launcherGrid) return [];
    return Array.from(_launcherGrid.querySelectorAll('.launcher-item:not(.hidden)'));
}

/**
 * 选中指定索引的卡片
 */
function selectItem(index) {
    const items = getVisibleItems();
    // 清除旧的选中
    items.forEach(el => el.classList.remove('selected'));
    if (index < 0 || index >= items.length) {
        _selectedIndex = -1;
        return;
    }
    _selectedIndex = index;
    items[index].classList.add('selected');
    // 确保选中项在视口中可见
    items[index].scrollIntoView({ block: 'nearest', behavior: 'smooth' });
}

/** 标签拼音索引缓存：label → { full: 全拼连续串, initials: 首字母串 } */
const _pinyinIndex = new Map();

/**
 * 获取标签的拼音检索键（懒计算并缓存）
 * 非汉字字符原样保留，便于英文标签同样可被全拼串命中
 */
function getPinyinKey(label) {
    let key = _pinyinIndex.get(label);
    if (!key) {
        key = {
            full: pinyin(label, { toneType: 'none', type: 'array' }).join('').toLowerCase(),
            initials: pinyin(label, { pattern: 'first', toneType: 'none', type: 'array' }).join('').toLowerCase()
        };
        _pinyinIndex.set(label, key);
    }
    return key;
}

/**
 * 根据搜索输入过滤网格卡片（支持中文原文 / 全拼片段 / 拼音首字母）
 */
function filterItems(query) {
    if (!_launcherGrid) return;
    const items = _launcherItemEls || _launcherGrid.querySelectorAll('.launcher-item');
    const trimmed = query.trim().toLowerCase();
    // 去掉空白后用于拼音连续串匹配（如输入 "dai ban" 也能命中 "daiban..."）
    const compact = trimmed.replace(/\s+/g, '');
    let visibleCount = 0;
    let firstMatchIndex = -1;

    items.forEach((el, i) => {
        const label = el.dataset.label || '';
        let matches = !trimmed;
        if (trimmed && !matches) {
            if (label.toLowerCase().includes(trimmed)) {
                matches = true;
            } else if (compact) {
                const key = getPinyinKey(label);
                matches = key.full.includes(compact) || key.initials.includes(compact);
            }
        }
        el.classList.toggle('hidden', !matches);
        if (matches) {
            if (firstMatchIndex === -1) firstMatchIndex = visibleCount;
            visibleCount++;
        }
    });

    // 显示/隐藏空结果提示
    if (_launcherEmpty) {
        _launcherEmpty.classList.toggle('visible', visibleCount === 0);
    }

    // 自动选中第一个匹配项
    selectItem(firstMatchIndex >= 0 ? firstMatchIndex : -1);
}

/**
 * 执行指定的功能操作
 */
function executeAction(action) {
    if (_executing) return;
    const win = window;
    const els = win.els;
    if (!els) return;

    _executing = true;
    closeLauncher(() => {
        _executing = false;
        switch (action) {
            case 'home':
                if (win.resetPagination) win.resetPagination();
                if (win.switchView) win.switchView('grid');
                if (win.loadNotes) win.loadNotes();
                break;
            case 'sidebar-toggle':
                if (win.state && win.state.currentView !== 'grid') {
                    // 非 grid 视图：先跳转首页再展开侧栏
                    if (win.switchView) win.switchView('grid');
                    if (win.resetPagination) win.resetPagination();
                    if (win.loadNotes) win.loadNotes();
                    if (els.notebookSidebar?.classList.contains('collapsed')) {
                        els.notebookSidebar.classList.remove('collapsed');
                        localStorage.setItem('jot_sidebar_collapsed', 'false');
                        if (win.updateNotebookSidebarToggleBtn) win.updateNotebookSidebarToggleBtn();
                        // 此路径绕过 toggleSidebar，需手动刷新笔记本计数（否则展开后计数为旧值）
                        if (win.loadNotebooks) win.loadNotebooks();
                    }
                } else {
                    if (win.toggleSidebar) win.toggleSidebar();
                    if (win.updateNotebookSidebarToggleBtn) win.updateNotebookSidebarToggleBtn();
                }
                break;
            case 'batch-mode':
                if (win.switchView) win.switchView('grid');
                if (win.toggleBatchMode) win.toggleBatchMode();
                break;
            case 'settings':
                if (win.switchView) win.switchView('settings');
                break;
            case 'data':
                if (win.switchView) win.switchView('data');
                break;
            case 'trash':
                if (win.switchView) win.switchView('trash');
                if (win.loadTrashNotes) win.loadTrashNotes();
                break;
            case 'md-ref':
                if (win.switchView) win.switchView('md-ref');
                break;
            case 'ai-chat':
                if (win.switchView) win.switchView('ai-chat');
                break;
            case 'calendar':
                if (win.switchView) win.switchView('calendar');
                break;
            case 'todo':
                if (win.switchView) win.switchView('todo');
                break;
            case 'password-manager':
                if (win.switchView) win.switchView('password-manager');
                break;
            case 'help':
                if (win.openShortcuts) win.openShortcuts();
                break;
            case 'about':
                if (win.showAbout) win.showAbout();
                break;
        }
    });
}

/**
 * 处理方向键和 Enter 导航
 */
function handleLauncherKeydown(e) {
    const items = getVisibleItems();
    if (items.length === 0) return;

    // 未选中任何项时，方向键直接跳转到第一个
    if (_selectedIndex === -1 && ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) {
        e.preventDefault();
        selectItem(0);
        return;
    }

    // 左/上 方向键：向上翻
    if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        e.preventDefault();
        const cols = 4;
        let newIndex;
        if (e.key === 'ArrowUp') {
            // 上一行（减列数）
            newIndex = _selectedIndex - cols;
            if (newIndex < 0) {
                // 循环到底行
                const rows = Math.ceil(items.length / cols);
                newIndex = _selectedIndex + (rows - 1) * cols;
                if (newIndex >= items.length) newIndex = items.length - 1;
            }
        } else {
            // ArrowLeft: 左移一位
            newIndex = _selectedIndex - 1;
            if (newIndex < 0) newIndex = items.length - 1;
        }
        selectItem(newIndex);
        return;
    }

    // 右/下 方向键：向下翻
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
        e.preventDefault();
        const cols = 4;
        let newIndex;
        if (e.key === 'ArrowDown') {
            // 下一行（加列数）
            newIndex = _selectedIndex + cols;
            if (newIndex >= items.length) {
                // 循环回到顶行
                newIndex = _selectedIndex % cols;
            }
        } else {
            // ArrowRight: 右移一位
            newIndex = _selectedIndex + 1;
            if (newIndex >= items.length) newIndex = 0;
        }
        selectItem(newIndex);
        return;
    }

    // Enter: 执行选中的操作
    if (e.key === 'Enter') {
        e.preventDefault();
        const items = getVisibleItems();
        if (_selectedIndex >= 0 && _selectedIndex < items.length) {
            const action = items[_selectedIndex].dataset.action;
            if (action) executeAction(action);
        }
        return;
    }

    // Tab: 阻止跳出启动器
    if (e.key === 'Tab') {
        e.preventDefault();
    }
}

/**
 * 渲染网格卡片
 */
function renderLauncherItems() {
    if (!_launcherGrid) return;
    _launcherGrid.innerHTML = launcherItems.map((item, index) =>
        `<div class="launcher-item" data-action="${item.action}" data-label="${item.label}" data-index="${index}">
            <div class="launcher-item-icon">${item.svg}</div>
            <span class="launcher-item-label">${item.label}</span>
        </div>`
    ).join('');
    // 缓存 item 元素列表，避免每次过滤都重新查询 DOM
    _launcherItemEls = _launcherGrid.querySelectorAll('.launcher-item');
}

/**
 * 打开启动器
 */
function openLauncher() {
    if (!_launcherEl) return;
    // 清除可能残留的离场状态
    _launcherEl.classList.remove('closing');
    // 设置 display: flex 使其变为可见
    _launcherEl.style.display = 'flex';
    // 重置搜索
    if (_launcherInput) {
        _launcherInput.value = '';
        _launcherInput.focus();
    }
    // 重置过滤（显示全部）
    filterItems('');
    // 列表滚动位置归顶
    if (_launcherGrid) _launcherGrid.scrollTop = 0;

    // 根据侧栏当前状态更新 sidebar-toggle 项的标签和图标
    const sidebarToggleEl = _launcherGrid?.querySelector('[data-action="sidebar-toggle"]');
    if (sidebarToggleEl) {
        const isCollapsed = window.els?.notebookSidebar?.classList.contains('collapsed');
        const label = isCollapsed ? '展开侧栏' : '折叠侧栏';
        const expandSvg = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="3" x2="9" y2="21"/></svg>';
        const collapseSvg = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="15" y1="3" x2="15" y2="21"/></svg>';
        sidebarToggleEl.querySelector('.launcher-item-label').textContent = label;
        sidebarToggleEl.querySelector('.launcher-item-icon').innerHTML = isCollapsed ? expandSvg : collapseSvg;
        sidebarToggleEl.dataset.label = label;
    }

    // 根据批量管理模式状态更新 batch-mode 项的标签
    const batchModeEl = _launcherGrid?.querySelector('[data-action="batch-mode"]');
    if (batchModeEl) {
        const batchLabel = window.state?.batchMode ? '退出管理' : '批量管理';
        batchModeEl.querySelector('.launcher-item-label').textContent = batchLabel;
        batchModeEl.dataset.label = batchLabel;
    }

    // 通过 requestAnimationFrame 确保 display 生效后再触发动画
    requestAnimationFrame(() => {
        _launcherEl.classList.add('visible');
    });
}

/**
 * 关闭启动器
 * @param {Function} [onDone] - 离场动画完成后调用
 */
function closeLauncher(onDone) {
    // 清理过滤防抖定时器，避免关闭后仍执行无意义的过滤
    clearTimeout(_filterDebounceTimer);
    if (!_launcherEl || !_launcherEl.classList.contains('visible')) {
        if (typeof onDone === 'function') onDone();
        return;
    }
    _launcherEl.classList.add('closing');
    _launcherEl.classList.remove('visible');

    let _closed = false;
    const onEnd = () => {
        if (_closed) return;
        _closed = true;
        _launcherEl.style.display = 'none';
        _launcherEl.classList.remove('closing');
        _launcherEl.removeEventListener('transitionend', onEnd);
        if (typeof onDone === 'function') onDone();
    };
    _launcherEl.addEventListener('transitionend', onEnd);
    // 保险：在离场动画最长时长后强制关闭
    setTimeout(onEnd, 300);
}

/**
 * 初始化启动器模块
 */
export function initLauncher() {
    // 获取 DOM 引用
    _launcherEl = document.getElementById('launcher');
    _launcherInput = document.getElementById('launcherInput');
    _launcherGrid = document.getElementById('launcherGrid');
    _launcherEmpty = document.getElementById('launcherEmpty');

    if (!_launcherEl || !_launcherInput || !_launcherGrid) {
        console.warn('[Launcher] DOM elements not found');
        return;
    }

    // 渲染卡片
    renderLauncherItems();

    // 输入框 input 事件：实时过滤（带防抖，避免快速输入时频繁过滤）
    _launcherInput.addEventListener('input', () => {
        clearTimeout(_filterDebounceTimer);
        _filterDebounceTimer = setTimeout(() => {
            filterItems(_launcherInput.value);
        }, 80);
    });

    // 输入框 keydown 事件：方向键导航
    _launcherInput.addEventListener('keydown', (e) => {
        // 只拦截方向键、Enter、Tab
        if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', 'Tab'].includes(e.key)) {
            handleLauncherKeydown(e);
        }
    });

    // 遮罩点击关闭
    const mask = _launcherEl.querySelector('.launcher-mask');
    if (mask) {
        mask.addEventListener('click', () => {
            closeLauncher();
        });
    }

    // 网格卡片点击
    _launcherGrid.addEventListener('click', (e) => {
        const item = e.target.closest('.launcher-item');
        if (item && item.dataset.action) {
            executeAction(item.dataset.action);
        }
    });

    // 将函数暴露到 window 上，供 main.js 或全局键盘处理调用
    window.openLauncher = openLauncher;
    window.closeLauncher = closeLauncher;

    // 标记是否已初始化
    _launcherEl.dataset.inited = 'true';
}
