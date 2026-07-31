/**
 * CSV 格式化工具模块
 * 纯字符串处理，按列对齐输出，零外部依赖。
 */

/**
 * 格式化 CSV：按列对齐，输出表格状文本
 * @param {string} text - 源 CSV 文本
 * @returns {string} 格式化后的 CSV
 * @throws {Error} 内容为空时抛出
 */
function format(text) {
    const trimmed = text.trim();
    if (!trimmed) throw new Error('不是合法的 CSV');

    const lines = trimmed.split('\n');
    // 解析每行：按逗号拆分并 trim
    const rows = lines.map(line => {
        // 简单 CSV 解析：支持引号包裹的字段（含逗号）
        return parseCSVLine(line);
    });

    if (rows.length === 0) throw new Error('不是合法的 CSV');

    // 计算每列最大宽度（取前 50 行作为参考，避免大文件性能问题）
    const maxCols = Math.max(...rows.map(r => r.length));
    const colWidths = [];
    for (let col = 0; col < maxCols; col++) {
        let maxWidth = 0;
        for (let rowIdx = 0; rowIdx < Math.min(rows.length, 50); rowIdx++) {
            const val = rows[rowIdx][col] || '';
            maxWidth = Math.max(maxWidth, val.length);
        }
        colWidths.push(maxWidth);
    }

    // 格式化输出：每列按最大宽度 padEnd 对齐，列间用 2 空格分隔
    return rows.map(row => {
        return row.map((cell, i) => {
            if (i < colWidths.length) {
                return cell.padEnd(colWidths[i]);
            }
            return cell;
        }).join('  ');
    }).join('\n');
}

/**
 * 简单 CSV 行解析（支持引号包裹字段）
 * @param {string} line
 * @returns {string[]}
 */
function parseCSVLine(line) {
    const fields = [];
    let current = '';
    let inQuotes = false;

    for (let i = 0; i < line.length; i++) {
        const ch = line[i];
        if (ch === '"') {
            if (inQuotes && i + 1 < line.length && line[i + 1] === '"') {
                // 转义引号
                current += '"';
                i++;
            } else {
                inQuotes = !inQuotes;
            }
        } else if (ch === ',' && !inQuotes) {
            fields.push(current.trim());
            current = '';
        } else {
            current += ch;
        }
    }
    fields.push(current.trim());

    // 去掉首尾引号
    return fields.map(f => {
        if (f.startsWith('"') && f.endsWith('"') && f.length >= 2) {
            return f.slice(1, -1);
        }
        return f;
    });
}

/**
 * CSV 没有压缩操作，此函数仅用于保持接口一致
 */
function minify(text) {
    // CSV 压缩无意义，直接返回格式化结果
    return format(text);
}

export { format, minify };