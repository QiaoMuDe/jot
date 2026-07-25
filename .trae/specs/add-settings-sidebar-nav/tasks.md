# Tasks

## Task 1: 重构 settings-panel CSS — 侧边栏样式 + 面板切换动画 + 布局体系统一
- [ ] Task 1.1: 将 `.settings-content` 改为 `display: flex` 左右分栏布局（移除 `max-width: 600px; margin: 0 auto`）
- [ ] Task 1.2: 新增 `.settings-nav` 侧边栏容器样式（宽 176px，`bg-secondary` 背景，`border-right` 分隔）
- [ ] Task 1.3: 新增 `.settings-nav-item` 导航项样式（正常/悬停/激活三种状态，左侧 3px accent 竖条激活指示器）
- [ ] Task 1.4: 新增 `.settings-panel` 内容面板容器样式（基础 `display: none`，`.active` 时显示）
- [ ] Task 1.5: 新增面板切换动画 CSS 类（`.panel-exit` / `.panel-enter` / `.panel-active` + `prefers-reduced-motion` 降级）
- [ ] Task 1.6: 删除旧的 `.font-setting-row` / `.font-settings` / `.settings-item` 布局样式
- [ ] Task 1.7: 调整 `.settings-section` 移除卡片背景、圆角、阴影、margin-bottom

## Task 2: 重构设置页 HTML 结构 — 新增侧边栏 DOM + 卡片改为面板容器
- [ ] Task 2.1: 在 `#viewSettings .settings-content` 内部新增 `<nav class="settings-nav">` 侧边栏，包含 9 个 `.settings-nav-item`（图标 + 标签文本）
- [ ] Task 2.2: 将每个 `.settings-section` 包裹在 `.settings-panel` 容器中，添加 `data-panel` 属性标识面板
- [ ] Task 2.3: 在第一个面板添加 `.active` 类（默认显示），其余为 `display: none`
- [ ] Task 2.4: 将外观/编辑器卡片中 `.font-setting-row` 迁移为 `.ai-setting-item` 布局
- [ ] Task 2.5: 将日志卡片中 `.settings-item` 迁移为 `.ai-setting-item` 布局
- [ ] Task 2.6: 从 `ai-group-header` 中提取对应 SVG 图标嵌入侧边栏导航项

## Task 3: 实现 JS 切换逻辑 + 面板动画
- [ ] Task 3.1: 新增 `switchSettingsTab(panelName)` 函数（切换侧边栏激活态 + 面板显隐 + 播放动画）
- [ ] Task 3.2: 为每个 `.settings-nav-item` 绑定 `click` 事件，调用 `switchSettingsTab()`
- [ ] Task 3.3: 实现面板切换动画逻辑：旧面板 `.panel-exit` → 等待 transitionend → 新面板 `.panel-enter` → 移除动画类
- [ ] Task 3.4: 添加 `prefers-reduced-motion` 检测，匹配时跳过动画直接切换
- [ ] Task 3.5: 在 `switchView('settings')` 中默认选中"外观"面板（初始化激活态）
- [ ] Task 3.6: 验证 `loadSettings()` 在切换面板后各设置项 DOM 状态正确

# Task Dependencies
- Task 1 (CSS) 和 Task 2 (HTML) 可并行开发
- Task 3 (JS) 依赖 Task 1 和 Task 2 完成
