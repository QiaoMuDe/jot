# 计划：移除技能菜单选中对号，改为背景色区分

## 摘要

去掉「更多技能」菜单选中项的右侧 `✓` 对号标记，改为用 `accent` 淡色背景来表示选中状态，让视觉更清爽且与 hover 态有明显区分。

## 现状分析

* 选中态（`.active`）：文字 + 图标变 accent 色，右侧 `::after { ✓ }` 标记

* 悬停态（`:hover`）：文字 + 图标变 accent 色，背景为 `var(--hover-bg)`

* 选中态和悬停态的**文字+图标颜色完全一样**，唯一区别是右侧多了一个小对号，区分度不足

## 修改方案

### 1. CSS — `frontend/src/css/components/ai-chat.css`

**删除** (line 1134-1139)：`.ai-chat-skills-item.active::after` 整个规则块（移除对号）

**修改** (line 1131-1133)：`.ai-chat-skills-item.active` 增加 `background` 属性

```css
.ai-chat-skills-item.active {
    color: var(--accent);
    background: color-mix(in srgb, var(--accent) 10%, transparent);
}
```

* 使用 `color-mix` 将 accent 色以 10% 比例混合到透明背景，生成淡色底色

* 选中态与 hover 态（`var(--hover-bg)` 是灰色/米色系）在色相上明显不同，用户能一眼区分

* hover 时在选中项上仍保留 `hover-bg` 覆盖效果（CSS 优先级 `.active` 与 `:hover` 分别独立，`:hover` 在后面会覆盖背景色，这是期望行为 —— 鼠标悬停时视觉反馈优先）

### 2. JS — `frontend/src/js/ai-chat.js`

**无需修改**。`updateSkillsMenuActiveState()` 仍然只负责 toggle `.active` class，逻辑不变。

### 3. HTML — `frontend/index.html`

**无需修改**。菜单结构不变。

## 视觉对比

| 状态    | 当前                          | 修改后                                |
| ----- | --------------------------- | ---------------------------------- |
| 默认    | 灰图标 + 主文字                   | 灰图标 + 主文字                          |
| 悬停    | accent 文字+图标 + hover-bg     | accent 文字+图标 + hover-bg            |
| 选中    | accent 文字+图标 + ✓            | accent 文字+图标 + **accent 10% 淡色背景** |
| 选中+悬停 | accent 文字+图标 + ✓ + hover-bg | accent 文字+图标 + hover-bg（hover 优先）  |

## 涉及文件

* `frontend/src/css/components/ai-chat.css` — 唯一修改文件

## 验证步骤

1. 打开 AI 助手，点击「更多技能」下拉菜单
2. 点击一个技能（如「翻译」），确认：a) 右侧 `✓` 已消失；b) 菜单项出现 accent 淡色背景
3. 鼠标悬停在选中项上，确认背景切换为 `hover-bg`
4. 点击其他技能，确认选中态切换正确
5. 点击已选中的技能取消，确认选中背景消失
6. 切换暗色主题，确认背景色表现正常

