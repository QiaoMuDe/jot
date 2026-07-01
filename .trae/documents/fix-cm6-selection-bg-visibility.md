# 修复 CM6 选中文字背景色可见性

## 问题

CM6 编辑器选中文字的背景色使用 `--accent-light` CSS 变量。在大部分主题中，`--accent-light` 与卡片背景色 `--card-bg` 过于接近，导致选中文字几乎看不出变化：

| 主题 | `--accent-light` | `--card-bg` | 问题 |
|------|-----------------|-------------|------|
| tokyo-night | `#1F2A4A` | `#24283B` | ❌ 深蓝融于深色背景 |
| catppuccin-latte | `#E6E9EF` | `#FFFFFF` | ❌ 近白色融于白色背景 |
| catppuccin-mocha | `#363A4F` | `#181825` | ❌ 深灰融于深色背景 |
| ayu-mirage | `#3D424E` | `#272D38` | ❌ 灰色融于深色背景 |
| dracula | `#44475A` | `#21222C` | ❌ 灰色融于深色背景 |
| dark | `#92400E` | `#1A1A1A` | ❌ 深褐融于深色背景 |
| gruvbox-light | `#F2E5C4` | `#F9F5D7` | ⚠️ 偏淡 |
| gruvbox-dark | `#504945` | `#1D2021` | ⚠️ 偏暗 |

## 方案

新增 CSS 变量 `--selection-bg`，每个主题独立定义其值。利用各主题已有的 `--accent-rgb` 变量，用 `rgba()` 生成半透明色，使选中背景成为**主题强调色的半透明覆盖层**：

```
rgba(var(--accent-rgb), 0.28)
```

原理：
- **Light 主题**（白底深色强调色）：28% 的强调色叠加在白色上 = 可见的柔和色块
- **Dark 主题**（暗底亮色强调色）：28% 的强调色叠加在深色上 = 发光感的选中效果
- 每个主题自动适配，视觉权重一致
- 无需逐个主题调色，维护成本低

## 修改文件

### 1. `frontend/src/css/variables.css`

在 `:root` 和各 `[data-theme="..."]` 块中添加 `--selection-bg` 变量。

对于 `--accent-light` 已经 OK 的主题（default, light, nord, monokai-pro）：
```
--selection-bg: rgba(var(--accent-rgb), 0.28);
```

对于 `--accent-light` 偏弱的主题（dark, tokyo-night, catppuccin-latte, catppuccin-mocha, gruvbox-light, gruvbox-dark, ayu-mirage, dracula），使用稍高透明度以确保可见性，并补一个显式色值作为兜底：
```
--selection-bg: rgba(var(--accent-rgb), 0.35);
```

最终每个主题的值见下表：

| Theme | accent-rgb | opacity | selection-bg (计算效果) |
|-------|-----------|---------|----------------------|
| default | 217,119,6 | 0.28 | 暖黄半透明层 |
| light | 37,99,235 | 0.28 | 淡蓝半透明层 |
| dark | 245,158,11 | 0.35 | 金色半透明层 |
| nord | 94,129,172 | 0.28 | 淡蓝半透明层 |
| monokai-pro | 255,97,136 | 0.28 | 淡粉半透明层 |
| tokyo-night | 122,162,247 | 0.35 | 亮蓝半透明层 |
| catppuccin-latte | 221,120,120 | 0.35 | 淡红半透明层 |
| catppuccin-mocha | 243,139,168 | 0.35 | 粉红半透明层 |
| gruvbox-light | 214,93,14 | 0.35 | 橙色半透明层 |
| gruvbox-dark | 254,128,25 | 0.35 | 橙色半透明层 |
| ayu-mirage | 255,213,128 | 0.35 | 淡黄半透明层 |
| dracula | 189,147,249 | 0.35 | 紫色半透明层 |

### 2. `frontend/src/js/cm6-syntax-highlight.js`

将 CM6 选中背景色从：
```javascript
backgroundColor: 'var(--accent-light) !important',
```
改为：
```javascript
backgroundColor: 'var(--selection-bg, var(--accent-light)) !important',
```

`var(--selection-bg, var(--accent-light))` 表示优先使用 `--selection-bg`，未定义时回退到 `--accent-light`。

## 影响范围

- 仅 CM6 编辑器的选中背景色（`.cm-selectionBackground`）受影响
- `--accent-light` 在其他 UI 元素（下拉菜单、搜索弹窗、卡片 hover 等）的使用不受影响
- 修改极小：1 行 CM6 代码 + 12 行 CSS 变量定义

## 验证

1. 切换每个主题，选中 CM6 编辑器中的文本，确保选中背景色清晰可见
2. 确保默认/静谧蓝/nord/monokai-pro 主题的选中背景色与原效果接近（无退化）
3. 确保文字在选中状态下仍然可读
