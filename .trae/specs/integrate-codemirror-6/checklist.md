# Checklist: CodeMirror 6 集成

## Phase 1: 基础设施
- [ ] npm 依赖安装完成且无版本冲突
- [ ] initCodeMirror() 函数可正确创建 EditorView 实例
- [ ] destroyCodeMirror() 可正确销毁实例并清理内存
- [ ] CM6 Theme 配色与应用整体风格一致

## Phase 2: HTML 结构
- [ ] textarea 已替换为 div 容器
- [ ] findOverlay 覆盖层已删除
- [ ] 自定义查找/替换条 DOM 已简化或删除

## Phase 3: 核心函数改造
- [ ] openEditor() 在新建模式下正确初始化 CM6 并加载内容
- [ ] openEditor() 在编辑模式下正确加载已有笔记内容到 CM6
- [ ] openEditor() 在只读查看模式下 CM6 为 readOnly 状态
- [ ] closeEditor() 正确销毁 CM6 实例
- [ ] closeEditor() 无内存泄漏（事件监听/定时器全部清理）
- [ ] switchEditorMode('edit') 正确显示 CM6、隐藏预览区
- [ ] switchEditorMode('preview') 正确隐藏 CM6、显示预览区
- [ ] onEditorInput() 通过 ViewUpdate.docChanged 触发自动保存
- [ ] updateWordCount() 从 CM6 state.doc 获取正确的字数统计
- [ ] createNote() 从 CM6 获取内容并成功创建笔记
- [ ] updateNote() 从 CM6 获取内容并成功更新笔记

## Phase 4: 快捷键与事件
- [ ] Ctrl+Z 撤销正常工作（CM6 history）
- [ ] Ctrl+Y 恢复正常工作（CM6 history）
- [ ] Ctrl+F 打开 CM6 查找面板
- [ ] Ctrl+H 打开 CM6 替换面板
- [ ] Ctrl+L 切换纯文本/预览模式不受影响
- [ ] Ctrl+N 新建笔记不受影响（编辑器外生效）
- [ ] Escape 多级优先级正确：search panel 关闭 > 全屏退出 > 编辑器关闭
- [ ] Tab 键在 CM6 中插入缩进而非移走焦点
- [ ] Shift+Tab 减少缩进

## Phase 5: 样式适配
- [ ] CM6 编辑区域视觉风格与原 textarea 一致（字体/大小/行高/圆角）
- [ ] Markdown 语法高亮正确显示（标题/加粗/斜体/链接/代码块/列表）
- [ ] 搜索高亮样式符合应用配色方案
- [ ] 行号显示正常
- [ ] 当前行高亮正常
- [ ] 废弃的查找 overlay / find-active / find-highlight CSS 已清理

## Phase 6: 代码清理
- [ ] FindReplaceManager 类已删除
- [ ] 全局变量 findReplace 已移除
- [ ] 无遗留的废弃代码引用（无 console 报错）

## Phase 7: 边界场景
- [ ] 新建空白笔记 → 输入文字 → Ctrl+Z 撤回 → Ctrl+Y 恢复 → 保存 → 成功
- [ ] 编辑长文档（1000+ 字）→ CM6 性能无明显卡顿
- [ ] 查找功能：输入关键词 → 高亮所有匹配 → 导航前后项 → 替换单个 → 替换全部
- [ ] 预览模式切换：编辑→预览→编辑 往返多次，内容不丢失
- [ ] 只读查看模式：不能修改但能选中和复制
- [ ] IME 中文输入法：输入过程中不触发撤销快照
- [ ] 全屏模式：CM6 在全屏状态下正常工作
- [ ] 编辑器关闭后重新打开另一个笔记：CM6 正确重新初始化
