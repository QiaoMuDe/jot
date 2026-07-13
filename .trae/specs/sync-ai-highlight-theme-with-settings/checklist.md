# Checklist

## hljs-themes.js
- [x] `hljs-themes.js` 和 `hljs-themes-data.js` 已创建，hljs CSS 内容嵌入 JS 数据文件
- [x] CM6 → hljs 主题映射表已定义
- [x] `applyAIHighlightTheme(themeName)` 函数已导出：注入/替换 `<style id="ai-hljs-theme">`
- [x] 未知 themeName 回退到 `github`

## main.js
- [x] `import 'highlight.js/styles/github.css'` 已移除
- [x] `loadSettings()` 中调用 `applyAIHighlightTheme()`
- [x] `applyCodeHighlightTheme()` 中同步调用 `applyAIHighlightTheme()`

## ai-chat.js
- [x] `applyAIHighlightTheme` 已导入
- [x] `renderMarkdown()` 无需额外调用（main.js 的启动/切换同步已保证）

## 验证
- [x] `npx vite build` 通过，零错误
- [x] 启动后 AI 代码块使用 monokai-sublime 主题（对应 CM6 monokai-dimmed）
- [x] 设置页切换代码高亮主题，AI 代码块即时更新
- [x] 新流式输出的 AI 代码块使用当前主题
