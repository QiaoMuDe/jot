# 代码高亮主题菜单中文化

## 现状分析

`codeHighlightThemeLabels` 对象定义在 `cm6-syntax-highlight.js:1372`，当前全英文显示：

```js
const codeHighlightThemeLabels = Object.freeze({
    'monokai-dimmed': 'Monokai',
    'vscode-dark-plus': 'Dark+',
    'vscode-light-plus': 'Light+',
    'one-dark-pro': 'One Dark Pro',
    'github-dark': 'GitHub Dark',
    'catppuccin-mocha': 'Catppuccin',
    'gruvbox-dark': 'Gruvbox',
    'dracula': 'Dracula',
    'ayu-mirage': 'Ayu Mirage',
    'material-palenight': 'Material',
    'github-light': 'GitHub Light',
    'one-light': 'One Light',
    'catppuccin-latte': 'Catppuccin Latte',
});
```

## 使用链路

所有显示入口都通过 `codeHighlightThemeLabels[key]` 取值，key（如 `monokai-dimmed`）用于存储和加载：

- **下拉菜单项** — `main.js:1466` `item.textContent = label;`
- **触发器标签** — `main.js:7927` `label.textContent = codeHighlightThemeLabels[themeName] || themeName;`
- **后端存储** — `main.js:8307` `code_highlight_theme: codeHighlightTheme`（存 key）
- **加载恢复** — `main.js:8129` `cfg.code_highlight_theme || 'monokai-dimmed'`（读 key）

Key 用于持久化和后端，仅 Value 用于 UI 显示。

## 修改方案

只需修改一个文件：

| 文件 | 改动 |
|------|------|
| `cm6-syntax-highlight.js` | 将 `codeHighlightThemeLabels` 的 value 替换为中文名 |

### 中英文对照表

| key | 当前英文 | 中文名 |
|-----|---------|--------|
| `monokai-dimmed` | Monokai | Monokai 暗色 |
| `vscode-dark-plus` | Dark+ | VS Code 暗色+ |
| `vscode-light-plus` | Light+ | VS Code 亮色+ |
| `one-dark-pro` | One Dark Pro | One Dark Pro |
| `github-dark` | GitHub Dark | GitHub 暗色 |
| `catppuccin-mocha` | Catppuccin | Catppuccin 摩卡 |
| `gruvbox-dark` | Gruvbox | Gruvbox 暗色 |
| `dracula` | Dracula | 德古拉 |
| `ayu-mirage` | Ayu Mirage | Ayu 幻境 |
| `material-palenight` | Material | Material 暗夜 |
| `github-light` | GitHub Light | GitHub 亮色 |
| `one-light` | One Light | One Light |
| `catppuccin-latte` | Catppuccin Latte | Catppuccin 拿铁 |

### 保持不变的逻辑

- `codeHighlightThemes` 对象（key → style 映射）
- `codeHighlightThemeNames` 数组
- `theme-config.js` 中的 `codeHighlightThemePairing` 配对映射
- `hljs-themes.js` 中的 `hljsFileMap`
- 后端存储/加载字段

## 验证

- `npx vite build` 通过
- 设置页代码高亮主题下拉菜单项显示中文名
- 触发器按钮标签显示中文名
- 保存/加载功能正常（key 不变）
