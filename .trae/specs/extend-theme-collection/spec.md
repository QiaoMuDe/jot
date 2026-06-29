# 扩展主题收藏集 Spec

## Why

当前应用已有 6 套系统主题和 5 套代码高亮主题，但覆盖范围仍存在明显缺口。本次扩展参考 VS Code Marketplace 上经过验证的流行主题（Catppuccin 120 万+ 安装、Gruvbox、Dracula 500 万+ 安装、Ayu、Material Theme），引入真实的主题配色方案，而非自创配色。

## What Changes

### 新增系统主题（6 套）

| 主题 ID | 展示名 | 参考来源 | 色系 | 填补的缺口 |
|---------|--------|---------|------|-----------|
| `catppuccin-latte` | 暖咖 | Catppuccin Latte (VS Code) | 暖粉彩浅色 | 柔和粉彩 × 浅色 |
| `catppuccin-mocha` | 暖夜 | Catppuccin Mocha (VS Code) | 暖粉彩深色 | 柔和粉彩 × 深色 |
| `gruvbox-light` | 旧纸 | Gruvbox Light (VS Code) | 暖大地浅色 | 复古大地 × 浅色 |
| `gruvbox-dark` | 陈酿 | Gruvbox Dark (VS Code) | 暖大地深色 | 复古大地 × 深色 |
| `ayu-mirage` | 暮光 | Ayu Mirage (VS Code) | 蓝灰金暖科技 | 冷底暖光 × 混合 |
| `dracula` | 德古拉 | Dracula Official (VS Code) | 深紫高饱和 | 最高对比度个性深色 |

> 配色数据来源：
> - Catppuccin: [catppuccin.com/palette/](https://catppuccin.com/palette/) — 官方色板
> - Gruvbox: [github.com/morhetz/gruvbox](https://github.com/morhetz/gruvbox) — 官方规范
> - Ayu: [github.com/ayu-theme](https://github.com/ayu-theme) — 官方色板
> - Dracula: [draculatheme.com/spec](https://draculatheme.com/spec) — 官方语法高亮规范

### 新增代码高亮主题（6 套）

| 主题 ID | 展示名 | 参考来源 | 风格 | 配对系统主题 |
|---------|--------|---------|------|-------------|
| `catppuccin-mocha` | Catppuccin | Catppuccin Mocha VS Code | 柔粉彩深色 | catppuccin-mocha |
| `gruvbox-dark` | Gruvbox | Gruvbox Dark VS Code | 复古大地色 | gruvbox-dark |
| `dracula` | Dracula | Dracula Official VS Code | 深紫高饱和 | dracula |
| `ayu-mirage` | Ayu Mirage | Ayu Mirage VS Code | 暖金科技色 | ayu-mirage |
| `material-palenight` | Material Palenight | Material Theme VS Code | 紫蓝经典 | — |
| `github-light` | GitHub Light | GitHub Light VS Code | 纯净浅色 | light（已有） |

> 代码高亮配色数据来源：各主题 VS Code 扩展的官方 JSON 主题文件中的 `tokenColors` / `semanticTokenColors` 定义。

## Impact

- Affected specs: 主题系统、代码高亮主题配置
- Affected code:
  - `frontend/src/css/variables.css` — 新增 6 个 `[data-theme]` 块
  - `frontend/src/js/cm6-syntax-highlight.js` — 新增 6 个 `HighlightStyle` 定义 + 注册
  - `frontend/index.html` — 更新主题选择器 UI
  - `frontend/src/css/components/settings-panel.css` — 调整分段控件样式（两行布局）

## ADDED Requirements

### Requirement: 系统主题扩展

系统 SHALL 新增 6 套系统 UI 主题。

#### Scenario: 主题切换

- **GIVEN** 用户在设置页的系统主题分段控件
- **WHEN** 选择新增主题（catppuccin-latte / catppuccin-mocha / gruvbox-light / gruvbox-dark / ayu-mirage / dracula）
- **THEN** 应用立即切换为该主题并持久化

#### Scenario: 主题定义完整性

- **GIVEN** 新增主题定义已写入 `variables.css`
- **WHEN** 通过 `[data-theme]` 属性激活
- **THEN** 所有 CSS 变量（基础色、语义色、分层阴影）均有覆盖，无变量缺失导致视觉断裂

### Requirement: 代码高亮主题扩展

系统 SHALL 新增 6 套代码语法高亮主题。

#### Scenario: 主题切换

- **GIVEN** 用户在设置页的代码高亮主题下拉菜单
- **WHEN** 选择新增主题
- **THEN** 编辑器中的代码高亮立即切换，`.md` 笔记的 Markdown 高亮不受影响

#### Scenario: 主题对等性

- **GIVEN** 新增的代码高亮主题定义了完整的语法节点
- **WHEN** 被选择
- **THEN** 关键字、类型、函数、字符串、注释、运算符等主要节点均有配色覆盖

### Requirement: 设置页 UI 适配

系统 SHALL 适配 12 个系统主题的 UI 布局。

#### Scenario: 分段控件两行布局

- **GIVEN** 系统主题数从 6 增至 12
- **WHEN** 设置页渲染主题选择器
- **THEN** 分段控件分两行排列（每行 6 个），保持按钮可读尺寸

### Requirement: 主题配对引导（可选）

系统 SHALL 在代码高亮主题下拉菜单中，对与当前系统主题配对的高亮主题添加视觉标记。

#### Scenario: 配对标记

- **GIVEN** 用户选择了 `catppuccin-mocha` 系统主题
- **WHEN** 打开代码高亮主题下拉菜单
- **THEN** `Catppuccin` 选项旁显示推荐配对标记

## REMOVED Requirements

无

## 主题配色参考

以下所有系统主题配色均参考对应 VS Code 主题的官方色板。

---

### catppuccin-latte（Catppuccin Latte — 暖粉彩浅色）

来源：https://catppuccin.com/palette/ — Latte flavor

```css
[data-theme="catppuccin-latte"] {
  --bg: #EFF1F5;           /* base */
  --card-bg: #FFFFFF;       /* 白色卡片 */
  --text-primary: #4C4F69;  /* text */
  --text-secondary: #8C8FA7;/* overlay 1 */
  --text-muted: #BCC0CC;    /* surface 1 */
  --accent: #DD7878;        /* flamingo — 暖粉 */
  --accent-rgb: 221, 120, 120;
  --accent-light: #E6E9EF;  /* mantle */
  --accent-lighter: #F0F2F6;/* 比 mantle 更浅 */
  --accent-dark: #C86868;
  --border: #DCE0E8;        /* crust */
  --divider: #E6E9EF;       /* mantle */
  --hover-bg: #E6E9EF;      /* mantle */
  --topbar-bg: #FFFFFF;
  --input-bg: #E6E9EF;      /* mantle */
  /* 语义色、阴影等参考已有 light 主题比例 */
}
```

### catppuccin-mocha（Catppuccin Mocha — 暖粉彩深色）

来源：https://catppuccin.com/palette/ — Mocha flavor

```css
[data-theme="catppuccin-mocha"] {
  --bg: #1E1E2E;           /* base */
  --card-bg: #181825;       /* mantle */
  --text-primary: #CDD6F4;  /* text */
  --text-secondary: #A6ADC8;/* subtext 0 */
  --text-muted: #6C7086;    /* overlay 0 */
  --accent: #F38BA8;        /* red — Catppuccin 标志性暖粉 */
  --accent-rgb: 243, 139, 168;
  --accent-light: #363A4F;  /* surface 0 */
  --accent-lighter: #2E3248;
  --accent-dark: #F5A9C0;
  --border: #313244;        /* surface 1 */
  --divider: #313244;
  --hover-bg: #313244;      /* surface 1 */
  --topbar-bg: #181825;     /* mantle */
  --input-bg: #313244;      /* surface 1 */
  /* 语义色、阴影参考已有 dark 主题比例 */
}
```

### gruvbox-light（Gruvbox Light — 暖大地浅色）

来源：https://github.com/morhetz/gruvbox — 官方配色规范

```css
[data-theme="gruvbox-light"] {
  --bg: #FBF1C7;           /* light0 */
  --card-bg: #F9F5D7;      /* light1 */
  --text-primary: #3C3836;  /* dark0 */
  --text-secondary: #7C6F64;/* dark3 */
  --text-muted: #A89984;    /* dark4 */
  --accent: #D65D0E;        /* neutral_orange — Gruvbox 标志橙 */
  --accent-rgb: 214, 93, 14;
  --accent-light: #F2E5C4;
  --accent-lighter: #F6EDD3;
  --accent-dark: #B84D00;
  --border: #D5C4A1;        /* light3 */
  --divider: #E0CFB0;
  --hover-bg: #EBDBB2;      /* light2 */
  --topbar-bg: #F9F5D7;     /* light1 */
  --input-bg: #EBDBB2;      /* light2 */
}
```

### gruvbox-dark（Gruvbox Dark — 暖大地深色）

来源：https://github.com/morhetz/gruvbox — 官方配色规范

```css
[data-theme="gruvbox-dark"] {
  --bg: #282828;            /* dark0 */
  --card-bg: #1D2021;       /* dark1 */
  --text-primary: #EBDBB2;  /* light1 */
  --text-secondary: #A89984;/* dark4 */
  --text-muted: #7C6F64;    /* dark3 */
  --accent: #FE8019;        /* bright_orange */
  --accent-rgb: 254, 128, 25;
  --accent-light: #504945;  /* dark2 */
  --accent-lighter: #3C3836;/* dark0_soft? 稍亮版 */
  --accent-dark: #FABD2F;   /* bright_yellow — 暖黄互补 */
  --border: #504945;        /* dark2 */
  --divider: #504945;
  --hover-bg: #3C3836;
  --topbar-bg: #1D2021;     /* dark1 */
  --input-bg: #3C3836;
}
```

### ayu-mirage（Ayu Mirage — 蓝灰金暖科技）

来源：https://github.com/ayu-theme — 官方配色

```css
[data-theme="ayu-mirage"] {
  --bg: #1F2430;            /* bg */
  --card-bg: #272D38;       /* sidebar bg */
  --text-primary: #CBDCF5;  /* fg */
  --text-secondary: #8A919C;/* subtext */
  --text-muted: #5C616A;    /* comment */
  --accent: #FFD580;        /* tag/黄金色 — Ayu 标志性高亮 */
  --accent-rgb: 255, 213, 128;
  --accent-light: #3D424E;  /* border */
  --accent-lighter: #333845;
  --accent-dark: #FFE6A3;
  --border: #3D424E;        /* border */
  --divider: #3D424E;
  --hover-bg: #333845;
  --topbar-bg: #272D38;     /* sidebar bg */
  --input-bg: #333845;
}
```

### dracula（Dracula — 深紫高饱和）

来源：https://draculatheme.com/spec — 官方规范

```css
[data-theme="dracula"] {
  --bg: #282A36;            /* background */
  --card-bg: #21222C;       /* ansiBlack / 更深的卡片底 */
  --text-primary: #F8F8F2;  /* foreground */
  --text-secondary: #989AB0;/* 比 foreground 稍暗 */
  --text-muted: #6272A4;    /* comment */
  --accent: #BD93F9;        /* purple — Dracula 标志紫 */
  --accent-rgb: 189, 147, 249;
  --accent-light: #44475A;  /* selection */
  --accent-lighter: #3A3C4E;
  --accent-dark: #D6ACFF;   /* bright purple */
  --border: #44475A;        /* selection */
  --divider: #44475A;
  --hover-bg: #44475A;
  --topbar-bg: #21222C;
  --input-bg: #44475A;
}
```
