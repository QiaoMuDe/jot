# 设置页主题预览方案

## 概述

用户在设置页选择系统主题和代码高亮主题时，需要回到笔记首页才能看到实际效果，体验割裂。本方案在两个下拉菜单附近添加实时预览区域，让用户在选择时就能直观看到配色效果。

## 当前状态

* **设置页布局**：两个 `.settings-section` 区域——"主题设置"(line 261)含系统主题下拉菜单，"编辑器选项"(line 306)含代码高亮主题下拉菜单

* **`.font-setting-row`**：flex row 布局（label + 控件），每行一个控件

* **CSS 变量系统**：每个 `[data-theme]` 块定义了约 40 个变量（基础色、语义色、阴影等）

* **CM6 HighlightStyle**：11 套代码高亮主题，均使用 `tags.*` 映射颜色

## 改动方案

### 1. HTML — `index.html`

#### 1.1 系统主题预览区

在"主题设置"section 的 `.font-settings` 内，下拉菜单 `.font-setting-row` 之后添加：

```html
<div class="theme-preview" id="themePreview">
    <div class="theme-palette">
        <div class="palette-swatch" style="--swatch: var(--bg)"><span>背景</span></div>
        <div class="palette-swatch" style="--swatch: var(--card-bg)"><span>卡片</span></div>
        <div class="palette-swatch" style="--swatch: var(--text-primary)"><span>文字</span></div>
        <div class="palette-swatch" style="--swatch: var(--text-secondary)"><span>次要文字</span></div>
        <div class="palette-swatch" style="--swatch: var(--accent)"><span>强调色</span></div>
        <div class="palette-swatch" style="--swatch: var(--border)"><span>边框</span></div>
        <div class="palette-swatch" style="--swatch: var(--hover-bg)"><span>悬停</span></div>
        <div class="palette-swatch" style="--swatch: var(--success)"><span>成功</span></div>
        <div class="palette-swatch" style="--swatch: var(--warning)"><span>警告</span></div>
        <div class="palette-swatch" style="--swatch: var(--error)"><span>错误</span></div>
    </div>
</div>
```

说明：每个 `.palette-swatch` 通过 `--swatch: var(--color-var)` 引用当前主题的 CSS 变量。切换主题时变量值自动变化，预览区即时更新，无需额外 JS。

#### 1.2 代码高亮预览区

在"编辑器选项"section 内，代码高亮下拉菜单 `.font-setting-row` 之后添加：

```html
<div class="code-preview" id="codePreview">
    <pre class="code-sample"><code>function hello() {
    // 这是一段预览代码
    const msg = "Hello, World!";
    console.log(msg);
    return 42;
}</code></pre>
</div>
```

说明：创建一个只读的微型 CodeMirror 6 实例，使用 `.javascript` 语言 + 当前选中的代码高亮主题 `HighlightStyle`。CM6 已在项目中使用，可复用 `getHighlightExtension()` 工厂函数。

### 2. CSS — `settings-panel.css`

新增样式块：

#### 系统主题预览样式

```css
.theme-preview {
    margin-top: 12px;
    padding: 12px;
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    transition: background 0.15s, border-color 0.15s;
}

.theme-palette {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.palette-swatch {
    width: 48px;
    height: 48px;
    border-radius: var(--radius-sm);
    background: var(--swatch);
    border: 1px solid var(--border);
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding-bottom: 4px;
    position: relative;
    transition: background 0.15s;
}

.palette-swatch span {
    font-size: 0.6rem;
    color: var(--text-primary);
    opacity: 0.7;
    background: var(--card-bg);
    padding: 1px 4px;
    border-radius: 2px;
    white-space: nowrap;
}
```

#### 代码高亮预览样式

```css
.code-preview {
    margin-top: 12px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    overflow: hidden;
    transition: border-color 0.15s;
}

.code-preview .cm-editor {
    height: auto;
    max-height: 120px;
}

.code-preview .cm-editor .cm-scroller {
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 12px;
    padding: 8px;
}

.code-preview .cm-editor.cm-focused {
    outline: none;
}
```

### 3. JS — `main.js`

#### 3.1 顶层导入

在已有 `@codemirror/view` 等导入行下方，新增 javascript 语言导入（用于代码预览的语法高亮）：

```js
import { javascript } from '@codemirror/lang-javascript';
```

（如果已有该导入行则跳过）

#### 3.2 新增 `initCodePreview()` 函数

在 `initCodeHighlightThemeSettings()` 之后新增：

```js
let _codePreviewInited = false;
let _codePreviewEditor = null;

function initCodePreview() {
    if (_codePreviewInited) return;
    _codePreviewInited = true;

    const container = document.getElementById('codePreview');
    if (!container) return;
    
    buildCodePreview(container, codeHighlightTheme);
}

function buildCodePreview(container, themeName) {
    // 销毁旧实例
    if (_codePreviewEditor) {
        _codePreviewEditor.destroy();
        _codePreviewEditor = null;
    }

    const previewCode = `function fibonacci(n) {\n  if (n <= 1) return n;\n  return fibonacci(n - 1) + fibonacci(n - 2);\n}\n\nlet result = fibonacci(10);\nconsole.log("Result:", result);`;

    const extensions = [
        javascript(),
        EditorView.editable.of(false),
        EditorView.theme({
            '&': { height: 'auto', maxHeight: '120px', backgroundColor: 'transparent' },
            '.cm-scroller': { overflow: 'hidden', fontFamily: 'Consolas, Monaco, monospace', fontSize: '12px' },
            '.cm-gutters': { display: 'none' },
            '.cm-line': { padding: '0 8px' },
            '.cm-editor': { outline: 'none' },
            '&.cm-focused': { outline: 'none' },
        }),
    ];

    const highlightExt = getHighlightExtension('.js', themeName);
    if (highlightExt.length > 0) extensions.push(...highlightExt);

    const state = EditorState.create({
        doc: previewCode,
        extensions,
    });

    _codePreviewEditor = new EditorView({
        state,
        parent: container,
    });
}
```

#### 3.3 切换主题时同步更新

在 `applyCodeHighlightTheme()` 的 `applyCodeHighlightThemeUI(themeName)` 调用之后追加：

```js
// 同步更新设置页代码预览
if (_codePreviewInited) {
    const container = document.getElementById('codePreview');
    if (container) {
        buildCodePreview(container, themeName);
    }
}
```

#### 3.4 调用入口

在 `initThemeSettings()` 调用之后（或附近）追加：

```js
initCodePreview();
```

#### 3.5 系统主题切换时同步预览（无需额外 JS）

系统主题预览区使用 CSS `var()` 引用，切换 `data-theme` 属性后 CSS 变量自动变化，预览色块即时更新，零 JS 开销。

### 4. 文件改动清单

| 文件                                               | 改动                                                                                        |
| ------------------------------------------------ | ----------------------------------------------------------------------------------------- |
| `frontend/index.html`                            | 新增 `#themePreview`（色板）+ `#codePreview`（代码预览容器）                                            |
| `frontend/src/css/components/settings-panel.css` | 新增 `.theme-preview` / `.palette-swatch` / `.code-preview` / `.code-preview .cm-editor` 样式 |
| `frontend/src/js/main.js`                        | 新增 `initCodePreview()`（创建微型 CM6 实例）+ 修改 `applyCodeHighlightTheme()` 切换时同步更新预览             |

### 5. 假设与决策

* **系统主题预览**：使用 CSS 变量色板（纯 CSS 方案），不依赖 JS，主题切换时自动变化

* **代码高亮预览**：使用微型 CM6 实例（`.javascript` 语言），高度限制 120px，只读模式，跟随当前选择的代码高亮主题

* **预览代码内容**：选择一段能展示关键语法节点（关键字、字符串、数字、函数、注释）的 JavaScript 片段

* **CM6 实例管理**：切换主题时销毁旧实例、创建新实例（与主编辑器行为一致）

### 6. 验证步骤

1. `npx vite build` 通过，零错误
2. 打开设置页，看到系统主题下方显示 10 个小色块（背景、卡片、文字、强调色等）
3. 切换系统主题下拉菜单，色块颜色自动变化
4. 代码高亮主题下方显示微型代码编辑器预览
5. 切换代码高亮主题下拉菜单，预览代码语法高亮即时更新
6. 预览区在窗口缩放时布局正常

