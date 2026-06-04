# Tasks

- [ ] 任务 1：安装 NPM 依赖 — 在 `frontend/` 下安装 `marked` 和 `highlight.js`，验证 Vite 能正常打包
- [ ] 任务 2：查看页 Markdown 渲染逻辑 — 在 `openEditor()` 查看分支中，用 rendered 内容替换 readonly textarea，空内容显示占位
- [ ] 任务 3：`.md-rendered` 样式 — 新增样式覆盖标题/列表/代码块/引用/表格/行内代码，适配三个主题（默认/浅色/深色）
- [ ] 任务 4：代码块高亮 — `openEditor()` 中调用 `hljs.highlightAll()`，为代码块自动着色
- [ ] 任务 5：更新记忆（AGENTS.md）

# 任务依赖关系
- [任务 1] 是 [任务 2, 4] 的前置条件
- [任务 2, 3] 无依赖，可并行
- [任务 4] 依赖 [任务 2] 的 DOM 结构
- [任务 5] 最后执行
