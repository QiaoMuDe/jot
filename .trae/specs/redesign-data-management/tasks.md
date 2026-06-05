# Tasks

- [x] Task 1: 重写数据管理页面 HTML 结构
  - [x] 统计卡片区去掉 stat-icon 元素，仅保留 stat-body（stat-value + stat-label）
  - [x] 操作按钮改为纵向 card 列表（导出/导入）
  - [x] 恢复出厂设置独立为底部危险区域
  - [x] 更多功能占位区带虚线边框
  - [x] 更新 section 标题结构（带左侧竖条装饰）

- [x] Task 2: 重写数据管理全部 CSS 样式
  - [x] 容器宽度保持 760px
  - [x] `.data-section-title` 左侧 accent 竖条 + 新字号间距
  - [x] `.data-stats` 4 列网格，卡片 padding 约 24px 16px
  - [x] `.stat-card` 去掉 flex 布局，改为居中文本（text-align: center）
  - [x] `.stat-icon` 删除所有 stat-icon 相关样式
  - [x] `.stat-value` 粗体大字（1.5rem 700w），居中
  - [x] `.stat-label` 小字（0.75rem），居中，上边距 6px
  - [x] `.data-actions` 改为纵向 flex
  - [x] `.data-action-btn` 改为 row 布局（左 icon + 右文本），白色卡片圆角样式
  - [x] `.data-action-btn-danger` 独立危险样式
  - [x] `.dab-icon`/`.dab-text`/`.dab-desc` 新样式
  - [x] `.data-section-empty` 虚线边框占位块样式
  - [x] 入场动画保留
  - [x] 窄屏响应式适配

- [x] Task 3: 更新 main.js 中 loadDataStats 的计数逻辑
  - [x] 确认 4 张卡片动画交错仍正确工作

- [x] Task 4: 验证构建
  - [x] `npx vite build` 通过
  - [x] 统计卡片 4 列无图标布局正确
  - [x] 操作按钮纵向列表正确
  - [x] 危险按钮样式正确
  - [x] 更多功能占位区显示正确
