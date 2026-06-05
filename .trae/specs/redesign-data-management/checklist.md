# Checklist

- [x] `.data-content` max-width 为 760px（保持）
- [x] 统计卡片为 4 列网格布局（grid-template-columns: repeat(4, 1fr)）
- [x] stat-card 无 icon 元素，仅 stat-body 居中
- [x] stat-card padding 约 24px 16px，text-align: center
- [x] stat-value 粗体大字（1.5rem 700w），居中
- [x] stat-label 小字（0.75rem），居中，上边距 6px
- [x] 入场动画（cardEnter + 交错延迟）保留
- [x] section-title 左侧有 accent 色竖条装饰（::before）
- [x] 操作按钮（导出/导入）为纵向列表，左 icon+右文本布局，白色卡片圆角样式
- [x] 恢复出厂设置为独立危险区域，红色边框 + 警告强调
- [x] 更多功能占位区为虚线边框样式
- [x] narrow screen（≤640px）统计卡片 2 列，操作列表 desc 隐藏
- [x] HTML 中已删除 stat-icon 元素
- [x] CSS 中无 stat-icon 残留定义
- [x] `npx vite build` 无错误
