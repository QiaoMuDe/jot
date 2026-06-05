# 新增主题方案（参照 VS Code 热门主题）

## 摘要

在现有 3 个主题（default/light/dark）的基础上，参照 VS Code 热门主题新增 3 个主题：**nord**（北极蓝调）、**monokai-pro**（荧光粉墨）、**tokyo-night**（靛紫夜幕），共计 6 个主题。

## 当前状态分析

### 现有主题一览

| 主题名     | 背景色        | 强调色        | 风格    |
| ------- | ---------- | ---------- | ----- |
| default | #F7F5F0 暖灰 | #D97706 琥珀 | 暖色调经典 |
| light   | #FAFAFA 冷白 | #2563EB 蓝  | 干净明亮  |
| dark    | #0D0D0D 纯黑 | #F59E0B 琥珀 | 暗色琥珀  |

### 主题系统架构

* **CSS 变量**: `app.css` 的 `:root`（默认）和 `[data-theme="light/dark"]` 各定义 26 个变量

* **切换**: `main.js` 中 `applyTheme()` 设置 `<html data-theme="...">`，CSS 级联覆盖

* **UI 选择器**: 设置页中 `.segmented-control`（分页控件），3 个按钮 + 滑动指示器

* **持久化**: 后端 `Setting('theme')` → 降级 `localStorage('jot_theme')` → 默认 `'default'`

***

## 新增主题设计方案

### theme-4: `nord`（北极蓝调 — 参照 VS Code Nord 主题）

北极、极光、冰川灵感，低饱和度冷色调，柔和护眼。

| 变量                                       | 值                       | 说明     |
| ---------------------------------------- | ----------------------- | ------ |
| --bg                                     | `#ECEFF4`               | 冰雪灰蓝底  |
| --card-bg                                | `#FFFFFF`               | 白卡片    |
| --text-primary                           | `#2E3440`               | 极深夜蓝字  |
| --text-secondary                         | `#6B7394`               | 灰蓝     |
| --text-muted                             | `#9199B0`               | 浅灰蓝    |
| --accent                                 | `#5E81AC`               | 北极蓝强调  |
| --accent-rgb                             | `94, 129, 172`          | <br /> |
| --accent-light                           | `#D8DEE9`               | 冰蓝浅    |
| --accent-lighter                         | `#E5E9F0`               | 冰蓝极浅   |
| --accent-dark                            | `#4C6F8C`               | 深海蓝深   |
| --border                                 | `#D8DEE9`               | 边框     |
| --divider                                | `#E2E6EF`               | 分割线    |
| --hover-bg                               | `#E8ECF3`               | 悬浮     |
| --topbar-bg                              | `#FFFFFF`               | <br /> |
| --input-bg                               | `#E8ECF3`               | <br /> |
| --overlay-bg                             | `rgba(46, 52, 64, 0.4)` | <br /> |
| --toast-bg                               | `#2E3440`               | <br /> |
| --toast-text                             | `#ECEFF4`               | <br /> |
| --scrollbar-thumb                        | `rgba(0, 0, 0, 0.14)`   | <br /> |
| --scrollbar-thumb-hover                  | `rgba(0, 0, 0, 0.26)`   | <br /> |
| --scrollbar-track                        | `rgba(0, 0, 0, 0.03)`   | <br /> |
| （danger/success/tag-delete 沿用 light 主题值） | <br />                  | <br /> |

### theme-5: `monokai-pro`（荧光粉墨 — 参照 VS Code Monokai Pro 主题）

深灰底色 + 荧光粉红强调色，高饱和高对比，充满活力的视觉风格。

| 变量                      | 值                           | 说明     |
| ----------------------- | --------------------------- | ------ |
| --bg                    | `#2D2A2E`                   | 深灰底    |
| --card-bg               | `#221F22`                   | 暗灰卡片   |
| --text-primary          | `#FCFCFA`                   | 白字     |
| --text-secondary        | `#CFCFC2`                   | 灰白     |
| --text-muted            | `#939293`                   | 中灰     |
| --accent                | `#FF6188`                   | 荧光粉红强调 |
| --accent-rgb            | `255, 97, 136`              | <br /> |
| --accent-light          | `#FF8DA8`                   | 粉浅     |
| --accent-lighter        | `#FFB3C6`                   | 粉极浅    |
| --accent-dark           | `#E64C73`                   | 粉深     |
| --border                | `#3C3A3D`                   | 边框     |
| --divider               | `#3C3A3D`                   | 分割线    |
| --danger                | `#FC9867`                   | 橙红     |
| --hover-bg              | `#3C3A3D`                   | 悬浮     |
| --topbar-bg             | `#221F22`                   | <br /> |
| --input-bg              | `#3C3A3D`                   | <br /> |
| --overlay-bg            | `rgba(0, 0, 0, 0.7)`        | <br /> |
| --toast-bg              | `#221F22`                   | <br /> |
| --toast-text            | `#FCFCFA`                   | <br /> |
| --danger-bg             | `rgba(252, 152, 103, 0.1)`  | <br /> |
| --danger-border         | `rgba(252, 152, 103, 0.25)` | <br /> |
| --success-bg            | `rgba(169, 219, 113, 0.1)`  | <br /> |
| --success-border        | `rgba(169, 219, 113, 0.25)` | <br /> |
| --success-text          | `#A9DB71`                   | <br /> |
| --scrollbar-thumb       | `rgba(255, 255, 255, 0.18)` | <br /> |
| --scrollbar-thumb-hover | `rgba(255, 255, 255, 0.32)` | <br /> |
| --scrollbar-track       | `rgba(255, 255, 255, 0.04)` | <br /> |
| --tag-delete-bg         | `rgba(255, 255, 255, 0.25)` | <br /> |
| --tag-delete-hover-bg   | `rgba(255, 255, 255, 0.4)`  | <br /> |

### theme-6: `tokyo-night`（靛紫夜幕 — 参照 VS Code Tokyo Night 主题）

深蓝靛紫底色 + 霓虹蓝/紫/粉强调色，赛博朋克风格。

| 变量                      | 值                           | 说明     |
| ----------------------- | --------------------------- | ------ |
| --bg                    | `#1A1B26`                   | 深靛蓝底   |
| --card-bg               | `#24283B`                   | 暗蓝卡片   |
| --text-primary          | `#C0CAF5`                   | 浅紫白字   |
| --text-secondary        | `#7982A9`                   | 灰紫     |
| --text-muted            | `#565F89`                   | 暗灰紫    |
| --accent                | `#7AA2F7`                   | 霓虹蓝强调  |
| --accent-rgb            | `122, 162, 247`             | <br /> |
| --accent-light          | `#1F2A4A`                   | 蓝浅     |
| --accent-lighter        | `#1A2340`                   | 蓝极浅    |
| --accent-dark           | `#A9B1D6`                   | 蓝深     |
| --border                | `#2F354A`                   | 边框     |
| --divider               | `#2F354A`                   | 分割线    |
| --danger                | `#F7768E`                   | 霓虹粉红   |
| --hover-bg              | `#292E42`                   | 悬浮     |
| --topbar-bg             | `#1F2335`                   | <br /> |
| --input-bg              | `#292E42`                   | <br /> |
| --overlay-bg            | `rgba(0, 0, 0, 0.7)`        | <br /> |
| --toast-bg              | `#24283B`                   | <br /> |
| --toast-text            | `#C0CAF5`                   | <br /> |
| --danger-bg             | `rgba(247, 118, 142, 0.1)`  | <br /> |
| --danger-border         | `rgba(247, 118, 142, 0.25)` | <br /> |
| --success-bg            | `rgba(158, 206, 106, 0.1)`  | <br /> |
| --success-border        | `rgba(158, 206, 106, 0.25)` | <br /> |
| --success-text          | `#9ECE6A`                   | 青草绿    |
| --scrollbar-thumb       | `rgba(255, 255, 255, 0.18)` | <br /> |
| --scrollbar-thumb-hover | `rgba(255, 255, 255, 0.32)` | <br /> |
| --scrollbar-track       | `rgba(255, 255, 255, 0.04)` | <br /> |
| --tag-delete-bg         | `rgba(255, 255, 255, 0.25)` | <br /> |
| --tag-delete-hover-bg   | `rgba(255, 255, 255, 0.4)`  | <br /> |

***

## 具体改动

### 1. `frontend/src/app.css` — 新增 3 个 `[data-theme]` 块

在 `[data-theme="dark"]` 闭括号后追加 3 个完整的 `[data-theme]` 块。

### 2. `frontend/index.html` — 更新主题分段控件为 6 按钮

```html
<div class="segmented-control theme-segmented" id="themeControl">
    <div class="segmented-indicator" id="themeIndicator"></div>
    <button class="segmented-btn" data-theme-value="default">默认</button>
    <button class="segmented-btn" data-theme-value="nord">北极</button>
    <button class="segmented-btn" data-theme-value="monokai-pro">粉墨</button>
    <button class="segmented-btn" data-theme-value="light">浅色</button>
    <button class="segmented-btn" data-theme-value="tokyo-night">夜幕</button>
    <button class="segmented-btn" data-theme-value="dark">深色</button>
</div>
```

按明度排列：默认(暖灰)→北极(浅蓝灰)→粉墨(深灰)→浅色(冷白)→夜幕(深蓝)→深色(纯黑)

### 3. `frontend/src/main.js` — 更新主题切换逻辑

`applyTheme()` + `initThemeSettings()` 中指示器计算改为动态按钮数：

```js
const segW = (cw - btns.length * 4) / btns.length;
els.themeIndicator.style.transform = `translateX(${2 + index * (segW + 4)}px)`;
```

### 4. `frontend/src/style.css` — 更新 `.theme-segmented` 宽度

从 200px 改为 320px。

***

## 影响范围

| 文件                       | 改动                                                  |
| ------------------------ | --------------------------------------------------- |
| `frontend/src/app.css`   | 新增 nord/monokai-pro/tokyo-night 三个 `[data-theme]` 块 |
| `frontend/index.html`    | 主题分段控件从 3 按钮改为 6 按钮                                 |
| `frontend/src/main.js`   | `applyTheme()` + `initThemeSettings()` 改为动态按钮数计算    |
| `frontend/src/style.css` | `.theme-segmented` 宽度 200px → 320px                 |

***

## 设计决策

1. **参考 VS Code 热门主题**: nord、monokai-pro、tokyo-night 是 VS Code 下载量 top 10 的经典主题
2. **明度排列按钮顺序**: 从浅到深渐变
3. **不改** **`loadThemeSetting()`**: 已支持任意主题名
4. **不改后端**: 纯前端 CSS 变量，持久化走现有 Setting 机制

## 验证

1. `npm run build` 构建通过
2. 6 个主题选择正确，指示器位置计算正常
3. 每个主题配色正确

