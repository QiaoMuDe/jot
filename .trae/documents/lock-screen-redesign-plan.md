# 锁屏重设计计划 — "Crystalline Guard" 晶态守卫

## 设计方向

**Aesthetic**: Refined luxury / Tech-noir — 深色背景 + 晶态玻璃容器 + 轨道能量环  
**灵感**: iOS 26 Liquid Glass 材质系统 + 三星 One UI FlexWindow 分层动效 + 门户网站的旋转流光边框  
**记忆点**: 4 层能量轨道环围绕晶态玻璃锁旋转，交互状态变化时环速/颜色/发光自适应

## 改动范围

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `frontend/index.html` | 微改 | 锁图标外围插入 4 个 `.lock-ring` div + 1 个 `.lock-icon-glass`(共 5 个新元素) |
| `frontend/src/css/components/modals.css` | CSS 核心 | 保留现有结构，重写锁屏全部样式(约 200 行→550 行) |
| `frontend/src/main.js` | 微改 | 延长 entering timeout (500→900ms)，add/remove `error` class 联动 |

## 具体改动

### 1. HTML — 锁屏结构 (`index.html` #L1944-L1979)

**目前**: 简单 div 结构 — `.lock-screen` > `.lock-screen-backdrop` + `.lock-screen-content`  
**改为**: 在 `.lock-screen-icon` 内插入 4 个 `.lock-ring` + 1 个 `.lock-icon-glass` 容器

```html
<div class="lock-screen-icon">
    <div class="lock-ring ring-1"></div>     <!-- 内层实线环，最快 -->
    <div class="lock-ring ring-2"></div>     <!-- 虚线环，反向 -->
    <div class="lock-ring ring-3"></div>     <!-- 渐变 border 环 -->
    <div class="lock-ring ring-4"></div>     <!-- 点状虚线环，最慢 -->
    <div class="lock-icon-glass"></div>       <!-- 晶态玻璃圆形容器 -->
    <svg width="44" height="44" ...>          <!-- 锁 SVG（原有） -->
</div>
```

其余 HTML 元素完全不变。

### 2. CSS — 锁屏样式重写 (`modals.css` #L829-L1096 → 完整替换)

#### 保留不动的属性
- 原有 `.lock-screen` 容器样式（fixed/inset-0/z-index/flex）
- 输入框/按钮/退出按钮的基本尺寸和交互
- 错误提示文字样式

#### 新增/重写内容

**A. 背景装饰**
- `.lock-screen-backdrop`: `opacity: 0` → `opacity: 1` + `backdrop-filter: blur(16px)`（入场后保持可见）
- 新增 `.lock-ambient-orb.orb-1/2/3`: 3 个大面积模糊渐变光晕作为背景景深（`filter: blur(80px)`），缓慢漂浮

**B. 晶态玻璃卡片**
- 新增 `.lock-screen-glass-card`: 包裹 `.lock-screen-content` 的毛玻璃容器
  - `background: color-mix(in srgb, var(--card-bg) 70%, transparent)`
  - `backdrop-filter: blur(24px) saturate(1.4)`
  - `border-radius: 24px` + 1px 半透明 accent 边框
  - 多层 `box-shadow`（深色投影 + 薄内发光 + 边缘光）
  - `::before` 斜向渐变光点缀
  - `::after` 顶部扫描光横线（20%→80% 渐变）
  - 入场前: `opacity: 0; transform: translateY(24px) scale(0.88)`
  - 入场后: `opacity: 1; transform: none`

**C. 轨道环系统 — 4 层同心圆**
- `.lock-ring`: 基类 — `position: absolute; border-radius: 50%` + `pointer-events: none`
- `ring-1` (最内): `inset: -8px`，实线 `border-top-color: var(--accent)`，3s 旋转
- `ring-2`: `inset: -18px`，`dashed` 虚线，5s 反向旋转
- `ring-3`: `inset: -28px`，2px 梯度 border，7s 旋转
- `ring-4` (最外): `inset: -38px`，`dotted` 点线，10s 反向旋转
- 每个环独立 `@keyframes ringOrbit1/2/3/4`

**D. 晶态锁图标容器**
- 新增 `.lock-icon-glass`: `position: absolute; inset: 0; border-radius: 50%`
  - `radial-gradient` 从右上到左下 accent 渐变填充
  - 1px 半透明边框 + 多层 inset shadow
  - 待机时 `animation: glassPulse 3s ease-in-out infinite` 呼吸

**E. 5 种状态动画**

| 状态 | 触发 Class | 动画内容 |
|------|-----------|----------|
| 入场 | `.entering` | 背景 fadeIn 0.5s → 光晕 fadeIn 1.2s → 卡片 `cardCrystalEnter` 0.6s spring → 4环 `ringFlyIn` 依次 0.5s spring → 内容 stagger 0.45s |
| 待机 | 无（默认） | 4 环持续旋转（3/5/7/10s），锁图标 `lockFloat` 3s 呼吸，玻璃容器 `glassPulse` 3s，3 光晕 `orbFloat` 12/16/14s |
| 聚焦 | `.focused` | 环速翻倍（1.5/3/4/5s），环加亮，玻璃容器发光增强 1.5x，锁图标 scale 1.06 + drop-shadow，卡片边框高亮 |
| 错误 | `.error-shake` + `.error` | 4 环变红色 + `ringErrorBurst` 爆发再收缩，玻璃容器 `glassErrorPulse` 红闪，锁 `lockShake` 0.45s，卡片 `cardErrorShake` 0.4s 震颤 |
| 退出 | `.exit` | 背景 fadeOut 0.5s，卡片 `cardCrystalExit` 0.55s 上飘收缩，锁 `shackleOpen` 0.45s + `bodyDrop` + `glassDissolve`，4 环 `ringExplode` 爆炸扩散，内容 `contentExitUp` 0.4s 上飘 |

**F. @keyframes 清单（20 个）**

| Keyframe | 用途 |
|----------|------|
| `lockBackdropEnter` | 背景模糊渐入 |
| `lockBackdropExit` | 背景模糊渐出 |
| `cardCrystalEnter` | 玻璃卡片 spring 弹入 |
| `cardCrystalExit` | 玻璃卡片上飘收缩 |
| `lockContentEnter` | 内容元素逐个淡入上移 |
| `contentExitUp` | 内容元素上飘淡出 |
| `ringOrbit1/2/3/4` | 4 环持续旋转（不同速度/方向） |
| `ringFlyIn` | 环飞入入场（-90°→0° 弹簧） |
| `ringExplode` | 环爆炸扩散（scale 2.5） |
| `ringErrorBurst` | 环错误爆发（scale 1.15→1） |
| `glassAppear` | 玻璃容器出现（scale 0.5→1） |
| `glassDissolve` | 玻璃容器溶解（scale 1→1.3 + fade） |
| `glassPulse` | 玻璃容器呼吸脉动 |
| `glassErrorPulse` | 玻璃容器错误红闪 |
| `lockFloat` | 锁 SVG 待机悬浮 |
| `lockShake` | 锁 SVG 错误受惊多段微颤 |
| `lockInputShake` | 输入框水平抖动 |
| `cardErrorShake` | 卡片水平震颤 |
| `lockErrorFadeIn` | 错误文字淡入 |
| `orbFloat` | 背景光晕漂浮 |
| `shackleOpen` | 解锁时锁梁旋转打开 |
| `bodyDrop` | 解锁时锁体下沉 |

### 3. JS — 两处微调 (`main.js`)

**A. entering timeout**: 500ms → 900ms（适配更长的入场动画链最后一个元素 ~870ms）
```js
// 原来
setTimeout(() => { lockScreen.classList.remove('entering'); }, 500);
// 改为
setTimeout(() => { lockScreen.classList.remove('entering'); }, 900);
```

**B. unlockApp 错误状态**: 为 `.lock-screen` 容器添加/移除 `error` class
- 密码为空 / 密码错误时: `lockScreen.classList.add('error')`
- 错误结束后: `lockScreen.classList.remove('error')`
- 解锁成功时: `lockScreen.classList.remove('error')`

## 不修改的部分
- 密码输入框基本尺寸和功能
- 显示/隐藏密码按钮
- 退出应用按钮
- 密码验证逻辑
- 其他所有非锁屏的 CSS 和 JS

## 构建验证
`cd frontend && npm run build`
