# Tasks

- [ ] Task 1: 更新所有主题的 `--scrollbar-thumb` / `--scrollbar-thumb-hover` 变量值
  - `app.css` 中 6 个主题（default/light/dark/nord/monokai-pro/tokyo-night）的变量值按 spec 表格调整
  - SubTask 1.1: 修改 default 主题
  - SubTask 1.2: 修改 light 主题
  - SubTask 1.3: 修改 dark 主题
  - SubTask 1.4: 修改 nord 主题
  - SubTask 1.5: 修改 monokai-pro 主题
  - SubTask 1.6: 修改 tokyo-night 主题
  - SubTask 1.7: 构建验证

- [ ] Task 2: 全局 `::-webkit-scrollbar` 宽度从 8px → 6px
  - `app.css` 中全局 `::-webkit-scrollbar { width/height: 6px }`
  - `app.css` 中 `::-webkit-scrollbar-thumb { border-radius: 3px }`（从 4px）
  - SubTask 2.1: 修改 app.css 全局滚动条规则
  - SubTask 2.2: 构建验证

- [ ] Task 3: 统一 `style.css` 中 CM6 scroller 和 `.md-rendered` 滚动条颜色
  - `.cm-scroller::-webkit-scrollbar-thumb` 从硬编码颜色改为 `var(--scrollbar-thumb)`
  - `.cm-scroller::-webkit-scrollbar-thumb:hover` 改为 `var(--scrollbar-thumb-hover)`
  - 深色主题 `.cm-scroller::-webkit-scrollbar-thumb` 的特殊规则移除（变量已适配）
  - `.md-rendered` 的 scrollbar-color 和 ::-webkit-scrollbar-thumb 改为用变量
  - SubTask 3.1: 修改 style.css 中对应规则
  - SubTask 3.2: 构建验证

- [ ] Task 4: 统一侧栏等区域滚动条样式
  - `.sidebar-notebook-list`：宽度 3px→6px，颜色从 `var(--border)` → `var(--scrollbar-thumb)`
  - `.font-family-options`：颜色从 `var(--divider)` → `var(--scrollbar-thumb)`
  - `.move-notebook-body`：颜色从 `var(--border)` → `var(--scrollbar-thumb)`
  - `.search-modal-filter-dropdown`：颜色从硬编码/var(--border) → `var(--scrollbar-thumb)`
  - SubTask 4.1: 修改 style.css 中对应规则
  - SubTask 4.2: 构建验证

- [ ] Task 5: 为缺少滚动条样式的可滚动区域添加样式
  - `.shortcuts-body`
  - `.search-modal-results`
  - `.batch-tag-body`
  - 使用 `scrollbar-width: thin` + `scrollbar-color` + `::-webkit-scrollbar` 统一规则
  - SubTask 5.1: 在 style.css 中添加各区域滚动条样式
  - SubTask 5.2: 构建验证

- [ ] Task 6: 统一 MD 语法页面面板内滚动条使用变量
  - `.md-ref-source-panel pre` 和 `.md-ref-preview-panel .md-ref-preview` 的滚动条颜色从硬编码改为变量
  - SubTask 6.1: 修改 style.css
  - SubTask 6.2: 构建验证

# Task Dependencies
- Task 1 和 Task 2 无依赖，可并行
- Task 3~6 无依赖，可并行
