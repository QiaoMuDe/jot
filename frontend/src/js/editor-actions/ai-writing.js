/* ===== AI 写作操作项 ===== */

/**
 * AI 写作操作项，通过流式 AI 对选中文本进行润色/续写/扩写等处理。
 * 每个操作项 type: 'ai'，op 为后端 AITextOperationStream 的 operation 参数，
 * 执行由 editor-actions.js 的 runAIStreamAction 驱动（事件推送 + 实时增量替换）。
 * @type {Array<{type: string, group: string, label: string, op: string}>}
 */
const AI_WRITING_ACTIONS = [
    {
        type: 'ai',
        group: 'AI 写作',
        label: '润色',
        op: 'polish',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '续写',
        op: 'continue',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '扩写',
        op: 'expand',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '缩写',
        op: 'condense',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '校对',
        op: 'proofread',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '改写',
        op: 'rewrite',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '翻译成中文',
        op: 'translate',
    },
    {
        type: 'ai',
        group: 'AI 写作',
        label: '翻译成英文',
        op: 'translate-en',
    },
];

export default AI_WRITING_ACTIONS;
