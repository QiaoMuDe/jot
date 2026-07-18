# 将主题配置数据从 `main.js` 提取到独立文件的计划

## 概要

将 `main.js` 中的两个主题配置数据对象（`themeLabels`、`codeHighlightThemePairing`）提取到 `frontend/src/js/` 下的独立文件中，通过 ES module 的 export/import 供 `main.js` 使用，减少 `main.js` 体积并让主题配置集中管理。

## 现状分析

### 要提取的数据

`main.js` 第 1351-1383 行定义了两个纯数据对象：

1. **`codeHighlightThemePairing`**（第 1351-1366 行）— 系统主题 → 推荐代码高亮主题的映射
2. **`themeLabels`**（第 1368-1383 行）— 系统主题 key → 中文显示名称的映射

这两个对象都是纯 JSON 数据，不依赖任何外部变量、函数或 DOM。

### 消费方

| 消费函数                                | 位置               | 使用的数据                       | 是否有 DOM 依赖                                                 |
| ----------------------------------- | ---------------- | --------------------------- | ---------------------------------------------------------- |
| `applyTheme()`                      | main.js 第 1385 行 | `themeLabels`               | 是（`els`, `document`）                                       |
| `updateCodeHighlightThemePairing()` | main.js 第 1405 行 | `codeHighlightThemePairing` | 是（`document.getElementById`）                               |
| `buildThemeDropdown()`              | main.js 第 1431 行 | `themeLabels`               | 是（`els`, `document`, `localStorage`, `saveSettings`, `nm`） |
| `getCurrentTheme()`                 | main.js 第 1423 行 | 不使用数据对象（仅返回字符串）             | 是（`document.documentElement`）                              |

### 已有类似模式

项目中已有多个 `js/` 下的独立模块通过 named export 供 `main.js` 使用：

```js
// main.js 现有的导入模式
import { SVGS, debounce, formatTime, getSummary } from './js/constants.js';
import { applyAIHighlightTheme } from './js/hljs-themes.js';
import { codeHighlightThemeLabels, getHighlightExtension, jotTheme } from './js/cm6-syntax-highlight.js';
```

## 修改方案

### 涉及文件

| 文件                                | 操作                                                      |
| --------------------------------- | ------------------------------------------------------- |
| `frontend/src/js/theme-config.js` | **新建** — 存放 `themeLabels` 和 `codeHighlightThemePairing` |
| `frontend/src/main.js`            | **修改** — 删除原定义，改为 import                                |

### 变更详情

#### 变更 1：新建 `frontend/src/js/theme-config.js`

```js
/* ===== 主题配置数据 ===== */

/** 系统主题名称 → 中文显示标签映射 */
export const themeLabels = {
    'default': '默认',
    'catppuccin-latte': '暖咖',
    'nord': '北极',
    'gruvbox-light': '旧纸',
    'light': '浅色',
    'one-dark-pro': '暗夜',
    'quiet-light': '静谧',
    'ysgrifennwr': '暖笺',
    'tokyo-night': '夜幕',
    'eye-protection': '护眼',
    'dark': '深色',
    'dracula': '德古拉',
    'alice': '爱丽丝',
    'lightmind': '山林',
};

/** 系统主题 → 推荐代码高亮主题配对映射 */
export const codeHighlightThemePairing = {
    'catppuccin-latte': 'catppuccin-mocha',
    'gruvbox-light': 'github-dark',
    'one-dark-pro': 'one-dark-pro',
    'quiet-light': 'vscode-light-plus',
    'ysgrifennwr': 'github-light',
    'dracula': 'dracula',
    'default': 'monokai-dimmed',
    'nord': 'monokai-dimmed',
    'light': 'github-light',
    'tokyo-night': 'github-dark',
    'dark': 'github-dark',
    'eye-protection': 'github-light',
    'alice': 'github-light',
    'lightmind': 'monokai-dimmed',
};
```

#### 变更 2：修改 `frontend/src/main.js`

* 在第 5 行附近（`hljs-themes.js` 导入之后）添加导入语句：

  ```js
  import { themeLabels, codeHighlightThemePairing } from './js/theme-config.js';
  ```

* 删除第 1350-1383 行（`codeHighlightThemePairing` 和 `themeLabels` 的定义及其注释）

### 不提取的内容

* **`codeHighlightThemeLabels`** — 已在 `cm6-syntax-highlight.js` 中，无需动

* **`hljsFileMap`** — 在 `hljs-themes.js` 中，属于 AI 代码块高亮范畴，不在此次范围

* **`applyTheme()`、`updateCodeHighlightThemePairing()`** **等函数** — 有 DOM 依赖，留在 `main.js`

### 为什么可行

1. 纯数据提取，无逻辑变更，零行为影响
2. `main.js` 已是 ES module，import 语法与现有导入一致
3. 新增文件遵循 `js/` 目录下的命名和导出惯例
4. 数据对象的值完全不变，消费方无需任何修改

## 验证步骤

1. 重新构建/运行应用
2. 检查默认主题下拉菜单显示是否正确（`themeLabels` 是否正常加载）
3. 切换主题，检查代码高亮主题推荐配对标记（✦）是否正常显示
4. 检查浏览器控制台无 import 相关错误

