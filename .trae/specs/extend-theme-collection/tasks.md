# Tasks

- [ ] Task 1: 在 `variables.css` 中新增 6 套系统主题 `[data-theme]` 定义
  - [ ] `catppuccin-latte` — Catppuccin Latte 暖粉彩浅色（完整覆盖所有 CSS 变量）
  - [ ] `catppuccin-mocha` — Catppuccin Mocha 暖粉彩深色（完整覆盖所有 CSS 变量）
  - [ ] `gruvbox-light` — Gruvbox Light 暖大地浅色（完整覆盖所有 CSS 变量）
  - [ ] `gruvbox-dark` — Gruvbox Dark 暖大地深色（完整覆盖所有 CSS 变量）
  - [ ] `ayu-mirage` — Ayu Mirage 蓝灰金暖科技（完整覆盖所有 CSS 变量）
  - [ ] `dracula` — Dracula 深紫高饱和（完整覆盖所有 CSS 变量）
  - [ ] 确保每个主题块包含：基础色、语义色、分层阴影全部覆盖（参考已有 dark/light 主题）

- [ ] Task 2: 在 `cm6-syntax-highlight.js` 中新增 6 套代码高亮主题
  - [ ] 定义 `catppuccinMochaHighlightStyle` — Catppuccin Mocha 柔粉彩深色
  - [ ] 定义 `gruvboxDarkHighlightStyle` — Gruvbox Dark 复古大地色
  - [ ] 定义 `draculaHighlightStyle` — Dracula 深紫高饱和
  - [ ] 定义 `ayuMirageHighlightStyle` — Ayu Mirage 暖金科技色
  - [ ] 定义 `materialPalenightHighlightStyle` — Material Palenight 紫蓝经典
  - [ ] 定义 `githubLightHighlightStyle` — GitHub Light 纯净浅色
  - [ ] 每个 `HighlightStyle` 覆盖：关键字、类型/类、函数、变量/属性、字面量（数字/字符串/regexp/atom）、注释、运算符、标点/分隔符、HTML/XML 标签、名称空间、预处理、行内代码、删除线

- [ ] Task 3: 将新增代码高亮主题注册到主题系统
  - [ ] 在 `codeHighlightThemes` 对象中添加 6 个新条目
  - [ ] 在 `codeHighlightThemeNames` 数组中添加 6 个新名称
  - [ ] 在 `codeHighlightThemeLabels` 对象中添加 6 个新显示标签

- [ ] Task 4: 更新设置页 UI（`index.html` + `settings-panel.css`）
  - [ ] 系统主题分段控件从 6 按钮改为 12 按钮（两行 × 每行 6 个）
  - [ ] 代码高亮主题下拉菜单新增 6 个选项
  - [ ] 调整 `.theme-segmented` 样式支持两行布局
  - [ ] 调整 `.segmented-indicator` 定位逻辑（JS 侧可能需要适配）

- [ ] Task 5: 更新 `main.js` — 配对标记逻辑（可选）
  - [ ] 定义系统主题 ↔ 代码高亮主题配对映射表
  - [ ] 切换系统主题时更新代码高亮下拉菜单的配对标记
  - [ ] 当下拉菜单打开时高亮推荐配对主题

- [x] Task 6: 验证构建和功能
  - [x] `npx vite build` 通过，零错误（exit code 0）
  - [ ] 12 个系统主题切换后颜色正确、无视觉断裂
  - [ ] 11 个代码高亮主题切换后编辑器高亮即时更新
  - [ ] `.md` 笔记 MD 语法高亮不受代码主题切换影响
  - [ ] 新主题选择跨会话持久化
  - [ ] 分段控件两行布局在所有分辨率下正常

# Task Dependencies

- [Task 1] 和 [Task 2] 无依赖，可并行执行（改不同文件）
- [Task 3] 依赖 [Task 2]（需要先有 HighlightStyle 定义才能注册）
- [Task 4] 和 [Task 1]/[Task 3] 无依赖，可并行执行
- [Task 5] 依赖 [Task 4]（需要先有 UI 元素才能绑定事件）
- [Task 6] 依赖所有前置任务
