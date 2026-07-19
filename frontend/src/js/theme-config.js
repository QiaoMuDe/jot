/* ===== 主题配置数据 ===== */

/** 系统主题名称 → 中文显示标签映射 */
export const themeLabels = {
    'default': '默认',
    'catppuccin-latte': '暖咖',
    'nord': '北极',
    'gruvbox-light': '旧纸',
    'light': '浅色',
    'one-dark-pro': '暗夜',
    'quiet-light': '静谧',
    'ysgrifennwr': '暖笺',
    'tokyo-night': '夜幕',
    'eye-protection': '护眼',
    'dark': '深色',
    'dracula': '德古拉',
    'alice': '爱丽丝',
    'lightmind': '山林',
};

/** 系统主题 → 推荐代码高亮主题配对映射 */
export const codeHighlightThemePairing = {
    'catppuccin-latte': 'catppuccin-mocha',
    'gruvbox-light': 'github-dark',
    'one-dark-pro': 'one-dark-pro',
    'quiet-light': 'vscode-light-plus',
    'ysgrifennwr': 'github-light',
    'dracula': 'dracula',
    'default': 'monokai-dimmed',
    'nord': 'monokai-dimmed',
    'light': 'github-light',
    'tokyo-night': 'github-dark',
    'dark': 'github-dark',
    'eye-protection': 'github-light',
    'alice': 'github-light',
    'lightmind': 'monokai-dimmed',
};

/** 系统主题 → Mermaid 明暗主题映射（true=暗色, false=亮色） */
export const isDarkTheme = {
    'default': false,
    'catppuccin-latte': false,
    'nord': false,
    'gruvbox-light': false,
    'light': false,
    'one-dark-pro': true,
    'quiet-light': false,
    'ysgrifennwr': false,
    'tokyo-night': true,
    'eye-protection': false,
    'dark': true,
    'dracula': true,
    'alice': false,
    'lightmind': false,
};