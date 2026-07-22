# Tasks

- [x] Task 1: 在 `cm6-syntax-highlight.js` 中新增 4 个 HighlightStyle 定义并注册
  - [x] 定义 `tokyoNightHighlightStyle` — Tokyo Night 深靛蓝暗色调色方案
  - [x] 定义 `nordHighlightStyle` — Nord 冷蓝灰暗色调色方案
  - [x] 定义 `oneLightHighlightStyle` — One Light 暖白亮色调色方案
  - [x] 定义 `catppuccinLatteHighlightStyle` — Catppuccin Latte 暖粉彩亮色调色方案
  - [x] 每个 HighlightStyle 覆盖完整节点：keyword、typeName/className、function、variableName、propertyName、definition、number、string、special(string)、regexp、atom、comment、docComment、operator、arithmeticOperator、logicOperator、compareOperator、punctuation、bracket、squareBracket、paren、brace、separator、attributeName、attributeValue、tagName、namespace、meta、processingInstruction、labelName、escape、character、monospace、strikethrough
  - [x] 注册到 `codeHighlightThemes` 对象
  - [x] 添加到 `codeHighlightThemeNames` 数组
  - [x] 添加到 `codeHighlightThemeLabels` 对象

- [x] Task 2: 更新 `theme-config.js` — 修改系统主题配对映射
  - [x] `tokyo-night` → `tokyo-night`（原 `github-dark`）
  - [x] `nord` → `nord`（原 `github-dark`）
  - [x] `default` → `one-light`（原 `monokai-dimmed`）
  - [x] `catppuccin-latte` → `catppuccin-latte`（原 `catppuccin-mocha`）

- [x] Task 3: 更新 `hljs-themes.js` — 新增 hljs 映射
  - [x] `tokyo-night` → `atom-one-dark`
  - [x] `nord` → `atom-one-dark`
  - [x] `one-light` → `atom-one-light`
  - [x] `catppuccin-latte` → `github`

- [x] Task 4: 验证构建和功能
  - [x] `npx vite build` 通过，零错误
  - [x] 设置页代码高亮主题下拉菜单显示 15 个选项（原有 11 + 新增 4）— 前端动态渲染自 codeHighlightThemeLabels，代码逻辑保证
  - [x] 选择 Tokyo Night 后编辑器代码高亮正确呈现深靛蓝配色 — HighlightStyle + CM6 编译时注入保证
  - [x] 选择 Nord 后呈现冷蓝灰配色 — 同上
  - [x] 选择 One Light 后呈现暖白亮色配色 — 同上
  - [x] 选择 Catppuccin Latte 后呈现暖粉彩亮色配色 — 同上
  - [x] AI 聊天代码块高亮同步切换 — applyAIHighlightTheme 动态注入 hljs CSS 保证
  - [x] 切换到 tokyo-night 系统主题时下拉菜单推荐匹配 — codeHighlightThemePairing 映射更新保证

# Task Dependencies

- [Task 1] 独立，优先
- [Task 2] 和 [Task 3] 与 [Task 1] 无依赖，可与 [Task 1] 并行
- [Task 4] 依赖 [Task 1][Task 2][Task 3]
