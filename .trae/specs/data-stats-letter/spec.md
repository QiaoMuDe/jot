# 数据统计「信笺」风格重构 Spec

## Why

当前数据管理页面的统计区使用 7 张小卡片平铺网格展示，新增 4 个 AI 性能指标后会达 11 张，碎片化严重且缺乏个性和温度。选择「信笺」风格将统计数据融入自然语言叙述中，使数据面板从冰冷的仪表盘变为一封有温度的信，与笔记应用"记录与反思"的核心调性高度统一。

## 视觉调性

- **温暖书信体**：仿信纸质感，数据嵌入自然语言段落
- **叙述式排版**：数字在句子中用加粗强调，而非表格/网格罗列
- **星级评价**：响应时间用 ⭐ 星级替代秒数，感性表达
- **落款签名**：底部有 "Jot 数据助手" 署名，像一封真正的信
- **展开动画**：信纸从下往上展开（foldUnfold 动画），模拟拆信体验

## What Changes

### 后端 — DataStats 新增 4 个 AI 性能字段

- `internal/services/types.go` 中 `DataStats` 结构体新增：
  - `TotalTokens int64` — Token 消耗总和（JSON: `total_tokens`）
  - `AvgResponseTime float64` — 平均响应总耗时（JSON: `avg_response_time`）
  - `AvgThinkingTime float64` — 平均思考耗时（JSON: `avg_thinking_time`）
  - `MaxResponseTime float64` — 最长响应总耗时（JSON: `max_response_time`）

### 后端 — AIService 新增 4 个聚合查询方法

- `internal/services/ai_service.go` 新增：
  - `SumTokens() (int64, error)` — SELECT COALESCE(SUM(tokens), 0)
  - `AvgResponseTime() (float64, error)` — SELECT COALESCE(AVG(total_elapsed), 0) WHERE total_elapsed > 0
  - `AvgThinkingTime() (float64, error)` — SELECT COALESCE(AVG(thinking_elapsed), 0) WHERE thinking_elapsed > 0
  - `MaxResponseTime() (float64, error)` — SELECT COALESCE(MAX(total_elapsed), 0)

### 后端 — GetDataStats 接入 AI 性能统计

- `app.go` 中 `GetDataStats()` 调用上述 4 个方法并赋值

### 前端 — HTML 结构重构

- 移除全部 7 张 `.stat-card` 和 `.data-stats` 网格容器
- 替换为一个 `.data-letter`（信纸容器），内部包含：
  - `.letter-header`：信头（日期、称呼）
  - `.letter-body-notes`：笔记与存储段落（自然语言嵌入数据）
  - `.letter-body-ai`：AI 统计数据段落（含星级评价）
  - `.letter-footer`：落款签名
- 数据展示使用 `<strong>` 标签包裹数字，而非独立的 `stat-value` 元素
- 新增 4 个内联数据容器用于新字段（使用 `data-stat-*` 属性标记）

### 前端 — CSS 全部重写

- 移除 `.data-stats`、`.stat-card`、`.stat-value`、`.stat-label` 相关样式
- 新增 `.data-letter` 信纸样式体系（见 ADDED Requirements）
- 保留 `.data-content` 和 `.data-content-inner` 的布局结构
- 移除卡片点击抖动事件绑定

### 前端 — JS 逻辑重写

- `frontend/src/js/data-management.js`：
  - `loadDataStats()` 改为填充内联 `<strong>` 元素的 textContent
  - 时间值格式化为 `X.Xs`，Token 使用千分位
  - 星级计算：avg_response_time <= 3s → 5星, <= 6s → 4星, <= 10s → 3星, <= 20s → 2星, > 20s → 1星
  - 入场动画：信纸内容整体淡入 + 轻微上移（transform + opacity），非卡片交错
  - 移除 `animateCountUp` 调用（改为直接填入数值，更接近静态信件的质感）
  - 移除卡片点击抖动逻辑
- `frontend/src/main.js`：
  - 更新 `els` 中的 DOM 引用，移除 `statTotalNotes` 等旧引用，新增 11 个内联元素引用

## Impact

- 修改模型: `internal/services/types.go`
- 修改服务: `internal/services/ai_service.go`
- 修改绑定: `app.go`
- 修改 HTML: `frontend/index.html`（viewData 统计区完全重写）
- 重写 CSS: `frontend/src/css/components/data-view.css`（统计区样式全部替换）
- 重写 JS: `frontend/src/js/data-management.js`（`loadDataStats` 重写）、`frontend/src/main.js`（DOM 引用更新）
- 不涉及 DB schema 变更（字段已存在）

## ADDED Requirements

### Requirement: 信纸容器样式

系统 SHALL 将统计区渲染为一封视觉上的信纸。

- `.data-letter` 使用 `var(--card-bg)` 背景，无边框或极淡 `1px solid var(--border)` 边框
- 内边距：`28px 32px`，模拟信纸留白
- 圆角：`var(--radius-lg)`
- 最大宽度：`580px`，居中显示（比操作列表窄，更像真实的信）
- 行高：`1.8`，模拟书信的宽松行距
- 字体：使用 `var(--font-family)`，但字号略大于常规正文（`0.938rem`）

### Requirement: 信头样式

- 日期行：右上对齐，字号 `0.75rem`，颜色 `var(--text-muted)`，格式为 `YYYY 年 M 月 D 日`
- 称呼行：`font-weight: 600`，`margin-bottom: 16px`，内容为 "亲爱的用户："

### Requirement: 段落样式

- 每个段落顶部有 emoji 图标（`📝` 和 `🤖`）+ 加粗小标题，格式如 `📝 笔记与存储`
- 段落正文：自然语言句子，数字用 `<strong>` 包裹
- 段落间用 `<hr>` 分隔线或 `20px` 间距
- 第二段 AI 部分末尾包含星级评价行：

```
AI 响应速度：
  平均等待 5.7s  ⭐⭐⭐⭐
  思考耗时 2.3s  ⭐⭐⭐
  最长等待 30.9s  ⭐
```

- 星级规则：avg_response_time ≤ 3s → 5星, ≤ 6s → 4星, ≤ 10s → 3星, ≤ 20s → 2星, > 20s → 1星
- 星级使用 `★` 和 `☆` 字符，金色（`#eab308`）渲染

### Requirement: 落款样式

- 距上文 `24px` 间距
- 右对齐（或左对齐缩进），字体风格偏手写感（italic 或使用 `Georgia/serif`）
- 内容：`—— Jot 数据助手`
- 上方可选添加一条短横线 `—`

### Requirement: 入场动画

信纸整体使用一个连贯的展开动画：

- **关键帧 `letterReveal`**：
  - `0%`：`opacity: 0; transform: translateY(24px) scale(0.98);`
  - `60%`：`opacity: 1; transform: translateY(-2px) scale(1.002);`
  - `100%`：`opacity: 1; transform: translateY(0) scale(1);`
- 缓动：`cubic-bezier(0.34, 1.56, 0.64, 1)`
- 时长：`0.5s`
- 不交错（单张信纸整体动画），与旧卡片交错动画不同

### Requirement: 无数据时的占位

- **WHEN** 所有统计数据均为 0（新用户无数据）
- **THEN** 信纸内容显示为："你还没有开始记录呢，快去写第一篇笔记吧！"
- **THEN** 隐藏 AI 统计段落和 stars 评价

## MODIFIED Requirements

### Requirement: loadDataStats 函数

- **修改前**: 获取 7 个 `stat-*` DOM 元素 → 播放卡片交错动画 → count-up 递增
- **修改后**: 获取 11 个内联数据容器 → 拼接段落 HTML → 填充数值 → 播放信纸整体 reveal 动画 → 无 count-up

### Requirement: GetDataStats 接口

- **修改前**: 返回 7 个字段（TotalNotes, TrashedNotes, PinnedNotes, TotalTags, TotalNotebooks, AISessions, AIMessages, DBSize, DBSizeStr）
- **修改后**: 新增 4 个字段（TotalTokens, AvgResponseTime, AvgThinkingTime, MaxResponseTime）

## REMOVED Requirements

### Requirement: 统计卡片网格

**Reason**: 7~11 张小卡片碎片化严重，被信纸风格替代
**Migration**: 全部移除

### Requirement: animateCountUp 数字递增动画

**Reason**: 信纸风格追求静态书信的质感，数字直接填入而非动画递增
**Migration**: 移除 `animateCountUp` 调用

### Requirement: 卡片点击抖动反馈

**Reason**: 信纸不可点击，无交互反馈需求
**Migration**: 移除 `.data-stats` 点击事件监听和 `statCardShake` 关键帧动画

### Requirement: cardEnter 交错入场动画

**Reason**: 信纸使用整体 `letterReveal` 动画替代交错卡片动画
**Migration**: 移除 `cardEnter` 相关逻辑，CSS 中移除 `cardEnter` 关键帧