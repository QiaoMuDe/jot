# 向量嵌入进度面板优化计划

## Summary

优化数据管理 → 向量嵌入 →「开始嵌入」后的进度视图：

1. **耗时 + 预计剩余时间**：前端估算（A1 方案，零后端改动），融合进主进度条 meta 行右侧的 stage 文案（"正在生成向量 · 已用 00:42 · 预计剩余约 01:30"）。
2. **实时刷新**：1s 定时器，只更新两个时间 `<span>` 的文本，不整行重绘。
3. **进度条动画细化**：现有 fill 已带 `width 0.25s ease` 与斜纹流动动画，仅微调块级进度条过渡时长，避免与整体不一致。
4. **当前标题字幕式横向滚动**：长标题在容器内做 marquee 滚动，短标题不滚动。

纯前端改动，零后端、零 wailsjs 重生成。

## Current State Analysis

### 相关代码

- **HTML 结构**：[index.html L2402-2425](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L2402-L2425)
  - 主进度 meta 行：`#vectorIndexProgressPercent`（左）+ `#vectorIndexProgressStage`（右）
  - 块级进度 meta 行：`#vectorIndexChunkPercent` + `#vectorIndexChunkStage`
  - `#vectorIndexCurrentTitle`（当前标题，单行）
  - `#vectorIndexSummary` / `#vectorIndexError`（收尾区块）

- **JS 更新逻辑**：[data-management.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js)
  - `updateVectorIndexProgress(payload)`（L1290）：stage 用 `textContent` 整体赋值（L1310）
  - `resetVectorIndexProgressUI()`（L1256）：复位全部进度元素
  - `showVectorIndexSummary()`（L1362）/ `showVectorIndexError()`（L1391）：收尾
  - `startVectorIndex()`（L1172）：`setVectorIndexView('progress')` + `resetVectorIndexProgressUI()` 后调后端
  - 已有模块级变量 `vectorIndexChunkResetTimer`（块级清零定时器，L540）
  - `closeVectorIndexModal()`（L705）清理定时器（L714-722）

- **后端可用字段**：[vector_service.go L183-287](file:///d:/峡谷/Dev/本地项目/jot/internal/services/vector_service.go#L183-L287)
  - progressCb 提供 `done / total / title / stage(embedding|done|error) / chunkDone / chunkTotal / errMsg`
  - 无耗时字段 → 前端估算（A1）

- **CSS**：[data-view.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/data-view.css)
  - `.vector-index-progress-fill`（L1110）：`transition: width 0.25s ease` + `vectorIndexStripes` 斜纹动画
  - `.vector-index-progress-stage`（L1150）：`color: var(--text-secondary)` + `tabular-nums`
  - `.vector-index-current`（L1155）：`white-space: nowrap; overflow: hidden; text-overflow: ellipsis` 单行截断，`::before` 内容"当前："

## Proposed Changes

### 1. 时间估算 + 前端融合（核心）

**文件**：[data-management.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js)

- 新增模块级变量（L540 附近）：
  - `vectorIndexStartAt = 0`（任务开始时刻）
  - `vectorIndexElapsedTimer = null`（1s 刷新定时器）
- **估算公式**（A1）：
  - 已用 `elapsed = now - vectorIndexStartAt`
  - 进度 `progress = done/total`（embedding 阶段当前篇按 `done + 0.5` 计，与现有百分比口径一致）
  - 预计剩余 `eta = elapsed / progress * (1 - progress)`，`progress <= 0` 时显示 `--`
  - 格式化 `mm:ss`（超过 1 小时显示 `h:mm:ss`）
- **DOM 结构**：stage 元素内拆 3 个子 `<span>`：
  - 主文案 span（"正在生成向量…" 等，保持现有 `stageMap` 逻辑）
  - 已用 span（`已用 00:42`，`--text-muted` 弱化）
  - 剩余 span（`预计剩余约 01:30`，`--text-muted` 弱化）
  - 定时器只更新两个时间 span 的 `textContent`，主文案不动
- **接入点**：
  - `startVectorIndex()` L1238 后：`vectorIndexStartAt = Date.now()` + 启动定时器
  - `updateVectorIndexProgress()`：当 `stage === 'embedding'` 时拼装时间文案；非 embedding 阶段不显示时间（仅主文案）
  - `showVectorIndexSummary()` / `showVectorIndexError()`：`clearInterval` 定时器
  - `resetVectorIndexProgressUI()`：清定时器 + `vectorIndexStartAt = 0`
  - `closeVectorIndexModal()`：清定时器（沿 L714-722 模式）
- **完成态总耗时**：`showVectorIndexSummary()` 摘要追加 ` · 总耗时 mm:ss`（用 `elapsed` 计算）
- 现有 `stageMap`（L1308）保留，主文案照旧

### 2. 进度条动画微调

**文件**：[data-view.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/data-view.css)

- 主进度条 fill（L1110）：维持现有 `transition: width 0.25s ease`
- 块级进度条 `#vectorIndexChunkFill`：给同一规则加 `transition: width 0.2s ease`（块级更频繁，过渡稍短更跟手）
- 说明：fill 已带斜纹流动动画，无需新增

### 3. 当前标题字幕式横向滚动

**文件**：[data-view.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/data-view.css) + [data-management.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js)

- **CSS**：
  - `.vector-index-current` 改为 marquee 容器：
    - 外层容器保留 `overflow: hidden; white-space: nowrap`
    - 内层滚动条（新增 `.vector-index-current-inner`）：`display: inline-block`，内容超宽时执行 `translateX` 动画
    - 动画：`@keyframes vectorIndexMarquee { from { transform: translateX(0); } to { transform: translateX(-50%); } }`，时长按内容宽度用 CSS 变量 `--marquee-duration` 控制
  - 短标题（不超宽）不加动画类
- **JS**：`updateVectorIndexProgress()` L1312 更新标题时：
  - 用 `scrollWidth > clientWidth` 判断是否超宽
  - 超宽 → 加 `.is-marquee` 类 + 设置 `--marquee-duration`（按 `scrollWidth / 50px/s` 估算）；内容复制两份保证无缝循环
  - 不超宽 → 移除类
- 简化实现（若 marquee 复杂度超标）：标题超宽时仍用 ellipsis 截断 + `title` 属性悬停查看全名（备选）

### 4. 完成态与生命周期收尾

- 完成/错误/关闭时统一 `clearInterval(vectorIndexElapsedTimer)`，避免定时器泄漏（沿用现有 `vectorIndexChunkResetTimer` 模式）

## Assumptions & Decisions

1. **耗时前端估算（A1）**：不新增后端字段，`elapsed` 用事件到达时刻估算。若未来要精确到毫秒级再改后端带 `elapsedSec`。
2. **时间位置**：融合进 stage 文案（"正在生成向量 · 已用 xx · 预计剩余约 xx"），不加新行；时间部分用 `--text-muted` 弱化。
3. **只有 embedding 阶段显示时间**；`准备中 / 完成 / 失败` 阶段仍显示原文案。
4. **完成态总耗时并入摘要**，不重复显示在 stage。
5. **进度条动画**仅微调块级过渡时长，不引入新动画层。
6. **marquee**：超宽才滚动；短标题静态。若实现复杂度高可降级为 ellipsis + title。

## Verification

1. `npm run build` 构建前端成功，无语法错误
2. 手动验证（需 `wails build` 出新二进制或 `wails dev`）：
   - 开始嵌入后：stage 显示"正在生成向量 · 已用 00:0x · 预计剩余约 …"，每秒只跳时间文本、主文案不重绘
   - 完成时：stage 恢复"嵌入完成"，摘要含"总耗时"
   - 失败/取消：定时器停止，无泄漏
   - 关闭弹窗再开：时间归零，无残留
   - 长标题笔记：标题横向滚动；短标题不滚动
   - 块级进度条过渡平滑
