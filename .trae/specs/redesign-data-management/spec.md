# Redesign Data Management Layout Spec

## Why

当前数据管理页面布局拥挤：4 张统计卡片在 760px 容器中文本区仅 ~56px，3 个操作按钮横向排列 + 长描述导致视觉过载，整体缺乏视觉层次感。

## What Changes

- 统计卡片去掉图标，仅保留标题 + 大字数值，文本区完全释放，4 列排列宽松
- 数据操作按钮从横向 3 列改为纵向 stack（第二层卡片），左图标 + 右文本布局
- 恢复出厂设置独立为底部危险区域
- 容器宽度 760px（除去图标后卡片内容极简，760px 下 4 列非常宽松）
- 新增「更多功能」预留区带虚线边框占位块
- 优化所有间距、圆角、阴影、视觉层次

## Impact

- Affected code: `frontend/index.html`（viewData 结构）, `frontend/src/style.css`（data-* 全部样式）, `frontend/src/main.js`（loadDataStats 中动画计数逻辑）
- No breaking changes

## ADDED Requirements

### Requirement: 统计卡片去掉图标，极致简约
- 4 张卡片 grid-template-columns: repeat(4, 1fr)
- 卡片去掉 stat-icon，仅 stat-body 居中，内部 stat-value（大字数值）+ stat-label（小字标题）
- 卡片 padding 约 24px 16px，每张卡片足够宽松
- 入场交错动画保留

### Requirement: 数据操作改为纵向 card 列表
- 导出 / 导入 / 恢复出厂设置 三个按钮改为纵向排列
- 左侧图标 + 右侧文字描述的 layout
- hover 效果卡片式微凸起阴影
- 恢复出厂设置独立为一个危险区域，带红色边框 + 警告图标 + 强调文字

### Requirement: 更多功能占位区
- 虚线边框的占位卡片
- 居中的「+」图标 + 提示文字

## MODIFIED Requirements

### Requirement: data-content 容器宽度
修改前: max-width 760px（旧样式）
修改后: max-width 760px（保持，通过缩小卡片内边距/icon 释放空间，无需放宽容器）

### Requirement: Section 标题样式
- 添加左侧 accent 色竖条装饰
- 字体略大 1rem，下方间距 20px

## REMOVED Requirements

无
