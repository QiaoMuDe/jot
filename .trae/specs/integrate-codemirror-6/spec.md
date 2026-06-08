# CodeMirror 6 编辑器集成计划

## Why

当前编辑器基于原生 `<textarea>` + `<input>`，在 Wails WebView2 环境中存在以下核心问题：
- 原生撤销/恢复（Ctrl+Z/Y）失效 → 手写了 ~90 行 `UndoRedoManager`
- 查找/替换功能需手动实现 overlay + TreeWalker 双模式高亮 → ~636 行 `FindReplaceManager`
- Tab 缩进不支持（原生 textarea 会移走焦点）
- 鼠标拖拽选中文本触发编辑器关闭（click vs mousedown 事件差异）
- 光标位置管理复杂（`element.value = xxx` 会丢失光标）
- IME 输入法兼容风险
- 以上补丁代码越打越多，维护成本持续上升

引入 CodeMirror 6 可以一次性解决上述所有问题，同时获得 Markdown 语法高亮、括号配对、自动补全等开箱即用的编辑能力。

## What Changes

### 整体方案矩阵

```
┌─────────────┬──────────────────┬──────────────────┐
│             │    纯文本模式     │     预览模式      │
├─────────────┼──────────────────┼──────────────────┤
│  新建/编辑  │   CodeMirror 6   │   marked.js      │
│  查看       │   CodeMirror 6   │   marked.js      │
└─────────────┴──────────────────┴──────────────────┘
```

- **纯文本模式**：CM6 替换 `<textarea>`（可编辑或只读）
- **预览模式**：保留现有 `marked.js` + `highlight.js` 渲染方案不变

### 具体变更清单

#### 新增依赖（npm）

| 包名 | 版本 | 用途 | 大小(gz) |
|------|------|------|----------|
| `@codemirror/state` | ^6.x | EditorState 核心状态管理 | ~8KB |
| `@codemirror/view` | ^6.x | EditorView DOM 渲染层 + 基础扩展 | ~25KB |
| `@codemirror/commands` | ^6.x | 默认快捷键 + history(撤销恢复) + indentWithTab | ~12KB |
| `@codemirror/search` | ^6.x | 内置查找/替换面板 | ~15KB |
| `@codemirror/lang-markdown` | ^6.x | Markdown 语法解析 + GFM 支持 | ~15KB |
| `@codemirror/language` | ^6.x | 语言支持基础框架（lang-markdown 依赖） | ~10KB |
| `@codemirror/autocomplete` | ^6.x | 自动补全 + closeBrackets(括号配对) | ~18KB |
| `@codemirror/lint` | ^6.x | Lint 检查框架（可选，Phase 2） | ~8KB |
| `lezer/highlight` | ^1.x | 语法高亮样式定义（lang-markdown 依赖） | ~3KB |

**最小可用集总大小：~75KB gzipped**（core + commands + search + lang-markdown + autocomplete）

#### HTML 变更 (`frontend/index.html`)

1. **删除** `<textarea id="editorNoteContent">` (L109)
2. **替换为** `<div id="editorNoteContent"></div>`（CM6 容器）
3. **删除** `<div id="findOverlay">` (L110) — CM6 内置查找不需要 overlay 覆盖层
4. **保留** `#mdRendered` (L111) — 预览模式继续使用
5. **大幅简化** `#editorFindBar` (L115-128)：
   - 删除自定义查找输入框、计数、导航按钮、关闭按钮
   - 删除整个替换行（`#replaceBarRow` 及其子元素）
   - 保留一个极简的容器 div 作为 CM6 search panel 的挂载点（或完全移除，让 CM6 自行管理面板 DOM）

#### JavaScript 变更 (`frontend/src/main.js`)

##### Phase 1: 核心集成（必须完成）

**1. 新增 CM6 初始化模块 (~100 行)**

```javascript
// 新增全局变量
let cmEditor = null;  // EditorView 实例

// 新增函数
function initCodeMirror(container, options) {
    // 创建 EditorState，配置 extensions:
    //   - basicSetup (默认键位映射)
    //   - history() + historyKeymap (撤销/恢复)
    //   - search({top: true}) + defaultSearchKeymap (查找/替换)
    //   - indentWithTab (Tab缩进)
    //   - markdown() (Markdown语法)
    //   - autocompletion() + closeBrackets() (自动补全+括号配对)
    //   - lineNumbers (行号)
    //   - highlightActiveLineGutter() (激活行高亮)
    //   - 自定义 theme (匹配现有 UI 风格)
    // 返回 new EditorView({ state, parent: container })
}

function destroyCodeMirror() {
    if (cmEditor) { cmEditor.destroy(); cmEditor = null; }
}
```

**2. 修改 openEditor() (~20 行变更)**
- 加载笔记数据后，调用 `initCodeMirror()` 替代直接设置 `textarea.value`
- 只读模式：CM6 设为 `editable: false`
- 删除 `new UndoRedoManager()` × 2 的调用（CM6 内置 history）
- 查看模式纯文本：CM6 readOnly 展示原始内容

**3. 修改 closeEditor() (~5 行变更)**
- 新增 `destroyCodeMirror()` 调用
- 删除 `undoRedoTitle.destroy()` / `undoRedoContent.destroy()`
- 删除 `findReplace.reset()` （CM6 search 自管理）

**4. 修改 switchEditorMode() (~10 行变更)**
- `'edit'` 模式：显示 CM6 容器，隐藏 `#mdRendered`
- `'preview'` 模式：隐藏 CM6 容器，显示 `#mdRendered`，调用 `updatePreview()`
- 切换时如果 CM6 search panel 打开则关闭

**5. 修改 onEditorInput() (~10 行变更)**
- 从监听 `input` 事件改为监听 CM6 的 `ViewUpdate.docChanged`
- 字数统计从 `textarea.value.length` 改为 `cmEditor.state.doc.length`
- 自动保存逻辑保持不变

**6. 修改 updateWordCount() (~5 行变更)**
- 标题仍从 `input.value` 获取（标题 input 保持原样）
- 内容从 `cmEditor.state.doc.toString()` 获取

**7. 修改 createNote() / updateNote() (~5 行变更)**
- 内容获取从 `els.editorNoteContent.value` 改为 `cmEditor.state.doc.toString()`

**8. 修改 handleKeyboardNavigation() (~30 行删减)**
- **删除**: Ctrl+Z / Ctrl+Y 处理（CM6 historyKeymap 内置处理）
- **删除**: `[` / `]` 导航匹配项（CM6 search 内置处理）
- **修改**: Ctrl+F 处理 — 编辑器打开时调用 CM6 的 `openSearchPanel`（而非 findReplace.open）
- **修改**: Ctrl+H 处理 — 同上，CM6 search 内置替换
- **修改**: Escape 优先级 — CM6 search panel 打开时由 CM6 自己处理 Escape 关闭
- **保留**: Ctrl+L（切换模式）、Ctrl+N（新建）、PgUp/PgDn 等

**9. 删除 FindReplaceManager 类（~636 行）**
- CM6 的 `@codemirror/search` 完全覆盖查找/替换功能
- 包括：搜索面板 UI、正则/大小写/全词选项、全部替换、匹配计数、前后导航
- 预览模式查找：继续在 `#mdRendered` 上使用简化版高亮（可选保留轻量实现）

**10. 删除 UndoRedoManager 类（~90 行）**
- CM6 的 `history()` extension 完全覆盖

##### Phase 2: 样式适配与优化（紧接 Phase 1）

**11. CSS 变更 (`frontend/src/style.css`)**

新增/修改：

```
a. .cm-editor 容器样式（替代 .editor-textarea）
   - flex:1, min-height:0, overflow:auto
   - font-family, font-size, line-height 与原 textarea 一致
   - 圆角、边框等视觉属性继承

b. CM6 Theme 定义（~50 行）
   - --cm-bg: var(--card-bg)
   - --cm-text: var(--text-primary)
   - --cm-cursor: var(--accent)
   - --cm-selection: var(--accent-light)
   - --cm-gutter-bg: transparent
   - --cm-gutter-border: transparent
   - --cm-active-line: rgba(var(--accent-rgb), 0.05)
   - --cm-matching-bracket: var(--accent-light)
   - --cm-search-match: var(--accent-light)
   - --cm-search-match-selected: var(--accent)

c. 删除的样式（可后续清理）
   - .find-overlay 相关（~20 行）
   - .editor-textarea.find-active（~5 行）
   - .find-highlight / .find-highlight.active（~15 行）
   - 大量 .editor-find-bar 子元素样式（~150 行）— CM6 自带 search panel 样式

d. Markdown 高亮样式（~40 行）
   - .cm-header-1 到 .cm-header-6（标题颜色/大小）
   - .cm-strong（加粗）
   - .cm-em（斜体）
   - .cm-link（链接颜色）
   - .cm-code / .cm-monospace（代码）
   - .cm-list（列表）
```

**12. 查找条 HTML 大幅简化**

CM6 的 search 扩展会自行创建查找/替换面板 DOM 并注入到 editor DOM 中。有两种选择：

- **方案 A（推荐）**：完全删除 `#editorFindBar` 及其所有子元素，让 CM6 在编辑区域内（顶部或底部）展示 search panel。这是 CM6 默认行为。
- **方案 B**：保留外部容器但清空内部内容，通过 CM6 配置将 search panel 渲染到指定容器。

推荐方案 A，减少维护成本。

##### Phase 3: 预览模式查找（可选增强）

**13. 预览区查找高亮保留轻量版 (~50 行)**

CM6 的 search 只作用于自身编辑区域。预览模式的 `#mdRendered` 是独立 DOM，需要单独处理：
- 保留现有的 `_highlightPreview()` / `_clearPreviewHighlight()` 逻辑
- 但从 FindReplaceManager 中剥离出来，作为独立的 2 个工具函数
- 或改用 CM6 的 `@codemirror/search` 在一个临时的只读 CM6 实例上操作 mdRendered 内容（更优雅但更复杂）

## Impact

### 受影响的文件

| 文件 | 变更类型 | 影响范围 |
|------|---------|----------|
| `frontend/package.json` | 新增依赖 | 新增 ~8 个 npm 包 |
| `frontend/index.html` | 结构调整 | textarea→div, 删除 findOverlay, 简化 findBar |
| `frontend/src/main.js` | **大改** | 新增 ~120 行 CM6 模块, 删除 ~726 行 (FindReplaceManager + UndoRedoManager), 修改 ~15 个函数 |
| `frontend/src/style.css` | **中改** | 新增 ~100 行 CM6 主题+MD高亮, 删除 ~190 行 查找/overlay 样式, 修改 ~20 行 |
| Go 后端 | **无变化** | 纯前端改动 |

### 净代码量变化

| 操作 | 行数 |
|------|------|
| 新增 CM6 初始化模块 | +120 |
| 新增 CM6 主题 CSS | +100 |
| 新增 MD 高亮 CSS | +40 |
| 删除 FindReplaceManager | -636 |
| 删除 UndoRedoManager | -90 |
| 删除查找相关 CSS | -190 |
| 修改现有函数 | ±50（净减） |
| **净变化** | **约 -600 行** |

### 功能对照表

| 当前功能 | 变更后 | 说明 |
|----------|--------|------|
| 撤销/恢复 | ✅ CM6 内置 | 删除 UndoRedoManager |
| 查找 (Ctrl+F) | ✅ CM6 内置 | 删除 FindReplaceManager 大部分代码 |
| 替换 (Ctrl+H) | ✅ CM6 内置 | 含正则/大小写/全部替换 |
| 匹配导航 ([/]) | ✅ CM6 内置 | F3/Shift+F3 也自动支持 |
| Tab 缩进 | ✅ CM6 内置 | indentWithTab |
| MD 语法高亮 | ✅ 新增 | lang-markdown 开箱即用 |
| 括号配对 | ✅ 新增 | closeBrackets |
| 自动补全 | ✅ 新增 | autocompletion（可配置提示规则） |
| 行号 | ✅ 新增 | lineNumbers |
| 当前行高亮 | ✅ 新增 | highlightActiveLineGutter |
| 选中文本不关闭 | ✅ 自然解决 | CM6 自管事件 |
| 光标不丢失 | ✅ 自然解决 | State 更新机制 |
| 预览渲染 | 不变 | 继续用 marked.js |
| 标题输入框 | 不变 | 继续 input 元素（标题太短不值得用 CM6） |
| 自动保存 | 微调 | 监听源从 input 事件变为 ViewUpdate |
| 字数统计 | 微调 | 数据源从 .value 变为 state.doc |
| 全屏切换 | 不变 | CSS class 驱动 |
| 标签选择 | 不变 | 独立区域不受影响 |

## ADDED Requirements

### Requirement: CodeMirror 6 编辑器集成

系统 SHALL 在新建/编辑页面的纯文本模式和查看页面的纯文本显示中使用 CodeMirror 6 替代原生 textarea。

#### Scenario: 新建笔记打开编辑器

- **WHEN** 用户点击 FAB "+" 按钮 / 按 Ctrl+N / 右键菜单"编辑"
- **THEN** 编辑器打开，内容区域为 CodeMirror 6 可编辑实例
- AND 标题输入框仍为原生 input（保持不变）
- AND CM6 配置包含：history(撤销恢复)、search(查找替换)、indentWithTab(Tab缩进)、markdown(MD语法)、autocompletion(自动补全)、lineNumbers(行号)、highlightActiveLine(当前行高亮)
- AND CM6 主题配色与应用整体风格一致

#### Scenario: 编辑已有笔记

- **WHEN** 用户点击笔记卡片的编辑按钮
- **THEN** 编辑器打开，CM6 实例加载笔记原文内容
- AND 所有编辑功能（撤销/恢复/查找/替换/缩进）均可用

#### Scenario: 查看笔记（只读）

- **WHEN** 用户左击笔记卡片进入查看模式
- **AND** 笔记类型为 text 或处于纯文本显示模式
- **THEN** CM6 以 readOnly/editable:false 模式展示内容
- AND 用户可以选中文本、复制、使用查找功能
- AND 不能修改任何内容

#### Scenario: 切换到预览模式

- **WHEN** 用户点击底部"预览"按钮或按 Ctrl+L
- **THEN** CM6 容器隐藏，#mdRendered 显示 marked.js 渲染结果
- AND 如果 CM6 search panel 处于打开状态则关闭
- AND 预览模式下按 Ctrl+F/H 仍在 #mdRendered 中提供查找能力（轻量高亮）

#### Scenario: 切换回纯文本模式

- **WHEN** 用户在预览模式下点击"纯文本"或按 Ctrl+L
- **THEN** #mdRendered 隐藏，CM6 容器显示
- AND CM6 内容与 #mdRendered 渲染前一致（双向同步）

#### Scenario: 保存笔记

- **WHEN** 用户点击保存按钮
- **THEN** 从 `cmEditor.state.doc.toString()` 获取内容提交后端
- AND 成功后关闭编辑器并销毁 CM6 实例

#### Scenario: 自动保存

- **WHEN** 用户在 CM6 中输入内容
- **THEN** 监听 `ViewUpdate.docChanged` 触发自动保存（3 秒防抖不变）
- AND 字数统计实时更新

#### Scenario: 快捷键兼容

- **WHEN** 用户在 CM6 焦点内按下以下快捷键
- **THEN** 对应行为如下：

| 快捷键 | 行为 |
|--------|------|
| Ctrl+Z | 撤销（CM6 history） |
| Ctrl+Y | 恢复（CM6 history） |
| Ctrl+F | 打开 CM6 查找面板 |
| Ctrl+H | 打开 CM6 查找+替换面板 |
| Ctrl+L | 切换纯文本/预览模式（应用级处理） |
| Ctrl+N | 新建笔记（应用级处理，不在 CM6 内时生效） |
| [ 或 ] | 导航上一个/下一个搜索匹配（CM6 search） |
| Escape | 关闭 search panel > 退出全屏 > 关闭编辑器（多级优先级） |
| Tab | 插入缩进（CM6 indentWithTab） |
| Shift+Tab | 减少缩进（CM6 indentMore/indentLess） |
| Ctrl+S | 保存（应用级处理） |

## MODIFIED Requirements

### Requirement: 编辑器纯文本/预览模式切换

原有 `switchEditorMode(mode)` 函数 SHALL 修改为：
- `'edit'` 模式：显示 CM6 容器（`.cm-editor`），隐藏 `#mdRendered`
- `'preview'` 模式：隐藏 CM6 容器，显示 `#mdRendered`，调用 `updatePreview()`
- 切换时重置 CM6 search panel 状态

### Requirement: 编辑器关闭清理

原有 `closeEditor()` 函数 SHALL 增加：
- 调用 `destroyCodeMirror()` 销毁 CM6 实例
- 移除对 `UndoRedoManager.destroy()` 和 `FindReplaceManager.reset()` 的调用

## REMOVED Requirements

### Requirement: UndoRedoManager

**Reason**: CodeMirror 6 内置 `history()` extension 完全覆盖撤销/恢复功能，且实现更完善（IME 安全、组合操作合并、持久化光标历史等）。

**Migration**: 直接删除类定义和所有引用。CM6 history 通过 `historyKeymap` 自动绑定 Ctrl+Z/Y。

### Requirement: FindReplaceManager

**Reason**: CodeMirror 6 内置 `@codemirror/search` extension 完全覆盖查找/替换功能，且自带 UI 面板（搜索输入框、替换输入框、正则/大小写/全词开关、匹配计数、前后导航、全部替换）。

**Migration**:
- 删除完整类定义（~636 行）
- 删除 `#editorFindBar` 下大部分自定义 DOM（CM6 自行创建 search panel）
- 删除 `#findOverlay` overlay 覆盖层（CM6 使用装饰器(decoration)实现高亮，无需额外 DOM）
- 预览模式查找：保留轻量版的 `_highlightPreview()` / `_clearPreviewHighlight()` 工具函数（~50 行），从 FindReplaceManager 中剥离
