# Checkpoints

- [x] 1. 侧边栏在设置页左侧正确显示，宽度 176px，背景色 `var(--bg-secondary)`
- [x] 2. 侧边栏包含 9 个导航项，每个项有 SVG 图标和标签文本
- [x] 3. 导航项悬停时有背景色变化过渡效果
- [x] 4. 激活的导航项显示左侧 3px accent 竖条 + `var(--accent-lighter)` 背景 + 加粗文字
- [x] 5. 首次打开设置页时默认选中"外观"，对应面板正确显示
- [x] 6. 点击不同导航项时，旧面板淡出左移（150ms），新面板从右侧淡入（200ms）
- [x] 7. 动画切换流畅无闪烁，弹性入效果（spring easing）感觉自然
- [x] 8. 系统设置为 `prefers-reduced-motion: reduce` 时跳过动画直接切换
- [x] 9. 所有设置项在面板切换后保持正确的状态（与 `loadSettings()` 同步一致）
- [x] 10. 外观/编辑器卡片之前使用 `.font-setting-row` 的布局已迁移为 `.ai-setting-item` 且显示正常
- [x] 11. 日志卡片之前使用 `.settings-item` 的布局已迁移为 `.ai-setting-item` 且显示正常
- [x] 12. `.font-setting-row` / `.font-settings` / `.settings-item` 相关 CSS 已删除，无残留引用
- [x] 13. 设置页无横向滚动条，侧边栏与内容区比例合理
- [x] 14. 侧边栏导航项可滚动（当设置项较多时内容不溢出）
- [x] 15. `saveSettings()` 在所有面板的设置项修改后正常工作
