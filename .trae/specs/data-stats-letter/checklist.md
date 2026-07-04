# Checklist

## 后端
- [x] DataStats 结构体新增 `TotalTokens`、`AvgResponseTime`、`AvgThinkingTime`、`MaxResponseTime` 四个字段
- [x] AIService 新增 `SumTokens()`、`AvgResponseTime()`、`AvgThinkingTime()`、`MaxResponseTime()` 四个聚合查询方法
- [x] GetDataStats 中调用 4 个新方法并正确赋值
- [x] 空表时返回零值（不报错，不 panic）

## 前端 — HTML 信纸结构
- [x] 移除旧卡片网格（`.data-stats` + 7 张 `.stat-card`）
- [x] 信纸容器包含信头（日期 + 称呼）、笔记段落、AI 段落、分隔线、落款
- [x] 笔记段落含 5 个数据点（笔记、标签、回收站、笔记本、数据库大小）
- [x] AI 段落含 6 个数据点（会话、消息、Token、平均响应、平均思考、最长响应）
- [x] AI 段落末尾有星级评价行
- [x] 数据点使用 `<strong>` + `data-stat-*` 属性标记

## 前端 — CSS 信纸样式
- [x] 信纸容器视觉正确（背景、内边距、圆角、行高、最大宽度）
- [x] 日期右上对齐，称呼字重正确
- [x] 星级 ★/☆ 使用金色渲染
- [x] 落款有手写感风格（italic 或 serif 字体）
- [x] `@keyframes letterReveal` 动画完整定义
- [x] 空数据占位样式正确
- [x] 移除所有旧卡片相关样式（`.stat-card`、`.stat-value`、`.stat-label`、`statCardShake`、`cardEnter`）

## 前端 — JS 逻辑
- [x] `frontend/src/main.js` 中 els 引用适配新结构
- [x] `loadDataStats()` 从 stats 正确读取 11 个字段
- [x] 信纸内容通过 innerHTML 动态拼接（非多个独立 DOM 元素分别填充）
- [x] 星级计算和渲染正确
- [x] 入场动画 `letterReveal` 按规范触发
- [x] 空数据时显示占位提示
- [x] 移除旧逻辑（`animateCountUp` 调用、抖动事件绑定、`_statShakeInited`）
- [x] 编译通过，无 JS 报错
