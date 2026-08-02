# 丝滑卡片入场动画方案

## 摘要

当前卡片入场有两种状态：一是无动画（容器淡入但卡片同时出现），二是之前的弹性弹入（有闪烁）。本次方案使用 **CSS `animation-fill-mode: backwards`** 替代之前的 `forwards`，从根本上解决动画阻塞 `:hover` 的问题，无需任何 `animationend` 清理逻辑，实现丝滑无闪烁的入场效果。

## 核心原理

| 概念 | 之前 (forwards) | 新方案 (backwards) |
|------|:-:|:-:|
| 延迟期间 | 应用 0% 关键帧（opacity: 0） | 应用 0% 关键帧（opacity: 0） |
| 动画结束后 | **保持 100% 关键帧的 transform**，阻塞 `:hover` | **恢复 CSS 默认值**（`transform: none`），`:hover` 正常 |
| 清理逻辑 | 需要 `animationend` + `setTimeout` 兜底 | **不需要清理** |
| 闪烁风险 | 高（清理时序导致） | **无**（无清理逻辑） |

**为什么不会闪烁？**
- 动画结束后，`transform` 恢复为 CSS 默认值 `none`，与 100% 关键帧的 `translateY(0) scale(1)` 视觉等价
- `opacity` 恢复为 CSS 默认值 `1`，与 100% 关键帧一致
- 无需清理内联样式，不存在时序问题

## 改变内容

### 变更 1: 添加 `cardEnter` 动画关键帧（简化版）

**文件**: `frontend/src/css/animations.css`

```css
@keyframes cardEnter {
    0%   { opacity: 0; transform: translateY(12px) scale(0.98); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}
```

**设计说明**:
- 极简两帧动画：`opacity` 从 0→1，`transform` 从 `translateY(12px) scale(0.98)` → `identity`
- 无弹性弹跳（弹性弹跳是之前闪烁的根源之一）
- 位移 12px、缩放 0.98，幅度适中，视觉柔和
- `ease-out` 缓出曲线，自然减速

### 变更 2: 移除容器淡入，恢复卡片自身动画

**文件**: `frontend/src/css/components/main-content.css`

- 移除 `.card-grid` 的 `animation: gridFadeIn 0.15s ease-out`
- 移除 `.note-card` 的 `opacity: 1` 注释（改为 1 保持默认可见）

### 变更 3: 更新 JS 动画逻辑

**文件**: `frontend/src/main.js`

替换 `renderCardGrid` 尾部（当前 3410-3413 行）的动画逻辑：

```javascript
// 卡片入场动画：交错淡入 + 微上移，使用 backwards 避免阻塞 :hover
const cards = els.cardGrid.querySelectorAll('.note-card');
if (animateMode === 'none') {
    // 批量操作，无动画
    cards.forEach(card => { card.style.opacity = '1'; });
} else if (animateMode === 'append' && typeof prevCount === 'number') {
    // 追加模式：已有卡片可见，新卡片带交错动画
    cards.forEach((card, index) => {
        if (index < prevCount) {
            card.style.opacity = '1';
            card.style.animation = 'none';
        } else {
            const delay = Math.min((index - prevCount) * 30, 360);
            card.style.animation = `cardEnter 0.3s ease-out backwards`;
            card.style.animationDelay = `${delay}ms`;
        }
    });
} else {
    // 全量刷新：所有卡片带交错动画（backwards 无需清理）
    cards.forEach((card, index) => {
        const delay = Math.min(index * 30, 360);
        card.style.animation = `cardEnter 0.3s ease-out backwards`;
        card.style.animationDelay = `${delay}ms`;
    });
}
```

**关键点**:
- `backwards` 确保延迟期间应用 0% 关键帧（opacity: 0）
- 动画结束后自动恢复 CSS 默认值，`transform` 不再阻塞 `:hover`
- 不需要 `animationend` 监听器，不需要 `setTimeout` 兜底
- 不需要清理内联样式

### 动画参数表

| 属性 | 值 |
|------|------|
| 动画时长 | 0.3s (300ms) |
| 缓动曲线 | `ease-out` |
| 交错间隔 | 30ms/卡片 |
| 最大延迟 | 360ms (12 张卡片) |
| 总动画时间 | 660ms (12 张卡片) |
| 初始位移 | `translateY(12px)` |
| 初始缩放 | `scale(0.98)` |
| 填充模式 | `backwards` |

## 假设与决策

- `backwards` + 100% 关键帧 `transform: translateY(0) scale(1)` 与 CSS 默认 `transform: none` 视觉等价，无跳变
- `backwards` 在动画结束后不保持任何值，因此 `:hover` 的 `transform` 完全不受影响
- 动画时长 0.3s 平衡了"丝滑感"和"不拖沓"
- 无需处理 `prefers-reduced-motion`（已有全局降级规则）

## 验证步骤

1. 运行 `npx vite build` 构建前端
2. 重启 Wails 应用
3. 切换笔记本，观察卡片入场：
   - 卡片是否从底部轻微上移 + 淡入
   - 是否有"波浪式"的逐张呈现效果
   - 是否有闪烁
4. 动画过程中 hover 卡片：
   - `:hover` 的 `translateY(-2px) scale(1.01)` **是否正常响应**
   - 动画结束后 hover 是否正常
5. 快速切换笔记本（动画未完成时切换）：
   - 是否有残留动画
   - 新笔记是否正确渲染
6. 测试置顶、批量选择、追加加载（滚动）：
   - 功能是否正常
   - 动画是否合理