# Checklist

- [x] 后端 SettingsConfig 结构体已新增 `EditorWordWrap bool` 字段
- [x] 前端 index.html 编辑器设置卡片已新增自动换行 toggle（id: `editorWordWrapToggle`）
- [x] loadSettings 已读取 `editor_word_wrap` 并更新 toggle 状态
- [x] saveSettings 已收集 `editor_word_wrap` 字段
- [x] initCodeMirror 新增 `enableWordWrap` 参数并条件性注入 `EditorView.lineWrapping()`
- [x] openEditor 调用 initCodeMirror 时已透传自动换行配置
- [x] 设置保存后重启应用，设置持久化（值未丢失）
- [x] 关闭自动换行时，新建/编辑/查看模式下长行均不换行
- [x] 开启自动换行时，新建/编辑/查看模式下长行均自动换行
