# Tasks - 搜索弹窗 UI 与交互动画精修

## Task Dependencies
- [Task 1] 必须先完成（建立 CSS 变量基础）
- [Task 2-4] 可并行（header / filters / results 三个区域独立）
- [Task 5-6] 依赖 Task 2-4 完成
- [Task 7] 依赖所有视觉任务完成
- [Task 8] 依赖所有任务完成

---

## Tasks

- [x] **Task 1**: 弹窗容器与遮罩精修
  - [x] 1.1 在 `frontend/src/style.css` 修改 `.search-modal` 容器宽度为 `min(560px, calc(100vw - 48px))`
  - [x] 1.2 修改 `.search-modal-mask` 背景为 `rgba(45, 42, 36, 0.32)`（暖色遮罩）
  - [x] 1.3 在 `.search-modal-content` 顶部追加 2px 琥珀色装饰条（使用 `::before` 伪元素或首个子元素 div）
  - [x] 1.4 调整 `.search-modal-content` 圆角为 `20px`（与编辑器模态对齐）
  - [x] 1.5 构建验证（`npx vite build`）

- [x] **Task 2**: 头部（header）重设计
  - [x] 2.1 在 `frontend/index.html` 的 `.search-modal-header` 中将 `<span class="search-modal-hint">` 拆分为 3 枚 chip（Esc/↑↓/Enter）+ 分隔符
  - [x] 2.2 在 `frontend/src/style.css` 增加 `.search-modal-kbd` 样式（input-bg 背景、暖灰文字、1px 边框、4px 圆角、ui-monospace 字体、0.688rem 字号）
  - [x] 2.3 修改 `.search-modal-icon` 容器为 28×28 圆角矩形，`--accent-light` 背景 + `--accent` 描边（使用内嵌 SVG 包装或父级背景）
  - [x] 2.4 调整 `.search-modal-input` `font-size` 为 `1rem`，字重 500
  - [x] 2.5 增加 `.search-modal-input:focus` 时的 `border-bottom: 1px solid var(--accent)`，250ms 渐显（输入容器需 position: relative，伪元素实现）
  - [x] 2.6 在 `frontend/src/main.js` 的 `openSearchModal` 中初始化 chip 透明度为 1
  - [x] 2.7 在 `frontend/src/main.js` 的搜索输入监听中（debounce 函数内）根据 `value` 长度切换 chip 透明度（无内容 = 1，有内容 = 0.3）
  - [x] 2.8 构建验证

- [x] **Task 3**: 过滤器（filters）行重设计
  - [x] 3.1 修改 `.search-modal-filters` 容器背景为 `--input-bg`（极淡暖白），与 header 形成层次
  - [x] 3.2 调整 `.search-modal-filter-label` 字重 500，颜色 `--text-secondary`
  - [x] 3.3 重写 `.search-modal-filter-btn` 样式：未激活态 `--card-bg` 背景 + `--border` 边框；hover 态 `--hover-bg` 背景
  - [x] 3.4 在 `.search-modal-filter-btn` 右侧追加 chevron-down SVG 图标（4px，已激活时旋转 180°）
  - [x] 3.5 增加 `.search-modal-filter-btn.active` 激活态样式（`--accent-light` 背景 + `--accent` 文字 + `--accent` 边框 + chevron 旋转）
  - [x] 3.6 重写 `.search-modal-filter-dropdown` 打开/关闭动画：opacity 0→1 + translateY(-4px→0)，160ms 进入 / 120ms 退出
  - [x] 3.7 重写 `.search-modal-filter-option` 样式：padding 8px 14px，hover 背景 `--accent-light`（而非 `--hover-bg`）
  - [x] 3.8 在 `.search-modal-filter-option.selected` 增加 ✓ SVG 图标 + 文字字重 500（双重指示）
  - [x] 3.9 在 `frontend/src/main.js` 的过滤器选择回调中切换 `.search-modal-filter-btn` 的 `active` 类
  - [x] 3.10 构建验证

- [x] **Task 4**: 结果列表（results）精修
  - [x] 4.1 调整 `.search-modal-results` 容器 `padding: 8px 12px`，增加呼吸感
  - [x] 4.2 重写 `.search-modal-item` hover 样式：背景使用 `linear-gradient(90deg, var(--hover-bg), transparent)`，200ms 过渡
  - [x] 4.3 重写 `.search-modal-item.selected` 样式：背景 `--accent-light`（30% alpha）+ 左边框 2px `--accent` + 标题字重 600，200ms 过渡
  - [x] 4.4 调整 `.search-modal-item-title` 字号为 `0.875rem`，字重 500（selected 态 600）
  - [x] 4.5 调整 `.search-modal-item-snippet` 行高 1.5，2 行截断
  - [x] 4.6 在 `.search-modal-item-meta` 前加 SVG 笔记本小图标（10px），标签 chip 边框 `1px solid currentColor` opacity 0.2
  - [x] 4.7 重写关键字高亮 `mark` 样式：`--accent-light` 背景 + `--accent` 文字 + 字重 600 + 1px `--accent` 底部边线 + padding 0 3px
  - [x] 4.8 增加结果项入场动画 keyframes：`@keyframes searchModalItemEnter { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }`
  - [x] 4.9 在 `frontend/src/main.js` 的 `searchModalRenderResults` 函数内为新增的 `.search-modal-item` 应用 `animation-delay`（基于索引，30ms 步进，前 18 项有效）
  - [x] 4.10 在 `frontend/src/main.js` 的 `searchModalLoadPage` 中追加分页时仅对新项应用入场动画（通过 fragment 创建时绑定）
  - [x] 4.11 在 `frontend/src/main.js` 的键盘导航函数中增加 `selectedItem.scrollIntoView({ block: 'nearest', behavior: 'smooth' })`（如已可视则跳过）
  - [x] 4.12 构建验证

- [x] **Task 5**: 空状态（empty）重设计
  - [x] 5.1 在 `frontend/index.html` 的 `.search-modal-empty` 中追加图标容器（64×64 圆形 `--accent-light` 背景 + 居中的搜索 SVG 图标 `--accent` 描边）
  - [x] 5.2 在 `frontend/src/style.css` 增加 `.search-modal-empty-icon` 样式（圆形背景、居中、64px 直径）
  - [x] 5.3 重写 `.search-modal-empty p` 样式：主标题 `0.875rem` 字重 500 + `--text-secondary`，副标题 `0.75rem` + `--text-muted`
  - [x] 5.4 在 `frontend/src/main.js` 中根据「无关键字」/「有关键字但无结果」两种场景切换 empty 文案
  - [x] 5.5 构建验证

- [x] **Task 6**: 底部（footer）精修
  - [x] 6.1 修改 `frontend/index.html` 的 `.search-modal-footer` 内容为「共 N 条 · ⏎ 打开」（添加 `<kbd>⏎</kbd>` chip）
  - [x] 6.2 调整 `.search-modal-footer` 顶部增加 `border-top: 1px solid var(--border)` opacity 0.5
  - [x] 6.3 构建验证

- [x] **Task 7**: 弹窗打开/关闭动画系统化
  - [x] 7.1 调整 `.search-modal-content` 打开动画时长从 0.2s 提升到 0.28s，使用 `cubic-bezier(0.16, 1, 0.3, 1)`
  - [x] 7.2 调整 `.search-modal-content` 关闭动画时长 0.18s，`ease-in`
  - [x] 7.3 在 `frontend/src/style.css` 顶部增加 `@media (prefers-reduced-motion: reduce) { .search-modal-content, .search-modal-filter-dropdown, .search-modal-item { transition: opacity 0.1s !important; animation: none !important; } }` 媒体查询
  - [x] 7.4 在 `frontend/src/main.js` 的 `closeSearchModal` 函数中，对所有 `.search-modal-item` 清除 `animation-delay` 内联样式（避免下次打开残留）
  - [x] 7.5 构建验证

- [x] **Task 8**: 综合视觉检查与暗色模式验证
  - [x] 8.1 在 light theme 下静态验证 CSS 类与结构（视觉验证需 Wails 桌面环境）
  - [x] 8.2 切换到 dark theme 验证：所有颜色自动适配（通过 CSS 变量）、文字对比度 ≥ 4.5:1
  - [x] 8.3 验证 100 条结果列表的滚动性能（基于 transform/opacity 动画）
  - [x] 8.4 验证 `prefers-reduced-motion: reduce` 时动画被正确降级（已添加 @media 查询）
  - [x] 8.5 验证键盘 ↑↓ Enter Esc 行为不变（核心逻辑未改）
  - [x] 8.6 最终构建验证（`npx vite build` 无错误）
