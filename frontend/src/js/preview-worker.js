/**
 * 预览渲染 Web Worker
 * 在后台线程执行 marked.parse() 离线程解析，不阻塞主线程 UI
 * 同时提取 Markdown 标题用于 TOC 侧栏
 */
import { marked } from 'marked';

// 与主线程一致的 marked 选项
marked.setOptions({
    breaks: true,
    gfm: true,
});

/**
 * 从 Markdown 内容中提取标题
 * @param {string} content - Markdown 原始文本
 * @returns {Array<{depth:number, text:string}>}
 */
function extractHeadings(content) {
    try {
        const tokens = marked.Lexer.lex(content);
        const headings = [];
        tokens.forEach(function walk(token) {
            if (token.type === 'heading') {
                headings.push({ depth: token.depth, text: token.text });
            }
            if (token.tokens && token.tokens.length) {
                token.tokens.forEach(walk);
            }
        });
        return headings;
    } catch {
        return [];
    }
}

// 监听主线程消息
self.onmessage = function (e) {
    const content = e.data;
    try {
        const html = marked.parse(content);
        const headings = extractHeadings(content);
        self.postMessage({ html, headings });
    } catch (err) {
        self.postMessage({ error: err.message });
    }
};
