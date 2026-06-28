# Tasks

- [x] Task 1: Install new dependencies
  - `npm install @codemirror/lang-javascript @codemirror/lang-css @codemirror/lang-html @codemirror/lang-json @codemirror/lang-python @codemirror/legacy-modes`
- [x] Task 2: Rewrite `frontend/src/js/cm6-syntax-highlight.js`
  - [x] Rename `jotHighlightStyle` → `mdHighlightStyle`，保留现有 16 个 MD 专有 tag 不动
  - [x] 新建 `codeHighlightStyle`，定义编程语言通用 tag（约 25 个），所有颜色引用 CSS 变量
  - [x] 导入 `StreamLanguage` from `@codemirror/language`
  - [x] 导入原生语言包：`javascript`、`css`、`html`、`json`、`python`
  - [x] 通过 `@codemirror/legacy-modes` 导入兜底语言（go、sql、sh、yaml、xml、rust 等约 15 个）
  - [x] 建立 `langMap`：文件扩展名 → 语言解析器工厂函数的映射表（覆盖约 30 种扩展名）
  - [x] 删除静态 `mdHighlight` 导出，替换为 `getHighlightExtension(fileExt)` 工厂函数
- [x] Task 3: Update `frontend/src/main.js`
  - [x] import 从 `{ jotTheme, mdHighlight }` → `{ jotTheme, getHighlightExtension }`
  - [x] `useMdHighlight` → `useSyntaxHighlight`，逻辑从 `.md 始终高亮` 简化为 toggle 全局控制
  - [x] 存储键 `md_highlight_plain` → `cm_syntax_highlight`（3 处）
  - [x] 扩展数组调用：`...(useMdHighlight ? mdHighlight : [])` → `...(useSyntaxHighlight ? getHighlightExtension(fileExt) : [])`
  - [x] `initCodeMirror()` 签名添加 `fileExt` 参数
  - [x] 调用处传递 `els.editorFileExt.textContent`
- [x] Task 4: Update settings page in `frontend/index.html`
  - [x] 标签：「纯文本编辑器启用 Markdown 语法高亮」→「启用 CM6 语法高亮」
  - [x] 提示：「开启后...」→「关闭后所有笔记不再显示代码语法高亮」
- [x] Task 5: Verify build and functionality
  - [x] `npx vite build` 通过，零错误
  - [ ] 启动 app，打开 `.md` 笔记 — MD 语法高亮正常
  - [ ] 打开 `.js` 笔记 — JS 语法高亮正常
  - [ ] 打开 `.py` 笔记 — Python 语法高亮正常
  - [ ] 打开 `.go` 笔记 — Go 语法高亮正常
  - [ ] 打开未知扩展名笔记 — 无语法高亮
  - [ ] Toggle 关闭/开启 功能正常
  - [ ] 切换 6 套主题 → 高亮颜色跟随 CSS 变量变化

# Task Dependencies

- [Task 1]（安装依赖）必须先于 [Task 2]
- [Task 2] 必须先于 [Task 3]
- [Task 3] 和 [Task 4] 可并行（改不同文件）
- [Task 5] 依赖前面所有任务