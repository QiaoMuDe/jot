# 编辑器标题「未保存改动」星号（记事本逻辑）

## 摘要

在**编辑已有笔记的编辑模式**与**新建笔记**下，当标题 / 正文 / 标签 / 扩展名相对「打开编辑器时记录的初始快照」发生任何改动且尚未保存时，在编辑器顶部标题前显示一个星号 `*`；一旦保存、切回查看或关闭，星号消失。查看模式不显示。全部改动集中在纯前端，后端零改动。

## 当前状态分析（基于代码调研）

- 快照基准已存在：`state._editSnapshot`，在进入编辑模式时记录 `{ title, content, tags, fileExt }`。
  - `switchEditorReadOnly(false)` 记录（`frontend/src/main.js` L361-L367）
  - `openEditor` 编辑分支记录（L4160-L4167）
  - `switchEditorReadOnly(true)` 切回查看时置 `null`（L368-L370）
  - `closeEditor` 清理时置 `null`（L5228）
- 脏检测比较逻辑已存在并散落多处（对比 `_editSnapshot`）：
  - 查看按钮 `editorViewBtn` click：L6350-L6358
  - `updateNote` 保存前脏检测：L1119-L1129
  - `closeEditorSafe` 脏检测：L5283-L5290
- 变更事件源（编辑模式下均会触发，可用于刷新星号）：
  - 标题 `input` → `onEditorInput()`（L4011 绑定；L5071 定义）
  - CM6 内容 `docChanged` → `onEditorInput()`（L141-L145）
  - 标签选择 `window.toggleEditorTag`（L5554-L5572）
  - 扩展名 `toggleFileExt`（L3877）
- 标题 DOM 结构：`index.html` L143-L146：
  ```html
  <div class="editor-title-wrap">
      <input type="text" id="editorNoteTitle" ... />
  ```
- `editor-title-wrap` 为 flex 容器（`editor.css` L106-L112）；`editor-input` 当前 `width:100%`（L115-L135）。
- `_editSnapshot` 目前只在「编辑模式且 `state.editingNoteId` 存在」时建立（L4160），因此新建模式与查看模式无快照、`_editSnapshot === null`。
- 查看模式天然不能产生差异（只读）；**新建模式实际可以有未保存改动**（记事本新建文档有内容未保存同样显示星号）。新建模式打开即有明确初始基准：默认标题 `${date} ☺️`（L3950）、空正文、空标签、默认扩展名 `.txt`（L3947-L3952）。
- `createNote`（L1060-L1104）**不读** `_editSnapshot`、不做脏检测，保存后 `state.editingNoteId === null` 时直接 `closeEditor`（L1099-L1101）。因此给新建模式也建立快照不会影响 `createNote` 的保存逻辑，风险极低。

## 变更方案

### 1. `frontend/index.html`
在 `.editor-title-wrap` 内、`#editorNoteTitle` 之前插入星号元素，并给 title-wrap 加 id：
```html
<div class="editor-title-wrap" id="editorTitleWrap">
    <span class="editor-dirty-star" aria-hidden="true">*</span>
    <input type="text" id="editorNoteTitle" class="editor-input" placeholder="笔记标题" />
```

### 2. `frontend/src/css/components/editor.css`
- `.editor-title-wrap` 增加 `align-items: center;`（让星号与 input 垂直对齐）。
- `.editor-title-wrap .editor-input` 由 `width:100%` 调整为 `flex: 1 1 auto; width: 100%;`（或仅加 `flex: 1 1 auto;`），使其吸收星号占位后的剩余空间，维持原有伸展行为。
- 新增星号样式（默认隐藏，`editor-dirty` 时显示，跟随主题 CSS 变量）：
```css
.editor-dirty-star {
  display: none;
  color: var(--accent);
  font-size: 1.05rem;
  font-weight: 700;
  line-height: 1;
  margin-right: 2px;
  flex: 0 0 auto;
  user-select: none;
}
.editor-title-wrap.editor-dirty .editor-dirty-star {
  display: inline;
}
```

### 3. `frontend/src/main.js`
新增两个纯函数，并接入刷新点。

**a) 新增统一脏检测函数（收敛现有散落比较）**
```js
function isEditorDirty() {
    const snapshot = state._editSnapshot;
    if (!snapshot) return false;
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    const tagsChanged = JSON.stringify([...state.selectedTags].sort()) !== JSON.stringify(snapshot.tags);
    const extChanged = els.editorFileExt.textContent !== snapshot.fileExt;
    return title !== snapshot.title || content !== snapshot.content || tagsChanged || extChanged;
}
```

**b) 新增星号刷新函数**
```js
function refreshDirtyStar() {
    els.editorTitleWrap?.classList.toggle('editor-dirty', isEditorDirty());
}
```

**c) els 引用补齐**：在 els 对象（L500 附近）增加 `editorTitleWrap: $('editorTitleWrap')`。

**d) 放宽快照建立条件（关键：让新建模式也能显示星号）**
将 `openEditor` 阶段二的快照建立条件（L4160 `if (!isReadOnly && state.editingNoteId)`）放宽为 `if (!isReadOnly)`，并在该处补建快照。这样新建模式（无 id）也会在打开时记录 `{ 默认标题, '', [], fileExt }`，`isEditorDirty()` 无特殊分支即可覆盖新建，改动任意字段即显示星号；`createNote` 保存后 `closeEditor` 清空快照 → 星号消失。注释同步更新说明「新建与编辑模式均记录快照」。

**e) 接入刷新点**（编辑模式变更即时刷新 + 快照生命周期刷新）：
- `onEditorInput()` 末尾（L5079 后）调用 `refreshDirtyStar()` —— 一次覆盖标题 input 与 CM6 内容变化两个来源。
- `toggleEditorTag` 末尾（L5572 前）调用 `refreshDirtyStar()` —— 标签变化。
- `toggleFileExt` 末尾调用 `refreshDirtyStar()` —— 扩展名变化。
- `switchEditorReadOnly` 末尾（L370 后）调用 `refreshDirtyStar()` —— 覆盖「进入编辑（快照建立，不脏）」「保存后切回查看/关闭（快照清空，不脏）」等所有通过该函数的转换路径，避免遗漏 viewBtn 保存与切回查看场景。
- `closeEditor` 清理段（L5231 附近，`_editSnapshot = null` 之后）调用 `refreshDirtyStar()`，兜底关闭编辑器时强制隐藏星号。

说明：`viewBtn` 保存路径最终回调 `switchEditorReadOnly(true)`，`updateNote`/`createNote` 保存后回调 `closeEditor`，均由上述刷新点覆盖；无需再在单点重复调用，保持改动最小。

## 假设与决策

- 星号语义 = 记事本「未保存改动」：编辑**已有笔记**产生未保存改动、或**新建笔记**相对初始默认态产生改动时显示。
- 快照基准 = 打开编辑器那一刻记录的值（编辑模式 = 数据库加载进编辑器的值；新建模式 = 默认标题/空正文/空标签/默认扩展名）。星号仅对比「当前输入 vs 打开时快照」，不直接查询数据库。
- 新建模式也建立快照（放宽 L4160 条件），`createNote` 不读快照、不受影响。
- 查看模式只读、无法产生差异，不显示星号。
- 星号颜色采用每套主题的 `--accent`（温和高亮），沿用 `--text-primary` 字号风格，保证 14 套主题下可见（不使用硬编码颜色）。
- 不改动现有散落的脏检测比较逻辑（仍各自保留），仅为星号新增统一判断函数，降低回归风险。若后续需重构，可统一收口。

## 验证步骤

1. 前端重建：`npm run build`（在 `frontend` 目录），随后 `wails build` 重新编译（CSS/HTML 改动需重建资源，见项目约定）。
2. 打开一篇**已有**笔记 → 点击「编辑」进入编辑模式：标题前**无**星号。
3. 修改标题 / 正文 / 切换标签 / 切换扩展名（任一）：标题前**立即出现** `*`。
4. 改完再改回原值：星号消失（与快照一致）。
5. 保存（`editorViewBtn` 或保存按钮）：星号消失。
6. 编辑中有改动后切回查看 / 关闭编辑器：星号消失。
7. **新建笔记**：打开时无星号；输入正文 / 改标题 / 加标签 / 切扩展名后星号出现；保存后星号消失。
8. **查看模式**：任何情况下均**不**显示星号。
9. 切换多套系统主题（如深/浅色主题）复核星号颜色可见性。