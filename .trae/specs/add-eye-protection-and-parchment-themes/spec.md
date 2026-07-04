# 新增 护眼 & 羊皮卷 主题 Spec

## Why

目前应用已有 8 套系统主题，但缺少专注于"长时间阅读舒适度"和"复古纸质感"的主题。
- **护眼**：经典的"豆沙绿"（绿豆沙色）背景，是中文互联网用户群体中广为人知的护眼配色方案，适合长时间码字和阅读。
- **羊皮卷**：模仿古旧羊皮纸的暖黄底色 + 棕墨感，满足追求复古书写体验的用户需求。

## What Changes

1. **`variables.css`** — 新增 2 个 `[data-theme]` CSS 变量块（护眼、羊皮卷），每个约 50 行 CSS 变量
2. **`main.js`** — `themeLabels` 新增 2 个条目；`codeHighlightThemePairing` 新增 2 个配对条目
3. **`index.html`** — `criticalColors` 新增 2 组防闪色；下拉菜单新增 2 个选项

### 护眼主题配色方案

参考自中文互联网经典的"豆沙绿"护眼风格。柔和低饱和的暖绿色调，减少蓝光对眼睛的刺激。

```css
--bg: #C7EDCC;             /* 经典豆沙绿背景 */
--card-bg: #D8F5DD;        /* 稍亮卡片 */
--text-primary: #2C4A3E;   /* 深墨绿主字 */
--text-secondary: #5B8A6E; /* 中绿灰次文字 */
--text-muted: #8AB89A;     /* 浅绿灰辅助文字 */
--accent: #D97706;         /* 暖琥珀色强调 — 与绿底形成温暖对比 */
--accent-rgb: 217, 119, 6;
--accent-light: #EBF5EC;   /* 极浅绿高亮背景 */
--accent-lighter: #F0F8F1; /* 更浅的悬浮背景 */
--accent-dark: #B45309;    /* 深琥珀色 */
--border: #B8DCC0;         /* 边框绿 */
--divider: #C4E4CB;        /* 分割线绿 */
--hover-bg: #D0ECD5;       /* 悬浮绿 */
--topbar-bg: #D8F5DD;      /* 顶栏同 card */
--input-bg: #D0ECD5;       /* 输入框悬浮绿 */
```

### 羊皮卷主题配色方案

参考自古老羊皮纸的暖黄 + 棕色调，营造复古书写氛围。

```css
--bg: #F5E6CA;             /* 暖黄羊皮纸背景 */
--card-bg: #FAF0DC;        /* 稍亮奶油色卡片 */
--text-primary: #3E2C1A;   /* 深棕墨主字 */
--text-secondary: #8B7355; /* 中棕次文字 */
--text-muted: #B8A080;     /* 浅棕辅助文字 */
--accent: #A0522D;         /* 铁锈棕强调色 — 复古暖感 */
--accent-rgb: 160, 82, 45;
--accent-light: #F2E4D0;   /* 暖米高亮背景 */
--accent-lighter: #F7EDDE; /* 更浅的悬浮背景 */
--accent-dark: #8B4513;    /* 深棕 */
--border: #E3D1B8;         /* 边框暖米 */
--divider: #EADCC8;        /* 分割线暖米 */
--hover-bg: #F0E4D0;       /* 悬浮暖米 */
--topbar-bg: #FAF0DC;      /* 顶栏同 card */
--input-bg: #F0E4D0;       /* 输入框悬浮暖米 */
```

## Impact

- Affected specs: 主题系统
- Affected code:
  - `frontend/src/css/variables.css` — 新增 2 个 `[data-theme]` 块
  - `frontend/src/main.js` — `themeLabels` 新增 2 个条目；`codeHighlightThemePairing` 新增 2 个条目
  - `frontend/index.html` — `criticalColors` 新增 2 组；下拉菜单新增 2 个选项

## ADDED Requirements

### Requirement: 护眼主题

系统 SHALL 新增 "护眼" 系统 UI 主题，基于豆沙绿色调。

#### Scenario: 主题切换

- **GIVEN** 用户在设置页的系统主题下拉菜单
- **WHEN** 选择"护眼"
- **THEN** 应用立即切换为柔和绿色护眼配色并持久化

#### Scenario: 主题定义完整性

- **GIVEN** 护眼主题定义已写入 `variables.css`
- **WHEN** 通过 `[data-theme="eye-protection"]` 属性激活
- **THEN** 所有 CSS 变量（基础色、语义色、分层阴影）均有覆盖

### Requirement: 羊皮卷主题

系统 SHALL 新增 "羊皮卷" 系统 UI 主题，基于羊皮纸暖黄色调。

#### Scenario: 主题切换

- **GIVEN** 用户在设置页的系统主题下拉菜单
- **WHEN** 选择"羊皮卷"
- **THEN** 应用立即切换为暖黄复古配色并持久化

#### Scenario: 主题定义完整性

- **GIVEN** 羊皮卷主题定义已写入 `variables.css`
- **WHEN** 通过 `[data-theme="parchment"]` 属性激活
- **THEN** 所有 CSS 变量（基础色、语义色、分层阴影）均有覆盖

## REMOVED Requirements

无
