# 验收清单

- [x] Task 1: cm6-syntax-highlight.js 中新增 4 个 HighlightStyle
  - [x] tokyoNightHighlightStyle 定义完整（全节点覆盖）
  - [x] nordHighlightStyle 定义完整（全节点覆盖）
  - [x] oneLightHighlightStyle 定义完整（全节点覆盖）
  - [x] catppuccinLatteHighlightStyle 定义完整（全节点覆盖）
  - [x] 注册到 codeHighlightThemes
  - [x] 注册到 codeHighlightThemeNames
  - [x] 注册到 codeHighlightThemeLabels
- [x] Task 2: theme-config.js 配对映射更新
- [x] Task 3: hljs-themes.js hljs 映射更新
- [x] Task 4: 构建验证
  - [x] `npx vite build` 通过
  - [x] 设置页显示 15 个主题选项（动态渲染自 codeHighlightThemeLabels）
  - [x] 4 个新主题切换后代码高亮正常工作（HighlightStyle 编译时注入）
  - [x] AI 聊天代码块高亮同步切换（hljsFileMap + applyAIHighlightTheme）
