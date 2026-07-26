# AI 助手「更多技能」按钮交互修改计划

## 概述

将选中技能后"隐藏按钮"的行为改为"禁用按钮"，取消技能后恢复按钮可用状态。

## 当前状态分析

- 技能按钮 `#aiChatMoreSkillsBtn` 和技能 chips 栏在 `renderSkillChips()` 函数中联动控制：
  - **无激活技能**：隐藏 `skillBar`，显示 `skillsBtn`（[ai-chat.js#L1656-L1659](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L1656-L1659)）
  - **有激活技能**：显示 `skillBar`，**隐藏** `skillsBtn`（[ai-chat.js#L1661-L1662](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L1661-L1662)）
- `skillsBtn` 只有这两处 `style.display` 操作，无其他引用
- `clearSkillsState()`（[ai-chat.js#L4737-L4742](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L4737-L4742)）切换会话时调用，隐藏 `skillBar` 但未处理 `skillsBtn` 显示状态
- 按钮基础样式在 `.ai-chat-toolbar-btn`（[ai-chat.css#L2314-L2329](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/css/components/ai-chat.css#L2314-L2329)），尚无 `:disabled` 样式

## 修改方案

### 文件 1：`frontend/src/js/ai-chat.js`

#### 1.1 `renderSkillChips()` — 按钮隐藏改为禁用

位置：[L1656-L1662](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L1656-L1662)

```js
// 修改前：
if (keys.length === 0) {
    skillBar.style.display = 'none';
    if (skillsBtn) skillsBtn.style.display = '';   // ← 恢复显示
    return;
}
skillBar.style.display = '';
if (skillsBtn) skillsBtn.style.display = 'none';  // ← 隐藏按钮

// 修改后：
if (keys.length === 0) {
    skillBar.style.display = 'none';
    if (skillsBtn) {
        skillsBtn.disabled = false;                // ← 恢复可用
        skillsBtn.classList.remove('is-disabled');
    }
    return;
}
skillBar.style.display = '';
if (skillsBtn) {
    skillsBtn.disabled = true;                     // ← 禁用按钮
    skillsBtn.classList.add('is-disabled');
}
```

#### 1.2 `clearSkillsState()` — 补充恢复按钮状态

位置：[L4737-L4742](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L4737-L4742)

```js
// 修改后：
function clearSkillsState() {
    activeSkills = {};
    if (skillBar) skillBar.style.display = 'none';
    if (skillChips) skillChips.innerHTML = '';
    if (skillsBtn) {                                // ← 新增：恢复按钮可用
        skillsBtn.disabled = false;
        skillsBtn.classList.remove('is-disabled');
    }
    closeLangPicker();
}
```

### 文件 2：`frontend/src/css/components/ai-chat.css`

#### 2.1 新增禁用样式

追加在 `.ai-chat-toolbar-btn` 相关样式块之后（[ai-chat.css#L2341](file:///d%3A/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/css/components/ai-chat.css#L2341) 之后）

```css
/* 更多技能按钮禁用态 */
#aiChatMoreSkillsBtn.is-disabled,
#aiChatMoreSkillsBtn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
    pointer-events: none;
}
```

## 实现步骤（执行顺序）

1. 修改 `ai-chat.js` 中 `renderSkillChips()` 的按钮显示逻辑（1.1）
2. 修改 `ai-chat.js` 中 `clearSkillsState()` 补充恢复逻辑（1.2）
3. 修改 `ai-chat.css` 新增禁用样式（2.1）
4. 验证：在浏览器中测试选中技能 → 按钮呈禁用态（半透明、不可点击）；取消技能 → 按钮恢复正常

## 决策说明

- 使用原生 `disabled` 属性 + CSS 类双保险，`disabled` 阻止 click 事件，CSS 类提供视觉反馈
- 保留 `pointer-events: none` 防止误触
- 禁用样式 opacity 0.45 与项目中其他禁用元素的视觉风格一致（参考工具栏其他按钮的禁用态）

## 验证方式

1. 打开 AI 助手界面，点击「更多技能」选择一个技能
2. 确认按钮变为禁用样式（半透明、不可点击），下方显示技能 chips
3. 点击 chip 的叉号取消技能
4. 确认按钮恢复正常样式，可点击展开下拉菜单
5. 切换 AI 会话，确认技能状态被正确清理，按钮恢复可用
