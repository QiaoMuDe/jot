# 锁屏页解锁 + 退出按钮布局设计方案

## 摘要
在锁屏页面添加"退出应用"按钮，与现有的"解锁"按钮组合摆放，让用户既可以选择解锁进入应用，也可以选择直接退出程序。

## 当前状态
- HTML: `index.html#L1943-L1971` — 锁屏结构：图标 → 标题 → 输入框（含 👁 切换） → 解锁按钮 → 错误提示
- CSS: `modals.css#L825-L989` — 居中 flex 布局，`.lock-screen-btn` 全宽 `max-width:280px` 紫色圆角按钮
- JS: `main.js#L7136-L7210` — `unlockApp()` 验证密码逻辑，`Quit()` 已从 wailsjs/runtime 导入
- 当前只有"解锁"一个按钮，无退出入口

## 设计方向

**风格定位**：极简克制、静谧沉稳。锁屏是"门"——聚焦于一个核心决策（进或退），因此两个按钮应当视觉一体但层级分明。

**按钮布局**：**竖向堆叠**（保持锁屏窄居中布局），"解锁"在上为主按钮，"退出应用"在下为次要操作。

| 元素 | 视觉方案 |
|------|----------|
| **解锁按钮** | 保留现有 full-width accent 填充按钮，高度 44px，圆角 `12px` |
| **退出按钮** | 轮廓线/文字按钮风格：`border:1.5px solid var(--border)` + 透明背景，hover 时变浅色底色，文字 `--text-secondary` |
| **间距** | 两个按钮间距 `12px`，与输入框间距 `20px` |
| **宽度** | 统一 `max-width:280px`，与输入框对齐 |
| **退出按钮图标** | 左侧加一个 `log-out` SVG 图标（`<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>`），16x16，与文字间距 6px |

## 改动文件及具体变更

### 1. `frontend/index.html` — 锁屏 HTML 结构调整

**变更前**（line 1968）：
```html
<button id="lockUnlockBtn" class="lock-screen-btn">解锁</button>
<p id="lockErrorMsg" class="lock-screen-error" style="display:none;">密码错误，请重试</p>
```

**变更后**（line 1968-1971）：
```html
<button id="lockUnlockBtn" class="lock-screen-btn">解锁</button>
<button id="lockQuitBtn" class="lock-screen-exit-btn">
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
        <polyline points="16 17 21 12 16 7"/>
        <line x1="21" y1="12" x2="9" y2="12"/>
    </svg>
    退出应用
</button>
<p id="lockErrorMsg" class="lock-screen-error" style="display:none;">密码错误，请重试</p>
```

### 2. `frontend/src/css/components/modals.css` — 新增退出按钮样式

在 `.lock-screen-btn` 后面新增 `.lock-screen-exit-btn`（line 972 之后）：

```css
/* ── 退出应用按钮 ── */
.lock-screen-exit-btn {
  width: 100%;
  max-width: 280px;
  height: 44px;
  background: transparent;
  color: var(--text-muted, #6b7280);
  border: 1.5px solid var(--border, rgba(255,255,255,0.08));
  border-radius: var(--radius-lg, 12px);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.15s, background 0.15s, border-color 0.15s;
  font-family: inherit;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  box-sizing: border-box;
}

.lock-screen-exit-btn:hover {
  color: var(--text-secondary, #9ca3af);
  background: var(--hover-bg, rgba(128,128,128,0.06));
  border-color: var(--text-muted, #6b7280);
}

.lock-screen-exit-btn:active {
  transform: scale(0.98);
}

.lock-screen-exit-btn svg {
  flex-shrink: 0;
}
```

### 3. `frontend/src/main.js` — 退出按钮事件绑定

在 `initLockScreenEvents()` 中新增退出按钮事件（line 7215 附近）：

```js
// 退出应用按钮
const quitBtn = document.getElementById('lockQuitBtn');
if (quitBtn) {
    quitBtn.addEventListener('click', () => {
        // 直接退出，无需确认（锁屏页面无未保存数据）
        Quit();
    });
}
```

无需修改 `unlockApp()`，也无需后端改动。

## 设计决策说明

| 决策 | 理由 |
|------|------|
| **竖向堆叠而非并排** | 锁屏为窄居中布局（max-width:360px），竖向更契合 flow，避免两按钮并排导致空间局促 |
| **退出按钮用 outline 风格** | 与 accent 填充的解锁按钮形成清晰主次关系：解锁是正操作（进入），退出是负操作（离开）|
| **退出按钮加 exit 图标** | 图标强化"退出"语义，降低误触风险，符合 `ui-ux-pro-max` 的"no color-only meaning"规则 |
| **高度与解锁按钮一致（44px）** | 满足 Apple HIG ≥44pt 触摸目标标准，视觉上对齐 |
| **hover 效果** | 解锁 hover 上浮+投影（凸显）、退出 hover 底色微变+边框加深（轻声反馈），主次分明 |

## 验证步骤
1. `npm run build` 前端构建通过
2. 启用锁屏 → 重启应用 → 观察锁屏显示两个按钮
3. 确认"解锁"按钮功能正常（空密码提示 / 密码错误提示 / 正确密码解锁）
4. 确认"退出应用"按钮点击后应用关闭
5. 切换 12 个系统主题，确认退出按钮样式适配
