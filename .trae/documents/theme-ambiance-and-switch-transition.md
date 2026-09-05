# 主题氛围质感 + 主题切换连贯

## Summary
针对用户选择的两个体验优化方向，实现两处低风险改动：
1. **主题氛围**：为背景叠加一层"似有若无"的光晕渐变，让纯色 `--bg` 不再平铺、更具层次。
2. **主题切换**：用 Web 平台原生 `View Transition` 做整体 cross-fade；在不支持或处于 `prefers-reduced-motion` 时，安全回落到现状（直接切换，不做全局透明度淡出，避免白/黑闪屏）。

全程遵守项目既有约定：主题配色放各主题块内维护、尊重 `prefers-reduced-motion`、改动克制、零回归。

## Current State Analysis（基于实际探索）
- **背景绘制**：`html`/`body`（[reset.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/reset.css#L9-L29)）与最大内容面 `#mainContent`（[main-content.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/main-content.css#L5-L12)）均用 `background-color: var(--bg)`。由于内层表面会再绘制纯色 `--bg`，仅在 `body` 加光晕会被覆盖，因此必须**同时在 `html`、`body`、`#mainContent` 这类大面积 `--bg` 表面**加 `background-image`。
- **全局变量**：11 套主题在 [variables.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/variables.css) 各自维护配色；`--accent` 每主题唯一，可用于衍生氛围色。项目已安全使用 `color-mix()`，可放心用它做主题自适应。
- **主题切换入口**：运行时切换到主题的唯一入口是 `applyTheme(themeName)`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L1711-L1724)），内部 `document.documentElement.setAttribute('data-theme', themeName)`。首屏 `data-theme` 由 index.html 内联脚本在 CSS 加载前提早设置，无需动。
- **动效约定**：动画库已有且包含 `prefers-reduced-motion` 全局降级（[animations.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/animations.css#L218-L225)），本次改动沿用同一理念。

## Proposed Changes

### 1) 主题氛围 · 仅光晕渐变
**为什么用"仅光晕"**：比噪点干净克制、回归风险最低，且能各主题自动适配。

**新增变量**（[variables.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/variables.css)）
- 在顶部 `:root` 共享令牌块（非底部堆加）增加一个**主题自适应**的 `--bg-glow`：
  ```css
  --bg-glow: radial-gradient(
      120% 90% at 50% 0%,
      color-mix(in srgb, var(--accent) 7%, transparent) 0%,
      transparent 60%
  );
  ```
  用每主题独有的 `--accent` 按低透明度 + 末尾透明生成光晕，所有主题一行天然适配；保持光晕柔和（7% 起、60% 处透明），永不遮挡内容。
- 允许个别主题在块内覆盖 `--bg-glow`（如暗色/特殊主题想调略强或略弱时可写回），默认不覆盖即可。

**应用光晕**（[index.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/index.css) 与两块具体样式）
- 在 `index.css`（入口）追加一组极简规则，给主要 `--bg` 大面积叠加光晕：
  ```css
  /* 主题光晕：覆盖在纯色背景之上，不改变背景色本身 */
  html, body, #mainContent { background-image: var(--bg-glow); }
  ```
- **注意**：`#mainContent` 当前 `background: var(--bg)` 是简写，会重置 `background-image`。因此需把 `#main-content` 那条改为显式两段（见第 2 小点），或让上面的组规则以更高优先级/靠后顺序把 `background-image` 覆盖回去。为保证稳定，采用下面"显式分离"。
  - [main-content.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/main-content.css#L9)：`background: var(--bg);` → `background-color: var(--bg); background-image: var(--bg-glow);`
- `html`/`body` 的 [reset.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/reset.css) 用的是 `background-color` 非简写，组规则的 `background-image` 会正常叠加，无需改动 reset.css。
- （可选，若侧栏主列表用 `--bg` 大面积绘制则同样叠加，否则跳过以控制范围。）

**设计约束**：
- 光晕为静态渐变、无动画，天然兼容 `prefers-reduced-motion`。
- 透明度控制在 5%–9% 区间，深色主题可略高、浅色主题略低（通过主题块覆盖微调）。

### 2) 主题切换 · View Transition + 回落
**改动函数** [applyTheme](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L1711-L1724)。

**实现**（将函数体改为 `apply` + 带条件的包裹）：
```js
function applyTheme(themeName) {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const apply = () => {
        document.documentElement.setAttribute('data-theme', themeName);
        if (els.themeLabel) els.themeLabel.textContent = themeLabels[themeName] || themeName;
        if (els.themeDropdown) els.themeDropdown.querySelectorAll('.theme-select-item').forEach(item => {
            item.classList.toggle('active', item.dataset.themeValue === themeName);
        });
        updateCodeHighlightThemePairing(themeName);
    };
    if (!reduced && document.startViewTransition) {
        document.startViewTransition(apply); // 原生 cross-fade（含默认 snapshots 混合）
    } else {
        apply(); // 回落：直接切换，无白/黑闪屏风险
    }
}
```
- **主路径**：`document.startViewTransition(apply)` 触发一次整体 cross-fade（默认混合即可，无需额外 keyframes）。
- **回落**：WebView2 不支持，或 `prefers-reduced-motion` → 直接 `apply()`，保持现状零风险。
- **为何回落不选全局透明度遮蔽**：对 `html`/`body` 整体降 opacity 会在浅色切换瞬间透出底色造成闪屏，体验更差；直接切换最稳。此为明确设计取舍。

**可选增强**（视预览效果决定是否加）：
- 在 `index.css` 追加一段非常克制的 view-transition 样式，避免默认混合在浅色下"变白一帧"：
  ```css
  ::view-transition-old(root) { animation: 140ms var(--anim-easing-out) both themeOut; }
  ::view-transition-new(root) { animation: 140ms var(--anim-easing-out) both themeIn; }
  @keyframes themeOut { to { opacity: 0; } }
  @keyframes themeIn  { from { opacity: 0; } }
  ```
  默认已够则省略，避免过度设计。

## Assumptions & Decisions
- **"仅光晕渐变"**：不做噪点，降低风险、保持克制。
- **氛围用 `--accent` 推导**：一行 `:root` 自适应所有主题；个别主题可块内覆盖。这与"不底部堆加、在共享令牌区维护自适应规则"一致。
- **切换回落=直接切换**：不支持 View Transition 或省动效时直接换主题，不做全局透明度淡出（避免闪屏）。此为用户所选方案的**安全回落形式**。
- 只动 `variables.css`、`index.css`、`main-content.css`、`main.js`（视预览追加主 CSS），不做任何行为/结构改动。

## Verification
1. `cd frontend && npm run build`（或项目既有构建命令）确认 CSS/JS 构建通过、无语法错误。
2. 运行应用，逐一切换 11 套主题：
   - 支持环境下应有一次整体 cross-fade，无白/黑闪屏、无闪烁；
   - 校验各主题编辑器/内容面能看到极淡的光晕渐变，且不影响文字可读性。
3. 系统开启"减少动态效果"后，切换主题应为直接切换、无动画。
4. 抽查具有 `--overlay-bg` 半透明遮罩的场景（如弹窗/锁屏），确认光晕不产生视觉干扰或层叠问题。
5. 浅色与深色主题各选一代表色，肉眼确认光晕强度恰到好处（"似有若无"）。