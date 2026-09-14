# Tasks

- [x] Task 1: Go 侧设置字段与持久化
  - [x] 1.1 `internal/services/types.go`：`SettingsConfig` 新增 `EditorScrollPastEnd bool \`json:"editor_scroll_past_end"\``（置于 `EditorWordWrap` 之后）；`GetAllSettings` 用 `parseBoolSetting(s.Get("editor_scroll_past_end"))`（缺 key → false，符合默认关闭）；`SaveAllSettings` map 新增 `"editor_scroll_past_end": strconv.FormatBool(cfg.EditorScrollPastEnd)`
  - [x] 1.2 `internal/database/db.go`：`InitDefaultSettings` 种子清单补充 `{Key: "editor_scroll_past_end", Value: "false"}`（审查后补录：种子清单为设置键有效性基准，孤儿键清理依赖它）
- [x] Task 2: 设置页 UI 与前端设置链路（可与 Task 1 并行）
  - [x] 2.1 `frontend/index.html`：「编辑器」分组「自动换行」下方新增开关行（id `editorScrollPastEndToggle`，label「滚动超出内容」，desc「启用后可在文档末尾继续向下滚动约一屏（同 VS Code 行为）」），DOM 结构复用 wordWrap 开关模板
  - [x] 2.2 `frontend/src/main.js` 设置链路四件套：`els` 注册 `editorScrollPastEndToggle`；`loadSettings` 回填 `checked = cfg.editor_scroll_past_end`（约 L11629 旁）；`collectSettings` 收集 `editor_scroll_past_end: els.editorScrollPastEndToggle?.checked || false`（约 L11813 旁）；change 监听（约 L6506 旁，先 `await saveSettings()` + 保存成功通知）
- [x] Task 3: 编辑器滚动行为联动（依赖 Task 2）
  - [x] 3.1 `frontend/src/js/cm6-syntax-highlight.js`：jotTheme `.cm-content` 的 `padding: '0 24px 40vh 0'` 改为 `padding: '0 24px var(--editor-tail-space, 2rem) 0'`（注释说明变量语义）
  - [x] 3.2 `frontend/src/main.js` `initCodeMirror`：新增 `cmScrollCompartment = new Compartment()`（每实例 new，同 `cmReadOnlyCompartment` L94-L95 先例）；扩展列表加 `cmScrollCompartment.of(开关开启 ? scrollPastEnd() : [])`，开关值读 `els.editorScrollPastEndToggle?.checked || false`（不动 4 个调用点签名）。补充修复：移除扩展列表中原有无条件 `scrollPastEnd()`（代理遗漏，已由主代理核验时修复）
  - [x] 3.3 `frontend/src/main.js` 即时生效：抽 `applyEditorScrollSetting(enabled)` 辅助——设 `document.documentElement.style.setProperty('--editor-tail-space', enabled ? '0px' : '2rem')` + `cmEditor?.dispatch({ effects: cmScrollCompartment.reconfigure(enabled ? [scrollPastEnd()] : []) })`；change 监听中调用；`loadSettings` 中也调用一次保证启动一致
- [x] Task 4: 构建与回归验证（依赖 Task 1-3）
  - [x] 4.1 Go：`go build ./...` 与 `go vet ./...` 通过
  - [x] 4.2 前端：`cd frontend && npx eslint src/main.js src/js/cm6-syntax-highlight.js` 与 `npm run build`（或 vite build）通过
  - [x] 4.3 按 `checklist.md` 逐项人工/代码走查验证
- [x] Task 5: 尾部留白收紧为 0（方案 A，用户验收后追加）
  - [x] 5.1 `cm6-syntax-highlight.js`：`.cm-content` 底部留白由 `var(--editor-tail-space, 2rem)` 简化为固定 `0`（半行距自带约 8px 视觉缓冲，与顶部 8px 留白对称；两态均为 0 后 CSS 变量机制失去意义，一并移除）
  - [x] 5.2 `main.js`：`applyEditorScrollSetting` 移除 `setProperty` 逻辑，仅保留缓存守卫 + compartment reconfigure
  - [x] 5.3 回归：eslint 0 错误 + vite build 通过

# Task Dependencies
- Task 3 依赖 Task 2（开关元素与设置链路存在）
- Task 1 与 Task 2 可并行
- Task 4 依赖 Task 1、2、3 全部完成
