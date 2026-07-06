# 移除翻译方向默认选中状态，改为点击后选中

## 问题描述

当前"更多技能 → 翻译"的方向选择区中，"翻译为中文"在 HTML 中硬编码了 `selected` 类和 `checked` 属性，导致：
- 首次打开菜单时，无任何翻译技能被激活，但视觉上"翻译为中文"却显示为选中态
- 与实际状态不匹配，造成困惑

需要改为：默认无任何选项选中，等用户真正点击选项后才标记为选中。

## 修改方案

### 修改文件

1. `frontend/index.html` — 移除硬编码的默认选中
2. `frontend/src/js/ai-chat.js` — 打开菜单时恢复之前选择的视觉状态

### 修改 1：HTML - 移除默认选中

`index.html` 第 879-880 行：

```html
<!-- 修改前 -->
<label class="ai-chat-skills-option selected">
    <input type="radio" name="translate-dir" value="to_chinese" checked>

<!-- 修改后 -->
<label class="ai-chat-skills-option">
    <input type="radio" name="translate-dir" value="to_chinese">
```

- 移除 `selected` 类，初始无视觉高亮
- 移除 `checked` 属性，初始无 radio 选中

### 修改 2：JS - 打开菜单时同步选中状态

`ai-chat.js` 第 807-814 行（菜单打开/关闭事件），在打开菜单时增加逻辑：

如果 `activeSkills.translate` 已存在（用户之前已选择过翻译方向），则自动恢复对应选项的 `.selected` 类和 `checked` 属性。这样：

- **首次打开**：无选中，符合用户预期
- **后续打开**（已选过方向）：恢复之前的选择，方便查看或切换

```js
skillsBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    skillsDropdown.classList.toggle('open');
    
    if (skillsDropdown.classList.contains('open')) {
        // 打开菜单时，恢复已选的翻译方向视觉状态
        if (skillsTranslateOptions && activeSkills.translate) {
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
        }
    } else {
        if (skillsTranslateOptions) skillsTranslateOptions.classList.remove('open');
    }
});
```

### 影响范围

- HTML 中移除两个属性（`selected` 类 + `checked` 属性）
- JS 中在菜单打开事件增加同步逻辑（约 12 行）
- 不影响点击处理逻辑（上次修复的 `selected` 类更新保持不变）
- 不影响其他技能（编程、写作等）
