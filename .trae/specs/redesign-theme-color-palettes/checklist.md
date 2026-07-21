# 主题配色方案重构 — 验证清单

## `variables.css` 配色更新验证
- [x] default 主题所有配色值已按 spec 更新
- [x] light 主题所有配色值已按 spec 更新
- [x] nord 主题所有配色值已按 spec 更新
- [x] gruvbox-light 主题所有配色值已按 spec 更新
- [x] one-dark-pro 主题所有配色值已按 spec 更新
- [x] quiet-light 主题所有配色值已按 spec 更新
- [x] ysgrifennwr 主题所有配色值已按 spec 更新
- [x] tokyo-night 主题所有配色值已按 spec 更新
- [x] eye-protection 主题所有配色值已按 spec 更新（含 accent 和文字颜色彻底重做）
- [x] dark 主题所有配色值已按 spec 更新
- [x] dracula 主题所有配色值已按 spec 更新
- [x] catppuccin-latte 主题所有配色值已按 spec 更新
- [x] alice 主题所有配色值已按 spec 更新
- [x] lightmind 主题所有配色值已按 spec 更新

## 构建验证
- [x] `npx vite build` 通过，零错误（exit code 0）
- [x] 无 CSS 变量定义遗漏导致的视觉断裂

## 视觉验证（手动）
- [ ] 14 套主题切换后页面背景色显示正确
- [ ] 14 套主题切换后卡片背景色与页面背景有清晰层次
- [ ] eye-protection 主题不再是刺目的高饱和绿色
- [ ] dark 主题不再是纯黑背景
- [ ] light 主题有明显的个性特征（不再是纯灰白）
- [ ] 代码块 `--bg-secondary` 在所有主题中均与背景有足够对比
