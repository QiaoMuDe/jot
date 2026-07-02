# Tasks

- [x] Task 1: 修改 `openEditor` 新增 `hideEditBtn` 参数
  - [x] SubTask 1.1: 修改 `openEditor` 函数签名，新增第 4 个参数 `hideEditBtn`（默认 false）
  - [x] SubTask 1.2: 当 `hideEditBtn` 为 true 时，`els.editorEditBtn.style.display = 'none'`
- [x] Task 2: 修改点击回调，改用 `window.openEditor`
  - [x] SubTask 2.1: 修改卡片点击回调，从 `openCardPreview(c)` 改为 `window.openEditor(c.id, true, false, true)`
- [x] Task 3: 清理已废弃的代码
  - [x] SubTask 3.1: 删除 `openCardPreview` 函数
  - [x] SubTask 3.2: 删除 `_cardPreviewWorker` 变量
  - [x] SubTask 3.3: 删除 `aiCardPreviewModal` HTML 结构
  - [x] SubTask 3.4: 删除 CSS 中卡片预览相关样式（`.ai-card-preview-*`）
  - [x] SubTask 3.5: 删除 JS 中预览浮层的事件绑定（previewModal, previewOverlay, previewClose）

# Task Dependencies

- [Task 1] 无依赖
- [Task 2] 依赖 [Task 1]（需要 `openEditor` 的 `hideEditBtn` 参数已就绪）
- [Task 3] 依赖 [Task 2]（确认替换成功后清理旧代码）
