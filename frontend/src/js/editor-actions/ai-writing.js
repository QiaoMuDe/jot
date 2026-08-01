/* ===== AI 写作操作项 ===== */

/**
 * AI 写作操作项，通过 AI 对选中文本进行润色/续写/扩写等处理。
 * 每个操作项 type: 'ai'，handler 为异步函数，调用后端 AITextOperation 绑定。
 * @type {Array<{type: string, group: string, label: string, errorLabel: string, handler: Function}>}
 */
const AI_WRITING_ACTIONS = [
    {
        type: 'ai',
        group: 'AI 写作',
        label: '润色',
        errorLabel: 'AI 操作',
        /**
         * 润色文本，改进语法、表达和风格
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 润色后的文本
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'polish');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '续写',
        errorLabel: 'AI 操作',
        /**
         * 根据选中文本的内容和风格续写
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 续写内容
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'continue');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '扩写',
        errorLabel: 'AI 操作',
        /**
         * 扩写文本，增加更多细节和例子
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 扩写后的文本
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'expand');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '缩写',
        errorLabel: 'AI 操作',
        /**
         * 缩写文本，保留关键信息，精简文字
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 缩写后的文本
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'condense');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '校对',
        errorLabel: 'AI 操作',
        /**
         * 校对文本，修正语法错误、拼写错误和标点符号
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 校对后的文本
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'proofread');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '改写',
        errorLabel: 'AI 操作',
        /**
         * 改写文本，保持原意不变，改变表达方式
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 改写后的文本
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'rewrite');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '翻译成中文',
        errorLabel: 'AI 操作',
        /**
         * 将选中文本翻译为中文
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 中文翻译
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'translate');
        }
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '翻译成英文',
        errorLabel: 'AI 操作',
        /**
         * 将选中文本翻译为英文
         * @param {string} text - 选中文本
         * @returns {Promise<string>} 英文翻译
         */
        async handler(text) {
            return await window.go.main.App.AITextOperation(text, 'translate-en');
        }
    },
];

export default AI_WRITING_ACTIONS;