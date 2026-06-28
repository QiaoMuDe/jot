# Checklist

- [x] `frontend/src/js/cm6-syntax-highlight.js` exists and exports `jotTheme` and `mdHighlight`
- [x] Module correctly imports `EditorView`, `syntaxHighlighting`, `HighlightStyle`, `tags`, `markdown`
- [x] `jotTheme` content matches the original inline definition (references CSS variables like `--card-bg`, `--text-primary`, `--accent`, etc.)
- [x] `mdHighlight` contains `[markdown(), syntaxHighlighting(jotHighlightStyle)]` with all 16 tag mappings
- [x] `main.js` imports `{ jotTheme, mdHighlight }` from `./js/cm6-syntax-highlight.js`
- [x] `main.js` no longer contains inline `jotTheme` or `jotHighlightStyle` definitions
- [x] `initCodeMirror()` extensions array uses `jotTheme` (imported) and `mdHighlight` (imported)
- [x] Vite build succeeds (`npx vite build` in frontend/)
- [ ] App runs without console errors
- [ ] Markdown syntax highlighting in editor looks identical to before (all 16 elements render correctly)
- [ ] Theme switching still updates highlight colors correctly
- [ ] "纯文本编辑器启用 MD 语法高亮" setting still works