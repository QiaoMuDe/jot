# Tasks

- [x] Task 1: 重构 `app.css` — 统一 CSS 变量体系，新增三套主题定义
  - 在 `:root` 中补充所有新增变量（`--accent-rgb`, `--danger-bg`, `--success-bg` 等）
  - 在 `:root` 下方添加 `[data-theme="light"]` 和 `[data-theme="dark"]` 主题覆盖块
  - 确保 `:root` 中 default 主题值不变，作为 fallback
  - 将 `app.css` 中的滚动条硬编码颜色替换为变量

- [x] Task 2: 重构 `style.css` — 替换所有硬编码颜色为 CSS 变量
  - `.btn-danger` / `.batch-btn.btn-danger` 系列：`#fef2f2` → `var(--danger-bg)`, `#fecaca` → `var(--danger-border)`, `#fee2e2` → `var(--danger-bg)`
  - `.btn-perm-delete` 系列：`#fef2f2` → `var(--danger-bg)`, `#fecaca` → `var(--danger-border)`, `#fee2e2` → `var(--danger-bg)`
  - `.import-result.success`：`#ecfdf5` / `#065f46` / `#a7f3d0` → `var(--success-*)`
  - `.import-result.error`：`#fef2f2` / `#b91c1c` / `#fecaca` → `var(--danger-*)`
  - `.undo-toast`：`#2D2A24` → `var(--toast-bg)`, `#F7F5F0` → `var(--toast-text)`
  - `.editor-overlay` / `.confirm-dialog-overlay`：`rgba(45, 42, 36, 0.4)` → `var(--overlay-bg)`
  - `.note-card.selected` shadow：`rgba(217, 119, 6, 0.15)` → `rgba(var(--accent-rgb), 0.15)`
  - `.loading-spinner`：`#e0e0e0` / `#f97316` → `var(--border)` / `var(--accent)`
  - `.segmented-control` fallback 值移除
  - `.context-menu-item.danger:hover`：`#fef2f2` → `var(--danger-bg)`
  - `.undo-toast-btn:hover`：`rgba(217, 119, 6, 0.15)` → `rgba(var(--accent-rgb), 0.15)`
  - `.tag-delete-btn`：`rgba(0, 0, 0, 0.2)` → `var(--tag-delete-bg)`, `rgba(0, 0, 0, 0.4)` → `var(--tag-delete-hover-bg)`

- [x] Task 3: 设置页新增主题切换下拉菜单
  - 在 `index.html` 设置页「字体设置」section 之后新增「主题设置」section
  - 使用与字体族选择器类似的自定义下拉菜单风格
  - 三个选项：默认 / 浅色 / 深色

- [x] Task 4: 前端主题切换逻辑
  - 实现 `applyTheme(themeName)` 函数：设置 `<html>` 的 `data-theme` 属性
  - 实现 `loadThemeSetting()` 函数：从后端读取已保存的主题
  - 实现 `saveThemeSetting(themeName)` 函数：保存到后端
  - 在 `init()` 启动时加载并应用主题
  - 在 `switchView('settings')` 调用时更新主题选择器 UI

- [x] Task 5: 验证与调试
  - 确认三个主题切换后颜色正确
  - 确认主题偏好跨会话持久化
  - 确认无硬编码颜色遗漏
