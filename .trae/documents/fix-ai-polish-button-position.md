# 修复 AI 助手「优化表达」按钮位置

## Summary

输入坞一体化重构后，AI 助手输入区右下角出现了「优化表达」按钮与卡片右边框之间的大片留白（按钮距右边框约 54px），且按钮悬浮在文字上方遮挡行尾内容。本计划将该按钮**对齐到输入坞右上角（与发送按钮同右缘）**，同时给 textarea 预留右侧空间，确保文字不被遮挡；单行右居中 / 多行右上角的动态逻辑保持不变。

## 当前状态分析

### 输入坞结构（[index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1168-L1320)）

```
.ai-chat-composer（flex row, align-items: flex-end, padding: 12px 12px 10px 14px, position: relative）
 ├─ .ai-chat-composer-main（flex: 1, column）
 │   ├─ .ai-chat-input-wrap（position: relative, flex: 1）  ← textarea + 优化按钮
 │   │   ├─ textarea#aiChatInput（padding: 6px 2px）
 │   │   └─ button#aiChatPolishBtn（absolute: right 6px / top 6px）
 │   └─ .ai-chat-toolbar（工具行）
 └─ .ai-chat-composer-actions（flex column, flex-shrink: 0）  ← 发送/停止按钮（右缘距卡片边 12px）
```

### 问题根因

1. **留白**：`.ai-chat-composer-actions`（38px 发送按钮 + 10px gap）在 flex 流内占据右侧约 48px，`.ai-chat-input-wrap` 的右缘止步于操作列左侧，按钮 `right: 6px` 使按钮距卡片右边框约 `6 + 48 + 12 ≈ 66px`（padding 12px 时约 54px 净空），操作列上方悬空 → 大片留白。
2. **文字被遮挡**：重构后 textarea 水平内边距由 `9px 14px` 缩为 `6px 2px`，行尾文字会延伸到悬浮按钮下方。
3. 打字光标（`.is-typing::before`，`right: 36px`）与新预留空间不同步。

### 未改动部分（确认无影响）

* JS 对按钮的引用全部走 `polishBtn.closest('.ai-chat-input-wrap')`（[ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js) L475/531/556/563/567/578/596、L1856），按钮保持位于 wrap 内，**无需改动 JS**。

* `.ai-chat-composer-actions` 在 JS 中无引用，仅 CSS L1271 一处定义。

## Proposed Changes

仅修改 [ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)，共 5 处：

### 1. `.ai-chat-composer-actions`（L1271）— 移出 flex 流，绝对定位到右下角

```css
.ai-chat-composer-actions {
    position: absolute;
    right: 12px;
    bottom: 10px;            /* 与 composer padding-bottom 对齐 */
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    gap: 6px;
    flex-shrink: 0;
    padding-bottom: 1px;
}
```

* 操作列移出流后，`.ai-chat-composer-main` 独占整行宽度，`.ai-chat-input-wrap` 延伸至卡片右内边距 → 优化按钮可贴近右边框。

* 发送按钮仍位于右下角，与现位置视觉一致（composer 高度不变）。

### 2. `.ai-chat-toolbar`（L1281）— 预留发送按钮的右列空间

```css
.ai-chat-toolbar {
    padding: 8px 50px 0 0;   /* 右侧留出 12px(padding) + 38px(发送按钮) 宽度，避免工具项被盖住 */
}
```

### 3. `.ai-chat-polish-btn`（L2190）— 右缘与发送按钮对齐

```css
.ai-chat-polish-btn {
    right: 12px;             /* 原 6px → 12px，与发送按钮右缘（12px）对齐，消除留白 */
    top: 6px;
}
```

* 单行逻辑 `.ai-chat-input-wrap.is-single-line .ai-chat-polish-btn { top: 50%; transform: translateY(-50%); }` 不变，此时按钮在输入区右侧垂直居中，与发送按钮共用右缘。

### 4. `.ai-chat-input`（L2275）— textarea 预留右侧空间

```css
.ai-chat-input {
    padding: 6px 72px 6px 2px;   /* 原 6px 2px → 右 72px，保证行尾文字不被按钮遮挡 */
}
```

* 72px ≈ 按钮右偏移 12px + 按钮宽度（图标14px + 间距3px + 文字「优化」约23px + 内边距16px + 边框2px ≈ 58px）≈ 70px，留 2px 余量。

* 「优化中」（3 字）时按钮略宽，但该状态输入区带 `.is-loading` 遮罩（压暗 + 禁用），轻微重叠不可感知。

### 5. `.ai-chat-input-wrap.is-typing::before`（L2252）— 光标与预留空间同步

```css
.ai-chat-input-wrap.is-typing::before {
    right: 74px;             /* 原 36px → 74px，跟随文字右缘（72px padding + 2px） */
}
```

## Assumptions & Decisions

1. **采用「操作列绝对定位」方案**：保留按钮在 `.ai-chat-input-wrap` 内（JS 零改动、单行/多行动态定位逻辑不变），通过将发送按钮列移出 flex 流使输入区满宽，按钮自然贴近右缘。相比「把按钮移到 composer 层」需改 JS 且单行居中需魔法数值，此方案更稳。
2. **保持单行右居中 / 多行右上角**逻辑（用户确认），仅右缘对齐 + 预留右边距。
3. 右侧预留约 72px 空列是用户明确选择的「加右边距」方案的必然结果；多行时该列空置属预期。
4. 14 主题变量全部沿用现有语义变量，不改动任何颜色。
5. 无 HTML / JS 改动。

## Verification

1. `cd frontend && npm run build`（或 `wails dev`）确认无报错。
2. 手动验证：

   * 输入内容后悬停输入区：优化按钮出现在卡片右上角，与发送按钮右缘对齐，二者之间无大片留白。

   * 单行输入：按钮在输入区右侧垂直居中；多行输入：按钮回到右上角。

   * 输入长文本：行尾文字不被按钮遮挡（在预留右列之前换行）。

   * 发送按钮仍在右下角，工具行最右侧「更多技能」不被发送按钮遮挡。

   * 点击优化：按钮变「优化中」旋转、停止按钮出现、发送按钮隐藏；完成后「还原」、打字光标出现在文字末尾。

   * 深色 / 浅色主题下按钮与卡片边框间距视觉一致。

