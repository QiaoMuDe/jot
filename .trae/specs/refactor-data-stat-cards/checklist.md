# Checklist

## 后端
- [ ] DataStats 结构体新增 `TotalTokens`、`AvgResponseTime`、`AvgThinkingTime`、`MaxResponseTime` 四个字段
- [ ] AIService 新增 `SumTokens()` 方法，返回所有消息 tokens 总和
- [ ] AIService 新增 `AvgResponseTime()` 方法，返回 total_elapsed 平均值
- [ ] AIService 新增 `AvgThinkingTime()` 方法，返回 thinking_elapsed 平均值
- [ ] AIService 新增 `MaxResponseTime()` 方法，返回 total_elapsed 最大值
- [ ] GetDataStats 中调用上述 4 个新方法并赋值
- [ ] 空表时所有聚合查询返回零值（不报错）

## 前端 — HTML 结构
- [ ] 笔记与存储 Section 包含 5 张卡片（笔记总数、标签总数、回收站、笔记本数、数据库大小）
- [ ] AI 统计数据 Section 包含 6 张卡片（AI 会话、AI 消息、Token 总消耗、平均响应时间、平均思考时间、最长响应时间）
- [ ] 每个 Section 有 `.data-section-title` 标题装饰
- [ ] 新增 4 张卡片含有正确的 ID（`statTotalTokens`、`statAvgResponseTime`、`statAvgThinkingTime`、`statMaxResponseTime`）

## 前端 — JS 逻辑
- [ ] main.js els 新增 4 个 DOM 引用
- [ ] loadDataStats 从 stats 读取 4 个新字段
- [ ] 两组卡片各自独立触发交错 cardEnter 入场动画
- [ ] count-up 数字递增正确（Token 千分位，时间保留 1 位小数 + `s` 后缀）
- [ ] 数据库大小为纯文本显示（不参与 count-up）

## 前端 — CSS 样式
- [ ] 笔记与存储网格为 5 列
- [ ] AI 统计网格为 3 列
- [ ] 窄屏 ≤640px 响应式布局正常（两组均为 2 列）
- [ ] 卡片样式、hover 效果、点击抖动行为保持不变