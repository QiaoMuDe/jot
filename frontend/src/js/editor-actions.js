/* ===== 编辑器操作注册表与执行引擎 ===== */

import { format as xmlFormat, minify as xmlMinify } from './formatters/xml-formatter.js';
import { format as htmlFormat, minify as htmlMinify } from './formatters/html-formatter.js';
import { format as csvFormat } from './formatters/csv-formatter.js';
import * as yaml from 'js-yaml';
import * as smolToml from 'smol-toml';
import { format as sqlFormat } from 'sql-formatter';
import * as beautify from 'js-beautify';

/**
 * 操作注册表，配置驱动，分组管理。
 * 每个操作项：{ group, label, errorLabel, handler(text) => string }
 * - group: 分组名称，相同 group 自动归并到同一分组下
 * - label: 操作项显示文本
 * - errorLabel: 错误提示中的格式名称（如 'XML'、'YAML'）
 * - handler: 接收源文本，返回处理后的文本；若无法处理则抛出 Error
 *
 * 后续新增操作只需在此数组追加条目，无需修改渲染/执行逻辑。
 * 空分组（无任何条目）不渲染。
 */
const EDITOR_ACTIONS = [
    // ── JSON ──
    {
        group: '格式化',
        subGroup: 'JSON',
        label: 'JSON 格式化',
        errorLabel: 'JSON',
        handler(text) {
            const parsed = JSON.parse(text);
            return JSON.stringify(parsed, null, 2);
        }
    },
    {
        group: '格式化',
        subGroup: 'JSON',
        label: 'JSON 压缩',
        errorLabel: 'JSON',
        handler(text) {
            const parsed = JSON.parse(text);
            return JSON.stringify(parsed);
        }
    },

    // ── XML ──
    {
        group: '格式化',
        subGroup: 'XML',
        label: 'XML 格式化',
        errorLabel: 'XML',
        handler(text) { return xmlFormat(text); }
    },
    {
        group: '格式化',
        subGroup: 'XML',
        label: 'XML 压缩',
        errorLabel: 'XML',
        handler(text) { return xmlMinify(text); }
    },

    // ── HTML ──
    {
        group: '格式化',
        subGroup: 'HTML',
        label: 'HTML 格式化',
        errorLabel: 'HTML',
        handler(text) { return htmlFormat(text); }
    },
    {
        group: '格式化',
        subGroup: 'HTML',
        label: 'HTML 压缩',
        errorLabel: 'HTML',
        handler(text) { return htmlMinify(text); }
    },

    // ── CSS ──
    {
        group: '格式化',
        subGroup: 'CSS',
        label: 'CSS 格式化',
        errorLabel: 'CSS',
        handler(text) { return beautify.css_beautify(text, { indent_size: 2 }); }
    },
    {
        group: '格式化',
        subGroup: 'CSS',
        label: 'CSS 压缩',
        errorLabel: 'CSS',
        handler(text) { return beautify.css_beautify(text, { indent_size: 0, preserve_newlines: false }); }
    },

    // ── JavaScript ──
    {
        group: '格式化',
        subGroup: 'JavaScript',
        label: 'JS 格式化',
        errorLabel: 'JavaScript',
        handler(text) { return beautify.js_beautify(text, { indent_size: 2 }); }
    },
    {
        group: '格式化',
        subGroup: 'JavaScript',
        label: 'JS 压缩',
        errorLabel: 'JavaScript',
        handler(text) { return beautify.js_beautify(text, { indent_size: 0, preserve_newlines: false }); }
    },

    // ── SQL ──
    {
        group: '格式化',
        subGroup: 'SQL',
        label: 'SQL 格式化',
        errorLabel: 'SQL',
        handler(text) { return sqlFormat(text, { indent: '  ' }); }
    },
    {
        group: '格式化',
        subGroup: 'SQL',
        label: 'SQL 压缩',
        errorLabel: 'SQL',
        handler(text) {
            const formatted = sqlFormat(text, { indent: '  ' });
            // 压缩：去掉多余换行
            return formatted.replace(/\n{3,}/g, '\n\n');
        }
    },

    // ── CSV ──
    {
        group: '格式化',
        subGroup: 'CSV',
        label: 'CSV 格式化',
        errorLabel: 'CSV',
        handler(text) { return csvFormat(text); }
    },

    // ── YAML ──
    {
        group: '格式化',
        subGroup: 'YAML',
        label: 'YAML 格式化',
        errorLabel: 'YAML',
        handler(text) {
            const parsed = yaml.load(text);
            return yaml.dump(parsed, { indent: 2, lineWidth: 120, noRefs: true });
        }
    },
    {
        group: '格式化',
        subGroup: 'YAML',
        label: 'YAML 压缩',
        errorLabel: 'YAML',
        handler(text) {
            const parsed = yaml.load(text);
            return yaml.dump(parsed, { indent: 2, lineWidth: -1, flowLevel: 0, noRefs: true });
        }
    },

    // ── TOML ──
    {
        group: '格式化',
        subGroup: 'TOML',
        label: 'TOML 格式化',
        errorLabel: 'TOML',
        handler(text) {
            const parsed = smolToml.parse(text);
            return smolToml.stringify(parsed);
        }
    },
    {
        group: '格式化',
        subGroup: 'TOML',
        label: 'TOML 压缩',
        errorLabel: 'TOML',
        handler(text) {
            const parsed = smolToml.parse(text);
            return smolToml.stringify(parsed);
        }
    },

    // ── 文本转换 ──
    {
        group: '文本转换',
        label: '大写',
        errorLabel: '文本',
        handler(text) { return text.toUpperCase(); }
    },
    {
        group: '文本转换',
        label: '小写',
        errorLabel: '文本',
        handler(text) { return text.toLowerCase(); }
    },
    {
        group: '文本转换',
        label: '首字母大写',
        errorLabel: '文本',
        handler(text) {
            return text.replace(/\b\w/g, ch => ch.toUpperCase());
        }
    },
    {
        group: '文本转换',
        label: '驼峰式',
        errorLabel: '文本',
        handler(text) {
            return text
                .replace(/[^a-zA-Z0-9]+/g, ' ')
                .trim()
                .split(/\s+/)
                .map((word, i) => {
                    if (i === 0) return word.toLowerCase();
                    return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
                })
                .join('');
        }
    },
    {
        group: '文本转换',
        label: '蛇形式',
        errorLabel: '文本',
        handler(text) {
            return text
                .replace(/([A-Z])/g, ' $1')
                .replace(/[^a-zA-Z0-9]+/g, ' ')
                .trim()
                .toLowerCase()
                .split(/\s+/)
                .join('_');
        }
    },
    {
        group: '文本转换',
        label: '行反转',
        errorLabel: '文本',
        handler(text) {
            return text.split('\n').reverse().join('\n');
        }
    },
    {
        group: '文本转换',
        label: '字符反转',
        errorLabel: '文本',
        handler(text) {
            return text.split('').reverse().join('');
        }
    },

    // ── 文本清理 ──
    {
        group: '文本清理',
        label: '去除多余空格',
        errorLabel: '文本',
        handler(text) {
            return text.replace(/\s+/g, ' ').trim();
        }
    },
    {
        group: '文本清理',
        label: '去除空行',
        errorLabel: '文本',
        handler(text) {
            return text.split('\n').filter(l => l.trim()).join('\n');
        }
    },
    {
        group: '文本清理',
        label: '行尾空格清理',
        errorLabel: '文本',
        handler(text) {
            return text.split('\n').map(l => l.trimEnd()).join('\n');
        }
    },
    {
        group: '文本清理',
        label: 'Tab 转空格',
        errorLabel: '文本',
        handler(text) {
            return text.replace(/\t/g, '  ');
        }
    },
    {
        group: '文本清理',
        label: '空格转 Tab',
        errorLabel: '文本',
        handler(text) {
            return text.replace(/  /g, '\t');
        }
    },

    // ── 编码解码 ──
    {
        group: '编码解码',
        subGroup: 'Base64',
        label: 'Base64 编码',
        errorLabel: 'Base64',
        handler(text) { return btoa(text); }
    },
    {
        group: '编码解码',
        subGroup: 'Base64',
        label: 'Base64 解码',
        errorLabel: 'Base64',
        handler(text) { return atob(text); }
    },
    {
        group: '编码解码',
        subGroup: 'URL',
        label: 'URL 编码',
        errorLabel: 'URL 编码',
        handler(text) { return encodeURIComponent(text); }
    },
    {
        group: '编码解码',
        subGroup: 'URL',
        label: 'URL 解码',
        errorLabel: 'URL 编码',
        handler(text) { return decodeURIComponent(text); }
    },
    {
        group: '编码解码',
        subGroup: 'HTML',
        label: 'HTML 编码',
        errorLabel: 'HTML',
        handler(text) {
            return text
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#39;');
        }
    },
    {
        group: '编码解码',
        subGroup: 'HTML',
        label: 'HTML 解码',
        errorLabel: 'HTML',
        handler(text) {
            const doc = new DOMParser().parseFromString(text, 'text/html');
            return doc.body.textContent || '';
        }
    },
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

    /**
     * 执行操作：读取选中文本或全文 → 交给 handler → 写回
     */
    function executeAction(handler, errorLabel) {
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
            // 统一错误提示：使用 errorLabel 保证格式一致
            const nm = window.nm;
            if (nm && nm.show) {
                nm.show(`不是合法的 ${errorLabel || '内容'}`, 'warning');
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
            executeAction(action.handler, action.errorLabel);
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