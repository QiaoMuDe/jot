/**
 * XML 格式化/压缩工具模块
 * 基于 DOMParser 递归遍历节点，零外部依赖。
 */

/**
 * 格式化 XML：带 2 空格缩进的美化输出
 * @param {string} text - 源 XML 文本
 * @returns {string} 格式化后的 XML
 * @throws {Error} 解析失败时抛出
 */
function format(text) {
    const parser = new DOMParser();
    const doc = parser.parseFromString(text, 'text/xml');
    const parseError = doc.querySelector('parsererror');
    if (parseError) throw new Error('不是合法的 XML');

    const root = doc.documentElement;
    return formatNode(root, 0);
}

/**
 * 递归格式化单个 XML 节点
 * @param {Node} node - DOM 节点
 * @param {number} indent - 当前缩进层级
 * @returns {string} 格式化后的字符串
 */
function formatNode(node, indent) {
    const indentStr = '  '.repeat(indent);
    const childIndent = '  '.repeat(indent + 1);
    let result = '';

    // 开始标签 + 属性
    result += `${indentStr}<${node.tagName}`;
    for (const attr of node.attributes) {
        const val = attr.value.replace(/"/g, '&quot;');
        result += ` ${attr.name}="${val}"`;
    }

    // 收集有意义的子节点（元素节点 + 非空文本节点）
    const children = [];
    for (const child of node.childNodes) {
        if (child.nodeType === 1) {
            children.push(child);
        } else if (child.nodeType === 3 && child.textContent.trim()) {
            children.push(child);
        } else if (child.nodeType === 4) {
            // CDATA 节点
            children.push(child);
        }
    }

    if (children.length === 0) {
        result += '/>\n';
    } else {
        result += '>\n';
        for (const child of children) {
            if (child.nodeType === 1) {
                result += formatNode(child, indent + 1);
            } else if (child.nodeType === 3) {
                result += `${childIndent}${child.textContent.trim()}\n`;
            } else if (child.nodeType === 4) {
                result += `${childIndent}<![CDATA[${child.textContent}]]>\n`;
            }
        }
        result += `${indentStr}</${node.tagName}>\n`;
    }

    return result;
}

/**
 * 压缩 XML：去掉多余空白文本节点，保留结构最小形式
 * @param {string} text - 源 XML 文本
 * @returns {string} 压缩后的 XML
 * @throws {Error} 解析失败时抛出
 */
function minify(text) {
    const parser = new DOMParser();
    const doc = parser.parseFromString(text, 'text/xml');
    const parseError = doc.querySelector('parsererror');
    if (parseError) throw new Error('不是合法的 XML');

    const root = doc.documentElement;
    return serializeNode(root);
}

/**
 * 递归序列化 XML 节点（紧凑形式）
 * @param {Node} node - DOM 节点
 * @returns {string} 序列化后的字符串
 */
function serializeNode(node) {
    let result = `<${node.tagName}`;
    for (const attr of node.attributes) {
        const val = attr.value.replace(/"/g, '&quot;');
        result += ` ${attr.name}="${val}"`;
    }

    const children = [];
    for (const child of node.childNodes) {
        if (child.nodeType === 1) {
            children.push(child);
        } else if (child.nodeType === 3 && child.textContent.trim()) {
            children.push(child);
        } else if (child.nodeType === 4) {
            children.push(child);
        }
    }

    if (children.length === 0) {
        result += '/>';
    } else {
        result += '>';
        for (const child of children) {
            if (child.nodeType === 1) {
                result += serializeNode(child);
            } else if (child.nodeType === 3) {
                result += child.textContent.trim();
            } else if (child.nodeType === 4) {
                result += `<![CDATA[${child.textContent}]]>`;
            }
        }
        result += `</${node.tagName}>`;
    }

    return result;
}

export { format, minify };