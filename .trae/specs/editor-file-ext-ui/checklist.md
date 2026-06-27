# Checklist

- [x] 后端 `UpdateNoteFileExt` API 实现并按预期工作
- [x] 编辑器状态栏显示文件后缀 `| .ext`
- [x] 新建笔记时根据 note_type 显示默认后缀
- [x] 点击后缀弹出编辑对话框
- [x] 后缀输入校验生效（非空、以 `.` 开头、长度 2-10、合法字符）
- [x] 保存后缀时调用后端 API 并即时更新显示
- [x] 取消按钮关闭对话框不做修改
- [x] `go build ./...` 编译通过
- [x] `npx vite build` 构建通过
