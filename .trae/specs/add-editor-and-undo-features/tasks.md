# Tasks

- [x] Task 1: 字数统计
  - [x] 在 `index.html` 编辑器底部添加字数显示区域（`#editorWordCount`）
  - [x] 在 `main.js` 中添加 `updateWordCount()` 函数，监听标题和内容 input 事件，实时计算字数统计
  - [x] 在 `style.css` 中添加字数统计样式

- [x] Task 2: 笔记复制
  - [x] 在 `main.js` 右键菜单添加「复制笔记」选项（`data-action="copy"`）
  - [x] 在 `handleContextAction` 中添加 `'copy'` 分支，调用 `CreateNote` 创建副本
  - [x] 复制完成后刷新列表并 Toast 提示

- [x] Task 3: 自动保存
  - [x] 在 `main.js` `openEditor()` 中，为标题和内容输入框添加 `input` 事件监听
  - [x] 实现 `startAutoSave()` 函数：3 秒 debounce，检测笔记有 ID 后调用 `UpdateNote`
  - [x] 编辑器关闭时清除自动保存定时器

- [x] Task 4: 回收站撤销 Toast
  - [x] 在 `index.html` 中添加 Toast 容器（`#undoToast`）
  - [x] 在 `style.css` 中添加 Toast 样式（底部固定、渐入渐出动画）
  - [x] 在 `main.js` 中实现 `showUndoToast()` 函数：5 秒倒计时，点击「撤销」调用 `RestoreNote`
  - [x] 修改 `deleteNote()` 函数，删除后显示撤销 Toast 而非 `confirm()`

# Task Dependencies
- 四个任务之间无依赖，可并行开发
- 每个任务只涉及前端文件修改
