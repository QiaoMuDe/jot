# 待办清单输入区优化 — 执行计划

## 用户需求
解决待办输入框无法录入多行内容的问题，将 `Ctrl+Enter` 映射为换行（与编辑模式保持一致）。

## 当前状态

### 输入区 (index.html #L1606-L1620)
- `<textarea id="todoInput" class="todo-input">` — auto-expand textarea，已有
- `<button class="todo-submit-btn" id="todoSubmitBtn">` — 勾号提交按钮，已有

### 键盘事件 (main.js #L5544-L5551)
- `Enter` → 提交 ✅（保留）
- `Shift+Enter` → 换行 ❌（需改为 `Ctrl+Enter`）

### 编辑模式对照 (main.js #L8770-L8784)
```js
// editTodo 模式：
Enter (无 modifier) → 保存
Ctrl+Enter         → 插入换行
Escape             → 取消
```

## 改动清单

### 1. JS — 键盘快捷键变更
**文件**: `frontend/src/main.js` #L5544-L5551

当前：
```js
els.todoInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        addTodo();
    }
});
```

改为：
```js
els.todoInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.ctrlKey && !e.shiftKey) {
        e.preventDefault();
        addTodo();
    } else if (e.key === 'Enter' && e.ctrlKey) {
        e.preventDefault();
        // 插入换行 + 触发 auto-resize
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        textarea.value = textarea.value.substring(0, start) + '\n' + textarea.value.substring(end);
        textarea.selectionStart = textarea.selectionEnd = start + 1;
        autoResizeTodoInput();
    }
});
```

**逻辑**: `Ctrl+Enter` 时手动插入 `\n` 并调用 `autoResizeTodoInput()` 更新 textarea 高度，与 `editTodo` 中的做法一致。

### 2. JS — 注释更新
**文件**: `frontend/src/main.js` #L5545

注释从 `// 键盘：Enter 提交，Shift+Enter 换行` 改为 `// 键盘：Enter 提交，Ctrl+Enter 换行`

### 3. HTML — title 属性更新
**文件**: `frontend/index.html` #L1609

提交按钮 title 从 `"添加 (Enter)"` 改为 `"添加 (Enter) | Ctrl+Enter 换行"`，给用户更清晰的提示。

## 验证
- `npx vite build` 构建无报错
- 输入文字后按 Enter → 提交
- 输入文字后按 Ctrl+Enter → 插入换行，textarea 自动扩展高度
- 输入文字后点击提交按钮 → 提交
