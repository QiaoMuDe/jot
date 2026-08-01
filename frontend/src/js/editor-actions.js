/* ===== 编辑器操作注册表与执行引擎 ===== */

import MD_SYNTAX_ACTIONS from './editor-actions/md-syntax.js';
import FORMAT_ACTIONS from './editor-actions/format.js';
import TEXT_TRANSFORM_ACTIONS from './editor-actions/text-transform.js';
import TEXT_CLEAN_ACTIONS from './editor-actions/text-clean.js';
import ENCODE_DECODE_ACTIONS from './editor-actions/encode-decode.js';
import AI_WRITING_ACTIONS from './editor-actions/ai-writing.js';

/**
 * 操作注册表，配置驱动，分组管理。
 * 每个操作项：{ group, label, errorLabel, handler(text) => string }
 * - group: 分组名称，相同 group 自动归并到同一分组下
 * - label: 操作项显示文本
 * - errorLabel: 错误提示中的格式名称（如 'XML'、'YAML'）
 * - handler: 接收源文本，返回处理后的文本；若无法处理则抛出 Error
 * - type: 可选，'insert' 表示插入/包裹模式（如 MD 语法），默认 'transform' 变换模式
 *
 * 操作项按分组拆分到 editor-actions/ 目录下各模块文件，此处通过展开导入聚合。
 * 后续新增操作只需在对应模块追加条目，无需修改渲染/执行逻辑。
 * 空分组（无任何条目）不渲染。
 */
const EDITOR_ACTIONS = [
    // 格式化（JSON/XML/HTML/CSS/JS/SQL/CSV/YAML/TOML）
    ...FORMAT_ACTIONS,

    // 文本转换
    ...TEXT_TRANSFORM_ACTIONS,

    // 文本清理
    ...TEXT_CLEAN_ACTIONS,

    // 编码解码（Base64/URL/HTML）
    ...ENCODE_DECODE_ACTIONS,

    // MD 语法
    ...MD_SYNTAX_ACTIONS,

    // AI 写作
    ...AI_WRITING_ACTIONS,
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

    // 渲染菜单（分组作为子菜单，子分组作为嵌套子菜单）
    const SUBMENU_ARROW = '<svg class="submenu-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';
    let html = '';
    for (const [groupName, actions] of groups) {
        if (actions.length === 0) continue; // 空分组跳过

        // 按 subGroup 进一步分组
        const subGroups = new Map();
        for (const action of actions) {
            if (!subGroups.has(action.subGroup)) {
                subGroups.set(action.subGroup, []);
            }
            subGroups.get(action.subGroup).push(action);
        }

        html += `<div class="dropdown-item has-submenu">`;
        html += `<span>${groupName}</span>`;
        html += SUBMENU_ARROW;
        html += `<div class="submenu">`;

        // 检查是否有 subGroup
        const hasSubGroups = subGroups.size > 1 || (subGroups.size === 1 && [...subGroups.keys()][0] !== undefined);

        if (hasSubGroups) {
            // 有显式 subGroup，渲染为嵌套子菜单
            for (const [subGroupName, subActions] of subGroups) {
                if (subActions.length === 0) continue;
                if (subActions.length === 1) {
                    // 单个操作项（如 CSV），直接渲染为操作项
                    html += `<div class="dropdown-item" data-action="${subActions[0].label}">${subActions[0].label}</div>`;
                } else {
                    // 多个操作项，渲染为嵌套子菜单
                    html += `<div class="dropdown-item has-submenu">`;
                    html += `<span>${subGroupName}</span>`;
                    html += SUBMENU_ARROW;
                    html += `<div class="submenu">`;
                    for (const action of subActions) {
                        html += `<div class="dropdown-item" data-action="${action.label}">${action.label}</div>`;
                    }
                    html += `</div></div>`;
                }
            }
        } else {
            // 无 subGroup（如文本转换、文本清理），直接渲染操作项
            for (const action of actions) {
                html += `<div class="dropdown-item" data-action="${action.label}">${action.label}</div>`;
            }
        }
        html += `</div></div>`;
    }
    menu.innerHTML = html;

    // AI 操作取消标志，防止取消后 catch 块弹出错误提示
    let _aiOperationCancelled = false;
    // 记录 AI 处理前编辑器的只读状态，处理完成后恢复
    let _aiEditorWasReadOnly = false;

    /**
     * 锁定/解锁编辑器输入（CM6 编辑器 + 标题输入框）
     * 只禁止用户输入，不影响程序化的 dispatch 写回
     * @param {boolean} lock - true=锁定输入
     */
    function setAIEditorLock(lock) {
        // CM6 编辑器（readOnly 仅阻止用户编辑，dispatch 写回不受影响）
        if (typeof window.setCMReadOnly === 'function') {
            window.setCMReadOnly(lock);
        }
        // 标题输入框
        const titleInput = document.getElementById('editorNoteTitle');
        if (titleInput) {
            titleInput.readOnly = lock;
            titleInput.classList.toggle('editor-input-readonly', lock);
        }
        // 遮罩视觉反馈：覆盖编辑器内容区，提示当前处于锁定状态
        const body = document.querySelector('.editor-body');
        let mask = document.getElementById('aiStatusMask');
        if (lock) {
            if (!mask && body) {
                mask = document.createElement('div');
                mask.id = 'aiStatusMask';
                body.appendChild(mask);
            }
        } else if (mask) {
            mask.remove();
        }
    }

    /**
     * 创建 AI 处理指示器
     * 在编辑器头部行的右侧显示一个脉冲动画圆球，点击可取消
     * @returns {HTMLElement} 圆球元素
     */
    function createAIStatusIndicator() {
        const body = document.querySelector('.editor-body');
        if (!body) return null;

        // 如果已存在则复用
        let ball = document.getElementById('aiStatusBall');
        if (ball) return ball;

        ball = document.createElement('div');
        ball.id = 'aiStatusBall';
        ball.className = 'ai-status-ball';
        ball.title = 'AI 处理中，点击取消';
        ball.addEventListener('click', () => {
            _aiOperationCancelled = true;
            if (window.go && window.go.main && window.go.main.App) {
                window.go.main.App.CancelAIStream();
            }
            removeAIStatusIndicator();
        });
        body.appendChild(ball);
        return ball;
    }

    /**
     * 移除 AI 处理指示器
     */
    function removeAIStatusIndicator() {
        const ball = document.getElementById('aiStatusBall');
        if (ball) ball.remove();
    }

    /**
     * 执行操作：读取选中文本或全文 → 交给 handler → 写回
     */
    async function executeAction(handler, errorLabel, actionType = 'transform') {
        const cmEditor = window.cmEditor;
        if (!cmEditor) return;

        // 预览模式自动切回编辑模式（调用全局 switchEditorMode 同步按钮显隐等状态）
        const overlay = document.getElementById('editorOverlay');
        if (overlay && overlay.dataset.mode === 'preview') {
            if (typeof switchEditorMode === 'function') {
                switchEditorMode('edit');
            } else {
                // 降级：手动切换
                const modeBtns = document.querySelectorAll('.mode-btn');
                modeBtns.forEach(btn => btn.classList.toggle('active', btn.dataset.mode === 'edit'));
                overlay.dataset.mode = 'edit';
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
            }
            cmEditor.focus();
        }

        const sel = cmEditor.state.selection.main;
        const hasSelection = !sel.empty;

        // AI 操作必须有选中文本，否则提示并返回
        if (actionType === 'ai' && !hasSelection) {
            const nm = window.nm;
            if (nm && nm.show) {
                nm.show('请先选择要处理的文本', 'warning');
            }
            return;
        }

        const from = hasSelection ? sel.from : (actionType === 'insert' ? sel.from : 0);
        const to = hasSelection ? sel.to : (actionType === 'insert' ? sel.from : cmEditor.state.doc.length);
        const sourceText = cmEditor.state.sliceDoc(from, to);

        const btn = document.getElementById('editorActionsBtn');

        try {
            // AI 操作：显示指示器 + 禁用按钮 + 锁定输入
            if (actionType === 'ai') {
                createAIStatusIndicator();
                if (btn) btn.disabled = true;
                _aiEditorWasReadOnly = cmEditor.state.readOnly;
                setAIEditorLock(true);
            }

            const result = await handler(sourceText);

            // AI 操作：移除指示器 + 恢复按钮 + 解锁输入
            if (actionType === 'ai') {
                removeAIStatusIndicator();
                if (btn) btn.disabled = false;
                setAIEditorLock(_aiEditorWasReadOnly);
            }

            cmEditor.dispatch({
                changes: { from, to, insert: result },
                // 保持选中范围（若无选中则光标移至末尾）
                selection: hasSelection
                    ? { anchor: from, head: from + result.length }
                    : { anchor: from + result.length }
            });
            cmEditor.focus();
        } catch (e) {
            const nm = window.nm;
            // AI 操作：无论成功失败都移除指示器 + 恢复按钮 + 解锁输入
            if (actionType === 'ai') {
                removeAIStatusIndicator();
                if (btn) btn.disabled = false;
                setAIEditorLock(_aiEditorWasReadOnly);
                // 用户主动取消时不提示
                if (_aiOperationCancelled) {
                    _aiOperationCancelled = false;
                    return;
                }
                const errMsg = e?.message || String(e) || '未知错误';
                if (nm && nm.show) {
                    nm.show(`AI 处理失败: ${errMsg}`, 'error');
                }
            } else {
                if (nm && nm.show) {
                    nm.show(`不是合法的 ${errorLabel || '内容'}`, 'warning');
                }
            }
        }
    }

    // 绑定子菜单切换事件（hover 展开/收起）
    menu.addEventListener('mouseover', (e) => {
        const item = e.target.closest('.dropdown-item');
        if (!item) return;

        if (item.classList.contains('has-submenu')) {
            // 悬停在子菜单触发项上，展开其子菜单并关闭同级
            const parent = item.parentElement;
            parent.querySelectorAll(':scope > .has-submenu.open').forEach(el => {
                if (el !== item) el.classList.remove('open');
            });
            item.classList.add('open');
        } else {
            // 悬停在非子菜单项（如 CSV 格式化）时，关闭同级展开的子菜单
            const parent = item.parentElement;
            parent.querySelectorAll(':scope > .has-submenu.open').forEach(el => {
                el.classList.remove('open');
            });
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
            executeAction(action.handler, action.errorLabel, action.type);
            closeMenu();
        }
    });

    // 按钮点击切换菜单
    btn.addEventListener('click', (e) => {
        e.stopPropagation();
        // 移除 inline 样式覆盖，由 CSS 统一控制定位
        menu.style.left = '';
        menu.style.right = '';
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