# 锁屏动画重新设计方案

## 摘要
将锁子 SVG 拆分为锁体（rect）和锁梁（path）两个独立可动画层，实现更有表现力的锁屏动画：入场锁梁滑入装配、待机微呼吸 + 光晕脉动、聚焦亮起觉醒、错误锁梁震颤、解锁锁梁开启。

## 当前不足分析
- **lockHover**：±4px 平移，3s 周期 — 太柔和，容易被忽略
- **entrance**：标准 fade-in + slide-up — 毫无记忆点
- **focus**：scale(1.08) + shadow — 不够醒目
- **error**：整体旋转 ±5° — 锁子应该"受惊"而不是整体摇晃
- **exit**：整体 fade-out — 锁子作为核心元素应该有个"开启"的动作

## 改动方案

### 1. HTML 锁子 SVG 重构

**`index.html`#L1947-L1951** — 将锁体 `<rect>` 和锁梁 `<path>` 分别包裹 `<g>`：

```html
<div class="lock-screen-icon">
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <g class="lock-body-svg">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
        </g>
        <g class="lock-shackle-svg">
            <path d="M7 11V7a5 5 0 0110 0v4"/>
        </g>
    </svg>
</div>
```

同时将 `.lock-screen-icon` 的外围容器增大为 `56px` 以容纳光晕。

### 2. CSS 动画全面重写（`modals.css`）

#### 2.1 待机动画 — "微呼吸 + 光晕脉动"

移除原来的 `lockHover` ±4px 平移。锁体做极慢的缩放脉动（视觉上有"心脏跳动"感），外围有微弱的光晕波纹。

```css
.lock-screen-icon {
  width: 56px;
  height: 56px;
  color: var(--accent);
  margin-bottom: 8px;
  position: relative;
}

/* 光晕脉动环 — 伪元素 */
.lock-screen-icon::before {
  content: '';
  position: absolute;
  inset: -8px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--accent) 15%, transparent) 0%, transparent 70%);
  animation: lockGlowPulse 2.5s ease-in-out infinite;
}

@keyframes lockGlowPulse {
  0%, 100% { transform: scale(1); opacity: 0.6; }
  50% { transform: scale(1.15); opacity: 1; }
}

/* 锁体呼吸 */
.lock-body-svg {
  transform-origin: center;
  animation: lockBodyBreathe 3s ease-in-out infinite;
}

@keyframes lockBodyBreathe {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.04); }
}

/* 锁梁独立微摆 */
.lock-shackle-svg {
  transform-origin: center bottom;
  animation: lockShackleSway 4s ease-in-out infinite;
}

@keyframes lockShackleSway {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(2deg); }
  75% { transform: rotate(-2deg); }
}
```

- 锁体每秒放大 4% → 回正，像心跳
- 锁梁 ±2° 缓慢摆动，像挂锁自然晃动
- 外围光晕环呼吸放大缩小

#### 2.2 入场动画 — "锁梁装配"

```css
/* 锁梁从上方滑入扣合 */
.lock-screen.entering .lock-shackle-svg {
  animation: lockShackleClose 0.5s cubic-bezier(0.34, 1.2, 0.64, 1) forwards;
}

@keyframes lockShackleClose {
  0% { transform: translateY(-20px); opacity: 0; }
  60% { transform: translateY(2px); opacity: 1; }
  100% { transform: translateY(0); opacity: 1; }
}

/* 锁体弹性弹入 */
.lock-screen.entering .lock-body-svg {
  animation: lockBodySpring 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) 0.08s both;
}

@keyframes lockBodySpring {
  0% { transform: scale(0.6); opacity: 0; }
  60% { transform: scale(1.08); }
  100% { transform: scale(1); opacity: 1; }
}

/* 光晕同步出现 */
.lock-screen.entering .lock-screen-icon::before {
  animation: lockGlowAppear 0.4s ease-out 0.1s both;
}

@keyframes lockGlowAppear {
  from { opacity: 0; transform: scale(0.8); }
  to   { opacity: 1; transform: scale(1); }
}
```

移除旧 `lockContentEnter` 对 icon 的引用，保留其他元素的 staggered 入场。

#### 2.3 聚焦动画 — "觉醒亮起"

```css
.lock-screen-icon.focused {
  /* 容器 class 触发整体效果，但不动画子元素 */
}

.lock-screen-icon.focused::before {
  background: radial-gradient(circle, color-mix(in srgb, var(--accent) 30%, transparent) 0%, transparent 70%);
  animation: lockGlowFocus 0.6s ease infinite;
}

@keyframes lockGlowFocus {
  0%, 100% { transform: scale(1); opacity: 0.8; }
  50% { transform: scale(1.25); opacity: 1; }
}

.lock-screen-icon.focused .lock-shackle-svg {
  animation: lockShackleLift 0.4s ease forwards;
}

@keyframes lockShackleLift {
  0% { transform: translateY(0); }
  100% { transform: translateY(-3px); }
}

.lock-screen-icon.focused .lock-body-svg {
  animation: lockBodyGlow 0.4s ease forwards;
}

@keyframes lockBodyGlow {
  0% { filter: brightness(1); }
  100% { filter: brightness(1.2); }
}
```

- 锁梁微微上抬 3px（"打开一条缝"）
- 锁体亮度提升 20%
- 光晕变为 pulsating 更强的呼吸

#### 2.4 错误动画 — "锁梁震颤"

```css
.lock-screen-icon.error-shake .lock-shackle-svg {
  animation: lockRattle 0.4s ease;
}

@keyframes lockRattle {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  10% { transform: translateY(-2px) rotate(-8deg); }
  25% { transform: translateY(0) rotate(6deg); }
  40% { transform: translateY(-2px) rotate(-6deg); }
  55% { transform: translateY(0) rotate(4deg); }
  70% { transform: translateY(-1px) rotate(-3deg); }
  85% { transform: translateY(0) rotate(2deg); }
}

.lock-screen-icon.error-shake .lock-body-svg {
  animation: lockBodyShake 0.4s ease;
}

@keyframes lockBodyShake {
  0%, 100% { transform: translateX(0); }
  15% { transform: translateX(-3px); }
  30% { transform: translateX(3px); }
  45% { transform: translateX(-2px); }
  60% { transform: translateX(2px); }
  75% { transform: translateX(-1px); }
  85% { transform: translateX(1px); }
}

.lock-screen-icon.error-shake::before {
  background: radial-gradient(circle, color-mix(in srgb, var(--error) 20%, transparent) 0%, transparent 70%);
}
```

- 锁梁剧烈震颤（旋转 ±8°）
- 锁体水平摇晃
- 光晕从 accent 变为 error 红色

#### 2.5 解锁动画 — "锁梁开启"

```css
.lock-screen.exit .lock-shackle-svg {
  animation: lockShackleOpen 0.45s cubic-bezier(0.34, 1.2, 0.64, 1) forwards;
}

@keyframes lockShackleOpen {
  0% { transform: translateY(0); opacity: 1; }
  60% { transform: translateY(-18px); opacity: 0.8; }
  100% { transform: translateY(-24px); opacity: 0; }
}

.lock-screen.exit .lock-body-svg {
  animation: lockBodyExit 0.45s ease-out forwards;
}

@keyframes lockBodyExit {
  0% { transform: scale(1); opacity: 1; }
  100% { transform: scale(0.3); opacity: 0; }
}

.lock-screen.exit .lock-screen-icon::before {
  animation: lockGlowExit 0.45s ease-out forwards;
}

@keyframes lockGlowExit {
  0% { transform: scale(1); opacity: 0.6; }
  100% { transform: scale(2); opacity: 0; }
}
```

锁梁向上弹开（类似真实锁打开的动作），锁体缩小消失，光晕向外扩散消散。

### 3. 旧动画清理

移除无用 keyframes：`lockHover`、`lockShake`、旧的 `lockContentEnter` 对 icon 的引用

保留仍在使用的：`lockBackdropEnter`、`lockContentEnter`（其他元素入场仍需）

### 4. 改动总览

| 文件 | 改动 |
|------|------|
| `index.html`#L1947-L1954 | SVG 加 `<g>` 包裹，单独 class 化锁体/锁梁 |
| `modals.css` | 全面重写锁屏图标区域 + 新增 10 个 keyframes + 移除旧 3 个 keyframes + 光晕伪元素 |
| `main.js` | 无改动（class 接口不变：`entering` / `focused` / `error-shake` / `exit`） |

### 5. 验证步骤

1. 启用锁屏 → 重启 → 观察锁梁从上方滑入、锁体弹性弹入、光晕浮现
2. 待机时锁体微呼吸 + 锁梁微摆 + 光晕脉动
3. 点击输入框 → 锁梁上抬、锁体变亮、光晕增强呼吸
4. 输入错误密码 → 锁梁震颤、锁体摇晃、光晕变红
5. 输入正确密码 → 锁梁弹开、锁体缩小消失、光晕扩散消散
6. 切换 12 个主题，检查颜色适配
