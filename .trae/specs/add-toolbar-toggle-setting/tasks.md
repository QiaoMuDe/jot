# Tasks

- [x] Task 1: 设置页添加工具栏开关 UI — 在「编辑器选项」区域新增一行 toggle，包含标签、开关、说明文字
  - [x] 在 `index.html` 中 `mdHighlightToggle` 下方插入新的 `.font-setting-row`
  - [x] toggle id 为 `mdToolbarToggle`，存储键 `md_toolbar_enabled`
  - [x] 说明文字「开启后，Markdown 笔记编辑时在顶部显示格式化工具栏（加粗、标题、链接等）」
- [x] Task 2: 工具栏显隐逻辑增加设置判断 — 修改 `populateEditor()` 和类型切换处的 toolbar display 逻辑，读取 `md_toolbar_enabled` 设置
  - [x] 在 `populateEditor()` 中读取设置并应用到工具栏显隐
  - [x] 在类型切换按钮回调中同步读取设置
- [x] Task 3: 设置切换即时生效 — 监听 `mdToolbarToggle` 的 change 事件，编辑器打开时实时更新工具栏显隐
  - [x] `initEventListeners()` 中为 `mdToolbarToggle` 绑定 change 事件
  - [x] change 事件中：存储值 + 如果编辑器打开且为 markdown 编辑模式，即时更新 `els.editorToolbar.style.display`

# Task Dependencies

- 无，三个任务可以并行执行
