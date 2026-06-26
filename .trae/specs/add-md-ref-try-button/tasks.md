# Tasks

- [x] Task 1: 在 index.html 的每张 MD 语法卡片底部添加「打开编辑器试试」按钮（10 个 `<button>`）
  - 插入位置：每张卡片的 `</div>` 闭合前（在 `</script>` 之后）
  - 按钮类名：`md-ref-try-btn`
  - 文本：`打开编辑器试试 ›`
  
- [x] Task 2: 在 style.css 中添加按钮样式
  - 按钮在卡片底部居中，与脚注保持间距
  - hover 时强调色变化
  - 设置 `cursor: pointer` 和过渡动画
  
- [x] Task 3: 在 main.js 中添加 `setupMdRefTryButtons()` 和 `openMdRefTryEditor()` 函数
  - 获取每张卡片的 `.md-ref-source` 内容和 `.md-ref-badge` 文字
  - 实现 `openMdRefTryEditor(source, badgeText)` 函数：
    1. 解码 HTML 实体（`&gt;` → `>` 等）
    2. `switchView('grid')`
    3. `openEditor(null)`（新建模式）
    4. 设置标题 `[MD 语法] {badgeText}`
    5. `setEditorContent(decodedSource)`
    6. 设置 `state.noteType = 'markdown'`
    7. 聚焦编辑器
  - 为所有 `.md-ref-try-btn` 绑定 click 事件
  
- [x] Task 4: 在 `renderMdRefCards()` 中调用 `setupMdRefTryButtons()`
  - 确保按钮在卡片渲染后绑定事件
  - 使用 `_mdRefRendered` 防重复绑定
  - 按钮自身用 `_mdRefTryBound` 防重复绑定

# Task Dependencies
- Task 3 depends on Task 1
