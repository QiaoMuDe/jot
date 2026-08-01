/* ===== MD 语法插入操作项 ===== */

/**
 * MD 语法操作项数组
 * 每个操作项包含 type: 'insert' 标记，用于编辑器执行引擎区分插入模式与变换模式。
 * 有选中文本时包裹/转换选中内容，无选中文本时在光标处插入语法模板。
 * @type {Array<{type: string, group: string, subGroup: string, label: string, errorLabel: string, handler: Function}>}
 */
const MD_SYNTAX_ACTIONS = [
    // ── 行内样式 ──
    /**
     * 粗体：有选中时包裹为 **选中文本**，无选中时插入 **粗体文本**
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '行内样式',
        label: '粗体',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `**${text}**` : '**粗体文本**';
        }
    },
    /**
     * 斜体：有选中时包裹为 *选中文本*，无选中时插入 *斜体文本*
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '行内样式',
        label: '斜体',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `*${text}*` : '*斜体文本*';
        }
    },
    /**
     * 删除线：有选中时包裹为 ~~选中文本~~，无选中时插入 ~~删除线~~
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '行内样式',
        label: '删除线',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `~~${text}~~` : '~~删除线~~';
        }
    },
    /**
     * 行内代码：有选中时包裹为 `选中文本`，无选中时插入 `代码`
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '行内样式',
        label: '行内代码',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `\`${text}\`` : '`代码`';
        }
    },

    // ── 标题 ──
    /**
     * H1：有选中时每行前加 # ，无选中时插入 # 标题
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '标题',
        label: 'H1',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '# 标题';
            return text.split('\n').map(l => `# ${l}`).join('\n');
        }
    },
    /**
     * H2：有选中时每行前加 ## ，无选中时插入 ## 标题
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '标题',
        label: 'H2',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '## 标题';
            return text.split('\n').map(l => `## ${l}`).join('\n');
        }
    },
    /**
     * H3：有选中时每行前加 ### ，无选中时插入 ### 标题
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '标题',
        label: 'H3',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '### 标题';
            return text.split('\n').map(l => `### ${l}`).join('\n');
        }
    },
    /**
     * H4：有选中时每行前加 #### ，无选中时插入 #### 标题
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '标题',
        label: 'H4',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '#### 标题';
            return text.split('\n').map(l => `#### ${l}`).join('\n');
        }
    },
    /**
     * H5：有选中时每行前加 ##### ，无选中时插入 ##### 标题
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '标题',
        label: 'H5',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '##### 标题';
            return text.split('\n').map(l => `##### ${l}`).join('\n');
        }
    },
    /**
     * H6：有选中时每行前加 ###### ，无选中时插入 ###### 标题
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '标题',
        label: 'H6',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '###### 标题';
            return text.split('\n').map(l => `###### ${l}`).join('\n');
        }
    },

    // ── 列表 ──
    /**
     * 无序列表：有选中时每行前加 - ，无选中时插入 - 列表项
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '列表',
        label: '无序列表',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '- 列表项';
            return text.split('\n').map(l => `- ${l}`).join('\n');
        }
    },
    /**
     * 有序列表：有选中时每行前加 1. ，无选中时插入 1. 列表项
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '列表',
        label: '有序列表',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '1. 列表项';
            return text.split('\n').map(l => `1. ${l}`).join('\n');
        }
    },
    /**
     * 任务列表：有选中时每行前加 - [ ] ，无选中时插入 - [ ] 待办事项
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '列表',
        label: '任务列表',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '- [ ] 待办事项';
            return text.split('\n').map(l => `- [ ] ${l}`).join('\n');
        }
    },

    // ── 块元素 ──
    /**
     * 代码块：有选中时包裹在三反引号中，无选中时插入代码块模板
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '块元素',
        label: '代码块',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '```\n语言\n```';
            return '```\n' + text + '\n```';
        }
    },
    /**
     * 引用：有选中时每行前加 > ，无选中时插入 > 引用内容
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '块元素',
        label: '引用',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '> 引用内容';
            return text.split('\n').map(l => `> ${l}`).join('\n');
        }
    },
    /**
     * 分割线：无选中时插入 ---，有选中时忽略选中内容直接插入
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '块元素',
        label: '分割线',
        errorLabel: 'MD 语法',
        handler(text) {
            return '---';
        }
    },
    /**
     * 折叠详情：无选中时插入 details 折叠块模板，有选中时包裹选中内容
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '块元素',
        label: '折叠详情',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '<details>\n<summary>标题</summary>\n\n内容\n</details>';
            return '<details>\n<summary>标题</summary>\n\n' + text + '\n</details>';
        }
    },

    // ── 链接/媒体 ──
    /**
     * 链接：有选中时包装为 [选中文本](url)，无选中时插入 [链接文本](url)
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '链接/媒体',
        label: '链接',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `[${text}](url)` : '[链接文本](url)';
        }
    },
    /**
     * 图片：无选中时插入 ![替代文本](url)，有选中时以选中文本作为替代文本
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '链接/媒体',
        label: '图片',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `![${text}](url)` : '![替代文本](url)';
        }
    },

    // ── 表格 ──
    /**
     * 表格：无选中时插入完整表格模板（表头、对齐行、数据行），有选中时包裹选中内容
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '表格',
        label: '表格',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '| 表头1 | 表头2 | 表头3 |\n|-------|-------|-------|\n| 内容1 | 内容2 | 内容3 |';
            // 有选中文本时，按行分割并包装为表格行
            const lines = text.split('\n').filter(l => l.trim());
            const rows = lines.map(l => `| ${l.split(/\s+/).join(' | ')} |`);
            return rows.join('\n');
        }
    },

    // ── 数学公式 ──
    /**
     * 行内公式：有选中时包裹为 $选中文本$，无选中时插入 $公式$
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '数学公式',
        label: '行内公式',
        errorLabel: 'MD 语法',
        handler(text) {
            return text ? `$${text}$` : '$公式$';
        }
    },
    /**
     * 块级公式：有选中时包裹在双美元符号中，无选中时插入块级公式模板
     */
    {
        type: 'insert',
        group: 'MD 语法',
        subGroup: '数学公式',
        label: '块级公式',
        errorLabel: 'MD 语法',
        handler(text) {
            if (!text) return '$$\n公式\n$$';
            return '$$\n' + text + '\n$$';
        }
    }
];

export default MD_SYNTAX_ACTIONS;