# 新增 4 个代码高亮主题 Spec

## Why

现有 11 个代码高亮主题中，4 个系统主题（tokyo-night、nord、catppuccin-latte、default）没有专属配对的高亮主题，且亮色主题仅 2 个（vs 9 暗色），亮色主题缺口明显。

## What Changes

- **cm6-syntax-highlight.js** — 新增 4 个 HighlightStyle 定义（Tokyo Night、Nord、One Light、Catppuccin Latte），注册到 `codeHighlightThemes` / `codeHighlightThemeNames` / `codeHighlightThemeLabels`
- **theme-config.js** — 更新 `codeHighlightThemePairing` 配对映射：tokyo-night → tokyo-night、nord → nord、default → one-light、catppuccin-latte → catppuccin-latte
- **hljs-themes.js** — 新增 4 条 `hljsFileMap` 映射

## Impact

- Affected specs: 代码高亮主题配置、系统主题配对
- Affected code:
  - `frontend/src/js/cm6-syntax-highlight.js` — HighlightStyle 定义 + 注册
  - `frontend/src/js/theme-config.js` — 配对映射更新
  - `frontend/src/js/hljs-themes.js` — hljs 回退映射

## ADDED Requirements

### Requirement: 新代码高亮主题定义

系统 SHALL 新增 4 套代码语法高亮主题。

#### Scenario: Tokyo Night（暗色）

- **GIVEN** `cm6-syntax-highlight.js` 中定义了 `tokyoNightHighlightStyle`
- **WHEN** 用户在设置页选择「Tokyo Night」
- **THEN** 编辑器代码高亮呈现深靛蓝底 + 蓝/青/粉高亮配色
- **AND** markdown 语法不受影响

#### Scenario: Nord（暗色）

- **GIVEN** `cm6-syntax-highlight.js` 中定义了 `nordHighlightStyle`
- **WHEN** 用户在设置页选择「Nord」
- **THEN** 编辑器代码高亮呈现冷灰底 + 蓝/青/绿高亮配色

#### Scenario: One Light（亮色）

- **GIVEN** `cm6-syntax-highlight.js` 中定义了 `oneLightHighlightStyle`
- **WHEN** 用户在设置页选择「One Light」
- **THEN** 编辑器代码高亮呈现白底 + 紫/蓝/绿/橙高亮配色

#### Scenario: Catppuccin Latte（亮色）

- **GIVEN** `cm6-syntax-highlight.js` 中定义了 `catppuccinLatteHighlightStyle`
- **WHEN** 用户在设置页选择「Catppuccin Latte」
- **THEN** 编辑器代码高亮呈现暖粉白底 + 粉/紫/绿/橙高亮配色

### Requirement: 系统主题配对更新

系统 SHALL 更新 `codeHighlightThemePairing` 配对映射。

#### Scenario: 配对一致性

- **WHEN** 用户切换到 `tokyo-night` 系统主题
- **THEN** 自动推荐 / 默认配对为 `tokyo-night` 代码高亮主题（不再是 `github-dark`）
- **AND** nord → nord、default → one-light、catppuccin-latte → catppuccin-latte

#### Scenario: 向后兼容

- **GIVEN** 已有配对映射（如 dracula、one-dark-pro 等）
- **WHEN** 更新配对表
- **THEN** 已存在的配对不受影响

## 新增主题配色方案

### Tokyo Night（深靛蓝暗色）
| 节点 | 色值 | 说明 |
|------|------|------|
| keyword | `#BB9AF7` | 靛紫 |
| typeName/className | `#2AC3DE` | 青蓝 |
| function | `#7AA2F7` | 中蓝 |
| variableName | `#C0CAF5` | 浅蓝白 |
| propertyName | `#73DACA` | 青绿 |
| number | `#FF9E64` | 橙 |
| string | `#9ECE6A` | 草绿 |
| operator | `#89DDFF` | 浅青 |
| comment | `#565F89` italic | 灰紫 |
| tagName | `#2AC3DE` | 青蓝 |

### Nord（冷蓝灰暗色）
| 节点 | 色值 | 说明 |
|------|------|------|
| keyword | `#81A1C1` | 冷蓝 |
| typeName/className | `#8FBCBB` | 青绿 |
| function | `#88C0D0` | 天蓝 |
| variableName | `#D8DEE9` | 冷白 |
| propertyName | `#8FBCBB` | 青绿 |
| number | `#B48EAD` | 紫粉 |
| string | `#A3BE8C` | 草绿 |
| operator | `#81A1C1` | 冷蓝 |
| comment | `#616E88` italic | 灰蓝 |
| tagName | `#81A1C1` | 冷蓝 |

### One Light（暖白亮色）
| 节点 | 色值 | 说明 |
|------|------|------|
| keyword | `#A626A4` | 紫 |
| typeName/className | `#E45649` | 红橙 |
| function | `#4078F2` | 蓝 |
| variableName | `#383A42` | 深灰 |
| propertyName | `#E45649` | 红橙 |
| number | `#986801` | 褐金 |
| string | `#50A14F` | 绿 |
| operator | `#0184BC` | 蓝 |
| comment | `#A0A1A7` italic | 灰 |
| tagName | `#E45649` | 红橙 |

### Catppuccin Latte（暖粉彩亮色）
| 节点 | 色值 | 说明 |
|------|------|------|
| keyword | `#8839EF` | 紫 |
| typeName/className | `#EA76CB` | 粉 |
| function | `#1E66F5` | 蓝 |
| variableName | `#4C4F69` | 深灰 |
| propertyName | `#EA76CB` | 粉 |
| number | `#FE640B` | 橙 |
| string | `#40A02B` | 绿 |
| operator | `#04A5E5` | 青蓝 |
| comment | `#8C8FA7` italic | 灰紫 |
| tagName | `#1E66F5` | 蓝 |
