# AI 消息代码高亮跟随设置页主题 Spec

## Why

当前 AI 消息代码块使用 highlight.js 且主题固定为 `github.css`（`main.js` 第 5 行静态导入），不跟随设置页 `code_highlight_theme` 设置。用户切换编辑器代码高亮主题后，AI 消息的代码块高亮不变，视觉不一致。

## What Changes

- **移除静态 hljs CSS import**: 去掉 `main.js` 第 5 行 `import 'highlight.js/styles/github.css'`
- **新建 `hljs-themes.js`**：封装 hljs 主题 CSS 的动态加载机制，将 11 套 CM6 主题名映射到对应的 highlight.js 主题
- **新建 `applyAIHighlightTheme()`**：注入/替换 `<style id="ai-hljs-theme">` 标签，动态切换 AI 聊天的代码高亮
- **联动设置**: 在 `loadSettings()` 和 `applyCodeHighlightTheme()` 中同步调用 `applyAIHighlightTheme()`
- **联动渲染**: `renderMarkdown()` 中高亮代码块后，确保已应用当前 hljs 主题 CSS

## Impact

- Affected specs: 代码高亮主题设置 → 扩展覆盖范围到 AI 聊天代码块
- Affected code:
  - `frontend/src/main.js` — 移除静态 import，在设置加载/切换时同步 AI 主题
  - `frontend/src/js/hljs-themes.js` — **新建**，hljs 主题 CSS 加载和注入
  - `frontend/src/js/ai-chat.js` — 引入 `applyAIHighlightTheme()` 联动

## ADDED Requirements

### Requirement: AI 消息代码高亮跟随设置

The system SHALL apply the same code highlight theme selected in settings to AI chat code blocks.

#### Scenario: 应用启动时同步
- **WHEN** 应用启动，`loadSettings()` 加载 `code_highlight_theme`
- **THEN** AI 消息代码块使用与编辑器相同的高亮主题

#### Scenario: 切换主题时同步
- **WHEN** 用户在设置页切换代码高亮主题
- **THEN** AI 聊天中已渲染和后续新渲染的代码块都使用新主题高亮

#### Scenario: 新消息渲染
- **WHEN** AI 流式输出完成后渲染代码块
- **THEN** 代码块使用当前 `code_highlight_theme` 设置的高亮主题

### Requirement: CM6 主题→hljs 主题映射

系统 SHALL 维护一份 CM6 主题名到 highlight.js CSS 文件名的映射表，确保切换后 AI 代码块配色与编辑器的代码高亮主题视觉接近。

| CM6 主题 | hljs CSS 文件 |
|-----------|---------------|
| monokai-dimmed | monokai-sublime |
| vscode-dark-plus | atom-one-dark-reasonable |
| vscode-light-plus | atom-one-light |
| one-dark-pro | atom-one-dark |
| github-dark | github-dark |
| catppuccin-mocha | monokai-sublime |
| gruvbox-dark | gruvbox-dark |
| dracula | dracula |
| ayu-mirage | monokai-sublime |
| material-palenight | atom-one-dark |
| github-light | github |
