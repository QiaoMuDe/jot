// 预设配置品牌徽章：按 API 地址域名离线识别常见 AI 服务商，生成圆角方形色块徽章
// 识别命中 → 品牌配色 + 简称；未命中 → 名称/域名首字符 + 哈希稳定配色
// 纯离线实现，不发起任何网络请求

// 服务商识别表：keywords 匹配 URL 的 host（小写，含端口）；name 为全名（用作悬浮提示）
const BRANDS = [
    { keywords: ['localhost', '127.0.0.1', '0.0.0.0', '11434'], name: 'Ollama', short: 'OL', bg: '#374151' },
    { keywords: ['openai'], name: 'OpenAI', short: 'O', bg: '#10A37F' },
    { keywords: ['deepseek'], name: 'DeepSeek', short: 'DS', bg: '#4D6BFE' },
    { keywords: ['dashscope', 'aliyun'], name: '通义千问 Qwen', short: 'QW', bg: '#615CED' },
    { keywords: ['moonshot'], name: 'Kimi（月之暗面）', short: 'K', bg: '#7C3AED' },
    { keywords: ['bigmodel', 'z.ai'], name: '智谱 GLM', short: 'GLM', bg: '#3859FF' },
    { keywords: ['siliconflow'], name: '硅基流动', short: 'SF', bg: '#2563EB' },
    { keywords: ['generativelanguage'], name: 'Google Gemini', short: 'GM', bg: '#4285F4' },
    { keywords: ['anthropic'], name: 'Anthropic Claude', short: 'CL', bg: '#D97757' },
    { keywords: ['groq'], name: 'Groq', short: 'GQ', bg: '#F55036' },
    { keywords: ['x.ai', 'grok'], name: 'xAI Grok', short: 'XA', bg: '#374151' },
    { keywords: ['mistral'], name: 'Mistral', short: 'MS', bg: '#FF7000' },
    { keywords: ['minimax'], name: 'MiniMax', short: 'MM', bg: '#7B61FF' },
    { keywords: ['volces'], name: '火山方舟（豆包）', short: 'DB', bg: '#3370FF' },
    { keywords: ['stepfun'], name: '阶跃星辰', short: 'SP', bg: '#3B82F6' },
    { keywords: ['xf-yun'], name: '讯飞星火', short: 'XF', bg: '#2479FF' },
    { keywords: ['baidubce'], name: '百度千帆', short: 'BF', bg: '#2932E1' },
    { keywords: ['hunyuan', 'tencent'], name: '腾讯混元', short: 'HY', bg: '#0052D9' },
    { keywords: ['baichuan'], name: '百川智能', short: 'BC', bg: '#3B6CFF' },
    { keywords: ['lingyiwanwu', '01.ai'], name: '零一万物', short: '01', bg: '#0EA5E9' },
    { keywords: ['openrouter'], name: 'OpenRouter', short: 'OR', bg: '#6B4AEA' },
    { keywords: ['gitee.ai'], name: 'Gitee AI', short: 'GE', bg: '#C71D23' },
    { keywords: ['360.cn'], name: '360 智脑', short: '360', bg: '#E60012' },
];

// 兜底色板：中深色调，保证白字/深字均可读
const FALLBACK_PALETTE = [
    '#0EA5E9', '#8B5CF6', '#F59E0B', '#10B981', '#EF4444',
    '#6366F1', '#EC4899', '#14B8A6', '#F97316', '#3B82F6',
];

/**
 * 解析 baseURL 的 host（小写，含端口）；解析失败返回空串
 */
function hostOf(baseURL) {
    if (!baseURL || typeof baseURL !== 'string') return '';
    try {
        return new URL(baseURL).host.toLowerCase();
    } catch {
        return '';
    }
}

/**
 * 按域名识别服务商
 * @param {string} baseURL - API 地址，如 https://api.deepseek.com/v1
 * @returns {{name:string, short:string, bg:string}|null} 命中返回品牌信息，否则 null
 */
export function detectBrand(baseURL) {
    const host = hostOf(baseURL);
    if (!host) return null;
    return BRANDS.find(b => b.keywords.some(k => host.includes(k))) || null;
}

/**
 * 简单 FNV 哈希 → 从兜底色板取稳定颜色
 */
function hashColor(seed) {
    let h = 0;
    for (let i = 0; i < seed.length; i++) {
        h = ((h << 5) - h + seed.charCodeAt(i)) | 0;
    }
    return FALLBACK_PALETTE[Math.abs(h) % FALLBACK_PALETTE.length];
}

/**
 * 取字符串首个可见字符（大写，中文原样）
 */
function firstChar(str) {
    const t = String(str || '').trim();
    if (!t) return '';
    return [...t][0].toUpperCase();
}

/**
 * 取 host 首个非空字符段的首字母（如 api.custom.io → A）
 */
function hostFirstChar(baseURL) {
    const host = hostOf(baseURL);
    const seg = host.split(/[.:]/).find(s => s.trim() !== '');
    return seg ? seg[0].toUpperCase() : '';
}

/**
 * 根据背景色亮度决定前景色（亮底深字，暗底白字）
 */
function contrastText(hex) {
    const n = parseInt(hex.slice(1), 16);
    const r = (n >> 16) & 255;
    const g = (n >> 8) & 255;
    const b = n & 255;
    const lum = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
    return lum > 0.62 ? '#1F2937' : '#FFFFFF';
}

/**
 * 创建预设品牌徽章 DOM（圆角方形色块 + 简称）
 * @param {string} baseURL - 预设 API 地址
 * @param {string} name - 预设名称（未识别服务商时用作首字符来源）
 * @param {boolean} small - 小号徽章（触发按钮用）
 * @returns {HTMLElement} <span class="preset-brand-badge[ sm]">
 */
export function createPresetBadge(baseURL, name, small = false) {
    const badge = document.createElement('span');
    badge.className = 'preset-brand-badge' + (small ? ' sm' : '');

    const brand = detectBrand(baseURL);
    let short;
    let bg;
    if (brand) {
        short = brand.short;
        bg = brand.bg;
        badge.title = brand.name;
    } else {
        short = firstChar(name) || hostFirstChar(baseURL) || '?';
        bg = hashColor(baseURL || name || '');
        badge.title = name || baseURL || '';
    }

    badge.textContent = short;
    badge.style.background = bg;
    badge.style.color = contrastText(bg);
    return badge;
}
