/* ===== 文本清理操作项 ===== */

/**
 * 文本清理操作项数组
 * 提供去空格、去空行、行尾清理、Tab 与空格互转等文本整理操作。
 * @type {Array<{group: string, label: string, errorLabel: string, handler: Function}>}
 */
const TEXT_CLEAN_ACTIONS = [
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
];

export default TEXT_CLEAN_ACTIONS;
