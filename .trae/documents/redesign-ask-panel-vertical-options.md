# 反问面板选项垂直列表改造 Plan

## 概要

将 Agent 反问面板（`#aiAskPanel`）的选项布局从「小椭圆 chips 横排换行」改为 **Trae 风格垂直列表：一行一个选项**；底部保留「输入框 + 唯一按钮」同排行（单选「发送」/ 多选「确认提交」）。

交互规则（用户已确认）：
- **单选**：点击选项行即发送该答案（`sendUserText(opt)`）+ 隐藏面板
- **多选**：点击选项行切换勾选（整行高亮 + 右侧对勾），再点「确认提交」汇总发送
- **选中态视觉**：整行高亮（accent 淡底 + accent 边框）+ 右侧对勾

## 现状分析

- 容器与结构：[index.html](frontend/index.html#L1151) `#aiAskPanel`（`display:none`，JS 动态填充），无需改动。
- JS：[ai-chat.js](frontend/src/js/ai-chat.js#L4105-L4209) `showAskPanel`——多选分支每个选项行已含 `check` span（`✓`）+ 文本节点；单选分支仅文本。**JS 无需改动**：多选行 DOM 顺序为 check 在前，可纯 CSS 用 `margin-left:auto` 将 check 推到行右侧。
- CSS（[ai-chat.css](frontend/src/css/components/ai-chat.css)）：
  - `.ai-ask-options`（L4083）：`display:flex; flex-wrap:wrap; gap:8px`——椭圆卡片横排换行的根源。
  - `.ai-ask-option`（L4090）：`padding:6px 12px`、`border-radius:8px` 的 chip。
  - `.ai-ask-option.selected`（L4107）：accent 实底白字。
  - `.ai-ask-check`（L4113）：`margin-right:4px`（左侧对勾）。
  - `.ai-ask-input-row` / `.ai-ask-submit`：底部输入行（用户满意，**保留不变**）。

## 变更方案

### 1. CSS：选项区改为垂直列表（核心改动，[ai-chat.css](frontend/src/css/components/ai-chat.css#L4083-L4115)）

**`.ai-ask-options`**：横排换行 → 垂直堆叠，一行一个。

```css
.ai-ask-options {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 10px; /* 保留：与输入行间距 */
}
```

**`.ai-ask-option`**：chip → 整行列表项。

```css
.ai-ask-option {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 8px 12px;
    font-size: 0.82rem;
    font-family: inherit;
    text-align: left;
    color: var(--text-primary);
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
}
```

**`.ai-ask-option:hover`**：行 hover 背景微亮（Trae 风格，替代原文字变色）。

```css
.ai-ask-option:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    border-color: var(--accent);
}
```

**`.ai-ask-option.selected`**：整行高亮（淡 accent 底 + accent 边框），文本保持 `--text-primary` 保证 14 主题可读。

```css
.ai-ask-option.selected {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    border-color: var(--accent);
}
```

**`.ai-ask-check`**：左侧对勾 → 右侧对勾；未选中隐藏、选中淡入（`opacity` 过渡，属微交互，无需额外 reduced-motion 处理）。

```css
.ai-ask-check {
    margin-left: auto;   /* 推到行右侧 */
    margin-right: 0;
    color: var(--accent);
    opacity: 0;          /* 未选中隐藏 */
    transition: opacity 0.15s;
}

.ai-ask-option.selected .ai-ask-check {
    opacity: 1;          /* 选中淡入 */
}
```

> 说明：`.ai-ask-option.selected` 现在对单选/多选共用——多选勾选态即此样式；单选点击即发（selected 一闪而过 + 立即隐藏面板），无冲突。

### 2. JS：无需改动（[ai-chat.js](frontend/src/js/ai-chat.js#L4105-L4209)）

- 多选分支：`check` span + 文本节点结构已满足「文本居左、对勾居右」；勾选/取消逻辑（`Set` + `classList.toggle('selected')`）不变。
- 单选分支：点击即发 + 临时 selected 反馈不变。
- 输入行、发送/确认提交逻辑、拼接行为均不变。

### 3. 文档同步

- [internal/agent/TOOLS.md](internal/agent/TOOLS.md#L318) §7.1：`selection 语义` 行微调——补一句"选项以垂直列表一行一个展示，单选点行即发、多选整行高亮（右侧对勾）勾选后确认提交"。无协议字段变化。
- [AGENTS.md](AGENTS.md#L722) 记忆点 10：变更概览 ③ 交互 末尾追加"选项垂直列表布局（一行一个，整行高亮 + 右侧对勾，单选点击即发 / 多选勾选确认）"。

## 假设与决策

- 单选点击即发、多选勾选后确认提交（用户已确认，保持现状行为）。
- 选中态 = 整行高亮 + 右侧对勾（用户已确认）。
- 对勾保留现有 `✓` 字符，不引入新 SVG（最小改动；后续如需精致图标可再换）。
- 纯 CSS 实现垂直布局（JS 结构已兼容），改动面最小、回归风险低。
- 选项行间距 6px、行内 padding 8px 12px、圆角 8px，贴合现有设计体系与触摸目标（行高约 34px，接近 44px 触摸规范下限，可接受——桌面端为主）。

## 影响范围

- **改动文件**：`frontend/src/css/components/ai-chat.css`（`.ai-ask-options` / `.ai-ask-option` / `.ai-ask-option.selected` / `.ai-ask-check` 四个规则块）。
- **文档**：`internal/agent/TOOLS.md` §7.1、`AGENTS.md` 记忆点 10。
- **不改动**：`ai-chat.js`、`index.html`、后端协议。

## 验证步骤

1. `cd frontend && npm run build` 通过（exit 0）。
2. Grep 确认 `.ai-ask-options` 为 `column` 布局、无 `flex-wrap` 残留（选项区）。
3. 手动验证（`wails dev`）：
   - 单选触发：选项垂直一行一个，点击任一行立即发送并隐藏面板。
   - 多选触发：点击行整行高亮 + 右侧对勾淡入；再次点击取消；「确认提交」发送 `我选择：A、B`（勾选 + 输入并存时拼接 `补充说明`）；均空抖动提示。
   - 切换 14 主题抽查深浅两色，选中态/对勾可读。
   - 生命周期回归：回答后隐藏、绕答/切会话/清空会话均隐藏。
4. 文档改动核对（TOOLS.md §7.1、AGENTS.md 记忆点 10 已同步）。
