# Tasks

- [x] Task 1: 重构 `cm6-syntax-highlight.js` 代码高亮主题系统
  - [x] 将 `codeHighlightStyle` 常量改为 `codeHighlightThemes` 对象，key 为主题名称，value 为 `HighlightStyle`
  - [x] 当前 Monokai Dimmed 配色作为 `monokai-dimmed` 主题
  - [x] 导出 `codeHighlightThemeNames` 和 `codeHighlightThemeLabels`（主题名/标签，用于 UI 渲染）
  - [x] 修改 `getHighlightExtension(fileExt, themeName)` 签名，从 `codeHighlightThemes[themeName]` 获取样式
  - [x] 未知 themeName 回退到 `codeHighlightThemes['monokai-dimmed']`
  - [x] 导出的 `codeHighlightStyle` 保持兼容（指向 `codeHighlightThemes['monokai-dimmed']`）

- [x] Task 2: 修改 `main.js` — 加载/保存/应用代码高亮主题设置
  - [x] 新增 `loadCodeHighlightThemeSetting()` 函数，存储键 `code_highlight_theme`，默认 `monokai-dimmed`
  - [x] 在 `init()` 中调用 `loadCodeHighlightThemeSetting()`
  - [x] 修改 `initCodeMirror()` 签名新增 `themeName` 参数（默认 `'monokai-dimmed'`）
  - [x] 传递 themeName 给 `getHighlightExtension(fileExt, themeName)`
  - [x] 在 `openEditor()` 调用 `initCodeMirror()` 处传递 `codeHighlightTheme`
  - [x] 添加 `applyCodeHighlightTheme(themeName)` 函数：更新 `codeHighlightTheme` + 若编辑器打开则销毁重创建实例
  - [x] 添加 `saveCodeHighlightThemeSetting(themeName)` 函数（后端优先 + localStorage fallback）
  - [x] 分段控件点击事件绑定

- [x] Task 3: 更新 `index.html` — 设置页 UI
  - [x] 在「编辑器选项」section 中「启用 CM6 语法高亮」下方新增一行代码高亮主题选择
  - [x] 使用 `.segmented-control` + `.segmented-btn` 结构
  - [x] 当前仅一个选项：「默认 Monokai」

- [x] Task 4: 验证构建和功能
  - [x] `npx vite build` 通过，零错误
  - [x] 启动后设置页显示代码高亮主题选择器，默认选中「默认 Monokai」
  - [x] `.md` 笔记 MD 语法高亮正常（不受影响）
  - [x] `.js`/`.py`/`.go` 等代码高亮正常（仍为 Monokai Dimmed 配色）
  - [x] 切换设置后立即生效（编辑器打开时即时更新）
  - [x] 重启后设置持久化

# Task Dependencies

- [Task 1] 必须先于 [Task 2]（模块导出新函数后 main.js 才能引用）
- [Task 2] 和 [Task 3] 可并行（改不同文件）
- [Task 4] 依赖前面所有任务