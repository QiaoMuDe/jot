# 修复移除技能后下拉菜单选中态残留问题

## 问题描述

在 chip 指示条中点击 X 移除翻译技能后，再次打开"更多技能"菜单，翻译方向选择区仍显示之前选中的选项为选中态（紫色文字 + 勾号）。

## 根因分析

**点击 X 移除时**（`ai-chat.js` 第 1586-1592 行）：

```js
delete activeSkills[skill];
renderSkillChips();
```

只删除了 `activeSkills.translate` 并重绘 chip 条，**没有清除菜单中** **`<label>`** **的** **`.selected`** **类**。

**重新打开菜单时**（`ai-chat.js` 第 810-823 行）：

```js
if (skillsTranslateOptions && activeSkills.translate) {  // ← 条件不满足，跳过
    // ... 恢复选中态
}
```

因为 `activeSkills.translate` 已被删除，同步代码**直接跳过**，旧的 `.selected` 类残留在 HTML 元素上。

## 修改方案

### 修改文件

`frontend/src/js/ai-chat.js`，第 810-823 行

### 修改内容

将原有的 `if (activeSkills.translate)` 条件块改为完整的 if/else 结构：

* **有翻译技能** → 恢复对应选项的选中态（不变）

* **无翻译技能** → 清除所有选项的选中态（新增）

```js
if (skillsDropdown.classList.contains('open')) {
    // 打开菜单时，同步翻译方向的视觉状态
    if (skillsTranslateOptions) {
        if (activeSkills.translate) {
            const dir = activeSkills.translate.direction;
            skillsTranslateOptions.querySelectorAll('.ai-chat-skills-option').forEach(el => {
                const radio = el.querySelector('input[type="radio"]');
                if (radio && radio.value === dir) {
                    el.classList.add('selected');
                    radio.checked = true;
                } else {
                    el.classList.remove('selected');
                }
            });
        } else {
            // 未激活翻译技能，清除所有选中态
            skillsTranslateOptions.querySelectorAll('.ai-chat-skills-option').forEach(el => {
                el.classList.remove('selected');
                const radio = el.querySelector('input[type="radio"]');
                if (radio) radio.checked = false;
            });
        }
    }
} else {
    if (skillsTranslateOptions) skillsTranslateOptions.classList.remove('open');
}
```

### 影响范围

* 仅修改菜单打开事件的同步逻辑

* 不涉及 chip 移除事件、点击处理等其他代码

* 不涉及 HTML 和 CSS

