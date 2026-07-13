# Tasks

- [x] Task 1: 创建 `frontend/src/js/hljs-themes.js` 和 `frontend/src/js/hljs-themes-data.js`
  - hljs-themes-data.js 嵌入 8 个 unique highlight.js 主题 CSS 内容（JSON 字符串）
  - 导出一份 CM6 主题名 → hljs CSS 文件名的映射表
  - 导出 `applyAIHighlightTheme(themeName)` 函数：注入/替换 `<style id="ai-hljs-theme">` 标签
  - 未知 themeName 回退到 `github`（原默认）

- [x] Task 2: 修改 `frontend/src/main.js`
  - 移除第 5 行 `import 'highlight.js/styles/github.css'`
  - 在 `loadSettings()` 中设置 `code_highlight_theme` 后调用 `applyAIHighlightTheme()`
  - 在 `applyCodeHighlightTheme()` 中切换主题时同步调用 `applyAIHighlightTheme()`

- [x] Task 3: 修改 `frontend/src/js/ai-chat.js`
  - 导入 `applyAIHighlightTheme` 函数
  - 无需在 renderMarkdown 中额外调用（main.js 的 loadSettings/applyCodeHighlightTheme 已保证同步）

- [x] Task 4: 验证
  - [x] `npx vite build` 通过，零错误
  - [x] 构建后零诊断错误

# Task Dependencies

- [Task 1] 必须先于 [Task 2] 和 [Task 3]（需要先有 hljs-themes.js 才能引用）
- [Task 2] 和 [Task 3] 可并行（改不同文件）
- [Task 4] 依赖前面所有任务
