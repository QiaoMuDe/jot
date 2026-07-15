# 编辑器骨架屏方案 — 点击笔记即动

## 摘要

重构 `openEditor` 流程，将面板动画从"等待一切就绪再播放"改为"先播放面板动画 + 骨架屏，后台加载数据和 CM6"。让用户点击笔记后立即看到编辑器面板弹入的动画效果，消除当前 200-700ms 的等待感。

## 当前状态分析

### 当前 `openEditor` 时序（简要）

```
用户点击笔记
  │
  ├─ GetNoteContent(noteId)          ← ~50-200ms
  ├─ await loadTagsForEditor()       ← ~10-50ms
  │
  ├─ overlay/panel opacity = '0'     ← 强制隐藏
  ├─ viewEditor.classList.add('active')
  │
  ├─ initCodeMirror(container, content)  ← ~100-500ms（最重）
  │   （EditorState.create + 15+ 扩展 + EditorView）
  │
  └─ 启动 animation（overlayFadeIn + modalEnter）
```

**问题**：所有异步操作（数据加载）+ 重操作（CM6 初始化）**串行阻塞动画启动**。最短等待约 200ms，大文件/慢设备可达 700ms+，期间页面已锁定（`overflow: hidden`）但画面不动。

### 已知可复用的基础设施

| 复用点                         | 位置              | 说明                                       |
| --------------------------- | --------------- | ---------------------------------------- |
| `setEditorContent(content)` | main.js#L259    | 使用 `cmEditor.dispatch` 更新 CM6 内容         |
| `destroyCodeMirror()`       | main.js#L252    | 销毁 CM6 实例                                |
| 骨架屏 spinner 样式              | editor.css#L994 | `.md-rendered-loading` 已有 spinner + 脉冲动画 |
| `cmFadeIn` 动画               | editor.css#L222 | CM6 内容淡入关键帧                              |

### 动画参数

| 动画                            | 持续时间  | 缓动函数                          |
| ----------------------------- | ----- | ----------------------------- |
| overlayFadeIn                 | 0.2s  | ease-out                      |
| modalEnter (scale 0.85→1)     | 0.3s  | cubic-bezier(0.16, 1, 0.3, 1) |
| viewEnter (translateY 12px→0) | 0.25s | ease-out                      |

## 目标时序

```
用户点击笔记
  │
  ├─ 锁定滚动（overflow = 'hidden'）
  ├─ 显示 overlay + panel
  ├─ 立即播放动画 overlayFadeIn + modalEnter  ← 0ms 响应！
  │
  │   ┌── 面板内现内容 ──────────────────┐
  │   │  标题栏：显示笔记标题或占位        │
  │   │  标签区：空                        │
  │   │  内容区：骨架屏（3-4 条脉冲线条）   │
  │   │  底部：字数统计（0）               │
  │   └───────────────────────────────────┘
  │
  ├─ 后台并行加载（互不阻塞）：
  │   ├─ GetNoteContent(noteId)      ← 异步
  │   ├─ loadTagsForEditor()         ← 异步
  │   └─ (等待动画完成 ~300ms 后) initCodeMirror(container, '')
  │
  ├─ 内容到达 → setEditorContent(content)  ← 瞬间更新
  ├─ 标签到达 → renderTagSelector(tags)
  └─ 骨架屏 → 淡出，正常编辑器内容展示
```

**关键变化**：

1. 动画不再等待任何异步操作
2. 数据加载和 CM6 初始化移到动画播放之后
3. CM6 先用空内容初始化（轻量？看下方分析），再用 `setEditorContent` 填入内容

### CM6 初始化分析

`initCodeMirror(container, '')` 和 `initCodeMirror(container, content)` 的主要成本差异：

* `EditorState.create({doc, extensions})` — doc 为空字符串时，解析开销极低（无语法树构建）

* 15+ 个扩展的实例化与 doc 长度无关

* `new EditorView()` — DOM 创建与 doc 无关

所以用空内容初始化和用实际内容初始化，**主要成本差异几乎为 0**。这意味着我们不需要等动画完成再初始化 CM6，而是可以在动画播放的同时初始化 CM6。

## 变更清单

### 文件 1：`frontend/src/css/components/editor.css`

在 `.cm-editor` 相关样式后（例如在 `cmFadeIn` 关键帧之后），新增骨架屏样式：

```css
/* ── 编辑器骨架屏 ── */
.editor-skeleton {
    padding: 0 4px;
}
.editor-skeleton-line {
    height: 14px;
    margin-bottom: 12px;
    border-radius: 4px;
    background: linear-gradient(90deg,
        var(--border) 25%,
        var(--hover-bg) 50%,
        var(--border) 75%
    );
    background-size: 200% 100%;
    animation: skeletonPulse 1.5s ease-in-out infinite;
}
.editor-skeleton-line:nth-child(1) { width: 92%; }
.editor-skeleton-line:nth-child(2) { width: 78%; }
.editor-skeleton-line:nth-child(3) { width: 85%; }
.editor-skeleton-line:nth-child(4) { width: 60%; }

@keyframes skeletonPulse {
    0%, 100% { background-position: 200% 0; }
    50% { background-position: -200% 0; }
}
```

设计说明：

* 使用 `background-position` 滑动的 shimmer 效果，视觉上更接近 VS Code 的加载效果

* 4 条不同宽度的线条（92%/78%/85%/60%）模拟自然文本段落

* `--border` 和 `--hover-bg` 为变量，跟随主题色

* `animation: skeletonPulse 1.5s` — 慢速呼吸

### 文件 2：`frontend/src/main.js`

#### 2.1 重构 `openEditor` 函数（L3164-L3384）

将函数主体分为三个阶段：

**阶段一：面板立即展示（同步，在 await 之前）**

```js
async function openEditor(noteId, readOnly, startFullscreen, hideEditBtn) {
    state.editingNoteId = noteId || null;
    state.selectedTags = [];

    const isReadOnly = readOnly && noteId != null;
    let noteData = null;
    let editorContent = '';

    // 笔记元数据从 state.notes 中同步获取（已有缓存）
    if (noteId) {
        noteData = state.notes.find((n) => n.id === noteId) || null;
        if (noteData) {
            els.editorNoteTitle.value = noteData.title || '';
            state.selectedTags = (noteData.tags || []).map((t) => t.id);
        }
    } else {
        // 新建模式：默认标题
        const now = new Date();
        const pad = (n) => String(n).padStart(2, '0');
        els.editorNoteTitle.value = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())} ☺️`;
        state._defaultNewNoteTitle = els.editorNoteTitle.value;
    }

    // 只读/编辑模式 UI 切换（同步）
    els.editorNoteTitle.readOnly = isReadOnly;
    els.editorNoteTitle.classList.toggle('editor-input-readonly', isReadOnly);
    els.editorSaveBtn.style.display = isReadOnly ? 'none' : '';
    els.editorCancelBtn.style.display = isReadOnly ? 'none' : '';
    els.editorPanel.classList.toggle('editor-view-mode', isReadOnly);
    if (els.editorTypeToggle)
        els.editorTypeToggle.style.display = isReadOnly ? 'none' : '';
    els.editorEditBtn.style.display = (isReadOnly && !hideEditBtn) ? '' : 'none';
    els.editorViewBtn.style.display = (!isReadOnly && state.enteredFromViewMode) ? '' : 'none';
    els.editorFileExt.classList.toggle('file-ext-readonly', !!isReadOnly);

    // 文件后缀和模式切换
    const ext = (noteData && noteData.file_ext) || '.txt';
    const isMd = ext === '.md';
    els.editorModes.style.display = isMd ? '' : 'none';
    updateFileExtDisplay(ext);
    if (els.tocToggleBtn)
        els.tocToggleBtn.classList.toggle('show-in-preview', isMd);
    if (els.editorTypeToggle) {
        els.editorTypeToggle.textContent = isMd ? 'M' : 'T';
        els.editorTypeToggle.title = isMd ? '切换为纯文本格式' : '切换为 Markdown 格式';
    }

    els.editorOverlay.dataset.mode = 'edit';

    // 查看模式：预览状态显示
    if (isReadOnly && noteData) {
        els.editorEditTime.textContent = '最近编辑 ' + formatTime(noteData.updated_at || noteData.created_at);
        if (!isMd) {
            els.mdRendered.style.display = 'none';
            _setPreviewLayout(false);
            _closeToc();
        } else {
            els.editorOverlay.dataset.mode = 'preview';
            // 查看模式的 markdown 预览需要内容，在阶段二处理
        }
    } else {
        els.editorEditTime.textContent = '';
        switchEditorMode('edit');
    }

    // 标题输入监听
    if (!isReadOnly) {
        els.editorNoteTitle.addEventListener('input', onEditorInput);
        state._titleInputListenerAttached = true;
    } else {
        els.editorNoteTitle.removeEventListener('input', onEditorInput);
        state._titleInputListenerAttached = false;
    }

    // ── 立即显示面板 + 骨架屏 ──
    els.mainContent.style.overflow = 'hidden';
    els.viewEditor.classList.add('active');

    // 显示骨架屏（仅在 CM6 位置放置 shimmer）
    const contentArea = document.getElementById('editorNoteContent');
    contentArea.innerHTML = ''
        + '<div class="editor-skeleton">'
        + '<div class="editor-skeleton-line"></div>'
        + '<div class="editor-skeleton-line"></div>'
        + '<div class="editor-skeleton-line"></div>'
        + '<div class="editor-skeleton-line"></div>'
        + '</div>';

    // 启动动画
    const overlay = els.editorOverlay;
    const panel = els.editorPanel;
    const body = panel.querySelector('.editor-body');
    document.getElementById('topbar').classList.add('editor-fullscreen');

    if (startFullscreen) {
        panel.style.transition = 'none';
        overlay.classList.add('fullscreening');
        panel.classList.add('fullscreen');
        void panel.offsetHeight;
        panel.style.transition = '';
        state._isFullscreen = true;
        if (els.editorFullscreenBtn) { /* ... */ }
        if (els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed'))
            els.notebookSidebar.classList.add('collapsed');
        overlay.style.opacity = '1';
        panel.style.opacity = '1';
        panel.style.transform = 'scale(1)';
        // 全屏模式（快速笔记）不需要入场动画，但骨架屏仍然显示
    } else {
        overlay.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
        panel.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
        if (body) {
            body.style.animation = 'viewEnter 0.25s ease-out forwards';
        }
    }

    // ── 阶段二：后台加载数据 + CM6 ──
    // 此处不 await，让面板动画和 CM6 初始化并行执行
    // 使用 Promise.all 并行加载
    try {
        const loadPromises = [];

        // 加载完整内容
        const contentPromise = (async () => {
            let fullContent = '';
            if (noteId) {
                if (noteData && noteData.content) {
                    // 优先使用列表缓存的内容
                    fullContent = noteData.content || '';
                }
                try {
                    if (window.go.main.App.GetNoteContent) {
                        fullContent = await window.go.main.App.GetNoteContent(noteId) || '';
                    }
                } catch (err) {
                    console.error('获取完整笔记内容失败:', err);
                }
            }
            editorContent = fullContent;
        })();
        loadPromises.push(contentPromise);

        // 加载标签（异步）
        const tagsPromise = loadTagsForEditor(isReadOnly);
        loadPromises.push(tagsPromise);

        await Promise.all(loadPromises);
    } catch (err) {
        console.error('编辑器数据加载失败:', err);
    }

    // 移除骨架屏
    contentArea.innerHTML = '';

    // 初始化 CM6（已拿到内容）
    const useSyntaxHighlight = els.mdHighlightToggle.checked;
    initCodeMirror(contentArea, editorContent, isReadOnly, useSyntaxHighlight, ext, codeHighlightTheme);
    updateWordCount();

    // 编辑模式下记录快照
    if (!isReadOnly && state.editingNoteId) {
        state._editSnapshot = {
            title: els.editorNoteTitle.value.trim(),
            content: getEditorContent().trim(),
            tags: [...state.selectedTags].sort(),
            fileExt: els.editorFileExt.textContent
        };
    }

    // 查看模式：Markdown 预览
    if (isReadOnly && isMd && els.editorOverlay.dataset.mode === 'preview') {
        updatePreview();
    }

    // 新建笔记时自动聚焦
    if (!state.editingNoteId && els.editorOverlay.dataset.mode !== 'preview' && document.hasFocus()) {
        window.focus();
        cmEditor?.focus();
    }
}
```

**关键改动说明**：

| 改动                                                  | 说明               |
| --------------------------------------------------- | ---------------- |
| `state.notes.find()` 同步获取标题/标签                      | 列表缓存中已有，无需等待后端接口 |
| `contentArea.innerHTML = skeleton`                  | 在 CM6 初始化前显示骨架屏  |
| `Promise.all([contentPromise, tagsPromise])`        | 内容 + 标签并行加载，互不阻塞 |
| `contentArea.innerHTML = ''` 在先，`initCodeMirror` 在后 | 清除骨架屏后立即初始化 CM6  |
| 全屏模式（快速笔记）不加动画                                      | 保持现有行为不变，但骨架屏仍显示 |

#### 2.2 处理 CM6 实例复用——提前初始化（可选优化）

当前策略是每次 `openEditor` 销毁并重建 CM6。但考虑到 CM6 初始化是最重操作，可以：

**方式 A（计划采用）**：每次 destroy + recreate，但在动画播放期间初始化空内容，数据到齐后 `setEditorContent`。——已经够快。

**方式 B（不采用）**：CM6 单例，只初始化一次，后续仅做 dispatch。需要处理 只读/可编辑模式切换时插件列表的变化，涉及 CM6 Compartment 复杂管理，超出本次范围。

### 文件 3：`frontend/src/css/components/editor.css` — 调用 `cmFadeIn`

为了在骨架屏移除 → CM6 出现之间有个平滑过渡，给 `#editorNoteContent`（挂载点）加淡入：

```css
#editorNoteContent.cm-loading {
    opacity: 0;
}
#editorNoteContent.cm-ready {
    animation: cmFadeIn 0.2s ease-out forwards;
}
```

在 JS 中：`initCodeMirror` 之前加 `contentArea.classList.add('cm-loading')`，之后 `classList.remove('cm-loading'); classList.add('cm-ready')`，或直接在 `initCodeMirror` 第一行做。

实际上更简单：在 `contentArea.innerHTML = ''`（骨架屏移除）之后，CM6 的 DOM 由 `new EditorView()` 创建，天然有一个从空到有内容的渲染过程。如果发现闪烁，再加 `cmFadeIn`。

### `closeEditor` 无需改动

* `destroyCodeMirror()` 仍会在 `closeEditor` 中调用

* 骨架屏在 CM6 创建前已被移除，`closeEditor` 不需要知道骨架屏的存在

* `contentArea.innerHTML = ''` 已在 `closeEditor` 的 setTimeout 中（通过 `destroyCodeMirror` + DOM 移除）

### 关于 `AGENTS.md`

新增第一百九十三节记录。

## 特别场景处理

| 场景                  | 处理方式                                                              |
| ------------------- | ----------------------------------------------------------------- |
| **新建笔记**            | 无异步内容加载（`editorContent = ''`），骨架屏 → CM6 空编辑器，几乎瞬间完成               |
| **快速笔记（全屏）**        | 跳过入场动画，但骨架屏流程走通（快速笔记通常内容短，CM6 初始化也快）                              |
| **召回卡片打开笔记**        | 与"从笔记列表打开"走同一路径，无差别                                               |
| **查看模式 + Markdown** | `updatePreview()` 在 CM6 初始化后调用，Worker 渲染自然有延迟（已有 loading spinner） |
| **查看模式 + 纯文本**      | CM6 只读实例，内容加载后直接展示，无额外渲染                                          |
| **慢速设备**            | 骨架屏 shimmer 动画持续到 CM6 就绪为止，无感知突变                                  |
| **加载失败**            | `console.error` 已有，骨架屏会卡住 → 需补充：加载失败时骨架屏替换为错误提示                   |

## 验证步骤

1. **点击笔记卡片验证**：点击后立即看到面板弹入动画（scale 0.85→1），骨架屏显示，无等待感
2. **骨架屏验证**：初始显示 4 条 shimmer 脉冲线条，和编辑器最终内容无重叠
3. **内容加载验证**：内容加载完成后骨架屏替换为 CM6 编辑器，内容正确
4. **标签加载验证**：标签区正确显示笔记关联的标签
5. **新建笔记验证**：新建笔记时无骨架屏（内容为空），面板立即弹入
6. **快速笔记（全屏）验证**：全屏模式无弹入动画，但无卡顿
7. **查看模式验证**：只读查看时内容正确展示，Markdown 预览正常渲染
8. **切换笔记验证**：关闭当前的 doc → 打开另一个 doc，骨架屏出现并正确替换
9. **构建验证**：`wails build` 成功

## 不变的部分

| 方面                     | 原因                                |
| ---------------------- | --------------------------------- |
| `closeEditor` 逻辑       | 清理流程不变，骨架屏在 CM6 初始化前已被覆盖          |
| 编辑模式/只读模式 UI           | 逻辑不变，只是状态设置在动画之前                  |
| `loadTagsForEditor` 函数 | 签名不变，调用位置从 await 改为并行 Promise     |
| 后端接口                   | `GetNoteContent`、`GetAllTags` 等不变 |
| CM6 插件/扩展              | `initCodeMirror` 的参数和逻辑不变         |
| 全屏模式行为                 | 动画跳过但骨架屏流程一致                      |

