# 统一笔记编辑器加载逻辑 + 大文件切回不闪烁

## Summary

当前打开笔记时存在**两套互不相关的加载反馈**（编辑/纯文本模式的"空白/骨架屏" vs Markdown 预览模式的"暂无内容/转圈"），且大文件在只读预览模式下会被"先预览 → 数据就绪后切回 CM6"导致**跳动/闪烁**。

本次改造目标：
1. 用**一个统一的加载覆盖层**收口"打开笔记"这一动作的加载反馈（编辑模式与预览模式一致）。
2. **把显示模式判定推迟到数据就绪后一次性完成**，消除"先预览后切回"的闪烁。
3. 模式切换沿用现有淡入动画，保证过渡丝滑。

> 范围说明：不做 CM6 大文件语法高亮降级（该优化涉及编辑器性能，独立于本方案，另行评估）。

## Current State Analysis（现状）

### 相关文件与关键代码位置
- `frontend/index.html`（[L167-175](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L167-L175)）：`.editor-panes`（`position:relative`）内含三个子节点——`#editorNoteContent.editor-textarea`（CM6 挂载点）、`.toc-sidebar`、`#mdRendered.md-rendered`。
- `frontend/src/main.js`：
  - `openEditor`：
    - 阶段一 [L3942-3969](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3942-L3969)：**在数据就绪前**就根据 `isReadOnly && noteData && isMd` 预判 `preview` 模式并塞 `暂无内容`；否则 `switchEditorMode('edit')`。
    - 阶段二 [L4147-4166](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4147-L4166)：数据就绪后，若是只读 markdown 且内容超阈值（`ai_large_file_preview_threshold`，默认 10000），`switchEditorMode('edit')` **从预览切回 CM6**；否则 `updatePreview()`。
  - `initCodeMirror` [L93-176](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L93-L176)：同步初始化 CM6 并填充全部内容。
  - `switchEditorMode` [L4851-4876](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4851-L4876)：仅改 `data-mode` + 按钮态 + 触发 `updatePreview`/关闭 TOC 等，容器显隐靠 CSS。
  - `updatePreview` [L4788-4823](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4788-L4823)：空内容→`暂无内容`；有内容→`加载中…`（转圈）→ Worker 返回后渲染。
  - `els` 定义 [L499-521](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L499-L521)：在此为统一加载层新增引用。
- `frontend/src/css/components/editor.css`：
  - `[data-mode="edit"/"preview"]` 的两容器显隐 [L306-349](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css#L306-L349)（含 `modeFadeIn 0.2s` 淡入）。
  - `crossfade-overlay` [L1245-1253](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css#L1245-L1253)（预览 Worker 返回时的交叉淡出）。

### 根因
1. 阶段一过早预判 `preview` + 塞 `暂无内容`，大文件数据就绪后又被切回 `edit`（`switchEditorMode`）→ "暂无内容 → 切回" 跳动闪烁。仅 `noteData` 命中缓存时才走这条分支，其余情况阶段一直接 `edit`，故"时有时无"。
2. 打开动作的加载反馈分散在两套实现（编辑空白 / 预览"暂无内容+转圈"），观感不一致。
3. CM6 大文件同步初始化 + 高亮会阻塞主线程数帧，切换瞬间易被感知为卡顿（本次仅靠加载层遮挡缓解，不做高亮降级）。

## Proposed Changes（改动方案）

### 1) 新增统一加载覆盖层（HTML + CSS + els 引用）

**`frontend/index.html`**（`.editor-panes` 内、`#mdRendered` 之后）追加：
```html
<div class="editor-loading" id="editorLoading" style="display:none">
  <span class="editor-loading-spinner"></span>
</div>
```
- 用 `position:absolute; inset:0` 覆盖整个 `.editor-panes`，z-index 高于两容器（现 `toc-sidebar`/`crossfade-overlay` 均在低 z 层），保证打开瞬间完全遮盖。
- 元素通过 `els.editorLoading` 引用操作显示/隐藏。

**`frontend/src/css/components/editor.css`**（在 `modeFadeIn` 块附近新增）：
```css
.editor-loading {
  position: absolute; inset: 0; z-index: 12;
  display: flex; align-items: center; justify-content: center;
  background: var(--card-bg);
  transition: opacity 0.2s ease-out;
}
.editor-loading.hidden { opacity: 0; pointer-events: none; }
.editor-loading-spinner {
  width: 20px; height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: editorSpin 0.7s linear infinite;
}
@keyframes editorSpin { to { transform: rotate(360deg); } }
```
- 淡出：加 `.hidden`（opacity 0）+ `transition`，`transitionend` 后 `display:none`（避免残留不可见占位）。
- `--card-bg` / `--border` / `--accent` 均为现成主题变量。

**`frontend/src/main.js`**（`els` 定义处）：
```js
editorLoading: $('editorLoading'),
```

### 2) 重构 `openEditor`：模式延迟到数据就绪后一次性判定

**阶段一 [L3942-3969](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3942-L3969)**：
- 保留 `[L3944-3946]`（清 `mdRendered` 与预览哈希缓存，防上条残留闪现）。
- **删除** `[L3949-3969]` 的 `if (isReadOnly...) { ... } else { switchEditorMode('edit') }` 预判块（不再在阶段一决定预览/编辑、不再塞 `暂无内容`）。
- 改为：立即显示统一加载层（`els.editorLoading` 显示）。后续具体模式统一在阶段二确定。
- 面板入场动画等其余阶段一逻辑保持不变。

**阶段二 [L4147-4166](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4147-L4166)**（数据已就绪、CM6 已 `initCodeMirror` 填充、`isMd`/`ext` 已由后端加载校正后）替换为"一次性定模式"逻辑：
```js
// ── 数据已就绪：一次性确定最终显示模式（避免"先预览后切回"的闪烁） ──
const largeFileThreshold = parseInt(document.getElementById('aiLargeFilePreviewThreshold')?.value) || 10000;
if (isReadOnly && isMd) {
    if (editorContent.length > largeFileThreshold) {
        // 超大内容：纯文本模式（CM6），不进入预览
        if (els.editorOverlay.dataset.mode !== 'edit') switchEditorMode('edit');
        _setPreviewLayout(false);
        _closeToc();
        window.showNotification?.('笔记内容超过纯文本预览阈值，已自动切换为纯文本模式', 'info');
    } else {
        // 查看预览
        if (els.editorOverlay.dataset.mode !== 'preview') switchEditorMode('preview');
        updatePreview();
    }
} else {
    // 纯文本笔记 / 非只读编辑
    if (els.editorOverlay.dataset.mode !== 'edit') switchEditorMode('edit');
}
// 新建笔记（非只读）自动聚焦
if (!state.editingNoteId && els.editorOverlay.dataset.mode !== 'preview' && document.hasFocus()) {
    window.focus();
    cmEditor?.focus();
}

// 数据就绪后隐藏统一加载层（置于 setMode + updatePreview 之后，避免空白闪现）
hideEditorLoading();
```
- 要点：**只在需要时才切换**（`dataset.mode !== target`），模式至多切换一次；若已被校正为对应模式则复用，减少 `switchEditorMode` 中 focus/关闭副作用。
- `updatePreview()` 内部自行处理"暂无内容"（空）与"加载中…转圈"（有内容），与加载层天然衔接：加载层淡出时预览区已就绪。

**侧链清理**：阶段二后端加载校正段 `[L4111-4118]`（`if (isReadOnly && isMd) { ... set preview ... }`）中手动设置 `preview` 的逻辑，与新逻辑重复，一并移除（其 `_setPreviewLayout(false)` 副作用由新定模式逻辑兜住）。

### 3) 新增 `hideEditorLoading` 小工具（就近定义于 `openEditor` 附近）

```js
function hideEditorLoading() {
    const el = els.editorLoading;
    if (!el) return;
    el.classList.add('hidden');
    el.addEventListener('transitionend', () => { el.style.display = 'none'; }, { once: true });
}
```
- 显示统一用 `els.editorLoading.style.display = 'flex'`（阶段一）。

### 行为结果
- 打开任何笔记：打开瞬间显示**同一个转圈加载层** → 数据就绪后淡出，直接呈现目标模式内容。
- 编辑/纯文本、预览模式初始反馈一致（消除"编辑空白 vs 预览转圈"割裂）。
- 大文件只读打开：不再闪现"暂无内容→切回"，而是加载层期间直接确定 `edit`，淡出后即 CM6 内容。
- 预览模式的实时渲染"转圈/暂无内容"保留（属单条内容的渲染态，与"打开"加载不同源）。

## Assumptions & Decisions
- 复用现有 `switchEditorMode` / `updatePreview` / `_setPreviewLayout`，不重写，最小改动。
- 统一加载层覆盖打开的"数据加载"阶段；预览模式内 Worker 渲染转圈（`md-rendered-loading`）保留，二者语义不同不强行合并。
- 不做 CM6 大文件高亮降级（超出本次范围）。
- 阈值读取逻辑不变（`ai_large_file_preview_threshold`，默认 10000）。

## Verification
1. `npm run dev`（或项目相应启动命令）启动，确认无编译错误。
2. **编辑/纯文本笔记**：打开时加载层转圈 → 内容淡出，无横条、无空白突兀。
3. **小 markdown 只读**：加载层 → 预览区（`暂无内容` 或渲染内容），无跳变。
4. **大 markdown 只读（>10000 字符）**：加载层 → 直接 CM6 纯文本内容，**无"暂无内容→切回"闪烁**；有"已自动切换为纯文本模式"提示。
5. **新建笔记**：加载层快速过渡到空编辑区，焦点自动落于 CM6。
6. 面板关闭再快速重开（<200ms）仍正常（`editorOpSeq` 竞态保护不受影响）。