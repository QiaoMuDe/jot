# 退出程序快捷键 + 关闭按钮保存笔记

## 摘要

1. 添加 `Ctrl+Q` 全局快捷键退出程序，退出前弹出保存确认对话框
2. 修复窗口关闭按钮（×）直接 `Quit()` 不提示保存笔记的问题
3. 新增 `showSaveConfirmDialog()` 三选一对话框（保存并退出 / 不保存 / 取消）
4. 更新快捷键说明面板

## 当前状态分析

| 项 | 位置 | 现状 |
|---|---|---|
| **`Quit()` import** | [line 3](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3) | ✅ 已导入 |
| **键盘处理入口** | [line 4083](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L4083) | `handleKeyboardNavigation()`，已有 `Ctrl+E`、`F11`、`Escape` |
| **窗口关闭按钮（×）** | [line 5051](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L5051) | ❌ 直接 `Quit()`，**不保存编辑器内容** |
| **自动保存** | [line 2424](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2424) | `startAutoSave()` 3 秒防抖，编辑态调 `UpdateNote()`，新建态调 `SaveDraft()` |
| **快捷键面板** | [line 4400-4426](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L4400-L4426) | 数组 `shortcuts.map(...)` 渲染 |

## 变更计划

### 1. HTML — 确认对话框添加「不保存」按钮

在 `index.html` 的 confirm-actions 中追加一个 `confirm-third` 按钮，文本为"不保存"。

### 2. CSS — 添加 `.confirm-third` 样式

在 `style.css` 中添加黄色警告风格的第三方按钮样式（与 `confirm-ok` 红色危险风格区分）。

### 3. 提取 `saveEditorContent()` 共用函数

在 `closeEditor()` 之前新增可复用的异步函数，供 `handleAppExit()` 调用：

```javascript
/**
 * 保存编辑器内容（退出程序前调用）
 * 编辑态调 UpdateNote，新建态调 SaveDraft
 */
async function saveEditorContent() {
    if (!els.viewEditor.classList.contains('active')) return;
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    if (!title || !content) return;
    if (!window.go || !window.go.main || !window.go.main.App) return;
    try {
        if (state.editingNoteId) {
            await window.go.main.App.UpdateNote(state.editingNoteId, title, content, state.noteType);
        } else {
            await window.go.main.App.SaveDraft(title, content);
        }
    } catch (err) {
        console.error('退出前保存失败:', err);
    }
}
```

### 4. `showSaveConfirmDialog()` 三选一对话框

在 `showConfirmDialog()` 之后新增，返回 `'save' | 'discard' | 'cancel'`。

### 5. `handleAppExit()` 统一退出流程

```javascript
async function handleAppExit() {
    if (编辑器打开且有内容) {
        const action = await showSaveConfirmDialog('笔记内容尚未保存，退出前是否保存？');
        if (action === 'cancel') return;       // 取消：不退出
        if (action === 'save') saveEditorContent();  // 保存后退出
        // action === 'discard': 跳过保存直接退出
    }
    Quit();
}
```

### 6. Ctrl+Q 和关闭按钮使用 `handleAppExit()`

- Ctrl+Q 快捷键块调用 `await handleAppExit()` 替代原保存+退出逻辑
- 窗口关闭按钮（×）同样调用 `await handleAppExit()`
- `handleKeyboardNavigation` 签名改为 `async`

### 7. 更新快捷键面板

在快捷键数组追加 `{ key: 'Ctrl + Q', desc: '退出程序' }`。

### 8. `AGENTS.md` — 更新记忆

- 已实现列表新增条目
- 关键记忆点新增条目

## 假设与决策

| 决策 | 选择 | 理由 |
|------|------|------|
| **保存函数** | 提取共用 `saveEditorContent()` | `handleAppExit()` 和未来可能的其他场景复用 |
| **对话框风格** | 三选一：保存并退出 / 不保存 / 取消 | 给用户完全的控制权 |
| **「不保存」按钮样式** | 黄色警告色，区别于「确定」红色和「取消」灰色 | 视觉层级清晰 |
| **保存方式** | 直接调 `UpdateNote`/`SaveDraft` | 复用 auto-save 已有逻辑，不等待 3s 防抖 |
| **编辑器未打开或内容为空** | 跳过对话框，直接 `Quit()` | 没有需要保存的内容 |
| **保存失败** | 打印错误后仍继续退出 | 不阻止用户退出 |
| **Ctrl+Q 位置** | 放在 F11 之后、Escape 之前 | 分组：窗口操作 → 退出 → 返回 |
| **`handleKeyboardNavigation` 签名** | 改为 `async` | 内部有 `await` 调用 |

## 验证步骤

1. 打开编辑器，输入新内容 → 按 `Ctrl+Q` → 弹出保存对话框 → 选「保存并退出」→ 程序退出，重启检查内容已保存
2. 打开编辑器，输入新内容 → 按 `Ctrl+Q` → 选「不保存」→ 程序直接退出，重启检查内容未保存
3. 打开编辑器，输入新内容 → 按 `Ctrl+Q` → 选「取消」→ 停留在当前页，不退出
4. 不打开编辑器 → 按 `Ctrl+Q` → 无对话框，程序直接退出
5. 打开编辑器，输入内容 → 点窗口关闭按钮（×）→ 弹出保存对话框，行为与 Ctrl+Q 一致
6. 打开快捷键面板（`Ctrl+7`）→ 确认 `Ctrl+Q 退出程序` 条目可见

