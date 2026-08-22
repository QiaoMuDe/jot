# 合并三个栏为一个统一栏

## 概述

将 `#aiChatRefBar`（引用笔记）、`#aiChatSkillBar`（技能）、`#aiChatFileBar`（上传文件）三个独立的栏合并成一个统一的 `#aiChatBarsArea`，所有 chip 在同一行排列，超出行宽自动换行。

## 当前状态

```
#aiChatBarsArea (flex-direction: column)
  ├── #aiChatRefBar (flex, flex-wrap, width: 100%)
  │     └── #aiChatRefChips (flex, flex-wrap, width: 100%)
  ├── #aiChatSkillBar (flex, flex-wrap, width: 100%)
  │     └── #aiChatSkillChips (flex, flex-wrap, width: 100%)
  └── #aiChatFileBar (flex, flex-wrap, width: 100%)
        └── #aiChatFileChips (flex, flex-wrap, width: 100%)
```

三个栏垂直堆叠，每栏独占一行，浪费空间。

## 变更内容

### 1. HTML — `index.html`

**删除**三个包装层 `#aiChatRefBar`、`#aiChatSkillBar`、`#aiChatFileBar`，将 chips 容器直接作为 `#aiChatBarsArea` 的子元素。

```html
<div id="aiChatBarsArea" class="ai-chat-bars-area" style="display:none;">
    <div id="aiChatRefChips" class="ai-chat-ref-chips"></div>
    <div id="aiChatSkillChips" class="ai-chat-skill-chips"></div>
    <div id="aiChatFileChips" class="ai-chat-file-chips"></div>
</div>
```

### 2. CSS — `ai-chat.css`

**修改** `.ai-chat-bars-area`：

* `flex-direction: column` → `flex-direction: row; flex-wrap: wrap`

* `align-items: flex-start`

* 保持 `gap: 6px`，chip 会在同一行排列，超出行宽自动换行

**修改** chips 容器（`.ai-chat-ref-chips`、`.ai-chat-skill-chips`、`.ai-chat-file-chips`）：

* `width: 100%` → `width: auto`

* 保持 `display: flex; flex-wrap: wrap; gap: 6px`

**删除** 三个 bar 包装层的 CSS 规则：

* `.ai-chat-ref-bar`（第 2497 行）

* `.ai-chat-skill-bar`（第 1431 行）

* `.ai-chat-file-bar`（第 2620 行）

### 3. JS — `ai-chat.js`

**删除** 三个变量声明：

* `let fileBar = null`（第 26 行）

* `let refBar = null`（第 84 行）

* `let skillBar = null`（第 127 行）

**删除** `initAIChat()` 中的初始化：

* `refBar = document.getElementById('aiChatRefBar')`（第 326 行）

* `skillBar = document.getElementById('aiChatSkillBar')`（第 353 行）

* `fileBar = document.getElementById('aiChatFileBar')`（第 360 行）

**修改** `updateRefChips()`（第 5048 行）：

* 移除 `if (!refChips || !refBar) return;` 中的 `!refBar` 检查

* 移除 `refBar.style.display = 'none'` 和 `refBar.style.display = ''` 控制

* 改为调用 `updateBarsAreaVisibility()` 统一控制 `barsArea` 显示

**修改** `renderSkillChips()`（第 1663 行）：

* 移除 `if (!skillBar || !skillChips) return;` 中的 `!skillBar` 检查

* 移除 `skillBar.style.display = 'none'` 和 `skillBar.style.display = ''` 控制

* 改为调用 `updateBarsAreaVisibility()` 统一控制 `barsArea` 显示

**修改** `renderFileChips()`（第 5128 行，实际函数名可能是 `updateFileChips`）：

* 移除 `if (!fileChips || !fileBar) return;` 中的 `!fileBar` 检查

* 移除 `fileBar.style.display = 'none'` 和 `fileBar.style.display = ''` 控制

* 改为调用 `updateBarsAreaVisibility()` 统一控制 `barsArea` 显示

**新增** `updateBarsAreaVisibility()` 函数：

```js
function updateBarsAreaVisibility() {
    if (!barsArea) return;
    const hasAny = refChips?.children.length > 0 || skillChips?.children.length > 0 || fileChips?.children.length > 0;
    barsArea.style.display = hasAny ? '' : 'none';
}
```

## 文件修改清单

| 文件                                        | 修改                                                                                         | 说明        |
| ----------------------------------------- | ------------------------------------------------------------------------------------------ | --------- |
| `frontend/index.html`                     | 删除 3 个 bar 包装层，chips 容器直接作为 barsArea 子元素                                                   | 结构扁平化     |
| `frontend/src/css/components/ai-chat.css` | 修改 bars-area 为 `flex-direction: row; flex-wrap: wrap`；chips 容器 `width: auto`；删除 3 个 bar 规则 | 行内排列，自动换行 |
| `frontend/src/js/ai-chat.js`              | 删除 3 个 bar 变量与初始化；3 个渲染函数中移除 bar 控制，改为统一 `updateBarsAreaVisibility()`                      | 逻辑简化      |

## 合并后结构

```
#aiChatBarsArea (flex row, flex-wrap, gap: 6px)
  ├── #aiChatRefChips (inline-flex, flex-wrap, gap: 6px, width: auto)
  ├── #aiChatSkillChips (inline-flex, flex-wrap, gap: 6px, width: auto)
  └── #aiChatFileChips (inline-flex, flex-wrap, gap: 6px, width: auto)
```

所有 chip 在同一行排列，空间不足时自动换行。

## 验证

1. 引用笔记 + 技能 + 上传文件同时存在，chip 在同一行排列，不浪费行
2. 单个类型有多个 chip 时，超出行宽自动换行
3. 所有 chip 类型都清空时，barsArea 自动隐藏
4. 切换会话后 barsArea 状态正确

