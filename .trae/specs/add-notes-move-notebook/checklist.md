# Checklist - 笔记跨笔记本迁移

## 后端迁移方法
- [x] `NoteService.MoveToNotebook()` 单条迁移正确执行（校验 + UpdateColumn）
- [x] `NoteService.BatchMoveToNotebook()` 批量迁移正确执行
- [x] 迁移时不更新时间戳 UpdatedAt
- [x] 目标笔记本不存在时返回错误
- [x] `App.MoveNoteToNotebook()` 绑定方法正确暴露
- [x] `App.BatchMoveNotesToNotebook()` 绑定方法正确暴露

## 前端选择器弹窗
- [x] 弹窗 HTML 结构完整（遮罩、卡片、标题、列表、按钮）
- [x] 弹窗 CSS 样式正确（遮罩 blur、卡片圆角阴影、入场弹簧动画）
- [x] radio 列表自定义样式正确（隐藏原生 radio，accent 色填充）
- [x] 空状态处理：无其他笔记本时显示提示 + 仅「关闭」按钮
- [x] 6 主题 CSS 变量自适应

## 选择器逻辑 + 迁移函数
- [x] `openMoveDialog(noteIds)` 正确加载并过滤笔记本列表（排除当前笔记本）
- [x] `closeMoveDialog()` 正确关闭弹窗
- [x] `confirmMoveNotes()` 正确调用后端迁移 API
- [x] 迁移后：`loadNotes()` + `loadNotebooks()` + `renderNotebookList()` 正确执行
- [x] Toast 通知文案正确显示迁移数量和目标笔记本名称
- [x] 批量迁移（多条笔记）正常工作
- [x] 单条迁移（右键菜单）正常工作

## 右键菜单入口
- [x] 右键菜单在「编辑」和「置顶」之间显示「移动到...」
- [x] 点击「移动到...」打开选择器弹窗

## 批量工具栏入口
- [x] 批量栏在「-标签」和「批量删除」之间显示「移动到」按钮
- [x] 点击「移动到」打开选择器弹窗，携带已选笔记 IDs

## 验证
- [x] `go vet ./...` 0 issues
