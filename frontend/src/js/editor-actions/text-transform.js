/* ===== 文本转换操作项 ===== */

/**
 * 文本转换操作项数组
 * 提供大小写转换、命名风格转换、行/字符反转等纯字符串操作。
 * @type {Array<{group: string, label: string, errorLabel: string, handler: Function}>}
 */
const TEXT_TRANSFORM_ACTIONS = [
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
            // 先按大写字母拆解（如 helloWorld → hello World），再处理分隔符
            return text
                .replace(/([A-Z])/g, ' $1')
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
];

export default TEXT_TRANSFORM_ACTIONS;
