# Tasks

- [ ] Task 1: 后端 `app.go` — 新增 `AIChatFileResult` 结构体和 `SelectAIChatFiles()` 方法
  - 定义 `AIChatFileResult` 结构体（Path, Name, Content, Size int64, Truncated bool, Error string）
  - 实现 `SelectAIChatFiles()`：
    1. 调用 `runtime.OpenFileDialog` 打开多选文件对话框（title: "选择要上传的文本文件"，无文件类型过滤）
    2. 用户取消时返回空数组（nil, nil）
    3. 遍历每个文件路径：
       - `os.Stat()` 获取文件信息
       - 拒绝目录
       - 大小 > 10MB → error
       - `fs.IsBinaryPath()` → error
       - `os.ReadFile()` 读取内容
       - 调用 `GetAIRefMaxChars()`（每次实时查询 DB）获取截断阈值 N
       - 内容长度 > N → 截断并追加截断提示，Truncated = true
       - 收集结果
    4. 返回 `[]AIChatFileResult`

- [ ] Task 2: 前端 `index.html` — 新增上传按钮和文件 chips 栏
  - 在工具栏 `#aiChatRefBtn` 后新增 `<button id="aiChatFileBtn" class="ai-chat-toolbar-btn" title="上传文件">`，内含自定义上传 SVG 图标和 `<span>上传</span>` 文字
  - 技能指示条 `#aiChatSkillBar` 后、输入行前新增 `#aiChatFileBar`（初始 `display:none`），内含 `#aiChatFileChips`

- [ ] Task 3: 前端 CSS `ai-chat.css` — 上传按钮和文件 chips 样式
  - 上传按钮：与现有 `.ai-chat-toolbar-btn` 一致，无需额外样式
  - 文件 chips 栏 `.ai-chat-file-bar`：与 `.ai-chat-ref-bar` 样式一致（flex 布局、padding、gap）
  - 文件 chips `.ai-chat-file-chip`：与 `.ai-chat-ref-chip` 样式一致（背景、边框、圆角、字号、入场动画），但调整 `max-width` 为 260px（文件名可能更长）
  - 文件 chip 图标 `.ai-chat-file-chip-icon`：与 `.ai-chat-ref-chip-icon` 一致
  - 文件 chip 标题 `.ai-chat-file-chip-name`：与 `.ai-chat-ref-chip-title` 一致（省略号截断）
  - 文件 chip 大小 `.ai-chat-file-chip-size`：灰色小字，与 `.ai-chat-ref-chip-trunc` 类似
  - 文件 chip 截断标记：复用 `.ai-chat-ref-chip-trunc` 样式
  - 文件 chip 删除按钮 `.ai-chat-file-chip-remove`：与 `.ai-chat-ref-chip-remove` 一致
  - 批量清除按钮：复用 `.ai-chat-ref-chip-remove-all` 样式，文字改为"清除所有上传文件"

- [ ] Task 4: 前端 JS `ai-chat.js` — 上传交互与消息拼接逻辑
  - 获取 DOM 元素：`#aiChatFileBtn`、`#aiChatFileBar`、`#aiChatFileChips`
  - 新增数据：`uploadedFiles = []`（每项: { name, content, size, truncated }）
  - 上传按钮 click 事件：
    - 按钮置 disabled → 调用 `window.go.main.App.SelectAIChatFiles()` → 恢复按钮
    - 遍历结果：error 的弹通知，成功的追加到 `uploadedFiles`
    - 调用 `renderFileChips()`
  - `renderFileChips()`：
    - 空列表 → 隐藏 `#aiChatFileBar`，return
    - 渲染每个文件 chip（文件图标 + 文件名 + 文件大小 + 截断标记 + × 按钮）
    - ≥ 3 文件 → 追加"清除所有上传文件"按钮（红色虚线边框）
    - 绑定单个删除（按 index 移除）
    - 绑定批量清除（清空数组）
    - 显示 `#aiChatFileBar`
  - `onSend()` 修改：
    - 在收集 `systemContext` 时（笔记引用 + 追问引用之后），遍历 `uploadedFiles` 拼接文件内容
    - 格式：`"用户上传了以下文件内容，请基于这些内容回答用户的提问：\n\n--- 文件: filename (size) ---\n{content}\n---"`
    - truncated 的文件在内容块后追加 `(内容已截断，完整文件共 X 字)`
    - 发送后清空 `uploadedFiles` 并调用 `renderFileChips()`

## Task Dependencies

- None（Task 1-4 可并行执行，前端的 Element ID 和 Go 方法名是约定好的接口）
