# Checklist

- [x] 后端 `BatchRemoveTagFromNotes(noteIDs, tagID)` 方法签名与 `BatchAddTagToNotes` 对称，跳过不存在的笔记
- [x] `app.go` 中 `BatchRemoveTagFromNotes` 绑定方法正确调用 service 层
- [x] `index.html` 批量操作栏新增「批量添加标签」和「批量移除标签」按钮
- [x] 标签选择弹窗（`.batch-tag-overlay`）在点击按钮时弹出，展示所有可用标签
- [x] 点击某个标签后立即执行操作并关闭弹窗
- [x] 无可用标签时弹窗显示「暂无标签」提示
- [x] 点击弹窗外区域关闭弹窗
- [x] 操作完成后刷新笔记列表（`loadNotes()`）并显示成功通知
- [x] 前端 `vite build` 0 errors
- [x] 后端 `golangci-lint` 0 issues
