# 优化卡片入场动画闪烁问题

## 摘要

当前卡片入场动画存在闪烁感，原因在于 `opacity: 1` 在动画 60% 处才出现，而动画总时长仅 250ms，导致卡片在 150ms 内不可见后突然显现。同时弹性回弹集中在最后 50ms，无法被感知。本计划通过调整动画关键帧、增加时长、缩短交错间隔来消除闪烁，让入场更丝滑。

## 当前状态分析

### `cardEnter` 动画 (animations.css:53-59)

```css
@keyframes cardEnter {
    0%   { opacity: 0; transform: translateY(24px) scale(0.96); }
    60%  { opacity: 1; transform: translateY(-2px) scale(1.005); }
    80%  { transform: translateY(1px) scale(0.998); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}
```

- 动画时长: 0.25s (250ms)
- `opacity: 1` 出现在 60% 处（150ms）→ 前 150ms 不可见
- 弹性回弹集中在 60%~100%（100ms 内完成两次变化）
- 初始位移 24px 偏大

### 交错延迟 (main.js:3422, 3441)

```javascript
const delay = Math.min(index * 40, 480);  // 或 (index - prevCount) * 40
```

- 每张卡片间隔 40ms
- 最大延迟 480ms（12 张卡片）
- 延迟期间 `forwards` 应用 0% 关键帧（`opacity: 0`），卡片不可见

### 闪烁根因

1. **主因**: `opacity: 1` 在 150ms 处才出现，从透明到可见的突变就是闪烁感
2. **次因**: 弹性回弹集中在最后 100ms，来不及看清就结束
3. **叠加**: 延迟期（最长 480ms）+ 动画前段（150ms）= 最长 630ms 不可见期

## 变更内容

### 变更 1: 修改 `cardEnter` 关键帧

**文件**: `frontend/src/css/animations.css`

**修改内容**:

```css
@keyframes cardEnter {
    0%   { opacity: 0; transform: translateY(16px) scale(0.97); }
    15%  { opacity: 1; }
    50%  { transform: translateY(-2px) scale(1.005); }
    75%  { transform: translateY(1px) scale(0.998); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}
```

**改动说明**:

| 项目 | 当前值 | 新值 | 原因 |
|------|--------|------|------|
| `opacity: 1` 位置 | 60% (150ms) | **15% (52ms)** | 卡片更快显现，消除闪烁 |
| 动画时长 | 0.25s | **0.35s** | 给弹性回弹留出展示时间 |
| 初始 `translateY` | 24px | **16px** | 减弱入场幅度，更柔和 |
| 初始 `scale` | 0.96 | **0.97** | 配合缩短位移，保持视觉协调 |
| 弹性上冲位置 | 60% → 80% | **50% → 75%** | 弹性过程更舒展 |
| 弹性回弹时间 | 100ms | **175ms** | 可感知的弹性效果 |

### 变更 2: 缩短交错间隔

**文件**: `frontend/src/main.js`

**修改内容**:

- 第 3422 行: `Math.min((index - prevCount) * 40, 480)` → `Math.min((index - prevCount) * 30, 360)`
- 第 3441 行: `Math.min(index * 40, 480)` → `Math.min(index * 30, 360)`

**改动说明**: 交错间隔从 40ms 缩短到 30ms，最大延迟从 480ms 缩短到 360ms。卡片更紧凑地呈现，减少"一波一波"的闪烁感。

### 变更 3: 更新安全兜底超时

**文件**: `frontend/src/main.js`

**修改内容**:

- 第 3433 行: `setTimeout(_cleanCardAnim, 1000)` → `setTimeout(_cleanCardAnim, 1200)`
- 第 3452 行: `setTimeout(_cleanCardAnim, 1000)` → `setTimeout(_cleanCardAnim, 1200)`

**改动说明**: 动画时长从 250ms 增加到 350ms，加上最大延迟 360ms，最晚完成时间约 710ms。兜底超时从 1000ms 调整到 1200ms，留出余量。

## 假设与决策

- `animationend` 清理逻辑不变（`_cleanCardAnim` 守卫 + 兜底 `setTimeout`）
- CSS 中 `transition` 属性不变（仍为 `transform` + `box-shadow` + `border-color`）
- 动画的 `animation-fill-mode: forwards` 不变（延迟期间保持 0% 关键帧）
- 交错间隔缩短到 30ms 后，12 张卡片的总入场时间从 730ms 降到 530ms

## 验证步骤

1. 运行 `npx vite build` 构建前端
2. 重启 Wails 应用
3. 切换笔记本，观察卡片入场动画：
   - 卡片是否在 50ms 内变为可见（不应有闪烁感）
   - 弹性回弹是否可感知（不应有"闪一下"的感觉）
   - 卡片是否紧凑呈现（不应有"一波一波"的延迟感）
4. 测试 hover 卡片，确认浮起 + 缩放 + 阴影效果正常
5. 测试置顶/取消置顶，确认功能正常
6. 测试批量选择，确认选中态 hover 正常