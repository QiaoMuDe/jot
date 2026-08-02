# Checklist - 笔记卡片 UI 重构（方案 G）

## 卡片容器样式
- [x] Task 1: `.note-card` 圆角 10px、极浅阴影、精致边框
- [x] Task 1: `.card-title` 字号 1.1rem、字重 700、字距 -0.01em
- [x] Task 1: `.card-content` 字号 0.85rem、行高 1.6
- [x] Task 1: `.card-body` padding 18px 18px 0
- [x] Task 1: `.card-footer` flex space-between 两端对齐
- [x] Task 1: `.card-time` 字号 0.75rem、带 SVG 图标
- [x] Task 1: hover 上移 2px + 阴影加深 + 边框变色
- [x] Task 1: active 缩小到 0.985

## 标签幽灵风格
- [x] Task 2: `.card-tag` 字号 0.75rem、字重 600、padding 4px 12px
- [x] Task 2: 幽灵风格（半透明背景 + 彩色文字 + 细边框）
- [x] Task 2: 内联 `style="background-color"` 通过 CSS 变量 `--tag-color` 转换
- [x] Task 2: hover 标签背景加深、上移 1px

## 置顶标记
- [x] Task 3: `.card-pin-badge` 右上角只读标签（仅置顶时显示）
- [x] Task 3: `.note-card.pinned` 左侧 3px 彩色边框
- [x] Task 3: `main.js` 模板移除 `.card-actions` 和 `pin-btn`
- [x] Task 3: `SVGS.pinFilled`/`SVGS.pinOutline` 引用已处理（保留在 constants.js 中）

## 选中态和批量模式
- [x] Task 4: `.note-card.selected` 样式适配新设计
- [x] Task 4: 批量模式下 `.batch-checkbox` 位置正常
- [x] Task 4: 批量选择、右键菜单交互正常

## 跨主题兼容
- [x] Task 5: 所有 14 个主题下卡片颜色、阴影、边框正常
- [x] Task 5: 暗色主题下幽灵标签可读
- [x] Task 5: `color-mix` 兼容性验证