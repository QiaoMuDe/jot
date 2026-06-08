# Tasks

- [x] Task 1: 创建查找/替换条 HTML 结构
  - 在 `index.html` 的编辑器面板（`.editor-panel`）中，`.editor-body` 和 `.editor-footer` 之间插入查找条和替换条容器
  - 查找条包含：搜索输入框、匹配计数、上一项/下一项按钮、关闭按钮
  - 替换条包含：替换输入框、替换单个按钮、替换全部按钮、关闭按钮
  - 替换条初始隐藏，由 Ctrl+H 或查找条展开触发

- [x] Task 2: 实现查找/替换条 CSS 样式
  - 在 `app.css` 中新增查找/替换条样式
  - 使用编辑器已有的 CSS 变量（`--card-bg`, `--border`, `--text-primary`, `--accent` 等）
  - 高亮样式：匹配标记使用 `--accent-light` 背景，当前激活匹配使用 `--accent` 背景加白色文字
  - 条与 footer 对齐，宽度与编辑器面板一致

- [x] Task 3: 实现查找功能核心逻辑
  - 在 `main.js` 中实现 `FindReplaceManager` 类或模块
  - 支持对 textarea 内容进行文本匹配搜索（不区分大小写）
  - 实时高亮所有匹配项（使用 overlay 高亮显示）
  - 维护当前激活匹配索引，支持 `[` 和 `]` 键导航
  - 导航时滚动到匹配位置
  - 更新匹配计数显示（"当前/总数"）

- [x] Task 4: 实现替换功能核心逻辑
  - 替换单个：替换当前激活匹配项，更新高亮和计数
  - 替换全部：替换所有匹配项，更新内容后重新渲染
  - Markdown 预览模式（`data-mode="preview"`）下禁用替换按钮并显示提示

- [x] Task 5: 实现快捷键绑定
  - 修改 `handleKeyboardNavigation` 函数
  - Ctrl+F：编辑器打开时触发查找条，编辑器关闭时保持现有行为（聚焦搜索栏）
  - Ctrl+H：编辑器打开时触发查找+替换条
  - `[` 和 `]`：查找条激活时导航匹配项
  - Escape：关闭查找/替换条（若已打开）

- [x] Task 6: 实现焦点管理
  - 打开查找/替换条时自动聚焦搜索输入框
  - 输入时阻止事件冒泡，确保焦点不跳回内容区
  - 关闭时恢复焦点到编辑器内容区（textarea）

- [x] Task 7: 集成与关闭清理
  - 编辑器关闭时自动关闭并重置查找/替换条状态
  - 切换编辑器模式（纯文本↔预览）时重新评估替换功能可用性
  - 全屏模式下查找/替换条正常显示

# Task Dependencies
- [Task 1] 依赖 [Task 0]（无依赖）
- [Task 2] 依赖 [Task 1]
- [Task 3] 依赖 [Task 1, Task 2]
- [Task 4] 依赖 [Task 3]
- [Task 5] 依赖 [Task 3, Task 4]
- [Task 6] 依赖 [Task 3, Task 4]
- [Task 7] 依赖 [Task 3, Task 4, Task 5]
