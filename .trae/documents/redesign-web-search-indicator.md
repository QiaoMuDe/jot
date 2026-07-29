# 联网搜索动画指示器重新设计方案

## 概要

重新设计 AI 助手联网搜索时的动画指示器，取代当前简陋的纯文字指示器（`createSimpleSearchIndicator`），支持两种场景（有/无精炼关键词）并恢复点击展开关键词下拉菜单功能。设计风格兼顾现代化动画与项目现有主题体系。

## 现状分析

### 当前状态
- 使用 `createSimpleSearchIndicator(text)` 函数生成一个 `<span>`，仅包含旋转地球 SVG + 纯文字
- **无法点击**，无下拉菜单，无关键词展示
- `refinedKeywords` 变量仍被监听存储，但从未用于 UI 展示
- CSS 中残存全套 `.ai-search-indicator`、`.ai-search-dropdown`、`.ai-search-keyword-tag` 样式（孤儿 CSS）

### 事件流
```
后端                   前端
 │                     │
 ├─ ai:search-status("refining") ──→ 显示"正在优化搜索词..."
 ├─ ai:refined-keywords("关键词") ──→ 存储到 refinedKeywords
 ├─ ai:search-status("searching") ──→ 显示"正在联网搜索..."
 ├─ ai:search-source-status(...)   ──→ 记录各源状态（仅记录）
 ├─ (多个 stream chunk)
 └─ ai:search-status("done")       ──→ 替换为打字动画或清空
```

## 设计

### 整体视觉

采用卡片式设计，一个紧凑的搜索状态卡片包含：
1. **左侧**: 旋转的地球 SVG（与现有相同风格，但更精致）
2. **中部**: 状态文字 + 来源数量/关键词数量徽标
3. **右侧**: 下拉箭头（`▾`），有展开/收起旋转动画
4. **下拉菜单**: 展关键词标签（当有关键词时）或搜索源进度条（当无关键词时）

### 颜色体系
- **主题色**: 使用现有 CSS 变量 `var(--accent)` 保持一致
- **背景**: 微透明背景 `color-mix(in srgb, var(--accent) 6%, transparent)` 配合圆角
- **关键词标签**: `var(--accent)` 背景 + 白色文字
- **悬停态**: `var(--hover-bg)` 微浅底色

### 动画设计

#### 1. 地球旋转 SVG
- 使用 `ai-search-spin` 动画（0.8s linear infinite）
- 与现有一致，保持兼容

#### 2. 入场动画（整体出现时）
- `opacity: 0` → `opacity: 1`，`translateY(-6px)` → `translateY(0)`
- duration: 0.25s, easing: `cubic-bezier(0.22, 1, 0.36, 1)`

#### 3. 下拉菜单展开/收起
- **展开**: `opacity: 0; transform: translateY(-8px) scale(0.95)` → `opacity: 1; transform: translateY(0) scale(1)`
- **收起**: `opacity: 0; transform: translateY(-4px) scale(0.97)`
- duration: 展开 0.2s / 收起 0.15s
- easing: `cubic-bezier(0.34, 1.56, 0.64, 1)`（spring 弹跳感）

#### 4. 关键词标签入场
- 交错入场：每 40ms 一个标签依次出现
- 每个标签: `opacity: 0; translateY(-4px)` → `opacity: 1; translateY(0)`
- 类似彩带展开效果

#### 5. 箭头旋转
- `data-open="true"` 时箭头旋转 180°
- `transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1)`（弹簧曲线）

#### 6. 文字微光脉动效果（可选）
- 精炼阶段：文字从右到左的渐变光晕扫描
- 搜索阶段：稳定显示，无多余动画

## 修改计划

### 文件变更清单

#### 1. `frontend/src/js/ai-chat.js`
**变更**: 新增 `createSearchIndicator(status, keywords)` 函数，替换 `createSimpleSearchIndicator` 的调用

**新函数设计**:
```
createSearchIndicator(status, keywords)
  ├─ 创建容器 <div class="ai-search-indicator" data-status={status}>
  ├─ 创建顶栏 <div class="ai-search-bar">
  │   ├─ 地球 SVG（公共）
  │   ├─ 文字区 <span class="ai-search-text">
  │   │   ├─ status=refining: "正在优化搜索词..."
  │   │   └─ status=searching (有keywords): "联网搜索中 (N 个关键词)"
  │   │   └─ status=searching (无keywords): "联网搜索中"
  │   ├─ 来源计数徽标（可选小圆点)
  │   └─ 下拉箭头 SVG（带旋转）
  ├─ 下拉菜单 <div class="ai-search-dropdown">
  │   ├─ keywords 存在:
  │   │   ├─ 标题: "精炼搜索词"
  │   │   └─ <div class="ai-search-keywords">
  │   │       └─ 每个关键词: <span class="ai-search-keyword-tag">{kw}</span>
  │   └─ keywords 不存在:
  │       └─ <div class="ai-search-source-status">
  │           └─ 显示各搜索源状态

交互：
- 点击 bar → 切换 data-open → 显示/隐藏 dropdown
- 点击外部 → 关闭 dropdown
- status=refining 时不可点击（无下拉）
```

**调用点变更**（startStreaming 函数内）:
```
--- 第 2226-2238 行 ---
// 原来
contentDiv.appendChild(createSimpleSearchIndicator('正在优化搜索词...'));
contentDiv.appendChild(createSimpleSearchIndicator('正在联网搜索...'));

// 改为
contentDiv.appendChild(createSearchIndicator('refining'));  // 精炼阶段无关键词
contentDiv.appendChild(createSearchIndicator('searching', refinedKeywords));  // 搜索阶段传入关键词
```

**清理**:
- 保留 `createSimpleSearchIndicator` 不动（以防其他地方使用），或全部替换后删除
- 移除 `refinedKeywords` 存储变量 → 直接通过函数参数传入

#### 2. `frontend/src/css/components/ai-chat.css`
**变更**: 删除旧的孤儿 CSS，替换为全新样式设计

**替换范围**:
```
/* ── 搜索指示器 ── */
覆盖 .ai-search-indicator, .ai-simple-search-indicator 起始到
/* ── 多源搜索指示器 ── */ 之前的所有样式
```

**保留**:
- `@keyframes ai-search-spin`（地球旋转）
- `.ai-simple-search-indicator`（如果还有其他地方使用）
- `.ai-multi-search-indicator` 及后续

**新增样式**:

```css
/* ── 搜索指示器容器 ── */
.ai-search-indicator {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0;
    animation: ai-search-indicator-in 0.25s cubic-bezier(0.22, 1, 0.36, 1);
}

@keyframes ai-search-indicator-in {
    from { opacity: 0; transform: translateY(-6px); }
    to { opacity: 1; transform: translateY(0); }
}

/* ── 搜索栏（点击区域） ── */
.ai-search-bar {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    border-radius: 8px;
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    color: var(--accent);
    font-size: 0.85rem;
    cursor: pointer;
    user-select: none;
    transition: background 0.15s ease;
    position: relative;
}

.ai-search-bar:hover {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
}

/* refining 状态不可点击 */
.ai-search-indicator[data-status="refining"] .ai-search-bar {
    cursor: default;
}
.ai-search-indicator[data-status="refining"] .ai-search-bar:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
}

/* 地球旋转 SVG */
.ai-search-bar > svg:first-child {
    animation: ai-search-spin 0.8s linear infinite;
    flex-shrink: 0;
}

/* 文字 */
.ai-search-text {
    white-space: nowrap;
}

/* 关键词数量徽标 */
.ai-search-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    border-radius: 9px;
    background: var(--accent);
    color: #fff;
    font-size: 0.7rem;
    font-weight: 600;
    line-height: 1;
}

/* 下拉箭头 */
.ai-search-arrow {
    width: 14px;
    height: 14px;
    color: color-mix(in srgb, var(--accent) 50%, transparent);
    transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
    flex-shrink: 0;
}

.ai-search-indicator[data-open="true"] .ai-search-arrow {
    transform: rotate(180deg);
}

/* ── 下拉菜单 ── */
.ai-search-dropdown {
    margin-top: 6px;
    min-width: 200px;
    max-width: 320px;
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.04);
    padding: 10px 14px;
    z-index: 100;
    opacity: 0;
    transform: translateY(-8px) scale(0.95);
    pointer-events: none;
    transition:
        opacity 0.2s ease,
        transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.ai-search-indicator[data-open="true"] .ai-search-dropdown {
    opacity: 1;
    transform: translateY(0) scale(1);
    pointer-events: auto;
}

/* 下拉菜单标题 */
.ai-search-dropdown-label {
    font-size: 0.72rem;
    color: var(--text-muted);
    margin-bottom: 6px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.03em;
}

/* ── 关键词标签容器 ── */
.ai-search-keywords {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
}

/* ── 单个关键词标签 ── */
.ai-search-keyword-tag {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 14px;
    font-size: 0.78rem;
    background: var(--accent);
    color: #fff;
    font-weight: 500;
    line-height: 1.4;
    animation: ai-tag-in 0.25s ease-out backwards;
}

.ai-search-keyword-tag:nth-child(1) { animation-delay: 0ms; }
.ai-search-keyword-tag:nth-child(2) { animation-delay: 40ms; }
.ai-search-keyword-tag:nth-child(3) { animation-delay: 80ms; }
.ai-search-keyword-tag:nth-child(4) { animation-delay: 120ms; }
.ai-search-keyword-tag:nth-child(5) { animation-delay: 160ms; }
.ai-search-keyword-tag:nth-child(6) { animation-delay: 200ms; }
.ai-search-keyword-tag:nth-child(7) { animation-delay: 240ms; }

@keyframes ai-tag-in {
    from { opacity: 0; transform: translateY(-4px) scale(0.9); }
    to { opacity: 1; transform: translateY(0) scale(1); }
}

/* ── 无关键词时的空状态 ── */
.ai-search-empty-keywords {
    font-size: 0.78rem;
    color: var(--text-muted);
    opacity: 0.8;
}
```

#### 3. `frontend/src/css/components/ai-chat.css` — 清理旧样式
**删除**以下旧 CSS 块（它们已被新样式替代或不再使用）:
- `.ai-search-indicator` 原有定义（第 281-289 行）
- `.ai-search-indicator[data-status="searching"]`（第 296-301 行）
- `.ai-search-indicator svg`（第 303-306 行）
- `.ai-search-indicator-bar`（第 308-316 行）
- `.ai-search-indicator-bar:hover`（第 319-321 行）
- `.ai-search-indicator-arrow`（第 323-326 行）
- `.ai-search-indicator[data-open="true"] .ai-search-indicator-arrow`（第 328-330 行）
- `.ai-search-dropdown`（第 338-351 行）
- `.ai-search-dropdown-loading`（第 358-370 行）

**保留**:
- `@keyframes ai-search-spin`（第 332-335 行）
- `.ai-simple-search-indicator`（第 281-289 行中有用到，但需要和 `.ai-search-indicator` 分离）
- `.ai-search-keywords`（移入新设计区块）
- `.ai-search-keyword-tag`（移入新设计区块）
- `.ai-multi-search-indicator` 及后续所有

## 验证步骤

1. 后端正常发射 `ai:search-status("refining")` → 前端显示精炼动画指示器（不可点击，无下拉）
2. 后端发射 `ai:refined-keywords` → 前端存储关键词
3. 后端发射 `ai:search-status("searching")` → 前端显示搜索指示器（可点击，有关键词时显示数量徽标）
4. 点击搜索栏 → 下拉菜单淡入展开，关键词标签交错入场
5. 再次点击 → 下拉菜单收起
6. 点击外部区域 → 下拉菜单收起
7. 有/无关键词两种场景分别测试
8. 无关键词时下拉菜单显示"暂无精炼关键词"空状态

## 假设与决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 关键词标签入场动画 | 交错 + scale 弹入 | 比纯 fade 更有活力，符合"精美"需求 |
| 下拉箭头动画 | 弹簧曲线 rotate(180°) | 与现有项目动画风格一致（`cubic-bezier(0.34, 1.56, 0.64, 1)`） |
| 容器背景 | `color-mix(in srgb, accent 8%, transparent)` | 非纯色，随主题色自适应，比旧纯色方案更精致 |
| refining 阶段 | 无下拉，不可点击 | 精炼阶段关键词尚未完成，交互无意义 |
| 删除旧 CSS | 只删除绝对废弃的孤儿样式 | 保留 `.ai-simple-search-indicator` 以防其他地方引用 |
| `refinedKeywords` 变量传递 | 作为 `createSearchIndicator` 参数传入 | 不再需要全局变量，保持函数纯粹 |
