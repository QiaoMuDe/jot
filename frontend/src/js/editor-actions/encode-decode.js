/* ===== 编码解码操作项 ===== */

/**
 * 编码解码操作项数组
 * 提供 Base64、URL、HTML 的编码与解码操作，全部使用浏览器原生 API，零依赖。
 * @type {Array<{group: string, subGroup: string, label: string, errorLabel: string, handler: Function}>}
 */
const ENCODE_DECODE_ACTIONS = [
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

export default ENCODE_DECODE_ACTIONS;
