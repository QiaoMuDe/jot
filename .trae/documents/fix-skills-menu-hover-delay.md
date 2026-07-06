# 修复"更多技能"菜单悬停背景延迟问题

## 问题描述

"更多技能"下拉菜单的菜单项，鼠标悬停时背景色（`background: var(--hover-bg)`）需要 0.5\~0.8 秒才出现，造成交互割裂感。

## 根因分析

### 问题定位文件

`frontend/src/css/components/ai-chat.css`

### 问题机制

1. **基础规则**（第 770 行）为菜单项定义了 transition：

   ```css
   transition: opacity 0.25s ease-out, transform 0.25s ease-out, background 0.1s;
   ```

   隐含的 `transition-delay` 为 `[0s, 0s, 0s]`。

2. **Stagger 延迟规则**（第 778-789 行）为每个菜单项的入场动画设置了递增延迟：

   ```css
   .ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(1) { transition-delay: 0.06s; }
   .ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(2) { transition-delay: 0.10s; }
   /* ... 依次递增到 0.50s */
   .ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(12) { transition-delay: 0.50s; }
   ```

3. **关键问题**：`transition-delay` 只设了一个值（如 `0.06s`），浏览器会将其**扩展到所有 transition 属性**上。于是 `background` 属性也被赋予了 stagger 延迟（如 `0.46s`）。

4. **后果**：当用户 hover 到第 11 个菜单项时，`background` 的 transition-delay 是 `0.46s` + duration `0.1s`，用户需要等约 0.56s 才能看到背景变化，与描述吻合。

## 修改方案

### 修改文件

`frontend/src/css/components/ai-chat.css`，第 778-789 行

### 修改方式

将 nth-child 各规则的 `transition-delay` 从单个值改为三个值，分别对应 `opacity`、`transform`、`background` 三个属性，其中 `background` 始终为 `0s`（无延迟）：

```css
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(1) { transition-delay: 0.06s, 0.06s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(2) { transition-delay: 0.10s, 0.10s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(3) { transition-delay: 0.14s, 0.14s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(4) { transition-delay: 0.18s, 0.18s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(5) { transition-delay: 0.22s, 0.22s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(6) { transition-delay: 0.26s, 0.26s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(7) { transition-delay: 0.30s, 0.30s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(8) { transition-delay: 0.34s, 0.34s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(9) { transition-delay: 0.38s, 0.38s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(10) { transition-delay: 0.42s, 0.42s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(11) { transition-delay: 0.46s, 0.46s, 0s; }
.ai-chat-skills-dropdown.open .ai-chat-skills-item:nth-child(12) { transition-delay: 0.50s, 0.50s, 0s; }
```

每个值的映射关系是按 `transition-property` 列表顺序：

* 第一个值（0.XXs）→ `opacity` 的延迟

* 第二个值（0.XXs）→ `transform` 的延迟

* 第三个值（0s）→ `background` 的延迟（始终为 0，不受 stagger 影响）

## 影响范围

* 仅修改 CSS 文件中的 transition-delay 值

* 不涉及 JS 逻辑改动

* 不涉及 HTML 结构改动

* 无副作用：hover 背景响应即时，入场滑入动画保持原有 stagger 效果

## 验证方式

1. 重新构建前端并打开 AI 助手页面
2. 点击"更多技能"打开下拉菜单
3. 快速在多个菜单项上移动鼠标，检查背景色是否能即时跟随
4. 确认菜单项的入场滑入动画仍保持逐个出现的 stagger 效果

