# Tasks: CodeMirror 6 集成

## Phase 1: 基础设施搭建

- [ ] Task 1: 安装 CodeMirror 6 npm 依赖包
  - [ ] 安装核心包：@codemirror/state, @codemirror/view, @codemirror/commands
  - [ ] 安装功能包：@codemirror/search, @codemirror/lang-markdown, @codemirror/language
  - [ ] 安装增强包：@codemirror/autocomplete, @lezer/highlight
  - [ ] 验证安装成功，确认无版本冲突

- [ ] Task 2: 创建 CM6 初始化模块
  - [ ] 新增 `initCodeMirror(container, options)` 函数，配置完整 extension 集合
  - [ ] 新增 `destroyCodeMirror()` 函数
  - [ ] 定义 CM6 自定义 Theme（匹配应用 UI 配色方案）
  - [ ] 全局变量 `let cmEditor = null` 声明

## Phase 2: HTML 结构调整

- [ ] Task 3: 修改 index.html 编辑器 DOM 结构
  - [ ] 将 `<textarea id="editorNoteContent">` 替换为 `<div id="editorNoteContent"></div>`
  - [ ] 删除 `<div id="findOverlay">` 覆盖层
  - [ ] 简化/删除 `#editorFindBar` 内的自定义查找替换 UI（CM6 search panel 自管理）
  - [ ] 保留 `#mdRendered` 预览区不变

## Phase 3: 核心函数改造

- [ ] Task 4: 改造 openEditor() 函数
  - [ ] 加载笔记数据后调用 initCodeMirror() 替代 textarea.value 赋值
  - [ ] 只读模式配置 CM6 editable:false / readOnly
  - [ ] 查看模式纯文本使用 CM6 readOnly 展示
  - [ ] 注意：initCodeMirror() 需在面板动画完成后调用，或初始化后调用 cmEditor.requestMeasure()

- [ ] Task 5: 改造 closeEditor() 函数
  - [ ] 新增 destroyCodeMirror() 调用
  - [ ] 删除 findReplace.reset() 调用

- [ ] Task 6: 改造 switchEditorMode() 函数
  - [ ] edit 模式：显示 .cm-editor 容器，隐藏 #mdRendered
  - [ ] preview 模式：隐藏 .cm-editor 容器，显示 #mdRendered
  - [ ] 切换时关闭 CM6 search panel

- [ ] Task 7: 改造 onEditorInput() 和 updateWordCount()
  - [ ] input 事件监听改为 ViewUpdate.docChanged 监听
  - [ ] 字数统计数据源改为 cmEditor.state.doc.length / .toString()

- [ ] Task 8: 改造 createNote() / updateNote()
  - [ ] 内容获取改为 cmEditor.state.doc.toString()

## Phase 4: 快捷键与事件处理

- [ ] Task 9: 改造 handleKeyboardNavigation()
  - [ ] 删除 Ctrl+Z/Y 手动处理（CM6 historyKeymap 接管）
  - [ ] 删除 [ / ] 匹配导航处理（CM6 search 接管）
  - [ ] 修改 Ctrl+F/H 为调用 CM6 openSearchPanel
  - [ ] 调整 Escape 多级优先级（增加 CM6 search panel 关闭判断）
  - [ ] 确保 Ctrl+L/N 等应用级快捷键不受影响

## Phase 5: 样式适配

- [ ] Task 10: 编写 CM6 主题 CSS
  - [ ] .cm-editor 容器样式（替代原 .editor-textarea，font 继承自 --font-family/--font-size-base）
  - [ ] 更新 CSS 模式切换选择器：`.editor-textarea` → `.cm-editor`（约 4 处）
  - [ ] CM6 Theme 变量定义（背景、文字、光标、选区、激活行、搜索高亮等），包含 --cm-font-family/--cm-font-size 实现字体联动
  - [ ] Markdown 语法高亮样式（标题/加粗/斜体/链接/代码/列表）
  - [ ] 搜索面板样式微调（融入应用风格）

- [ ] Task 11: 清理废弃 CSS
  - [ ] 删除 .find-overlay 相关样式
  - [ ] 删除 .editor-textarea.find-active 样式
  - [ ] 删除 .find-highlight / .find-highlight.active 样式
  - [ ] 删除大量 .editor-find-bar 子元素自定义样式（CM6 自带）

## Phase 6: 代码清理

- [ ] Task 12: 删除 FindReplaceManager 类（~636 行）
  - [ ] 保留预览模式查找高亮的轻量工具函数（从类中剥离为独立函数）
  - [ ] 删除 findReplace 全局单例变量

## Phase 7: 预览模式查找（可选增强）

- [ ] Task 13: 预览区查找高亮保留轻量版
  - [ ] 从 FindReplaceManager 中剥离 _highlightPreview() / _clearPreviewHighlight()
  - [ ] 在 switchEditorMode('preview') 时绑定 Ctrl+F/H 到预览区查找
  - [ ] 或评估是否用临时只读 CM6 实例替代

# Task Dependencies
- [Task 1] 无依赖，最先执行
- [Task 2] 依赖 [Task 1]
- [Task 3] 可与 [Task 2] 并行
- [Task 4] 依赖 [Task 2] + [Task 3]
- [Task 5] 依赖 [Task 4]
- [Task 6] 依赖 [Task 2]
- [Task 7] 依赖 [Task 4]
- [Task 8] 依赖 [Task 4]
- [Task 9] 依赖 [Task 2] + [Task 12]（或并行，删除后重写）
- [Task 10] 依赖 [Task 1]
- [Task 11] 依赖 [Task 10]
- [Task 12] 可在 [Task 9] 之后执行（先改快捷键引用再删原实现）
- [Task 13] 依赖 [Task 6] + [Task 9]，可选
