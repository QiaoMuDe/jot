# 锁屏动画方案 — 实施计划

## 摘要
为锁屏页面添加四档动画交互：待机呼吸悬浮、输入聚焦注视、解锁门扉消融、入场雾气凝聚。纯 CSS keyframes + 少量 JS class 切换，不修改 HTML 结构。

## 当前状态
- **锁子图标**: 静态 SVG（`<rect>` 锁体 + `<path>` 锁梁），`lock-screen-icon` 容器无动画
- **解锁过渡**: 仅有 backdrop blur 渐隐 0.4s → 450ms 后 `display:none`，内容区无动画
- **入场**: `display:none` → `display:flex` 瞬间弹出
- **密码错误**: 仅输入框 `shake` class，锁子无反馈

## 改动内容

### 文件 1: `modals.css` — 新增动画（~80 行）

#### 1.1 待机呼吸动画（锁子图标）

**`modals.css`** — 在 `.lock-screen-icon` 添加：
```css
.lock-screen-icon {
  animation: lockHover 3s ease-in-out infinite;
}

@keyframes lockHover {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}
```

周期 3s，幅度 ±4px，`translateY` GPU 合成。

#### 1.2 输入聚焦注视效果

当密码输入框获得焦点时，锁子图标微放大 + 投影增强。由于 `.lock-screen-icon` 在 input 之前（CSS 无法前向兄弟选择），使用 JS toggle class `.focused` 实现。

**`modals.css`** — 新增：
```css
.lock-screen-icon.focused {
  transform: scale(1.08);
  filter: drop-shadow(0 0 12px color-mix(in srgb, var(--accent) 50%, transparent));
  transition: transform 0.3s ease, filter 0.3s ease;
}
/* 覆盖呼吸动画 — 聚焦时暂停呼吸，固定到放大状态 */
.lock-screen-icon.focused {
  animation: none;
}
```

#### 1.3 解锁门扉消融（内容区向上飘散）

**`modals.css`** — 修改 `.lock-screen.exit` 相关规则：

```css
.lock-screen.exit .lock-screen-content {
  opacity: 0;
  transform: translateY(-24px) scale(0.95);
  transition: all 0.45s cubic-bezier(0.34, 1.2, 0.64, 1);
}

.lock-screen.exit .lock-screen-backdrop {
  backdrop-filter: blur(0px);
  -webkit-backdrop-filter: blur(0px);
  opacity: 0;
}
```

将原 `.lock-screen.exit .lock-screen-backdrop` 保留（已有），新增 `.lock-screen.exit .lock-screen-content` 让内容区同步消散。

#### 1.4 入场雾气凝聚

**`modals.css`** — 修改 `.lock-screen`：
```css
.lock-screen {
  opacity: 0;
  /* ... 其余不变 ... */
}

.lock-screen.entering {
  opacity: 1;
  transition: opacity 0.35s ease-out;
}

@keyframes lockContentEnter {
  from { opacity: 0; transform: translateY(16px); }
  to   { opacity: 1; transform: translateY(0); }
}

.lock-screen.entering .lock-screen-content > * {
  opacity: 0;
  transform: translateY(16px);
}

.lock-screen.entering .lock-screen-icon     { animation: lockContentEnter 0.4s ease-out 0.05s both; }
.lock-screen.entering .lock-screen-title    { animation: lockContentEnter 0.4s ease-out 0.1s both; }
.lock-screen.entering .lock-screen-input-wrap { animation: lockContentEnter 0.4s ease-out 0.15s both; }
.lock-screen.entering .lock-screen-btn      { animation: lockContentEnter 0.4s ease-out 0.2s both; }
.lock-screen.entering .lock-screen-exit-btn { animation: lockContentEnter 0.4s ease-out 0.25s both; }
```

各元素 staggered 入场（50ms 间隔），图标最先浮现。

#### 1.5 密码错误时锁子受惊微颤

**`modals.css`** — 新增：
```css
@keyframes lockShake {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  20% { transform: translateY(-2px) rotate(-5deg); }
  40% { transform: translateY(-1px) rotate(5deg); }
  60% { transform: translateY(-2px) rotate(-3deg); }
  80% { transform: translateY(-1px) rotate(3deg); }
}

.lock-screen-icon.error-shake {
  animation: lockShake 0.4s ease;
}
```

---

### 文件 2: `main.js` — JS 行为调整（~20 行）

#### 2.1 入场动画（`showLockScreen` 或解锁后重新显示时）

找到设置页密码开启/应用启动时显示锁屏的代码，添加 `entering` class：

```js
// 显示锁屏
lockScreen.style.display = 'flex';
requestAnimationFrame(() => {
  lockScreen.classList.add('entering');
});

// 入场动画结束后清除 entering class（避免闪烁）
lockScreen.addEventListener('animationend', (e) => {
  if (e.target === lockScreen && lockScreen.classList.contains('entering')) {
    lockScreen.classList.remove('entering');
  }
}, { once: true });
```

#### 2.2 聚焦时锁子注视

```js
// 在 initLockScreenEvents 中
const lockIcon = document.querySelector('.lock-screen-icon');
if (input && lockIcon) {
  input.addEventListener('focus', () => lockIcon.classList.add('focused'));
  input.addEventListener('blur', () => lockIcon.classList.remove('focused'));
}
```

#### 2.3 解锁过渡时序

```js
if (ok) {
  // 解锁成功 - 内容向上飘散 + 模糊渐隐
  lockScreen.classList.add('exit');
  setTimeout(() => {
    lockScreen.style.display = 'none';
    lockScreen.classList.remove('exit');
    if (unlockBtn) unlockBtn.disabled = false;
  }, 500); // 从 450ms 调整为 500ms，配合内容区动画
}
```

#### 2.4 密码错误锁子受惊

```js
else {
  // 解锁失败
  input.classList.add('shake');
  if (lockIcon) lockIcon.classList.add('error-shake');
  // ...
  setTimeout(() => {
    input.classList.remove('shake');
    if (lockIcon) lockIcon.classList.remove('error-shake');
    input.focus();
  }, 400);
}
```

#### 2.5 清理 entrance/exit class 防止状态泄漏

在 `lockScreen.style.display = 'none'` 等处确保 class 被正确移除。

### 无需修改 `index.html`

HTML 结构完全不变，所有动效通过 CSS keyframes + JS class toggle 实现。

## 实施步骤

1. **`modals.css`**：新增 4 组 keyframes（`lockHover` / `lockContentEnter` / `lockShake`）+ 修改 `.lock-screen` / `.lock-screen-content` / `.lock-screen-icon` 样式
2. **`main.js`**：`initLockScreenEvents` 中添加聚焦注视 + 错误锁子颤 + 入场 `entering` class
3. **`npm run build`** 验证

## 验证方法
1. 启用锁屏 → 重启应用 → 观察"雾气凝聚"入场动画
2. 锁屏待机时锁子图标缓慢上下浮动
3. 点击密码输入框 → 锁子微微放大+高亮
4. 输入错误密码 → 锁子 + 输入框同时抖动
5. 输入正确密码 → 全部内容向上飘散 + 模糊渐隐后解锁
6. 切换 12 个主题，检查动画颜色适配上没有问题
