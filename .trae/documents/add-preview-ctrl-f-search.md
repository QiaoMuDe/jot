# 预览模式下支持 Ctrl+F 搜索（方案 A：DOM 高亮搜索）

## Summary

在 Markdown 预览模式下按 Ctrl+F 时，不再强制切回纯文本模式，而是在预览渲染区（`#mdRendered`）内直接打开一个查找条，对渲染后的 DOM 文本做高亮搜索，支持计数、Enter/Shift+Enter 导航、自动滚动、Esc 关闭并清除高亮。编辑模式保持 CM6 内置搜索不变。

## Current State Analysis（现状）

* 搜索能力完全依赖 CodeMirror 6：`initCodeMirror` 注入了 `searchKeymap`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L91-L100)），搜索面板挂在 `.cm-editor` DOM 内部，只能搜索 CM6 文档。

* 预览模式下 CM6 容器被 `display: none` 隐藏（[editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css#L265-L271)），隐藏元素不接收键盘事件，CM6 的 Ctrl+F 失效。

* 全局兜底逻辑在 `handleKeyboardNavigation`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6349-L6371)）：预览模式按 Ctrl+F 会 `switchEditorMode('edit')` 切回纯文本再 `openSearchPanel(cmEditor)`——这就是"只能切到纯文本搜索"的来源。

* 预览区 `#mdRendered` 是 marked 渲染的静态 HTML（Worker 异步渲染，见 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4795-L4832) 与 worker `onmessage` [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4205-L4253)），无任何搜索逻辑。

* [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L215-L229) 存在一个废弃的 `editorFindBar` 查找/替换条 HTML 骨架（对应已废弃的 `add-find-replace` spec），全仓库**无任何 JS/CSS 引用**（纯死代码），本次复用其查找行骨架。

* 事件初始化集中在 `initEventListeners()`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L5802)），全局快捷键在 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6156) 绑定。

* 编辑器关闭清理在 `closeEditor()`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4997-L5056)）。

## Proposed Changes

### 1. `frontend/src/css/components/editor.css` — 新增查找条与 mark 高亮样式

在文件末尾追加（复用现有 CSS 变量，风格与 footer 一致）：

```css
/* ===== 预览模式查找条（Ctrl+F） ===== */
.editor-find-bar {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 12px;
  border-top: 1px solid var(--border);
  background: var(--card-bg);
}
.find-bar-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.find-input {
  flex: 1;
  min-width: 0;
  height: 26px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--input-bg, var(--bg-secondary));
  color: var(--text-primary);
  font-size: 0.875rem;
  outline: none;
}
.find-input:focus {
  border-color: var(--accent);
}
.find-count {
  flex-shrink: 0;
  font-size: 0.75rem;
  color: var(--text-secondary);
  min-width: 44px;
  text-align: center;
}
.find-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}
.find-btn:hover {
  background: var(--bg-hover, rgba(128,128,128,0.12));
  color: var(--text-primary);
}
.find-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

/* 预览区搜索高亮 */
.md-rendered mark {
  background: rgba(250, 204, 21, 0.35);
  color: inherit;
  border-radius: 2px;
  padding: 0 1px;
}
[data-theme="dark"] .md-rendered mark {
  background: rgba(250, 204, 21, 0.3);
}
.md-rendered mark.active {
  background: var(--accent, #6c63ff);
  color: #fff;
}
```

注：`replace-bar-row`（替换行）为废弃死代码，保持原样不动（预览为只读，无需替换）。

### 2. `frontend/src/main.js` — 预览搜索逻辑

#### 2a. 新增模块级状态（放在 `_previewWorkerLoading` 声明附近，约 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4194)）

```javascript
// 预览模式查找状态
let _previewFindBarVisible = false;   // 查找条是否打开
let _previewSearchQuery = '';         // 当前搜索关键词
let _previewMarkMatches = [];         // 当前高亮的 <mark> 元素数组
let _previewMarkCurrent = -1;         // 当前激活匹配索引
let _previewFindTimer = null;         // 输入防抖定时器
```

#### 2b. 新增预览搜索函数（放在 `switchEditorMode` 附近或 `updatePreview` 之后）

```javascript
/** 打开预览查找条（预览模式下 Ctrl+F 触发） */
function openPreviewFindBar() {
    const bar = document.getElementById('editorFindBar');
    const input = document.getElementById('findInput');
    if (!bar || !input) return;
    bar.style.display = '';
    _previewFindBarVisible = true;
    input.value = '';
    _previewSearchQuery = '';
    _clearPreviewMarks();
    _previewMarkMatches = [];
    _previewMarkCurrent = -1;
    updateFindCount(0);
    input.focus();
    input.select();
}

/** 关闭预览查找条并清除高亮 */
function closePreviewFindBar() {
    const bar = document.getElementById('editorFindBar');
    if (bar) bar.style.display = 'none';
    _previewFindBarVisible = false;
    _previewSearchQuery = '';
    _clearPreviewMarks();
    _previewMarkMatches = [];
    _previewMarkCurrent = -1;
}

/** 清除预览区所有 mark 高亮（还原为文本节点） */
function _clearPreviewMarks() {
    _previewMarkMatches.forEach((m) => {
        if (m.parentNode) m.replaceWith(document.createTextNode(m.textContent));
    });
    _previewMarkMatches = [];
    _previewMarkCurrent = -1;
}

/**
 * 在 mdRendered DOM 中执行搜索并高亮
 * 使用 TreeWalker 遍历文本节点，跳过 svg( Mermaid)/script/style 内部文本
 */
function runPreviewSearch(query) {
    _clearPreviewMarks();
    const root = els.mdRendered;
    const text = query.trim();
    updateFindCount(0);
    if (!text) { _previewSearchQuery = ''; return; }
    _previewSearchQuery = query;

    const lower = text.toLowerCase();
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
        acceptNode(node) {
            if (node.parentElement && node.parentElement.closest('svg, script, style')) {
                return NodeFilter.FILTER_REJECT;
            }
            return NodeFilter.FILTER_ACCEPT;
        }
    });
    // 收集 (node, ranges) —— 同一节点多个匹配合并处理
    const grouped = [];
    const nodes = [];
    let n;
    while ((n = walker.nextNode())) {
        const nodeText = n.nodeValue;
        const lowerNode = nodeText.toLowerCase();
        const ranges = [];
        let idx = 0;
        while ((idx = lowerNode.indexOf(lower, idx)) !== -1) {
            ranges.push({ start: idx, end: idx + text.length });
            idx += text.length; // 不重叠匹配
            if (ranges.length >= 500) break; // 单节点匹配上限，防极端卡顿
        }
        if (ranges.length) grouped.push({ node: n, ranges });
        if (grouped.length >= 1000) break; // 总分组上限，防极端卡顿
    }

    // 拆分文本节点并包裹 mark
    for (const { node, ranges } of grouped) {
        const frag = document.createDocumentFragment();
        let last = 0;
        for (const r of ranges) {
            if (r.start > last) frag.appendChild(document.createTextNode(node.nodeValue.slice(last, r.start)));
            const mark = document.createElement('mark');
            mark.textContent = node.nodeValue.slice(r.start, r.end);
            frag.appendChild(mark);
            _previewMarkMatches.push(mark);
            last = r.end;
        }
        if (last < node.nodeValue.length) frag.appendChild(document.createTextNode(node.nodeValue.slice(last)));
        node.parentNode.replaceChild(frag, node);
    }

    updateFindCount(_previewMarkMatches.length);
}

/** 导航到上一个/下一个匹配（dir = 1 下一个，-1 上一个） */
function navigatePreviewMatch(dir) {
    const total = _previewMarkMatches.length;
    if (!total) return;
    _previewMarkCurrent = (_previewMarkCurrent + dir + total) % total;
    _previewMarkMatches.forEach((m, i) => m.classList.toggle('active', i === _previewMarkCurrent));
    updateFindCount(total);
    const active = _previewMarkMatches[_previewMarkCurrent];
    if (active) active.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

/** 更新查找条计数显示（0/0 或 (当前+1)/总数） */
function updateFindCount(total) {
    const el = document.getElementById('findCount');
    if (!el) return;
    el.textContent = total ? `${_previewMarkCurrent + 1}/${total}` : '0/0';
}
```

#### 2c. 修改 `handleKeyboardNavigation` 的 Ctrl+F 分支（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6349-L6371)）

预览模式改为打开预览查找条，不再切回编辑模式：

```javascript
// Ctrl/Cmd+F: 编辑器内搜索（预览模式在预览区直接搜索；编辑模式用 CM6 搜索并填充选中文本）;编辑器外则打开搜索弹窗
if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
    e.preventDefault();
    if (els.viewEditor.classList.contains('active') && cmEditor) {
        // 预览模式：直接在预览渲染区搜索（不切回纯文本）
        if (els.editorOverlay.dataset.mode === 'preview') {
            openPreviewFindBar();
            return;
        }
        // 编辑模式：CM6 内置搜索，填充选中文本
        const sel = cmEditor.state.selection.main;
        if (!sel.empty) {
            const selectedText = cmEditor.state.sliceDoc(sel.from, sel.to);
            cmEditor.dispatch({
                effects: setSearchQuery.of(new SearchQuery({ search: selectedText }))
            });
        }
        openSearchPanel(cmEditor);
        return;
    }
    // 编辑器外:打开搜索弹窗
    openSearchModal();
    return;
}
```

#### 2d. 修改 `switchEditorMode`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4860-L4882)）

切回编辑模式时关闭预览查找条并清除高亮：

```javascript
} else if (mode === 'edit') {
    _setPreviewLayout(false);
    _closeToc();
    closePreviewFindBar();   // 新增
}
```

#### 2e. `initEventListeners()` 中新增查找条事件绑定（约 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L6156) 附近）

```javascript
// 预览模式查找条事件
const findInput = document.getElementById('findInput');
const findPrevBtn = document.getElementById('findPrevBtn');
const findNextBtn = document.getElementById('findNextBtn');
const findCloseBtn = document.getElementById('findCloseBtn');
findInput.addEventListener('input', () => {
    clearTimeout(_previewFindTimer);
    _previewFindTimer = setTimeout(() => runPreviewSearch(findInput.value), 150);
});
findInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        e.preventDefault();
        navigatePreviewMatch(e.shiftKey ? -1 : 1);
    } else if (e.key === 'Escape') {
        e.preventDefault();
        closePreviewFindBar();
    }
});
findPrevBtn.addEventListener('click', () => navigatePreviewMatch(-1));
findNextBtn.addEventListener('click', () => navigatePreviewMatch(1));
findCloseBtn.addEventListener('click', closePreviewFindBar);
```

注意：导航键采用 **Enter/Shift+Enter**（而非废弃 spec 的 `[`/`]`，因为 `[`/`]` 与查找输入框输入冲突）。Esc 键已由 `handleKeyboardNavigation` 全局兜底处理关闭弹窗，但查找条输入框自身的 Esc 事件优先在这里拦截（focus 在输入框内时事件不会冒泡到全局处理器之前被 CM6 拦截，需在此显式处理）。

#### 2f. 预览重新渲染后恢复搜索（Worker `onmessage` 末尾 + 主线程回退路径）

Worker `onmessage` 在 `_previewWorkerLoading = false;` 之前追加（约 [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4252)）：

```javascript
// 若预览查找条仍打开，重新执行搜索（innerHTML 替换后 mark 已丢失）
if (_previewFindBarVisible && _previewSearchQuery) {
    runPreviewSearch(_previewSearchQuery);
}
```

主线程回退路径 `updatePreview`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4824-L4832)）末尾同样追加上述恢复逻辑。

#### 2g. `closeEditor()` 清理（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L5045-L5048) 附近）

在 `els.mdRendered.innerHTML = '';` 之前追加：

```javascript
closePreviewFindBar();
```

## Assumptions & Decisions

* **复用废弃的** **`editorFindBar`** **HTML 骨架**：其查找行（input/count/prev/next/close）结构完整、id 全局唯一，无需改 HTML；`replace-bar-row` 保持不动（预览只读、编辑走 CM6）。

* **导航键为 Enter/Shift+Enter**（`[`/`]` 会与输入框输入冲突，废弃 spec 的设计不再沿用）。

* **大小写不敏感**（与原 spec 一致，不做 toggle，保持最小改动）。

* **搜索范围是渲染后文本**（所见即所得）：代码块、表格内部文本可命中；跳过 Mermaid `<svg>` 内部文本。

* **匹配上限保护**：单节点 500 个、总分组 1000 个，防止极端文档卡顿（正常文档远达不到）。

* **查看模式（readOnly，打开笔记默认预览）同样生效**，符合"MD 笔记预览搜索"诉求。

* 编辑模式（纯文本）Ctrl+F / Ctrl+H 行为完全不变（CM6 内置）。

* 非 MD（.txt 等）笔记无预览，Ctrl+F 走 CM6，行为不变。

## Verification

1. `wails dev` 或前端 dev server 启动应用。
2. 打开一个 MD 笔记（查看模式，默认预览）：按 Ctrl+F → 底部出现查找条且输入框自动聚焦，**不切换到纯文本**。
3. 输入关键词 → 预览区所有匹配被 `mark` 高亮，计数显示 `x/N`。
4. Enter 下一个 / Shift+Enter 上一个：当前项高亮变色并平滑滚动到可视区；按钮点击同样生效。
5. Esc 或点关闭按钮：查找条隐藏、高亮全部清除。
6. 编辑模式内切到"预览"后按 Ctrl+F：同样生效。
7. 编辑模式按 Ctrl+F：仍走 CM6 内置搜索面板（行为与改动前一致）。
8. 搜索打开状态下修改内容触发预览重渲染：查找条保持，高亮自动重跑。
9. 关闭编辑器 / 从预览切回纯文本：查找条与高亮被清理，无残留。
10. 检查无 console 报错。

