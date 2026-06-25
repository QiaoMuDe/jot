# Tasks

- [x] Task 1: 移除 topbar 搜索框 + 降低 topbar/sidebar 高度
  - [x] SubTask 1.1: 从 `index.html` 删除 `<input id="searchInput">` 元素和 `.topbar-search` 容器
  - [x] SubTask 1.2: 调整 `style.css` 4 处数值（`#topbar` 40px / `.sidebar-header` 40px / `.editor-overlay.fullscreening` top 40px / `.editor-panel.fullscreen` calc 中 40px）+ 同步注释
  - [x] SubTask 1.3: 验证无 CSS 残留（grep 48px / 56px 引用）

- [x] Task 2: 新增搜索弹窗 HTML 结构
  - [x] SubTask 2.1: 在 `index.html` 的 `</body>` 前添加 `<div id="searchModal">` 容器
  - [x] SubTask 2.2: 容器包含：背景遮罩 `.search-modal-mask`、弹窗内容 `.search-modal-content`、搜索输入框、过滤器区（笔记本/标签/日期下拉）、结果列表 `.search-modal-results`、底部状态区 `.search-modal-footer`
  - [x] SubTask 2.3: 关键元素 id：输入框 `searchModalInput`、结果列表 `searchModalResults`、空状态 `searchModalEmpty`、结果计数 `searchModalCount`

- [x] Task 3: 搜索弹窗 CSS 样式
  - [x] SubTask 3.1: 在 `style.css` 新增 `.search-modal` 相关样式（遮罩全屏 + 弹窗居中 520×auto,最大高度 70vh）
  - [x] SubTask 3.2: 弹窗使用 `--card-bg` 背景、`--radius-2xl` 圆角、`var(--shadow-xl)` 阴影（与 `.editor-panel` 一致）
  - [x] SubTask 3.3: 遮罩 `background: rgba(0,0,0,0.4)` + `backdrop-filter: blur(4px)`,z-index 高于 topbar
  - [x] SubTask 3.4: 弹窗打开/关闭动画（`scale(0.96) → scale(1)` + `opacity 0 → 1`,0.2s cubic-bezier(0.16, 1, 0.3, 1)）
  - [x] SubTask 3.5: 结果项 hover 背景 `--hover-bg`,选中项额外加左边框 `--accent` 2px,使用与笔记卡片一致的 `--radius-md` 圆角

- [x] Task 4: 搜索弹窗核心逻辑
  - [x] SubTask 4.1: 在 `main.js` 新增 `els` 引用：`searchModal`、`searchModalInput`、`searchModalResults`、`searchModalEmpty`、`searchModalCount`（含 14 个新元素引用）
  - [x] SubTask 4.2: 实现 `openSearchModal()`：添加 `.visible` class、自动 focus 输入框并 select 已有内容、重置过滤器、记录 `state._searchModalPrevFocus = document.activeElement`、body 锁定滚动
  - [x] SubTask 4.3: 实现 `closeSearchModal()`：移除 `.visible` class、清空输入和结果、重置分页状态(`currentPage=1`)、恢复 body 滚动、恢复 `state._searchModalPrevFocus` 焦点
  - [x] SubTask 4.4: 实现 `handleSearchModalInput()`：防抖 200ms 后重置分页(回到第 1 页),调用 `SearchNotes(kw, 1, pageSize, notebookID)`,pageSize 取自全局 `pageSize` 变量(默认 18,用户在设置中可调 9/18/27/36/45/54)
  - [x] SubTask 4.5: 实现 `renderSearchModalResults(results, append)`：渲染结果列表,关键字高亮(新建 `highlightModalMatch`,因 `highlightMatch` 已被 font-search 占用),支持 append 模式(滚动加载更多时追加到列表末尾)
  - [x] SubTask 4.6: 实现 `handleSearchModalKeydown(e)`：↑↓ 导航(循环)、Enter 打开当前选中项(无选中则打开第一项)、Esc 关闭弹窗
  - [x] SubTask 4.7: 实现 `handleSearchModalClick(e)`：点击遮罩区域时关闭（mask 区域和弹窗容器都判）

- [x] Task 5: 滚动加载更多
  - [x] SubTask 5.1: 维护弹窗分页状态:`state.searchModalPage`(当前页,默认 1)、`state.searchModalTotal`(总条数)、`state.searchModalHasMore`(是否还有更多)、`state.searchModalLoading`(是否正在请求)
  - [x] SubTask 5.2: 监听弹窗结果区 `scroll` 事件,当 `scrollTop + clientHeight >= scrollHeight - 200` 时触发下一页加载
  - [x] SubTask 5.3: 加载下一页时:`SearchNotes(kw, ++state.searchModalPage, pageSize, notebookID)`,将结果 append 到列表,根据 `result.Total` 和已加载数判断 `hasMore`
  - [x] SubTask 5.4: `state.searchModalLoading = true` 时禁止重复请求(防止滚动抖动触发多次)
  - [x] SubTask 5.5: 全部加载完毕时,弹窗底部 `.search-modal-footer` 显示「共 X 条结果」,`hasMore = false`,停止触发

- [x] Task 6: 过滤器功能
  - [x] SubTask 6.1: 笔记本下拉：复用现有 `state.notebooks`（实际是 `state.notebooks`,Agent B 已确认）,点击选项时设置 `state.searchModalNotebookId`,重新触发搜索
  - [x] SubTask 6.2: 标签下拉：复用现有 `state.tags`（实际字段名,非 spec 中的 `state.allTags`）,多选支持(点击切换,状态保存在 `state.searchModalTagIds: Set`)
  - [x] SubTask 6.3: 日期范围下拉：预设今天/最近 7 天/最近 30 天/自定义（HTML 中只有 4 个预设,无自定义）,选定后附加到 UI 状态
  - [x] SubTask 6.4: 任一过滤器变化时调用 `handleSearchModalInput()` 重新触发(重置分页到第 1 页)

- [x] Task 7: Ctrl+F 触发逻辑修改
  - [x] SubTask 7.1: 修改 `main.js:4152-4174` 的 Ctrl+F 处理分支
  - [x] SubTask 7.2: 编辑器关闭时（`!els.viewEditor.classList.contains('active')`）：改为调用 `openSearchModal()`,移除原 `els.searchInput.focus()` + `els.searchInput.select()`
  - [x] SubTask 7.3: 编辑器打开时：保持 `openSearchPanel(cmEditor)`(不变),且预览模式自动切到编辑模式 + 选中文本填充到搜索框
  - [x] SubTask 7.4: `e.preventDefault()` 阻止浏览器默认查找行为

- [x] Task 8: 集成验证（静态）
  - [x] SubTask 8.1: Grep 验证 `#topbar { height: 40px; }`、`.sidebar-header { height: 40px; }`、`.editor-overlay.fullscreening { top: 40px; }`、`.editor-panel.fullscreen { height: calc(100vh - 40px); }`
  - [x] SubTask 8.2: Grep 验证 `els.searchInput` 0 处运行时使用（仅留 1 处字段定义为 null + 已废弃注释）
  - [x] SubTask 8.3: Grep 验证 `style.css` 中无 `48px`/`56px` 高度残留、无 `topbar-search` 死代码
  - [x] SubTask 8.4: 验证 `main.js` 中 7 个核心函数 + 2 个下拉渲染函数 + 1 个 keydown 处理函数都已定义
  - [x] SubTask 8.5: 验证 `els` 对象 14 个新元素引用 + `state` 10 个新字段都存在
  - [x] SubTask 8.6: 验证 `index.html` 中 13 个弹窗 id 都在 DOM 中（`#searchModal`/`searchModalInput`/`searchModalResults`/`searchModalEmpty`/`searchModalFooter`/`searchModalCount`/Notebook×3/Tag×3/Date×3）
  - [x] SubTask 8.7: 启动 `wails dev`、目视验证（待用户在本地运行）

# Task Dependencies
- [Task 2] 依赖 [Task 1]（先移除旧搜索框）
- [Task 3] 依赖 [Task 2]（先有 HTML 结构）
- [Task 4] 依赖 [Task 2, Task 3]（DOM 和样式先就位）
- [Task 5] 依赖 [Task 4]（核心搜索逻辑先有,分页才可能实现）
- [Task 6] 依赖 [Task 4]（核心逻辑先实现,过滤器才能重新触发）
- [Task 7] 依赖 [Task 4]（弹窗打开函数先实现）
- [Task 8] 依赖 [Task 1-7]（所有任务完成后验证）
