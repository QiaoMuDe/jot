/**
 * CM6 通用语法高亮模块
 *
 * 封装 CodeMirror 6 编辑器主题 + 两套独立的语法高亮配色方案：
 * - mdHighlightStyle：Markdown 语法（标题、加粗、链接等 16 种节点）
 * - codeHighlightStyle：编程语言通用（关键字、字符串、函数名等约 20 种节点）
 *
 * 语言解析器选择策略（getHighlightExtension）：
 * - 有 @codemirror/lang-* 原生包的优先使用
 * - 其余语言通过 @codemirror/legacy-modes + StreamLanguage 加载
 * - 未知扩展名返回空数组（无高亮）
 *
 * 所有颜色引用 CSS 变量，跟随应用 6 套主题自动变化。
 */
import { css } from '@codemirror/lang-css';
import { html } from '@codemirror/lang-html';
import { javascript } from '@codemirror/lang-javascript';
import { json } from '@codemirror/lang-json';
import { markdown } from '@codemirror/lang-markdown';
import { python } from '@codemirror/lang-python';
import { HighlightStyle, StreamLanguage, syntaxHighlighting } from '@codemirror/language';
import { EditorView } from '@codemirror/view';
import { tags } from '@lezer/highlight';

// @codemirror/legacy-modes 兜底语言
import { cpp, csharp, dart, java, scala } from '@codemirror/legacy-modes/mode/clike';
import { dockerFile } from '@codemirror/legacy-modes/mode/dockerfile';
import { go } from '@codemirror/legacy-modes/mode/go';
import { haskell } from '@codemirror/legacy-modes/mode/haskell';
import { lua } from '@codemirror/legacy-modes/mode/lua';
import { perl } from '@codemirror/legacy-modes/mode/perl';
import { powerShell } from '@codemirror/legacy-modes/mode/powershell';
import { ruby } from '@codemirror/legacy-modes/mode/ruby';
import { rust } from '@codemirror/legacy-modes/mode/rust';
import { shell } from '@codemirror/legacy-modes/mode/shell';
import { sql } from '@codemirror/legacy-modes/mode/sql';
import { swift } from '@codemirror/legacy-modes/mode/swift';
import { xml } from '@codemirror/legacy-modes/mode/xml';
import { yaml } from '@codemirror/legacy-modes/mode/yaml';

/* ======================================================================== */

/** 编辑器视觉主题，引用 CSS 变量实现 6 主题联动 */
export const jotTheme = EditorView.theme({
    '&': {
        backgroundColor: 'var(--card-bg)',
        color: 'var(--text-primary)',
        fontFamily: 'var(--font-family)',
        fontSize: 'var(--font-size-base)',
        flex: '1 1 0',
        minHeight: 0,
    },
    '.cm-scroller': {
        fontFamily: 'var(--font-family)',
        lineHeight: '1.7',
        overflow: 'auto',
    },
    '.cm-content': {
        caretColor: 'var(--accent)',
        padding: '0',
        fontFamily: 'var(--font-family)',
        fontSize: '0.938rem',
    },
    '.cm-cursor': {
        borderLeftColor: 'var(--accent)',
        borderLeftWidth: '2px',
    },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
        backgroundColor: 'var(--accent-light) !important',
    },
    '.cm-activeLine': {
        backgroundColor: 'rgba(var(--accent-rgb), 0.05)',
    },
    '.cm-gutters': {
        backgroundColor: 'var(--card-bg)',
        border: 'none',
    },
    '.cm-lineNumbers .cm-gutterElement': {
        color: 'var(--text-muted)',
        fontSize: '0.75rem',
        lineHeight: '2.13',
        padding: '0 4px 0 4px',
    },
    '.cm-foldGutter .cm-gutterElement': {
        color: 'var(--text-muted)',
        fontSize: '0.75rem',
        padding: '0 2px 0 8px',
        cursor: 'default',
    },
    '.cm-matchingBracket': {
        backgroundColor: 'var(--accent-light)',
        outline: 'none',
    },
    '&.cm-focused .cm-matchingBracket': {
        backgroundColor: 'var(--accent-light)',
    },
    '.cm-searchMatch': {
        backgroundColor: 'var(--accent-light)',
    },
    '.cm-searchMatch.selected': {
        backgroundColor: 'var(--accent)',
    },
});

/* ======================================================================== */
/* 第一套配色：Markdown 语法高亮                                       */
/* ======================================================================== */

/** Markdown 语法节点 → CSS 样式映射，覆盖 16 种元素 */
const mdHighlightStyle = HighlightStyle.define([
    { tag: tags.heading1, fontSize: '1.5rem', fontWeight: '700', color: 'var(--accent)' },
    { tag: tags.heading2, fontSize: '1.25rem', fontWeight: '700', color: 'var(--accent)' },
    { tag: tags.heading3, fontSize: '1.1rem', fontWeight: '600', color: 'var(--accent)' },
    { tag: tags.heading4, fontSize: '1rem', fontWeight: '600', color: 'var(--text-primary)' },
    { tag: tags.heading5, fontSize: '0.938rem', fontWeight: '600', color: 'var(--text-primary)' },
    { tag: tags.heading6, fontSize: '0.875rem', fontWeight: '600', color: 'var(--text-secondary)' },
    { tag: tags.strong, fontWeight: '700' },
    { tag: tags.emphasis, fontStyle: 'italic' },
    { tag: tags.strikethrough, textDecoration: 'line-through' },
    { tag: tags.link, color: 'var(--accent)', textDecoration: 'underline', cursor: 'pointer' },
    { tag: tags.url, color: 'var(--text-muted)', fontStyle: 'italic' },
    { tag: tags.quote, color: 'var(--text-secondary)', fontStyle: 'italic' },
    { tag: tags.monospace, background: 'var(--hover-bg)', borderRadius: '3px', padding: '1px 4px', fontFamily: 'Consolas, Monaco, monospace', fontSize: '0.85em' },
    { tag: tags.comment, color: 'var(--text-muted)', fontStyle: 'italic' },
    { tag: tags.list, color: 'var(--accent)', fontWeight: '500' },
    { tag: tags.contentSeparator, borderTop: '1px solid var(--border)', display: 'block', margin: '0.5em 0' },
    { tag: tags.escape, color: 'var(--text-muted)', fontWeight: '600' },
    { tag: tags.character, color: 'var(--text-muted)' },
    { tag: tags.labelName, color: 'var(--text-secondary)', fontStyle: 'italic' },
    { tag: tags.string, color: 'var(--text-secondary)' },
    { tag: tags.processingInstruction, color: 'var(--text-muted)', opacity: '0.6' },
]);

/* ======================================================================== */
/* 第二套配色：编程语言通用语法高亮                                     */
/* ======================================================================== */

/** 编程语言通用语法节点 → CSS 样式映射（Monokai Dimmed 风格配色）
 *
 *  配色参考 VS Code Monokai Dimmed 主题：
 *  - 关键字/类型用高饱和度冷色，代码逻辑结构一目了然
 *  - 字符串用醒目的金色，与代码逻辑全程区分
 *  - 运算符/标签用亮粉色，操作符位置一瞥即知
 *  - JSON 键名用橙色，是数据与代码之间明确的视觉锚点
 *  - 注释用低饱和度橄榄色，安静不抢眼
 */
const codeHighlightStyle = HighlightStyle.define([
    // ── 关键字 ──────────────────────────────────────────────────
    { tag: tags.keyword, color: '#AE81FF', fontWeight: '600' },
    { tag: [tags.keyword, tags.operator], color: '#AE81FF', fontWeight: '600' },
    { tag: [tags.keyword, tags.modifier], color: '#AE81FF', fontWeight: '600' },
    { tag: [tags.keyword, tags.typeName], color: '#AE81FF', fontWeight: '600' },
    { tag: tags.controlKeyword, color: '#AE81FF', fontWeight: '600' },
    { tag: tags.moduleKeyword, color: '#AE81FF', fontWeight: '600' },

    // ── 类型 / 类 ──────────────────────────────────────────────
    { tag: tags.typeName, color: '#66D9EF' },               // 青色
    { tag: tags.className, color: '#66D9EF', fontWeight: '700' },

    // ── 函数 ────────────────────────────────────────────────────
    { tag: tags.definition(tags.variableName), color: '#A6E22E', fontWeight: '600' },

    // ── 变量 / 属性 ────────────────────────────────────────────
    { tag: tags.variableName, color: 'var(--text-primary)' },
    { tag: tags.definition(tags.variableName), color: 'var(--text-primary)' },
    { tag: tags.propertyName, color: '#FD971F' },            // 橙（JSON 键名）

    // ── 字面量 ──────────────────────────────────────────────────
    { tag: tags.number, color: '#AE81FF' },                  // 紫（同关键字）
    { tag: tags.string, color: '#E6DB74' },                  // 金色
    { tag: tags.special(tags.string), color: '#E6DB74', fontWeight: '600' },
    { tag: tags.regexp, color: '#E6DB74', background: 'rgba(230, 219, 116, 0.1)', borderRadius: '2px' },
    { tag: tags.atom, color: '#AE81FF' },                    // null / true / false

    // ── 注释 ──────────────────────────────────────────────────
    { tag: tags.comment, color: '#75715E', fontStyle: 'italic' },
    { tag: tags.docComment, color: '#75715E', fontStyle: 'italic' },

    // ── 运算符 ──────────────────────────────────────────────────
    { tag: tags.operator, color: '#F92672' },
    { tag: tags.arithmeticOperator, color: '#F92672' },
    { tag: tags.logicOperator, color: '#F92672' },
    { tag: tags.compareOperator, color: '#F92672' },

    // ── 标点 / 分隔符 ──────────────────────────────────────────
    { tag: tags.punctuation, color: 'var(--text-secondary)' },
    { tag: tags.bracket, color: 'var(--text-secondary)' },
    { tag: tags.squareBracket, color: 'var(--text-secondary)' },
    { tag: tags.paren, color: 'var(--text-secondary)' },
    { tag: tags.brace, color: 'var(--text-secondary)' },
    { tag: tags.separator, color: 'var(--text-muted)' },

    // ── HTML / XML ────────────────────────────────────────────
    { tag: tags.attributeName, color: '#A6E22E' },
    { tag: tags.attributeValue, color: '#E6DB74' },
    { tag: tags.tagName, color: '#F92672' },

    // ── 名称空间 / 模块 ──────────────────────────────────────
    { tag: tags.namespace, color: 'var(--text-muted)', fontStyle: 'italic' },

    // ── 预处理 / 元信息 ────────────────────────────────────────
    { tag: tags.meta, color: '#F92672', fontWeight: '600' },
    { tag: tags.processingInstruction, color: 'var(--text-muted)', opacity: '0.6' },

    // ── 标签 / goto ────────────────────────────────────────────
    { tag: tags.labelName, color: 'var(--text-secondary)', fontStyle: 'italic' },

    // ── 转义 / 特殊字符 ──────────────────────────────────────
    { tag: tags.escape, color: '#F92672', fontWeight: '600' },
    { tag: tags.character, color: '#E6DB74' },

    // ── 行内代码 ──────────────────────────────────────────────
    { tag: tags.monospace, background: 'var(--hover-bg)', borderRadius: '3px', padding: '1px 4px', fontFamily: 'Consolas, Monaco, monospace', fontSize: '0.85em' },

    // ── 删除线 ────────────────────────────────────────────────
    { tag: tags.strikethrough, textDecoration: 'line-through' },
]);

/* ======================================================================== */
/* 语言解析器映射表                                                       */
/* ======================================================================== */

/**
 * 文件扩展名 → CM6 语言解析器工厂函数
 *
 * 优先使用 @codemirror/lang-* 原生 Lezer 解析器（精度更高），
 * 其余语言通过 @codemirror/legacy-modes + StreamLanguage 做高亮。
 */
const langMap = {
    // —— 原生解析器（@codemirror/lang-*） ——
    '.md':      () => markdown(),
    '.js':      () => javascript(),
    '.mjs':     () => javascript(),
    '.cjs':     () => javascript(),
    '.jsx':     () => javascript({ jsx: true }),
    '.ts':      () => javascript({ typescript: true }),
    '.tsx':     () => javascript({ typescript: true, jsx: true }),
    '.css':     () => css(),
    '.scss':    () => css(),
    '.less':    () => css(),
    '.html':    () => html(),
    '.htm':     () => html(),
    '.json':    () => json(),
    '.jsonc':   () => json(),
    '.py':      () => python(),

    // —— 兜底解析器（@codemirror/legacy-modes + StreamLanguage） ——
    '.go':      () => StreamLanguage.define(go),
    '.sql':     () => StreamLanguage.define(sql),
    '.sh':      () => StreamLanguage.define(shell),
    '.bash':    () => StreamLanguage.define(shell),
    '.zsh':     () => StreamLanguage.define(shell),
    '.yaml':    () => StreamLanguage.define(yaml),
    '.yml':     () => StreamLanguage.define(yaml),
    '.xml':     () => StreamLanguage.define(xml),
    '.svg':     () => StreamLanguage.define(xml),
    '.rs':      () => StreamLanguage.define(rust),
    '.rb':      () => StreamLanguage.define(ruby),
    '.swift':   () => StreamLanguage.define(swift),
    '.pl':      () => StreamLanguage.define(perl),
    '.pm':      () => StreamLanguage.define(perl),
    '.lua':     () => StreamLanguage.define(lua),
    '.hs':      () => StreamLanguage.define(haskell),
    '.ps1':     () => StreamLanguage.define(powerShell),
    '.dockerfile': () => StreamLanguage.define(dockerFile),
    '.c':       () => StreamLanguage.define(cpp),
    '.h':       () => StreamLanguage.define(cpp),
    '.cpp':     () => StreamLanguage.define(cpp),
    '.cxx':     () => StreamLanguage.define(cpp),
    '.hpp':     () => StreamLanguage.define(cpp),
    '.java':    () => StreamLanguage.define(java),
    '.cs':      () => StreamLanguage.define(csharp),
    '.scala':   () => StreamLanguage.define(scala),
    '.dart':    () => StreamLanguage.define(dart),
};

/* ======================================================================== */
/* 工厂函数                                                               */
/* ======================================================================== */

/**
 * 根据文件扩展名获取 CM6 语法高亮扩展数组
 *
 * @param {string} fileExt - 文件扩展名（如 '.js', '.py', '.md'），包含前导点号
 * @returns {Array} CM6 扩展数组，无匹配时返回空数组（纯文本，无高亮）
 */
export function getHighlightExtension(fileExt) {
    const langFn = langMap[fileExt];
    if (!langFn) return [];

    // .md 使用 MD 专属配色
    if (fileExt === '.md') {
        return [langFn(), syntaxHighlighting(mdHighlightStyle)];
    }

    // 其他语言使用代码通用配色
    return [langFn(), syntaxHighlighting(codeHighlightStyle)];
}