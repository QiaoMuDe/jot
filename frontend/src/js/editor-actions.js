/* ===== 编辑器操作注册表与执行引擎 ===== */

/**
 * 操作注册表，配置驱动，分组管理。
 * 每个操作项：{ group, label, handler(text) => string }
 * - group: 分组名称，相同 group 自动归并到同一分组下
 * - label: 操作项显示文本
 * - handler: 接收源文本，返回处理后的文本；若无法处理则抛出 Error
 *
 * 后续新增操作只需在此数组追加条目，无需修改渲染/执行逻辑。
 * 空分组（无任何条目）不渲染。
 */
const EDITOR_ACTIONS = [
    // ── 格式化分组 ──
    {
        group: '格式化',
        label: 'JSON 格式化',
        handler(text) {
            const parsed = JSON.parse(text);
            return JSON.stringify(parsed, null, 2);
        }
    },
    {
        group: '格式化',
        label: 'JSON 压缩',
        handler(text) {
            const parsed = JSON.parse(text);
            return JSON.stringify(parsed);
        }
    },

    // ── 后续分组示例（首期不启用，解除注释即可） ──
    // { group: '编码解码', label: 'Base64 编码', handler(text) { return btoa(text); } },
    // { group: '编码解码', label: 'Base64 解码', handler(text) { return atob(text); } },
    // { group: 'AI', label: '润色文本', handler(text) { return text; } },
];

/**
 * 初始化编辑器操作菜单
 * - 渲染分组下拉菜单
 * - 绑定按钮点击开合事件
 * - 绑定外部点击 / Esc 关闭
 */
function initEditorActionsMenu() {
    const btn = document.getElementById('editorActionsBtn');
    const menu = document.getElementById('editorActionsMenu');
    if (!btn || !menu) return;

    // 按 group 分组聚合
    const groups = new Map();
    for (const action of EDITOR_ACTIONS) {
        if (!groups.has(action.group)) {
            groups.set(action.group, []);
        }
        groups.get(action.group).push(action);
    }

    // 渲染菜单（分组作为子菜单）
    let html = '';
    for (const [groupName, actions] of groups) {
        if (actions.length === 0) continue; // 空分组跳过
        html += `<div class="dropdown-item has-submenu">`;
        html += `<span>${groupName}</span>`;
        html += `<svg class="submenu-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>`;
        html += `<div class="submenu">`;
        for (const action of actions) {
            html += `<div class="dropdown-item" data-action="${action.label}">${action.label}</div>`;
        }
        html += `</div></div>`;
    }
    menu.innerHTML = html;

    /**
     * 执行操作：读取选中文本或全文 → 交给 handler → 写回
     */
    function executeAction(handler) {
        const cmEditor = window.cmEditor;
        if (!cmEditor) return;

        // 预览模式自动切回编辑模式
        const overlay = document.getElementById('editorOverlay');
        if (overlay && overlay.dataset.mode === 'preview') {
            // 调用全局 switchEditorMode 切回编辑模式
            const modeBtns = document.querySelectorAll('.mode-btn');
            modeBtns.forEach(btn => btn.classList.toggle('active', btn.dataset.mode === 'edit'));
            overlay.dataset.mode = 'edit';
            // 切换后关闭预览相关布局
            const tocSidebar = document.getElementById('tocSidebar');
            if (tocSidebar) {
                const tocBody = document.getElementById('tocBody');
                if (tocBody) tocBody.innerHTML = '';
                tocSidebar.classList.remove('visible');
            }
            const previewEl = document.getElementById('mdRendered');
            if (previewEl) previewEl.style.display = 'none';
            const textarea = document.getElementById('editorNoteContent');
            if (textarea) textarea.style.display = 'flex';
            cmEditor.focus();
        }

        const sel = cmEditor.state.selection.main;
        const hasSelection = !sel.empty;
        const from = hasSelection ? sel.from : 0;
        const to = hasSelection ? sel.to : cmEditor.state.doc.length;
        const sourceText = cmEditor.state.sliceDoc(from, to);

        try {
            const result = handler(sourceText);
            cmEditor.dispatch({
                changes: { from, to, insert: result },
                // 保持选中范围（若无选中则光标移至末尾）
                selection: hasSelection
                    ? { anchor: from, head: from + result.length }
                    : { anchor: from + result.length }
            });
            cmEditor.focus();
        } catch (e) {
            // JSON 解析失败等场景
            const nm = window.nm;
            if (nm && nm.show) {
                nm.show('不是合法的 JSON', 'warning');
            }
        }
    }

    // 绑定子菜单切换事件（hover 展开/收起）
    menu.addEventListener('mouseover', (e) => {
        const submenuTrigger = e.target.closest('.has-submenu');
        if (submenuTrigger) {
            // 关闭同级其他子菜单
            menu.querySelectorAll('.has-submenu.open').forEach(el => {
                if (el !== submenuTrigger) el.classList.remove('open');
            });
            submenuTrigger.classList.add('open');
        }
    });

    // 鼠标离开菜单区域时关闭所有子菜单
    menu.addEventListener('mouseleave', () => {
        menu.querySelectorAll('.has-submenu.open').forEach(el => {
            el.classList.remove('open');
        });
    });

    // 绑定操作项点击事件（事件代理）
    menu.addEventListener('click', (e) => {
        const item = e.target.closest('.dropdown-item');
        if (!item) return;
        // 子菜单触发项本身不执行操作，仅切换子菜单
        if (item.classList.contains('has-submenu')) return;
        const label = item.dataset.action;
        const action = EDITOR_ACTIONS.find(a => a.label === label);
        if (action) {
            executeAction(action.handler);
            closeMenu();
        }
    });

    // 按钮点击切换菜单
    btn.addEventListener('click', (e) => {
        e.stopPropagation();
        // 确保菜单锚定在按钮左侧（右缘贴按钮左缘），与 CSS 保持一致
        menu.style.left = 'auto';
        menu.style.right = 'calc(100% + 4px)';
        menu.classList.toggle('active');
    });

    // 关闭菜单（同时收起所有子菜单）
    function closeMenu() {
        menu.classList.remove('active');
        menu.querySelectorAll('.has-submenu.open').forEach(el => {
            el.classList.remove('open');
        });
    }

    // 外部点击关闭
    document.addEventListener('click', (e) => {
        if (menu.classList.contains('active') &&
            !btn.contains(e.target) &&
            !menu.contains(e.target)) {
            closeMenu();
        }
    });

    // Esc 关闭
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && menu.classList.contains('active')) {
            closeMenu();
            btn.focus();
        }
    });
}

// 暴露初始化函数供 main.js 调用
window.initEditorActionsMenu = initEditorActionsMenu;