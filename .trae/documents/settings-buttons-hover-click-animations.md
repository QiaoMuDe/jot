# 设置页按钮悬停与点击反馈动画优化

## 摘要

设置页按钮（包含 `.btn-save`、`.btn-cancel`、`.btn-secondary`、`.btn-danger`、`.btn-link`、`.segmented-btn`、`.font-size-btn`、`.preset-modal-close`、`.tag-delete-btn` 等）当前缺少 `:active` 微交互动画、点击反馈和 `:focus-visible` 样式，悬停效果也比较基础。

本次优化将统一提升所有按钮的交互反馈品质，引入弹性缩放（按下缩小）、点击闪光（点击瞬间亮度脉冲）、焦点环等机制，同时复用项目已有的视觉效果变量。

## 当前状态分析

**问题清单：**
1. **缺少 `:active`（按下）状态**（第 13 项）—— 没有任何按钮定义了 `:active`，用户点击时无任何物理按压反馈
2. **`.btn-save` 悬停效果混杂** —— 同时使用 `background: var(--accent-dark)` + `filter: brightness(1.1)`，二者叠加效果不可预测
3. **`.btn-cancel` 悬停色差不够明显** —— `--hover-bg` → `--border` 的对比度很低
4. **`.segmented-btn` 缺少悬停效果** —— 分段按钮只有 `.active` 状态，hover 无任何反馈
5. **`.font-size-btn` 悬停仅仅是边框变色** —— 可以更丰富
6. **没有焦点环** —— 键盘导航用户无法感知焦点位置
7. **过渡曲线过于简单** —— `150ms ease-out` 缺少弹性/个性化

**已有可复用资源：**
- `--anim-easing-spring`: `cubic-bezier(0.16, 1, 0.3, 1)` —— 弹性缓动曲线
- `--anim-easing-out`: `ease-out`
- `pulseClick` 动画关键帧：`scale(1) → 1.12 → 0.95 → 1`（0.35s）
- `--radius-*` 尺寸变量
- `--shadow-sm/lg/modal` 阴影变量

## 变更内容

| # | 文件 | 变更 |
|---|------|------|
| 1 | `settings-panel.css` | 新增全局按钮 `:active` 状态（按下缩小）；重写按钮悬停效果（统一使用 transform + brighten pattern）；添加 `:focus-visible` 焦点环；为 `.segmented-btn` 添加鼠标悬停效果；优化 `.font-size-btn` 悬停；替换过渡曲线为 spring 缓动 |
| 2 | `main.js` | 在新旧交替时需确认没有冲突的 JS 样式覆盖 |

## 影响范围

- 仅 CSS 改动（极少量 JS 辅助代码）
- 不影响布局、按钮尺寸、颜色变量 —— 只改变交互反馈表现
- 区分正常悬停、按下、点击三个阶段
- 所有按钮类型统一覆盖

## 详细实现

### 文件 1：`frontend/src/css/components/settings-panel.css`

#### 1.1 新增 `.btn` 通用 `:active` 状态（约第 5~14 行之间）

```css
.btn:active {
  transform: scale(0.97);
}
```

效果：所有按钮按下时微缩 3%，松开回弹（此时已在 transition 里声明了 `transition: var(--transition)`，会自动恢复）。

**注意事项**：`.btn` 默认 `transition: var(--transition)` 只包含 `all 150ms ease-out`，其中的 `all` 已覆盖 `transform`，在松开时直接恢复到 `scale(1)`。但为了更好回弹，需要将过渡曲线改为更自然的曲线。

#### 1.2 优化 `.btn` 默认过渡曲线

```css
.btn {
  /* ... 原有属性 ... */
  transition: all var(--anim-duration-fast) var(--anim-easing-spring);
}
```

将 `150ms ease-out` 改为 `150ms cubic-bezier(0.16, 1, 0.3, 1)` —— 按下压缩时更快（150ms），松开回弹带有轻微过冲，手感更自然。

**说明**：只改 `.btn` 类，不影响输入框、选择器等控件的过渡。

#### 1.3 优化 `.btn-save` 悬停（第 37~40 行）

```css
.btn-save:hover {
  background: var(--accent-dark);
  filter: brightness(1.05);
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}
```

将 brightness 从 1.1 调整到 1.05（减弱叠加效果），同时增加轻微阴影悬浮感。

#### 1.4 优化 `.btn-cancel` 悬停（第 27~30 行）

```css
.btn-cancel:hover {
  background: var(--border);
  color: var(--text-primary);
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0,0,0,0.04);
}
```

添加上浮 1px 和轻微阴影，让 hover 效果有物理感知。

#### 1.5 新增 `.btn-secondary` 悬停增强（第 739~742 行）

```css
.btn-secondary:hover {
  background: var(--border);
  color: var(--text-primary);
  transform: translateY(-1px);
}
```

#### 1.6 新增 `.btn-danger` 悬停增强（第 748~750 行）

```css
.btn-danger:hover {
  filter: brightness(1.1);
  box-shadow: 0 2px 8px rgba(220, 38, 38, 0.15);
}
```

添加红色阴影（使用 `--error` 值），悬停时更明显的危险提示感。

#### 1.7 新增按钮 `:focus-visible` 焦点环（全局）

在所有按钮类之后统一添加：

```css
.btn:focus-visible,
.segmented-btn:focus-visible,
.font-size-btn:focus-visible,
.preset-modal-close:focus-visible,
.tag-delete-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
```

无侵入键盘焦点指示（不破坏 UI，对鼠标用户不可见）。

#### 1.8 新增 `.segmented-btn` 悬停效果（第 363~384 行）

```css
.segmented-btn:hover {
  color: var(--text-primary);
  background: rgba(0,0,0,0.03);
}
```

分段按钮悬停时文字变深 + 背景微变（极浅色遮罩）。

#### 1.9 优化 `.font-size-btn` 悬停（第 73 行附近）

```css
.font-size-btn:hover {
  border-color: var(--accent-light);
  background: var(--accent-lighter);
  transform: translateY(-1px);
}
```

悬停时点亮边框 + 背景 + 上浮。

#### 1.10 优化 `.preset-modal-close` 悬停（第 870~873 行）

```css
.preset-modal-close:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
  transform: scale(1.1);
}
```

悬停时微微放大 10%，消歧义（告诉用户这是一个可点击按钮）。

#### 1.11 优化 `.tag-delete-btn` 悬停（第 218~220 行）

```css
.tag-delete-btn:hover {
  background: var(--tag-delete-hover-bg);
  transform: scale(1.15);
}
```

悬停时放大 15%，更明确可点击感。

### 文件 2：`frontend/src/main.js`

无需变更 JS 逻辑。所有动画都通过 CSS 实现，无侵入。

## 假设与决策

| 决策 | 原因 |
|------|------|
| 不使用 JS 驱动的 ripple 动效 | CSS 方案更轻量，硬件加速无卡顿，且容易维护 |
| 统一使用 `transform: scale(0.97)` 作为按下状态 | 0.97 是业界最佳实践（Google Material Design 使用 ~0.96-0.98），手感适中 |
| 不使用 `filter: brightness(...)` 作为按下反馈 | `brightness` 会造成文字同时变暗/变亮，scale 方案更优雅 |
| 悬停同时增加 `box-shadow` + `translateY(-1px)` | 物理上浮 + 阴影深度增加模拟"抬起"效果，与按下的"压入"形成对比 |
| 采用 spring 缓动曲线 | 项目中已定义 `--anim-easing-spring`，且 micro-interaction 使用弹性曲线感知更好 |
| 不添加 `prefers-reduced-motion` 特殊处理 | 项目已有全局阻断，无需每个组件单独处理 |

## 验证步骤

1. **编译验证**：`go build ./...` —— 前端无改动涉及编译
2. **视觉检查**：手动确认设置页所有按钮的 hover/active/focus 状态
3. **键盘导航**：Tab 键导航确认所有按钮都有可见焦点环
4. **回归验证**：确认 `.btn-loading` 状态不受影响（`pointer-events: none` 应阻断 `:active`）
5. **分段控件**：确认 `.segmented-btn` 在选中后 hover 无异常
