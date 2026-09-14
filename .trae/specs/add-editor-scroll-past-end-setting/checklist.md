# Checklist

- [x] 设置页「编辑器」分组出现「滚动超出内容」开关，DOM 结构与 wordWrap 开关一致（id `editorScrollPastEndToggle`），默认未勾选
- [x] Go 侧持久化正确：`SettingsConfig` 含 `EditorScrollPastEnd` 字段；`GetAllSettings` 用 `parseBoolSetting`（缺 key → false）；`SaveAllSettings` map 含 `editor_scroll_past_end` 键
- [x] `InitDefaultSettings` 种子清单含 `editor_scroll_past_end`（默认 "false"）；增量插入仅补缺失键，不覆盖存量用户已保存值
- [x] 开关关闭时：编辑器滚动止于内容末尾，滚动条长度与实际内容一致，无额外尾部留白（不再出现约一屏空白，40vh 不复活）
- [x] 开关打开时：恢复 scrollPastEnd 行为（文末约一屏空白，最后一行可滚至编辑区顶部）
- [x] 即时生效：当前笔记处于编辑模式时切换开关，无需切换笔记/重启，滚动行为立即更新，且弹出保存成功通知
- [x] jotTheme `.cm-content` 底部留白为固定 `0`（滚动严格止于内容末尾，无 CSS 变量依赖）；开启态尾部空间由 scrollPastEnd 内联 padding 接管
- [x] `initCodeMirror` 使用独立 `cmScrollCompartment`（每实例 new），按开关挂载/不挂载 `scrollPastEnd()`；4 个调用点签名未被修改（含补充修复：原扩展列表无条件 `scrollPastEnd()` 已移除，避免恒定生效）
- [x] 切换/新建笔记后（initCodeMirror 重建），新实例滚动行为与开关状态一致
- [x] 重启应用后设置保持；存量升级（KV 无 key）默认为关闭
- [x] 预览面板、AI 消息代码块、MD 语法手册等其他区域滚动行为不受影响（grep 确认 scrollPastEnd 仅存在于笔记编辑器链路）
- [x] `go build ./...` 与 `go vet ./...` 通过
- [x] 前端 eslint 与 build 通过（eslint 0 错误，2 个 warning 为与本次改动无关的存量代码）
