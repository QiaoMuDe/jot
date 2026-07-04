# 移除四个主题计划

## 概述

从设置选项和主题系统中移除以下 4 个主题：**暖夜** (`catppuccin-mocha`)、**暮光** (`ayu-mirage`)、**粉墨** (`monokai-pro`)、**陈酿** (`gruvbox-dark`)。

移除后保留 8 个主题：默认、浅色、深色、北极、夜幕、暖咖、旧纸、德古拉。

## 当前状态分析

主题系统基于 `[data-theme="..."]` CSS 属性选择器 + CSS 自定义属性实现。四个主题分布在以下文件和位置：

| 主题 | key              | variables.css | main.js labels | main.js pairing | index.html 选择器 | index.html 防闪色 |
| -- | ---------------- | ------------- | -------------- | --------------- | -------------- | -------------- |
| 粉墨 | monokai-pro      | L286-341      | L1311          | L1298           | L316           | L20            |
| 陈酿 | gruvbox-dark     | L553-602      | L1310          | L1292           | L315           | L25            |
| 暮光 | ayu-mirage       | L604-653      | L1312          | L1294           | L317           | L26            |
| 暖夜 | catppuccin-mocha | L451-500      | L1314          | L1291           | L319           | L23            |

## 修改内容

### 1. `frontend/src/css/variables.css`

**操作**：删除 4 个 `[data-theme="..."]` CSS 变量块。

* **粉墨**：删除 L286-341 (`[data-theme="monokai-pro"]`)

* **暖夜**：删除 L451-500 (`[data-theme="catppuccin-mocha"]`)

* **陈酿**：删除 L553-602 (`[data-theme="gruvbox-dark"]`)

* **暮光**：删除 L604-653 (`[data-theme="ayu-mirage"]`)

### 2. `frontend/src/main.js`

**操作**：从 `themeLabels` 对象和 `codeHighlightThemePairing` 对象中删除对应条目。

**`themeLabels`** (L1304-1317)：

```
删除：'gruvbox-dark': '陈酿'
删除：'monokai-pro': '粉墨'
删除：'ayu-mirage': '暮光'
删除：'catppuccin-mocha': '暖夜'
```

**`codeHighlightThemePairing`** (L1289-1302)：

```
删除：'catppuccin-mocha': 'catppuccin-mocha'
删除：'gruvbox-dark': 'gruvbox-dark'
删除：'gruvbox-light': 'gruvbox-dark'   ← 注意：这是 gruvbox-light 的配对指向陈酿，需要改为其他
删除：'monokai-pro': 'monokai-dimmed'
```

**注意**：`gruvbox-light`（旧纸）的配对原为 `gruvbox-dark`（陈酿），陈酿删除后需要将旧纸的配对改为其他主题（如 `'monokai-dimmed'` 或 `'github-dark'`）。

### 3. `frontend/index.html`

**操作**：从两处移除对应条目。

**内联脚本关键色** (L7-34 的 `criticalColors` 对象)：

```
删除：'monokai-pro': ['#2D2A2E','#221F22'],
删除：'catppuccin-mocha': ['#1E1E2E','#181825'],
删除：'gruvbox-dark': ['#282828','#1D2021'],
删除：'ayu-mirage': ['#1F2430','#272D38'],
```

**主题选择下拉菜单** (L309-322)：

```
删除：<div class="theme-select-item" data-theme-value="gruvbox-dark">陈酿</div>
删除：<div class="theme-select-item" data-theme-value="monokai-pro">粉墨</div>
删除：<div class="theme-select-item" data-theme-value="ayu-mirage">暮光</div>
删除：<div class="theme-select-item" data-theme-value="catppuccin-mocha">暖夜</div>
```

## 决策说明

1. **`gruvbox-light`** **的代码高亮配对**：原配 `gruvbox-dark`（陈酿），陈酿删除后将默认改为 `github-dark`，与其他明亮主题保持一致。
2. **数据库已有数据的处理**：如果用户之前选择了被删除的主题（如 `catppuccin-mocha`），`localStorage` 和后端 DB 中的 `theme` 值会保持旧值。`applyTheme` 仍会设置 `data-theme` 属性，只是不会有对应的 CSS 变量——页面会降级使用 `:root` 默认变量（暖色风格），不会崩溃。用户重新选择即可。这是可接受的边界情况。

## 验证步骤

1. 打开设置页，确认主题下拉菜单中已无四个被删主题
2. 逐一测试剩余 8 个主题，确认切换正常
3. 确认代码高亮主题推荐配对仍正常工作
4. 确认页面加载时无 CSS/JS 控制台错误

