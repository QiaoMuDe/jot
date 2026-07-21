# 优化代码高亮主题推荐配对方案

## Summary

系统主题配色已全面重构，但 `codeHighlightThemePairing` 映射仍基于旧配色推荐，部分推荐不再与主题气质匹配。需要更新 3 个条目的推荐值，使其与新配色方案呼应。

## Current State Analysis

### 系统架构

- **映射定义**：[theme-config.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/theme-config.js#L22-L37) 中的 `codeHighlightThemePairing` 对象定义 14 个系统主题 → 推荐代码高亮主题的映射
- **推荐展示**：[main.js#L1374-L1387](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L1374-L1387) 的 `updateCodeHighlightThemePairing()` 在下拉菜单中给推荐项加 `✦` 前缀标记
- **应用时机**：[main.js#L1366](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L1366) 每次 `applyTheme()` 调用时更新配对标记
- **不自动切换**：配对仅用于"推荐标记"，用户实际使用的代码高亮主题是单独保存的设置（[main.js#L8095](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L8095)），不受配对影响

### 当前配对的问题

| 系统主题（新配色情绪） | 当前推荐 | 问题 |
|---|---|---|
| **nord**（极光黎明 `#E4E9F0` 冷蓝灰底） | `monokai-dimmed`（暖黄色调） | ❌ 冷北极底 + 暖黄色代码，视觉温度冲突 |
| **light**（雪光 `#F4F4F7` 冷白底） | `github-light`（暖色调） | ❌ 冷雪光底 + 暖代码，色温不协调 |
| **quiet-light**（灰玫瑰 `#F0EAEC` 粉紫调底） | `vscode-light-plus`（中性） | ❌ 粉紫底 + 中性代码，未呼应主题个性 |

其余 11 个主题的配对在当前新配色下仍然合适。

## Proposed Changes

### 变更 1：nord → `github-dark`

- **理由**：nord 新底色 `#E4E9F0` 是冷调极地蓝灰，`github-dark` 以冷蓝为基调，与之和谐共鸣
- **对比**：old `monokai-dimmed` 的暖黄橙色与冷灰底冲突

### 变更 2：light → `vscode-light-plus`

- **理由**：light 新底色 `#F4F4F7` 是冷雪光白，`vscode-light-plus` 的冷蓝代码配色（`#0000FF` 关键字、`#267F99` 类型）与之协调
- **对比**：old `github-light` 的暖红关键字（`#D73A49`）与冷白底不搭

### 变更 3：quiet-light → `material-palenight`

- **理由**：quiet-light 新底色 `#F0EAEC` 带有明确粉紫调，`material-palenight` 以紫色为主调的代码配色（`#C792EA` 关键字）与之呼应
- **对比**：old `vscode-light-plus` 的中性蓝调无法体现粉紫性格

### 不变的主题（附简要理由）

| 主题 | 保持推荐 | 理由 |
|------|---------|------|
| default | `monokai-dimmed` | 暖纸本 + 暖 monokai，气质一致 |
| catppuccin-latte | `catppuccin-mocha` | 品牌一致 |
| gruvbox-light | `github-dark` | 暖旧纸 + 暖暗色代码，有对比但也有温度 |
| one-dark-pro | `one-dark-pro` | 品牌一致 |
| ysgrifennwr | `github-light` | 暖羊皮纸 + 暖代码 |
| tokyo-night | `github-dark` | 深靛蓝 + 蓝调暗色代码 |
| eye-protection | `github-light` | 柔和米绿 + 整洁浅代码 |
| dark | `github-dark` | 柔黑 + GitHub Dark |
| dracula | `dracula` | 品牌一致 |
| alice | `github-light` | 暖米 + 暖代码 |
| lightmind | `monokai-dimmed` | 竹纸绿 + 暖 monokai |

## Assumptions & Decisions

- **仅改推荐标记**：不改变"配对只是推荐标记，不自动切换用户设置"的现有机制
- **不新增代码高亮主题**：仅调整映射关系，所有可选的代码高亮主题保持不变
- **不涉及 `hljsFileMap`**：hljs 代码块高亮跟随用户实际选择的代码高亮主题，不受此影响
- **不涉及 `isDarkTheme`**：Mermaid 明暗判断已随系统主题配色正常，无需修改

## Verification

1. 确认 `theme-config.js` 中 `codeHighlightThemePairing` 的 3 个条目已更新
2. 运行 `npx vite build` 确保构建通过
3. 视觉确认：切换 nord / light / quiet-light 主题时，下拉菜单的 ✦ 标记指向新的推荐主题
