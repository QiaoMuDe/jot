/**
 * HTML 格式化/压缩工具模块
 * 基于 DOMParser 递归遍历节点，零外部依赖。
 * 注意：HTML 标签名不区分大小写，DOMParser 会统一转为小写。
 */

/** HTML 自闭合（void）元素集合 */
const VOID_ELEMENTS = new Set([
    'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
    'link', 'meta', 'param', 'source', 'track', 'wbr'
]);

/** 块级元素集合（用于决定是否换行缩进） */
const BLOCK_ELEMENTS = new Set([
    'html', 'head', 'body', 'div', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'ul', 'ol', 'li', 'dl', 'dt', 'dd', 'table', 'thead', 'tbody', 'tfoot',
    'tr', 'th', 'td', 'form', 'section', 'article', 'nav', 'aside', 'header',
    'footer', 'main', 'figure', 'figcaption', 'details', 'summary', 'fieldset',
    'legend', 'blockquote', 'pre', 'hr', 'br', 'select', 'option', 'optgroup'
]);

/**
 * 判断输入是否为完整 HTML 文档
 * @param {string} text
 * @returns {boolean}
 */
function isFullDocument(text) {
    const trimmed = text.trimStart();
    return trimmed.startsWith('<!') || trimmed.toLowerCase().startsWith('<html');
}

/**
 * 格式化 HTML：带 2 空格缩进的美化输出
 * @param {string} text - 源 HTML 文本
 * @returns {string} 格式化后的 HTML
 * @throws {Error} 解析失败时抛出
 */
function format(text) {
    const parser = new DOMParser();
    const doc = parser.parseFromString(text, 'text/html');

    if (isFullDocument(text)) {
        return formatNode(doc.documentElement, 0);
    } else {
        // 片段模式：只取 body 的子节点
        const body = doc.body;
        let result = '';
        for (const child of body.childNodes) {
            if (child.nodeType === 1) {
                result += formatNode(child, 0);
            } else if (child.nodeType === 3 && child.textContent.trim()) {
                result += child.textContent.trim() + '\n';
            }
        }
        return result;
    }
}

/**
 * 递归格式化单个 HTML 节点
 * @param {Element} node - DOM 元素节点
 * @param {number} indent - 当前缩进层级
 * @returns {string} 格式化后的字符串
 */
function formatNode(node, indent) {
    const tagName = node.tagName.toLowerCase();
    const indentStr = '  '.repeat(indent);
    const childIndent = '  '.repeat(indent + 1);
    let result = '';

    // 开始标签 + 属性
    result += `${indentStr}<${tagName}`;
    for (const attr of node.attributes) {
        const val = attr.value.replace(/"/g, '&quot;');
        result += ` ${attr.name}="${val}"`;
    }

    if (VOID_ELEMENTS.has(tagName)) {
        result += '>\n';
        return result;
    }

    // 收集子节点
    const children = [];
    let hasBlockChild = false;
    for (const child of node.childNodes) {
        if (child.nodeType === 1) {
            children.push(child);
            const childTag = child.tagName.toLowerCase();
            if (BLOCK_ELEMENTS.has(childTag) || VOID_ELEMENTS.has(childTag)) {
                hasBlockChild = true;
            }
        } else if (child.nodeType === 3 && child.textContent.trim()) {
            children.push(child);
        }
    }

    if (children.length === 0) {
        result += `>\n${indentStr}</${tagName}>\n`;
    } else if (!hasBlockChild && children.every(c => c.nodeType === 3)) {
        // 纯文本内容：保持单行
        result += '>';
        for (const child of children) {
            if (child.nodeType === 3) {
                result += child.textContent.trim();
            }
        }
        result += `</${tagName}>\n`;
    } else {
        result += '>\n';
        for (const child of children) {
            if (child.nodeType === 1) {
                result += formatNode(child, indent + 1);
            } else if (child.nodeType === 3) {
                result += `${childIndent}${child.textContent.trim()}\n`;
            }
        }
        result += `${indentStr}</${tagName}>\n`;
    }

    return result;
}

/**
 * 压缩 HTML：去掉多余空白文本节点
 * @param {string} text - 源 HTML 文本
 * @returns {string} 压缩后的 HTML
 * @throws {Error} 解析失败时抛出
 */
function minify(text) {
    const parser = new DOMParser();
    const doc = parser.parseFromString(text, 'text/html');

    if (isFullDocument(text)) {
        return serializeNode(doc.documentElement);
    } else {
        const body = doc.body;
        let result = '';
        for (const child of body.childNodes) {
            if (child.nodeType === 1) {
                result += serializeNode(child);
            } else if (child.nodeType === 3 && child.textContent.trim()) {
                result += child.textContent.trim();
            }
        }
        return result;
    }
}

/**
 * 递归序列化 HTML 节点（紧凑形式）
 * @param {Element} node - DOM 元素节点
 * @returns {string} 序列化后的字符串
 */
function serializeNode(node) {
    const tagName = node.tagName.toLowerCase();
    let result = `<${tagName}`;
    for (const attr of node.attributes) {
        const val = attr.value.replace(/"/g, '&quot;');
        result += ` ${attr.name}="${val}"`;
    }

    if (VOID_ELEMENTS.has(tagName)) {
        result += '>';
        return result;
    }

    const children = [];
    for (const child of node.childNodes) {
        if (child.nodeType === 1) {
            children.push(child);
        } else if (child.nodeType === 3 && child.textContent.trim()) {
            children.push(child);
        }
    }

    if (children.length === 0) {
        result += `></${tagName}>`;
    } else {
        result += '>';
        for (const child of children) {
            if (child.nodeType === 1) {
                result += serializeNode(child);
            } else if (child.nodeType === 3) {
                result += child.textContent.trim();
            }
        }
        result += `</${tagName}>`;
    }

    return result;
}

export { format, minify };