/* ===== highlight.js 主题动态加载 ===== */
/**
 * 根据 CM6 主题名 → hljs 文件名映射，动态注入 <style id="ai-hljs-theme"> 标签，
 * 使 AI 消息代码块高亮跟随设置页的代码高亮主题。
 */

import { hljsCssData } from './hljs-themes-data.js';

// CM6 主题名 → hljs CSS 文件名字典
const hljsFileMap = {
    'monokai-dimmed':    'monokai-sublime',
    'vscode-dark-plus':  'atom-one-dark-reasonable',
    'vscode-light-plus': 'atom-one-light',
    'one-dark-pro':      'atom-one-dark',
    'github-dark':       'github-dark',
    'catppuccin-mocha':  'monokai-sublime',
    'gruvbox-dark':      'gruvbox-dark',
    'dracula':           'dracula',
    'ayu-mirage':        'monokai-sublime',
    'material-palenight':'atom-one-dark',
    'github-light':      'github',
    'one-light':         'atom-one-light',
    'catppuccin-latte':  'github',
};

/**
 * 应用 AI 聊天代码块的高亮主题
 * @param {string} themeName - CM6 主题名（如 'monokai-dimmed'），未知值回退到 github
 */
export function applyAIHighlightTheme(themeName) {
    const hljsFileName = hljsFileMap[themeName] || 'github';
    const css = hljsCssData[hljsFileName];

    let existing = document.getElementById('ai-hljs-theme');
    if (existing) {
        existing.textContent = css;
    } else {
        const style = document.createElement('style');
        style.id = 'ai-hljs-theme';
        style.textContent = css;
        document.head.appendChild(style);
    }
}
