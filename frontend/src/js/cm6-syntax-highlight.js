/**
 * CM6 编辑器 Markdown 语法高亮模块
 * 
 * 封装 CodeMirror 6 编辑器主题和 Markdown 语法高亮样式定义，
 * 所有颜色引用 CSS 变量，跟随应用主题自动变化。
 */
import { EditorView } from '@codemirror/view';
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language';
import { markdown } from '@codemirror/lang-markdown';
import { tags } from '@lezer/highlight';

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

/** Markdown 语法节点 → CSS 样式映射，覆盖 16 种元素 */
const jotHighlightStyle = HighlightStyle.define([
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

/**
 * 预组合的 Markdown 语法高亮扩展数组
 * 包含 markdown() 解析器 + syntaxHighlighting() 样式注入
 */
export const mdHighlight = [markdown(), syntaxHighlighting(jotHighlightStyle)];