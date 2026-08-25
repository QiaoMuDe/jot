# MCP 导入对话框 JSON 语法高亮（CodeMirror 6）

## 概述

将 MCP 服务器导入对话框中的 `<textarea>` 替换为 CodeMirror 6 编辑器，实现 JSON 语法高亮。项目已依赖 CM6 全家桶，无需新增依赖。

## 涉及文件

| 文件                                               | 改动类型                      |
| ------------------------------------------------ | ------------------------- |
| `frontend/index.html`                            | 修改：textarea → div 容器      |
| `frontend/src/main.js`                           | 修改：CM6 初始化 + 读取内容方式适配     |
| `frontend/src/css/components/settings-panel.css` | 修改：textarea 样式 → CM6 容器样式 |

## 步骤

### 1. HTML：textarea 替换为 div 容器

**文件**: `frontend/index.html` 第 730 行

将：

```html
<textarea id="mcpServerImportInput" class="settings-input mcp-form-textarea mcp-server-import-textarea" rows="10" spellcheck="false" placeholder="在此粘贴 JSON 配置…"></textarea>
```

替换为：

```html
<div id="mcpServerImportInput" class="mcp-server-import-editor"></div>
```

保留 `id="mcpServerImportInput"` 不变，JS 中通过此 id 获取容器。

### 2. JS：创建 CM6 导入编辑器实例

**文件**: `frontend/src/main.js`

#### 2a. 导入 `json` 语言包

在现有 CM6 导入区（第 10-18 行）添加：

```js
import { json } from '@codemirror/lang-json';
```

#### 2b. 声明实例变量（模块级，约第 61 行附近）

```js
let _mcpImportEditor = null;
```

#### 2c. 新增初始化函数

在 `openMCPImportDialog` 函数之前（约第 9526 行）添加：

```js
/**
 * 初始化 MCP 导入对话框的 CM6 JSON 编辑器（懒初始化，仅首次打开时创建）
 */
function initMCPImportEditor() {
    const container = document.getElementById('mcpServerImportInput');
    if (!container || _mcpImportEditor) return;

    const extensions = [
        json(),
        getHighlightExtension('.json', codeHighlightTheme),
        EditorView.theme({
            '&': {
                backgroundColor: 'var(--input-bg)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-md)',
                minHeight: '220px',
            },
            '&.cm-focused': {
                borderColor: 'var(--accent)',
                backgroundColor: 'var(--card-bg)',
            },
            '.cm-scroller': {
                fontFamily: 'var(--font-mono)',
                fontSize: '0.8rem',
                lineHeight: '1.55',
                overflow: 'auto',
            },
            '.cm-content': {
                fontFamily: 'var(--font-mono)',
                padding: '8px 12px',
            },
            '.cm-gutters': { display: 'none' },
            '.cm-activeLine': { backgroundColor: 'transparent' },
        }),
        EditorView.lineWrapping,
        placeholder('在此粘贴 JSON 配置…'),
    ];

    const state = EditorState.create({ doc: '', extensions });
    _mcpImportEditor = new EditorView({ state, parent: container });
}
```

关键设计：

* 使用已有的 `getHighlightExtension('.json', codeHighlightTheme)` 跟随全局代码高亮主题

* 复用 `jotTheme` 中的 CSS 变量风格，但用自定义主题覆盖尺寸（对话框内不需要行号、折叠等）

* `placeholder` 来自已导入的 `@codemirror/view`

#### 2d. 修改 `openMCPImportDialog`（约第 9526-9539 行）

```js
function openMCPImportDialog() {
    const dialog = document.getElementById('mcpServerImportDialog');
    if (!dialog) return;
    initMCPImportEditor();          // ← 新增：懒初始化
    dialog.style.display = 'flex';
    requestAnimationFrame(() => dialog.classList.add('visible'));
    setTimeout(() => {
        if (_mcpImportEditor) {
            _mcpImportEditor.focus();
        }
    }, 200);
}
```

焦点从 `input.focus()` 改为 `_mcpImportEditor.focus()`。

#### 2e. 修改 `closeMCPImportDialog`（约第 9541-9555 行）

清空逻辑从 `input.value = ''` 改为：

```js
if (_mcpImportEditor) {
    _mcpImportEditor.dispatch({
        changes: { from: 0, to: _mcpImportEditor.state.doc.length },
    });
}
```

#### 2f. 修改 `handleMCPImport`（约第 9557-9660 行）

读取内容从 `input.value` 改为：

```js
const text = _mcpImportEditor
    ? _mcpImportEditor.state.doc.toString().trim()
    : '';
```

`shakeMCPFormInput(input)` 的参数改为容器 div：

```js
const container = document.getElementById('mcpServerImportInput');
// ... shakeMCPFormInput(container);
```

### 3. CSS：textarea 样式 → CM6 编辑器容器样式

**文件**: `frontend/src/css/components/settings-panel.css`

#### 3a. 删除 `.mcp-server-import-textarea` 相关规则（第 2117-2127 行）

整块删除，因为不再有 textarea。

#### 3b. 删除 `.mcp-server-import-textarea.mcp-input-invalid` 规则（第 2130-2133 行）

抖动动画改由 CM6 容器 `.mcp-server-import-editor` 承接。

#### 3c. 新增 CM6 容器样式

```css
/* MCP 导入 CM6 编辑器容器：承接 settings-input 外框 + mcp-input-invalid 抖动 */
.mcp-server-import-editor {
  width: 100%;
  min-height: 220px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: var(--transition);
}

.mcp-server-import-editor.mcp-input-invalid {
  animation: mcpFormInputError 0.6s ease;
  border-color: var(--danger) !important;
}

/* CM6 编辑器去除默认 outline */
.mcp-server-import-editor .cm-editor {
  outline: none;
}
```

#### 3d. 修改无障碍减弱动画规则（约第 2200 行）

将 `.mcp-server-import-textarea.mcp-input-invalid` 替换为 `.mcp-server-import-editor.mcp-input-invalid`。

### 4. 移除不再需要的 CSS 类

HTML 中 textarea 原有 `settings-input mcp-form-textarea` 类，替换为 div 后不再需要这两个类，已在步骤 1 中处理。

## 验证

1. 打开设置页 → MCP 服务器 → 点击"导入"按钮
2. 确认对话框中出现 CM6 编辑器，有 JSON 语法高亮（字符串绿色、key 颜色等）
3. 粘贴一段 JSON，确认高亮正确
4. 点击"导入"→ 空内容时确认抖动动画正常
5. 粘贴非法 JSON → 确认校验失败时抖动正常
6. 关闭对话框再打开 → 确认内容已清空，placeholder 正常显示
7. 切换代码高亮主题后重新打开 → 确认高亮主题跟随

