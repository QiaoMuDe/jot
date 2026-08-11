# 反问面板选项宽度自适应（最长长度 + 280px 最小宽度）Plan

## 概要

将反问面板（`#aiAskPanel`）的选项列表宽度从「固定撑满面板」改为**内容自适应**：容器宽度 = 最长选项行（文本 + 多选 checkbox + 内边距）的宽度，整体靠左；同时设置 **`min-width: 280px`** 兜底——当选项文本很短（如"是/否"）时保证选择框不至于过窄。`max-width: 100%` 防止超长选项溢出面板。

## 现状分析

* [ai-chat.css](frontend/src/css/components/ai-chat.css#L4113-L4118) `.ai-ask-options`：`display:flex; flex-direction:column; gap:6px; margin-bottom:10px`——块级元素默认占满父容器（`.ai-ask-panel`）全宽，所有选项行 `width:100%` 撑满。

* [.ai-ask-option](frontend/src/css/components/ai-chat.css#L4120-L4134)：`width:100%`、`padding:8px 12px` 的整行列表项（无 checkbox 时内容很窄也会被撑满）。

* 单选/多选共用 `.ai-ask-options`（[ai-chat.js](frontend/src/js/ai-chat.js#L4146-L4169) 多选分支含 checkbox span，[L4186-L4199](frontend/src/js/ai-chat.js#L4186-L4199) 单选分支纯文本）——两种模式共用同一宽度规则，无需区分。

* 输入行 `.ai-ask-input-row` 保持全宽不变（用户未要求改动）。

## 变更方案

**唯一改动文件**：`frontend/src/css/components/ai-chat.css` 的 `.ai-ask-options` 规则块（L4113-L4118）。

```css
.ai-ask-options {
    display: flex;
    flex-direction: column;
    align-items: flex-start; /* 收缩后整体靠左 */
    width: fit-content;      /* 容器宽度 = 最长选项行的内容宽度 */
    min-width: 280px;        /* 短选项时保证选择框宽度（用户确认 280px） */
    max-width: 100%;         /* 超长选项不溢出面板 */
    gap: 6px;
    margin-bottom: 10px;
}
```

要点说明：

* `.ai-ask-option`（`width:100%`）**保持不动**——父容器收缩后，行宽 = 容器宽 = 最长选项宽，所有行统一对齐到同一宽度。

* `align-items: flex-start`：容器宽度小于面板时整体靠左，不居中也无多余占位。

* `min-width: 280px`：最短宽度兜底（覆盖常见中文选项 4-10 字 + 多选 checkbox + padding 的舒适点击宽度）。

* `max-width: 100%`：`fit-content` 在文本不换行时可超出父级，此约束保证不溢出面板（超长选项文本在行内换行）。

* 单选行无 checkbox，容器宽度按单选文本计算；多选行含 checkbox，宽度自动含 checkbox + margin——各自模式内一致。

## 假设与决策

* 最小宽度 280px（用户通过 AskUserQuestion 确认）。

* 所有选项行保持等宽（= 最长选项），不做"每行各自内容宽"——符合"根据最长长度设置宽度"的表述。

* 输入行（全宽）与关闭按钮、问句标题等保持不变。

* 纯 CSS 改动，JS/后端/协议零改动；`width: fit-content` 在 Wails WebView2（Chromium）完全支持。

## 影响范围

* **改动文件**：`frontend/src/css/components/ai-chat.css`（仅 `.ai-ask-options` 一个规则块）。

* **不改动**：`ai-chat.js`、`index.html`、后端、其他 CSS 规则。

## 验证步骤

1. `cd frontend && npm run build` 通过（exit 0）。
2. 手动验证（`wails dev`）：

   * 短选项（如"是 / 否"）触发：选项容器宽 = 280px（min-width 生效），整体靠左。

   * 中长选项触发：容器宽 = 最长选项宽（>280px 时），行内文本、hover/选中整行高亮正常。

   * 超长选项：不超过面板宽度，文本行内换行，无横向溢出。

   * 单选/多选两种模式均验证；多选 checkbox 在收缩后仍左侧常显、选中填充正常。

   * 关闭按钮、自定义输入行布局不受影响。
3. 主题抽查：深浅两色下选项行边框/高亮/checkbox 可读。

