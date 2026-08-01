/* ===== 格式化操作项 ===== */

import * as beautify from 'js-beautify';
import * as yaml from 'js-yaml';
import * as smolToml from 'smol-toml';
import { format as sqlFormat } from 'sql-formatter';
import { format as csvFormat } from '../formatters/csv-formatter.js';
import { format as htmlFormat, minify as htmlMinify } from '../formatters/html-formatter.js';
import { format as xmlFormat, minify as xmlMinify } from '../formatters/xml-formatter.js';

/**
 * 格式化操作项数组
 * 包含 JSON/XML/HTML/CSS/JS/SQL/CSV/YAML/TOML 共 9 类格式的格式化与压缩操作。
 * @type {Array<{group: string, subGroup: string, label: string, errorLabel: string, handler: Function}>}
 */
const FORMAT_ACTIONS = [
    // ── JSON ──
    {
        group: '格式化',
        subGroup: 'JSON',
        label: 'JSON 格式化',
        errorLabel: 'JSON',
        handler(text) {
            const parsed = JSON.parse(text);
            return JSON.stringify(parsed, null, 2);
        }
    },
    {
        group: '格式化',
        subGroup: 'JSON',
        label: 'JSON 压缩',
        errorLabel: 'JSON',
        handler(text) {
            const parsed = JSON.parse(text);
            return JSON.stringify(parsed);
        }
    },

    // ── XML ──
    {
        group: '格式化',
        subGroup: 'XML',
        label: 'XML 格式化',
        errorLabel: 'XML',
        handler(text) { return xmlFormat(text); }
    },
    {
        group: '格式化',
        subGroup: 'XML',
        label: 'XML 压缩',
        errorLabel: 'XML',
        handler(text) { return xmlMinify(text); }
    },

    // ── HTML ──
    {
        group: '格式化',
        subGroup: 'HTML',
        label: 'HTML 格式化',
        errorLabel: 'HTML',
        handler(text) { return htmlFormat(text); }
    },
    {
        group: '格式化',
        subGroup: 'HTML',
        label: 'HTML 压缩',
        errorLabel: 'HTML',
        handler(text) { return htmlMinify(text); }
    },

    // ── CSS ──
    {
        group: '格式化',
        subGroup: 'CSS',
        label: 'CSS 格式化',
        errorLabel: 'CSS',
        handler(text) { return beautify.css_beautify(text, { indent_size: 2 }); }
    },
    {
        group: '格式化',
        subGroup: 'CSS',
        label: 'CSS 压缩',
        errorLabel: 'CSS',
        handler(text) { return beautify.css_beautify(text, { indent_size: 0, preserve_newlines: false }); }
    },

    // ── JavaScript ──
    {
        group: '格式化',
        subGroup: 'JavaScript',
        label: 'JS 格式化',
        errorLabel: 'JavaScript',
        handler(text) { return beautify.js_beautify(text, { indent_size: 2 }); }
    },
    {
        group: '格式化',
        subGroup: 'JavaScript',
        label: 'JS 压缩',
        errorLabel: 'JavaScript',
        handler(text) { return beautify.js_beautify(text, { indent_size: 0, preserve_newlines: false }); }
    },

    // ── SQL ──
    {
        group: '格式化',
        subGroup: 'SQL',
        label: 'SQL 格式化',
        errorLabel: 'SQL',
        handler(text) {
            // sql-formatter 默认每个子句关键字后换行，每个列独占一行
            // 后处理压缩：关键字行 + 后续缩进内容 → 合并为一行
            const formatted = sqlFormat(text, { indent: '  ', expressionWidth: 120, keywordCase: 'upper' });
            return compactSQL(formatted);
        }
    },
    {
        group: '格式化',
        subGroup: 'SQL',
        label: 'SQL 压缩',
        errorLabel: 'SQL',
        handler(text) {
            // 压缩：去掉所有多余空白，紧凑单行
            return text
                .replace(/\s*[\r\n]+\s*/g, ' ')
                .replace(/\s{2,}/g, ' ')
                .trim();
        }
    },
    {
        group: '格式化',
        subGroup: 'SQL',
        label: 'SQL 关键字大写',
        errorLabel: 'SQL',
        handler(text) { return convertSQLCase(text, true); }
    },
    {
        group: '格式化',
        subGroup: 'SQL',
        label: 'SQL 关键字小写',
        errorLabel: 'SQL',
        handler(text) { return convertSQLCase(text, false); }
    },

    // ── CSV ──
    {
        group: '格式化',
        subGroup: 'CSV',
        label: 'CSV 格式化',
        errorLabel: 'CSV',
        handler(text) { return csvFormat(text); }
    },

    // ── YAML ──
    {
        group: '格式化',
        subGroup: 'YAML',
        label: 'YAML 格式化',
        errorLabel: 'YAML',
        handler(text) {
            const parsed = yaml.load(text);
            return yaml.dump(parsed, { indent: 2, lineWidth: 120, noRefs: true });
        }
    },
    {
        group: '格式化',
        subGroup: 'YAML',
        label: 'YAML 压缩',
        errorLabel: 'YAML',
        handler(text) {
            const parsed = yaml.load(text);
            // flowLevel: 1 表示仅 1 层以上嵌套使用 flow 风格，保持 YAML 格式但更紧凑
            return yaml.dump(parsed, { indent: 2, lineWidth: -1, flowLevel: 1, noRefs: true });
        }
    },

    // ── TOML ──
    {
        group: '格式化',
        subGroup: 'TOML',
        label: 'TOML 格式化',
        errorLabel: 'TOML',
        handler(text) {
            const parsed = smolToml.parse(text);
            return smolToml.stringify(parsed);
        }
    },
    {
        group: '格式化',
        subGroup: 'TOML',
        label: 'TOML 压缩',
        errorLabel: 'TOML',
        handler(text) {
            const parsed = smolToml.parse(text);
            // 压缩：去掉所有空白行，紧凑输出
            return smolToml.stringify(parsed)
                .replace(/^\s+$/gm, '')
                .replace(/\n{2,}/g, '\n');
        }
    },
];

/**
 * SQL 格式化后处理压缩
 * sql-formatter 默认每个子句关键字后换行，每个列独占一行。
 * 此函数将关键字行与后续缩进内容合并为一行，使输出更紧凑易读。
 * @param {string} formattedSQL - sql-formatter 输出的格式化结果
 * @returns {string} 压缩后的 SQL
 */
function compactSQL(formattedSQL) {
    const lines = formattedSQL.split('\n');
    const result = [];
    let i = 0;

    while (i < lines.length) {
        const line = lines[i];
        const trimmed = line.trim();

        // 跳过首尾空行
        if (!trimmed) { i++; continue; }

        // 判断是否为子句关键字行（非缩进行）
        // sql-formatter 输出中：关键字行在行首（无缩进），内容行缩进
        const isKeyword = !line.startsWith(' ') && !line.startsWith('\t');

        if (isKeyword) {
            // 关键字行以 ( 结尾（如 CREATE TABLE users (），说明是括号块
            // 保持原始格式，不压缩列定义
            if (trimmed.endsWith('(')) {
                // 输出关键字行本身
                result.push(line);
                i++;
                // 原样输出后续缩进内容，直到下一个非缩进行
                while (i < lines.length) {
                    const nextLine = lines[i];
                    if (!nextLine.startsWith(' ') && !nextLine.startsWith('\t')) break;
                    result.push(nextLine);
                    i++;
                }
            } else {
                const parts = [trimmed];
                i++;
                // 收集后续所有缩进行（子句内容）
                while (i < lines.length) {
                    const nextLine = lines[i];
                    const nextTrimmed = nextLine.trim();
                    if (!nextTrimmed || !nextLine.startsWith(' ')) break;
                    parts.push(nextTrimmed);
                    i++;
                }
                result.push(parts.join(' '));
            }
        } else {
            result.push(line);
            i++;
        }
    }

    return result.join('\n');
}

/**
 * SQL 关键字大小写转换
 * 用占位符替换字符串/标识符后，再对关键字做 \b 边界匹配替换，
 * 避免误伤引号内的内容。不改变任何格式（缩进、换行、空白）。
 * @param {string} sql - 源 SQL 文本
 * @param {boolean} toUpper - true 转大写，false 转小写
 * @returns {string} 转换后的 SQL
 */
function convertSQLCase(sql, toUpper) {
    // 常见 SQL 关键字列表
    const keywords = [
        'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'IS', 'NULL', 'AS',
        'ON', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'CROSS', 'FULL', 'NATURAL',
        'ORDER', 'BY', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET', 'SET', 'INSERT', 'INTO',
        'VALUES', 'UPDATE', 'DELETE', 'CREATE', 'TABLE', 'ALTER', 'DROP', 'INDEX',
        'VIEW', 'DISTINCT', 'ALL', 'UNION', 'EXCEPT', 'INTERSECT', 'EXISTS',
        'BETWEEN', 'LIKE', 'ILIKE', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
        'ASC', 'DESC', 'WITH', 'RECURSIVE', 'RETURNING', 'PRIMARY', 'KEY',
        'FOREIGN', 'REFERENCES', 'CONSTRAINT', 'UNIQUE', 'DEFAULT', 'CHECK',
        'AUTO_INCREMENT', 'SERIAL', 'INTEGER', 'INT', 'VARCHAR', 'CHAR', 'TEXT',
        'BOOLEAN', 'FLOAT', 'DOUBLE', 'DECIMAL', 'DATE', 'DATETIME', 'TIMESTAMP',
        'TIME', 'TRUE', 'FALSE', 'IF', 'FOR', 'DO', 'FUNCTION', 'PROCEDURE',
        'TRIGGER', 'BEGIN', 'COMMIT', 'ROLLBACK', 'SAVEPOINT', 'GRANT', 'REVOKE',
        'CASCADE', 'RESTRICT', 'OF', 'SOME', 'ANY', 'EACH', 'ROW', 'ROWS',
        'RANGE', 'UNBOUNDED', 'PRECEDING', 'FOLLOWING', 'CURRENT', 'SESSION',
        'USER', 'USING', 'NATURAL', 'OUTER', 'INNER', 'CROSS', 'FULL',
    ];

    // 用占位符替换字符串和标识符，避免误替换
    const placeholders = [];
    let processed = sql
        .replace(/'[^']*'/g, m => { placeholders.push(m); return `\x00SQL_STR${placeholders.length - 1}\x00`; })
        .replace(/"[^"]*"/g, m => { placeholders.push(m); return `\x00SQL_STR${placeholders.length - 1}\x00`; })
        .replace(/`[^`]*`/g, m => { placeholders.push(m); return `\x00SQL_STR${placeholders.length - 1}\x00`; });

    // 关键字匹配替换（\b 边界确保只匹配完整单词）
    const regex = new RegExp(`\\b(${keywords.join('|')})\\b`, 'gi');
    processed = processed.replace(regex, m => toUpper ? m.toUpperCase() : m.toLowerCase());

    // 恢复占位符
    processed = processed.replace(/\x00SQL_STR(\d+)\x00/g, (_, idx) => placeholders[+idx]);

    return processed;
}

export default FORMAT_ACTIONS;
