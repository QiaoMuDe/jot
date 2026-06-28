# 新增 VS Code 常用代码高亮主题

## Summary

在已有的 `codeHighlightThemes` 系统中新增 4 个 VS Code 风格的代码高亮主题（Dark+/Light+/One Dark Pro/GitHub Dark），使设置页的代码高亮主题选择器从单个「默认 Monokai」扩展到 5 个选项。

## Current State

`frontend/src/js/cm6-syntax-highlight.js` 当前仅有一个 `monokai-dimmed` 主题存储在 `codeHighlightThemes` 中，`codeHighlightThemeNames` 和 `codeHighlightThemeLabels` 各只有一项。设置页分段控件硬编码了一个按钮。

## Proposed Changes

### 1. `frontend/src/js/cm6-syntax-highlight.js` — 新增 4 个主题

在 `codeHighlightStyle` 定义之后（第 238 行附近），新增 4 个 `HighlightStyle.define([...])` 常量：

**命名约定**：每个主题一个 const，`xxxHighlightStyle`。

**公共设计原则**（所有主题一致）：
- 标点/括号/分隔符：使用 `var(--text-secondary)` / `var(--text-muted)` — 跟随应用主题
- 变量名/命名空间：使用 `var(--text-primary)` / `var(--text-muted)` — 跟随应用主题
- 行内代码/删除线：使用 `var(--hover-bg)` / 纯样式 — 跟随应用主题
- 语义着色（关键字/字符串/类型/函数名等）：硬编码色值 — 跨主题一致性

#### 主题 A：`vscode-dark-plus` — VS Code 默认深色 + 主题

| Tag | 色值 | 说明 |
|-----|------|------|
| keyword / controlKeyword / moduleKeyword | `#569CD6` | 经典 VS Code 蓝 |
| [keyword, modifier] | `#569CD6` | |
| typeName / className | `#4EC9B0` | 青色 |
| definition(variableName) [function] | `#DCDCAA` | 浅黄 |
| variableName / definition(variableName) [var] | `var(--text-primary)` | |
| propertyName | `#9CDCFE` | 浅蓝 |
| number | `#B5CEA8` | 浅绿 |
| string / special(string) | `#CE9178` | 橙褐色 |
| regexp | `#D16969` bg:`rgba(209,105,105,0.1)` | 红色 |
| atom | `#569CD6` | 同关键字 |
| comment / docComment | `#6A9955` italic | 墨绿 |
| operator / arithmetic / logic / compare | `#D4D4D4` | 白色 |
| attributeName | `#9CDCFE` | 同 property |
| attributeValue | `#CE9178` | 同 string |
| tagName | `#569CD6` | 同关键字 |
| meta | `#DCDCAA` | 同函数 |

#### 主题 B：`vscode-light-plus` — VS Code 默认浅色 + 主题

| Tag | 色值 | 说明 |
|-----|------|------|
| keyword / controlKeyword / moduleKeyword | `#0000FF` | 经典蓝 |
| [keyword, modifier] | `#0000FF` | |
| typeName / className | `#267F99` | 深青 |
| definition(variableName) [function] | `#795E26` | 棕褐色 |
| variableName / definition(variableName) [var] | `var(--text-primary)` | |
| propertyName | `#001080` | 深蓝 |
| number | `#098658` | 深绿 |
| string / special(string) | `#A31515` | 暗红 |
| regexp | `#811F3F` bg:`rgba(129,31,63,0.1)` | |
| atom | `#0000FF` | |
| comment / docComment | `#008000` italic | 绿 |
| operator / arithmetic / logic / compare | `var(--text-primary)` | |
| attributeName | `#001080` | |
| attributeValue | `#A31515` | |
| tagName | `#0000FF` | |
| meta | `#795E26` | |

#### 主题 C：`one-dark-pro` — Atom One Dark Pro

| Tag | 色值 | 说明 |
|-----|------|------|
| keyword / controlKeyword / moduleKeyword | `#C678DD` | 紫 |
| [keyword, modifier] | `#C678DD` | |
| typeName / className | `#E5C07B` | 金黄 |
| definition(variableName) [function] | `#61AFEF` | 亮蓝 |
| variableName / definition(variableName) [var] | `var(--text-primary)` | |
| propertyName | `#E06C75` | 粉红 |
| number | `#D19A66` | 橙 |
| string / special(string) | `#98C379` | 绿 |
| regexp | `#98C379` bg:`rgba(152,195,121,0.1)` | |
| atom | `#C678DD` | 同关键字 |
| comment / docComment | `#5C6370` italic | 灰 |
| operator / arithmetic / logic / compare | `#ABB2BF` | 浅灰 |
| attributeName | `#E06C75` | |
| attributeValue | `#98C379` | |
| tagName | `#E06C75` | |
| meta | `#ABB2BF` | |

#### 主题 D：`github-dark` — GitHub Dark

| Tag | 色值 | 说明 |
|-----|------|------|
| keyword / controlKeyword / moduleKeyword | `#FF7B72` | 珊瑚红 |
| [keyword, modifier] | `#FF7B72` | |
| typeName / className | `#FFA657` | 橙 |
| definition(variableName) [function] | `#D2A8FF` | 浅紫 |
| variableName / definition(variableName) [var] | `var(--text-primary)` | |
| propertyName | `#79C0FF` | 蓝 |
| number | `#79C0FF` | 同蓝 |
| string / special(string) | `#A5D6FF` | 浅蓝 |
| regexp | `#A5D6FF` bg:`rgba(165,214,255,0.1)` | |
| atom | `#FF7B72` | |
| comment / docComment | `#8B949E` italic | 灰 |
| operator / arithmetic / logic / compare | `#FF7B72` | 同关键字 |
| attributeName | `#FFA657` | |
| attributeValue | `#A5D6FF` | |
| tagName | `#7EE787` | 绿 |
| meta | `#FF7B72` | |

#### 注册到主题系统

在 `codeHighlightThemes` 对象中新增 4 项：

```js
const codeHighlightThemes = {
    'monokai-dimmed': codeHighlightStyle,
    'vscode-dark-plus': vscodeDarkPlusHighlightStyle,
    'vscode-light-plus': vscodeLightPlusHighlightStyle,
    'one-dark-pro': oneDarkProHighlightStyle,
    'github-dark': githubDarkHighlightStyle,
};
```

更新 `codeHighlightThemeNames` 和 `codeHighlightThemeLabels`：

```js
const codeHighlightThemeNames = Object.freeze([
    'monokai-dimmed',
    'vscode-dark-plus',
    'vscode-light-plus',
    'one-dark-pro',
    'github-dark',
]);

const codeHighlightThemeLabels = Object.freeze({
    'monokai-dimmed': '默认 Monokai',
    'vscode-dark-plus': 'Dark+',
    'vscode-light-plus': 'Light+',
    'one-dark-pro': 'One Dark Pro',
    'github-dark': 'GitHub Dark',
});
```

### 2. `frontend/index.html` — 更新分段控件

在 `#codeHighlightThemeControl` 中新增 4 个按钮：

```html
<div class="segmented-control" id="codeHighlightThemeControl">
    <div class="segmented-indicator" id="codeHighlightThemeIndicator"></div>
    <button class="segmented-btn active" data-theme-value="monokai-dimmed">默认 Monokai</button>
    <button class="segmented-btn" data-theme-value="vscode-dark-plus">Dark+</button>
    <button class="segmented-btn" data-theme-value="vscode-light-plus">Light+</button>
    <button class="segmented-btn" data-theme-value="one-dark-pro">One Dark Pro</button>
    <button class="segmented-btn" data-theme-value="github-dark">GitHub Dark</button>
</div>
```

### 3. 无需修改 `main.js`

`loadCodeHighlightThemeSetting()` / `saveCodeHighlightThemeSetting()` / `applyCodeHighlightTheme()` / `initCodeHighlightThemeSettings()` 均已通用化，新主题通过 `codeHighlightThemes[themeName]` 自动适配，无需改动。

## Assumptions & Decisions

1. **颜色取值依据**：各主题色值参考对应 VS Code 主题的 tokenColors 定义，适配 CM6 `@lezer/highlight` tag 体系。由于 CM6 tag 与 TextMate scope 并非一一对应（CM6 tag 粒度更粗），色值选取为近似等效。
2. **Light+ 主题的深色/浅色适配**：Light+ 的标点/变量等使用 `var(--text-primary)`，在深色应用主题下仍可读，但在浅色应用主题下表现最佳。
3. **无需改动 main.js**：现有代码已通用化，新主题自动适配。
4. **扩展性**：后续新增主题只需：① 在 `cm6-syntax-highlight.js` 定义 `xxxHighlightStyle` 并注册到 3 处（themes/names/labels），② 在 `index.html` 加一个按钮。

## Verification

1. `npx vite build` 通过，零错误、零诊断
2. 设置页「代码高亮主题」显示 5 个选项
3. 逐一切换 5 个主题，验证 `.js` 笔记的关键字/字符串/注释等颜色变化正确
4. `.md` 笔记 MD 语法高亮不受影响
5. 切换应用主题（深色/浅色/Nord 等），代码高亮颜色不受应用主题影响（语义色硬编码）
6. 重启后设置持久化