# 笔记本重命名改为对话框方案

## 摘要
将笔记本侧边栏的内联重命名（contentEditable）替换为弹出对话框方式，复用已有的 `new-notebook-overlay` / `new-notebook-dialog` CSS 模式。

## 当前状态
- 右键菜单"重命名"和双击笔记本名称均调用 `startInlineRename(notebookId, currentName)`
- 内联方案用 `contentEditable` 在原 `.notebook-name` 上编辑，CSS 用 `margin: -7px -10px -7px -20px` 铺满条目
- 已有 `showNewNotebookDialog()` 是完美的参考模板（overlay + dialog + input + confirm/cancel）

## 变更方案

### 1. JS — `main.js`：重写 `startInlineRename` → `showRenameNotebookDialog`

将 `startInlineRename` 函数（第7366行起）替换为基于对话框的 `showRenameNotebookDialog(notebookId, currentName)`：

- 创建 overlay（复用 `.new-notebook-overlay` / `.new-notebook-dialog` 类名）
- dialog 标题改为「重命名笔记本」
- input 预填 `currentName`，`maxlength=50`，自动聚焦+全选
- 按钮：取消 + 确认（居中显示，`justify-content: center`）
- 确认逻辑：调用 `window.go.main.App.RenameNotebook(notebookId, newName)` → `loadNotebooks()` → toast
- 关闭：overlay 移除 + `setTimeout 200ms` 清理
- 变量 `_renameNotebookDialog` 跟踪当前打开的对话框实例，防止重复打开

### 2. JS — `main.js`：全局 ESC 处理（`handleKeyboardNavigation`，约第6522行）

在 ESC 处理链中新增分支（在确认框 `confirmDialog` 之前）：
```
if (_renameNotebookDialog) { closeRenameDialog(); return; }
```

### 3. CSS — `sidebar.css`：清理内联编辑残留样式

删除不再需要的样式：
- `.notebook-name[contenteditable="true"]` 规则（第168-176行）
- `.notebook-item:has(.notebook-name[contenteditable="true"]) .notebook-badge`（第178-180行）
- `.notebook-item:has(.notebook-name[contenteditable="true"])::before`（第182-184行）
- `.notebook-rename-input` 规则（第318-331行）— 早已不使用

### 4. CSS — `sidebar.css`：对话框按钮居中

新建笔记本和重命名对话框共用结构，在 `.new-notebook-actions` 已有 `justify-content: center`，无需额外修改。

## 涉及文件
- `frontend/src/main.js` — `startInlineRename` 重写 + ESC 分支
- `frontend/src/css/components/sidebar.css` — 清理内联编辑 CSS

## 验证
1. 右键笔记本 → 点击"重命名" → 弹出对话框，input 预填名称
2. 双击笔记本名称 → 同样弹出对话框
3. 输入新名称 → 点确认 → 名称更新 + toast
4. 点取消 / 点遮罩 / 按 ESC → 对话框关闭
5. 对话框打开时按 Enter → 确认重命名
6. 默认笔记本（id=1）双击/右键不弹对话框（已有 disabled 保护）
