# Checklist

## cm6-syntax-highlight.js 重构
- [x] `codeHighlightThemes` 对象已定义，`monokai-dimmed` 主题配色与原 `codeHighlightStyle` 完全一致
- [x] `codeHighlightThemeNames` 和 `codeHighlightThemeLabels` 已导出
- [x] `getHighlightExtension(fileExt, themeName)` 签名已修改，默认 themeName 为 `'monokai-dimmed'`
- [x] 未知 themeName 回退到 `codeHighlightThemes['monokai-dimmed']`
- [x] 导出的 `codeHighlightStyle` 保持兼容（指向 `codeHighlightThemes['monokai-dimmed']`）
- [x] `npx vite build` 通过，零错误

## main.js 设置加载/保存
- [x] `loadCodeHighlightThemeSetting()` 函数已实现，存储键 `code_highlight_theme`，默认 `monokai-dimmed`
- [x] `init()` 中已调用 `loadCodeHighlightThemeSetting()`
- [x] `initCodeMirror()` 签名已增加 `themeName` 参数
- [x] `openEditor()` 中传递 `codeHighlightTheme` 给 `initCodeMirror()`
- [x] `applyCodeHighlightTheme(themeName)` 函数已实现：更新 state + 编辑器打开时重创建
- [x] `saveCodeHighlightThemeSetting(themeName)` 函数已实现（后端优先 + localStorage fallback）
- [x] 分段控件点击事件正确绑定

## index.html 设置页 UI
- [x] 「编辑器选项」section 中新增代码高亮主题选择行
- [x] 使用 `.segmented-control` / `.segmented-btn` 结构
- [x] 标签文案为「代码高亮主题」，选项为「默认 Monokai」
- [x] 无新增 CSS，复用已有分段控件样式

## 运行时验证
- [x] 构建通过，零错误，零诊断
- [x] `.md` 笔记不受影响（MD 配色独立于 codeHighlightThemes）
- [x] 代码高亮使用 `codeHighlightThemes['monokai-dimmed']`（与原 `codeHighlightStyle` 相同）
- [x] 切换主题后编辑器重建即时生效
- [x] 设置持久化到后端/ localStorage