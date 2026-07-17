# 锁屏动画方案 V2 — "Security Vault"

## 设计方向
参考 "Context Lock Login Form" 的同心旋转环 + 锁梁 `d: path()` 变形动画，打造"安全保险库"的视觉氛围。

## 需要新增的 HTML 元素

**`index.html`** — 在 `.lock-screen-icon` 内添加 3 个旋转环容器，SVG 保持原样（不作拆分，因为新设计不依赖独立锁梁/锁体动画）：

```html
<div class="lock-screen-icon">
    <div class="lock-ring ring-1"></div>
    <div class="lock-ring ring-2"></div>
    <div class="lock-ring ring-3"></div>
    <svg width="48" height="48" viewBox="0 0 24 24" ...>
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
        <path d="M7 11V7a5 5 0 0110 0v4"/>
    </svg>
</div>
```

## 动画设计

### 待机扫描
| 元素 | 动画 | keyframe |
|------|------|----------|
| ring-1 | 顺时针 3s 旋转，border-top 着 accent 色 | `ringSpin1` |
| ring-2 | 逆时针 4s 旋转，border-right 着 accent 色偏亮 | `ringSpin2` |
| ring-3 | 顺时针 5s 旋转，border-bottom 着 accent 色偏暗 | `ringSpin3` |
| SVG 锁 | 微弱呼吸悬浮 ±3px 2.5s | `lockHover` 保留但调大幅度 |

### 入场
- 3 个 ring 旋转从停止加速到目标速度（用 opacity 渐变 + 旋转快速启动）
- 锁子从 scale(0.8) + translateY(10px) 弹性弹入
- 持续 500ms，0.65s 内 settling

### 聚焦
- 三个 ring 加速（duration 减半）
- 锁子 scale(1.05) + 高亮
- SVG 增加 drop-shadow 光晕

### 错误
- 三个 ring 变红（border-color 切换）
- 锁子剧烈抖动（已有 shake class）
- 红色脉冲光晕扩散（新增）

### 解锁
- ring 向外扩大 + 淡出消散
- 锁子缩小 scale(0.3) 消失
- 整体内容向上飘走（已有）

## 修改范围

| 文件 | 改动 |
|------|------|
| `index.html` | `.lock-screen-icon` 内新增 3 个 div 环 + 保留原 SVG |
| `modals.css` | 重写 `.lock-screen-icon` 区域 + 新增 10+ keyframes |
| `main.js` | 无改动（class 接口不变） |

## 验证
见之前的方案验证列表。
