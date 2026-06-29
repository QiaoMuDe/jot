# 扩展主题收藏集 - 验证清单

## `variables.css` 系统主题
- [ ] `[data-theme="catppuccin-latte"]` 主题块定义完整（基础色 + 语义色 + 阴影）
- [ ] `[data-theme="catppuccin-mocha"]` 主题块定义完整
- [ ] `[data-theme="gruvbox-light"]` 主题块定义完整
- [ ] `[data-theme="gruvbox-dark"]` 主题块定义完整
- [ ] `[data-theme="ayu-mirage"]` 主题块定义完整
- [ ] `[data-theme="dracula"]` 主题块定义完整
- [ ] 每个主题的 CSS 变量数量与已有 dark/light 主题一致（无遗漏）
- [ ] 新增主题的语义色（success/warning/error/info）使用对应 Official 色板的近似色

## `cm6-syntax-highlight.js` 代码高亮主题
- [ ] `catppuccinMochaHighlightStyle` 定义了所有主要语法节点
- [ ] `gruvboxDarkHighlightStyle` 定义了所有主要语法节点
- [ ] `draculaHighlightStyle` 定义了所有主要语法节点
- [ ] `ayuMirageHighlightStyle` 定义了所有主要语法节点
- [ ] `materialPalenightHighlightStyle` 定义了所有主要语法节点
- [ ] `githubLightHighlightStyle` 定义了所有主要语法节点
- [ ] 6 个新主题已注册到 `codeHighlightThemes`
- [ ] 6 个新名称已添加到 `codeHighlightThemeNames`
- [ ] 6 个新标签已添加到 `codeHighlightThemeLabels`

## 设置页 UI
- [ ] 系统主题分段控件显示 12 个按钮（两行 × 6 个）
- [ ] 代码高亮主题下拉菜单显示 11 个选项
- [ ] 两行布局在所有窗口宽度下无样式断裂
- [ ] 新增主题的 data-theme-value 与 JS 逻辑匹配

## 运行时验证
- [x] `npx vite build` 通过，零错误（exit code 0）
- [ ] 12 个系统主题切换后颜色正确、无视觉断裂
- [ ] 11 个代码高亮主题切换后编辑器即时更新
- [ ] `.md` 笔记不受影响（MD 语法高亮独立于代码主题）
- [ ] 新主题选择跨会话持久化
- [ ] 配对标记（如有实现）显示正确
