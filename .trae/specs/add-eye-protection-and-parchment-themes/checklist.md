# 新增护眼 & 羊皮卷主题 - 验证清单

## `variables.css` 系统主题
- [x] `[data-theme="eye-protection"]` 主题块定义完整（基础色 + 语义色 + 阴影）
- [x] `[data-theme="parchment"]` 主题块定义完整
- [x] 每个主题的 CSS 变量数量与已有浅色主题一致（无遗漏）
- [ ] 护眼主题的语义色与绿色背景协调
- [ ] 羊皮卷主题的语义色与暖黄背景协调

## `main.js` 主题注册
- [x] `themeLabels` 包含 `eye-protection` → 护眼 和 `parchment` → 羊皮卷
- [x] `codeHighlightThemePairing` 为两个新主题配置了推荐代码高亮配对

## `index.html` UI
- [x] `criticalColors` 包含护眼和羊皮卷的防闪色
- [x] 下拉菜单包含"护眼"和"羊皮卷"选项

## 运行时验证
- [x] 编译构建通过（npx vite build）
- [ ] 护眼主题切换后颜色正确、无视觉断裂
- [ ] 羊皮卷主题切换后颜色正确、无视觉断裂
- [ ] 新主题选择跨会话持久化
