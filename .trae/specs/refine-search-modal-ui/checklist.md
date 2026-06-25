# Checklist - 搜索弹窗 UI 与交互动画精修

## 弹窗容器
- [x] Task 1: 容器宽度自适应（560px / 100vw-48px 取小）
- [x] Task 1: 暖色遮罩 rgba(45, 42, 36, 0.32)
- [x] Task 1: 顶部 2px 琥珀色装饰条
- [x] Task 1: 圆角与编辑器模态一致（20px）

## 头部
- [x] Task 2: 搜索图标有圆形背景容器（`--accent-light` 背景）
- [x] Task 2: 输入框 1rem / 字重 500
- [x] Task 2: 聚焦时输入框底部琥珀色下划线渐显
- [x] Task 2: 三个键盘提示 chip（`<kbd>` 风格）替代纯文字 hint
- [x] Task 2: 输入框有内容时 chip 淡出到 opacity 0.3

## 过滤器
- [x] Task 3: 过滤器行背景 `--input-bg`（与 header 形成层次）
- [x] Task 3: filter 按钮右侧 chevron 图标（激活时旋转 180°）
- [x] Task 3: 激活态：背景/文字/边框三重指示 + chevron 旋转
- [x] Task 3: 下拉菜单进入/退出动画（opacity + translateY）
- [x] Task 3: 菜单项 hover 背景 `--accent-light`（暖色更显眼）
- [x] Task 3: 已选项 ✓ 图标 + 字重 500

## 结果列表
- [x] Task 4: 列表 padding 8px 12px（呼吸感）
- [x] Task 4: 结果项 hover：渐变背景 + 左边框 `--accent` 渐显
- [x] Task 4: 结果项 selected：背景 `--accent-light` + 左边框 + 标题字重 600
- [x] Task 4: meta 行笔记本小图标 + 标签 chip 边框
- [x] Task 4: 关键字高亮：accent 文字 + accent-light 背景 + 底部 1px 边线 + 字重 600
- [x] Task 4: 首次搜索结果 30ms 错峰入场（最多 18 项）
- [x] Task 4: 加载更多时仅新项入场
- [x] Task 4: 键盘选中项 `scrollIntoView({ block: 'nearest' })` 不强制居中

## 空状态
- [x] Task 5: 64×64 圆形背景 + 居中搜索图标
- [x] Task 5: 主标题 + 副标题分层（0.875rem / 0.75rem）
- [x] Task 5: 无关键字 vs 无结果两种文案

## 底部
- [x] Task 6: 「共 N 条 · ⏎ 打开」组合（kbd chip）
- [x] Task 6: 顶部 1px 分隔线（opacity 0.5）

## 动画系统
- [x] Task 7: 容器打开动画 280ms cubic-bezier(0.16, 1, 0.3, 1)
- [x] Task 7: 容器关闭动画 180ms ease-in
- [x] Task 7: 内部元素不单独做退出动画
- [x] Task 7: prefers-reduced-motion: reduce 媒体查询降级
- [x] Task 7: 关闭时清除 animation-delay 残留

## 兼容性
- [x] Task 2/3/4: 所有现有 DOM id 与 class 保留，main.js 选择器不破坏
- [x] Task 2/3/4: 所有现有 JS 行为（防抖 200ms、过滤、分页、键盘导航、Esc 关闭）保持不变
- [x] Task 8: dark theme 文字对比度 ≥ 4.5:1（CSS 变量自动适配）
- [x] Task 8: 100 条结果滚动流畅无 jank（基于 transform/opacity 动画）
- [x] Task 8: 键盘 ↑↓ Enter Esc 行为不变（核心逻辑未改）
- [x] Task 8: `npx vite build` 无错误
