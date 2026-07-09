# 编辑器拖拽图片上传 — 实现计划

## 概要

在 `OnFileDrop` 回调中通过 `elementFromPoint(x, y)` 判断拖放目标是否在 CM6 编辑器上，若是且为 `.md` 编辑/新建模式、且文件为图片，则路由到图片上传逻辑而非笔记导入。需在后端新增 `SaveImageFromPath` 方法读取本地文件路径。

## 现状分析

- `main.go:79` — `EnableFileDrop: true`，Wails 在 OS 层拦截所有文件拖入
- `main.js:6881-6931` — `initFileDrop()`：HTML5 `dragenter/dragover/dragleave/drop` 仅控制遮罩显隐，文件处理走 `window.runtime.OnFileDrop`
- `main.js:6920-6929` — `OnFileDrop` 回调：所有文件一律调 `handleFileDropPaths` → `App.ImportFiles` 创建为笔记
- `main.js:128-167` — `handlePaste()`：粘贴图片时 `FileReader` + `App.SaveImage(base64)` 上传 + 插入 Markdown
- `app.go:133-161` — `SaveImage(name, data string)`：接受 base64，写入 `~/.jot/images/`
- `.cm-editor` 是编辑器根 DOM，可通过 `elementFromPoint` + `closest('.cm-editor')` 定位

## 改动方案

### Go 后端 — 新增 `SaveImageFromPath`

**文件**: [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)（在 `SaveImage` 附近添加）

新增方法 `SaveImageFromPath(localPath string) (string, error)`：
1. `os.ReadFile(localPath)` 读取本地文件
2. 生成 UUID 文件名
3. `os.WriteFile` 写入 `~/.jot/images/`
4. 返回 `/images/uuid_name.ext`

与 `SaveImage` 的区别只在于输入：一个接 base64，一个接本地路径。内部可抽取共享逻辑（生成 UUID + 写入），但考虑到行数不多，直接复制实现即可。

### 前端 — `OnFileDrop` 回调路由分流

**文件**: [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L6920-L6929)

修改 `OnFileDrop` 回调逻辑：

```
OnFileDrop(x, y, paths)
  │
  ├─ 1. elementFromPoint(x, y) → target
  ├─ 2. target.closest('.cm-editor') → 是否拖到编辑器上？
  │      AND editorFileExt === '.md'
  │      AND editorNoteTitle 非只读（编辑/新建模式）
  │
  ├─ YES → 路径分流：
  │      ├─ 图片路径 → SaveImageFromPath → 插入 ![](/images/xxx)
  │      └─ 非图片路径 → handleFileDropPaths（创建为笔记）
  │
  └─ NO  → handleFileDropPaths (原逻辑不变)
```

具体实现：

```javascript
// 在 OnFileDrop 回调内
const target = document.elementFromPoint(x, y);
const cmEditorEl = target?.closest('.cm-editor');
const isMdEditMode = cmEditorEl && 
    els.editorFileExt.textContent === '.md' && 
    !els.editorNoteTitle.readOnly;

if (isMdEditMode) {
    _dragCounter = 0;
    if (dropOverlay) dropOverlay.classList.remove('active');
    
    const imgPaths = paths.filter(p => /\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(p));
    const otherPaths = paths.filter(p => !/\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(p));
    
    for (const p of imgPaths) {
        try {
            const url = await window.go.main.App.SaveImageFromPath(p);
            const filename = p.split(/[/\\]/).pop();
            const markdown = `![${filename}](${url})`;
            // 在光标处插入
            const pos = cmEditor.state.selection.main.head;
            cmEditor.dispatch({
                changes: { from: pos, insert: markdown },
                selection: { anchor: pos + markdown.length }
            });
        } catch (err) { ... }
    }
    if (otherPaths.length > 0) handleFileDropPaths(otherPaths, ...);
} else {
    handleFileDropPaths(paths, ...);
}
```

### 前端 — 编辑器拖拽悬停视觉反馈

**文件**: [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L6887-L6914) + [editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css)

在现有的 `dragenter`/`dragleave` 事件中，用 `e.target` 或 `document.elementFromPoint` 检测是否悬停在编辑器区域，给 `.cm-editor` 切换 `dragover` 类。

CSS 新增：
```css
.cm-editor.dragover {
  outline: 2px dashed var(--accent-color, #6c63ff);
  outline-offset: -2px;
  border-radius: 8px;
}
```

注意：Wails `OnFileDrop` 会拦截 OS 级的 drop，但不影响 HTML5 `dragenter`/`dragover`/`dragleave` 事件（这些事件仍然在 DOM 中正常冒泡）。

### 边界情况处理

| 场景 | 处理 |
|------|------|
| 拖拽到**编辑器外**（卡片网格、空白区域） | 走原有 `handleFileDropPaths` |
| 拖拽**非图片**到编辑器 | 不走图片上传，这些文件转 `handleFileDropPaths` 创建为笔记 |
| 拖拽**混合文件**（图片 + 非图片）到编辑器 | 图片上传 + 插入，非图片创建笔记 |
| **非 `.md`** 笔记拖拽图片 | 不处理（默认走 `handleFileDropPaths` 创建笔记） |
| **查看模式**（只读）拖拽图片 | 同上，不处理 |
| 拖拽到编辑器上但**编辑器已关闭/不存在** | `elementFromPoint` 获取不到 `.cm-editor`，走原逻辑 |
| 连续拖拽多张图片 | 依次上传 + 依次插入，光标定位到最后一张末尾 |

## 假设与决策

1. **不改 `EnableFileDrop` 配置** — 保持原有拖拽导入笔记功能不变
2. **不依赖 CM6 原生 `drop` 事件** — 因为 `dataTransfer.files` 在 Wails 拦截后可能为空，改用 `OnFileDrop` 的 `paths` 路径参数
3. **新增 Go 方法而非改造现有方法** — `SaveImageFromPath` 和 `SaveImage` 各自独立，不改已有粘贴逻辑
4. **不抽取共享 go 函数** — 两个方法仅 ~15 行，直接写不抽象，避免过度工程

## 验证步骤

1. `go build ./...` 编译通过
2. `npx vite build` 前端构建通过
3. 运行时测试：
   - 拖拽一张 `png` 到 `.md` 编辑器 → 上传成功 + 插入 `![]()` + 光标定位正确
   - 拖拽 `txt` 到 `.md` 编辑器 → 不做上传，创建为新笔记
   - 混合拖拽（图片+txt）→ 图片上传，txt 创建笔记
   - 拖拽到编辑器外 → 全部创建为笔记（原逻辑不变）
   - 查看模式拖拽图片 → 全部创建为笔记
   - 拖拽悬停时编辑器显示虚线边框，松开后消失
