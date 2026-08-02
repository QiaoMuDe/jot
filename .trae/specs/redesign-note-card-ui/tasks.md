# Tasks

- [x] Task 1: 重写卡片容器样式（`.note-card`、`.card-body`、`.card-title`、`.card-content`）
  - [x] 1.1 更新 `.note-card` 样式：圆角 10px、极浅阴影、精致边框
  - [x] 1.2 更新 `.card-title`：字号 1.1rem、字重 700、字距收紧
  - [x] 1.3 更新 `.card-content`：字号 0.85rem、行高 1.6
  - [x] 1.4 更新 `.card-body`：padding 18px 18px 0
  - [x] 1.5 更新 `.card-footer`：padding、flex space-between 布局
  - [x] 1.6 更新 `.card-time`：字号 0.75rem、带 SVG 图标
  - [x] 1.7 更新 hover/active 交互状态

- [x] Task 2: 重写标签样式为幽灵风格
  - [x] 2.1 更新 `.card-tag`：字号 0.75rem、字重 600、padding 4px 12px
  - [x] 2.2 幽灵风格：`color-mix` 半透明背景 + 彩色文字 + 细边框
  - [x] 2.3 保留 `style="background-color: ..."` 内联样式，通过 CSS 变量 `--tag-color` 转换
  - [x] 2.4 hover 交互：背景加深、上移 1px

- [x] Task 3: 修改置顶标记方式
  - [x] 3.1 更新 `main.js` 模板：移除 `.card-actions` 和 `pin-btn`，添加 `.card-pin-badge`
  - [x] 3.2 添加 `.card-pin-badge` CSS 样式（右上角绝对定位，仅置顶时显示）
  - [x] 3.3 添加 `.note-card.pinned` 左侧 3px 彩色边框样式
  - [x] 3.4 更新 `SVGS.pinFilled`/`SVGS.pinOutline` 引用处理

- [x] Task 4: 更新选中态和批量模式样式
  - [x] 4.1 更新 `.note-card.selected` 样式适配新卡片设计
  - [x] 4.2 更新批量模式下 `.batch-checkbox` 位置样式
  - [x] 4.3 验证批量选择、右键菜单等交互是否正常

- [x] Task 5: 验证跨主题兼容性
  - [x] 5.1 检查所有主题下卡片颜色、阴影、边框表现
  - [x] 5.2 验证幽灵风格标签在暗色主题下的可读性
  - [x] 5.3 验证 `color-mix` 在各浏览器兼容性

# Task Dependencies
- [Task 3] 依赖 [Task 1] 完成卡片容器样式
- [Task 4] 依赖 [Task 1] 和 [Task 2] 完成
- [Task 5] 依赖 [Task 1]、[Task 2]、[Task 3] 完成