# 搜索弹窗开启动画优化 Spec

## Why

当前搜索弹窗的开/关动画存在多处不丝滑的问题：遮罩（mask）无过渡动画直接闪现、背景模糊（backdrop-filter）瞬间切入/切出、遮罩与内容同时动画无层次感、关闭动画时长偏长且缺少退出过渡、结果项只有入场动画没有出场动画、输入框聚焦依赖脆弱的 50ms setTimeout。整体感觉「够用但不精致」，与全站温暖极简的设计语言存在差距。

## Animation 设计方向

遵循 ui-ux-pro-max 动画规范：
- **时长**：微交互 150-300ms，复杂过渡 ≤400ms
- **性能**：仅使用 transform/opacity
- **缓动**：进入 ease-out，退出 ease-in，退出快于进入（~60-70%）
- **错峰**：遮罩 → 内容 → 结果项，30-50ms 间隔序列
- **物理感**：使用 spring 曲线实现轻盈落地感
- **尊重 reduced-motion**：完全降级为无动画

## What Changes

1. **遮罩（mask）增加 opacity + backdrop-filter 过渡**：从 `opacity: 0` + `backdrop-filter: blur(0px)` 过渡到 `opacity: 1` + `backdrop-filter: blur(4px)`
2. **错峰入场**：遮罩先入（0ms）→ 内容延迟 50ms → 结果项延迟 80ms，形成层次感
3. **出场动画优化**：关闭时结果项快速淡出（100ms）→ 内容缩小淡出（150ms）→ 遮罩淡出（200ms），退出总时长控制在 200ms 内
4. **结果项增加出场动画**：关闭时结果项从 opacity:1 + translateY(0) → opacity:0 + translateY(-6px)
5. **输入框聚焦使用 animationend 事件**：替代脆弱的 50ms setTimeout
6. **prefers-reduced-motion 完整支持**：所有过渡时长归零
7. **`closeSearchModal()` 中移除 animation-delay 清理**：改用更可靠的方式确保动画状态重置

## Impact

- Affected specs: refine-search-modal-ui（动画参数变更）、enhance-interaction-animation（动画参数变更）
- Affected code:
  - `frontend/src/style.css` — `.search-modal`/`.search-modal-mask`/`.search-modal-content`/`.search-modal-item` 动画属性修改
  - `frontend/src/main.js` — `openSearchModal()`/`closeSearchModal()` 中聚焦和动画清理逻辑优化
- 无需后端变更，无需 HTML 结构变更

## ADDED Requirements

### Requirement: 遮罩动画

#### Scenario: 遮罩渐入
- **WHEN** search modal opens
- **THEN** mask opacity transitions from 0 to 1 over 200ms ease-out
- **AND** mask backdrop-filter transitions from blur(0px) to blur(4px) over 200ms ease-out

#### Scenario: 遮罩渐出
- **WHEN** search modal closes
- **THEN** mask opacity transitions from 1 to 0 over 150ms ease-in

### Requirement: 错峰入场序列

#### Scenario: 层次化入场
- **WHEN** search modal opens
- **THEN** mask fades in immediately (0ms delay, 200ms duration)
- **AND** content card appears after 50ms delay (scale+translateY spring, 280ms duration)
- **AND** result items stagger with 30ms intervals starting after content is fully visible

### Requirement: 出场动画优化

#### Scenario: 快速优雅退出
- **WHEN** search modal closes
- **THEN** result items fade out first (opacity → 0, translateY → -6px, 100ms ease-in)
- **AND** content card scales down+fades out (scale → 0.96, opacity → 0, 150ms ease-in)
- **AND** mask fades out last (opacity → 0, 150ms ease-in)
- **AND** total close duration ≤ 200ms (所有阶段并发而非串行)

### Requirement: 输入框聚焦优化

#### Scenario: 动画完成后聚焦
- **WHEN** search modal opens and entrance animation completes
- **THEN** search input receives focus
- **AND** focus is triggered by `animationend` or `transitionend` event (or `requestAnimationFrame` fallback)
- **AND** no setTimeout is used

### Requirement: 结果项出场动画

#### Scenario: 结果优雅退出
- **WHEN** search modal closes while results are visible
- **THEN** each result item transitions from opacity:1 + translateY(0) to opacity:0 + translateY(-6px) over 100ms

### Requirement: prefers-reduced-motion

#### Scenario: 无障碍降级
- **WHEN** `prefers-reduced-motion: reduce` is active
- **THEN** all transitions and animations on `.search-modal`, `.search-modal-mask`, `.search-modal-content`, `.search-modal-item` are set to 0ms / none
- **AND** input focus happens immediately without waiting for animation

## MODIFIED Requirements

### Requirement: 弹窗容器动画参数

**Before:**
```css
.search-modal {
    transition: opacity 0.18s ease-in;
}
.search-modal-content {
    transform: scale(0.96) translateY(-8px);
    transition: transform 0.28s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.28s cubic-bezier(0.16, 1, 0.3, 1);
}
```

**After:**
```css
.search-modal {
    transition: none; /* 容器不直接控制透明度，由 mask 和 content 各自控制 */
}
.search-modal-content {
    transform: scale(0.96) translateY(-8px);
    opacity: 0;
    transition: transform 0.28s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.2s ease-out;
    transition-delay: 0.05s; /* 错峰 50ms */
}
.search-modal.visible .search-modal-content {
    transform: scale(1) translateY(0);
    opacity: 1;
    transition-delay: 0.05s;
}
```

### Requirement: 遮罩动画参数

**Before:**
```css
.search-modal-mask {
    /* 无 transition */
}
```

**After:**
```css
.search-modal-mask {
    opacity: 0;
    transition: opacity 0.2s ease-out, backdrop-filter 0.2s ease-out;
    backdrop-filter: blur(0px);
    -webkit-backdrop-filter: blur(0px);
}
.search-modal.visible .search-modal-mask {
    opacity: 1;
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
}
```

### Requirement: 结果项出场动画

**Before:**
```css
.search-modal-item {
    animation: searchModalItemEnter 0.2s cubic-bezier(0.16, 1, 0.3, 1) backwards;
}
/* 无退出动画 */
```

**After:**
```css
.search-modal-item {
    animation: searchModalItemEnter 0.2s cubic-bezier(0.16, 1, 0.3, 1) backwards;
    transition: opacity 0.1s ease-in, transform 0.1s ease-in;
}
.search-modal.closing .search-modal-item {
    opacity: 0;
    transform: translateY(-6px);
}
```

## REMOVED Requirements

无

## 交互时序图

### 打开（总时长 ~330ms）
```
0ms      ── 遮罩淡入 (opacity 0→1, 200ms)
50ms     ── 内容卡片 (scale+translateY, 280ms)
80ms     ── 结果项逐条错峰 (30ms 间隔)
~330ms   ── 全部完成 → 聚焦输入框
```

### 关闭（总时长 ~200ms 内并发完成）
```
0ms      ── 结果项淡出 (100ms)
          ── 内容卡片缩小淡出 (150ms)
          ── 遮罩淡出 (150ms)
~150ms   ── 全部完成
```
